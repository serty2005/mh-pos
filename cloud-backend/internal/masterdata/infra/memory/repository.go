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
	mu               sync.Mutex
	restaurants      map[string]domain.Restaurant
	roles            map[string]domain.Role
	employees        map[string]domain.Employee
	catalogItems     map[string]domain.CatalogItem
	folders          map[string]domain.CatalogFolder
	parameters       map[string]domain.FolderParameter
	tags             map[string]domain.CatalogTag
	itemTags         map[string]domain.CatalogItemTag
	modifierGroups   map[string]domain.ModifierGroup
	modifierOptions  map[string]domain.ModifierOption
	modifierBindings map[string]domain.ModifierGroupBinding
	pricingPolicies  map[string]domain.PricingPolicy
	categories       map[string]domain.Category
	halls            map[string]domain.Hall
	tables           map[string]domain.Table
	menuItems        map[string]domain.MenuItem
	publications     map[string][]domain.Publication
	packages         map[string]app.StreamPackage
}

// NewRepository создает пустой in-memory repository.
func NewRepository() *Repository {
	return &Repository{
		roles:            map[string]domain.Role{},
		employees:        map[string]domain.Employee{},
		catalogItems:     map[string]domain.CatalogItem{},
		folders:          map[string]domain.CatalogFolder{},
		parameters:       map[string]domain.FolderParameter{},
		tags:             map[string]domain.CatalogTag{},
		itemTags:         map[string]domain.CatalogItemTag{},
		modifierGroups:   map[string]domain.ModifierGroup{},
		modifierOptions:  map[string]domain.ModifierOption{},
		modifierBindings: map[string]domain.ModifierGroupBinding{},
		pricingPolicies:  map[string]domain.PricingPolicy{},
		categories:       map[string]domain.Category{},
		halls:            map[string]domain.Hall{},
		tables:           map[string]domain.Table{},
		menuItems:        map[string]domain.MenuItem{},
		publications:     map[string][]domain.Publication{},
		packages:         map[string]app.StreamPackage{},
		restaurants:      map[string]domain.Restaurant{},
	}
}

func (r *Repository) CreateRestaurant(_ context.Context, v domain.Restaurant) (domain.Restaurant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.restaurants[v.ID] = v
	return v, nil
}

func (r *Repository) UpdateRestaurant(_ context.Context, v domain.Restaurant) (domain.Restaurant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.restaurants[v.ID]; !ok {
		return domain.Restaurant{}, domain.ErrNotFound
	}
	r.restaurants[v.ID] = v
	return v, nil
}

func (r *Repository) GetRestaurant(_ context.Context, id string) (domain.Restaurant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.restaurants[strings.TrimSpace(id)]
	if !ok {
		return domain.Restaurant{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) ListRestaurants(_ context.Context) ([]domain.Restaurant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.Restaurant, 0, len(r.restaurants))
	for _, item := range r.restaurants {
		out = append(out, item)
	}
	return out, nil
}

func (r *Repository) CreateRole(_ context.Context, v domain.Role) (domain.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.roles[v.ID] = v
	return v, nil
}

func (r *Repository) UpdateRole(_ context.Context, v domain.Role) (domain.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.roles[v.ID]; !ok {
		return domain.Role{}, domain.ErrNotFound
	}
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
	for _, item := range r.catalogItems {
		if item.RestaurantID == v.RestaurantID && strings.EqualFold(item.SKU, v.SKU) && item.Status != domain.StatusArchived {
			return domain.CatalogItem{}, domain.ErrConflict
		}
	}
	r.catalogItems[v.ID] = v
	return v, nil
}

func (r *Repository) UpdateCatalogItem(_ context.Context, v domain.CatalogItem) (domain.CatalogItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.catalogItems[v.ID]; !ok {
		return domain.CatalogItem{}, domain.ErrNotFound
	}
	for _, item := range r.catalogItems {
		if item.ID != v.ID && item.RestaurantID == v.RestaurantID && strings.EqualFold(item.SKU, v.SKU) && item.Status != domain.StatusArchived && v.Status != domain.StatusArchived {
			return domain.CatalogItem{}, domain.ErrConflict
		}
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

func (r *Repository) CreateCatalogFolder(_ context.Context, v domain.CatalogFolder) (domain.CatalogFolder, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.folders[v.ID] = v
	return v, nil
}

func (r *Repository) UpdateCatalogFolder(_ context.Context, v domain.CatalogFolder) (domain.CatalogFolder, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.folders[v.ID]; !ok {
		return domain.CatalogFolder{}, domain.ErrNotFound
	}
	r.folders[v.ID] = v
	return v, nil
}

func (r *Repository) GetCatalogFolder(_ context.Context, id string) (domain.CatalogFolder, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.folders[strings.TrimSpace(id)]
	if !ok {
		return domain.CatalogFolder{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) ListCatalogFolders(_ context.Context, restaurantID string) ([]domain.CatalogFolder, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.CatalogFolder
	for _, item := range r.folders {
		if item.RestaurantID == strings.TrimSpace(restaurantID) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *Repository) CreateFolderParameter(_ context.Context, v domain.FolderParameter) (domain.FolderParameter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.parameters[v.ID] = v
	return v, nil
}

func (r *Repository) UpdateFolderParameter(_ context.Context, v domain.FolderParameter) (domain.FolderParameter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.parameters[v.ID]; !ok {
		return domain.FolderParameter{}, domain.ErrNotFound
	}
	r.parameters[v.ID] = v
	return v, nil
}

func (r *Repository) GetFolderParameter(_ context.Context, id string) (domain.FolderParameter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.parameters[strings.TrimSpace(id)]
	if !ok {
		return domain.FolderParameter{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) ListFolderParameters(_ context.Context, restaurantID string) ([]domain.FolderParameter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.FolderParameter
	for _, item := range r.parameters {
		if item.RestaurantID == strings.TrimSpace(restaurantID) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *Repository) CreateCatalogTag(_ context.Context, v domain.CatalogTag) (domain.CatalogTag, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tags[v.ID] = v
	return v, nil
}

func (r *Repository) UpdateCatalogTag(_ context.Context, v domain.CatalogTag) (domain.CatalogTag, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tags[v.ID]; !ok {
		return domain.CatalogTag{}, domain.ErrNotFound
	}
	r.tags[v.ID] = v
	return v, nil
}

func (r *Repository) GetCatalogTag(_ context.Context, id string) (domain.CatalogTag, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.tags[strings.TrimSpace(id)]
	if !ok {
		return domain.CatalogTag{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) ListCatalogTags(_ context.Context, restaurantID string) ([]domain.CatalogTag, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.CatalogTag
	for _, item := range r.tags {
		if item.RestaurantID == strings.TrimSpace(restaurantID) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *Repository) AssignCatalogItemTag(_ context.Context, v domain.CatalogItemTag) (domain.CatalogItemTag, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.itemTags[v.CatalogItemID+"|"+v.TagID] = v
	return v, nil
}

func (r *Repository) ListCatalogItemTags(_ context.Context, restaurantID string) ([]domain.CatalogItemTag, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.CatalogItemTag
	for _, item := range r.itemTags {
		if item.RestaurantID == strings.TrimSpace(restaurantID) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *Repository) CreateModifierGroup(_ context.Context, v domain.ModifierGroup) (domain.ModifierGroup, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modifierGroups[v.ID] = v
	return v, nil
}

func (r *Repository) UpdateModifierGroup(_ context.Context, v domain.ModifierGroup) (domain.ModifierGroup, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.modifierGroups[v.ID]; !ok {
		return domain.ModifierGroup{}, domain.ErrNotFound
	}
	r.modifierGroups[v.ID] = v
	return v, nil
}

func (r *Repository) GetModifierGroup(_ context.Context, id string) (domain.ModifierGroup, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.modifierGroups[strings.TrimSpace(id)]
	if !ok {
		return domain.ModifierGroup{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) ListModifierGroups(_ context.Context, restaurantID string) ([]domain.ModifierGroup, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.ModifierGroup
	for _, item := range r.modifierGroups {
		if item.RestaurantID == strings.TrimSpace(restaurantID) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *Repository) CreateModifierOption(_ context.Context, v domain.ModifierOption) (domain.ModifierOption, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modifierOptions[v.ID] = v
	return v, nil
}

func (r *Repository) UpdateModifierOption(_ context.Context, v domain.ModifierOption) (domain.ModifierOption, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.modifierOptions[v.ID]; !ok {
		return domain.ModifierOption{}, domain.ErrNotFound
	}
	r.modifierOptions[v.ID] = v
	return v, nil
}

func (r *Repository) GetModifierOption(_ context.Context, id string) (domain.ModifierOption, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.modifierOptions[strings.TrimSpace(id)]
	if !ok {
		return domain.ModifierOption{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) ListModifierOptions(_ context.Context, restaurantID string) ([]domain.ModifierOption, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.ModifierOption
	for _, item := range r.modifierOptions {
		if item.RestaurantID == strings.TrimSpace(restaurantID) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *Repository) CreateModifierGroupBinding(_ context.Context, v domain.ModifierGroupBinding) (domain.ModifierGroupBinding, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modifierBindings[v.ID] = v
	return v, nil
}

func (r *Repository) UpdateModifierGroupBinding(_ context.Context, v domain.ModifierGroupBinding) (domain.ModifierGroupBinding, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.modifierBindings[v.ID]; !ok {
		return domain.ModifierGroupBinding{}, domain.ErrNotFound
	}
	r.modifierBindings[v.ID] = v
	return v, nil
}

func (r *Repository) GetModifierGroupBinding(_ context.Context, id string) (domain.ModifierGroupBinding, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.modifierBindings[strings.TrimSpace(id)]
	if !ok {
		return domain.ModifierGroupBinding{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) ListModifierGroupBindings(_ context.Context, restaurantID string) ([]domain.ModifierGroupBinding, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.ModifierGroupBinding
	for _, item := range r.modifierBindings {
		if item.RestaurantID == strings.TrimSpace(restaurantID) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *Repository) CreatePricingPolicy(_ context.Context, v domain.PricingPolicy) (domain.PricingPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pricingPolicies[v.ID] = v
	return v, nil
}

func (r *Repository) UpdatePricingPolicy(_ context.Context, v domain.PricingPolicy) (domain.PricingPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.pricingPolicies[v.ID]; !ok {
		return domain.PricingPolicy{}, domain.ErrNotFound
	}
	r.pricingPolicies[v.ID] = v
	return v, nil
}

func (r *Repository) GetPricingPolicy(_ context.Context, id string) (domain.PricingPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.pricingPolicies[strings.TrimSpace(id)]
	if !ok {
		return domain.PricingPolicy{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) ListPricingPolicies(_ context.Context, restaurantID string) ([]domain.PricingPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.PricingPolicy
	for _, item := range r.pricingPolicies {
		if item.RestaurantID == strings.TrimSpace(restaurantID) {
			out = append(out, item)
		}
	}
	slices.SortFunc(out, func(a, b domain.PricingPolicy) int {
		if a.ApplicationIndex != b.ApplicationIndex {
			return a.ApplicationIndex - b.ApplicationIndex
		}
		return strings.Compare(a.ID, b.ID)
	})
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

func (r *Repository) CreateHall(_ context.Context, v domain.Hall) (domain.Hall, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range r.halls {
		if item.RestaurantID == v.RestaurantID && strings.EqualFold(item.Name, v.Name) && item.Status != domain.StatusArchived {
			return domain.Hall{}, domain.ErrConflict
		}
	}
	r.halls[v.ID] = v
	return v, nil
}

func (r *Repository) UpdateHall(_ context.Context, v domain.Hall) (domain.Hall, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.halls[v.ID]; !ok {
		return domain.Hall{}, domain.ErrNotFound
	}
	r.halls[v.ID] = v
	return v, nil
}

func (r *Repository) GetHall(_ context.Context, id string) (domain.Hall, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.halls[strings.TrimSpace(id)]
	if !ok {
		return domain.Hall{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) ListHalls(_ context.Context, restaurantID string) ([]domain.Hall, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Hall
	for _, item := range r.halls {
		if item.RestaurantID == strings.TrimSpace(restaurantID) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *Repository) CreateTable(_ context.Context, v domain.Table) (domain.Table, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range r.tables {
		if item.HallID == v.HallID && strings.EqualFold(item.Name, v.Name) && item.Status != domain.StatusArchived {
			return domain.Table{}, domain.ErrConflict
		}
	}
	r.tables[v.ID] = v
	return v, nil
}

func (r *Repository) UpdateTable(_ context.Context, v domain.Table) (domain.Table, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tables[v.ID]; !ok {
		return domain.Table{}, domain.ErrNotFound
	}
	r.tables[v.ID] = v
	return v, nil
}

func (r *Repository) GetTable(_ context.Context, id string) (domain.Table, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.tables[strings.TrimSpace(id)]
	if !ok {
		return domain.Table{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) ListTables(_ context.Context, restaurantID string) ([]domain.Table, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Table
	for _, item := range r.tables {
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

func (r *Repository) GetPublication(_ context.Context, restaurantID, packageID string) (domain.Publication, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range r.publications[strings.TrimSpace(restaurantID)] {
		if item.ID == strings.TrimSpace(packageID) {
			return clonePublication(item), nil
		}
	}
	return domain.Publication{}, domain.ErrNotFound
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
