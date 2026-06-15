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
