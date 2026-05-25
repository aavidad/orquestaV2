package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestCodexReceiptDirectorDecisionFileDescriptorProviderV0RegistraReceiptSidecar(t *testing.T) {
	store, provider, request, _ := directorDecisionSidecarFixtureV0(t)

	got, err := provider.ListDirectorAgentDecisionFilesV0(context.Background(), request)
	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionFilesV0: %v", err)
	}
	if len(got) != 1 || got[0].SidecarReceipt == nil {
		t.Fatalf("descriptor=%+v", got)
	}
	if got[0].SidecarReceipt.Status != "pending" ||
		got[0].SidecarReceipt.ProducerAckRef != "ack-ref-director-001" ||
		got[0].SidecarReceipt.AgentRef != "agent-director-001" ||
		got[0].SidecarReceipt.CorrelationID != "corr-sidecar-001" ||
		got[0].SidecarReceipt.SHA256 == "" {
		t.Fatalf("sidecar_receipt=%+v", got[0].SidecarReceipt)
	}
	descriptors, err := store.ListCodexReceiptDescriptorsV0(
		context.Background(),
		CodexReceiptDescriptorRequestV0{RunID: "run-ref-001"},
	)
	if err != nil {
		t.Fatalf("ListCodexReceiptDescriptorsV0: %v", err)
	}
	if descriptors[0].DirectorDecisionSidecarReceipt == nil ||
		descriptors[0].DirectorDecisionSidecarReceipt.ReceiptRef != got[0].SidecarReceipt.ReceiptRef {
		t.Fatalf("store descriptor=%+v", descriptors[0])
	}
}

func TestCodexReceiptDirectorDecisionFileDescriptorProviderV0OmiteSidecarConsumido(t *testing.T) {
	store, provider, request, _ := directorDecisionSidecarFixtureV0(t)
	got, err := provider.ListDirectorAgentDecisionFilesV0(context.Background(), request)
	if err != nil || len(got) != 1 || got[0].SidecarReceipt == nil {
		t.Fatalf("primer list got=%+v err=%v", got, err)
	}
	consumed := *got[0].SidecarReceipt
	consumed.Status = "consumed"
	if err := store.RecordDirectorAgentDecisionFileConsumptionV0(context.Background(), consumed); err != nil {
		t.Fatalf("RecordDirectorAgentDecisionFileConsumptionV0: %v", err)
	}
	got, err = provider.ListDirectorAgentDecisionFilesV0(context.Background(), request)
	if err != nil {
		t.Fatalf("segundo list: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("descriptors consumidos=%+v", got)
	}
}

func TestCodexReceiptDirectorDecisionFileDescriptorProviderV0BloqueaSidecarConsumidoMutado(t *testing.T) {
	store, provider, request, decisionPath := directorDecisionSidecarFixtureV0(t)
	got, err := provider.ListDirectorAgentDecisionFilesV0(context.Background(), request)
	if err != nil || len(got) != 1 || got[0].SidecarReceipt == nil {
		t.Fatalf("primer list got=%+v err=%v", got, err)
	}
	consumed := *got[0].SidecarReceipt
	consumed.Status = "consumed"
	if err := store.RecordDirectorAgentDecisionFileConsumptionV0(context.Background(), consumed); err != nil {
		t.Fatalf("RecordDirectorAgentDecisionFileConsumptionV0: %v", err)
	}
	if err := os.WriteFile(decisionPath, []byte(`[{"schema_version":"mutado"}]`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err = provider.ListDirectorAgentDecisionFilesV0(context.Background(), request)
	if err == nil || !strings.Contains(err.Error(), "sidecar_receipt_conflict") {
		t.Fatalf("err=%v want sidecar_receipt_conflict", err)
	}
}

func directorDecisionSidecarFixtureV0(
	t *testing.T,
) (*InMemoryCodexReceiptDescriptorStoreV0, CodexReceiptDirectorDecisionFileDescriptorProviderV0, orquestadirectoragentfilesource.DirectorAgentDecisionFileListRequestV0, string) {
	t.Helper()
	baseDir := t.TempDir()
	ackPath := filepath.Join(baseDir, "run-ref-001", "agent-director-001", "agent_ack.json")
	writeCodexReceiptDecisionFilesForTestV0(t, ackPath, DefaultDirectorAgentDecisionFileNameV0)
	store := NewInMemoryCodexReceiptDescriptorStoreV0(CodexReceiptDescriptorV0{
		DescriptorRef: "codex-receipt-ref-director-001",
		RunID:         "run-ref-001",
		AgentRef:      "agent-director-001",
		AckPath:       ackPath,
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			RequestID: "agent-director-001",
			AgentPacket: orquestaruntime.AgentStartPacketV0{
				DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{AckRef: "ack-ref-director-001"},
			},
		},
	})
	request := orquestadirectoragentfilesource.DirectorAgentDecisionFileListRequestV0{
		RunID:          "run-ref-001",
		PhaseArtifacts: []string{"ack-ref-director-001#phase:brainstorming_arquitectura"},
		CorrelationID:  "corr-sidecar-001",
	}
	provider := CodexReceiptDirectorDecisionFileDescriptorProviderV0{Store: store}
	decisionPath := filepath.Join(filepath.Dir(ackPath), DefaultDirectorAgentDecisionFileNameV0)
	return store, provider, request, decisionPath
}
