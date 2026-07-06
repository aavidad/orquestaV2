package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	channel "orquesta/modulos/orquesta-operator-director-channel"
	operator "orquesta/modulos/orquesta-operator-mcp"
)

func TestOperatorDirectorChannelServerBridgeV0DespachaPorPuertoHermesTemporal(t *testing.T) {
	query := &fakeOperatorDirectorChannelQueryV0{}
	store := newOperatorDirectorChannelMemoryStoreV0()
	service := newOperatorDirectorChannelServiceV0(query, store)

	response, issues, err := channel.DispatchOperatorDirectorMessageV0(context.Background(), service, channel.OperatorMessageV0{
		RequestRef: "request-ref-1",
		TargetRef:  "director-ref-1",
		AdapterRef: "query-connector-ref-operator-director-channel",
		Intent:     "status",
		Body:       "estado compacto",
	})
	if err != nil || len(issues) != 0 {
		t.Fatalf("dispatch err=%v issues=%+v", err, issues)
	}
	if query.input.Question != "estado compacto" ||
		query.input.QueryConnectorRef != "query-connector-ref-operator-director-channel" ||
		response.ResponseRef != "answer-ref-1" {
		t.Fatalf("bridge inesperado query=%+v response=%+v", query.input, response)
	}
	if got := store.ExchangesV0(); len(got) != 1 || got[0].Response.AckRef == "" {
		t.Fatalf("ack no persistido: %+v", got)
	}
}

func TestMCPOperatorDirectorChannelServerJSONRPCV0ExponeToolYDevuelveAck(t *testing.T) {
	query := &fakeOperatorDirectorChannelQueryV0{}
	store := newOperatorDirectorChannelMemoryStoreV0()
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{
		OperatorDirectorMessage: newOperatorDirectorChannelServiceV0(query, store),
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	defer server.Close()

	var tools mcpToolListResultV0
	callMCPJSONRPCTestV0(t, server.URL+mcpRealHTTPPathV0, "tools/list", map[string]any{}, &tools)
	if mcpToolByNameTestV0(tools.Tools, channel.OperatorDirectorMessageToolNameV0) == nil {
		t.Fatalf("tool operator-director no listado")
	}

	var call mcpToolCallResultV0
	callMCPJSONRPCTestV0(t, server.URL+mcpRealHTTPPathV0, "tools/call", map[string]any{
		"name": channel.OperatorDirectorMessageToolNameV0,
		"arguments": map[string]any{
			"request_ref": "request-ref-1",
			"target_ref":  "director-ref-1",
			"adapter_ref": "query-connector-ref-operator-director-channel",
			"intent":      "observe_run",
			"body":        "observa run-ref-1",
		},
	}, &call)
	if len(call.Content) != 1 || call.IsError {
		t.Fatalf("call inesperado: %+v", call)
	}
	var result orquestamcp.MCPOperatorDirectorMessageToolResultV0
	if err := json.Unmarshal([]byte(call.Content[0].Text), &result); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if result.Estado != "ok" || result.Response == nil || result.Response.AckRef == "" || len(store.ExchangesV0()) != 1 {
		t.Fatalf("ack esperado: result=%+v store=%+v", result, store.ExchangesV0())
	}
}

func TestOperatorDirectorChannelStackV0CableaServicioConStore(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerStateDirV0, stateDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexCommandV0, filepath.Join(projectDir, "codex-bin"))
	t.Setenv(envOPESBaseURLV0, "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv(envDomainWorkFileEnabledV0, "")
	t.Setenv(envDomainWorkFileDirV0, "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvWithGoalBackendV0(config, serverCodexGoalBackendV0{})
	if err != nil {
		t.Fatalf("buildStackFromEnvWithGoalBackendV0: %v", err)
	}
	result := orquestamcp.MCPTransportOperatorDirectorMessageExecutorV0{
		Service: stack.MCPTransportBindings.OperatorDirectorMessage,
	}.Execute(context.Background(), channel.OperatorMessageV0{
		RequestRef: "request-ref-stack-operator-director-001",
		TargetRef:  "director-ref-stack-operator-director-001",
		Body:       "estado",
	})
	if result.Estado != orquestamcp.MCPOperatorDirectorMessageEstadoErrorV0 ||
		result.ErrorCode != channel.ErrOperatorMessagePortUnavailableV0 {
		t.Fatalf("servicio stack debe existir con store y fallar por dispatch ausente: %+v", result)
	}
}

type fakeOperatorDirectorChannelQueryV0 struct {
	input operator.OperatorDirectedQueryV0
}

func (fake *fakeOperatorDirectorChannelQueryV0) RaiseOperatorDirectedQueryV0(
	input operator.OperatorDirectedQueryV0,
) (operator.OperatorMCPDirectedQueryResultV0, error) {
	fake.input = input
	return operator.OperatorMCPDirectedQueryResultV0{
		Accepted:   true,
		AnswerRef:  "answer-ref-1",
		NextAction: "operator_message_accepted",
		TraceRefs:  []string{"evidence-ref-operator-director-channel-bridge"},
	}, nil
}
