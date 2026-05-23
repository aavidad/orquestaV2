package orquestadirectorscheduler

import (
	"testing"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildDirectorSchedulerTickV0WaitsForRepairableLiveWorkOverlap(t *testing.T) {
	input := validSchedulerTickInputWithLiveOverlapV0()

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerSequenceActionV0(t, plan, 0, SchedulerWorkSequenceActionWaitLiveWorkV0)
	assertSchedulerWaitingReasonV0(t, plan, SchedulerWaitingLiveWorkOverlapV0)
	if len(plan.BlockedRefs) != 0 {
		t.Fatalf("blocked_refs=%v", plan.BlockedRefs)
	}
}

func TestBuildDirectorSchedulerTickV0QueuesCandidateAfterLiveDependency(t *testing.T) {
	input := validSchedulerTickInputWithLiveOverlapV0()
	liveClaimRef := input.WorkClaims[0].ClaimRef
	input.WorkClaims[1].WriteSet[0].Ref = "modulos/independent/new.go"
	input.WorkClaims[1].DependsOn = []string{liveClaimRef}
	input.WorkCandidates[0].Claims = []orquestacoreconcurrency.WorksetClaimV0{input.WorkClaims[1]}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerSequenceActionV0(t, plan, 0, SchedulerWorkSequenceActionQueueAfterLiveWorkV0)
}

func TestBuildDirectorSchedulerTickV0CreatesReviewTaskForLiveOverlap(t *testing.T) {
	input := validSchedulerTickInputWithLiveOverlapV0()
	input.WorkCandidates[0].LiveWorkPolicy = &SchedulerLiveWorkSequencePolicyV0{
		OnOverlap:           SchedulerWorkSequenceActionCreateReviewTaskV0,
		ReviewTaskCandidate: validSchedulerSequenceMicrotaskV0("review", "Revision de solape vivo."),
		EvidenceRefs:        []string{"evidence-ref-live-policy-review"},
	}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
	assertSchedulerCommandTypesV0(t, plan, orquestacoreworkflow.OrchestrationCommandCreateMicrotaskV0)
	assertSchedulerSequenceActionV0(t, plan, 0, SchedulerWorkSequenceActionCreateReviewTaskV0)
}

func TestBuildDirectorSchedulerTickV0CreatesStudyTaskWhenLiveOverlapNeedsContext(t *testing.T) {
	input := validSchedulerTickInputWithLiveOverlapV0()
	input.WorkCandidates[0].LiveWorkPolicy = &SchedulerLiveWorkSequencePolicyV0{
		OnOverlap:          SchedulerWorkSequenceActionCreateStudyTaskV0,
		StudyTaskCandidate: validSchedulerSequenceMicrotaskV0("study", "Estudio acotado de solape vivo."),
		EvidenceRefs:       []string{"evidence-ref-live-policy-study"},
	}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
	assertSchedulerCommandTypesV0(t, plan, orquestacoreworkflow.OrchestrationCommandCreateMicrotaskV0)
	assertSchedulerSequenceActionV0(t, plan, 0, SchedulerWorkSequenceActionCreateStudyTaskV0)
}

func TestBuildDirectorSchedulerTickV0LiveOverlapDoesNotBlockIndependentWork(t *testing.T) {
	input := validSchedulerTickInputWithLiveOverlapV0()
	independent := schedulableCandidateVariantV0("independent", "modulos/independent/worker.go")
	input.WorkClaims = append(input.WorkClaims, independent.Claims...)
	input.WorkCandidates = append(input.WorkCandidates, independent)
	input.Snapshot.CapacityDecisions = append(
		input.Snapshot.CapacityDecisions,
		"capacity-ref-scheduler-independent",
	)

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 2)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRecordConcurrencyGateV0,
		orquestacoreworkflow.OrchestrationCommandRequestAgentV0,
	)
	assertSchedulerSequenceActionV0(t, plan, 0, SchedulerWorkSequenceActionWaitLiveWorkV0)
	assertSchedulerSequenceActionV0(t, plan, 1, SchedulerWorkSequenceActionExecuteNowV0)
}

func validSchedulerTickInputWithLiveOverlapV0() DirectorSchedulerTickInputV0 {
	input := validSchedulerTickInputV0()
	live := validWorksetClaimV0("claim-ref-live-001", "agent-ref-live-001", "modulos/demo")
	candidate := schedulableCandidateVariantV0("overlap", "modulos/demo/main.go")
	input.WorkClaims = append([]orquestacoreconcurrency.WorksetClaimV0{live}, candidate.Claims...)
	input.WorkCandidates = []SchedulableWorkCandidateV0{candidate}
	input.Snapshot.StartedAgents = []string{"agent-ref-live-001"}
	input.Snapshot.CapacityDecisions = []string{"capacity-ref-scheduler-overlap"}
	return input
}

func validSchedulerSequenceMicrotaskV0(suffix string, title string) *SchedulerCreateMicrotaskCandidateV0 {
	return &SchedulerCreateMicrotaskCandidateV0{
		CommandMeta: schedulerMetaV0("cmd-sequence-"+suffix, "sequence-"+suffix),
		Payload: orquestacoreworkflow.CreateMicrotaskCommandPayloadV0{
			Task: orquestacoreworkflow.WorkflowTaskV0{
				SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
				TaskID:        "task-ref-sequence-" + suffix,
				RunID:         "run-scheduler-001",
				PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
				Title:         title,
				Summary:       "Paso acotado antes de competir por el mismo alcance.",
				WriteSet:      []string{"modulos/orquesta-director-scheduler/" + suffix + ".md"},
				AcceptanceCriteria: []string{
					"Explica secuencia compatible con trabajo vivo.",
					"No compite por alcance en uso.",
				},
				FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{
					{ContractRef: "contract:function:live-work-sequence:v0", FunctionName: "BuildDirectorSchedulerTickV0"},
				},
			},
		},
	}
}

func assertSchedulerSequenceActionV0(
	t *testing.T,
	plan DirectorSchedulerTickPlanV0,
	index int,
	want SchedulerWorkSequenceActionV0,
) {
	t.Helper()
	if len(plan.WorkSequenceDecisions) <= index {
		t.Fatalf("sequence_decisions=%v, index=%d", plan.WorkSequenceDecisions, index)
	}
	if plan.WorkSequenceDecisions[index].Action != want {
		t.Fatalf("sequence_decision[%d]=%s, want %s", index, plan.WorkSequenceDecisions[index].Action, want)
	}
}

func assertSchedulerWaitingReasonV0(
	t *testing.T,
	plan DirectorSchedulerTickPlanV0,
	want SchedulerWaitingReasonV0,
) {
	t.Helper()
	for _, reason := range plan.WaitingReasons {
		if reason == want {
			return
		}
	}
	t.Fatalf("waiting_reasons=%v, want %s", plan.WaitingReasons, want)
}
