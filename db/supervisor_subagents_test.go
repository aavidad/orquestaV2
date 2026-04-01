package db

import (
	"strings"
	"testing"
	"time"
)

func TestSupervisorSubagentsPersistAndProfile(t *testing.T) {
	prepararDBTemporal(t)

	base := time.Now().UTC().Add(-2 * time.Minute)
	item, err := UpsertSupervisorSubagent(UpsertSupervisorSubagentInput{
		Supervisor:     "OpenClaw",
		ProyectoSlug:   "orquestador",
		SessionID:      "sess-1",
		ParentThreadID: "leader-1",
		ThreadID:       "sub-1",
		SubagentName:   "OpenClaw-Explore-1",
		SubagentType:   "Explore",
		ManifestPath:   "/tmp/sub-1/manifest.json",
		OutputPath:     "/tmp/sub-1/output.md",
		MetadataJSON:   `{"source":"spawn"}`,
		CreatedAt:      &base,
		StartedAt:      &base,
	})
	if err != nil {
		t.Fatalf("upsert supervisor subagent: %v", err)
	}
	if item == nil || item.SubagentType != "explore" || item.Status != "running" {
		t.Fatalf("subagent inesperado: %+v", item)
	}
	if !strings.Contains(item.ToolProfileJSON, `"subagent_type":"explore"`) {
		t.Fatalf("tool profile inesperado: %s", item.ToolProfileJSON)
	}

	doneAt := base.Add(30 * time.Second)
	item, err = UpsertSupervisorSubagent(UpsertSupervisorSubagentInput{
		Supervisor:     "OpenClaw",
		ProyectoSlug:   "orquestador",
		SessionID:      "sess-1",
		ParentThreadID: "leader-1",
		ThreadID:       "sub-1",
		SubagentName:   "OpenClaw-Explore-1",
		SubagentType:   "explore",
		Status:         "completed",
		CompletedAt:    &doneAt,
	})
	if err != nil {
		t.Fatalf("complete supervisor subagent: %v", err)
	}
	if item == nil || item.Status != "completed" || item.CompletedAt == nil {
		t.Fatalf("subagent completado inesperado: %+v", item)
	}

	items, err := ListarSupervisorSubagents(FiltroSupervisorSubagents{
		Supervisor: "OpenClaw",
		SessionID:  "sess-1",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("listar supervisor subagents: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("subagents inesperados: %+v", items)
	}
}

func TestSupervisorSubagentToolProfiles(t *testing.T) {
	profile := SupervisorSubagentToolProfileForType("Verification")
	if profile.SubagentType != "verification" {
		t.Fatalf("tipo normalizado inesperado: %+v", profile)
	}
	if len(profile.AllowedTools) == 0 {
		t.Fatalf("allowed tools vacio: %+v", profile)
	}
	items := ListSupervisorSubagentToolProfiles()
	if len(items) < 4 {
		t.Fatalf("profiles insuficientes: %+v", items)
	}
}
