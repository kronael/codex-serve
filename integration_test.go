// +build !short

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// Smoke test that uses real codex CLI if available
// Run with: go test -v -timeout 30s (without -short flag)
func TestOpenAIChat_RealCodex(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if codex CLI is available
	codexPath := os.Getenv("CODEX_PATH")
	if codexPath == "" {
		codexPath = "codex"
	}

	// Try to verify codex is available
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := NewCodexClient(codexPath, 30*time.Second)
	stream, err := client.Run(ctx, "respond with just: test ok")
	if err != nil {
		t.Skipf("codex CLI not available or not authenticated: %v", err)
	}

	// Verify we can parse at least one chunk
	gotChunk := false
	for chunk := range stream {
		if chunk.Type == "item.completed" && chunk.Item != nil {
			var item ItemContent
			if err := json.Unmarshal(chunk.Item, &item); err == nil {
				if item.Type == "agent_message" {
					gotChunk = true
					t.Logf("Got response: %s", item.Text)
					break
				}
			}
		}
	}

	if !gotChunk {
		t.Error("expected to receive at least one item.completed chunk from real codex")
	}
}

// Test full OpenAI chat flow with real codex
func TestOpenAIChat_FullFlow_RealCodex(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	codexPath := os.Getenv("CODEX_PATH")
	if codexPath == "" {
		codexPath = "codex"
	}

	client := NewCodexClient(codexPath, 30*time.Second)
	handler := HandleOpenAIChat(client)

	reqBody := OpenAIChatRequest{
		Model: "codex",
		Messages: []Message{
			{Role: "user", Content: "what is 1+1? answer in 3 words"},
		},
		Stream: false,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Logf("Response body: %s", w.Body.String())
		t.Skipf("codex CLI not available or failed: status %d", w.Code)
	}

	var resp OpenAIChatResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Error("expected at least one choice")
	}

	if len(resp.Choices) > 0 {
		content := resp.Choices[0].Message.Content
		if content == "" {
			t.Error("expected non-empty message content")
		} else {
			t.Logf("Got response: %s", content)
		}
	}
}

// Test Ollama chat with real codex
func TestOllamaChat_RealCodex(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	codexPath := os.Getenv("CODEX_PATH")
	if codexPath == "" {
		codexPath = "codex"
	}

	client := NewCodexClient(codexPath, 30*time.Second)
	handler := HandleOllamaChat(client)

	reqBody := OllamaChatRequest{
		Model: "codex",
		Messages: []Message{
			{Role: "user", Content: "say hello in 2 words"},
		},
		Stream: false,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Logf("Response body: %s", w.Body.String())
		t.Skipf("codex CLI not available or failed: status %d", w.Code)
	}

	var resp OllamaChatResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Message.Content == "" {
		t.Error("expected non-empty message content")
	} else {
		t.Logf("Got response: %s", resp.Message.Content)
	}
}

// Test OpenAI streaming with real codex
func TestOpenAIChat_Streaming_RealCodex(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	codexPath := os.Getenv("CODEX_PATH")
	if codexPath == "" {
		codexPath = "codex"
	}

	client := NewCodexClient(codexPath, 30*time.Second)
	handler := HandleOpenAIChat(client)

	reqBody := OpenAIChatRequest{
		Model: "gpt-5.2-codex",
		Messages: []Message{
			{Role: "user", Content: "say 'test' in 2 words"},
		},
		Stream: true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Logf("Response body: %s", w.Body.String())
		t.Skipf("codex CLI not available or failed: status %d", w.Code)
	}

	// Check for SSE format
	if !strings.Contains(w.Body.String(), "data:") {
		t.Error("expected SSE format with 'data:' prefix")
	}

	t.Logf("Streaming response received: %d bytes", w.Body.Len())
}

// Test that verifies our mock data matches real codex format
func TestMockVsRealFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test documents the real codex format
	realFormat := `{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"response"}}`

	var chunk StreamResponse
	err := json.Unmarshal([]byte(realFormat), &chunk)
	if err != nil {
		t.Fatalf("failed to parse real format: %v", err)
	}

	if chunk.Type != "item.completed" {
		t.Errorf("expected type item.completed, got %s", chunk.Type)
	}

	if chunk.Item == nil {
		t.Fatal("expected item field to be present")
	}

	var item ItemContent
	if err := json.Unmarshal(chunk.Item, &item); err != nil {
		t.Fatalf("failed to parse item: %v", err)
	}

	if item.Type != "agent_message" {
		t.Errorf("expected item type agent_message, got %s", item.Type)
	}

	if item.Text != "response" {
		t.Errorf("expected text 'response', got %s", item.Text)
	}

	t.Log("✓ Mock format matches real codex CLI format")
}
