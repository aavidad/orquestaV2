package orquestaautoprogramming

import (
	"testing"

	orquestaautonomyprogram "orquesta/modulos/orquesta-autonomy-program"
)

func TestBuildAutoprogrammingProgrammableWorkV0CompilesParentAutonomyProgramV0(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{TaskRef: "task-domain", Area: "domain", WriteSet: []string{"internal/domain"}}, {TaskRef: "task-adapter", Area: "adapter", WriteSet: []string{"internal/adapter"}, DependsOn: []string{"task-domain"}}}
		request.WriteSet = []string{"internal/domain", "internal/adapter"}
	})
	result := BuildAutoprogrammingProgrammableWorkV0(request)
	if !result.Accepted {
		t.Fatalf("issues=%+v", result.Issues)
	}
	program := result.Work.AutonomyProgram
	if program.ProgramRef == "" || program.ProjectRef != request.ProjectRef || program.RootRef == "" || len(program.Nodes) != 2 {
		t.Fatalf("program=%+v", program)
	}
	if len(program.Nodes[1].DependsOn) != 1 || program.Nodes[1].DependsOn[0] != program.Nodes[0].NodeRef {
		t.Fatalf("nodes=%+v", program.Nodes)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0KeepsLiveWorkAsResolvableExternalDependencyV0(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.LiveWorks = []AutoprogrammingLiveWorkV0{{TaskRef: "live-task", Status: "running", WriteSet: append([]string(nil), request.WriteSet...)}}
	})
	result := BuildAutoprogrammingProgrammableWorkV0(request)
	if !result.Accepted {
		t.Fatalf("issues=%+v", result.Issues)
	}
	program := result.Work.AutonomyProgram
	if len(program.Nodes) != 1 || len(program.Nodes[0].ExternalDependsOn) != 1 || program.Nodes[0].ExternalDependsOn[0] != "live-task" || program.Status != orquestaautonomyprogram.AutonomyProgramWaitExternalV0 {
		t.Fatalf("program=%+v", program)
	}
	receipt := orquestaautonomyprogram.ExternalDependencyReceiptV0{ReceiptRef: "live-receipt", ProgramRef: program.ProgramRef, ProjectRef: program.ProjectRef, RootRef: program.RootRef, NodeRef: program.Nodes[0].NodeRef, DependencyRef: "live-task", Decision: "satisfied", CausalRef: "live-task"}
	resolved, err := orquestaautonomyprogram.ResolveAutonomyProgramExternalDependencyV0(program, receipt)
	if err != nil {
		t.Fatal(err)
	}
	frontier, err := orquestaautonomyprogram.PrepareAutonomyProgramFrontierV0(resolved)
	if err != nil || len(frontier.Launches) != 1 {
		t.Fatalf("frontier=%+v err=%v", frontier, err)
	}
}
