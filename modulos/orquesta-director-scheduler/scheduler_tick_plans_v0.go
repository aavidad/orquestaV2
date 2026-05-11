package orquestadirectorscheduler

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

func schedulerCommandsPlanV0(
	input DirectorSchedulerTickInputV0,
	commands []orquestacoreworkflow.OrchestrationCommandV0,
) DirectorSchedulerTickPlanV0 {
	return schedulerCommandsWithBlockedPlanV0(input, commands, nil)
}

func schedulerCommandsWithBlockedPlanV0(
	input DirectorSchedulerTickInputV0,
	commands []orquestacoreworkflow.OrchestrationCommandV0,
	blockedRefs []string,
) DirectorSchedulerTickPlanV0 {
	return DirectorSchedulerTickPlanV0{
		TickRef:      input.TickRef,
		RunRef:       input.RunRef,
		Status:       SchedulerTickStatusCommandsReadyV0,
		Commands:     commands,
		BlockedRefs:  compactSchedulerStringsV0(blockedRefs),
		Summary:      "scheduler tick produced workflow commands",
		EvidenceRefs: input.EvidenceRefs,
	}
}

func schedulerWaitingPlanV0(
	input DirectorSchedulerTickInputV0,
	reason SchedulerWaitingReasonV0,
) DirectorSchedulerTickPlanV0 {
	return schedulerWaitingReasonsPlanV0(input, []SchedulerWaitingReasonV0{reason})
}

func schedulerWaitingReasonsPlanV0(
	input DirectorSchedulerTickInputV0,
	reasons []SchedulerWaitingReasonV0,
) DirectorSchedulerTickPlanV0 {
	return DirectorSchedulerTickPlanV0{
		TickRef:        input.TickRef,
		RunRef:         input.RunRef,
		Status:         SchedulerTickStatusWaitingV0,
		WaitingReasons: compactSchedulerWaitingReasonsV0(reasons),
		Summary:        "scheduler tick waiting",
		EvidenceRefs:   input.EvidenceRefs,
	}
}

func schedulerNeedsDirectorPlanV0(
	input DirectorSchedulerTickInputV0,
	reason SchedulerWaitingReasonV0,
) DirectorSchedulerTickPlanV0 {
	return schedulerNeedsDirectorReasonsPlanV0(input, []SchedulerWaitingReasonV0{reason})
}

func schedulerNeedsDirectorReasonsPlanV0(
	input DirectorSchedulerTickInputV0,
	reasons []SchedulerWaitingReasonV0,
) DirectorSchedulerTickPlanV0 {
	return DirectorSchedulerTickPlanV0{
		TickRef:        input.TickRef,
		RunRef:         input.RunRef,
		Status:         SchedulerTickStatusNeedsDirectorV0,
		WaitingReasons: compactSchedulerWaitingReasonsV0(reasons),
		Summary:        "scheduler tick needs director",
		EvidenceRefs:   input.EvidenceRefs,
	}
}

func schedulerNeedsDirectorWithCommandsV0(
	input DirectorSchedulerTickInputV0,
	commands []orquestacoreworkflow.OrchestrationCommandV0,
) DirectorSchedulerTickPlanV0 {
	return DirectorSchedulerTickPlanV0{
		TickRef:      input.TickRef,
		RunRef:       input.RunRef,
		Status:       SchedulerTickStatusNeedsDirectorV0,
		Commands:     commands,
		Summary:      "scheduler tick needs director",
		EvidenceRefs: input.EvidenceRefs,
	}
}

func schedulerBlockedPlanV0(
	input DirectorSchedulerTickInputV0,
	blockedRefs []string,
	commands []orquestacoreworkflow.OrchestrationCommandV0,
) DirectorSchedulerTickPlanV0 {
	return DirectorSchedulerTickPlanV0{
		TickRef:      input.TickRef,
		RunRef:       input.RunRef,
		Status:       SchedulerTickStatusBlockedV0,
		Commands:     commands,
		BlockedRefs:  compactSchedulerStringsV0(blockedRefs),
		Summary:      "scheduler tick blocked",
		EvidenceRefs: input.EvidenceRefs,
	}
}

func schedulerQuiescentPlanV0(input DirectorSchedulerTickInputV0) DirectorSchedulerTickPlanV0 {
	return DirectorSchedulerTickPlanV0{
		TickRef:      input.TickRef,
		RunRef:       input.RunRef,
		Status:       SchedulerTickStatusQuiescentV0,
		Summary:      "scheduler tick quiescent",
		EvidenceRefs: input.EvidenceRefs,
	}
}
