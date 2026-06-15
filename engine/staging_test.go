package engine

import "testing"

func TestStageStoreCreatesAndReadsStage(t *testing.T) {
	dir := t.TempDir()
	store := NewStageStore(dir)

	stage, err := store.Create(StageCreateRequest{
		Path:     "main.go",
		Original: "a",
		Modified: "b",
		Diff:     "@@",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if stage.ID == "" {
		t.Fatalf("expected stage id")
	}

	read, err := store.Read(stage.ID)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if read.Modified != "b" {
		t.Fatalf("unexpected modified: %q", read.Modified)
	}
}
