// Package staticcapacity publica una observación de capacidad inmutable
// configurada por la composición.
package staticcapacity

import (
	"context"
	"errors"

	"orquesta/internal/application"
)

var (
	ErrInvalidConfig = errors.New("staticcapacity.invalid_config")
	ErrScopeMismatch = errors.New("staticcapacity.scope_mismatch")
)

// Config contiene una observación inmutable suministrada por la composición.
type Config struct {
	Observation application.AgentCapacityObservation
}

type Observer struct {
	observation application.AgentCapacityObservation
}

var _ application.AgentCapacityObserver = Observer{}

func New(config Config) (Observer, error) {
	observation := config.Observation
	if application.ValidateAgentCapacityObservation(observation) != nil ||
		observation.Resources.Messages.Applicability != application.AgentCapacityApplicabilityNotApplicable ||
		observation.Resources.Credits.Applicability != application.AgentCapacityApplicabilityNotApplicable {
		return Observer{}, ErrInvalidConfig
	}
	return Observer{observation: observation}, nil
}

func (observer Observer) ObserveCapacity(
	ctx context.Context,
	source application.AgentCapacitySourceRef,
	pool application.AgentCapacityPoolRef,
) (application.AgentCapacityObservation, error) {
	if ctx == nil {
		return application.AgentCapacityObservation{}, ErrInvalidConfig
	}
	if err := ctx.Err(); err != nil {
		return application.AgentCapacityObservation{}, err
	}
	if source != observer.observation.SourceRef || pool != observer.observation.PoolRef {
		return application.AgentCapacityObservation{}, ErrScopeMismatch
	}
	return observer.observation, nil
}
