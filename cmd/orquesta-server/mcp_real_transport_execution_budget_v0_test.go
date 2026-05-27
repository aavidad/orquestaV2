package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func TestMCPRealTransportV0TimeoutToolSinEcoDeArgumentosV0(t *testing.T) {
	var observations []orquestaobservability.MCPExecutionObservationV0
	registry := newMCPExecutionBudgetRegistryTestV0(func(observation orquestaobservability.MCPExecutionObservationV0) {
		observations = append(observations, observation)
	})
	err := registry.RegisterToolV0(orquestamcp.MCPTransportToolEnvelopeV0{
		Name: "orquesta.test.slow_tool.v0",
		ExecutionBudget: orquestamcp.MCPTransportExecutionBudgetV0{
			Profile:       orquestamcp.MCPTransportExecutionProfileControlPlaneMutationV0,
			MaxDuration:   time.Millisecond,
			TimeoutCode:   orquestamcp.MCPTransportExecutionTimeoutV0,
			CancelledCode: orquestamcp.MCPTransportExecutionCancelledV0,
		},
		Handler: func(context.Context, json.RawMessage) (json.RawMessage, error) {
			time.Sleep(50 * time.Millisecond)
			return json.RawMessage(`{"ok":true}`), nil
		},
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	_, rpcErr := registry.callToolV0(context.Background(), json.RawMessage(`{
		"name":"orquesta.test.slow_tool.v0",
		"arguments":{"secret":"raw prompt token /home/alberto"}
	}`))
	encoded, _ := json.Marshal(rpcErr)
	text := strings.ToLower(string(encoded))
	if rpcErr == nil ||
		rpcErr.Message != orquestamcp.MCPTransportExecutionTimeoutV0 ||
		rpcErr.Data["error_code"] != orquestamcp.MCPTransportExecutionTimeoutV0 ||
		rpcErr.Data["profile"] != orquestamcp.MCPTransportExecutionProfileControlPlaneMutationV0 {
		t.Fatalf("rpcErr=%s", string(encoded))
	}
	for _, forbidden := range []string{"secret", "prompt", "/home/", "token"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("rpc filtra %q: %s", forbidden, text)
		}
	}
	if len(observations) != 1 ||
		observations[0].Method != "orquesta.test.slow_tool.v0" ||
		observations[0].ReasonCode != orquestaobservability.MCPExecutionReasonTimeoutV0 ||
		observations[0].DurationBucket == "" {
		t.Fatalf("observations=%+v", observations)
	}
}

func TestMCPRealTransportV0CancelacionRequestMapeaCodigoPublicoV0(t *testing.T) {
	registry := newMCPExecutionBudgetRegistryTestV0(nil)
	err := registry.RegisterToolV0(orquestamcp.MCPTransportToolEnvelopeV0{
		Name:            "orquesta.test.cancel_tool.v0",
		ExecutionBudget: orquestamcp.MCPTransportToolExecutionBudgetV0(orquestamcp.MCPTransportExecutionProfileDefaultToolV0),
		Handler: func(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, rpcErr := registry.callToolV0(ctx, json.RawMessage(`{
		"name":"orquesta.test.cancel_tool.v0",
		"arguments":{}
	}`))
	if rpcErr == nil ||
		rpcErr.Message != orquestamcp.MCPTransportExecutionCancelledV0 ||
		rpcErr.Data["error_code"] != orquestamcp.MCPTransportExecutionCancelledV0 {
		t.Fatalf("rpcErr=%+v", rpcErr)
	}
}

func TestMCPRealTransportV0TimeoutResourceUsaPerfilReadOnlyV0(t *testing.T) {
	registry := newMCPExecutionBudgetRegistryTestV0(nil)
	err := registry.RegisterResourceV0(orquestamcp.MCPTransportResourceEnvelopeV0{
		Name:        "orquesta.test.slow_resource.v0",
		URI:         "orquesta://test/slow-resource/v0",
		ContentType: "application/json",
		ExecutionBudget: orquestamcp.MCPTransportExecutionBudgetV0{
			Profile:       orquestamcp.MCPTransportExecutionProfileResourceReadV0,
			MaxDuration:   time.Millisecond,
			TimeoutCode:   orquestamcp.MCPTransportExecutionTimeoutV0,
			CancelledCode: orquestamcp.MCPTransportExecutionCancelledV0,
		},
		Handler: func(context.Context) (json.RawMessage, error) {
			time.Sleep(50 * time.Millisecond)
			return json.RawMessage(`{"ok":true}`), nil
		},
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	_, rpcErr := registry.readResourceV0(context.Background(), json.RawMessage(`{
		"name":"orquesta.test.slow_resource.v0"
	}`))
	if rpcErr == nil ||
		rpcErr.Message != orquestamcp.MCPTransportExecutionTimeoutV0 ||
		rpcErr.Data["error_code"] != orquestamcp.MCPTransportExecutionTimeoutV0 ||
		rpcErr.Data["profile"] != orquestamcp.MCPTransportExecutionProfileResourceReadV0 {
		encoded, _ := json.Marshal(rpcErr)
		t.Fatalf("rpcErr=%s", string(encoded))
	}
}

func newMCPExecutionBudgetRegistryTestV0(
	observer mcpRealExecutionObserverV0,
) *mcpRealTransportRegistryV0 {
	return &mcpRealTransportRegistryV0{
		resourcesByName: map[string]orquestamcp.MCPTransportResourceEnvelopeV0{},
		resourcesByURI:  map[string]orquestamcp.MCPTransportResourceEnvelopeV0{},
		tools:           map[string]orquestamcp.MCPTransportToolEnvelopeV0{},
		observer:        observer,
	}
}
