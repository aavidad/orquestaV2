package application

import (
	"context"
	"encoding/json"
	"sort"
	"testing"
	"time"

	"orquesta/internal/council"
	"orquesta/internal/ports"
)

func newCouncilSystem(t *testing.T, policy council.Policy) *testAttestationSystem {
	t.Helper()
	clock := &mutableClock{now: time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("candidate output"),
	}}}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	control := &scriptedVersionControl{}
	attestor := &scriptedTestAttestor{verdict: ports.TestAttestationPassed}
	orchestrator.versionControl, orchestrator.testAttestor = control, attestor
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	plan := workspaceWritePlan()
	plan.WorkItems[0].CouncilPolicy = policy
	submitted, err := orchestrator.Submit(context.Background(), access, SubmitRequest{
		RequestRef: "request:council", Statement: "produce council candidate", Confirm: true, Plan: plan,
	})
	appTestNoError(t, err)
	return &testAttestationSystem{orchestrator: orchestrator, repository: repository, control: control,
		attestor: attestor, access: access, goalRef: submitted.Record.Goal.Ref()}
}

func TestCouncilAutoOpensAfterExactApprovedGateAndDecidesAtThree(t *testing.T) {
	system := newCouncilSystem(t, council.PolicyAuto)
	system.processCommit(t)
	system.process(t, ActionAttestTest, ActionLaunchAgent, ActionLaunchAgent)
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
	}
	system.process(t, ActionLaunchAgent, ActionLaunchAgent, ActionLaunchAgent)
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
	agent := system.orchestrator.observer.(*scriptedAgent)
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

func councilExecutions(record GoalRecord) []ExecutionRecord {
	result := make([]ExecutionRecord, 0, 3)
	for _, execution := range record.Executions {
		if isCouncilExecution(execution) {
			result = append(result, execution)
		}
	}
	return result
}
