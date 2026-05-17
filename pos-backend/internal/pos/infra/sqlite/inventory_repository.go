package sqlite

import (
	"context"
	"database/sql"
	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateRecipeVersion(ctx context.Context, v *domain.RecipeVersion) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO recipe_versions(id,dish_catalog_item_id,version,name,status,yield_quantity,yield_unit,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.DishCatalogItemID, v.Version, v.Name, string(v.Status), v.YieldQuantity, v.YieldUnit, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListRecipeVersions(ctx context.Context) ([]domain.RecipeVersion, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,dish_catalog_item_id,version,name,status,yield_quantity,yield_unit,active,created_at,updated_at FROM recipe_versions ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RecipeVersion
	for rows.Next() {
		var v domain.RecipeVersion
		var status, created, updated string
		var active int
		if err := rows.Scan(&v.ID, &v.DishCatalogItemID, &v.Version, &v.Name, &status, &v.YieldQuantity, &v.YieldUnit, &active, &created, &updated); err != nil {
			return nil, err
		}
		v.Status = domain.RecipeVersionStatus(status)
		v.Active = active == 1
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateRecipeLine(ctx context.Context, v *domain.RecipeLine) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO recipe_lines(id,recipe_version_id,catalog_item_id,quantity,unit,loss_percent,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		v.ID, v.RecipeVersionID, v.CatalogItemID, v.Quantity, v.Unit, v.LossPercent, dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListRecipeLines(ctx context.Context, recipeVersionID string) ([]domain.RecipeLine, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,recipe_version_id,catalog_item_id,quantity,unit,loss_percent,created_at,updated_at FROM recipe_lines WHERE recipe_version_id = ? ORDER BY created_at`, recipeVersionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RecipeLine
	for rows.Next() {
		var v domain.RecipeLine
		var created, updated string
		if err := rows.Scan(&v.ID, &v.RecipeVersionID, &v.CatalogItemID, &v.Quantity, &v.Unit, &v.LossPercent, &created, &updated); err != nil {
			return nil, err
		}
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreatePurchaseReceipt(ctx context.Context, v *domain.PurchaseReceipt) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO purchase_receipts(id,restaurant_id,device_id,supplier_name,document_number,status,received_at,total_amount,currency,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, v.DeviceID, v.SupplierName, v.DocumentNumber, string(v.Status), dbTime(v.ReceivedAt), v.TotalAmount, v.Currency, dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListPurchaseReceipts(ctx context.Context) ([]domain.PurchaseReceipt, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,restaurant_id,device_id,supplier_name,document_number,status,received_at,total_amount,currency,created_at,updated_at FROM purchase_receipts ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PurchaseReceipt
	for rows.Next() {
		var v domain.PurchaseReceipt
		var status, received, created, updated string
		if err := rows.Scan(&v.ID, &v.RestaurantID, &v.DeviceID, &v.SupplierName, &v.DocumentNumber, &status, &received, &v.TotalAmount, &v.Currency, &created, &updated); err != nil {
			return nil, err
		}
		v.Status = domain.PurchaseReceiptStatus(status)
		v.ReceivedAt = parseTime(received)
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreatePurchaseReceiptLine(ctx context.Context, v *domain.PurchaseReceiptLine) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO purchase_receipt_lines(id,purchase_receipt_id,catalog_item_id,quantity,unit,unit_cost,total_cost,currency,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.PurchaseReceiptID, v.CatalogItemID, v.Quantity, v.Unit, v.UnitCost, v.TotalCost, v.Currency, dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListPurchaseReceiptLines(ctx context.Context, purchaseReceiptID string) ([]domain.PurchaseReceiptLine, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,purchase_receipt_id,catalog_item_id,quantity,unit,unit_cost,total_cost,currency,created_at,updated_at FROM purchase_receipt_lines WHERE purchase_receipt_id = ? ORDER BY created_at`, purchaseReceiptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PurchaseReceiptLine
	for rows.Next() {
		var v domain.PurchaseReceiptLine
		var created, updated string
		if err := rows.Scan(&v.ID, &v.PurchaseReceiptID, &v.CatalogItemID, &v.Quantity, &v.Unit, &v.UnitCost, &v.TotalCost, &v.Currency, &created, &updated); err != nil {
			return nil, err
		}
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateStockDocument(ctx context.Context, v *domain.StockDocument) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO stock_documents(id,restaurant_id,device_id,document_type,source_type,source_id,check_id,precheck_id,financial_operation_id,business_date_local,shift_id,cash_session_id,created_by_employee_id,status,occurred_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, v.DeviceID, string(v.Type), nullableString(v.SourceType), nullableString(v.SourceID), nullableString(v.CheckID), nullableString(v.PrecheckID), nullableString(v.FinancialOperationID), v.BusinessDateLocal, nullableString(v.ShiftID), nullableString(v.CashSessionID), nullableString(v.CreatedByEmployeeID), string(v.Status), dbTime(v.OccurredAt), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListStockDocuments(ctx context.Context) ([]domain.StockDocument, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,restaurant_id,device_id,document_type,source_type,source_id,check_id,precheck_id,financial_operation_id,business_date_local,shift_id,cash_session_id,created_by_employee_id,status,occurred_at,created_at,updated_at FROM stock_documents ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.StockDocument
	for rows.Next() {
		var v domain.StockDocument
		var typ, status, occurred, created, updated string
		var sourceType, sourceID, checkID, precheckID, operationID, businessDate, shiftID, cashSessionID, createdBy sql.NullString
		if err := rows.Scan(&v.ID, &v.RestaurantID, &v.DeviceID, &typ, &sourceType, &sourceID, &checkID, &precheckID, &operationID, &businessDate, &shiftID, &cashSessionID, &createdBy, &status, &occurred, &created, &updated); err != nil {
			return nil, err
		}
		v.Type = domain.StockDocumentType(typ)
		v.SourceType = stringPtr(sourceType)
		v.SourceID = stringPtr(sourceID)
		v.CheckID = stringPtr(checkID)
		v.PrecheckID = stringPtr(precheckID)
		v.FinancialOperationID = stringPtr(operationID)
		if businessDate.Valid {
			v.BusinessDateLocal = businessDate.String
		}
		v.ShiftID = stringPtr(shiftID)
		v.CashSessionID = stringPtr(cashSessionID)
		v.CreatedByEmployeeID = stringPtr(createdBy)
		v.Status = domain.StockDocumentStatus(status)
		v.OccurredAt = parseTime(occurred)
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateStockMove(ctx context.Context, v *domain.StockMove) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO stock_moves(id,stock_document_id,catalog_item_id,order_line_id,location_id,movement_type,quantity,unit,unit_cost,total_cost,occurred_at,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.StockDocumentID, v.CatalogItemID, nullableString(v.OrderLineID), nullableString(v.LocationID), string(v.Type), v.Quantity, v.Unit, nullableInt64(v.UnitCost), nullableInt64(v.TotalCost), dbTime(v.OccurredAt), dbTime(v.CreatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListStockMoves(ctx context.Context, stockDocumentID string) ([]domain.StockMove, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,stock_document_id,catalog_item_id,order_line_id,location_id,movement_type,quantity,unit,unit_cost,total_cost,occurred_at,created_at FROM stock_moves WHERE stock_document_id = ? ORDER BY created_at`, stockDocumentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.StockMove
	for rows.Next() {
		var v domain.StockMove
		var orderLineID, location sql.NullString
		var unitCost, totalCost sql.NullInt64
		var typ, occurred, created string
		if err := rows.Scan(&v.ID, &v.StockDocumentID, &v.CatalogItemID, &orderLineID, &location, &typ, &v.Quantity, &v.Unit, &unitCost, &totalCost, &occurred, &created); err != nil {
			return nil, err
		}
		v.OrderLineID = stringPtr(orderLineID)
		v.LocationID = stringPtr(location)
		v.Type = domain.StockMoveType(typ)
		v.UnitCost = int64Ptr(unitCost)
		v.TotalCost = int64Ptr(totalCost)
		v.OccurredAt = parseTime(occurred)
		v.CreatedAt = parseTime(created)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertStockBalance(ctx context.Context, v *domain.StockBalance) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO stock_balances(id,catalog_item_id,location_id,quantity,unit,updated_at) VALUES (?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET catalog_item_id = excluded.catalog_item_id, location_id = excluded.location_id, quantity = excluded.quantity, unit = excluded.unit, updated_at = excluded.updated_at`,
		v.ID, v.CatalogItemID, nullableString(v.LocationID), v.Quantity, v.Unit, dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListStockBalances(ctx context.Context) ([]domain.StockBalance, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,catalog_item_id,location_id,quantity,unit,updated_at FROM stock_balances ORDER BY updated_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.StockBalance
	for rows.Next() {
		var v domain.StockBalance
		var location sql.NullString
		var updated string
		if err := rows.Scan(&v.ID, &v.CatalogItemID, &location, &v.Quantity, &v.Unit, &updated); err != nil {
			return nil, err
		}
		v.LocationID = stringPtr(location)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateItemCost(ctx context.Context, v *domain.ItemCost) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO item_costs(id,catalog_item_id,cost_type,amount,currency,source_type,source_id,effective_at,created_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		v.ID, v.CatalogItemID, string(v.Type), v.Amount, v.Currency, nullableString(v.SourceType), nullableString(v.SourceID), dbTime(v.EffectiveAt), dbTime(v.CreatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetLastItemCost(ctx context.Context, catalogItemID string) (*domain.ItemCost, error) {
	return r.scanItemCost(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,catalog_item_id,cost_type,amount,currency,source_type,source_id,effective_at,created_at FROM item_costs WHERE catalog_item_id = ? AND cost_type = 'last_purchase' ORDER BY effective_at DESC, created_at DESC LIMIT 1`, catalogItemID))
}

func (r *Repository) ListItemCosts(ctx context.Context, catalogItemID string) ([]domain.ItemCost, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,catalog_item_id,cost_type,amount,currency,source_type,source_id,effective_at,created_at FROM item_costs WHERE catalog_item_id = ? ORDER BY effective_at`, catalogItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ItemCost
	for rows.Next() {
		v, err := scanItemCostRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *Repository) scanItemCost(row *sql.Row) (*domain.ItemCost, error) {
	v, err := scanItemCostRows(row)
	if err != nil {
		return nil, normalizeErr(err)
	}
	return v, nil
}

func scanItemCostRows(row outboxScanner) (*domain.ItemCost, error) {
	var v domain.ItemCost
	var typ, effective, created string
	var sourceType, sourceID sql.NullString
	if err := row.Scan(&v.ID, &v.CatalogItemID, &typ, &v.Amount, &v.Currency, &sourceType, &sourceID, &effective, &created); err != nil {
		return nil, err
	}
	v.Type = domain.ItemCostType(typ)
	v.SourceType = stringPtr(sourceType)
	v.SourceID = stringPtr(sourceID)
	v.EffectiveAt = parseTime(effective)
	v.CreatedAt = parseTime(created)
	return &v, nil
}
