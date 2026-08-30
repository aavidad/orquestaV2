package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/tooling"
)

type toolRecorder struct {
	sync.Mutex
	requests []ToolExecutionRequest
	output   []byte
	usage    governance.ResourceUsage
	err      error
	change   func(*ToolObservation)
	now      func() time.Time
}

func (recorder *toolRecorder) InvokeTool(_ context.Context, request ToolExecutionRequest) (ToolObservation, error) {
	recorder.Lock()
	defer recorder.Unlock()
	recorder.requests = append(recorder.requests, request)
	if recorder.err != nil {
		return ToolObservation{}, recorder.err
	}
	observation := ToolObservation{
		ToolID: request.ToolID, Version: request.Version, RequestRef: request.RequestRef,
		IdempotencyKey: request.IdempotencyKey, SpecDigest: request.SpecDigest, Catalog: request.Catalog,
		CapabilityRef: request.CapabilityRef, Execution: request.Execution, PrincipalRef: request.PrincipalRef,
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		AuthorizationReceiptRefs: append([]string(nil), request.AuthorizationReceiptRefs...),
		Output:                   append([]byte(nil), recorder.output...), ObservedAt: recorder.now().UTC(), Usage: recorder.usage,
	}
	if recorder.change != nil {
		recorder.change(&observation)
	}
	if observation.ReceiptRef == "" {
		observation.ReceiptRef, _ = ToolObservationReceiptRef(request, observation.Output, observation.Usage, observation.ObservedAt)
	}
	return observation, nil
}

func (recorder *toolRecorder) calls() []ToolExecutionRequest {
	recorder.Lock()
	defer recorder.Unlock()
	return append([]ToolExecutionRequest(nil), recorder.requests...)
}

func TestInvokeToolAdmissionIsManualCuratedAndExactlyBound(t *testing.T) {
	otherGoal, _ := goal.NewGoalRef("goal:other")
	otherWorkItem, _ := goal.NewWorkItemRef("work-item:other")
	otherExecution, _ := goal.NewExecutionRef("execution:other")
	type testCase struct {
		name      string
		raw       string
		edit      func(*toolFixture, *InvokeToolRequest)
		code      string
		auth      int
		forbidden bool
	}
	tests := []testCase{
		{"raw-before-hash-decode", strings.Repeat(" ", maxToolRawInputBytes) + `{}`, nil, ToolErrorInputInvalid, 0, false},
		{"catalog", "", func(_ *toolFixture, r *InvokeToolRequest) {
			r.Curation.CatalogDigest = "sha256:" + strings.Repeat("0", 64)
		}, ToolErrorRequestInvalid, 0, false},
		{"tool-ref", "", func(_ *toolFixture, r *InvokeToolRequest) {
			r.Curation.Execution.ToolRef, _ = goal.NewToolRef("tool:other.read")
		}, ToolErrorRequestInvalid, 0, false},
		{"capability", "", func(_ *toolFixture, r *InvokeToolRequest) {
			r.Curation.CapabilityRef = wizardCapabilityRef(t, "TLS-11")
		}, ToolErrorNotFound, 1, false},
		{"goal", "", func(_ *toolFixture, r *InvokeToolRequest) { r.Curation.Execution.GoalRef = otherGoal }, ToolErrorNotFound, 1, false},
		{"work-item", "", func(_ *toolFixture, r *InvokeToolRequest) { r.Curation.Execution.WorkItemRef = otherWorkItem }, ToolErrorNotFound, 1, false},
		{"execution", "", func(_ *toolFixture, r *InvokeToolRequest) { r.Curation.Execution.ExecutionRef = otherExecution }, ToolErrorNotFound, 1, false},
		{"attempt", "", func(_ *toolFixture, r *InvokeToolRequest) { r.Curation.Execution.ExecutionAttempt++ }, ToolErrorNotFound, 1, false},
		{"plan-generation", "", func(_ *toolFixture, r *InvokeToolRequest) { r.Curation.Execution.PlanGeneration++ }, ToolErrorNotFound, 1, false},
		{"appspec-generation", "", func(_ *toolFixture, r *InvokeToolRequest) { r.Curation.Execution.AppSpecGeneration++ }, ToolErrorNotFound, 1, false},
		{"spec-hash", "", func(_ *toolFixture, r *InvokeToolRequest) { r.Curation.Execution.SpecHash = strings.Repeat("0", 64) }, ToolErrorNotFound, 1, false},
		{"input-schema", `{"query":""}`, nil, ToolErrorInputInvalid, 1, false},
		{"service-execution-access", "", func(f *toolFixture, r *InvokeToolRequest) {
			service := testPrincipal(t, "principal:tool-service", "actor:tool-service", identity.PrincipalKindService)
			r.Access, _ = NewExecutionAccess(service, f.project, f.binding.ExecutionRef)
		}, "", 0, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newToolFixture(t, toolSpec(identity.PermissionArtifactsRead), true)
			request := fixture.request(`{"query":"status"}`)
			if test.raw != "" {
				request.Input = json.RawMessage(test.raw)
			}
			if test.edit != nil {
				test.edit(fixture, &request)
			}
			_, err := fixture.orchestrator.InvokeTool(context.Background(), request)
			if ToolInvocationErrorCode(err) != test.code ||
				(test.forbidden && !errors.Is(err, ErrForbidden)) || len(fixture.executor.calls()) != 0 ||
				fixture.authorizations() != test.auth {
				t.Fatalf("error=%v auth=%d calls=%d", err, fixture.authorizations(), len(fixture.executor.calls()))
			}
		})
	}
}

func TestInvokeToolEnforcesPolicyCurationRBACAndCurrentAuthority(t *testing.T) {
	spec := toolSpec(identity.PermissionEffectsApprove)
	spec.Idempotency, spec.Receipt = tooling.IdempotencyRequired, tooling.ReceiptApplication
	policy := newToolFixture(t, spec, true)
	if _, err := policy.orchestrator.InvokeTool(context.Background(), policy.request(`{"query":"status"}`)); ToolInvocationErrorCode(err) != ToolErrorPolicyUnsupported || len(policy.executor.calls()) != 0 {
		t.Fatalf("policy error=%v", err)
	}

	denied := newToolFixture(t, toolSpec(identity.PermissionArtifactsRead, identity.PermissionChangesIntegrate), true)
	denied.accessStore.setRole(denied.principal.Ref, denied.project, identity.RoleViewer)
	if _, err := denied.orchestrator.InvokeTool(context.Background(), denied.request(`{"query":"status"}`)); !errors.Is(err, ErrForbidden) || len(denied.executor.calls()) != 0 {
		t.Fatalf("RBAC error=%v", err)
	}
	denied.accessStore.setRole(denied.principal.Ref, denied.project, identity.RoleReviewer)
	if !denied.orchestrator.toolAuthorizationCurrent(context.Background(), denied.principal.Ref, denied.project, identity.RoleReviewer, 1) ||
		denied.orchestrator.toolAuthorizationCurrent(context.Background(), denied.principal.Ref, denied.project, identity.RoleReviewer, 2) {
		t.Fatal("membership revision was not fenced")
	}
	denied.accessStore.setRole(denied.principal.Ref, denied.project, "")
	if denied.orchestrator.toolAuthorizationCurrent(context.Background(), denied.principal.Ref, denied.project, identity.RoleReviewer, 3) {
		t.Fatal("revoked membership remained current")
	}
}

func TestInvokeToolRequiresRunningCurrentSubjectAndRechecksIt(t *testing.T) {
	for _, state := range []string{"pending", "terminal", "replaced", "superseded", "duplicate"} {
		t.Run(state, func(t *testing.T) {
			fixture := newToolFixture(t, toolSpec(identity.PermissionArtifactsRead), state != "pending")
			if state != "pending" {
				fixture.invalidate(t, state)
			}
			if _, err := fixture.orchestrator.InvokeTool(context.Background(), fixture.request(`{"query":"status"}`)); ToolInvocationErrorCode(err) != ToolErrorNotFound || len(fixture.executor.calls()) != 0 {
				t.Fatalf("state=%s error=%v", state, err)
			}
		})
	}
	fixture := newToolFixture(t, toolSpec(identity.PermissionArtifactsRead), true)
	changing := &changingToolExecutionState{StateRepository: fixture.orchestrator.state, executionRef: fixture.binding.ExecutionRef}
	fixture.orchestrator.state = changing
	if _, err := fixture.orchestrator.InvokeTool(context.Background(), fixture.request(`{"query":"status"}`)); ToolInvocationErrorCode(err) != ToolErrorNotFound || changing.calls != 2 || len(fixture.executor.calls()) != 0 {
		t.Fatalf("recheck reads=%d error=%v", changing.calls, err)
	}
}

func TestInvokeToolValidatesObservationUsageReceiptAndDelivery(t *testing.T) {
	fixture := newToolFixture(t, toolSpec(identity.PermissionArtifactsRead), true)
	fixture.executor.usage.Quality = governance.UsageQualityMeasured
	raw := strings.Repeat(" ", maxToolRawInputBytes-len(`{"query":"status"}`)) + `{"query":"status"}`
	result, err := fixture.orchestrator.InvokeTool(context.Background(), fixture.request(raw))
	calls := fixture.executor.calls()
	if err != nil || len(calls) != 1 || string(result.InlineOutput) != `{"value":"ok"}` || result.Execution != fixture.binding || result.ArtifactStored ||
		calls[0].Execution != fixture.binding || calls[0].Catalog.Digest != fixture.orchestrator.curatedToolCatalog.Digest() ||
		calls[0].CapabilityRef.String() != "TLS-12" || len(calls[0].AuthorizationReceiptRefs) != 2 {
		t.Fatalf("result=%+v calls=%+v error=%v", result, calls, err)
	}
	want, receiptErr := ToolObservationReceiptRef(calls[0], result.InlineOutput, result.Usage, result.ObservedAt)
	if receiptErr != nil || want != result.ObservationReceiptRef {
		t.Fatalf("receipt=%q want=%q error=%v", result.ObservationReceiptRef, want, receiptErr)
	}

	tests := []struct {
		name, code string
		change     func(*ToolObservation)
	}{
		{"project", ToolErrorReceiptInvalid, func(v *ToolObservation) { v.ProjectRef, _ = goal.NewProjectRef("project:other") }},
		{"authorization", ToolErrorReceiptInvalid, func(v *ToolObservation) { v.AuthorizationReceiptRefs = []string{"authorization-receipt:other"} }},
		{"future", ToolErrorReceiptInvalid, func(v *ToolObservation) { v.ObservedAt = time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC) }},
		{"known-zero", ToolErrorReceiptInvalid, func(v *ToolObservation) { v.Usage.Known = 0 }},
		{"known-missing", ToolErrorReceiptInvalid, func(v *ToolObservation) { v.Usage.Known &^= governance.ResourceDisk }},
		{"known-extra", ToolErrorReceiptInvalid, func(v *ToolObservation) { v.Usage.Known |= governance.ResourceDimensions(1 << 7) }},
		{"quality-unknown", ToolErrorReceiptInvalid, func(v *ToolObservation) { v.Usage.Quality = governance.UsageQualityUnknown }},
		{"cost", ToolErrorReceiptInvalid, func(v *ToolObservation) { v.Usage.Resources.DiskBytes = 4097 }},
		{"receipt", ToolErrorReceiptInvalid, func(v *ToolObservation) { v.ReceiptRef = "tool-observation-receipt:invented" }},
		{"schema", ToolErrorOutputInvalid, func(v *ToolObservation) { v.Output = []byte(`{"unknown":true}`) }},
		{"canonical", ToolErrorOutputInvalid, func(v *ToolObservation) { v.Output = []byte(" \n{\"value\":\"ok\"}") }},
		{"too-large", ToolErrorOutputTooLarge, func(v *ToolObservation) { v.Output = []byte(`{"value":"` + strings.Repeat("x", 5000) + `"}`) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newToolFixture(t, toolSpec(identity.PermissionArtifactsRead), true)
			fixture.executor.change = test.change
			_, err := fixture.orchestrator.InvokeTool(context.Background(), fixture.request(`{"query":"status"}`))
			if ToolInvocationErrorCode(err) != test.code || len(fixture.executor.calls()) != 1 {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestInvokeToolSpillFailuresAndConcurrency(t *testing.T) {
	fixture := newToolFixture(t, toolSpec(identity.PermissionArtifactsRead), true)
	value := strings.Repeat("x", 200)
	fixture.executor.output = []byte(fmt.Sprintf(`{"value":%q}`, value))
	result, err := fixture.orchestrator.InvokeTool(context.Background(), fixture.request(`{"query":"status"}`))
	content, getErr := fixture.artifacts.Get(context.Background(), result.Artifact.Ref, result.Artifact.Size)
	if err != nil || getErr != nil || !result.ArtifactStored || result.InlineOutput != nil || string(content.Content) != string(fixture.executor.output) {
		t.Fatalf("spill=%+v error=%v/%v", result, err, getErr)
	}

	bad := newToolFixture(t, toolSpec(identity.PermissionArtifactsRead), true)
	bad.executor.output = fixture.executor.output
	bad.orchestrator.artifacts = invalidArtifactStore{}
	if _, err := bad.orchestrator.InvokeTool(context.Background(), bad.request(`{"query":"status"}`)); ToolInvocationErrorCode(err) != ToolErrorArtifactFailed {
		t.Fatalf("artifact error=%v", err)
	}
	leak := newToolFixture(t, toolSpec(identity.PermissionArtifactsRead), true)
	leak.executor.err = errors.New("connector secret-value")
	if _, err := leak.orchestrator.InvokeTool(context.Background(), leak.request(`{"query":"secret-value"}`)); ToolInvocationErrorCode(err) != ToolErrorConnectorFailed || strings.Contains(fmt.Sprintf("%v", err), "secret-value") || errors.Unwrap(err) != nil {
		t.Fatalf("leaked error=%#v unwrap=%v", err, errors.Unwrap(err))
	}
	concurrent := newToolFixture(t, toolSpec(identity.PermissionArtifactsRead), true)
	const workers = 16
	errorsByWorker := make(chan error, workers)
	for index := range workers {
		go func() {
			request := concurrent.request(`{"query":"status"}`)
			request.RequestRef = fmt.Sprintf("request:tool:%d", index)
			_, err := concurrent.orchestrator.InvokeTool(context.Background(), request)
			errorsByWorker <- err
		}()
	}
	for range workers {
		err := <-errorsByWorker
		if err != nil {
			t.Fatal(err)
		}
	}
}

type toolFixture struct {
	*controlTestSystem
	executor   *toolRecorder
	artifacts  *memoryArtifactStore
	binding    ToolExecutionBinding
	capability goal.CapabilityRef
	principal  identity.Principal
	project    goal.ProjectRef
}

func newToolFixture(t *testing.T, spec tooling.CapabilitySpec, running bool) *toolFixture {
	plan := &PlanSpec{Phases: []PhaseSpec{{Ref: "phase-instance:tools", Key: "phase:tools", TemplateRef: "phase-template:tools"}}, WorkItems: []WorkItemSpec{{
		Key: "tool", Objective: "observe", Phase: "phase:tools", Role: "role:worker",
		ToolRefs: []string{"tool:status.read", "tool:workspace.read"}, CapabilityRefs: []string{"TLS-12"}, OutputContract: goal.OutputContractEvidenceBundle,
	}}}
	system := newControlTestSystemWithPlan(t, &scriptedAgent{}, plan)
	registry, err := tooling.NewRegistry(spec)
	if err != nil {
		t.Fatal(err)
	}
	catalog := toolCatalog(t, registry)
	recorder := &toolRecorder{now: system.clock.Now, output: []byte(`{"value":"ok"}`), usage: governance.ResourceUsage{Known: governance.AllResourceDimensions, Quality: governance.UsageQualityExact}}
	system.orchestrator.toolRegistry, system.orchestrator.curatedToolCatalog, system.orchestrator.toolExecutor = registry, catalog, recorder
	if running {
		system.launch(t)
	}
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	executionRef, _ := goal.NewExecutionRef("execution:pending")
	binding := ToolExecutionBinding{ToolRef: item.ToolRefs()[0], GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: executionRef,
		PlanGeneration: record.Goal.PlanGeneration(), AppSpecGeneration: record.Goal.AppSpec().Generation(), ExecutionAttempt: 1, SpecHash: record.Goal.SpecHash()}
	if current, bound := item.Execution(); bound {
		execution, _ := executionByRef(record.Executions, current)
		binding.ExecutionRef, binding.PlanGeneration, binding.AppSpecGeneration, binding.ExecutionAttempt, binding.SpecHash = execution.Ref, execution.PlanGeneration, execution.AppSpecGeneration, execution.AttemptNo, execution.SpecHash
	}
	principal, project, _ := system.access.values()
	system.accessStore.mu.Lock()
	system.accessStore.authorizations = nil
	system.accessStore.mu.Unlock()
	return &toolFixture{controlTestSystem: system, executor: recorder, artifacts: system.orchestrator.artifacts.(*memoryArtifactStore), binding: binding, capability: wizardCapabilityRef(t, "TLS-12"), principal: principal, project: project}
}

func (fixture *toolFixture) request(raw string) InvokeToolRequest {
	return InvokeToolRequest{Access: fixture.access, RequestRef: "request:tool", ToolID: "status.read", Version: "1",
		Curation: ToolCurationRequest{CatalogDigest: fixture.orchestrator.curatedToolCatalog.Digest(), CapabilityRef: fixture.capability, Execution: fixture.binding}, Input: json.RawMessage(raw)}
}
func (fixture *toolFixture) authorizations() int {
	fixture.accessStore.mu.Lock()
	defer fixture.accessStore.mu.Unlock()
	return len(fixture.accessStore.authorizations)
}
func (fixture *toolFixture) invalidate(t *testing.T, state string) {
	fixture.repository.mu.Lock()
	defer fixture.repository.mu.Unlock()
	record := fixture.repository.records[fixture.goalRef]
	item, _ := record.Goal.WorkItem(fixture.binding.WorkItemRef)
	at := fixture.clock.Now().Add(time.Second)
	switch state {
	case "terminal":
		record.Goal, _ = record.Goal.FailWorkItem(record.Goal.Revision(), item.Revision(), item.Ref(), at)
		record.Executions[0].State = ExecutionFailed
	case "replaced":
		replacement, _ := goal.NewExecutionRef("execution:replacement")
		record.Goal, _ = record.Goal.ReplaceWorkItemExecution(record.Goal.Revision(), item.Revision(), item.Ref(), fixture.binding.ExecutionRef, replacement, at)
	case "superseded":
		ref, _ := goal.NewWorkItemRef("work-item:successor")
		successor, _ := goal.NewWorkItem(goal.NewWorkItemInput{Ref: ref, Goal: record.Goal.Ref(), Actor: record.Goal.Actor(), Project: record.Goal.Project(), Objective: "replacement", CreatedAt: at, Phase: item.Phase(), Role: item.Role(), ToolRefs: item.ToolRefs(), CapabilityRefs: item.CapabilityRefs(), OutputContract: item.OutputContract()})
		record.Goal, _ = record.Goal.ApplyReplan(record.Goal.Revision(), goal.ReplanInput{ExpectedPlanGeneration: record.Goal.PlanGeneration(), Source: item.Ref(), ExpectedSourceRevision: item.Revision(), Cause: goal.ReplanCauseGovernanceDecision, CausalExecution: fixture.binding.ExecutionRef, Successors: []goal.WorkItem{successor}, At: at})
	case "duplicate":
		record.Executions = append(record.Executions, record.Executions[0])
	}
	fixture.repository.records[fixture.goalRef] = record
}

type changingToolExecutionState struct {
	StateRepository
	executionRef goal.ExecutionRef
	calls        int
}

func (state *changingToolExecutionState) GetGoal(ctx context.Context, ref goal.GoalRef) (GoalRecord, error) {
	record, err := state.StateRepository.GetGoal(ctx, ref)
	state.calls++
	if err == nil && state.calls == 2 {
		for index := range record.Executions {
			if record.Executions[index].Ref == state.executionRef {
				record.Executions[index].State = ExecutionStopped
			}
		}
	}
	return record, err
}

func toolCatalog(t *testing.T, registry *tooling.Registry) *tooling.CuratedCatalog {
	registration, _ := registry.Lookup("status.read", "1")
	skills, _ := tooling.NewSkillRegistry(registry)
	releases, _ := tooling.NewSkillReleaseCatalog(skills, nil, nil, nil)
	plugins, _ := tooling.NewPluginCatalog(registry, releases)
	catalog, err := tooling.NewCuratedCatalog(registry, releases, plugins, tooling.CuratedCatalogSpec{ID: "tool.execution", Version: "1", DescriptionKey: "curation.tool.execution.description", ReviewRef: "review:tool-execution:1", ReviewDigest: "sha256:" + strings.Repeat("a", 64), Capabilities: []string{"TLS-12"}, Scopes: []tooling.SkillScope{{Kind: tooling.SkillScopeProject, ProjectRef: "project:controls"}}, Tools: []tooling.CuratedToolSelection{{ID: "status.read", Version: "1", SpecDigest: registration.Digest}}})
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}
func toolSpec(permissions ...identity.Permission) tooling.CapabilitySpec {
	return tooling.CapabilitySpec{ID: "status.read", Version: "1", InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","minLength":1,"maxLength":64}},"required":["query"],"additionalProperties":false}`), OutputSchema: json.RawMessage(`{"type":"object","properties":{"value":{"type":"string","minLength":1,"maxLength":4096}},"required":["value"],"additionalProperties":false}`), Permissions: permissions, Cost: tooling.CostContract{Mode: tooling.CostMaximum, Maximum: governance.ResourceVector{Tokens: 100, ActiveTimeNS: int64(time.Second), DiskBytes: 4096}}, Output: tooling.OutputDelivery{MaxBytes: 4096, InlineBytes: 64}, Idempotency: tooling.IdempotencyReadReexecute, Receipt: tooling.ReceiptObservation}
}
