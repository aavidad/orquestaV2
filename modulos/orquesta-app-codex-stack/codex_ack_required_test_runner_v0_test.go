package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestCodexAckRequiredTestRunnerV0MaterializaReceiptsComoEvidenciaDurable(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-codex-ack-required-test-001"
	taskRef := "task-ref-codex-ack-required-test-001"
	agentRef := "agent-ref-codex-ack-required-test-001"
	deliveryRef := "ack-ref-codex-ack-required-test-001"
	requiredTests := []string{
		"go test -count=1 ./modulos/orquesta-app-codex-stack",
		"validar contexto required ref_only mediante lectura local, consulta al director o evidencia explicita",
	}
	ackPath := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	ack := orquestaruntimecodex.CodexAgentAckV0{
		SchemaVersion: orquestaruntimecodex.CodexAgentAckSchemaVersionV0,
		RequestID:     agentRef,
		CorrelationID: "corr-codex-ack-required-test-001",
		AckRef:        deliveryRef,
		TargetModule:  "orquesta-app-stack-programacion",
		TaskRef:       taskRef,
		Status:        "completed",
		Tests:         orquestaruntimecodex.EvidenceListV0(requiredTests),
		TestReceipts:  codexStackRequiredTestReceiptsV0(requiredTests),
	}
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("Marshal ACK: %v", err)
	}
	if err := os.WriteFile(ackPath, data, 0o600); err != nil {
		t.Fatalf("Write ACK: %v", err)
	}
	descriptor := orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef: "descriptor-ref-codex-ack-required-test-001",
		RunID:         runRef,
		AgentRef:      agentRef,
		AckPath:       ackPath,
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			RequestID:     agentRef,
			CorrelationID: "corr-codex-ack-required-test-001",
			AgentPacket: orquestaruntime.AgentStartPacketV0{
				RequestID:     agentRef,
				CorrelationID: "corr-codex-ack-required-test-001",
				TargetModule:  "orquesta-app-stack-programacion",
				Task: orquestaruntime.AgentStartTaskV0{
					TaskRef:       taskRef,
					RequiredTests: requiredTests,
				},
				DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
					AckRef:       deliveryRef,
					MailboxRef:   "mailbox-ref-codex-ack-required-test-001",
					ReadinessRef: "readiness-ref-codex-ack-required-test-001",
				},
			},
		},
	}
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	runner := codexAckRequiredTestRunnerV0{
		ReceiptStore:   orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor),
		EvidenceReader: evidenceStore,
		EvidenceWriter: evidenceStore,
	}
	request := orquestacionnucleoapp.RequiredTestExecutionRequestV0{
		RunRef:            runRef,
		TaskRef:           taskRef,
		TestCommands:      requiredTests,
		DeliveryRef:       deliveryRef,
		ReviewRequestID:   "review-request-ref-codex-ack-required-test-001",
		ReviewResultRef:   "review-result-ref-codex-ack-required-test-001",
		AcceptedReviewRef: "accepted-review-ref-codex-ack-required-test-001",
		OccurredAt:        "2026-05-25T16:30:00Z",
		CorrelationID:     "corr-codex-ack-required-test-001",
		EvidenceRefs:      []string{"review-evidence-ref-codex-ack-required-test-001"},
	}

	result, err := runner.RunRequiredTestsV0(ctx, request)
	if err != nil {
		t.Fatalf("RunRequiredTestsV0: %v", err)
	}
	if len(result.EvidenceRefs) != len(requiredTests) ||
		len(result.PassedEvidenceRefs) != len(requiredTests) ||
		len(result.FailedEvidenceRefs) != 0 {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	evidence, err := evidenceStore.LoadRequiredTestEvidenceV0(ctx, runRef, result.EvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if len(evidence) != len(requiredTests) {
		t.Fatalf("evidence=%+v", evidence)
	}
	for index, item := range evidence {
		if item.Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 ||
			item.TaskRef != taskRef ||
			item.TestCommand != requiredTests[index] ||
			item.DeliveryRef != deliveryRef ||
			len(item.EvidenceRefs) != 1 {
			t.Fatalf("evidencia invalida[%d]=%+v", index, item)
		}
	}

	replayed, err := runner.RunRequiredTestsV0(ctx, request)
	if err != nil {
		t.Fatalf("RunRequiredTestsV0 replay: %v", err)
	}
	if len(replayed.EvidenceRefs) != len(result.EvidenceRefs) ||
		replayed.EvidenceRefs[0] != result.EvidenceRefs[0] {
		t.Fatalf("replay no idempotente: first=%+v second=%+v", result, replayed)
	}
}

func TestCodexAckRequiredTestRunnerV0DelegaSiNoHayReceiptsCompletos(t *testing.T) {
	ctx := context.Background()
	inner := &fakeCodexAckRequiredTestInnerRunnerV0{}
	runner := codexAckRequiredTestRunnerV0{
		Inner:          inner,
		ReceiptStore:   orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(),
		EvidenceWriter: orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(),
	}
	_, err := runner.RunRequiredTestsV0(ctx, orquestacionnucleoapp.RequiredTestExecutionRequestV0{
		RunRef:            "run-ref-codex-ack-required-test-delegate",
		TaskRef:           "task-ref-codex-ack-required-test-delegate",
		TestCommands:      []string{"go test ./..."},
		DeliveryRef:       "ack-ref-codex-ack-required-test-delegate",
		ReviewRequestID:   "review-request-ref-codex-ack-required-test-delegate",
		ReviewResultRef:   "review-result-ref-codex-ack-required-test-delegate",
		AcceptedReviewRef: "accepted-review-ref-codex-ack-required-test-delegate",
		OccurredAt:        "2026-05-25T16:31:00Z",
	})
	if err != nil {
		t.Fatalf("RunRequiredTestsV0: %v", err)
	}
	if !inner.called {
		t.Fatalf("no delego al runner interno")
	}
}

func TestCodexAckRequiredTestRunnerV0NoMaterializaACKLegacyComoEvidencia(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-codex-ack-required-test-legacy"
	taskRef := "task-ref-codex-ack-required-test-legacy"
	agentRef := "agent-ref-codex-ack-required-test-legacy"
	deliveryRef := "ack-ref-codex-ack-required-test-legacy"
	requiredTests := []string{"go test ./..."}
	ackPath := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	ack := orquestaruntimecodex.CodexAgentAckV0{
		SchemaVersion: orquestaruntimecodex.CodexAgentAckSchemaVersionV0,
		RequestID:     agentRef,
		CorrelationID: "corr-codex-ack-required-test-legacy",
		AckRef:        deliveryRef,
		TargetModule:  "orquesta-app-stack-programacion",
		TaskRef:       taskRef,
		Status:        "completed",
		Tests:         orquestaruntimecodex.EvidenceListV0(requiredTests),
	}
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("Marshal ACK: %v", err)
	}
	if err := os.WriteFile(ackPath, data, 0o600); err != nil {
		t.Fatalf("Write ACK: %v", err)
	}
	descriptor := orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef: "descriptor-ref-codex-ack-required-test-legacy",
		RunID:         runRef,
		AgentRef:      agentRef,
		AckPath:       ackPath,
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			RequestID:     agentRef,
			CorrelationID: "corr-codex-ack-required-test-legacy",
			AgentPacket: orquestaruntime.AgentStartPacketV0{
				RequestID:     agentRef,
				CorrelationID: "corr-codex-ack-required-test-legacy",
				TargetModule:  "orquesta-app-stack-programacion",
				Task: orquestaruntime.AgentStartTaskV0{
					TaskRef:       taskRef,
					RequiredTests: requiredTests,
				},
				DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
					AckRef: deliveryRef,
				},
			},
		},
	}
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	runner := codexAckRequiredTestRunnerV0{
		ReceiptStore:   orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor),
		EvidenceReader: evidenceStore,
		EvidenceWriter: evidenceStore,
	}

	result, err := runner.RunRequiredTestsV0(ctx, orquestacionnucleoapp.RequiredTestExecutionRequestV0{
		RunRef:            runRef,
		TaskRef:           taskRef,
		TestCommands:      requiredTests,
		DeliveryRef:       deliveryRef,
		ReviewRequestID:   "review-request-ref-codex-ack-required-test-legacy",
		ReviewResultRef:   "review-result-ref-codex-ack-required-test-legacy",
		AcceptedReviewRef: "accepted-review-ref-codex-ack-required-test-legacy",
		OccurredAt:        "2026-05-25T16:32:00Z",
	})
	if err != nil {
		t.Fatalf("RunRequiredTestsV0: %v", err)
	}
	if len(result.EvidenceRefs) != 0 || len(result.PassedEvidenceRefs) != 0 {
		t.Fatalf("ACK legacy no debe materializar evidencia: %+v", result)
	}
}

func TestCodexAckRequiredTestRunnerV0NoMaterializaACKNoCorreladoAunqueTraigaReceipts(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-codex-ack-required-test-mismatch"
	taskRef := "task-ref-codex-ack-required-test-mismatch"
	agentRef := "agent-ref-codex-ack-required-test-mismatch"
	deliveryRef := "ack-ref-codex-ack-required-test-mismatch"
	requiredTests := []string{"go test ./..."}
	ackPath := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	ack := orquestaruntimecodex.CodexAgentAckV0{
		SchemaVersion: orquestaruntimecodex.CodexAgentAckSchemaVersionV0,
		RequestID:     agentRef,
		CorrelationID: "corr-codex-ack-required-test-otro",
		AckRef:        deliveryRef,
		TargetModule:  "orquesta-app-stack-programacion",
		TaskRef:       taskRef,
		Status:        "completed",
		Tests:         orquestaruntimecodex.EvidenceListV0(requiredTests),
		TestReceipts:  codexStackRequiredTestReceiptsV0(requiredTests),
	}
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("Marshal ACK: %v", err)
	}
	if err := os.WriteFile(ackPath, data, 0o600); err != nil {
		t.Fatalf("Write ACK: %v", err)
	}
	descriptor := orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef: "descriptor-ref-codex-ack-required-test-mismatch",
		RunID:         runRef,
		AgentRef:      agentRef,
		AckPath:       ackPath,
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			RequestID:     agentRef,
			CorrelationID: "corr-codex-ack-required-test-mismatch",
			AgentPacket: orquestaruntime.AgentStartPacketV0{
				RequestID:     agentRef,
				CorrelationID: "corr-codex-ack-required-test-mismatch",
				TargetModule:  "orquesta-app-stack-programacion",
				Task: orquestaruntime.AgentStartTaskV0{
					TaskRef:       taskRef,
					RequiredTests: requiredTests,
				},
				DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
					AckRef: deliveryRef,
				},
			},
		},
	}
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	runner := codexAckRequiredTestRunnerV0{
		ReceiptStore:   orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor),
		EvidenceReader: evidenceStore,
		EvidenceWriter: evidenceStore,
	}

	result, err := runner.RunRequiredTestsV0(ctx, orquestacionnucleoapp.RequiredTestExecutionRequestV0{
		RunRef:            runRef,
		TaskRef:           taskRef,
		TestCommands:      requiredTests,
		DeliveryRef:       deliveryRef,
		ReviewRequestID:   "review-request-ref-codex-ack-required-test-mismatch",
		ReviewResultRef:   "review-result-ref-codex-ack-required-test-mismatch",
		AcceptedReviewRef: "accepted-review-ref-codex-ack-required-test-mismatch",
		OccurredAt:        "2026-05-25T16:33:00Z",
		CorrelationID:     "corr-codex-ack-required-test-mismatch",
	})
	if err != nil {
		t.Fatalf("RunRequiredTestsV0: %v", err)
	}
	if len(result.EvidenceRefs) != 0 || len(result.PassedEvidenceRefs) != 0 {
		t.Fatalf("ACK no correlado no debe materializar evidencia: %+v", result)
	}
}

type fakeCodexAckRequiredTestInnerRunnerV0 struct {
	called bool
}

func (runner *fakeCodexAckRequiredTestInnerRunnerV0) RunRequiredTestsV0(
	context.Context,
	orquestacionnucleoapp.RequiredTestExecutionRequestV0,
) (orquestacionnucleoapp.RequiredTestExecutionResultV0, error) {
	runner.called = true
	return orquestacionnucleoapp.RequiredTestExecutionResultV0{}, nil
}
