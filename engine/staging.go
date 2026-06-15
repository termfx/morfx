package engine

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/internal/securefs"
)

type StageStore interface {
	Create(ctx context.Context, req StageCreateRequest) (Stage, error)
	Get(ctx context.Context, id string) (Stage, error)
	List(ctx context.Context, filter StageFilter) ([]Stage, error)
	Update(ctx context.Context, stage Stage) error
}

type FileStageStore struct {
	dir string
}

func NewFileStageStore(dir string) *FileStageStore {
	_ = securefs.MkdirAll(dir, 0o700)
	return &FileStageStore{dir: dir}
}

func NewStageStore(dir string) *FileStageStore {
	return NewFileStageStore(dir)
}

func (s *FileStageStore) Create(ctx context.Context, req StageCreateRequest) (Stage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Stage{}, err
	}
	id, err := newStageID()
	if err != nil {
		return Stage{}, err
	}
	now := time.Now().UTC()
	expiresAt := req.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = now.Add(24 * time.Hour)
	}
	stage := Stage{
		ID:          id,
		CreatedAt:   now,
		ExpiresAt:   expiresAt.UTC(),
		Status:      StageStatusPending,
		Path:        req.Path,
		Language:    req.Language,
		Operation:   req.Operation,
		Original:    req.Original,
		Modified:    req.Modified,
		Diff:        req.Diff,
		BaseDigest:  req.BaseDigest,
		AfterDigest: req.AfterDigest,
		Confidence:  req.Confidence,
		Metadata:    cloneMetadata(req.Metadata),
	}
	if err := s.write(stage); err != nil {
		return Stage{}, err
	}
	return stage, nil
}

func (s *FileStageStore) Get(ctx context.Context, id string) (Stage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Stage{}, err
	}
	return s.read(id)
}

func (s *FileStageStore) Read(id string) (Stage, error) {
	return s.read(id)
}

func (s *FileStageStore) List(ctx context.Context, filter StageFilter) ([]Stage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := filepath.Glob(filepath.Join(s.dir, "*.json"))
	if err != nil {
		return nil, err
	}
	stages := make([]Stage, 0, len(entries))
	for _, entry := range entries {
		stage, err := s.read(strings.TrimSuffix(filepath.Base(entry), ".json"))
		if err != nil {
			return nil, err
		}
		if filter.Status != "" && stage.Status != filter.Status {
			continue
		}
		stages = append(stages, stage)
	}
	sort.Slice(stages, func(i, j int) bool {
		return stages[i].CreatedAt.After(stages[j].CreatedAt)
	})
	return stages, nil
}

func (s *FileStageStore) Update(ctx context.Context, stage Stage) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.write(stage)
}

func (s *FileStageStore) read(id string) (Stage, error) {
	b, err := securefs.ReadFile(filepath.Join(s.dir, id+".json"))
	if err != nil {
		return Stage{}, err
	}
	var stage Stage
	if err := json.Unmarshal(b, &stage); err != nil {
		return Stage{}, err
	}
	stage.Metadata = cloneMetadata(stage.Metadata)
	return stage, nil
}

func (s *FileStageStore) write(stage Stage) error {
	if strings.TrimSpace(stage.ID) == "" {
		return fmt.Errorf("stage id is required")
	}
	stage.Metadata = cloneMetadata(stage.Metadata)
	b, err := json.Marshal(stage)
	if err != nil {
		return err
	}
	return securefs.WriteFile(filepath.Join(s.dir, stage.ID+".json"), b, 0o600)
}

func cloneMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

func newStageID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func defaultStageDirFromRoot(root string) string {
	if root == "" {
		return ""
	}
	return filepath.Join(root, ".morfx-stages")
}

func isDirWritable(dir string) error {
	if err := securefs.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "probe-*")
	if err != nil {
		return err
	}
	securefs.CloseBestEffort(f)
	securefs.RemoveBestEffort(f.Name())
	return nil
}

func (e *Engine) stageStore() (StageStore, error) {
	stageDir := strings.TrimSpace(e.cfg.StageDir)
	if stageDir == "" && len(e.cfg.AllowedRoots) == 1 {
		stageDir = defaultStageDirFromRoot(e.cfg.AllowedRoots[0])
	}
	if stageDir == "" {
		return nil, fmt.Errorf("stage dir is required")
	}
	if err := isDirWritable(stageDir); err != nil {
		return nil, err
	}
	return NewFileStageStore(stageDir), nil
}

func (e *Engine) CreateStage(ctx context.Context, req StageCreateRequest) (Stage, error) {
	store, err := e.stageStore()
	if err != nil {
		return Stage{}, err
	}
	return store.Create(ctx, req)
}

func (e *Engine) GetStage(ctx context.Context, id string) (Stage, error) {
	store, err := e.stageStore()
	if err != nil {
		return Stage{}, err
	}
	return store.Get(ctx, id)
}

func (e *Engine) ListStages(ctx context.Context, filter StageFilter) ([]Stage, error) {
	store, err := e.stageStore()
	if err != nil {
		return nil, err
	}
	return store.List(ctx, filter)
}

func (e *Engine) ExpireStages(ctx context.Context, now time.Time) (int, error) {
	stages, err := e.ListStages(ctx, StageFilter{Status: StageStatusPending})
	if err != nil {
		return 0, err
	}
	store, err := e.stageStore()
	if err != nil {
		return 0, err
	}
	expired := 0
	for _, stage := range stages {
		if stage.ExpiresAt.IsZero() || !now.After(stage.ExpiresAt) {
			continue
		}
		stage.Status = StageStatusExpired
		if err := store.Update(ctx, stage); err != nil {
			return expired, err
		}
		expired++
	}
	return expired, nil
}

func (e *Engine) ApplyStage(ctx context.Context, req StageApplyRequest) (ApplyResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	stage, err := e.GetStage(ctx, req.ID)
	if err != nil {
		return ApplyResult{}, err
	}
	if stage.Status != StageStatusPending {
		return ApplyResult{}, fmt.Errorf("stage already %s", stage.Status)
	}

	store, err := e.stageStore()
	if err != nil {
		return ApplyResult{}, err
	}

	if !stage.ExpiresAt.IsZero() && time.Now().After(stage.ExpiresAt) {
		stage.Status = StageStatusExpired
		if err := store.Update(ctx, stage); err != nil {
			return ApplyResult{}, err
		}
		return ApplyResult{}, fmt.Errorf("stage expired")
	}

	path, err := newRootPolicy(e.cfg.AllowedRoots).ValidatePath(stage.Path)
	if err != nil {
		return ApplyResult{}, err
	}
	stage.Path = path

	if stage.BaseDigest != "" {
		current, err := securefs.ReadFile(path)
		if err != nil {
			return ApplyResult{}, err
		}
		if calculateSHA256(string(current)) != stage.BaseDigest {
			return ApplyResult{}, fmt.Errorf("file integrity check failed")
		}
	}

	writer := core.NewAtomicWriter(core.DefaultAtomicConfig())
	if err := writer.WriteFile(path, stage.Modified); err != nil {
		return ApplyResult{}, err
	}

	now := time.Now().UTC()
	stage.Status = StageStatusApplied
	stage.AppliedAt = &now
	if err := store.Update(ctx, stage); err != nil {
		return ApplyResult{}, err
	}

	return ApplyResult{
		StageID:     stage.ID,
		Applied:     true,
		AutoApplied: req.AutoApplied,
		Status:      stage.Status,
		AppliedAt:   stage.AppliedAt,
	}, nil
}

func calculateSHA256(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
