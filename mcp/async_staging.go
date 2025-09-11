package mcp

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/termfx/morfx/models"
)

// AsyncStagingManager handles staging with goroutine pool
type AsyncStagingManager struct {
	*StagingManager

	// Worker pool
	workers    int
	stageChan  chan stageRequest
	resultChan chan stageResult
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewAsyncStagingManager creates concurrent staging manager
func NewAsyncStagingManager(db *gorm.DB, config Config) *AsyncStagingManager {
	ctx, cancel := context.WithCancel(context.Background())

	asm := &AsyncStagingManager{
		StagingManager: NewStagingManager(db, config),
		workers:        10, // DB connection pool size
		stageChan:      make(chan stageRequest, 100),
		resultChan:     make(chan stageResult, 100),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Start worker pool
	for i := 0; i < asm.workers; i++ {
		asm.wg.Add(1)
		go asm.stageWorker()
	}

	// Result collector
	go asm.resultCollector()

	return asm
}

type stageRequest struct {
	stage    *models.Stage
	callback chan<- error
}

type stageResult struct {
	stageID string
	err     error
	latency time.Duration
}

// CreateStageAsync stages transformation without blocking
func (asm *AsyncStagingManager) CreateStageAsync(stage *models.Stage) <-chan error {
	callback := make(chan error, 1)

	select {
	case asm.stageChan <- stageRequest{stage: stage, callback: callback}:
		// Queued successfully
	case <-time.After(100 * time.Millisecond):
		// Queue full, fallback to sync
		go func() {
			callback <- asm.CreateStage(stage)
		}()
	}

	return callback
}

// stageWorker processes staging requests
func (asm *AsyncStagingManager) stageWorker() {
	defer asm.wg.Done()

	for {
		select {
		case <-asm.ctx.Done():
			return

		case req := <-asm.stageChan:
			start := time.Now()
			err := asm.CreateStage(req.stage)

			// Send result
			req.callback <- err
			close(req.callback)

			// Track metrics
			asm.resultChan <- stageResult{
				stageID: req.stage.ID,
				err:     err,
				latency: time.Since(start),
			}
		}
	}
}

// resultCollector aggregates metrics
func (asm *AsyncStagingManager) resultCollector() {
	var (
		totalStages  int64
		totalLatency time.Duration
		errorCount   int64
		maxLatency   time.Duration
	)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-asm.ctx.Done():
			return

		case result := <-asm.resultChan:
			totalStages++
			totalLatency += result.latency

			if result.err != nil {
				errorCount++
			}

			if result.latency > maxLatency {
				maxLatency = result.latency
			}

		case <-ticker.C:
			if totalStages > 0 {
				avgLatency := totalLatency / time.Duration(totalStages)
				asm.debugLog("Staging metrics: total=%d, errors=%d, avg=%v, max=%v",
					totalStages, errorCount, avgLatency, maxLatency)
			}
		}
	}
}

// BatchCreateStages creates multiple stages concurrently
func (asm *AsyncStagingManager) BatchCreateStages(stages []*models.Stage) []error {
	results := make([]error, len(stages))
	callbacks := make([]<-chan error, len(stages))

	// Fire all requests
	for i, stage := range stages {
		callbacks[i] = asm.CreateStageAsync(stage)
	}

	// Collect results
	for i, callback := range callbacks {
		results[i] = <-callback
	}

	return results
}

// Close shuts down worker pool
func (asm *AsyncStagingManager) Close() {
	asm.cancel()
	close(asm.stageChan)
	asm.wg.Wait()
	close(asm.resultChan)
}

// debugLog is a helper for logging
func (asm *AsyncStagingManager) debugLog(format string, args ...any) {
	if asm.config.Debug {
		fmt.Fprintf(os.Stderr, "[STAGING] "+format+"\n", args...)
	}
}
