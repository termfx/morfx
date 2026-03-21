package mcp

import (
	"sync"
	"time"

	"github.com/termfx/morfx/mcp/types"
)

// SessionState captures negotiated protocol details and client preferences for
// the active MCP connection.
type SessionState struct {
	mu                 sync.RWMutex
	initialized        bool
	protocolVersion    string
	clientCapabilities map[string]any
	loggingLevel       LogLevel
	clientRoots        []string
	samplingHistory    []types.SamplingRecord
	elicitationHistory []types.ElicitationRecord
}

// NewSessionState returns a session state with sensible defaults.
func NewSessionState() *SessionState {
	return &SessionState{
		clientCapabilities: make(map[string]any),
		loggingLevel:       LogLevelInfo,
	}
}

// MarkInitialized records the negotiated protocol version and client
// capabilities.
func (s *SessionState) MarkInitialized(protocolVersion string, capabilities map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initialized = true
	s.protocolVersion = protocolVersion
	s.clientRoots = nil
	s.samplingHistory = nil
	s.elicitationHistory = nil

	if capabilities == nil {
		s.clientCapabilities = make(map[string]any)
	} else {
		clone := make(map[string]any, len(capabilities))
		for k, v := range capabilities {
			clone[k] = v
		}
		s.clientCapabilities = clone
	}
}

// Initialized reports whether the handshake has completed.
func (s *SessionState) Initialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initialized
}

// ProtocolVersion returns the negotiated protocol version.
func (s *SessionState) ProtocolVersion() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.protocolVersion
}

// ClientCapabilities returns a shallow copy of the negotiated capabilities.
func (s *SessionState) ClientCapabilities() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clone := make(map[string]any, len(s.clientCapabilities))
	for k, v := range s.clientCapabilities {
		clone[k] = v
	}
	return clone
}

// SetLoggingLevel stores the requested minimum logging level.
func (s *SessionState) SetLoggingLevel(level LogLevel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loggingLevel = level
}

// LoggingLevel returns the currently configured minimum logging level.
func (s *SessionState) LoggingLevel() LogLevel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loggingLevel
}

// SetClientRoots records the roots returned by the client.
func (s *SessionState) SetClientRoots(roots []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	clone := make([]string, len(roots))
	copy(clone, roots)
	s.clientRoots = clone
}

// ClientRoots returns the negotiated root directories from the client, if any.
func (s *SessionState) ClientRoots() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clone := make([]string, len(s.clientRoots))
	copy(clone, s.clientRoots)
	return clone
}

// AppendSamplingRecord stores a sampling exchange for later inspection.
func (s *SessionState) AppendSamplingRecord(params map[string]any, result map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := types.SamplingRecord{
		Timestamp: time.Now().UTC(),
		Params:    cloneMap(params),
		Result:    cloneMap(result),
	}
	s.samplingHistory = append(s.samplingHistory, record)
}

// SamplingHistory retrieves a copy of recorded sampling exchanges.
func (s *SessionState) SamplingHistory() []types.SamplingRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clone := make([]types.SamplingRecord, len(s.samplingHistory))
	copy(clone, s.samplingHistory)
	return clone
}

// AppendElicitationRecord stores an elicitation exchange.
func (s *SessionState) AppendElicitationRecord(params map[string]any, result map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := types.ElicitationRecord{
		Timestamp: time.Now().UTC(),
		Params:    cloneMap(params),
		Result:    cloneMap(result),
	}
	s.elicitationHistory = append(s.elicitationHistory, record)
}

// ElicitationHistory returns recorded elicitation exchanges.
func (s *SessionState) ElicitationHistory() []types.ElicitationRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clone := make([]types.ElicitationRecord, len(s.elicitationHistory))
	copy(clone, s.elicitationHistory)
	return clone
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	clone := make(map[string]any, len(input))
	for k, v := range input {
		clone[k] = v
	}
	return clone
}
