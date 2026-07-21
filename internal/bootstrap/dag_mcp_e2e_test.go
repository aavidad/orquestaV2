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
	"strings"
	"sync"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"orquesta/internal/adapters/auth/localtoken"
	"orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/application"
	"orquesta/internal/config"
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

	// Writer lifecycle is prepare -> launch -> observe -> commit -> integrate.
	harness.process(t, 5)
	afterRoot := harness.get(t, created.GoalRef)
	status, err := harness.repository.Status(context.Background(), harness.projectRef)
	if err != nil || len(afterRoot.Executions) != 3 || len(afterRoot.Artifacts) != 1 || status.PendingActions != 2 {
		t.Fatalf("atomic fan-out = executions:%d artifacts:%d status:%+v err:%v", len(afterRoot.Executions), len(afterRoot.Artifacts), status, err)
	}

	harness.process(t, 4)
	if harness.agent.concurrent() != 2 {
		t.Fatalf("disjoint successors were not concurrently launchable: %d", harness.agent.concurrent())
	}
	harness.process(t, 6)
	afterBranches := harness.get(t, created.GoalRef)
	if len(afterBranches.Executions) != 4 || afterBranches.State != string(goal.GoalStateRunning) {
		t.Fatalf("join was not scheduled exactly once: %+v", afterBranches)
	}
	harness.process(t, 5)
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
		"project_ref": harness.projectRef.String(), "request_ref": "request:mcp-invalid-plan",
		"statement": "invalid plan", "confirm": true,
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

func TestMCPRejectsUnknownParentMetadataAsInvalidRequest(t *testing.T) {
	harness := newDAGHarness(t, nil)
	item := planItem("child", "invalid child", "phase:work", nil, nil)
	item["parent"] = "missing"
	result := callMCPTool(t, context.Background(), harness.clientSession, mcpiface.ToolGoalsCreate, map[string]any{
		"project_ref": harness.projectRef.String(), "request_ref": "request:mcp-v05-parent-invalid",
		"statement": "reject unknown parent", "confirm": true,
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
	result, err := harness.orchestrator.Submit(context.Background(), harness.access, application.SubmitRequest{
		RequestRef: "request:application-v05-multi",
		Statement:  "schedule maximal cohort", Confirm: true,
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

	harness.process(t, 2)
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

func TestMCPParentMetadataClosesWithoutMailboxOrRequeue(t *testing.T) {
	ctx := context.Background()
	harness := newDAGHarness(t, nil)
	if _, exposed := reflect.TypeOf(mcpiface.WorkItemInput{}).FieldByName("HandoffRequired"); exposed {
		t.Fatal("public MCP WorkItemInput exposes the internal handoff policy")
	}
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
		t.Fatalf("parent lineage did not round trip: parent=%+v child=%+v", parentView, childView)
	}
	record, err := harness.orchestrator.GetGoal(ctx, harness.access, mustDAGRef(t, created.GoalRef, goal.NewGoalRef))
	if err != nil {
		t.Fatalf("get parent metadata Goal: %v", err)
	}
	var parentItem, childItem goal.WorkItem
	for _, item := range record.Goal.WorkItems() {
		switch item.Objective() {
		case "coordinate":
			parentItem = item
		case "implement":
			childItem = item
		}
	}
	if parentItem.HandoffRequired() || childItem.HandoffRequired() {
		t.Fatalf("public parent metadata activated handoff: parent=%v child=%v",
			parentItem.HandoffRequired(), childItem.HandoffRequired())
	}

	harness.process(t, 7)
	closed := harness.get(t, created.GoalRef)
	states := workStates(closed.WorkItems)
	status, statusErr := harness.repository.Status(ctx, harness.projectRef)
	idle, idleErr := harness.orchestrator.ProcessNext(ctx, "worker:dag:idle-parent-metadata")
	if closed.State != string(goal.GoalStateSucceeded) ||
		states["coordinate"] != string(goal.WorkItemStateSucceeded) ||
		states["implement"] != string(goal.WorkItemStateSucceeded) ||
		len(closed.Artifacts) != 2 || len(closed.Attestations) != 2 ||
		len(closed.Executions) != 2 || harness.agent.launchCount() != 2 ||
		statusErr != nil || status.PendingActions != 0 || idleErr != nil || idle.Processed {
		t.Fatalf("public parent metadata did not close without requeue: goal=%+v status=%+v/%v idle=%+v/%v launches=%d",
			closed, status, statusErr, idle, idleErr, harness.agent.launchCount())
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
	harness.process(t, 2)
	if harness.agent.launchCount() != 1 || harness.agent.concurrent() != 1 {
		t.Fatalf("conflicting root reached provider: launches=%d concurrent=%d", harness.agent.launchCount(), harness.agent.concurrent())
	}
	harness.process(t, 3)
	harness.clock.Advance(2 * time.Second)
	harness.process(t, 5)
	closed := harness.get(t, created.GoalRef)
	if closed.State != string(goal.GoalStateSucceeded) || harness.agent.launchCount() != 2 || harness.agent.concurrent() != 1 {
		t.Fatalf("serialized writer did not progress: goal=%+v launches=%d max=%d", closed, harness.agent.launchCount(), harness.agent.concurrent())
	}
}

func TestExhaustedRootInterruptsAndKeepsGoalOpenForDirectorReplan(t *testing.T) {
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
	harness.process(t, 5)
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
	// Exhaustion interrupts its WorkItem but leaves Goal and dependants open for
	// an explicit Director replan; scheduler must not invent terminal skips.
	harness.process(t, 1)
	harness.clock.Advance(time.Second)
	harness.process(t, 2)
	harness.clock.Advance(2 * time.Second)
	harness.process(t, 2)
	// Second replacement is launched above. Its failure schedules the final
	// attempt after the two-second backoff; only that final failed observation
	// interrupts the WorkItem.
	harness.process(t, 1)
	harness.clock.Advance(2 * time.Second)
	harness.process(t, 3)
	open := harness.get(t, created.GoalRef)
	states = workStates(open.WorkItems)
	if open.State != string(goal.GoalStateRunning) || open.ClosedAt != nil ||
		states["fail root"] != string(goal.WorkItemStateInterrupted) ||
		states["child"] != string(goal.WorkItemStatePending) ||
		states["grandchild"] != string(goal.WorkItemStatePending) ||
		states["independent"] != string(goal.WorkItemStateSucceeded) ||
		len(open.Executions) != 4 || len(open.Artifacts) != 1 {
		t.Fatalf("exhausted DAG did not remain open for replan = goal:%+v states:%+v", open, states)
	}
	record, err := harness.orchestrator.GetGoal(
		context.Background(), harness.access, mustDAGRef(t, created.GoalRef, goal.NewGoalRef),
	)
	if err != nil {
		t.Fatalf("read durable interrupted DAG: %v", err)
	}
	var root goal.WorkItem
	for _, item := range record.Goal.WorkItems() {
		if item.Objective() == "fail root" {
			root = item
			break
		}
	}
	cause, interrupted := root.InterruptCause()
	if root.Ref().String() == "" || !interrupted || cause != goal.WorkItemInterruptExecutionFailed {
		t.Fatalf("exhausted root interrupt cause=%q interrupted=%v item=%+v", cause, interrupted, root)
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
	orchestrator   *application.Orchestrator
	repository     *sqlite.Repository
	access         application.Access
	projectRef     goal.ProjectRef
	clock          *dagClock
	agent          *dagAgent
	workspace      *dagWorkspace
	versionControl *dagVersionControl
	clientSession  *sdkmcp.ClientSession
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
	actorRef, _ := goal.NewActorRef("actor:local")
	projectRef, _ := goal.NewProjectRef("project:local")
	principalRef, _ := identity.NewPrincipalRef(actorRef.String())
	principal, err := identity.NewPrincipal(
		principalRef, actorRef, identity.PrincipalKindHuman, localtoken.AuthenticationMethod,
	)
	if err != nil {
		t.Fatalf("new DAG principal: %v", err)
	}
	workspaceRef, _ := identity.NewWorkspaceRef(projectRef.String())
	groupRef, _ := identity.NewGroupRef(projectRef.String())
	repositoryRef, _ := identity.NewRepositoryRef(projectRef.String())
	hierarchy, err := identity.NewProjectHierarchy(identity.ProjectHierarchyInput{
		WorkspaceRef: workspaceRef, GroupRef: groupRef, GroupParentWorkspaceRef: workspaceRef,
		ProjectRef: projectRef, ProjectParentGroupRef: groupRef,
		RepositoryRef: repositoryRef, RepositoryParentProjectRef: projectRef,
	})
	if err != nil {
		t.Fatalf("new DAG hierarchy: %v", err)
	}
	if err := repository.ProvisionLocalAccess(
		context.Background(), principal, hierarchy, identity.RoleProjectOwner, clock.Now(),
	); err != nil {
		t.Fatalf("provision DAG access: %v", err)
	}
	access, err := application.NewAccess(principal, projectRef)
	if err != nil {
		t.Fatalf("new DAG access: %v", err)
	}
	agent := newDAGAgent(clock, failures)
	artifacts := newDAGArtifacts()
	workspace := newDAGWorkspace()
	versionControl := newDAGVersionControl()
	snapshot, err := config.Resolve(config.ResolveOptions{})
	if err != nil {
		t.Fatalf("resolve canonical DAG config: %v", err)
	}
	budgetPolicy, err := buildBudgetPolicy(snapshot, clock.Now())
	if err != nil {
		t.Fatalf("build canonical DAG budget policy: %v", err)
	}
	orchestrator, err := application.New(application.Dependencies{
		State: repository, Access: repository,
		Launcher: agent, Observer: agent, Artifacts: artifacts,
		WorkspaceManager: workspace, VersionControl: versionControl,
		Clock: clock, IDs: &dagSequentialIDs{}, MaxOutputBytes: 4096,
		MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts:    3,
		MaxChildrenPerParent:    int(snapshot.SchedulerMaxChildrenPerParent()),
		ClaimLease:              time.Minute, DirectorLeaseDuration: 2 * time.Minute,
		EffectApprovalTTL: snapshot.GovernanceEffectApprovalTTL(), BudgetPolicy: budgetPolicy,
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities: dagAgentCapabilities(),
	})
	if err != nil {
		t.Fatalf("new DAG orchestrator: %v", err)
	}
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	server, err := mcpiface.New(mcpiface.Config{
		Orchestrator: orchestrator, Identity: identity.ContextProvider{}, Catalog: catalog, Locale: "es",
		MaxListLimit: 10, MaxRequestBytes: 64 * 1024, Version: "dag-test",
	})
	if err != nil {
		t.Fatalf("new DAG MCP: %v", err)
	}
	httpServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		bound, bindErr := identity.BindPrincipal(request.Context(), principal)
		if bindErr != nil {
			http.Error(writer, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		server.Handler().ServeHTTP(writer, request.WithContext(bound))
	}))
	t.Cleanup(httpServer.Close)
	return &dagHarness{
		orchestrator: orchestrator, repository: repository, access: access, projectRef: projectRef,
		clock: clock, agent: agent, workspace: workspace, versionControl: versionControl,
		clientSession: connectDAGClient(t, httpServer.URL),
	}
}

func mustDAGRef[T any](t *testing.T, value string, constructor func(string) (T, error)) T {
	t.Helper()
	ref, err := constructor(value)
	if err != nil {
		t.Fatalf("parse DAG ref %q: %v", value, err)
	}
	return ref
}

func (harness *dagHarness) create(t *testing.T, requestRef, statement string, plan map[string]any) mcpiface.GoalView {
	t.Helper()
	result := callMCPTool(t, context.Background(), harness.clientSession, mcpiface.ToolGoalsCreate, map[string]any{
		"project_ref": harness.projectRef.String(), "request_ref": requestRef,
		"statement": statement, "confirm": true, "plan": plan,
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
	result := callMCPTool(t, context.Background(), harness.clientSession, mcpiface.ToolGoalsGet, map[string]any{
		"project_ref": harness.projectRef.String(), "goal_ref": goalRef,
	})
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
	return fmt.Sprintf("%s:dag-%06d", namespace, ids.next), nil
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
		if err := harness.admitCommittedChanges(context.Background()); err != nil {
			t.Fatalf("admit DAG integrations after step %d: %v", index, err)
		}
	}
}

// admitCommittedChanges is deliberately harness-owned. A completed agent only
// stages output; non-empty WriteSets require an explicit owner integration.
func (harness *dagHarness) admitCommittedChanges(ctx context.Context) error {
	changes, err := harness.orchestrator.ListPendingChanges(ctx, harness.access, application.ListPendingChangesRequest{Limit: 100})
	if err != nil {
		return err
	}
	for _, pending := range changes.Changes {
		_, err := harness.orchestrator.IntegrateChange(ctx, harness.access, application.IntegrateChangeRequest{
			RequestRef: "request:dag-integrate:" + pending.ChangeSet.Ref.String(),
			GoalRef:    pending.ChangeSet.GoalRef, ChangeRef: pending.ChangeSet.Ref,
			ExpectedTargetOID: pending.ChangeSet.BaseOID,
		})
		if err != nil {
			return err
		}
	}
	return nil
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
		ReceiptRef: "dag-launch:" + request.ExecutionRef.String(), AcceptedAt: agent.clock.Now(),
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
			ErrorCode: "dag_agent.failed", Usage: unknownTestUsage(), ObservedAt: agent.clock.Now(),
		}, nil
	}
	return ports.AgentObservation{
		ExecutionRef: executionRef, SpecHash: request.SpecHash, Status: ports.AgentCompleted, MediaType: request.ArtifactMediaType,
		Content: []byte("artifact:" + executionRef.String()), Usage: unknownTestUsage(), ObservedAt: agent.clock.Now(),
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

// dagWorkspace and dagVersionControl are neutral contractual fakes. They keep
// only opaque refs and deterministic object IDs; no filesystem or Git process
// participates in this MCP lifecycle test.
type dagWorkspace struct {
	mu       sync.Mutex
	prepared map[ports.ExecutionWorkspaceRef]ports.WorkspacePrepared
}

func newDAGWorkspace() *dagWorkspace {
	return &dagWorkspace{prepared: make(map[ports.ExecutionWorkspaceRef]ports.WorkspacePrepared)}
}

func (workspace *dagWorkspace) Prepare(_ context.Context, request ports.WorkspacePrepareRequest) (ports.WorkspacePrepared, error) {
	workspace.mu.Lock()
	defer workspace.mu.Unlock()
	if prepared, found := workspace.prepared[request.WorkspaceRef]; found {
		return prepared, nil
	}
	prepared := ports.WorkspacePrepared{
		WorkspaceRef: request.WorkspaceRef, RepositoryRef: request.RepositoryRef, ExecutionRef: request.ExecutionRef,
		TargetRef: "refs/heads/main", BaseOID: dagGitOID('a'), ObjectFormat: ports.GitObjectFormatSHA1,
		WriteSetDigest: request.WriteSetDigest, AdapterRef: "workspace-adapter:dag",
		ReceiptRef: "workspace-receipt:" + request.WorkspaceRef.String(), PreparedAt: request.PreparedAt,
	}
	workspace.prepared[request.WorkspaceRef] = prepared
	return prepared, nil
}

func (workspace *dagWorkspace) Inspect(_ context.Context, request ports.WorkspaceInspectRequest) (ports.WorkspaceInspection, error) {
	workspace.mu.Lock()
	defer workspace.mu.Unlock()
	prepared, found := workspace.prepared[request.WorkspaceRef]
	if !found {
		return ports.WorkspaceInspection{}, errors.New("dag_workspace.not_prepared")
	}
	return ports.WorkspaceInspection{WorkspaceRef: request.WorkspaceRef, RepositoryRef: request.RepositoryRef,
		ExecutionRef: request.ExecutionRef, BaseOID: prepared.BaseOID, HeadOID: prepared.BaseOID,
		TreeOID: dagGitOID('b'), WriteSetDigest: prepared.WriteSetDigest, AdapterRef: prepared.AdapterRef,
		InspectedAt: prepared.PreparedAt}, nil
}

func (workspace *dagWorkspace) Release(_ context.Context, request ports.WorkspaceReleaseRequest) (ports.WorkspaceReleaseReceipt, error) {
	return ports.WorkspaceReleaseReceipt{WorkspaceRef: request.WorkspaceRef, RepositoryRef: request.RepositoryRef,
		ExecutionRef: request.ExecutionRef, Released: false, ReceiptRef: "workspace-release:" + request.WorkspaceRef.String(),
		ReleasedAt: request.RequestedAt}, nil
}

type dagVersionControl struct {
	mu           sync.Mutex
	commits      map[ports.ChangeSetRef]ports.CommitResult
	integrations map[string]ports.IntegrationResult
}

func newDAGVersionControl() *dagVersionControl {
	return &dagVersionControl{commits: make(map[ports.ChangeSetRef]ports.CommitResult), integrations: make(map[string]ports.IntegrationResult)}
}

func (control *dagVersionControl) Commit(_ context.Context, request ports.CommitRequest) (ports.CommitResult, error) {
	control.mu.Lock()
	defer control.mu.Unlock()
	if result, found := control.commits[request.ChangeSetRef]; found {
		return result, nil
	}
	result := ports.CommitResult{ChangeSetRef: request.ChangeSetRef, WorkspaceRef: request.WorkspaceRef,
		RepositoryRef: request.RepositoryRef, ExecutionRef: request.ExecutionRef, BaseOID: request.BaseOID,
		ParentOID: request.BaseOID, HeadOID: dagGitOID('c'), TreeOID: dagGitOID('d'), ObjectFormat: request.ObjectFormat,
		DiffDigest: dagDigest("commit:" + request.ChangeSetRef.String()), ChangedPaths: append([]string(nil), request.WriteSet...),
		WriteSetDigest: request.WriteSetDigest, ParentChangeRef: request.ParentChangeRef, AdapterRef: "version-control:dag",
		ReceiptRef: "commit-receipt:" + request.ChangeSetRef.String(), CommittedAt: request.CommittedAt}
	control.commits[request.ChangeSetRef] = result
	return result, nil
}

func (control *dagVersionControl) PreviewIntegration(_ context.Context, request ports.IntegrationPreviewRequest) (ports.IntegrationPreview, error) {
	return ports.IntegrationPreview{ChangeSetRef: request.ChangeSetRef, RepositoryRef: request.RepositoryRef,
		SourceOID: request.SourceOID, TargetRef: request.TargetRef, TargetOID: request.TargetOID,
		ObjectFormat: request.ObjectFormat, Status: ports.MergeStatusClean, CandidateTreeOID: dagGitOID('e'),
		AdapterRef: "version-control:dag", ObservedAt: request.RequestedAt}, nil
}

func (control *dagVersionControl) Integrate(_ context.Context, request ports.IntegrationRequest) (ports.IntegrationResult, error) {
	control.mu.Lock()
	defer control.mu.Unlock()
	if result, found := control.integrations[request.IdempotencyKey]; found {
		return result, nil
	}
	result := ports.IntegrationResult{ChangeSetRef: request.ChangeSetRef, RepositoryRef: request.RepositoryRef,
		SourceOID: request.SourceOID, TargetRef: request.TargetRef, TargetBeforeOID: request.ExpectedTargetOID,
		TargetAfterOID: dagGitOID('f'), TreeOID: dagGitOID('e'), ObjectFormat: request.ObjectFormat,
		Status: ports.IntegrationStatusIntegrated, MarkerRef: "refs/orquesta/effects:dag", AdapterRef: "version-control:dag",
		ReceiptRef: "integration-receipt:" + request.ChangeSetRef.String(), RecordedAt: request.RequestedAt}
	control.integrations[request.IdempotencyKey] = result
	return result, nil
}

func dagGitOID(value byte) string { return strings.Repeat(string([]byte{value}), 40) }

func dagDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
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
