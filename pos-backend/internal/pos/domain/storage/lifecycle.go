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
	StockDocuments          int `json:"stock_documents"`
	StockMoves              int `json:"stock_moves"`
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
	FormatVersion             string                     `json:"format_version"`
	RestaurantID              string                     `json:"restaurant_id,omitempty"`
	BusinessDateRange         BusinessDateRange          `json:"business_date_range"`
	CutoffBusinessDateLocal   string                     `json:"cutoff_business_date_local"`
	Tables                    []ArchivePlanTableManifest `json:"tables"`
}

// ArchiveExportPlan описывает безопасный export-plan без записи archive files и без мутации runtime rows.
type ArchiveExportPlan struct {
	GeneratedAt                 time.Time               `json:"generated_at"`
	CutoffBusinessDateLocal     string                  `json:"cutoff_business_date_local"`
	Mode                        string                  `json:"mode"`
	ResultMode                  string                  `json:"result_mode"`
	DestructiveApplySupported   bool                    `json:"destructive_apply_supported"`
	Blocked                     bool                    `json:"blocked"`
	BlockReasons                []string                `json:"block_reasons"`
	ArchiveSet                  RetentionEligibleCounts   `json:"archive_set"`
	Protected                   ArchivePlanProtectedFlags `json:"protected"`
	ActiveOrders                int                       `json:"active_orders"`
	OpenShifts                  int                       `json:"open_shifts"`
	OpenCashSessions            int                       `json:"open_cash_sessions"`
	BlockingOutboxMessages      int                       `json:"blocking_outbox_messages"`
	Manifest                    ArchivePlanManifest       `json:"manifest"`
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
