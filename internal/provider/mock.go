// Just a dummy provider for testing purposes ====> Fake AI
package provider

import (
	"context"
	"fmt"
)

type MockProvider struct{}

func (p MockProvider) Name() string { return "mock" }

func (p MockProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	userText := ""
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			userText = msg.Content
			break
		}
	}
	if userText == "" {
		userText = "hello"
	}

	responseText := fmt.Sprintf("Mock reply: %s", userText)
	return ChatResponse{
		ID:       "mock-1",
		Provider: p.Name(),
		Content:  responseText,
		Usage: Usage{
			InputTokens:  3,
			OutputTokens: 9,
		},
	}, nil
}
