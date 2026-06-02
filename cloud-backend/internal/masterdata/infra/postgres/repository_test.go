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

	"cloud-backend/internal/masterdata/domain"
	platformpg "cloud-backend/internal/platform/postgres"
)

func TestCatalogSuggestionsReadAndUpdateAssignmentFields(t *testing.T) {
	ctx := context.Background()
	pool, closeFn := openPostgresWithBaseline(t, ctx)
	defer closeFn()
	repo := NewRepository(pool)

	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	assignedAt := now.Add(10 * time.Minute)
	if _, err := pool.Exec(ctx, `
INSERT INTO cloud_catalog_suggestions(
  id,suggestion_id,restaurant_id,catalog_item_id,proposal_group_id,action,reason,status,
  assigned_to_employee_id,assigned_by_employee_id,assigned_at,assignment_note,
  suggested_at,cloud_received_at,payload_json,created_at,updated_at
) VALUES (
  'catalog-review-1','catalog-suggestion-1','restaurant-1',NULL,NULL,'create','','pending',
  'employee-1','manager-1',$1,'initial assignment',
  $2,$3,'{}'::jsonb,$4,$5
)`, assignedAt, now, now.Add(time.Minute), now, now); err != nil {
		t.Fatal(err)
	}

	list, err := repo.ListCatalogSuggestions(ctx, "restaurant-1", "pending", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one catalog suggestion, got %+v", list)
	}
	assertCatalogAssignment(t, list[0], "employee-1", "manager-1", assignedAt, "initial assignment")
	if list[0].CatalogItemID != "" || list[0].ProposalGroupID != "" {
		t.Fatalf("expected nullable catalog fields to round-trip as empty strings, got %+v", list[0])
	}

	got, err := repo.GetCatalogSuggestion(ctx, "catalog-review-1")
	if err != nil {
		t.Fatal(err)
	}
	assertCatalogAssignment(t, got, "employee-1", "manager-1", assignedAt, "initial assignment")

	updatedAt := now.Add(30 * time.Minute)
	reassignedAt := now.Add(20 * time.Minute)
	got.Status = domain.SuggestionStatusApproved
	got.ReviewComment = "approved"
	got.ReviewedByEmployeeID = "reviewer-1"
	got.ReviewedAt = &updatedAt
	got.AppliedCatalogItemID = "catalog-item-applied"
	got.AssignedToEmployeeID = "employee-2"
	got.AssignedByEmployeeID = "manager-2"
	got.AssignedAt = &reassignedAt
	got.AssignmentNote = "reassigned for final review"
	got.UpdatedAt = updatedAt

	updated, err := repo.UpdateCatalogSuggestion(ctx, got)
	if err != nil {
		t.Fatal(err)
	}
	assertCatalogAssignment(t, updated, "employee-2", "manager-2", reassignedAt, "reassigned for final review")
	if !updated.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("expected updated_at %s, got %s", updatedAt, updated.UpdatedAt)
	}
}

func TestRecipeSuggestionsReadAndUpdateAssignmentFields(t *testing.T) {
	ctx := context.Background()
	pool, closeFn := openPostgresWithBaseline(t, ctx)
	defer closeFn()
	repo := NewRepository(pool)

	now := time.Date(2026, 5, 20, 11, 0, 0, 0, time.UTC)
	assignedAt := now.Add(10 * time.Minute)
	if _, err := pool.Exec(ctx, `
INSERT INTO cloud_recipe_suggestions(
  id,suggestion_id,restaurant_id,recipe_version_id,owner_catalog_item_id,owner_catalog_suggestion_id,proposal_group_id,
  action,reason,prep_time_delta_minutes,status,assigned_to_employee_id,assigned_by_employee_id,assigned_at,assignment_note,
  suggested_at,cloud_received_at,payload_json,created_at,updated_at
) VALUES (
  'recipe-review-1','recipe-suggestion-1','restaurant-1',NULL,NULL,NULL,NULL,
  'update','',5,'pending','employee-3','manager-3',$1,'recipe assignment',
  $2,$3,'{}'::jsonb,$4,$5
)`, assignedAt, now, now.Add(time.Minute), now, now); err != nil {
		t.Fatal(err)
	}

	list, err := repo.ListRecipeSuggestions(ctx, "restaurant-1", "pending", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one recipe suggestion, got %+v", list)
	}
	assertRecipeAssignment(t, list[0], "employee-3", "manager-3", assignedAt, "recipe assignment")
	if list[0].RecipeVersionID != "" || list[0].OwnerCatalogItemID != "" || list[0].OwnerCatalogSuggestionID != "" || list[0].ProposalGroupID != "" {
		t.Fatalf("expected nullable recipe fields to round-trip as empty strings, got %+v", list[0])
	}

	got, err := repo.GetRecipeSuggestion(ctx, "recipe-review-1")
	if err != nil {
		t.Fatal(err)
	}
	assertRecipeAssignment(t, got, "employee-3", "manager-3", assignedAt, "recipe assignment")

	updatedAt := now.Add(30 * time.Minute)
	reassignedAt := now.Add(20 * time.Minute)
	got.Status = domain.SuggestionStatusChangesRequest
	got.ReviewComment = "please adjust prep steps"
	got.ReviewedByEmployeeID = "reviewer-2"
	got.ReviewedAt = &updatedAt
	got.AssignedToEmployeeID = "employee-4"
	got.AssignedByEmployeeID = "manager-4"
	got.AssignedAt = &reassignedAt
	got.AssignmentNote = "needs recipe owner"
	got.UpdatedAt = updatedAt

	updated, err := repo.UpdateRecipeSuggestion(ctx, got)
	if err != nil {
		t.Fatal(err)
	}
	assertRecipeAssignment(t, updated, "employee-4", "manager-4", reassignedAt, "needs recipe owner")
	if !updated.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("expected updated_at %s, got %s", updatedAt, updated.UpdatedAt)
	}
}

func TestStopListUpdateReviewsReadAndPersistAssignmentFields(t *testing.T) {
	ctx := context.Background()
	pool, closeFn := openPostgresWithBaseline(t, ctx)
	defer closeFn()
	repo := NewRepository(pool)

	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	assignedAt := now.Add(10 * time.Minute)
	if _, err := pool.Exec(ctx, `
INSERT INTO cloud_projection_stop_list_updates(
  source_event_id,queue_id,restaurant_id,device_id,stop_list_id,warehouse_id,catalog_item_id,available_quantity,active,
  conflict_policy,source,reason,projection_action,review_status,assigned_to_employee_id,assigned_by_employee_id,assigned_at,
  assignment_note,updated_at,occurred_at,projected_at
) VALUES (
  'stop-event-1','queue-1','restaurant-1','device-1','stop-list-edge-1',NULL,'catalog-item-1',NULL,true,
  'edge_overlay_requires_manager_review','edge',NULL,'requires_manager_review','pending','employee-5','manager-5',$1,
  'stop-list assignment',$2,$3,$4
)`, assignedAt, now, now.Add(-time.Minute), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	list, err := repo.ListStopListUpdateReviews(ctx, "restaurant-1", "pending", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one stop-list review, got %+v", list)
	}
	assertStopListAssignment(t, list[0], "employee-5", "manager-5", assignedAt, "stop-list assignment")
	if list[0].WarehouseID != "" || list[0].Reason != "" || list[0].AvailableQuantity != nil {
		t.Fatalf("expected nullable stop-list fields to round-trip as empty values, got %+v", list[0])
	}

	got, err := repo.GetStopListUpdateReview(ctx, "stop-event-1")
	if err != nil {
		t.Fatal(err)
	}
	assertStopListAssignment(t, got, "employee-5", "manager-5", assignedAt, "stop-list assignment")

	reviewedAt := now.Add(25 * time.Minute)
	reassignedAt := now.Add(20 * time.Minute)
	updatedAt := now.Add(30 * time.Minute)
	got.Status = domain.SuggestionStatusApproved
	got.ReviewComment = "approved for cloud stop list"
	got.ReviewedByEmployeeID = "reviewer-3"
	got.ReviewedAt = &reviewedAt
	got.AppliedStopListID = "stop-list-cloud-1"
	got.AssignedToEmployeeID = "employee-6"
	got.AssignedByEmployeeID = "manager-6"
	got.AssignedAt = &reassignedAt
	got.AssignmentNote = "final stop-list owner"
	got.UpdatedAt = updatedAt

	updated, err := repo.UpdateStopListUpdateReview(ctx, got)
	if err != nil {
		t.Fatal(err)
	}
	assertStopListAssignment(t, updated, "employee-6", "manager-6", reassignedAt, "final stop-list owner")
	if !updated.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("expected updated_at %s, got %s", updatedAt, updated.UpdatedAt)
	}

	var storedTo, storedBy, storedNote string
	var storedAssignedAt, storedUpdatedAt time.Time
	if err := pool.QueryRow(ctx, `
SELECT assigned_to_employee_id,assigned_by_employee_id,assigned_at,assignment_note,updated_at
FROM cloud_projection_stop_list_updates
WHERE source_event_id = 'stop-event-1'`).Scan(&storedTo, &storedBy, &storedAssignedAt, &storedNote, &storedUpdatedAt); err != nil {
		t.Fatal(err)
	}
	if storedTo != "employee-6" || storedBy != "manager-6" || storedNote != "final stop-list owner" || !storedAssignedAt.Equal(reassignedAt) || !storedUpdatedAt.Equal(updatedAt) {
		t.Fatalf("unexpected stored stop-list assignment: to=%q by=%q assigned_at=%s note=%q updated_at=%s", storedTo, storedBy, storedAssignedAt, storedNote, storedUpdatedAt)
	}
}

func TestReviewAssignmentAuditEventAppendGetAndNotFound(t *testing.T) {
	ctx := context.Background()
	pool, closeFn := openPostgresWithBaseline(t, ctx)
	defer closeFn()
	repo := NewRepository(pool)

	occurredAt := time.Date(2026, 5, 20, 13, 0, 0, 0, time.UTC)
	event := domain.ReviewAssignmentAuditEvent{
		EventID:          "assignment-event-1",
		CommandID:        "assignment-command-1",
		ReviewType:       "stop_list_update",
		ReviewID:         "stop-event-1",
		Action:           "assigned",
		ActorEmployeeID:  "manager-7",
		TargetEmployeeID: "employee-7",
		Reason:           "manual assignment",
		OccurredAt:       occurredAt,
	}
	if err := repo.AppendReviewAssignmentAuditEvent(ctx, event); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM cloud_review_assignment_audit_events WHERE event_id = $1`, event.EventID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected appended audit event, got count %d", count)
	}

	got, err := repo.GetReviewAssignmentAuditEvent(ctx, event.CommandID)
	if err != nil {
		t.Fatal(err)
	}
	if got.EventID != event.EventID ||
		got.CommandID != event.CommandID ||
		got.ReviewType != event.ReviewType ||
		got.ReviewID != event.ReviewID ||
		got.Action != event.Action ||
		got.ActorEmployeeID != event.ActorEmployeeID ||
		got.TargetEmployeeID != event.TargetEmployeeID ||
		got.Reason != event.Reason ||
		!got.OccurredAt.Equal(event.OccurredAt) {
		t.Fatalf("unexpected audit event: got %+v want %+v", got, event)
	}

	_, err = repo.GetReviewAssignmentAuditEvent(ctx, "missing-command")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing audit event, got %v", err)
	}
}

func TestReviewAssignmentAuditEventsListFiltersAndSortsNewestFirst(t *testing.T) {
	ctx := context.Background()
	pool, closeFn := openPostgresWithBaseline(t, ctx)
	defer closeFn()
	repo := NewRepository(pool)

	base := time.Date(2026, 5, 20, 13, 0, 0, 0, time.UTC)
	events := []domain.ReviewAssignmentAuditEvent{
		{
			EventID:          "assignment-event-001",
			CommandID:        "assignment-command-001",
			ReviewType:       "stop_list_update",
			ReviewID:         "stop-event-list-1",
			Action:           "assigned",
			ActorEmployeeID:  "manager-1",
			TargetEmployeeID: "employee-1",
			Reason:           "old owner",
			OccurredAt:       base,
		},
		{
			EventID:          "assignment-event-002",
			CommandID:        "assignment-command-002",
			ReviewType:       "stop_list_update",
			ReviewID:         "stop-event-list-1",
			Action:           "unassigned",
			ActorEmployeeID:  "manager-2",
			TargetEmployeeID: "employee-1",
			Reason:           "same timestamp lower id",
			OccurredAt:       base.Add(time.Hour),
		},
		{
			EventID:          "assignment-event-003",
			CommandID:        "assignment-command-003",
			ReviewType:       "stop_list_update",
			ReviewID:         "stop-event-list-1",
			Action:           "assigned",
			ActorEmployeeID:  "manager-3",
			TargetEmployeeID: "employee-3",
			Reason:           "same timestamp higher id",
			OccurredAt:       base.Add(time.Hour),
		},
		{
			EventID:          "assignment-event-other",
			CommandID:        "assignment-command-other",
			ReviewType:       "stop_list_update",
			ReviewID:         "stop-event-list-other",
			Action:           "assigned",
			ActorEmployeeID:  "manager-other",
			TargetEmployeeID: "employee-other",
			Reason:           "must be filtered",
			OccurredAt:       base.Add(2 * time.Hour),
		},
		{
			EventID:          "assignment-event-catalog-same-review",
			CommandID:        "assignment-command-catalog-same-review",
			ReviewType:       "catalog_suggestion",
			ReviewID:         "stop-event-list-1",
			Action:           "assigned",
			ActorEmployeeID:  "manager-catalog",
			TargetEmployeeID: "employee-catalog",
			Reason:           "must be filtered by type",
			OccurredAt:       base.Add(3 * time.Hour),
		},
		{
			EventID:          "assignment-event-recipe",
			CommandID:        "assignment-command-recipe",
			ReviewType:       "recipe_suggestion",
			ReviewID:         "recipe-event-list-1",
			Action:           "assigned",
			ActorEmployeeID:  "manager-recipe",
			TargetEmployeeID: "employee-recipe",
			Reason:           "recipe assignment audit",
			OccurredAt:       base.Add(4 * time.Hour),
		},
	}
	for _, event := range events {
		if err := repo.AppendReviewAssignmentAuditEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}

	got, err := repo.ListReviewAssignmentAuditEvents(ctx, "stop_list_update", "stop-event-list-1", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected three filtered audit events, got %+v", got)
	}
	wantOrder := []string{"assignment-event-003", "assignment-event-002", "assignment-event-001"}
	for i, want := range wantOrder {
		if got[i].EventID != want || got[i].ReviewID != "stop-event-list-1" || got[i].ReviewType != "stop_list_update" {
			t.Fatalf("unexpected event at %d: got %+v want event_id=%s", i, got[i], want)
		}
	}

	paged, err := repo.ListReviewAssignmentAuditEvents(ctx, "stop_list_update", "stop-event-list-1", 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(paged) != 2 || paged[0].EventID != "assignment-event-002" || paged[1].EventID != "assignment-event-001" {
		t.Fatalf("unexpected paged audit events: %+v", paged)
	}

	catalog, err := repo.ListReviewAssignmentAuditEvents(ctx, "catalog_suggestion", "stop-event-list-1", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog) != 1 || catalog[0].EventID != "assignment-event-catalog-same-review" {
		t.Fatalf("expected catalog audit to be filtered by review type and id, got %+v", catalog)
	}

	recipe, err := repo.ListReviewAssignmentAuditEvents(ctx, "recipe_suggestion", "recipe-event-list-1", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(recipe) != 1 || recipe[0].EventID != "assignment-event-recipe" {
		t.Fatalf("expected recipe audit to be filtered by review type and id, got %+v", recipe)
	}
}

func openPostgresIntegrationPool(t *testing.T) *pgxpool.Pool {
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
	lockPostgresIntegration(t, t.Context(), pool)
	return pool
}

func openPostgresWithBaseline(t *testing.T, ctx context.Context) (*pgxpool.Pool, func()) {
	t.Helper()
	pool := openPostgresIntegrationPool(t)
	resetPublicSchema(t, ctx, pool)
	if err := platformpg.MigrateDirWithPolicy(ctx, pool, actualMigrationsDir(), platformpg.MigrationOptions{
		ModuleName:         "cloud-backend",
		ModuleVersion:      "0.1.0",
		BackupDir:          t.TempDir(),
		SchemaRequirements: masterdataReviewSchemaRequirements(),
	}); err != nil {
		t.Fatalf("postgres baseline migration failed: %v", err)
	}
	return pool, func() {
		resetPublicSchema(t, context.Background(), pool)
	}
}

func actualMigrationsDir() string {
	return filepath.Join("..", "..", "..", "..", "migrations", "postgres")
}

func masterdataReviewSchemaRequirements() []platformpg.SchemaRequirement {
	return []platformpg.SchemaRequirement{
		{
			Table:         "cloud_catalog_suggestions",
			RequiredBy:    "masterdata postgres repository suggestion assignment tests",
			MigrationFile: "001_init.sql",
			Columns: []string{
				"id", "suggestion_id", "restaurant_id", "catalog_item_id", "proposal_group_id", "status",
				"assigned_to_employee_id", "assigned_by_employee_id", "assigned_at", "assignment_note", "updated_at",
			},
		},
		{
			Table:         "cloud_recipe_suggestions",
			RequiredBy:    "masterdata postgres repository recipe assignment tests",
			MigrationFile: "001_init.sql",
			Columns: []string{
				"id", "suggestion_id", "restaurant_id", "recipe_version_id", "owner_catalog_item_id", "owner_catalog_suggestion_id",
				"proposal_group_id", "status", "assigned_to_employee_id", "assigned_by_employee_id", "assigned_at", "assignment_note", "updated_at",
			},
		},
		{
			Table:         "cloud_projection_stop_list_updates",
			RequiredBy:    "masterdata postgres repository stop-list assignment tests",
			MigrationFile: "001_init.sql",
			Columns: []string{
				"source_event_id", "restaurant_id", "warehouse_id", "reason", "projection_action", "review_status",
				"assigned_to_employee_id", "assigned_by_employee_id", "assigned_at", "assignment_note", "updated_at",
			},
		},
		{
			Table:         "cloud_review_assignment_audit_events",
			RequiredBy:    "masterdata postgres repository assignment audit tests",
			MigrationFile: "001_init.sql",
			Columns: []string{
				"event_id", "command_id", "review_type", "review_id", "action", "actor_employee_id", "target_employee_id", "reason", "occurred_at",
			},
		},
	}
}

func resetPublicSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatalf("reset public schema: %v", err)
	}
}

func lockPostgresIntegration(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
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

func assertCatalogAssignment(t *testing.T, got domain.CatalogSuggestion, wantTo, wantBy string, wantAt time.Time, wantNote string) {
	t.Helper()
	if got.AssignedToEmployeeID != wantTo || got.AssignedByEmployeeID != wantBy || got.AssignmentNote != wantNote {
		t.Fatalf("unexpected catalog assignment: %+v", got)
	}
	assertTimePtrEqual(t, got.AssignedAt, wantAt)
}

func assertRecipeAssignment(t *testing.T, got domain.RecipeSuggestion, wantTo, wantBy string, wantAt time.Time, wantNote string) {
	t.Helper()
	if got.AssignedToEmployeeID != wantTo || got.AssignedByEmployeeID != wantBy || got.AssignmentNote != wantNote {
		t.Fatalf("unexpected recipe assignment: %+v", got)
	}
	assertTimePtrEqual(t, got.AssignedAt, wantAt)
}

func assertStopListAssignment(t *testing.T, got domain.StopListUpdateReview, wantTo, wantBy string, wantAt time.Time, wantNote string) {
	t.Helper()
	if got.AssignedToEmployeeID != wantTo || got.AssignedByEmployeeID != wantBy || got.AssignmentNote != wantNote {
		t.Fatalf("unexpected stop-list assignment: %+v", got)
	}
	assertTimePtrEqual(t, got.AssignedAt, wantAt)
}

func assertTimePtrEqual(t *testing.T, got *time.Time, want time.Time) {
	t.Helper()
	if got == nil {
		t.Fatalf("expected time %s, got nil", want)
	}
	if !got.Equal(want) {
		t.Fatalf("expected time %s, got %s", want, *got)
	}
}
