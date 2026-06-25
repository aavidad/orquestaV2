package orquestastatefile

import (
	"context"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestStoreV0AppDirectorGoalStateSobreviveRecreate(t *testing.T) {
	root := t.TempDir()
	store, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	state := appDirectorGoalStateForTestV0()
	if err := store.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	recovered, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatalf("NewStoreV0 recovered: %v", err)
	}
	got, err := recovered.LoadGoalWorkStateV0(context.Background(), state.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if got.RunRef != state.RunRef ||
		got.GoalRef != state.GoalRef ||
		got.ExternalGoalRef != state.ExternalGoalRef ||
		got.LastResult == nil ||
		got.LastResult.Status != orquestagoal.GoalStatusRunningV0 {
		t.Fatalf("got=%+v", got)
	}
}

func TestStoreV0AppDirectorGoalStateRechazaRunRefInconsistente(t *testing.T) {
	store, err := NewStoreV0(ConfigV0{RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	state := appDirectorGoalStateForTestV0()
	state.Spec.RunRef = "run-ref-state-file-distinto"
	if err := store.SaveGoalWorkStateV0(context.Background(), state); err == nil {
		t.Fatalf("esperaba error por run_ref inconsistente")
	}
}

func appDirectorGoalStateForTestV0() orquestagoal.GoalWorkStateV0 {
	spec := orquestagoal.NormalizeGoalWorkSpecV0(orquestagoal.GoalWorkSpecV0{
		GoalRef:      "goal-ref-state-file-app-001",
		RunRef:       "run-ref-state-file-app-001",
		Objective:    "Construir app desde goal",
		DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
		WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "generated-apps/state-file-app"}},
		ArtifactContracts: []orquestagoal.GoalArtifactContractV0{
			{ArtifactRef: "artifact-ref-state-file-source", ArtifactType: "source_tree", Required: true},
		},
	})
	result := orquestagoal.NormalizeGoalWorkResultV0(orquestagoal.GoalWorkResultV0{
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: "thread-ref-state-file-app-001",
		EvidenceRefs:    []string{"evidence-ref-state-file-observed"},
	})
	return orquestagoal.GoalWorkStateV0{
		SchemaVersion:   orquestagoal.GoalWorkStateSchemaV0,
		RunRef:          spec.RunRef,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: "thread-ref-state-file-app-001",
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec:            spec,
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         spec.GoalRef,
			ExternalGoalRef: "thread-ref-state-file-app-001",
		},
		LastResult:   &result,
		EvidenceRefs: []string{"evidence-ref-state-file-launched"},
	}
}
