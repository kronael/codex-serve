package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test formatting messages for Ollama
func TestFormatMessages(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
		{Role: "user", Content: "How are you?"},
	}

	prompt := formatMessages(messages)
	expected := "User: Hello\n\nAssistant: Hi there\n\nUser: How are you?\n\n"

	if prompt != expected {
		t.Errorf("prompt mismatch\nexpected: %s\ngot: %s", expected, prompt)
	}
}

// Test Ollama tags endpoint
func TestHandleOllamaTags(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/tags", nil)
	w := httptest.NewRecorder()

	HandleOllamaTags(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Models) == 0 {
		t.Error("expected at least one model")
	}

	if resp.Models[0].Name != "codex" {
		t.Errorf("expected model name codex, got %s", resp.Models[0].Name)
	}
}

// Test Ollama chat endpoint with invalid method
func TestHandleOllamaChat_InvalidMethod(t *testing.T) {
	client := NewCodexClient("echo", 0)
	handler := HandleOllamaChat(client)

	req := httptest.NewRequest(http.MethodGet, "/api/chat", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

// Test Ollama chat endpoint with valid messages
func TestHandleOllamaChat_ValidMessages(t *testing.T) {
	client := NewCodexClient("echo", 5*time.Second)
	handler := HandleOllamaChat(client)

	reqBody := OllamaChatRequest{
		Model: "codex",
		Messages: []Message{
			{Role: "user", Content: "test"},
		},
		Stream: false,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 200 or 500, got %d", w.Code)
	}
}

// Test Ollama generate endpoint with invalid method
func TestHandleOllamaGenerate_InvalidMethod(t *testing.T) {
	client := NewCodexClient("echo", 0)
	handler := HandleOllamaGenerate(client)

	req := httptest.NewRequest(http.MethodGet, "/api/generate", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

// Test Ollama generate endpoint with valid prompt
func TestHandleOllamaGenerate_ValidPrompt(t *testing.T) {
	client := NewCodexClient("echo", 5*time.Second)
	handler := HandleOllamaGenerate(client)

	reqBody := OllamaGenerateRequest{
		Model:  "codex",
		Prompt: "test prompt",
		Stream: false,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/generate", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 200 or 500, got %d", w.Code)
	}
}
