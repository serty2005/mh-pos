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
	"pos-backend/internal/platform/logging"
	platformsqlite "pos-backend/internal/platform/sqlite"
	"pos-backend/internal/platform/version"
	"pos-backend/internal/pos/api"
	"pos-backend/internal/pos/app"
	poscloudsync "pos-backend/internal/pos/infra/cloudsync"
	possqlite "pos-backend/internal/pos/infra/sqlite"
	"pos-backend/internal/pos/syncsender"
)

func main() {
	if err := run(); err != nil {
		slog.Error("pos edge backend stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	slog.SetDefault(logging.NewJSONLogger("POS_LOG_LEVEL"))

	addr := env("POS_HTTP_ADDR", ":8080")
	dbPath := env("POS_SQLITE_PATH", "data/pos-edge.db")
	migrationsDir := env("POS_SQLITE_MIGRATIONS_DIR", "migrations/sqlite")
	backupDir := env("POS_SQLITE_BACKUP_DIR", "data/backups")
	moduleVersion := version.Resolve("MH_POS_VERSION")

	db, err := platformsqlite.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open sqlite and verify runtime gate: %w", err)
	}
	defer db.Close()

	if err := platformsqlite.MigrateDirWithPolicy(context.Background(), db, dbPath, migrationsDir, platformsqlite.MigrationOptions{
		ModuleName:         "pos-backend",
		ModuleVersion:      moduleVersion,
		BackupDir:          backupDir,
		SchemaRequirements: possqlite.RequiredSchema(),
	}); err != nil {
		return err
	}

	repo := possqlite.NewRepository(db)
	tx := platformsqlite.NewTxManager(db)
	service := app.NewServiceWithOptions(repo, tx, idgen.UUIDGenerator{}, clock.SystemClock{}, app.ServiceOptions{
		MasterDataBackupBeforeFullSnapshot: func(ctx context.Context, req app.MasterDataBackupRequest) error {
			_, err := platformsqlite.BackupDatabase(ctx, db, dbPath, backupDir, platformsqlite.BackupOptions{
				Action:         "backup_before_data_load",
				ModuleName:     "pos-backend",
				CurrentVersion: "local_snapshot",
				TargetVersion:  fmt.Sprintf("cloud_version_%d", req.CloudVersion),
				Reason:         req.FullSnapshotReason,
			})
			return err
		},
	})
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	server := &http.Server{
		Addr:              addr,
		Handler:           api.NewRouter(service),
		ReadHeaderTimeout: 5 * time.Second,
	}

	if envBool("POS_SYNC_SENDER_ENABLED", true) {
		cloudEndpoint := env("POS_CLOUD_SYNC_URL", "http://localhost:8090/api/v1/sync/edge-events")
		worker := syncsender.NewWorker(service, poscloudsync.NewClient(cloudEndpoint), syncsender.Config{
			WorkerID:     env("POS_SYNC_SENDER_ID", "pos-sync-sender-main"),
			BatchSize:    envInt("POS_SYNC_SENDER_BATCH_SIZE", 25),
			PollInterval: envDuration("POS_SYNC_SENDER_POLL_INTERVAL", 2*time.Second),
			ReclaimAfter: envDuration("POS_SYNC_SENDER_RECLAIM_AFTER", 5*time.Minute),
			SendTimeout:  envDuration("POS_SYNC_SENDER_SEND_TIMEOUT", 10*time.Second),
		}, slog.Default())
		go worker.Run(rootCtx)
	} else {
		slog.Info("POS sync sender disabled")
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
	rootCancel()

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

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "TRUE", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "no", "NO", "off", "OFF":
		return false
	default:
		return fallback
	}
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var parsed int
	if _, err := fmt.Sscanf(v, "%d", &parsed); err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(v)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
