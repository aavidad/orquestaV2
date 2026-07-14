package application

import (
	"context"
	"errors"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

// AgentLauncher is the outbound launch port consumed by the orchestrator.
type AgentLauncher interface {
	Capabilities(context.Context) (ports.AgentCapabilities, error)
	// Launch is causally idempotent for an equal execution, key and request.
	Launch(context.Context, ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error)
}

// temporaryAgentError is implemented structurally by adapters when a launch
// can be retried safely. The application owns retry timing and durable queues.
type temporaryAgentError interface {
	error
	Temporary() bool
}

func isTemporaryAgentError(err error) bool {
	var temporary temporaryAgentError
	return errors.As(err, &temporary) && temporary.Temporary()
}

// AgentObserver recovers observations, including terminal state, by execution.
type AgentObserver interface {
	Observe(context.Context, goal.ExecutionRef) (ports.AgentObservation, error)
}

// ArtifactStore persists and reads immutable content-addressed blobs.
type ArtifactStore interface {
	Put(context.Context, ports.PutArtifactRequest) (ports.StoredArtifact, error)
	Get(context.Context, goal.ArtifactRef, int64) (ports.ArtifactContent, error)
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID(context.Context, string) (string, error)
}
