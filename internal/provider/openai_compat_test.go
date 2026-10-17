package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAICompatibleProviderChat(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-upstream-key" {
			t.Errorf("Authorization = %q, want bearer test key", got)
		}

		var request struct {
			Model    string        `json:"model"`
			Messages []ChatMessage `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if request.Model != "local-model" {
			t.Errorf("model = %q, want local-model", request.Model)
		}
		if len(request.Messages) != 1 || request.Messages[0].Content != "hello from gateway" {
			t.Errorf("messages = %#v, want one user message", request.Messages)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl-test","choices":[{"message":{"content":"Hello from local API"}}],"usage":{"prompt_tokens":4,"completion_tokens":5}}`))
	}))
	defer upstream.Close()

	client := NewOpenAICompatibleProvider(upstream.URL+"/v1", "test-upstream-key", "local-model")
	response, err := client.Chat(context.Background(), ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hello from gateway"}},
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if response.ID != "chatcmpl-test" || response.Content != "Hello from local API" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response.Usage.InputTokens != 4 || response.Usage.OutputTokens != 5 {
		t.Fatalf("unexpected usage: %#v", response.Usage)
	}
}
