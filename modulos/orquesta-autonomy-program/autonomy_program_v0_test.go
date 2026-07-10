package orquestaautonomyprogram

import "testing"

func TestAutonomyProgramThreeNodeDAGLaunchesOnlyDisjointReadyFrontierV0(t *testing.T) {
	program := autonomyProgramForTestV0(t, []AutonomyProgramNodeV0{
		{NodeRef: "node-a", WriteSet: []string{"domain"}, RequiredTests: []string{"test-a"}},
		{NodeRef: "node-b", DependsOn: []string{"node-a"}, WriteSet: []string{"adapter"}, RequiredTests: []string{"test-b"}},
		{NodeRef: "node-c", WriteSet: []string{"docs"}, RequiredTests: []string{"test-c"}},
	})
	frontier, err := PrepareAutonomyProgramFrontierV0(program)
	if err != nil {
		t.Fatal(err)
	}
	assertLaunchRefsV0(t, frontier.Launches, "node-a", "node-c")
	frontier, err = PrepareAutonomyProgramFrontierV0(frontier.Program)
	if err != nil {
		t.Fatal(err)
	}
	if len(frontier.Launches) != 0 {
		t.Fatalf("replay launches=%+v", frontier.Launches)
	}
	program = closeNodeForTestV0(t, frontier.Program, "node-a", AutonomyNodeAcceptedV0, "receipt-a")
	program = closeNodeForTestV0(t, program, "node-c", AutonomyNodeAcceptedV0, "receipt-c")
	frontier, err = PrepareAutonomyProgramFrontierV0(program)
	if err != nil {
		t.Fatal(err)
	}
	assertLaunchRefsV0(t, frontier.Launches, "node-b")
}

func TestAutonomyProgramSerializesOverlappingWriteSetsV0(t *testing.T) {
	program := autonomyProgramForTestV0(t, []AutonomyProgramNodeV0{
		{NodeRef: "node-a", WriteSet: []string{"scripts"}, RequiredTests: []string{"test-a"}},
		{NodeRef: "node-b", WriteSet: []string{"scripts/smoke.sh"}, RequiredTests: []string{"test-b"}},
	})
	frontier, err := PrepareAutonomyProgramFrontierV0(program)
	if err != nil {
		t.Fatal(err)
	}
	assertLaunchRefsV0(t, frontier.Launches, "node-a")
	program = closeNodeForTestV0(t, frontier.Program, "node-a", AutonomyNodeAcceptedV0, "receipt-a")
	frontier, err = PrepareAutonomyProgramFrontierV0(program)
	if err != nil {
		t.Fatal(err)
	}
	assertLaunchRefsV0(t, frontier.Launches, "node-b")
}

func TestAutonomyProgramExactlyOneReworkAndCausalTestClosureV0(t *testing.T) {
	program := autonomyProgramForTestV0(t, []AutonomyProgramNodeV0{{NodeRef: "node-a", WriteSet: []string{"domain"}, RequiredTests: []string{"test-a"}}})
	frontier, err := PrepareAutonomyProgramFrontierV0(program)
	if err != nil {
		t.Fatal(err)
	}
	program = closeNodeForTestV0(t, frontier.Program, "node-a", AutonomyNodeReworkV0, "receipt-rework-1")
	frontier, err = PrepareAutonomyProgramFrontierV0(program)
	if err != nil {
		t.Fatal(err)
	}
	assertLaunchRefsV0(t, frontier.Launches, "node-a")
	_, err = CloseAutonomyProgramNodeV0(frontier.Program, closureForTestV0("node-a", AutonomyNodeReworkV0, "receipt-rework-2", "test-a"))
	if err == nil {
		t.Fatal("expected second rework rejection")
	}
	program = closeNodeForTestV0(t, frontier.Program, "node-a", AutonomyNodeAcceptedV0, "receipt-accepted")
	if program.Status != AutonomyProgramCompletedV0 {
		t.Fatalf("status=%s", program.Status)
	}
}

func TestAutonomyProgramWaitExternalResumeIsScopedAndNotTerminalV0(t *testing.T) {
	program := autonomyProgramForTestV0(t, []AutonomyProgramNodeV0{{NodeRef: "node-a", WriteSet: []string{"domain"}, RequiredTests: []string{"test-a"}}})
	frontier, err := PrepareAutonomyProgramFrontierV0(program)
	if err != nil {
		t.Fatal(err)
	}
	wait, err := PutAutonomyProgramNodeWaitExternalV0(frontier.Program, OperatorTaskV0{OperatorTaskRef: "operator-task-a", ProgramRef: frontier.Program.ProgramRef, ProjectRef: frontier.Program.ProjectRef, RootRef: frontier.Program.RootRef, NodeRef: "node-a", Action: "verify", CausalRefs: []string{frontier.Launches[0].LaunchRef}, Status: "open"})
	if err != nil {
		t.Fatal(err)
	}
	if wait.Status != AutonomyProgramWaitExternalV0 {
		t.Fatalf("status=%s", wait.Status)
	}
	_, err = ResumeAutonomyProgramNodeV0(wait, OperatorReceiptV0{ReceiptRef: "operator-receipt-wrong", OperatorTaskRef: "operator-task-a", ProgramRef: wait.ProgramRef, ProjectRef: "other-project", RootRef: wait.RootRef, NodeRef: "node-a", Decision: "verified", CausalRef: "operator-task-a"})
	if err == nil {
		t.Fatal("expected out-of-scope receipt rejection")
	}
	resumed, err := ResumeAutonomyProgramNodeV0(wait, OperatorReceiptV0{ReceiptRef: "operator-receipt-a", OperatorTaskRef: "operator-task-a", ProgramRef: wait.ProgramRef, ProjectRef: wait.ProjectRef, RootRef: wait.RootRef, NodeRef: "node-a", Decision: "verified", CausalRef: "operator-task-a"})
	if err != nil {
		t.Fatal(err)
	}
	frontier, err = PrepareAutonomyProgramFrontierV0(resumed)
	if err != nil {
		t.Fatal(err)
	}
	if len(frontier.Launches) != 0 || resumed.Nodes[0].Status != AutonomyNodeLaunchedV0 || resumed.Nodes[0].LaunchRef != wait.Nodes[0].LaunchRef {
		t.Fatalf("resume created a new launch: %+v", frontier)
	}
}

func autonomyProgramForTestV0(t *testing.T, nodes []AutonomyProgramNodeV0) AutonomyProgramV0 {
	t.Helper()
	program, err := NewAutonomyProgramV0(AutonomyProgramV0{ProgramRef: "program-001", ProjectRef: "project-001", RootRef: "root-001", Nodes: nodes})
	if err != nil {
		t.Fatal(err)
	}
	return program
}
func closureForTestV0(nodeRef string, decision AutonomyNodeStatusV0, receiptRef string, testRef string) AutonomyNodeClosureV0 {
	status := "passed"
	if decision == AutonomyNodeBlockedV0 {
		status = "failed"
	}
	return AutonomyNodeClosureV0{NodeRef: nodeRef, Decision: decision, ReceiptRef: receiptRef, CausalRefs: []string{"review:" + nodeRef}, RequiredTestEvidence: []RequiredTestProofV0{{TestRef: testRef, Status: status, EvidenceRef: "evidence:" + testRef, CausalRef: "review:" + nodeRef}}}
}
func closeNodeForTestV0(t *testing.T, program AutonomyProgramV0, nodeRef string, decision AutonomyNodeStatusV0, receiptRef string) AutonomyProgramV0 {
	t.Helper()
	node := autonomyProgramNodeIndexV0(program.Nodes)[nodeRef]
	closure := closureForTestV0(nodeRef, decision, receiptRef, node.RequiredTests[0])
	closure.CausalRefs = append(closure.CausalRefs, node.LaunchRef)
	closed, err := CloseAutonomyProgramNodeV0(program, closure)
	if err != nil {
		t.Fatal(err)
	}
	return closed
}
func assertLaunchRefsV0(t *testing.T, launches []AutonomyProgramLaunchV0, want ...string) {
	t.Helper()
	if len(launches) != len(want) {
		t.Fatalf("launches=%+v want=%v", launches, want)
	}
	for index, nodeRef := range want {
		if launches[index].NodeRef != nodeRef {
			t.Fatalf("launches=%+v want=%v", launches, want)
		}
	}
}
