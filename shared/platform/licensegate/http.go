package licensegate

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
)

// Middleware блокирует только маршруты, которым resolver назначил module ID.
func Middleware(gate Gate, resolver func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			moduleID := resolver(r)
			if moduleID == "" || gate == nil {
				next.ServeHTTP(w, r)
				return
			}
			if err := gate.Require(r.Context(), moduleID); err != nil {
				correlationID := r.Header.Get("X-Request-ID")
				if correlationID == "" {
					var raw [12]byte
					_, _ = rand.Read(raw[:])
					correlationID = hex.EncodeToString(raw[:])
				}
				status, code, key := http.StatusForbidden, "LICENSE_ENTITLEMENT_REQUIRED", "errors.license.entitlementRequired"
				if errors.Is(err, ErrUnavailable) {
					status, code, key = http.StatusServiceUnavailable, "LICENSE_AUTHORITY_UNAVAILABLE", "errors.license.unavailable"
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Error-Code", code)
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message_key": key, "correlation_id": correlationID}})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func StatusHandler(gate Gate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshot, err := gate.Current(r.Context())
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "LICENSE_AUTHORITY_UNAVAILABLE", "message_key": "errors.license.unavailable"}})
			return
		}
		_ = json.NewEncoder(w).Encode(snapshot)
	}
}
