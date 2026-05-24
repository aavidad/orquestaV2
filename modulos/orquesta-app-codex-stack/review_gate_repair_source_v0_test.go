package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
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
