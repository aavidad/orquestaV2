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
		"phases": []any{"phase:build", "phase:review"},
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

func TestSchedulerSerializesOverlappingRootsBeforeProvider(t *testing.T) {
	harness := newDAGHarness(t, nil)
	created := harness.create(t, "request:mcp-overlap", "serialize conflicting writers", map[string]any{
		"phases": []any{"phase:work"},
		"work_items": []any{
			planItem("a", "writer parent", "phase:work", nil, []string{"internal/shared"}),
			planItem("b", "writer child", "phase:work", nil, []string{"internal/shared/file.go"}),
		},
	})
	if len(created.Executions) != 2 {
		t.Fatalf("overlap roots not initially scheduled: %+v", created)
	}
	harness.process(t, 2)
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
		"phases": []any{"phase:work"},
		"work_items": []any{
			planItem("a", "fail root", "phase:work", nil, []string{"internal/fail"}),
			planItem("b", "child", "phase:work", []string{"a"}, []string{"internal/child"}),
			planItem("c", "grandchild", "phase:work", []string{"b"}, []string{"internal/grandchild"}),
			planItem("d", "independent", "phase:work", nil, []string{"internal/independent"}),
		},
	})
	harness.process(t, 3)
	afterFailure := harness.get(t, created.GoalRef)
	if afterFailure.State != string(goal.GoalStateRunning) {
		t.Fatalf("Goal closed before independent work: %+v", afterFailure)
	}
	states := workStates(afterFailure.WorkItems)
	if states["fail root"] != string(goal.WorkItemStateFailed) ||
		states["child"] != string(goal.WorkItemStateSkipped) ||
		states["grandchild"] != string(goal.WorkItemStateSkipped) ||
		states["independent"] != string(goal.WorkItemStateRunning) {
		t.Fatalf("failure cascade = %+v", states)
	}
	harness.process(t, 1)
	closed := harness.get(t, created.GoalRef)
	states = workStates(closed.WorkItems)
	if closed.State != string(goal.GoalStateFailed) || states["independent"] != string(goal.WorkItemStateSucceeded) ||
		len(closed.Executions) != 2 || len(closed.Artifacts) != 1 {
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
		Clock: clock, IDs: &dagSequentialIDs{}, MaxOutputBytes: 4096, MaxActionAttempts: 3,
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
	return ports.AgentCapabilities{ProviderRef: "provider:dag-test"}, nil
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
		ExecutionRef: request.ExecutionRef, SpecHash: request.SpecHash, ProviderRef: "provider:dag-test",
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
