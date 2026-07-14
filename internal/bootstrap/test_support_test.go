package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/config"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type bearerRoundTripper struct {
	token string
	base  http.RoundTripper
}

func (transport bearerRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	cloned := request.Clone(request.Context())
	cloned.Header = request.Header.Clone()
	cloned.Header.Set("Authorization", "Bearer "+transport.token)
	base := transport.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(cloned)
}

func authorizedHTTPClient(t *testing.T, tokenPath string) *http.Client {
	t.Helper()
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("read local auth token: %v", err)
	}
	return &http.Client{Transport: bearerRoundTripper{token: string(token)}}
}

func writeTestConfig(t *testing.T, root string) string {
	t.Helper()
	configPath := root + "/orquesta.toml"
	content := fmt.Sprintf(`[server]
listen = "127.0.0.1:0"
shutdown_timeout = "2s"

[state.sqlite]
path = %s
busy_timeout = "1s"
max_open_connections = 4

[artifact.filesystem]
root = %s

[runtime]
max_output_bytes = 65536

[runtime.codex]
timeout = "1s"
max_concurrent_executions = 4
work_root = %s

[identity]
local_token_path = %s

[scheduler]
poll_interval = "10ms"
observation_interval = "10ms"
claim_lease = "1s"
max_action_attempts = 100
execution_timeout = "10s"

[config]
effective_path = %s
`, strconv.Quote(root+"/state/orquesta.sqlite"), strconv.Quote(root+"/artifacts"),
		strconv.Quote(root+"/work"), strconv.Quote(root+"/secrets/local-owner.token"),
		strconv.Quote(root+"/effective_config.json"))
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

type countingAgent struct {
	now      func() time.Time
	launches *atomic.Int64
	content  []byte
	mu       sync.Mutex
	requests map[goal.ExecutionRef]ports.AgentLaunchRequest
	closed   bool
}

func newCountingAgent(clock application.Clock, launches *atomic.Int64) *countingAgent {
	return &countingAgent{now: clock.Now, launches: launches, requests: make(map[goal.ExecutionRef]ports.AgentLaunchRequest)}
}

func (agent *countingAgent) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return ports.AgentCapabilities{ProviderRef: "provider:test"}, nil
}

func (agent *countingAgent) Launch(ctx context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	if err := ctx.Err(); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	agent.mu.Lock()
	defer agent.mu.Unlock()
	if agent.closed {
		return ports.AgentLaunchReceipt{}, errors.New("test_agent.closed")
	}
	if existing, ok := agent.requests[request.ExecutionRef]; ok {
		if existing != request {
			return ports.AgentLaunchReceipt{}, errors.New("test_agent.conflict")
		}
	} else {
		agent.requests[request.ExecutionRef] = request
		agent.launches.Add(1)
	}
	return ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, ProviderRef: "provider:test",
		ExternalRef:    "test:" + request.ExecutionRef.String(),
		IdempotencyKey: request.IdempotencyKey, AcceptedAt: agent.now(),
	}, nil
}

func (agent *countingAgent) Observe(ctx context.Context, executionRef goal.ExecutionRef) (ports.AgentObservation, error) {
	if err := ctx.Err(); err != nil {
		return ports.AgentObservation{}, err
	}
	agent.mu.Lock()
	request, ok := agent.requests[executionRef]
	closed := agent.closed
	agent.mu.Unlock()
	if closed || !ok {
		return ports.AgentObservation{}, errors.New("test_agent.not_found")
	}
	content := agent.content
	if len(content) == 0 {
		content = []byte("artifact:" + request.Objective)
	}
	return ports.AgentObservation{
		ExecutionRef: executionRef, Status: ports.AgentCompleted,
		MediaType: "text/plain", Content: append([]byte(nil), content...),
		ObservedAt: agent.now(),
	}, nil
}

func (agent *countingAgent) Shutdown(context.Context) error {
	agent.mu.Lock()
	agent.closed = true
	agent.mu.Unlock()
	return nil
}

func countingFactory(launches *atomic.Int64) AgentFactory {
	return func(_ config.Snapshot, clock application.Clock) (AgentAdapter, error) {
		return newCountingAgent(clock, launches), nil
	}
}

func constantContentFactory(launches *atomic.Int64, content []byte) AgentFactory {
	return func(_ config.Snapshot, clock application.Clock) (AgentAdapter, error) {
		agent := newCountingAgent(clock, launches)
		agent.content = append([]byte(nil), content...)
		return agent, nil
	}
}

func submitTestGoal(t *testing.T, runtime *Runtime, requestRef string) goal.GoalRef {
	t.Helper()
	actor, _ := goal.NewActorRef("actor:local-owner")
	project, _ := goal.NewProjectRef("project:default")
	result, err := runtime.Orchestrator().Submit(context.Background(), application.SubmitRequest{
		RequestRef: requestRef, ActorRef: actor, ProjectRef: project, Statement: "produce restart evidence",
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	return result.Record.Goal.Ref()
}

func waitTerminalGoal(t *testing.T, runtime *Runtime, ref goal.GoalRef) application.GoalRecord {
	t.Helper()
	actor, _ := goal.NewActorRef("actor:local-owner")
	project, _ := goal.NewProjectRef("project:default")
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		record, err := runtime.Orchestrator().GetGoal(context.Background(), application.GoalQuery{
			ActorRef: actor, ProjectRef: project, GoalRef: ref,
		})
		if err == nil && record.Goal.IsTerminal() {
			return record
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("goal %s did not become terminal", ref.String())
	return application.GoalRecord{}
}

const sleeperMarker = "bootstrap-helper-sleep"

func TestBootstrapSleeper(t *testing.T) {
	marked := false
	for _, argument := range os.Args {
		if argument == sleeperMarker {
			marked = true
			break
		}
	}
	if !marked {
		return
	}
	time.Sleep(30 * time.Second)
}

type processAgent struct {
	ctx       context.Context
	cancel    context.CancelFunc
	now       func() time.Time
	started   chan int
	mu        sync.Mutex
	execution goal.ExecutionRef
	request   ports.AgentLaunchRequest
	command   *exec.Cmd
	done      chan struct{}
	wait      sync.WaitGroup
	closed    bool
}

func newProcessAgent(clock application.Clock) *processAgent {
	ctx, cancel := context.WithCancel(context.Background())
	return &processAgent{ctx: ctx, cancel: cancel, now: clock.Now, started: make(chan int, 1)}
}

func (agent *processAgent) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return ports.AgentCapabilities{ProviderRef: "provider:process-test"}, nil
}

func (agent *processAgent) Launch(_ context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	if agent.closed {
		return ports.AgentLaunchReceipt{}, errors.New("process_agent.closed")
	}
	if agent.command != nil {
		if agent.execution != request.ExecutionRef || agent.request.IdempotencyKey != request.IdempotencyKey {
			return ports.AgentLaunchReceipt{}, errors.New("process_agent.conflict")
		}
		return agent.receipt(request), nil
	}
	command := exec.CommandContext(agent.ctx, os.Args[0], "-test.run=^TestBootstrapSleeper$", "--", sleeperMarker)
	if err := command.Start(); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	agent.command = command
	agent.execution = request.ExecutionRef
	agent.request = request
	agent.done = make(chan struct{})
	agent.wait.Add(1)
	agent.started <- command.Process.Pid
	go func() {
		defer agent.wait.Done()
		_ = command.Wait()
		close(agent.done)
	}()
	return agent.receipt(request), nil
}

func (agent *processAgent) receipt(request ports.AgentLaunchRequest) ports.AgentLaunchReceipt {
	return ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, ProviderRef: "provider:process-test",
		ExternalRef: "pid-owned", IdempotencyKey: request.IdempotencyKey, AcceptedAt: agent.now(),
	}
}

func (agent *processAgent) Observe(context.Context, goal.ExecutionRef) (ports.AgentObservation, error) {
	return ports.AgentObservation{
		ExecutionRef: agent.execution, Status: ports.AgentRunning, ObservedAt: agent.now(),
	}, nil
}

func (agent *processAgent) Shutdown(ctx context.Context) error {
	agent.mu.Lock()
	if !agent.closed {
		agent.closed = true
		agent.cancel()
	}
	agent.mu.Unlock()
	finished := make(chan struct{})
	go func() {
		agent.wait.Wait()
		close(finished)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-finished:
		return nil
	}
}
