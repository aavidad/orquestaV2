package orquestadirectortickinput

import (
	"encoding/json"
	"fmt"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func TestBuildDirectorSchedulerTickInputV0CompactaCarrilReviewGate(t *testing.T) {
	run := tickInputRevisionRunWithHistoryV0(t)
	candidate := tickInputReviewGateCandidateV0("delivery-ref-review-target")

	input, err := BuildDirectorSchedulerTickInputV0(DirectorTickInputBuildRequestV0{
		TickRef:              "tick-ref-review-compact-001",
		OccurredAt:           "2026-05-06T12:07:00Z",
		Run:                  run,
		ReviewGateCandidates: []orquestadirectorscheduler.SchedulableReviewGateCandidateV0{candidate},
		EvidenceRefs:         []string{"evidence-ref-review-compact-001"},
	})
	if err != nil {
		t.Fatalf("BuildDirectorSchedulerTickInputV0: %v", err)
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	if len(data) > 16*1024 {
		t.Fatalf("scheduler input no compacto: bytes=%d", len(data))
	}
	if len(input.Snapshot.Deliveries) != 1 ||
		input.Snapshot.Deliveries[0] != "delivery-ref-review-target" {
		t.Fatalf("deliveries=%v", input.Snapshot.Deliveries)
	}
	if len(input.Snapshot.Tasks) != 0 ||
		len(input.Snapshot.Agents) != 0 ||
		len(input.Snapshot.StartedAgents) != 0 ||
		len(input.WorkCandidates) != 0 {
		t.Fatalf("snapshot conserva historial innecesario: %+v", input.Snapshot)
	}
}

func tickInputRevisionRunWithHistoryV0(t *testing.T) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run := tickInputProgramacionRunV0(t)
	run = tickInputApplyCommandV0(t, run, tickInputOpenRevisionCommandV0(t))
	run.Deliveries = append(run.Deliveries, "delivery-ref-review-target")
	for i := 0; i < 80; i++ {
		delivery := fmt.Sprintf("delivery-ref-review-history-%03d", i)
		review := fmt.Sprintf("review-request-ref-history-%03d", i)
		result := fmt.Sprintf(
			"review-result-ref-history-%03d#review_result:accepted#review_request:%s#delivery:%s",
			i,
			review,
			delivery,
		)
		run.Tasks = append(run.Tasks, fmt.Sprintf("task-ref-review-history-%03d", i))
		run.Agents = append(run.Agents, fmt.Sprintf("agent-ref-review-history-%03d", i))
		run.StartedAgents = append(run.StartedAgents, fmt.Sprintf("agent-ref-review-history-%03d", i))
		run.Deliveries = append(run.Deliveries, delivery)
		run.Reviews = append(run.Reviews, review)
		run.ReviewResults = append(run.ReviewResults, result)
	}
	if issues := orquestacoreworkflow.ValidateOrchestrationRunV0(run); len(issues) > 0 {
		t.Fatalf("run invalido: %+v", issues)
	}
	return run
}

func tickInputOpenRevisionCommandV0(t *testing.T) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		tickInputCommandMetaV0("cmd-tick-input-open-revision", "open-revision"),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			Reason:  "Preparar revision",
		},
	)
	if err != nil {
		t.Fatalf("open revision command: %v", err)
	}
	return command
}

func tickInputReviewGateCandidateV0(
	deliveryRef string,
) orquestadirectorscheduler.SchedulableReviewGateCandidateV0 {
	reviewRef := "review-request-ref-review-target"
	resultRef := "review-result-ref-review-target"
	return orquestadirectorscheduler.SchedulableReviewGateCandidateV0{
		CandidateRef: "review-gate-candidate-ref-review-target",
		RequestReview: &orquestadirectorscheduler.SchedulerRequestReviewCandidateV0{
			CommandMeta: tickInputCommandMetaV0("cmd-review-request-target", "review-request-target"),
			Payload: orquestacoreworkflow.RequestReviewCommandPayloadV0{
				ReviewRequestID: reviewRef,
				PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
				DeliveryRef:     deliveryRef,
				Summary:         "Revision compacta.",
				EvidenceRefs:    []string{"evidence-ref-review-request-target"},
			},
		},
		RecordReviewResult: &orquestadirectorscheduler.SchedulerRecordReviewResultCandidateV0{
			CommandMeta: tickInputCommandMetaV0("cmd-review-result-target", "review-result-target"),
			Payload: orquestacoreworkflow.RecordReviewResultCommandPayloadV0{
				ReviewResultRef: resultRef,
				ReviewRequestID: reviewRef,
				DeliveryRef:     deliveryRef,
				Status:          orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
				Summary:         "Revision solicita cambios.",
				EvidenceRefs:    []string{"evidence-ref-review-result-target"},
			},
		},
		RequestRework: &orquestadirectorscheduler.SchedulerRequestReworkCandidateV0{
			CommandMeta: tickInputCommandMetaV0("cmd-review-rework-target", "review-rework-target"),
			Payload: orquestacoreworkflow.RequestReworkCommandPayloadV0{
				ReworkRequestRef: "rework-request-ref-review-target",
				PhaseID:          string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
				ReviewResultRef:  resultRef,
				ReviewRequestID:  reviewRef,
				DeliveryRef:      deliveryRef,
				Summary:          "Retrabajo compacto.",
				EvidenceRefs:     []string{"evidence-ref-review-rework-target"},
			},
		},
		EvidenceRefs: []string{"evidence-ref-review-gate-target"},
	}
}
