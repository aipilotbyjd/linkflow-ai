package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	configserver "github.com/linkflow-ai/linkflow-ai/internal/config/server"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
)

func main() {
	cfg, err := config.Load("config")
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	log := logger.New(cfg.Logger)
	log.Info("Starting Config Service", "version", cfg.Version, "port", cfg.HTTP.Port)

	srv, err := configserver.New(
		configserver.WithConfig(cfg),
		configserver.WithLogger(log),
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

	log.Info("Config Service stopped gracefully")
}
