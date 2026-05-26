package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestCodexStackV0ReviewGateNormalizaTestsGoEquivalentesV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	spec := codexStackReviewGateSpecForTestV0("ack-ref-stack-review-test-alias-001")
	spec.AgentPacket.Task.RequiredTests = []string{
		"go test -count=1 ./modulos/orquesta-app-codex-stack -run TestAutoprogrammingResidentModeV0",
	}
	projectDir := t.TempDir()
	writeStackReviewGateFileForTestV0(t, projectDir, "internal/api/handler.go", "package api\n")
	recordStackReviewDescriptorWithTestsForTestV0(t, stack, projectDir, spec, []string{
		"go test ./modulos/orquesta-app-codex-stack -run TestAutoprogrammingResidentModeV0 -count=1",
	})

	observations, err := stack.Ports.ReviewGateSource.BuildReviewGateObservationsV0(
		context.Background(),
		codexStackReviewGateRequestForTestV0(spec, nil),
	)
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 ||
		observations[0].Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 ||
		observations[0].AcceptedReviewRef == "" ||
		!codexStackReviewGateHasEvidenceForTestV0(observations[0].EvidenceRefs, codexStackReviewGateNormalizedTestsEvidenceV0) {
		t.Fatalf("observations=%+v", observations)
	}
}

func TestReviewReworkMissingWriteSetTargetsV0NormalizaWebAdminComoWebV0(t *testing.T) {
	spec := codexStackReviewGateSpecForTestV0("ack-ref-stack-review-webadmin-001")
	spec.AgentPacket.Task.WriteSet = []string{"web"}
	projectDir := t.TempDir()
	writeStackReviewGateFileForTestV0(t, projectDir, "internal/webadmin/handler.go", "package webadmin\n")
	descriptor := orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef:  "receipt-ref-webadmin",
		RunID:          "run-ref-stack-review-001",
		AgentRef:       spec.RequestID,
		Spec:           spec,
		ProjectWorkDir: projectDir,
	}

	missing := reviewReworkMissingWriteSetTargetsV0(spec.AgentPacket.DeliveryRefs.AckRef, []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{descriptor})
	if len(missing) != 0 {
		t.Fatalf("missing=%v", missing)
	}
}

func TestCodexStackReviewGateReworkAcceptanceObservaPadreTrasFollowupCerradoV0(t *testing.T) {
	const (
		runRef           = "run-ref-stack-review-rework-acceptance-001"
		parentTaskRef    = "task-ref-stack-review-parent-001"
		parentDelivery   = "ack-ref-stack-review-parent-001"
		followupTaskRef  = "task-ref-stack-review-followup-001"
		followupDelivery = "ack-ref-stack-review-followup-001"
	)
	parentSpec := codexStackReviewGateSpecForTestV0(parentDelivery)
	parentSpec.RequestID = "agent-ref-stack-review-parent-001"
	parentSpec.AgentPacket.RequestID = parentSpec.RequestID
	parentSpec.AgentPacket.Task.TaskRef = parentTaskRef
	parentSpec.AgentPacket.DeliveryRefs.AckRef = parentDelivery
	followupSpec := codexStackReviewGateSpecForTestV0(followupDelivery)
	followupSpec.RequestID = "agent-ref-stack-review-followup-001"
	followupSpec.AgentPacket.RequestID = followupSpec.RequestID
	followupSpec.AgentPacket.Task.TaskRef = followupTaskRef
	followupSpec.AgentPacket.DeliveryRefs.AckRef = followupDelivery

	reviewRequestRef := "review-request-ref-" + parentDelivery
	reviewResultRef := "review-result-ref-" + parentDelivery
	reworkRequestRef := "rework-request-ref-" + reviewResultRef
	request := orquestacionnucleoapp.ReviewGateObservationRequestV0{
		Run: orquestacoreworkflow.OrchestrationRunV0{
			RunID:        runRef,
			CurrentPhase: orquestacoreworkflow.OrchestrationPhaseRevisionV0,
			Tasks:        []string{parentTaskRef, followupTaskRef},
			Deliveries:   []string{parentDelivery, followupDelivery},
			ReviewResults: []string{
				reviewResultRef + "#review_result:changes_requested#review_request:" + reviewRequestRef + "#delivery:" + parentDelivery,
				"review-result-ref-" + followupDelivery + "#review_result:accepted#review_request:review-request-ref-" + followupDelivery + "#delivery:" + followupDelivery,
			},
			ReworkRequests: []string{
				reworkRequestRef + "#review_result:" + reviewResultRef + "#review_request:" + reviewRequestRef + "#delivery:" + parentDelivery,
			},
			ReplanDecisions: []string{
				"replan-ref-stack-review-rework-001#source:" + reworkRequestRef + "#task:" + parentTaskRef + "#action:split_task#followups:" + followupTaskRef,
			},
			AcceptedReviews: []string{"accepted-review-ref-" + followupDelivery},
			ClosedTasks:     []string{followupTaskRef},
		},
	}
	descriptors := []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		{RunID: runRef, AgentRef: parentSpec.RequestID, Spec: parentSpec},
		{RunID: runRef, AgentRef: followupSpec.RequestID, Spec: followupSpec},
	}

	observations := codexStackReviewGateReworkAcceptanceObservationsV0(request, descriptors)
	if len(observations) != 1 {
		t.Fatalf("observations=%+v", observations)
	}
	got := observations[0]
	if got.DeliveryRef != parentDelivery ||
		got.ReviewRequestID != reviewRequestRef ||
		got.Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 ||
		got.AcceptedReviewRef == "" ||
		got.ReviewResultRef == "" {
		t.Fatalf("observacion padre invalida: %+v", got)
	}
	if !codexStackReviewGateHasEvidenceForTestV0(got.EvidenceRefs, followupTaskRef) ||
		!codexStackReviewGateHasEvidenceForTestV0(got.EvidenceRefs, "accepted-review-ref-"+followupDelivery) {
		t.Fatalf("evidence_refs=%v", got.EvidenceRefs)
	}
}

func recordStackReviewDescriptorWithTestsForTestV0(
	t *testing.T,
	stack StackV0,
	projectDir string,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	tests []string,
) {
	t.Helper()
	ackPath := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	ack := orquestaruntimecodex.CodexAgentAckV0{
		SchemaVersion: orquestaruntimecodex.CodexAgentAckSchemaVersionV0,
		RequestID:     spec.RequestID,
		CorrelationID: spec.CorrelationID,
		AckRef:        spec.AgentPacket.DeliveryRefs.AckRef,
		TargetModule:  spec.AgentPacket.TargetModule,
		TaskRef:       spec.AgentPacket.Task.TaskRef,
		Status:        "completed",
		Files:         orquestaruntimecodex.EvidenceListV0{"internal/api/handler.go"},
		Tests:         orquestaruntimecodex.EvidenceListV0(tests),
		TestReceipts:  codexStackRequiredTestReceiptsV0(tests),
	}
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	if err := os.WriteFile(ackPath, data, 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}
	err = stack.Stores.ReceiptStore.RecordCodexReceiptDescriptorV0(
		context.Background(),
		orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
			DescriptorRef:  "receipt-ref-" + spec.AgentPacket.DeliveryRefs.AckRef,
			RunID:          "run-ref-stack-review-001",
			AgentRef:       spec.RequestID,
			Spec:           spec,
			AckPath:        ackPath,
			ProjectWorkDir: projectDir,
		},
	)
	if err != nil {
		t.Fatalf("RecordCodexReceiptDescriptorV0: %v", err)
	}
}
