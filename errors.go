package main

import (
	"encoding/json"
	"net/http"
)

// Error codes
const (
	ErrInvalidRequest = "invalid_request"
	ErrUnauthorized   = "unauthorized"
	ErrTimeout        = "timeout"
	ErrCodexFailed    = "codex_failed"
)

// APIError represents a structured error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return e.Message
}

// WriteError writes an APIError as JSON response
func WriteError(w http.ResponseWriter, err *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(err)
}
