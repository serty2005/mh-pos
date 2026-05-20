package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
	domainstorage "pos-backend/internal/pos/domain/storage"
	"pos-backend/internal/pos/ports"
)

const retentionDryRunOnlyReason = "physical archive/delete is not implemented; financial ledger, immutable snapshots and audit/sync rows remain protected"
const archiveExportVersion = "pos_storage_archive_export_v1"
const archivePlanModeManifestOnly = "manifest_only"

// Service предоставляет read-only lifecycle surface для локальной POS Edge БД.
type Service struct {
	repo       ports.StorageLifecycleRepository
	auth       ports.Repository
	ids        idgen.Generator
	clock      clock.Clock
	archiveDir string
}

// StorageStatusCommand несет operator metadata для чтения lifecycle status.
type StorageStatusCommand struct {
	shared.CommandMeta
}

// RetentionDryRunCommand описывает безопасную оценку retention cutoff без записи данных.
type RetentionDryRunCommand struct {
	shared.CommandMeta
	CutoffBusinessDateLocal string `json:"cutoff_business_date_local"`
}

// ArchiveExportPlanCommand строит manifest-only план будущего archive/export без записи файлов.
type ArchiveExportPlanCommand struct {
	shared.CommandMeta
	CutoffBusinessDateLocal string `json:"cutoff_business_date_local"`
	Mode                    string `json:"mode"`
}

// ArchiveExportCommand описывает export-only создание архивного артефакта без удаления данных.
type ArchiveExportCommand struct {
	shared.CommandMeta
	CutoffBusinessDateLocal string `json:"cutoff_business_date_local"`
	Reason                  string `json:"reason"`
}

// Options задает runtime настройки storage lifecycle service.
type Options struct {
	ArchiveDir string
}

// NewService создает storage lifecycle service поверх общего POS repository.
func NewService(repo ports.Repository, ids idgen.Generator, clock clock.Clock, options Options) *Service {
	archiveDir := strings.TrimSpace(options.ArchiveDir)
	if archiveDir == "" {
		archiveDir = filepath.Join(os.TempDir(), "mh-pos-storage-archives")
	}
	return &Service{repo: repo, auth: repo, ids: ids, clock: clock, archiveDir: archiveDir}
}

// GetStatus возвращает текущую оценку локального storage только для оператора с sync-view permission.
func (s *Service) GetStatus(ctx context.Context, cmd StorageStatusCommand) (domainstorage.LifecycleStatus, error) {
	if _, err := shared.EnsureOperatorSession(ctx, s.auth, cmd.CommandMeta, string(shared.PermissionSyncView)); err != nil {
		return domainstorage.LifecycleStatus{}, err
	}
	status, err := s.repo.GetStorageLifecycleStatus(ctx)
	if err != nil {
		return domainstorage.LifecycleStatus{}, err
	}
	status.GeneratedAt = s.clock.Now()
	status.Retention = retentionCapability()
	return status, nil
}

// DryRunRetention считает потенциальный retention scope и намеренно не выполняет destructive apply.
func (s *Service) DryRunRetention(ctx context.Context, cmd RetentionDryRunCommand) (domainstorage.RetentionDryRunResult, error) {
	if _, err := shared.EnsureOperatorSession(ctx, s.auth, cmd.CommandMeta, string(shared.PermissionSyncView)); err != nil {
		return domainstorage.RetentionDryRunResult{}, err
	}
	cutoff := strings.TrimSpace(cmd.CutoffBusinessDateLocal)
	if _, err := time.Parse("2006-01-02", cutoff); err != nil {
		return domainstorage.RetentionDryRunResult{}, fmt.Errorf("%w: cutoff_business_date_local must use YYYY-MM-DD", domain.ErrInvalid)
	}
	result, err := s.repo.DryRunStorageRetention(ctx, cutoff)
	if err != nil {
		return domainstorage.RetentionDryRunResult{}, err
	}
	result.GeneratedAt = s.clock.Now()
	result.CutoffBusinessDateLocal = cutoff
	result.Mode = "dry_run_only"
	result.DestructiveApplySupported = false
	result.FinancialLedgerProtected = true
	result.ImmutableSnapshotsProtected = true
	result.BlockReasons = appendUnique(result.BlockReasons, "dry_run_only_no_archive_policy")
	result.Blocked = true
	return result, nil
}

// BuildArchiveExportPlan возвращает deterministic manifest-only archive scope без мутации runtime rows.
func (s *Service) BuildArchiveExportPlan(ctx context.Context, cmd ArchiveExportPlanCommand) (domainstorage.ArchiveExportPlan, error) {
	if _, err := shared.EnsureOperatorSession(ctx, s.auth, cmd.CommandMeta, string(shared.PermissionSyncView)); err != nil {
		return domainstorage.ArchiveExportPlan{}, err
	}
	cutoff := strings.TrimSpace(cmd.CutoffBusinessDateLocal)
	if _, err := time.Parse("2006-01-02", cutoff); err != nil {
		return domainstorage.ArchiveExportPlan{}, fmt.Errorf("%w: cutoff_business_date_local must use YYYY-MM-DD", domain.ErrInvalid)
	}
	mode := strings.TrimSpace(cmd.Mode)
	if mode == "" {
		mode = archivePlanModeManifestOnly
	}
	if mode != archivePlanModeManifestOnly {
		return domainstorage.ArchiveExportPlan{}, fmt.Errorf("%w: storage archive export-plan supports manifest_only mode only", domain.ErrInvalid)
	}
	result, err := s.repo.BuildStorageArchiveExportPlan(ctx, cutoff)
	if err != nil {
		return domainstorage.ArchiveExportPlan{}, err
	}
	result.GeneratedAt = s.clock.Now()
	result.CutoffBusinessDateLocal = cutoff
	result.Mode = archivePlanModeManifestOnly
	result.DestructiveApplySupported = false
	result.Blocked = true
	return result, nil
}

// ExportArchive создает typed JSONL archive и JSON manifest для старых closed orders без изменения source DB rows.
func (s *Service) ExportArchive(ctx context.Context, cmd ArchiveExportCommand) (domainstorage.ArchiveExportResult, error) {
	if _, err := shared.EnsureOperatorSession(ctx, s.auth, cmd.CommandMeta, string(shared.PermissionSyncView)); err != nil {
		return domainstorage.ArchiveExportResult{}, err
	}
	cutoff, err := s.validateCutoff(cmd.CutoffBusinessDateLocal)
	if err != nil {
		return domainstorage.ArchiveExportResult{}, err
	}
	reason := strings.TrimSpace(cmd.Reason)
	scope, err := s.repo.BuildStorageArchiveExportScope(ctx, cutoff)
	if err != nil {
		return domainstorage.ArchiveExportResult{}, err
	}

	generatedAt := s.clock.Now()
	archiveID := s.ids.NewID()
	archiveRoot, err := filepath.Abs(s.archiveDir)
	if err != nil {
		return domainstorage.ArchiveExportResult{}, fmt.Errorf("storage archive export path: %w", err)
	}
	exportDir := filepath.Join(archiveRoot, archiveID)
	if err := os.MkdirAll(exportDir, 0o750); err != nil {
		return domainstorage.ArchiveExportResult{}, fmt.Errorf("storage archive export mkdir: %w", err)
	}
	archivePath := filepath.Join(exportDir, "archive.jsonl")
	manifestPath := filepath.Join(exportDir, "manifest.json")

	sha, tableCounts, err := writeArchiveJSONL(archivePath, scope.Rows)
	if err != nil {
		return domainstorage.ArchiveExportResult{}, err
	}
	manifest := domainstorage.ArchiveManifest{
		Version:                     archiveExportVersion,
		Mode:                        "export_only",
		DestructiveApplySupported:   false,
		GeneratedAt:                 generatedAt,
		ArchiveID:                   archiveID,
		CutoffBusinessDateLocal:     cutoff,
		Reason:                      reason,
		ArchivePath:                 archivePath,
		ManifestPath:                manifestPath,
		SHA256:                      sha,
		Counts:                      scope.Counts,
		BusinessDateRange:           scope.BusinessDateRange,
		Tables:                      archiveTableManifest(tableCounts),
		Source:                      scope.Source,
		Blocked:                     true,
		BlockReasons:                scope.BlockReasons,
		FinancialLedgerProtected:    true,
		ImmutableSnapshotsProtected: true,
		ExportCreated:               true,
	}
	if err := writeManifest(manifestPath, manifest); err != nil {
		return domainstorage.ArchiveExportResult{}, err
	}
	return manifest, nil
}

func retentionCapability() domainstorage.RetentionCapability {
	return domainstorage.RetentionCapability{
		Mode:                        "dry_run_only",
		DestructiveApplySupported:   false,
		FinancialLedgerProtected:    true,
		ImmutableSnapshotsProtected: true,
		Reason:                      retentionDryRunOnlyReason,
	}
}

func (s *Service) validateCutoff(raw string) (string, error) {
	cutoff := strings.TrimSpace(raw)
	parsed, err := time.Parse("2006-01-02", cutoff)
	if err != nil {
		return "", fmt.Errorf("%w: cutoff_business_date_local must use YYYY-MM-DD", domain.ErrInvalid)
	}
	today, err := time.Parse("2006-01-02", s.clock.Now().Format("2006-01-02"))
	if err != nil {
		return "", fmt.Errorf("storage archive clock date: %w", err)
	}
	if parsed.After(today) {
		return "", fmt.Errorf("%w: cutoff_business_date_local must not be in the future", domain.ErrInvalid)
	}
	return cutoff, nil
}

func writeArchiveJSONL(path string, rows []domainstorage.ArchiveExportRow) (string, map[string]int, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return "", nil, fmt.Errorf("storage archive write: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	writer := io.MultiWriter(file, hash)
	encoder := json.NewEncoder(writer)
	tableCounts := map[string]int{}
	for _, row := range rows {
		if err := encoder.Encode(row); err != nil {
			return "", nil, fmt.Errorf("storage archive encode: %w", err)
		}
		tableCounts[row.Table]++
	}
	return hex.EncodeToString(hash.Sum(nil)), tableCounts, nil
}

func writeManifest(path string, manifest domainstorage.ArchiveManifest) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return fmt.Errorf("storage archive manifest write: %w", err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("storage archive manifest encode: %w", err)
	}
	return nil
}

func archiveTableManifest(counts map[string]int) []domainstorage.ArchiveTableManifest {
	names := []string{
		"orders",
		"order_lines",
		"order_line_modifiers",
		"order_line_discounts",
		"order_surcharges",
		"prechecks",
		"precheck_lines",
		"precheck_line_modifiers",
		"precheck_discounts",
		"precheck_surcharges",
		"precheck_taxes",
		"payments",
		"payment_attempts",
		"checks",
		"financial_operations",
		"financial_operation_items",
		"local_event_log_summary",
		"pos_sync_outbox_summary",
	}
	out := make([]domainstorage.ArchiveTableManifest, 0, len(names))
	for _, name := range names {
		rows := counts[name]
		content := "rows"
		payloadIncluded := true
		if name == "local_event_log_summary" || name == "pos_sync_outbox_summary" {
			content = "summary_without_payload"
			payloadIncluded = false
		}
		out = append(out, domainstorage.ArchiveTableManifest{Name: name, Rows: rows, PayloadIncluded: payloadIncluded, Content: content})
	}
	return out
}

func appendUnique(values []string, next string) []string {
	for _, value := range values {
		if value == next {
			return values
		}
	}
	return append(values, next)
}
