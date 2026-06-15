package engine

import "github.com/oxhq/morfx/core"

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
	MatchCount int
	Diff       string
	Confidence core.ConfidenceScore
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
