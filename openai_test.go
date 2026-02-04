package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test OpenAI models endpoint
func TestHandleOpenAIModels(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()

	HandleOpenAIModels(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 4 {
		t.Errorf("expected 4 models, got %d", len(resp.Data))
	}

	if resp.Data[0].ID != "gpt-5.2-codex" {
		t.Errorf("expected model id gpt-5.2-codex, got %s", resp.Data[0].ID)
	}
}

// Test OpenAI models endpoint with invalid method
func TestHandleOpenAIModels_InvalidMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/models", nil)
	w := httptest.NewRecorder()

	HandleOpenAIModels(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

// Test OpenAI chat endpoint with invalid method
func TestHandleOpenAIChat_InvalidMethod(t *testing.T) {
	client := NewCodexClient("echo", 0)
	handler := HandleOpenAIChat(client)

	req := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

// Test OpenAI chat endpoint with valid messages
func TestHandleOpenAIChat_ValidMessages(t *testing.T) {
	client := NewCodexClient("echo", 5*time.Second)
	handler := HandleOpenAIChat(client)

	reqBody := OpenAIChatRequest{
		Model: "codex",
		Messages: []Message{
			{Role: "user", Content: "test"},
		},
		Stream: false,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 200 or 500, got %d", w.Code)
	}
}

// Test OpenAI chat endpoint with invalid JSON
func TestHandleOpenAIChat_InvalidJSON(t *testing.T) {
	client := NewCodexClient("echo", 0)
	handler := HandleOpenAIChat(client)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}
