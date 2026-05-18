package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"pos-backend/internal/platform/clock"
	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
	domainstorage "pos-backend/internal/pos/domain/storage"
	"pos-backend/internal/pos/ports"
)

const retentionDryRunOnlyReason = "physical archive/delete is not implemented; financial ledger, immutable snapshots and audit/sync rows remain protected"

// Service предоставляет read-only lifecycle surface для локальной POS Edge БД.
type Service struct {
	repo  ports.StorageLifecycleRepository
	auth  ports.Repository
	clock clock.Clock
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

// NewService создает storage lifecycle service поверх общего POS repository.
func NewService(repo ports.Repository, clock clock.Clock) *Service {
	return &Service{repo: repo, auth: repo, clock: clock}
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

func retentionCapability() domainstorage.RetentionCapability {
	return domainstorage.RetentionCapability{
		Mode:                        "dry_run_only",
		DestructiveApplySupported:   false,
		FinancialLedgerProtected:    true,
		ImmutableSnapshotsProtected: true,
		Reason:                      retentionDryRunOnlyReason,
	}
}

func appendUnique(values []string, next string) []string {
	for _, value := range values {
		if value == next {
			return values
		}
	}
	return append(values, next)
}
