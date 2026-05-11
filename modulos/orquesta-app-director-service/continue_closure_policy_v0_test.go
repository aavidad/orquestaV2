package orquestaappdirectorservice

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func TestContinueAppDirectorV0BloqueaCierreAppCompletaNormalSinEvidencias(t *testing.T) {
	cases := []struct {
		name     string
		phase    orquestacoreworkflow.OrchestrationPhaseIDV0
		decision orquestadirectoragent.DirectorAgentDecisionV0
	}{
		{
			name:     "register_final_validation",
			phase:    orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0,
			decision: serviceContinueFinalValidationDecisionForTestV0("run-continue-closure-block-final-001"),
		},
		{
			name:     "close_run",
			phase:    orquestacoreworkflow.OrchestrationPhaseCierreV0,
			decision: serviceContinueCloseRunDecisionForTestV0("run-continue-closure-block-close-001"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			run := serviceContinueClosureRunForTestV0(tc.decision.RunID, tc.phase)
			store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
			sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
			ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()

			_, err := ContinueAppDirectorV0(
				context.Background(),
				serviceContinueClosureRequestForTestV0(run.RunID),
				serviceContinueClosurePortsForTestV0(store, sink, ledger, tc.decision),
			)
			if err == nil || err.Error() != "app_director_service_invalido: director_decision.request_policy.contratos" {
				t.Fatalf("error=%v", err)
			}
			got, loadErr := store.LoadRunV0(context.Background(), run.RunID)
			if loadErr != nil {
				t.Fatalf("LoadRunV0: %v", loadErr)
			}
			if len(got.Validations) != 0 || len(got.Closures) != 0 {
				t.Fatalf("cierre no debe aplicarse: validations=%v closures=%v", got.Validations, got.Closures)
			}
		})
	}
}

func TestContinueAppDirectorV0PermiteCierreAppCompletaNormalConEvidencias(t *testing.T) {
	t.Run("register_final_validation", func(t *testing.T) {
		run := serviceContinueClosureRunForTestV0(
			"run-continue-closure-pass-final-001",
			orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0,
		)
		run = serviceContinueRunWithFullAppEvidenceForTestV0(run)
		run.Validations = nil
		decision := serviceContinueFinalValidationDecisionForTestV0(run.RunID)
		store, sink, ledger := serviceContinueClosureStoresForTestV0(run)

		result, err := ContinueAppDirectorV0(
			context.Background(),
			serviceContinueClosureRequestForTestV0(run.RunID),
			serviceContinueClosurePortsForTestV0(store, sink, ledger, decision),
		)
		if err != nil {
			t.Fatalf("ContinueAppDirectorV0: %v", err)
		}
		if !serviceStringInSetV0(result.Run.Validations, "validation-ref-continue-final-001") {
			t.Fatalf("validations=%v", result.Run.Validations)
		}
		if !serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventFinalValidationRegisteredV0) {
			t.Fatalf("eventos sin FinalValidationRegistered: %+v", sink.EventsV0())
		}
	})

	t.Run("close_run", func(t *testing.T) {
		run := serviceContinueClosureRunForTestV0(
			"run-continue-closure-pass-close-001",
			orquestacoreworkflow.OrchestrationPhaseCierreV0,
		)
		run = serviceContinueRunWithFullAppEvidenceForTestV0(run)
		decision := serviceContinueCloseRunDecisionForTestV0(run.RunID)
		store, sink, ledger := serviceContinueClosureStoresForTestV0(run)

		result, err := ContinueAppDirectorV0(
			context.Background(),
			serviceContinueClosureRequestForTestV0(run.RunID),
			serviceContinueClosurePortsForTestV0(store, sink, ledger, decision),
		)
		if err != nil {
			t.Fatalf("ContinueAppDirectorV0: %v", err)
		}
		if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
			t.Fatalf("status=%s", result.Run.Status)
		}
		if !serviceStringInSetV0(result.Run.Closures, "closure-ref-continue-001") {
			t.Fatalf("closures=%v", result.Run.Closures)
		}
		if !serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunClosedV0) {
			t.Fatalf("eventos sin RunClosed: %+v", sink.EventsV0())
		}
	})
}

func serviceContinueClosureStoresForTestV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) (
	*orquestacionnucleoapp.InMemoryRunStoreV0,
	*orquestacionnucleoapp.InMemoryEventSinkV0,
	*orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) {
	return orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		orquestacionnucleoapp.NewInMemoryEventSinkV0(),
		orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
}

func serviceContinueClosureRequestForTestV0(runRef string) ContinueAppDirectorRequestV0 {
	return ContinueAppDirectorRequestV0{
		RunRef:            runRef,
		OccurredAt:        "2026-05-10T11:30:00Z",
		CorrelationID:     "corr-" + runRef,
		RequestedBy:       "orquesta-app-director-service-test",
		MaxBursts:         2,
		MaxStepsPerBurst:  1,
		MaxDecisionCycles: 2,
	}
}

func serviceContinueClosurePortsForTestV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
	decision orquestadirectoragent.DirectorAgentDecisionV0,
) StartAppDirectorPortsV0 {
	return StartAppDirectorPortsV0{
		RunStore:               store,
		EventSink:              sink,
		OutboxLedger:           ledger,
		DirectorTaskStore:      orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(),
		DirectorDecisionSource: serviceDirectorDecisionSourceForTestV0{Decisions: []orquestadirectoragent.DirectorAgentDecisionV0{decision}},
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
		},
	}
}

func serviceContinueClosureRunForTestV0(
	runRef string,
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	phases := orquestacoreworkflow.OrchestrationPhaseCatalogV0()
	for i := range phases {
		if phases[i].ID == phase {
			phases[i].Status = orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
		}
	}
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "project-ref-continue-closure-001",
		AppSpecRef:    "app-spec-ref-continue-closure-001",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  phase,
		Phases:        phases,
		LastEventID:   "event-ref-continue-closure-seed-001",
		LastSequence:  40,
	}
}

func serviceContinueRunWithFullAppEvidenceForTestV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	run.FunctionContracts = []string{"contract-function-agenda-v0"}
	run.Tasks = []string{"task-ref-continue-001"}
	run.Deliveries = []string{"delivery-ref-continue-001"}
	run.ClosedTasks = []string{"task-ref-continue-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-continue-001"}
	run.Validations = []string{"validation-ref-continue-final-001"}
	return run
}

func serviceContinueFinalValidationDecisionForTestV0(
	runRef string,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-continue-final-validation-001",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandRegisterFinalValidationV0,
		CommandRef:    "command-ref-continue-final-validation-001",
		Summary:       "Registrar validacion final.",
		EvidenceRefs:  []string{"evidence-ref-continue-final-validation-001"},
		RegisterFinalValidation: &orquestadirectoragent.DirectorAgentFinalValidationCommandV0{
			ValidationRef: "validation-ref-continue-final-001",
			PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0),
			ClosedTaskRef: "task-ref-continue-001",
			Summary:       "Validacion final con tareas cerradas y revision aceptada.",
			RequestKind:   orquestafactory.RequestKindCrearAppCompletaV0,
			ExecutionMode: orquestafactory.ExecutionModeNormalV0,
			EvidenceRefs:  []string{"evidence-ref-continue-final-validation-001"},
		},
	}
}

func serviceContinueCloseRunDecisionForTestV0(
	runRef string,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-continue-close-run-001",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseCierreV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandCloseRunV0,
		CommandRef:    "command-ref-continue-close-run-001",
		Summary:       "Cerrar run.",
		EvidenceRefs:  []string{"evidence-ref-continue-close-run-001"},
		CloseRun: &orquestadirectoragent.DirectorAgentCloseRunCommandV0{
			ClosureRef:    "closure-ref-continue-001",
			PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseCierreV0),
			ValidationRef: "validation-ref-continue-final-001",
			Summary:       "Cierre con contratos, tareas, entregas, revision y validacion.",
			RequestKind:   orquestafactory.RequestKindCrearAppCompletaV0,
			ExecutionMode: orquestafactory.ExecutionModeNormalV0,
			EvidenceRefs:  []string{"evidence-ref-continue-close-run-001"},
		},
	}
}
