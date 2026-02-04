package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// AnthropicHandler handles Anthropic Messages API requests
type AnthropicHandler struct {
	client       *CodexClient
	metrics      *MetricsCollector
	defaultModel string
}

// NewAnthropicHandler creates a new Anthropic API handler
func NewAnthropicHandler(client *CodexClient, metrics *MetricsCollector, cfg *Config) *AnthropicHandler {
	return &AnthropicHandler{
		client:       client,
		metrics:      metrics,
		defaultModel: cfg.DefaultModel,
	}
}

// HandleMessages handles POST /v1/messages
func (h *AnthropicHandler) HandleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, &APIError{
			Code:    ErrInvalidRequest,
			Message: "method not allowed",
			Status:  http.StatusMethodNotAllowed,
		})
		return
	}

	var req AnthropicMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, &APIError{
			Code:    ErrInvalidRequest,
			Message: fmt.Sprintf("invalid request body: %v", err),
			Status:  http.StatusBadRequest,
		})
		return
	}

	if len(req.Messages) == 0 {
		WriteError(w, &APIError{
			Code:    ErrInvalidRequest,
			Message: "messages cannot be empty",
			Status:  http.StatusBadRequest,
		})
		return
	}

	if req.Stream {
		h.handleStream(w, r, &req)
	} else {
		h.handleNonStream(w, r, &req)
	}
}

// handleStream handles streaming responses using Anthropic SSE format
func (h *AnthropicHandler) handleStream(w http.ResponseWriter, r *http.Request, req *AnthropicMessageRequest) {
	ctx := r.Context()
	messageID := fmt.Sprintf("msg_%s", uuid.New().String())
	model := req.Model
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteError(w, &APIError{
			Code:    ErrCodexFailed,
			Message: "streaming not supported",
			Status:  http.StatusInternalServerError,
		})
		return
	}

	// Send message_start event
	h.writeSSE(w, flusher, "message_start", map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":            messageID,
			"type":          "message",
			"role":          "assistant",
			"model":         model,
			"content":       []interface{}{},
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]int{
				"input_tokens":  0,
				"output_tokens": 0,
			},
		},
	})

	// Send content_block_start event
	h.writeSSE(w, flusher, "content_block_start", map[string]interface{}{
		"type":  "content_block_start",
		"index": 0,
		"content_block": map[string]string{
			"type": "text",
			"text": "",
		},
	})

	// Build prompt from messages
	prompt := h.buildPrompt(req.Messages)

	// Execute codex
	stream, err := h.client.Run(ctx, prompt)
	if err != nil {
		h.writeSSE(w, flusher, "error", map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "api_error",
				"message": fmt.Sprintf("failed to execute codex: %v", err),
			},
		})
		return
	}

	// Stream content_block_delta events
	var textBuffer string
	for chunk := range stream {
		if chunk.Type == "error" {
			h.writeSSE(w, flusher, "error", map[string]interface{}{
				"type": "error",
				"error": map[string]string{
					"type":    "api_error",
					"message": chunk.Error,
				},
			})
			return
		}

		if chunk.Type == "item.completed" && chunk.Item != nil {
			var item ItemContent
			if err := json.Unmarshal(chunk.Item, &item); err == nil {
				if item.Type == "agent_message" {
					textBuffer += item.Text
					h.writeSSE(w, flusher, "content_block_delta", map[string]interface{}{
						"type":  "content_block_delta",
						"index": 0,
						"delta": map[string]string{
							"type": "text_delta",
							"text": item.Text,
						},
					})
				}
			}
		}
	}

	// Send content_block_stop event
	h.writeSSE(w, flusher, "content_block_stop", map[string]interface{}{
		"type":  "content_block_stop",
		"index": 0,
	})

	// Send message_delta event with usage
	h.writeSSE(w, flusher, "message_delta", map[string]interface{}{
		"type": "message_delta",
		"delta": map[string]interface{}{
			"stop_reason":   "end_turn",
			"stop_sequence": nil,
		},
		"usage": map[string]int{
			"output_tokens": len(textBuffer) / 4,
		},
	})

	// Send message_stop event
	h.writeSSE(w, flusher, "message_stop", map[string]interface{}{
		"type": "message_stop",
	})

	if h.metrics != nil {
		h.metrics.RecordRequest("anthropic", req.Model, time.Since(time.Now()))
	}
}

// handleNonStream handles non-streaming responses
func (h *AnthropicHandler) handleNonStream(w http.ResponseWriter, r *http.Request, req *AnthropicMessageRequest) {
	ctx := r.Context()
	messageID := fmt.Sprintf("msg_%s", uuid.New().String())
	model := req.Model
	if model == "" {
		model = h.defaultModel
	}

	prompt := h.buildPrompt(req.Messages)

	stream, err := h.client.Run(ctx, prompt)
	if err != nil {
		WriteError(w, &APIError{
			Code:    ErrCodexFailed,
			Message: fmt.Sprintf("failed to execute codex: %v", err),
			Status:  http.StatusInternalServerError,
		})
		return
	}

	var textBuffer string
	for chunk := range stream {
		if chunk.Type == "error" {
			WriteError(w, &APIError{
				Code:    ErrCodexFailed,
				Message: chunk.Error,
				Status:  http.StatusInternalServerError,
			})
			return
		}

		if chunk.Type == "item.completed" && chunk.Item != nil {
			var item ItemContent
			if err := json.Unmarshal(chunk.Item, &item); err == nil {
				if item.Type == "agent_message" {
					textBuffer += item.Text
				}
			}
		}
	}

	resp := AnthropicMessageResponse{
		ID:    messageID,
		Type:  "message",
		Role:  "assistant",
		Model: model,
		Content: []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{
			{
				Type: "text",
				Text: textBuffer,
			},
		},
		StopReason: "end_turn",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	if h.metrics != nil {
		h.metrics.RecordRequest("anthropic", req.Model, time.Since(time.Now()))
	}
}

// writeSSE writes an SSE event to the response writer
func (h *AnthropicHandler) writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}

	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flusher.Flush()
}

// buildPrompt converts messages to a single prompt string
func (h *AnthropicHandler) buildPrompt(messages []Message) string {
	var prompt string
	for i, msg := range messages {
		if i > 0 {
			prompt += "\n\n"
		}
		prompt += fmt.Sprintf("%s: %s", msg.Role, msg.GetTextContent())
	}
	return prompt
}
