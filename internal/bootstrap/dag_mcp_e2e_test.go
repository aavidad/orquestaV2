package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	commandcore "orquesta/internal/commands"
	"orquesta/internal/config"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/i18n"
	"orquesta/internal/identity"
	mcpiface "orquesta/internal/interfaces/mcp"
	"orquesta/internal/ports"
	"orquesta/internal/review"
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

	// Writer lifecycle is prepare -> launch -> observe -> commit -> attest -> integrate.
	harness.process(t, 6)
	afterRoot := harness.get(t, created.GoalRef)
	status, err := harness.repository.Status(context.Background(), harness.projectRef)
	if err != nil || len(afterRoot.Executions) != 5 || len(afterRoot.Artifacts) != 5 ||
		len(afterRoot.Attestations) != 2 || status.PendingActions != 2 {
		t.Fatalf("atomic fan-out = executions:%d artifacts:%d status:%+v err:%v", len(afterRoot.Executions), len(afterRoot.Artifacts), status, err)
	}

	harness.process(t, 4)
	if harness.agent.concurrent() != 2 {
		t.Fatalf("disjoint successors were not concurrently launchable: %d", harness.agent.concurrent())
	}
	harness.process(t, 8)
	afterBranches := harness.get(t, created.GoalRef)
	if len(afterBranches.Executions) != 10 || afterBranches.State != string(goal.GoalStateRunning) {
		t.Fatalf("join was not scheduled exactly once: %+v", afterBranches)
	}
	harness.processAvailable(t, 64)
	closed := harness.get(t, created.GoalRef)
	if closed.State != string(goal.GoalStateSucceeded) || len(closed.Executions) != 12 ||
		len(closed.Artifacts) != 20 || len(closed.Attestations) != 8 {
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
	result := harness.callCreate(t, "request:mcp-invalid-plan", "invalid plan", map[string]any{
		"phases": []any{}, "work_items": []any{},
	})
	if !result.IsError || result.Result.Failure == nil || result.Result.Failure.Code != "invalid_request" {
		t.Fatalf("invalid typed plan result=%+v", result)
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
	result := harness.callCreate(t, "request:mcp-v05-parent-invalid", "reject unknown parent", map[string]any{
		"phases":     []any{planPhase("phase-instance:work", "phase:work", "phase-template:program", nil, nil)},
		"work_items": []any{item},
	})
	if !result.IsError || result.Result.Failure == nil || result.Result.Failure.Code != "invalid_request" {
		t.Fatalf("unknown parent result=%+v failure=%+v", result.Result, result.Result.Failure)
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
				{Key: "a", Objective: "writer a", Phase: "phase:work", Role: "role:worker", WriteSet: []string{"internal/shared"}, CouncilPolicy: council.PolicySkipByOperator, RequiredTests: dagRequiredTests("a"), OutputContract: goal.OutputContractEvidenceBundle},
				{Key: "b", Objective: "writer b", Phase: "phase:work", Role: "role:worker", WriteSet: []string{"internal/shared/file.go"}, CouncilPolicy: council.PolicySkipByOperator, RequiredTests: dagRequiredTests("b"), OutputContract: goal.OutputContractEvidenceBundle},
				{Key: "c", Objective: "writer c", Phase: "phase:work", Role: "role:worker", WriteSet: []string{"docs/free.md"}, CouncilPolicy: council.PolicySkipByOperator, RequiredTests: dagRequiredTests("c"), OutputContract: goal.OutputContractEvidenceBundle},
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
	byObjective := make(map[string]dagWorkItemView, len(stored.WorkItems))
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

	harness.process(t, 8)
	closed := harness.get(t, created.GoalRef)
	states := workStates(closed.WorkItems)
	status, statusErr := harness.repository.Status(ctx, harness.projectRef)
	idle, idleErr := harness.orchestrator.ProcessNext(ctx, "worker:dag:idle-parent-metadata")
	if closed.State != string(goal.GoalStateSucceeded) ||
		states["coordinate"] != string(goal.WorkItemStateSucceeded) ||
		states["implement"] != string(goal.WorkItemStateSucceeded) ||
		len(closed.Artifacts) != 6 || len(closed.Attestations) != 3 ||
		len(closed.Executions) != 4 || harness.agent.launchCount() != 2 ||
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
	harness.process(t, 4)
	harness.clock.Advance(2 * time.Second)
	harness.process(t, 6)
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
	harness.processAvailable(t, 64)
	harness.clock.Advance(time.Second)
	harness.processAvailable(t, 64)
	harness.clock.Advance(2 * time.Second)
	harness.processAvailable(t, 64)
	open := harness.get(t, created.GoalRef)
	states = workStates(open.WorkItems)
	if open.State != string(goal.GoalStateRunning) || open.ClosedAt != nil ||
		states["fail root"] != string(goal.WorkItemStateInterrupted) ||
		states["child"] != string(goal.WorkItemStatePending) ||
		states["grandchild"] != string(goal.WorkItemStatePending) ||
		states["independent"] != string(goal.WorkItemStateSucceeded) ||
		len(open.Executions) != 6 || len(open.Artifacts) != 5 || len(open.Attestations) != 2 {
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
	item := map[string]any{
		"key": key, "objective": objective, "phase": phase, "role": "role:worker",
		"dependencies": append([]string{}, dependencies...), "write_set": append([]string{}, writeSet...),
		"output_contract": string(goal.OutputContractEvidenceBundle),
	}
	if len(writeSet) != 0 {
		item["council_policy"] = string(council.PolicySkipByOperator)
		item["required_tests"] = []any{map[string]any{
			"ref": "required-test:dag-" + key, "tool_ref": "tool:go",
			"arguments": []string{"test", "./..."}, "working_directory": ".",
		}}
	}
	return item
}

func dagRequiredTests(key string) []application.RequiredTestSpec {
	return []application.RequiredTestSpec{{
		Ref: "required-test:dag-" + key, ToolRef: "tool:go",
		Arguments: []string{"test", "./..."}, WorkingDirectory: ".",
	}}
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

func workStates(items []dagWorkItemView) map[string]string {
	result := make(map[string]string, len(items))
	for _, item := range items {
		result[item.Objective] = item.State
	}
	return result
}

type dagPhaseView struct {
	PhaseRef      string
	TemplateRef   string
	InputRefs     []string
	CriterionRefs []string
}

type dagWorkItemView struct {
	WorkItemRef    string
	Objective      string
	ParentRef      string
	ChildRefs      []string
	State          string
	SkillRefs      []string
	ToolRefs       []string
	CapabilityRefs []string
}

type dagGoalView struct {
	GoalRef        string
	State          string
	PlanGeneration uint64
	ClosedAt       *time.Time
	Phases         []dagPhaseView
	WorkItems      []dagWorkItemView
	Executions     []application.ExecutionRecord
	Artifacts      []application.ArtifactRecord
	Attestations   []application.AttestationRecord
}

func projectDAGGoal(record application.GoalRecord) dagGoalView {
	snapshot := record.Goal.Snapshot()
	phases := make([]dagPhaseView, 0, len(snapshot.Phases))
	for _, phase := range snapshot.Phases {
		phases = append(phases, dagPhaseView{
			PhaseRef: phase.Ref, TemplateRef: phase.TemplateRef,
			InputRefs:     append([]string(nil), phase.InputRefs...),
			CriterionRefs: append([]string(nil), phase.CriterionRefs...),
		})
	}
	children := make(map[string][]string, len(snapshot.WorkItems))
	for _, item := range snapshot.WorkItems {
		if item.ParentRef != "" {
			children[item.ParentRef] = append(children[item.ParentRef], item.Ref)
		}
	}
	items := make([]dagWorkItemView, 0, len(snapshot.WorkItems))
	for _, item := range snapshot.WorkItems {
		items = append(items, dagWorkItemView{
			WorkItemRef: item.Ref, Objective: item.Objective, ParentRef: item.ParentRef,
			ChildRefs: append([]string(nil), children[item.Ref]...), State: string(item.State),
			SkillRefs: append([]string(nil), item.SkillRefs...), ToolRefs: append([]string(nil), item.ToolRefs...),
			CapabilityRefs: append([]string(nil), item.CapabilityRefs...),
		})
	}
	var closedAt *time.Time
	if !snapshot.ClosedAt.IsZero() {
		value := snapshot.ClosedAt
		closedAt = &value
	}
	return dagGoalView{
		GoalRef: snapshot.Ref, State: string(snapshot.State), PlanGeneration: uint64(snapshot.PlanGeneration),
		ClosedAt: closedAt, Phases: phases, WorkItems: items,
		Executions:   append([]application.ExecutionRecord(nil), record.Executions...),
		Artifacts:    append([]application.ArtifactRecord(nil), record.Artifacts...),
		Attestations: append([]application.AttestationRecord(nil), record.Attestations...),
	}
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
		TestAttestor: legacyPassingTestAttestor{}, TestAttestationPolicy: legacyTestAttestationPolicy(),
		Clock: clock, IDs: &dagSequentialIDs{}, MaxOutputBytes: 4096,
		MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts:    3,
		MaxChildrenPerParent:    int(snapshot.SchedulerMaxChildrenPerParent()),
		ClaimLease:              time.Minute, AttestTestClaimLease: time.Minute, DirectorLeaseDuration: 2 * time.Minute,
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
	dispatcher, err := commandcore.NewDispatcher(orchestrator, repository, commandcore.APILimits{
		MaxListLimit: 10, MaxRequestBytes: 64 * 1024,
	})
	if err != nil {
		t.Fatalf("new DAG dispatcher: %v", err)
	}
	server, err := mcpiface.New(mcpiface.Config{
		Dispatcher: dispatcher, Identity: identity.ContextProvider{}, Catalog: catalog, Locale: "es",
		Version: "dag-test",
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

type dagCommandCall struct {
	Result  commandcore.Result
	IsError bool
}

func (harness *dagHarness) callCreate(t *testing.T, requestRef, statement string, plan map[string]any) dagCommandCall {
	t.Helper()
	result := callMCPTool(t, context.Background(), harness.clientSession, "orquesta.goals.create", map[string]any{
		"version": "1", "project_ref": harness.projectRef.String(), "request_ref": requestRef,
		"payload": map[string]any{"statement": statement, "confirm": true, "plan": plan},
	})
	var output mcpiface.CommandToolOutput
	decodeMCPOutput(t, result, &output)
	return dagCommandCall{Result: output.Result, IsError: result.IsError}
}

func (harness *dagHarness) create(t *testing.T, requestRef, statement string, plan map[string]any) dagGoalView {
	t.Helper()
	result := harness.callCreate(t, requestRef, statement, plan)
	if result.IsError || result.Result.Failure != nil {
		t.Fatalf("create DAG result=%+v failure=%+v", result.Result, result.Result.Failure)
	}
	var data struct {
		Goal struct {
			GoalRef string `json:"goal_ref"`
		} `json:"goal"`
	}
	if err := json.Unmarshal(result.Result.Data, &data); err != nil || data.Goal.GoalRef == "" {
		t.Fatalf("decode create DAG data=%s err=%v", result.Result.Data, err)
	}
	return harness.get(t, data.Goal.GoalRef)
}

func (harness *dagHarness) get(t *testing.T, goalRef string) dagGoalView {
	t.Helper()
	ref, err := goal.NewGoalRef(goalRef)
	if err != nil {
		t.Fatalf("parse DAG goal ref: %v", err)
	}
	record, err := harness.orchestrator.GetGoal(context.Background(), harness.access, ref)
	if err != nil {
		t.Fatalf("get DAG goal: %v", err)
	}
	return projectDAGGoal(record)
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
		if result.Action == application.ActionAttestTest {
			if err := harness.completeReviewsAndAdmit(context.Background(), index); err != nil {
				t.Fatalf("admit DAG integrations after step %d: %v", index, err)
			}
		}
	}
}

func (harness *dagHarness) processAvailable(t *testing.T, limit int) {
	t.Helper()
	for index := 0; index < limit; index++ {
		result, err := harness.orchestrator.ProcessNext(context.Background(), fmt.Sprintf("worker:dag:available:%d", index))
		if err != nil {
			var stateErr *application.StateError
			if errors.As(err, &stateErr) {
				t.Fatalf("process available DAG step %d = %+v state=%s cause=%v", index, result, stateErr.Code, stateErr.Cause)
			}
			t.Fatalf("process available DAG step %d = %+v err=%v", index, result, err)
		}
		if !result.Processed {
			return
		}
		if result.Action == application.ActionAttestTest {
			if err := harness.completeReviewsAndAdmit(context.Background(), index); err != nil {
				t.Fatalf("admit available DAG integrations after step %d: %v", index, err)
			}
		}
	}
	t.Fatalf("DAG still had work after %d available steps", limit)
}

func (harness *dagHarness) completeReviewsAndAdmit(ctx context.Context, step int) error {
	for attempt := 0; attempt < 64; attempt++ {
		changes, err := harness.orchestrator.ListPendingChanges(ctx, harness.access, application.ListPendingChangesRequest{Limit: 100})
		if err != nil {
			return err
		}
		waiting := false
		for _, pending := range changes.Changes {
			record, err := harness.orchestrator.GetGoal(ctx, harness.access, pending.ChangeSet.GoalRef)
			if err != nil {
				return err
			}
			if !dagHasRequiredTestPass(record, pending.ChangeSet.Ref) {
				waiting = true
				continue
			}
			if !dagReviewPairSucceeded(record, pending.ChangeSet) {
				waiting = true
				continue
			}
			if err = harness.authorizeCouncilSkip(ctx, record, pending.ChangeSet); err != nil {
				return fmt.Errorf("skip Council for %s: %w", pending.ChangeSet.Ref, err)
			}
			if _, err = harness.orchestrator.IntegrateChange(ctx, harness.access, application.IntegrateChangeRequest{
				RequestRef: "request:dag-integrate:" + pending.ChangeSet.Ref.String(),
				GoalRef:    pending.ChangeSet.GoalRef, ChangeRef: pending.ChangeSet.Ref,
				ExpectedTargetOID: pending.ChangeSet.BaseOID,
			}); err != nil {
				return fmt.Errorf("integrate %s: %w", pending.ChangeSet.Ref, err)
			}
		}
		if !waiting {
			return nil
		}
		result, err := harness.orchestrator.ProcessNext(ctx, fmt.Sprintf("worker:dag:%d:review:%d", step, attempt))
		if err != nil || !result.Processed {
			return fmt.Errorf("process review round = %+v: %w", result, err)
		}
	}
	return errors.New("dag_harness.review_round_did_not_settle")
}

func (harness *dagHarness) authorizeCouncilSkip(
	ctx context.Context,
	record application.GoalRecord,
	change application.ChangeSet,
) error {
	for _, existing := range record.CouncilSkips {
		if existing.Subject.ChangeSetRef == change.Ref.String() {
			return nil
		}
	}
	item, found := record.Goal.WorkItem(change.WorkItemRef)
	if !found {
		return errors.New("dag_harness.skip_work_item_not_found")
	}
	_, err := harness.orchestrator.SkipCouncil(ctx, harness.access, application.SkipCouncilRequest{
		RequestRef:           "request:dag-council-skip:" + change.Ref.String(),
		GoalRef:              change.GoalRef,
		ChangeRef:            change.Ref,
		ExpectedGoalRevision: record.Goal.Revision(),
		ExpectedItemRevision: item.Revision(),
		Reason:               "human test operator approved legacy DAG integration",
	})
	return err
}

func dagHasRequiredTestPass(record application.GoalRecord, changeRef ports.ChangeSetRef) bool {
	for _, attestation := range record.Attestations {
		if attestation.Kind == application.AttestationKindRequiredTests &&
			attestation.Verdict == application.AttestationVerdictPassed &&
			attestation.ChangeSetRef == changeRef {
			return true
		}
	}
	return false
}

func dagReviewPairSucceeded(record application.GoalRecord, change application.ChangeSet) bool {
	primary, adversarial := false, false
	for _, execution := range record.Executions {
		if execution.WorkItemRef != change.WorkItemRef || execution.ExecutionWorkspaceRef != change.WorkspaceRef ||
			execution.State != application.ExecutionSucceeded || execution.ReviewSubjectDigest == "" {
			continue
		}
		switch execution.Purpose {
		case application.ExecutionPurposePrimaryReview:
			primary = true
		case application.ExecutionPurposeAdversarialReview:
			adversarial = true
		}
	}
	return primary && adversarial
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
	if request.ArtifactMediaType != review.AssessmentMediaType {
		agent.launches++
		concurrent := 0
		for ref := range agent.inFlight {
			if agent.requests[ref].ArtifactMediaType != review.AssessmentMediaType {
				concurrent++
			}
		}
		if concurrent > agent.maxConcurrent {
			agent.maxConcurrent = concurrent
		}
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
	if request.ArtifactMediaType == review.AssessmentMediaType {
		role := review.RolePrimary
		if strings.Contains(request.Objective, `"role":"adversarial"`) {
			role = review.RoleAdversarial
		}
		payload, err := json.Marshal(review.Artifact{
			SchemaVersion: 1, SubjectDigest: dagReviewSubjectDigest(request.Objective),
			Role: role, Verdict: review.VerdictApprove, Summary: "DAG exact subject approved",
			Findings: []review.Finding{},
		})
		if err != nil {
			return ports.AgentObservation{}, err
		}
		return ports.AgentObservation{
			ExecutionRef: executionRef, SpecHash: request.SpecHash, Status: ports.AgentCompleted,
			MediaType: review.AssessmentMediaType, Content: payload,
			Usage: unknownTestUsage(), ObservedAt: agent.clock.Now(),
		}, nil
	}
	return ports.AgentObservation{
		ExecutionRef: executionRef, SpecHash: request.SpecHash, Status: ports.AgentCompleted, MediaType: request.ArtifactMediaType,
		Content: []byte("artifact:" + executionRef.String()), Usage: unknownTestUsage(), ObservedAt: agent.clock.Now(),
	}, nil
}

func dagReviewSubjectDigest(objective string) string {
	const prefix = "Review exact immutable evidence "
	value := strings.TrimPrefix(objective, prefix)
	end := strings.Index(value, ". Inspect with ")
	if end < 0 {
		return ""
	}
	var evidence struct {
		SubjectDigest string `json:"subject_digest"`
	}
	if json.Unmarshal([]byte(value[:end]), &evidence) != nil {
		return ""
	}
	return evidence.SubjectDigest
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
