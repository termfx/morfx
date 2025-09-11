package mcp

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"sync"
	"time"
)

// OAuthSession represents an active OAuth flow session
type OAuthSession struct {
	State         string    `json:"state"`          // CSRF protection
	CodeVerifier  string    `json:"code_verifier"`  // PKCE verifier
	CodeChallenge string    `json:"code_challenge"` // PKCE challenge
	Provider      string    `json:"provider"`       // oauth provider
	RedirectURI   string    `json:"redirect_uri"`   // where to redirect after auth
	Scopes        []string  `json:"scopes"`         // requested scopes
	Resource      string    `json:"resource"`       // RFC 8707 resource indicator
	ExpiresAt     time.Time `json:"expires_at"`     // session expiration

	// Client context
	ClientID string            `json:"client_id,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// OAuthSessionStore manages active OAuth sessions
type OAuthSessionStore struct {
	sessions map[string]*OAuthSession
	mu       sync.RWMutex

	// Cleanup
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// NewOAuthSessionStore creates a new session store
func NewOAuthSessionStore(cleanupInterval time.Duration) *OAuthSessionStore {
	store := &OAuthSessionStore{
		sessions:        make(map[string]*OAuthSession),
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	// Start background cleanup
	go store.cleanupExpiredSessions()

	return store
}

// CreateSession creates a new OAuth session with PKCE
func (s *OAuthSessionStore) CreateSession(
	provider, redirectURI, resource string,
	scopes []string,
) (*OAuthSession, error) {
	// Generate CSRF state
	state, err := generateRandomString(32)
	if err != nil {
		return nil, err
	}

	return s.createSessionInternal(state, provider, redirectURI, resource, scopes)
}

// CreateSessionWithState creates OAuth session using provided state (for MCP)
func (s *OAuthSessionStore) CreateSessionWithState(
	state, provider string,
	scopes []string,
) (*OAuthSession, error) {
	// Use ChatGPT's state directly - no redirect/resource needed for MCP
	return s.createSessionInternal(state, provider, "", "", scopes)
}

// CreateSessionWithChatGPTPKCE creates session using ChatGPT's PKCE parameters (MCP flow)
func (s *OAuthSessionStore) CreateSessionWithChatGPTPKCE(
	state, provider string,
	scopes []string,
	codeChallenge, challengeMethod string,
) (*OAuthSession, error) {
	// MCP flow: ChatGPT provides both state and PKCE challenge
	// We don't generate code_verifier since ChatGPT owns it
	session := &OAuthSession{
		State:         state,
		CodeVerifier:  "", // ChatGPT owns the verifier
		CodeChallenge: codeChallenge,
		Provider:      provider,
		RedirectURI:   "", // Not needed for MCP
		Scopes:        scopes,
		Resource:      "", // Not needed for MCP
		ExpiresAt:     time.Now().Add(10 * time.Minute),
		Metadata: map[string]string{
			"challenge_method": challengeMethod,
			"mcp_flow":         "true",
		},
	}

	s.mu.Lock()
	s.sessions[state] = session
	s.mu.Unlock()

	return session, nil
}

// createSessionInternal handles the actual session creation logic
func (s *OAuthSessionStore) createSessionInternal(
	state, provider, redirectURI, resource string,
	scopes []string,
) (*OAuthSession, error) {
	// Generate PKCE verifier and challenge
	verifier, err := generateRandomString(43) // base64url(32 bytes) = 43 chars
	if err != nil {
		return nil, err
	}

	challenge := generateCodeChallenge(verifier)

	session := &OAuthSession{
		State:         state,
		CodeVerifier:  verifier,
		CodeChallenge: challenge,
		Provider:      provider,
		RedirectURI:   redirectURI,
		Scopes:        scopes,
		Resource:      resource,
		ExpiresAt:     time.Now().Add(10 * time.Minute), // 10min session timeout
		Metadata:      make(map[string]string),
	}

	s.mu.Lock()
	s.sessions[state] = session
	s.mu.Unlock()

	return session, nil
}

// GetSession retrieves a session by state
func (s *OAuthSessionStore) GetSession(state string) (*OAuthSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[state]
	if !exists {
		return nil, false
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		// Clean up expired session
		go func() {
			s.mu.Lock()
			delete(s.sessions, state)
			s.mu.Unlock()
		}()
		return nil, false
	}

	return session, true
}

// ConsumeSession retrieves and deletes a session (one-time use)
func (s *OAuthSessionStore) ConsumeSession(state string) (*OAuthSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[state]
	if !exists {
		return nil, false
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		delete(s.sessions, state)
		return nil, false
	}

	// Remove session (one-time use)
	delete(s.sessions, state)
	return session, true
}

// cleanupExpiredSessions periodically removes expired sessions
func (s *OAuthSessionStore) cleanupExpiredSessions() {
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			s.mu.Lock()

			for state, session := range s.sessions {
				if now.After(session.ExpiresAt) {
					delete(s.sessions, state)
				}
			}

			s.mu.Unlock()
		case <-s.stopCleanup:
			return
		}
	}
}

// Close stops the session store
func (s *OAuthSessionStore) Close() {
	close(s.stopCleanup)
}

// generateRandomString creates a cryptographically secure random string
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// generateCodeChallenge creates a PKCE code challenge from verifier using S256
func generateCodeChallenge(verifier string) string {
	// RFC 7636: base64url(sha256(code_verifier))
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// Stats returns session store statistics
func (s *OAuthSessionStore) Stats() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	active := 0
	expired := 0
	now := time.Now()

	for _, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			expired++
		} else {
			active++
		}
	}

	return map[string]any{
		"active_sessions":  active,
		"expired_sessions": expired,
		"total_sessions":   len(s.sessions),
	}
}
