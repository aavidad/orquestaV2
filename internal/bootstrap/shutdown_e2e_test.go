package bootstrap

import (
	"context"
	"errors"
	"syscall"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/config"
)

func TestShutdownTerminatesAndReapsOwnedAgentProcess(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	var agent *processAgent
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath,
		Version:    "test",
		AgentFactory: func(_ config.Snapshot, clock application.Clock) (AgentAdapter, error) {
			agent = newProcessAgent(clock)
			return agent, nil
		},
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	_ = submitTestGoal(t, runtime, "request:shutdown")
	var pid int
	select {
	case pid = <-agent.started:
	case <-time.After(5 * time.Second):
		t.Fatal("owned process did not start")
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := runtime.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if activeErr := syscall.Kill(pid, 0); !errors.Is(activeErr, syscall.ESRCH) {
		t.Fatalf("owned pid %d still exists: %v", pid, activeErr)
	}
	if err := runtime.Wait(); err != nil {
		t.Fatalf("server wait after shutdown: %v", err)
	}
}
