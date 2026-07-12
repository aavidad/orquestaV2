package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	channel "orquesta/modulos/orquesta-operator-director-channel"
	operator "orquesta/modulos/orquesta-operator-mcp"
)

func operatorDirectorMailboxMessageForTestV0() channel.OperatorMessageV0 {
	return channel.OperatorMessageV0{
		RequestRef:      "request-ref-operator-mailbox-001",
		ConversationRef: "conv-ref-operator-mailbox-001",
		SenderRef:       "agent-ref-revisor",
		TargetRef:       "agent-ref-hermes",
		Intent:          channel.OperatorMessageIntentInstructionV0,
		Body:            "retoma el hito H0a",
		EvidenceRefs:    []string{"docs/instrucciones_hermes_2026-07-12.md"},
	}
}

// Sin conector MCP del operador el canal NO puede quedarse sin puerto: debe
// degradar al buzon durable y dejar el mensaje entregado y trazable.
func TestOperatorDirectorChannelSinConectorEntregaEnBuzonDurableV0(t *testing.T) {
	stateDir := t.TempDir()
	service := newOperatorDirectorChannelServiceWithMailboxV0(
		nil,
		newOperatorDirectorChannelMemoryStoreV0(),
		stateDir,
		true,
	)
	if service.Dispatcher == nil {
		t.Fatal("el canal quedo sin dispatcher: operator_message_port_unavailable")
	}

	response, issues, err := channel.DispatchOperatorDirectorMessageV0(
		context.Background(),
		service,
		operatorDirectorMailboxMessageForTestV0(),
	)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if response.Status != operatorDirectorMailboxStatusV0 || strings.TrimSpace(response.MessageRef) == "" {
		t.Fatalf("response=%+v", response)
	}

	raw, err := os.ReadFile(filepath.Join(stateDir, operatorDirectorMailboxFileNameV0))
	if err != nil {
		t.Fatalf("el mensaje no quedo en el buzon durable: %v", err)
	}
	var entry operatorDirectorMailboxEntryV0
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(raw))), &entry); err != nil {
		t.Fatalf("entrada de buzon invalida: %v", err)
	}
	if entry.TargetRef != "agent-ref-hermes" ||
		entry.SenderRef != "agent-ref-revisor" ||
		entry.Intent != channel.OperatorMessageIntentInstructionV0 ||
		entry.Body != "retoma el hito H0a" ||
		strings.TrimSpace(entry.QueuedAt) == "" {
		t.Fatalf("entry=%+v", entry)
	}
}

// Prueba la composicion completa (stack -> registro MCP -> tools/call ->
// fichero durable). Si el bootstrap deja de inyectar el dispatcher, vuelve a
// aparecer operator_message_port_unavailable y este test falla.
func TestOperatorDirectorMailboxStackToolsCallPersisteMensajeV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	enabled := true

	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerStateDirV0, stateDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexCommandV0, filepath.Join(projectDir, "codex-bin"))
	t.Setenv(envOPESBaseURLV0, "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv(envDomainWorkFileEnabledV0, "")
	t.Setenv(envDomainWorkFileDirV0, "")
	t.Setenv(envHermesEnabledV0, "false")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromProjectConfigV0(config, serverCodexGoalBackendV0{}, serverProjectConfigFileV0{
		OperatorDirectorMailbox: serverProjectConfigOperatorDirectorMailboxV0{Enabled: &enabled},
	})
	if err != nil {
		t.Fatalf("buildStackFromProjectConfigV0: %v", err)
	}
	handler, err := newMCPRealHTTPHandlerV0(stack.MCPTransportBindings)
	if err != nil {
		t.Fatalf("newMCPRealHTTPHandlerV0: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	defer server.Close()

	var call mcpToolCallResultV0
	callMCPJSONRPCTestV0(t, server.URL+mcpRealHTTPPathV0, "tools/call", map[string]any{
		"name": channel.OperatorDirectorMessageToolNameV0,
		"arguments": map[string]any{
			"request_ref": "request-ref-h0d-tools-call-001",
			"sender_ref":  "operator-ref-h0d-test",
			"target_ref":  "agent-ref-hermes",
			"intent":      channel.OperatorMessageIntentInstructionV0,
			"body":        "valida el canal durable H0d",
		},
	}, &call)
	if call.IsError || len(call.Content) != 1 {
		t.Fatalf("tools/call fallo: %+v", call)
	}
	var result orquestamcp.MCPOperatorDirectorMessageToolResultV0
	if err := json.Unmarshal([]byte(call.Content[0].Text), &result); err != nil {
		t.Fatalf("decode tools/call: %v", err)
	}
	if result.Estado != orquestamcp.MCPOperatorDirectorMessageEstadoOKV0 ||
		result.Response == nil || result.Response.Status != operatorDirectorMailboxStatusV0 {
		t.Fatalf("respuesta sin binding durable: %+v", result)
	}

	raw, err := os.ReadFile(filepath.Join(stateDir, operatorDirectorMailboxFileNameV0))
	if err != nil {
		t.Fatalf("tools/call no dejo registro durable: %v", err)
	}
	var entry operatorDirectorMailboxEntryV0
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(raw))), &entry); err != nil {
		t.Fatalf("registro durable invalido: %v", err)
	}
	if entry.RequestRef != "request-ref-h0d-tools-call-001" ||
		entry.TargetRef != "agent-ref-hermes" || entry.Body != "valida el canal durable H0d" {
		t.Fatalf("registro durable inesperado: %+v", entry)
	}
}

// El buzon acumula mensajes sin perder ninguno (una linea JSON por mensaje).
func TestOperatorDirectorMailboxAcumulaMensajesV0(t *testing.T) {
	stateDir := t.TempDir()
	dispatcher := newOperatorDirectorMailboxDispatcherV0(stateDir)
	if dispatcher == nil {
		t.Fatal("dispatcher nil con state dir valido")
	}
	for _, ref := range []string{"request-ref-a", "request-ref-b", "request-ref-c"} {
		message := operatorDirectorMailboxMessageForTestV0()
		message.RequestRef = ref
		message.MessageRef = ""
		if _, err := dispatcher.DispatchOperatorMessageV0(context.Background(), message); err != nil {
			t.Fatalf("dispatch %s: %v", ref, err)
		}
	}
	raw, err := os.ReadFile(filepath.Join(stateDir, operatorDirectorMailboxFileNameV0))
	if err != nil {
		t.Fatalf("read mailbox: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 3 {
		t.Fatalf("mensajes perdidos: %d lineas", len(lines))
	}
}

// Un conector MCP real sigue teniendo prioridad sobre el buzon.
func TestOperatorDirectorChannelConConectorNoUsaBuzonV0(t *testing.T) {
	stateDir := t.TempDir()
	service := newOperatorDirectorChannelServiceWithMailboxV0(
		operatorDirectedQueryStubForMailboxTestV0{},
		newOperatorDirectorChannelMemoryStoreV0(),
		stateDir,
		true,
	)
	if _, _, err := channel.DispatchOperatorDirectorMessageV0(
		context.Background(),
		service,
		operatorDirectorMailboxMessageForTestV0(),
	); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stateDir, operatorDirectorMailboxFileNameV0)); err == nil {
		t.Fatal("con conector real no debe escribirse el buzon")
	}
}

type operatorDirectedQueryStubForMailboxTestV0 struct{}

func (operatorDirectedQueryStubForMailboxTestV0) RaiseOperatorDirectedQueryV0(
	query operator.OperatorDirectedQueryV0,
) (operator.OperatorMCPDirectedQueryResultV0, error) {
	return operator.OperatorMCPDirectedQueryResultV0{
		Accepted:   true,
		AnswerRef:  "answer-ref-" + query.QueryRef,
		NextAction: "observe_later",
	}, nil
}

// Sin opt-in, el canal mantiene su contrato fail-closed: no hay buzon.
func TestOperatorDirectorChannelSinOptInSigueFailClosedV0(t *testing.T) {
	stateDir := t.TempDir()
	service := newOperatorDirectorChannelServiceWithMailboxV0(
		nil,
		newOperatorDirectorChannelMemoryStoreV0(),
		stateDir,
		false,
	)
	if _, _, err := channel.DispatchOperatorDirectorMessageV0(
		context.Background(),
		service,
		operatorDirectorMailboxMessageForTestV0(),
	); err == nil {
		t.Fatal("sin conector ni opt-in el canal debe fallar cerrado")
	}
	if _, err := os.Stat(filepath.Join(stateDir, operatorDirectorMailboxFileNameV0)); err == nil {
		t.Fatal("sin opt-in no debe escribirse el buzon")
	}
}
