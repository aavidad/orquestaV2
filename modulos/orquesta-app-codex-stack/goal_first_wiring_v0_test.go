package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
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
	if !ok || stats.GoalStateSource == nil {
		t.Fatalf("director stats sin goal state source: ok=%v stats=%+v", ok, stats)
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
	calls int
}

func (launcher *codexStackGoalLauncherForTestV0) LaunchGoalWorkV0(
	_ context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	launcher.calls++
	return orquestagoal.GoalLaunchReceiptV0{
		SchemaVersion: orquestagoal.GoalWorkLaunchReceiptSchemaV0,
		Status:        orquestagoal.GoalStatusRunningV0,
		GoalRef:       spec.GoalRef,
	}, nil
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
