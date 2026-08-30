package acceptance_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
	"orquesta/internal/tooling"
)

func TestV26ToolExecutionContractBindsRunningSQLiteSubjectAndReplaysAdmission(t *testing.T) {
	ctx := context.Background()
	clock := &v06Clock{now: time.Date(2026, 8, 21, 16, 30, 0, 0, time.UTC)}
	registry, curated := v26ExecutionCatalog(t)
	path := v06PrivateDatabasePath(t, "v26-tools.sqlite")
	repository := v06OpenSQLite(t, ctx, path, clock)
	agent := newV06Agent(clock, ports.AgentCapabilities{ProviderRef: "provider:v06", ModelRef: "model:v06", AgentRef: "agent:v06", Unrestricted: true})
	artifacts, executor := newV06ArtifactStore(), &v26ToolExecutor{clock: clock, output: []byte(`{"value":"observed"}`)}
	dependencies := v26ExecutionDependencies(t, repository, clock, agent, artifacts, registry, curated, executor)
	orchestrator, err := application.New(dependencies)
	if err != nil {
		t.Fatal(err)
	}
	submitted, err := orchestrator.Submit(ctx, v06Access(t), v06SubmitRequest(t, "request:v26-tool-goal", &v06OpaqueRequirements{
		RoleKey: "role:worker", ToolRefs: []string{"tool:status.read"}, CapabilityRefs: []string{"TLS-12"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	for attempts := 0; attempts < 3; attempts++ {
		processed, processErr := orchestrator.ProcessNext(ctx, "worker:v26-tool")
		if processErr != nil {
			t.Fatal(processErr)
		}
		if processed.Action == application.ActionLaunchAgent {
			break
		}
	}
	record := v06GetGoal(t, repository, submitted.Record.Goal.Ref())
	item := record.Goal.WorkItems()[0]
	executionRef, bound := item.Execution()
	execution := record.Executions[0]
	if !bound || execution.Ref != executionRef || item.State() != goal.WorkItemStateRunning || execution.State != application.ExecutionRunning {
		t.Fatalf("item=%+v execution=%+v", item, execution)
	}
	toolRef, _ := goal.NewToolRef("tool:status.read")
	capability, _ := goal.NewCapabilityRef("TLS-12")
	binding := application.ToolExecutionBinding{ToolRef: toolRef, GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		PlanGeneration: execution.PlanGeneration, AppSpecGeneration: execution.AppSpecGeneration, ExecutionAttempt: execution.AttemptNo, SpecHash: execution.SpecHash}
	request := application.InvokeToolRequest{Access: v06Access(t), RequestRef: "request:v26-tool", ToolID: "status.read", Version: "1",
		Curation: application.ToolCurationRequest{CatalogDigest: curated.Digest(), CapabilityRef: capability, Execution: binding}, Input: json.RawMessage(`{"query":"status"}`)}
	first, err := orchestrator.InvokeTool(ctx, request)
	executed, _ := executor.snapshot()
	wantReceipt, receiptErr := application.ToolObservationReceiptRef(executed, first.InlineOutput, first.Usage, first.ObservedAt)
	if err != nil || receiptErr != nil || string(first.InlineOutput) != `{"value":"observed"}` || first.Execution != binding ||
		first.Catalog.Digest != curated.Digest() || len(first.AuthorizationReceiptRefs) != 3 || first.ObservationReceiptRef != wantReceipt {
		t.Fatalf("first=%+v error=%v receipt_error=%v", first, err, receiptErr)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	repository = v06OpenSQLite(t, ctx, path, clock)
	t.Cleanup(func() { _ = repository.Close() })
	dependencies = v26ExecutionDependencies(t, repository, clock, agent, artifacts, registry, curated, executor)
	orchestrator, err = application.New(dependencies)
	if err != nil {
		t.Fatal(err)
	}
	second, err := orchestrator.InvokeTool(ctx, request)
	_, calls := executor.snapshot()
	if err != nil || calls != 2 || second.ObservationReceiptRef != first.ObservationReceiptRef {
		t.Fatalf("restart=%+v first=%+v error=%v calls=%d", second, first, err, calls)
	}
	request.Input = json.RawMessage(`{"query":"different"}`)
	if _, err := orchestrator.InvokeTool(ctx, request); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("divergent replay error=%v", err)
	}
}

type v26ToolExecutor struct {
	sync.Mutex
	clock  *v06Clock
	output []byte
	last   application.ToolExecutionRequest
	calls  int
}

func (executor *v26ToolExecutor) InvokeTool(_ context.Context, request application.ToolExecutionRequest) (application.ToolObservation, error) {
	executor.Lock()
	defer executor.Unlock()
	executor.last, executor.calls = request, executor.calls+1
	observation := application.ToolObservation{ToolID: request.ToolID, Version: request.Version, RequestRef: request.RequestRef,
		IdempotencyKey: request.IdempotencyKey, SpecDigest: request.SpecDigest, Catalog: request.Catalog,
		CapabilityRef: request.CapabilityRef, Execution: request.Execution, PrincipalRef: request.PrincipalRef,
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef, AuthorizationReceiptRefs: append([]string(nil), request.AuthorizationReceiptRefs...),
		Output: append([]byte(nil), executor.output...), ObservedAt: executor.clock.Now(),
		Usage: governance.ResourceUsage{Known: governance.AllResourceDimensions, Quality: governance.UsageQualityExact, Resources: governance.ResourceVector{Tokens: 1}}}
	observation.ReceiptRef, _ = application.ToolObservationReceiptRef(request, observation.Output, observation.Usage, observation.ObservedAt)
	return observation, nil
}
func (executor *v26ToolExecutor) snapshot() (application.ToolExecutionRequest, int) {
	executor.Lock()
	defer executor.Unlock()
	return executor.last, executor.calls
}

func v26ExecutionCatalog(t *testing.T) (*tooling.Registry, *tooling.CuratedCatalog) {
	t.Helper()
	spec := tooling.CapabilitySpec{ID: "status.read", Version: "1",
		InputSchema:  json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","minLength":1}},"required":["query"],"additionalProperties":false}`),
		OutputSchema: json.RawMessage(`{"type":"object","properties":{"value":{"type":"string","minLength":1}},"required":["value"],"additionalProperties":false}`),
		Permissions:  []identity.Permission{identity.PermissionArtifactsRead, identity.PermissionChangesIntegrate},
		Cost:         tooling.CostContract{Mode: tooling.CostMaximum, Maximum: governance.ResourceVector{Tokens: 100, ActiveTimeNS: int64(time.Second), DiskBytes: 4096}},
		Output:       tooling.OutputDelivery{MaxBytes: 4096, InlineBytes: 64}, Idempotency: tooling.IdempotencyReadReexecute, Receipt: tooling.ReceiptObservation}
	registry, err := tooling.NewRegistry(spec)
	if err != nil {
		t.Fatal(err)
	}
	registration, _ := registry.Lookup(spec.ID, spec.Version)
	skills, _ := tooling.NewSkillRegistry(registry)
	releases, _ := tooling.NewSkillReleaseCatalog(skills, nil, nil, nil)
	plugins, _ := tooling.NewPluginCatalog(registry, releases)
	curated, err := tooling.NewCuratedCatalog(registry, releases, plugins, tooling.CuratedCatalogSpec{ID: "tool.execution", Version: "1",
		DescriptionKey: "curation.tool.execution.description", ReviewRef: "review:tool-execution:acceptance", ReviewDigest: "sha256:" + strings.Repeat("a", 64),
		Capabilities: []string{"TLS-12"}, Scopes: []tooling.SkillScope{{Kind: tooling.SkillScopeProject, ProjectRef: v06Project(t).String()}},
		Tools: []tooling.CuratedToolSelection{{ID: spec.ID, Version: spec.Version, SpecDigest: registration.Digest}}})
	if err != nil {
		t.Fatal(err)
	}
	return registry, curated
}

func v26ExecutionDependencies(t *testing.T, repository v06StateAccessRepository, clock *v06Clock, agent *v06Agent, artifacts application.ArtifactStore, registry *tooling.Registry, curated *tooling.CuratedCatalog, executor application.ToolExecutor) application.Dependencies {
	t.Helper()
	policy := v06BudgetPolicy(t, clock.Now())
	return application.Dependencies{State: repository, Access: repository, Launcher: agent, Observer: agent, Artifacts: artifacts,
		ToolRegistry: registry, CuratedToolCatalog: curated, ToolExecutor: executor, Clock: clock, IDs: &v06IDs{}, MaxOutputBytes: 1 << 20,
		MaxMailboxEnvelopeBytes: 64 << 10, MaxExecutionAttempts: 3, MaxChildrenPerParent: 6, ClaimLease: time.Minute,
		DirectorLeaseDuration: 2 * time.Minute, EffectApprovalTTL: policy.EffectApprovalTTL, BudgetPolicy: policy,
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour, AgentCapabilities: agent.capabilities,
		CapacityObservationWait: time.Second, CapacitySources: []application.FuenteCapacidadColocacionAgente{v06ProvisionCapacity(t, repository, artifacts, clock)}}
}
