package engine

import (
	"time"

	"github.com/oxhq/morfx/core"
)

type Engine struct {
	cfg     Config
	runtime *Runtime
}

type QueryRequest struct {
	Language string
	Source   string
	Query    core.AgentQuery
	DSL      string
}

type QueryResult struct {
	Matches int
	Results []core.Match
}

type TransformRequest struct {
	Language string
	Source   string
	Op       core.TransformOp
}

type TransformResult struct {
	MatchCount int
	Modified   string
	Diff       string
	Confidence core.ConfidenceScore
}

type FileTransformRequest struct {
	Language string
	Path     string
	Op       core.TransformOp
}

type FileTransformResult struct {
	Applied    bool
	StageID    string
	MatchCount int
	Diff       string
	Confidence core.ConfidenceScore
}

type StageStatus string

const (
	StageStatusPending StageStatus = "pending"
	StageStatusApplied StageStatus = "applied"
	StageStatusExpired StageStatus = "expired"
	StageStatusFailed  StageStatus = "failed"
)

type Stage struct {
	ID          string               `json:"id"`
	CreatedAt   time.Time            `json:"created_at"`
	ExpiresAt   time.Time            `json:"expires_at"`
	AppliedAt   *time.Time           `json:"applied_at,omitempty"`
	Status      StageStatus          `json:"status"`
	Language    string               `json:"language"`
	Operation   string               `json:"operation"`
	Path        string               `json:"path"`
	Original    string               `json:"original"`
	Modified    string               `json:"modified"`
	Diff        string               `json:"diff"`
	BaseDigest  string               `json:"base_digest"`
	AfterDigest string               `json:"after_digest"`
	Confidence  core.ConfidenceScore `json:"confidence"`
	Metadata    map[string]any       `json:"metadata,omitempty"`
}

type StageCreateRequest struct {
	Path        string
	Language    string
	Operation   string
	Original    string
	Modified    string
	Diff        string
	BaseDigest  string
	AfterDigest string
	Confidence  core.ConfidenceScore
	Metadata    map[string]any
	ExpiresAt   time.Time
}

type StageFilter struct {
	Status StageStatus
}

type StageApplyRequest struct {
	ID          string
	AutoApplied bool
}

type ApplyResult struct {
	StageID       string
	Applied       bool
	AutoApplied   bool
	Status        StageStatus
	AppliedAt     *time.Time
	FailureReason string
}

type FileQueryRequest struct {
	Scope core.FileScope
	Query core.AgentQuery
	DSL   string
}

type FileQueryResult struct {
	Results []core.FileMatch
}

type FileReplaceRequest struct {
	Scope  core.FileScope
	Op     core.TransformOp
	DryRun bool
	Backup bool
}

type FileReplaceResult struct {
	FilesScanned  int
	FilesModified int
	TotalMatches  int
	Errors        []string
	TransactionID string
	Details       []core.FileTransformDetail
}
