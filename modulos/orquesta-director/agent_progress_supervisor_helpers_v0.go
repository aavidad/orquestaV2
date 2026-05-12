package orquestadirector

import (
	"strconv"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

var agentProgressSupervisorForbiddenFragmentsV0 = strings.Fields(
	"secret secreto token password credential credencial api_key oauth transcript prompt completion " +
		"raw_text full_text full_context contexto_completo massive_context db database sql dsn " +
		"table tabla provider proveedor model modelo home openai anthropic claude gpt gemini " +
		"mysql postgres sqlite mongodb redis",
)

func assessmentFromAgentProgressReportV0(
	input AgentProgressSupervisionInputV0,
) orquestacoreworkflow.AssessAgentWorkCommandPayloadV0 {
	verdict, action, severity := assessmentDecisionFromProgressStatusV0(input)
	return orquestacoreworkflow.AssessAgentWorkCommandPayloadV0{
		AssessmentRef:  input.AssessmentRef,
		PhaseID:        input.PhaseID,
		AgentRequestID: input.Report.AgentRequestID,
		TaskRef:        input.TaskRef,
		DeliveryRef:    input.DeliveryRef,
		Verdict:        verdict,
		Action:         action,
		Severity:       severity,
		Summary:        assessmentSummaryFromProgressReportV0(input.Report),
		EvidenceRefs:   supervisionEvidenceRefsV0(input.Report),
	}
}

func assessmentDecisionFromProgressStatusV0(input AgentProgressSupervisionInputV0) (string, string, string) {
	if agentProgressCapacityLimitedV0(input.Report) {
		if !agentProgressStopAllowedV0(input) {
			return orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
				orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
				orquestacoreworkflow.AgentAssessmentSeverityHighV0
		}
		return orquestacoreworkflow.AgentAssessmentVerdictCapacityLimitedV0,
			orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
			orquestacoreworkflow.AgentAssessmentSeverityHighV0
	}
	switch input.Report.Status {
	case orquestaruntime.AgentStalledV0:
		return orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
			orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
			stalledSeverityFromCountersV0(input.Report)
	case orquestaruntime.AgentLoopDetectedV0:
		if !agentProgressStopAllowedV0(input) {
			return orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
				orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
				orquestacoreworkflow.AgentAssessmentSeverityCriticalV0
		}
		return orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0,
			orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
			orquestacoreworkflow.AgentAssessmentSeverityCriticalV0
	case orquestaruntime.AgentStoppedV0:
		if !agentProgressStopAllowedV0(input) {
			return orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
				orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
				orquestacoreworkflow.AgentAssessmentSeverityHighV0
		}
		return orquestacoreworkflow.AgentAssessmentVerdictGarbageV0,
			orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
			orquestacoreworkflow.AgentAssessmentSeverityHighV0
	default:
		return orquestacoreworkflow.AgentAssessmentVerdictAcceptableV0,
			orquestacoreworkflow.AgentAssessmentActionContinueV0,
			orquestacoreworkflow.AgentAssessmentSeverityLowV0
	}
}

func agentProgressCapacityLimitedV0(report orquestaruntime.AgentProgressReportV0) bool {
	return report.BudgetStatus == orquestaruntime.AgentProgressBudgetCapacityLimitedV0
}

func agentProgressStopAllowedV0(input AgentProgressSupervisionInputV0) bool {
	return input.StopAllowed == nil || *input.StopAllowed
}

func stalledSeverityFromCountersV0(report orquestaruntime.AgentProgressReportV0) string {
	if report.NoProgressTicks >= stalledHighNoProgressTicksV0 ||
		report.RepeatedActionCount >= stalledHighRepeatedActionCountV0 {
		return orquestacoreworkflow.AgentAssessmentSeverityHighV0
	}
	return orquestacoreworkflow.AgentAssessmentSeverityMediumV0
}

func assessmentSummaryFromProgressReportV0(report orquestaruntime.AgentProgressReportV0) string {
	switch report.Status {
	case orquestaruntime.AgentStalledV0:
		return compactCounterSummaryV0("Estancamiento observado; consultar direccion.", report)
	case orquestaruntime.AgentLoopDetectedV0:
		return compactCounterSummaryV0("Bucle detectado; detener agente logico.", report)
	case orquestaruntime.AgentStoppedV0:
		if agentProgressCapacityLimitedV0(report) {
			return "Capacidad externa limitada; cerrar agente y replanificar."
		}
		return "Proceso detenido sin entrega; detener agente logico y replanificar."
	default:
		return "Progreso aceptable observado; continuar."
	}
}

func compactCounterSummaryV0(prefix string, report orquestaruntime.AgentProgressReportV0) string {
	return prefix +
		" ticks_sin_avance=" + strconv.Itoa(report.NoProgressTicks) +
		" acciones_repetidas=" + strconv.Itoa(report.RepeatedActionCount) + "."
}

func askDirectorPayloadFromStalledReportV0(
	input AgentProgressSupervisionInputV0,
	assessment orquestacoreworkflow.AssessAgentWorkCommandPayloadV0,
) orquestacoreworkflow.AskDirectorCommandPayloadV0 {
	return orquestacoreworkflow.AskDirectorCommandPayloadV0{
		QuestionID:   input.QuestionID,
		SourceGroup:  agentProgressSupervisorSourceGroupV0,
		TargetGroup:  orquestacoreworkflow.DirectorQuestionTargetDirectorV0,
		Summary:      assessment.Summary,
		Options:      []string{"continuar", "replanificar", "detener_agente"},
		EvidenceRefs: assessment.EvidenceRefs,
		Blocking:     false,
	}
}

func askDirectorMetaFromSupervisionV0(
	meta orquestacoreworkflow.OrchestrationCommandMetaV0,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	meta.CommandID += "-ask-director"
	meta.IdempotencyKey += "-ask-director"
	return meta
}

func supervisionEvidenceRefsV0(report orquestaruntime.AgentProgressReportV0) []string {
	candidates := append([]string{report.ReportID}, report.EvidenceRefs...)
	result := make([]string, 0, len(candidates))
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		ref := strings.TrimSpace(candidate)
		if ref == "" || supervisorHasForbiddenDetailV0(ref) {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		result = append(result, ref)
	}
	return result
}

func compactSupervisorStringsV0(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func supervisorHasForbiddenDetailV0(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	for _, fragment := range agentProgressSupervisorForbiddenFragmentsV0 {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}
