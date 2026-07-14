package bootstrap

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/goal"
)

func TestRestartReplaysTerminalStateWithoutDuplicateExecutionOrEvidence(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	var launches atomic.Int64

	first, err := Build(context.Background(), Options{
		ConfigPath: configPath, Version: "test", AgentFactory: countingFactory(&launches),
	})
	if err != nil {
		t.Fatalf("build first: %v", err)
	}
	if err := first.Start(context.Background()); err != nil {
		t.Fatalf("start first: %v", err)
	}
	goalRef := submitTestGoal(t, first, "request:restart")
	terminal := waitTerminalGoal(t, first, goalRef)
	if terminal.Goal.State() != goal.GoalStateSucceeded || launches.Load() != 1 ||
		len(terminal.Artifacts) != 1 || len(terminal.Attestations) != 1 {
		t.Fatalf("first closure: state=%s launches=%d artifacts=%d attestations=%d", terminal.Goal.State(), launches.Load(), len(terminal.Artifacts), len(terminal.Attestations))
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := first.Shutdown(shutdownCtx); err != nil {
		cancel()
		t.Fatalf("shutdown first: %v", err)
	}
	cancel()

	second, err := Build(context.Background(), Options{
		ConfigPath: configPath, Version: "test", AgentFactory: countingFactory(&launches),
	})
	if err != nil {
		t.Fatalf("build second: %v", err)
	}
	if err := second.Start(context.Background()); err != nil {
		t.Fatalf("start second: %v", err)
	}
	t.Cleanup(func() {
		ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_ = second.Shutdown(ctx)
	})
	time.Sleep(100 * time.Millisecond)
	replayed := waitTerminalGoal(t, second, goalRef)
	if launches.Load() != 1 || len(replayed.Artifacts) != 1 || len(replayed.Attestations) != 1 {
		t.Fatalf("restart duplicated work: launches=%d artifacts=%d attestations=%d", launches.Load(), len(replayed.Artifacts), len(replayed.Attestations))
	}
}
