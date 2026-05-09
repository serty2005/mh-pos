package licensehttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud-backend/internal/provisioning/app"
	"cloud-backend/internal/provisioning/domain"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), http: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) RegisterPairingCode(ctx context.Context, payload app.LicensePairingPayload) error {
	if c == nil || c.baseURL == "" {
		return domain.ErrLicenseServerUnavailable
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/pairing-codes", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrLicenseServerUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: license server status %d", domain.ErrLicenseServerUnavailable, resp.StatusCode)
	}
	return nil
}
