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

func TestStoreV0AppDirectorGoalFirstRunMarkerSobreviveRecreate(t *testing.T) {
	root := t.TempDir()
	store, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	marker := orquestagoal.GoalWorkRunMarkerV0{
		RunRef:          "run-ref-state-file-goal-marker-001",
		GoalRef:         "goal-ref-state-file-goal-marker-001",
		ExternalGoalRef: "thread-ref-state-file-goal-marker-001",
		DirectorKind:    orquestagoal.GoalDirectorKindCodexGoalV0,
		Status:          orquestagoal.GoalStatusRunningV0,
		EvidenceRefs:    []string{"evidence-ref-state-file-goal-marker-001"},
	}
	if err := store.SaveGoalWorkRunMarkerV0(context.Background(), marker); err != nil {
		t.Fatalf("SaveGoalWorkRunMarkerV0: %v", err)
	}
	recovered, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatalf("NewStoreV0 recovered: %v", err)
	}
	got, err := recovered.LoadGoalWorkRunMarkerV0(context.Background(), marker.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkRunMarkerV0: %v", err)
	}
	if got.SchemaVersion != orquestagoal.GoalWorkRunMarkerSchemaV0 ||
		got.RunRef != marker.RunRef ||
		got.GoalRef != marker.GoalRef ||
		got.DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 ||
		len(got.EvidenceRefs) != 1 ||
		got.EvidenceRefs[0] != "evidence-ref-state-file-goal-marker-001" {
		t.Fatalf("got=%+v", got)
	}
}

func TestStoreV0AppDirectorGoalFirstRunMarkerListaActivosTrasRecreate(t *testing.T) {
	root := t.TempDir()
	store, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	running := appDirectorGoalRunMarkerWithRefsForTestV0(
		"run-ref-state-file-marker-active-001",
		"goal-ref-state-file-marker-active-001",
		orquestagoal.GoalStatusRunningV0,
	)
	complete := appDirectorGoalRunMarkerWithRefsForTestV0(
		"run-ref-state-file-marker-complete-001",
		"goal-ref-state-file-marker-complete-001",
		orquestagoal.GoalStatusCompleteV0,
	)
	blocked := appDirectorGoalRunMarkerWithRefsForTestV0(
		"run-ref-state-file-marker-blocked-001",
		"goal-ref-state-file-marker-blocked-001",
		orquestagoal.GoalStatusBlockedV0,
	)
	for _, marker := range []orquestagoal.GoalWorkRunMarkerV0{complete, running, blocked} {
		if err := store.SaveGoalWorkRunMarkerV0(context.Background(), marker); err != nil {
			t.Fatalf("SaveGoalWorkRunMarkerV0: %v", err)
		}
	}
	recovered, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatalf("NewStoreV0 recovered: %v", err)
	}
	active, err := recovered.ListGoalWorkRunMarkersV0(context.Background(), orquestagoal.GoalWorkRunMarkerListRequestV0{
		ActiveOnly: true,
	})
	if err != nil {
		t.Fatalf("ListGoalWorkRunMarkersV0 active: %v", err)
	}
	if len(active) != 1 || active[0].RunRef != running.RunRef {
		t.Fatalf("active=%+v", active)
	}
	terminal, err := recovered.ListGoalWorkRunMarkersV0(context.Background(), orquestagoal.GoalWorkRunMarkerListRequestV0{
		RunRefs:  []string{blocked.RunRef, complete.RunRef, "run-ref-state-file-marker-missing"},
		Statuses: []string{orquestagoal.GoalStatusBlockedV0, orquestagoal.GoalStatusCompleteV0},
		MaxItems: 1,
	})
	if err != nil {
		t.Fatalf("ListGoalWorkRunMarkersV0 terminal: %v", err)
	}
	if len(terminal) != 1 || terminal[0].RunRef != blocked.RunRef {
		t.Fatalf("terminal=%+v", terminal)
	}
}

func TestStoreV0AppDirectorGoalStateListaActivosTrasRecreate(t *testing.T) {
	root := t.TempDir()
	store, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	running := appDirectorGoalStateWithRefsForTestV0(
		"run-ref-state-file-active-001",
		"goal-ref-state-file-active-001",
		orquestagoal.GoalStatusRunningV0,
	)
	complete := appDirectorGoalStateWithRefsForTestV0(
		"run-ref-state-file-complete-001",
		"goal-ref-state-file-complete-001",
		orquestagoal.GoalStatusCompleteV0,
	)
	completeAccepted := appDirectorGoalStateWithRefsForTestV0(
		"run-ref-state-file-complete-accepted-001",
		"goal-ref-state-file-complete-accepted-001",
		orquestagoal.GoalStatusCompleteV0,
	)
	completeAccepted.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:   orquestagoal.GoalStatusAcceptedV0,
		Accepted: true,
	}
	blocked := appDirectorGoalStateWithRefsForTestV0(
		"run-ref-state-file-blocked-001",
		"goal-ref-state-file-blocked-001",
		orquestagoal.GoalStatusBlockedV0,
	)
	for _, state := range []orquestagoal.GoalWorkStateV0{complete, completeAccepted, running, blocked} {
		if err := store.SaveGoalWorkStateV0(context.Background(), state); err != nil {
			t.Fatalf("SaveGoalWorkStateV0: %v", err)
		}
	}
	recovered, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatalf("NewStoreV0 recovered: %v", err)
	}
	active, err := recovered.ListGoalWorkStatesV0(context.Background(), orquestagoal.GoalWorkStateListRequestV0{
		ActiveOnly: true,
	})
	if err != nil {
		t.Fatalf("ListGoalWorkStatesV0 active: %v", err)
	}
	if len(active) != 2 || active[0].RunRef != running.RunRef || active[1].RunRef != complete.RunRef {
		t.Fatalf("active=%+v", active)
	}
	terminal, err := recovered.ListGoalWorkStatesV0(context.Background(), orquestagoal.GoalWorkStateListRequestV0{
		RunRefs:  []string{blocked.RunRef, complete.RunRef, "run-ref-state-file-missing"},
		Statuses: []string{orquestagoal.GoalStatusBlockedV0, orquestagoal.GoalStatusCompleteV0},
		MaxItems: 1,
	})
	if err != nil {
		t.Fatalf("ListGoalWorkStatesV0 terminal: %v", err)
	}
	if len(terminal) != 1 || terminal[0].RunRef != blocked.RunRef {
		t.Fatalf("terminal=%+v", terminal)
	}
}

func appDirectorGoalRunMarkerWithRefsForTestV0(
	runRef string,
	goalRef string,
	status string,
) orquestagoal.GoalWorkRunMarkerV0 {
	return orquestagoal.GoalWorkRunMarkerV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: "thread-ref-state-file-marker-001",
		DirectorKind:    orquestagoal.GoalDirectorKindCodexGoalV0,
		Status:          status,
		EvidenceRefs:    []string{"evidence-ref-state-file-goal-marker-list"},
	}
}

func appDirectorGoalStateForTestV0() orquestagoal.GoalWorkStateV0 {
	return appDirectorGoalStateWithRefsForTestV0(
		"run-ref-state-file-app-001",
		"goal-ref-state-file-app-001",
		orquestagoal.GoalStatusRunningV0,
	)
}

func appDirectorGoalStateWithRefsForTestV0(
	runRef string,
	goalRef string,
	status string,
) orquestagoal.GoalWorkStateV0 {
	spec := orquestagoal.NormalizeGoalWorkSpecV0(orquestagoal.GoalWorkSpecV0{
		GoalRef:      goalRef,
		RunRef:       runRef,
		Objective:    "Construir app desde goal",
		DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
		WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "generated-apps/state-file-app"}},
		ArtifactContracts: []orquestagoal.GoalArtifactContractV0{
			{ArtifactRef: "artifact-ref-state-file-source", ArtifactType: "source_tree", Required: true},
		},
	})
	result := orquestagoal.NormalizeGoalWorkResultV0(orquestagoal.GoalWorkResultV0{
		Status:          status,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: "thread-ref-state-file-app-001",
		EvidenceRefs:    []string{"evidence-ref-state-file-observed"},
	})
	return orquestagoal.GoalWorkStateV0{
		SchemaVersion:   orquestagoal.GoalWorkStateSchemaV0,
		RunRef:          spec.RunRef,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: "thread-ref-state-file-app-001",
		Status:          status,
		Spec:            spec,
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:          status,
			GoalRef:         spec.GoalRef,
			ExternalGoalRef: "thread-ref-state-file-app-001",
		},
		LastResult:   &result,
		EvidenceRefs: []string{"evidence-ref-state-file-launched"},
	}
}
