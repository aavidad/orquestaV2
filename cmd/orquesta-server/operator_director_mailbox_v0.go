package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	channel "orquesta/modulos/orquesta-operator-director-channel"
)

const (
	operatorDirectorMailboxFileNameV0 = "operator_director_mailbox_v0.jsonl"
	operatorDirectorMailboxStatusV0   = "queued"
	operatorDirectorMailboxSummaryV0  = "mensaje entregado al buzon durable; el destinatario lo leera de su bandeja"
)

// operatorDirectorMailboxDispatcherV0 entrega mensajes cuando no hay conector
// MCP remoto del operador. En vez de fallar con port_unavailable, persiste el
// mensaje en una bandeja durable (JSONL bajo el state dir) que el destinatario
// lee por su cuenta. No inventa respuesta del destinatario: publica `queued`.
type operatorDirectorMailboxDispatcherV0 struct {
	Path string

	mu sync.Mutex
}

type operatorDirectorMailboxEntryV0 struct {
	SchemaVersion   string   `json:"schema_version"`
	MessageRef      string   `json:"message_ref,omitempty"`
	RequestRef      string   `json:"request_ref"`
	ConversationRef string   `json:"conversation_ref,omitempty"`
	SenderRef       string   `json:"sender_ref,omitempty"`
	TargetRef       string   `json:"target_ref"`
	Intent          string   `json:"intent,omitempty"`
	Body            string   `json:"body"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
	QueuedAt        string   `json:"queued_at"`
}

func newOperatorDirectorMailboxDispatcherV0(stateDir string) *operatorDirectorMailboxDispatcherV0 {
	stateDir = strings.TrimSpace(stateDir)
	if stateDir == "" {
		return nil
	}
	return &operatorDirectorMailboxDispatcherV0{
		Path: filepath.Join(stateDir, operatorDirectorMailboxFileNameV0),
	}
}

func (dispatcher *operatorDirectorMailboxDispatcherV0) DispatchOperatorMessageV0(
	_ context.Context,
	message channel.OperatorMessageV0,
) (channel.OperatorDirectorResponseV0, error) {
	if dispatcher == nil || strings.TrimSpace(dispatcher.Path) == "" {
		return channel.OperatorDirectorResponseV0{}, channel.ErrOperatorDirectorDispatchPortUnavailableV0
	}
	messageRef := strings.TrimSpace(message.MessageRef)
	if messageRef == "" {
		messageRef = "message-ref-" + strings.TrimSpace(message.RequestRef)
	}
	entry := operatorDirectorMailboxEntryV0{
		SchemaVersion:   "orquesta_operator_director_mailbox.v0",
		MessageRef:      messageRef,
		RequestRef:      strings.TrimSpace(message.RequestRef),
		ConversationRef: strings.TrimSpace(message.ConversationRef),
		SenderRef:       strings.TrimSpace(message.SenderRef),
		TargetRef:       strings.TrimSpace(message.TargetRef),
		Intent:          strings.TrimSpace(message.Intent),
		Body:            message.Body,
		EvidenceRefs:    message.EvidenceRefs,
		QueuedAt:        time.Now().UTC().Format(time.RFC3339),
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return channel.OperatorDirectorResponseV0{}, fmt.Errorf("operator_director_mailbox_encode_failed: %w", err)
	}

	dispatcher.mu.Lock()
	defer dispatcher.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(dispatcher.Path), 0o700); err != nil {
		return channel.OperatorDirectorResponseV0{}, fmt.Errorf("operator_director_mailbox_dir_failed: %w", err)
	}
	file, err := os.OpenFile(dispatcher.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return channel.OperatorDirectorResponseV0{}, fmt.Errorf("operator_director_mailbox_open_failed: %w", err)
	}
	defer func() { _ = file.Close() }()
	if _, err := file.Write(append(raw, '\n')); err != nil {
		return channel.OperatorDirectorResponseV0{}, fmt.Errorf("operator_director_mailbox_write_failed: %w", err)
	}

	return channel.OperatorDirectorResponseV0{
		MessageRef:   messageRef,
		Status:       operatorDirectorMailboxStatusV0,
		Summary:      operatorDirectorMailboxSummaryV0,
		ResponseRef:  "mailbox-ref-" + messageRef,
		EvidenceRefs: []string{"evidence-ref-operator-director-mailbox-queued"},
	}, nil
}
