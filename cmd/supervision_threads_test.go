package cmd

import (
	"testing"

	"orquesta/db"
)

func TestBuildSupervisorThreadsSnapshotOmiteThreadsCrudos(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if _, err := registrarSupervisorThreadLigero("OpenClaw", "orquestador", "sess-1", "leader-1", "leader", "", "active", "server", "turn-1"); err != nil {
			t.Fatalf("registrar leader: %v", err)
		}
		if _, err := registrarSupervisorThreadLigero("OpenClaw", "orquestador", "sess-1", "sub-1", "subagent", "", "active", "server", "turn-2"); err != nil {
			t.Fatalf("registrar subagent: %v", err)
		}

		snapshot, err := buildSupervisorThreadsSnapshot("OpenClaw", "", 10)
		if err != nil {
			t.Fatalf("buildSupervisorThreadsSnapshot: %v", err)
		}
		if _, ok := snapshot["threads"]; ok {
			t.Fatalf("thread_sessions no deberia exponer threads crudos: %#v", snapshot)
		}
		sessions, _ := snapshot["sessions"].([]*db.SupervisorThreadSessionSummary)
		if len(sessions) != 1 || sessions[0] == nil {
			t.Fatalf("sessions inesperadas: %#v", snapshot["sessions"])
		}
		if len(sessions[0].Threads) != 0 {
			t.Fatalf("la sesion no deberia arrastrar threads crudos: %#v", sessions[0].Threads)
		}
	})
}
