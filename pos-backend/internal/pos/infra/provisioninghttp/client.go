package provisioninghttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	appmastersync "pos-backend/internal/pos/app/mastersync"
	"pos-backend/internal/pos/app/provisioning"
	"pos-backend/internal/pos/domain"
)

type CloudClient struct {
	http *http.Client
}

type LicenseClient struct {
	http *http.Client
}

func NewCloudClient(timeout time.Duration) *CloudClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &CloudClient{http: &http.Client{Timeout: timeout}}
}

func NewLicenseClient(timeout time.Duration) *LicenseClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &LicenseClient{http: &http.Client{Timeout: timeout}}
}

func (c *CloudClient) RegisterDevice(ctx context.Context, cloudURL string, req provisioning.CloudRegisterRequest) (provisioning.CloudRegisterResponse, error) {
	var out provisioning.CloudRegisterResponse
	return out, c.post(ctx, strings.TrimRight(cloudURL, "/")+"/api/v1/devices/register", req, &out)
}

func (c *CloudClient) AssignmentStatus(ctx context.Context, cloudURL, nodeDeviceID string) (provisioning.CloudAssignmentStatus, error) {
	var out provisioning.CloudAssignmentStatus
	url := strings.TrimRight(cloudURL, "/") + "/api/v1/devices/" + nodeDeviceID + "/assignment-status"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return out, err
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, fmt.Errorf("cloud assignment status returned %d", resp.StatusCode)
	}
	return out, json.NewDecoder(resp.Body).Decode(&out)
}

func (c *CloudClient) ConsumePairingCode(ctx context.Context, cloudURL string, req provisioning.CloudPairingConsumeRequest) (provisioning.CloudPairingConsumeResponse, error) {
	var out provisioning.CloudPairingConsumeResponse
	return out, c.post(ctx, strings.TrimRight(cloudURL, "/")+"/api/v1/devices/pairing/consume", req, &out)
}

func (c *CloudClient) DownloadSnapshot(ctx context.Context, url string) (appmastersync.ApplyMasterDataCommand, error) {
	var out appmastersync.ApplyMasterDataCommand
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return out, err
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, fmt.Errorf("cloud snapshot download returned %d", resp.StatusCode)
	}
	return out, json.NewDecoder(resp.Body).Decode(&out)
}

func (c *CloudClient) post(ctx context.Context, url string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("cloud request returned %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *LicenseClient) Resolve(ctx context.Context, licenseURL string, req provisioning.LicenseResolveRequest) (provisioning.LicenseResolveResponse, error) {
	var out provisioning.LicenseResolveResponse
	body, err := json.Marshal(req)
	if err != nil {
		return out, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(licenseURL, "/")+"/api/v1/pairing-codes/resolve", bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, classifyLicenseResolveError(resp)
	}
	return out, json.NewDecoder(resp.Body).Decode(&out)
}

func classifyLicenseResolveError(resp *http.Response) error {
	code := remoteErrorCode(resp)
	switch {
	case resp.StatusCode == http.StatusBadRequest && code == "PAIRING_CODE_EXPIRED":
		return fmt.Errorf("%w: license pairing code expired", domain.ErrInvalid)
	case resp.StatusCode == http.StatusBadRequest && code == "PAIRING_CODE_INVALID":
		return fmt.Errorf("%w: license pairing code invalid", domain.ErrInvalid)
	case resp.StatusCode == http.StatusBadRequest:
		return fmt.Errorf("%w: license resolve rejected with %s", domain.ErrInvalid, code)
	case resp.StatusCode == http.StatusConflict:
		return fmt.Errorf("%w: license resolve rejected with %s", domain.ErrConflict, code)
	default:
		return fmt.Errorf("license resolve returned %d", resp.StatusCode)
	}
}

func remoteErrorCode(resp *http.Response) string {
	if code := strings.TrimSpace(resp.Header.Get("X-Error-Code")); code != "" {
		return code
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil || len(body) == 0 {
		return fmt.Sprintf("HTTP_%d", resp.StatusCode)
	}
	var parsed struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || strings.TrimSpace(parsed.Error.Code) == "" {
		return fmt.Sprintf("HTTP_%d", resp.StatusCode)
	}
	return strings.TrimSpace(parsed.Error.Code)
}
