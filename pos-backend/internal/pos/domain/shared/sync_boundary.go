package shared

type DataOwner string
type SyncDirection string
type SyncMode string
type MasterDataStream string

const (
	DataOwnerCloud DataOwner = "cloud"
	DataOwnerEdge  DataOwner = "edge"
	DataOwnerMixed DataOwner = "mixed"

	SyncDirectionCloudToEdge SyncDirection = "cloud_to_edge"
	SyncDirectionEdgeToCloud SyncDirection = "edge_to_cloud"
	SyncDirectionLocalOnly   SyncDirection = "local_only"

	SyncModeFullSnapshot SyncMode = "full_snapshot"
	SyncModeIncremental  SyncMode = "incremental"

	MasterDataStreamRestaurants MasterDataStream = "restaurants"
	MasterDataStreamDevices     MasterDataStream = "devices"
	MasterDataStreamStaff       MasterDataStream = "staff"
	MasterDataStreamFloor       MasterDataStream = "floor"
	MasterDataStreamCatalog     MasterDataStream = "catalog"
	MasterDataStreamMenu        MasterDataStream = "menu"
	MasterDataStreamRecipes     MasterDataStream = "recipes"
	MasterDataStreamInventory   MasterDataStream = "inventory_reference"
)

type MasterDataSyncState struct {
	ID                 string           `json:"id"`
	RestaurantID       *string          `json:"restaurant_id,omitempty"`
	NodeDeviceID       string           `json:"node_device_id"`
	StreamName         MasterDataStream `json:"stream_name"`
	Direction          SyncDirection    `json:"direction"`
	SyncMode           SyncMode         `json:"sync_mode"`
	CheckpointToken    *string          `json:"checkpoint_token,omitempty"`
	LastCloudVersion   int64            `json:"last_cloud_version"`
	LastCloudUpdatedAt *string          `json:"last_cloud_updated_at,omitempty"`
	LastAppliedAt      *string          `json:"last_applied_at,omitempty"`
	Status             string           `json:"status"`
	LastError          *string          `json:"last_error,omitempty"`
	CreatedAt          string           `json:"created_at"`
	UpdatedAt          string           `json:"updated_at"`
}

type MasterRecordSyncMeta struct {
	CloudVersion   int64
	CloudUpdatedAt *string
	CloudDeletedAt *string
	LastSyncedAt   string
}

func IsEdgeToCloudOperationalEvent(eventType string) bool {
	switch eventType {
	case "ShiftOpened",
		"ShiftClosed",
		"CashSessionOpened",
		"CashSessionClosed",
		"CashDrawerEventRecorded",
		"OrderCreated",
		"OrderLineAdded",
		"OrderLineQuantityChanged",
		"OrderLineVoided",
		"PrecheckIssued",
		"PrecheckCancelled",
		"PaymentCaptured",
		"CheckCreated",
		"OrderClosed",
		"AuthSessionStarted",
		"AuthSessionRevoked",
		"DeviceRegistered":
		return true
	default:
		return false
	}
}

func IsCloudOwnedAggregate(aggregateType string) bool {
	switch aggregateType {
	case "Restaurant", "Device", "Role", "Employee", "Hall", "Table", "CatalogItem", "MenuItem", "RecipeVersion", "RecipeLine", "InventoryReference":
		return true
	default:
		return false
	}
}

func DirectionForOutbox(origin CommandOrigin, aggregateType, eventType string) SyncDirection {
	if origin == OriginEdgeDevice && IsEdgeToCloudOperationalEvent(eventType) {
		return SyncDirectionEdgeToCloud
	}
	if origin == OriginCloudSync || IsCloudOwnedAggregate(aggregateType) {
		return SyncDirectionCloudToEdge
	}
	return SyncDirectionLocalOnly
}
