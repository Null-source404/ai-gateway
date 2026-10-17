// Testing using the mock model(dummy)
package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/annoying-octopus/ai-gateway/internal/provider"
)

func TestNewServer_HealthzAndChat(t *testing.T) {
	mux := NewServer(provider.MockProvider{}, "dev-gateway-key-change-me", 60)

	t.Run("healthz returns ok", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		res := httptest.NewRecorder()

		mux.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", res.Code)
		}
		if !strings.Contains(res.Body.String(), "status") {
			t.Fatalf("expected healthz response body, got %q", res.Body.String())
		}
	})

	t.Run("chat requires valid gateway key", func(t *testing.T) {
		body := `{"model":"default","messages":[{"role":"user","content":"hello there"}]}`
		req := httptest.NewRequest(http.MethodPost, "/v1/chat", strings.NewReader(body))
		res := httptest.NewRecorder()

		mux.ServeHTTP(res, req)

		if res.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", res.Code)
		}
	})

	t.Run("chat forwards to the mock provider with valid key", func(t *testing.T) {
		body := `{"model":"default","messages":[{"role":"user","content":"hello there"}]}`
		req := httptest.NewRequest(http.MethodPost, "/v1/chat", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer dev-gateway-key-change-me")
		res := httptest.NewRecorder()

		mux.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", res.Code)
		}
		if !strings.Contains(res.Body.String(), "Mock reply") {
			t.Fatalf("expected mock provider response, got %q", res.Body.String())
		}
	})
}
