package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/telemetry"
	"github.com/linkflow-ai/linkflow-ai/internal/webhook/server"
)

func main() {
	cfg, err := config.Load("webhook")
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	log := logger.New(cfg.Logger)
	log.Info("Starting Webhook Service", "version", cfg.Version, "port", cfg.HTTP.Port)

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

	srv, err := server.New(
		server.WithConfig(cfg),
		server.WithLogger(log),
		server.WithTelemetry(tel),
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

	log.Info("Webhook Service stopped gracefully")
}
