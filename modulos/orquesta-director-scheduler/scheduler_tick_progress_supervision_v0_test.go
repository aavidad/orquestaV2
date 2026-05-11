package orquestadirectorscheduler

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestBuildDirectorSchedulerTickV0ProgressLoopPriorityOverWorkCandidate(t *testing.T) {
	input := validSchedulerTickInputWithProgressV0(orquestaruntime.AgentLoopDetectedV0)

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
	assertSchedulerCommandTypesV0(t, plan, orquestacoreworkflow.OrchestrationCommandAssessAgentWorkV0)
}

func TestBuildDirectorSchedulerTickV0ProgressStalledAsksDirectorBeforeWork(t *testing.T) {
	input := validSchedulerTickInputWithProgressV0(orquestaruntime.AgentStalledV0)

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 2)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandAssessAgentWorkV0,
		orquestacoreworkflow.OrchestrationCommandAskDirectorV0,
	)
}

func TestBuildDirectorSchedulerTickV0ProgressDoesNotRepeatAssessment(t *testing.T) {
	input := validSchedulerTickInputWithProgressV0(orquestaruntime.AgentProgressingV0)
	input.Snapshot.AgentAssessments = []string{"assessment-ref-progress-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusQuiescentV0, 0)
}

func TestBuildDirectorSchedulerTickV0ProgressQuestionPendingWaits(t *testing.T) {
	input := validSchedulerTickInputWithProgressV0(orquestaruntime.AgentStalledV0)
	input.Snapshot.AgentAssessments = []string{"assessment-ref-progress-scheduler-001"}
	input.Snapshot.DirectorQuestions = []string{"question-ref-progress-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingDirectorQuestionPendingV0)
}

func TestBuildDirectorSchedulerTickV0ProgressQuestionAnsweredQuiescent(t *testing.T) {
	input := validSchedulerTickInputWithProgressV0(orquestaruntime.AgentStalledV0)
	input.Snapshot.AgentAssessments = []string{"assessment-ref-progress-scheduler-001"}
	input.Snapshot.DirectorAnsweredQuestions = []string{"question-ref-progress-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusQuiescentV0, 0)
}

func TestBuildDirectorSchedulerTickV0StoppedProgressCandidateKeepsInFlightWait(t *testing.T) {
	input := validSchedulerTickInputWithProgressV0(orquestaruntime.AgentLoopDetectedV0)
	input.Snapshot.StartedAgents = []string{"agent-ref-scheduler-001", "agent-ref-scheduler-002"}
	input.Snapshot.StoppedAgents = []string{"agent-ref-scheduler-001"}
	input.Snapshot.Agents = []string{"agent-ref-scheduler-001", "agent-ref-scheduler-002"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingAgentDeliveryPendingV0)
}

func TestBuildDirectorSchedulerTickV0ProgressAllowsPartialStopReconstruction(t *testing.T) {
	input := validSchedulerTickInputWithProgressV0(orquestaruntime.AgentLoopDetectedV0)
	input.Snapshot.AgentAssessments = []string{"assessment-ref-progress-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
	assertSchedulerCommandTypesV0(t, plan, orquestacoreworkflow.OrchestrationCommandAssessAgentWorkV0)
}

func TestBuildDirectorSchedulerTickV0PendingOutboxBlocksProgressSupervision(t *testing.T) {
	input := validSchedulerTickInputWithProgressV0(orquestaruntime.AgentLoopDetectedV0)
	input.Snapshot.PendingOutboxRefs = []string{"outbox-ref-progress-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingOutboxPendingV0)
}

func TestBuildDirectorSchedulerTickV0LeasePriorityOverProgressSupervision(t *testing.T) {
	input := validSchedulerTickInputWithLeaseV0(orquestacoreworkflow.AgentLeaseActionStopAgentV0)
	input.ProgressSupervisionCandidates = []SchedulableProgressSupervisionCandidateV0{
		validSchedulableProgressSupervisionCandidateV0(orquestaruntime.AgentLoopDetectedV0),
	}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 2)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRegisterAgentLeaseExpiredV0,
		orquestacoreworkflow.OrchestrationCommandStopAgentV0,
	)
}

func TestBuildDirectorSchedulerTickV0RejectsForeignRunProgressCandidate(t *testing.T) {
	input := validSchedulerTickInputWithProgressV0(orquestaruntime.AgentLoopDetectedV0)
	input.ProgressSupervisionCandidates[0].SupervisionInput.Report.RunID = "run-externo-001"

	_, err := BuildDirectorSchedulerTickV0(input)
	assertSchedulerTickErrorV0(t, err, "progress_supervision_candidates.report.run_id")
}

func TestBuildDirectorSchedulerTickV0ProgressAcceptsOpaqueEvidenceRefsWithOperationalTokens(t *testing.T) {
	input := validSchedulerTickInputWithProgressV0(orquestaruntime.AgentLoopDetectedV0)
	input.ProgressSupervisionCandidates[0].EvidenceRefs = []string{
		"runtime-process-session-ref-progress-candidate-001",
	}
	input.ProgressSupervisionCandidates[0].SupervisionInput.Report.EvidenceRefs = []string{
		"runtime-process-session-ref-progress-report-001",
	}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
}

func TestBuildDirectorSchedulerTickV0ProgressRejectsForbiddenSummaryDetails(t *testing.T) {
	tests := []string{
		"model operativo seleccionado",
		"home operativo seleccionado",
	}

	for _, summary := range tests {
		t.Run(summary, func(t *testing.T) {
			input := validSchedulerTickInputWithProgressV0(orquestaruntime.AgentLoopDetectedV0)
			input.ProgressSupervisionCandidates[0].SupervisionInput.Report.Summary = summary

			_, err := BuildDirectorSchedulerTickV0(input)
			assertSchedulerTickErrorV0(t, err, "payload")
		})
	}
}

func validSchedulerTickInputWithProgressV0(
	status orquestaruntime.AgentProgressStatusV0,
) DirectorSchedulerTickInputV0 {
	input := validSchedulerTickInputV0()
	input.Snapshot.Agents = []string{"agent-ref-scheduler-001"}
	input.ProgressSupervisionCandidates = []SchedulableProgressSupervisionCandidateV0{
		validSchedulableProgressSupervisionCandidateV0(status),
	}
	return input
}

func validSchedulableProgressSupervisionCandidateV0(
	status orquestaruntime.AgentProgressStatusV0,
) SchedulableProgressSupervisionCandidateV0 {
	return SchedulableProgressSupervisionCandidateV0{
		CandidateRef:     "progress-supervision-candidate-ref-001",
		SupervisionInput: validProgressSupervisionInputForSchedulerV0(status),
		EvidenceRefs:     []string{"evidence-ref-progress-candidate-001"},
	}
}

func validProgressSupervisionInputForSchedulerV0(
	status orquestaruntime.AgentProgressStatusV0,
) orquestadirector.AgentProgressSupervisionInputV0 {
	return orquestadirector.AgentProgressSupervisionInputV0{
		CommandMeta:   schedulerMetaV0("cmd-progress-scheduler-001", "progress-supervision"),
		Report:        validProgressReportForSchedulerV0(status),
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:       "task-ref-scheduler-001",
		AssessmentRef: "assessment-ref-progress-scheduler-001",
		QuestionID:    "question-ref-progress-scheduler-001",
	}
}

func validProgressReportForSchedulerV0(
	status orquestaruntime.AgentProgressStatusV0,
) orquestaruntime.AgentProgressReportV0 {
	return orquestaruntime.AgentProgressReportV0{
		ReportID:            "agent-progress-report-ref-scheduler-001",
		RunID:               "run-scheduler-001",
		AgentRequestID:      "agent-ref-scheduler-001",
		Status:              status,
		NoProgressTicks:     4,
		RepeatedActionCount: 3,
		Summary:             "Evidencia compacta de progreso del agente.",
		EvidenceRefs:        []string{"evidence-ref-progress-001"},
	}
}
