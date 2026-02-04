package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// CodexClient manages codex CLI process execution
type CodexClient struct {
	path    string
	timeout time.Duration
	mu      sync.Mutex
}

// NewCodexClient creates a new CodexClient
func NewCodexClient(path string, timeout time.Duration) *CodexClient {
	return &CodexClient{
		path:    path,
		timeout: timeout,
	}
}

// StreamResponse represents a single streaming response chunk
type StreamResponse struct {
	Type    string          `json:"type"`
	Item    json.RawMessage `json:"item,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// ItemContent represents the item content from codex
type ItemContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Run executes codex with the given prompt and streams responses
func (c *CodexClient) Run(ctx context.Context, prompt string) (<-chan StreamResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cmd := exec.CommandContext(ctx, c.path, "exec", "--json", prompt)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start codex: %w", err)
	}

	// Setup streaming channel
	ch := make(chan StreamResponse, 10)

	go func() {
		defer close(ch)
		defer stdout.Close()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			var resp StreamResponse
			if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
				ch <- StreamResponse{
					Type:  "error",
					Error: fmt.Sprintf("failed to parse response: %v", err),
				}
				continue
			}
			ch <- resp
		}

		if err := scanner.Err(); err != nil && err != io.EOF {
			ch <- StreamResponse{
				Type:  "error",
				Error: fmt.Sprintf("stream error: %v", err),
			}
		}

		// Wait for process to finish and handle cleanup
		if err := cmd.Wait(); err != nil {
			// Check if context was cancelled
			if ctx.Err() == context.Canceled {
				// SIGTERM was sent, process should have terminated gracefully
				return
			}
			ch <- StreamResponse{
				Type:  "error",
				Error: fmt.Sprintf("codex failed: %v", err),
			}
		}
	}()

	// Handle context cancellation with graceful shutdown
	go func() {
		<-ctx.Done()
		if cmd.Process != nil {
			// Send SIGTERM for graceful shutdown
			cmd.Process.Signal(syscall.SIGTERM)

			// Wait 5 seconds for graceful shutdown
			timer := time.NewTimer(5 * time.Second)
			done := make(chan struct{})

			go func() {
				cmd.Wait()
				close(done)
			}()

			select {
			case <-done:
				timer.Stop()
			case <-timer.C:
				// Timeout, force kill
				cmd.Process.Kill()
			}
		}
	}()

	return ch, nil
}
