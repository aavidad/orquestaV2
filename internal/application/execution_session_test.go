package application

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type testExecutionSessionBroker struct {
	requests []ports.ExecutionSessionEnsureRequest
	at       time.Time
}

func (broker *testExecutionSessionBroker) Ensure(_ context.Context, request ports.ExecutionSessionEnsureRequest) (ports.ExecutionSessionReceipt, error) {
	broker.requests = append(broker.requests, request)
	authority, err := DeriveExecutionSessionAuthority(request, "execution_token")
	return ports.ExecutionSessionReceipt{Authority: authority, EnsuredAt: broker.at}, err
}

func (broker *testExecutionSessionBroker) Revoke(_ context.Context, request ports.ExecutionSessionEnsureRequest) error {
	_, err := DeriveExecutionSessionAuthority(request, "execution_token")
	return err
}

func TestProcessLaunchEnsuresExactSessionAndCarriesOpaqueRef(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	broker := &testExecutionSessionBroker{at: clock.Now()}
	orchestrator.executionSessions = broker
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(ctx, accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:execution-session-launch", Statement: "perform scoped work", Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result, err := orchestrator.ProcessNext(ctx, "worker:execution-session-launch"); err != nil ||
		!result.Processed || len(broker.requests) != 1 {
		t.Fatalf("ProcessNext result=%+v ensures=%d err=%v", result, len(broker.requests), err)
	}
	record, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil || len(record.Executions) != 1 {
		t.Fatalf("GetGoal executions=%d err=%v", len(record.Executions), err)
	}
	execution := record.Executions[0]
	want := ExecutionSessionRequest(record.Goal, execution)
	if broker.requests[0] != want {
		t.Fatalf("Ensure request=%+v want=%+v", broker.requests[0], want)
	}
	agent.mu.Lock()
	if len(agent.launchRequests) != 1 || agent.launchRequests[0].SessionRef.String() == "" ||
		agent.launchRequests[0].AccessAuthority.ArtifactAccessRef.String() == "" ||
		agent.launchRequests[0].AccessAuthority.MCPAccessRef.String() == "" ||
		agent.launchRequests[0].AccessAuthority.MailboxEndpointRef.String() == "" {
		agent.mu.Unlock()
		t.Fatalf("launch requests=%+v", agent.launchRequests)
	}
	sessionRef := agent.launchRequests[0].SessionRef
	accessAuthority := agent.launchRequests[0].AccessAuthority
	agent.mu.Unlock()
	authority, err := DeriveExecutionSessionAuthority(want, "execution_token")
	if err != nil || sessionRef != authority.SessionRef || execution.ExecutionSessionRef != sessionRef ||
		accessAuthority.ArtifactAccessRef != authority.ArtifactAccessRef ||
		accessAuthority.MCPAccessRef != authority.MCPAccessRef ||
		accessAuthority.MailboxEndpointRef != authority.MailboxEndpointRef {
		t.Fatalf("SessionRef=%s durable=%s authority=%+v err=%v",
			sessionRef, execution.ExecutionSessionRef, authority, err)
	}

	agent.mu.Lock()
	agent.observations = []ports.AgentObservation{{
		Status: ports.AgentRunning, Usage: unknownUsage(), ObservedAt: clock.Now(),
	}}
	agent.mu.Unlock()
	if result, err := orchestrator.ProcessNext(ctx, "worker:execution-session-observe"); err != nil ||
		!result.Processed {
		t.Fatalf("ProcessNext observe result=%+v err=%v", result, err)
	}
	agent.mu.Lock()
	observed := append([]ports.AgentObserveRequest(nil), agent.observeRequests...)
	agent.mu.Unlock()
	wantObservation := agentObserveRequest(execution)
	if len(observed) != 1 || !reflect.DeepEqual(observed[0], wantObservation) {
		t.Fatalf("observe requests=%+v want=%+v", observed, wantObservation)
	}
}

func TestExecutionSessionDerivationBindsAttemptGenerationAndSuccessor(t *testing.T) {
	project, _ := goal.NewProjectRef("project:session")
	goalRef, _ := goal.NewGoalRef("goal:session")
	item, _ := goal.NewWorkItemRef("work-item:session")
	first, _ := goal.NewExecutionRef("execution:session:first")
	second, _ := goal.NewExecutionRef("execution:session:second")
	base := ports.ExecutionSessionEnsureRequest{
		ProjectRef: project, GoalRef: goalRef, WorkItemRef: item, ExecutionRef: first,
		ExecutionAttempt: 1, PlanGeneration: 1, AppSpecGeneration: 1,
		SpecHash: testDigest("session"),
	}
	authority, err := DeriveExecutionSessionAuthority(base, "execution_token")
	if err != nil {
		t.Fatal(err)
	}
	changed := base
	changed.ExecutionAttempt = 2
	changed.ReplacesExecutionRef = first
	changed.ExecutionRef = second
	successor, err := DeriveExecutionSessionAuthority(changed, "execution_token")
	if err != nil || successor.SessionRef == authority.SessionRef ||
		successor.ServicePrincipal == authority.ServicePrincipal {
		t.Fatalf("successor=%+v authority=%+v err=%v", successor, authority, err)
	}
	if successor.Request.ReplacesExecutionRef != authority.Request.ExecutionRef ||
		successor.Request.ExecutionAttempt != authority.Request.ExecutionAttempt+1 {
		t.Fatalf("successor lineage=%+v authority=%+v", successor, authority)
	}
	if authority.ArtifactAccessRef.String() == authority.MCPAccessRef.String() ||
		authority.ArtifactAccessRef.String() == authority.MailboxEndpointRef.String() ||
		authority.MCPAccessRef.String() == authority.MailboxEndpointRef.String() {
		t.Fatalf("access refs not domain-separated: %+v", authority)
	}
	artifactSuffix := strings.TrimPrefix(authority.ArtifactAccessRef.String(), "artifact-access:execution:sha256:")
	mcpSuffix := strings.TrimPrefix(authority.MCPAccessRef.String(), "mcp-access:execution:sha256:")
	mailboxSuffix := strings.TrimPrefix(authority.MailboxEndpointRef.String(), "mailbox-endpoint:execution:sha256:")
	if artifactSuffix == mcpSuffix || artifactSuffix == mailboxSuffix || mcpSuffix == mailboxSuffix {
		t.Fatalf("access digests not domain-separated: %+v", authority)
	}
}
