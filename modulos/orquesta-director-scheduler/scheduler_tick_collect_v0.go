package orquestadirectorscheduler

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

type schedulerTickCollectorV0 struct {
	input                     DirectorSchedulerTickInputV0
	commands                  []orquestacoreworkflow.OrchestrationCommandV0
	waitingReasons            []SchedulerWaitingReasonV0
	blockedRefs               []string
	workSequenceDecisions     []SchedulerWorkSequenceDecisionV0
	needsDirector             bool
	blocked                   bool
	hasReadyCommands          bool
	tasks                     map[string]bool
	plannedTasks              map[string]bool
	capacityRequests          map[string]bool
	capacityDecisions         map[string]bool
	concurrencyGates          map[string]bool
	agents                    map[string]bool
	startedAgents             map[string]bool
	failedAgents              map[string]bool
	stoppedAgents             map[string]bool
	phaseArtifacts            map[string]bool
	deliveries                map[string]bool
	reviews                   map[string]bool
	reviewResults             []string
	acceptedReviews           map[string]bool
	reworkRequests            []string
	agentAssessments          map[string]bool
	directorQuestions         map[string]bool
	directorAnsweredQuestions map[string]bool
	expiredLeaseRefs          map[string]bool
	replanRefs                map[string]bool
	plannedPhaseArtifacts     map[string]bool
	plannedDeliveries         map[string]bool
	plannedReviews            map[string]bool
	plannedReviewResults      map[string]bool
	plannedAcceptedReviews    map[string]bool
	plannedReworkRequests     map[string]bool
	plannedCapacityRequests   map[string]bool
	plannedConcurrencyGates   map[string]bool
	plannedAgents             map[string]bool
	plannedAgentAssessments   map[string]bool
	plannedDirectorQuestions  map[string]bool
	plannedReplanRefs         map[string]bool
}

func newSchedulerTickCollectorV0(input DirectorSchedulerTickInputV0) *schedulerTickCollectorV0 {
	return &schedulerTickCollectorV0{
		input:                     input,
		tasks:                     schedulerStringSetV0(input.Snapshot.Tasks),
		plannedTasks:              map[string]bool{},
		capacityRequests:          schedulerStringSetV0(input.Snapshot.CapacityRequests),
		capacityDecisions:         schedulerStringSetV0(input.Snapshot.CapacityDecisions),
		concurrencyGates:          schedulerStringSetV0(input.Snapshot.ConcurrencyGates),
		agents:                    schedulerStringSetV0(input.Snapshot.Agents),
		startedAgents:             schedulerStringSetV0(input.Snapshot.StartedAgents),
		failedAgents:              schedulerStringSetV0(input.Snapshot.FailedAgents),
		stoppedAgents:             schedulerStringSetV0(input.Snapshot.StoppedAgents),
		phaseArtifacts:            schedulerStringSetV0(input.Snapshot.PhaseArtifacts),
		deliveries:                schedulerStringSetV0(input.Snapshot.Deliveries),
		reviews:                   schedulerStringSetV0(input.Snapshot.Reviews),
		reviewResults:             compactSchedulerStringsV0(input.Snapshot.ReviewResults),
		acceptedReviews:           schedulerStringSetV0(input.Snapshot.AcceptedReviews),
		reworkRequests:            compactSchedulerStringsV0(input.Snapshot.ReworkRequests),
		agentAssessments:          schedulerStringSetV0(input.Snapshot.AgentAssessments),
		directorQuestions:         schedulerStringSetV0(input.Snapshot.DirectorQuestions),
		directorAnsweredQuestions: schedulerStringSetV0(input.Snapshot.DirectorAnsweredQuestions),
		expiredLeaseRefs:          schedulerStringSetV0(input.Snapshot.ExpiredLeaseRefs),
		replanRefs:                schedulerStringSetV0(input.Snapshot.ReplanRefs),
		plannedPhaseArtifacts:     map[string]bool{},
		plannedDeliveries:         map[string]bool{},
		plannedReviews:            map[string]bool{},
		plannedReviewResults:      map[string]bool{},
		plannedAcceptedReviews:    map[string]bool{},
		plannedReworkRequests:     map[string]bool{},
		plannedCapacityRequests:   map[string]bool{},
		plannedConcurrencyGates:   map[string]bool{},
		plannedAgents:             map[string]bool{},
		plannedAgentAssessments:   map[string]bool{},
		plannedDirectorQuestions:  map[string]bool{},
		plannedReplanRefs:         map[string]bool{},
	}
}

func (collector *schedulerTickCollectorV0) planV0() DirectorSchedulerTickPlanV0 {
	waiting := compactSchedulerWaitingReasonsV0(collector.waitingReasons)
	blockedRefs := compactSchedulerStringsV0(collector.blockedRefs)
	if len(collector.commands) > 0 {
		return collector.withSequenceDecisionsV0(collector.commandsPlanV0(blockedRefs))
	}
	if collector.blocked {
		return collector.withSequenceDecisionsV0(schedulerBlockedPlanV0(collector.input, blockedRefs, nil))
	}
	if collector.needsDirector {
		return collector.withSequenceDecisionsV0(schedulerNeedsDirectorReasonsPlanV0(collector.input, waiting))
	}
	if len(waiting) > 0 {
		return collector.withSequenceDecisionsV0(schedulerWaitingReasonsPlanV0(collector.input, waiting))
	}
	return collector.withSequenceDecisionsV0(schedulerQuiescentPlanV0(collector.input))
}

func (collector *schedulerTickCollectorV0) hasSchedulingEffectsV0() bool {
	return len(collector.commands) > 0 ||
		len(collector.waitingReasons) > 0 ||
		len(collector.blockedRefs) > 0 ||
		collector.needsDirector ||
		collector.blocked
}

func (collector *schedulerTickCollectorV0) commandsPlanV0(blockedRefs []string) DirectorSchedulerTickPlanV0 {
	if collector.hasReadyCommands {
		return schedulerCommandsWithBlockedPlanV0(collector.input, collector.commands, blockedRefs)
	}
	if collector.blocked {
		return schedulerBlockedPlanV0(collector.input, blockedRefs, collector.commands)
	}
	if collector.needsDirector {
		return schedulerNeedsDirectorWithCommandsV0(collector.input, collector.commands)
	}
	return schedulerCommandsPlanV0(collector.input, collector.commands)
}

func (collector *schedulerTickCollectorV0) addCommandV0(
	command orquestacoreworkflow.OrchestrationCommandV0,
) {
	collector.commands = append(collector.commands, command)
}

func (collector *schedulerTickCollectorV0) addReadyCommandV0(
	command orquestacoreworkflow.OrchestrationCommandV0,
) {
	collector.commands = append(collector.commands, command)
	collector.hasReadyCommands = true
}

func (collector *schedulerTickCollectorV0) addWaitingV0(reason SchedulerWaitingReasonV0) {
	collector.waitingReasons = append(collector.waitingReasons, reason)
}

func (collector *schedulerTickCollectorV0) addInFlightAgentWaitIfAnyV0() {
	if schedulerHasInFlightAgentsV0(collector.input.Snapshot) {
		collector.addWaitingV0(SchedulerWaitingAgentDeliveryPendingV0)
	}
}

func (collector *schedulerTickCollectorV0) addNeedsDirectorV0(reason SchedulerWaitingReasonV0) {
	collector.needsDirector = true
	if reason != "" {
		collector.addWaitingV0(reason)
	}
}

func (collector *schedulerTickCollectorV0) addBlockedRefsV0(refs []string) {
	collector.blocked = true
	collector.blockedRefs = append(collector.blockedRefs, refs...)
}

func (collector *schedulerTickCollectorV0) addWorkSequenceDecisionV0(
	decision SchedulerWorkSequenceDecisionV0,
) {
	collector.workSequenceDecisions = append(collector.workSequenceDecisions, decision)
}

func (collector *schedulerTickCollectorV0) withSequenceDecisionsV0(
	plan DirectorSchedulerTickPlanV0,
) DirectorSchedulerTickPlanV0 {
	plan.WorkSequenceDecisions = compactSchedulerWorkSequenceDecisionsV0(collector.workSequenceDecisions)
	return plan
}
