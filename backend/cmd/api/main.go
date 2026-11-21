package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chami2002/resolveflow-api/internal/config"
	"github.com/chami2002/resolveflow-api/internal/database"
	"github.com/chami2002/resolveflow-api/internal/routes"
)

func main() {
	cfg := config.Load()

	// Connect to PostgreSQL.
	dbCtx, dbCancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer dbCancel()

	dbPool, err := database.Connect(dbCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer dbPool.Close()

	log.Println("Successfully connected to PostgreSQL")

	// Configure HTTP server.
	router := routes.NewRouter(dbPool)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf(
			"ResolveFlow API starting on port %s (environment: %s)",
			cfg.Port,
			cfg.Environment,
		)

		serverErrors <- server.ListenAndServe()
	}()

	// Handle graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server error: %v", err)
		}

	case sig := <-stop:
		log.Printf("Received signal: %v. Shutting down...", sig)

		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Graceful shutdown failed: %v", err)
		}

		log.Println("ResolveFlow API stopped")
	}
}
