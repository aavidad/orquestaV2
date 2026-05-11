package orquestadirectorscheduler

import (
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestBuildDirectorSchedulerTickV0OverBudgetButActiveAsksDirectorReview(t *testing.T) {
	input := validSchedulerTickInputWithProgressV0(orquestaruntime.AgentProgressingV0)
	input.ProgressSupervisionCandidates[0].SupervisionInput.Report.BudgetStatus =
		orquestaruntime.AgentProgressBudgetOverBudgetButActiveV0
	input.ProgressSupervisionCandidates[0].SupervisionInput.Report.BudgetReason =
		"Presupuesto superado con actividad reciente."
	input.ProgressSupervisionCandidates[0].SupervisionInput.StopAllowed = schedulerProgressBudgetBoolPtrV0(true)

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 2)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandAssessAgentWorkV0,
		orquestacoreworkflow.OrchestrationCommandAskDirectorV0,
	)
	assertSchedulerProgressAssessmentV0(t, plan.Commands[0],
		orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
		orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
	)
}

func TestBuildDirectorSchedulerTickV0OverBudgetNoActivityProgrammingCanStopAssessment(t *testing.T) {
	input := validSchedulerTickInputWithProgressV0(orquestaruntime.AgentProgressingV0)
	input.ProgressSupervisionCandidates[0].SupervisionInput.Report.BudgetStatus =
		orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0
	input.ProgressSupervisionCandidates[0].SupervisionInput.Report.BudgetReason =
		"Presupuesto superado sin actividad reciente."
	input.ProgressSupervisionCandidates[0].SupervisionInput.StopAllowed = schedulerProgressBudgetBoolPtrV0(true)

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
	assertSchedulerCommandTypesV0(t, plan, orquestacoreworkflow.OrchestrationCommandAssessAgentWorkV0)
	assertSchedulerProgressAssessmentV0(t, plan.Commands[0],
		orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0,
		orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
	)
}

func TestBuildDirectorSchedulerTickV0ProtectedDirectorBudgetNeverStops(t *testing.T) {
	input := validSchedulerTickInputWithProgressV0(orquestaruntime.AgentProgressingV0)
	input.Snapshot.CurrentPhaseID = string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0)
	input.ProgressSupervisionCandidates[0].SupervisionInput.PhaseID =
		string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0)
	input.ProgressSupervisionCandidates[0].SupervisionInput.Report.BudgetStatus =
		orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0
	input.ProgressSupervisionCandidates[0].SupervisionInput.Report.BudgetReason =
		"Presupuesto superado sin actividad reciente."
	input.ProgressSupervisionCandidates[0].SupervisionInput.StopAllowed = schedulerProgressBudgetBoolPtrV0(true)

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 2)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandAssessAgentWorkV0,
		orquestacoreworkflow.OrchestrationCommandAskDirectorV0,
	)
	assertSchedulerProgressAssessmentV0(t, plan.Commands[0],
		orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
		orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
	)
}

func TestBuildDirectorSchedulerTickV0OverBudgetButActiveQuestionPendingWaits(t *testing.T) {
	input := validSchedulerTickInputWithProgressV0(orquestaruntime.AgentProgressingV0)
	input.ProgressSupervisionCandidates[0].SupervisionInput.Report.BudgetStatus =
		orquestaruntime.AgentProgressBudgetOverBudgetButActiveV0
	input.ProgressSupervisionCandidates[0].SupervisionInput.Report.BudgetReason =
		"Presupuesto superado con actividad reciente."
	input.ProgressSupervisionCandidates[0].SupervisionInput.StopAllowed = schedulerProgressBudgetBoolPtrV0(true)
	input.Snapshot.AgentAssessments = []string{"assessment-ref-progress-scheduler-001"}
	input.Snapshot.DirectorQuestions = []string{"question-ref-progress-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingDirectorQuestionPendingV0)
}

func schedulerProgressBudgetBoolPtrV0(value bool) *bool {
	return &value
}

func assertSchedulerProgressAssessmentV0(
	t *testing.T,
	command orquestacoreworkflow.OrchestrationCommandV0,
	verdict string,
	action string,
) {
	t.Helper()
	var payload orquestacoreworkflow.AssessAgentWorkCommandPayloadV0
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		t.Fatalf("decode assessment payload: %v", err)
	}
	if payload.Verdict != verdict || payload.Action != action {
		t.Fatalf("assessment verdict/action=%s/%s, want %s/%s",
			payload.Verdict, payload.Action, verdict, action)
	}
}
