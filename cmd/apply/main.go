package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/oxhq/morfx/engine"
	"github.com/oxhq/morfx/internal/toolenv"
)

const applyHelp = `Usage: apply [--root path] [--stage-dir path] [-h]

Reads a JSON request from stdin and emits a JSON response to stdout.

Input schema:
{
  "id": "<stage id>",          // optional; applies specific stage
  "all": <bool>,                // optional; apply every pending stage
  "latest": <bool>,             // optional; apply the most recent stage
  "session_id": "<session id>" // optional filter against stage metadata
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
  --root <path>       Allowed root for staged file application (default current directory)
  --stage-dir <path>  Stage directory (default <root>/.morfx-stages)
  -h, --help          Show this help message
`

type applyRequest struct {
	ID        string `json:"id,omitempty"`
	All       bool   `json:"all,omitempty"`
	Latest    bool   `json:"latest,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

func main() {
	var (
		rootDir  string
		stageDir string
		showHelp bool
	)

	flag.StringVar(&rootDir, "root", "", "Allowed root for staged file application")
	flag.StringVar(&stageDir, "stage-dir", "", "Directory containing engine stage files")
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

	lifecycle, err := newLifecycle(rootDir, stageDir)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "failed to initialize engine lifecycle", err)
		os.Exit(1)
	}

	appliedIDs, err := applyStages(context.Background(), lifecycle, req, mode)
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

func newLifecycle(rootDir string, stageDir string) (*engine.Engine, error) {
	if rootDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		rootDir = cwd
	}
	if stageDir == "" {
		stageDir = filepath.Join(rootDir, ".morfx-stages")
	}
	return engine.New(engine.Config{
		AllowedRoots:  []string{rootDir},
		WriteMode:     engine.WriteModeStage,
		EnableStaging: true,
		StageDir:      stageDir,
	})
}

func applyStages(ctx context.Context, lifecycle *engine.Engine, req *applyRequest, mode string) ([]string, error) {
	if lifecycle == nil {
		return nil, errors.New("engine lifecycle is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	switch mode {
	case "single":
		if req.ID == "" {
			return nil, errors.New("id is required for single mode")
		}
		if err := applyStage(ctx, lifecycle, req.ID); err != nil {
			return nil, err
		}
		return []string{req.ID}, nil

	case "all":
		ids, err := listPendingStageIDs(ctx, lifecycle, req.SessionID)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return nil, errors.New("no pending stages available")
		}
		var applied []string
		for _, id := range ids {
			if err := applyStage(ctx, lifecycle, id); err != nil {
				return applied, err
			}
			applied = append(applied, id)
		}
		return applied, nil

	case "latest":
		ids, err := listPendingStageIDs(ctx, lifecycle, req.SessionID)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return nil, errors.New("no pending stages available")
		}
		latestID := ids[0]
		if err := applyStage(ctx, lifecycle, latestID); err != nil {
			return nil, err
		}
		return []string{latestID}, nil

	default:
		return nil, fmt.Errorf("unsupported mode: %s", mode)
	}
}

func applyStage(ctx context.Context, lifecycle *engine.Engine, stageID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	_, err := lifecycle.ApplyStage(ctx, engine.StageApplyRequest{ID: stageID})
	return err
}

func listPendingStageIDs(ctx context.Context, lifecycle *engine.Engine, sessionID string) ([]string, error) {
	if lifecycle == nil {
		return nil, errors.New("engine lifecycle is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := lifecycle.ExpireStages(ctx, timeNow()); err != nil {
		return nil, err
	}
	stages, err := lifecycle.ListStages(ctx, engine.StageFilter{Status: engine.StageStatusPending})
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(stages))
	for _, stage := range stages {
		if sessionID != "" {
			if value, ok := stage.Metadata["session_id"].(string); !ok || value != sessionID {
				continue
			}
		}
		ids = append(ids, stage.ID)
	}
	return ids, nil
}

var timeNow = func() time.Time {
	return time.Now()
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
