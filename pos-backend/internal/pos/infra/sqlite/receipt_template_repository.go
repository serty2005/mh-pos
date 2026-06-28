package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"pos-backend/internal/pos/domain/receipt"
)

// ReplaceMasterReceiptTemplates атомарно заменяет Edge read model шаблонов печати.
// Вызывается внутри master-data apply tx: удаляет все строки и вставляет эффективный
// набор из Cloud package, чтобы отозванные/удалённые шаблоны не оставались на Edge.
func (r *Repository) ReplaceMasterReceiptTemplates(ctx context.Context, templates []receipt.Template, syncedAt string) error {
	exec := r.execer(ctx)
	if _, err := exec.ExecContext(ctx, `DELETE FROM receipt_templates`); err != nil {
		return normalizeErr(err)
	}
	for i := range templates {
		v := templates[i]
		var restaurantID any
		if scope := strings.TrimSpace(v.RestaurantID); scope != "" {
			restaurantID = scope
		}
		if _, err := exec.ExecContext(ctx, `INSERT INTO receipt_templates(id,restaurant_id,document_type,name,content,level,cpl,printer_class,is_default,version,synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			v.ID, restaurantID, string(v.DocumentType), v.Name, v.Content, v.Level, v.CPL, v.PrinterClass, boolInt(v.IsDefault), v.Version, syncedAt); err != nil {
			return normalizeErr(err)
		}
	}
	return nil
}

// ListReceiptTemplates возвращает Edge read model шаблонов печати в детерминированном порядке.
func (r *Repository) ListReceiptTemplates(ctx context.Context) ([]receipt.Template, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,restaurant_id,document_type,name,content,level,cpl,printer_class,is_default,version FROM receipt_templates ORDER BY id`)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	var out []receipt.Template
	for rows.Next() {
		var v receipt.Template
		var restaurantID sql.NullString
		var documentType string
		var isDefault int
		if err := rows.Scan(&v.ID, &restaurantID, &documentType, &v.Name, &v.Content, &v.Level, &v.CPL, &v.PrinterClass, &isDefault, &v.Version); err != nil {
			return nil, normalizeErr(err)
		}
		v.RestaurantID = strings.TrimSpace(restaurantID.String)
		v.DocumentType = receipt.DocumentType(documentType)
		v.IsDefault = isDefault != 0
		out = append(out, v)
	}
	return out, rows.Err()
}
