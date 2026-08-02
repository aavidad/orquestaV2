package application

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"orquesta/internal/goal"
)

type AgentCapacityObservationSubmission struct {
	Ref, IdempotencyKey string
	Observation         AgentCapacityObservation
}

type AgentCapacityObservationRecord struct {
	AgentCapacityObservationSubmission
	ExpectedRevision, Revision uint64
}

func NuevaEntregaObservacionCapacidad(observacion AgentCapacityObservation) (AgentCapacityObservationSubmission, error) {
	if ValidateAgentCapacityObservation(observacion) != nil {
		return AgentCapacityObservationSubmission{}, ErrAgentCapacityInvalid
	}
	observacion.ObservedAt, observacion.ExpiresAt = observacion.ObservedAt.Round(0).UTC(), observacion.ExpiresAt.Round(0).UTC()
	observacion.ResetAt, observacion.RetryAt = observacion.ResetAt.Round(0).UTC(), observacion.RetryAt.Round(0).UTC()
	identidad := []any{string(observacion.SourceRef), string(observacion.PoolRef), string(observacion.WindowRef),
		string(observacion.Status), string(observacion.Quality), observacion.ObservedAt.UnixNano(),
		observacion.ExpiresAt.UnixNano(), observacion.ResetAt.UnixNano(), observacion.RetryAt.UnixNano(),
		observacion.Resources, observacion.ArtifactRef.String()}
	contenido, err := json.Marshal(identidad)
	if err != nil {
		return AgentCapacityObservationSubmission{}, ErrAgentCapacityInvalid
	}
	digest := sha256.Sum256(append([]byte("orquesta.agent-capacity-observation.v1\x00"), contenido...))
	referencia := fmt.Sprintf("capacity-observation:v1:sha256:%x", digest)
	return AgentCapacityObservationSubmission{Ref: referencia, IdempotencyKey: referencia, Observation: observacion}, nil
}

type AgentCapacityReservationState string

const AgentCapacityReserved, AgentCapacityConsumed, AgentCapacityReleased, AgentCapacityQuarantined AgentCapacityReservationState = "reserved", "consumed", "released", "quarantined"

type AgentCapacityAllocation struct{ Slots, Seconds, Messages, Tokens, Credits int64 }

type AgentCapacityReservation struct {
	Ref, ObservationRef, ActionRef, EffectIntentRef string
	IdempotencyKey, LastTransitionRef, LastCauseRef string
	ObservationRevision                             uint64
	ProjectRef                                      goal.ProjectRef
	GoalRef                                         goal.GoalRef
	WorkItemRef                                     goal.WorkItemRef
	ExecutionRef                                    goal.ExecutionRef
	PlanGeneration                                  goal.PlanGeneration
	WorkItemGeneration                              goal.Revision
	Revision, Fence                                 uint64
	Allocation                                      AgentCapacityAllocation
	State                                           AgentCapacityReservationState
	ReservedAt, UpdatedAt, SettledAt                time.Time
}

type AgentCapacityTransitionCause string

const AgentCapacityCauseEffectReceipt, AgentCapacityCauseDefinitelyNotApplied, AgentCapacityCauseExecutionTerminal, AgentCapacityCauseUnknownApplied, AgentCapacityCauseReconciliation AgentCapacityTransitionCause = "effect_receipt", "definitely_not_applied", "execution_terminal", "unknown_applied", "reconciliation"

type AgentCapacityTransition struct {
	Ref, ReservationRef, CauseRef, EffectAttemptRef string
	EffectReceiptRef, IdempotencyKey                string
	ProjectRef                                      goal.ProjectRef
	Fence, ExpectedRevision, Revision               uint64
	Outcome                                         AgentCapacityReservationState
	Cause                                           AgentCapacityTransitionCause
	RecordedAt                                      time.Time
}

func MaterializeAgentCapacityObservation(submission AgentCapacityObservationSubmission, expected uint64) (AgentCapacityObservationRecord, error) {
	if expected == ^uint64(0) || !validAgentCapacityRef(submission.Ref) ||
		!validAgentCapacityRef(submission.IdempotencyKey) ||
		ValidateAgentCapacityObservation(submission.Observation) != nil {
		return AgentCapacityObservationRecord{}, ErrAgentCapacityInvalid
	}
	return AgentCapacityObservationRecord{AgentCapacityObservationSubmission: submission, ExpectedRevision: expected, Revision: expected + 1}, nil
}

func ValidateAgentCapacityReservation(record AgentCapacityReservation) error {
	minimum := min(record.Allocation.Seconds, record.Allocation.Messages, record.Allocation.Tokens, record.Allocation.Credits)
	if !validAgentCapacityReservationRefs(record) || record.ObservationRevision == 0 || record.Revision == 0 ||
		record.PlanGeneration == 0 || record.WorkItemGeneration == 0 ||
		record.Fence == 0 || record.Allocation.Slots <= 0 || minimum < 0 || record.ReservedAt.IsZero() {
		return ErrAgentCapacityInvalid
	}
	initial := record.Revision == 1 && record.State == AgentCapacityReserved &&
		record.LastTransitionRef == "" && record.LastCauseRef == "" &&
		record.UpdatedAt.Equal(record.ReservedAt) && record.SettledAt.IsZero()
	transitioned := record.Revision > 1 &&
		(record.State == AgentCapacityConsumed || record.State == AgentCapacityReleased || record.State == AgentCapacityQuarantined) &&
		validAgentCapacityRef(record.LastTransitionRef) && validAgentCapacityRef(record.LastCauseRef) &&
		!record.UpdatedAt.Before(record.ReservedAt) &&
		(record.State == AgentCapacityReleased && record.SettledAt.Equal(record.UpdatedAt) ||
			record.State != AgentCapacityReleased && record.SettledAt.IsZero())
	if !initial && !transitioned {
		return ErrAgentCapacityInvalid
	}
	return nil
}

func ValidateAgentCapacityTransition(current AgentCapacityReservation, next AgentCapacityTransition) error {
	if ValidateAgentCapacityReservation(current) != nil ||
		!validAgentCapacityRef(next.Ref) || next.ReservationRef != current.Ref ||
		next.ProjectRef != current.ProjectRef || next.Fence != current.Fence ||
		next.ExpectedRevision != current.Revision || next.Revision != current.Revision+1 ||
		!validAgentCapacityRef(next.CauseRef) || !validAgentCapacityRef(next.IdempotencyKey) ||
		next.RecordedAt.Before(current.UpdatedAt) || next.RecordedAt.IsZero() ||
		next.EffectAttemptRef != "" && !validApplicationRef(next.EffectAttemptRef) ||
		next.Outcome == AgentCapacityConsumed && (!validApplicationRef(next.EffectAttemptRef) ||
			!validApplicationRef(next.EffectReceiptRef)) ||
		next.Outcome != AgentCapacityConsumed && next.EffectReceiptRef != "" ||
		!validAgentCapacityTransition(current.State, next) {
		return ErrAgentCapacityInvalid
	}
	return nil
}

func validAgentCapacityReservationRefs(record AgentCapacityReservation) bool {
	return validAgentCapacityRef(record.Ref) && validAgentCapacityRef(record.ObservationRef) && validApplicationRef(record.ProjectRef.String()) && validApplicationRef(record.GoalRef.String()) && validApplicationRef(record.WorkItemRef.String()) && validApplicationRef(record.ExecutionRef.String()) && validApplicationRef(record.ActionRef) && validApplicationRef(record.EffectIntentRef) && validApplicationRef(record.IdempotencyKey)
}

func validAgentCapacityTransition(from AgentCapacityReservationState, next AgentCapacityTransition) bool {
	return from == AgentCapacityReserved && next.Outcome == AgentCapacityConsumed && next.Cause == AgentCapacityCauseEffectReceipt || from == AgentCapacityReserved && next.Outcome == AgentCapacityQuarantined && next.Cause == AgentCapacityCauseUnknownApplied ||
		from == AgentCapacityReserved && next.Outcome == AgentCapacityReleased && next.Cause == AgentCapacityCauseDefinitelyNotApplied ||
		from == AgentCapacityQuarantined && next.Outcome == AgentCapacityConsumed && (next.Cause == AgentCapacityCauseEffectReceipt || next.Cause == AgentCapacityCauseReconciliation) ||
		from == AgentCapacityQuarantined && next.Outcome == AgentCapacityReleased && next.Cause == AgentCapacityCauseReconciliation ||
		from == AgentCapacityConsumed && next.Outcome == AgentCapacityReleased && (next.Cause == AgentCapacityCauseExecutionTerminal || next.Cause == AgentCapacityCauseReconciliation)
}
