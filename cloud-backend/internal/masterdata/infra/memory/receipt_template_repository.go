package memory

import (
	"context"
	"sort"
	"strings"

	"cloud-backend/internal/masterdata/app"
	"cloud-backend/internal/masterdata/domain"
)

// defaultScopeKey воспроизводит partial unique по (org_id, restaurant-scope, document_type)
// среди is_default AND is_active строк, как в managed baseline (NULL restaurant_id -> tenant scope).
func defaultScopeKey(v domain.ReceiptTemplate) string {
	return strings.Join([]string{strings.TrimSpace(v.OrgID), strings.TrimSpace(v.RestaurantID), string(v.DocumentType)}, "\x1f")
}

func (r *Repository) ensureSingleDefault(v domain.ReceiptTemplate) error {
	if !v.IsDefault || !v.IsActive {
		return nil
	}
	key := defaultScopeKey(v)
	for id, existing := range r.receiptTemplates {
		if id == v.ID {
			continue
		}
		if existing.IsDefault && existing.IsActive && defaultScopeKey(existing) == key {
			return domain.ErrConflict
		}
	}
	return nil
}

// CreateReceiptTemplate сохраняет Cloud-owned шаблон печати с проверкой single-default.
func (r *Repository) CreateReceiptTemplate(_ context.Context, v domain.ReceiptTemplate) (domain.ReceiptTemplate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureSingleDefault(v); err != nil {
		return domain.ReceiptTemplate{}, err
	}
	r.receiptTemplates[v.ID] = v
	return v, nil
}

// UpdateReceiptTemplate обновляет шаблон по id.
func (r *Repository) UpdateReceiptTemplate(_ context.Context, v domain.ReceiptTemplate) (domain.ReceiptTemplate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.receiptTemplates[v.ID]; !ok {
		return domain.ReceiptTemplate{}, domain.ErrNotFound
	}
	if err := r.ensureSingleDefault(v); err != nil {
		return domain.ReceiptTemplate{}, err
	}
	r.receiptTemplates[v.ID] = v
	return v, nil
}

// GetReceiptTemplate возвращает один шаблон по id.
func (r *Repository) GetReceiptTemplate(_ context.Context, id string) (domain.ReceiptTemplate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.receiptTemplates[strings.TrimSpace(id)]
	if !ok {
		return domain.ReceiptTemplate{}, domain.ErrNotFound
	}
	return v, nil
}

// ListReceiptTemplates возвращает шаблоны с фильтрами в детерминированном порядке.
func (r *Repository) ListReceiptTemplates(_ context.Context, filter app.ReceiptTemplateFilter) ([]domain.ReceiptTemplate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.ReceiptTemplate, 0, len(r.receiptTemplates))
	for _, v := range r.receiptTemplates {
		if filter.OrgID != "" && v.OrgID != filter.OrgID {
			continue
		}
		if filter.RestaurantID != "" && v.RestaurantID != filter.RestaurantID {
			continue
		}
		if filter.DocumentType != "" && string(v.DocumentType) != filter.DocumentType {
			continue
		}
		if filter.IsDefault != nil && v.IsDefault != *filter.IsDefault {
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

// ListActiveReceiptTemplatesForRestaurant возвращает активные restaurant-specific и tenant-level шаблоны.
func (r *Repository) ListActiveReceiptTemplatesForRestaurant(_ context.Context, restaurantID string) ([]domain.ReceiptTemplate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	restaurantID = strings.TrimSpace(restaurantID)
	out := make([]domain.ReceiptTemplate, 0, len(r.receiptTemplates))
	for _, v := range r.receiptTemplates {
		if !v.IsActive {
			continue
		}
		scope := strings.TrimSpace(v.RestaurantID)
		if scope == "" || scope == restaurantID {
			out = append(out, v)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
