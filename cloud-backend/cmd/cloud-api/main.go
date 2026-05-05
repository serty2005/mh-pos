package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud-backend/internal/cloudsync/api"
	"cloud-backend/internal/cloudsync/app"
	syncpg "cloud-backend/internal/cloudsync/infra/postgres"
	"cloud-backend/internal/platform/clock"
	platformpg "cloud-backend/internal/platform/postgres"
)

func main() {
	if err := run(); err != nil {
		slog.Error("cloud backend stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	addr := env("CLOUD_HTTP_ADDR", ":8090")
	dsn := env("CLOUD_POSTGRES_DSN", "")
	migrationsDir := env("CLOUD_POSTGRES_MIGRATIONS_DIR", "migrations/postgres")
	if dsn == "" {
		return errors.New("CLOUD_POSTGRES_DSN is required")
	}

	ctx := context.Background()
	pool, err := platformpg.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := platformpg.MigrateDir(ctx, pool, migrationsDir); err != nil {
		return err
	}

	repo := syncpg.NewRepository(pool)
	service := app.NewService(repo, clock.SystemClock{})
	server := &http.Server{
		Addr:              addr,
		Handler:           api.NewRouter(service),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("Cloud Backend listening", "addr", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server failed", "error", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
