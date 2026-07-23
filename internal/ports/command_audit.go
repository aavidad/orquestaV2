package ports

import (
	"context"
	"errors"
	"time"
)

type CommandReplayMode string

const (
	CommandReplayApplicationReceipt CommandReplayMode = "application_receipt"
	CommandReplayReadReexecute      CommandReplayMode = "read_reexecute"
)

var ErrCommandAuditConflict = errors.New("commands.audit_conflict")

// CommandAuditRecord is the transport-neutral immutable admission fact.
type CommandAuditRecord struct {
	Ref                       string            `json:"ref"`
	CommandID                 string            `json:"command_id"`
	CommandVersion            string            `json:"command_version"`
	RegistryDigest            string            `json:"registry_digest"`
	SchemaDigest              string            `json:"schema_digest"`
	RequestRef                string            `json:"request_ref"`
	InputDigest               string            `json:"input_digest"`
	PrincipalRef              string            `json:"principal_ref"`
	ProjectRef                string            `json:"project_ref"`
	AuthenticatedExecutionRef string            `json:"authenticated_execution_ref,omitempty"`
	ReplayMode                CommandReplayMode `json:"replay_mode"`
	Status                    string            `json:"status"`
	ErrorCode                 string            `json:"error_code,omitempty"`
	OutputDigest              string            `json:"output_digest,omitempty"`
	AdmittedAt                time.Time         `json:"admitted_at"`
	CompletedAt               time.Time         `json:"completed_at,omitempty"`
}

type CommandAuditSession struct {
	Record           CommandAuditRecord
	OutcomeRef       string
	AdmissionCreated bool
	Terminal         *CommandAuditTerminal
}

type CommandAuditTerminal struct {
	OutcomeRef   string    `json:"outcome_ref"`
	Status       string    `json:"status"`
	ErrorCode    string    `json:"error_code,omitempty"`
	OutputDigest string    `json:"output_digest"`
	CompletedAt  time.Time `json:"completed_at"`
}

type CommandAuditCompletionRequest struct {
	RecordRef string
	Terminal  CommandAuditTerminal
}

type CommandAuditCompletion struct {
	Terminal CommandAuditTerminal
	Created  bool
}

// CommandAuditStore persists immutable admission and outcome facts. Identity
// is principal_ref + project_ref + command_id/version + request_ref. Exact
// replay returns the stored fact; changed input, execution or schema identity
// returns ErrCommandAuditConflict.
type CommandAuditStore interface {
	Begin(context.Context, CommandAuditRecord) (CommandAuditSession, error)
	Complete(context.Context, CommandAuditCompletionRequest) (CommandAuditCompletion, error)
}
