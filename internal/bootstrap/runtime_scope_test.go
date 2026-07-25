package bootstrap

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/adapters/agent/codex"
	"orquesta/internal/application"
	"orquesta/internal/config"
	"orquesta/internal/ports"
)

func TestBindAgentRuntimeScopeUsesStateIdentitySource(t *testing.T) {
	agent := newRuntimeScopeCaptureAgent()
	state := &runtimeIdentitySource{identity: "local-state:sha256:active", supported: true}
	if err := bindAgentRuntimeScope(context.Background(), agent, state); err != nil {
		t.Fatalf("bind runtime scope: %v", err)
	}
	want := runtimeScopeForLocalStateIdentity(state.identity)
	if state.calls != 1 || agent.scope != want || agent.scope == state.identity {
		t.Fatalf("binding calls=%d scope=%q want=%q", state.calls, agent.scope, want)
	}
}

func TestBindAgentRuntimeScopeLeavesUnsupportedPlatformDisabled(t *testing.T) {
	agent := newRuntimeScopeCaptureAgent()
	state := &runtimeIdentitySource{supported: false}
	if err := bindAgentRuntimeScope(context.Background(), agent, state); err != nil {
		t.Fatalf("unsupported identity: %v", err)
	}
	if state.calls != 1 || agent.scope != "" {
		t.Fatalf("unsupported binding calls=%d scope=%q", state.calls, agent.scope)
	}
}

func TestBuildBindsAgentToOpenedRepositoryIdentity(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	var captured *runtimeScopeCaptureAgent
	var launches atomic.Int64
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath,
		AgentFactory: func(_ config.Snapshot, clock application.Clock) (AgentAdapter, error) {
			captured = &runtimeScopeCaptureAgent{AgentAdapter: newCountingAgent(clock, &launches)}
			return captured, nil
		},
	})
	if err != nil {
		t.Fatalf("build composition: %v", err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	identity, supported, err := runtime.repository.LocalStateIdentity()
	if err != nil {
		t.Fatalf("opened repository identity: %v", err)
	}
	if captured == nil {
		t.Fatal("agent factory result was not retained")
	}
	if !supported {
		if captured.scope != "" {
			t.Fatalf("unsupported repository identity bound scope %q", captured.scope)
		}
		return
	}
	want := runtimeScopeForLocalStateIdentity(identity)
	if captured.scope != want {
		t.Fatalf("production binding scope=%q want=%q identity=%q", captured.scope, want, identity)
	}
}

func TestBindCodexAgentAuthorityBindsSessionBeforeRuntimeDiscovery(t *testing.T) {
	agent := &orderedRuntimeScopeAgent{
		AgentAdapter: newCountingAgent(fixedBootstrapClock{}, &atomic.Int64{}),
	}
	state := &runtimeIdentitySource{identity: "local-state:sha256:restart", supported: true}
	if err := bindCodexAgentAuthority(context.Background(), agent, state, &sessionResolverStub{}); err != nil {
		t.Fatalf("bind Codex authority: %v", err)
	}
	if len(agent.bindings) != 2 || agent.bindings[0] != "session" || agent.bindings[1] != "runtime" {
		t.Fatalf("authority binding order=%v", agent.bindings)
	}
}

type orderedRuntimeScopeAgent struct {
	AgentAdapter
	scope    string
	bindings []string
}

func (agent *orderedRuntimeScopeAgent) BindSessionResolver(codex.SessionResolver) error {
	agent.bindings = append(agent.bindings, "session")
	return nil
}

func (agent *orderedRuntimeScopeAgent) BindRuntimeScope(_ context.Context, scope string) error {
	agent.bindings = append(agent.bindings, "runtime")
	agent.scope = scope
	return nil
}

type sessionResolverStub struct{}

func (*sessionResolverStub) ResolveCodexSession(context.Context, ports.AgentLaunchRequest) (codex.Session, error) {
	return codex.Session{}, nil
}

func (*sessionResolverStub) RecoverCodexSession(context.Context, ports.AgentLaunchRequest) (codex.Session, error) {
	return codex.Session{}, nil
}

func TestCredentialAgentDelegatesRuntimeScopeAndController(t *testing.T) {
	root, configPath := credentialFactoryFixture(t)
	snapshot, err := loadConfigSnapshot(context.Background(), configPath)
	if err != nil {
		t.Fatalf("load config in %s: %v", root, err)
	}
	agent, err := productionAgentFactory(snapshot, fixedBootstrapClock{})
	if err != nil {
		t.Fatalf("production agent factory: %v", err)
	}
	t.Cleanup(func() { _ = agent.Shutdown(context.Background()) })
	binder, ok := agent.(runtimeScopeBinder)
	if !ok {
		t.Fatalf("credential wrapper does not expose runtime scope binding: %T", agent)
	}
	if err := binder.BindRuntimeScope(context.Background(), "runtime-scope:test-credential-wrapper"); runtime.GOOS == "linux" {
		if codex.ErrorCode(err) != codex.CodeCgroupRootRequired {
			t.Fatalf("bind without delegated cgroup = %v", err)
		}
	} else if err != nil {
		t.Fatalf("bind delegated runtime scope: %v", err)
	}
	controller, ok := agent.(application.AgentController)
	if !ok {
		t.Fatalf("credential wrapper does not expose controller: %T", agent)
	}
	if _, err := controller.ControlCapabilities(context.Background()); err != nil {
		t.Fatalf("delegated control capabilities: %v", err)
	}
}

type fixedBootstrapClock struct{}

func (fixedBootstrapClock) Now() time.Time {
	return time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
}

type runtimeIdentitySource struct {
	identity  string
	supported bool
	err       error
	calls     int
}

func (source *runtimeIdentitySource) LocalStateIdentity() (string, bool, error) {
	source.calls++
	return source.identity, source.supported, source.err
}

type runtimeScopeCaptureAgent struct {
	AgentAdapter
	scope string
}

func newRuntimeScopeCaptureAgent() *runtimeScopeCaptureAgent {
	return &runtimeScopeCaptureAgent{
		AgentAdapter: newCountingAgent(fixedBootstrapClock{}, &atomic.Int64{}),
	}
}

func (agent *runtimeScopeCaptureAgent) BindRuntimeScope(_ context.Context, scope string) error {
	agent.scope = scope
	return nil
}
