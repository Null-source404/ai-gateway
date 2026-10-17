// wires all routes together
package server

import (
	"net/http"

	"github.com/john/ai-gateway/internal/provider"
)

func NewServer(p provider.Provider, apiKey string, ratePerMin int) *http.ServeMux {
	h := Handler{Provider: p}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", h.Healthz)

	var chatHandler http.Handler = http.HandlerFunc(h.Chat)
	if apiKey != "" {
		chatHandler = RequireGatewayKey(apiKey)(chatHandler)
	}
	chatHandler = RateLimit(chatHandler, ratePerMin)
	chatHandler = LogRequests(chatHandler)
	mux.Handle("POST /v1/chat", chatHandler)

	return mux
}
