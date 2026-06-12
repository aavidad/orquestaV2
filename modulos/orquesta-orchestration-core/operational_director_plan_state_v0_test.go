package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
)

func TestOperationalDirectorPlanStateV0NormalizaYValida(t *testing.T) {
	state, err := NewOperationalDirectorPlanStateV0(operationalDirectorPlanStateForTestV0())
	if err != nil {
		t.Fatalf("NewOperationalDirectorPlanStateV0: %v", err)
	}
	if state.Status != OperationalDirectorPlanStateActiveV0 ||
		state.Steps[0].Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		len(state.Steps[0].TaskRefs) != 1 {
		t.Fatalf("state=%+v", state)
	}
}

func TestOperationalDirectorPlanStateV0RechazaActiveStepDesconocido(t *testing.T) {
	state := operationalDirectorPlanStateForTestV0()
	state.ActiveStepID = "step-no-existe"
	if _, err := NewOperationalDirectorPlanStateV0(state); err == nil {
		t.Fatal("err=nil, want active_step_id invalido")
	}
}

func TestInMemoryOperationalDirectorPlanStateStoreV0SobrescribeEstadoVivo(t *testing.T) {
	state := operationalDirectorPlanStateForTestV0()
	store := NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	state.ActiveStepID = "step-wait-subagents"
	state.Steps[0].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
	state.Steps[1].Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
	state.Steps[1].WaitRefs = []string{"wait-ref-plan-state-001"}
	state.UpdatedAt = "2026-05-17T16:05:00Z"
	if err := store.SaveOperationalDirectorPlanStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
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

func TestOperationalDirectorPlanStateV0RechazaWaitActivoSinWaitRef(t *testing.T) {
	state := operationalDirectorPlanStateForTestV0()
	state.ActiveStepID = "step-wait-subagents"
	state.Steps[0].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
	state.Steps[1].Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
	if _, err := NewOperationalDirectorPlanStateV0(state); err == nil {
		t.Fatal("err=nil, want wait_ref requerido")
	}
}

func TestOperationalDirectorPlanStateV0RechazaScopeActivoInconsistente(t *testing.T) {
	state := operationalDirectorPlanStateForTestV0()
	state.ActiveWaveRef = "wave-distinta"
	if _, err := NewOperationalDirectorPlanStateV0(state); err == nil {
		t.Fatal("err=nil, want active_wave_ref inconsistente")
	}
}

func TestOperationalDirectorPlanStateV0NormalizaAcceptedReviewRefs(t *testing.T) {
	state := operationalDirectorPlanStateWithReviewAcceptedForTestV0()
	state.Steps[2].AcceptedReviewRefs = []string{" accepted-review-ref-plan-state-001 ", "accepted-review-ref-plan-state-001"}
	got, err := NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		t.Fatalf("NewOperationalDirectorPlanStateV0: %v", err)
	}
	if len(got.Steps[2].AcceptedReviewRefs) != 1 ||
		got.Steps[2].AcceptedReviewRefs[0] != "accepted-review-ref-plan-state-001" {
		t.Fatalf("accepted_review_refs=%+v", got.Steps[2].AcceptedReviewRefs)
	}
}

func TestOperationalDirectorPlanStateV0PermiteAcceptedReviewRefsEnPasosCausales(t *testing.T) {
	state := operationalDirectorPlanStateWithReviewAcceptedForTestV0()
	state.Steps = append(state.Steps, OperationalDirectorPlanStepStateV0{
		StepID:             "step-run-required-tests",
		Kind:               orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
		Status:             orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
		WaveRef:            "wave-001",
		CohortRef:          "cohort-001",
		TaskRefs:           []string{"task-ref-plan-state-001"},
		AgentRefs:          []string{"agent-ref-plan-state-001"},
		DeliveryRefs:       []string{"delivery-ref-plan-state-001"},
		ReviewResultRefs:   []string{"review-result-ref-plan-state-001"},
		AcceptedReviewRefs: []string{"accepted-review-ref-plan-state-001"},
	})
	state.Steps[3].AcceptedReviewRefs = []string{"accepted-review-ref-plan-state-001"}
	got, err := NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		t.Fatalf("NewOperationalDirectorPlanStateV0: %v", err)
	}
	if len(got.Steps[3].AcceptedReviewRefs) != 1 ||
		got.Steps[3].AcceptedReviewRefs[0] != "accepted-review-ref-plan-state-001" {
		t.Fatalf("replan accepted_review_refs=%+v", got.Steps[3].AcceptedReviewRefs)
	}
	if len(got.Steps[4].AcceptedReviewRefs) != 1 ||
		got.Steps[4].AcceptedReviewRefs[0] != "accepted-review-ref-plan-state-001" {
		t.Fatalf("required-tests accepted_review_refs=%+v", got.Steps[4].AcceptedReviewRefs)
	}
}

func TestOperationalDirectorPlanStateV0RechazaAcceptedReviewRefsFueraDePasosCausales(t *testing.T) {
	state := operationalDirectorPlanStateForTestV0()
	state.Steps[1].AcceptedReviewRefs = []string{"accepted-review-ref-plan-state-001"}
	if _, err := NewOperationalDirectorPlanStateV0(state); err == nil {
		t.Fatal("err=nil, want accepted_review_refs fuera de pasos causales")
	}
}

func TestOperationalDirectorPlanStateV0RechazaAcceptedReviewRefsCausalesSinReviewPropietaria(t *testing.T) {
	state := operationalDirectorPlanStateWithReviewAcceptedForTestV0()
	state.Steps[3].AcceptedReviewRefs = []string{"accepted-review-ref-plan-state-fantasma"}
	if _, err := NewOperationalDirectorPlanStateV0(state); err == nil {
		t.Fatal("err=nil, want accepted_review_refs causal sin review propietaria")
	}
}

func TestOperationalDirectorPlanStateV0RechazaReviewAceptadoSinRefsCausales(t *testing.T) {
	state := operationalDirectorPlanStateWithReviewAcceptedForTestV0()
	state.Steps[2].AcceptedReviewRefs = nil
	if _, err := NewOperationalDirectorPlanStateV0(state); err == nil {
		t.Fatal("err=nil, want accepted_review_refs requerido")
	}
}

func TestOperationalDirectorPlanStateV0NormalizaReworkReplanRefs(t *testing.T) {
	state := operationalDirectorPlanStateForTestV0()
	state.ActiveStepID = "step-review-deliveries"
	state.Steps[0].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
	state.Steps[1].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
	state.Steps = append(state.Steps, OperationalDirectorPlanStepStateV0{
		StepID:             "step-review-deliveries",
		Kind:               orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
		Status:             orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0,
		WaveRef:            "wave-001",
		CohortRef:          "cohort-001",
		TaskRefs:           []string{"task-ref-plan-state-001"},
		AgentRefs:          []string{"agent-ref-plan-state-001"},
		DeliveryRefs:       []string{"delivery-ref-plan-state-001"},
		ReviewResultRefs:   []string{"review-result-ref-plan-state-001"},
		ReworkRequestRefs:  []string{"", " rework-request-ref-plan-state-001 ", " ", "rework-request-ref-plan-state-001"},
		ReplanDecisionRefs: []string{" ", " replan-ref-plan-state-001 ", "", "replan-ref-plan-state-001"},
	})
	got, err := NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		t.Fatalf("NewOperationalDirectorPlanStateV0: %v", err)
	}
	reviewStep := got.Steps[2]
	if len(reviewStep.ReworkRequestRefs) != 1 ||
		reviewStep.ReworkRequestRefs[0] != "rework-request-ref-plan-state-001" ||
		len(reviewStep.ReplanDecisionRefs) != 1 ||
		reviewStep.ReplanDecisionRefs[0] != "replan-ref-plan-state-001" {
		t.Fatalf("reviewStep=%+v", reviewStep)
	}
}

func TestOperationalDirectorPlanStateV0RechazaAcceptedReviewRefDuplicado(t *testing.T) {
	state := operationalDirectorPlanStateWithReviewAcceptedForTestV0()
	state.Steps = append(state.Steps, OperationalDirectorPlanStepStateV0{
		StepID:             "step-review-deliveries-duplicate",
		Kind:               orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
		Status:             orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
		WaveRef:            "wave-001",
		CohortRef:          "cohort-001",
		DeliveryRefs:       []string{"delivery-ref-plan-state-duplicate"},
		ReviewResultRefs:   []string{"review-result-ref-plan-state-duplicate"},
		AcceptedReviewRefs: []string{"accepted-review-ref-plan-state-001"},
	})
	if _, err := NewOperationalDirectorPlanStateV0(state); err == nil {
		t.Fatal("err=nil, want accepted_review_ref duplicado")
	}
}

func operationalDirectorPlanStateForTestV0() OperationalDirectorPlanStateV0 {
	return OperationalDirectorPlanStateV0{
		SchemaVersion:    OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:         "operational-director-plan-state-test-001",
		PlanRef:          "operational-director-plan-test-001",
		RequestRef:       "operational-director-request-test-001",
		RunRef:           "run-operational-director-plan-state-001",
		ProjectRef:       "project-operational-director-plan-state-001",
		Mode:             orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:           OperationalDirectorPlanStateActiveV0,
		ActiveStepID:     "step-launch-subagents",
		ActiveWaveRef:    "wave-001",
		ActiveCohortRef:  "cohort-001",
		EvidenceRefs:     []string{"evidence-ref-plan-state-001"},
		RequiredTestRefs: []string{"go test ./..."},
		CorrelationID:    "corr-operational-director-plan-state-001",
		ObservedAt:       "2026-05-17T16:00:00Z",
		Steps: []OperationalDirectorPlanStepStateV0{
			{
				StepID:    "step-launch-subagents",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepRunningV0,
				WaveRef:   "wave-001",
				CohortRef: "cohort-001",
				TaskRefs:  []string{"task-ref-plan-state-001"},
				AgentRefs: []string{
					"agent-ref-plan-state-001",
				},
			},
			{
				StepID:    "step-wait-subagents",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:   "wave-001",
				CohortRef: "cohort-001",
			},
		},
	}
}

func operationalDirectorPlanStateWithReviewAcceptedForTestV0() OperationalDirectorPlanStateV0 {
	state := operationalDirectorPlanStateForTestV0()
	state.ActiveStepID = "step-replan-or-close"
	state.Steps[0].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
	state.Steps[1].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
	state.Steps = append(state.Steps,
		OperationalDirectorPlanStepStateV0{
			StepID:             "step-review-deliveries",
			Kind:               orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
			Status:             orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
			WaveRef:            "wave-001",
			CohortRef:          "cohort-001",
			TaskRefs:           []string{"task-ref-plan-state-001"},
			AgentRefs:          []string{"agent-ref-plan-state-001"},
			DeliveryRefs:       []string{"delivery-ref-plan-state-001"},
			ReviewResultRefs:   []string{"review-result-ref-plan-state-001"},
			AcceptedReviewRefs: []string{"accepted-review-ref-plan-state-001"},
		},
		OperationalDirectorPlanStepStateV0{
			StepID:    "step-replan-or-close",
			Kind:      orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
			Status:    orquestadirectoroperativo.OperationalDirectorStepRunningV0,
			WaveRef:   "wave-001",
			CohortRef: "cohort-001",
			TaskRefs:  []string{"task-ref-plan-state-001"},
			AgentRefs: []string{"agent-ref-plan-state-001"},
		},
	)
	return state
}
