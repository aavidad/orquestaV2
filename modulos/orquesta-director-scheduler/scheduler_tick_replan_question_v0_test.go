package orquestadirectorscheduler

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildDirectorSchedulerTickV0ReplanAskDirectorBuildsDecisionAndQuestion(t *testing.T) {
	input := validSchedulerTickInputWithReplanV0(orquestacoreworkflow.ReplanDecisionActionAskDirectorV0)

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 2)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRecordReplanDecisionV0,
		orquestacoreworkflow.OrchestrationCommandAskDirectorV0,
	)
}

func TestBuildDirectorSchedulerTickV0ReplanQuestionPendingWaits(t *testing.T) {
	input := validSchedulerTickInputWithReplanV0(orquestacoreworkflow.ReplanDecisionActionAskDirectorV0)
	input.Snapshot.ReplanRefs = []string{"replan-ref-scheduler-001"}
	input.Snapshot.DirectorQuestions = []string{"question-ref-replan-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingDirectorQuestionPendingV0)
}

func TestBuildDirectorSchedulerTickV0ReplanQuestionAnsweredQuiescent(t *testing.T) {
	input := validSchedulerTickInputWithReplanV0(orquestacoreworkflow.ReplanDecisionActionAskDirectorV0)
	input.Snapshot.ReplanRefs = []string{"replan-ref-scheduler-001"}
	input.Snapshot.DirectorAnsweredQuestions = []string{"question-ref-replan-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusQuiescentV0, 0)
}

func TestBuildDirectorSchedulerTickV0ReplanPriorityOverWorkCandidate(t *testing.T) {
	input := validSchedulerTickInputV0()
	input.ReplanFollowupCandidates = []SchedulableReplanFollowupCandidateV0{
		validSchedulableReplanFollowupCandidateV0(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0),
	}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 2)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRecordReplanDecisionV0,
		orquestacoreworkflow.OrchestrationCommandRequestCapacityV0,
	)
}

func TestBuildDirectorSchedulerTickV0PendingOutboxBlocksReplan(t *testing.T) {
	input := validSchedulerTickInputWithReplanV0(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0)
	input.Snapshot.PendingOutboxRefs = []string{"outbox-ref-replan-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingOutboxPendingV0)
}
