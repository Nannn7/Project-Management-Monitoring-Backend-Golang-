package config

import (
	"log"
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

	jwtExpiry, err := time.ParseDuration(viper.GetString("JWT_EXPIRY"))
	if err != nil {
		jwtExpiry = 15 * time.Minute
	}
	refreshExpiry, err := time.ParseDuration(viper.GetString("REFRESH_TOKEN_EXPIRY"))
	if err != nil {
		refreshExpiry = 168 * time.Hour
	}

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
}
