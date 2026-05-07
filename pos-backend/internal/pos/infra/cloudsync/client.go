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

func (c *Client) SendBatch(ctx context.Context, messages []domain.OutboxMessage) ([]syncsender.BatchSendResult, error) {
	if c.endpoint == "" {
		return nil, fmt.Errorf("cloud sync endpoint is not configured")
	}
	if len(messages) == 0 {
		return nil, nil
	}
	items := make([]json.RawMessage, 0, len(messages))
	for _, msg := range messages {
		items = append(items, json.RawMessage(msg.PayloadJSON))
	}
	body, err := json.Marshal(map[string]any{
		"items": items,
	})
	if err != nil {
		return nil, syncsender.NonRetryableError{Reason: fmt.Sprintf("cloud batch request is not valid JSON: %v", err)}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, batchEndpoint(c.endpoint), bytes.NewReader(body))
	if err != nil {
		return nil, syncsender.NonRetryableError{Reason: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rawBody, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return parseBatchAck(rawBody, messages)
	}
	reason := fmt.Sprintf("cloud receiver returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(rawBody)))
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, fmt.Errorf("%s", reason)
	}
	out := make([]syncsender.BatchSendResult, 0, len(messages))
	for _, msg := range messages {
		out = append(out, syncsender.BatchSendResult{
			OutboxID: msg.ID,
			Status:   syncsender.BatchSendRejected,
			Reason:   reason,
		})
	}
	return out, nil
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

func parseBatchAck(body []byte, messages []domain.OutboxMessage) ([]syncsender.BatchSendResult, error) {
	var ack struct {
		Status string `json:"status"`
		Items  []struct {
			Index     int    `json:"index"`
			Status    string `json:"status"`
			ErrorCode string `json:"error_code"`
			Error     string `json:"error"`
			Ack       *struct {
				Status  string `json:"status"`
				EventID string `json:"event_id"`
			} `json:"ack"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &ack); err != nil {
		return nil, fmt.Errorf("cloud batch ack is not valid JSON: %w", err)
	}
	out := make([]syncsender.BatchSendResult, 0, len(messages))
	for _, item := range ack.Items {
		if item.Index < 0 || item.Index >= len(messages) {
			continue
		}
		msg := messages[item.Index]
		result := syncsender.BatchSendResult{
			OutboxID: msg.ID,
		}
		switch strings.ToLower(strings.TrimSpace(item.Status)) {
		case "accepted":
			if item.Ack == nil || item.Ack.Status != "accepted" {
				result.Status = syncsender.BatchSendRetryable
				result.Reason = "cloud batch ack item has invalid accepted payload"
			} else if err := validateAckEventID(item.Ack.EventID, msg); err != nil {
				result.Status = syncsender.BatchSendRejected
				result.Reason = err.Error()
			} else {
				result.Status = syncsender.BatchSendAccepted
			}
		case "rejected":
			result.Status = syncsender.BatchSendRejected
			result.Reason = strings.TrimSpace(item.Error)
			if result.Reason == "" {
				result.Reason = "cloud batch item rejected"
			}
		case "retryable":
			result.Status = syncsender.BatchSendRetryable
			result.Reason = strings.TrimSpace(item.Error)
			if result.Reason == "" {
				result.Reason = "cloud batch item failed with retryable status"
			}
		default:
			result.Status = syncsender.BatchSendRetryable
			result.Reason = "cloud batch item status is unknown"
		}
		out = append(out, result)
	}
	return out, nil
}

func validateAckEventID(eventID string, msg domain.OutboxMessage) error {
	var envelope struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal([]byte(msg.PayloadJSON), &envelope); err != nil {
		return fmt.Errorf("outbox payload is not a valid SyncEnvelope: %w", err)
	}
	if strings.TrimSpace(eventID) != "" && eventID != envelope.EventID {
		return fmt.Errorf("cloud ack event_id %q does not match outbox event_id %q", eventID, envelope.EventID)
	}
	return nil
}

func batchEndpoint(singleEndpoint string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(singleEndpoint), "/")
	if strings.HasSuffix(trimmed, "/batch") {
		return trimmed
	}
	return trimmed + "/batch"
}
