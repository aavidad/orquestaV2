package codex

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

type workspacePathResolverFunc func(context.Context, ports.ExecutionWorkspaceRef) (string, error)

func (function workspacePathResolverFunc) ResolveExecutionWorkspace(ctx context.Context, ref ports.ExecutionWorkspaceRef) (string, error) {
	return function(ctx, ref)
}

func TestCodexCommandArgumentsForceNonInteractiveApproval(t *testing.T) {
	adapter := openTestAdapter(t, testConfig(t))
	session := &resolvedSession{endpoint: "http://127.0.0.1:7777/mcp"}
	for _, test := range []struct {
		name      string
		arguments []string
	}{
		{name: "unbound", arguments: adapter.commandArguments("run:unbound")},
		{name: "review", arguments: adapter.commandArguments("run:review", true, false)},
		{name: "writer", arguments: adapter.commandArguments("run:writer", true, true)},
		{name: "session", arguments: adapter.commandArgumentsWithSession(
			"run:session", false, false, governance.ReasoningEffortMedium, session,
		)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if len(test.arguments) < 3 ||
				test.arguments[0] != "--ask-for-approval" ||
				test.arguments[1] != "never" ||
				test.arguments[2] != "exec" {
				t.Fatalf("non-interactive prefix = %q", test.arguments)
			}
			seen := 0
			for _, argument := range test.arguments {
				if argument == "--ask-for-approval" {
					seen++
				}
			}
			if seen != 1 {
				t.Fatalf("approval policy occurrences = %d, want 1", seen)
			}
		})
	}
}

func TestBoundReviewerWorkspaceUsesReadOnlySandbox(t *testing.T) {
	adapter := &Adapter{config: testConfig(t)}
	arguments := adapter.commandArguments("run:review", true, false)
	for index := range arguments {
		if arguments[index] == "--sandbox" && index+1 < len(arguments) {
			if arguments[index+1] != "read-only" {
				t.Fatalf("review sandbox=%q", arguments[index+1])
			}
			return
		}
	}
	t.Fatal("sandbox argument missing")
}

func TestLaunchUsesExactOpaqueWorkspaceBinding(t *testing.T) {
	config := testConfig(t)
	workspace := filepath.Join(t.TempDir(), "exact-workspace")
	if err := os.Mkdir(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	bound, err := ports.NewExecutionWorkspaceRef("execution-workspace:exact-binding")
	if err != nil {
		t.Fatal(err)
	}
	var resolved ports.ExecutionWorkspaceRef
	config.WorkspacePathResolver = workspacePathResolverFunc(func(_ context.Context, ref ports.ExecutionWorkspaceRef) (string, error) {
		resolved = ref
		return workspace, nil
	})
	adapter := openTestAdapter(t, config)
	request := testRequest(t, "workspace-binding", "helper:success", 1024)
	request.ExecutionWorkspaceRef = bound

	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	observation := awaitTerminal(t, adapter, request.ExecutionRef)
	if observation.Status != ports.AgentCompleted || resolved != bound {
		t.Fatalf("workspace launch status=%q resolved=%q want=%q", observation.Status, resolved.String(), bound.String())
	}
	if _, err := os.Stat(filepath.Join(workspace, "helper-invocations")); err != nil {
		t.Fatalf("agent did not run inside exact workspace: %v", err)
	}
	runPath := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)), "helper-invocations")
	if _, err := os.Stat(runPath); !os.IsNotExist(err) {
		t.Fatalf("agent unexpectedly wrote in private control root: %v", err)
	}
}

func TestWorkspaceEvidenceLeaksNoPrivateAdapterData(t *testing.T) {
	config := testConfig(t)
	privateWorkspace := filepath.Join(t.TempDir(), "private-workspace-name-must-not-leak")
	if err := os.Mkdir(privateWorkspace, 0o700); err != nil {
		t.Fatal(err)
	}
	bound, err := ports.NewExecutionWorkspaceRef("execution-workspace:no-leak")
	if err != nil {
		t.Fatal(err)
	}
	config.WorkspacePathResolver = workspacePathResolverFunc(func(context.Context, ports.ExecutionWorkspaceRef) (string, error) {
		return privateWorkspace, nil
	})
	adapter := openTestAdapter(t, config)
	request := testRequest(t, "workspace-no-leak", "helper:success", 1024)
	request.ExecutionWorkspaceRef = bound
	receipt, err := adapter.Launch(context.Background(), request)
	if err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	_ = awaitTerminal(t, adapter, request.ExecutionRef)
	if strings.Contains(receipt.ExternalRef, privateWorkspace) || strings.Contains(receipt.ReceiptRef, privateWorkspace) {
		t.Fatalf("receipt leaks private workspace: %+v", receipt)
	}
	runDirectory := filepath.Join(config.WorkRoot, filepath.FromSlash(executionPath(request.ExecutionRef)))
	payload, err := os.ReadFile(filepath.Join(runDirectory, requestFileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), privateWorkspace) || !strings.Contains(string(payload), bound.String()) {
		t.Fatalf("journal path leak or opaque binding absent: %s", payload)
	}
}
