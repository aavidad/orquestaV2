package orquestastatefile

import (
	"context"
	"testing"

	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestStoreV0RecuperaOperationalDirectorPlanStateV0(t *testing.T) {
	rootDir := t.TempDir()
	store := mustStoreV0(t, rootDir)
	state := stateFileOperationalDirectorPlanStateV0()
	if err := store.SaveOperationalDirectorPlanStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}
	recovered := mustStoreV0(t, rootDir)
	got, err := recovered.LoadOperationalDirectorPlanStateV0(context.Background(), state.RunRef, state.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if got.StateRef != state.StateRef ||
		got.ActiveStepID != state.ActiveStepID ||
		len(got.Steps) != 2 {
		t.Fatalf("state=%+v", got)
	}
}

func TestStoreV0ActualizaOperationalDirectorPlanStateV0(t *testing.T) {
	rootDir := t.TempDir()
	store := mustStoreV0(t, rootDir)
	state := stateFileOperationalDirectorPlanStateV0()
	if err := store.SaveOperationalDirectorPlanStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}
	state.ActiveStepID = "step-wait-subagents"
	state.Steps[0].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
	state.Steps[1].Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
	state.Steps[1].WaitRefs = []string{"wait-ref-state-file-plan-state-001"}
	state.UpdatedAt = "2026-05-17T16:35:00Z"
	if err := store.SaveOperationalDirectorPlanStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0 update: %v", err)
	}
	got, err := store.LoadOperationalDirectorPlanStateV0(context.Background(), state.RunRef, state.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if got.ActiveStepID != "step-wait-subagents" ||
		got.Steps[1].Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		t.Fatalf("state=%+v", got)
	}
}

func TestStoreV0RecuperaOperationalDirectorPlanStateConWaitActivoYPendingRefs(t *testing.T) {
	rootDir := t.TempDir()
	store := mustStoreV0(t, rootDir)
	state := stateFileOperationalDirectorPlanStateV0()
	state.ActiveStepID = "step-wait-subagents"
	state.PendingAgentRefs = []string{"agent-ref-state-file-plan-state-001"}
	state.Steps[0].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
	state.Steps[1].Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
	state.Steps[1].WaitRefs = []string{"wait-ref-state-file-plan-state-001"}
	state.Steps[1].PendingAgentRefs = []string{"agent-ref-state-file-plan-state-001"}
	if err := store.SaveOperationalDirectorPlanStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}
	recovered := mustStoreV0(t, rootDir)
	got, err := recovered.LoadOperationalDirectorPlanStateV0(context.Background(), state.RunRef, state.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := got.Steps[1]
	if got.ActiveStepID != "step-wait-subagents" ||
		len(got.PendingAgentRefs) != 1 ||
		len(waitStep.WaitRefs) != 1 ||
		len(waitStep.PendingAgentRefs) != 1 {
		t.Fatalf("state=%+v waitStep=%+v", got, waitStep)
	}
}

func TestStoreV0RecuperaOperationalDirectorPlanStateConAcceptedReviewRefs(t *testing.T) {
	rootDir := t.TempDir()
	store := mustStoreV0(t, rootDir)
	state := stateFileOperationalDirectorPlanStateV0()
	state.ActiveStepID = "step-replan-or-close"
	state.Steps[0].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
	state.Steps[1].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
	state.Steps = append(state.Steps,
		orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			StepID:             "step-review-deliveries",
			Kind:               orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
			Status:             orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
			WaveRef:            "wave-state-file-001",
			CohortRef:          "cohort-state-file-001",
			TaskRefs:           []string{"task-ref-state-file-plan-state-001"},
			AgentRefs:          []string{"agent-ref-state-file-plan-state-001"},
			DeliveryRefs:       []string{"delivery-ref-state-file-plan-state-001"},
			ReviewResultRefs:   []string{"review-result-ref-state-file-plan-state-001"},
			AcceptedReviewRefs: []string{"accepted-review-ref-state-file-plan-state-001"},
		},
		orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			StepID:    "step-replan-or-close",
			Kind:      orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
			Status:    orquestadirectoroperativo.OperationalDirectorStepRunningV0,
			WaveRef:   "wave-state-file-001",
			CohortRef: "cohort-state-file-001",
			TaskRefs:  []string{"task-ref-state-file-plan-state-001"},
			AgentRefs: []string{"agent-ref-state-file-plan-state-001"},
		},
	)
	if err := store.SaveOperationalDirectorPlanStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}
	got, err := store.LoadOperationalDirectorPlanStateV0(context.Background(), state.RunRef, state.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	reviewStep := got.Steps[2]
	if len(reviewStep.AcceptedReviewRefs) != 1 ||
		reviewStep.AcceptedReviewRefs[0] != "accepted-review-ref-state-file-plan-state-001" {
		t.Fatalf("state=%+v reviewStep=%+v", got, reviewStep)
	}
}

func TestStoreV0RecuperaOperationalDirectorPlanStateConReworkYReplanRefs(t *testing.T) {
	rootDir := t.TempDir()
	store := mustStoreV0(t, rootDir)
	state := stateFileOperationalDirectorPlanStateV0()
	state.ActiveStepID = "step-review-deliveries"
	state.ReplanAttempts = 1
	state.Steps[0].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
	state.Steps[1].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
	state.Steps = append(state.Steps, orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
		StepID:             "step-review-deliveries",
		Kind:               orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
		Status:             orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0,
		WaveRef:            "wave-state-file-001",
		CohortRef:          "cohort-state-file-001",
		TaskRefs:           []string{"task-ref-state-file-plan-state-001"},
		AgentRefs:          []string{"agent-ref-state-file-plan-state-001"},
		DeliveryRefs:       []string{"delivery-ref-state-file-plan-state-001"},
		ReviewResultRefs:   []string{"review-result-ref-state-file-plan-state-001"},
		ReworkRequestRefs:  []string{"rework-request-ref-state-file-plan-state-001"},
		ReplanDecisionRefs: []string{"replan-ref-state-file-plan-state-001"},
		BlockerRefs:        []string{"review-rework-replan-recorded"},
		Reason:             "review-rework-replan-recorded",
	})
	if err := store.SaveOperationalDirectorPlanStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}
	recovered := mustStoreV0(t, rootDir)
	got, err := recovered.LoadOperationalDirectorPlanStateV0(context.Background(), state.RunRef, state.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	reviewStep := got.Steps[2]
	if got.ReplanAttempts != 1 ||
		len(reviewStep.ReworkRequestRefs) != 1 ||
		reviewStep.ReworkRequestRefs[0] != "rework-request-ref-state-file-plan-state-001" ||
		len(reviewStep.ReplanDecisionRefs) != 1 ||
		reviewStep.ReplanDecisionRefs[0] != "replan-ref-state-file-plan-state-001" {
		t.Fatalf("state=%+v reviewStep=%+v", got, reviewStep)
	}
}

func TestStoreV0RecuperaOperationalDirectorPlanStateConReworkRequestRefsYReplanDecisionRefs(t *testing.T) {
	rootDir := t.TempDir()
	store := mustStoreV0(t, rootDir)
	state := stateFileOperationalDirectorPlanStateV0()
	state.ActiveStepID = "step-review-deliveries"
	state.Steps[0].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
	state.Steps[1].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
	state.Steps = append(state.Steps,
		orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			StepID:             "step-review-deliveries",
			Kind:               orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
			Status:             orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0,
			WaveRef:            "wave-state-file-001",
			CohortRef:          "cohort-state-file-001",
			TaskRefs:           []string{"task-ref-state-file-plan-state-001"},
			AgentRefs:          []string{"agent-ref-state-file-plan-state-001"},
			DeliveryRefs:       []string{"delivery-ref-state-file-plan-state-001"},
			ReviewResultRefs:   []string{"review-result-ref-state-file-plan-state-001"},
			ReworkRequestRefs:  []string{"rework-request-ref-state-file-plan-state-001", "rework-request-ref-state-file-plan-state-002"},
			ReplanDecisionRefs: []string{"replan-decision-ref-state-file-plan-state-001", "replan-decision-ref-state-file-plan-state-002"},
		},
	)
	if err := store.SaveOperationalDirectorPlanStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}
	recovered := mustStoreV0(t, rootDir)
	got, err := recovered.LoadOperationalDirectorPlanStateV0(context.Background(), state.RunRef, state.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	reviewStep := got.Steps[2]
	if len(reviewStep.ReworkRequestRefs) != 2 ||
		reviewStep.ReworkRequestRefs[0] != "rework-request-ref-state-file-plan-state-001" ||
		reviewStep.ReworkRequestRefs[1] != "rework-request-ref-state-file-plan-state-002" ||
		len(reviewStep.ReplanDecisionRefs) != 2 ||
		reviewStep.ReplanDecisionRefs[0] != "replan-decision-ref-state-file-plan-state-001" ||
		reviewStep.ReplanDecisionRefs[1] != "replan-decision-ref-state-file-plan-state-002" {
		t.Fatalf("state=%+v reviewStep=%+v", got, reviewStep)
	}
}

func TestStoreV0RechazaOperationalDirectorPlanStateConRefInternaInconsistente(t *testing.T) {
	rootDir := t.TempDir()
	store := mustStoreV0(t, rootDir)
	state := stateFileOperationalDirectorPlanStateV0()
	document := operationalDirectorPlanStateDocumentV0{
		SchemaVersion: operationalDirectorPlanStateSchemaV0,
		RunRef:        state.RunRef,
		PlanRef:       state.PlanRef,
		StateRef:      state.StateRef,
		State:         state,
	}
	document.State.PlanRef = "operational-director-plan-state-file-distinto"
	if err := writeJSONAtomicV0(store.operationalDirectorPlanStatePathV0(state.RunRef, state.PlanRef), document); err != nil {
		t.Fatalf("writeJSONAtomicV0: %v", err)
	}
	if _, err := store.LoadOperationalDirectorPlanStateV0(context.Background(), state.RunRef, state.PlanRef); err == nil {
		t.Fatal("err=nil, want ref interna inconsistente")
	}
}

func stateFileOperationalDirectorPlanStateV0() orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	return orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:    orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:         "operational-director-plan-state-file-001",
		PlanRef:          "operational-director-plan-state-file-001",
		RequestRef:       "operational-director-request-state-file-001",
		RunRef:           "run-state-file-operational-plan-state-001",
		ProjectRef:       "project-state-file-operational-plan-state-001",
		Mode:             orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:           orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:     "step-launch-subagents",
		ActiveWaveRef:    "wave-state-file-001",
		ActiveCohortRef:  "cohort-state-file-001",
		RequiredTestRefs: []string{"go test ./..."},
		CorrelationID:    "corr-state-file-operational-plan-state-001",
		ObservedAt:       "2026-05-17T16:30:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			{
				StepID:    "step-launch-subagents",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepRunningV0,
				WaveRef:   "wave-state-file-001",
				CohortRef: "cohort-state-file-001",
				TaskRefs:  []string{"task-ref-state-file-plan-state-001"},
				AgentRefs: []string{
					"agent-ref-state-file-plan-state-001",
				},
			},
			{
				StepID:    "step-wait-subagents",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:   "wave-state-file-001",
				CohortRef: "cohort-state-file-001",
			},
		},
	}
}
