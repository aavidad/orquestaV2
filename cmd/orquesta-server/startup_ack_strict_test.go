package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestCompactStartupStateFilesV0NoArchivaACKMinimoNiSalidaTerminalV0(t *testing.T) {
	stateDir := t.TempDir()
	runtimeDir := t.TempDir()
	runStateDir := filepath.Join(stateDir, "run-state")
	if err := os.MkdirAll(runStateDir, 0o755); err != nil {
		t.Fatalf("mkdir run-state: %v", err)
	}
	queue := startupQueueSnapshotV0{
		SchemaVersion: "orquesta.run_file.queue.v0",
		Records: []startupQueueRecordV0{{
			RunRef:   "run-ready-minimal-ack",
			QueueRef: "global",
			Candidate: orquestarunqueue.RunSchedulingCandidateV0{
				RunRef: "run-ready-minimal-ack",
				Status: "ready",
			},
		}},
	}
	writeStartupJSONForTestV0(t, filepath.Join(runStateDir, "queue_v0.json"), queue)
	writeStartupJSONForTestV0(t, filepath.Join(runStateDir, "control_v0.json"), startupControlSnapshotV0{})
	agentDir := filepath.Join(runtimeDir, "run-ready-minimal-ack", "agent-1")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent: %v", err)
	}
	writeStartupAgentPacketForTestV0(t, agentDir, startupStrictAckFixtureV0{
		RunRef:   "run-ready-minimal-ack",
		AgentRef: "agent-1",
	})
	if err := os.WriteFile(filepath.Join(agentDir, "agent_ack.json"), []byte(`{"schema_version":"codex_agent_ack.v0","status":"completed"}`), 0o644); err != nil {
		t.Fatalf("write ack: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "codex_last_message.txt"), []byte("ACK ack-run-ready-minimal-ack completed"), 0o644); err != nil {
		t.Fatalf("write last message: %v", err)
	}

	check := serverStartupCheckV0{ServerConfig: orquestaserver.ConfigV0{
		StateDir:       stateDir,
		RuntimeWorkDir: runtimeDir,
	}}
	got, err := check.compactStartupStateFilesV0(orquestaserver.StartupCheckCommandV0{
		StateDir:       stateDir,
		RuntimeWorkDir: runtimeDir,
		OccurredAt:     time.Date(2026, 5, 18, 15, 31, 31, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("compactStartupStateFilesV0: %v", err)
	}
	if got.CompactionNeeded || got.QueueRemoved != 0 {
		t.Fatalf("compaction=%+v", got)
	}
	queueAfter := readStartupQueueForTestV0(t, filepath.Join(runStateDir, "queue_v0.json"))
	if len(queueAfter.Records) != 1 || queueAfter.Records[0].RunRef != "run-ready-minimal-ack" {
		t.Fatalf("queue after=%+v", queueAfter)
	}
}
