package main

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SessionState represents the current state of a session
type SessionState string

const (
	StateCreated   SessionState = "CREATED"
	StateRunning   SessionState = "RUNNING"
	StateCompleted SessionState = "COMPLETED"
	StateFailed    SessionState = "FAILED"
	StateClosed    SessionState = "CLOSED"
)

// Usage tracks token usage for a session
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Session represents a conversation session with Claude
type Session struct {
	ID        string       `json:"id"`
	State     SessionState `json:"state"`
	History   []Message    `json:"history"`
	Usage     Usage        `json:"usage"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	mu        sync.RWMutex
}

// SessionManager manages multiple sessions with thread-safe operations
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	ttl      time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewSessionManager creates a new session manager with cleanup goroutine
func NewSessionManager(ttl time.Duration) *SessionManager {
	ctx, cancel := context.WithCancel(context.Background())
	sm := &SessionManager{
		sessions: make(map[string]*Session),
		ttl:      ttl,
		ctx:      ctx,
		cancel:   cancel,
	}
	go sm.cleanupLoop()
	return sm
}

// Create creates a new session and returns its ID
func (sm *SessionManager) Create() string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	session := &Session{
		ID:        uuid.New().String(),
		State:     StateCreated,
		History:   make([]Message, 0),
		Usage:     Usage{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	sm.sessions[session.ID] = session
	return session.ID
}

// Get retrieves a session by ID
func (sm *SessionManager) Get(id string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[id]
	return session, exists
}

// Delete removes a session by ID
func (sm *SessionManager) Delete(id string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.sessions[id]; !exists {
		return false
	}

	delete(sm.sessions, id)
	return true
}

// List returns all session IDs
func (sm *SessionManager) List() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	ids := make([]string, 0, len(sm.sessions))
	for id := range sm.sessions {
		ids = append(ids, id)
	}
	return ids
}

// UpdateState updates the session state atomically
func (s *Session) UpdateState(state SessionState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.State = state
	s.UpdatedAt = time.Now()
}

// AddMessage adds a message to the session history
func (s *Session) AddMessage(msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.History = append(s.History, msg)
	s.UpdatedAt = time.Now()
}

// UpdateUsage updates the token usage atomically
func (s *Session) UpdateUsage(input, output int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Usage.InputTokens += input
	s.Usage.OutputTokens += output
	s.UpdatedAt = time.Now()
}

// GetState returns the current session state
func (s *Session) GetState() SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.State
}

// GetHistory returns a copy of the session history
func (s *Session) GetHistory() []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history := make([]Message, len(s.History))
	copy(history, s.History)
	return history
}

// GetUsage returns a copy of the usage stats
func (s *Session) GetUsage() Usage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.Usage
}

// cleanupLoop periodically removes stale sessions
func (sm *SessionManager) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			sm.cleanup()
		}
	}
}

// cleanup removes sessions older than TTL
func (sm *SessionManager) cleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for id, session := range sm.sessions {
		session.mu.RLock()
		age := now.Sub(session.UpdatedAt)
		session.mu.RUnlock()

		if age > sm.ttl {
			delete(sm.sessions, id)
		}
	}
}

// Close stops the cleanup goroutine
func (sm *SessionManager) Close() {
	sm.cancel()
}
