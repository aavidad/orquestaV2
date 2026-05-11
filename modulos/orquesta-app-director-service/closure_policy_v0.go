package orquestaappdirectorservice

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

func guardStartAppDirectorClosurePolicyV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	decision orquestadirectoragent.DirectorAgentDecisionV0,
) error {
	kind, mode, ok := directorDecisionClosurePolicyV0(decision)
	if !ok {
		return nil
	}
	if !orquestafactory.RequestKindSupportedV0(kind) {
		return AppDirectorServiceIssueV0{
			Field: "director_decision.request_policy.request_kind",
		}
	}
	if !orquestafactory.ExecutionModeSupportedV0(mode) {
		return AppDirectorServiceIssueV0{
			Field: "director_decision.request_policy.execution_mode",
		}
	}
	policy := orquestafactory.ResolveRequestPolicyV0(kind, mode)
	if policy.AllowsReducedScope {
		return nil
	}
	missing := missingClosurePolicyEvidenceV0(policy, run, decision.CommandType)
	if len(missing) == 0 {
		return nil
	}
	return AppDirectorServiceIssueV0{
		Field: "director_decision.request_policy." + missing[0],
	}
}

func directorDecisionClosurePolicyV0(
	decision orquestadirectoragent.DirectorAgentDecisionV0,
) (string, string, bool) {
	switch decision.CommandType {
	case orquestadirectoragent.DirectorAgentCommandRegisterFinalValidationV0:
		if decision.RegisterFinalValidation == nil {
			return "", "", false
		}
		payload := decision.RegisterFinalValidation
		return closurePolicyValueV0(payload.RequestKind, orquestafactory.DefaultRequestKindV0),
			closurePolicyValueV0(payload.ExecutionMode, orquestafactory.DefaultExecutionModeV0),
			true
	case orquestadirectoragent.DirectorAgentCommandCloseRunV0:
		if decision.CloseRun == nil {
			return "", "", false
		}
		payload := decision.CloseRun
		return closurePolicyValueV0(payload.RequestKind, orquestafactory.DefaultRequestKindV0),
			closurePolicyValueV0(payload.ExecutionMode, orquestafactory.DefaultExecutionModeV0),
			true
	default:
		return "", "", false
	}
}

func closurePolicyValueV0(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}

func missingClosurePolicyEvidenceV0(
	policy orquestafactory.RequestPolicyV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	commandType string,
) []string {
	switch policy.RequestKind {
	case orquestafactory.RequestKindCrearAppCompletaV0:
		return missingFullAppClosureEvidenceV0(run, commandType)
	case orquestafactory.RequestKindProgramarModuloV0,
		orquestafactory.RequestKindModificarAppExistenteV0,
		orquestafactory.RequestKindMigracionRefactorV0:
		return missingProgrammingClosureEvidenceV0(run)
	case orquestafactory.RequestKindDocumentarAppV0,
		orquestafactory.RequestKindPlanificarAppV0,
		orquestafactory.RequestKindBrainstormingArquitecturaV0,
		orquestafactory.RequestKindDeployV0:
		return missingArtifactClosureEvidenceV0(run)
	default:
		return missingOutcomeClosureEvidenceV0(run)
	}
}

func missingFullAppClosureEvidenceV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	commandType string,
) []string {
	missing := []string{}
	if len(compactServiceRefsV0(run.FunctionContracts)) == 0 {
		missing = append(missing, "contratos")
	}
	if len(compactServiceRefsV0(run.Tasks)) == 0 {
		missing = append(missing, "programacion_tareas")
	}
	if len(compactServiceRefsV0(run.Deliveries)) == 0 {
		missing = append(missing, "programacion_entregas")
	}
	if len(compactServiceRefsV0(run.ClosedTasks)) == 0 {
		missing = append(missing, "programacion_cerrada")
	}
	if len(compactServiceRefsV0(run.AcceptedReviews)) == 0 {
		missing = append(missing, "revision_final")
	}
	if commandType == orquestadirectoragent.DirectorAgentCommandCloseRunV0 &&
		len(compactServiceRefsV0(run.Validations)) == 0 {
		missing = append(missing, "validacion_final")
	}
	return missing
}

func missingProgrammingClosureEvidenceV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	missing := []string{}
	if len(compactServiceRefsV0(run.Deliveries)) == 0 {
		missing = append(missing, "programacion_entregas")
	}
	if len(compactServiceRefsV0(run.ClosedTasks)) == 0 {
		missing = append(missing, "programacion_cerrada")
	}
	return missing
}

func missingArtifactClosureEvidenceV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	if len(compactServiceRefsV0(run.PhaseArtifacts)) == 0 &&
		len(compactServiceRefsV0(run.Deliveries)) == 0 {
		return []string{"artefactos"}
	}
	return nil
}

func missingOutcomeClosureEvidenceV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	total := len(compactServiceRefsV0(run.PhaseArtifacts)) +
		len(compactServiceRefsV0(run.Deliveries)) +
		len(compactServiceRefsV0(run.ReviewResults)) +
		len(compactServiceRefsV0(run.QualityGates)) +
		len(compactServiceRefsV0(run.Decisions))
	if total == 0 {
		return []string{"resultado"}
	}
	return nil
}
