package orquestadirectorscheduler

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

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
