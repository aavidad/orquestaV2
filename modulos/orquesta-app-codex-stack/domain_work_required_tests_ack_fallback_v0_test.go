package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestDomainWorkRequiredTestRunnerV0DelegaAAckRunnerSinSubmissionLedger(t *testing.T) {
	ctx := context.Background()
	record := codexStackDomainWorkPolicyRecordForTestV0("ack-fallback")
	request := codexStackDomainWorkRequiredTestRequestForTestV0(record, "ack-fallback")
	ackPath := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	agentRef := "agent-ref-domain-work-ack-fallback"
	ack := orquestaruntimecodex.CodexAgentAckV0{
		SchemaVersion: orquestaruntimecodex.CodexAgentAckSchemaVersionV0,
		RequestID:     agentRef,
		CorrelationID: request.CorrelationID,
		AckRef:        request.DeliveryRef,
		TargetModule:  "orquesta-app-stack-programacion",
		TaskRef:       request.TaskRef,
		Status:        "completed",
		Tests:         orquestaruntimecodex.EvidenceListV0(request.TestCommands),
		TestReceipts:  codexStackRequiredTestReceiptsV0(request.TestCommands),
	}
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("Marshal ACK: %v", err)
	}
	if err := os.WriteFile(ackPath, data, 0o600); err != nil {
		t.Fatalf("Write ACK: %v", err)
	}
	descriptor := orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef: "descriptor-ref-domain-work-ack-fallback",
		RunID:         request.RunRef,
		AgentRef:      agentRef,
		AckPath:       ackPath,
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			RequestID:     agentRef,
			CorrelationID: request.CorrelationID,
			AgentPacket: orquestaruntime.AgentStartPacketV0{
				RequestID:     agentRef,
				CorrelationID: request.CorrelationID,
				TargetModule:  "orquesta-app-stack-programacion",
				Task: orquestaruntime.AgentStartTaskV0{
					TaskRef:       request.TaskRef,
					RequiredTests: request.TestCommands,
				},
				DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
					AckRef: request.DeliveryRef,
				},
			},
		},
	}
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	ackRunner := codexAckRequiredTestRunnerV0{
		ReceiptStore:   orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor),
		EvidenceReader: evidenceStore,
		EvidenceWriter: evidenceStore,
	}
	runner := DomainWorkRequiredTestRunnerV0{
		Inner:          ackRunner,
		Policy:         orquestadomainwork.DeclaredDomainWorkRequiredTestPolicyV0{},
		AppChangeStore: orquestaappchange.NewInMemoryAppChangeStoreV0(record),
		EvidenceReader: evidenceStore,
		EvidenceWriter: evidenceStore,
	}

	result, err := runner.RunRequiredTestsV0(ctx, request)
	if err != nil {
		t.Fatalf("RunRequiredTestsV0: %v", err)
	}
	if len(result.PassedEvidenceRefs) != len(request.TestCommands) || len(result.FailedEvidenceRefs) != 0 {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	evidence, err := evidenceStore.LoadRequiredTestEvidenceV0(ctx, request.RunRef, result.EvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if len(evidence) != len(request.TestCommands) ||
		evidence[0].TaskRef != request.TaskRef ||
		evidence[0].DeliveryRef != request.DeliveryRef ||
		evidence[0].AcceptedReviewRef != request.AcceptedReviewRef {
		t.Fatalf("evidencia no causal: %+v", evidence)
	}
}
