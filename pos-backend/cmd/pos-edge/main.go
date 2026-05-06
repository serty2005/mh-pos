package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	platformsqlite "pos-backend/internal/platform/sqlite"
	"pos-backend/internal/pos/api"
	"pos-backend/internal/pos/app"
	possqlite "pos-backend/internal/pos/infra/sqlite"
)

func main() {
	if err := run(); err != nil {
		slog.Error("pos edge backend stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	addr := env("POS_HTTP_ADDR", ":8080")
	dbPath := env("POS_SQLITE_PATH", "data/pos-edge.db")
	migrationsDir := env("POS_SQLITE_MIGRATIONS_DIR", "migrations/sqlite")

	db, err := platformsqlite.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open sqlite and verify runtime gate: %w", err)
	}
	defer db.Close()

	if err := platformsqlite.MigrateDir(context.Background(), db, migrationsDir); err != nil {
		return err
	}

	repo := possqlite.NewRepository(db)
	tx := platformsqlite.NewTxManager(db)
	service := app.NewService(repo, tx, idgen.UUIDGenerator{}, clock.SystemClock{})

	server := &http.Server{
		Addr:              addr,
		Handler:           api.NewRouter(service),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("POS Edge Backend listening", "addr", addr, "sqlite", dbPath)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server failed", "error", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(ctx)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
