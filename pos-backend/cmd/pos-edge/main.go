package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"mh-pos-platform/config"
	"mh-pos-platform/licensegate"
	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	"pos-backend/internal/platform/logging"
	platformsqlite "pos-backend/internal/platform/sqlite"
	"pos-backend/internal/platform/version"
	"pos-backend/internal/pos/api"
	"pos-backend/internal/pos/app"
	poscloudsync "pos-backend/internal/pos/infra/cloudsync"
	posprovisioninghttp "pos-backend/internal/pos/infra/provisioninghttp"
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
	cfg, err := config.Load("POS_CONFIG_PATH", "config/pos-edge.json")
	if err != nil {
		return err
	}
	slog.SetDefault(logging.NewJSONLoggerWithLevel(cfg.Get("POS_LOG_LEVEL", "")))
	if cfg.Path() != "" {
		slog.Info("POS config file applied", "config_path", cfg.Path())
	}

	addr := cfg.Get("POS_HTTP_ADDR", ":8080")
	dbPath := cfg.Get("POS_SQLITE_PATH", "data/pos-edge.db")
	migrationsDir := cfg.Get("POS_SQLITE_MIGRATIONS_DIR", "migrations/sqlite")
	backupDir := cfg.Get("POS_SQLITE_BACKUP_DIR", "data/backups")
	archiveDir := cfg.Get("POS_SQLITE_ARCHIVE_DIR", filepath.Join(filepath.Dir(dbPath), "archives"))
	moduleVersion := cfg.Get("MH_POS_VERSION", version.Resolve("MH_POS_VERSION"))
	rawCloudURL := cfg.Get("POS_CLOUD_SYNC_URL", "")
	cloudProvisioningURL := rawCloudURL
	if strings.HasSuffix(cloudProvisioningURL, "/api/v1/sync/edge-events") {
		cloudProvisioningURL = strings.TrimSuffix(cloudProvisioningURL, "/api/v1/sync/edge-events")
	}
	licenseURL := cfg.Get("LICENSE_SERVER_URL", "")
	if licenseURL == "" {
		return errors.New("LICENSE_SERVER_URL is required")
	}
	licenseGate := licensegate.NewClient(licenseURL, cfg.Get("LICENSE_TENANT_ID", "local-tenant"), cfg.Get("LICENSE_SERVER_ID", "edge-local"), time.Duration(cfg.Int("LICENSE_STALE_GRACE_SECONDS", 0))*time.Second)

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
		CloudProvisioningURL:                    cloudProvisioningURL,
		LicenseServerURL:                        licenseURL,
		CloudProvisioningClient:                 posprovisioninghttp.NewCloudClient(10 * time.Second),
		LicenseProvisioningClient:               posprovisioninghttp.NewLicenseClient(10 * time.Second),
		StorageArchiveDir:                       archiveDir,
		RecipeSuggestionMaxPrepTimeDeltaMinutes: cfg.Int("POS_RECIPE_SUGGESTION_MAX_TIME_DELTA_MINUTES", 120),
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
		Handler:           api.NewRouterWithLicense(service, licenseGate),
		ReadHeaderTimeout: 5 * time.Second,
	}

	if cfg.Bool("POS_SYNC_SENDER_ENABLED", true) {
		cloudEndpoint := syncEndpoint(rawCloudURL)
		worker := syncsender.NewWorker(service, poscloudsync.NewClient(cloudEndpoint), syncsender.Config{
			WorkerID:                   cfg.Get("POS_SYNC_SENDER_ID", "pos-sync-sender-main"),
			BatchSize:                  cfg.Int("POS_SYNC_SENDER_BATCH_SIZE", 25),
			PollInterval:               envDuration(cfg.Get("POS_SYNC_SENDER_POLL_INTERVAL", ""), 30*time.Second),
			PollJitter:                 envDuration(cfg.Get("POS_SYNC_SENDER_POLL_JITTER", ""), 3*time.Second),
			CloudPullInterval:          envDuration(cfg.Get("POS_SYNC_SENDER_CLOUD_PULL_INTERVAL", ""), 30*time.Second),
			ReclaimAfter:               envDuration(cfg.Get("POS_SYNC_SENDER_RECLAIM_AFTER", ""), 5*time.Minute),
			SendTimeout:                envDuration(cfg.Get("POS_SYNC_SENDER_SEND_TIMEOUT", ""), 10*time.Second),
			EmergencyPendingThreshold:  cfg.Int("POS_SYNC_SENDER_EMERGENCY_PENDING_THRESHOLD", 100),
			CloudPackageBurstThreshold: cfg.Int("POS_SYNC_SENDER_CLOUD_PACKAGE_BURST_THRESHOLD", 2),
		}, slog.Default())
		go worker.Run(rootCtx)
	} else {
		slog.Info("POS sync sender disabled")
	}

	if cloudProvisioningURL != "" {
		go func() {
			ticker := time.NewTicker(envDuration(cfg.Get("POS_PROVISIONING_POLL_INTERVAL", ""), 3*time.Second))
			defer ticker.Stop()
			for {
				select {
				case <-rootCtx.Done():
					return
				case <-ticker.C:
					_, _ = service.MaintainCloudProvisioning(rootCtx, app.RegisterCloudProvisioningCommand{CloudURL: cloudProvisioningURL, DisplayName: cfg.Get("POS_EDGE_DISPLAY_NAME", "POS Terminal"), AppVersion: moduleVersion})
				}
			}
		}()
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

func syncEndpoint(rawCloudURL string) string {
	if rawCloudURL == "" {
		return "http://localhost:8090/api/v1/sync/edge-events"
	}
	trimmed := strings.TrimRight(rawCloudURL, "/")
	if strings.HasSuffix(trimmed, "/api/v1/sync/edge-events") {
		return trimmed
	}
	return trimmed + "/api/v1/sync/edge-events"
}

func envDuration(raw string, fallback time.Duration) time.Duration {
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
