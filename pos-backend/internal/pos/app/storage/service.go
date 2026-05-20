package storage

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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
const archiveApplyPlanModeOnly = "plan_only"

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

// ArchiveApplyPlanCommand проверяет готовность archive artifact к будущему apply без удаления строк.
type ArchiveApplyPlanCommand struct {
	shared.CommandMeta
	CutoffBusinessDateLocal string `json:"cutoff_business_date_local"`
	ArchivePath             string `json:"archive_path"`
	ManifestPath            string `json:"manifest_path"`
	Mode                    string `json:"mode"`
}

// ArchiveReadPlanCommand проверяет archive artifact без восстановления или изменения runtime rows.
type ArchiveReadPlanCommand struct {
	shared.CommandMeta
	ManifestPath string `json:"manifest_path"`
	ArchivePath  string `json:"archive_path"`
}

// ArchiveLookupCommand ищет один archived check/order и возвращает только immutable preview.
type ArchiveLookupCommand struct {
	shared.CommandMeta
	ManifestPath string `json:"manifest_path"`
	ArchivePath  string `json:"archive_path"`
	CheckID      string `json:"check_id"`
	OrderID      string `json:"order_id"`
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
	result.ResultMode = "dry_run_only"
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
	result.ResultMode = "plan_only"
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
		ResultMode:                  "export_only",
		DestructiveApplySupported:   false,
		RuntimeRowsDeleted:          false,
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

// BuildArchiveApplyPlan проверяет archive artifact и всегда блокирует destructive apply в текущем runtime.
func (s *Service) BuildArchiveApplyPlan(ctx context.Context, cmd ArchiveApplyPlanCommand) (domainstorage.ArchiveApplyPlan, error) {
	if _, err := shared.EnsureOperatorSession(ctx, s.auth, cmd.CommandMeta, string(shared.PermissionSyncView)); err != nil {
		return domainstorage.ArchiveApplyPlan{}, err
	}
	result := domainstorage.ArchiveApplyPlan{
		GeneratedAt:               s.clock.Now(),
		CutoffBusinessDateLocal:   strings.TrimSpace(cmd.CutoffBusinessDateLocal),
		Mode:                      archiveApplyPlanModeOnly,
		ResultMode:                "apply_blocked",
		DestructiveApplySupported: false,
		RuntimeRowsDeleted:        false,
		Blocked:                   true,
		BlockReasons: []string{
			"destructive_apply_not_enabled",
			"runtime_restore_apply_path_missing",
		},
		Protected: domainstorage.ArchivePlanProtectedFlags{
			FinancialLedgerProtected:    true,
			ImmutableSnapshotsProtected: true,
			LocalEventsProtected:        true,
			OutboxProtected:             true,
		},
	}
	mode := strings.TrimSpace(cmd.Mode)
	if mode == "" {
		mode = archiveApplyPlanModeOnly
	}
	if mode != archiveApplyPlanModeOnly {
		result.BlockReasons = appendUnique(result.BlockReasons, "invalid_apply_plan_mode")
		return result, nil
	}
	cutoff, cutoffOK := s.planCutoff(result.CutoffBusinessDateLocal, &result)
	if cutoffOK {
		runtimeScope, err := s.repo.BuildStorageArchiveApplyRuntimeScope(ctx, cutoff)
		if err != nil {
			return domainstorage.ArchiveApplyPlan{}, err
		}
		result.EligibleCounts = runtimeScope.Counts
		result.ActiveOrders = runtimeScope.ActiveOrders
		result.OpenShifts = runtimeScope.OpenShifts
		result.OpenCashSessions = runtimeScope.OpenCashSessions
		result.BlockingOutboxMessages = runtimeScope.BlockingOutboxMessages
		if runtimeScope.BlockingOutboxMessages > 0 {
			result.BlockReasons = appendUnique(result.BlockReasons, "pending_edge_to_cloud_outbox")
		}
		if runtimeScope.ActiveOrders > 0 || runtimeScope.OpenShifts > 0 || runtimeScope.OpenCashSessions > 0 {
			result.BlockReasons = appendUnique(result.BlockReasons, "open_operational_boundary")
		}
	}
	verification, manifest, reasons := s.verifyArchive(cmd.ArchivePath, cmd.ManifestPath)
	result.Verification = verification
	result.ArchiveCounts = verification.Counts
	result.ArchiveSHA256 = verification.ArchiveSHA256
	if manifest != nil {
		result.ArchiveID = manifest.ArchiveID
		result.ArchiveSHA256 = manifest.SHA256
		if cutoffOK && manifest.CutoffBusinessDateLocal != "" && manifest.CutoffBusinessDateLocal != cutoff {
			reasons = appendUnique(reasons, "archive_cutoff_mismatch")
		}
	}
	for _, reason := range reasons {
		result.BlockReasons = appendUnique(result.BlockReasons, reason)
	}
	if cutoffOK && verification.ArchiveExists && !archiveCountsEqual(result.EligibleCounts, verification.Counts) {
		result.BlockReasons = appendUnique(result.BlockReasons, "archive_counts_mismatch")
	}
	return result, nil
}

// BuildArchiveReadPlan проверяет archive manifest/JSONL как non-destructive read preview foundation.
func (s *Service) BuildArchiveReadPlan(ctx context.Context, cmd ArchiveReadPlanCommand) (domainstorage.ArchiveReadPlan, error) {
	if _, err := shared.EnsureOperatorSession(ctx, s.auth, cmd.CommandMeta, string(shared.PermissionSyncView)); err != nil {
		return domainstorage.ArchiveReadPlan{}, err
	}
	verification, manifest, reasons := s.verifyArchive(cmd.ArchivePath, cmd.ManifestPath)
	result := domainstorage.ArchiveReadPlan{
		GeneratedAt:    s.clock.Now(),
		ResultMode:     "read_plan_only",
		Blocked:        len(reasons) > 0,
		BlockReasons:   reasons,
		ArchiveSHA256:  verification.ArchiveSHA256,
		ComputedSHA256: verification.ComputedSHA256,
		Counts:         verification.Counts,
		Verification:   verification,
	}
	if manifest != nil {
		result.ArchiveID = manifest.ArchiveID
		result.CutoffBusinessDateLocal = manifest.CutoffBusinessDateLocal
		result.ArchiveSHA256 = manifest.SHA256
		result.BusinessDateRange = manifest.BusinessDateRange
		result.Tables = manifest.Tables
	}
	return result, nil
}

// LookupArchivePreview ищет archived check/order streaming-способом и не мутирует runtime SQLite.
func (s *Service) LookupArchivePreview(ctx context.Context, cmd ArchiveLookupCommand) (domainstorage.ArchiveLookupPreview, error) {
	if _, err := shared.EnsureOperatorSession(ctx, s.auth, cmd.CommandMeta, string(shared.PermissionSyncView)); err != nil {
		return domainstorage.ArchiveLookupPreview{}, err
	}
	checkID := strings.TrimSpace(cmd.CheckID)
	orderID := strings.TrimSpace(cmd.OrderID)
	if (checkID == "" && orderID == "") || (checkID != "" && orderID != "") {
		return domainstorage.ArchiveLookupPreview{}, fmt.Errorf("%w: exactly one of check_id or order_id is required", domain.ErrInvalid)
	}

	verification, manifest, reasons := s.verifyArchive(cmd.ArchivePath, cmd.ManifestPath)
	result := domainstorage.ArchiveLookupPreview{
		GeneratedAt:  s.clock.Now(),
		ResultMode:   "archive_lookup_preview",
		Blocked:      len(reasons) > 0,
		BlockReasons: reasons,
		Lookup: domainstorage.ArchiveLookupKey{
			CheckID: checkID,
			OrderID: orderID,
		},
		Verification: verification,
	}
	if manifest != nil {
		result.ArchiveID = manifest.ArchiveID
	}
	if len(reasons) > 0 {
		return result, nil
	}

	identity, err := findArchiveLookupIdentity(verification.ArchivePath, checkID, orderID)
	if err != nil {
		result.Blocked = true
		result.BlockReasons = appendUnique(result.BlockReasons, "archive_unreadable")
		return result, nil
	}
	if !identity.found {
		result.Blocked = true
		result.BlockReasons = appendUnique(result.BlockReasons, "archive_record_not_found")
		result.Lookup.Found = false
		return result, nil
	}

	preview, err := collectArchiveLookupPreview(verification.ArchivePath, identity)
	if err != nil {
		result.Blocked = true
		result.BlockReasons = appendUnique(result.BlockReasons, "archive_unreadable")
		return result, nil
	}
	result.Lookup.CheckID = identity.checkID
	result.Lookup.OrderID = identity.orderID
	result.Lookup.Found = true
	result.Check = preview.Check
	result.Precheck = preview.Precheck
	result.RelatedCounts = preview.RelatedCounts
	return result, nil
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

func (s *Service) planCutoff(raw string, result *domainstorage.ArchiveApplyPlan) (string, bool) {
	cutoff := strings.TrimSpace(raw)
	parsed, err := time.Parse("2006-01-02", cutoff)
	if err != nil {
		result.BlockReasons = appendUnique(result.BlockReasons, "invalid_cutoff")
		return cutoff, false
	}
	today, err := time.Parse("2006-01-02", s.clock.Now().Format("2006-01-02"))
	if err != nil {
		result.BlockReasons = appendUnique(result.BlockReasons, "invalid_runtime_clock")
		return cutoff, false
	}
	if parsed.After(today) {
		result.BlockReasons = appendUnique(result.BlockReasons, "future_cutoff")
		return cutoff, false
	}
	return cutoff, true
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

func (s *Service) verifyArchive(archivePath, manifestPath string) (domainstorage.ArchiveVerificationSummary, *domainstorage.ArchiveManifest, []string) {
	verification := domainstorage.ArchiveVerificationSummary{
		ArchivePath:            strings.TrimSpace(archivePath),
		ManifestPath:           strings.TrimSpace(manifestPath),
		SnapshotPayloadPresent: true,
	}
	reasons := []string{}
	if verification.ManifestPath == "" {
		reasons = appendUnique(reasons, "archive_manifest_missing")
		return verification, nil, reasons
	}
	if !s.archivePathAllowed(verification.ManifestPath) {
		reasons = appendUnique(reasons, "archive_manifest_outside_archive_dir")
		reasons = appendUnique(reasons, "archive_path_outside_archive_dir")
		return verification, nil, reasons
	}
	manifestFile, err := os.Open(verification.ManifestPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			reasons = appendUnique(reasons, "archive_manifest_missing")
			return verification, nil, reasons
		}
		reasons = appendUnique(reasons, "archive_manifest_unreadable")
		return verification, nil, reasons
	}
	verification.ManifestExists = true
	var manifest domainstorage.ArchiveManifest
	if err := json.NewDecoder(manifestFile).Decode(&manifest); err != nil {
		_ = manifestFile.Close()
		reasons = appendUnique(reasons, "archive_manifest_invalid")
		return verification, nil, reasons
	}
	if err := manifestFile.Close(); err != nil {
		reasons = appendUnique(reasons, "archive_manifest_unreadable")
		return verification, &manifest, reasons
	}
	verification.ArchiveSHA256 = manifest.SHA256
	verification.ManifestVersionMatched = manifest.Version == archiveExportVersion
	if !verification.ManifestVersionMatched {
		reasons = appendUnique(reasons, "archive_manifest_version_mismatch")
	}
	if verification.ArchivePath == "" {
		verification.ArchivePath = manifest.ArchivePath
	}
	if verification.ArchivePath == "" {
		reasons = appendUnique(reasons, "archive_missing")
		return verification, &manifest, reasons
	}
	if !s.archivePathAllowed(verification.ArchivePath) {
		reasons = appendUnique(reasons, "archive_path_outside_archive_dir")
		return verification, &manifest, reasons
	}
	counts, sha, snapshotPresent, err := verifyArchiveJSONL(verification.ArchivePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			reasons = appendUnique(reasons, "archive_missing")
			return verification, &manifest, reasons
		}
		reasons = appendUnique(reasons, "archive_unreadable")
		return verification, &manifest, reasons
	}
	verification.ArchiveExists = true
	verification.Counts = counts
	verification.ComputedSHA256 = sha
	verification.SHA256Matched = sha == manifest.SHA256
	verification.CountsMatchedManifest = archiveCountsEqual(counts, manifest.Counts)
	verification.SnapshotPayloadPresent = snapshotPresent
	if !verification.SHA256Matched {
		reasons = appendUnique(reasons, "archive_sha_mismatch")
	}
	if !verification.CountsMatchedManifest {
		reasons = appendUnique(reasons, "archive_manifest_counts_mismatch")
	}
	if !verification.SnapshotPayloadPresent {
		reasons = appendUnique(reasons, "archive_snapshot_payload_missing")
	}
	return verification, &manifest, reasons
}

func (s *Service) archivePathAllowed(path string) bool {
	archiveRoot, err := filepath.Abs(s.archiveDir)
	if err != nil {
		return false
	}
	candidate, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(archiveRoot, candidate)
	if err != nil {
		return false
	}
	return rel == "." || (rel != "" && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != "..")
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

func verifyArchiveJSONL(path string) (domainstorage.ArchiveExportCounts, string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return domainstorage.ArchiveExportCounts{}, "", false, err
	}
	defer file.Close()
	hash := sha256.New()
	scanner := bufio.NewScanner(io.TeeReader(file, hash))
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	tableCounts := map[string]int{}
	snapshotPayloadPresent := true
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		var row domainstorage.ArchiveExportRow
		if err := json.Unmarshal(line, &row); err != nil {
			return domainstorage.ArchiveExportCounts{}, "", false, err
		}
		tableCounts[row.Table]++
		if (row.Table == "checks" || row.Table == "prechecks") && !archiveRowHasSnapshotPayload(row.Row) {
			snapshotPayloadPresent = false
		}
	}
	if err := scanner.Err(); err != nil {
		return domainstorage.ArchiveExportCounts{}, "", false, err
	}
	return archiveCountsFromTableCounts(tableCounts), hex.EncodeToString(hash.Sum(nil)), snapshotPayloadPresent, nil
}

func archiveRowHasSnapshotPayload(row map[string]any) bool {
	raw, ok := row["snapshot"]
	if !ok || raw == nil {
		return false
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v) != ""
	default:
		return true
	}
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

func archiveCountsFromTableCounts(tableCounts map[string]int) domainstorage.ArchiveExportCounts {
	var counts domainstorage.ArchiveExportCounts
	for table, n := range tableCounts {
		counts.ArchivedRows += n
		switch table {
		case "orders":
			counts.ClosedOrders += n
		case "order_lines":
			counts.OrderLines += n
		case "order_line_modifiers":
			counts.OrderLineModifiers += n
		case "order_line_discounts":
			counts.OrderLineDiscounts += n
		case "order_surcharges":
			counts.OrderSurcharges += n
		case "prechecks":
			counts.Prechecks += n
		case "precheck_lines":
			counts.PrecheckLines += n
		case "precheck_line_modifiers":
			counts.PrecheckLineModifiers += n
		case "precheck_discounts":
			counts.PrecheckDiscounts += n
		case "precheck_surcharges":
			counts.PrecheckSurcharges += n
		case "precheck_taxes":
			counts.PrecheckTaxes += n
		case "payments":
			counts.Payments += n
		case "payment_attempts":
			counts.PaymentAttempts += n
		case "checks":
			counts.Checks += n
		case "financial_operations":
			counts.FinancialOperations += n
		case "financial_operation_items":
			counts.FinancialOperationItems += n
		case "local_event_log_summary":
			counts.LocalEventReferences += n
		case "pos_sync_outbox_summary":
			counts.OutboxMessageReferences += n
		}
	}
	return counts
}

func archiveCountsEqual(left, right domainstorage.ArchiveExportCounts) bool {
	return left.ClosedOrders == right.ClosedOrders &&
		left.OrderLines == right.OrderLines &&
		left.OrderLineModifiers == right.OrderLineModifiers &&
		left.OrderLineDiscounts == right.OrderLineDiscounts &&
		left.OrderSurcharges == right.OrderSurcharges &&
		left.Prechecks == right.Prechecks &&
		left.PrecheckLines == right.PrecheckLines &&
		left.PrecheckLineModifiers == right.PrecheckLineModifiers &&
		left.PrecheckDiscounts == right.PrecheckDiscounts &&
		left.PrecheckSurcharges == right.PrecheckSurcharges &&
		left.PrecheckTaxes == right.PrecheckTaxes &&
		left.Payments == right.Payments &&
		left.PaymentAttempts == right.PaymentAttempts &&
		left.Checks == right.Checks &&
		left.FinancialOperations == right.FinancialOperations &&
		left.FinancialOperationItems == right.FinancialOperationItems &&
		left.LocalEventReferences == right.LocalEventReferences &&
		left.OutboxMessageReferences == right.OutboxMessageReferences &&
		left.ArchivedRows == right.ArchivedRows
}

func archiveCountsTotalRows(counts domainstorage.ArchiveExportCounts) int {
	return counts.ClosedOrders +
		counts.OrderLines +
		counts.OrderLineModifiers +
		counts.OrderLineDiscounts +
		counts.OrderSurcharges +
		counts.Prechecks +
		counts.PrecheckLines +
		counts.PrecheckLineModifiers +
		counts.PrecheckDiscounts +
		counts.PrecheckSurcharges +
		counts.PrecheckTaxes +
		counts.Payments +
		counts.PaymentAttempts +
		counts.Checks +
		counts.FinancialOperations +
		counts.FinancialOperationItems +
		counts.LocalEventReferences +
		counts.OutboxMessageReferences
}

type archiveLookupIdentity struct {
	found   bool
	checkID string
	orderID string
}

type archiveLookupCollected struct {
	Check         *domainstorage.ArchiveLookupDocument
	Precheck      *domainstorage.ArchiveLookupDocument
	RelatedCounts domainstorage.ArchiveLookupRelatedCounts
}

func findArchiveLookupIdentity(path, checkID, orderID string) (archiveLookupIdentity, error) {
	var identity archiveLookupIdentity
	err := scanArchiveJSONL(path, func(row domainstorage.ArchiveExportRow) (bool, error) {
		if row.Table != "checks" {
			return false, nil
		}
		rowCheckID := archiveStringValue(row.Row, "id")
		rowOrderID := archiveStringValue(row.Row, "order_id")
		if checkID != "" && rowCheckID == checkID {
			identity = archiveLookupIdentity{found: true, checkID: rowCheckID, orderID: rowOrderID}
			return true, nil
		}
		if orderID != "" && rowOrderID == orderID {
			identity = archiveLookupIdentity{found: true, checkID: rowCheckID, orderID: rowOrderID}
			return true, nil
		}
		return false, nil
	})
	return identity, err
}

func collectArchiveLookupPreview(path string, identity archiveLookupIdentity) (archiveLookupCollected, error) {
	var out archiveLookupCollected
	precheckIDs := map[string]struct{}{}
	operationIDs := map[string]struct{}{}
	err := scanArchiveJSONL(path, func(row domainstorage.ArchiveExportRow) (bool, error) {
		switch row.Table {
		case "order_lines":
			if archiveStringValue(row.Row, "order_id") == identity.orderID {
				out.RelatedCounts.OrderLines++
			}
		case "prechecks":
			if archiveStringValue(row.Row, "order_id") == identity.orderID {
				precheckID := archiveStringValue(row.Row, "id")
				precheckIDs[precheckID] = struct{}{}
				out.Precheck = &domainstorage.ArchiveLookupDocument{
					ID:       precheckID,
					Snapshot: archiveSnapshotValue(row.Row),
				}
			}
		case "payments":
			if _, ok := precheckIDs[archiveStringValue(row.Row, "precheck_id")]; ok {
				out.RelatedCounts.Payments++
			}
		case "checks":
			if archiveStringValue(row.Row, "id") == identity.checkID {
				out.Check = &domainstorage.ArchiveLookupDocument{
					ID:                identity.checkID,
					BusinessDateLocal: archiveStringValue(row.Row, "business_date_local"),
					Snapshot:          archiveSnapshotValue(row.Row),
				}
			}
		case "financial_operations":
			if archiveStringValue(row.Row, "check_id") == identity.checkID {
				out.RelatedCounts.FinancialOperations++
				operationIDs[archiveStringValue(row.Row, "id")] = struct{}{}
			}
		case "financial_operation_items":
			if _, ok := operationIDs[archiveStringValue(row.Row, "operation_id")]; ok {
				out.RelatedCounts.FinancialOperationItems++
			}
		}
		return false, nil
	})
	return out, err
}

func scanArchiveJSONL(path string, visit func(domainstorage.ArchiveExportRow) (bool, error)) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		var row domainstorage.ArchiveExportRow
		if err := json.Unmarshal(line, &row); err != nil {
			return err
		}
		stop, err := visit(row)
		if err != nil {
			return err
		}
		if stop {
			return nil
		}
	}
	return scanner.Err()
}

func archiveStringValue(row map[string]any, key string) string {
	raw, ok := row[key]
	if !ok || raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return v.String()
	default:
		return fmt.Sprint(v)
	}
}

func archiveSnapshotValue(row map[string]any) map[string]any {
	raw := row["snapshot"]
	switch v := raw.(type) {
	case map[string]any:
		return v
	case string:
		var parsed map[string]any
		if err := json.Unmarshal([]byte(v), &parsed); err == nil && parsed != nil {
			return parsed
		}
	}
	return map[string]any{}
}

func appendUnique(values []string, next string) []string {
	for _, value := range values {
		if value == next {
			return values
		}
	}
	return append(values, next)
}
