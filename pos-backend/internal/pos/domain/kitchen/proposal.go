package kitchen

import (
	"encoding/json"
	"time"
)

// ProposalKind разделяет локальные kitchen proposals по Cloud-owned review queue.
type ProposalKind string

const (
	ProposalKindCatalog ProposalKind = "catalog"
	ProposalKindRecipe  ProposalKind = "recipe"
)

// ProposalStatus хранит локальный Edge статус предложения до Cloud feedback/publication.
type ProposalStatus string

const (
	ProposalDraft            ProposalStatus = "draft"
	ProposalPendingSync      ProposalStatus = "pending_sync"
	ProposalSynced           ProposalStatus = "synced"
	ProposalApproved         ProposalStatus = "approved"
	ProposalRejected         ProposalStatus = "rejected"
	ProposalChangesRequested ProposalStatus = "changes_requested"
	ProposalFailed           ProposalStatus = "failed"
)

// Proposal описывает Edge-owned запись предложения кухни; payload остается immutable snapshot.
type Proposal struct {
	ID                       string          `json:"id"`
	RestaurantID             string          `json:"restaurant_id"`
	ProposalGroupID          string          `json:"proposal_group_id,omitempty"`
	Kind                     ProposalKind    `json:"kind"`
	Status                   ProposalStatus  `json:"status"`
	Action                   string          `json:"action"`
	OwnerCatalogItemID       string          `json:"owner_catalog_item_id,omitempty"`
	OwnerCatalogSuggestionID string          `json:"owner_catalog_suggestion_id,omitempty"`
	RecipeVersionID          string          `json:"recipe_version_id,omitempty"`
	Payload                  json.RawMessage `json:"payload"`
	OutboxCommandID          string          `json:"outbox_command_id"`
	OutboxEventType          string          `json:"outbox_event_type"`
	CreatedByEmployeeID      string          `json:"created_by_employee_id"`
	CreatedAt                time.Time       `json:"created_at"`
	UpdatedAt                time.Time       `json:"updated_at"`
	CloudVersion             *int64          `json:"cloud_version,omitempty"`
	CloudUpdatedAt           *string         `json:"cloud_updated_at,omitempty"`
	Replayed                 bool            `json:"replayed,omitempty"`
}

type ProposalListQuery struct {
	RestaurantID       string
	Kind               ProposalKind
	Status             ProposalStatus
	OwnerCatalogItemID string
	RecipeVersionID    string
	OutboxCommandID    string
	OutboxEventType    string
	IncludeTerminal    bool
	Limit              int
	Offset             int
}
