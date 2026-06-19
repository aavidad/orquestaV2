package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
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
			item.OccurredAt != "2026-05-24T10:00:00Z" ||
			len(item.EvidenceRefs) != 1 {
			t.Fatalf("evidencia invalida[%d]=%+v", index, item)
		}
	}

	replayRequest := request
	replayRequest.OccurredAt = "2026-05-25T17:45:00Z"
	replayed, err := runner.RunRequiredTestsV0(ctx, replayRequest)
	if err != nil {
		t.Fatalf("RunRequiredTestsV0 replay: %v", err)
	}
	if len(replayed.EvidenceRefs) != len(result.EvidenceRefs) ||
		replayed.EvidenceRefs[0] != result.EvidenceRefs[0] {
		t.Fatalf("replay no idempotente: first=%+v second=%+v", result, replayed)
	}
}

func TestCodexAckRequiredTestRunnerV0MaterializaContextoRefOnlyDesdeNotaSinReceipt(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-codex-ack-required-test-ref-only"
	taskRef := "task-ref-codex-ack-required-test-ref-only"
	agentRef := "agent-ref-codex-ack-required-test-ref-only"
	deliveryRef := "ack-ref-codex-ack-required-test-ref-only"
	refOnlyCommand := externalContextRefOnlyRequiredTestV0
	requiredTests := []string{"go test ./...", refOnlyCommand}
	ackPath := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	ack := orquestaruntimecodex.CodexAgentAckV0{
		SchemaVersion: orquestaruntimecodex.CodexAgentAckSchemaVersionV0,
		RequestID:     agentRef,
		CorrelationID: "corr-codex-ack-required-test-ref-only",
		AckRef:        deliveryRef,
		TargetModule:  "orquesta-app-stack-programacion",
		TaskRef:       taskRef,
		Status:        "completed",
		Tests:         orquestaruntimecodex.EvidenceListV0(requiredTests),
		TestReceipts:  codexStackRequiredTestReceiptsV0([]string{"go test ./..."}),
		Notes:         orquestaruntimecodex.EvidenceListV0{"contexto_ref_only_resuelto: agent_packet context ref_only validado"},
	}
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("Marshal ACK: %v", err)
	}
	if err := os.WriteFile(ackPath, data, 0o600); err != nil {
		t.Fatalf("Write ACK: %v", err)
	}
	descriptor := orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef: "descriptor-ref-codex-ack-required-test-ref-only",
		RunID:         runRef,
		AgentRef:      agentRef,
		AckPath:       ackPath,
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			RequestID:     agentRef,
			CorrelationID: "corr-codex-ack-required-test-ref-only",
			AgentPacket: orquestaruntime.AgentStartPacketV0{
				RequestID:     agentRef,
				CorrelationID: "corr-codex-ack-required-test-ref-only",
				TargetModule:  "orquesta-app-stack-programacion",
				Task: orquestaruntime.AgentStartTaskV0{
					TaskRef:       taskRef,
					RequiredTests: requiredTests,
				},
				Context: orquestacontext.ContextMaterializedBundleV0{
					SchemaVersion: orquestacontext.ContextMaterializedBundleSchemaVersionV0,
					BundleRef:     "bundle-ref-codex-ack-required-test-ref-only",
					WorkOrderRef:  taskRef,
					TargetModule:  "orquesta-app-stack-programacion",
					Entries: []orquestacontext.ContextMaterializedEntryV0{{
						EntryRef:          "entry-ref-codex-ack-required-test-ref-only",
						Layer:             orquestacontext.ContextLayerTaskContextV0,
						Kind:              orquestacontext.ContextEntryDocRefV0,
						SourceRef:         "source-ref-codex-ack-required-test-ref-only",
						Mode:              orquestacontext.ContextMaterializationModeRefOnlyV0,
						Required:          true,
						RefOnlyReason:     orquestacontext.ContextRefOnlyReasonMaterializationMissingV0,
						RequiredRefAction: orquestacontext.ContextRequiredRefActionAckEvidenceV0,
					}},
				},
				DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
					AckRef:       deliveryRef,
					MailboxRef:   "mailbox-ref-codex-ack-required-test-ref-only",
					ReadinessRef: "readiness-ref-codex-ack-required-test-ref-only",
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
		ReviewRequestID:   "review-request-ref-codex-ack-required-test-ref-only",
		ReviewResultRef:   "review-result-ref-codex-ack-required-test-ref-only",
		AcceptedReviewRef: "accepted-review-ref-codex-ack-required-test-ref-only",
		OccurredAt:        "2026-05-25T16:35:00Z",
		CorrelationID:     "corr-codex-ack-required-test-ref-only",
		EvidenceRefs:      []string{"review-evidence-ref-codex-ack-required-test-ref-only"},
	}

	result, err := runner.RunRequiredTestsV0(ctx, request)
	if err != nil {
		t.Fatalf("RunRequiredTestsV0: %v", err)
	}
	if len(result.EvidenceRefs) != len(requiredTests) ||
		len(result.PassedEvidenceRefs) != len(requiredTests) {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	evidence, err := evidenceStore.LoadRequiredTestEvidenceV0(ctx, runRef, result.EvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	foundRefOnly := false
	for _, item := range evidence {
		if item.TestCommand == refOnlyCommand {
			foundRefOnly = item.Status == orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 &&
				len(item.EvidenceRefs) > 0
		}
	}
	if !foundRefOnly {
		t.Fatalf("no materializo evidencia contextual ref_only: %+v", evidence)
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

func TestCodexAckRequiredTestRunnerV0MaterializaReceiptsAunqueNotaDeclareDeudaFuturaOPES(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-codex-ack-required-test-opes-future-debt"
	taskRef := "task-ref-codex-ack-required-test-opes-future-debt"
	agentRef := "agent-ref-codex-ack-required-test-opes-future-debt"
	deliveryRef := "ack-ref-codex-ack-required-test-opes-future-debt"
	requiredTests := []string{
		"validar criterios de aceptacion del cambio",
		"validar contrato externo de dominio",
	}
	ackPath := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	ack := orquestaruntimecodex.CodexAgentAckV0{
		SchemaVersion: orquestaruntimecodex.CodexAgentAckSchemaVersionV0,
		RequestID:     agentRef,
		CorrelationID: "corr-codex-ack-required-test-opes-future-debt",
		AckRef:        deliveryRef,
		TargetModule:  "orquesta-app-stack-programacion",
		TaskRef:       taskRef,
		Status:        "completed",
		Tests:         orquestaruntimecodex.EvidenceListV0(requiredTests),
		TestReceipts:  codexStackRequiredTestReceiptsV0(requiredTests),
		Notes:         orquestaruntimecodex.EvidenceListV0{"estado_no_publicable: faltan ampliacion A1, tests/tutor, HTML, RAG, audio y paquete"},
	}
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("Marshal ACK: %v", err)
	}
	if err := os.WriteFile(ackPath, data, 0o600); err != nil {
		t.Fatalf("Write ACK: %v", err)
	}
	descriptor := orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef: "descriptor-ref-codex-ack-required-test-opes-future-debt",
		RunID:         runRef,
		AgentRef:      agentRef,
		AckPath:       ackPath,
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			RequestID:     agentRef,
			CorrelationID: "corr-codex-ack-required-test-opes-future-debt",
			AgentPacket: orquestaruntime.AgentStartPacketV0{
				RequestID:     agentRef,
				CorrelationID: "corr-codex-ack-required-test-opes-future-debt",
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
		ReviewRequestID:   "review-request-ref-codex-ack-required-test-opes-future-debt",
		ReviewResultRef:   "review-result-ref-codex-ack-required-test-opes-future-debt",
		AcceptedReviewRef: "accepted-review-ref-codex-ack-required-test-opes-future-debt",
		OccurredAt:        "2026-05-25T16:34:00Z",
		CorrelationID:     "corr-codex-ack-required-test-opes-future-debt",
	})
	if err != nil {
		t.Fatalf("RunRequiredTestsV0: %v", err)
	}
	if len(result.EvidenceRefs) != len(requiredTests) ||
		len(result.PassedEvidenceRefs) != len(requiredTests) ||
		len(result.FailedEvidenceRefs) != 0 {
		t.Fatalf("debe materializar receipts causales pese a nota generica: %+v", result)
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
