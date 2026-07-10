package orquestaautonomyprogram

import (
	"strings"
	"testing"
)

func TestAutonomyProgramRejectsInvalidDAGRefsAndCyclesV0(t *testing.T) {
	tests := []struct {
		name  string
		nodes []AutonomyProgramNodeV0
	}{
		{"missing", []AutonomyProgramNodeV0{{NodeRef: "a", DependsOn: []string{"missing"}, WriteSet: []string{"a"}, RequiredTests: []string{"test-a"}}}},
		{"duplicate dependency", []AutonomyProgramNodeV0{{NodeRef: "a", WriteSet: []string{"a"}, RequiredTests: []string{"test-a"}}, {NodeRef: "b", DependsOn: []string{"a", " a "}, WriteSet: []string{"b"}, RequiredTests: []string{"test-b"}}}},
		{"cycle", []AutonomyProgramNodeV0{{NodeRef: "a", DependsOn: []string{"b"}, WriteSet: []string{"a"}, RequiredTests: []string{"test-a"}}, {NodeRef: "b", DependsOn: []string{"a"}, WriteSet: []string{"b"}, RequiredTests: []string{"test-b"}}}},
		{"internal external collision", []AutonomyProgramNodeV0{{NodeRef: "a", WriteSet: []string{"a"}, RequiredTests: []string{"test-a"}}, {NodeRef: "b", DependsOn: []string{"a"}, ExternalDependsOn: []string{"a"}, WriteSet: []string{"b"}, RequiredTests: []string{"test-b"}}}},
		{"goal collision", []AutonomyProgramNodeV0{{NodeRef: "a", GoalRef: "goal", WriteSet: []string{"a"}, RequiredTests: []string{"test-a"}}, {NodeRef: "b", GoalRef: "goal", WriteSet: []string{"b"}, RequiredTests: []string{"test-b"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewAutonomyProgramV0(AutonomyProgramV0{ProgramRef: "program", ProjectRef: "project", RootRef: "root", Nodes: tt.nodes}); err == nil {
				t.Fatal("expected invalid DAG")
			}
		})
	}
}

func TestAutonomyProgramWriteSetNormalizesEquivalentSeparatorsButRejectsRootAndTraversalV0(t *testing.T) {
	program := autonomyProgramForTestV0(t, []AutonomyProgramNodeV0{{NodeRef: "a", WriteSet: []string{"dir//child/", `dir\child`, "dir/./child"}, RequiredTests: []string{"test-a"}}})
	if len(program.Nodes[0].WriteSet) != 1 || program.Nodes[0].WriteSet[0] != "dir/child" {
		t.Fatalf("write_set=%v", program.Nodes[0].WriteSet)
	}
	for _, unsafe := range []string{"/tmp/escape", `/`, `../escape`, `dir/../escape`, `C:\escape`, `C:relative`, `\\server\share`, `file:/tmp/escape`, `https:opaque`} {
		t.Run(strings.ReplaceAll(unsafe, "/", "_"), func(t *testing.T) {
			_, err := NewAutonomyProgramV0(AutonomyProgramV0{ProgramRef: "program", ProjectRef: "project", RootRef: "root", Nodes: []AutonomyProgramNodeV0{{NodeRef: "a", WriteSet: []string{unsafe}, RequiredTests: []string{"test-a"}}}})
			if err == nil {
				t.Fatalf("accepted unsafe path %q", unsafe)
			}
		})
	}
}

func TestAutonomyProgramDoesNotLaunchOverlapWithAlreadyActiveNodeV0(t *testing.T) {
	program := autonomyProgramForTestV0(t, []AutonomyProgramNodeV0{{NodeRef: "a", WriteSet: []string{"scripts"}, RequiredTests: []string{"test-a"}}, {NodeRef: "b", WriteSet: []string{"scripts/run.sh"}, RequiredTests: []string{"test-b"}}})
	frontier, err := PrepareAutonomyProgramFrontierV0(program)
	if err != nil {
		t.Fatal(err)
	}
	assertLaunchRefsV0(t, frontier.Launches, "a")
	replay, err := PrepareAutonomyProgramFrontierV0(frontier.Program)
	if err != nil {
		t.Fatal(err)
	}
	if len(replay.Launches) != 0 {
		t.Fatalf("false parallel frontier=%+v", replay.Launches)
	}
}

func TestAutonomyProgramPureTransitionsDoNotMutateInputSlicesV0(t *testing.T) {
	program := autonomyProgramForTestV0(t, []AutonomyProgramNodeV0{{NodeRef: "a", WriteSet: []string{"a"}, RequiredTests: []string{"test-a"}}})
	frontier, err := PrepareAutonomyProgramFrontierV0(program)
	if err != nil {
		t.Fatal(err)
	}
	if program.Nodes[0].Status != AutonomyNodePendingV0 || program.Nodes[0].LaunchRef != "" {
		t.Fatalf("input mutated=%+v", program.Nodes[0])
	}
	if frontier.Program.Nodes[0].Status != AutonomyNodeLaunchedV0 {
		t.Fatalf("frontier=%+v", frontier)
	}
}

func TestAutonomyProgramRecoveryReusesDurableLaunchAndResumeRefsV0(t *testing.T) {
	program := autonomyProgramForTestV0(t, []AutonomyProgramNodeV0{{NodeRef: "a", WriteSet: []string{"a"}, RequiredTests: []string{"test-a"}}})
	frontier, err := PrepareAutonomyProgramFrontierV0(program)
	if err != nil {
		t.Fatal(err)
	}
	recovery, err := RecoverAutonomyProgramActionsV0(frontier.Program)
	if err != nil || len(recovery.Launches) != 1 || recovery.Launches[0].LaunchRef != frontier.Launches[0].LaunchRef {
		t.Fatalf("recovery=%+v err=%v", recovery, err)
	}
	waiting, err := PutAutonomyProgramNodeWaitExternalV0(frontier.Program, OperatorTaskV0{OperatorTaskRef: "operator", ProgramRef: program.ProgramRef, ProjectRef: program.ProjectRef, RootRef: program.RootRef, NodeRef: "a", Action: "verify", CausalRefs: []string{frontier.Launches[0].LaunchRef}})
	if err != nil {
		t.Fatal(err)
	}
	receipt := OperatorReceiptV0{ReceiptRef: "operator-receipt", OperatorTaskRef: "operator", ProgramRef: program.ProgramRef, ProjectRef: program.ProjectRef, RootRef: program.RootRef, NodeRef: "a", Decision: "verified", CausalRef: "operator"}
	resumed, err := ResumeAutonomyProgramNodeV0(waiting, receipt)
	if err != nil {
		t.Fatal(err)
	}
	recovery, err = RecoverAutonomyProgramActionsV0(resumed)
	if err != nil || len(recovery.Launches) != 0 || len(recovery.Resumes) != 1 || recovery.Resumes[0].ReceiptRef != receipt.ReceiptRef {
		t.Fatalf("resume recovery=%+v err=%v", recovery, err)
	}
}

func TestAutonomyProgramPathPrefixRequiresSegmentBoundaryV0(t *testing.T) {
	program := autonomyProgramForTestV0(t, []AutonomyProgramNodeV0{{NodeRef: "a", WriteSet: []string{"scripts"}, RequiredTests: []string{"test-a"}}, {NodeRef: "b", WriteSet: []string{"scripts2/run.sh"}, RequiredTests: []string{"test-b"}}})
	frontier, err := PrepareAutonomyProgramFrontierV0(program)
	if err != nil {
		t.Fatal(err)
	}
	assertLaunchRefsV0(t, frontier.Launches, "a", "b")
}

func TestAutonomyProgramBlockedDependencyStopsAggregateV0(t *testing.T) {
	program := autonomyProgramForTestV0(t, []AutonomyProgramNodeV0{{NodeRef: "a", WriteSet: []string{"a"}, RequiredTests: []string{"test-a"}}, {NodeRef: "b", DependsOn: []string{"a"}, WriteSet: []string{"b"}, RequiredTests: []string{"test-b"}}, {NodeRef: "c", WriteSet: []string{"c"}, RequiredTests: []string{"test-c"}}})
	frontier, err := PrepareAutonomyProgramFrontierV0(program)
	if err != nil {
		t.Fatal(err)
	}
	program = closeNodeForTestV0(t, frontier.Program, "a", AutonomyNodeBlockedV0, "blocked-a")
	if program.Status != AutonomyProgramBlockedV0 {
		t.Fatalf("status=%s", program.Status)
	}
	frontier, err = PrepareAutonomyProgramFrontierV0(program)
	if err != nil || len(frontier.Launches) != 0 {
		t.Fatalf("frontier=%+v err=%v", frontier, err)
	}
	recovery, err := RecoverAutonomyProgramActionsV0(program)
	if err != nil || len(recovery.Launches) != 0 || len(recovery.Resumes) != 0 {
		t.Fatalf("blocked recovery=%+v err=%v", recovery, err)
	}
}

func TestAutonomyProgramClosureRequiresLaunchAndProofCausalityV0(t *testing.T) {
	program := autonomyProgramForTestV0(t, []AutonomyProgramNodeV0{{NodeRef: "a", WriteSet: []string{"a"}, RequiredTests: []string{"test-a"}}})
	frontier, err := PrepareAutonomyProgramFrontierV0(program)
	if err != nil {
		t.Fatal(err)
	}
	closure := closureForTestV0("a", AutonomyNodeAcceptedV0, "receipt", "test-a")
	if _, err := CloseAutonomyProgramNodeV0(frontier.Program, closure); err == nil {
		t.Fatal("accepted closure without launch causal ref")
	}
	closure.CausalRefs = append(closure.CausalRefs, frontier.Launches[0].LaunchRef)
	closure.RequiredTestEvidence[0].Status = "unknown"
	if _, err := CloseAutonomyProgramNodeV0(frontier.Program, closure); err == nil {
		t.Fatal("accepted invalid proof status")
	}
	closure.RequiredTestEvidence[0].Status = "passed"
	closure.RequiredTestEvidence[0].CausalRef = "outside-review"
	if _, err := CloseAutonomyProgramNodeV0(frontier.Program, closure); err == nil {
		t.Fatal("accepted proof outside closure causality")
	}
}

func TestAutonomyProgramExternalDependencyNeedsScopedDurableReceiptV0(t *testing.T) {
	program := autonomyProgramForTestV0(t, []AutonomyProgramNodeV0{{NodeRef: "a", ExternalDependsOn: []string{"live-work"}, WriteSet: []string{"a"}, RequiredTests: []string{"test-a"}}})
	if program.Status != AutonomyProgramWaitExternalV0 {
		t.Fatalf("status=%s", program.Status)
	}
	frontier, err := PrepareAutonomyProgramFrontierV0(program)
	if err != nil || len(frontier.Launches) != 0 {
		t.Fatalf("frontier=%+v err=%v", frontier, err)
	}
	receipt := ExternalDependencyReceiptV0{ReceiptRef: "external-receipt", ProgramRef: program.ProgramRef, ProjectRef: "wrong", RootRef: program.RootRef, NodeRef: "a", DependencyRef: "live-work", Decision: "satisfied", CausalRef: "live-work"}
	if _, err := ResolveAutonomyProgramExternalDependencyV0(program, receipt); err == nil {
		t.Fatal("accepted external receipt outside project")
	}
	receipt.ProjectRef = program.ProjectRef
	program, err = ResolveAutonomyProgramExternalDependencyV0(program, receipt)
	if err != nil {
		t.Fatal(err)
	}
	frontier, err = PrepareAutonomyProgramFrontierV0(program)
	if err != nil {
		t.Fatal(err)
	}
	assertLaunchRefsV0(t, frontier.Launches, "a")
}

func TestAutonomyProgramLaunchRefsIncludeFullScopeAndAttemptV0(t *testing.T) {
	first := autonomyProgramForTestV0(t, []AutonomyProgramNodeV0{{NodeRef: "a", WriteSet: []string{"a"}, RequiredTests: []string{"test-a"}}})
	second := first
	second.ProjectRef = "other-project"
	second, err := NewAutonomyProgramV0(second)
	if err != nil {
		t.Fatal(err)
	}
	firstFrontier, firstErr := PrepareAutonomyProgramFrontierV0(first)
	secondFrontier, secondErr := PrepareAutonomyProgramFrontierV0(second)
	if firstErr != nil || secondErr != nil || len(firstFrontier.Launches) != 1 || len(secondFrontier.Launches) != 1 {
		t.Fatalf("first=%+v err=%v second=%+v err=%v", firstFrontier, firstErr, secondFrontier, secondErr)
	}
	if firstFrontier.Launches[0].LaunchRef == secondFrontier.Launches[0].LaunchRef {
		t.Fatal("launch_ref collided across project scope")
	}
	reworked := closeNodeForTestV0(t, firstFrontier.Program, "a", AutonomyNodeReworkV0, "rework")
	reworkFrontier, err := PrepareAutonomyProgramFrontierV0(reworked)
	if err != nil {
		t.Fatal(err)
	}
	if reworkFrontier.Launches[0].LaunchRef == firstFrontier.Launches[0].LaunchRef {
		t.Fatal("rework reused launch_ref")
	}
}
