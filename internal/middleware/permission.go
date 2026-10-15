package middleware

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/pmo-agi-be/internal/apperr"
	"github.com/yourorg/pmo-agi-be/internal/response"
)

type permissionKey string

const PermissionsKey permissionKey = "permissions"

// Permission dari DB: { "projects": { "view": true, "edit": true, ... } }
type Permission struct {
	View   bool
	Create bool
	Edit   bool
	Delete bool
}

type MenuAction struct{ Menu, Action string }

func RequirePermission(pool *pgxpool.Pool, menuKey, action string) func(http.Handler) http.Handler {
	return RequireAnyPermission(pool, MenuAction{menuKey, action})
}

// RequirePermission — cek apakah user punya akses menu.action
// menuKey contoh: "projects", "master-data", "dashboard"
// action: "view", "create", "edit", "delete"
func RequireAnyPermission(pool *pgxpool.Pool, perms ...MenuAction) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r)
			if user == nil {
				response.Unauthorized(w)
				return
			}
			for _, p := range perms {
				allowed, err := checkPermission(r.Context(), pool, user.UserID, p.Menu, p.Action)
				if err != nil {
					// DB error != "nggak punya akses"
					response.FromError(w, r, apperr.Internal(err))
					return
				}
				if allowed {
					next.ServeHTTP(w, r)
					return
				}
			}
			response.Forbidden(w)
		})
	}
}

// checkPermission — query DB untuk cek permission
// Pengguna dengan banyak role = gabungan (OR) — sesuai spec §2.4
func checkPermission(ctx context.Context, pool *pgxpool.Pool, userID, menuKey, action string) (bool, error) {
	query := `
		SELECT
	  	EXISTS (SELECT 1 FROM user_roles ur JOIN roles r ON r.id = ur.role_id
	          	WHERE ur.user_id = $1 AND r.name = 'Administrator')
	  	OR EXISTS (
	    	SELECT 1
	    	FROM user_roles ur
	    	JOIN role_permissions rp ON rp.role_id = ur.role_id
	    	WHERE ur.user_id = $1
	      	AND rp.menu_key = $2
	      	AND (
	        	($3 = 'view'   AND rp.can_view   = true) OR
	        	($3 = 'create' AND rp.can_create = true) OR
	        	($3 = 'edit'   AND rp.can_edit   = true) OR
	        	($3 = 'delete' AND rp.can_delete = true)
	      	)
	  	)
	`
	var allowed bool
	err := pool.QueryRow(ctx, query, userID, menuKey, action).Scan(&allowed)
	return allowed, err
}

// IsAdministrator — cek apakah user adalah Administrator
func IsAdministrator(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM user_roles ur
			JOIN roles r ON r.id = ur.role_id
			WHERE ur.user_id = $1 AND r.name = 'Administrator'
		)
	`
	var isAdmin bool
	err := pool.QueryRow(ctx, query, userID).Scan(&isAdmin)
	return isAdmin, err
}
