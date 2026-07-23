package commands

import "orquesta/internal/ports"

var ErrAuditConflict = ports.ErrCommandAuditConflict

// AuditPort is consumed by the command application boundary. Implementations
// must persist immutable admission and outcome facts and enforce exact replay.
// Audit identity is scoped by principal_ref + project_ref + command_id/version
// + request_ref. Different scopes may coexist. Reusing one identity with any
// different input, execution or schema digest must return ErrAuditConflict.
// Begin and Complete derive timestamps and Created atomically from store state.
type AuditPort = ports.CommandAuditStore
type AuditSession = ports.CommandAuditSession
type CommandAuditTerminal = ports.CommandAuditTerminal
type AuditCompletionRequest = ports.CommandAuditCompletionRequest
type AuditCompletion = ports.CommandAuditCompletion
