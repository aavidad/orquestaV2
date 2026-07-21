package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/adapters/agent/codex"
	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type v16E2EFixture struct {
	GitFixture struct {
		MinimumVersion string           `json:"minimum_version"`
		ObjectFormat   string           `json:"object_format"`
		TargetRef      string           `json:"target_ref"`
		AuthorName     string           `json:"author_name"`
		AuthorEmail    string           `json:"author_email"`
		BaseOID        string           `json:"base_oid"`
		BaseTreeOID    string           `json:"base_tree_oid"`
		TargetOID      string           `json:"target_oid"`
		TargetTreeOID  string           `json:"target_tree_oid"`
		BaseUnixTime   int64            `json:"base_unix_time"`
		TargetUnixTime int64            `json:"target_unix_time"`
		BaseFiles      []v16FixtureFile `json:"base_files"`
		TargetChanges  []v16FixtureFile `json:"target_changes"`
	} `json:"git_fixture"`
	Actors []struct {
		PrincipalRef string `json:"principal_ref"`
		ActorRef     string `json:"actor_ref"`
	} `json:"actors"`
	Changes []struct {
		Name         string           `json:"name"`
		ExecutionRef string           `json:"execution_ref"`
		WriteSet     []string         `json:"write_set"`
		Writes       []v16FixtureFile `json:"writes"`
		Expected     string           `json:"expected"`
	} `json:"changes"`
	CrashFrontiers     []string `json:"crash_frontiers"`
	PrivateLeakMarkers []string `json:"private_leak_markers"`
}

type v16FixtureFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type v16Write map[string]string

type v16WorkspaceAgent struct {
	now      func() time.Time
	writes   map[string]v16Write
	launches *atomic.Int64

	mu       sync.Mutex
	resolver codex.WorkspacePathResolver
	requests map[goal.ExecutionRef]ports.AgentLaunchRequest
	closed   bool
}

func (agent *v16WorkspaceAgent) BindWorkspacePathResolver(resolver codex.WorkspacePathResolver) error {
	if resolver == nil {
		return errors.New("v16_test.workspace_resolver_required")
	}
	agent.mu.Lock()
	defer agent.mu.Unlock()
	if agent.resolver != nil {
		return errors.New("v16_test.workspace_resolver_rebound")
	}
	agent.resolver = resolver
	return nil
}

func (agent *v16WorkspaceAgent) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return v16AgentCapabilities(), nil
}

func (agent *v16WorkspaceAgent) Launch(
	ctx context.Context,
	request ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	agent.mu.Lock()
	if agent.closed || agent.resolver == nil {
		agent.mu.Unlock()
		return ports.AgentLaunchReceipt{}, errors.New("v16_test.agent_unavailable")
	}
	if previous, found := agent.requests[request.ExecutionRef]; found {
		agent.mu.Unlock()
		if previous.IdempotencyKey != request.IdempotencyKey ||
			previous.ExecutionWorkspaceRef != request.ExecutionWorkspaceRef {
			return ports.AgentLaunchReceipt{}, errors.New("v16_test.launch_conflict")
		}
		return agent.launchReceipt(request), nil
	}
	resolver := agent.resolver
	writes := agent.writes[request.Objective]
	agent.mu.Unlock()
	if len(writes) == 0 {
		return ports.AgentLaunchReceipt{}, errors.New("v16_test.agent_script_missing")
	}
	workspace, err := resolver.ResolveExecutionWorkspace(ctx, request.ExecutionWorkspaceRef)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	for relative, content := range writes {
		target := filepath.Join(workspace, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return ports.AgentLaunchReceipt{}, err
		}
		if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
			return ports.AgentLaunchReceipt{}, err
		}
	}
	agent.mu.Lock()
	agent.requests[request.ExecutionRef] = request
	agent.launches.Add(1)
	agent.mu.Unlock()
	return agent.launchReceipt(request), nil
}

func (agent *v16WorkspaceAgent) launchReceipt(request ports.AgentLaunchRequest) ports.AgentLaunchReceipt {
	return ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: "provider:v16-fake-agent", ModelRef: "model:v16-fake-agent",
		AgentRef: "agent:v16-fake-agent", ExternalRef: "execution:v16-fake:" + request.ExecutionRef.String(),
		IdempotencyKey: request.IdempotencyKey, AcceptedAt: agent.now(),
		ReceiptRef: "receipt:v16-launch:" + request.ExecutionRef.String(),
	}
}

func (agent *v16WorkspaceAgent) Observe(
	ctx context.Context,
	execution goal.ExecutionRef,
) (ports.AgentObservation, error) {
	if err := ctx.Err(); err != nil {
		return ports.AgentObservation{}, err
	}
	agent.mu.Lock()
	request, found := agent.requests[execution]
	closed := agent.closed
	agent.mu.Unlock()
	if closed || !found {
		return ports.AgentObservation{}, errors.New("v16_test.observation_not_found")
	}
	return ports.AgentObservation{
		ExecutionRef: execution, SpecHash: request.SpecHash, Status: ports.AgentCompleted,
		MediaType: "text/plain", Content: []byte("v16 fake agent wrote the bound workspace"),
		Usage: unknownTestUsage(), ObservedAt: agent.now(),
	}, nil
}

func (agent *v16WorkspaceAgent) Shutdown(context.Context) error {
	agent.mu.Lock()
	agent.closed = true
	agent.mu.Unlock()
	return nil
}

func v16AgentCapabilities() ports.AgentCapabilities {
	return ports.AgentCapabilities{
		ProviderRef: "provider:v16-fake-agent", ModelRef: "model:v16-fake-agent",
		AgentRef: "agent:v16-fake-agent", Unrestricted: true,
	}
}

type v16Harness struct {
	t             *testing.T
	fixture       v16E2EFixture
	root          string
	seed          string
	workspaceRoot string
	configPath    string
	git           string
	writes        map[string]v16Write
	launches      atomic.Int64
	runtime       *Runtime
	access        application.Access
}
