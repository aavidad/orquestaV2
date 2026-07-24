// Package commands exposes the single public command registry and dispatcher.
// It translates transport-neutral invocations into existing application use
// cases; it never owns Goal lifecycle or durable state.
package commands

import (
	"context"
	"encoding/json"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

type Kind string
type Audience string
type ReplayMode = ports.CommandReplayMode

const (
	KindCommand Kind = "command"
	KindQuery   Kind = "query"

	AudiencePrincipal Audience = "principal"
	AudienceExecution Audience = "execution"

	ReplayApplicationReceipt ReplayMode = ports.CommandReplayApplicationReceipt
	ReplayReadReexecute      ReplayMode = ports.CommandReplayReadReexecute
)

type HTTPBinding struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type MCPBinding struct {
	Tool string `json:"tool"`
}

type CLIBinding struct {
	Path []string `json:"path"`
}

type Definition struct {
	ID             string          `json:"id"`
	Version        string          `json:"version"`
	Kind           Kind            `json:"kind"`
	Handler        string          `json:"handler"`
	Permission     string          `json:"permission"`
	Audience       Audience        `json:"audience"`
	ExecutionBound bool            `json:"execution_bound"`
	ReplayMode     ReplayMode      `json:"replay_mode"`
	InputSchema    json.RawMessage `json:"input_schema"`
	OutputSchema   json.RawMessage `json:"output_schema"`
	DescriptionKey string          `json:"description_key"`
	ErrorCodes     []string        `json:"error_codes"`
	HTTP           HTTPBinding     `json:"http"`
	MCP            MCPBinding      `json:"mcp"`
	CLI            CLIBinding      `json:"cli"`
}

// ExecutionAuthorityResolver is the single trusted boundary that can bind one
// authenticated principal and project to one exact execution claim. The
// dispatcher, never a transport or caller-provided Invocation field, consumes
// this authority.
type ExecutionAuthorityResolver interface {
	ResolveExecution(
		context.Context,
		identity.Principal,
		goal.ProjectRef,
		goal.ExecutionRef,
	) (goal.ExecutionRef, error)
}

// ExecutionPrincipalClassifier distinguishes a generic service principal from
// one derived from a durable execution tuple. Found remains true after the
// execution becomes terminal, superseded, or is observed through another
// project scope, so it can never fall back to project-membership RBAC.
type ExecutionPrincipalClassifier interface {
	ClassifyExecutionPrincipal(
		context.Context,
		identity.Principal,
		goal.ProjectRef,
	) (found bool, active bool, executionRef goal.ExecutionRef, err error)
}

// Invocation contains authenticated principal identity separately from the
// payload. ClaimedExecutionRef remains untrusted input until the dispatcher
// resolves it through ExecutionAuthorityResolver.
type Invocation struct {
	CommandID           string
	CommandVersion      string
	RequestRef          string
	ProjectRef          string
	ClaimedExecutionRef string
	Principal           identity.Principal
	Payload             json.RawMessage
}

type Result struct {
	CommandID      string          `json:"command_id"`
	CommandVersion string          `json:"command_version"`
	RequestRef     string          `json:"request_ref"`
	Data           json.RawMessage `json:"data,omitempty"`
	Failure        *Failure        `json:"failure,omitempty"`
	AuditRef       string          `json:"audit_ref,omitempty"`
}

type Failure struct {
	Code       string `json:"code"`
	MessageKey string `json:"message_key"`
}

type CommandAuditRecord = ports.CommandAuditRecord

type APILimits struct {
	MaxRequestBytes int64
	MaxListLimit    int
}

func (limits APILimits) Valid() bool { return limits.MaxRequestBytes > 0 && limits.MaxListLimit > 0 }
