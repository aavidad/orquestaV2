package application

import (
	"errors"

	"orquesta/internal/ports"
)

var (
	ErrAgentHistoricalRuntimeAuthorityNotFound  = errors.New("application.agent_historical_runtime_authority_not_found")
	ErrAgentHistoricalRuntimeAuthorityAmbiguous = errors.New("application.agent_historical_runtime_authority_ambiguous")
	ErrAgentHistoricalRuntimeAuthorityMismatch  = errors.New("application.agent_historical_runtime_authority_mismatch")
)

// ResolveAgentHistoricalRuntimeAuthority accepts a read result, not a store.
// Persistence will supply the history in the next causal task. Exactly one
// record must own the requested ExecutionRef+historical launch fence; even
// duplicate identical records are corruption rather than an implicit replay.
func ResolveAgentHistoricalRuntimeAuthority(
	history []ports.AgentHistoricalRuntimeAuthority,
	expected ports.AgentHistoricalRuntimeAuthority,
) (ports.AgentHistoricalRuntimeAuthority, error) {
	if ports.ValidateAgentHistoricalRuntimeAuthority(expected) != nil {
		return ports.AgentHistoricalRuntimeAuthority{}, ErrAgentHistoricalRuntimeAuthorityMismatch
	}
	matches := make([]ports.AgentHistoricalRuntimeAuthority, 0, 1)
	for _, candidate := range history {
		if candidate.Key != expected.Key {
			continue
		}
		if ports.ValidateAgentHistoricalRuntimeAuthority(candidate) != nil {
			return ports.AgentHistoricalRuntimeAuthority{}, ErrAgentHistoricalRuntimeAuthorityMismatch
		}
		matches = append(matches, candidate)
	}
	if len(matches) == 0 {
		return ports.AgentHistoricalRuntimeAuthority{}, ErrAgentHistoricalRuntimeAuthorityNotFound
	}
	if len(matches) != 1 {
		return ports.AgentHistoricalRuntimeAuthority{}, ErrAgentHistoricalRuntimeAuthorityAmbiguous
	}
	if matches[0] != expected {
		return ports.AgentHistoricalRuntimeAuthority{}, ErrAgentHistoricalRuntimeAuthorityMismatch
	}
	return matches[0], nil
}
