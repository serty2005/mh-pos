package httpx

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"pos-backend/internal/pos/domain"
)

func TestClassifyErrorReturnsSafeStableContract(t *testing.T) {
	status, body := ClassifyError(fmt.Errorf("%w: permission pos.sync.retry_failed is required", domain.ErrForbidden))
	if status != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", status)
	}
	if body.Code != "PERMISSION_DENIED" || body.MessageKey != "errors.permission.denied" {
		t.Fatalf("unexpected error body: %+v", body)
	}
	if strings.Contains(fmt.Sprint(body), "pos.sync.retry_failed") {
		t.Fatalf("expected permission id not to leak into user response: %+v", body)
	}
}

func TestClassifyErrorMapsSessionAndRateLimit(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		wantKey    string
	}{
		{
			name:       "revoked session",
			err:        fmt.Errorf("%w: session is not active", domain.ErrForbidden),
			wantStatus: http.StatusUnauthorized,
			wantCode:   "SESSION_REVOKED",
			wantKey:    "errors.session.revoked",
		},
		{
			name:       "pin rate limit",
			err:        fmt.Errorf("%w: retry after lockout window", domain.ErrTooManyRequests),
			wantStatus: http.StatusTooManyRequests,
			wantCode:   "RATE_LIMITED",
			wantKey:    "errors.rateLimit",
		},
		{
			name:       "duplicate pin",
			err:        fmt.Errorf("%w: pin must uniquely identify one active employee", domain.ErrConflict),
			wantStatus: http.StatusConflict,
			wantCode:   "DUPLICATE_PIN",
			wantKey:    "errors.conflict.duplicatePin",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := ClassifyError(tc.err)
			if status != tc.wantStatus || body.Code != tc.wantCode || body.MessageKey != tc.wantKey {
				t.Fatalf("got status=%d body=%+v", status, body)
			}
		})
	}
}

func TestClassifyErrorHidesInternalError(t *testing.T) {
	status, body := ClassifyError(errors.New("constraint failed: FOREIGN KEY constraint failed (787)"))
	if status != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", status)
	}
	if body.Code != "INTERNAL_ERROR" || body.MessageKey != "errors.server" {
		t.Fatalf("unexpected internal error body: %+v", body)
	}
	if strings.Contains(fmt.Sprint(body), "FOREIGN KEY") {
		t.Fatalf("expected SQL details not to be exposed: %+v", body)
	}
}
