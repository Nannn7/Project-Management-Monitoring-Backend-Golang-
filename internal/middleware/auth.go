package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yourorg/pmo-agi-be/internal/config"
	"github.com/yourorg/pmo-agi-be/internal/response"
)

type contextKey string

const UserContextKey contextKey = "user"

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

var (
	ErrInvalidToken   = errors.New("token is not valid")
	ErrWrongTokenType = errors.New("wrong token type")
)

// Claims adalah payload JWT
type Claims struct {
	UserID    string   `json:"userId"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Roles     []string `json:"roles"`
	TokenType string   `json:"typ"`
	jwt.RegisteredClaims
}

func ParseToken(tokenStr, wantType string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims,
		func(t *jwt.Token) (any, error) { return []byte(config.App.JWTSecret), nil },
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	if claims.TokenType != wantType {
		return nil, fmt.Errorf("%w: got %q want %q", ErrWrongTokenType, claims.TokenType, wantType)
	}
	return claims, nil
}

// Auth middleware — validasi JWT dari header Authorization: Bearer <token>
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			response.Unauthorized(w)
			return
		}
		claims, err := ParseToken(strings.TrimPrefix(h, "Bearer "), TokenTypeAccess)
		if err != nil {
			response.Unauthorized(w)
			return
		}
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUser — ambil claims dari context
func GetUser(r *http.Request) *Claims {
	claims, _ := r.Context().Value(UserContextKey).(*Claims)
	return claims
}
