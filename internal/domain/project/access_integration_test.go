package project

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Memastikan semua query di ownerSQL valid terhadap schema asli (nama tabel/kolom, join).
// Pakai UUID nol, jadi hasilnya HARUS "tidak ada baris". Typo SQL = error lain, test gagal.
// Read-only (hanya SELECT); jangan diarahkan ke DB produksi.
func TestOwnerSQLValid(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL untuk menjalankan test ini") // satu-satunya alasan skip
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("buat pool: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("DB tidak bisa dihubungi (env var sudah di-set, jadi ini kegagalan, bukan skip): %v", err)
	}

	const zero = "00000000-0000-0000-0000-000000000000"
	for name, query := range ownerSQL {
		t.Run(name, func(t *testing.T) {
			var projectID string
			err := pool.QueryRow(ctx, query, zero).Scan(&projectID)
			if !errors.Is(err, pgx.ErrNoRows) {
				t.Fatalf("harusnya pgx.ErrNoRows, dapat: %v", err)
			}
		})
	}
}
