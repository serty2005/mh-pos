package postgres

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	platformpg "cloud-backend/internal/platform/postgres"
	"cloud-backend/internal/provisioning/domain"
)

func TestRepositoryRegisterUnassignedInsertAndConflictUpdatePersistsActualRow(t *testing.T) {
	ctx := t.Context()
	pool, repo, closeFn := openProvisioningRepositoryWithBaseline(t, ctx)
	defer closeFn()
	insertProvisioningRestaurant(t, ctx, pool, "restaurant-1")

	firstSeen := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	first, err := repo.RegisterUnassigned(ctx, domain.UnassignedEdgeNode{
		ID:              "unassigned-1",
		NodeDeviceID:    "node-1",
		ClaimedCloudURL: "https://cloud.initial",
		DisplayName:     "POS Edge",
		AppVersion:      "1.0.0",
		Status:          domain.UnassignedPending,
		FirstSeenAt:     firstSeen,
		LastSeenAt:      firstSeen,
		CreatedAt:       firstSeen,
		UpdatedAt:       firstSeen,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != "unassigned-1" || first.NodeDeviceID != "node-1" || first.Status != domain.UnassignedPending {
		t.Fatalf("unexpected inserted unassigned row id=%q node=%q status=%q", first.ID, first.NodeDeviceID, first.Status)
	}

	assignedAt := firstSeen.Add(15 * time.Minute)
	if _, err := pool.Exec(ctx, `
UPDATE cloud_unassigned_edge_nodes
SET status = 'rejected', assigned_restaurant_id = 'restaurant-1', assigned_at = $2
WHERE node_device_id = $1`, "node-1", assignedAt); err != nil {
		t.Fatal(err)
	}

	updateSeen := firstSeen.Add(30 * time.Minute)
	updated, err := repo.RegisterUnassigned(ctx, domain.UnassignedEdgeNode{
		ID:                   "unassigned-replacement",
		NodeDeviceID:         "node-1",
		ClaimedCloudURL:      "https://cloud.updated",
		DisplayName:          "POS Edge Updated",
		AppVersion:           "1.1.0",
		Status:               domain.UnassignedPending,
		FirstSeenAt:          updateSeen,
		LastSeenAt:           updateSeen,
		AssignedRestaurantID: "restaurant-other",
		AssignedAt:           &updateSeen,
		CreatedAt:            updateSeen,
		UpdatedAt:            updateSeen,
	})
	if err != nil {
		t.Fatal(err)
	}

	if updated.ID != "unassigned-1" || updated.Status != domain.UnassignedRejected || updated.AssignedRestaurantID != "restaurant-1" {
		t.Fatalf("stable fields were overwritten after conflict update: id=%q status=%q assigned_restaurant_id=%q", updated.ID, updated.Status, updated.AssignedRestaurantID)
	}
	assertTimeEqual(t, updated.FirstSeenAt, firstSeen, "first_seen_at")
	assertTimePtrEqual(t, updated.AssignedAt, assignedAt, "assigned_at")
	if updated.ClaimedCloudURL != "https://cloud.updated" || updated.DisplayName != "POS Edge Updated" || updated.AppVersion != "1.1.0" {
		t.Fatalf("mutable fields were not updated: url=%q name=%q version=%q", updated.ClaimedCloudURL, updated.DisplayName, updated.AppVersion)
	}
	assertTimeEqual(t, updated.LastSeenAt, updateSeen, "last_seen_at")
	assertTimeEqual(t, updated.UpdatedAt, updateSeen, "updated_at")

	stored := readUnassignedRow(t, ctx, pool, "node-1")
	if stored.ID != updated.ID || stored.Status != updated.Status || !stored.LastSeenAt.Equal(updated.LastSeenAt) || !stored.UpdatedAt.Equal(updated.UpdatedAt) {
		t.Fatalf("returned unassigned row does not match persisted row id=%q status=%q", updated.ID, updated.Status)
	}
}

func TestRepositoryListUnassignedFiltersOrdersAndRoundTripsNullables(t *testing.T) {
	ctx := t.Context()
	pool, repo, closeFn := openProvisioningRepositoryWithBaseline(t, ctx)
	defer closeFn()
	insertProvisioningRestaurant(t, ctx, pool, "restaurant-1")

	empty, err := repo.ListUnassigned(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty unassigned list, got %d rows", len(empty))
	}

	base := time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC)
	insertUnassignedRow(t, ctx, pool, domain.UnassignedEdgeNode{
		ID:              "pending-old",
		NodeDeviceID:    "node-pending-old",
		ClaimedCloudURL: "https://cloud.local",
		DisplayName:     "Old Pending",
		AppVersion:      "1.0.0",
		Status:          domain.UnassignedPending,
		FirstSeenAt:     base,
		LastSeenAt:      base,
		CreatedAt:       base,
		UpdatedAt:       base,
	})
	pendingWithAssignedAt := base.Add(10 * time.Minute)
	insertUnassignedRow(t, ctx, pool, domain.UnassignedEdgeNode{
		ID:              "pending-middle",
		NodeDeviceID:    "node-pending-middle",
		ClaimedCloudURL: "https://cloud.local",
		DisplayName:     "Middle Pending",
		AppVersion:      "1.1.0",
		Status:          domain.UnassignedPending,
		FirstSeenAt:     base,
		LastSeenAt:      pendingWithAssignedAt,
		AssignedAt:      &pendingWithAssignedAt,
		CreatedAt:       base,
		UpdatedAt:       pendingWithAssignedAt,
	})
	insertUnassignedRow(t, ctx, pool, domain.UnassignedEdgeNode{
		ID:                   "assigned-row",
		NodeDeviceID:         "node-assigned",
		ClaimedCloudURL:      "https://cloud.local",
		DisplayName:          "Assigned",
		AppVersion:           "1.2.0",
		Status:               domain.UnassignedAssigned,
		FirstSeenAt:          base,
		LastSeenAt:           base.Add(30 * time.Minute),
		AssignedRestaurantID: "restaurant-1",
		AssignedAt:           ptrTime(base.Add(30 * time.Minute)),
		CreatedAt:            base,
		UpdatedAt:            base.Add(30 * time.Minute),
	})
	insertUnassignedRow(t, ctx, pool, domain.UnassignedEdgeNode{
		ID:              "pending-new",
		NodeDeviceID:    "node-pending-new",
		ClaimedCloudURL: "https://cloud.local",
		DisplayName:     "New Pending",
		AppVersion:      "1.3.0",
		Status:          domain.UnassignedPending,
		FirstSeenAt:     base,
		LastSeenAt:      base.Add(45 * time.Minute),
		CreatedAt:       base,
		UpdatedAt:       base.Add(45 * time.Minute),
	})

	got, err := repo.ListUnassigned(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wantOrder := []string{"node-pending-new", "node-pending-middle", "node-pending-old"}
	if len(got) != len(wantOrder) {
		t.Fatalf("expected %d pending rows, got %d", len(wantOrder), len(got))
	}
	for i, wantNode := range wantOrder {
		if got[i].NodeDeviceID != wantNode || got[i].Status != domain.UnassignedPending {
			t.Fatalf("unexpected unassigned row at %d: node=%q status=%q", i, got[i].NodeDeviceID, got[i].Status)
		}
		if got[i].AssignedRestaurantID != "" {
			t.Fatalf("expected nullable assigned_restaurant_id to round-trip empty for node=%q", got[i].NodeDeviceID)
		}
	}
	if got[0].AssignedAt != nil || got[2].AssignedAt != nil {
		t.Fatalf("expected nil assigned_at on rows without assignment timestamps")
	}
	assertTimePtrEqual(t, got[1].AssignedAt, pendingWithAssignedAt, "assigned_at")
}

func TestRepositoryUpsertEdgeNodeGetAndListByRestaurant(t *testing.T) {
	ctx := t.Context()
	pool, repo, closeFn := openProvisioningRepositoryWithBaseline(t, ctx)
	defer closeFn()
	insertProvisioningRestaurant(t, ctx, pool, "restaurant-1")
	insertProvisioningRestaurant(t, ctx, pool, "restaurant-2")

	base := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	firstLastSeen := base.Add(5 * time.Minute)
	first, err := repo.UpsertEdgeNode(ctx, domain.EdgeNode{
		ID:              "edge-node-1",
		RestaurantID:    "restaurant-1",
		NodeDeviceID:    "node-edge-1",
		DisplayName:     "POS Edge",
		Status:          domain.EdgeNodeAssigned,
		CredentialsHash: "sha256:first-token",
		LastSeenAt:      &firstLastSeen,
		AssignedAt:      &base,
		CreatedAt:       base,
		UpdatedAt:       base,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != "edge-node-1" || first.RestaurantID != "restaurant-1" || first.CredentialsHash != "sha256:first-token" {
		t.Fatalf("unexpected inserted edge node id=%q restaurant=%q credentials_set=%t", first.ID, first.RestaurantID, first.CredentialsHash != "")
	}

	revokedAt := base.Add(40 * time.Minute)
	secondLastSeen := base.Add(35 * time.Minute)
	updated, err := repo.UpsertEdgeNode(ctx, domain.EdgeNode{
		ID:              "edge-node-replacement",
		RestaurantID:    "restaurant-2",
		NodeDeviceID:    "node-edge-1",
		DisplayName:     "POS Edge Moved",
		Status:          domain.EdgeNodeRevoked,
		CredentialsHash: "sha256:rotated-token",
		LastSeenAt:      &secondLastSeen,
		AssignedAt:      nil,
		RevokedAt:       &revokedAt,
		CreatedAt:       base.Add(30 * time.Minute),
		UpdatedAt:       base.Add(45 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != "edge-node-1" || updated.RestaurantID != "restaurant-2" || updated.DisplayName != "POS Edge Moved" || updated.Status != domain.EdgeNodeRevoked {
		t.Fatalf("unexpected updated edge node id=%q restaurant=%q status=%q", updated.ID, updated.RestaurantID, updated.Status)
	}
	if updated.CredentialsHash != "sha256:rotated-token" {
		t.Fatalf("credentials hash was not updated for node=%q", updated.NodeDeviceID)
	}
	assertTimePtrEqual(t, updated.AssignedAt, base, "assigned_at")
	assertTimePtrEqual(t, updated.LastSeenAt, secondLastSeen, "last_seen_at")
	assertTimePtrEqual(t, updated.RevokedAt, revokedAt, "revoked_at")

	got, err := repo.GetEdgeNode(ctx, "node-edge-1")
	if err != nil {
		t.Fatal(err)
	}
	assertEdgeNodeEqual(t, got, updated)

	olderUpdatedAt := base.Add(15 * time.Minute)
	_, err = repo.UpsertEdgeNode(ctx, domain.EdgeNode{
		ID:              "edge-node-2",
		RestaurantID:    "restaurant-2",
		NodeDeviceID:    "node-edge-2",
		DisplayName:     "Other POS Edge",
		Status:          domain.EdgeNodeAssigned,
		CredentialsHash: "sha256:other-token",
		LastSeenAt:      &olderUpdatedAt,
		AssignedAt:      &olderUpdatedAt,
		CreatedAt:       base,
		UpdatedAt:       olderUpdatedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.UpsertEdgeNode(ctx, domain.EdgeNode{
		ID:              "edge-node-restaurant-1",
		RestaurantID:    "restaurant-1",
		NodeDeviceID:    "node-restaurant-1",
		DisplayName:     "Restaurant One Edge",
		Status:          domain.EdgeNodeAssigned,
		CredentialsHash: "sha256:restaurant-one-token",
		LastSeenAt:      &olderUpdatedAt,
		AssignedAt:      &olderUpdatedAt,
		CreatedAt:       base,
		UpdatedAt:       base.Add(60 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}

	list, err := repo.ListEdgeNodesByRestaurant(ctx, "restaurant-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected two restaurant-2 nodes, got %d", len(list))
	}
	if list[0].NodeDeviceID != "node-edge-1" || list[0].Status != domain.EdgeNodeRevoked || list[1].NodeDeviceID != "node-edge-2" {
		t.Fatalf("unexpected restaurant node order/status: first=%q/%q second=%q/%q", list[0].NodeDeviceID, list[0].Status, list[1].NodeDeviceID, list[1].Status)
	}
}

func TestRepositoryMarkUnassignedAssignedIsIdempotentAndConflictsOnDifferentRestaurant(t *testing.T) {
	ctx := t.Context()
	pool, repo, closeFn := openProvisioningRepositoryWithBaseline(t, ctx)
	defer closeFn()
	insertProvisioningRestaurant(t, ctx, pool, "restaurant-1")
	insertProvisioningRestaurant(t, ctx, pool, "restaurant-2")

	base := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	insertUnassignedRow(t, ctx, pool, domain.UnassignedEdgeNode{
		ID:              "unassigned-assign",
		NodeDeviceID:    "node-assign",
		ClaimedCloudURL: "https://cloud.local",
		DisplayName:     "Assignable",
		AppVersion:      "1.0.0",
		Status:          domain.UnassignedPending,
		FirstSeenAt:     base,
		LastSeenAt:      base,
		CreatedAt:       base,
		UpdatedAt:       base,
	})

	assignedAt := base.Add(10 * time.Minute)
	if err := repo.MarkUnassignedAssigned(ctx, "node-assign", "restaurant-1", assignedAt); err != nil {
		t.Fatal(err)
	}
	stored := readUnassignedRow(t, ctx, pool, "node-assign")
	if stored.Status != domain.UnassignedAssigned || stored.AssignedRestaurantID != "restaurant-1" {
		t.Fatalf("unexpected assigned row state status=%q restaurant=%q", stored.Status, stored.AssignedRestaurantID)
	}
	assertTimePtrEqual(t, stored.AssignedAt, assignedAt, "assigned_at")

	list, err := repo.ListUnassigned(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("assigned unassigned-row must be hidden from pending list, got %d rows", len(list))
	}

	replayAt := base.Add(20 * time.Minute)
	if err := repo.MarkUnassignedAssigned(ctx, "node-assign", "restaurant-1", replayAt); err != nil {
		t.Fatal(err)
	}
	replayed := readUnassignedRow(t, ctx, pool, "node-assign")
	if replayed.AssignedRestaurantID != "restaurant-1" {
		t.Fatalf("same assignment replay changed restaurant to %q", replayed.AssignedRestaurantID)
	}
	assertTimePtrEqual(t, replayed.AssignedAt, assignedAt, "assigned_at")

	err = repo.MarkUnassignedAssigned(ctx, "node-assign", "restaurant-2", base.Add(30*time.Minute))
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict for conflicting assignment, got %v", err)
	}
	conflicted := readUnassignedRow(t, ctx, pool, "node-assign")
	if conflicted.AssignedRestaurantID != "restaurant-1" {
		t.Fatalf("conflicting assignment changed restaurant to %q", conflicted.AssignedRestaurantID)
	}
	assertTimePtrEqual(t, conflicted.AssignedAt, assignedAt, "assigned_at")
}

func TestRepositoryPairingCodeCreateAndRevokeActiveCodes(t *testing.T) {
	ctx := t.Context()
	pool, repo, closeFn := openProvisioningRepositoryWithBaseline(t, ctx)
	defer closeFn()
	insertProvisioningRestaurant(t, ctx, pool, "restaurant-1")
	insertProvisioningRestaurant(t, ctx, pool, "restaurant-2")

	base := time.Date(2026, 6, 5, 8, 0, 0, 0, time.UTC)
	active, err := repo.CreatePairingCode(ctx, domain.PairingCode{
		ID:              "pairing-active-r1",
		PairingCodeHash: "sha256:pairing-active-r1",
		PairingKey:      "derived-key-r1",
		RestaurantID:    "restaurant-1",
		CloudURL:        "https://cloud.local",
		Status:          domain.PairingCodeActive,
		ExpiresAt:       base.Add(time.Hour),
		CreatedAt:       base,
		UpdatedAt:       base,
	})
	if err != nil {
		t.Fatal(err)
	}
	if active.ID != "pairing-active-r1" || active.RestaurantID != "restaurant-1" || active.NodeDeviceID != "" || active.ConsumedAt != nil {
		t.Fatalf("unexpected created active pairing id=%q restaurant=%q node=%q consumed_set=%t", active.ID, active.RestaurantID, active.NodeDeviceID, active.ConsumedAt != nil)
	}
	if active.PairingCodeHash != "sha256:pairing-active-r1" || active.PairingKey != "derived-key-r1" || active.Status != domain.PairingCodeActive {
		t.Fatalf("unexpected persisted pairing secret metadata id=%q status=%q", active.ID, active.Status)
	}
	assertTimeEqual(t, active.ExpiresAt, base.Add(time.Hour), "expires_at")
	assertTimeEqual(t, active.CreatedAt, base, "created_at")

	consumedAt := base.Add(10 * time.Minute)
	insertPairingRow(t, ctx, pool, domain.PairingCode{
		ID:              "pairing-consumed-r1",
		PairingCodeHash: "sha256:pairing-consumed-r1",
		PairingKey:      "derived-key-consumed",
		RestaurantID:    "restaurant-1",
		NodeDeviceID:    "node-consumed",
		CloudURL:        "https://cloud.local",
		Status:          domain.PairingCodeConsumed,
		ExpiresAt:       base.Add(time.Hour),
		ConsumedAt:      &consumedAt,
		CreatedAt:       base,
		UpdatedAt:       base,
	})
	insertPairingRow(t, ctx, pool, domain.PairingCode{
		ID:              "pairing-expired-r1",
		PairingCodeHash: "sha256:pairing-expired-r1",
		PairingKey:      "derived-key-expired",
		RestaurantID:    "restaurant-1",
		CloudURL:        "https://cloud.local",
		Status:          domain.PairingCodeExpired,
		ExpiresAt:       base.Add(-time.Hour),
		CreatedAt:       base,
		UpdatedAt:       base,
	})
	insertPairingRow(t, ctx, pool, domain.PairingCode{
		ID:              "pairing-revoked-r1",
		PairingCodeHash: "sha256:pairing-revoked-r1",
		PairingKey:      "derived-key-revoked",
		RestaurantID:    "restaurant-1",
		CloudURL:        "https://cloud.local",
		Status:          domain.PairingCodeRevoked,
		ExpiresAt:       base.Add(time.Hour),
		CreatedAt:       base,
		UpdatedAt:       base,
	})
	_, err = repo.CreatePairingCode(ctx, domain.PairingCode{
		ID:              "pairing-active-r2",
		PairingCodeHash: "sha256:pairing-active-r2",
		PairingKey:      "derived-key-r2",
		RestaurantID:    "restaurant-2",
		CloudURL:        "https://cloud.local",
		Status:          domain.PairingCodeActive,
		ExpiresAt:       base.Add(time.Hour),
		CreatedAt:       base,
		UpdatedAt:       base,
	})
	if err != nil {
		t.Fatal(err)
	}

	revokedAt := base.Add(30 * time.Minute)
	if err := repo.RevokeActivePairingCodes(ctx, "restaurant-1", revokedAt); err != nil {
		t.Fatal(err)
	}
	assertPairingStatus(t, ctx, pool, "pairing-active-r1", domain.PairingCodeRevoked, revokedAt)
	assertPairingStatus(t, ctx, pool, "pairing-consumed-r1", domain.PairingCodeConsumed, base)
	assertPairingStatus(t, ctx, pool, "pairing-expired-r1", domain.PairingCodeExpired, base)
	assertPairingStatus(t, ctx, pool, "pairing-revoked-r1", domain.PairingCodeRevoked, base)
	assertPairingStatus(t, ctx, pool, "pairing-active-r2", domain.PairingCodeActive, base)

	if err := repo.RevokeActivePairingCodes(ctx, "restaurant-1", base.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	assertPairingStatus(t, ctx, pool, "pairing-active-r1", domain.PairingCodeRevoked, revokedAt)
}

func TestRepositoryPairingCodeGetAndConsumeContracts(t *testing.T) {
	ctx := t.Context()
	pool, repo, closeFn := openProvisioningRepositoryWithBaseline(t, ctx)
	defer closeFn()
	insertProvisioningRestaurant(t, ctx, pool, "restaurant-1")

	base := time.Date(2026, 6, 6, 9, 0, 0, 0, time.UTC)
	consumedSeedAt := base.Add(5 * time.Minute)
	for _, code := range []domain.PairingCode{
		{
			ID:              "pairing-active",
			PairingCodeHash: "sha256:pairing-active",
			PairingKey:      "derived-key-active",
			RestaurantID:    "restaurant-1",
			CloudURL:        "https://cloud.local",
			Status:          domain.PairingCodeActive,
			ExpiresAt:       base.Add(time.Hour),
			CreatedAt:       base,
			UpdatedAt:       base,
		},
		{
			ID:              "pairing-consumed-existing",
			PairingCodeHash: "sha256:pairing-consumed-existing",
			PairingKey:      "derived-key-consumed-existing",
			RestaurantID:    "restaurant-1",
			NodeDeviceID:    "node-existing",
			CloudURL:        "https://cloud.local",
			Status:          domain.PairingCodeConsumed,
			ExpiresAt:       base.Add(time.Hour),
			ConsumedAt:      &consumedSeedAt,
			CreatedAt:       base,
			UpdatedAt:       consumedSeedAt,
		},
		{
			ID:              "pairing-revoked",
			PairingCodeHash: "sha256:pairing-revoked-consume",
			PairingKey:      "derived-key-revoked-consume",
			RestaurantID:    "restaurant-1",
			CloudURL:        "https://cloud.local",
			Status:          domain.PairingCodeRevoked,
			ExpiresAt:       base.Add(time.Hour),
			CreatedAt:       base,
			UpdatedAt:       base,
		},
		{
			ID:              "pairing-expired",
			PairingCodeHash: "sha256:pairing-expired-consume",
			PairingKey:      "derived-key-expired-consume",
			RestaurantID:    "restaurant-1",
			CloudURL:        "https://cloud.local",
			Status:          domain.PairingCodeExpired,
			ExpiresAt:       base.Add(-time.Hour),
			CreatedAt:       base,
			UpdatedAt:       base,
		},
	} {
		if code.Status == domain.PairingCodeActive {
			if _, err := repo.CreatePairingCode(ctx, code); err != nil {
				t.Fatal(err)
			}
			continue
		}
		insertPairingRow(t, ctx, pool, code)
	}

	existing, err := repo.GetPairingCode(ctx, "pairing-consumed-existing")
	if err != nil {
		t.Fatal(err)
	}
	if existing.NodeDeviceID != "node-existing" || existing.ConsumedAt == nil || existing.Status != domain.PairingCodeConsumed {
		t.Fatalf("unexpected existing pairing fields id=%q node=%q status=%q consumed_set=%t", existing.ID, existing.NodeDeviceID, existing.Status, existing.ConsumedAt != nil)
	}
	assertTimePtrEqual(t, existing.ConsumedAt, consumedSeedAt, "consumed_at")

	_, err = repo.GetPairingCode(ctx, "pairing-missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing pairing code, got %v", err)
	}

	consumeAt := base.Add(20 * time.Minute)
	if err := repo.ConsumePairingCode(ctx, "pairing-active", "node-consumer-1", consumeAt); err != nil {
		t.Fatal(err)
	}
	consumed, err := repo.GetPairingCode(ctx, "pairing-active")
	if err != nil {
		t.Fatal(err)
	}
	if consumed.Status != domain.PairingCodeConsumed || consumed.NodeDeviceID != "node-consumer-1" {
		t.Fatalf("unexpected consumed pairing id=%q node=%q status=%q", consumed.ID, consumed.NodeDeviceID, consumed.Status)
	}
	assertTimePtrEqual(t, consumed.ConsumedAt, consumeAt, "consumed_at")

	err = repo.ConsumePairingCode(ctx, "pairing-active", "node-consumer-1", base.Add(30*time.Minute))
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict for same-node consume replay, got %v", err)
	}
	err = repo.ConsumePairingCode(ctx, "pairing-active", "node-consumer-2", base.Add(40*time.Minute))
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict for different-node consume replay, got %v", err)
	}
	afterReplay, err := repo.GetPairingCode(ctx, "pairing-active")
	if err != nil {
		t.Fatal(err)
	}
	if afterReplay.NodeDeviceID != "node-consumer-1" {
		t.Fatalf("consume replay changed first consumer to %q", afterReplay.NodeDeviceID)
	}
	assertTimePtrEqual(t, afterReplay.ConsumedAt, consumeAt, "consumed_at")

	if err := repo.ConsumePairingCode(ctx, "pairing-revoked", "node-revoked", base.Add(50*time.Minute)); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict for revoked pairing consume, got %v", err)
	}
	if err := repo.ConsumePairingCode(ctx, "pairing-expired", "node-expired", base.Add(50*time.Minute)); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict for expired pairing consume, got %v", err)
	}
}

func TestRepositoryNormalizesConflictsAndDoesNotLeakSecrets(t *testing.T) {
	ctx := t.Context()
	pool, repo, closeFn := openProvisioningRepositoryWithBaseline(t, ctx)
	defer closeFn()
	insertProvisioningRestaurant(t, ctx, pool, "restaurant-1")
	insertProvisioningRestaurant(t, ctx, pool, "restaurant-2")

	base := time.Date(2026, 6, 7, 11, 0, 0, 0, time.UTC)
	plainSecretMarker := "secret-token-marker"

	_, err := repo.CreatePairingCode(ctx, domain.PairingCode{
		ID:              "pairing-secret-1",
		PairingCodeHash: "hash-" + plainSecretMarker,
		PairingKey:      "derived-key-1",
		RestaurantID:    "restaurant-1",
		CloudURL:        "https://cloud.local",
		Status:          domain.PairingCodeActive,
		ExpiresAt:       base.Add(time.Hour),
		CreatedAt:       base,
		UpdatedAt:       base,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.CreatePairingCode(ctx, domain.PairingCode{
		ID:              "pairing-secret-2",
		PairingCodeHash: "hash-" + plainSecretMarker,
		PairingKey:      "derived-key-2",
		RestaurantID:    "restaurant-2",
		CloudURL:        "https://cloud.local",
		Status:          domain.PairingCodeActive,
		ExpiresAt:       base.Add(time.Hour),
		CreatedAt:       base,
		UpdatedAt:       base,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict for duplicate pairing hash, got %v", err)
	}
	assertErrorDoesNotContain(t, err, plainSecretMarker)

	lastSeen := base.Add(5 * time.Minute)
	_, err = repo.UpsertEdgeNode(ctx, domain.EdgeNode{
		ID:              "edge-duplicate-id",
		RestaurantID:    "restaurant-1",
		NodeDeviceID:    "node-duplicate-a",
		DisplayName:     "Duplicate A",
		Status:          domain.EdgeNodeAssigned,
		CredentialsHash: "sha256:node-secret-a",
		LastSeenAt:      &lastSeen,
		AssignedAt:      &lastSeen,
		CreatedAt:       base,
		UpdatedAt:       base,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.UpsertEdgeNode(ctx, domain.EdgeNode{
		ID:              "edge-duplicate-id",
		RestaurantID:    "restaurant-1",
		NodeDeviceID:    "node-duplicate-b",
		DisplayName:     "Duplicate B",
		Status:          domain.EdgeNodeAssigned,
		CredentialsHash: "sha256:" + plainSecretMarker,
		LastSeenAt:      &lastSeen,
		AssignedAt:      &lastSeen,
		CreatedAt:       base,
		UpdatedAt:       base,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict for duplicate edge node id, got %v", err)
	}
	assertErrorDoesNotContain(t, err, plainSecretMarker)

	_, err = repo.GetPairingCode(ctx, "missing-pairing-secret-token-marker")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing pairing code, got %v", err)
	}
	assertErrorDoesNotContain(t, err, plainSecretMarker)

	activePublic, err := repo.GetPairingCode(ctx, "pairing-secret-1")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(publicPairingFields(activePublic), plainSecretMarker) {
		t.Fatalf("public pairing fields leaked plaintext marker for id=%q", activePublic.ID)
	}
	edgePublic, err := repo.GetEdgeNode(ctx, "node-duplicate-a")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(publicEdgeNodeFields(edgePublic), plainSecretMarker) {
		t.Fatalf("public edge node fields leaked plaintext marker for node=%q", edgePublic.NodeDeviceID)
	}
}

func openProvisioningRepositoryWithBaseline(t *testing.T, ctx context.Context) (*pgxpool.Pool, *Repository, func()) {
	t.Helper()
	pool := openProvisioningPostgresIntegrationPool(t)
	resetProvisioningPublicSchema(t, ctx, pool)
	if err := platformpg.MigrateDirWithPolicy(ctx, pool, provisioningMigrationsDir(), platformpg.MigrationOptions{
		ModuleName:         "cloud-backend",
		ModuleVersion:      "0.1.0",
		BackupDir:          t.TempDir(),
		SchemaRequirements: provisioningSchemaRequirements(),
	}); err != nil {
		t.Fatalf("postgres baseline migration failed: %v", err)
	}
	return pool, NewRepository(pool), func() {
		resetProvisioningPublicSchema(t, context.Background(), pool)
	}
}

func openProvisioningPostgresIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("CLOUD_POSTGRES_TEST_DSN"))
	if dsn == "" {
		t.Skip("CLOUD_POSTGRES_TEST_DSN is not set")
	}
	pool, err := pgxpool.New(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	lockProvisioningPostgresIntegration(t, t.Context(), pool)
	return pool
}

func provisioningMigrationsDir() string {
	return filepath.Join("..", "..", "..", "..", "migrations", "postgres")
}

func provisioningSchemaRequirements() []platformpg.SchemaRequirement {
	return []platformpg.SchemaRequirement{
		{
			Table:         "cloud_restaurants",
			RequiredBy:    "provisioning postgres repository integration tests",
			MigrationFile: "001_init.sql",
			Columns:       []string{"id", "name", "timezone", "currency", "business_day_mode", "business_day_boundary_local_time", "status", "created_at", "updated_at"},
			Indexes:       []string{"cloud_restaurants_status_updated"},
		},
		{
			Table:         "cloud_edge_nodes",
			RequiredBy:    "provisioning postgres assigned edge node tests",
			MigrationFile: "001_init.sql",
			Columns:       []string{"id", "restaurant_id", "node_device_id", "display_name", "status", "credentials_hash", "last_seen_at", "assigned_at", "revoked_at", "created_at", "updated_at"},
			Indexes:       []string{"cloud_edge_nodes_restaurant_status"},
		},
		{
			Table:         "cloud_unassigned_edge_nodes",
			RequiredBy:    "provisioning postgres unassigned edge node tests",
			MigrationFile: "001_init.sql",
			Columns:       []string{"id", "node_device_id", "claimed_cloud_url", "display_name", "app_version", "status", "first_seen_at", "last_seen_at", "assigned_restaurant_id", "assigned_at", "created_at", "updated_at"},
			Indexes:       []string{"cloud_unassigned_edge_nodes_status_seen"},
		},
		{
			Table:         "cloud_pairing_codes",
			RequiredBy:    "provisioning postgres pairing code tests",
			MigrationFile: "001_init.sql",
			Columns:       []string{"id", "pairing_code_hash", "pairing_key", "restaurant_id", "node_device_id", "cloud_url", "status", "expires_at", "consumed_at", "created_at", "updated_at"},
			Indexes:       []string{"cloud_pairing_codes_restaurant_status", "cloud_pairing_codes_one_active_per_restaurant"},
		},
	}
}

func resetProvisioningPublicSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatalf("reset public schema: %v", err)
	}
}

func lockProvisioningPostgresIntegration(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `SELECT pg_advisory_lock(72905101)`); err != nil {
		t.Fatalf("lock postgres integration db: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `SELECT pg_advisory_unlock(72905101)`); err != nil {
			t.Logf("unlock postgres integration db: %v", err)
		}
	})
}

func insertProvisioningRestaurant(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id string) {
	t.Helper()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
INSERT INTO cloud_restaurants(
  id,name,timezone,currency,business_day_mode,business_day_boundary_local_time,status,created_at,updated_at
) VALUES ($1,$2,'Europe/Moscow','RUB','standard','04:00','active',$3,$3)`,
		id, "Restaurant "+id, now); err != nil {
		t.Fatalf("insert restaurant %s: %v", id, err)
	}
}

func insertUnassignedRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool, v domain.UnassignedEdgeNode) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
INSERT INTO cloud_unassigned_edge_nodes(
  id,node_device_id,claimed_cloud_url,display_name,app_version,status,first_seen_at,last_seen_at,
  assigned_restaurant_id,assigned_at,created_at,updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		v.ID, v.NodeDeviceID, v.ClaimedCloudURL, v.DisplayName, v.AppVersion, v.Status, v.FirstSeenAt, v.LastSeenAt,
		nullableText(v.AssignedRestaurantID), v.AssignedAt, v.CreatedAt, v.UpdatedAt); err != nil {
		t.Fatalf("insert unassigned row id=%q status=%q: %v", v.ID, v.Status, err)
	}
}

func insertPairingRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool, v domain.PairingCode) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
INSERT INTO cloud_pairing_codes(
  id,pairing_code_hash,pairing_key,restaurant_id,node_device_id,cloud_url,status,expires_at,consumed_at,created_at,updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		v.ID, v.PairingCodeHash, v.PairingKey, v.RestaurantID, nullableText(v.NodeDeviceID), v.CloudURL, v.Status, v.ExpiresAt, v.ConsumedAt, v.CreatedAt, v.UpdatedAt); err != nil {
		t.Fatalf("insert pairing row id=%q status=%q: %v", v.ID, v.Status, err)
	}
}

func readUnassignedRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool, nodeDeviceID string) domain.UnassignedEdgeNode {
	t.Helper()
	v, err := scanUnassigned(pool.QueryRow(ctx, `
SELECT id,node_device_id,claimed_cloud_url,display_name,app_version,status,first_seen_at,last_seen_at,
  COALESCE(assigned_restaurant_id,''),assigned_at,created_at,updated_at
FROM cloud_unassigned_edge_nodes
WHERE node_device_id = $1`, nodeDeviceID))
	if err != nil {
		t.Fatalf("read unassigned row node=%q: %v", nodeDeviceID, err)
	}
	return v
}

func assertPairingStatus(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id string, wantStatus domain.PairingCodeStatus, wantUpdatedAt time.Time) {
	t.Helper()
	var status string
	var updatedAt time.Time
	if err := pool.QueryRow(ctx, `SELECT status,updated_at FROM cloud_pairing_codes WHERE id = $1`, id).Scan(&status, &updatedAt); err != nil {
		t.Fatalf("read pairing status id=%q: %v", id, err)
	}
	if domain.PairingCodeStatus(status) != wantStatus || !updatedAt.Equal(wantUpdatedAt) {
		t.Fatalf("unexpected pairing status id=%q status=%q updated_at=%s", id, status, updatedAt)
	}
}

func assertEdgeNodeEqual(t *testing.T, got, want domain.EdgeNode) {
	t.Helper()
	if got.ID != want.ID || got.RestaurantID != want.RestaurantID || got.NodeDeviceID != want.NodeDeviceID || got.DisplayName != want.DisplayName || got.Status != want.Status || got.CredentialsHash != want.CredentialsHash {
		t.Fatalf("edge node mismatch id=%q node=%q status=%q", got.ID, got.NodeDeviceID, got.Status)
	}
	assertTimePtrValueEqual(t, got.LastSeenAt, want.LastSeenAt, "last_seen_at")
	assertTimePtrValueEqual(t, got.AssignedAt, want.AssignedAt, "assigned_at")
	assertTimePtrValueEqual(t, got.RevokedAt, want.RevokedAt, "revoked_at")
	assertTimeEqual(t, got.CreatedAt, want.CreatedAt, "created_at")
	assertTimeEqual(t, got.UpdatedAt, want.UpdatedAt, "updated_at")
}

func assertTimeEqual(t *testing.T, got, want time.Time, field string) {
	t.Helper()
	if !got.Equal(want) {
		t.Fatalf("expected %s=%s, got %s", field, want, got)
	}
}

func assertTimePtrEqual(t *testing.T, got *time.Time, want time.Time, field string) {
	t.Helper()
	if got == nil {
		t.Fatalf("expected %s=%s, got nil", field, want)
	}
	if !got.Equal(want) {
		t.Fatalf("expected %s=%s, got %s", field, want, *got)
	}
}

func assertTimePtrValueEqual(t *testing.T, got, want *time.Time, field string) {
	t.Helper()
	if got == nil || want == nil {
		if got != want {
			t.Fatalf("expected %s nil equality, got_nil=%t want_nil=%t", field, got == nil, want == nil)
		}
		return
	}
	if !got.Equal(*want) {
		t.Fatalf("expected %s=%s, got %s", field, *want, *got)
	}
}

func assertErrorDoesNotContain(t *testing.T, err error, marker string) {
	t.Helper()
	if err != nil && strings.Contains(err.Error(), marker) {
		t.Fatalf("error leaked plaintext marker")
	}
}

func ptrTime(v time.Time) *time.Time {
	return &v
}

func publicPairingFields(v domain.PairingCode) string {
	return strings.Join([]string{v.ID, v.RestaurantID, v.NodeDeviceID, v.CloudURL, string(v.Status)}, "|")
}

func publicEdgeNodeFields(v domain.EdgeNode) string {
	return strings.Join([]string{v.ID, v.RestaurantID, v.NodeDeviceID, v.DisplayName, string(v.Status)}, "|")
}
