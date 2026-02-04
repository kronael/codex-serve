package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test Anthropic handler with default model
func TestAnthropicHandler_DefaultModel(t *testing.T) {
	cfg := &Config{
		DefaultModel: "gpt-5.2-codex",
	}
	client := NewCodexClient("echo", 0)
	metrics := NewMetricsCollector()
	handler := NewAnthropicHandler(client, metrics, cfg)

	if handler.defaultModel != "gpt-5.2-codex" {
		t.Errorf("expected default model gpt-5.2-codex, got %s", handler.defaultModel)
	}
}

// Test Anthropic messages endpoint with invalid method
func TestHandleMessages_InvalidMethod(t *testing.T) {
	cfg := &Config{DefaultModel: "gpt-5.2-codex"}
	client := NewCodexClient("echo", 0)
	metrics := NewMetricsCollector()
	handler := NewAnthropicHandler(client, metrics, cfg)

	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	w := httptest.NewRecorder()

	handler.HandleMessages(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

// Test Anthropic messages endpoint with empty messages
func TestHandleMessages_EmptyMessages(t *testing.T) {
	cfg := &Config{DefaultModel: "gpt-5.2-codex"}
	client := NewCodexClient("echo", 0)
	metrics := NewMetricsCollector()
	handler := NewAnthropicHandler(client, metrics, cfg)

	reqBody := AnthropicMessageRequest{
		Model:    "codex",
		Messages: []Message{},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleMessages(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// Test Anthropic messages endpoint with invalid JSON
func TestHandleMessages_InvalidJSON(t *testing.T) {
	cfg := &Config{DefaultModel: "gpt-5.2-codex"}
	client := NewCodexClient("echo", 0)
	metrics := NewMetricsCollector()
	handler := NewAnthropicHandler(client, metrics, cfg)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handler.HandleMessages(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// Test build prompt function
func TestBuildPrompt(t *testing.T) {
	cfg := &Config{DefaultModel: "gpt-5.2-codex"}
	client := NewCodexClient("echo", 0)
	metrics := NewMetricsCollector()
	handler := NewAnthropicHandler(client, metrics, cfg)

	messages := []Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi"},
		{Role: "user", Content: "How are you?"},
	}

	prompt := handler.buildPrompt(messages)
	expected := "user: Hello\n\nassistant: Hi\n\nuser: How are you?"

	if prompt != expected {
		t.Errorf("prompt mismatch\nexpected: %s\ngot: %s", expected, prompt)
	}
}
