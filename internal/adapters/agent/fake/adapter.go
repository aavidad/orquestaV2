package fake

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

const (
	ModelRef = "fake-default"
	AgentRef = "agent:fake"
)

type Config struct {
	ProviderRef         string
	MediaType           string
	Content             []byte
	Now                 func() time.Time
	ControlCapabilities *ports.AgentControlCapabilities
	Usage               governance.ResourceUsage
}

type Adapter struct {
	config              Config
	controlCapabilities ports.AgentControlCapabilities
	mu                  sync.Mutex
	runs                map[goal.ExecutionRef]run
}

type run struct {
	request  ports.AgentLaunchRequest
	receipt  ports.AgentLaunchReceipt
	terminal ports.AgentStatus
	stopped  bool
	stops    map[string]stopRecord
}

func New(config Config) (*Adapter, error) {
	if config.Usage == (governance.ResourceUsage{}) {
		config.Usage = governance.ResourceUsage{Quality: governance.UsageQualityUnknown}
	}
	if config.MediaType == "" || len(config.Content) == 0 || config.Now == nil ||
		ports.ValidateAgentCapabilities(agentCapabilities(config.ProviderRef)) != nil ||
		governance.ValidateResourceUsage(config.Usage) != nil {
		return nil, errors.New("fake_agent.config_invalid")
	}
	controls := ports.AgentControlCapabilities{CooperativeStop: true, ForcedStop: true}
	if config.ControlCapabilities != nil {
		controls = *config.ControlCapabilities
	}
	return &Adapter{config: config, controlCapabilities: controls, runs: make(map[goal.ExecutionRef]run)}, nil
}

func (adapter *Adapter) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	if adapter == nil {
		return ports.AgentCapabilities{}, errors.New("fake_agent.unavailable")
	}
	return agentCapabilities(adapter.config.ProviderRef), nil
}

func agentCapabilities(providerRef string) ports.AgentCapabilities {
	return ports.AgentCapabilities{
		ProviderRef:  providerRef,
		ModelRef:     ModelRef,
		AgentRef:     AgentRef,
		Unrestricted: true,
	}
}

func (adapter *Adapter) Launch(ctx context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	if adapter == nil {
		return ports.AgentLaunchReceipt{}, errors.New("fake_agent.unavailable")
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if existing, ok := adapter.runs[request.ExecutionRef]; ok {
		if !reflect.DeepEqual(existing.request, request) {
			return ports.AgentLaunchReceipt{}, errors.New("fake_agent.execution_conflict")
		}
		return existing.receipt, nil
	}
	receipt := ports.AgentLaunchReceipt{
		ExecutionRef:      request.ExecutionRef,
		GoalRef:           request.GoalRef,
		WorkItemRef:       request.WorkItemRef,
		PlanGeneration:    request.PlanGeneration,
		AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt:  request.ExecutionAttempt,
		LaunchActionFence: request.EffectAuthority.ActionFence,
		SpecHash:          request.SpecHash,
		ProviderRef:       adapter.config.ProviderRef,
		ModelRef:          ModelRef,
		AgentRef:          AgentRef,
		ExternalRef:       "fake:" + request.ExecutionRef.String(),
		IdempotencyKey:    request.IdempotencyKey,
		ReceiptRef:        fakeReceiptRef("launch", request.ExecutionRef, request.IdempotencyKey),
		AcceptedAt:        adapter.config.Now(),
	}
	if err := ports.ValidateAgentLaunchReceipt(request, receipt); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	adapter.runs[request.ExecutionRef] = run{request: request, receipt: receipt}
	return receipt, nil
}

func (adapter *Adapter) Observe(ctx context.Context, executionRef goal.ExecutionRef) (ports.AgentObservation, error) {
	if adapter == nil {
		return ports.AgentObservation{}, errors.New("fake_agent.unavailable")
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentObservation{}, err
	}
	adapter.mu.Lock()
	run, ok := adapter.runs[executionRef]
	if !ok {
		adapter.mu.Unlock()
		return ports.AgentObservation{}, errors.New("fake_agent.execution_not_found")
	}
	if run.stopped {
		adapter.mu.Unlock()
		return ports.AgentObservation{}, errors.New("fake_agent.execution_stopped")
	}
	run.terminal = ports.AgentCompleted
	adapter.runs[executionRef] = run
	adapter.mu.Unlock()
	content := append([]byte(nil), adapter.config.Content...)
	return ports.AgentObservation{
		ExecutionRef: executionRef,
		SpecHash:     run.request.SpecHash,
		Status:       ports.AgentCompleted,
		MediaType:    adapter.config.MediaType,
		Content:      content,
		Usage:        adapter.config.Usage,
		ObservedAt:   adapter.config.Now(),
	}, nil
}

func (adapter *Adapter) ObserveAgent(
	ctx context.Context,
	request ports.AgentObserveRequest,
) (ports.AgentObservation, error) {
	if err := ports.ValidateAgentObserveRequest(request); err != nil {
		return ports.AgentObservation{}, err
	}
	if adapter == nil {
		return ports.AgentObservation{}, errors.New("fake_agent.unavailable")
	}
	adapter.mu.Lock()
	run, found := adapter.runs[request.ExecutionRef]
	adapter.mu.Unlock()
	if !found {
		return ports.AgentObservation{}, errors.New("fake_agent.execution_not_found")
	}
	if err := ports.ValidateAgentObserveTarget(run.request, run.receipt, request); err != nil {
		return ports.AgentObservation{}, errors.New("fake_agent.observation_conflict")
	}
	return adapter.Observe(ctx, request.ExecutionRef)
}

var _ application.AgentLauncher = (*Adapter)(nil)
var _ application.AgentObserver = (*Adapter)(nil)
var _ application.AgentController = (*Adapter)(nil)
