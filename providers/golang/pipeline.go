package golang

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/termfx/morfx/core"
)

// TransformPipeline processes transformations in parallel stages
type TransformPipeline struct {
	provider *Provider
	workers  int
}

// NewTransformPipeline creates concurrent transform pipeline
func (p *Provider) NewTransformPipeline() *TransformPipeline {
	return &TransformPipeline{
		provider: p,
		workers:  runtime.NumCPU() * 2, // Oversubscribe for I/O
	}
}

// BatchTransform applies transformation to multiple targets concurrently
func (tp *TransformPipeline) BatchTransform(
	source string,
	op core.TransformOp,
	targets []*sitter.Node,
) core.TransformResult {
	if len(targets) == 0 {
		return core.TransformResult{Error: fmt.Errorf("no targets")}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Stage 1: Fan-out transformations
	transformChan := make(chan transformJob, len(targets))
	resultChan := make(chan transformResult, len(targets))

	// Start transform workers
	var wg sync.WaitGroup
	for i := 0; i < tp.workers; i++ {
		wg.Add(1)
		go tp.transformWorker(ctx, &wg, transformChan, resultChan)
	}

	// Queue all transforms
	for i, target := range targets {
		transformChan <- transformJob{
			index:  i,
			source: source,
			target: target,
			op:     op,
		}
	}
	close(transformChan)

	// Stage 2: Collect and merge results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Merge transforms in order
	results := make([]transformResult, len(targets))
	for res := range resultChan {
		results[res.index] = res
	}

	// Apply transforms from end to beginning (preserve positions)
	modified := source
	for i := len(results) - 1; i >= 0; i-- {
		if results[i].err != nil {
			return core.TransformResult{Error: results[i].err}
		}
		modified = tp.applyTransform(modified, results[i])
	}

	// Calculate final confidence (minimum of all)
	confidence := tp.mergeConfidence(results)

	return core.TransformResult{
		Modified:   modified,
		Diff:       tp.provider.generateDiff(source, modified),
		Confidence: confidence,
		MatchCount: len(targets),
	}
}

type transformJob struct {
	index  int
	source string
	target *sitter.Node
	op     core.TransformOp
}

type transformResult struct {
	index      int
	startByte  uint32
	endByte    uint32
	newContent string
	confidence core.ConfidenceScore
	err        error
}

// transformWorker processes individual transformations
func (tp *TransformPipeline) transformWorker(
	ctx context.Context,
	wg *sync.WaitGroup,
	jobs <-chan transformJob,
	results chan<- transformResult,
) {
	defer wg.Done()

	for job := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
			result := tp.processTransform(job)
			results <- result
		}
	}
}

// processTransform handles single transformation
func (tp *TransformPipeline) processTransform(job transformJob) transformResult {
	result := transformResult{
		index:     job.index,
		startByte: job.target.StartByte(),
		endByte:   job.target.EndByte(),
	}

	// Generate new content based on operation
	switch job.op.Method {
	case "replace":
		result.newContent = job.op.Replacement
	case "delete":
		result.newContent = ""
	case "insert_before":
		original := job.source[job.target.StartByte():job.target.EndByte()]
		result.newContent = job.op.Content + "\n" + original
	case "insert_after":
		original := job.source[job.target.StartByte():job.target.EndByte()]
		result.newContent = original + "\n" + job.op.Content
	case "append":
		// For append in pipeline, we need to handle it like insert_after but inside the target scope
		// This is a simplified version - for complex append logic, consider calling doAppendToTarget
		original := job.source[job.target.StartByte():job.target.EndByte()]

		// Check if target is a function/method and append inside body
		if job.target.Type() == "function_declaration" || job.target.Type() == "method_declaration" {
			// Find body and insert before closing brace
			if body := job.target.ChildByFieldName("body"); body != nil {
				bodyEnd := body.EndByte() - job.target.StartByte()
				if bodyEnd > 0 {
					// Insert before closing }
					beforeBody := original[:bodyEnd-1]
					afterBody := original[bodyEnd-1:]
					indent := tp.provider.getIndentation(job.source, job.target)
					innerIndent := tp.provider.detectInnerIndentation(job.source, job.target)
					result.newContent = beforeBody + "\n" + innerIndent + job.op.Content + "\n" + indent + afterBody
				} else {
					// Fallback to insert_after behavior
					result.newContent = original + "\n" + job.op.Content
				}
			} else {
				// No body, append after
				result.newContent = original + "\n" + job.op.Content
			}
		} else {
			// For non-functions, use insert_after behavior
			result.newContent = original + "\n" + job.op.Content
		}
	default:
		result.err = fmt.Errorf("unknown method: %s", job.op.Method)
		return result
	}

	// Calculate confidence for this specific transform
	result.confidence = tp.provider.calculateTransformConfidence(job.op, job.target, job.source)

	return result
}

// applyTransform merges single transform into source
func (tp *TransformPipeline) applyTransform(source string, result transformResult) string {
	before := source[:result.startByte]
	after := source[result.endByte:]
	return before + result.newContent + after
}

// mergeConfidence calculates aggregate confidence
func (tp *TransformPipeline) mergeConfidence(results []transformResult) core.ConfidenceScore {
	if len(results) == 0 {
		return core.ConfidenceScore{Score: 1.0, Level: "high"}
	}

	// Use minimum confidence
	minScore := 1.0
	var factors []core.ConfidenceFactor

	for _, res := range results {
		if res.confidence.Score < minScore {
			minScore = res.confidence.Score
		}
		factors = append(factors, res.confidence.Factors...)
	}

	// Add factor for multiple targets
	if len(results) > 1 {
		penalty := float64(len(results)) * 0.05
		if penalty > 0.3 {
			penalty = 0.3
		}
		minScore -= penalty

		factors = append(factors, core.ConfidenceFactor{
			Name:   "multiple_targets",
			Impact: -penalty,
			Reason: fmt.Sprintf("Affecting %d locations", len(results)),
		})
	}

	level := "high"
	if minScore < 0.8 {
		level = "medium"
	}
	if minScore < 0.5 {
		level = "low"
	}

	return core.ConfidenceScore{
		Score:   minScore,
		Level:   level,
		Factors: factors,
	}
}

// calculateTransformConfidence calculates confidence for single transform
func (p *Provider) calculateTransformConfidence(
	op core.TransformOp,
	target *sitter.Node,
	source string,
) core.ConfidenceScore {
	score := 1.0
	factors := []core.ConfidenceFactor{}

	// Operation type impacts
	switch op.Method {
	case "delete":
		score -= 0.2
		factors = append(factors, core.ConfidenceFactor{
			Name:   "delete_operation",
			Impact: -0.2,
			Reason: "Destructive operation",
		})
	case "replace":
		name := p.extractNodeName(target, source)
		if p.isExported(name) {
			score -= 0.15
			factors = append(factors, core.ConfidenceFactor{
				Name:   "exported_api",
				Impact: -0.15,
				Reason: "Modifying public API",
			})
		}
	}

	return core.ConfidenceScore{
		Score:   score,
		Level:   "high",
		Factors: factors,
	}
}
