package licensegate_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"mh-pos-platform/licensegate"
)

type fixedGate struct{ err error }

func (g fixedGate) Require(context.Context, string) error { return g.err }
func (g fixedGate) Current(context.Context) (licensegate.Snapshot, error) {
	return licensegate.Snapshot{}, g.err
}

func TestMiddlewareFailsClosedWithSafeErrors(t *testing.T) {
	for _, test := range []struct {
		err    error
		status int
		code   string
	}{
		{licensegate.ErrDenied, http.StatusForbidden, "LICENSE_ENTITLEMENT_REQUIRED"},
		{licensegate.ErrUnavailable, http.StatusServiceUnavailable, "LICENSE_AUTHORITY_UNAVAILABLE"},
	} {
		handler := licensegate.Middleware(fixedGate{err: test.err}, func(*http.Request) string { return licensegate.TableMode })(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("gated handler called") }))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tables", nil))
		if rec.Code != test.status || rec.Header().Get("X-Error-Code") != test.code {
			t.Fatalf("status=%d code=%s body=%s", rec.Code, rec.Header().Get("X-Error-Code"), rec.Body.String())
		}
		if errors.Is(test.err, licensegate.ErrDenied) && rec.Body.String() == "" {
			t.Fatal("safe error body missing")
		}
	}
}
