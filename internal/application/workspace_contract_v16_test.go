package application

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestReworkRequiresExplicitParentChangeRef(t *testing.T) {
	system := newDirectorTestSystem(t)
	lease := claimDirectorForTest(t, system, system.ownerAccess, "director-claim:v16-parent-change")
	current := directorCurrentRecord(t, system)
	source := current.Goal.WorkItems()[0]
	if len(current.Executions) != 1 {
		t.Fatalf("source executions=%d, want one", len(current.Executions))
	}
	execution := current.Executions[0]
	if execution.WorkItemRef != source.Ref() {
		t.Fatal("source execution missing")
	}
	request := directorReplanRequest(current, lease, goal.ReplanCauseSplitPending, source, execution, "director-plan:v16-parent-change")
	if _, err := system.orchestrator.ProposeDirectorPlan(context.Background(), system.ownerAccess, request); err != nil {
		t.Fatalf("create successor: %v", err)
	}

	replanned := directorCurrentRecord(t, system)
	successor := replanned.Goal.WorkItems()[1]
	if parent, linked := successor.ReworkOf(); !linked || parent != source.Ref() {
		t.Fatalf("successor ReworkOf=%s/%v, want %s/true", parent, linked, source.Ref())
	}
	if _, err := parentChangeRef(replanned, successor); err == nil || err.Error() != "application.parent_change_ref_missing" {
		t.Fatalf("missing parent ChangeSet error=%v", err)
	}

	parentRef, err := ports.NewChangeSetRef("change-set:v16-parent")
	if err != nil {
		t.Fatal(err)
	}
	replanned.ChangeSets = append(replanned.ChangeSets, ChangeSet{Ref: parentRef, WorkItemRef: source.Ref()})
	actual, err := parentChangeRef(replanned, successor)
	if err != nil || actual != parentRef {
		t.Fatalf("parent ChangeSet=%s err=%v, want %s", actual, err, parentRef)
	}
}

func TestWorkspaceArchitectureKeepsOneWriterStateOutboxScheduler(t *testing.T) {
	dependencies := reflect.TypeOf(Dependencies{})
	statePort := reflect.TypeOf((*StateRepository)(nil)).Elem()
	workspacePort := reflect.TypeOf((*WorkspaceManager)(nil)).Elem()
	versionControlPort := reflect.TypeOf((*VersionControl)(nil)).Elem()
	for _, requirement := range []struct {
		name string
		port reflect.Type
	}{{"StateRepository", statePort}, {"WorkspaceManager", workspacePort}, {"VersionControl", versionControlPort}} {
		count := 0
		for index := 0; index < dependencies.NumField(); index++ {
			if dependencies.Field(index).Type == requirement.port {
				count++
			}
		}
		if count != 1 {
			t.Errorf("Dependencies %s fields=%d, want one", requirement.name, count)
		}
	}

	orchestrator := reflect.TypeOf(Orchestrator{})
	for _, requirement := range []struct {
		name string
		port reflect.Type
	}{{"StateRepository", statePort}, {"WorkspaceManager", workspacePort}, {"VersionControl", versionControlPort}} {
		count := 0
		for index := 0; index < orchestrator.NumField(); index++ {
			if orchestrator.Field(index).Type == requirement.port {
				count++
			}
		}
		if count != 1 {
			t.Errorf("Orchestrator %s fields=%d, want one", requirement.name, count)
		}
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	forbidden := map[string]struct{}{
		"WorkspaceStore": {}, "WorkspaceDatabase": {}, "WorkspaceQueue": {}, "WorkspaceLoop": {}, "WorkspaceScheduler": {},
		"GitStore": {}, "GitDatabase": {}, "GitQueue": {}, "GitLoop": {}, "GitScheduler": {}, "MergeLifecycle": {},
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(token.NewFileSet(), entry.Name(), nil, 0)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", entry.Name(), parseErr)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			specification, ok := node.(*ast.TypeSpec)
			if !ok {
				return true
			}
			if _, blocked := forbidden[specification.Name.Name]; blocked {
				t.Errorf("parallel workspace/Git authority %s in %s", specification.Name.Name, entry.Name())
			}
			return true
		})
	}
}
