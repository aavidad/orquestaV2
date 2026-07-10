package orquestamcp

import (
	"context"
	"testing"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
)

func TestMCPAutoprogrammingStatusExecutorV0ProcesoMuertoPublicaVeredictoYReconcileV0(t *testing.T) {
	runRef := "run-ref-status-causal-dead-001"
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		EstadoVivoSource: &fakeMCPAutoprogrammingEstadoVivoSourceV0{
			evidencias: []orquestaestadovivo.EvidenciaEstadoV0{
				{RunRef: runRef, Fuente: "state", Estado: "running"},
				{
					RunRef:                      runRef,
					Fuente:                      "process_snapshot",
					Scope:                       orquestaestadovivo.ScopeGoalExecutionV0,
					RuntimeIdentityRef:          "runtime-ref-status-causal-dead-001",
					RuntimeObservationAttempted: true,
					RuntimeObservado:            true,
					ProcesoVivo:                 false,
					EvidenceRefs:                []string{"evidence-ref-status-causal-dead-snapshot"},
				},
			},
		},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RunRef:     runRef,
		OccurredAt: "2026-07-10T10:00:00Z",
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.CausalVerdict != string(orquestaestadovivo.VeredictoProcessDeadStateStaleV0) ||
		result.CausalReasonCode != orquestaestadovivo.RazonVeredictoProcesoMuertoEstadoStaleV0 {
		t.Fatalf("veredicto causal no publicado: %+v", result)
	}
	if !hasMCPAutoprogrammingActionCodeV0(result.StaleRunning, mcpAutoprogrammingActionEstadoVivoReconcileGoalStateV0) {
		t.Fatalf("stale_running sin reconcile: %+v", result.StaleRunning)
	}
	for _, action := range result.StaleRunning {
		if action.Code != mcpAutoprogrammingActionEstadoVivoReconcileGoalStateV0 {
			continue
		}
		if action.RecommendedAction != mcpObserveAppDirectorGoalActionReconcileGoalStateV0 ||
			action.CausalVerdict != string(orquestaestadovivo.VeredictoProcessDeadStateStaleV0) ||
			action.CausalReasonCode == "" ||
			action.RunRef != runRef {
			t.Fatalf("actionable reconcile invalido: %+v", action)
		}
	}
	if result.QueueHealth != nil && result.QueueHealth.RunningLive != 0 {
		t.Fatalf("no debe publicar running vivo: %+v", result.QueueHealth)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0RunningConfirmadoNoPideReconcileV0(t *testing.T) {
	runRef := "run-ref-status-causal-live-001"
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		EstadoVivoSource: &fakeMCPAutoprogrammingEstadoVivoSourceV0{
			evidencias: []orquestaestadovivo.EvidenciaEstadoV0{
				{RunRef: runRef, Fuente: "state", Estado: "running"},
				{
					RunRef:                      runRef,
					Fuente:                      "process_snapshot",
					Scope:                       orquestaestadovivo.ScopeGoalExecutionV0,
					RuntimeIdentityRef:          "runtime-ref-status-causal-live-001",
					RuntimeObservationAttempted: true,
					RuntimeObservado:            true,
					ProcesoVivo:                 true,
				},
			},
		},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RunRef:     runRef,
		OccurredAt: "2026-07-10T10:00:00Z",
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.CausalVerdict != string(orquestaestadovivo.VeredictoRunningConfirmedV0) ||
		result.CausalReasonCode != orquestaestadovivo.RazonVeredictoRuntimeConfirmadoV0 {
		t.Fatalf("veredicto causal no publicado: %+v", result)
	}
	if hasMCPAutoprogrammingActionCodeV0(result.StaleRunning, mcpAutoprogrammingActionEstadoVivoReconcileGoalStateV0) {
		t.Fatalf("running confirmado no debe pedir reconcile: %+v", result.StaleRunning)
	}
}
