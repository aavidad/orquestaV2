package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
)

func TestDecisionCouncilPlanMaterializerV0CreaRondasConWaitScopeYDeps(t *testing.T) {
	runRef := "run-decision-council-materializer-001"
	contractRef := "contract:function:decision-council:v0"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.FunctionContracts = []string{contractRef}
	store := NewInMemoryRunStoreV0(run)
	taskStore := NewInMemoryWorkflowTaskStoreV0()
	plan := decisionCouncilPlanForMaterializerTestV0(t, runRef)

	result, err := (DecisionCouncilPlanMaterializerV0{
		RunStore:    store,
		EventSink:   NewInMemoryEventSinkV0(),
		TaskWriter:  taskStore,
		RequestedBy: "orquesta-council-test",
	}).MaterializeDecisionCouncilPlanV0(context.Background(), DecisionCouncilPlanMaterializeRequestV0{
		Plan:       plan,
		OccurredAt: "2026-05-24T12:00:00Z",
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  contractRef,
			FunctionName: "DecisionCouncilRound",
		}},
		CorrelationID: "corr-decision-council-materializer-001",
	})
	if err != nil {
		t.Fatalf("MaterializeDecisionCouncilPlanV0: %v", err)
	}
	if len(result.Issues) != 0 || len(result.Tasks) != 9 || len(result.Rounds) != 3 {
		t.Fatalf("result=%+v", result)
	}
	proposals := decisionCouncilTasksForRoleV0(result.Tasks, orquestadecisioncouncil.CouncilRoleProposalV0)
	critiques := decisionCouncilTasksForRoleV0(result.Tasks, orquestadecisioncouncil.CouncilRoleCritiqueV0)
	votes := decisionCouncilTasksForRoleV0(result.Tasks, orquestadecisioncouncil.CouncilRoleVoteV0)
	if len(proposals) != 3 || len(critiques) != 3 || len(votes) != 3 {
		t.Fatalf("proposals=%d critiques=%d votes=%d", len(proposals), len(critiques), len(votes))
	}
	if critiques[0].CohortRef == proposals[0].CohortRef || votes[0].CohortRef == critiques[0].CohortRef {
		t.Fatalf("cohort refs not isolated: p=%s c=%s v=%s", proposals[0].CohortRef, critiques[0].CohortRef, votes[0].CohortRef)
	}
	if len(critiques[0].DependsOn) != 1 || len(votes[0].DependsOn) != 3 {
		t.Fatalf("critique deps=%v vote deps=%v", critiques[0].DependsOn, votes[0].DependsOn)
	}
}

func TestDecisionCouncilWorkflowTaskProviderV0GateaRondasPorEntregas(t *testing.T) {
	runRef := "run-decision-council-provider-001"
	fixture := decisionCouncilMaterializedFixtureV0(t, runRef)
	run := mustOpenPhaseForCouncilTestV0(t, fixture.Run, orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0)
	fixture.Store.SaveRunV0(context.Background(), run)

	candidates := decisionCouncilWorkCandidatesForRunV0(t, fixture.Store, fixture.TaskStore, run)
	if len(candidates.WorkCandidates) != 3 ||
		candidates.WorkCandidates[0].AgentCandidate.Payload.Role != orquestadecisioncouncil.CouncilRoleProposalV0 {
		t.Fatalf("proposal candidates=%+v", candidates.WorkCandidates)
	}
	run.DeliveredTasks = decisionCouncilTaskIDsV0(fixture.Tasks, orquestadecisioncouncil.CouncilRoleProposalV0)
	fixture.Store.SaveRunV0(context.Background(), run)
	candidates = decisionCouncilWorkCandidatesForRunV0(t, fixture.Store, fixture.TaskStore, run)
	if len(candidates.WorkCandidates) != 3 ||
		candidates.WorkCandidates[0].AgentCandidate.Payload.Role != orquestadecisioncouncil.CouncilRoleCritiqueV0 {
		t.Fatalf("critique candidates=%+v", candidates.WorkCandidates)
	}

	run = mustOpenPhaseForCouncilTestV0(t, run, orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0)
	run.DeliveredTasks = append(run.DeliveredTasks, decisionCouncilTaskIDsV0(fixture.Tasks, orquestadecisioncouncil.CouncilRoleCritiqueV0)...)
	fixture.Store.SaveRunV0(context.Background(), run)
	candidates = decisionCouncilWorkCandidatesForRunV0(t, fixture.Store, fixture.TaskStore, run)
	if len(candidates.WorkCandidates) != 3 ||
		candidates.WorkCandidates[0].AgentCandidate.Payload.Role != orquestadecisioncouncil.CouncilRoleVoteV0 {
		t.Fatalf("vote candidates=%+v", candidates.WorkCandidates)
	}
}

func TestBuildDecisionCouncilAcceptDecisionCommandV0ExigeQuorumYVoteDurable(t *testing.T) {
	runRef := "run-decision-council-accept-001"
	run := mustActiveProgrammingRunV0(t, runRef)
	run = mustOpenPhaseForCouncilTestV0(t, run, orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0)
	voteCommand, err := orquestacoreworkflow.NewRequestVoteCommandV0(
		commandMetaV0(runRef, "cmd-council-vote-001", "idem-council-vote-001"),
		orquestacoreworkflow.RequestVoteCommandPayloadV0{
			VoteRequestID:              "vote-ref-council-001",
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			DecisionTopicRef:           "topic-council-001",
			BrainstormRef:              "brainstorm-ref-council-001",
			Summary:                    "Votacion de consejo.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
			EvidenceRefs:               []string{"evidence-ref-council-vote-request"},
		},
	)
	if err != nil {
		t.Fatalf("NewRequestVoteCommandV0: %v", err)
	}
	run = mustApplyCommandV0(t, run, voteCommand)
	voteResult := decisionCouncilVoteResultForAcceptanceTestV0(t)

	command, ok, err := BuildDecisionCouncilAcceptDecisionCommandV0(DecisionCouncilAcceptDecisionCommandRequestV0{
		Run:         run,
		VoteRef:     "vote-ref-council-001",
		DecisionRef: "decision-ref-council-001",
		VoteResult:  voteResult,
		OccurredAt:  "2026-05-24T12:30:00Z",
	})
	if err != nil || !ok {
		t.Fatalf("BuildDecisionCouncilAcceptDecisionCommandV0 ok=%v err=%v", ok, err)
	}
	next := mustApplyCommandV0(t, run, command)
	if !stringInSetV0("decision-ref-council-001", next.Decisions) {
		t.Fatalf("decisions=%v", next.Decisions)
	}
}
