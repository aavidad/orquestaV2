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

type AgentQuotaObservationRecord struct {
	Ref                                     string
	WindowRef                               AgentQuotaWindowRef
	Revision                                uint64
	Status                                  AgentQuotaObservationStatus
	Quality                                 governance.UsageQuality
	ObservedAt, ExpiresAt, ResetAt, RetryAt time.Time
	EvidenceRef                             goal.ArtifactRef
}

func DecideAgentQuotaGate(now time.Time, record *AgentQuotaObservationRecord) (AgentCapacityAdmissionReason, error) {
	if record == nil {
		return AgentCapacityAdmissionUnknown, nil
	}
	if now.IsZero() || !validAgentQuotaObservation(*record) || now.Before(record.ObservedAt) {
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

func validAgentQuotaObservation(record AgentQuotaObservationRecord) bool {
	validStatus := record.Status == AgentQuotaAvailable ||
		record.Status == AgentQuotaExhausted || record.Status == AgentQuotaUnknown
	validTimes := (record.ResetAt.IsZero() || record.ResetAt.After(record.ObservedAt)) &&
		(record.RetryAt.IsZero() || record.RetryAt.After(record.ObservedAt))
	return validAgentCapacityRef(record.Ref) && validAgentCapacityRef(string(record.WindowRef)) &&
		record.Revision > 0 && validStatus && validAgentCapacityQuality(record.Quality) &&
		!record.ObservedAt.IsZero() && record.ExpiresAt.After(record.ObservedAt) && validTimes
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
		ValidateAgentCapacityReservation(reservation) != nil || !validAgentQuotaObservation(quota) ||
		candidate.Physical.ObservationRef != reservation.ObservationRef ||
		candidate.Physical.ObservationRevision != reservation.ObservationRevision ||
		candidate.Quota.ObservationRef != quota.Ref || candidate.Quota.ObservationRevision != quota.Revision {
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
