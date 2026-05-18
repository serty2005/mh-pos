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

// RetentionDryRunResult описывает read-only оценку cutoff без каких-либо удалений или архивных записей.
type RetentionDryRunResult struct {
	GeneratedAt                 time.Time               `json:"generated_at"`
	CutoffBusinessDateLocal     string                  `json:"cutoff_business_date_local"`
	Mode                        string                  `json:"mode"`
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
