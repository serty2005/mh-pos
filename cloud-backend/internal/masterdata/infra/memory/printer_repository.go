package memory

import (
	"context"
	"sort"
	"strings"

	"cloud-backend/internal/masterdata/app"
	"cloud-backend/internal/masterdata/domain"
)

// CreatePrinter сохраняет Cloud-owned принтер.
func (r *Repository) CreatePrinter(_ context.Context, v domain.Printer) (domain.Printer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.printers[v.ID] = v
	return v, nil
}

// UpdatePrinter обновляет принтер по id.
func (r *Repository) UpdatePrinter(_ context.Context, v domain.Printer) (domain.Printer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.printers[v.ID]; !ok {
		return domain.Printer{}, domain.ErrNotFound
	}
	r.printers[v.ID] = v
	return v, nil
}

// GetPrinter возвращает один принтер по id.
func (r *Repository) GetPrinter(_ context.Context, id string) (domain.Printer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.printers[strings.TrimSpace(id)]
	if !ok {
		return domain.Printer{}, domain.ErrNotFound
	}
	return v, nil
}

// ListPrinters возвращает принтеры с фильтрами в детерминированном порядке.
func (r *Repository) ListPrinters(_ context.Context, filter app.PrinterFilter) ([]domain.Printer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.Printer, 0, len(r.printers))
	for _, v := range r.printers {
		if filter.OrgID != "" && v.OrgID != filter.OrgID {
			continue
		}
		if filter.RestaurantID != "" && v.RestaurantID != filter.RestaurantID {
			continue
		}
		if filter.IsActive != nil && v.IsActive != *filter.IsActive {
			continue
		}
		out = append(out, v)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// ListActivePrintersForRestaurant возвращает активные принтеры ресторана для сборки Cloud -> Edge package.
func (r *Repository) ListActivePrintersForRestaurant(_ context.Context, restaurantID string) ([]domain.Printer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	restaurantID = strings.TrimSpace(restaurantID)
	out := make([]domain.Printer, 0, len(r.printers))
	for _, v := range r.printers {
		if v.IsActive && v.RestaurantID == restaurantID {
			out = append(out, v)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
