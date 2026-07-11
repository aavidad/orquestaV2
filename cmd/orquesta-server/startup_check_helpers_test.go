package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func resultStartupEvidenceContainsV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func writeStartupJSONForTestV0(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readStartupQueueForTestV0(t *testing.T, path string) startupQueueSnapshotV0 {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var snapshot startupQueueSnapshotV0
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return snapshot
}

func readStartupControlForTestV0(t *testing.T, path string) startupControlSnapshotV0 {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var snapshot startupControlSnapshotV0
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return snapshot
}

type startupStrictAckFixtureV0 struct {
	RunRef   string
	AgentRef string
}

func writeStartupStrictCompletedAckForTestV0(
	t *testing.T,
	agentDir string,
	fixture startupStrictAckFixtureV0,
) {
	t.Helper()
	packet := writeStartupAgentPacketForTestV0(t, agentDir, fixture)
	ack := orquestaruntimecodex.CodexAgentAckV0{
		SchemaVersion: orquestaruntimecodex.CodexAgentAckSchemaVersionV0,
		RequestID:     packet.RequestID,
		CorrelationID: packet.CorrelationID,
		AckRef:        packet.DeliveryRefs.AckRef,
		TargetModule:  packet.TargetModule,
		TaskRef:       packet.Task.TaskRef,
		Status:        "completed",
		Files:         orquestaruntimecodex.EvidenceListV0{"README.md"},
		Tests:         orquestaruntimecodex.EvidenceListV0{"go test ./..."},
		TestReceipts: []orquestaruntimecodex.CodexRequiredTestReceiptV0{{
			SchemaVersion:  orquestaruntimecodex.CodexRequiredTestReceiptSchemaVersionV0,
			Command:        "go test ./...",
			Status:         "passed",
			ExitCode:       startupIntPtrForTestV0(0),
			EvidenceRefs:   []string{"required-test-receipt-ref-startup"},
			OccurredAt:     "2026-05-24T10:00:00Z",
			Sequence:       1,
			OutputRedacted: startupBoolPtrForTestV0(true),
		}},
	}
	writeStartupJSONForTestV0(t, filepath.Join(agentDir, "agent_ack.json"), ack)
}

func startupIntPtrForTestV0(value int) *int {
	return &value
}

func startupBoolPtrForTestV0(value bool) *bool {
	return &value
}

func writeStartupAgentPacketForTestV0(
	t *testing.T,
	agentDir string,
	fixture startupStrictAckFixtureV0,
) orquestaruntime.AgentStartPacketV0 {
	t.Helper()
	packet := orquestaruntime.AgentStartPacketV0{
		SchemaVersion: orquestaruntime.AgentStartPacketSchemaVersionV0,
		RequestID:     fixture.AgentRef,
		CorrelationID: "corr-" + fixture.RunRef,
		WorkOrderRef:  "work-order-" + fixture.RunRef,
		TargetModule:  "orquesta-app-stack-programacion",
		Phase:         "programacion",
		Task: orquestaruntime.AgentStartTaskV0{
			TaskRef:       "task-" + fixture.RunRef,
			WriteSet:      []string{"README.md"},
			RequiredTests: []string{"go test ./..."},
		},
		DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
			AckRef: "ack-" + fixture.RunRef,
		},
	}
	writeStartupJSONForTestV0(t, filepath.Join(agentDir, "agent_packet.json"), packet)
	return packet
}
