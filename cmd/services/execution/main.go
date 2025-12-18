package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/execution/server"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/telemetry"
)

func main() {
	// Initialize configuration
	cfg, err := config.Load("execution")
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// Initialize logger
	log := logger.New(cfg.Logger)
	log.Info("Starting Execution Service", "version", cfg.Version, "port", cfg.HTTP.Port)

	// Initialize telemetry
	telConfig := telemetry.Config{
		ServiceName:    cfg.Service.Name,
		JaegerEndpoint: cfg.Telemetry.JaegerEndpoint,
		MetricsEnabled: cfg.Telemetry.MetricsEnabled,
		TracingEnabled: cfg.Telemetry.TracingEnabled,
	}
	tel, err := telemetry.New(telConfig)
	if err != nil {
		log.Fatal("failed to initialize telemetry", "error", err)
	}
	defer tel.Close()

	// Create server
	srv, err := server.New(
		server.WithConfig(cfg),
		server.WithLogger(log),
		server.WithTelemetry(tel),
	)
	if err != nil {
		log.Fatal("failed to create server", "error", err)
	}

	// Start server
	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errCh <- err
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		log.Error("server error", "error", err)
	case sig := <-sigCh:
		log.Info("received shutdown signal", "signal", sig)
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown error", "error", err)
	}

	log.Info("Execution Service stopped gracefully")
}
