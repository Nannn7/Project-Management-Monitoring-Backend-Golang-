package auth

import "time"

// --- Request models ---

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

// --- Response models ---

type LoginResponse struct {
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
	ExpiresIn    int      `json:"expiresIn"` // detik
	User         UserInfo `json:"user"`
}

type UserInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Email       string                 `json:"email"`
	Roles       []string               `json:"roles"`
	IsAdministrator bool              `json:"isAdministrator"`
	Permissions map[string]PermMap    `json:"permissions"`
}

type PermMap struct {
	View   bool `json:"view"`
	Create bool `json:"create"`
	Edit   bool `json:"edit"`
	Delete bool `json:"delete"`
}

// --- DB row model ---

type AppUser struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	IsActive     bool
	UpdatedAt    time.Time
}
