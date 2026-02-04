package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// OllamaTagsResponse represents Ollama's /api/tags response
type OllamaTagsResponse struct {
	Models []OllamaModel `json:"models"`
}

// OllamaModel represents a single model in the tags response
type OllamaModel struct {
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
	Digest     string `json:"digest"`
}

// OllamaGenerateRequest represents Ollama's /api/generate request
type OllamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream,omitempty"`
}

// OllamaGenerateResponse represents Ollama's /api/generate response
type OllamaGenerateResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

// HandleOllamaTags handles GET /api/tags - returns list of available models
func HandleOllamaTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, &APIError{
			Code:    ErrInvalidRequest,
			Message: "method not allowed",
			Status:  http.StatusMethodNotAllowed,
		})
		return
	}

	// Return hardcoded Codex model as available
	resp := OllamaTagsResponse{
		Models: []OllamaModel{
			{
				Name:       "codex",
				ModifiedAt: time.Now().Format(time.RFC3339),
				Size:       0,
				Digest:     "codex-sonnet",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleOllamaChat handles POST /api/chat - chat completion with streaming
func HandleOllamaChat(client *CodexClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteError(w, &APIError{
				Code:    ErrInvalidRequest,
				Message: "method not allowed",
				Status:  http.StatusMethodNotAllowed,
			})
			return
		}

		var req OllamaChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, &APIError{
				Code:    ErrInvalidRequest,
				Message: "invalid request body",
				Status:  http.StatusBadRequest,
			})
			return
		}

		// Convert messages to prompt
		prompt := formatMessages(req.Messages)

		// Create context with timeout
		ctx, cancel := context.WithTimeout(r.Context(), client.timeout)
		defer cancel()

		// Execute codex
		stream, err := client.Run(ctx, prompt)
		if err != nil {
			WriteError(w, &APIError{
				Code:    ErrCodexFailed,
				Message: err.Error(),
				Status:  http.StatusInternalServerError,
			})
			return
		}

		// Handle non-streaming response
		if !req.Stream {
			var fullResponse string
			for chunk := range stream {
				if chunk.Error != "" {
					WriteError(w, &APIError{
						Code:    ErrCodexFailed,
						Message: chunk.Error,
						Status:  http.StatusInternalServerError,
					})
					return
				}
				// Accumulate text content
				if chunk.Type == "text" {
					var content string
					json.Unmarshal(chunk.Content, &content)
					fullResponse += content
				}
			}

			resp := OllamaChatResponse{
				Model:     req.Model,
				CreatedAt: time.Now().Format(time.RFC3339),
				Message: Message{
					Role:    "assistant",
					Content: fullResponse,
				},
				Done: true,
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Handle streaming response
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Transfer-Encoding", "chunked")

		flusher, ok := w.(http.Flusher)
		if !ok {
			WriteError(w, &APIError{
				Code:    ErrCodexFailed,
				Message: "streaming not supported",
				Status:  http.StatusInternalServerError,
			})
			return
		}

		encoder := json.NewEncoder(w)
		for chunk := range stream {
			if chunk.Error != "" {
				resp := OllamaChatResponse{
					Model:     req.Model,
					CreatedAt: time.Now().Format(time.RFC3339),
					Message: Message{
						Role:    "assistant",
						Content: chunk.Error,
					},
					Done: true,
				}
				encoder.Encode(resp)
				flusher.Flush()
				return
			}

			// Stream text content
			if chunk.Type == "text" {
				var content string
				json.Unmarshal(chunk.Content, &content)

				resp := OllamaChatResponse{
					Model:     req.Model,
					CreatedAt: time.Now().Format(time.RFC3339),
					Message: Message{
						Role:    "assistant",
						Content: content,
					},
					Done: false,
				}
				encoder.Encode(resp)
				flusher.Flush()
			}
		}

		// Send final done message
		resp := OllamaChatResponse{
			Model:     req.Model,
			CreatedAt: time.Now().Format(time.RFC3339),
			Message: Message{
				Role:    "assistant",
				Content: "",
			},
			Done: true,
		}
		encoder.Encode(resp)
		flusher.Flush()
	}
}

// HandleOllamaGenerate handles POST /api/generate - text generation with streaming
func HandleOllamaGenerate(client *CodexClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteError(w, &APIError{
				Code:    ErrInvalidRequest,
				Message: "method not allowed",
				Status:  http.StatusMethodNotAllowed,
			})
			return
		}

		var req OllamaGenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, &APIError{
				Code:    ErrInvalidRequest,
				Message: "invalid request body",
				Status:  http.StatusBadRequest,
			})
			return
		}

		// Create context with timeout
		ctx, cancel := context.WithTimeout(r.Context(), client.timeout)
		defer cancel()

		// Execute codex
		stream, err := client.Run(ctx, req.Prompt)
		if err != nil {
			WriteError(w, &APIError{
				Code:    ErrCodexFailed,
				Message: err.Error(),
				Status:  http.StatusInternalServerError,
			})
			return
		}

		// Handle non-streaming response
		if !req.Stream {
			var fullResponse string
			for chunk := range stream {
				if chunk.Error != "" {
					WriteError(w, &APIError{
						Code:    ErrCodexFailed,
						Message: chunk.Error,
						Status:  http.StatusInternalServerError,
					})
					return
				}
				// Accumulate text content
				if chunk.Type == "text" {
					var content string
					json.Unmarshal(chunk.Content, &content)
					fullResponse += content
				}
			}

			resp := OllamaGenerateResponse{
				Model:     req.Model,
				CreatedAt: time.Now().Format(time.RFC3339),
				Response:  fullResponse,
				Done:      true,
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Handle streaming response
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Transfer-Encoding", "chunked")

		flusher, ok := w.(http.Flusher)
		if !ok {
			WriteError(w, &APIError{
				Code:    ErrCodexFailed,
				Message: "streaming not supported",
				Status:  http.StatusInternalServerError,
			})
			return
		}

		encoder := json.NewEncoder(w)
		for chunk := range stream {
			if chunk.Error != "" {
				resp := OllamaGenerateResponse{
					Model:     req.Model,
					CreatedAt: time.Now().Format(time.RFC3339),
					Response:  chunk.Error,
					Done:      true,
				}
				encoder.Encode(resp)
				flusher.Flush()
				return
			}

			// Stream text content
			if chunk.Type == "text" {
				var content string
				json.Unmarshal(chunk.Content, &content)

				resp := OllamaGenerateResponse{
					Model:     req.Model,
					CreatedAt: time.Now().Format(time.RFC3339),
					Response:  content,
					Done:      false,
				}
				encoder.Encode(resp)
				flusher.Flush()
			}
		}

		// Send final done message
		resp := OllamaGenerateResponse{
			Model:     req.Model,
			CreatedAt: time.Now().Format(time.RFC3339),
			Response:  "",
			Done:      true,
		}
		encoder.Encode(resp)
		flusher.Flush()
	}
}

// formatMessages converts message array to a single prompt string
func formatMessages(messages []Message) string {
	var result string
	for _, msg := range messages {
		if msg.Role == "system" {
			result += "System: " + msg.Content + "\n\n"
		} else if msg.Role == "user" {
			result += "User: " + msg.Content + "\n\n"
		} else if msg.Role == "assistant" {
			result += "Assistant: " + msg.Content + "\n\n"
		}
	}
	return result
}
