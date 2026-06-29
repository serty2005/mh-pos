// Package licensegate проверяет module entitlements через внешний License Server.
package licensegate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	CloudSubscription = "cloud-subscription"
	TableMode         = "table-mode"
	KitchenSpace      = "kitchen-space"
	WarehouseMode     = "warehouse-mode"
	WaiterSpace       = "waiter-space"
	TelegramWorker    = "telegram-worker"
	TicketMode        = "ticket-mode"
)

var (
	ErrDenied      = errors.New("license entitlement denied")
	ErrUnavailable = errors.New("license authority unavailable")
)

// ProductModule описывает один canonical product module entitlement.
type ProductModule struct {
	ID          string
	Label       string
	Description string
}

// CanonicalModules возвращает полный product-owned catalog в стабильном порядке.
func CanonicalModules() []ProductModule {
	return []ProductModule{
		{ID: CloudSubscription, Label: "Tenant Cloud", Description: "Cloud backoffice, tenant management, delivery, analytics, templates, printers"},
		{ID: TableMode, Label: "Table mode", Description: "Halls, tables, table-bound flow, floor settings and floor delivery"},
		{ID: KitchenSpace, Label: "Kitchen space", Description: "KDS, kitchen routes/actions, recipes, kitchen events and proposals"},
		{ID: WarehouseMode, Label: "Warehouse mode", Description: "Inventory reference, stock operations, worker, ledger, balances and costing"},
		{ID: WaiterSpace, Label: "Waiter space", Description: "Dedicated mobile-first waiter access when backend-discriminated"},
		{ID: TelegramWorker, Label: "Telegram worker", Description: "Telegram settings, queues, worker and scheduled reports"},
		{ID: TicketMode, Label: "Ticket mode", Description: "QR services, ticket issuance/templates/printing and checker runtime"},
	}
}

// CanonicalModuleIDs возвращает только machine-readable IDs product catalog.
func CanonicalModuleIDs() []string {
	modules := CanonicalModules()
	ids := make([]string, 0, len(modules))
	for _, module := range modules {
		ids = append(ids, module.ID)
	}
	return ids
}

// EntitlementsForModules строит snapshot map, где перечислены все canonical IDs.
func EntitlementsForModules(enabledIDs ...string) map[string]bool {
	out := make(map[string]bool, len(CanonicalModules()))
	for _, module := range CanonicalModules() {
		out[module.ID] = false
	}
	for _, id := range enabledIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			out[id] = true
		}
	}
	return out
}

// ModuleForCloudStream возвращает module entitlement для Cloud -> Edge stream.
func ModuleForCloudStream(streamName string) string {
	switch strings.TrimSpace(streamName) {
	case "floor":
		return TableMode
	case "recipes":
		return KitchenSpace
	case "inventory_reference":
		return WarehouseMode
	case "receipt_templates", "printers":
		return CloudSubscription
	default:
		return ""
	}
}

// ModuleForEdgeEvent возвращает module entitlement для Edge -> Cloud module-owned event.
func ModuleForEdgeEvent(eventType string) string {
	switch strings.TrimSpace(eventType) {
	case "TicketIssued":
		return TicketMode
	case "KitchenTicketStatusChanged", "ItemServed", "CatalogItemChangeSuggested", "RecipeChangeSuggested":
		return KitchenSpace
	case "StockReceiptCaptured", "InventoryCountCaptured", "StockWriteOffCaptured", "ProductionCompleted", "StopListUpdated":
		return WarehouseMode
	default:
		return ""
	}
}

// Snapshot является versioned ответом внешнего licensing authority.
type Snapshot struct {
	TenantID     string          `json:"tenant_id"`
	ServerID     string          `json:"server_id"`
	Version      int64           `json:"version"`
	Status       string          `json:"status"`
	Entitlements map[string]bool `json:"entitlements"`
	IssuedAt     time.Time       `json:"issued_at"`
	ExpiresAt    time.Time       `json:"expires_at"`
}

// Gate является минимальной границей проверки module entitlement.
type Gate interface {
	Require(context.Context, string) error
	Current(context.Context) (Snapshot, error)
}

// Client запрашивает authority на каждой проверке и использует только bounded provider grace.
type Client struct {
	endpoint   string
	grace      time.Duration
	http       *http.Client
	mu         sync.RWMutex
	cached     Snapshot
	usingGrace bool
}

func NewClient(baseURL, tenantID, serverID string, grace time.Duration) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	endpoint := baseURL + "/api/v1/entitlements/" + url.PathEscape(strings.TrimSpace(tenantID)) + "/" + url.PathEscape(strings.TrimSpace(serverID))
	return &Client{endpoint: endpoint, grace: max(grace, 0), http: &http.Client{Timeout: 5 * time.Second}}
}

func (c *Client) Current(ctx context.Context) (Snapshot, error) {
	if c == nil || strings.TrimSpace(c.endpoint) == "" {
		return Snapshot{}, ErrUnavailable
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err == nil {
		var resp *http.Response
		resp, err = c.http.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var snapshot Snapshot
				if decodeErr := json.NewDecoder(resp.Body).Decode(&snapshot); decodeErr == nil && valid(snapshot) {
					c.mu.Lock()
					if snapshot.Version >= c.cached.Version {
						c.cached = snapshot
					}
					c.usingGrace = false
					cached := c.cached
					c.mu.Unlock()
					return cached, nil
				}
			}
			err = fmt.Errorf("license authority status %d", resp.StatusCode)
		}
	}
	c.mu.RLock()
	cached := c.cached
	c.mu.RUnlock()
	if valid(cached) && cached.Status == "active" && time.Now().UTC().Before(cached.ExpiresAt.Add(c.grace)) {
		c.mu.Lock()
		c.usingGrace = true
		c.mu.Unlock()
		return cached, nil
	}
	return Snapshot{}, fmt.Errorf("%w", ErrUnavailable)
}

func (c *Client) Require(ctx context.Context, moduleID string) error {
	snapshot, err := c.Current(ctx)
	if err != nil {
		return err
	}
	c.mu.RLock()
	usingGrace := c.usingGrace
	c.mu.RUnlock()
	if snapshot.Status != "active" || (!usingGrace && !snapshot.ExpiresAt.After(time.Now().UTC())) || !snapshot.Entitlements[strings.TrimSpace(moduleID)] {
		return fmt.Errorf("%w: module unavailable", ErrDenied)
	}
	return nil
}

func valid(v Snapshot) bool {
	return strings.TrimSpace(v.TenantID) != "" && strings.TrimSpace(v.ServerID) != "" && v.Version > 0 && !v.ExpiresAt.IsZero() && (v.Status == "active" || v.Status == "revoked")
}
