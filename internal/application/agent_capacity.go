package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
)

var ErrAgentCapacityInvalid = errors.New("application.agent_capacity_invalid")

type AgentCapacitySourceRef string
type AgentCapacityPoolRef string
type AgentCapacityWindowRef string

type AgentCapacityStatus string

const (
	AgentCapacityAvailable   AgentCapacityStatus = "available"
	AgentCapacityUnavailable AgentCapacityStatus = "unavailable"
)

type AgentCapacityAmount struct {
	Present bool
	Value   int64
}

type AgentCapacityApplicability string

const (
	AgentCapacityApplicabilityUnknown       AgentCapacityApplicability = "unknown"
	AgentCapacityApplicabilityNotApplicable AgentCapacityApplicability = "not_applicable"
	AgentCapacityApplicabilityApplicable    AgentCapacityApplicability = "applicable"
)

type AgentCapacityDimension struct {
	Applicability AgentCapacityApplicability
	Limit         AgentCapacityAmount
	Remaining     AgentCapacityAmount
}

type AgentCapacityResources struct {
	Slots    AgentCapacityDimension
	Seconds  AgentCapacityDimension
	Messages AgentCapacityDimension
	Tokens   AgentCapacityDimension
	Credits  AgentCapacityDimension
}

// AgentCapacityDemand makes zero and missing quantities different facts.
type AgentCapacityDemand struct {
	Slots, Seconds, Messages, Tokens, Credits AgentCapacityAmount
}

type AgentCapacityObservation struct {
	SourceRef   AgentCapacitySourceRef
	PoolRef     AgentCapacityPoolRef
	WindowRef   AgentCapacityWindowRef
	Status      AgentCapacityStatus
	Quality     governance.UsageQuality
	ObservedAt  time.Time
	ExpiresAt   time.Time
	ResetAt     time.Time
	RetryAt     time.Time
	Resources   AgentCapacityResources
	ArtifactRef goal.ArtifactRef
}

type AgentCapacityObserver interface {
	ObserveCapacity(context.Context, AgentCapacitySourceRef, AgentCapacityPoolRef) (AgentCapacityObservation, error)
}

type AgentCapacityAdmissionReason string

const (
	AgentCapacityAdmissionAvailable   AgentCapacityAdmissionReason = "available"
	AgentCapacityAdmissionUnknown     AgentCapacityAdmissionReason = "unknown"
	AgentCapacityAdmissionStale       AgentCapacityAdmissionReason = "stale"
	AgentCapacityAdmissionUnavailable AgentCapacityAdmissionReason = "unavailable"
	AgentCapacityAdmissionExhausted   AgentCapacityAdmissionReason = "exhausted"
)

type AgentCapacityAdmission struct {
	WindowRef      AgentCapacityWindowRef
	Reason         AgentCapacityAdmissionReason
	NewAdmissions  int64
	ControlAllowed bool
}

func ValidateAgentCapacityObservation(observation AgentCapacityObservation) error {
	if !validAgentCapacityRef(string(observation.SourceRef)) ||
		!validAgentCapacityRef(string(observation.PoolRef)) ||
		!validAgentCapacityRef(string(observation.WindowRef)) {
		return ErrAgentCapacityInvalid
	}
	if observation.Status != AgentCapacityAvailable &&
		observation.Status != AgentCapacityUnavailable ||
		!validAgentCapacityQuality(observation.Quality) {
		return ErrAgentCapacityInvalid
	}
	if observation.ObservedAt.IsZero() || observation.ExpiresAt.IsZero() ||
		!observation.ExpiresAt.After(observation.ObservedAt) {
		return ErrAgentCapacityInvalid
	}
	if !observation.ResetAt.IsZero() && !observation.ResetAt.After(observation.ObservedAt) ||
		!observation.RetryAt.IsZero() && !observation.RetryAt.After(observation.ObservedAt) {
		return ErrAgentCapacityInvalid
	}
	for _, dimension := range observation.Resources.dimensions() {
		if err := validateAgentCapacityDimension(dimension); err != nil {
			return err
		}
	}
	return nil
}

func DecideAgentCapacityAdmission(
	clock Clock,
	observation AgentCapacityObservation,
	sourceErr error,
	demand AgentCapacityDemand,
	held AgentCapacityDemand,
) (AgentCapacityAdmission, error) {
	decision := AgentCapacityAdmission{ControlAllowed: true}
	if clock == nil {
		return decision, ErrAgentCapacityInvalid
	}
	now := clock.Now()
	if now.IsZero() {
		return decision, ErrAgentCapacityInvalid
	}
	if sourceErr != nil {
		decision.Reason = AgentCapacityAdmissionUnavailable
		return decision, nil
	}
	if err := ValidateAgentCapacityObservation(observation); err != nil {
		return decision, err
	}
	if err := validateAgentCapacityDemand(demand); err != nil {
		return decision, err
	}
	if err := validateAgentCapacityDemand(held); err != nil {
		return decision, err
	}
	decision.WindowRef = observation.WindowRef
	if now.Before(observation.ObservedAt) {
		return decision, ErrAgentCapacityInvalid
	}
	switch {
	case observation.Status == AgentCapacityUnavailable:
		decision.Reason = AgentCapacityAdmissionUnavailable
	case !now.Before(observation.ExpiresAt):
		decision.Reason = AgentCapacityAdmissionStale
	case observation.Quality == governance.UsageQualityUnknown ||
		agentCapacityInputsUnknown(observation.Resources, demand, held):
		decision.Reason = AgentCapacityAdmissionUnknown
	default:
		admissions, err := agentCapacityNewAdmissions(observation.Resources, demand, held)
		if err != nil {
			return decision, err
		}
		if admissions == 0 {
			decision.Reason = AgentCapacityAdmissionExhausted
		} else {
			decision.Reason = AgentCapacityAdmissionAvailable
			decision.NewAdmissions = admissions
		}
	}
	return decision, nil
}

func AgentCapacityDemandFromBudget(demand governance.BudgetDemand) (AgentCapacityDemand, error) {
	if err := governance.ValidateBudgetDemand(demand); err != nil {
		return AgentCapacityDemand{}, ErrAgentCapacityInvalid
	}
	resources := demand.Resources
	seconds := resources.ActiveTimeNS / int64(time.Second)
	if resources.ActiveTimeNS%int64(time.Second) != 0 {
		seconds++
	}
	return AgentCapacityDemand{
		Slots:   AgentCapacityAmount{Present: true, Value: resources.ProcessSlots},
		Seconds: AgentCapacityAmount{Present: true, Value: seconds},
		Tokens:  AgentCapacityAmount{Present: true, Value: resources.Tokens},
	}, nil
}

func validateAgentCapacityDimension(dimension AgentCapacityDimension) error {
	if dimension.Applicability != AgentCapacityApplicabilityUnknown &&
		dimension.Applicability != AgentCapacityApplicabilityNotApplicable &&
		dimension.Applicability != AgentCapacityApplicabilityApplicable {
		return ErrAgentCapacityInvalid
	}
	for _, amount := range []AgentCapacityAmount{dimension.Limit, dimension.Remaining} {
		if !amount.Present && amount.Value != 0 || amount.Value < 0 {
			return ErrAgentCapacityInvalid
		}
	}
	if dimension.Applicability != AgentCapacityApplicabilityApplicable &&
		(dimension.Limit.Present || dimension.Remaining.Present) {
		return ErrAgentCapacityInvalid
	}
	if dimension.Limit.Present && dimension.Remaining.Present &&
		dimension.Remaining.Value > dimension.Limit.Value {
		return ErrAgentCapacityInvalid
	}
	return nil
}

func validateAgentCapacityDemand(demand AgentCapacityDemand) error {
	for _, amount := range demand.amounts() {
		if !amount.Present && amount.Value != 0 || amount.Value < 0 {
			return ErrAgentCapacityInvalid
		}
	}
	return nil
}

func agentCapacityInputsUnknown(
	resources AgentCapacityResources,
	demand AgentCapacityDemand,
	held AgentCapacityDemand,
) bool {
	if resources.Slots.Applicability != AgentCapacityApplicabilityApplicable {
		return true
	}
	dimensions, demands, retained := resources.dimensions(), demand.amounts(), held.amounts()
	for index, dimension := range dimensions {
		if dimension.Applicability == AgentCapacityApplicabilityUnknown ||
			dimension.Applicability == AgentCapacityApplicabilityApplicable &&
				(!dimension.Remaining.Present || !demands[index].Present || !retained[index].Present) {
			return true
		}
	}
	return false
}

func agentCapacityNewAdmissions(
	resources AgentCapacityResources,
	demand AgentCapacityDemand,
	held AgentCapacityDemand,
) (int64, error) {
	dimensions, demands, retained := resources.dimensions(), demand.amounts(), held.amounts()
	var admissions int64
	for index, dimension := range dimensions {
		if dimension.Applicability != AgentCapacityApplicabilityApplicable {
			continue
		}
		if index == 0 && demands[index].Value == 0 {
			return 0, ErrAgentCapacityInvalid
		}
		if demands[index].Value == 0 {
			continue
		}
		available := dimension.Remaining.Value - retained[index].Value
		if available <= 0 {
			return 0, nil
		}
		fit := available / demands[index].Value
		if fit == 0 {
			return 0, nil
		}
		if admissions == 0 || fit < admissions {
			admissions = fit
		}
	}
	return admissions, nil
}

func (resources AgentCapacityResources) dimensions() [5]AgentCapacityDimension {
	return [5]AgentCapacityDimension{
		resources.Slots, resources.Seconds, resources.Messages,
		resources.Tokens, resources.Credits,
	}
}

func (demand AgentCapacityDemand) amounts() [5]AgentCapacityAmount {
	return [5]AgentCapacityAmount{
		demand.Slots, demand.Seconds, demand.Messages, demand.Tokens, demand.Credits,
	}
}

func validAgentCapacityQuality(quality governance.UsageQuality) bool {
	return quality == governance.UsageQualityUnknown ||
		quality == governance.UsageQualityEstimated ||
		quality == governance.UsageQualityMeasured ||
		quality == governance.UsageQualityExact
}

func validAgentCapacityRef(value string) bool {
	return value != "" && strings.TrimSpace(value) == value &&
		!strings.ContainsRune(value, '\x00')
}
