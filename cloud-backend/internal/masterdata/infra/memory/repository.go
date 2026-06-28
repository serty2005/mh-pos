package memory

import (
	"context"
	"encoding/json"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud-backend/internal/masterdata/app"
	"cloud-backend/internal/masterdata/domain"
)

// Repository хранит Cloud master-data state в памяти для app/api tests.
type Repository struct {
	mu                      sync.Mutex
	restaurants             map[string]domain.Restaurant
	roles                   map[string]domain.Role
	employees               map[string]domain.Employee
	catalogItems            map[string]domain.CatalogItem
	folders                 map[string]domain.CatalogFolder
	parameters              map[string]domain.FolderParameter
	tags                    map[string]domain.CatalogTag
	itemTags                map[string]domain.CatalogItemTag
	modifierGroups          map[string]domain.ModifierGroup
	modifierOptions         map[string]domain.ModifierOption
	modifierBindings        map[string]domain.ModifierGroupBinding
	pricingPolicies         map[string]domain.PricingPolicy
	recipeItems             map[string]domain.RecipeItem
	recipeVersions          map[string]domain.RecipeVersion
	recipeLines             map[string][]domain.RecipeLine
	stopLists               map[string]domain.StopListEntry
	categories              map[string]domain.Category
	halls                   map[string]domain.Hall
	tables                  map[string]domain.Table
	menuItems               map[string]domain.MenuItem
	publications            map[string][]domain.Publication
	packages                map[string]app.StreamPackage
	assignedNodes           map[string]map[string]struct{}
	deliveryStates          map[string]app.DeliveryStatus
	catalogSuggestions      map[string]domain.CatalogSuggestion
	recipeSuggestions       map[string]domain.RecipeSuggestion
	recipeSuggestionChanges map[string][]domain.RecipeSuggestionChange
	stopListUpdates         map[string]domain.StopListUpdateReview
	assignmentAuditEvents   map[string]domain.ReviewAssignmentAuditEvent
	receiptTemplates        map[string]domain.ReceiptTemplate
	printers                map[string]domain.Printer
	deliveryAssemblyErr     error
}

// NewRepository создает пустой in-memory repository.
func NewRepository() *Repository {
	return &Repository{
		roles:                   map[string]domain.Role{},
		employees:               map[string]domain.Employee{},
		catalogItems:            map[string]domain.CatalogItem{},
		folders:                 map[string]domain.CatalogFolder{},
		parameters:              map[string]domain.FolderParameter{},
		tags:                    map[string]domain.CatalogTag{},
		itemTags:                map[string]domain.CatalogItemTag{},
		modifierGroups:          map[string]domain.ModifierGroup{},
		modifierOptions:         map[string]domain.ModifierOption{},
		modifierBindings:        map[string]domain.ModifierGroupBinding{},
		pricingPolicies:         map[string]domain.PricingPolicy{},
		recipeItems:             map[string]domain.RecipeItem{},
		recipeVersions:          map[string]domain.RecipeVersion{},
		recipeLines:             map[string][]domain.RecipeLine{},
		stopLists:               map[string]domain.StopListEntry{},
		categories:              map[string]domain.Category{},
		halls:                   map[string]domain.Hall{},
		tables:                  map[string]domain.Table{},
		menuItems:               map[string]domain.MenuItem{},
		publications:            map[string][]domain.Publication{},
		packages:                map[string]app.StreamPackage{},
		assignedNodes:           map[string]map[string]struct{}{},
		deliveryStates:          map[string]app.DeliveryStatus{},
		restaurants:             map[string]domain.Restaurant{},
		catalogSuggestions:      map[string]domain.CatalogSuggestion{},
		recipeSuggestions:       map[string]domain.RecipeSuggestion{},
		recipeSuggestionChanges: map[string][]domain.RecipeSuggestionChange{},
		stopListUpdates:         map[string]domain.StopListUpdateReview{},
		assignmentAuditEvents:   map[string]domain.ReviewAssignmentAuditEvent{},
		receiptTemplates:        map[string]domain.ReceiptTemplate{},
		printers:                map[string]domain.Printer{},
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

func (r *Repository) ListRoles(_ context.Context) ([]domain.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.deliveryAssemblyErr != nil {
		return nil, r.deliveryAssemblyErr
	}
	var out []domain.Role
	for _, item := range r.roles {
		out = append(out, item)
	}
	return out, nil
}

// FailDeliveryAssemblyForTest управляет ошибкой чтения только для проверки retry workflow.
func (r *Repository) FailDeliveryAssemblyForTest(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deliveryAssemblyErr = err
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

func (r *Repository) ListEmployees(_ context.Context) ([]domain.Employee, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Employee
	for _, item := range r.employees {
		out = append(out, item)
	}
	return out, nil
}

func (r *Repository) CreateCatalogItem(_ context.Context, v domain.CatalogItem) (domain.CatalogItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range r.catalogItems {
		if strings.EqualFold(item.SKU, v.SKU) && item.Status != domain.StatusArchived {
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
		if item.ID != v.ID && strings.EqualFold(item.SKU, v.SKU) && item.Status != domain.StatusArchived && v.Status != domain.StatusArchived {
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
	restaurantID = strings.TrimSpace(restaurantID)
	for _, item := range r.catalogItems {
		if restaurantID == "" || item.RestaurantID == "" || item.RestaurantID == restaurantID {
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
	restaurantID = strings.TrimSpace(restaurantID)
	for _, item := range r.folders {
		if restaurantID == "" || item.RestaurantID == "" || item.RestaurantID == restaurantID {
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
	restaurantID = strings.TrimSpace(restaurantID)
	for _, item := range r.tags {
		if restaurantID == "" || item.RestaurantID == "" || item.RestaurantID == restaurantID {
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
	restaurantID = strings.TrimSpace(restaurantID)
	for _, item := range r.itemTags {
		if restaurantID == "" || item.RestaurantID == "" || item.RestaurantID == restaurantID {
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

func (r *Repository) CreateRecipeItem(_ context.Context, v domain.RecipeItem) (domain.RecipeItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range r.recipeItems {
		if item.RecipeOwnerCatalogItemID == v.RecipeOwnerCatalogItemID && item.ComponentCatalogItemID == v.ComponentCatalogItemID {
			return domain.RecipeItem{}, domain.ErrConflict
		}
	}
	r.recipeItems[v.ID] = v
	return v, nil
}

func (r *Repository) UpdateRecipeItem(_ context.Context, v domain.RecipeItem) (domain.RecipeItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.recipeItems[v.ID]; !ok {
		return domain.RecipeItem{}, domain.ErrNotFound
	}
	r.recipeItems[v.ID] = v
	return v, nil
}

func (r *Repository) GetRecipeItem(_ context.Context, id string) (domain.RecipeItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.recipeItems[strings.TrimSpace(id)]
	if !ok {
		return domain.RecipeItem{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) ListRecipeItems(_ context.Context, restaurantID string) ([]domain.RecipeItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.RecipeItem
	for _, item := range r.recipeItems {
		if item.RestaurantID == strings.TrimSpace(restaurantID) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *Repository) CreateRecipeVersion(_ context.Context, v domain.RecipeVersion, lines []domain.RecipeLine) (domain.RecipeVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.recipeVersions[v.ID]; ok {
		return domain.RecipeVersion{}, domain.ErrConflict
	}
	r.recipeVersions[v.ID] = v
	r.recipeLines[v.ID] = slices.Clone(lines)
	return v, nil
}

func (r *Repository) UpdateRecipeVersion(_ context.Context, v domain.RecipeVersion) (domain.RecipeVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.recipeVersions[v.ID]; !ok {
		return domain.RecipeVersion{}, domain.ErrNotFound
	}
	r.recipeVersions[v.ID] = v
	return v, nil
}

func (r *Repository) GetRecipeVersion(_ context.Context, id string) (domain.RecipeVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.recipeVersions[strings.TrimSpace(id)]
	if !ok {
		return domain.RecipeVersion{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) ListRecipeVersions(_ context.Context, restaurantID, ownerCatalogItemID, status string, limit, offset int) ([]domain.RecipeVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.RecipeVersion
	for _, item := range r.recipeVersions {
		if restaurantID != "" && item.RestaurantID != strings.TrimSpace(restaurantID) {
			continue
		}
		if ownerCatalogItemID != "" && item.OwnerCatalogItemID != strings.TrimSpace(ownerCatalogItemID) {
			continue
		}
		if status != "" && string(item.Status) != strings.TrimSpace(status) {
			continue
		}
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].OwnerCatalogItemID == out[j].OwnerCatalogItemID {
			return out[i].Version > out[j].Version
		}
		return out[i].OwnerCatalogItemID < out[j].OwnerCatalogItemID
	})
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset > len(out) {
		return []domain.RecipeVersion{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return slices.Clone(out[offset:end]), nil
}

func (r *Repository) ListRecipeLines(_ context.Context, recipeVersionID string) ([]domain.RecipeLine, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.recipeLines[strings.TrimSpace(recipeVersionID)]), nil
}

func (r *Repository) SubmitRecipeSuggestion(_ context.Context, v domain.RecipeSuggestion, changes []domain.RecipeSuggestionChange) (domain.RecipeSuggestion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.recipeSuggestions {
		if existing.SuggestionID == v.SuggestionID {
			return existing, nil
		}
	}
	r.recipeSuggestions[v.ID] = v
	r.recipeSuggestionChanges[v.ID] = slices.Clone(changes)
	return v, nil
}

func (r *Repository) ActivateRecipeVersion(_ context.Context, versionID, approvedBy string, now time.Time) (domain.RecipeVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	target, ok := r.recipeVersions[strings.TrimSpace(versionID)]
	if !ok {
		return domain.RecipeVersion{}, domain.ErrNotFound
	}
	for id, item := range r.recipeVersions {
		if item.RestaurantID == target.RestaurantID && item.OwnerCatalogItemID == target.OwnerCatalogItemID && item.Status == domain.RecipeVersionStatusActive {
			item.Status = domain.RecipeVersionStatusArchived
			item.UpdatedAt = now
			r.recipeVersions[id] = item
		}
	}
	target.Status = domain.RecipeVersionStatusActive
	target.ApprovedByEmployeeID = strings.TrimSpace(approvedBy)
	target.ApprovedAt = &now
	target.UpdatedAt = now
	r.recipeVersions[target.ID] = target
	return target, nil
}

func (r *Repository) UpsertStopListEntry(_ context.Context, v domain.StopListEntry) (domain.StopListEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, item := range r.stopLists {
		if item.RestaurantID == v.RestaurantID && item.CatalogItemID == v.CatalogItemID {
			v.ID = item.ID
			r.stopLists[id] = v
			return v, nil
		}
	}
	r.stopLists[v.ID] = v
	return v, nil
}

func (r *Repository) GetStopListEntry(_ context.Context, id string) (domain.StopListEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.stopLists[strings.TrimSpace(id)]
	if !ok {
		return domain.StopListEntry{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) ListStopListEntries(_ context.Context, restaurantID string) ([]domain.StopListEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.StopListEntry
	for _, item := range r.stopLists {
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

func (r *Repository) ListAssignedNodeDeviceIDs(_ context.Context, restaurantID string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	nodes := r.assignedNodes[strings.TrimSpace(restaurantID)]
	out := make([]string, 0, len(nodes))
	for nodeDeviceID := range nodes {
		out = append(out, nodeDeviceID)
	}
	sort.Strings(out)
	return out, nil
}

func (r *Repository) ListAssignedRestaurantIDs(_ context.Context) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.assignedNodes))
	for restaurantID, nodes := range r.assignedNodes {
		if restaurantID != "" && len(nodes) > 0 {
			out = append(out, restaurantID)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (r *Repository) AssignEdgeNodeForTest(restaurantID, nodeDeviceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	restaurantID = strings.TrimSpace(restaurantID)
	nodeDeviceID = strings.TrimSpace(nodeDeviceID)
	if restaurantID == "" || nodeDeviceID == "" {
		return
	}
	if r.assignedNodes[restaurantID] == nil {
		r.assignedNodes[restaurantID] = map[string]struct{}{}
	}
	r.assignedNodes[restaurantID][nodeDeviceID] = struct{}{}
}

func (r *Repository) NextPublicationVersion(_ context.Context, restaurantID string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return int64(len(r.publications[strings.TrimSpace(restaurantID)]) + 1), nil
}

func (r *Repository) GetDeliveryState(_ context.Context, nodeDeviceID string) (app.DeliveryStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.deliveryStates[strings.TrimSpace(nodeDeviceID)]
	if !ok {
		return app.DeliveryStatus{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) ListDeliveryStates(_ context.Context, restaurantID string) ([]app.DeliveryStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]app.DeliveryStatus, 0)
	for _, v := range r.deliveryStates {
		if v.RestaurantID == strings.TrimSpace(restaurantID) {
			out = append(out, v)
		}
	}
	slices.SortFunc(out, func(a, b app.DeliveryStatus) int { return strings.Compare(a.NodeDeviceID, b.NodeDeviceID) })
	return out, nil
}

func (r *Repository) MarkDeliveryError(_ context.Context, restaurantID, nodeDeviceID, errorCode string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	v := r.deliveryStates[strings.TrimSpace(nodeDeviceID)]
	v.NodeDeviceID = strings.TrimSpace(nodeDeviceID)
	v.RestaurantID = strings.TrimSpace(restaurantID)
	v.Status = "error"
	v.LastErrorCode = strings.TrimSpace(errorCode)
	v.ConsecutiveFailures++
	v.NextRetryAt = &now
	v.UpdatedAt = now
	r.deliveryStates[v.NodeDeviceID] = v
	return nil
}

func (r *Repository) SavePublication(_ context.Context, pub domain.Publication, packages []app.StreamPackage, deliveries []app.DeliveryStatus) (domain.Publication, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.publications[pub.RestaurantID] = append(r.publications[pub.RestaurantID], clonePublication(pub))
	for _, pkg := range packages {
		r.packages[pkg.StreamName+"|"+pkg.NodeDeviceID] = clonePackage(pkg)
	}
	for _, delivery := range deliveries {
		r.deliveryStates[delivery.NodeDeviceID] = delivery
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

func (r *Repository) ListCatalogSuggestions(_ context.Context, restaurantID, status string, limit, offset int) ([]domain.CatalogSuggestion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.CatalogSuggestion, 0)
	for _, item := range r.catalogSuggestions {
		if restaurantID != "" && item.RestaurantID != restaurantID {
			continue
		}
		if status != "" && string(item.Status) != status {
			continue
		}
		out = append(out, item)
	}
	if offset > len(out) {
		return []domain.CatalogSuggestion{}, nil
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 50
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return slices.Clone(out[offset:end]), nil
}

func (r *Repository) GetCatalogSuggestion(_ context.Context, id string) (domain.CatalogSuggestion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.catalogSuggestions[strings.TrimSpace(id)]
	if !ok {
		return domain.CatalogSuggestion{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) UpdateCatalogSuggestion(_ context.Context, v domain.CatalogSuggestion) (domain.CatalogSuggestion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.catalogSuggestions[v.ID]; !ok {
		return domain.CatalogSuggestion{}, domain.ErrNotFound
	}
	r.catalogSuggestions[v.ID] = v
	return v, nil
}

// SeedCatalogSuggestion добавляет safe projection row для service/API tests.
func (r *Repository) SeedCatalogSuggestion(v domain.CatalogSuggestion) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.catalogSuggestions[v.ID] = v
}

func (r *Repository) ListRecipeSuggestions(_ context.Context, restaurantID, status string, limit, offset int) ([]domain.RecipeSuggestion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.RecipeSuggestion, 0)
	for _, item := range r.recipeSuggestions {
		if restaurantID != "" && item.RestaurantID != restaurantID {
			continue
		}
		if status != "" && string(item.Status) != status {
			continue
		}
		out = append(out, item)
	}
	if offset > len(out) {
		return []domain.RecipeSuggestion{}, nil
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 50
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return slices.Clone(out[offset:end]), nil
}

func (r *Repository) GetRecipeSuggestion(_ context.Context, id string) (domain.RecipeSuggestion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.recipeSuggestions[strings.TrimSpace(id)]
	if !ok {
		return domain.RecipeSuggestion{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) UpdateRecipeSuggestion(_ context.Context, v domain.RecipeSuggestion) (domain.RecipeSuggestion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.recipeSuggestions[v.ID]; !ok {
		return domain.RecipeSuggestion{}, domain.ErrNotFound
	}
	r.recipeSuggestions[v.ID] = v
	return v, nil
}

func (r *Repository) ListRecipeSuggestionChanges(_ context.Context, recipeSuggestionID string) ([]domain.RecipeSuggestionChange, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.recipeSuggestionChanges[strings.TrimSpace(recipeSuggestionID)]), nil
}

func (r *Repository) ListStopListUpdateReviews(_ context.Context, restaurantID, status string, limit, offset int) ([]domain.StopListUpdateReview, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.StopListUpdateReview, 0)
	for _, item := range r.stopListUpdates {
		if item.ProjectionAction != "requires_manager_review" {
			continue
		}
		if restaurantID != "" && item.RestaurantID != restaurantID {
			continue
		}
		if status != "" && string(item.Status) != status {
			continue
		}
		out = append(out, item)
	}
	slices.SortFunc(out, func(a, b domain.StopListUpdateReview) int {
		if cmp := b.ProjectedAt.Compare(a.ProjectedAt); cmp != 0 {
			return cmp
		}
		return strings.Compare(b.ID, a.ID)
	})
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset > len(out) {
		return []domain.StopListUpdateReview{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return slices.Clone(out[offset:end]), nil
}

func (r *Repository) GetStopListUpdateReview(_ context.Context, id string) (domain.StopListUpdateReview, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.stopListUpdates[strings.TrimSpace(id)]
	if !ok || v.ProjectionAction != "requires_manager_review" {
		return domain.StopListUpdateReview{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) UpdateStopListUpdateReview(_ context.Context, v domain.StopListUpdateReview) (domain.StopListUpdateReview, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.stopListUpdates[v.ID]; !ok {
		return domain.StopListUpdateReview{}, domain.ErrNotFound
	}
	r.stopListUpdates[v.ID] = v
	return v, nil
}

// SeedStopListUpdateReview добавляет safe projection row для service/API tests.
func (r *Repository) SeedStopListUpdateReview(v domain.StopListUpdateReview) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopListUpdates[v.ID] = v
}

func (r *Repository) GetReviewAssignmentAuditEvent(_ context.Context, commandID string) (domain.ReviewAssignmentAuditEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.assignmentAuditEvents[strings.TrimSpace(commandID)]
	if !ok {
		return domain.ReviewAssignmentAuditEvent{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *Repository) AppendReviewAssignmentAuditEvent(_ context.Context, v domain.ReviewAssignmentAuditEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	commandID := strings.TrimSpace(v.CommandID)
	if _, exists := r.assignmentAuditEvents[commandID]; exists {
		return domain.ErrConflict
	}
	r.assignmentAuditEvents[commandID] = v
	return nil
}

func (r *Repository) ListReviewAssignmentAuditEvents(_ context.Context, reviewType, reviewID string, limit, offset int) ([]domain.ReviewAssignmentAuditEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	reviewType = strings.TrimSpace(reviewType)
	reviewID = strings.TrimSpace(reviewID)
	out := make([]domain.ReviewAssignmentAuditEvent, 0)
	for _, event := range r.assignmentAuditEvents {
		if event.ReviewType != reviewType || event.ReviewID != reviewID {
			continue
		}
		out = append(out, event)
	}
	slices.SortFunc(out, func(a, b domain.ReviewAssignmentAuditEvent) int {
		if cmp := b.OccurredAt.Compare(a.OccurredAt); cmp != 0 {
			return cmp
		}
		return strings.Compare(b.EventID, a.EventID)
	})
	if offset > len(out) {
		return []domain.ReviewAssignmentAuditEvent{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return slices.Clone(out[offset:end]), nil
}

func (r *Repository) ReviewAssignmentAuditEvents() []domain.ReviewAssignmentAuditEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.ReviewAssignmentAuditEvent, 0, len(r.assignmentAuditEvents))
	for _, event := range r.assignmentAuditEvents {
		out = append(out, event)
	}
	slices.SortFunc(out, func(a, b domain.ReviewAssignmentAuditEvent) int {
		if cmp := a.OccurredAt.Compare(b.OccurredAt); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.CommandID, b.CommandID)
	})
	return out
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
