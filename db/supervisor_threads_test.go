package db

import (
	"testing"
	"time"
)

func TestRecordSupervisorThreadTurnYSummary(t *testing.T) {
	prepararDBTemporal(t)
	t.Run("summary", func(t *testing.T) {
		base := time.Now().UTC().Add(-time.Minute)
		if _, err := RecordSupervisorThreadTurn(RecordSupervisorThreadInput{
			Supervisor:   "OpenClaw",
			ProyectoSlug: "orquestador",
			SessionID:    "sess-1",
			ThreadID:     "leader-1",
			Kind:         "leader",
			Mode:         "review",
			Source:       "mcp_prompt",
			TurnID:       "turn-1",
			Timestamp:    base,
		}); err != nil {
			t.Fatalf("record leader: %v", err)
		}
		if _, err := RecordSupervisorThreadTurn(RecordSupervisorThreadInput{
			Supervisor:   "OpenClaw",
			ProyectoSlug: "orquestador",
			SessionID:    "sess-1",
			ThreadID:     "sub-1",
			Kind:         "subagent",
			Mode:         "review",
			Source:       "mcp_tool",
			TurnID:       "turn-2",
			Timestamp:    base.Add(10 * time.Second),
		}); err != nil {
			t.Fatalf("record subagent: %v", err)
		}
		if _, err := RecordSupervisorThreadTurn(RecordSupervisorThreadInput{
			Supervisor:   "OpenClaw",
			ProyectoSlug: "orquestador",
			SessionID:    "sess-1",
			ThreadID:     "sub-1",
			Kind:         "subagent",
			Mode:         "review",
			Status:       "idle",
			Source:       "mcp_tool",
			TurnID:       "turn-3",
			Timestamp:    base.Add(20 * time.Second),
		}); err != nil {
			t.Fatalf("record subagent turn 2: %v", err)
		}
		items, err := ListarSupervisorThreads(FiltroSupervisorThreads{Supervisor: "OpenClaw", SessionID: "sess-1"})
		if err != nil {
			t.Fatalf("listar supervisor_threads: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("hilos inesperados: %+v", items)
		}
		var sub *SupervisorThread
		for _, item := range items {
			if item != nil && item.ThreadID == "sub-1" {
				sub = item
			}
		}
		if sub == nil || sub.TurnCount != 2 || sub.Status != "idle" {
			t.Fatalf("subagent inesperado: %+v", sub)
		}
		summary, err := SummarizeSupervisorThreadSession("OpenClaw", "sess-1", 2*time.Minute)
		if err != nil {
			t.Fatalf("summary: %v", err)
		}
		if summary == nil || summary.LeaderThreadID != "leader-1" {
			t.Fatalf("leader inesperado: %+v", summary)
		}
		if len(summary.AllSubagentThreadIDs) != 1 || summary.AllSubagentThreadIDs[0] != "sub-1" {
			t.Fatalf("subagents inesperados: %+v", summary)
		}
		if len(summary.ActiveSubagentThreadIDs) != 1 || summary.ActiveSubagentThreadIDs[0] != "sub-1" {
			t.Fatalf("activos inesperados: %+v", summary)
		}
	})
}
