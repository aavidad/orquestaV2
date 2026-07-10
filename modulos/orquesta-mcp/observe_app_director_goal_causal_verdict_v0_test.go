package orquestamcp

import (
	"context"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

type mcpObserveGoalRunningObserverForTestV0 struct {
	result orquestagoal.GoalWorkResultV0
}

func (observer mcpObserveGoalRunningObserverForTestV0) ObserveGoalWorkV0(
	_ context.Context,
	_ orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	return observer.result, nil
}

func mcpObserveGoalCausalVerdictStateForTestV0(t *testing.T, runRef string) orquestagoal.GoalWorkStateV0 {
	t.Helper()
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       "goal-ref-" + runRef,
			RunRef:        runRef,
			Objective:     "Publicar veredicto causal en observe_goal.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "docs/causal-verdict"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         "goal-ref-" + runRef,
			ExternalGoalRef: "thread-ref-" + runRef,
			EvidenceRefs:    []string{"evidence-ref-" + runRef + "-launch"},
		},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	return state
}

func mcpObserveGoalCausalVerdictExecutorForTestV0(
	t *testing.T,
	state orquestagoal.GoalWorkStateV0,
	evidencias []orquestaestadovivo.EvidenciaEstadoV0,
) MCPObserveAppDirectorGoalToolExecutorV0 {
	t.Helper()
	store := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := store.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	executor := NewMCPObserveAppDirectorGoalToolExecutorV0(orquestaappdirectorservice.StartAppDirectorPortsV0{
		GoalStateStore: store,
		GoalObserver: mcpObserveGoalRunningObserverForTestV0{result: orquestagoal.GoalWorkResultV0{
			SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         state.GoalRef,
			ExternalGoalRef: state.ExternalGoalRef,
		}},
	})
	executor.EstadoVivoSource = &fakeMCPAutoprogrammingEstadoVivoSourceV0{evidencias: evidencias}
	return executor
}

func TestMCPObserveAppDirectorGoalExecuteProcesoMuertoNoPublicaRunningV0(t *testing.T) {
	runRef := "run-ref-observe-causal-dead-001"
	state := mcpObserveGoalCausalVerdictStateForTestV0(t, runRef)
	executor := mcpObserveGoalCausalVerdictExecutorForTestV0(t, state, []orquestaestadovivo.EvidenciaEstadoV0{
		{RunRef: runRef, Fuente: "state", Estado: "running"},
		{
			RunRef: runRef, Fuente: "snapshot", Scope: orquestaestadovivo.ScopeGoalExecutionV0,
			RuntimeIdentityRef: "runtime-ref-causal-dead-001", RuntimeObservationAttempted: true,
			RuntimeObservado: true, ProcesoVivo: false,
			EvidenceRefs: []string{"evidence-ref-causal-dead-snapshot"},
		},
	})

	result, err := executor.Execute(context.Background(), MCPObserveAppDirectorGoalToolInputV0{
		RequestID: "req-observe-causal-dead-001",
		RunRef:    runRef,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.CausalVerdict != string(orquestaestadovivo.VeredictoProcessDeadStateStaleV0) ||
		result.CausalReasonCode != orquestaestadovivo.RazonVeredictoProcesoMuertoEstadoStaleV0 ||
		result.GoalStatus == orquestagoal.GoalStatusRunningV0 ||
		result.RecommendedAction != mcpObserveAppDirectorGoalActionReconcileGoalStateV0 ||
		!containsStringMCPV0(result.EvidenceRefs, "evidence-ref-causal-dead-snapshot") {
		t.Fatalf("proceso muerto publicado como running: %+v", result)
	}
}

func TestMCPObserveAppDirectorGoalExecuteTerminalDurableNoPublicaRunningV0(t *testing.T) {
	runRef := "run-ref-observe-causal-terminal-001"
	state := mcpObserveGoalCausalVerdictStateForTestV0(t, runRef)
	executor := mcpObserveGoalCausalVerdictExecutorForTestV0(t, state, []orquestaestadovivo.EvidenciaEstadoV0{
		{
			RunRef: runRef, Fuente: "result", Terminal: true, Aceptado: true,
			EvidenceRefs: []string{"evidence-ref-causal-terminal-result"},
		},
	})

	result, err := executor.Execute(context.Background(), MCPObserveAppDirectorGoalToolInputV0{
		RequestID: "req-observe-causal-terminal-001",
		RunRef:    runRef,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.CausalVerdict != string(orquestaestadovivo.VeredictoTerminalByArtifactV0) ||
		result.CausalReasonCode != orquestaestadovivo.RazonVeredictoResultadoDurableTerminalV0 ||
		result.GoalStatus == orquestagoal.GoalStatusRunningV0 ||
		result.RecommendedAction != mcpObserveAppDirectorGoalActionReconcileGoalStateV0 ||
		!containsStringMCPV0(result.EvidenceRefs, "evidence-ref-causal-terminal-result") {
		t.Fatalf("terminal durable publicado como running: %+v", result)
	}
}

func TestMCPObserveAppDirectorGoalExecuteRunningConfirmadoConservaRunningV0(t *testing.T) {
	runRef := "run-ref-observe-causal-live-001"
	state := mcpObserveGoalCausalVerdictStateForTestV0(t, runRef)
	executor := mcpObserveGoalCausalVerdictExecutorForTestV0(t, state, []orquestaestadovivo.EvidenciaEstadoV0{
		{
			RunRef: runRef, Fuente: "snapshot", Scope: orquestaestadovivo.ScopeGoalExecutionV0,
			RuntimeIdentityRef: "runtime-ref-causal-live-001", RuntimeObservationAttempted: true,
			RuntimeObservado: true, ProcesoVivo: true,
		},
	})

	result, err := executor.Execute(context.Background(), MCPObserveAppDirectorGoalToolInputV0{
		RequestID: "req-observe-causal-live-001",
		RunRef:    runRef,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.CausalVerdict != string(orquestaestadovivo.VeredictoRunningConfirmedV0) ||
		result.CausalReasonCode != orquestaestadovivo.RazonVeredictoRuntimeConfirmadoV0 ||
		result.GoalStatus != orquestagoal.GoalStatusRunningV0 ||
		result.RecommendedAction != "observe_later" {
		t.Fatalf("running confirmado alterado: %+v", result)
	}
}

func TestMCPObserveAppDirectorGoalExecuteSinFuenteNoPublicaVeredictoV0(t *testing.T) {
	runRef := "run-ref-observe-causal-nosource-001"
	state := mcpObserveGoalCausalVerdictStateForTestV0(t, runRef)
	store := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := store.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	executor := NewMCPObserveAppDirectorGoalToolExecutorV0(orquestaappdirectorservice.StartAppDirectorPortsV0{
		GoalStateStore: store,
		GoalObserver: mcpObserveGoalRunningObserverForTestV0{result: orquestagoal.GoalWorkResultV0{
			SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         state.GoalRef,
			ExternalGoalRef: state.ExternalGoalRef,
		}},
	})

	result, err := executor.Execute(context.Background(), MCPObserveAppDirectorGoalToolInputV0{
		RequestID: "req-observe-causal-nosource-001",
		RunRef:    runRef,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.CausalVerdict != "" || result.CausalReasonCode != "" ||
		result.GoalStatus != orquestagoal.GoalStatusRunningV0 ||
		result.RecommendedAction != "observe_later" {
		t.Fatalf("comportamiento sin fuente alterado: %+v", result)
	}
}
