package postgres

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"cloud-backend/internal/masterdata/app"
	"cloud-backend/internal/masterdata/domain"
)

const receiptTemplateColumns = `id,org_id,restaurant_id,document_type,name,description,content,level,cpl,printer_class,is_default,version,is_active,created_at,updated_at`

// nullableRestaurantID конвертирует пустой restaurant_id в SQL NULL (tenant-level default).
func nullableRestaurantID(restaurantID string) any {
	restaurantID = strings.TrimSpace(restaurantID)
	if restaurantID == "" {
		return nil
	}
	return restaurantID
}

func scanReceiptTemplate(row scanner) (domain.ReceiptTemplate, error) {
	var v domain.ReceiptTemplate
	var restaurantID sql.NullString
	var documentType string
	err := row.Scan(
		&v.ID, &v.OrgID, &restaurantID, &documentType, &v.Name, &v.Description, &v.Content,
		&v.Level, &v.CPL, &v.PrinterClass, &v.IsDefault, &v.Version, &v.IsActive, &v.CreatedAt, &v.UpdatedAt,
	)
	v.RestaurantID = strings.TrimSpace(restaurantID.String)
	v.DocumentType = domain.ReceiptTemplateDocumentType(documentType)
	return v, err
}

// CreateReceiptTemplate сохраняет Cloud-owned шаблон печати.
func (r *Repository) CreateReceiptTemplate(ctx context.Context, v domain.ReceiptTemplate) (domain.ReceiptTemplate, error) {
	out, err := scanReceiptTemplate(r.pool.QueryRow(ctx, `
INSERT INTO cloud_receipt_templates(`+receiptTemplateColumns+`)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
RETURNING `+receiptTemplateColumns,
		v.ID, v.OrgID, nullableRestaurantID(v.RestaurantID), string(v.DocumentType), v.Name, v.Description, v.Content,
		v.Level, v.CPL, v.PrinterClass, v.IsDefault, v.Version, v.IsActive, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

// UpdateReceiptTemplate обновляет шаблон по id с инкрементированным version.
func (r *Repository) UpdateReceiptTemplate(ctx context.Context, v domain.ReceiptTemplate) (domain.ReceiptTemplate, error) {
	out, err := scanReceiptTemplate(r.pool.QueryRow(ctx, `
UPDATE cloud_receipt_templates
SET org_id=$2,restaurant_id=$3,document_type=$4,name=$5,description=$6,content=$7,level=$8,cpl=$9,printer_class=$10,is_default=$11,version=$12,is_active=$13,updated_at=$14
WHERE id=$1
RETURNING `+receiptTemplateColumns,
		v.ID, v.OrgID, nullableRestaurantID(v.RestaurantID), string(v.DocumentType), v.Name, v.Description, v.Content,
		v.Level, v.CPL, v.PrinterClass, v.IsDefault, v.Version, v.IsActive, v.UpdatedAt))
	return out, normalizeErr(err)
}

// GetReceiptTemplate возвращает один шаблон по id.
func (r *Repository) GetReceiptTemplate(ctx context.Context, id string) (domain.ReceiptTemplate, error) {
	v, err := scanReceiptTemplate(r.pool.QueryRow(ctx, `SELECT `+receiptTemplateColumns+` FROM cloud_receipt_templates WHERE id = $1`, strings.TrimSpace(id)))
	return v, normalizeErr(err)
}

// ListReceiptTemplates возвращает шаблоны с фильтрами document_type/is_default/is_active.
func (r *Repository) ListReceiptTemplates(ctx context.Context, filter app.ReceiptTemplateFilter) ([]domain.ReceiptTemplate, error) {
	query := `SELECT ` + receiptTemplateColumns + ` FROM cloud_receipt_templates WHERE 1=1`
	args := []any{}
	add := func(clause string, value any) {
		args = append(args, value)
		query += clause + "$" + strconv.Itoa(len(args))
	}
	if v := strings.TrimSpace(filter.OrgID); v != "" {
		add(" AND org_id = ", v)
	}
	if v := strings.TrimSpace(filter.RestaurantID); v != "" {
		add(" AND restaurant_id = ", v)
	}
	if v := strings.TrimSpace(filter.DocumentType); v != "" {
		add(" AND document_type = ", v)
	}
	if filter.IsDefault != nil {
		add(" AND is_default = ", *filter.IsDefault)
	}
	if filter.IsActive != nil {
		add(" AND is_active = ", *filter.IsActive)
	}
	query += ` ORDER BY id`
	return r.queryReceiptTemplates(ctx, query, args...)
}

// ListActiveReceiptTemplatesForRestaurant возвращает активные restaurant-specific и tenant-level шаблоны
// для сборки Cloud -> Edge package receipt_templates.
func (r *Repository) ListActiveReceiptTemplatesForRestaurant(ctx context.Context, restaurantID string) ([]domain.ReceiptTemplate, error) {
	return r.queryReceiptTemplates(ctx, `
SELECT `+receiptTemplateColumns+`
FROM cloud_receipt_templates
WHERE is_active = TRUE AND (restaurant_id = $1 OR restaurant_id IS NULL)
ORDER BY id`, strings.TrimSpace(restaurantID))
}

func (r *Repository) queryReceiptTemplates(ctx context.Context, query string, args ...any) ([]domain.ReceiptTemplate, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ReceiptTemplate
	for rows.Next() {
		v, err := scanReceiptTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
