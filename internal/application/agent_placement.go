// agent_placement define observaciones de cuota no reservable y su compuerta.
package application

import (
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

type AgentQuotaWindowRef string
type AgentQuotaObservationStatus string

const (
	AgentQuotaAvailable AgentQuotaObservationStatus = "available"
	AgentQuotaExhausted AgentQuotaObservationStatus = "exhausted"
	AgentQuotaUnknown   AgentQuotaObservationStatus = "unknown"
)

type AgentQuotaObservation struct {
	PlacementRef                            ports.AgentPlacementRef
	WindowRef                               AgentQuotaWindowRef
	Status                                  AgentQuotaObservationStatus
	Quality                                 governance.UsageQuality
	ObservedAt, ExpiresAt, ResetAt, RetryAt time.Time
	EvidenceRef                             goal.ArtifactRef
}

type AgentQuotaObservationSubmission struct {
	AgentQuotaObservation
	Ref, IdempotencyKey string
}

type AgentQuotaObservationRecord struct {
	AgentQuotaObservationSubmission
	ExpectedRevision, Revision uint64
}

func MaterializeAgentQuotaObservation(submission AgentQuotaObservationSubmission, expected uint64) (AgentQuotaObservationRecord, error) {
	if expected == ^uint64(0) || !validAgentCapacityRef(submission.Ref) ||
		!validAgentCapacityRef(submission.IdempotencyKey) ||
		!validAgentQuotaObservation(submission.AgentQuotaObservation) {
		return AgentQuotaObservationRecord{}, ErrAgentCapacityInvalid
	}
	return AgentQuotaObservationRecord{submission, expected, expected + 1}, nil
}

func DecideAgentQuotaGate(now time.Time, record *AgentQuotaObservationRecord) (AgentCapacityAdmissionReason, error) {
	if record == nil {
		return AgentCapacityAdmissionUnknown, nil
	}
	if now.IsZero() || !validAgentQuotaObservationRecord(*record) || now.Before(record.ObservedAt) {
		return "", ErrAgentCapacityInvalid
	}
	if !now.Before(record.ExpiresAt) {
		return AgentCapacityAdmissionStale, nil
	}
	if record.Quality == governance.UsageQualityUnknown || record.Status == AgentQuotaUnknown {
		return AgentCapacityAdmissionUnknown, nil
	}
	if record.Status == AgentQuotaExhausted {
		return AgentCapacityAdmissionExhausted, nil
	}
	return AgentCapacityAdmissionAvailable, nil
}

func validAgentQuotaObservationRecord(record AgentQuotaObservationRecord) bool {
	return record.ExpectedRevision != ^uint64(0) && record.Revision == record.ExpectedRevision+1 &&
		validAgentCapacityRef(record.Ref) && validAgentCapacityRef(record.IdempotencyKey) &&
		validAgentQuotaObservation(record.AgentQuotaObservation)
}

func validAgentQuotaObservation(observation AgentQuotaObservation) bool {
	validStatus := observation.Status == AgentQuotaAvailable ||
		observation.Status == AgentQuotaExhausted || observation.Status == AgentQuotaUnknown
	validTimes := (observation.ResetAt.IsZero() || observation.ResetAt.After(observation.ObservedAt)) &&
		(observation.RetryAt.IsZero() || observation.RetryAt.After(observation.ObservedAt))
	return observation.PlacementRef.String() != "" && validAgentCapacityRef(string(observation.WindowRef)) &&
		validStatus && validAgentCapacityQuality(observation.Quality) &&
		!observation.ObservedAt.IsZero() && observation.ExpiresAt.After(observation.ObservedAt) && validTimes
}

type AgentPlacementObservationPresentation struct {
	ObservationRef      string
	ObservationRevision uint64
}
type AgentCapacityPlacementCandidate struct {
	PlacementRef    ports.AgentPlacementRef
	Physical, Quota AgentPlacementObservationPresentation
}

func ValidateAgentCapacityPlacementCandidate(candidate AgentCapacityPlacementCandidate) error {
	if candidate.PlacementRef.String() == "" ||
		!validAgentCapacityRef(candidate.Physical.ObservationRef) ||
		!validAgentCapacityRef(candidate.Quota.ObservationRef) ||
		candidate.Physical.ObservationRef == candidate.Quota.ObservationRef ||
		candidate.Physical.ObservationRevision == 0 || candidate.Quota.ObservationRevision == 0 {
		return ErrAgentCapacityInvalid
	}
	return nil
}

type AgentPlacementBinding struct {
	placementRef                        ports.AgentPlacementRef
	reservationRef, quotaObservationRef string
	quotaObservationRevision            uint64
}

func NewAgentPlacementBinding(candidate AgentCapacityPlacementCandidate, reservation AgentCapacityReservation, quota AgentQuotaObservationRecord) (AgentPlacementBinding, error) {
	if ValidateAgentCapacityPlacementCandidate(candidate) != nil ||
		ValidateAgentCapacityReservation(reservation) != nil || !validAgentQuotaObservationRecord(quota) ||
		candidate.Physical.ObservationRef != reservation.ObservationRef ||
		candidate.Physical.ObservationRevision != reservation.ObservationRevision ||
		candidate.Quota.ObservationRef != quota.Ref || candidate.Quota.ObservationRevision != quota.Revision ||
		candidate.PlacementRef != quota.PlacementRef {
		return AgentPlacementBinding{}, ErrAgentCapacityInvalid
	}
	return AgentPlacementBinding{candidate.PlacementRef, reservation.Ref, quota.Ref, quota.Revision}, nil
}

func (binding AgentPlacementBinding) PlacementRef() ports.AgentPlacementRef {
	return binding.placementRef
}
func (binding AgentPlacementBinding) ReservationRef() string { return binding.reservationRef }
func (binding AgentPlacementBinding) QuotaObservation() (string, uint64) {
	return binding.quotaObservationRef, binding.quotaObservationRevision
}
