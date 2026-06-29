package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestWithOptionalPOSUIServesSPAAndKeepsAPI(t *testing.T) {
	uiDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(uiDir, "index.html"), []byte("pos ui"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(uiDir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "assets", "app.js"), []byte("asset"), 0o644); err != nil {
		t.Fatal(err)
	}

	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("api"))
	})
	handler := withOptionalPOSUI(apiHandler, uiDir)

	for _, tc := range []struct {
		path string
		want string
	}{
		{path: "/", want: "pos ui"},
		{path: "/orders/current", want: "pos ui"},
		{path: "/assets/app.js", want: "asset"},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if got := res.Body.String(); got != tc.want {
			t.Fatalf("%s: got %q, want %q", tc.path, got, tc.want)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/menu/items", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusTeapot || res.Body.String() != "api" {
		t.Fatalf("API route was not delegated: code=%d body=%q", res.Code, res.Body.String())
	}
}
