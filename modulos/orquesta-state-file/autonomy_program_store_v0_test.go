package orquestastatefile

import (
	"context"
	"errors"
	"reflect"
	"testing"

	orquestaautonomyprogram "orquesta/modulos/orquesta-autonomy-program"
)

func TestStoreV0RecoversAutonomyProgramAfterRestartWithStableRefsV0(t *testing.T) {
	ctx, rootDir := context.Background(), t.TempDir()
	store := mustStoreV0(t, rootDir)
	program, err := orquestaautonomyprogram.NewAutonomyProgramV0(orquestaautonomyprogram.AutonomyProgramV0{ProgramRef: "program-autonomy-001", ProjectRef: "project-autonomy-001", RootRef: "root-autonomy-001", Nodes: []orquestaautonomyprogram.AutonomyProgramNodeV0{{NodeRef: "node-a", WriteSet: []string{"domain"}, RequiredTests: []string{"test-a"}}, {NodeRef: "node-b", DependsOn: []string{"node-a"}, WriteSet: []string{"adapter"}, RequiredTests: []string{"test-b"}}}})
	if err != nil {
		t.Fatal(err)
	}
	frontier, err := orquestaautonomyprogram.PrepareAutonomyProgramFrontierV0(program)
	if err != nil {
		t.Fatal(err)
	}
	afterFirstClosure, err := orquestaautonomyprogram.CloseAutonomyProgramNodeV0(frontier.Program, orquestaautonomyprogram.AutonomyNodeClosureV0{
		NodeRef:    "node-a",
		Decision:   orquestaautonomyprogram.AutonomyNodeAcceptedV0,
		ReceiptRef: "receipt-node-a",
		CausalRefs: []string{"review-node-a", frontier.Launches[0].LaunchRef},
		RequiredTestEvidence: []orquestaautonomyprogram.RequiredTestProofV0{{
			TestRef: "test-a", Status: "passed", EvidenceRef: "evidence-test-a", CausalRef: "review-node-a",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveAutonomyProgramV0(ctx, afterFirstClosure); err != nil {
		t.Fatal(err)
	}
	recoveredStore := mustStoreV0(t, rootDir)
	recovered, err := recoveredStore.LoadAutonomyProgramV0(ctx, program.ProjectRef, program.RootRef, program.ProgramRef)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(recovered, afterFirstClosure) {
		t.Fatalf("recovered=%+v want=%+v", recovered, afterFirstClosure)
	}
	replay, err := orquestaautonomyprogram.PrepareAutonomyProgramFrontierV0(recovered)
	if err != nil {
		t.Fatal(err)
	}
	if len(replay.Launches) != 1 || replay.Launches[0].NodeRef != "node-b" {
		t.Fatalf("restart frontier=%+v", replay.Launches)
	}
	_, err = recoveredStore.LoadAutonomyProgramV0(ctx, "other-project", program.RootRef, program.ProgramRef)
	if err == nil {
		t.Fatal("expected out-of-scope program miss")
	}
	changedTopology := afterFirstClosure
	changedTopology.Nodes[1].WriteSet = []string{"other"}
	if err := recoveredStore.SaveAutonomyProgramV0(ctx, changedTopology); err == nil {
		t.Fatal("expected immutable DAG conflict")
	}
}

func TestStoreV0AutonomyProgramCASClaimsFrontierOnceAcrossStoreInstancesV0(t *testing.T) {
	ctx, rootDir := context.Background(), t.TempDir()
	firstStore, secondStore := mustStoreV0(t, rootDir), mustStoreV0(t, rootDir)
	program, err := orquestaautonomyprogram.NewAutonomyProgramV0(orquestaautonomyprogram.AutonomyProgramV0{
		ProgramRef: "program-cas", ProjectRef: "project-cas", RootRef: "root-cas",
		Nodes: []orquestaautonomyprogram.AutonomyProgramNodeV0{{NodeRef: "node", WriteSet: []string{"domain"}, RequiredTests: []string{"test"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := firstStore.SaveAutonomyProgramV0(ctx, program); err != nil {
		t.Fatal(err)
	}

	type result struct {
		frontier orquestaautonomyprogram.AutonomyProgramFrontierV0
		err      error
	}
	start, results := make(chan struct{}), make(chan result, 2)
	claim := func(store *StoreV0) {
		<-start
		frontier, claimErr := orquestaautonomyprogram.ClaimAutonomyProgramFrontierV0(ctx, store, program.ProjectRef, program.RootRef, program.ProgramRef)
		results <- result{frontier: frontier, err: claimErr}
	}
	go claim(firstStore)
	go claim(secondStore)
	close(start)
	launches := 0
	for range 2 {
		result := <-results
		if result.err != nil && !errors.Is(result.err, orquestaautonomyprogram.ErrAutonomyProgramCASConflictV0) {
			t.Fatal(result.err)
		}
		launches += len(result.frontier.Launches)
	}
	if launches != 1 {
		t.Fatalf("claimed launches=%d want 1", launches)
	}
	recovered, err := firstStore.LoadAutonomyProgramV0(ctx, program.ProjectRef, program.RootRef, program.ProgramRef)
	if err != nil || recovered.Nodes[0].Status != orquestaautonomyprogram.AutonomyNodeLaunchedV0 {
		t.Fatalf("recovered=%+v err=%v", recovered, err)
	}
}

func TestStoreV0AutonomyProgramRejectsBlindOverwriteStaleCASAndRollbackV0(t *testing.T) {
	ctx, rootDir := context.Background(), t.TempDir()
	store := mustStoreV0(t, rootDir)
	program, err := orquestaautonomyprogram.NewAutonomyProgramV0(orquestaautonomyprogram.AutonomyProgramV0{
		ProgramRef: "program-monotonic", ProjectRef: "project-monotonic", RootRef: "root-monotonic",
		Nodes: []orquestaautonomyprogram.AutonomyProgramNodeV0{{NodeRef: "node", WriteSet: []string{"domain"}, RequiredTests: []string{"test"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveAutonomyProgramV0(ctx, program); err != nil {
		t.Fatal(err)
	}
	frontier, err := orquestaautonomyprogram.PrepareAutonomyProgramFrontierV0(program)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveAutonomyProgramV0(ctx, frontier.Program); err == nil {
		t.Fatal("blind overwrite accepted")
	}
	swapped, err := store.CompareAndSwapAutonomyProgramV0(ctx, program, frontier.Program)
	if err != nil || !swapped {
		t.Fatalf("swapped=%v err=%v", swapped, err)
	}
	swapped, err = store.CompareAndSwapAutonomyProgramV0(ctx, program, frontier.Program)
	if err != nil || swapped {
		t.Fatalf("stale CAS swapped=%v err=%v", swapped, err)
	}
	if _, err := store.CompareAndSwapAutonomyProgramV0(ctx, frontier.Program, program); err == nil {
		t.Fatal("rollback CAS accepted")
	}
}

func TestStoreV0RecoversDurableOperatorTaskAndResumesScopedNodeV0(t *testing.T) {
	ctx, rootDir := context.Background(), t.TempDir()
	store := mustStoreV0(t, rootDir)
	program, err := orquestaautonomyprogram.NewAutonomyProgramV0(orquestaautonomyprogram.AutonomyProgramV0{
		ProgramRef: "program-operator-001", ProjectRef: "project-operator-001", RootRef: "root-operator-001",
		Nodes: []orquestaautonomyprogram.AutonomyProgramNodeV0{{NodeRef: "node-operator", WriteSet: []string{"domain"}, RequiredTests: []string{"test-operator"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	frontier, err := orquestaautonomyprogram.PrepareAutonomyProgramFrontierV0(program)
	if err != nil {
		t.Fatal(err)
	}
	waiting, err := orquestaautonomyprogram.PutAutonomyProgramNodeWaitExternalV0(frontier.Program, orquestaautonomyprogram.OperatorTaskV0{
		OperatorTaskRef: "operator-task-001", ProgramRef: program.ProgramRef, ProjectRef: program.ProjectRef, RootRef: program.RootRef,
		NodeRef: "node-operator", Action: "verify", CausalRefs: []string{frontier.Launches[0].LaunchRef},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveAutonomyProgramV0(ctx, waiting); err != nil {
		t.Fatal(err)
	}
	recoveredStore := mustStoreV0(t, rootDir)
	recovered, err := recoveredStore.LoadAutonomyProgramV0(ctx, program.ProjectRef, program.RootRef, program.ProgramRef)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Status != orquestaautonomyprogram.AutonomyProgramWaitExternalV0 || len(recovered.OperatorTasks) != 1 || recovered.OperatorTasks[0].OperatorTaskRef != "operator-task-001" || recovered.OperatorTasks[0].Status != "open" {
		t.Fatalf("recovered=%+v", recovered)
	}
	resumed, err := orquestaautonomyprogram.ResumeAutonomyProgramNodeV0(recovered, orquestaautonomyprogram.OperatorReceiptV0{
		ReceiptRef: "operator-receipt-001", OperatorTaskRef: "operator-task-001", ProgramRef: program.ProgramRef, ProjectRef: program.ProjectRef,
		RootRef: program.RootRef, NodeRef: "node-operator", Decision: "verified", CausalRef: "operator-task-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	frontier, err = orquestaautonomyprogram.PrepareAutonomyProgramFrontierV0(resumed)
	if err != nil || len(frontier.Launches) != 0 || resumed.Nodes[0].Status != orquestaautonomyprogram.AutonomyNodeLaunchedV0 {
		t.Fatalf("frontier=%+v err=%v", frontier, err)
	}
}
