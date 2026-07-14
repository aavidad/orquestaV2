package application

import (
	"errors"
	"time"
)

const (
	agentArtifactMediaType  = "text/plain"
	outputAttestationPolicy = "agent_output_present"
)

type Dependencies struct {
	State             StateRepository
	Launcher          AgentLauncher
	Observer          AgentObserver
	Artifacts         ArtifactStore
	Clock             Clock
	IDs               IDGenerator
	MaxOutputBytes    int64
	MaxActionAttempts uint64
	ClaimLease        time.Duration
	ObservationDelay  time.Duration
	ExecutionTimeout  time.Duration
}

type Orchestrator struct {
	state             StateRepository
	launcher          AgentLauncher
	observer          AgentObserver
	artifacts         ArtifactStore
	clock             Clock
	ids               IDGenerator
	maxOutputBytes    int64
	maxActionAttempts uint64
	claimLease        time.Duration
	observationDelay  time.Duration
	executionTimeout  time.Duration
}

func New(dependencies Dependencies) (*Orchestrator, error) {
	switch {
	case dependencies.State == nil:
		return nil, errors.New("application.state_required")
	case dependencies.Launcher == nil:
		return nil, errors.New("application.launcher_required")
	case dependencies.Observer == nil:
		return nil, errors.New("application.observer_required")
	case dependencies.Artifacts == nil:
		return nil, errors.New("application.artifacts_required")
	case dependencies.Clock == nil:
		return nil, errors.New("application.clock_required")
	case dependencies.IDs == nil:
		return nil, errors.New("application.ids_required")
	case dependencies.MaxOutputBytes <= 0:
		return nil, errors.New("application.max_output_bytes_invalid")
	case dependencies.MaxActionAttempts == 0:
		return nil, errors.New("application.max_action_attempts_invalid")
	case dependencies.ClaimLease <= 0:
		return nil, errors.New("application.claim_lease_invalid")
	case dependencies.ObservationDelay <= 0:
		return nil, errors.New("application.observation_delay_invalid")
	case dependencies.ExecutionTimeout <= 0:
		return nil, errors.New("application.execution_timeout_invalid")
	}
	return &Orchestrator{
		state:             dependencies.State,
		launcher:          dependencies.Launcher,
		observer:          dependencies.Observer,
		artifacts:         dependencies.Artifacts,
		clock:             dependencies.Clock,
		ids:               dependencies.IDs,
		maxOutputBytes:    dependencies.MaxOutputBytes,
		maxActionAttempts: dependencies.MaxActionAttempts,
		claimLease:        dependencies.ClaimLease,
		observationDelay:  dependencies.ObservationDelay,
		executionTimeout:  dependencies.ExecutionTimeout,
	}, nil
}
