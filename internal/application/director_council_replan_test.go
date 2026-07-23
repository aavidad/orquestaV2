package application

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestCouncilGovernanceDecisionAllowsOnlyNegativeOutcomes(t *testing.T) {
	for _, outcome := range []council.Outcome{council.OutcomeRejected, council.OutcomeNoConsensus, council.OutcomeBlockedSecurity} {
		t.Run(string(outcome), func(t *testing.T) {
			record, source, author, decision, system := councilDecisionRecord(t, outcome)
			request := councilReplanRequest(record, source, author, decision)
			if !councilGovernanceReplanValid(record, source, author, request) {
				t.Fatalf("negative Council decision rejected: %+v", decision)
			}
			principal, project, _ := system.access.values()
			system.orchestrator.access.(*memoryAccessRepository).setRole(principal.Ref, project, identity.RoleProjectOwner)
			lease, err := system.orchestrator.ClaimDirector(context.Background(), system.access, ClaimDirectorRequest{
				RequestRef: "director-claim:council-" + string(outcome), GoalRef: record.Goal.Ref(),
			})
			appTestNoError(t, err)
			request.RequestRef, request.GoalRef = "director-plan:council-"+string(outcome), record.Goal.Ref()
			request.ExpectedGoalRevision, request.ExpectedPlanGeneration = record.Goal.Revision(), record.Goal.PlanGeneration()
			request.LeaseToken, request.LeaseFence, request.Reason = lease.Lease.Token, lease.Lease.Fence, "replan exact negative Council decision"
			request.Plan = PlanSpec{WorkItems: []WorkItemSpec{{Key: "successor-" + string(outcome), Objective: "rework exact Council source",
				Phase: source.Phase().String(), Role: source.Role().String(), WriteSet: []string{"internal/workspace"},
				CouncilPolicy: council.PolicyRequired, RequiredTests: requiredTestSpecs("required-test:council-replan-" + string(outcome)),
				OutputContract: goal.OutputContractEvidenceBundle}}}
			result, err := system.orchestrator.ProposeDirectorPlan(context.Background(), system.access, request)
			if err != nil || !result.Created || result.Decision.CouncilDecisionRef != decision.Ref ||
				result.Decision.CouncilDecisionDigest != decision.DecisionDigest {
				t.Fatalf("Council replan result=%+v err=%v", result, err)
			}
			after := system.record(t)
			updated, _ := after.Goal.WorkItem(source.Ref())
			successor := after.Goal.WorkItems()[len(after.Goal.WorkItems())-1]
			afterAuthor, _ := executionByRef(after.Executions, author.Ref)
			policy, declared := successor.CouncilPolicy()
			if updated.State() != goal.WorkItemStateSuperseded || afterAuthor.State != ExecutionCanceled ||
				afterAuthor.FailureCode != "application.execution_superseded" || !declared || policy != council.PolicyRequired {
				t.Fatalf("Council successor causal/policy=%+v source=%+v", successor, updated)
			}
			replay, err := system.orchestrator.ProposeDirectorPlan(context.Background(), system.access, request)
			if err != nil || replay.Created || replay.Decision != result.Decision {
				t.Fatalf("Council replay=%+v err=%v", replay, err)
			}
		})
	}
}

func TestCouncilGovernanceDecisionRejectsAcceptedAndSpoofedCausality(t *testing.T) {
	record, source, author, accepted, _ := councilDecisionRecord(t, council.OutcomeAccepted)
	if councilGovernanceReplanValid(record, source, author, councilReplanRequest(record, source, author, accepted)) {
		t.Fatal("accepted Council decision authorized replan")
	}
	record, source, author, rejected, _ := councilDecisionRecord(t, council.OutcomeRejected)
	request := councilReplanRequest(record, source, author, rejected)
	request.CouncilDecisionDigest = CouncilSubjectDigest("sha256:" + string(makeDigest('f')))
	if councilGovernanceReplanValid(record, source, author, request) {
		t.Fatal("spoofed Council digest authorized replan")
	}
	request = councilReplanRequest(record, source, author, rejected)
	request.CouncilSubjectDigest = CouncilSubjectDigest("sha256:" + string(makeDigest('e')))
	if councilGovernanceReplanValid(record, source, author, request) {
		t.Fatal("cross Council subject authorized replan")
	}
}

func TestCouncilReplanRequestBindsFingerprintAndExplicitSuccessorPolicy(t *testing.T) {
	record, source, author, decision, _ := councilDecisionRecord(t, council.OutcomeRejected)
	request := councilReplanRequest(record, source, author, decision)
	request.Plan = PlanSpec{WorkItems: []WorkItemSpec{{Key: "council-successor", Objective: "revise after Council",
		Phase: source.Phase().String(), Role: source.Role().String(), WriteSet: []string{"internal/workspace"},
		CouncilPolicy: council.PolicyRequired, RequiredTests: requiredTestSpecs("required-test:council-successor"),
		OutputContract: goal.OutputContractEvidenceBundle}}}
	first := directorPlanFingerprint(authorGoalPrincipal(t, record), record.Goal.Project(), request)
	request.CouncilDecisionRef = "council-decision:spoof"
	if first == directorPlanFingerprint(authorGoalPrincipal(t, record), record.Goal.Project(), request) {
		t.Fatal("Council decision ref absent from replay fingerprint")
	}
}

func TestCouncilReplanRejectsUnsafeDecisionRef(t *testing.T) {
	record, source, author, decision, _ := councilDecisionRecord(t, council.OutcomeRejected)
	request := councilReplanRequest(record, source, author, decision)
	request.RequestRef, request.GoalRef = "director-plan:council-unsafe", record.Goal.Ref()
	request.ExpectedGoalRevision, request.ExpectedPlanGeneration = record.Goal.Revision(), record.Goal.PlanGeneration()
	request.LeaseToken, request.LeaseFence, request.Reason = "director-lease:test", 1, "strict ref"
	request.Plan = PlanSpec{WorkItems: []WorkItemSpec{{Key: "successor", Objective: "strict", Phase: source.Phase().String(), Role: source.Role().String()}}}
	for _, unsafe := range []string{"council-decision\nspoof", "council-decision\x00spoof"} {
		request.CouncilDecisionRef = unsafe
		if err := validateProposeDirectorPlanRequest(request); err == nil || err.Error() != "application.director_plan_council_fence_invalid" {
			t.Fatalf("unsafe decision ref %q error=%v", unsafe, err)
		}
	}
}

func councilDecisionRecord(t *testing.T, outcome council.Outcome) (GoalRecord, goal.WorkItem, ExecutionRecord, CouncilDecisionRecord, *testAttestationSystem) {
	t.Helper()
	system := newCouncilSystem(t, council.PolicyAuto)
	system.processCommit(t)
	system.process(t, ActionAttestTest, ActionLaunchAgent, ActionLaunchAgent, ActionObserveAgent, ActionObserveAgent)
	record := system.record(t)
	round := record.CouncilRounds[0]
	executions := councilExecutions(record)
	system.process(t, ActionLaunchAgent, ActionLaunchAgent, ActionLaunchAgent)
	sort.Slice(executions, func(i, j int) bool { return executions[i].Ref.String() < executions[j].Ref.String() })
	ballots := councilBallots(outcome)
	observations := make([]ports.AgentObservation, 0, len(executions))
	for index, execution := range executions {
		role, _ := councilRole(execution)
		evidence := []council.Evidence{{Kind: "review_gate", Ref: round.Subject.ReviewGateDigest}}
		if ballots[index] == council.BallotSecurityVeto {
			evidence = append(evidence, council.Evidence{Kind: "security_veto", Ref: "evidence:security-veto"})
		}
		content, err := json.Marshal(council.Contribution{Schema: council.ContributionSchema, SubjectDigest: string(round.SubjectDigest),
			Role: role, Body: "Council outcome", Ballot: ballots[index], Evidence: evidence})
		appTestNoError(t, err)
		observations = append(observations, ports.AgentObservation{Status: ports.AgentCompleted, MediaType: council.ContributionMediaType, Content: content})
	}
	agent := system.orchestrator.observer.(*scriptedAgent)
	agent.mu.Lock()
	agent.observations = observations
	agent.mu.Unlock()
	system.process(t, ActionObserveAgent, ActionObserveAgent, ActionObserveAgent)
	record = system.record(t)
	item := record.Goal.WorkItems()[0]
	author, found := authorBound(record, item)
	if !found || len(record.CouncilDecisions) != 1 {
		t.Fatalf("Council outcome record invalid: %+v", record)
	}
	return record, item, author, record.CouncilDecisions[0], system
}

func councilBallots(outcome council.Outcome) []council.Ballot {
	switch outcome {
	case council.OutcomeAccepted:
		return []council.Ballot{council.BallotAccept, council.BallotAccept, council.BallotReject}
	case council.OutcomeRejected:
		return []council.Ballot{council.BallotReject, council.BallotReject, council.BallotAccept}
	case council.OutcomeNoConsensus:
		return []council.Ballot{council.BallotAccept, council.BallotReject, council.BallotAbstain}
	default:
		return []council.Ballot{council.BallotSecurityVeto, council.BallotAccept, council.BallotAccept}
	}
}

func councilReplanRequest(record GoalRecord, source goal.WorkItem, author ExecutionRecord, decision CouncilDecisionRecord) ProposeDirectorPlanRequest {
	return ProposeDirectorPlanRequest{Cause: goal.ReplanCauseGovernanceDecision, SourceWorkItemRef: source.Ref(),
		ExpectedWorkItemRevision: source.Revision(), SourceExecutionRef: author.Ref, SourceExecutionAttempt: author.AttemptNo,
		CouncilSubjectDigest: decision.SubjectDigest, CouncilDecisionRef: decision.Ref, CouncilDecisionDigest: decision.DecisionDigest}
}

func authorGoalPrincipal(t *testing.T, record GoalRecord) identity.PrincipalRef {
	t.Helper()
	return record.WorkItemAuthorities[0].PrincipalRef
}
