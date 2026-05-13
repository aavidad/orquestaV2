package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMCPTransportV0ServerShutdownQuedaOptInSinPuerto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(context.Background(), MCPServerShutdownToolNameV0, MCPServerShutdownToolInputV0{})
	if err != nil {
		t.Fatalf("call server shutdown unbound: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode unbound: %v", err)
	}
	if result.ErrorCode != MCPTransportToolUnboundV0 ||
		result.Tool != MCPServerShutdownToolNameV0 {
		t.Fatalf("server shutdown debe ser opt-in: %+v", result)
	}
}

func TestMCPTransportV0ServerShutdownInvocaExecutor(t *testing.T) {
	executor := &fakeMCPServerShutdownTransportExecutorV0{
		result: MCPServerShutdownToolResultV0{
			Estado:        MCPServerShutdownEstadoOKV0,
			ShutdownReady: true,
			RunsRequested: 1,
		},
	}
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		ServerShutdown: executor,
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPServerShutdownToolNameV0,
		MCPServerShutdownToolInputV0{Forced: true, RequestedBy: "operator"},
	)
	if err != nil {
		t.Fatalf("call server shutdown: %v", err)
	}
	var result MCPServerShutdownToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !result.ShutdownReady || !executor.input.Forced {
		t.Fatalf("result=%+v input=%+v", result, executor.input)
	}
}

type fakeMCPServerShutdownTransportExecutorV0 struct {
	input  MCPServerShutdownToolInputV0
	result MCPServerShutdownToolResultV0
}

func (executor *fakeMCPServerShutdownTransportExecutorV0) Execute(
	_ context.Context,
	input MCPServerShutdownToolInputV0,
) (MCPServerShutdownToolResultV0, error) {
	executor.input = input
	return executor.result, nil
}
