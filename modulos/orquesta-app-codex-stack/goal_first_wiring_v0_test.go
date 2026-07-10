package orquestaappcodexstack

import (
	"context"
	"errors"
	"strings"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestBuildDirectorPortsV0CableaAppGoalLauncher(t *testing.T) {
	launcher := &codexStackGoalLauncherForTestV0{}
	reworkLauncher := &codexStackGoalLauncherForTestV0{}
	observer := &codexStackGoalObserverForTestV0{}
	stateStore := &codexStackGoalStateStoreForTestV0{}
	ports := buildDirectorPortsV0(ConfigV0{
		AppGoalLauncher:       launcher,
		AppGoalReworkLauncher: reworkLauncher,
		AppGoalObserver:       observer,
		Stores:                StoresV0{AppGoalStateStore: stateStore},
	})
	if ports.GoalLauncher == nil ||
		ports.GoalReworkLauncher == nil ||
		ports.GoalObserver == nil ||
		ports.GoalClosureValidator == nil ||
		ports.GoalStateStore == nil ||
		ports.GoalFirstRunMarkerStore == nil {
		t.Fatalf("puertos goal-first incompletos: %+v", ports)
	}
	receipt, err := ports.GoalLauncher.LaunchGoalWorkV0(context.Background(), orquestagoal.GoalWorkSpecV0{
		GoalRef:      "goal-ref-stack-wiring-001",
		Objective:    "Probar cableado goal-first.",
		DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
		WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "generated-apps/test"}},
		EvidenceRefs: []string{"evidence-ref-stack-wiring-001"},
		RuleRefs:     []orquestagoal.GoalRuleRefV0{{Ref: "AGENTS.md"}},
		ContextRefs:  []orquestagoal.GoalContextRefV0{{Ref: "context-ref-stack-wiring-001"}},
	})
	if err != nil || receipt.GoalRef != "goal-ref-stack-wiring-001" || launcher.calls != 1 {
		t.Fatalf("receipt=%+v err=%v calls=%d", receipt, err, launcher.calls)
	}
	reworkReceipt, err := ports.GoalReworkLauncher.LaunchGoalWorkV0(context.Background(), orquestagoal.GoalWorkSpecV0{
		GoalRef:      "goal-ref-stack-wiring-rework-001",
		Objective:    "Probar cableado de rework goal-first.",
		DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
		WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "generated-apps/test"}},
		EvidenceRefs: []string{"evidence-ref-stack-wiring-rework-001"},
		RuleRefs:     []orquestagoal.GoalRuleRefV0{{Ref: "AGENTS.md"}},
		ContextRefs:  []orquestagoal.GoalContextRefV0{{Ref: "context-ref-stack-wiring-rework-001"}},
	})
	if err != nil || reworkReceipt.GoalRef != "goal-ref-stack-wiring-rework-001" || reworkLauncher.calls != 1 {
		t.Fatalf("reworkReceipt=%+v err=%v calls=%d", reworkReceipt, err, reworkLauncher.calls)
	}
	result, err := ports.GoalObserver.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef: "goal-ref-stack-wiring-001",
	})
	if err != nil || result.GoalRef != "goal-ref-stack-wiring-001" || observer.calls != 1 {
		t.Fatalf("result=%+v err=%v calls=%d", result, err, observer.calls)
	}
	bindings := buildStackMCPTransportBindingsV0(
		ConfigV0{Stores: StoresV0{AppGoalStateStore: stateStore}},
		ports,
		RunQueueConfigV0{},
		&StackV0{},
	)
	stats, ok := bindings.DirectorStats.(orquestamcp.MCPDirectorStatsToolExecutorV0)
	if !ok || stats.GoalStateSource == nil || stats.GoalMarkerSource == nil {
		t.Fatalf("director stats sin goal state source: ok=%v stats=%+v", ok, stats)
	}
}

func TestBuildDirectorPortsV0CableaAttestorIndependienteOptIn(t *testing.T) {
	attestor := &codexStackRequiredTestAttestorForTestV0{}
	store := &codexStackRequiredTestAttestationStoreForTestV0{}
	verifier := independentIdentityVerifierForStackTestV0{}
	binder := independentSpecBinderForStackTestV0{}
	snapshotter := independentSnapshotObserverForStackTestV0{}
	ports := buildDirectorPortsV0(ConfigV0{
		AppGoalRequiredTestSpecBinder:       binder,
		AppGoalRequiredTestSnapshotObserver: snapshotter,
		AppGoalRequiredTestAttestor:         attestor,
		AppGoalRequiredTestIdentityVerifier: verifier,
		Stores:                              StoresV0{GoalRequiredTestAttestationStore: store},
	})
	if ports.GoalRequiredTestSpecBinder == nil || ports.GoalRequiredTestSnapshotObserver == nil ||
		ports.GoalRequiredTestAttestor != attestor || ports.GoalRequiredTestAttestationStore != store ||
		ports.GoalRequiredTestIdentityVerifier == nil {
		t.Fatalf("wiring attestation incompleto: %+v", ports)
	}
	if _, ok := ports.GoalClosureValidator.(orquestagoal.IndependentGoalRequiredTestAttestationClosureValidatorV0); !ok {
		t.Fatalf("closure validator no exige reader independiente: %T", ports.GoalClosureValidator)
	}
}

func TestValidateConfigV0RejectsPartialRequiredTestAttestationPorts(t *testing.T) {
	err := validateConfigV0(ConfigV0{AppGoalRequiredTestAttestor: &codexStackRequiredTestAttestorForTestV0{}})
	if err == nil || !strings.Contains(err.Error(), "goal_required_test_attestation_config_incomplete") {
		t.Fatalf("err=%v", err)
	}
}

func TestBuildDirectorPortsV0CableaPoliticaAutonomaV0(t *testing.T) {
	policy := &codexStackAutonomousDirectorPolicyForTestV0{}
	ports := buildDirectorPortsV0(ConfigV0{AutonomousDirectorPolicy: policy})
	if ports.AutonomousDirectorPolicy == nil {
		t.Fatalf("politica autonoma no cableada")
	}
	decision, err := ports.AutonomousDirectorPolicy.DecideAutonomousDirectorV0(
		context.Background(),
		orquestacionnucleoapp.AutonomousDirectorDecisionInputV0{
			Run: orquestacoreworkflow.OrchestrationRunV0{
				RunID: "run-stack-policy-001",
				Tasks: []string{"task-1", "task-2", "task-3"},
			},
			Stats: orquestacionnucleoapp.DirectorRunStatsV0{
				RunRef: "run-stack-policy-001",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					TasksOpen: 3,
				},
			},
		},
	)
	if err != nil || policy.calls != 1 || decision.TeamSize != 3 {
		t.Fatalf("decision=%+v err=%v calls=%d", decision, err, policy.calls)
	}
}

func TestBuildDirectorPortsV0PrecargaCodeContextParaGoalCodigo(t *testing.T) {
	launcher := &codexStackGoalLauncherForTestV0{}
	codeContext := &codexStackCodeContextForTestV0{
		result: orquestacontext.CodeContextResultV0{
			SchemaVersion: orquestacontext.CodeContextResultSchemaVersionV0,
			Estado:        orquestacontext.CodeContextEstadoOKV0,
			QueryHash:     "code-context-sha256-test",
			CacheStatus:   orquestacontext.CodeContextCacheStoreV0,
			EvidenceRefs:  []string{"evidence-ref-code-context-prepared"},
		},
	}
	ports := buildDirectorPortsV0(ConfigV0{
		AppGoalLauncher: launcher,
		CodeContext:     codeContext,
	})

	receipt, err := ports.GoalLauncher.LaunchGoalWorkV0(context.Background(), orquestagoal.GoalWorkSpecV0{
		GoalRef:      "goal-ref-stack-code-context-001",
		ProjectRef:   "repo-ref-orquesta",
		Objective:    "Modificar broker de codigo.",
		DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
		WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "cmd/orquesta-server"}},
		RuleRefs:     []orquestagoal.GoalRuleRefV0{{Ref: "AGENTS.md"}},
		ContextRefs:  []orquestagoal.GoalContextRefV0{{Ref: "context-ref-existing"}},
	})
	if err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	if codeContext.calls != 1 ||
		codeContext.last.QueryKind != orquestacontext.CodeContextQueryKindRepoMapV0 ||
		len(codeContext.last.Scope) != 1 ||
		codeContext.last.Scope[0] != "cmd/orquesta-server" {
		t.Fatalf("query=%+v calls=%d", codeContext.last, codeContext.calls)
	}
	launched := launcher.lastSpec
	if !goalContextRefPrefixForTestV0(launched.ContextRefs, "code_context_prepared:repo_map:code-context-sha256-test") ||
		!goalStringInSetForTestV0(launched.EvidenceRefs, "evidence-ref-code-context-prepared") {
		t.Fatalf("spec no enriquecido: %+v", launched)
	}
	if receipt.ContextBudget.CodeContextCacheStatus != orquestacontext.CodeContextCacheStoreV0 {
		t.Fatalf("receipt sin cache status: %+v", receipt.ContextBudget)
	}
}

func TestBuildDirectorPortsV0NoPrecargaCodeContextParaGoalDocumental(t *testing.T) {
	launcher := &codexStackGoalLauncherForTestV0{}
	codeContext := &codexStackCodeContextForTestV0{}
	ports := buildDirectorPortsV0(ConfigV0{
		AppGoalLauncher: launcher,
		CodeContext:     codeContext,
	})

	_, err := ports.GoalLauncher.LaunchGoalWorkV0(context.Background(), orquestagoal.GoalWorkSpecV0{
		GoalRef:      "goal-ref-stack-doc-context-001",
		ProjectRef:   "repo-ref-orquesta",
		Objective:    "Actualizar documentacion.",
		DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
		WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		RuleRefs:     []orquestagoal.GoalRuleRefV0{{Ref: "AGENTS.md"}},
	})
	if err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	if codeContext.calls != 0 {
		t.Fatalf("codeContext calls=%d", codeContext.calls)
	}
	if goalContextRefPrefixForTestV0(launcher.lastSpec.ContextRefs, "code_context_prepared:") {
		t.Fatalf("spec documental enriquecido: %+v", launcher.lastSpec.ContextRefs)
	}
}

func TestBuildDirectorPortsV0CodeContextFallidoNoBloqueaGoal(t *testing.T) {
	launcher := &codexStackGoalLauncherForTestV0{}
	codeContext := &codexStackCodeContextForTestV0{err: errors.New("broker_down")}
	ports := buildDirectorPortsV0(ConfigV0{
		AppGoalLauncher: launcher,
		CodeContext:     codeContext,
	})

	receipt, err := ports.GoalLauncher.LaunchGoalWorkV0(context.Background(), orquestagoal.GoalWorkSpecV0{
		GoalRef:      "goal-ref-stack-code-context-failed-001",
		ProjectRef:   "repo-ref-orquesta",
		Objective:    "Modificar codigo con broker caido.",
		DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
		WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-context"}},
		RuleRefs:     []orquestagoal.GoalRuleRefV0{{Ref: "AGENTS.md"}},
	})
	if err != nil || receipt.GoalRef != "goal-ref-stack-code-context-failed-001" {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
	if !goalContextRefPrefixForTestV0(launcher.lastSpec.ContextRefs, "code_context_prepare_failed") {
		t.Fatalf("spec sin diagnostico de fallo: %+v", launcher.lastSpec.ContextRefs)
	}
}

func TestQueuedArrancarDirectorExecutorV0NoEncolaGoalFirst(t *testing.T) {
	writer := &queuedGoalFirstWriterForTestV0{}
	executor := NewQueuedArrancarDirectorExecutorV0(QueuedArrancarDirectorConfigV0{
		Inner:  queuedGoalFirstInnerForTestV0{},
		Writer: writer,
		Queue:  RunQueueConfigV0{QueueRef: "global", DefaultPriorityScore: 50},
	})

	result, err := executor.Execute(context.Background(), orquestamcp.MCPArrancarDirectorAppToolInputV0{})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.GoalRef != "goal-ref-queued-goal-first-001" || writer.calls != 0 {
		t.Fatalf("result=%+v writer_calls=%d", result, writer.calls)
	}
}

type codexStackGoalLauncherForTestV0 struct {
	calls    int
	lastSpec orquestagoal.GoalWorkSpecV0
}

type codexStackAutonomousDirectorPolicyForTestV0 struct {
	calls int
}

func (policy *codexStackAutonomousDirectorPolicyForTestV0) DecideAutonomousDirectorV0(
	_ context.Context,
	input orquestacionnucleoapp.AutonomousDirectorDecisionInputV0,
) (orquestacionnucleoapp.AutonomousDirectorDecisionV0, error) {
	policy.calls++
	return orquestacionnucleoapp.AutonomousDirectorDecisionV0{
		TeamSize:     input.Stats.Counts.TasksOpen,
		EvidenceRefs: []string{"evidence-ref-stack-policy"},
	}, nil
}

func (launcher *codexStackGoalLauncherForTestV0) LaunchGoalWorkV0(
	_ context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	launcher.calls++
	launcher.lastSpec = spec
	return orquestagoal.GoalLaunchReceiptV0{
		SchemaVersion: orquestagoal.GoalWorkLaunchReceiptSchemaV0,
		Status:        orquestagoal.GoalStatusRunningV0,
		GoalRef:       spec.GoalRef,
	}, nil
}

type codexStackCodeContextForTestV0 struct {
	calls  int
	last   orquestacontext.CodeContextQueryV0
	result orquestacontext.CodeContextResultV0
	err    error
}

func (fake *codexStackCodeContextForTestV0) QueryCodeContextV0(
	_ context.Context,
	query orquestacontext.CodeContextQueryV0,
) (orquestacontext.CodeContextResultV0, error) {
	fake.calls++
	fake.last = query
	if fake.err != nil {
		return orquestacontext.CodeContextResultV0{}, fake.err
	}
	if fake.result.SchemaVersion == "" {
		fake.result = orquestacontext.CodeContextResultV0{
			SchemaVersion: orquestacontext.CodeContextResultSchemaVersionV0,
			Estado:        orquestacontext.CodeContextEstadoOKV0,
			QueryHash:     "code-context-sha256-default",
			CacheStatus:   orquestacontext.CodeContextCacheStoreV0,
		}
	}
	return fake.result, nil
}

func goalContextRefPrefixForTestV0(refs []orquestagoal.GoalContextRefV0, prefix string) bool {
	for _, ref := range refs {
		if strings.HasPrefix(ref.Ref, prefix) {
			return true
		}
	}
	return false
}

func goalStringInSetForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

type codexStackGoalObserverForTestV0 struct {
	calls int
}

func (observer *codexStackGoalObserverForTestV0) ObserveGoalWorkV0(
	_ context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	observer.calls++
	return orquestagoal.GoalWorkResultV0{
		SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
		Status:        orquestagoal.GoalStatusRunningV0,
		GoalRef:       request.GoalRef,
	}, nil
}

type codexStackGoalStateStoreForTestV0 struct{}

type codexStackRequiredTestAttestorForTestV0 struct{}

func (codexStackRequiredTestAttestorForTestV0) AttestGoalRequiredTestsV0(
	context.Context,
	orquestagoal.GoalRequiredTestAttestationRequestV0,
) ([]orquestagoal.GoalRequiredTestAttestationV0, error) {
	return nil, nil
}

type codexStackRequiredTestAttestationStoreForTestV0 struct{}

func (codexStackRequiredTestAttestationStoreForTestV0) SaveGoalRequiredTestAttestationV0(
	context.Context,
	orquestagoal.GoalRequiredTestAttestationV0,
) error {
	return nil
}

func (codexStackRequiredTestAttestationStoreForTestV0) ListGoalRequiredTestAttestationsV0(
	context.Context,
	orquestagoal.GoalRequiredTestAttestationQueryV0,
) ([]orquestagoal.GoalRequiredTestAttestationV0, error) {
	return nil, nil
}

func (codexStackRequiredTestAttestationStoreForTestV0) FreezeGoalRequiredTestFinalSnapshotV0(
	_ context.Context,
	snapshot orquestagoal.GoalRequiredTestFinalSnapshotV0,
) (orquestagoal.GoalRequiredTestFinalSnapshotV0, error) {
	return snapshot, nil
}

func (codexStackRequiredTestAttestationStoreForTestV0) LoadGoalRequiredTestFinalSnapshotV0(
	context.Context,
	string,
	string,
) (orquestagoal.GoalRequiredTestFinalSnapshotV0, error) {
	return orquestagoal.GoalRequiredTestFinalSnapshotV0{}, nil
}

func (codexStackRequiredTestAttestationStoreForTestV0) AcquireGoalRequiredTestAttestationClaimV0(
	context.Context,
	orquestagoal.GoalRequiredTestAttestationClaimRequestV0,
) (orquestagoal.GoalRequiredTestAttestationClaimResultV0, error) {
	return orquestagoal.GoalRequiredTestAttestationClaimResultV0{}, nil
}

func (codexStackRequiredTestAttestationStoreForTestV0) CompleteGoalRequiredTestAttestationClaimV0(
	context.Context,
	orquestagoal.GoalRequiredTestAttestationClaimV0,
	orquestagoal.GoalRequiredTestAttestationV0,
) error {
	return nil
}

func (codexStackGoalStateStoreForTestV0) SaveGoalWorkStateV0(
	context.Context,
	orquestagoal.GoalWorkStateV0,
) error {
	return nil
}

func (codexStackGoalStateStoreForTestV0) LoadGoalWorkStateV0(
	context.Context,
	string,
) (orquestagoal.GoalWorkStateV0, error) {
	return orquestagoal.GoalWorkStateV0{}, nil
}

func (codexStackGoalStateStoreForTestV0) SaveGoalWorkRunMarkerV0(
	context.Context,
	orquestagoal.GoalWorkRunMarkerV0,
) error {
	return nil
}

func (codexStackGoalStateStoreForTestV0) LoadGoalWorkRunMarkerV0(
	context.Context,
	string,
) (orquestagoal.GoalWorkRunMarkerV0, error) {
	return orquestagoal.GoalWorkRunMarkerV0{}, nil
}

type queuedGoalFirstInnerForTestV0 struct{}

func (queuedGoalFirstInnerForTestV0) Execute(
	context.Context,
	orquestamcp.MCPArrancarDirectorAppToolInputV0,
) (orquestamcp.MCPArrancarDirectorAppToolResultV0, error) {
	return orquestamcp.MCPArrancarDirectorAppToolResultV0{
		Estado:  orquestamcp.MCPArrancarDirectorAppEstadoOKV0,
		RunRef:  "run-ref-queued-goal-first-001",
		GoalRef: "goal-ref-queued-goal-first-001",
	}, nil
}

type queuedGoalFirstWriterForTestV0 struct {
	calls int
}

func (writer *queuedGoalFirstWriterForTestV0) SetRunPriorityV0(
	_ context.Context,
	command orquestarunqueue.RunQueuePriorityCommandV0,
) (orquestarunqueue.RunSchedulingCandidateV0, error) {
	writer.calls++
	return orquestarunqueue.RunSchedulingCandidateV0{RunRef: command.RunRef}, nil
}
