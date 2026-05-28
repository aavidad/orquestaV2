package orquestadirector

import (
	"strconv"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestarails "orquesta/modulos/orquesta-rails"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
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
	if agentProgressAuthConfigBlockerV0(input.Report) {
		return orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
			orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
			orquestacoreworkflow.AgentAssessmentSeverityCriticalV0
	}
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
	if agentProgressNoACKV0(input.Report) {
		return orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
			orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
			orquestacoreworkflow.AgentAssessmentSeverityHighV0
	}
	if agentProgressOverBudgetNoActivityV0(input.Report) {
		if !agentProgressStopAllowedV0(input) {
			return orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
				orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
				orquestacoreworkflow.AgentAssessmentSeverityHighV0
		}
		return orquestacoreworkflow.AgentAssessmentVerdictTimeoutV0,
			orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
			orquestacoreworkflow.AgentAssessmentSeverityHighV0
	}
	if agentProgressOverBudgetButActiveV0(input.Report) {
		return orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
			orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
			orquestacoreworkflow.AgentAssessmentSeverityHighV0
	}
	if agentProgressHasArtifactWithoutAckV0(input.Report) {
		return orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
			orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
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
		if agentProgressHasArtifactWithoutAckV0(input.Report) {
			return orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
				orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
				orquestacoreworkflow.AgentAssessmentSeverityHighV0
		}
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

func agentProgressOverBudgetNoActivityV0(report orquestaruntime.AgentProgressReportV0) bool {
	return report.BudgetStatus == orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0
}

func agentProgressOverBudgetButActiveV0(report orquestaruntime.AgentProgressReportV0) bool {
	return report.BudgetStatus == orquestaruntime.AgentProgressBudgetOverBudgetButActiveV0
}

func agentProgressHasArtifactWithoutAckV0(report orquestaruntime.AgentProgressReportV0) bool {
	for _, ref := range report.EvidenceRefs {
		if strings.TrimSpace(ref) == "evidence-ref-artifact-without-ack" {
			return true
		}
	}
	return false
}

func agentProgressNoACKV0(report orquestaruntime.AgentProgressReportV0) bool {
	for _, ref := range report.EvidenceRefs {
		switch strings.TrimSpace(ref) {
		case "evidence-ref-no-ack", "evidence-ref-no-ack-interrupted":
			return true
		}
	}
	return false
}

func agentProgressAuthConfigBlockerV0(report orquestaruntime.AgentProgressReportV0) bool {
	for _, ref := range report.EvidenceRefs {
		if strings.TrimSpace(ref) == "evidence-ref-auth-config-blocker" {
			return true
		}
	}
	return false
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
	if agentProgressAuthConfigBlockerV0(report) {
		return "Autenticacion externa invalida; pausar reintentos y solicitar reautorizacion."
	}
	if agentProgressCapacityLimitedV0(report) {
		return "Capacidad externa limitada; cerrar agente y replanificar."
	}
	if agentProgressNoACKV0(report) {
		if agentProgressHasArtifactWithoutAckV0(report) {
			return "Entrega sin ACK con artefacto materializado; validar antes de replanificar."
		}
		return "Proceso detenido sin ACK; revisar evidencias antes de replanificar."
	}
	if agentProgressOverBudgetNoActivityV0(report) {
		if agentProgressHasArtifactWithoutAckV0(report) {
			return "Tiempo excedido sin actividad reciente y artefacto sin ACK; detener agente logico y conservar evidencias."
		}
		return "Tiempo excedido sin actividad reciente; detener agente logico."
	}
	if agentProgressOverBudgetButActiveV0(report) {
		return "Tiempo excedido con actividad reciente; consultar direccion antes de intervenir."
	}
	switch report.Status {
	case orquestaruntime.AgentStalledV0:
		if agentProgressHasArtifactWithoutAckV0(report) {
			return "Entrega sin ACK con artefacto materializado; validar antes de replanificar."
		}
		return compactCounterSummaryV0("Estancamiento observado; consultar direccion.", report)
	case orquestaruntime.AgentLoopDetectedV0:
		if agentProgressHasArtifactWithoutAckV0(report) {
			return "Entrega sin ACK con artefacto materializado; validar antes de replanificar."
		}
		return compactCounterSummaryV0("Bucle detectado; detener agente logico.", report)
	case orquestaruntime.AgentStoppedV0:
		if agentProgressHasArtifactWithoutAckV0(report) {
			return "Entrega sin ACK con artefacto materializado; validar antes de replanificar."
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

func registerLostMetaFromSupervisionV0(
	meta orquestacoreworkflow.OrchestrationCommandMetaV0,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	meta.CommandID += "-register-lost"
	meta.IdempotencyKey += "-register-lost"
	return meta
}

func registerLostPayloadFromProgressReportV0(
	report orquestaruntime.AgentProgressReportV0,
	occurredAt string,
) orquestacoreworkflow.RegisterAgentLostCommandPayloadV0 {
	return orquestacoreworkflow.RegisterAgentLostCommandPayloadV0{
		AgentRequestID: strings.TrimSpace(report.AgentRequestID),
		LossRef:        "loss-ref-" + strings.TrimSpace(report.ReportID),
		ReasonCode:     "external_process_ended_without_ack",
		ObservedAt:     strings.TrimSpace(occurredAt),
		Retryable:      true,
		EvidenceRefs:   supervisionEvidenceRefsV0(report),
	}
}

func agentProgressShouldRegisterLostV0(
	report orquestaruntime.AgentProgressReportV0,
	assessment orquestacoreworkflow.AssessAgentWorkCommandPayloadV0,
) bool {
	return report.Status == orquestaruntime.AgentStoppedV0 &&
		assessment.Action == orquestacoreworkflow.AgentAssessmentActionAskDirectorV0 &&
		(agentProgressNoACKV0(report) || agentProgressAuthConfigBlockerV0(report))
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
	if orquestarails.TextContainsOperationalRawDetailForFieldV0(
		"agent_progress_supervisor",
		"evidence_refs",
		value,
	) {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(lower, "prompt-ref") || strings.Contains(lower, "transcript-ref")
}
