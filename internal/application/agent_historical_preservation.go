package application

import (
	"context"
	"errors"

	"orquesta/internal/ports"
)

var ErrAgentHistoricalPreservationInvalid = errors.New("application.agent_historical_preservation_invalid")

// AgentHistoricalRuntimePreserver is the authority-aware physical boundary.
// The ordinary lifecycle request deliberately does not carry historical
// launch digests or its action fence, so application resolves those facts
// before crossing the adapter.
type AgentHistoricalRuntimePreserver interface {
	PreserveWithAuthority(
		context.Context,
		ports.AgentHistoricalRuntimeAuthority,
		ports.AgentPreserveRequest,
	) (ports.AgentPreserveReceipt, error)
}

// PreserveAgentEnvironmentWithHistoricalAuthority resolves exactly one
// immutable launch authority and never lets caller-supplied lifecycle fields
// substitute its subject or digests.
func PreserveAgentEnvironmentWithHistoricalAuthority(
	ctx context.Context,
	resolver AgentHistoricalRuntimeAuthorityResolver,
	key ports.AgentHistoricalRuntimeAuthorityKey,
	request ports.AgentPreserveRequest,
	preserver AgentHistoricalRuntimePreserver,
) (ports.AgentPreserveReceipt, error) {
	if ctx == nil || preserver == nil || ports.ValidateAgentPreserveRequest(request) != nil {
		return ports.AgentPreserveReceipt{}, ErrAgentHistoricalPreservationInvalid
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentPreserveReceipt{}, err
	}
	authority, err := ResolveAgentHistoricalRuntimeAuthority(ctx, resolver, key)
	if err != nil {
		return ports.AgentPreserveReceipt{}, err
	}
	if authority.Subject != request.Subject {
		return ports.AgentPreserveReceipt{}, ErrAgentHistoricalPreservationInvalid
	}
	return preserver.PreserveWithAuthority(ctx, authority, request)
}
