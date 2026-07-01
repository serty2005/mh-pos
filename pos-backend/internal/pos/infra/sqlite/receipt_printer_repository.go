package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"pos-backend/internal/pos/domain/receipt"
)

// ReplaceMasterReceiptPrinters атомарно заменяет Cloud-owned принтеры одного ресторана.
// Вызывается внутри master-data apply tx, поэтому удаление и вставки коммитятся вместе
// с sync state конкретного stream package.
func (r *Repository) ReplaceMasterReceiptPrinters(ctx context.Context, restaurantID string, printers []receipt.Printer, syncedAt string) error {
	restaurantID = strings.TrimSpace(restaurantID)
	exec := r.execer(ctx)
	if _, err := exec.ExecContext(ctx, `DELETE FROM receipt_printers WHERE restaurant_id = ?`, restaurantID); err != nil {
		return normalizeErr(err)
	}
	for i := range printers {
		v := printers[i]
		documentTypes, err := json.Marshal(v.DocumentTypes)
		if err != nil {
			return normalizeErr(err)
		}
		if _, err := exec.ExecContext(ctx, `INSERT INTO receipt_printers(
id,restaurant_id,name,type,address,port,document_types,codepage,paper_cut_type,cpl,is_active,cloud_version,synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			v.ID, v.RestaurantID, v.Name, v.Type, nullableStringValue(v.Address), nullableInt(v.Port), string(documentTypes), v.Codepage, v.PaperCutType, v.CPL, boolInt(v.IsActive), v.CloudVersion, syncedAt); err != nil {
			return normalizeErr(err)
		}
	}
	return nil
}

// ListReceiptPrinters возвращает активные принтеры ресторана для указанного document_type.
func (r *Repository) ListReceiptPrinters(ctx context.Context, restaurantID string, documentType receipt.DocumentType) ([]receipt.Printer, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,restaurant_id,name,type,address,port,document_types,codepage,paper_cut_type,cpl,is_active,cloud_version
FROM receipt_printers
WHERE restaurant_id = ? AND is_active = 1 AND EXISTS (
  SELECT 1 FROM json_each(receipt_printers.document_types) WHERE value = ?
)
ORDER BY name ASC, id ASC`, strings.TrimSpace(restaurantID), string(documentType))
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	var out []receipt.Printer
	for rows.Next() {
		v, err := scanReceiptPrinter(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, normalizeErr(rows.Err())
}

func (r *Repository) ListAllReceiptPrinters(ctx context.Context, restaurantID string) ([]receipt.Printer, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,restaurant_id,name,type,address,port,document_types,codepage,paper_cut_type,cpl,is_active,cloud_version
FROM receipt_printers
WHERE restaurant_id = ? AND is_active = 1
ORDER BY name ASC, id ASC`, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	var out []receipt.Printer
	for rows.Next() {
		v, err := scanReceiptPrinter(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, normalizeErr(rows.Err())
}

func (r *Repository) GetReceiptPrinter(ctx context.Context, id string) (*receipt.Printer, error) {
	return scanReceiptPrinter(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,restaurant_id,name,type,address,port,document_types,codepage,paper_cut_type,cpl,is_active,cloud_version
FROM receipt_printers
WHERE id = ?`, strings.TrimSpace(id)))
}

func scanReceiptPrinter(row scanner) (*receipt.Printer, error) {
	var v receipt.Printer
	var address, codepage, paperCutType sql.NullString
	var port sql.NullInt64
	var documentTypesRaw string
	var active int
	if err := row.Scan(&v.ID, &v.RestaurantID, &v.Name, &v.Type, &address, &port, &documentTypesRaw, &codepage, &paperCutType, &v.CPL, &active, &v.CloudVersion); err != nil {
		return nil, normalizeErr(err)
	}
	v.Address = strings.TrimSpace(address.String)
	if port.Valid {
		p := int(port.Int64)
		v.Port = &p
	}
	if err := json.Unmarshal([]byte(documentTypesRaw), &v.DocumentTypes); err != nil {
		return nil, normalizeErr(err)
	}
	v.Codepage = strings.TrimSpace(codepage.String)
	v.PaperCutType = strings.TrimSpace(paperCutType.String)
	v.IsActive = active != 0
	return &v, nil
}
