package bootstrap

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

func TestTwoGoalsCanReferenceOneContentAddressedArtifact(t *testing.T) {
	root := t.TempDir()
	var launches atomic.Int64
	runtime, err := Build(context.Background(), Options{
		ConfigPath: writeTestConfig(t, root), Version: "test-cas-sharing",
		AgentFactory: constantContentFactory(&launches, []byte("identical durable artifact")),
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = runtime.Shutdown(ctx)
	})

	actor, _ := goal.NewActorRef("actor:local-owner")
	project, _ := goal.NewProjectRef("project:default")
	refs := make([]goal.GoalRef, 0, 2)
	for index, statement := range []string{"first objective", "second objective"} {
		result, submitErr := runtime.Orchestrator().Submit(context.Background(), application.SubmitRequest{
			RequestRef: "request:shared-cas:" + string(rune('a'+index)),
			ActorRef:   actor, ProjectRef: project, Statement: statement, Confirm: true,
		})
		if submitErr != nil {
			t.Fatalf("submit %d: %v", index, submitErr)
		}
		refs = append(refs, result.Record.Goal.Ref())
	}

	first := waitTerminalGoal(t, runtime, refs[0])
	second := waitTerminalGoal(t, runtime, refs[1])
	if first.Goal.State() != goal.GoalStateSucceeded || second.Goal.State() != goal.GoalStateSucceeded {
		t.Fatalf("terminal states = %s/%s", first.Goal.State(), second.Goal.State())
	}
	if len(first.Artifacts) != 1 || len(second.Artifacts) != 1 ||
		first.Artifacts[0].Stored.Ref != second.Artifacts[0].Stored.Ref {
		t.Fatalf("content-addressed refs were not shared: first=%+v second=%+v", first.Artifacts, second.Artifacts)
	}
	if launches.Load() != 2 {
		t.Fatalf("launches = %d, want 2 independent executions", launches.Load())
	}
}
