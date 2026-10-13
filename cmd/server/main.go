package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourorg/pmo-agi-be/internal/config"
	"github.com/yourorg/pmo-agi-be/internal/db"
	"github.com/yourorg/pmo-agi-be/internal/router"
)

func main() {
	// Load config dari .env
	config.Load()

	// Koneksi ke PostgreSQL
	db.Connect()
	defer db.Close()

	// Setup router
	handler := router.New(db.Pool)

	// HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", config.App.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Jalankan server di goroutine supaya bisa graceful shutdown
	go func() {
		log.Printf("🚀 Server jalan di http://localhost:%s", config.App.Port)
		log.Printf("📋 API tersedia di http://localhost:%s/api/v1", config.App.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	// Graceful shutdown saat dapat sinyal SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %v", err)
	}
	log.Println("✅ Server stopped")
}
