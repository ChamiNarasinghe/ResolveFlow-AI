package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chami2002/resolveflow-api/internal/auth"
	"github.com/chami2002/resolveflow-api/internal/config"
	"github.com/chami2002/resolveflow-api/internal/database"
	"github.com/chami2002/resolveflow-api/internal/logger"
	"github.com/chami2002/resolveflow-api/internal/routes"
	"github.com/chami2002/resolveflow-api/internal/users"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from the project root.
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("Could not load .env file: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}
	appLog := logger.New(cfg.Environment, cfg.LogLevel)
	slog.SetDefault(appLog)
	// Initialize JWT authentication.
	jwtService, err := auth.NewJWTService(
		cfg.JWTSecret,
		cfg.JWTExpiration,
	)
	if err != nil {
		appLog.Error("JWT configuration error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Connect to PostgreSQL.
	dbCtx, dbCancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer dbCancel()

	dbPool, err := database.Connect(dbCtx, cfg.DatabaseURL)
	if err != nil {
		appLog.Error("Database connection failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbPool.Close()

	appLog.Info("Successfully connected to PostgreSQL")

	// One-time admin bootstrap — only runs when all three BOOTSTRAP_ADMIN_*
	// environment variables are set. Silently skips if an admin already exists.
	// Fails fast with a clear error if only some variables are set.
	bootName := cfg.BootstrapAdminName
	bootEmail := cfg.BootstrapAdminEmail
	bootPass := cfg.BootstrapAdminPass
	bootVarsSet := (bootName != "") || (bootEmail != "") || (bootPass != "")
	bootVarsComplete := (bootName != "") && (bootEmail != "") && (bootPass != "")

	if bootVarsSet && !bootVarsComplete {
		appLog.Error("Bootstrap configuration is incomplete: set all three of BOOTSTRAP_ADMIN_NAME, BOOTSTRAP_ADMIN_EMAIL, BOOTSTRAP_ADMIN_PASSWORD or none of them")
		os.Exit(1)
	}

	if bootVarsComplete {
		userRepo := users.NewUserRepository(dbPool)
		authSvc := auth.NewAuthService(userRepo, jwtService)
		bootCtx, bootCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer bootCancel()
		_, bootErr := authSvc.Bootstrap(bootCtx, bootName, bootEmail, bootPass)
		switch {
		case bootErr == nil:
			appLog.Info("Bootstrap: initial administrator account created",
				slog.String("email", bootEmail))
		case errors.Is(bootErr, auth.ErrAdminExists):
			appLog.Info("Bootstrap: administrator already exists, skipping")
		default:
			appLog.Error("Bootstrap failed", slog.String("error", bootErr.Error()))
			os.Exit(1)
		}
	}
	// Root context — cancelled on shutdown to stop background goroutines
	// (e.g. rate-limiter cleanup loops).
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// Configure HTTP server.
	router := routes.NewRouter(
		rootCtx,
		dbPool,
		jwtService,
		cfg.Environment == "development",
		cfg.AIServiceURL,
		cfg.AIServiceAPIKey,
		appLog,
	)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		appLog.Info("ResolveFlow API starting",
			slog.String("port", cfg.Port),
			slog.String("environment", cfg.Environment),
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
			appLog.Error("Server error", slog.String("error", err.Error()))
			os.Exit(1)
		}

	case sig := <-stop:
		appLog.Info("Shutting down", slog.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			appLog.Error("Graceful shutdown failed", slog.String("error", err.Error()))
		}

		appLog.Info("ResolveFlow API stopped")
	}
}
