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
	inventoryapp "cloud-backend/internal/inventory/app"
	inventorypg "cloud-backend/internal/inventory/infra/postgres"
	masterapp "cloud-backend/internal/masterdata/app"
	masterpg "cloud-backend/internal/masterdata/infra/postgres"
	olapapp "cloud-backend/internal/olap/app"
	olapch "cloud-backend/internal/olap/infra/clickhouse"
	olappg "cloud-backend/internal/olap/infra/postgres"
	"cloud-backend/internal/platform/clock"
	"cloud-backend/internal/platform/idgen"
	"cloud-backend/internal/platform/logging"
	platformpg "cloud-backend/internal/platform/postgres"
	"cloud-backend/internal/platform/version"
	provisioningapp "cloud-backend/internal/provisioning/app"
	"cloud-backend/internal/provisioning/infra/licensehttp"
	provisioningpg "cloud-backend/internal/provisioning/infra/postgres"
	"mh-pos-platform/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error("cloud backend stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load("CLOUD_CONFIG_PATH", "config/cloud-api.json")
	if err != nil {
		return err
	}
	slog.SetDefault(logging.NewJSONLoggerWithLevel(cfg.Get("CLOUD_LOG_LEVEL", "")))
	if cfg.Path() != "" {
		slog.Info("Cloud config file applied", "config_path", cfg.Path())
	}

	addr := cfg.Get("CLOUD_HTTP_ADDR", ":8090")
	publicURL := cfg.Get("CLOUD_PUBLIC_URL", "http://localhost:8090")
	licenseURL := cfg.Get("LICENSE_SERVER_URL", "")
	dsn := cfg.Get("CLOUD_POSTGRES_DSN", "")
	migrationsDir := cfg.Get("CLOUD_POSTGRES_MIGRATIONS_DIR", "migrations/postgres")
	backupDir := cfg.Get("CLOUD_POSTGRES_BACKUP_DIR", "data/cloud-backups")
	moduleVersion := cfg.Get("MH_POS_VERSION", version.Resolve("MH_POS_VERSION"))
	clickHouseURL := cfg.Get("CLOUD_CLICKHOUSE_URL", "")
	clickHouseDatabase := cfg.Get("CLOUD_CLICKHOUSE_DATABASE", "mh_pos_cloud")
	clickHouseUser := cfg.Get("CLOUD_CLICKHOUSE_USER", "")
	clickHousePassword := cfg.Get("CLOUD_CLICKHOUSE_PASSWORD", "")
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

	var olapService *olapapp.Service
	var olapForwarder *olapapp.Forwarder
	var olapStockMoveForwarder *olapapp.StockMoveForwarder
	if clickHouseURL != "" {
		clickHouseRepo := olapch.NewRepository(olapch.Config{
			URL:      clickHouseURL,
			Database: clickHouseDatabase,
			Username: clickHouseUser,
			Password: clickHousePassword,
		})
		if err := clickHouseRepo.Migrate(ctx); err != nil {
			return err
		}
		olapService = olapapp.NewService(clickHouseRepo)
		olapForwarder = olapapp.NewForwarder(olappg.NewRepository(pool), clickHouseRepo, clock.SystemClock{}, olapapp.ForwarderConfig{
			WorkerID:      cfg.Get("CLOUD_OLAP_FORWARDER_ID", "cloud-olap-forwarder"),
			BatchSize:     cfg.Int("CLOUD_OLAP_FORWARDER_BATCH_SIZE", 1000),
			RetryDelay:    time.Duration(cfg.Int("CLOUD_OLAP_FORWARDER_RETRY_SECONDS", 60)) * time.Second,
			ProcessingTTL: time.Duration(cfg.Int("CLOUD_OLAP_FORWARDER_PROCESSING_TTL_SECONDS", 300)) * time.Second,
		})
		olapStockMoveForwarder = olapapp.NewStockMoveForwarder(olappg.NewRepository(pool), clickHouseRepo, clock.SystemClock{}, olapapp.ForwarderConfig{
			WorkerID:      cfg.Get("CLOUD_OLAP_STOCK_MOVES_FORWARDER_ID", "cloud-olap-stock-moves-forwarder"),
			BatchSize:     cfg.Int("CLOUD_OLAP_STOCK_MOVES_FORWARDER_BATCH_SIZE", 1000),
			RetryDelay:    time.Duration(cfg.Int("CLOUD_OLAP_STOCK_MOVES_FORWARDER_RETRY_SECONDS", 60)) * time.Second,
			ProcessingTTL: time.Duration(cfg.Int("CLOUD_OLAP_STOCK_MOVES_FORWARDER_PROCESSING_TTL_SECONDS", 300)) * time.Second,
		})
	}

	repo := syncpg.NewRepository(pool)
	service := app.NewServiceWithOptions(repo, clock.SystemClock{}, app.Options{
		MaxCloudPackagesPerExchange: cfg.Int("CLOUD_SYNC_MAX_CLOUD_PACKAGES_PER_EXCHANGE", 3),
	})
	inventoryWorker := inventoryapp.NewWorker(inventorypg.NewRepository(pool), idgen.UUIDGenerator{}, clock.SystemClock{}, inventoryapp.Config{
		WorkerID:  cfg.Get("CLOUD_INVENTORY_WORKER_ID", "cloud-inventory-worker"),
		BatchSize: 25,
	})
	masterRepo := masterpg.NewRepository(pool)
	masterService := masterapp.NewService(masterRepo, clock.SystemClock{}, masterapp.RandomIDGenerator{})
	var licenseClient provisioningapp.LicenseClient
	if licenseURL != "" {
		licenseClient = licensehttp.NewClient(licenseURL)
	}
	provisioningService := provisioningapp.NewService(provisioningpg.NewRepository(pool), masterService, clock.SystemClock{}, masterapp.RandomIDGenerator{}, publicURL, licenseClient)
	server := &http.Server{
		Addr:              addr,
		Handler:           api.NewRouterWithProvisioningAndOLAP(service, provisioningService, olapService, masterService),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("Cloud Backend listening", "addr", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server failed", "error", err)
		}
	}()
	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()
	go runInventoryWorker(workerCtx, inventoryWorker, 2*time.Second)
	if olapForwarder != nil {
		go runOLAPForwarder(workerCtx, olapForwarder, time.Duration(cfg.Int("CLOUD_OLAP_FORWARDER_INTERVAL_SECONDS", 5))*time.Second)
	}
	if olapStockMoveForwarder != nil {
		go runOLAPStockMoveForwarder(workerCtx, olapStockMoveForwarder, time.Duration(cfg.Int("CLOUD_OLAP_STOCK_MOVES_FORWARDER_INTERVAL_SECONDS", 5))*time.Second)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

func runInventoryWorker(ctx context.Context, worker *inventoryapp.Worker, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := worker.RunOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("cloud inventory worker failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func runOLAPForwarder(ctx context.Context, worker *olapapp.Forwarder, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := worker.RunOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("cloud olap forwarder failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func runOLAPStockMoveForwarder(ctx context.Context, worker *olapapp.StockMoveForwarder, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := worker.RunOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("cloud olap stock moves forwarder failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
