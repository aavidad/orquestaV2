package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/i18n"
	"orquesta/internal/identity"
	mcpiface "orquesta/internal/interfaces/mcp"
	"orquesta/internal/ports"
)

func TestMCPCreatesAndExecutesDiamondDAGAtomically(t *testing.T) {
	harness := newDAGHarness(t, nil)
	created := harness.create(t, "request:mcp-diamond", "execute diamond", map[string]any{
		"phases": []any{
			planPhase("phase-instance:build", "phase:build", "phase-template:program", []string{"input:app-spec"}, []string{"criterion:build-green"}),
			planPhase("phase-instance:review", "phase:review", "phase-template:review", []string{"input:build-artifacts"}, []string{"criterion:review-accepted"}),
		},
		"work_items": []any{
			planItem("a", "root", "phase:build", nil, []string{"internal/root"}),
			planItem("b", "left", "phase:review", []string{"a"}, []string{"internal/left"}),
			planItem("c", "right", "phase:review", []string{"a"}, []string{"internal/right"}),
			planItem("d", "join", "phase:review", []string{"b", "c"}, []string{"internal/join"}),
		},
	})
	if created.PlanGeneration != 1 || len(created.Phases) != 2 || len(created.WorkItems) != 4 || len(created.Executions) != 1 {
		t.Fatalf("created diamond = %+v", created)
	}

	harness.process(t, 2)
	afterRoot := harness.get(t, created.GoalRef)
	status, err := harness.repository.Status(context.Background())
	if err != nil || len(afterRoot.Executions) != 3 || len(afterRoot.Artifacts) != 1 || status.PendingActions != 2 {
		t.Fatalf("atomic fan-out = executions:%d artifacts:%d status:%+v err:%v", len(afterRoot.Executions), len(afterRoot.Artifacts), status, err)
	}

	harness.process(t, 2)
	if harness.agent.concurrent() != 2 {
		t.Fatalf("disjoint successors were not concurrently launchable: %d", harness.agent.concurrent())
	}
	harness.process(t, 2)
	afterBranches := harness.get(t, created.GoalRef)
	if len(afterBranches.Executions) != 4 || afterBranches.State != string(goal.GoalStateRunning) {
		t.Fatalf("join was not scheduled exactly once: %+v", afterBranches)
	}
	harness.process(t, 2)
	closed := harness.get(t, created.GoalRef)
	if closed.State != string(goal.GoalStateSucceeded) || len(closed.Executions) != 4 ||
		len(closed.Artifacts) != 4 || len(closed.Attestations) != 4 {
		t.Fatalf("closed diamond = %+v", closed)
	}
	for _, item := range closed.WorkItems {
		if item.State != string(goal.WorkItemStateSucceeded) {
			t.Fatalf("non-succeeded diamond item = %+v", item)
		}
	}
}

func TestMCPRejectsMalformedTypedPlanAsInvalidRequest(t *testing.T) {
	harness := newDAGHarness(t, nil)
	result := callMCPTool(t, context.Background(), harness.clientSession, mcpiface.ToolGoalsCreate, map[string]any{
		"request_ref": "request:mcp-invalid-plan", "statement": "invalid plan", "confirm": true,
		"plan": map[string]any{"phases": []any{}, "work_items": []any{}},
	})
	var output mcpiface.CreateGoalOutput
	decodeMCPOutput(t, result, &output)
	if !result.IsError || output.Error == nil || output.Error.Code != "invalid_request" {
		t.Fatalf("invalid typed plan output=%+v result=%+v", output, result)
	}
}

func TestMCPAllowsOmittedOptionalPhaseAndExecutionRefs(t *testing.T) {
	harness := newDAGHarness(t, nil)
	created := harness.create(t, "request:mcp-v05-optional-refs", "accept omitted optional refs", map[string]any{
		"phases": []any{planPhase("phase-instance:work", "phase:work", "phase-template:program", nil, nil)},
		"work_items": []any{
			planItem("work", "work without optional refs", "phase:work", nil, nil),
		},
	})
	if len(created.WorkItems) != 1 || len(created.Executions) != 1 {
		t.Fatalf("optional execution refs changed plan admission: %+v", created)
	}
}

func TestMCPRejectsUnknownContractualParentAsInvalidRequest(t *testing.T) {
	harness := newDAGHarness(t, nil)
	item := planItem("child", "invalid child", "phase:work", nil, nil)
	item["parent"] = "missing"
	result := callMCPTool(t, context.Background(), harness.clientSession, mcpiface.ToolGoalsCreate, map[string]any{
		"request_ref": "request:mcp-v05-parent-invalid", "statement": "reject unknown parent", "confirm": true,
		"plan": map[string]any{
			"phases":     []any{planPhase("phase-instance:work", "phase:work", "phase-template:program", nil, nil)},
			"work_items": []any{item},
		},
	})
	var output mcpiface.CreateGoalOutput
	decodeMCPOutput(t, result, &output)
	if !result.IsError || output.Error == nil || output.Error.Code != "invalid_request" {
		t.Fatalf("unknown parent output=%+v result=%+v", output, result)
	}
}

func TestApplicationSQLiteCreatesMultiItemMaximalCohort(t *testing.T) {
	harness := newDAGHarness(t, nil)
	actor, _ := goal.NewActorRef("actor:local")
	project, _ := goal.NewProjectRef("project:local")
	result, err := harness.orchestrator.Submit(context.Background(), application.SubmitRequest{
		RequestRef: "request:application-v05-multi", ActorRef: actor, ProjectRef: project,
		Statement: "schedule maximal cohort", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:work", Key: "phase:work", TemplateRef: "phase-template:program",
			}},
			WorkItems: []application.WorkItemSpec{
				{Key: "a", Objective: "writer a", Phase: "phase:work", Role: "role:worker", WriteSet: []string{"internal/shared"}, OutputContract: goal.OutputContractEvidenceBundle},
				{Key: "b", Objective: "writer b", Phase: "phase:work", Role: "role:worker", WriteSet: []string{"internal/shared/file.go"}, OutputContract: goal.OutputContractEvidenceBundle},
				{Key: "c", Objective: "writer c", Phase: "phase:work", Role: "role:worker", WriteSet: []string{"docs/free.md"}, OutputContract: goal.OutputContractEvidenceBundle},
			},
		},
	})
	if err != nil {
		var stateErr *application.StateError
		if errors.As(err, &stateErr) {
			t.Fatalf("submit multi-item state=%s cause=%v", stateErr.Code, stateErr.Cause)
		}
		t.Fatalf("submit multi-item: %T %v", err, err)
	}
	if len(result.Record.Executions) != 2 {
		t.Fatalf("scheduled executions = %d, want deterministic maximal cohort of 2", len(result.Record.Executions))
	}
}

func TestMCPRoundTripsPhaseMetadataAndExecutionRequirementsToAgent(t *testing.T) {
	harness := newDAGHarness(t, nil)
	item := planItem("research", "research exact sources", "phase:research", nil, []string{"docs/research"})
	item["skill_refs"] = []string{"skill:web-research"}
	item["tool_refs"] = []string{"tool:web-search"}
	item["capability_refs"] = []string{"capability:cited-synthesis"}
	created := harness.create(t, "request:mcp-v05-metadata", "preserve phase and work requirements", map[string]any{
		"phases": []any{planPhase(
			"phase-instance:research", "phase:research", "phase-template:research-web",
			[]string{"input:app-spec"}, []string{"criterion:sources-cited"},
		)},
		"work_items": []any{item},
	})

	stored := harness.get(t, created.GoalRef)
	if len(stored.Phases) != 1 || stored.Phases[0].PhaseRef != "phase-instance:research" ||
		stored.Phases[0].TemplateRef != "phase-template:research-web" ||
		!reflect.DeepEqual(stored.Phases[0].InputRefs, []string{"input:app-spec"}) ||
		!reflect.DeepEqual(stored.Phases[0].CriterionRefs, []string{"criterion:sources-cited"}) {
		t.Fatalf("phase metadata did not round trip through MCP/SQLite: %+v", stored.Phases)
	}
	if len(stored.WorkItems) != 1 || !reflect.DeepEqual(stored.WorkItems[0].SkillRefs, []string{"skill:web-research"}) ||
		!reflect.DeepEqual(stored.WorkItems[0].ToolRefs, []string{"tool:web-search"}) ||
		!reflect.DeepEqual(stored.WorkItems[0].CapabilityRefs, []string{"capability:cited-synthesis"}) {
		t.Fatalf("work requirements did not round trip through MCP/SQLite: %+v", stored.WorkItems)
	}

	harness.process(t, 1)
	request, ok := harness.agent.requestForObjective("research exact sources")
	if !ok || request.PhaseRef != "phase-instance:research" || request.PhaseKey != "phase:research" ||
		request.PhaseTemplateRef != "phase-template:research-web" ||
		!reflect.DeepEqual(request.PhaseInputRefs, []string{"input:app-spec"}) ||
		!reflect.DeepEqual(request.PhaseCriterionRefs, []string{"criterion:sources-cited"}) ||
		!reflect.DeepEqual(request.SkillRefs, []string{"skill:web-research"}) ||
		!reflect.DeepEqual(request.ToolRefs, []string{"tool:web-search"}) ||
		!reflect.DeepEqual(request.CapabilityRefs, []string{"capability:cited-synthesis"}) {
		t.Fatalf("execution requirements did not reach provider port: found=%v request=%+v", ok, request)
	}
}

func TestMCPRoundTripsContractualParentChildRefs(t *testing.T) {
	harness := newDAGHarness(t, nil)
	parent := planItem("parent", "coordinate", "phase:work", nil, nil)
	child := planItem("child", "implement", "phase:work", nil, []string{"internal/child"})
	child["parent"] = "parent"
	created := harness.create(t, "request:mcp-v05-parent-child", "preserve recursion lineage", map[string]any{
		"phases":     []any{planPhase("phase-instance:work", "phase:work", "phase-template:program", nil, nil)},
		"work_items": []any{parent, child},
	})
	stored := harness.get(t, created.GoalRef)
	if len(stored.WorkItems) != 2 {
		t.Fatalf("parent/child work items = %+v", stored.WorkItems)
	}
	byObjective := make(map[string]mcpiface.WorkItemView, len(stored.WorkItems))
	for _, workItem := range stored.WorkItems {
		byObjective[workItem.Objective] = workItem
	}
	parentView, childView := byObjective["coordinate"], byObjective["implement"]
	if childView.ParentRef == "" || childView.ParentRef != parentView.WorkItemRef ||
		!reflect.DeepEqual(parentView.ChildRefs, []string{childView.WorkItemRef}) || len(childView.ChildRefs) != 0 {
		t.Fatalf("contractual lineage did not round trip: parent=%+v child=%+v", parentView, childView)
	}
	harness.process(t, 4)
	closed := harness.get(t, created.GoalRef)
	if closed.State != string(goal.GoalStateSucceeded) || len(closed.Artifacts) != 2 || len(closed.Attestations) != 2 {
		t.Fatalf("static parent/child DAG did not close safely: %+v", closed)
	}
}

func TestSchedulerSerializesOverlappingRootsBeforeProvider(t *testing.T) {
	harness := newDAGHarness(t, nil)
	created := harness.create(t, "request:mcp-overlap", "serialize conflicting writers", map[string]any{
		"phases": []any{planPhase("phase-instance:work", "phase:work", "phase-template:program", nil, nil)},
		"work_items": []any{
			planItem("a", "writer parent", "phase:work", nil, []string{"internal/shared"}),
			planItem("b", "writer child", "phase:work", nil, []string{"internal/shared/file.go"}),
		},
	})
	if len(created.Executions) != 1 {
		t.Fatalf("overlap roots not initially scheduled: %+v", created)
	}
	harness.process(t, 1)
	if harness.agent.launchCount() != 1 || harness.agent.concurrent() != 1 {
		t.Fatalf("conflicting root reached provider: launches=%d concurrent=%d", harness.agent.launchCount(), harness.agent.concurrent())
	}
	harness.process(t, 1)
	harness.clock.Advance(2 * time.Second)
	harness.process(t, 2)
	closed := harness.get(t, created.GoalRef)
	if closed.State != string(goal.GoalStateSucceeded) || harness.agent.launchCount() != 2 || harness.agent.concurrent() != 1 {
		t.Fatalf("serialized writer did not progress: goal=%+v launches=%d max=%d", closed, harness.agent.launchCount(), harness.agent.concurrent())
	}
}

func TestFailedRootSkipsDescendantsButIndependentWorkFinishesBeforeGoal(t *testing.T) {
	harness := newDAGHarness(t, map[string]bool{"fail root": true})
	created := harness.create(t, "request:mcp-failure-dag", "continue independent work", map[string]any{
		"phases": []any{planPhase("phase-instance:work", "phase:work", "phase-template:program", nil, nil)},
		"work_items": []any{
			planItem("a", "fail root", "phase:work", nil, []string{"internal/fail"}),
			planItem("b", "child", "phase:work", []string{"a"}, []string{"internal/child"}),
			planItem("c", "grandchild", "phase:work", []string{"b"}, []string{"internal/grandchild"}),
			planItem("d", "independent", "phase:work", nil, []string{"internal/independent"}),
		},
	})
	harness.process(t, 3)
	afterFirstFailure := harness.get(t, created.GoalRef)
	if afterFirstFailure.State != string(goal.GoalStateRunning) || len(afterFirstFailure.Executions) != 3 {
		t.Fatalf("replaceable provider failure did not preserve Goal: %+v", afterFirstFailure)
	}
	states := workStates(afterFirstFailure.WorkItems)
	if states["fail root"] != string(goal.WorkItemStateRunning) ||
		states["child"] != string(goal.WorkItemStatePending) ||
		states["grandchild"] != string(goal.WorkItemStatePending) ||
		states["independent"] != string(goal.WorkItemStateRunning) {
		t.Fatalf("first provider failure became terminal or changed unrelated work: %+v", states)
	}

	// Independent work closes while the failed provider execution follows its
	// own bounded replacement policy. Backoff is one second, then two seconds.
	harness.process(t, 1)
	harness.clock.Advance(time.Second)
	harness.process(t, 2)
	harness.clock.Advance(2 * time.Second)
	harness.process(t, 2)
	closed := harness.get(t, created.GoalRef)
	states = workStates(closed.WorkItems)
	if closed.State != string(goal.GoalStateFailed) || states["fail root"] != string(goal.WorkItemStateFailed) ||
		states["child"] != string(goal.WorkItemStateSkipped) || states["grandchild"] != string(goal.WorkItemStateSkipped) ||
		states["independent"] != string(goal.WorkItemStateSucceeded) ||
		len(closed.Executions) != 4 || len(closed.Artifacts) != 1 {
		t.Fatalf("failed DAG closure = goal:%+v states:%+v", closed, states)
	}
}

func planItem(key, objective, phase string, dependencies, writeSet []string) map[string]any {
	return map[string]any{
		"key": key, "objective": objective, "phase": phase, "role": "role:worker",
		"dependencies": dependencies, "write_set": writeSet,
		"output_contract": string(goal.OutputContractEvidenceBundle),
	}
}

func planPhase(ref, key, templateRef string, inputRefs, criterionRefs []string) map[string]any {
	result := map[string]any{
		"ref": ref, "key": key, "template_ref": templateRef,
	}
	if inputRefs != nil {
		result["input_refs"] = inputRefs
	}
	if criterionRefs != nil {
		result["criterion_refs"] = criterionRefs
	}
	return result
}

func workStates(items []mcpiface.WorkItemView) map[string]string {
	result := make(map[string]string, len(items))
	for _, item := range items {
		result[item.Objective] = item.State
	}
	return result
}

type dagHarness struct {
	orchestrator  *application.Orchestrator
	repository    *sqlite.Repository
	clock         *dagClock
	agent         *dagAgent
	clientSession *sdkmcp.ClientSession
}

func newDAGHarness(t *testing.T, failures map[string]bool) *dagHarness {
	t.Helper()
	clock := &dagClock{now: dagTestNow()}
	repository, err := sqlite.Open(context.Background(), sqlite.Options{
		Path: t.TempDir() + "/state/orquesta.sqlite", BusyTimeout: time.Second, MaxOpenConnections: 4,
		Now: clock.Now,
	})
	if err != nil {
		t.Fatalf("open DAG state: %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	agent := newDAGAgent(clock, failures)
	artifacts := newDAGArtifacts()
	orchestrator, err := application.New(application.Dependencies{
		State: repository, Launcher: agent, Observer: agent, Artifacts: artifacts,
		Clock: clock, IDs: &dagSequentialIDs{}, MaxOutputBytes: 4096,
		MaxExecutionAttempts: 3, AgentCapabilities: dagAgentCapabilities(),
		ClaimLease: time.Minute, ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
	})
	if err != nil {
		t.Fatalf("new DAG orchestrator: %v", err)
	}
	actorRef, _ := goal.NewActorRef("actor:local")
	projectRef, _ := goal.NewProjectRef("project:local")
	provider, _ := identity.NewLocalOwnerProvider(actorRef, projectRef)
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	server, err := mcpiface.New(mcpiface.Config{
		Orchestrator: orchestrator, Identity: provider, Catalog: catalog, Locale: "es",
		MaxListLimit: 10, MaxRequestBytes: 64 * 1024, Version: "dag-test",
	})
	if err != nil {
		t.Fatalf("new DAG MCP: %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	return &dagHarness{
		orchestrator: orchestrator, repository: repository, clock: clock, agent: agent,
		clientSession: connectDAGClient(t, httpServer.URL),
	}
}

func (harness *dagHarness) create(t *testing.T, requestRef, statement string, plan map[string]any) mcpiface.GoalView {
	t.Helper()
	result := callMCPTool(t, context.Background(), harness.clientSession, mcpiface.ToolGoalsCreate, map[string]any{
		"request_ref": requestRef, "statement": statement, "confirm": true, "plan": plan,
	})
	var output mcpiface.CreateGoalOutput
	decodeMCPOutput(t, result, &output)
	if result.IsError || !output.Created || output.Goal == nil {
		t.Fatalf("create DAG output=%+v result=%+v", output, result)
	}
	return *output.Goal
}

func (harness *dagHarness) get(t *testing.T, goalRef string) mcpiface.GoalView {
	t.Helper()
	result := callMCPTool(t, context.Background(), harness.clientSession, mcpiface.ToolGoalsGet, map[string]any{"goal_ref": goalRef})
	var output mcpiface.GetGoalOutput
	decodeMCPOutput(t, result, &output)
	if result.IsError || output.Goal == nil {
		t.Fatalf("get DAG output=%+v result=%+v", output, result)
	}
	return *output.Goal
}

func connectDAGClient(t *testing.T, endpoint string) *sdkmcp.ClientSession {
	t.Helper()
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "dag-e2e", Version: "1"}, nil)
	session, err := client.Connect(context.Background(), &sdkmcp.StreamableClientTransport{
		Endpoint:   endpoint,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}, nil)
	if err != nil {
		t.Fatalf("connect DAG MCP client: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func dagTestNow() time.Time {
	return time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
}

type dagSequentialIDs struct {
	mu   sync.Mutex
	next uint64
}

func (ids *dagSequentialIDs) NewID(ctx context.Context, namespace string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	ids.mu.Lock()
	defer ids.mu.Unlock()
	ids.next++
	return fmt.Sprintf("%s:dag-%d", namespace, ids.next), nil
}

func (harness *dagHarness) process(t *testing.T, count int) {
	t.Helper()
	for index := 0; index < count; index++ {
		result, err := harness.orchestrator.ProcessNext(context.Background(), fmt.Sprintf("worker:dag:%d", index))
		if err != nil || !result.Processed {
			var stateErr *application.StateError
			if errors.As(err, &stateErr) {
				t.Fatalf("process DAG step %d = %+v state=%s cause=%v", index, result, stateErr.Code, stateErr.Cause)
			}
			t.Fatalf("process DAG step %d = %+v err=%v", index, result, err)
		}
	}
}

type dagClock struct {
	mu  sync.Mutex
	now time.Time
}

func (clock *dagClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *dagClock) Advance(duration time.Duration) {
	clock.mu.Lock()
	clock.now = clock.now.Add(duration)
	clock.mu.Unlock()
}

type dagAgent struct {
	mu            sync.Mutex
	clock         *dagClock
	failures      map[string]bool
	requests      map[goal.ExecutionRef]ports.AgentLaunchRequest
	receipts      map[goal.ExecutionRef]ports.AgentLaunchReceipt
	inFlight      map[goal.ExecutionRef]struct{}
	launches      int
	maxConcurrent int
}

func newDAGAgent(clock *dagClock, failures map[string]bool) *dagAgent {
	return &dagAgent{
		clock: clock, failures: failures, requests: make(map[goal.ExecutionRef]ports.AgentLaunchRequest),
		receipts: make(map[goal.ExecutionRef]ports.AgentLaunchReceipt), inFlight: make(map[goal.ExecutionRef]struct{}),
	}
}

func (agent *dagAgent) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return dagAgentCapabilities(), nil
}

func dagAgentCapabilities() ports.AgentCapabilities {
	return ports.AgentCapabilities{
		ProviderRef: "provider:dag-test", ModelRef: "model:dag-test", AgentRef: "agent:dag-test", Unrestricted: true,
	}
}

func (agent *dagAgent) Launch(ctx context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	if err := ctx.Err(); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	agent.mu.Lock()
	defer agent.mu.Unlock()
	if existing, found := agent.requests[request.ExecutionRef]; found {
		if !reflect.DeepEqual(existing, request) {
			return ports.AgentLaunchReceipt{}, errors.New("dag_agent.execution_conflict")
		}
		return agent.receipts[request.ExecutionRef], nil
	}
	receipt := ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash, ProviderRef: "provider:dag-test",
		ModelRef: "model:dag-test", AgentRef: "agent:dag-test",
		ExternalRef: "external:" + request.ExecutionRef.String(), IdempotencyKey: request.IdempotencyKey,
		AcceptedAt: agent.clock.Now(),
	}
	agent.requests[request.ExecutionRef] = request
	agent.receipts[request.ExecutionRef] = receipt
	agent.inFlight[request.ExecutionRef] = struct{}{}
	agent.launches++
	if len(agent.inFlight) > agent.maxConcurrent {
		agent.maxConcurrent = len(agent.inFlight)
	}
	return receipt, nil
}

func (agent *dagAgent) Observe(ctx context.Context, executionRef goal.ExecutionRef) (ports.AgentObservation, error) {
	if err := ctx.Err(); err != nil {
		return ports.AgentObservation{}, err
	}
	agent.mu.Lock()
	defer agent.mu.Unlock()
	request, found := agent.requests[executionRef]
	if !found {
		return ports.AgentObservation{}, errors.New("dag_agent.execution_not_found")
	}
	delete(agent.inFlight, executionRef)
	if agent.failures[request.Objective] {
		return ports.AgentObservation{
			ExecutionRef: executionRef, SpecHash: request.SpecHash, Status: ports.AgentFailed,
			ErrorCode: "dag_agent.failed", ObservedAt: agent.clock.Now(),
		}, nil
	}
	return ports.AgentObservation{
		ExecutionRef: executionRef, SpecHash: request.SpecHash, Status: ports.AgentCompleted, MediaType: request.ArtifactMediaType,
		Content: []byte("artifact:" + executionRef.String()), ObservedAt: agent.clock.Now(),
	}, nil
}

func (agent *dagAgent) launchCount() int {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	return agent.launches
}

func (agent *dagAgent) concurrent() int {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	return agent.maxConcurrent
}

func (agent *dagAgent) requestForObjective(objective string) (ports.AgentLaunchRequest, bool) {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	for _, request := range agent.requests {
		if request.Objective == objective {
			return request, true
		}
	}
	return ports.AgentLaunchRequest{}, false
}

type dagArtifacts struct {
	mu      sync.Mutex
	content map[goal.ArtifactRef]ports.ArtifactContent
}

func newDAGArtifacts() *dagArtifacts {
	return &dagArtifacts{content: make(map[goal.ArtifactRef]ports.ArtifactContent)}
}

func (store *dagArtifacts) Put(ctx context.Context, request ports.PutArtifactRequest) (ports.StoredArtifact, error) {
	if err := ctx.Err(); err != nil {
		return ports.StoredArtifact{}, err
	}
	digest := sha256.Sum256(request.Content)
	digestText := hex.EncodeToString(digest[:])
	ref, _ := goal.NewArtifactRef("artifact:sha256:" + digestText)
	stored := ports.StoredArtifact{Ref: ref, Digest: digestText, MediaType: request.MediaType, Size: int64(len(request.Content))}
	store.mu.Lock()
	store.content[ref] = ports.ArtifactContent{
		Ref: ref, Digest: digestText, MediaType: request.MediaType,
		Size: stored.Size, Content: append([]byte(nil), request.Content...),
	}
	store.mu.Unlock()
	return stored, nil
}

func (store *dagArtifacts) Get(_ context.Context, ref goal.ArtifactRef, expectedSize int64) (ports.ArtifactContent, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	content, found := store.content[ref]
	if !found || content.Size != expectedSize {
		return ports.ArtifactContent{}, errors.New("artifact.not_found")
	}
	content.Content = append([]byte(nil), content.Content...)
	return content, nil
}
