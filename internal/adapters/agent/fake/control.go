package fake

import (
	"context"
	"errors"
	"reflect"

	"orquesta/internal/ports"
)

type stopRecord struct {
	request ports.AgentStopRequest
	receipt ports.AgentStopReceipt
}

func (adapter *Adapter) ControlCapabilities(ctx context.Context) (ports.AgentControlCapabilities, error) {
	if adapter == nil {
		return ports.AgentControlCapabilities{}, errors.New("fake_agent.unavailable")
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentControlCapabilities{}, err
	}
	return adapter.controlCapabilities, nil
}

func (adapter *Adapter) Stop(ctx context.Context, request ports.AgentStopRequest) (ports.AgentStopReceipt, error) {
	if adapter == nil {
		return ports.AgentStopReceipt{}, errors.New("fake_agent.unavailable")
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentStopReceipt{}, err
	}
	if err := ports.ValidateAgentStopRequest(request); err != nil {
		return ports.AgentStopReceipt{}, err
	}

	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	current, ok := adapter.runs[request.ExecutionRef]
	if !ok {
		return ports.AgentStopReceipt{}, errors.New("fake_agent.execution_not_found")
	}
	if err := ports.ValidateAgentStopTarget(current.receipt, request); err != nil {
		return ports.AgentStopReceipt{}, err
	}
	if existing, found := current.stops[request.IdempotencyKey]; found {
		if !reflect.DeepEqual(existing.request, request) {
			return ports.AgentStopReceipt{}, errors.New("fake_agent.stop_conflict")
		}
		return existing.receipt, nil
	}

	status := ports.AgentStopped
	if !ports.SupportsAgentStopMode(adapter.controlCapabilities, request.Mode) {
		status = ports.AgentStopUnsupported
	} else if current.stopped {
		status = ports.AgentStopAlreadyStopped
	} else if current.terminal == ports.AgentCompleted {
		status = ports.AgentStopAlreadyCompleted
	} else if current.terminal == ports.AgentFailed {
		status = ports.AgentStopAlreadyFailed
	}
	receipt := adapter.newStopReceipt(request, status)
	if err := ports.ValidateAgentStopReceipt(request, receipt); err != nil {
		return ports.AgentStopReceipt{}, err
	}
	if current.stops == nil {
		current.stops = make(map[string]stopRecord)
	}
	current.stops[request.IdempotencyKey] = stopRecord{request: request, receipt: receipt}
	current.stopped = current.stopped || status == ports.AgentStopped
	adapter.runs[request.ExecutionRef] = current
	return receipt, nil
}

func (adapter *Adapter) newStopReceipt(request ports.AgentStopRequest, status ports.AgentStopStatus) ports.AgentStopReceipt {
	receipt := ports.AgentStopReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: request.ProviderRef, ModelRef: request.ModelRef, AgentRef: request.AgentRef,
		ExternalRef: request.ExternalRef, Mode: request.Mode, IdempotencyKey: request.IdempotencyKey,
		Status: status,
	}
	if status == ports.AgentStopped || status == ports.AgentStopAlreadyStopped ||
		status == ports.AgentStopAlreadyCompleted || status == ports.AgentStopAlreadyFailed {
		receipt.ReceiptRef = fakeReceiptRef("stop", request.ExecutionRef, request.IdempotencyKey)
		receipt.ConfirmedAt = adapter.config.Now()
	}
	return receipt
}
