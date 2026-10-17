// Loads the config and starts the server
package main

import (
	"log"
	"net/http"

	"github.com/john/ai-gateway/internal/config"
	"github.com/john/ai-gateway/internal/provider"
	"github.com/john/ai-gateway/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	providers := []provider.Provider{provider.MockProvider{}}
	switch cfg.Provider {
	case "openai", "ollama":
		providers = []provider.Provider{
			provider.NewOpenAICompatibleProvider(cfg.OpenAIBaseURL, cfg.OpenAIAPIKey, cfg.Model),
			provider.MockProvider{},
		}
	case "mock":
		providers = []provider.Provider{provider.MockProvider{}}
	default:
		providers = []provider.Provider{provider.MockProvider{}}
	}

	selectedProvider := provider.NewRouter(providers)
	mux := server.NewServer(selectedProvider, cfg.APIKey, cfg.RatePerMin)
	log.Printf("gateway listening on %s using provider %s", cfg.Addr, selectedProvider.Name())

	if err := http.ListenAndServe(cfg.Addr, mux); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
