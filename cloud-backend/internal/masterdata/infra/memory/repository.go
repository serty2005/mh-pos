package memory

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync"

	"cloud-backend/internal/masterdata/app"
	"cloud-backend/internal/masterdata/domain"
)

// Repository хранит Cloud master-data state в памяти для app/api tests.
type Repository struct {
	mu           sync.Mutex
	roles        map[string]domain.Role
	employees    map[string]domain.Employee
	catalogItems map[string]domain.CatalogItem
	categories   map[string]domain.Category
	menuItems    map[string]domain.MenuItem
	publications map[string][]domain.Publication
	packages     map[string]app.StreamPackage
}

// NewRepository создает пустой in-memory repository.
func NewRepository() *Repository {
	return &Repository{
		roles:        map[string]domain.Role{},
		employees:    map[string]domain.Employee{},
		catalogItems: map[string]domain.CatalogItem{},
		categories:   map[string]domain.Category{},
		menuItems:    map[string]domain.MenuItem{},
		publications: map[string][]domain.Publication{},
		packages:     map[string]app.StreamPackage{},
	}
}

func (r *Repository) CreateRole(_ context.Context, v domain.Role) (domain.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.roles[v.ID] = v
	return v, nil
}

func (r *Repository) GetRole(_ context.Context, id string) (domain.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.roles[strings.TrimSpace(id)]
	if !ok {
		return domain.Role{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) ListRoles(_ context.Context, restaurantID string) ([]domain.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Role
	for _, item := range r.roles {
		if item.RestaurantID == strings.TrimSpace(restaurantID) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *Repository) CreateEmployee(_ context.Context, v domain.Employee) (domain.Employee, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.employees[v.ID] = v
	return v, nil
}

func (r *Repository) UpdateEmployee(_ context.Context, v domain.Employee) (domain.Employee, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.employees[v.ID]; !ok {
		return domain.Employee{}, domain.ErrNotFound
	}
	r.employees[v.ID] = v
	return v, nil
}

func (r *Repository) GetEmployee(_ context.Context, id string) (domain.Employee, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.employees[strings.TrimSpace(id)]
	if !ok {
		return domain.Employee{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) ListEmployees(_ context.Context, restaurantID string) ([]domain.Employee, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Employee
	for _, item := range r.employees {
		if item.RestaurantID == strings.TrimSpace(restaurantID) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *Repository) CreateCatalogItem(_ context.Context, v domain.CatalogItem) (domain.CatalogItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.catalogItems[v.ID] = v
	return v, nil
}

func (r *Repository) UpdateCatalogItem(_ context.Context, v domain.CatalogItem) (domain.CatalogItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.catalogItems[v.ID]; !ok {
		return domain.CatalogItem{}, domain.ErrNotFound
	}
	r.catalogItems[v.ID] = v
	return v, nil
}

func (r *Repository) GetCatalogItem(_ context.Context, id string) (domain.CatalogItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.catalogItems[strings.TrimSpace(id)]
	if !ok {
		return domain.CatalogItem{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) ListCatalogItems(_ context.Context, restaurantID string) ([]domain.CatalogItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.CatalogItem
	for _, item := range r.catalogItems {
		if item.RestaurantID == strings.TrimSpace(restaurantID) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *Repository) CreateCategory(_ context.Context, v domain.Category) (domain.Category, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.categories[v.ID] = v
	return v, nil
}

func (r *Repository) ListCategories(_ context.Context, restaurantID string) ([]domain.Category, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Category
	for _, item := range r.categories {
		if item.RestaurantID == strings.TrimSpace(restaurantID) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *Repository) CreateMenuItem(_ context.Context, v domain.MenuItem) (domain.MenuItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.menuItems[v.ID] = v
	return v, nil
}

func (r *Repository) UpdateMenuItem(_ context.Context, v domain.MenuItem) (domain.MenuItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.menuItems[v.ID]; !ok {
		return domain.MenuItem{}, domain.ErrNotFound
	}
	r.menuItems[v.ID] = v
	return v, nil
}

func (r *Repository) GetMenuItem(_ context.Context, id string) (domain.MenuItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.menuItems[strings.TrimSpace(id)]
	if !ok {
		return domain.MenuItem{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) ListMenuItems(_ context.Context, restaurantID string) ([]domain.MenuItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.MenuItem
	for _, item := range r.menuItems {
		if item.RestaurantID == strings.TrimSpace(restaurantID) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *Repository) NextPublicationVersion(_ context.Context, restaurantID string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return int64(len(r.publications[strings.TrimSpace(restaurantID)]) + 1), nil
}

func (r *Repository) SavePublication(_ context.Context, pub domain.Publication, packages []app.StreamPackage) (domain.Publication, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.publications[pub.RestaurantID] = append(r.publications[pub.RestaurantID], clonePublication(pub))
	for _, pkg := range packages {
		r.packages[pkg.StreamName+"|"+pkg.NodeDeviceID] = clonePackage(pkg)
	}
	return clonePublication(pub), nil
}

func (r *Repository) GetCurrentPublication(_ context.Context, restaurantID string) (domain.Publication, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := r.publications[strings.TrimSpace(restaurantID)]
	if len(items) == 0 {
		return domain.Publication{}, domain.ErrNotFound
	}
	return clonePublication(items[len(items)-1]), nil
}

func (r *Repository) Package(streamName, nodeDeviceID string) (app.StreamPackage, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.packages[strings.TrimSpace(streamName)+"|"+strings.TrimSpace(nodeDeviceID)]
	if !ok {
		v, ok = r.packages[strings.TrimSpace(streamName)+"|"]
	}
	return clonePackage(v), ok
}

func clonePublication(v domain.Publication) domain.Publication {
	out := v
	out.PackageJSON = slices.Clone(v.PackageJSON)
	return out
}

func clonePackage(v app.StreamPackage) app.StreamPackage {
	out := v
	out.PayloadJSON = slices.Clone(v.PayloadJSON)
	return out
}

// DecodePackage декодирует package payload в тестах без дублирования boilerplate.
func DecodePackage[T any](raw json.RawMessage) (T, error) {
	var out T
	err := json.Unmarshal(raw, &out)
	return out, err
}
