package engine

import (
	"context"
	"fmt"

	"github.com/oxhq/morfx/core"
)

func (e *Engine) writePipelineApply(path string, modified string, meta StageCreateRequest) (bool, Stage, error) {
	switch e.cfg.WriteMode {
	case WriteModePreview:
		return false, Stage{}, nil
	case WriteModeApply:
		writer := core.NewAtomicWriter(core.DefaultAtomicConfig())
		if err := writer.WriteFile(path, modified); err != nil {
			return false, Stage{}, err
		}
		return true, Stage{}, nil
	case WriteModeStage:
		if !e.cfg.EnableStaging {
			return false, Stage{}, fmt.Errorf("staging disabled")
		}
		stage, err := e.CreateStage(context.Background(), meta)
		if err != nil {
			return false, Stage{}, err
		}
		return false, stage, nil
	default:
		return false, Stage{}, fmt.Errorf("unknown write mode: %s", e.cfg.WriteMode)
	}
}
