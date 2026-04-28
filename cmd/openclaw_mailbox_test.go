package cmd

import (
	"testing"
	"time"

	"orquesta/db"
)

func TestOpenClawMailboxSummarizePayload(t *testing.T) {
	action, context := summarizeOpenClawMailboxPayload(`{"supervisor_action":"revisar_worktree_desfasada","carril":"premium_worktree","tarea_objetivo_id":604,"worktree_id":88,"write_set":["cmd/controlplane_support.go","db/controlplane_entities.go"]}`)
	if action != "revisar_worktree_desfasada" {
		t.Fatalf("accion inesperada: %q", action)
	}
	want := "premium_worktree · tarea#604 · wt#88 · cmd/controlplane_support.go (+1)"
	if context != want {
		t.Fatalf("contexto inesperado: got=%q want=%q", context, want)
	}
}

func TestBuildOpenClawMailboxLiteFromItemsAggregatesContexts(t *testing.T) {
	now := time.Now().UTC()
	items := []*db.RuntimeMailboxMessage{
		{
			ToAgente:    "Codex3",
			Kind:        "autonomia",
			PayloadJSON: `{"supervisor_action":"revisar_worktree_desfasada","carril":"premium_worktree","tarea_objetivo_id":604,"worktree_id":88,"write_set":["cmd/controlplane_support.go","db/controlplane_entities.go"]}`,
			CreatedAt:   now.Add(-5 * time.Minute),
		},
		{
			ToAgente:    "Codex3",
			Kind:        "nudge",
			PayloadJSON: `{"carril":"premium_worktree","tarea_objetivo_id":604,"worktree_id":88,"write_set":["cmd/controlplane_support.go","db/controlplane_entities.go"]}`,
			CreatedAt:   now.Add(-2 * time.Minute),
		},
	}
	out := buildOpenClawMailboxLiteFromItems(items, map[string]struct{}{"codex3": {}})
	if len(out) != 1 {
		t.Fatalf("mailbox inesperada: %#v", out)
	}
	if out[0].Count != 2 || out[0].SupervisorActionsCSV != "revisar_worktree_desfasada" {
		t.Fatalf("agregacion inesperada: %+v", out[0])
	}
	want := "premium_worktree · tarea#604 · wt#88 · cmd/controlplane_support.go (+1)"
	if len(out[0].Contexts) != 1 || out[0].ContextsCSV != want {
		t.Fatalf("contextos inesperados: %+v", out[0])
	}
	if out[0].OldestCreatedAt == nil || out[0].OldestCreatedAt.After(now.Add(-4*time.Minute)) {
		t.Fatalf("oldest_created_at inesperado: %+v", out[0])
	}
}

func TestOpenClawMailboxSummarizePayloadReconoceTareaIDCanonica(t *testing.T) {
	action, context := summarizeOpenClawMailboxPayload(`{"supervisor_action":"continuar_trabajo","carril":"premium_worktree","tarea_id":41,"worktree_id":7}`)
	if action != "continuar_trabajo" {
		t.Fatalf("accion inesperada: %q", action)
	}
	want := "premium_worktree · tarea#41 · wt#7"
	if context != want {
		t.Fatalf("contexto inesperado para tarea_id: got=%q want=%q", context, want)
	}
}
