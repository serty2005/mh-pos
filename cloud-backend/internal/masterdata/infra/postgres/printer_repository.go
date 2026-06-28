package postgres

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"cloud-backend/internal/masterdata/app"
	"cloud-backend/internal/masterdata/domain"
)

const printerColumns = `id,org_id,restaurant_id,name,type,address,port,document_types,codepage,paper_cut_type,cpl,is_active,version,created_at,updated_at`

func scanPrinter(row scanner) (domain.Printer, error) {
	var v domain.Printer
	var port *int
	var rawDocTypes string
	err := row.Scan(
		&v.ID, &v.OrgID, &v.RestaurantID, &v.Name, &v.Type, &v.Address, &port, &rawDocTypes,
		&v.Codepage, &v.PaperCutType, &v.CPL, &v.IsActive, &v.Version, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return domain.Printer{}, err
	}
	v.Port = port
	var docTypeStrings []string
	if err := json.Unmarshal([]byte(rawDocTypes), &docTypeStrings); err != nil {
		return domain.Printer{}, err
	}
	v.DocumentTypes = make([]domain.PrinterDocumentType, 0, len(docTypeStrings))
	for _, s := range docTypeStrings {
		v.DocumentTypes = append(v.DocumentTypes, domain.PrinterDocumentType(s))
	}
	return v, nil
}

func marshalDocumentTypes(types []domain.PrinterDocumentType) string {
	out := make([]string, 0, len(types))
	for _, dt := range types {
		out = append(out, string(dt))
	}
	b, _ := json.Marshal(out)
	return string(b)
}

// CreatePrinter сохраняет Cloud-owned принтер.
func (r *Repository) CreatePrinter(ctx context.Context, v domain.Printer) (domain.Printer, error) {
	out, err := scanPrinter(r.pool.QueryRow(ctx, `
INSERT INTO cloud_printers(`+printerColumns+`)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
RETURNING `+printerColumns,
		v.ID, v.OrgID, v.RestaurantID, v.Name, string(v.Type), v.Address, v.Port,
		marshalDocumentTypes(v.DocumentTypes), string(v.Codepage), string(v.PaperCutType),
		v.CPL, v.IsActive, v.Version, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

// UpdatePrinter обновляет принтер по id.
func (r *Repository) UpdatePrinter(ctx context.Context, v domain.Printer) (domain.Printer, error) {
	out, err := scanPrinter(r.pool.QueryRow(ctx, `
UPDATE cloud_printers
SET org_id=$2,restaurant_id=$3,name=$4,type=$5,address=$6,port=$7,document_types=$8,
    codepage=$9,paper_cut_type=$10,cpl=$11,is_active=$12,version=$13,updated_at=$14
WHERE id=$1
RETURNING `+printerColumns,
		v.ID, v.OrgID, v.RestaurantID, v.Name, string(v.Type), v.Address, v.Port,
		marshalDocumentTypes(v.DocumentTypes), string(v.Codepage), string(v.PaperCutType),
		v.CPL, v.IsActive, v.Version, v.UpdatedAt))
	return out, normalizeErr(err)
}

// GetPrinter возвращает один принтер по id.
func (r *Repository) GetPrinter(ctx context.Context, id string) (domain.Printer, error) {
	v, err := scanPrinter(r.pool.QueryRow(ctx, `SELECT `+printerColumns+` FROM cloud_printers WHERE id = $1`, strings.TrimSpace(id)))
	return v, normalizeErr(err)
}

// ListPrinters возвращает принтеры с фильтрами.
func (r *Repository) ListPrinters(ctx context.Context, filter app.PrinterFilter) ([]domain.Printer, error) {
	query := `SELECT ` + printerColumns + ` FROM cloud_printers WHERE 1=1`
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
	if filter.IsActive != nil {
		add(" AND is_active = ", *filter.IsActive)
	}
	query += ` ORDER BY id`
	return r.queryPrinters(ctx, query, args...)
}

// ListActivePrintersForRestaurant возвращает все активные принтеры ресторана для сборки Cloud -> Edge package.
func (r *Repository) ListActivePrintersForRestaurant(ctx context.Context, restaurantID string) ([]domain.Printer, error) {
	return r.queryPrinters(ctx, `
SELECT `+printerColumns+`
FROM cloud_printers
WHERE is_active = TRUE AND restaurant_id = $1
ORDER BY id`, strings.TrimSpace(restaurantID))
}

func (r *Repository) queryPrinters(ctx context.Context, query string, args ...any) ([]domain.Printer, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Printer
	for rows.Next() {
		v, err := scanPrinter(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
