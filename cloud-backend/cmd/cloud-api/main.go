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
	"cloud-backend/internal/cloudsync/contracts"
	syncpg "cloud-backend/internal/cloudsync/infra/postgres"
	masterapp "cloud-backend/internal/masterdata/app"
	masterpg "cloud-backend/internal/masterdata/infra/postgres"
	"cloud-backend/internal/platform/clock"
	"cloud-backend/internal/platform/logging"
	platformpg "cloud-backend/internal/platform/postgres"
	"cloud-backend/internal/platform/version"
)

func main() {
	if err := run(); err != nil {
		slog.Error("cloud backend stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	slog.SetDefault(logging.NewJSONLogger("CLOUD_LOG_LEVEL"))

	addr := env("CLOUD_HTTP_ADDR", ":8090")
	dsn := env("CLOUD_POSTGRES_DSN", "")
	migrationsDir := env("CLOUD_POSTGRES_MIGRATIONS_DIR", "migrations/postgres")
	backupDir := env("CLOUD_POSTGRES_BACKUP_DIR", "data/cloud-backups")
	moduleVersion := version.Resolve("MH_POS_VERSION")
	if dsn == "" {
		return errors.New("CLOUD_POSTGRES_DSN is required")
	}

	ctx := context.Background()
	pool, err := platformpg.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := platformpg.MigrateDirWithPolicy(ctx, pool, migrationsDir, platformpg.MigrationOptions{
		ModuleName:         "cloud-backend",
		ModuleVersion:      moduleVersion,
		BackupDir:          backupDir,
		SchemaRequirements: syncpg.RequiredSchema(),
	}); err != nil {
		return err
	}
	if err := syncpg.EnsureCurrencyReferenceCatalog(ctx, pool, contracts.CanonicalActiveCurrencyProfiles()); err != nil {
		return err
	}

	repo := syncpg.NewRepository(pool)
	service := app.NewService(repo, clock.SystemClock{})
	masterRepo := masterpg.NewRepository(pool)
	masterService := masterapp.NewService(masterRepo, clock.SystemClock{}, masterapp.RandomIDGenerator{})
	server := &http.Server{
		Addr:              addr,
		Handler:           api.NewRouter(service, masterService),
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
