package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
