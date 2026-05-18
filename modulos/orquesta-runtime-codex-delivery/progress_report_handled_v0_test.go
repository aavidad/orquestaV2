package orquestaruntimecodexdelivery

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestCodexProgressReportAlreadyHandledV0TimeoutNoQuedaTapadoPorStalledPrevio(t *testing.T) {
	const agentRef = "agent-request-ref-progress-timeout-001"
	report := orquestaruntime.AgentProgressReportV0{
		ReportID:         "agent-progress-report-ref-progress-timeout-001",
		RunID:            "run-ref-progress-timeout-001",
		AgentRequestID:   agentRef,
		Status:           orquestaruntime.AgentStalledV0,
		BudgetStatus:     orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0,
		DecisionRequired: true,
		Summary:          "Tiempo excedido sin actividad reciente.",
	}
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID: "run-ref-progress-timeout-001",
		AgentAssessments: []string{
			orquestacoreworkflow.AgentAssessmentProjectionRefV0(
				orquestacoreworkflow.AgentWorkAssessedPayloadV0{
					AssessmentRef:  "assessment-ref-" + report.ReportID,
					PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
					AgentRequestID: agentRef,
					Verdict:        orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
					Action:         orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
					Severity:       orquestacoreworkflow.AgentAssessmentSeverityMediumV0,
					Summary:        "Stalled previo ya preguntado.",
				},
			),
		},
	}

	if codexProgressReportAlreadyHandledV0(run, report) {
		t.Fatalf("timeout sin actividad no debe quedar suprimido por assessment stalled previo")
	}

	run.StoppedAgents = []string{agentRef}
	if !codexProgressReportAlreadyHandledV0(run, report) {
		t.Fatalf("timeout ya parado debe quedar tratado")
	}
}
