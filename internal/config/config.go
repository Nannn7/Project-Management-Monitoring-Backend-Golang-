package config

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Port               string
	DatabaseURL        string
	JWTSecret          string
	JWTExpiry          time.Duration
	RefreshTokenExpiry time.Duration
	CORSOrigin         string
	UploadDir          string
	MaxFileSizeMB      int64
}

var App Config

func Load() {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("⚠️  .env file not found, using environment variables: %v", err)
	}

	jwtExpiry := parseDuration("JWT_EXPIRY", 15*time.Minute)
	refreshExpiry := parseDuration("REFRESH_TOKEN_EXPIRY", 168*time.Hour)

	App = Config{
		Port:               viper.GetString("PORT"),
		DatabaseURL:        viper.GetString("DATABASE_URL"),
		JWTSecret:          viper.GetString("JWT_SECRET"),
		JWTExpiry:          jwtExpiry,
		RefreshTokenExpiry: refreshExpiry,
		CORSOrigin:         viper.GetString("CORS_ORIGIN"),
		UploadDir:          viper.GetString("UPLOAD_DIR"),
		MaxFileSizeMB:      viper.GetInt64("MAX_FILE_SIZE_MB"),
	}

	if App.Port == "" {
		App.Port = "8080"
	}
	if App.UploadDir == "" {
		App.UploadDir = "./uploads"
	}
	if App.MaxFileSizeMB == 0 {
		App.MaxFileSizeMB = 10
	}

	// Validasi SETELAH semua nilai terisi
	if err := App.Validate(); err != nil {
		log.Fatalf("❌ Invalid configuration:\n%v", err)
	}
}

// parseDuration: fallback ke default kalau kosong, tapi WARN kalau
// isinya ada tapi salah format (misal "15mins"), biar typo nggak diam-diam ketutup.
func parseDuration(key string, def time.Duration) time.Duration {
	raw := viper.GetString(key)
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		log.Printf("⚠️  %s=%q tidak valid, pakai default %s", key, raw, def)
		return def
	}
	return d
}

func (c Config) Validate() error {
	var problems []string

	if c.DatabaseURL == "" {
		problems = append(problems, "DATABASE_URL is required")
	}

	if len(c.JWTSecret) < 32 {
		problems = append(problems, "JWT_SECRET must be at least 32 characters")
	}
	for _, prefix := range []string{"ganti-dengan", "replace-with"} {
		if strings.HasPrefix(c.JWTSecret, prefix) {
			problems = append(problems, "JWT_SECRET is still the placeholder from .env.example")
			break
		}
	}

	if c.CORSOrigin == "" || c.CORSOrigin == "*" {
		problems = append(problems, "CORS_ORIGIN must be a specific origin (not empty or *)")
	}

	if len(problems) > 0 {
		return fmt.Errorf(" - %s", strings.Join(problems, "\n - "))
	}
	return nil
}
