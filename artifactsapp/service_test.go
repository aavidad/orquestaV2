package artifactsapp

import (
	"testing"
	"time"
)

func TestRegisterAssignsVersionedTaskArtifacts(t *testing.T) {
	taskID := int64(35)
	repo := NewMemoryRepository()
	clock := time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)
	service := NewServiceWithClock(repo, func() time.Time { return clock })

	first, err := service.Register(RegisterInput{
		Scope:   ScopeTask,
		Kind:    KindPatch,
		TaskID:  &taskID,
		Agent:   " Codex1 ",
		Project: " orquesta ",
		Path:    " artifacts/task-35/patch.diff ",
	})
	if err != nil {
		t.Fatalf("Register first: %v", err)
	}
	second, err := service.Register(RegisterInput{
		Scope:       ScopeTask,
		Kind:        KindPatch,
		TaskID:      &taskID,
		Agent:       "Codex1",
		Project:     "orquesta",
		BlobRef:     "sha256:abc",
		ContentType: "text/x-diff",
	})
	if err != nil {
		t.Fatalf("Register second: %v", err)
	}

	if first.Version != 1 || second.Version != 2 {
		t.Fatalf("versiones inesperadas: first=%d second=%d", first.Version, second.Version)
	}
	if first.ID == "" || second.ID == "" || first.ID == second.ID {
		t.Fatalf("ids inesperados: first=%q second=%q", first.ID, second.ID)
	}
	if first.Agent != "Codex1" || first.Project != "orquesta" {
		t.Fatalf("vinculos normalizados inesperados: %+v", first)
	}
	if first.ContentType != "text/x-diff" {
		t.Fatalf("content type por defecto inesperado: %s", first.ContentType)
	}
}

func TestListFiltersArtifactsByTaskAgentProjectAndKind(t *testing.T) {
	task35 := int64(35)
	task36 := int64(36)
	repo := NewMemoryRepository()
	service := NewServiceWithClock(repo, func() time.Time {
		return time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)
	})
	mustRegister(t, service, RegisterInput{Scope: ScopeTask, Kind: KindPatch, TaskID: &task35, Agent: "Codex1", Project: "orquesta", Path: "patch.diff"})
	mustRegister(t, service, RegisterInput{Scope: ScopeTask, Kind: KindTests, TaskID: &task35, Agent: "Codex1", Project: "orquesta", Path: "tests.txt"})
	mustRegister(t, service, RegisterInput{Scope: ScopeTask, Kind: KindPatch, TaskID: &task36, Agent: "Codex1", Project: "orquesta", Path: "other.diff"})
	mustRegister(t, service, RegisterInput{Scope: ScopeProject, Kind: KindTranscript, Project: "orquesta", Path: "transcript.txt"})

	items, err := service.List(ListFilter{
		Scope:   ScopeTask,
		Kind:    KindPatch,
		TaskID:  &task35,
		Agent:   "Codex1",
		Project: "orquesta",
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items inesperados: %+v", items)
	}
	if items[0].Kind != KindPatch || items[0].TaskID == nil || *items[0].TaskID != task35 {
		t.Fatalf("artifact inesperado: %+v", items[0])
	}
}

func TestRegisterValidatesMinimalLinks(t *testing.T) {
	service := NewService(NewMemoryRepository())
	if _, err := service.Register(RegisterInput{Scope: ScopeTask, Kind: KindPatch, Path: "patch.diff"}); err == nil {
		t.Fatal("se esperaba error sin task_id")
	}
	taskID := int64(35)
	if _, err := service.Register(RegisterInput{Scope: ScopeTask, Kind: KindPatch, TaskID: &taskID}); err == nil {
		t.Fatal("se esperaba error sin path ni blob_ref")
	}
	if _, err := service.Register(RegisterInput{Scope: ScopeAgent, Kind: KindTranscript, Agent: "Codex1", Path: "transcript.txt"}); err != nil {
		t.Fatalf("artifact de agente deberia ser valido: %v", err)
	}
}

func mustRegister(t *testing.T, service *Service, input RegisterInput) {
	t.Helper()
	if _, err := service.Register(input); err != nil {
		t.Fatalf("Register(%+v): %v", input, err)
	}
}
