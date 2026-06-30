package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestCodexStackObserveAppDirectorGoalExecutorV0TimeoutSnapshotIncluyeProcessRefsV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-stack-observe-goal-snapshot-001"
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       "goal-ref-stack-observe-snapshot-001",
			RunRef:        runRef,
			Objective:     "Publicar snapshot parcial con proceso vivo.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet: []orquestagoal.GoalWriteScopeV0{{
				Path: "docs/goal_snapshot.md",
			}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         "goal-ref-stack-observe-snapshot-001",
			ExternalGoalRef: "thread-ref-stack-observe-snapshot-001",
		},
		EvidenceRefs: []string{"evidence-ref-stack-observe-snapshot-state-001"},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	if err := goalStates.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	processRegistry := orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0()
	if err := processRegistry.RecordAgentProcessV0(ctx, orquestaagentprocessregistrymemory.AgentProcessRecordV0{
		RunID:          runRef,
		AgentRequestID: "agent-ref-stack-observe-snapshot-001",
		ProcessRef:     "process-ref-stack-observe-snapshot-001",
		SessionRef:     "session-ref-stack-observe-snapshot-001",
		LaunchRef:      "launch-ref-stack-observe-snapshot-001",
		ReadinessRef:   "readiness-ref-stack-observe-snapshot-001",
		EvidenceRefs:   []string{"evidence-ref-stack-observe-snapshot-process-001"},
	}); err != nil {
		t.Fatalf("RecordAgentProcessV0: %v", err)
	}
	stack := &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalStateStore: goalStates,
		},
		Stores: StoresV0{
			ProcessRegistry: processRegistry,
		},
	}

	result, err := NewCodexStackObserveAppDirectorGoalExecutorV0(stack).ObserveAppDirectorGoalTimeoutSnapshotV0(
		ctx,
		orquestamcp.MCPObserveAppDirectorGoalToolInputV0{RunRef: runRef},
	)

	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalTimeoutSnapshotV0: %v", err)
	}
	if !result.Partial ||
		result.GoalStatus != orquestagoal.GoalStatusRunningV0 ||
		!codexStackStringInSetForTestV0(result.ProcessRefs, "process-ref-stack-observe-snapshot-001") ||
		!codexStackStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-stack-observe-snapshot-process-001") {
		t.Fatalf("result=%+v", result)
	}
}
