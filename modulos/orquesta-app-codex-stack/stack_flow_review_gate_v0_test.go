package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestCodexStackV0ReviewGateAceptaEntregaConEvidenciaReal(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	spec := codexStackReviewGateSpecForTestV0("ack-ref-stack-review-ok-001")
	projectDir := t.TempDir()
	writeStackReviewGateFileForTestV0(t, projectDir, "internal/api/handler.go", "package api\nfunc Handler() {}\n")
	recordStackReviewDescriptorForTestV0(t, stack, projectDir, spec, []string{"internal/api/handler.go"})

	observations, err := stack.Ports.ReviewGateSource.BuildReviewGateObservationsV0(
		context.Background(),
		codexStackReviewGateRequestForTestV0(spec, nil),
	)
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 || observations[0].Status != orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
		t.Fatalf("observations=%+v", observations)
	}
	if observations[0].AcceptedReviewRef == "" {
		t.Fatalf("accepted_review_ref vacio: %+v", observations[0])
	}
}

func TestCodexStackV0ReviewGatePideCambiosSiFicheroEsDemasiadoGrande(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	spec := codexStackReviewGateSpecForTestV0("ack-ref-stack-review-big-001")
	projectDir := t.TempDir()
	writeStackReviewGateFileForTestV0(t, projectDir, "internal/api/handler.go", strings.Repeat("linea\n", 301))
	recordStackReviewDescriptorForTestV0(t, stack, projectDir, spec, []string{"internal/api/handler.go"})

	observations, err := stack.Ports.ReviewGateSource.BuildReviewGateObservationsV0(
		context.Background(),
		codexStackReviewGateRequestForTestV0(spec, nil),
	)
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 ||
		observations[0].Status != orquestacoreworkflow.ReviewResultStatusChangesRequestedV0 ||
		observations[0].AcceptedReviewRef != "" {
		t.Fatalf("observations=%+v", observations)
	}
	if !codexStackReviewGateHasEvidenceForTestV0(observations[0].EvidenceRefs, "gate-issue:file_too_large") {
		t.Fatalf("evidence_refs=%v", observations[0].EvidenceRefs)
	}
}

func TestCodexStackV0ReviewGatePideCambiosSiFaltaDestinoWriteSet(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	spec := codexStackReviewGateSpecForTestV0("ack-ref-stack-review-missing-web-001")
	spec.AgentPacket.Task.WriteSet = []string{"internal/api", "web"}
	projectDir := t.TempDir()
	writeStackReviewGateFileForTestV0(t, projectDir, "internal/api/handler.go", "package api\nfunc Handler() {}\n")
	recordStackReviewDescriptorForTestV0(t, stack, projectDir, spec, []string{"internal/api/handler.go"})

	observations, err := stack.Ports.ReviewGateSource.BuildReviewGateObservationsV0(
		context.Background(),
		codexStackReviewGateRequestForTestV0(spec, nil),
	)
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 ||
		observations[0].Status != orquestacoreworkflow.ReviewResultStatusChangesRequestedV0 ||
		observations[0].AcceptedReviewRef != "" {
		t.Fatalf("observations=%+v", observations)
	}
	if !codexStackReviewGateHasEvidenceForTestV0(
		observations[0].EvidenceRefs,
		"gate-issue:write_set_target_missing:web",
	) {
		t.Fatalf("evidence_refs=%v", observations[0].EvidenceRefs)
	}
}

func recordStackReviewDescriptorForTestV0(
	t *testing.T,
	stack StackV0,
	projectDir string,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	files []string,
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
		Files:         orquestaruntimecodex.EvidenceListV0(files),
		Tests:         orquestaruntimecodex.EvidenceListV0(spec.AgentPacket.Task.RequiredTests),
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

func codexStackReviewGateRequestForTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	reviewResults []string,
) orquestacionnucleoapp.ReviewGateObservationRequestV0 {
	return orquestacionnucleoapp.ReviewGateObservationRequestV0{
		Run: orquestacoreworkflow.OrchestrationRunV0{
			RunID:         "run-ref-stack-review-001",
			CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseRevisionV0,
			Agents:        []string{spec.RequestID},
			StartedAgents: []string{spec.RequestID},
			Deliveries:    []string{spec.AgentPacket.DeliveryRefs.AckRef},
			ReviewResults: reviewResults,
		},
		CorrelationID: "corr-stack-review-gate-001",
		EvidenceRefs:  []string{"evidence-ref-stack-review-gate-001"},
	}
}

func codexStackReviewGateSpecForTestV0(
	ackRef string,
) orquestaruntime.ExternalAgentLaunchSpecV0 {
	return orquestaruntime.ExternalAgentLaunchSpecV0{
		RequestID:     "agent-ref-stack-review-001",
		CorrelationID: "corr-agent-stack-review-001",
		AgentPacket: orquestaruntime.AgentStartPacketV0{
			RequestID:     "agent-ref-stack-review-001",
			CorrelationID: "corr-agent-stack-review-001",
			TargetModule:  "agenda-app",
			Phase:         string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Task: orquestaruntime.AgentStartTaskV0{
				TaskRef:       "task-ref-stack-review-001",
				WriteSet:      []string{"internal/api"},
				RequiredTests: []string{"go test ./..."},
			},
			DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
				MailboxRef:   "mailbox-ref-stack-review-001",
				AckRef:       ackRef,
				ReadinessRef: "readiness-ref-stack-review-001",
			},
		},
	}
}

func writeStackReviewGateFileForTestV0(
	t *testing.T,
	root string,
	rel string,
	content string,
) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func codexStackReviewGateHasEvidenceForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
