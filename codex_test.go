package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// Test successful codex execution
func TestCodexClient_Run_Success(t *testing.T) {
	client := NewCodexClient("echo", 5*time.Second)

	ctx := context.Background()
	stream, err := client.Run(ctx, "test prompt")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stream == nil {
		t.Fatal("expected non-nil stream")
	}

	chunks := []StreamResponse{}
	for chunk := range stream {
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

// Test context cancellation
func TestCodexClient_Run_ContextCancellation(t *testing.T) {
	client := NewCodexClient("sleep", 30*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stream, err := client.Run(ctx, "10")

	if err == nil {
		t.Error("expected error due to cancelled context")
	}

	if stream != nil {
		for range stream {
		}
	}
}

// Test timeout handling
func TestCodexClient_Run_Timeout(t *testing.T) {
	client := NewCodexClient("sleep", 100*time.Millisecond)

	ctx := context.Background()
	stream, err := client.Run(ctx, "5")

	if err != nil {
		t.Fatalf("expected no error starting command, got %v", err)
	}

	hadError := false
	for chunk := range stream {
		if chunk.Type == "error" {
			hadError = true
		}
	}

	if !hadError {
		t.Error("expected timeout error in stream")
	}
}

// Test parse error handling
func TestCodexClient_Run_ParseError(t *testing.T) {
	client := NewCodexClient("echo", 5*time.Second)

	ctx := context.Background()
	stream, err := client.Run(ctx, "invalid-json{}")

	if err != nil {
		t.Fatalf("expected no error starting, got %v", err)
	}

	chunks := []StreamResponse{}
	for chunk := range stream {
		chunks = append(chunks, chunk)
	}

	if len(chunks) > 0 {
		lastChunk := chunks[len(chunks)-1]
		if lastChunk.Type == "error" && lastChunk.Error == "" {
			t.Error("error chunk should have error message")
		}
	}
}

// Test graceful shutdown
func TestCodexClient_Run_GracefulShutdown(t *testing.T) {
	client := NewCodexClient("echo", 5*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := client.Run(ctx, `{"type":"content","content":{"text":"test"}}`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	count := 0
	for range stream {
		count++
	}

	if count == 0 {
		t.Error("expected at least one chunk before shutdown")
	}
}

// Test chunk parsing with valid JSON
func TestStreamResponse_Parsing(t *testing.T) {
	jsonLine := `{"type":"content","content":{"text":"Hello"}}`

	var chunk StreamResponse
	err := json.Unmarshal([]byte(jsonLine), &chunk)

	if err != nil {
		t.Fatalf("failed to parse chunk: %v", err)
	}

	if chunk.Type != "content" {
		t.Errorf("expected type content, got %s", chunk.Type)
	}

	if chunk.Content == nil {
		t.Fatal("expected non-nil content")
	}
}
