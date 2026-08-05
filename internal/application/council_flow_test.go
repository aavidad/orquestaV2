package application

import (
	"context"
	"encoding/json"
	"sort"
	"testing"
	"time"

	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func newCouncilSystem(t *testing.T, policy council.Policy) *testAttestationSystem {
	return newCouncilSystemWithEgress(t, policy, EgressPolicyAuthority{})
}

func newCouncilSystemWithEgress(
	t *testing.T,
	policy council.Policy,
	egress EgressPolicyAuthority,
) *testAttestationSystem {
	t.Helper()
	clock := &mutableClock{now: time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("candidate output"),
	}}}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	if egress != (EgressPolicyAuthority{}) {
		orchestrator.egressPolicies = &egressPolicyResolverStub{authority: egress}
	}
	control := &scriptedVersionControl{}
	attestor := &scriptedTestAttestor{verdict: ports.TestAttestationPassed}
	orchestrator.versionControl, orchestrator.testAttestor = control, attestor
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	plan := workspaceWritePlan()
	plan.WorkItems[0].CouncilPolicy = policy
	plan.WorkItems[0].EgressPolicyRef = egress.PolicyRef.String()
	submitted, err := orchestrator.Submit(context.Background(), access, SubmitRequest{
		RequestRef: "request:council", Statement: "produce council candidate", Confirm: true, Plan: plan,
	})
	appTestNoError(t, err)
	return &testAttestationSystem{orchestrator: orchestrator, repository: repository, control: control,
		attestor: attestor, access: access, goalRef: submitted.Record.Goal.Ref()}
}

func TestCouncilAutoOpensAfterExactApprovedGateAndDecidesAtThree(t *testing.T) {
	policy := testEgressPolicyAuthority(t, "egress-policy:council", `{"destinations":["council.example"]}`)
	system := newCouncilSystemWithEgress(t, council.PolicyAuto, policy)
	system.processCommit(t)
	system.process(t, ActionAttestTest, ActionLaunchAgent, ActionLaunchAgent)
	agent := system.orchestrator.observer.(*scriptedAgent)
	if len(agent.launchRequests) < 3 {
		t.Fatalf("review dispatches=%d, want author plus two reviewers", len(agent.launchRequests))
	}
	wantEgress := ports.AgentLaunchEgressAuthority{PolicyRef: policy.PolicyRef.String(),
		PayloadSHA256: policy.PayloadSHA256, CanonicalPayload: policy.CanonicalPayload}
	for _, request := range agent.launchRequests[len(agent.launchRequests)-2:] {
		if request.EgressAuthority != wantEgress {
			t.Fatalf("review dispatch egress=%+v want=%+v", request.EgressAuthority, wantEgress)
		}
	}
	system.process(t, ActionObserveAgent)
	if record := system.record(t); len(record.CouncilRounds) != 0 {
		t.Fatalf("Council opened before second approval: %+v", record.CouncilRounds)
	}
	system.process(t, ActionObserveAgent)
	record := system.record(t)
	if len(record.CouncilRounds) != 1 || len(record.CouncilDecisions) != 0 {
		t.Fatalf("auto Council opening=%+v decisions=%+v", record.CouncilRounds, record.CouncilDecisions)
	}
	round := record.CouncilRounds[0]
	councilExecutions := councilExecutions(record)
	if len(councilExecutions) != 3 || round.SubjectDigest == "" {
		t.Fatalf("Council cohort=%+v round=%+v", councilExecutions, round)
	}
	for _, execution := range councilExecutions {
		if execution.ReviewSubjectDigest != "" || execution.CouncilSubjectDigest != round.SubjectDigest {
			t.Fatalf("Council reused V18 subject: %+v", execution)
		}
		phase, found := phaseForWorkItem(record.Goal, record.Goal.WorkItems()[0])
		if !found {
			t.Fatal("Council phase missing")
		}
		request, err := councilAgentLaunchRequestForSubject(record, record.Goal.WorkItems()[0], execution, phase, round.Subject)
		request.ReferenciaColocacion, _ = ports.NewAgentPlacementRef("placement:application-test")
		if err != nil || request.OutputContract != string(goal.OutputContractArtifact) || request.ArtifactMediaType != council.ContributionMediaType ||
			request.EgressAuthority != (ports.AgentLaunchEgressAuthority{PolicyRef: policy.PolicyRef.String(),
				PayloadSHA256: policy.PayloadSHA256, CanonicalPayload: policy.CanonicalPayload}) ||
			ports.ValidateAgentLaunchRequest(request) != nil {
			t.Fatalf("Council launch contract request=%+v err=%v", request, err)
		}
	}
	system.process(t, ActionLaunchAgent, ActionLaunchAgent, ActionLaunchAgent)
	if len(agent.launchRequests) < 6 {
		t.Fatalf("Council dispatches=%d, want author, reviewers and council", len(agent.launchRequests))
	}
	for _, request := range agent.launchRequests[len(agent.launchRequests)-3:] {
		if request.EgressAuthority != wantEgress {
			t.Fatalf("Council dispatch egress=%+v want=%+v", request.EgressAuthority, wantEgress)
		}
	}
	observations := make([]ports.AgentObservation, 0, 3)
	sort.Slice(councilExecutions, func(i, j int) bool { return councilExecutions[i].Ref.String() < councilExecutions[j].Ref.String() })
	for _, execution := range councilExecutions {
		role, _ := councilRole(execution)
		content, err := json.Marshal(council.Contribution{Schema: council.ContributionSchema,
			SubjectDigest: string(round.SubjectDigest), Role: role, Body: "strict contribution", Ballot: council.BallotAccept,
			Evidence: []council.Evidence{{Kind: "review_gate", Ref: round.Subject.ReviewGateDigest}}})
		appTestNoError(t, err)
		observations = append(observations, ports.AgentObservation{Status: ports.AgentCompleted,
			MediaType: council.ContributionMediaType, Content: content})
	}
	agent.mu.Lock()
	agent.observations = observations
	agent.mu.Unlock()
	system.process(t, ActionObserveAgent, ActionObserveAgent, ActionObserveAgent)
	record = system.record(t)
	if len(record.CouncilFacts) != 3 || len(record.CouncilDecisions) != 1 ||
		record.CouncilDecisions[0].Decision.Outcome != council.OutcomeAccepted {
		t.Fatalf("Council decision not 3/3 accepted: facts=%+v decisions=%+v", record.CouncilFacts, record.CouncilDecisions)
	}
}

func TestCouncilMalformedContributionRetriesWithoutChangingAuthor(t *testing.T) {
	system := newCouncilSystem(t, council.PolicyAuto)
	system.processCommit(t)
	system.process(t, ActionAttestTest, ActionLaunchAgent, ActionLaunchAgent, ActionObserveAgent, ActionObserveAgent)
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	author, found := authorBound(record, item)
	if !found || author.State != ExecutionAwaitingIntegration {
		t.Fatalf("author=%+v", author)
	}
	system.process(t, ActionLaunchAgent, ActionLaunchAgent, ActionLaunchAgent)
	agent := system.orchestrator.observer.(*scriptedAgent)
	agent.mu.Lock()
	agent.observations = []ports.AgentObservation{{Status: ports.AgentCompleted, MediaType: council.ContributionMediaType, Content: []byte(`{"schema":"bad"}`)}}
	agent.mu.Unlock()
	system.process(t, ActionObserveAgent)
	record = system.record(t)
	item = record.Goal.WorkItems()[0]
	afterAuthor, found := authorBound(record, item)
	if !found || afterAuthor.Ref != author.Ref || afterAuthor.State != ExecutionAwaitingIntegration || len(record.CouncilFacts) != 0 {
		t.Fatalf("Council retry changed author/facts: author=%+v facts=%+v", afterAuthor, record.CouncilFacts)
	}
	failed, retry := 0, 0
	for _, execution := range record.Executions {
		if !isCouncilExecution(execution) {
			continue
		}
		if execution.State == ExecutionFailed {
			failed++
		}
		if execution.State == ExecutionQueued && execution.ReplacesExecutionRef.String() != "" {
			retry++
		}
	}
	if failed != 1 || retry != 1 {
		t.Fatalf("Council retry failed=%d retry=%d", failed, retry)
	}
}

func TestCouncilTemporaryCapacityKeepsTheSameExecutionsUntilSlotsRecover(t *testing.T) {
	system := newCouncilSystem(t, council.PolicyAuto)
	system.processCommit(t)
	system.process(t, ActionAttestTest, ActionLaunchAgent, ActionLaunchAgent, ActionObserveAgent, ActionObserveAgent)

	before := system.record(t)
	original := councilExecutions(before)
	if len(original) != len(council.Roles()) {
		t.Fatalf("Council cohort before temporary capacity=%+v", original)
	}
	originalRefs := make(map[goal.ExecutionRef]struct{}, len(original))
	for _, execution := range original {
		originalRefs[execution.Ref] = struct{}{}
	}

	agent := system.orchestrator.launcher.(*scriptedAgent)
	agent.mu.Lock()
	agent.launchErr = definitelyUnappliedTemporaryError{}
	agent.mu.Unlock()
	system.process(t, ActionLaunchAgent, ActionLaunchAgent, ActionLaunchAgent)

	requeued := system.record(t)
	actual := councilExecutions(requeued)
	if len(actual) != len(original) {
		t.Fatalf("temporary capacity changed Council cardinality: before=%+v after=%+v", original, actual)
	}
	for _, execution := range actual {
		_, same := originalRefs[execution.Ref]
		if !same || execution.AttemptNo != 1 || execution.ReplacesExecutionRef.String() != "" ||
			execution.State != ExecutionDispatching || execution.FailureCode != "" {
			t.Fatalf("temporary capacity replaced or failed Council execution: %+v", execution)
		}
	}
	if len(requeued.BudgetSettlements) < len(council.Roles()) {
		t.Fatalf("temporary capacity lacks exact releases: %+v", requeued.BudgetSettlements)
	}
	for _, settlement := range requeued.BudgetSettlements[len(requeued.BudgetSettlements)-len(council.Roles()):] {
		if !governance.IsExactZeroRelease(settlement) {
			t.Fatalf("temporary capacity settlement is not an exact zero release: %+v", settlement)
		}
	}

	agent.mu.Lock()
	agent.launchErr = nil
	agent.mu.Unlock()
	clock := system.orchestrator.clock.(*mutableClock)
	clock.Advance(system.orchestrator.observationDelay)
	system.process(t, ActionLaunchAgent, ActionLaunchAgent, ActionLaunchAgent)

	recovered := system.record(t)
	for _, execution := range councilExecutions(recovered) {
		_, same := originalRefs[execution.Ref]
		if !same || execution.AttemptNo != 1 || execution.ReplacesExecutionRef.String() != "" ||
			execution.State != ExecutionRunning || execution.FailureCode != "" {
			t.Fatalf("capacity recovery did not launch the original Council execution: %+v", execution)
		}
	}
}

func TestCouncilRequiredAndSkipDoNotAutoOpen(t *testing.T) {
	for _, policy := range []council.Policy{council.PolicyRequired, council.PolicySkipByOperator} {
		t.Run(string(policy), func(t *testing.T) {
			system := newCouncilSystem(t, policy)
			system.processCommit(t)
			system.process(t, ActionAttestTest, ActionLaunchAgent, ActionLaunchAgent, ActionObserveAgent, ActionObserveAgent)
			if record := system.record(t); len(record.CouncilRounds) != 0 || len(councilExecutions(record)) != 0 {
				t.Fatalf("policy %s auto opened Council: rounds=%+v executions=%+v", policy, record.CouncilRounds, councilExecutions(record))
			}
		})
	}
}

func TestCouncilAutoAndRequiredUseSameCohortBuilder(t *testing.T) {
	auto := newCouncilSystem(t, council.PolicyAuto)
	auto.processCommit(t)
	auto.process(t, ActionAttestTest, ActionLaunchAgent, ActionLaunchAgent, ActionObserveAgent, ActionObserveAgent)
	autoRecord := auto.record(t)

	required := newCouncilSystem(t, council.PolicyRequired)
	required.processCommit(t)
	required.process(t, ActionAttestTest, ActionLaunchAgent, ActionLaunchAgent, ActionObserveAgent, ActionObserveAgent)
	requiredRecord := required.record(t)
	principal, project, err := required.access.values()
	appTestNoError(t, err)
	required.orchestrator.access.(*memoryAccessRepository).setRole(principal.Ref, project, identity.RoleProjectOwner)
	claim, err := required.orchestrator.ClaimDirector(context.Background(), required.access,
		ClaimDirectorRequest{RequestRef: "claim:council-cohort", GoalRef: required.goalRef})
	appTestNoError(t, err)
	change := requiredRecord.ChangeSets[0]
	item := requiredRecord.Goal.WorkItems()[0]
	opened, err := required.orchestrator.OpenCouncilRound(context.Background(), required.access, OpenCouncilRoundRequest{
		RequestRef: "open:council-cohort", GoalRef: required.goalRef, ChangeRef: change.Ref,
		ExpectedGoalRevision: requiredRecord.Goal.Revision(), ExpectedItemRevision: item.Revision(),
		LeaseToken: claim.Lease.Token, LeaseFence: claim.Lease.Fence,
	})
	appTestNoError(t, err)
	if !opened.Created {
		t.Fatal("required Council cohort was not created")
	}
	requiredRecord = required.record(t)

	if len(autoRecord.CouncilRounds) != 1 || len(requiredRecord.CouncilRounds) != 1 ||
		autoRecord.CouncilRounds[0].Opener != CouncilRoundOpenerAuto ||
		requiredRecord.CouncilRounds[0].Opener != CouncilRoundOpenerDirector {
		t.Fatalf("Council openers auto=%+v required=%+v", autoRecord.CouncilRounds, requiredRecord.CouncilRounds)
	}
	assertSameCouncilCohortShape(t, autoRecord, requiredRecord)
}

func assertSameCouncilCohortShape(t *testing.T, auto, required GoalRecord) {
	t.Helper()
	autoCohort, requiredCohort := councilExecutions(auto), councilExecutions(required)
	if len(autoCohort) != len(council.Roles()) || len(requiredCohort) != len(council.Roles()) {
		t.Fatalf("Council cohort cardinality auto=%d required=%d", len(autoCohort), len(requiredCohort))
	}
	for _, role := range council.Roles() {
		autoExecution, autoOK := councilExecutionForRole(autoCohort, role)
		requiredExecution, requiredOK := councilExecutionForRole(requiredCohort, role)
		if !autoOK || !requiredOK || autoExecution.Purpose != requiredExecution.Purpose ||
			autoExecution.ArtifactMediaType != council.ContributionMediaType || requiredExecution.ArtifactMediaType != council.ContributionMediaType ||
			autoExecution.State != ExecutionQueued || requiredExecution.State != ExecutionQueued ||
			autoExecution.AttemptNo != 1 || requiredExecution.AttemptNo != 1 ||
			autoExecution.CouncilSubjectDigest == "" || requiredExecution.CouncilSubjectDigest == "" ||
			!councilLaunchIntentPresent(auto, autoExecution) || !councilLaunchIntentPresent(required, requiredExecution) {
			t.Fatalf("Council cohort role=%s auto=%+v required=%+v", role, autoExecution, requiredExecution)
		}
	}
}

func councilExecutionForRole(executions []ExecutionRecord, role council.Role) (ExecutionRecord, bool) {
	for _, execution := range executions {
		if actual, ok := councilRole(execution); ok && actual == role {
			return execution, true
		}
	}
	return ExecutionRecord{}, false
}

func councilLaunchIntentPresent(record GoalRecord, execution ExecutionRecord) bool {
	for _, intent := range record.EffectIntents {
		if intent.Subject.ExecutionRef == execution.Ref && intent.ActionKind == ActionLaunchAgent && intent.Kind == EffectKindAgentLaunch {
			return true
		}
	}
	return false
}

func councilExecutions(record GoalRecord) []ExecutionRecord {
	result := make([]ExecutionRecord, 0, 3)
	for _, execution := range record.Executions {
		if isCouncilExecution(execution) {
			result = append(result, execution)
		}
	}
	return result
}
