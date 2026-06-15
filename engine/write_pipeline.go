package engine

import "github.com/oxhq/morfx/core"

func (e *Engine) writePipelineApply(path string, _ string, modified string) (bool, error) {
	if e.cfg.WriteMode == WriteModePreview {
		return false, nil
	}

	if e.cfg.WriteMode == WriteModeApply {
		writer := core.NewAtomicWriter(core.DefaultAtomicConfig())
		if err := writer.WriteFile(path, modified); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}
