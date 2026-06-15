package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/oxhq/morfx/core"
)

func (e *Engine) writePipelineApply(path string, original string, modified string) (bool, string, error) {
	switch e.cfg.WriteMode {
	case WriteModePreview:
		return false, "", nil
	case WriteModeApply:
		writer := core.NewAtomicWriter(core.DefaultAtomicConfig())
		if err := writer.WriteFile(path, modified); err != nil {
			return false, "", err
		}
		return true, "", nil
	case WriteModeStage:
		if !e.cfg.EnableStaging {
			return false, "", fmt.Errorf("staging disabled")
		}
		stageDir := strings.TrimSpace(e.cfg.StageDir)
		if stageDir == "" {
			if len(e.cfg.AllowedRoots) == 1 {
				stageDir = defaultStageDirFromRoot(e.cfg.AllowedRoots[0])
			}
		}
		if stageDir == "" {
			return false, "", fmt.Errorf("stage dir is required")
		}
		if err := isDirWritable(stageDir); err != nil {
			return false, "", err
		}
		stage, err := e.CreateStage(context.Background(), StageCreateRequest{
			Path:     path,
			Original: original,
			Modified: modified,
			Diff:     "",
		})
		if err != nil {
			return false, "", err
		}
		return false, stage.ID, nil
	default:
		return false, "", fmt.Errorf("unknown write mode: %s", e.cfg.WriteMode)
	}
}
