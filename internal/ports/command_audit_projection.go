package ports

import (
	"context"
	"errors"
)

var ErrCommandAuditProjectionNotFound = errors.New("command_audit.projection_not_found")

type CommandGovernanceFactKind string

const (
	CommandGovernanceAuthorization    CommandGovernanceFactKind = "authorization_receipt"
	CommandGovernanceDirectorDecision CommandGovernanceFactKind = "director_decision"
	CommandGovernanceEffectDecision   CommandGovernanceFactKind = "effect_decision"
	CommandGovernanceClosureControl   CommandGovernanceFactKind = "closure_control"
	CommandGovernanceCouncilRound     CommandGovernanceFactKind = "council_round"
	CommandGovernanceCouncilSkip      CommandGovernanceFactKind = "council_skip"
	CommandGovernanceEffectIntent     CommandGovernanceFactKind = "effect_intent"
	CommandGovernanceEffectAttempt    CommandGovernanceFactKind = "effect_attempt"
	CommandGovernanceEffectReceipt    CommandGovernanceFactKind = "effect_receipt"
	CommandGovernanceCausalEvent      CommandGovernanceFactKind = "causal_event"
	CommandGovernanceTerminalEvent    CommandGovernanceFactKind = "terminal_event"
	CommandGovernanceAttestation      CommandGovernanceFactKind = "terminal_attestation"
)

type CommandGovernanceFact struct {
	Kind      CommandGovernanceFactKind
	Ref       string
	CausalRef string // exact traversal link; it does not assert temporal direction
}

// CommandGovernanceProjection joins one immutable outcome to existing causal
// facts. It is read-only and never copies their authority into command audit.
type CommandGovernanceProjection struct {
	InvocationRef           string
	OutcomeRef              string
	CommandID               string
	RequestRef              string
	PrincipalRef            string
	ProjectRef              string
	OutcomeStatus           string
	ErrorCode               string
	AuthorizationReceiptRef string
	FactKind                CommandGovernanceFactKind
	FactRef                 string
	Facts                   []CommandGovernanceFact
}

type CommandGovernanceAuditReader interface {
	ProjectCommandGovernance(context.Context, string) (CommandGovernanceProjection, error)
}
