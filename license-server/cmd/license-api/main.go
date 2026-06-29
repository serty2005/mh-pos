package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "modernc.org/sqlite"

	"license-server/internal/license/api"
	"license-server/internal/license/app"
	"license-server/internal/license/infra/sqlite"
	"mh-pos-platform/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error("license server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load("LICENSE_CONFIG_PATH", "config/license-api.json")
	if err != nil {
		return err
	}
	if cfg.Path() != "" {
		slog.Info("License config file applied", "config_path", cfg.Path())
	}

	addr := cfg.Get("LICENSE_HTTP_ADDR", ":8095")
	dbPath := cfg.Get("LICENSE_SQLITE_PATH", "data/license-server.db")
	backupDir := cfg.Get("LICENSE_SQLITE_BACKUP_DIR", filepath.Join(filepath.Dir(dbPath), "backups"))
	adminLogin := cfg.Get("LICENSE_SUPER_ADMIN_LOGIN", "")
	adminPassword := cfg.Get("LICENSE_SUPER_ADMIN_PASSWORD", "")
	if adminLogin == "" || adminPassword == "" {
		return errors.New("LICENSE_SUPER_ADMIN_LOGIN and LICENSE_SUPER_ADMIN_PASSWORD are required")
	}
	if err := backupSQLiteFiles(dbPath, backupDir); err != nil {
		return err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	repo := sqlite.NewRepository(db)
	if err := repo.Migrate(context.Background()); err != nil {
		return err
	}
	service := app.NewService(repo)
	if err := service.BootstrapSuperAdmin(context.Background(), adminLogin, adminPassword); err != nil {
		return err
	}
	server := &http.Server{Addr: addr, Handler: api.NewRouter(service), ReadHeaderTimeout: 5 * time.Second}
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

func backupSQLiteFiles(dbPath, backupDir string) error {
	info, err := os.Stat(dbPath)
	if errors.Is(err, os.ErrNotExist) || (err == nil && info.Size() == 0) {
		return nil
	}
	if err != nil {
		return err
	}
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	targetDir := filepath.Join(backupDir, stamp)
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return err
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		source := dbPath + suffix
		in, openErr := os.Open(source)
		if errors.Is(openErr, os.ErrNotExist) {
			continue
		}
		if openErr != nil {
			return openErr
		}
		out, createErr := os.OpenFile(filepath.Join(targetDir, filepath.Base(source)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if createErr != nil {
			_ = in.Close()
			return createErr
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		_ = in.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	slog.Info("license sqlite backup created", "operation", "db.backup", "result", "success", "backup_dir", targetDir)
	return nil
}
