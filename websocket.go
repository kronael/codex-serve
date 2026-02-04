package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type      string          `json:"type"`
	SessionID string          `json:"session_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// WSHandler manages WebSocket connections
type WSHandler struct {
	sessions *SessionManager
	codex    *CodexClient
	mu       sync.RWMutex
	conns    map[string]*websocket.Conn
}

// NewWSHandler creates a new WebSocket handler
func NewWSHandler(sessions *SessionManager, codex *CodexClient) *WSHandler {
	return &WSHandler{
		sessions: sessions,
		codex:    codex,
		conns:    make(map[string]*websocket.Conn),
	}
}

// HandleSession handles WebSocket connections for interactive sessions
func (h *WSHandler) HandleSession(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("failed to upgrade connection: %v", err)
		return
	}

	sessionID := ""
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go func() {
		defer conn.Close()

		for {
			var msg WSMessage
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("websocket error: %v", err)
				}
				cancel()
				return
			}

			switch msg.Type {
			case "create":
				h.handleCreate(conn, &msg)

			case "resume":
				sessionID = msg.SessionID
				h.handleResume(conn, &msg)

			case "prompt":
				if sessionID == "" {
					h.sendError(conn, "no active session")
					continue
				}
				h.handlePrompt(ctx, conn, sessionID, &msg)

			case "close":
				if sessionID != "" {
					h.handleClose(sessionID)
					sessionID = ""
				}
				cancel()
				return

			default:
				h.sendError(conn, fmt.Sprintf("unknown message type: %s", msg.Type))
			}
		}
	}()

	<-ctx.Done()

	if sessionID != "" {
		h.mu.Lock()
		delete(h.conns, sessionID)
		h.mu.Unlock()
	}
}

// handleCreate creates a new session
func (h *WSHandler) handleCreate(conn *websocket.Conn, msg *WSMessage) {
	sessionID := h.sessions.Create()

	h.mu.Lock()
	h.conns[sessionID] = conn
	h.mu.Unlock()

	response := WSMessage{
		Type:      "session_created",
		SessionID: sessionID,
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Printf("failed to send session_created: %v", err)
	}
}

// handleResume resumes an existing session
func (h *WSHandler) handleResume(conn *websocket.Conn, msg *WSMessage) {
	if msg.SessionID == "" {
		h.sendError(conn, "session_id required")
		return
	}

	session, exists := h.sessions.Get(msg.SessionID)
	if !exists {
		h.sendError(conn, "session not found")
		return
	}

	h.mu.Lock()
	h.conns[msg.SessionID] = conn
	h.mu.Unlock()

	state := session.GetState()
	history := session.GetHistory()
	usage := session.GetUsage()

	resumeData := struct {
		SessionID string       `json:"session_id"`
		State     SessionState `json:"state"`
		History   []Message    `json:"history"`
		Usage     Usage        `json:"usage"`
	}{
		SessionID: msg.SessionID,
		State:     state,
		History:   history,
		Usage:     usage,
	}

	content, err := json.Marshal(resumeData)
	if err != nil {
		h.sendError(conn, "failed to marshal session data")
		return
	}

	response := WSMessage{
		Type:    "session_resumed",
		Content: content,
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Printf("failed to send session_resumed: %v", err)
	}
}

// handlePrompt processes a prompt and streams responses
func (h *WSHandler) handlePrompt(ctx context.Context, conn *websocket.Conn, sessionID string, msg *WSMessage) {
	session, exists := h.sessions.Get(sessionID)
	if !exists {
		h.sendError(conn, "session not found")
		return
	}

	var promptContent struct {
		Prompt string `json:"prompt"`
	}

	if err := json.Unmarshal(msg.Content, &promptContent); err != nil {
		h.sendError(conn, "invalid prompt content")
		return
	}

	session.UpdateState(StateRunning)
	session.AddMessage(Message{
		Role:    "user",
		Content: promptContent.Prompt,
	})

	promptCtx, promptCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer promptCancel()

	ch, err := h.codex.Run(promptCtx, promptContent.Prompt)
	if err != nil {
		session.UpdateState(StateFailed)
		h.sendError(conn, fmt.Sprintf("codex failed: %v", err))
		return
	}

	for response := range ch {
		wsMsg := WSMessage{
			Type:      "stream",
			SessionID: sessionID,
		}

		if response.Error != "" {
			wsMsg.Type = "error"
			wsMsg.Error = response.Error
		} else {
			wsMsg.Content = response.Content
		}

		if err := conn.WriteJSON(wsMsg); err != nil {
			log.Printf("failed to send stream message: %v", err)
			promptCancel()
			break
		}
	}

	if promptCtx.Err() == context.DeadlineExceeded {
		session.UpdateState(StateFailed)
		h.sendError(conn, "prompt timeout")
		return
	}

	session.UpdateState(StateCompleted)

	completedMsg := WSMessage{
		Type:      "completed",
		SessionID: sessionID,
	}

	if err := conn.WriteJSON(completedMsg); err != nil {
		log.Printf("failed to send completed message: %v", err)
	}
}

// handleClose closes a session
func (h *WSHandler) handleClose(sessionID string) {
	if session, exists := h.sessions.Get(sessionID); exists {
		session.UpdateState(StateClosed)
	}

	h.mu.Lock()
	delete(h.conns, sessionID)
	h.mu.Unlock()
}

// sendError sends an error message to the client
func (h *WSHandler) sendError(conn *websocket.Conn, message string) {
	response := WSMessage{
		Type:  "error",
		Error: message,
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Printf("failed to send error: %v", err)
	}
}
