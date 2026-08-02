// Package staticcapacity publica capacidad bruta renovable configurada por la composición.
package staticcapacity

import (
	"context"
	"errors"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/governance"
)

var (
	ErrInvalidConfig = errors.New("staticcapacity.invalid_config")
	ErrScopeMismatch = errors.New("staticcapacity.scope_mismatch")
)

type Config struct {
	ReferenciaFuente application.AgentCapacitySourceRef
	ReferenciaPool   application.AgentCapacityPoolRef
	Plazas           int64
	Vigencia         time.Duration
	Ahora            func() time.Time
}

type Observer struct{ config Config }

var _ application.AgentCapacityObserver = Observer{}

func New(config Config) (Observer, error) {
	if config.ReferenciaFuente == "" || config.ReferenciaPool == "" || config.Plazas < 0 ||
		config.Vigencia <= 0 || config.Ahora == nil || config.Ahora().IsZero() {
		return Observer{}, ErrInvalidConfig
	}
	observador := Observer{config: config}
	muestra, err := observador.ObserveCapacity(context.Background(), config.ReferenciaFuente, config.ReferenciaPool)
	if err != nil || application.ValidateAgentCapacityObservation(muestra) != nil {
		return Observer{}, ErrInvalidConfig
	}
	return observador, nil
}

func (observer Observer) ObserveCapacity(ctx context.Context, source application.AgentCapacitySourceRef, pool application.AgentCapacityPoolRef) (application.AgentCapacityObservation, error) {
	if ctx == nil || observer.config.Ahora == nil {
		return application.AgentCapacityObservation{}, ErrInvalidConfig
	}
	if err := ctx.Err(); err != nil {
		return application.AgentCapacityObservation{}, err
	}
	if source != observer.config.ReferenciaFuente || pool != observer.config.ReferenciaPool {
		return application.AgentCapacityObservation{}, ErrScopeMismatch
	}
	ahora := observer.config.Ahora().Round(0).UTC()
	if ahora.IsZero() {
		return application.AgentCapacityObservation{}, ErrInvalidConfig
	}
	noAplicable := application.AgentCapacityDimension{Applicability: application.AgentCapacityApplicabilityNotApplicable}
	plazas := application.AgentCapacityAmount{Present: true, Value: observer.config.Plazas}
	return application.AgentCapacityObservation{
		SourceRef: source, PoolRef: pool, WindowRef: "capacity-window:static",
		Status: application.AgentCapacityAvailable, Quality: governance.UsageQualityExact,
		ObservedAt: ahora, ExpiresAt: ahora.Add(observer.config.Vigencia),
		Resources: application.AgentCapacityResources{
			Slots:   application.AgentCapacityDimension{Applicability: application.AgentCapacityApplicabilityApplicable, Limit: plazas, Remaining: plazas},
			Seconds: noAplicable, Messages: noAplicable, Tokens: noAplicable, Credits: noAplicable,
		},
	}, nil
}
