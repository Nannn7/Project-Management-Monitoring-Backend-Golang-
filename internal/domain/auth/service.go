package auth

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/pmo-agi-be/internal/apperr"
	"github.com/yourorg/pmo-agi-be/internal/config"
	"github.com/yourorg/pmo-agi-be/internal/middleware"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)
var errInvalidCreds = apperr.Unauthenticated("wrong email or password")

// Login — verifikasi email+password, return token pair
func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	var user AppUser
	err := s.db.QueryRow(ctx, `
		SELECT id, name, email, COALESCE(password_hash, ''), is_active
		FROM app_users WHERE lower(email) = lower($1)
	`, req.Email).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.IsActive)

	if errors.Is(err, pgx.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password)) // samain waktu respons
		return nil, errInvalidCreds
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}

	hash := []byte(user.PasswordHash)
	if len(hash) == 0 {
		hash = dummyHash
	}
	if bcrypt.CompareHashAndPassword(hash, []byte(req.Password)) != nil {
		return nil, errInvalidCreds
	}
	if !user.IsActive {
		return nil, apperr.Unauthenticated("akun tidak aktif")
	}

	roles, err := s.getUserRoles(ctx, user.ID)
	if err != nil {
		return nil, apperr.Internal(err)
	}

	accessToken, err := s.generateToken(user, roles, config.App.JWTExpiry, middleware.TokenTypeAccess)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	refreshToken, err := s.generateToken(user, nil, config.App.RefreshTokenExpiry, middleware.TokenTypeRefresh)
	if err != nil {
		return nil, apperr.Internal(err)
	}

	permissions, isAdmin, err := s.getUserPermissions(ctx, user.ID)
	if err != nil {
		return nil, apperr.Internal(err)
	}

	// last_login_at itu nice-to-have: gagal update jangan bikin login gagal
	if _, err := s.db.Exec(ctx, `UPDATE app_users SET last_login_at = now() WHERE id = $1`, user.ID); err != nil {
		log.Printf("warn: gagal update last_login_at: %v", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(config.App.JWTExpiry.Seconds()),
		User: UserInfo{
			ID:              user.ID,
			Name:            user.Name,
			Email:           user.Email,
			Roles:           roles,
			IsAdministrator: isAdmin,
			Permissions:     permissions,
		},
	}, nil
}

// Me — ambil info user dari userID (dari JWT)
func (s *Service) Me(ctx context.Context, userID string) (*UserInfo, error) {
	var user AppUser
	err := s.db.QueryRow(ctx, `
		SELECT id, name, email FROM app_users WHERE id = $1 AND is_active = true
	`, userID).Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		return nil, errors.New("user tidak ditemukan")
	}

	roles, err := s.getUserRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	permissions, isAdmin, err := s.getUserPermissions(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &UserInfo{
		ID:              user.ID,
		Name:            user.Name,
		Email:           user.Email,
		Roles:           roles,
		IsAdministrator: isAdmin,
		Permissions:     permissions,
	}, nil
}

// RefreshToken — validasi refresh token, return access token baru
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	claims, err := middleware.ParseToken(refreshToken, middleware.TokenTypeRefresh)
	if err != nil {
		return nil, apperr.Unauthenticated("refresh token tidak valid")
	}

	var user AppUser
	err = s.db.QueryRow(ctx,
		`SELECT id, name, email, is_active FROM app_users WHERE id = $1`, claims.UserID,
	).Scan(&user.ID, &user.Name, &user.Email, &user.IsActive)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !user.IsActive) {
		return nil, apperr.Unauthenticated("user tidak ditemukan atau tidak aktif")
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}

	roles, err := s.getUserRoles(ctx, user.ID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	accessToken, err := s.generateToken(user, roles, config.App.JWTExpiry, middleware.TokenTypeAccess)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	permissions, isAdmin, err := s.getUserPermissions(ctx, user.ID)
	if err != nil {
		return nil, apperr.Internal(err)
	}

	// Ambil user info terbaru
	// var user AppUser
	err = s.db.QueryRow(ctx, `
		SELECT id, name, email, is_active FROM app_users WHERE id = $1
	`, claims.UserID).Scan(&user.ID, &user.Name, &user.Email, &user.IsActive)
	if err != nil || !user.IsActive {
		return nil, errors.New("user tidak ditemukan atau tidak aktif")
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken, // belum rotation, dicatat sebagai task terpisah
		ExpiresIn:    int(config.App.JWTExpiry.Seconds()),
		User: UserInfo{
			ID:              user.ID,
			Name:            user.Name,
			Email:           user.Email,
			Roles:           roles,
			IsAdministrator: isAdmin,
			Permissions:     permissions,
		},
	}, nil
}

// --- Private helpers ---

func (s *Service) generateToken(user AppUser, roles []string, expiry time.Duration, typ string) (string, error) {
	claims := middleware.Claims{
		UserID:    user.ID,
		TokenType: typ,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	if typ == middleware.TokenTypeAccess {
		claims.Email, claims.Name, claims.Roles = user.Email, user.Name, roles
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(config.App.JWTSecret))
}

func (s *Service) getUserRoles(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT r.name FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		roles = append(roles, name)
	}
	return roles, nil
}

func (s *Service) getUserPermissions(ctx context.Context, userID string) (map[string]PermMap, bool, error) {
	// Cek admin
	var isAdmin bool
	s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM user_roles ur JOIN roles r ON r.id = ur.role_id
			WHERE ur.user_id = $1 AND r.name = 'Administrator'
		)
	`, userID).Scan(&isAdmin)

	rows, err := s.db.Query(ctx, `
		SELECT rp.menu_key,
		       bool_or(rp.can_view)   as can_view,
		       bool_or(rp.can_create) as can_create,
		       bool_or(rp.can_edit)   as can_edit,
		       bool_or(rp.can_delete) as can_delete
		FROM user_roles ur
		JOIN role_permissions rp ON rp.role_id = ur.role_id
		WHERE ur.user_id = $1
		GROUP BY rp.menu_key
	`, userID)
	if err != nil {
		return nil, isAdmin, err
	}
	defer rows.Close()

	permissions := make(map[string]PermMap)
	for rows.Next() {
		var menuKey string
		var p PermMap
		rows.Scan(&menuKey, &p.View, &p.Create, &p.Edit, &p.Delete)
		permissions[menuKey] = p
	}

	// Admin dapat semua akses
	if isAdmin {
		for _, key := range []string{"dashboard", "projects", "master-data"} {
			permissions[key] = PermMap{View: true, Create: true, Edit: true, Delete: true}
		}
	}

	return permissions, isAdmin, nil
}
