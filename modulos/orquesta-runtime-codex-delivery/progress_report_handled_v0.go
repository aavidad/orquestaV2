package orquestaruntimecodexdelivery

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func codexProgressReportAlreadyHandledV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	report orquestaruntime.AgentProgressReportV0,
) bool {
	agentRef := strings.TrimSpace(report.AgentRequestID)
	if agentRef == "" {
		return false
	}
	switch report.Status {
	case orquestaruntime.AgentStoppedV0, orquestaruntime.AgentLoopDetectedV0:
		return codexProgressAgentStopHandledV0(run, report, agentRef)
	case orquestaruntime.AgentStalledV0:
		if codexProgressRequiresFreshStopDecisionV0(report) {
			return codexProgressAgentTerminalHandledV0(run, agentRef)
		}
		return codexProgressAssessmentHandledV0(run.AgentAssessments, report, agentRef)
	default:
		if codexProgressRequiresFreshStopDecisionV0(report) {
			return codexProgressAgentTerminalHandledV0(run, agentRef)
		}
		return false
	}
}

func codexProgressRequiresFreshStopDecisionV0(
	report orquestaruntime.AgentProgressReportV0,
) bool {
	return report.BudgetStatus == orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0
}

func codexProgressAgentTerminalHandledV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRef string,
) bool {
	return codexProgressStringInSetV0(run.StoppedAgents, agentRef) ||
		codexProgressStringInSetV0(run.ConfirmedStoppedAgents, agentRef) ||
		codexProgressStringInSetV0(run.FailedAgents, agentRef) ||
		codexProgressStringInSetV0(run.LostAgents, agentRef)
}

func codexProgressAgentStopHandledV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	report orquestaruntime.AgentProgressReportV0,
	agentRef string,
) bool {
	if codexProgressAgentTerminalHandledV0(run, agentRef) {
		return true
	}
	if report.Status == orquestaruntime.AgentLoopDetectedV0 &&
		codexProgressProtectedLoopAssessmentHandledV0(run.AgentAssessments, agentRef) {
		return true
	}
	projection, ok := codexProgressExactAssessmentForReportV0(run.AgentAssessments, report)
	return ok && projection.Action == orquestacoreworkflow.AgentAssessmentActionAskDirectorV0
}

func codexProgressAssessmentHandledV0(
	assessments []string,
	report orquestaruntime.AgentProgressReportV0,
	agentRef string,
) bool {
	_, ok := codexProgressAssessmentForReportV0(assessments, report, agentRef)
	return ok
}

func codexProgressAssessmentForReportV0(
	assessments []string,
	report orquestaruntime.AgentProgressReportV0,
	agentRef string,
) (orquestacoreworkflow.AgentWorkAssessmentProjectionV0, bool) {
	if projection, ok := codexProgressExactAssessmentForReportV0(assessments, report); ok {
		return projection, true
	}
	for _, raw := range assessments {
		projection, ok := orquestacoreworkflow.ParseAgentAssessmentProjectionV0(raw)
		if !ok {
			continue
		}
		if projection.AgentRequestID == agentRef &&
			codexProgressAssessmentMatchesStatusV0(projection, report.Status) {
			return projection, true
		}
	}
	return orquestacoreworkflow.AgentWorkAssessmentProjectionV0{}, false
}

func codexProgressExactAssessmentForReportV0(
	assessments []string,
	report orquestaruntime.AgentProgressReportV0,
) (orquestacoreworkflow.AgentWorkAssessmentProjectionV0, bool) {
	wantAssessmentRef := "assessment-ref-" + strings.TrimSpace(report.ReportID)
	for _, raw := range assessments {
		projection, ok := orquestacoreworkflow.ParseAgentAssessmentProjectionV0(raw)
		if ok && projection.AssessmentRef == wantAssessmentRef {
			return projection, true
		}
	}
	return orquestacoreworkflow.AgentWorkAssessmentProjectionV0{}, false
}

func codexProgressProtectedLoopAssessmentHandledV0(
	assessments []string,
	agentRef string,
) bool {
	for _, raw := range assessments {
		projection, ok := orquestacoreworkflow.ParseAgentAssessmentProjectionV0(raw)
		if !ok || projection.AgentRequestID != agentRef {
			continue
		}
		if projection.Action == orquestacoreworkflow.AgentAssessmentActionAskDirectorV0 &&
			projection.Severity == orquestacoreworkflow.AgentAssessmentSeverityCriticalV0 {
			return true
		}
	}
	return false
}

func codexProgressAssessmentMatchesStatusV0(
	projection orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
	status orquestaruntime.AgentProgressStatusV0,
) bool {
	switch status {
	case orquestaruntime.AgentStalledV0:
		return projection.Action == orquestacoreworkflow.AgentAssessmentActionAskDirectorV0
	case orquestaruntime.AgentLoopDetectedV0, orquestaruntime.AgentStoppedV0:
		return projection.Action == orquestacoreworkflow.AgentAssessmentActionStopAgentV0
	default:
		return false
	}
}

func codexProgressStringInSetV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}
