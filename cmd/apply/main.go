package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/termfx/morfx/db"
	"github.com/termfx/morfx/internal/toolenv"
	"github.com/termfx/morfx/mcp"
	"github.com/termfx/morfx/models"
	"gorm.io/gorm"
)

const applyHelp = `Usage: apply [--db path] [-h]

Reads a JSON request from stdin and emits a JSON response to stdout.

Input schema:
{
  "id": "<stage id>",          // optional; applies specific stage
  "all": <bool>,                // optional; apply every pending stage
  "latest": <bool>,             // optional; apply the most recent stage
  "session_id": "<session id>" // optional filter when using database persistence
}
Exactly one of "id", "all", or "latest" may be set. If none are provided the
command defaults to "latest".

Output schema:
{
  "content": [{"type": "text", "text": "<summary>"}],
  "applied": ["<stage ids>", ...],
  "structuredContent": {
    "mode": "single|all|latest",
    "applied": ["<stage ids>", ...],
    "appliedCount": <int> // present only for mode "all"
  }
}

Flags:
  --db <path>   Path to the Morfx SQLite database (default ./.morfx/db/morfx.db)
  -h, --help    Show this help message
`

type applyRequest struct {
	ID        string `json:"id,omitempty"`
	All       bool   `json:"all,omitempty"`
	Latest    bool   `json:"latest,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

func main() {
	var (
		dbPath   string
		showHelp bool
	)

	flag.StringVar(&dbPath, "db", "./.morfx/db/morfx.db", "Path to the Morfx SQLite database")
	flag.BoolVar(&showHelp, "h", false, "Show help message")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.Usage = func() {
		fmt.Print(applyHelp)
	}
	flag.Parse()

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	req, err := toolenv.ReadJSON[applyRequest](os.Stdin)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "invalid input", err)
		os.Exit(1)
	}

	mode, err := determineMode(req)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "invalid parameters", err)
		os.Exit(1)
	}

	cfg := mcp.DefaultConfig()
	cfg.DatabaseURL = dbPath
	cfg.Debug = false
	cfg.LogWriter = io.Discard

	gormDB, err := db.Connect(cfg.DatabaseURL, cfg.Debug)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "failed to connect to database", err)
		os.Exit(1)
	}
	defer func() {
		if sqlDB, err := gormDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	safety := mcp.NewSafetyManager(cfg.Safety)
	staging := mcp.NewStagingManager(gormDB, cfg, safety)

	appliedIDs, err := applyStages(context.Background(), staging, gormDB, req, mode)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "apply operation failed", err)
		os.Exit(1)
	}

	responseText := buildApplyMessage(mode, appliedIDs)

	structured := map[string]any{"mode": mode}
	if len(appliedIDs) > 0 {
		structured["applied"] = append([]string{}, appliedIDs...)
	}
	if mode == "all" {
		structured["appliedCount"] = len(appliedIDs)
	}

	payload := map[string]any{
		"content": []map[string]any{{
			"type": "text",
			"text": responseText,
		}},
		"applied":           appliedIDs,
		"structuredContent": structured,
	}

	if err := toolenv.WriteJSON(os.Stdout, payload); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
		os.Exit(1)
	}
}

func determineMode(req *applyRequest) (string, error) {
	if req == nil {
		return "", errors.New("request cannot be nil")
	}

	candidates := 0
	if req.ID != "" {
		candidates++
	}
	if req.All {
		candidates++
	}
	if req.Latest {
		candidates++
	}

	if candidates > 1 {
		return "", errors.New("specify only one of 'id', 'all', or 'latest'")
	}

	if req.ID != "" {
		return "single", nil
	}
	if req.All {
		return "all", nil
	}

	req.Latest = true
	return "latest", nil
}

func applyStages(ctx context.Context, staging *mcp.StagingManager, gormDB *gorm.DB, req *applyRequest, mode string) ([]string, error) {
	switch mode {
	case "single":
		if req.ID == "" {
			return nil, errors.New("id is required for single mode")
		}
		if err := applyStage(ctx, staging, req.ID); err != nil {
			return nil, err
		}
		return []string{req.ID}, nil

	case "all":
		ids, err := fetchPendingStageIDs(gormDB, req.SessionID)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return nil, errors.New("no pending stages available")
		}
		var applied []string
		for _, id := range ids {
			if err := applyStage(ctx, staging, id); err != nil {
				return applied, err
			}
			applied = append(applied, id)
		}
		return applied, nil

	case "latest":
		ids, err := fetchPendingStageIDs(gormDB, req.SessionID)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return nil, errors.New("no pending stages available")
		}
		latestID := ids[0]
		if err := applyStage(ctx, staging, latestID); err != nil {
			return nil, err
		}
		return []string{latestID}, nil

	default:
		return nil, fmt.Errorf("unsupported mode: %s", mode)
	}
}

func applyStage(ctx context.Context, staging *mcp.StagingManager, stageID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	_, err := staging.ApplyStage(ctx, stageID, false)
	return err
}

func fetchPendingStageIDs(gormDB *gorm.DB, sessionID string) ([]string, error) {
	if gormDB == nil {
		return nil, errors.New("database handle is nil")
	}

	query := gormDB.Model(&models.Stage{}).
		Where("status = ?", "pending")
	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}

	var ids []string
	if err := query.Order("created_at DESC").Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func buildApplyMessage(mode string, applied []string) string {
	switch mode {
	case "single":
		if len(applied) == 0 {
			return "No stages applied"
		}
		return fmt.Sprintf("Applied stage: %s", applied[0])
	case "latest":
		if len(applied) == 0 {
			return "No stages applied"
		}
		return fmt.Sprintf("Applied latest stage: %s", applied[0])
	case "all":
		return fmt.Sprintf("Applied %d stage(s)", len(applied))
	default:
		return "Apply operation completed"
	}
}
