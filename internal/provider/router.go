package provider

import (
	"context"
	"fmt"
)

type Router struct {
	Providers []Provider
}

func NewRouter(providers []Provider) Router {
	if len(providers) == 0 {
		return Router{Providers: []Provider{MockProvider{}}}
	}
	return Router{Providers: providers}
}

func (r Router) Name() string {
	if len(r.Providers) == 0 {
		return "mock"
	}
	return r.Providers[0].Name()
}

func (r Router) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	var lastErr error
	for _, p := range r.Providers {
		resp, err := p.Chat(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return ChatResponse{}, fmt.Errorf("all providers failed: %w", lastErr)
	}
	return ChatResponse{}, fmt.Errorf("no providers configured")
}
