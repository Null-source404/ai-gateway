package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAICompatibleProvider struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

func NewOpenAICompatibleProvider(baseURL, apiKey, model string) OpenAICompatibleProvider {
	if model == "" {
		model = "default"
	}
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1"
	}
	return OpenAICompatibleProvider{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		Model:   model,
		Client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (p OpenAICompatibleProvider) Name() string { return "openai-compatible" }

func (p OpenAICompatibleProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	if req.Model == "" {
		req.Model = p.Model
	}

	payload := map[string]any{
		"model":    req.Model,
		"messages": toOpenAIMessages(req.Messages),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return ChatResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	res, err := p.Client.Do(httpReq)
	if err != nil {
		return ChatResponse{}, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(res.Body)
		return ChatResponse{}, fmt.Errorf("provider returned %d: %s", res.StatusCode, string(bodyBytes))
	}

	var openAIResp struct {
		ID      string `json:"id"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(res.Body).Decode(&openAIResp); err != nil {
		return ChatResponse{}, err
	}
	if len(openAIResp.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("provider response had no choices")
	}

	return ChatResponse{
		ID:       openAIResp.ID,
		Provider: p.Name(),
		Content:  openAIResp.Choices[0].Message.Content,
		Usage: Usage{
			InputTokens:  openAIResp.Usage.PromptTokens,
			OutputTokens: openAIResp.Usage.CompletionTokens,
		},
	}, nil
}

func toOpenAIMessages(messages []ChatMessage) []map[string]string {
	out := make([]map[string]string, 0, len(messages))
	for _, msg := range messages {
		out = append(out, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}
	return out
}
