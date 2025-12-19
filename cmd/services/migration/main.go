package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/migration/adapters/repository/postgres"
	"github.com/linkflow-ai/linkflow-ai/internal/migration/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/migration/server"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/database"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
)

func main() {
	cfg, err := config.Load("migration")
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	log := logger.New(cfg.Logger)
	log.Info("Starting Migration Service", "version", cfg.Version, "port", cfg.HTTP.Port)

	// Initialize database
	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatal("failed to connect to database", "error", err)
	}
	defer db.Close()

	// Initialize repository
	migrationRepo := postgres.NewMigrationRepository(db)

	// Ensure migration table exists
	if err := migrationRepo.(*postgres.MigrationRepository).EnsureMigrationTable(context.Background()); err != nil {
		log.Error("failed to ensure migration table", "error", err)
	}

	// Initialize service
	migrationSvc := service.NewMigrationService(migrationRepo, cfg.MigrationsPath)

	srv, err := server.New(
		server.WithConfig(cfg),
		server.WithLogger(log),
		server.WithMigrationService(migrationSvc),
	)
	if err != nil {
		log.Fatal("failed to create server", "error", err)
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		log.Error("server error", "error", err)
	case sig := <-sigCh:
		log.Info("received shutdown signal", "signal", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown error", "error", err)
	}

	log.Info("Migration Service stopped gracefully")
}
