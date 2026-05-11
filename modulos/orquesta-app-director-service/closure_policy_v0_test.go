package orquestaappdirectorservice

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestGuardStartAppDirectorClosurePolicyV0BloqueaAppCompletaSinCodigo(t *testing.T) {
	err := guardStartAppDirectorClosurePolicyV0(
		orquestacoreworkflow.OrchestrationRunV0{
			RunID:          "run-ref-closure-policy-001",
			PhaseArtifacts: []string{"artifact-ref-docs-001#phase:documentacion"},
		},
		closurePolicyFinalValidationDecisionV0(orquestafactory.RequestKindCrearAppCompletaV0, orquestafactory.ExecutionModeNormalV0),
	)
	if err == nil {
		t.Fatalf("esperaba bloqueo de cierre sin programacion")
	}
	if got := err.Error(); got != "app_director_service_invalido: director_decision.request_policy.contratos" {
		t.Fatalf("error=%q", got)
	}
}

func TestGuardStartAppDirectorClosurePolicyV0PermiteDebugReducido(t *testing.T) {
	err := guardStartAppDirectorClosurePolicyV0(
		orquestacoreworkflow.OrchestrationRunV0{
			RunID:          "run-ref-closure-policy-debug-001",
			PhaseArtifacts: []string{"artifact-ref-docs-001#phase:documentacion"},
		},
		closurePolicyFinalValidationDecisionV0(orquestafactory.RequestKindCrearAppCompletaV0, orquestafactory.ExecutionModeDebugV0),
	)
	if err != nil {
		t.Fatalf("debug no debe bloquear cierre reducido: %v", err)
	}
}

func TestGuardStartAppDirectorClosurePolicyV0RechazaPoliticaDesconocida(t *testing.T) {
	err := guardStartAppDirectorClosurePolicyV0(
		orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-closure-policy-unknown-001"},
		closurePolicyFinalValidationDecisionV0("flujo_inventado", orquestafactory.ExecutionModeNormalV0),
	)
	if err == nil || err.Error() != "app_director_service_invalido: director_decision.request_policy.request_kind" {
		t.Fatalf("error=%v", err)
	}
}

func TestGuardStartAppDirectorClosurePolicyV0ExigeResultadoEnPeticionesParciales(t *testing.T) {
	err := guardStartAppDirectorClosurePolicyV0(
		orquestacoreworkflow.OrchestrationRunV0{RunID: "run-ref-closure-policy-result-001"},
		closurePolicyFinalValidationDecisionV0(orquestafactory.RequestKindAnalizarAppV0, orquestafactory.ExecutionModeNormalV0),
	)
	if err == nil || err.Error() != "app_director_service_invalido: director_decision.request_policy.resultado" {
		t.Fatalf("error=%v", err)
	}
}

func TestGuardStartAppDirectorClosurePolicyV0PermiteAppCompletaConEvidencia(t *testing.T) {
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:             "run-ref-closure-policy-complete-001",
		FunctionContracts: []string{"contract:function:agenda:v0"},
		Tasks:             []string{"task-ref-agenda-001"},
		Deliveries:        []string{"delivery-ref-agenda-001"},
		ClosedTasks:       []string{"task-ref-agenda-001"},
		AcceptedReviews:   []string{"accepted-review-ref-agenda-001"},
		Validations:       []string{"validation-ref-agenda-001"},
	}
	err := guardStartAppDirectorClosurePolicyV0(
		run,
		closurePolicyCloseRunDecisionV0(orquestafactory.RequestKindCrearAppCompletaV0, orquestafactory.ExecutionModeNormalV0),
	)
	if err != nil {
		t.Fatalf("cierre con evidencia suficiente no debe bloquear: %v", err)
	}
}

func closurePolicyFinalValidationDecisionV0(
	kind string,
	mode string,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		CommandType: orquestadirectoragent.DirectorAgentCommandRegisterFinalValidationV0,
		RegisterFinalValidation: &orquestadirectoragent.DirectorAgentFinalValidationCommandV0{
			RequestKind:   kind,
			ExecutionMode: mode,
		},
	}
}

func closurePolicyCloseRunDecisionV0(
	kind string,
	mode string,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		CommandType: orquestadirectoragent.DirectorAgentCommandCloseRunV0,
		CloseRun: &orquestadirectoragent.DirectorAgentCloseRunCommandV0{
			RequestKind:   kind,
			ExecutionMode: mode,
		},
	}
}
