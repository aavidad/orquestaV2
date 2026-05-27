package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
)

type decisionCouncilFixtureV0 struct {
	Store     *InMemoryRunStoreV0
	TaskStore *InMemoryWorkflowTaskStoreV0
	Run       orquestacoreworkflow.OrchestrationRunV0
	Tasks     []orquestacoreworkflow.WorkflowTaskV0
}

func decisionCouncilPlanForMaterializerTestV0(
	t *testing.T,
	runRef string,
) orquestadecisioncouncil.DecisionCouncilPlanV0 {
	t.Helper()
	plan, err := orquestadecisioncouncil.BuildDecisionCouncilPlanV0(orquestadecisioncouncil.DecisionCouncilPlanInputV0{
		RunRef:                  runRef,
		DecisionTopicRef:        "topic-council-001",
		BrainstormRequestRef:    "brainstorm-ref-council-001",
		VoteRequestRef:          "vote-ref-council-001",
		MinimumAgents:           3,
		MinimumDistinctFamilies: 3,
		RequiredFamilyRefs:      []string{"family-a", "family-b", "family-c"},
		EvidenceRefs:            []string{"evidence-ref-council-plan-001"},
		Candidates: []orquestadecisioncouncil.CouncilAgentCandidateV0{
			{AgentRef: "agent-a", FamilyRef: "family-a", CapacityLevel: orquestadecisioncouncil.CouncilCapacityHighV0, Active: true},
			{AgentRef: "agent-b", FamilyRef: "family-b", CapacityLevel: orquestadecisioncouncil.CouncilCapacityHighV0, Active: true},
			{AgentRef: "agent-c", FamilyRef: "family-c", CapacityLevel: orquestadecisioncouncil.CouncilCapacityHighV0, Active: true},
		},
	})
	if err != nil {
		t.Fatalf("BuildDecisionCouncilPlanV0: %v", err)
	}
	return plan
}

func decisionCouncilMaterializedFixtureV0(
	t *testing.T,
	runRef string,
) decisionCouncilFixtureV0 {
	t.Helper()
	contractRef := "contract:function:decision-council:v0"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.FunctionContracts = []string{contractRef}
	store := NewInMemoryRunStoreV0(run)
	taskStore := NewInMemoryWorkflowTaskStoreV0()
	result, err := (DecisionCouncilPlanMaterializerV0{
		RunStore:   store,
		EventSink:  NewInMemoryEventSinkV0(),
		TaskWriter: taskStore,
	}).MaterializeDecisionCouncilPlanV0(context.Background(), DecisionCouncilPlanMaterializeRequestV0{
		Plan:       decisionCouncilPlanForMaterializerTestV0(t, runRef),
		OccurredAt: "2026-05-24T12:10:00Z",
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef: contractRef,
		}},
	})
	if err != nil || len(result.Issues) > 0 {
		t.Fatalf("materialize err=%v issues=%+v", err, result.Issues)
	}
	run, err = store.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	return decisionCouncilFixtureV0{Store: store, TaskStore: taskStore, Run: run, Tasks: result.Tasks}
}

func decisionCouncilWorkCandidatesForRunV0(
	t *testing.T,
	store *InMemoryRunStoreV0,
	taskStore *InMemoryWorkflowTaskStoreV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) SchedulerCandidateSetV0 {
	t.Helper()
	latest, err := store.LoadRunV0(context.Background(), run.RunID)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	candidates, err := (WorkflowTaskCandidateProviderV0{TaskStore: taskStore}).BuildSchedulerCandidatesV0(
		context.Background(),
		SchedulerCandidateRequestV0{Run: latest, OccurredAt: "2026-05-24T12:20:00Z"},
	)
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	return candidates
}

func mustOpenPhaseForCouncilTestV0(
	t *testing.T,
	run orquestacoreworkflow.OrchestrationRunV0,
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		commandMetaV0(run.RunID, "cmd-open-"+string(phase), "idem-open-"+string(phase)),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{PhaseID: string(phase), Reason: "council-test"},
	)
	if err != nil {
		t.Fatalf("NewOpenPhaseCommandV0: %v", err)
	}
	return mustApplyCommandV0(t, run, command)
}

func decisionCouncilTasksForRoleV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	role string,
) []orquestacoreworkflow.WorkflowTaskV0 {
	out := []orquestacoreworkflow.WorkflowTaskV0{}
	for _, task := range tasks {
		if decisionCouncilRoleFromContextRefsV0(task.ContextRefs) == role {
			out = append(out, task)
		}
	}
	return out
}

func decisionCouncilTaskIDsV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	role string,
) []string {
	out := []string{}
	for _, task := range decisionCouncilTasksForRoleV0(tasks, role) {
		out = append(out, task.TaskID)
	}
	return out
}

func decisionCouncilVoteResultForAcceptanceTestV0(
	t *testing.T,
) orquestadecisioncouncil.DecisionCouncilVoteResultV0 {
	t.Helper()
	result, err := orquestadecisioncouncil.EvaluateDecisionCouncilVotesV0(orquestadecisioncouncil.DecisionCouncilVoteInputV0{
		DecisionTopicRef:        "topic-council-001",
		OptionRefs:              []string{"option-a"},
		MinimumVotes:            3,
		MinimumDistinctFamilies: 3,
		Votes: []orquestadecisioncouncil.CouncilVoteV0{
			{VoteRef: "vote-a", VoterRef: "agent-a", FamilyRef: "family-a", OptionRef: "option-a", Position: orquestadecisioncouncil.CouncilVoteApproveV0, EvidenceRefs: []string{"evidence-a"}},
			{VoteRef: "vote-b", VoterRef: "agent-b", FamilyRef: "family-b", OptionRef: "option-a", Position: orquestadecisioncouncil.CouncilVoteApproveV0, EvidenceRefs: []string{"evidence-b"}},
			{VoteRef: "vote-c", VoterRef: "agent-c", FamilyRef: "family-c", OptionRef: "option-a", Position: orquestadecisioncouncil.CouncilVoteApproveV0, EvidenceRefs: []string{"evidence-c"}},
		},
	})
	if err != nil || !result.Accepted {
		t.Fatalf("EvaluateDecisionCouncilVotesV0 result=%+v err=%v", result, err)
	}
	return result
}
