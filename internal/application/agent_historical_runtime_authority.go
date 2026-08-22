package application

import (
	"context"
	"errors"

	"orquesta/internal/ports"
)

var (
	ErrAgentHistoricalRuntimeAuthorityNotFound  = errors.New("application.agent_historical_runtime_authority_not_found")
	ErrAgentHistoricalRuntimeAuthorityAmbiguous = errors.New("application.agent_historical_runtime_authority_ambiguous")
	ErrAgentHistoricalRuntimeAuthorityMismatch  = errors.New("application.agent_historical_runtime_authority_mismatch")
)

// AgentHistoricalRuntimeAuthorityResolver is the consumer-owned historical
// lookup boundary. The request carries only the immutable causal key; callers
// cannot supply digests or receipt references that could steer the projection.
type AgentHistoricalRuntimeAuthorityResolver interface {
	ResolveAgentHistoricalRuntimeAuthority(context.Context, ports.AgentHistoricalRuntimeAuthorityKey) (ports.AgentHistoricalRuntimeAuthority, error)
}

// ResolveAgentHistoricalRuntimeAuthority resolves one immutable launch fact
// and verifies that the adapter did not substitute another execution or fence.
func ResolveAgentHistoricalRuntimeAuthority(
	ctx context.Context,
	resolver AgentHistoricalRuntimeAuthorityResolver,
	key ports.AgentHistoricalRuntimeAuthorityKey,
) (ports.AgentHistoricalRuntimeAuthority, error) {
	if ctx == nil || resolver == nil || ports.ValidateAgentHistoricalRuntimeAuthorityKey(key) != nil {
		return ports.AgentHistoricalRuntimeAuthority{}, ErrAgentHistoricalRuntimeAuthorityMismatch
	}
	authority, err := resolver.ResolveAgentHistoricalRuntimeAuthority(ctx, key)
	if err != nil {
		return ports.AgentHistoricalRuntimeAuthority{}, err
	}
	if authority.Key != key || ports.ValidateAgentHistoricalRuntimeAuthority(authority) != nil {
		return ports.AgentHistoricalRuntimeAuthority{}, ErrAgentHistoricalRuntimeAuthorityMismatch
	}
	return authority, nil
}
