package engine

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/oxhq/morfx/internal/securefs"
)

type StageStore struct {
	dir string
}

type StageCreateRequest struct {
	Path     string
	Original string
	Modified string
	Diff     string
}

type Stage struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Path      string    `json:"path"`
	Original  string    `json:"original"`
	Modified  string    `json:"modified"`
	Diff      string    `json:"diff"`
}

func NewStageStore(dir string) *StageStore {
	_ = securefs.MkdirAll(dir, 0o700)
	return &StageStore{dir: dir}
}

func (s *StageStore) Create(req StageCreateRequest) (Stage, error) {
	id, err := newStageID()
	if err != nil {
		return Stage{}, err
	}
	stage := Stage{
		ID:        id,
		CreatedAt: time.Now(),
		Path:      req.Path,
		Original:  req.Original,
		Modified:  req.Modified,
		Diff:      req.Diff,
	}
	b, err := json.Marshal(stage)
	if err != nil {
		return Stage{}, err
	}
	if err := securefs.WriteFile(filepath.Join(s.dir, id+".json"), b, 0o600); err != nil {
		return Stage{}, err
	}
	return stage, nil
}

func (s *StageStore) Read(id string) (Stage, error) {
	b, err := securefs.ReadFile(filepath.Join(s.dir, id+".json"))
	if err != nil {
		return Stage{}, err
	}
	var stage Stage
	if err := json.Unmarshal(b, &stage); err != nil {
		return Stage{}, err
	}
	return stage, nil
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
