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

const retentionApplySupportedReason = "destructive apply requires a verified JSONL archive, scoped sent outbox and no open operational boundaries for the cutoff period"
const archiveExportVersion = "pos_storage_archive_export_v1"
const archivePlanModeManifestOnly = "manifest_only"
const archiveApplyPlanModeOnly = "plan_only"
const archiveReadPlanDefaultLimit = 50
const archiveReadPlanMaxLimit = 100

var errArchiveJSONLMalformed = errors.New("archive jsonl malformed")

// Service предоставляет lifecycle surface для локальной POS Edge БД.
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

// ArchiveApplyPlanCommand применяет verified archive artifact к active SQLite, если policy gate открыт.
type ArchiveApplyPlanCommand struct {
	shared.CommandMeta
	ArchiveID               string `json:"archive_id"`
	CutoffBusinessDateLocal string `json:"cutoff_business_date_local"`
	ArchivePath             string `json:"archive_path"`
	ManifestPath            string `json:"manifest_path"`
	Mode                    string `json:"mode"`
}

// ArchiveApplyReadinessCommand агрегирует archive/runtime readiness без destructive apply.
type ArchiveApplyReadinessCommand struct {
	shared.CommandMeta
	ArchiveID               string `json:"archive_id"`
	CutoffBusinessDateLocal string `json:"cutoff_business_date_local"`
	ArchivePath             string `json:"archive_path"`
	ManifestPath            string `json:"manifest_path"`
	Mode                    string `json:"mode"`
}

// ArchiveVerifyCommand явно проверяет ранее экспортированный archive artifact без изменения runtime rows.
type ArchiveVerifyCommand struct {
	shared.CommandMeta
	ArchiveID    string `json:"archive_id"`
	ManifestPath string `json:"manifest_path"`
	ArchivePath  string `json:"archive_path"`
}

// ArchiveReadPlanCommand проверяет archive artifact без восстановления или изменения runtime rows.
type ArchiveReadPlanCommand struct {
	shared.CommandMeta
	ArchiveID         string `json:"archive_id"`
	ManifestPath      string `json:"manifest_path"`
	ArchivePath       string `json:"archive_path"`
	BusinessDateLocal string `json:"business_date_local"`
	OrderID           string `json:"order_id"`
	CheckID           string `json:"check_id"`
	Limit             int    `json:"limit"`
	Offset            int    `json:"offset"`
}

// ArchiveLookupCommand ищет один archived check/order и возвращает только immutable preview.
type ArchiveLookupCommand struct {
	shared.CommandMeta
	ArchiveID    string `json:"archive_id"`
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
	cutoff, err := s.validateCutoff(cmd.CutoffBusinessDateLocal)
	if err != nil {
		return domainstorage.RetentionDryRunResult{}, err
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
	cutoff, err := s.validateCutoff(cmd.CutoffBusinessDateLocal)
	if err != nil {
		return domainstorage.ArchiveExportPlan{}, err
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
		DestructiveApplySupported:   true,
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
		Blocked:                     scope.Blocked,
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

// BuildArchiveApplyPlan проверяет archive artifact и выполняет destructive apply, если readiness gate открыт.
func (s *Service) BuildArchiveApplyPlan(ctx context.Context, cmd ArchiveApplyPlanCommand) (domainstorage.ArchiveApplyPlan, error) {
	if _, err := shared.EnsureOperatorSession(ctx, s.auth, cmd.CommandMeta, string(shared.PermissionSyncView)); err != nil {
		return domainstorage.ArchiveApplyPlan{}, err
	}
	result, cutoff, ready, err := s.evaluateArchiveApplyPlan(ctx, cmd)
	if err != nil {
		return domainstorage.ArchiveApplyPlan{}, err
	}
	if !ready {
		return result, nil
	}
	deleted, err := s.repo.ApplyStorageArchiveDestructive(ctx, cutoff)
	if err != nil {
		return domainstorage.ArchiveApplyPlan{}, fmt.Errorf("storage archive destructive apply: %w", err)
	}
	result.GeneratedAt = s.clock.Now()
	result.ResultMode = "destructive_apply"
	result.RuntimeRowsDeleted = true
	result.Blocked = false
	result.BlockReasons = nil
	result.EligibleCounts = deleted
	return result, nil
}

func (s *Service) evaluateArchiveApplyPlan(ctx context.Context, cmd ArchiveApplyPlanCommand) (domainstorage.ArchiveApplyPlan, string, bool, error) {
	result := domainstorage.ArchiveApplyPlan{
		GeneratedAt:               s.clock.Now(),
		CutoffBusinessDateLocal:   strings.TrimSpace(cmd.CutoffBusinessDateLocal),
		Mode:                      archiveApplyPlanModeOnly,
		ResultMode:                "apply_blocked",
		DestructiveApplySupported: true,
		RuntimeRowsDeleted:        false,
		Blocked:                   true,
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
		return result, "", false, nil
	}
	cutoff, cutoffOK := s.planCutoff(result.CutoffBusinessDateLocal, &result)
	if cutoffOK {
		runtimeScope, err := s.repo.BuildStorageArchiveApplyRuntimeScope(ctx, cutoff)
		if err != nil {
			return domainstorage.ArchiveApplyPlan{}, "", false, err
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
	archivePath, manifestPath := s.archivePaths(cmd.ArchiveID, cmd.ArchivePath, cmd.ManifestPath)
	verification, manifest, reasons := s.verifyArchive(archivePath, manifestPath)
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
	result.Blocked = len(result.BlockReasons) > 0
	ready := cutoffOK && !result.Blocked && archiveApplyVerificationReady(verification)
	return result, cutoff, ready, nil
}

// BuildArchiveApplyReadiness возвращает отдельный read-only policy verdict для destructive apply.
func (s *Service) BuildArchiveApplyReadiness(ctx context.Context, cmd ArchiveApplyReadinessCommand) (domainstorage.ArchiveApplyReadiness, error) {
	if _, err := shared.EnsureOperatorSession(ctx, s.auth, cmd.CommandMeta, string(shared.PermissionSyncView)); err != nil {
		return domainstorage.ArchiveApplyReadiness{}, err
	}
	plan, _, _, err := s.evaluateArchiveApplyPlan(ctx, ArchiveApplyPlanCommand{
		ArchiveID:               cmd.ArchiveID,
		CutoffBusinessDateLocal: cmd.CutoffBusinessDateLocal,
		ArchivePath:             cmd.ArchivePath,
		ManifestPath:            cmd.ManifestPath,
		Mode:                    cmd.Mode,
	})
	if err != nil {
		return domainstorage.ArchiveApplyReadiness{}, err
	}
	return archiveApplyReadinessFromPlan(plan), nil
}

// VerifyArchive проверяет manifest/JSONL artifact и не обращается к active runtime tables.
func (s *Service) VerifyArchive(ctx context.Context, cmd ArchiveVerifyCommand) (domainstorage.ArchiveVerifyResult, error) {
	if _, err := shared.EnsureOperatorSession(ctx, s.auth, cmd.CommandMeta, string(shared.PermissionSyncView)); err != nil {
		return domainstorage.ArchiveVerifyResult{}, err
	}
	archivePath, manifestPath := s.archivePaths(cmd.ArchiveID, cmd.ArchivePath, cmd.ManifestPath)
	verification, manifest, reasons := s.verifyArchive(archivePath, manifestPath)
	result := domainstorage.ArchiveVerifyResult{
		GeneratedAt:       s.clock.Now(),
		Valid:             len(reasons) == 0,
		Errors:            reasons,
		ArchivePath:       verification.ArchivePath,
		ManifestPath:      verification.ManifestPath,
		Counts:            verification.Counts,
		Verification:      verification,
		BusinessDateRange: verification.BusinessDateRange,
	}
	if manifest != nil {
		result.ArchiveID = manifest.ArchiveID
		result.CutoffBusinessDateLocal = manifest.CutoffBusinessDateLocal
		result.ArchivePath = verification.ArchivePath
		result.ManifestPath = verification.ManifestPath
		result.RuntimeRowsDeleted = manifest.RuntimeRowsDeleted
		result.DestructiveApplySupported = manifest.DestructiveApplySupported
		result.BusinessDateRange = manifest.BusinessDateRange
		result.Tables = manifest.Tables
	}
	return result, nil
}

// BuildArchiveReadPlan проверяет archive manifest/JSONL как non-destructive read preview foundation.
func (s *Service) BuildArchiveReadPlan(ctx context.Context, cmd ArchiveReadPlanCommand) (domainstorage.ArchiveReadPlan, error) {
	if _, err := shared.EnsureOperatorSession(ctx, s.auth, cmd.CommandMeta, string(shared.PermissionSyncView)); err != nil {
		return domainstorage.ArchiveReadPlan{}, err
	}
	limit, offset, err := normalizeArchiveReadPlanBounds(cmd.Limit, cmd.Offset)
	if err != nil {
		return domainstorage.ArchiveReadPlan{}, err
	}
	businessDate := strings.TrimSpace(cmd.BusinessDateLocal)
	if businessDate != "" {
		if _, err := time.Parse("2006-01-02", businessDate); err != nil {
			return domainstorage.ArchiveReadPlan{}, fmt.Errorf("%w: business_date_local must use YYYY-MM-DD", domain.ErrInvalid)
		}
	}
	archivePath, manifestPath := s.archivePaths(cmd.ArchiveID, cmd.ArchivePath, cmd.ManifestPath)
	verification, manifest, reasons := s.verifyArchive(archivePath, manifestPath)
	result := domainstorage.ArchiveReadPlan{
		GeneratedAt:        s.clock.Now(),
		ResultMode:         "read_plan_only",
		Blocked:            len(reasons) > 0,
		BlockReasons:       reasons,
		ArchiveSHA256:      verification.ArchiveSHA256,
		ComputedSHA256:     verification.ComputedSHA256,
		Counts:             verification.Counts,
		Verification:       verification,
		Limit:              limit,
		Offset:             offset,
		RuntimeRowsDeleted: false,
		RuntimeRestored:    false,
		PayloadPolicy:      "archive_preview_without_sync_payloads",
	}
	if manifest != nil {
		result.ArchiveID = manifest.ArchiveID
		result.CutoffBusinessDateLocal = manifest.CutoffBusinessDateLocal
		result.ArchiveSHA256 = manifest.SHA256
		result.BusinessDateRange = manifest.BusinessDateRange
		result.Tables = manifest.Tables
		result.RuntimeRowsDeleted = manifest.RuntimeRowsDeleted
	}
	if len(reasons) == 0 {
		preview, err := collectArchiveReadPlanPreview(verification.ArchivePath, archiveReadPlanFilter{
			businessDateLocal: businessDate,
			orderID:           strings.TrimSpace(cmd.OrderID),
			checkID:           strings.TrimSpace(cmd.CheckID),
			limit:             limit,
			offset:            offset,
		})
		if err != nil {
			result.Blocked = true
			result.BlockReasons = appendUnique(result.BlockReasons, "archive_unreadable")
			return result, nil
		}
		result.ArchivedClosedOrders = preview
		result.Returned = len(preview)
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

	archivePath, manifestPath := s.archivePaths(cmd.ArchiveID, cmd.ArchivePath, cmd.ManifestPath)
	verification, manifest, reasons := s.verifyArchive(archivePath, manifestPath)
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
		Mode:                        "archive_apply_supported",
		DestructiveApplySupported:   true,
		FinancialLedgerProtected:    true,
		ImmutableSnapshotsProtected: true,
		Reason:                      retentionApplySupportedReason,
	}
}

func archiveApplyVerificationReady(verification domainstorage.ArchiveVerificationSummary) bool {
	return verification.ArchiveExists &&
		verification.ManifestExists &&
		verification.ManifestVersionMatched &&
		verification.SHA256Matched &&
		verification.CountsMatchedManifest &&
		verification.IdentityFieldsPresent &&
		verification.BusinessDateConsistent &&
		verification.RuntimeRowsNotDeleted &&
		verification.PayloadPolicyPreserved &&
		verification.SnapshotPayloadPresent
}

func archiveApplyReadinessFromPlan(plan domainstorage.ArchiveApplyPlan) domainstorage.ArchiveApplyReadiness {
	verification := plan.Verification
	openBoundaries := domainstorage.ArchiveOpenOperationalBoundaries{
		ActiveOrders:     plan.ActiveOrders,
		OpenShifts:       plan.OpenShifts,
		OpenCashSessions: plan.OpenCashSessions,
	}
	openBoundaries.Open = openBoundaries.ActiveOrders > 0 || openBoundaries.OpenShifts > 0 || openBoundaries.OpenCashSessions > 0
	runtimeScopeVerified := !containsString(plan.BlockReasons, "invalid_cutoff") &&
		!containsString(plan.BlockReasons, "future_cutoff") &&
		!containsString(plan.BlockReasons, "archive_counts_mismatch") &&
		!containsString(plan.BlockReasons, "pending_edge_to_cloud_outbox") &&
		!containsString(plan.BlockReasons, "open_operational_boundary")
	archiveVerified := archiveApplyVerificationReady(verification)
	manifestVerified := verification.ManifestExists && verification.ManifestVersionMatched
	snapshotVerified := verification.SnapshotPayloadPresent
	ready := plan.DestructiveApplySupported &&
		archiveVerified &&
		manifestVerified &&
		snapshotVerified &&
		runtimeScopeVerified &&
		len(plan.BlockReasons) == 0
	humanSummary := "destructive archive apply is blocked until archive integrity and runtime safety checks pass"
	if ready {
		humanSummary = "archive is verified and runtime scope is ready for destructive apply"
	}
	readiness := domainstorage.ArchiveApplyReadiness{
		GeneratedAt:               plan.GeneratedAt,
		CutoffBusinessDateLocal:   plan.CutoffBusinessDateLocal,
		ArchiveID:                 plan.ArchiveID,
		ArchiveSHA256:             plan.ArchiveSHA256,
		ResultMode:                "apply_readiness_only",
		DestructiveApplySupported: plan.DestructiveApplySupported,
		ReadyForDestructiveApply:  ready,
		RuntimeRowsDeleted:        false,
		ArchiveVerified:           archiveVerified,
		ManifestVerified:          manifestVerified,
		SnapshotPayloadVerified:   snapshotVerified,
		RuntimeScopeVerified:      runtimeScopeVerified,
		BlockingOutboxCount:       plan.BlockingOutboxMessages,
		PendingEdgeToCloudOutbox:  plan.BlockingOutboxMessages > 0,
		OpenOperationalBoundaries: openBoundaries,
		ProtectedData:             plan.Protected,
		BlockReasons:              plan.BlockReasons,
		HumanSummary:              humanSummary,
		EligibleCounts:            plan.EligibleCounts,
		ArchiveCounts:             plan.ArchiveCounts,
		Verification:              verification,
	}
	if containsString(plan.BlockReasons, "archive_snapshot_payload_missing") {
		readiness.SnapshotPayloadVerified = false
	}
	return readiness
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

func (s *Service) archivePaths(archiveID, archivePath, manifestPath string) (string, string) {
	archivePath = strings.TrimSpace(archivePath)
	manifestPath = strings.TrimSpace(manifestPath)
	archiveID = strings.TrimSpace(archiveID)
	if archiveID != "" {
		root, err := filepath.Abs(s.archiveDir)
		if err == nil {
			if archivePath == "" {
				archivePath = filepath.Join(root, archiveID, "archive.jsonl")
			}
			if manifestPath == "" {
				manifestPath = filepath.Join(root, archiveID, "manifest.json")
			}
		}
	}
	return archivePath, manifestPath
}

func (s *Service) verifyArchive(archivePath, manifestPath string) (domainstorage.ArchiveVerificationSummary, *domainstorage.ArchiveManifest, []string) {
	verification := domainstorage.ArchiveVerificationSummary{
		ArchivePath:            strings.TrimSpace(archivePath),
		ManifestPath:           strings.TrimSpace(manifestPath),
		SnapshotPayloadPresent: true,
		IdentityFieldsPresent:  true,
		BusinessDateConsistent: true,
		RuntimeRowsNotDeleted:  true,
		PayloadPolicyPreserved: true,
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
	verification.RuntimeRowsNotDeleted = !manifest.RuntimeRowsDeleted
	if !verification.ManifestVersionMatched {
		reasons = appendUnique(reasons, "archive_manifest_version_mismatch")
	}
	if manifest.RuntimeRowsDeleted {
		reasons = appendUnique(reasons, "archive_runtime_rows_deleted_true")
	}
	if !manifestPayloadPolicyPreserved(manifest.Tables) {
		verification.PayloadPolicyPreserved = false
		reasons = appendUnique(reasons, "archive_sensitive_payload_policy_violation")
	}
	if verification.ArchivePath == "" {
		verification.ArchivePath = manifest.ArchivePath
	}
	if verification.ArchivePath != "" && manifest.ArchivePath != "" && verification.ArchivePath != manifest.ArchivePath {
		reasons = appendUnique(reasons, "archive_path_manifest_mismatch")
	}
	if verification.ArchivePath == "" {
		reasons = appendUnique(reasons, "archive_missing")
		return verification, &manifest, reasons
	}
	if !s.archivePathAllowed(verification.ArchivePath) {
		reasons = appendUnique(reasons, "archive_path_outside_archive_dir")
		return verification, &manifest, reasons
	}
	scan, err := verifyArchiveJSONL(verification.ArchivePath, manifest)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			reasons = appendUnique(reasons, "archive_missing")
			return verification, &manifest, reasons
		}
		if errors.Is(err, errArchiveJSONLMalformed) {
			reasons = appendUnique(reasons, "archive_jsonl_malformed")
			return verification, &manifest, reasons
		}
		reasons = appendUnique(reasons, "archive_unreadable")
		return verification, &manifest, reasons
	}
	verification.ArchiveExists = true
	verification.Counts = scan.counts
	verification.BusinessDateRange = scan.businessDateRange
	verification.ComputedSHA256 = scan.sha256
	verification.SHA256Matched = scan.sha256 == manifest.SHA256
	verification.CountsMatchedManifest = archiveCountsEqual(scan.counts, manifest.Counts)
	verification.SnapshotPayloadPresent = scan.snapshotPayloadPresent
	verification.IdentityFieldsPresent = scan.identityFieldsPresent
	verification.BusinessDateConsistent = scan.businessDateConsistent
	verification.PayloadPolicyPreserved = verification.PayloadPolicyPreserved && scan.payloadPolicyPreserved
	if !verification.SHA256Matched {
		reasons = appendUnique(reasons, "archive_sha_mismatch")
	}
	if !verification.CountsMatchedManifest {
		reasons = appendUnique(reasons, "archive_manifest_counts_mismatch")
	}
	if !verification.SnapshotPayloadPresent {
		reasons = appendUnique(reasons, "archive_snapshot_payload_missing")
	}
	if !verification.IdentityFieldsPresent {
		reasons = appendUnique(reasons, "archive_identity_fields_missing")
	}
	if !verification.BusinessDateConsistent {
		reasons = appendUnique(reasons, "archive_business_date_range_mismatch")
	}
	if !verification.PayloadPolicyPreserved {
		reasons = appendUnique(reasons, "archive_sensitive_payload_policy_violation")
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

type archiveJSONLScan struct {
	counts                 domainstorage.ArchiveExportCounts
	sha256                 string
	businessDateRange      domainstorage.BusinessDateRange
	snapshotPayloadPresent bool
	identityFieldsPresent  bool
	businessDateConsistent bool
	payloadPolicyPreserved bool
}

func verifyArchiveJSONL(path string, manifest domainstorage.ArchiveManifest) (archiveJSONLScan, error) {
	file, err := os.Open(path)
	if err != nil {
		return archiveJSONLScan{}, err
	}
	defer file.Close()
	hash := sha256.New()
	scanner := bufio.NewScanner(io.TeeReader(file, hash))
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	tableCounts := map[string]int{}
	businessDateRange := domainstorage.BusinessDateRange{}
	snapshotPayloadPresent := true
	identityFieldsPresent := true
	businessDateConsistent := true
	payloadPolicyPreserved := true
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		var row domainstorage.ArchiveExportRow
		if err := json.Unmarshal(line, &row); err != nil {
			return archiveJSONLScan{}, fmt.Errorf("%w: %v", errArchiveJSONLMalformed, err)
		}
		tableCounts[row.Table]++
		if !archiveRowHasRequiredIdentity(row) {
			identityFieldsPresent = false
		}
		if (row.Table == "checks" || row.Table == "prechecks") && !archiveRowHasSnapshotPayload(row.Row) {
			snapshotPayloadPresent = false
		}
		if row.Table == "checks" {
			businessDate := archiveStringValue(row.Row, "business_date_local")
			if businessDate != "" {
				if businessDateRange.Oldest == "" || businessDate < businessDateRange.Oldest {
					businessDateRange.Oldest = businessDate
				}
				if businessDateRange.Newest == "" || businessDate > businessDateRange.Newest {
					businessDateRange.Newest = businessDate
				}
				if manifest.CutoffBusinessDateLocal != "" && businessDate >= manifest.CutoffBusinessDateLocal {
					businessDateConsistent = false
				}
			}
		}
		if (row.Table == "local_event_log_summary" || row.Table == "pos_sync_outbox_summary") && !summaryRowPayloadPolicyPreserved(row.Row) {
			payloadPolicyPreserved = false
		}
	}
	if err := scanner.Err(); err != nil {
		return archiveJSONLScan{}, err
	}
	if businessDateRange != manifest.BusinessDateRange {
		businessDateConsistent = false
	}
	return archiveJSONLScan{
		counts:                 archiveCountsFromTableCounts(tableCounts),
		sha256:                 hex.EncodeToString(hash.Sum(nil)),
		businessDateRange:      businessDateRange,
		snapshotPayloadPresent: snapshotPayloadPresent,
		identityFieldsPresent:  identityFieldsPresent,
		businessDateConsistent: businessDateConsistent,
		payloadPolicyPreserved: payloadPolicyPreserved,
	}, nil
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

func archiveRowHasRequiredIdentity(row domainstorage.ArchiveExportRow) bool {
	fields := map[string][]string{
		"orders":                    {"id"},
		"order_lines":               {"id", "order_id"},
		"order_line_modifiers":      {"id", "order_line_id"},
		"kitchen_tickets":           {"id", "order_id", "order_line_id"},
		"kitchen_ticket_events":     {"id", "ticket_id", "order_line_id"},
		"order_line_discounts":      {"id", "order_id"},
		"order_surcharges":          {"id", "order_id"},
		"prechecks":                 {"id", "order_id"},
		"precheck_lines":            {"id", "precheck_id"},
		"precheck_line_modifiers":   {"id", "precheck_id"},
		"precheck_discounts":        {"id", "precheck_id"},
		"precheck_surcharges":       {"id", "precheck_id"},
		"precheck_taxes":            {"id", "precheck_id"},
		"payments":                  {"id", "precheck_id"},
		"payment_attempts":          {"id", "payment_id"},
		"checks":                    {"id", "order_id"},
		"financial_operations":      {"id", "check_id"},
		"financial_operation_items": {"id", "operation_id"},
		"local_event_log_summary":   {"id", "event_id", "aggregate_type", "aggregate_id"},
		"pos_sync_outbox_summary":   {"id", "command_id", "aggregate_type", "aggregate_id"},
	}
	required, ok := fields[row.Table]
	if !ok {
		return false
	}
	for _, field := range required {
		if archiveStringValue(row.Row, field) == "" {
			return false
		}
	}
	return true
}

func manifestPayloadPolicyPreserved(tables []domainstorage.ArchiveTableManifest) bool {
	for _, table := range tables {
		if table.Name != "local_event_log_summary" && table.Name != "pos_sync_outbox_summary" {
			continue
		}
		if table.PayloadIncluded || table.Content != "summary_without_payload" {
			return false
		}
	}
	return true
}

func summaryRowPayloadPolicyPreserved(row map[string]any) bool {
	if _, ok := row["payload_json"]; ok {
		return false
	}
	if _, ok := row["payload"]; ok {
		return false
	}
	return archiveStringValue(row, "payload_policy") == "summary_without_payload"
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
		case "kitchen_tickets":
			counts.KitchenTickets += n
		case "kitchen_ticket_events":
			counts.KitchenTicketEvents += n
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

type archiveReadPlanFilter struct {
	businessDateLocal string
	orderID           string
	checkID           string
	limit             int
	offset            int
}

func normalizeArchiveReadPlanBounds(rawLimit, rawOffset int) (int, int, error) {
	if rawOffset < 0 {
		return 0, 0, fmt.Errorf("%w: archive read-plan offset must be non-negative", domain.ErrInvalid)
	}
	limit := rawLimit
	if limit <= 0 {
		limit = archiveReadPlanDefaultLimit
	}
	if limit > archiveReadPlanMaxLimit {
		limit = archiveReadPlanMaxLimit
	}
	return limit, rawOffset, nil
}

func collectArchiveReadPlanPreview(path string, filter archiveReadPlanFilter) ([]domainstorage.ArchiveReadPlanClosedOrder, error) {
	selected := []domainstorage.ArchiveReadPlanClosedOrder{}
	selectedByOrder := map[string]int{}
	selectedByCheck := map[string]int{}
	skipped := 0
	err := scanArchiveJSONL(path, func(row domainstorage.ArchiveExportRow) (bool, error) {
		if row.Table != "checks" {
			return false, nil
		}
		orderID := archiveStringValue(row.Row, "order_id")
		checkID := archiveStringValue(row.Row, "id")
		businessDate := archiveStringValue(row.Row, "business_date_local")
		if filter.businessDateLocal != "" && businessDate != filter.businessDateLocal {
			return false, nil
		}
		if filter.orderID != "" && orderID != filter.orderID {
			return false, nil
		}
		if filter.checkID != "" && checkID != filter.checkID {
			return false, nil
		}
		if skipped < filter.offset {
			skipped++
			return false, nil
		}
		item := domainstorage.ArchiveReadPlanClosedOrder{
			OrderID:              orderID,
			CheckID:              checkID,
			BusinessDateLocal:    businessDate,
			ClosedAt:             archiveStringValue(row.Row, "closed_at"),
			CurrencyCode:         archiveStringValue(row.Row, "currency_code"),
			Total:                archiveInt64Value(row.Row, "total"),
			DocumentState:        "archived_preview",
			RuntimeRestored:      false,
			CheckSnapshotPresent: archiveRowHasSnapshotPayload(row.Row),
		}
		selectedByOrder[orderID] = len(selected)
		selectedByCheck[checkID] = len(selected)
		selected = append(selected, item)
		return len(selected) >= filter.limit, nil
	})
	if err != nil || len(selected) == 0 {
		return selected, err
	}
	operationToCheck := map[string]string{}
	err = scanArchiveJSONL(path, func(row domainstorage.ArchiveExportRow) (bool, error) {
		switch row.Table {
		case "order_lines":
			if idx, ok := selectedByOrder[archiveStringValue(row.Row, "order_id")]; ok {
				selected[idx].RelatedCounts.OrderLines++
			}
		case "prechecks":
			if idx, ok := selectedByOrder[archiveStringValue(row.Row, "order_id")]; ok {
				selected[idx].PrecheckID = archiveStringValue(row.Row, "id")
				selected[idx].PrecheckSnapshotPresent = archiveRowHasSnapshotPayload(row.Row)
			}
		case "payments":
			precheckID := archiveStringValue(row.Row, "precheck_id")
			for i := range selected {
				if selected[i].PrecheckID == precheckID {
					selected[i].RelatedCounts.Payments++
					break
				}
			}
		case "financial_operations":
			if idx, ok := selectedByCheck[archiveStringValue(row.Row, "check_id")]; ok {
				operationToCheck[archiveStringValue(row.Row, "id")] = archiveStringValue(row.Row, "check_id")
				selected[idx].RelatedCounts.FinancialOperations++
			}
		case "financial_operation_items":
			if idx, ok := selectedByCheck[operationToCheck[archiveStringValue(row.Row, "operation_id")]]; ok {
				selected[idx].RelatedCounts.FinancialOperationItems++
			}
		}
		return false, nil
	})
	return selected, err
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

func archiveInt64Value(row map[string]any, key string) int64 {
	raw, ok := row[key]
	if !ok || raw == nil {
		return 0
	}
	switch v := raw.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	case string:
		var n int64
		if _, err := fmt.Sscan(strings.TrimSpace(v), &n); err == nil {
			return n
		}
	}
	return 0
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

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
