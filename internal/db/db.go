package db

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/pmo-agi-be/internal/config"
)

var Pool *pgxpool.Pool

func Connect() {
	var err error
	Pool, err = pgxpool.New(context.Background(), config.App.DatabaseURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}

	if err = Pool.Ping(context.Background()); err != nil {
		log.Fatalf("❌ Database ping failed: %v", err)
	}

	log.Println("✅ Connected to PostgreSQL")
}

func Close() {
	if Pool != nil {
		Pool.Close()
		log.Println("🔌 Database connection closed")
	}
}
