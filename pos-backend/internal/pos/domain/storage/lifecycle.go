package storage

import "time"

// SQLiteDatabaseStats описывает безопасные page-level метрики SQLite без чтения файловой системы.
type SQLiteDatabaseStats struct {
	PageCount          int64  `json:"page_count"`
	PageSizeBytes      int64  `json:"page_size_bytes"`
	FreelistCount      int64  `json:"freelist_count"`
	EstimatedSizeBytes int64  `json:"estimated_size_bytes"`
	FreelistBytes      int64  `json:"freelist_bytes"`
	JournalMode        string `json:"journal_mode"`
}

// TableCounts содержит high-level объемы runtime tables, влияющих на рост local Edge DB.
type TableCounts struct {
	Orders                  int `json:"orders"`
	OpenOrders              int `json:"open_orders"`
	LockedOrders            int `json:"locked_orders"`
	ClosedOrders            int `json:"closed_orders"`
	CancelledOrders         int `json:"cancelled_orders"`
	OrderLines              int `json:"order_lines"`
	OrderLineModifiers      int `json:"order_line_modifiers"`
	Prechecks               int `json:"prechecks"`
	PrecheckLines           int `json:"precheck_lines"`
	PrecheckLineModifiers   int `json:"precheck_line_modifiers"`
	PrecheckDiscounts       int `json:"precheck_discounts"`
	PrecheckSurcharges      int `json:"precheck_surcharges"`
	PrecheckTaxes           int `json:"precheck_taxes"`
	Payments                int `json:"payments"`
	PaymentAttempts         int `json:"payment_attempts"`
	Checks                  int `json:"checks"`
	FinancialOperations     int `json:"financial_operations"`
	FinancialOperationItems int `json:"financial_operation_items"`
	Shifts                  int `json:"shifts"`
	OpenShifts              int `json:"open_shifts"`
	CashSessions            int `json:"cash_sessions"`
	OpenCashSessions        int `json:"open_cash_sessions"`
	LocalEvents             int `json:"local_events"`
	OutboxMessages          int `json:"outbox_messages"`
}

// BusinessDateRange показывает фактический диапазон business_date_local у закрытых чеков.
type BusinessDateRange struct {
	Oldest string `json:"oldest,omitempty"`
	Newest string `json:"newest,omitempty"`
}

// ClosedOrdersBusinessDateCount агрегирует закрытые заказы по business date для оценки retention window.
type ClosedOrdersBusinessDateCount struct {
	BusinessDateLocal string `json:"business_date_local"`
	Orders            int    `json:"orders"`
	Checks            int    `json:"checks"`
	TotalAmount       int64  `json:"total_amount"`
	FirstClosedAt     string `json:"first_closed_at,omitempty"`
	LastClosedAt      string `json:"last_closed_at,omitempty"`
}

// OutboxStatusCount показывает outbox объемы по status/direction без раскрытия payload.
type OutboxStatusCount struct {
	Status        string `json:"status"`
	SyncDirection string `json:"sync_direction"`
	Count         int    `json:"count"`
}

// RetentionCapability фиксирует, что текущий runtime умеет только безопасный status/dry-run.
type RetentionCapability struct {
	Mode                        string `json:"mode"`
	DestructiveApplySupported   bool   `json:"destructive_apply_supported"`
	FinancialLedgerProtected    bool   `json:"financial_ledger_protected"`
	ImmutableSnapshotsProtected bool   `json:"immutable_snapshots_protected"`
	Reason                      string `json:"reason"`
}

// LifecycleStatus является read-only снимком объема локальной POS Edge SQLite БД.
type LifecycleStatus struct {
	GeneratedAt                  time.Time                       `json:"generated_at"`
	SQLite                       SQLiteDatabaseStats             `json:"sqlite"`
	Tables                       TableCounts                     `json:"tables"`
	ClosedOrderBusinessDateRange BusinessDateRange               `json:"closed_order_business_date_range"`
	ClosedOrdersByBusinessDate   []ClosedOrdersBusinessDateCount `json:"closed_orders_by_business_date"`
	Outbox                       []OutboxStatusCount             `json:"outbox"`
	BlockingOutboxMessages       int                             `json:"blocking_outbox_messages"`
	Retention                    RetentionCapability             `json:"retention"`
}

// RetentionEligibleCounts считает документы старше cutoff, которые могли бы войти в будущую архивную стратегию.
type RetentionEligibleCounts struct {
	ClosedOrders            int `json:"closed_orders"`
	Checks                  int `json:"checks"`
	Prechecks               int `json:"prechecks"`
	PrecheckLines           int `json:"precheck_lines"`
	PrecheckLineModifiers   int `json:"precheck_line_modifiers"`
	PrecheckDiscounts       int `json:"precheck_discounts"`
	PrecheckSurcharges      int `json:"precheck_surcharges"`
	PrecheckTaxes           int `json:"precheck_taxes"`
	Payments                int `json:"payments"`
	PaymentAttempts         int `json:"payment_attempts"`
	OrderLines              int `json:"order_lines"`
	OrderLineModifiers      int `json:"order_line_modifiers"`
	FinancialOperations     int `json:"financial_operations"`
	FinancialOperationItems int `json:"financial_operation_items"`
}

// ArchiveExportCounts считает строки, включенные в export-only архив закрытых заказов.
type ArchiveExportCounts struct {
	ClosedOrders            int `json:"closed_orders"`
	OrderLines              int `json:"order_lines"`
	OrderLineModifiers      int `json:"order_line_modifiers"`
	OrderLineDiscounts      int `json:"order_line_discounts"`
	OrderSurcharges         int `json:"order_surcharges"`
	Prechecks               int `json:"prechecks"`
	PrecheckLines           int `json:"precheck_lines"`
	PrecheckLineModifiers   int `json:"precheck_line_modifiers"`
	PrecheckDiscounts       int `json:"precheck_discounts"`
	PrecheckSurcharges      int `json:"precheck_surcharges"`
	PrecheckTaxes           int `json:"precheck_taxes"`
	Payments                int `json:"payments"`
	PaymentAttempts         int `json:"payment_attempts"`
	Checks                  int `json:"checks"`
	FinancialOperations     int `json:"financial_operations"`
	FinancialOperationItems int `json:"financial_operation_items"`
	LocalEventReferences    int `json:"local_event_references"`
	OutboxMessageReferences int `json:"outbox_message_references"`
	BlockingOutboxMessages  int `json:"blocking_outbox_messages"`
	ArchivedRows            int `json:"archived_rows"`
}

// ArchiveSourceMetadata описывает доступные runtime metadata активной SQLite БД.
type ArchiveSourceMetadata struct {
	SQLite             SQLiteDatabaseStats  `json:"sqlite"`
	SourceNodeDeviceID string               `json:"source_node_device_id,omitempty"`
	SourceDeviceCode   string               `json:"source_device_code,omitempty"`
	SourceDeviceName   string               `json:"source_device_name,omitempty"`
	SourceDeviceType   string               `json:"source_device_type,omitempty"`
	SourcePairedAt     string               `json:"source_paired_at,omitempty"`
	RuntimeVersions    []ArchiveMetadataRow `json:"runtime_versions,omitempty"`
	SchemaMigrations   []ArchiveMetadataRow `json:"schema_migrations,omitempty"`
}

// ArchiveMetadataRow хранит безопасный metadata row без business payload.
type ArchiveMetadataRow map[string]any

// ArchiveExportRow является одной строкой typed JSONL archive.
type ArchiveExportRow struct {
	Table string         `json:"table"`
	Row   map[string]any `json:"row"`
}

// ArchiveTableManifest описывает состав typed JSONL archive по таблицам.
type ArchiveTableManifest struct {
	Name            string `json:"name"`
	Rows            int    `json:"rows"`
	PayloadIncluded bool   `json:"payload_included"`
	Content         string `json:"content"`
}

// ArchiveManifest является проверяемым JSON manifest для export-only archive.
type ArchiveManifest struct {
	Version                     string                 `json:"version"`
	Mode                        string                 `json:"mode"`
	ResultMode                  string                 `json:"result_mode"`
	DestructiveApplySupported   bool                   `json:"destructive_apply_supported"`
	RuntimeRowsDeleted          bool                   `json:"runtime_rows_deleted"`
	GeneratedAt                 time.Time              `json:"generated_at"`
	ArchiveID                   string                 `json:"archive_id"`
	CutoffBusinessDateLocal     string                 `json:"cutoff_business_date_local"`
	Reason                      string                 `json:"reason,omitempty"`
	ArchivePath                 string                 `json:"archive_path"`
	ManifestPath                string                 `json:"manifest_path"`
	SHA256                      string                 `json:"sha256"`
	Counts                      ArchiveExportCounts    `json:"counts"`
	BusinessDateRange           BusinessDateRange      `json:"business_date_range"`
	Tables                      []ArchiveTableManifest `json:"tables"`
	Source                      ArchiveSourceMetadata  `json:"source"`
	Blocked                     bool                   `json:"blocked"`
	BlockReasons                []string               `json:"block_reasons"`
	FinancialLedgerProtected    bool                   `json:"financial_ledger_protected"`
	ImmutableSnapshotsProtected bool                   `json:"immutable_snapshots_protected"`
	ExportCreated               bool                   `json:"export_created"`
}

// ArchiveExportResult возвращается HTTP API после создания export-only archive.
type ArchiveExportResult = ArchiveManifest

// ArchiveExportScope содержит read-only выборку и metadata для записи archive файлов.
type ArchiveExportScope struct {
	Counts            ArchiveExportCounts
	BusinessDateRange BusinessDateRange
	Source            ArchiveSourceMetadata
	BlockReasons      []string
	Blocked           bool
	Rows              []ArchiveExportRow
}

// ArchiveApplyRuntimeScope содержит только read-only runtime blockers и counts для будущего apply.
type ArchiveApplyRuntimeScope struct {
	Counts                 ArchiveExportCounts
	ActiveOrders           int
	OpenShifts             int
	OpenCashSessions       int
	BlockingOutboxMessages int
}

// ArchiveOpenOperationalBoundaries агрегирует открытые runtime boundaries,
// которые блокируют будущий destructive apply/delete/compaction.
type ArchiveOpenOperationalBoundaries struct {
	ActiveOrders     int  `json:"active_orders"`
	OpenShifts       int  `json:"open_shifts"`
	OpenCashSessions int  `json:"open_cash_sessions"`
	Open             bool `json:"open"`
}

// ArchiveVerificationSummary описывает результат streaming-проверки archive JSONL.
type ArchiveVerificationSummary struct {
	ArchivePath            string              `json:"archive_path,omitempty"`
	ManifestPath           string              `json:"manifest_path,omitempty"`
	ArchiveSHA256          string              `json:"archive_sha256,omitempty"`
	ComputedSHA256         string              `json:"computed_sha256,omitempty"`
	Counts                 ArchiveExportCounts `json:"counts"`
	BusinessDateRange      BusinessDateRange   `json:"business_date_range"`
	ArchiveExists          bool                `json:"archive_exists"`
	ManifestExists         bool                `json:"manifest_exists"`
	ManifestVersionMatched bool                `json:"manifest_version_matched"`
	SHA256Matched          bool                `json:"sha256_matched"`
	CountsMatchedManifest  bool                `json:"counts_matched_manifest"`
	SnapshotPayloadPresent bool                `json:"snapshot_payload_present"`
	IdentityFieldsPresent  bool                `json:"identity_fields_present"`
	BusinessDateConsistent bool                `json:"business_date_consistent"`
	RuntimeRowsNotDeleted  bool                `json:"runtime_rows_not_deleted"`
	PayloadPolicyPreserved bool                `json:"payload_policy_preserved"`
}

// ArchiveVerifyResult возвращает явный integrity verdict для export-only archive artifact.
type ArchiveVerifyResult struct {
	GeneratedAt               time.Time                  `json:"generated_at"`
	Valid                     bool                       `json:"valid"`
	Warnings                  []string                   `json:"warnings,omitempty"`
	Errors                    []string                   `json:"errors,omitempty"`
	ArchiveID                 string                     `json:"archive_id,omitempty"`
	CutoffBusinessDateLocal   string                     `json:"cutoff_business_date_local,omitempty"`
	ArchivePath               string                     `json:"archive_path,omitempty"`
	ManifestPath              string                     `json:"manifest_path,omitempty"`
	RuntimeRowsDeleted        bool                       `json:"runtime_rows_deleted"`
	DestructiveApplySupported bool                       `json:"destructive_apply_supported"`
	Counts                    ArchiveExportCounts        `json:"counts"`
	BusinessDateRange         BusinessDateRange          `json:"business_date_range"`
	Tables                    []ArchiveTableManifest     `json:"tables,omitempty"`
	Verification              ArchiveVerificationSummary `json:"verification"`
}

// ArchiveReadPlanClosedOrder содержит bounded preview archived closed order без восстановления в runtime SQLite.
type ArchiveReadPlanClosedOrder struct {
	OrderID                 string                     `json:"order_id"`
	CheckID                 string                     `json:"check_id"`
	PrecheckID              string                     `json:"precheck_id,omitempty"`
	BusinessDateLocal       string                     `json:"business_date_local,omitempty"`
	ClosedAt                string                     `json:"closed_at,omitempty"`
	CurrencyCode            string                     `json:"currency_code,omitempty"`
	Total                   int64                      `json:"total"`
	DocumentState           string                     `json:"document_state"`
	RuntimeRestored         bool                       `json:"runtime_restored"`
	CheckSnapshotPresent    bool                       `json:"check_snapshot_present"`
	PrecheckSnapshotPresent bool                       `json:"precheck_snapshot_present"`
	RelatedCounts           ArchiveLookupRelatedCounts `json:"related_counts"`
}

// ArchiveReadPlan описывает non-destructive проверку archive artifact без чтения business payload наружу.
type ArchiveReadPlan struct {
	GeneratedAt             time.Time                    `json:"generated_at"`
	ResultMode              string                       `json:"result_mode"`
	Blocked                 bool                         `json:"blocked"`
	BlockReasons            []string                     `json:"block_reasons,omitempty"`
	ArchiveID               string                       `json:"archive_id,omitempty"`
	CutoffBusinessDateLocal string                       `json:"cutoff_business_date_local,omitempty"`
	ArchiveSHA256           string                       `json:"archive_sha256,omitempty"`
	ComputedSHA256          string                       `json:"computed_sha256,omitempty"`
	Counts                  ArchiveExportCounts          `json:"counts"`
	BusinessDateRange       BusinessDateRange            `json:"business_date_range"`
	Tables                  []ArchiveTableManifest       `json:"tables"`
	Verification            ArchiveVerificationSummary   `json:"verification"`
	Limit                   int                          `json:"limit"`
	Offset                  int                          `json:"offset"`
	Returned                int                          `json:"returned"`
	RuntimeRowsDeleted      bool                         `json:"runtime_rows_deleted"`
	RuntimeRestored         bool                         `json:"runtime_restored"`
	PayloadPolicy           string                       `json:"payload_policy"`
	ArchivedClosedOrders    []ArchiveReadPlanClosedOrder `json:"archived_closed_orders"`
}

// ArchiveLookupKey фиксирует разрешенный способ поиска archived preview без произвольных table names.
type ArchiveLookupKey struct {
	CheckID string `json:"check_id,omitempty"`
	OrderID string `json:"order_id,omitempty"`
	Found   bool   `json:"found"`
}

// ArchiveLookupDocument содержит immutable snapshot одного archived документа.
type ArchiveLookupDocument struct {
	ID                string         `json:"id"`
	BusinessDateLocal string         `json:"business_date_local,omitempty"`
	Snapshot          map[string]any `json:"snapshot"`
}

// ArchiveLookupRelatedCounts содержит безопасные связанные счетчики archived graph.
type ArchiveLookupRelatedCounts struct {
	OrderLines              int `json:"order_lines"`
	Payments                int `json:"payments"`
	FinancialOperations     int `json:"financial_operations"`
	FinancialOperationItems int `json:"financial_operation_items"`
}

// ArchiveLookupPreview возвращает audit preview archived check/order без восстановления в runtime SQLite.
type ArchiveLookupPreview struct {
	GeneratedAt   time.Time                  `json:"generated_at"`
	ResultMode    string                     `json:"result_mode"`
	Blocked       bool                       `json:"blocked,omitempty"`
	BlockReasons  []string                   `json:"block_reasons,omitempty"`
	ArchiveID     string                     `json:"archive_id,omitempty"`
	Lookup        ArchiveLookupKey           `json:"lookup"`
	Check         *ArchiveLookupDocument     `json:"check,omitempty"`
	Precheck      *ArchiveLookupDocument     `json:"precheck,omitempty"`
	RelatedCounts ArchiveLookupRelatedCounts `json:"related_counts"`
	Verification  ArchiveVerificationSummary `json:"verification,omitempty"`
}

// ArchivePlanProtectedFlags фиксирует таблицы, которые future archive apply не может менять без отдельной политики.
type ArchivePlanProtectedFlags struct {
	FinancialLedgerProtected    bool `json:"financial_ledger_protected"`
	ImmutableSnapshotsProtected bool `json:"immutable_snapshots_protected"`
	LocalEventsProtected        bool `json:"local_events_protected"`
	OutboxProtected             bool `json:"outbox_protected"`
}

// ArchivePlanTableManifest описывает deterministic manifest entry без чтения business payload.
type ArchivePlanTableManifest struct {
	Name     string `json:"name"`
	Rows     int    `json:"rows"`
	KeyField string `json:"key_field"`
}

// ArchivePlanManifest является manifest-only описанием future archive/export scope.
type ArchivePlanManifest struct {
	FormatVersion           string                     `json:"format_version"`
	RestaurantID            string                     `json:"restaurant_id,omitempty"`
	BusinessDateRange       BusinessDateRange          `json:"business_date_range"`
	CutoffBusinessDateLocal string                     `json:"cutoff_business_date_local"`
	Tables                  []ArchivePlanTableManifest `json:"tables"`
}

// ArchiveExportPlan описывает безопасный export-plan без записи archive files и без мутации runtime rows.
type ArchiveExportPlan struct {
	GeneratedAt               time.Time                 `json:"generated_at"`
	CutoffBusinessDateLocal   string                    `json:"cutoff_business_date_local"`
	Mode                      string                    `json:"mode"`
	ResultMode                string                    `json:"result_mode"`
	DestructiveApplySupported bool                      `json:"destructive_apply_supported"`
	Blocked                   bool                      `json:"blocked"`
	BlockReasons              []string                  `json:"block_reasons"`
	ArchiveSet                RetentionEligibleCounts   `json:"archive_set"`
	Protected                 ArchivePlanProtectedFlags `json:"protected"`
	ActiveOrders              int                       `json:"active_orders"`
	OpenShifts                int                       `json:"open_shifts"`
	OpenCashSessions          int                       `json:"open_cash_sessions"`
	BlockingOutboxMessages    int                       `json:"blocking_outbox_messages"`
	Manifest                  ArchivePlanManifest       `json:"manifest"`
}

// RetentionDryRunResult описывает read-only оценку cutoff без каких-либо удалений или архивных записей.
type RetentionDryRunResult struct {
	GeneratedAt                 time.Time               `json:"generated_at"`
	CutoffBusinessDateLocal     string                  `json:"cutoff_business_date_local"`
	Mode                        string                  `json:"mode"`
	ResultMode                  string                  `json:"result_mode"`
	DestructiveApplySupported   bool                    `json:"destructive_apply_supported"`
	Blocked                     bool                    `json:"blocked"`
	BlockReasons                []string                `json:"block_reasons"`
	Eligible                    RetentionEligibleCounts `json:"eligible"`
	ActiveOrders                int                     `json:"active_orders"`
	OpenShifts                  int                     `json:"open_shifts"`
	OpenCashSessions            int                     `json:"open_cash_sessions"`
	BlockingOutboxMessages      int                     `json:"blocking_outbox_messages"`
	FinancialLedgerProtected    bool                    `json:"financial_ledger_protected"`
	ImmutableSnapshotsProtected bool                    `json:"immutable_snapshots_protected"`
}

// ArchiveApplyPlan описывает blocked-by-default planning/verification для будущего destructive apply.
type ArchiveApplyPlan struct {
	GeneratedAt               time.Time                  `json:"generated_at"`
	CutoffBusinessDateLocal   string                     `json:"cutoff_business_date_local"`
	ArchiveID                 string                     `json:"archive_id,omitempty"`
	ArchiveSHA256             string                     `json:"archive_sha256,omitempty"`
	ResultMode                string                     `json:"result_mode"`
	Mode                      string                     `json:"mode"`
	DestructiveApplySupported bool                       `json:"destructive_apply_supported"`
	RuntimeRowsDeleted        bool                       `json:"runtime_rows_deleted"`
	Blocked                   bool                       `json:"blocked"`
	BlockReasons              []string                   `json:"block_reasons"`
	EligibleCounts            ArchiveExportCounts        `json:"eligible_counts"`
	ArchiveCounts             ArchiveExportCounts        `json:"archive_counts"`
	Protected                 ArchivePlanProtectedFlags  `json:"protected"`
	ActiveOrders              int                        `json:"active_orders"`
	OpenShifts                int                        `json:"open_shifts"`
	OpenCashSessions          int                        `json:"open_cash_sessions"`
	BlockingOutboxMessages    int                        `json:"blocking_outbox_messages"`
	Verification              ArchiveVerificationSummary `json:"verification"`
}

// ArchiveApplyReadiness описывает отдельный read-only policy gate для будущего
// destructive archive apply/delete/compaction без смешения с apply-plan.
type ArchiveApplyReadiness struct {
	GeneratedAt                    time.Time                         `json:"generated_at"`
	CutoffBusinessDateLocal        string                            `json:"cutoff_business_date_local"`
	ArchiveID                      string                            `json:"archive_id,omitempty"`
	ArchiveSHA256                  string                            `json:"archive_sha256,omitempty"`
	ResultMode                     string                            `json:"result_mode"`
	DestructiveApplySupported      bool                              `json:"destructive_apply_supported"`
	ReadyForDestructiveApply       bool                              `json:"ready_for_destructive_apply"`
	RuntimeRowsDeleted             bool                              `json:"runtime_rows_deleted"`
	ArchiveVerified                bool                              `json:"archive_verified"`
	ManifestVerified               bool                              `json:"manifest_verified"`
	SnapshotPayloadVerified        bool                              `json:"snapshot_payload_verified"`
	RuntimeScopeVerified           bool                              `json:"runtime_scope_verified"`
	BlockingOutboxCount            int                               `json:"blocking_outbox_count"`
	PendingEdgeToCloudOutbox        bool                              `json:"pending_edge_to_cloud_outbox"`
	OpenOperationalBoundaries       ArchiveOpenOperationalBoundaries  `json:"open_operational_boundaries"`
	ProtectedData                  ArchivePlanProtectedFlags         `json:"protected_data"`
	BlockReasons                   []string                          `json:"block_reasons"`
	HumanSummary                   string                            `json:"human_summary"`
	EligibleCounts                 ArchiveExportCounts               `json:"eligible_counts"`
	ArchiveCounts                  ArchiveExportCounts               `json:"archive_counts"`
	Verification                   ArchiveVerificationSummary        `json:"verification"`
}
