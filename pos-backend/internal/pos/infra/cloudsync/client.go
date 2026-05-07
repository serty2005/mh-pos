package cloudsync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/syncsender"
)

type Client struct {
	endpoint   string
	httpClient *http.Client
}

type Option func(*Client)

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

func NewClient(endpoint string, opts ...Option) *Client {
	c := &Client{
		endpoint: strings.TrimSpace(endpoint),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) Send(ctx context.Context, msg domain.OutboxMessage) error {
	if c.endpoint == "" {
		return fmt.Errorf("cloud sync endpoint is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewBufferString(msg.PayloadJSON))
	if err != nil {
		return syncsender.NonRetryableError{Reason: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := validateAck(body, msg); err != nil {
			return syncsender.NonRetryableError{Reason: err.Error()}
		}
		return nil
	}
	reason := fmt.Sprintf("cloud receiver returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return fmt.Errorf("%s", reason)
	}
	return syncsender.NonRetryableError{Reason: reason}
}

func validateAck(body []byte, msg domain.OutboxMessage) error {
	var envelope struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal([]byte(msg.PayloadJSON), &envelope); err != nil {
		return fmt.Errorf("outbox payload is not a valid SyncEnvelope: %w", err)
	}
	var ack struct {
		Status  string `json:"status"`
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(body, &ack); err != nil {
		return fmt.Errorf("cloud ack is not valid JSON: %w", err)
	}
	if ack.Status != "accepted" {
		return fmt.Errorf("cloud ack status %q is not accepted", ack.Status)
	}
	if strings.TrimSpace(ack.EventID) != "" && ack.EventID != envelope.EventID {
		return fmt.Errorf("cloud ack event_id %q does not match outbox event_id %q", ack.EventID, envelope.EventID)
	}
	return nil
}
