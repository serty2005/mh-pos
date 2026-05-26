package clickhouse

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cloud-backend/internal/olap/app"
)

type Config struct {
	URL      string
	Database string
	Username string
	Password string
}

type Repository struct {
	endpoint string
	database string
	username string
	password string
	client   *http.Client
}

func NewRepository(config Config) *Repository {
	database := strings.TrimSpace(config.Database)
	if database == "" {
		database = "mh_pos_cloud"
	}
	return &Repository{
		endpoint: strings.TrimRight(strings.TrimSpace(config.URL), "/"),
		database: database,
		username: strings.TrimSpace(config.Username),
		password: config.Password,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (r *Repository) Migrate(ctx context.Context) error {
	if r == nil || r.endpoint == "" {
		return app.ErrOLAPUnavailable
	}
	if err := r.exec(ctx, fmt.Sprintf(`CREATE DATABASE IF NOT EXISTS %s`, ident(r.database))); err != nil {
		return err
	}
	return r.exec(ctx, fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s.raw_business_events (
  event_id String,
  tenant_id String,
  restaurant_id String,
  device_id String,
  employee_id String,
  event_type LowCardinality(String),
  occurred_at DateTime64(3, 'UTC'),
  cloud_received_at DateTime64(3, 'UTC'),
  raw_payload_sha256_hex String,
  payload String,
  exported_at DateTime64(3, 'UTC')
) ENGINE = ReplacingMergeTree(exported_at)
PARTITION BY toYYYYMM(occurred_at)
ORDER BY (tenant_id, event_type, event_id)`, ident(r.database)))
}

func (r *Repository) InsertRawBusinessEvents(ctx context.Context, events []app.InboxEvent, exportedAt time.Time) error {
	if len(events) == 0 {
		return nil
	}
	var body bytes.Buffer
	body.WriteString(fmt.Sprintf("INSERT INTO %s.raw_business_events FORMAT JSONEachRow\n", ident(r.database)))
	enc := json.NewEncoder(&body)
	for _, event := range events {
		row := map[string]string{
			"event_id":               event.EventID,
			"tenant_id":              event.TenantID,
			"restaurant_id":          event.RestaurantID,
			"device_id":              event.DeviceID,
			"employee_id":            event.EmployeeID,
			"event_type":             event.EventType,
			"occurred_at":            chTime(event.OccurredAt),
			"cloud_received_at":      chTime(event.CloudReceivedAt),
			"raw_payload_sha256_hex": event.RawPayloadSHA256Hex,
			"payload":                string(event.RawPayload),
			"exported_at":            chTime(exportedAt),
		}
		if err := enc.Encode(row); err != nil {
			return err
		}
	}
	return r.post(ctx, body.String())
}

func (r *Repository) ListRawBusinessEvents(ctx context.Context, filter app.RawBusinessEventFilter) ([]app.RawBusinessEvent, error) {
	query := strings.Builder{}
	query.WriteString("SELECT event_id,tenant_id,restaurant_id,device_id,employee_id,event_type,occurred_at,cloud_received_at,raw_payload_sha256_hex FROM ")
	query.WriteString(ident(r.database))
	query.WriteString(".raw_business_events FINAL WHERE 1=1")
	if filter.RestaurantID != "" {
		query.WriteString(" AND restaurant_id = ")
		query.WriteString(quote(filter.RestaurantID))
	}
	if filter.EventType != "" {
		query.WriteString(" AND event_type = ")
		query.WriteString(quote(filter.EventType))
	}
	if filter.OccurredFrom != nil {
		query.WriteString(" AND occurred_at >= parseDateTime64BestEffort(")
		query.WriteString(quote(filter.OccurredFrom.UTC().Format(time.RFC3339Nano)))
		query.WriteString(", 3, 'UTC')")
	}
	if filter.OccurredTo != nil {
		query.WriteString(" AND occurred_at <= parseDateTime64BestEffort(")
		query.WriteString(quote(filter.OccurredTo.UTC().Format(time.RFC3339Nano)))
		query.WriteString(", 3, 'UTC')")
	}
	query.WriteString(fmt.Sprintf(" ORDER BY occurred_at DESC, event_id DESC LIMIT %d OFFSET %d FORMAT JSONEachRow", filter.Limit, filter.Offset))

	respBody, err := r.query(ctx, query.String())
	if err != nil {
		return nil, err
	}
	defer respBody.Close()

	rows := make([]app.RawBusinessEvent, 0, filter.Limit)
	scanner := bufio.NewScanner(respBody)
	for scanner.Scan() {
		var row struct {
			EventID             string `json:"event_id"`
			TenantID            string `json:"tenant_id"`
			RestaurantID        string `json:"restaurant_id"`
			DeviceID            string `json:"device_id"`
			EmployeeID          string `json:"employee_id"`
			EventType           string `json:"event_type"`
			OccurredAt          string `json:"occurred_at"`
			CloudReceivedAt     string `json:"cloud_received_at"`
			RawPayloadSHA256Hex string `json:"raw_payload_sha256_hex"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return nil, err
		}
		occurredAt, err := parseCHTime(row.OccurredAt)
		if err != nil {
			return nil, err
		}
		receivedAt, err := parseCHTime(row.CloudReceivedAt)
		if err != nil {
			return nil, err
		}
		rows = append(rows, app.RawBusinessEvent{
			EventID:             row.EventID,
			TenantID:            row.TenantID,
			RestaurantID:        row.RestaurantID,
			DeviceID:            row.DeviceID,
			EmployeeID:          row.EmployeeID,
			EventType:           row.EventType,
			OccurredAt:          occurredAt,
			CloudReceivedAt:     receivedAt,
			RawPayloadSHA256Hex: row.RawPayloadSHA256Hex,
		})
	}
	return rows, scanner.Err()
}

func (r *Repository) exec(ctx context.Context, query string) error {
	return r.post(ctx, query)
}

func (r *Repository) query(ctx context.Context, query string) (io.ReadCloser, error) {
	return r.do(ctx, query)
}

func (r *Repository) post(ctx context.Context, query string) error {
	body, err := r.do(ctx, query)
	if err != nil {
		return err
	}
	defer body.Close()
	_, _ = io.Copy(io.Discard, body)
	return nil
}

func (r *Repository) do(ctx context.Context, query string) (io.ReadCloser, error) {
	if r == nil || r.endpoint == "" {
		return nil, app.ErrOLAPUnavailable
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, strings.NewReader(query))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	if r.username != "" {
		req.SetBasicAuth(r.username, r.password)
	}
	q := req.URL.Query()
	q.Set("database", r.database)
	req.URL.RawQuery = q.Encode()
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("clickhouse http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return resp.Body, nil
}

func ident(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "default"
	}
	var b strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "default"
	}
	return b.String()
}

func quote(value string) string {
	return "'" + strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `'`, `\'`) + "'"
}

func chTime(value time.Time) string {
	return value.UTC().Format("2006-01-02 15:04:05.000")
}

func parseCHTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
	} {
		if t, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid ClickHouse timestamp %q", value)
}
