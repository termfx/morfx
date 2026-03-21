package mcp

import "github.com/termfx/morfx/core"

// fileSafetyAdapter bridges the core file processor with the SafetyManager.
type fileSafetyAdapter struct {
	manager *SafetyManager
}

func newFileSafetyAdapter(manager *SafetyManager) core.FileSafety {
	if manager == nil {
		return nil
	}
	return &fileSafetyAdapter{manager: manager}
}

func (a *fileSafetyAdapter) ValidateBatch(_ core.FileScope, files []core.WalkResult) error {
	if len(files) == 0 {
		return nil
	}

	op := &SafetyOperation{GlobalConfidence: 1.0}
	op.Files = make([]SafetyFile, 0, len(files))

	for _, file := range files {
		size := int64(0)
		if file.Info != nil {
			size = file.Info.Size()
		}

		op.Files = append(op.Files, SafetyFile{
			Path:       file.Path,
			Size:       size,
			Confidence: 1.0,
		})
	}

	return a.manager.ValidateOperation(op)
}

func (a *fileSafetyAdapter) ValidateFileChange(file core.WalkResult, confidence core.ConfidenceScore) error {
	score := confidence.Score
	if score < 0 {
		score = 0
	}

	op := &SafetyOperation{
		Files: []SafetyFile{{
			Path: file.Path,
			Size: func() int64 {
				if file.Info != nil {
					return file.Info.Size()
				}
				return 0
			}(),
			Confidence: score,
		}},
		GlobalConfidence: score,
	}

	return a.manager.ValidateOperation(op)
}
