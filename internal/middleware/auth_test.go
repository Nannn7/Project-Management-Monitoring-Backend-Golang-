package middleware

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yourorg/pmo-agi-be/internal/config"
)

func TestParseToken_RejectsWrongType(t *testing.T) {
	config.App.JWTSecret = "ini adalah contoh jwt secret untuk keperluan testing dan panjangnya lebih dari 32 karakter"

	mk := func(typ string, exp time.Time) string {
		c := Claims{UserID: "u1", TokenType: typ,
			RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(exp)}}
		s, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(config.App.JWTSecret))
		return s
	}
	future, past := time.Now().Add(time.Hour), time.Now().Add(-time.Hour)

	tests := []struct {
		name, token, want string
		wantIs            error
		wantAlso          error
	}{
		{"access ok", mk(TokenTypeAccess, future), TokenTypeAccess, nil, nil},
		{"refresh dipakai sbg access", mk(TokenTypeRefresh, future), TokenTypeAccess, ErrWrongTokenType, nil},
		{"access dipakai sbg refresh", mk(TokenTypeAccess, future), TokenTypeRefresh, ErrWrongTokenType, nil},
		{"expired", mk(TokenTypeAccess, past), TokenTypeAccess, ErrInvalidToken, jwt.ErrTokenExpired},
		{"sampah", "bukan.jwt.token", TokenTypeAccess, ErrInvalidToken, jwt.ErrTokenMalformed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseToken(tt.token, tt.want)
			if tt.wantIs == nil {
				if err != nil {
					t.Fatalf("harusnya sukses, dapat: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantIs) {
				t.Fatalf("got %v, want %v", err, tt.wantIs)
			}
			if tt.wantAlso != nil && !errors.Is(err, tt.wantAlso) {
				t.Fatalf("ditolak karena alasan salah: %v (want %v)", err, tt.wantAlso)
			}
		})
	}
}
