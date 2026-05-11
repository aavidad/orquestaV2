package orquestadirectorscheduler

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func TestBuildDirectorSchedulerTickV0ReplanRequestsCapacityBeforeAgent(t *testing.T) {
	input := validSchedulerTickInputWithReplanV0(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0)

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 2)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRecordReplanDecisionV0,
		orquestacoreworkflow.OrchestrationCommandRequestCapacityV0,
	)
}

func TestBuildDirectorSchedulerTickV0ReplanOpenPhaseBeforeCapacityFromRevision(t *testing.T) {
	input := validSchedulerTickInputWithReplanV0(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0)
	input.Snapshot.CurrentPhaseID = string(orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	input.ReplanFollowupCandidates[0].ReplanFollowupsInput.OpenPhaseCandidate = validReplanOpenPhaseCandidateForSchedulerV0()

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 3)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRecordReplanDecisionV0,
		orquestacoreworkflow.OrchestrationCommandOpenPhaseV0,
		orquestacoreworkflow.OrchestrationCommandRequestCapacityV0,
	)
}

func TestBuildDirectorSchedulerTickV0ReplanWaitsWhenFollowupCapacityPending(t *testing.T) {
	input := validSchedulerTickInputWithReplanV0(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0)
	input.Snapshot.CapacityRequests = []string{"capacity-ref-replan-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingCapacityPendingV0)
}

func TestBuildDirectorSchedulerTickV0ReplanAgentAfterDecidedCapacityDoesNotRepeatDecision(t *testing.T) {
	input := validSchedulerTickInputWithReplanV0(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0)
	input.Snapshot.ReplanRefs = []string{"replan-ref-scheduler-001"}
	input.Snapshot.CapacityDecisions = []string{"capacity-ref-replan-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRequestAgentV0,
	)
	assertSchedulerAgentRefV0(t, plan.Commands[0], "agent-ref-replan-scheduler-001")
}

func TestBuildDirectorSchedulerTickV0ReplanMissingCandidatesNeedsDirector(t *testing.T) {
	input := validSchedulerTickInputWithReplanV0(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0)
	input.ReplanFollowupCandidates[0].ReplanFollowupsInput.CapacityCandidate = nil
	input.ReplanFollowupCandidates[0].ReplanFollowupsInput.AgentCandidate = nil

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusNeedsDirectorV0, 1)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRecordReplanDecisionV0,
	)
}

func TestBuildDirectorSchedulerTickV0ReplanDoesNotReuseRequestedAgent(t *testing.T) {
	input := validSchedulerTickInputWithReplanV0(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0)
	input.Snapshot.ReplanRefs = []string{"replan-ref-scheduler-001"}
	input.Snapshot.CapacityDecisions = []string{"capacity-ref-replan-scheduler-001"}
	input.Snapshot.Agents = []string{"agent-ref-replan-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingAgentLifecyclePendingV0)
}

func TestBuildDirectorSchedulerTickV0ReplanBlocksFailedAgentCandidate(t *testing.T) {
	input := validSchedulerTickInputWithReplanV0(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0)
	input.Snapshot.CapacityDecisions = []string{"capacity-ref-replan-scheduler-001"}
	input.Snapshot.FailedAgents = []string{"agent-ref-replan-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusBlockedV0, 1)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRecordReplanDecisionV0,
	)
	if len(plan.BlockedRefs) != 1 || plan.BlockedRefs[0] != "agent-ref-replan-scheduler-001" {
		t.Fatalf("blocked_refs=%v", plan.BlockedRefs)
	}
}

func TestBuildDirectorSchedulerTickV0ReplanBlocksStoppedAgentCandidate(t *testing.T) {
	input := validSchedulerTickInputWithReplanV0(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0)
	input.Snapshot.CapacityDecisions = []string{"capacity-ref-replan-scheduler-001"}
	input.Snapshot.StoppedAgents = []string{"agent-ref-replan-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusBlockedV0, 1)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRecordReplanDecisionV0,
	)
	if len(plan.BlockedRefs) != 1 || plan.BlockedRefs[0] != "agent-ref-replan-scheduler-001" {
		t.Fatalf("blocked_refs=%v", plan.BlockedRefs)
	}
}

func TestBuildDirectorSchedulerTickV0RejectsForeignRunReplanCandidate(t *testing.T) {
	input := validSchedulerTickInputWithReplanV0(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0)
	input.ReplanFollowupCandidates[0].ReplanFollowupsInput.AgentCandidate.CommandMeta.RunID = "run-externo-001"

	_, err := BuildDirectorSchedulerTickV0(input)
	assertSchedulerTickErrorV0(t, err, "replan_followup_candidates.agent_candidate.command_meta.run_id")
}

func validSchedulerTickInputWithReplanV0(
	action orquestacoreworkflow.ReplanDecisionActionV0,
) DirectorSchedulerTickInputV0 {
	input := validSchedulerTickInputV0()
	input.WorkCandidates = nil
	input.ReplanFollowupCandidates = []SchedulableReplanFollowupCandidateV0{
		validSchedulableReplanFollowupCandidateV0(action),
	}
	return input
}

func validSchedulableReplanFollowupCandidateV0(
	action orquestacoreworkflow.ReplanDecisionActionV0,
) SchedulableReplanFollowupCandidateV0 {
	return SchedulableReplanFollowupCandidateV0{
		CandidateRef:         "replan-followup-candidate-ref-001",
		ReplanFollowupsInput: validReplanFollowupsInputForSchedulerV0(action),
		EvidenceRefs:         []string{"evidence-ref-replan-candidate-001"},
	}
}

func validReplanFollowupsInputForSchedulerV0(
	action orquestacoreworkflow.ReplanDecisionActionV0,
) orquestadirector.ReplanFollowupsInputV0 {
	return orquestadirector.ReplanFollowupsInputV0{
		DecisionCommandMeta: schedulerMetaV0("cmd-replan-scheduler-001", "replan-decision"),
		DecisionPayload: orquestacoreworkflow.RecordReplanDecisionCommandPayloadV0{
			ReplanRef:      "replan-ref-scheduler-001",
			RunRef:         "run-scheduler-001",
			TaskRef:        "task-ref-scheduler-001",
			SourceRef:      "source-ref-replan-scheduler-001",
			AcceptedAction: action,
			FollowupRefs:   []string{"followup-ref-replan-scheduler-001"},
			Summary:        "decision explicita para replan scheduler",
			EvidenceRefs:   []string{"evidence-ref-replan-001"},
		},
		CapacityCandidate: validReplanCapacityCandidateForSchedulerV0(),
		AgentCandidate:    validReplanAgentCandidateForSchedulerV0(),
		AskDirectorCandidate: &orquestadirector.ReplanAskDirectorCandidateV0{
			CommandMeta: schedulerMetaV0("cmd-replan-ask-scheduler-001", "replan-ask"),
			Payload: orquestacoreworkflow.AskDirectorCommandPayloadV0{
				QuestionID:   "question-ref-replan-scheduler-001",
				SourceGroup:  "orquesta-director-scheduler",
				Summary:      "consulta explicita de replan scheduler",
				Options:      []string{"aprobar", "rechazar"},
				EvidenceRefs: []string{"evidence-ref-replan-ask-001"},
				Blocking:     true,
			},
		},
	}
}

func validReplanOpenPhaseCandidateForSchedulerV0() *orquestadirector.ReplanOpenPhaseCandidateV0 {
	return &orquestadirector.ReplanOpenPhaseCandidateV0{
		CommandMeta: schedulerMetaV0("cmd-replan-open-phase-scheduler-001", "replan-open-phase"),
		Payload: orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Reason:  "replan followup scheduler",
		},
	}
}

func validReplanCapacityCandidateForSchedulerV0() *orquestadirector.ReplanCapacityCandidateV0 {
	return &orquestadirector.ReplanCapacityCandidateV0{
		CommandMeta: schedulerMetaV0("cmd-replan-capacity-scheduler-001", "replan-capacity"),
		Payload: orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          "capacity-ref-replan-scheduler-001",
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:                    "task-ref-scheduler-001",
			ReasonCode:                 "replan_followup",
			Summary:                    "capacidad explicita para replan scheduler",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
			EvidenceRefs:               []string{"evidence-ref-replan-capacity-001"},
		},
	}
}

func validReplanAgentCandidateForSchedulerV0() *orquestadirector.ReplanAgentCandidateV0 {
	return &orquestadirector.ReplanAgentCandidateV0{
		CommandMeta: schedulerMetaV0("cmd-replan-agent-scheduler-001", "replan-agent"),
		Payload: orquestacoreworkflow.RequestAgentCommandPayloadV0{
			AgentRequestID:     "agent-ref-replan-scheduler-001",
			PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:            "task-ref-scheduler-001",
			CapacityRequestRef: "capacity-ref-replan-scheduler-001",
			Role:               "implementacion",
			Summary:            "agente explicito para replan scheduler",
			EvidenceRefs:       []string{"evidence-ref-replan-agent-001"},
		},
	}
}
