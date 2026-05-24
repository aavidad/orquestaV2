package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexStackV0FileOutsideWriteSetNoReplanificaNiArrancaSegundoPadre(t *testing.T) {
	ctx := context.Background()
	runtime := newDecisionWritingFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)

	drainCodexStackUntilProgrammingDeliveryForSoftRailV0(t, ctx, stack, director.RunRef)

	descriptor := mustCodexStackDescriptorByTaskRefV0(t, stack, "task-ref-stack-agenda-001")
	addOutsideWriteSetFileToCodexStackACKForTestV0(t, descriptor.AckPath, descriptor.Spec)
	writeStackReviewGateFileForTestV0(
		t,
		descriptor.ProjectWorkDir,
		"docs/no-autorizado.md",
		"evidencia de rail blando fuera del write-set\n",
	)
	deliveryRef := descriptor.Spec.AgentPacket.DeliveryRefs.AckRef
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	openCodexStackPhaseForTestV0(
		t,
		stack,
		director.RunRef,
		orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		"Revisar entrega con fichero fuera del write-set.",
	)
	assertCodexStackReviewObservationAcceptedWithSoftRailV0(
		t,
		stack,
		run,
		deliveryRef,
		"gate-issue:file_outside_write_set",
	)

	drain, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:            director.RunRef,
		CorrelationID:     "corr-stack-review-rework-replan-outside-001",
		MaxBursts:         24,
		MaxStepsPerBurst:  8,
		MaxCommands:       20,
		MaxOutboxPerCycle: 8,
		MaxExternalWaits:  2,
	})
	if err != nil {
		t.Fatalf("DrainRunV0 review/rework: %v issues=%+v", err, codexStackBurstIssuesForErrorV0(err))
	}
	run = mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if !codexStackRefsContainPartV0(run.ReviewResults, "#review_result:accepted") ||
		!codexStackRefsContainPartV0(run.AcceptedReviews, deliveryRef) ||
		codexStackRefsContainPartV0(run.ReworkRequests, deliveryRef) ||
		codexStackRefsContainPartV0(run.ReplanDecisions, "#action:retry_task") ||
		codexStackRefsContainPartV0(run.StartedAgents, "agent-ref-task-ref-stack-agenda-001-") ||
		codexStackRefsContainPartV0(run.StartedAgents, "agent-ref-rework-request-ref-") {
		t.Fatalf("file_outside_write_set no debe replanificar drain=%s run=%+v", drain.Status, run)
	}
}

func drainCodexStackUntilProgrammingDeliveryForSoftRailV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
) {
	t.Helper()
	for _, correlationID := range []string{
		"corr-stack-review-rework-program-outside-001",
		"corr-stack-review-rework-delivery-outside-001",
	} {
		if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
			RunRef:               runRef,
			CorrelationID:        correlationID,
			MaxBursts:            16,
			MaxStepsPerBurst:     8,
			MaxDispatchesPerWait: 8,
			MaxExternalWaits:     1,
		}); err != nil {
			t.Fatalf("DrainRunV0 %s: %v", correlationID, err)
		}
	}
}

func addOutsideWriteSetFileToCodexStackACKForTestV0(
	t *testing.T,
	ackPath string,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) {
	t.Helper()
	ack, issues := orquestaruntimecodex.ReadAndValidateCodexAgentAckFileV0(ackPath, spec)
	if len(issues) > 0 {
		t.Fatalf("ack invalido antes de rail blando: %+v", issues)
	}
	ack.Files = append(ack.Files, "docs/no-autorizado.md")
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	if err := os.WriteFile(ackPath, data, 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}
}
