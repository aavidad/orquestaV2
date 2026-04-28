package artifactsapp

import (
	"path/filepath"
	"testing"
	"time"
)

func TestFileRepositoryPersistsArtifactsAsJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifacts.jsonl")
	taskID := int64(35)
	service := NewServiceWithClock(NewFileRepository(path), func() time.Time {
		return time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)
	})
	if _, err := service.Register(RegisterInput{
		Scope:   ScopeTask,
		Kind:    KindTests,
		TaskID:  &taskID,
		Agent:   "Codex1",
		Project: "orquesta",
		Path:    "artifacts/task-35/tests.txt",
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	reopened := NewService(NewFileRepository(path))
	items, err := reopened.List(ListFilter{Scope: ScopeTask, Kind: KindTests, TaskID: &taskID})
	if err != nil {
		t.Fatalf("List reopened: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items persistidos inesperados: %+v", items)
	}
	if items[0].Version != 1 || items[0].Path != "artifacts/task-35/tests.txt" {
		t.Fatalf("artifact persistido inesperado: %+v", items[0])
	}
}
