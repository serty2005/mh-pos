package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"cloud-backend/internal/masterdata/app"
	"cloud-backend/internal/masterdata/domain"
	httpx "cloud-backend/internal/platform/httpx"
)

// Handler содержит thin HTTP handlers для Cloud master-data API.
type Handler struct {
	service *app.Service
}

// RegisterRoutes подключает Cloud master-data routes к общему API router.
func RegisterRoutes(r chi.Router, service *app.Service) {
	if service == nil {
		return
	}
	h := &Handler{service: service}
	r.Route("/master-data", func(r chi.Router) {
		r.Post("/roles", h.createRole)
		r.Get("/roles", h.listRoles)
		r.Get("/roles/{id}", h.getRole)
		r.Patch("/roles/{id}", h.updateRole)
		r.Post("/roles/{id}/archive", h.archiveRole)
		r.Post("/employees", h.createEmployee)
		r.Get("/employees", h.listEmployees)
		r.Get("/employees/{id}", h.getEmployee)
		r.Patch("/employees/{id}", h.updateEmployee)
		r.Post("/employees/{id}/suspend", h.suspendEmployee)
		r.Post("/employees/{id}/activate", h.activateEmployee)
		r.Post("/employees/{id}/archive", h.archiveEmployee)
		r.Post("/employees/{id}/role", h.assignEmployeeRole)
		r.Post("/employees/{id}/pin", h.rotateEmployeePIN)
		r.Post("/employees/{id}/pin/rotate", h.rotateEmployeePIN)
		r.Post("/catalog/items", h.createCatalogItem)
		r.Get("/catalog/items", h.listCatalogItems)
		r.Get("/catalog/items/{id}", h.getCatalogItem)
		r.Patch("/catalog/items/{id}", h.updateCatalogItem)
		r.Post("/catalog/items/{id}/archive", h.archiveCatalogItem)
		r.Post("/catalog/folders", h.createCatalogFolder)
		r.Get("/catalog/folders", h.listCatalogFolders)
		r.Patch("/catalog/folders/{id}", h.updateCatalogFolder)
		r.Post("/catalog/folders/{id}/archive", h.archiveCatalogFolder)
		r.Post("/catalog/folder-parameters", h.createFolderParameter)
		r.Get("/catalog/folder-parameters", h.listFolderParameters)
		r.Patch("/catalog/folder-parameters/{id}", h.updateFolderParameter)
		r.Post("/catalog/tags", h.createCatalogTag)
		r.Get("/catalog/tags", h.listCatalogTags)
		r.Patch("/catalog/tags/{id}", h.updateCatalogTag)
		r.Post("/catalog/item-tags", h.assignCatalogItemTag)
		r.Post("/modifiers/groups", h.createModifierGroup)
		r.Get("/modifiers/groups", h.listModifierGroups)
		r.Patch("/modifiers/groups/{id}", h.updateModifierGroup)
		r.Post("/modifiers/options", h.createModifierOption)
		r.Get("/modifiers/options", h.listModifierOptions)
		r.Patch("/modifiers/options/{id}", h.updateModifierOption)
		r.Post("/modifiers/bindings", h.createModifierGroupBinding)
		r.Get("/modifiers/bindings", h.listModifierGroupBindings)
		r.Patch("/modifiers/bindings/{id}", h.updateModifierGroupBinding)
		r.Post("/pricing/policies", h.createPricingPolicy)
		r.Get("/pricing/policies", h.listPricingPolicies)
		r.Patch("/pricing/policies/{id}", h.updatePricingPolicy)
		r.Post("/recipes/versions/drafts", h.createRecipeVersionDraft)
		r.Get("/recipes/versions", h.listRecipeVersions)
		r.Post("/recipes/versions/{id}/submit", h.submitRecipeVersion)
		r.Post("/recipes/items", h.createRecipeItem)
		r.Get("/recipes/items", h.listRecipeItems)
		r.Patch("/recipes/items/{id}", h.updateRecipeItem)
		r.Post("/inventory/stop-list", h.upsertStopListEntry)
		r.Get("/inventory/stop-list", h.listStopListEntries)
		r.Patch("/inventory/stop-list/{id}", h.updateStopListEntry)
		r.Post("/inventory/stop-list/{id}/deactivate", h.deactivateStopListEntry)
		r.Post("/menu/categories", h.createCategory)
		r.Post("/floor/halls", h.createHall)
		r.Get("/floor/halls", h.listHalls)
		r.Patch("/floor/halls/{id}", h.updateHall)
		r.Post("/floor/halls/{id}/archive", h.archiveHall)
		r.Post("/floor/tables", h.createTable)
		r.Get("/floor/tables", h.listTables)
		r.Patch("/floor/tables/{id}", h.updateTable)
		r.Post("/floor/tables/{id}/archive", h.archiveTable)
		r.Post("/menu/items", h.createMenuItem)
		r.Get("/menu/items", h.listMenuItems)
		r.Get("/menu/items/{id}", h.getMenuItem)
		r.Patch("/menu/items/{id}", h.updateMenuItem)
		r.Post("/menu/items/{id}/archive", h.archiveMenuItem)
		r.Post("/publications", h.publish)
		r.Get("/published", h.getPublished)
		r.Get("/catalog-suggestions", h.listCatalogSuggestions)
		r.Post("/catalog-suggestions/{id}/approve", h.approveCatalogSuggestion)
		r.Post("/catalog-suggestions/{id}/reject", h.rejectCatalogSuggestion)
		r.Post("/catalog-suggestions/{id}/request-changes", h.requestChangesCatalogSuggestion)
		r.Get("/recipe-suggestions", h.listRecipeSuggestions)
		r.Post("/recipe-suggestions/{id}/approve", h.approveRecipeSuggestion)
		r.Post("/recipe-suggestions/{id}/reject", h.rejectRecipeSuggestion)
		r.Post("/recipe-suggestions/{id}/request-changes", h.requestChangesRecipeSuggestion)
	})
	r.Route("/manager", func(r chi.Router) {
		r.Get("/catalog-suggestions/{id}/audit", h.listCatalogSuggestionReviewAudit)
		r.Get("/recipe-suggestions/{id}/audit", h.listRecipeSuggestionReviewAudit)
		r.Get("/stop-list-updates", h.listStopListUpdateReviews)
		r.Get("/stop-list-updates/{id}", h.getStopListUpdateReview)
		r.Get("/stop-list-updates/{id}/audit", h.listStopListUpdateReviewAudit)
		r.Post("/stop-list-updates/{id}/approve", h.approveStopListUpdateReview)
		r.Post("/stop-list-updates/{id}/reject", h.rejectStopListUpdateReview)
		r.Post("/stop-list-updates/{id}/request-changes", h.requestChangesStopListUpdateReview)
		r.Post("/stop-list-updates/{id}/assign", h.assignStopListUpdateReview)
		r.Post("/stop-list-updates/{id}/unassign", h.unassignStopListUpdateReview)
	})
	r.Post("/restaurants", h.createRestaurant)
	r.Get("/restaurants", h.listRestaurants)
	r.Get("/restaurants/{id}", h.getRestaurant)
	r.Patch("/restaurants/{id}", h.updateRestaurant)
	r.Post("/restaurants/{id}/archive", h.archiveRestaurant)
	r.Patch("/restaurants/{id}/archive", h.archiveRestaurant)
	r.Post("/roles", h.createRole)
	r.Get("/roles", h.listRoles)
	r.Get("/roles/{id}", h.getRole)
	r.Patch("/roles/{id}", h.updateRole)
	r.Post("/roles/{id}/archive", h.archiveRole)
	r.Patch("/roles/{id}/archive", h.archiveRole)
	r.Post("/employees", h.createEmployee)
	r.Get("/employees", h.listEmployees)
	r.Get("/employees/{id}", h.getEmployee)
	r.Patch("/employees/{id}", h.updateEmployee)
	r.Post("/employees/{id}/suspend", h.suspendEmployee)
	r.Post("/employees/{id}/activate", h.activateEmployee)
	r.Post("/employees/{id}/archive", h.archiveEmployee)
	r.Post("/employees/{id}/pin", h.rotateEmployeePIN)
	r.Post("/employees/{id}/pin/rotate", h.rotateEmployeePIN)
	r.Post("/catalog/items", h.createCatalogItem)
	r.Get("/catalog/items", h.listCatalogItems)
	r.Get("/catalog/items/{id}", h.getCatalogItem)
	r.Patch("/catalog/items/{id}", h.updateCatalogItem)
	r.Post("/catalog/items/{id}/archive", h.archiveCatalogItem)
	r.Post("/menu/items", h.createMenuItem)
	r.Get("/menu/items", h.listMenuItems)
	r.Get("/menu/items/{id}", h.getMenuItem)
	r.Patch("/menu/items/{id}", h.updateMenuItem)
	r.Post("/menu/items/{id}/archive", h.archiveMenuItem)
	r.Post("/halls", h.createHall)
	r.Get("/halls", h.listHalls)
	r.Patch("/halls/{id}", h.updateHall)
	r.Post("/halls/{id}/archive", h.archiveHall)
	r.Post("/tables", h.createTable)
	r.Get("/tables", h.listTables)
	r.Patch("/tables/{id}", h.updateTable)
	r.Post("/tables/{id}/archive", h.archiveTable)
	r.Post("/restaurants/{id}/master-data/publish", h.publishRestaurant)
	r.Get("/restaurants/{id}/master-data/publication-state", h.getRestaurantPublished)
	r.Get("/restaurants/{id}/master-data/packages/latest", h.getLatestRestaurantPackage)
	r.Get("/restaurants/{id}/master-data/packages/{package_id}", h.getRestaurantPackage)
	r.Get("/restaurants/{id}/edge-nodes/{node_device_id}/master-data/snapshot", h.getEdgeNodeSnapshot)
}

func (h *Handler) createRestaurant(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateRestaurantCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateRestaurant(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) listRestaurants(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListRestaurants(r.Context())
	write(w, http.StatusOK, v, err)
}

func (h *Handler) getRestaurant(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetRestaurant(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) updateRestaurant(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateRestaurantCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateRestaurant(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) archiveRestaurant(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ArchiveRestaurant(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) createRole(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateRoleCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateRole(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListRoles(r.Context(), r.URL.Query().Get("restaurant_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) getRole(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetRole(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) updateRole(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateRoleCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateRole(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) archiveRole(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ArchiveRole(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) createEmployee(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateEmployeeCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateEmployee(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) listEmployees(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListEmployees(r.Context(), r.URL.Query().Get("restaurant_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) getEmployee(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetEmployee(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) updateEmployee(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateEmployeeCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateEmployee(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) suspendEmployee(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.SuspendEmployee(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) activateEmployee(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ActivateEmployee(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) archiveEmployee(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ArchiveEmployee(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) assignEmployeeRole(w http.ResponseWriter, r *http.Request) {
	var cmd app.AssignRoleCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.AssignEmployeeRole(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) rotateEmployeePIN(w http.ResponseWriter, r *http.Request) {
	var cmd app.RotatePINCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.RotateEmployeePIN(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) createCatalogItem(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateCatalogItemCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateCatalogItem(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) listCatalogItems(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListCatalogItems(r.Context(), r.URL.Query().Get("restaurant_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) getCatalogItem(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetCatalogItem(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) updateCatalogItem(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateCatalogItemCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateCatalogItem(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) archiveCatalogItem(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ArchiveCatalogItem(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) createCatalogFolder(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateCatalogFolderCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateCatalogFolder(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) listCatalogFolders(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListCatalogFolders(r.Context(), r.URL.Query().Get("restaurant_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) updateCatalogFolder(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateCatalogFolderCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateCatalogFolder(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) archiveCatalogFolder(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ArchiveCatalogFolder(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) createFolderParameter(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateFolderParameterCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateFolderParameter(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) listFolderParameters(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListFolderParameters(r.Context(), r.URL.Query().Get("restaurant_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) updateFolderParameter(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateFolderParameterCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateFolderParameter(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) createCatalogTag(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateCatalogTagCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateCatalogTag(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) listCatalogTags(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListCatalogTags(r.Context(), r.URL.Query().Get("restaurant_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) updateCatalogTag(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateCatalogTagCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateCatalogTag(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) assignCatalogItemTag(w http.ResponseWriter, r *http.Request) {
	var cmd app.AssignCatalogItemTagCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.AssignCatalogItemTag(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) createModifierGroup(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateModifierGroupCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateModifierGroup(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) listModifierGroups(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListModifierGroups(r.Context(), r.URL.Query().Get("restaurant_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) updateModifierGroup(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateModifierGroupCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateModifierGroup(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) createModifierOption(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateModifierOptionCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateModifierOption(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) listModifierOptions(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListModifierOptions(r.Context(), r.URL.Query().Get("restaurant_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) updateModifierOption(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateModifierOptionCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateModifierOption(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) createModifierGroupBinding(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateModifierGroupBindingCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateModifierGroupBinding(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) listModifierGroupBindings(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListModifierGroupBindings(r.Context(), r.URL.Query().Get("restaurant_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) updateModifierGroupBinding(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateModifierGroupBindingCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateModifierGroupBinding(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) createPricingPolicy(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreatePricingPolicyCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreatePricingPolicy(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) listPricingPolicies(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListPricingPolicies(r.Context(), r.URL.Query().Get("restaurant_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) updatePricingPolicy(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdatePricingPolicyCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdatePricingPolicy(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) createRecipeItem(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateRecipeItemCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateRecipeItem(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) createRecipeVersionDraft(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateRecipeVersionDraftCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateRecipeVersionDraft(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) listRecipeVersions(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListRecipeVersions(
		r.Context(),
		r.URL.Query().Get("restaurant_id"),
		r.URL.Query().Get("owner_catalog_item_id"),
		r.URL.Query().Get("status"),
		intQuery(r, "limit", 50),
		intQuery(r, "offset", 0),
	)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) submitRecipeVersion(w http.ResponseWriter, r *http.Request) {
	var cmd app.SubmitRecipeVersionCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.SubmitRecipeVersion(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) listRecipeItems(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListRecipeItems(r.Context(), r.URL.Query().Get("restaurant_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) updateRecipeItem(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateRecipeItemCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateRecipeItem(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) upsertStopListEntry(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpsertStopListEntryCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpsertStopListEntry(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) listStopListEntries(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListStopListEntries(r.Context(), r.URL.Query().Get("restaurant_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) updateStopListEntry(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpsertStopListEntryCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateStopListEntry(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) deactivateStopListEntry(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.DeactivateStopListEntry(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateCategoryCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateCategory(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) createHall(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateHallCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateHall(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) listHalls(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListHalls(r.Context(), r.URL.Query().Get("restaurant_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) updateHall(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateHallCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateHall(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) archiveHall(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ArchiveHall(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) createTable(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateTableCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateTable(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) listTables(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListTables(r.Context(), r.URL.Query().Get("restaurant_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) updateTable(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateTableCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateTable(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) archiveTable(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ArchiveTable(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) createMenuItem(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateMenuItemCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateMenuItem(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) listMenuItems(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListMenuItems(r.Context(), r.URL.Query().Get("restaurant_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) getMenuItem(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetMenuItem(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) updateMenuItem(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateMenuItemCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateMenuItem(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) archiveMenuItem(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ArchiveMenuItem(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	var cmd app.PublishCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.Publish(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) publishRestaurant(w http.ResponseWriter, r *http.Request) {
	var cmd app.PublishCommand
	if !decode(w, r, &cmd) {
		return
	}
	cmd.RestaurantID = chi.URLParam(r, "id")
	v, err := h.service.Publish(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) getPublished(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetCurrentPublishedState(r.Context(), r.URL.Query().Get("restaurant_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) getRestaurantPublished(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetCurrentPublishedState(r.Context(), chi.URLParam(r, "id"))
	writeOptional(w, http.StatusOK, v, err)
}

func (h *Handler) getLatestRestaurantPackage(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetCurrentPublishedPackage(r.Context(), chi.URLParam(r, "id"), r.URL.Query().Get("node_device_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) getRestaurantPackage(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetPublishedPackage(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "package_id"), r.URL.Query().Get("node_device_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) getEdgeNodeSnapshot(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetCurrentPublishedPackage(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "node_device_id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) listCatalogSuggestions(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListCatalogSuggestions(r.Context(), r.URL.Query().Get("restaurant_id"), r.URL.Query().Get("status"), intQuery(r, "limit", 50), intQuery(r, "offset", 0))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) approveCatalogSuggestion(w http.ResponseWriter, r *http.Request) {
	var cmd app.SuggestionReviewCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.ApproveCatalogSuggestion(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) rejectCatalogSuggestion(w http.ResponseWriter, r *http.Request) {
	var cmd app.SuggestionReviewCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.RejectCatalogSuggestion(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) requestChangesCatalogSuggestion(w http.ResponseWriter, r *http.Request) {
	var cmd app.SuggestionReviewCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.RequestChangesCatalogSuggestion(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) listRecipeSuggestions(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListRecipeSuggestions(r.Context(), r.URL.Query().Get("restaurant_id"), r.URL.Query().Get("status"), intQuery(r, "limit", 50), intQuery(r, "offset", 0))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) approveRecipeSuggestion(w http.ResponseWriter, r *http.Request) {
	var cmd app.SuggestionReviewCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.ApproveRecipeSuggestion(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) rejectRecipeSuggestion(w http.ResponseWriter, r *http.Request) {
	var cmd app.SuggestionReviewCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.RejectRecipeSuggestion(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) requestChangesRecipeSuggestion(w http.ResponseWriter, r *http.Request) {
	var cmd app.SuggestionReviewCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.RequestChangesRecipeSuggestion(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) listStopListUpdateReviews(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListStopListUpdateReviews(r.Context(), r.URL.Query().Get("restaurant_id"), r.URL.Query().Get("status"), intQuery(r, "limit", 50), intQuery(r, "offset", 0))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) getStopListUpdateReview(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetStopListUpdateReview(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) listStopListUpdateReviewAudit(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListStopListUpdateReviewAudit(r.Context(), chi.URLParam(r, "id"), intQuery(r, "limit", 50), intQuery(r, "offset", 0))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) listCatalogSuggestionReviewAudit(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListCatalogSuggestionReviewAudit(r.Context(), chi.URLParam(r, "id"), intQuery(r, "limit", 50), intQuery(r, "offset", 0))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) listRecipeSuggestionReviewAudit(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListRecipeSuggestionReviewAudit(r.Context(), chi.URLParam(r, "id"), intQuery(r, "limit", 50), intQuery(r, "offset", 0))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) approveStopListUpdateReview(w http.ResponseWriter, r *http.Request) {
	var cmd app.SuggestionReviewCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.ApproveStopListUpdateReview(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) rejectStopListUpdateReview(w http.ResponseWriter, r *http.Request) {
	var cmd app.SuggestionReviewCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.RejectStopListUpdateReview(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) requestChangesStopListUpdateReview(w http.ResponseWriter, r *http.Request) {
	var cmd app.SuggestionReviewCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.RequestChangesStopListUpdateReview(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) assignStopListUpdateReview(w http.ResponseWriter, r *http.Request) {
	var cmd app.ReviewAssignCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.AssignReviewItem(r.Context(), "stop_list_update", chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) unassignStopListUpdateReview(w http.ResponseWriter, r *http.Request) {
	var cmd app.ReviewUnassignCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UnassignReviewItem(r.Context(), "stop_list_update", chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func intQuery(r *http.Request, key string, fallback int) int {
	v, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get(key)))
	if err != nil {
		return fallback
	}
	return v
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("%w: %v", domain.ErrInvalid, err))
		return false
	}
	return true
}

func write[T any](w http.ResponseWriter, status int, v T, err error) {
	if err != nil {
		code := http.StatusInternalServerError
		switch {
		case errors.Is(err, domain.ErrInvalid):
			code = http.StatusBadRequest
		case errors.Is(err, domain.ErrNotFound):
			code = http.StatusNotFound
		case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrPINAlreadyExists):
			code = http.StatusConflict
		}
		writeError(w, code, err)
		return
	}
	writeJSON(w, status, v)
}

func writeOptional[T any](w http.ResponseWriter, status int, v T, err error) {
	if errors.Is(err, domain.ErrNotFound) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte("null\n"))
		return
	}
	write(w, status, v, err)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	httpx.JSON(w, status, v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	_ = status
	httpx.Error(w, err)
}
