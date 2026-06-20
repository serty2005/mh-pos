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
	TableMode      = "table-mode"
	TelegramWorker = "telegram-worker"
	KitchenSpace   = "kitchen-space"
	WaiterSpace    = "waiter-space"
	CheckerFlow    = "checker-flow"
	WarehouseMode  = "warehouse-mode"
)

var (
	ErrDenied      = errors.New("license entitlement denied")
	ErrUnavailable = errors.New("license authority unavailable")
)

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
