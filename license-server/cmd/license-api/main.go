package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "modernc.org/sqlite"

	"license-server/internal/license/api"
	"license-server/internal/license/app"
	"license-server/internal/license/infra/sqlite"
)

func main() {
	if err := run(); err != nil {
		slog.Error("license server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	addr := env("LICENSE_HTTP_ADDR", ":8095")
	dbPath := env("LICENSE_SQLITE_PATH", "data/license-server.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	repo := sqlite.NewRepository(db)
	if err := repo.Migrate(context.Background()); err != nil {
		return err
	}
	server := &http.Server{Addr: addr, Handler: api.NewRouter(app.NewService(repo)), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		slog.Info("License Server listening", "addr", addr, "sqlite", dbPath)
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
