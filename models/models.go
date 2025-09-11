package models

import (
	"time"

	"gorm.io/datatypes"
)

// Stage represents a pending code transformation
type Stage struct {
	ID        string `gorm:"primaryKey;type:varchar(20)"`
	SessionID string `gorm:"type:varchar(20);index"`

	// Operation details
	Language  string `gorm:"type:varchar(50);not null"`
	Operation string `gorm:"type:varchar(20);not null"` // query, replace, delete, etc

	// Target information
	TargetType  string         `gorm:"type:varchar(50)"`  // function, struct, class
	TargetName  string         `gorm:"type:varchar(255)"` // name pattern
	TargetQuery datatypes.JSON `gorm:"type:jsonb"`        // full query object

	// Content
	Original string `gorm:"type:text"`
	Modified string `gorm:"type:text"`
	Content  string `gorm:"type:text"` // For insert operations
	Diff     string `gorm:"type:text"`

	// Checksums for validation
	BaseDigest  string `gorm:"type:varchar(64)"` // SHA256 of original
	AfterDigest string `gorm:"type:varchar(64)"` // SHA256 of modified

	// Confidence scoring
	ConfidenceScore   float64        `gorm:"type:decimal(3,2)"`
	ConfidenceLevel   string         `gorm:"type:varchar(10)"`
	ConfidenceFactors datatypes.JSON `gorm:"type:jsonb"`

	// Scope AST for advanced operations
	ScopeAST datatypes.JSON `gorm:"type:jsonb"`

	// Status tracking
	Status    string    `gorm:"type:varchar(20);default:'pending'"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	ExpiresAt time.Time `gorm:"index"`
	AppliedAt *time.Time

	// Relationships
	Apply *Apply `gorm:"foreignKey:StageID"`
}

// Apply represents a committed transformation
type Apply struct {
	ID      string `gorm:"primaryKey;type:varchar(20)"`
	StageID string `gorm:"type:varchar(20);uniqueIndex"`

	// Checksums for validation
	BaseDigest  string `gorm:"type:varchar(64)"` // SHA256 of original
	AfterDigest string `gorm:"type:varchar(64)"` // SHA256 of modified

	// Metadata
	AutoApplied bool      `gorm:"default:false"`
	AppliedBy   string    `gorm:"type:varchar(100)"` // User or "auto"
	AppliedAt   time.Time `gorm:"autoCreateTime"`

	// Revert tracking
	Reverted   bool   `gorm:"default:false"`
	RevertedBy string `gorm:"type:varchar(100)"`
	RevertedAt *time.Time

	// Relationship
	Stage Stage `gorm:"foreignKey:StageID"`
}

// Session tracks a complete Morfx transformation session
type Session struct {
	ID        string    `gorm:"primaryKey;type:varchar(20)"`
	StartedAt time.Time `gorm:"autoCreateTime"`
	EndedAt   *time.Time

	// Statistics
	StagesCount  int `gorm:"default:0"`
	AppliesCount int `gorm:"default:0"`

	// Client info
	ClientInfo datatypes.JSON `gorm:"type:jsonb"`
}

// MCPSession tracks HTTP MCP protocol sessions
type MCPSession struct {
	ID           string    `gorm:"primaryKey;type:varchar(20)"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	LastActivity time.Time `gorm:"autoUpdateTime"`
	ExpiresAt    time.Time `gorm:"index"`
	Active       bool      `gorm:"default:true;index"`

	// Request tracking
	RequestCount int    `gorm:"default:0"`
	UserAgent    string `gorm:"type:varchar(255)"`
	RemoteAddr   string `gorm:"type:varchar(45)"` // IPv6 max length

	// Protocol info
	ProtocolVersion string         `gorm:"type:varchar(20)"`
	ClientInfo      datatypes.JSON `gorm:"type:jsonb"`

	// OAuth integration
	OAuthSubject   string `gorm:"type:varchar(255);index"`
	OAuthTokenHash string `gorm:"type:varchar(64)"` // SHA256 of token for revocation
}

// OAuthClient represents RFC7591 registered OAuth clients
type OAuthClient struct {
	ID        string    `gorm:"primaryKey;type:varchar(20)"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	// OAuth RFC7591 fields
	ClientID      string         `gorm:"type:varchar(64);uniqueIndex;not null"`
	ClientSecret  string         `gorm:"type:varchar(128);not null"`
	RedirectURIs  datatypes.JSON `gorm:"type:jsonb;not null"`
	GrantTypes    datatypes.JSON `gorm:"type:jsonb;not null"`
	ResponseTypes datatypes.JSON `gorm:"type:jsonb;not null"`
	Scope         string         `gorm:"type:varchar(512)"`

	// Client metadata
	ClientName string         `gorm:"type:varchar(255)"`
	ClientURI  string         `gorm:"type:varchar(512)"`
	LogoURI    string         `gorm:"type:varchar(512)"`
	Contacts   datatypes.JSON `gorm:"type:jsonb"`
	TosURI     string         `gorm:"type:varchar(512)"`
	PolicyURI  string         `gorm:"type:varchar(512)"`

	// Technical config
	TokenEndpointAuthMethod string `gorm:"type:varchar(50);default:'client_secret_basic'"`
	ApplicationType         string `gorm:"type:varchar(20);default:'web'"`

	// Secret management
	SecretExpiresAt *time.Time
	Active          bool `gorm:"default:true;index"`

	// Additional metadata
	Metadata datatypes.JSON `gorm:"type:jsonb"`
}

// TableName customizations for cleaner names
func (Stage) TableName() string       { return "stages" }
func (Apply) TableName() string       { return "applies" }
func (Session) TableName() string     { return "sessions" }
func (MCPSession) TableName() string  { return "mcp_sessions" }
func (OAuthClient) TableName() string { return "oauth_clients" }
