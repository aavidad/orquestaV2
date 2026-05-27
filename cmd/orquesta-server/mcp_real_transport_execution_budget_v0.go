package main

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaobservability "orquesta/modulos/orquesta-observability"
)

const (
	mcpDurationBucketUnder100msV0 = "under_100ms"
	mcpDurationBucketUnder1sV0    = "100ms_1s"
	mcpDurationBucketUnder10sV0   = "1s_10s"
	mcpDurationBucketUnder30sV0   = "10s_30s"
	mcpDurationBucketOver30sV0    = "over_30s"
)

var (
	errMCPExecutionTimeoutV0   = errors.New(orquestamcp.MCPTransportExecutionTimeoutV0)
	errMCPExecutionCancelledV0 = errors.New(orquestamcp.MCPTransportExecutionCancelledV0)
)

type mcpRealCorrelationContextKeyV0 struct{}

type mcpExecutionResultV0 struct {
	payload json.RawMessage
	err     error
}

func (registry *mcpRealTransportRegistryV0) executeResourceHandlerV0(
	ctx context.Context,
	resource orquestamcp.MCPTransportResourceEnvelopeV0,
) (json.RawMessage, error) {
	budget := orquestamcp.NormalizeMCPTransportExecutionBudgetV0(
		resource.ExecutionBudget,
		orquestamcp.MCPTransportExecutionProfileResourceReadV0,
	)
	return registry.executeHandlerWithBudgetV0(ctx, resource.Name, budget, func(ctx context.Context) (json.RawMessage, error) {
		return resource.Handler(ctx)
	})
}

func (registry *mcpRealTransportRegistryV0) executeToolHandlerV0(
	ctx context.Context,
	tool orquestamcp.MCPTransportToolEnvelopeV0,
	arguments json.RawMessage,
) (json.RawMessage, error) {
	budget := orquestamcp.NormalizeMCPTransportExecutionBudgetV0(
		tool.ExecutionBudget,
		orquestamcp.MCPTransportExecutionProfileControlPlaneMutationV0,
	)
	return registry.executeHandlerWithBudgetV0(ctx, tool.Name, budget, func(ctx context.Context) (json.RawMessage, error) {
		return tool.Handler(ctx, arguments)
	})
}

func (registry *mcpRealTransportRegistryV0) executeHandlerWithBudgetV0(
	parent context.Context,
	method string,
	budget orquestamcp.MCPTransportExecutionBudgetV0,
	handler func(context.Context) (json.RawMessage, error),
) (json.RawMessage, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeoutCause(parent, budget.MaxDuration, errMCPExecutionTimeoutV0)
	defer cancel()

	resultc := make(chan mcpExecutionResultV0, 1)
	go func() {
		payload, err := handler(ctx)
		resultc <- mcpExecutionResultV0{payload: payload, err: err}
	}()

	reason := orquestaobservability.MCPExecutionReasonOKV0
	var result mcpExecutionResultV0
	select {
	case result = <-resultc:
		if result.err != nil {
			reason = orquestaobservability.MCPExecutionReasonHandlerErrorV0
			if executionErr := mcpExecutionErrorForContextV0(ctx); executionErr != nil {
				result.err = executionErr
				reason = mcpExecutionReasonForErrorV0(executionErr)
			}
		}
	case <-ctx.Done():
		result.err = mcpExecutionErrorForContextV0(ctx)
		reason = mcpExecutionReasonForErrorV0(result.err)
	}
	registry.observeMCPExecutionV0(method, budget.Profile, reason, time.Since(start), parent)
	return result.payload, result.err
}

func mcpExecutionErrorForContextV0(ctx context.Context) error {
	if errors.Is(context.Cause(ctx), errMCPExecutionTimeoutV0) {
		return errMCPExecutionTimeoutV0
	}
	if err := ctx.Err(); errors.Is(err, context.Canceled) {
		return errMCPExecutionCancelledV0
	}
	return ctx.Err()
}

func mcpExecutionReasonForErrorV0(err error) string {
	switch {
	case errors.Is(err, errMCPExecutionTimeoutV0):
		return orquestaobservability.MCPExecutionReasonTimeoutV0
	case errors.Is(err, errMCPExecutionCancelledV0):
		return orquestaobservability.MCPExecutionReasonCancelledV0
	default:
		return orquestaobservability.MCPExecutionReasonHandlerErrorV0
	}
}

func (registry *mcpRealTransportRegistryV0) mcpExecutionRPCErrorV0(
	name string,
	budget orquestamcp.MCPTransportExecutionBudgetV0,
	err error,
) *mcpJSONRPCErrorV0 {
	return registry.mcpExecutionRPCErrorWithDefaultProfileV0(
		name,
		budget,
		orquestamcp.MCPTransportExecutionProfileControlPlaneMutationV0,
		err,
	)
}

func (registry *mcpRealTransportRegistryV0) mcpResourceExecutionRPCErrorV0(
	name string,
	budget orquestamcp.MCPTransportExecutionBudgetV0,
	err error,
) *mcpJSONRPCErrorV0 {
	return registry.mcpExecutionRPCErrorWithDefaultProfileV0(
		name,
		budget,
		orquestamcp.MCPTransportExecutionProfileResourceReadV0,
		err,
	)
}

func (registry *mcpRealTransportRegistryV0) mcpExecutionRPCErrorWithDefaultProfileV0(
	name string,
	budget orquestamcp.MCPTransportExecutionBudgetV0,
	defaultProfile string,
	err error,
) *mcpJSONRPCErrorV0 {
	budget = orquestamcp.NormalizeMCPTransportExecutionBudgetV0(
		budget,
		defaultProfile,
	)
	switch {
	case errors.Is(err, errMCPExecutionTimeoutV0):
		return registry.mcpExecutionBudgetRPCErrorV0(name, budget, budget.TimeoutCode)
	case errors.Is(err, errMCPExecutionCancelledV0):
		return registry.mcpExecutionBudgetRPCErrorV0(name, budget, budget.CancelledCode)
	default:
		return nil
	}
}

func (registry *mcpRealTransportRegistryV0) mcpExecutionBudgetRPCErrorV0(
	name string,
	budget orquestamcp.MCPTransportExecutionBudgetV0,
	code string,
) *mcpJSONRPCErrorV0 {
	return &mcpJSONRPCErrorV0{
		Code:    -32000,
		Message: code,
		Data: map[string]string{
			"error_code": code,
			"method_ref": publicMCPRegistrationRefV0(name, false),
			"profile":    budget.Profile,
		},
	}
}

func (registry *mcpRealTransportRegistryV0) observeMCPExecutionV0(
	method string,
	profile string,
	reason string,
	duration time.Duration,
	ctx context.Context,
) {
	if registry.observer == nil {
		return
	}
	registry.observer(orquestaobservability.NormalizeMCPExecutionObservationV0(
		orquestaobservability.MCPExecutionObservationV0{
			Method:         publicMCPRegistrationRefV0(method, false),
			Profile:        profile,
			ReasonCode:     reason,
			DurationBucket: mcpExecutionDurationBucketV0(duration),
			CorrelationID:  mcpCorrelationIDFromContextV0(ctx),
		},
	))
}

func mcpExecutionDurationBucketV0(duration time.Duration) string {
	switch {
	case duration < 100*time.Millisecond:
		return mcpDurationBucketUnder100msV0
	case duration < time.Second:
		return mcpDurationBucketUnder1sV0
	case duration < 10*time.Second:
		return mcpDurationBucketUnder10sV0
	case duration < 30*time.Second:
		return mcpDurationBucketUnder30sV0
	default:
		return mcpDurationBucketOver30sV0
	}
}

func mcpJSONRPCIDCorrelationV0(raw json.RawMessage) string {
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return publicMCPRegistrationRefV0(value, false)
	}
	return ""
}

func mcpCorrelationIDFromContextV0(ctx context.Context) string {
	value, _ := ctx.Value(mcpRealCorrelationContextKeyV0{}).(string)
	return value
}
