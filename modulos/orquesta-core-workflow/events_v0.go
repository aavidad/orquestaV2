package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	OrchestrationEventPayloadVersionV0 = "orchestration_event_payload.v0"

	OrchestrationEventRunStartedV0                   = "RunStarted"
	OrchestrationEventPhaseOpenedV0                  = "PhaseOpened"
	OrchestrationEventPhaseClosedV0                  = "PhaseClosed"
	OrchestrationEventRunBlockedV0                   = "RunBlocked"
	OrchestrationEventDirectorQuestionRaisedV0       = "DirectorQuestionRaised"
	OrchestrationEventDirectorQuestionAnsweredV0     = "DirectorQuestionAnswered"
	OrchestrationEventBrainstormRequestedV0          = "BrainstormRequested"
	OrchestrationEventVoteRequestedV0                = "VoteRequested"
	OrchestrationEventArchitectureDecisionAcceptedV0 = "ArchitectureDecisionAccepted"
	OrchestrationEventFunctionContractPublishedV0    = "FunctionContractPublished"
	OrchestrationEventMicrotaskCreatedV0             = "MicrotaskCreated"
	OrchestrationEventCapacityRequestedV0            = "CapacityRequested"
	OrchestrationEventCapacityDecidedV0              = "CapacityDecided"
	OrchestrationEventAgentRequestedV0               = "AgentRequested"
	OrchestrationEventAgentStartedV0                 = "AgentStarted"
	OrchestrationEventAgentFailedV0                  = "AgentFailed"
	OrchestrationEventAgentLeaseExpiredV0            = "AgentLeaseExpired"
	OrchestrationEventAgentStopRequestedV0           = "AgentStopRequested"
	OrchestrationEventAgentStopConfirmedV0           = "AgentStopConfirmed"
	OrchestrationEventAgentWorkAssessedV0            = "AgentWorkAssessed"
	OrchestrationEventConcurrencyGateRecordedV0      = "ConcurrencyGateRecorded"
	OrchestrationEventQualityGateRecordedV0          = "QualityGateRecorded"
	OrchestrationEventPhaseArtifactRegisteredV0      = "PhaseArtifactRegistered"
	OrchestrationEventDeliveryRegisteredV0           = "DeliveryRegistered"
	OrchestrationEventReviewRequestedV0              = "ReviewRequested"
	OrchestrationEventReviewAcceptedV0               = "ReviewAccepted"
	OrchestrationEventReviewResultRecordedV0         = "ReviewResultRecorded"
	OrchestrationEventReworkRequestedV0              = "ReworkRequested"
	OrchestrationEventReplanDecisionRecordedV0       = "ReplanDecisionRecorded"
	OrchestrationEventTaskClosedV0                   = "TaskClosed"
	OrchestrationEventFinalValidationRegisteredV0    = "FinalValidationRegistered"
	OrchestrationEventRunClosedV0                    = "RunClosed"

	ErrEventoNoSoportadoV0 = "evento_no_soportado"
	ErrEventoInvalidoV0    = "evento_invalido"
	ErrSecuenciaInvalidaV0 = "secuencia_invalida"
	ErrPayloadInvalidoV0   = "payload_invalido"
	ErrDetalleProhibidoV0  = "detalle_prohibido"
)

type OrchestrationEventV0 struct {
	EventID        string          `json:"event_id"`
	EventType      string          `json:"event_type"`
	RunID          string          `json:"run_id"`
	Sequence       int64           `json:"sequence"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	CorrelationID  string          `json:"correlation_id,omitempty"`
	CausationID    string          `json:"causation_id,omitempty"`
	OccurredAt     string          `json:"occurred_at"`
	PayloadVersion string          `json:"payload_version"`
	Payload        json.RawMessage `json:"payload"`
}

type OrchestrationEventMetaV0 struct {
	EventID        string
	RunID          string
	Sequence       int64
	IdempotencyKey string
	CorrelationID  string
	CausationID    string
	OccurredAt     string
}

type RunStartedPayloadV0 struct {
	ProjectRef  string `json:"project_ref"`
	AppSpecRef  string `json:"app_spec_ref"`
	RequestedBy string `json:"requested_by,omitempty"`
}

type PhaseOpenedPayloadV0 struct {
	PhaseID  string `json:"phase_id"`
	OpenedBy string `json:"opened_by,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

type PhaseClosedPayloadV0 struct {
	PhaseID      string   `json:"phase_id"`
	ClosureRef   string   `json:"closure_ref"`
	Summary      string   `json:"summary"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type RunBlockedPayloadV0 struct {
	BlockerID    string   `json:"blocker_id"`
	ReasonCode   string   `json:"reason_code"`
	Summary      string   `json:"summary"`
	SourceGroup  string   `json:"source_group,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type OrchestrationEventErrorV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func (err OrchestrationEventErrorV0) Error() string {
	return err.Code
}

func NewRunStartedEventV0(meta OrchestrationEventMetaV0, payload RunStartedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventRunStartedV0, payload)
}

func NewPhaseOpenedEventV0(meta OrchestrationEventMetaV0, payload PhaseOpenedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventPhaseOpenedV0, payload)
}

func NewPhaseClosedEventV0(meta OrchestrationEventMetaV0, payload PhaseClosedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventPhaseClosedV0, payload)
}

func NewRunBlockedEventV0(meta OrchestrationEventMetaV0, payload RunBlockedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventRunBlockedV0, payload)
}

func ValidateOrchestrationEventV0(event OrchestrationEventV0) error {
	if strings.TrimSpace(event.EventID) == "" {
		return eventErrorV0(ErrEventoInvalidoV0, "event_id")
	}
	if !isSupportedEventTypeV0(event.EventType) {
		return eventErrorV0(ErrEventoNoSoportadoV0, "event_type")
	}
	if strings.TrimSpace(event.RunID) == "" {
		return eventErrorV0(ErrEventoInvalidoV0, "run_id")
	}
	if event.Sequence < 1 {
		return eventErrorV0(ErrEventoInvalidoV0, "sequence")
	}
	if strings.TrimSpace(event.OccurredAt) == "" {
		return eventErrorV0(ErrEventoInvalidoV0, "occurred_at")
	}
	if event.PayloadVersion != OrchestrationEventPayloadVersionV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload_version")
	}
	if err := validateEventPayloadV0(event); err != nil {
		return err
	}
	return nil
}

func newOrchestrationEventV0(meta OrchestrationEventMetaV0, eventType string, payload any) (OrchestrationEventV0, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return OrchestrationEventV0{}, eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	event := OrchestrationEventV0{
		EventID:        strings.TrimSpace(meta.EventID),
		EventType:      eventType,
		RunID:          strings.TrimSpace(meta.RunID),
		Sequence:       meta.Sequence,
		IdempotencyKey: strings.TrimSpace(meta.IdempotencyKey),
		CorrelationID:  strings.TrimSpace(meta.CorrelationID),
		CausationID:    strings.TrimSpace(meta.CausationID),
		OccurredAt:     strings.TrimSpace(meta.OccurredAt),
		PayloadVersion: OrchestrationEventPayloadVersionV0,
		Payload:        payloadJSON,
	}
	if err := ValidateOrchestrationEventV0(event); err != nil {
		return OrchestrationEventV0{}, err
	}
	return event, nil
}
