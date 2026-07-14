package fake

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type Config struct {
	ProviderRef string
	MediaType   string
	Content     []byte
	Now         func() time.Time
}

type Adapter struct {
	config Config
	mu     sync.Mutex
	runs   map[goal.ExecutionRef]run
}

type run struct {
	request ports.AgentLaunchRequest
	receipt ports.AgentLaunchReceipt
}

func New(config Config) (*Adapter, error) {
	if config.ProviderRef == "" || config.MediaType == "" || len(config.Content) == 0 || config.Now == nil {
		return nil, errors.New("fake_agent.config_invalid")
	}
	return &Adapter{config: config, runs: make(map[goal.ExecutionRef]run)}, nil
}

func (adapter *Adapter) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	if adapter == nil {
		return ports.AgentCapabilities{}, errors.New("fake_agent.unavailable")
	}
	return ports.AgentCapabilities{ProviderRef: adapter.config.ProviderRef}, nil
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
		ExecutionRef:   request.ExecutionRef,
		SpecHash:       request.SpecHash,
		ProviderRef:    adapter.config.ProviderRef,
		ExternalRef:    "fake:" + request.ExecutionRef.String(),
		IdempotencyKey: request.IdempotencyKey,
		AcceptedAt:     adapter.config.Now(),
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
	adapter.mu.Unlock()
	if !ok {
		return ports.AgentObservation{}, errors.New("fake_agent.execution_not_found")
	}
	content := append([]byte(nil), adapter.config.Content...)
	return ports.AgentObservation{
		ExecutionRef: executionRef,
		SpecHash:     run.request.SpecHash,
		Status:       ports.AgentCompleted,
		MediaType:    adapter.config.MediaType,
		Content:      content,
		ObservedAt:   adapter.config.Now(),
	}, nil
}

var _ application.AgentLauncher = (*Adapter)(nil)
var _ application.AgentObserver = (*Adapter)(nil)
