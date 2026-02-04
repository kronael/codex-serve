package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// OpenAIModelsResponse represents OpenAI's /v1/models response
type OpenAIModelsResponse struct {
	Object string        `json:"object"`
	Data   []OpenAIModel `json:"data"`
}

// OpenAIModel represents a single model in the models response
type OpenAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// OpenAIStreamChoice represents a streaming choice delta
type OpenAIStreamChoice struct {
	Index int `json:"index"`
	Delta struct {
		Content string `json:"content,omitempty"`
		Role    string `json:"role,omitempty"`
	} `json:"delta"`
	FinishReason *string `json:"finish_reason"`
}

// OpenAIStreamResponse represents a streaming response chunk
type OpenAIStreamResponse struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []OpenAIStreamChoice `json:"choices"`
}

// HandleOpenAIModels handles GET /v1/models
func HandleOpenAIModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, &APIError{
			Code:    ErrInvalidRequest,
			Message: "method not allowed",
			Status:  http.StatusMethodNotAllowed,
		})
		return
	}

	resp := OpenAIModelsResponse{
		Object: "list",
		Data: []OpenAIModel{
			{
				ID:      "codex",
				Object:  "model",
				Created: time.Now().Unix(),
				OwnedBy: "anthropic",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleOpenAIChat handles POST /v1/chat/completions
func HandleOpenAIChat(client *CodexClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteError(w, &APIError{
				Code:    ErrInvalidRequest,
				Message: "method not allowed",
				Status:  http.StatusMethodNotAllowed,
			})
			return
		}

		var req OpenAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, &APIError{
				Code:    ErrInvalidRequest,
				Message: "invalid request body",
				Status:  http.StatusBadRequest,
			})
			return
		}

		prompt := formatMessages(req.Messages)
		ctx, cancel := context.WithTimeout(r.Context(), client.timeout)
		defer cancel()

		stream, err := client.Run(ctx, prompt)
		if err != nil {
			WriteError(w, &APIError{
				Code:    ErrCodexFailed,
				Message: err.Error(),
				Status:  http.StatusInternalServerError,
			})
			return
		}

		completionID := fmt.Sprintf("chatcmpl-%s", uuid.New().String())
		model := req.Model
		if model == "" {
			model = "codex"
		}

		// Non-streaming response
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
				if chunk.Type == "text" {
					var content string
					json.Unmarshal(chunk.Content, &content)
					fullResponse += content
				}
			}

			resp := OpenAIChatResponse{
				ID:      completionID,
				Object:  "chat.completion",
				Created: time.Now().Unix(),
				Model:   model,
				Choices: []struct {
					Index   int     `json:"index"`
					Message Message `json:"message"`
					Finish  string  `json:"finish_reason,omitempty"`
				}{
					{
						Index: 0,
						Message: Message{
							Role:    "assistant",
							Content: fullResponse,
						},
						Finish: "stop",
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Streaming response (SSE)
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

		for chunk := range stream {
			if chunk.Error != "" {
				errResp := OpenAIStreamResponse{
					ID:      completionID,
					Object:  "chat.completion.chunk",
					Created: time.Now().Unix(),
					Model:   model,
					Choices: []OpenAIStreamChoice{
						{
							Index:        0,
							FinishReason: strPtr("error"),
						},
					},
				}
				data, _ := json.Marshal(errResp)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
				return
			}

			if chunk.Type == "text" {
				var content string
				json.Unmarshal(chunk.Content, &content)

				resp := OpenAIStreamResponse{
					ID:      completionID,
					Object:  "chat.completion.chunk",
					Created: time.Now().Unix(),
					Model:   model,
					Choices: []OpenAIStreamChoice{
						{
							Index: 0,
							Delta: struct {
								Content string `json:"content,omitempty"`
								Role    string `json:"role,omitempty"`
							}{
								Content: content,
							},
							FinishReason: nil,
						},
					},
				}
				data, _ := json.Marshal(resp)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}

		// Send final chunk with finish_reason
		finalResp := OpenAIStreamResponse{
			ID:      completionID,
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   model,
			Choices: []OpenAIStreamChoice{
				{
					Index:        0,
					FinishReason: strPtr("stop"),
				},
			},
		}
		data, _ := json.Marshal(finalResp)
		fmt.Fprintf(w, "data: %s\n\n", data)
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}
}

func strPtr(s string) *string {
	return &s
}
