package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	OrchestrationCommandPayloadVersionV0 = "orchestration_command_payload.v0"

	OrchestrationCommandStartRunV0                   = "StartRun"
	OrchestrationCommandOpenPhaseV0                  = "OpenPhase"
	OrchestrationCommandClosePhaseV0                 = "ClosePhase"
	OrchestrationCommandBlockRunV0                   = "BlockRun"
	OrchestrationCommandAskDirectorV0                = "AskDirector"
	OrchestrationCommandAnswerDirectorQuestionV0     = "AnswerDirectorQuestion"
	OrchestrationCommandRequestBrainstormV0          = "RequestBrainstorm"
	OrchestrationCommandRequestVoteV0                = "RequestVote"
	OrchestrationCommandAcceptDecisionV0             = "AcceptDecision"
	OrchestrationCommandPublishFunctionContractV0    = "PublishFunctionContract"
	OrchestrationCommandCreateMicrotaskV0            = "CreateMicrotask"
	OrchestrationCommandRequestCapacityV0            = "RequestCapacity"
	OrchestrationCommandRegisterCapacityDecisionV0   = "RegisterCapacityDecision"
	OrchestrationCommandRequestAgentV0               = "RequestAgent"
	OrchestrationCommandRegisterAgentStartedV0       = "RegisterAgentStarted"
	OrchestrationCommandRegisterAgentFailedV0        = "RegisterAgentFailed"
	OrchestrationCommandRegisterAgentLeaseExpiredV0  = "RegisterAgentLeaseExpired"
	OrchestrationCommandStopAgentV0                  = "StopAgent"
	OrchestrationCommandRegisterAgentStopConfirmedV0 = "RegisterAgentStopConfirmed"
	OrchestrationCommandAssessAgentWorkV0            = "AssessAgentWork"
	OrchestrationCommandRecordConcurrencyGateV0      = "RecordConcurrencyGate"
	OrchestrationCommandRecordQualityGateV0          = "RecordQualityGate"
	OrchestrationCommandRegisterPhaseArtifactV0      = "RegisterPhaseArtifact"
	OrchestrationCommandRegisterDeliveryV0           = "RegisterDelivery"
	OrchestrationCommandRequestReviewV0              = "RequestReview"
	OrchestrationCommandAcceptReviewV0               = "AcceptReview"
	OrchestrationCommandRecordReviewResultV0         = "RecordReviewResult"
	OrchestrationCommandRequestReworkV0              = "RequestRework"
	OrchestrationCommandRecordReplanDecisionV0       = "RecordReplanDecision"
	OrchestrationCommandCloseTaskV0                  = "CloseTask"
	OrchestrationCommandRegisterFinalValidationV0    = "RegisterFinalValidation"
	OrchestrationCommandCloseRunV0                   = "CloseRun"

	ErrComandoNoSoportadoV0      = "comando_no_soportado"
	ErrComandoInvalidoV0         = "comando_invalido"
	ErrIdempotencyKeyRequeridaV0 = "idempotency_key_requerida"
	ErrTransicionInvalidaV0      = "transicion_invalida"
)

type OrchestrationCommandV0 struct {
	CommandID      string          `json:"command_id"`
	CommandType    string          `json:"command_type"`
	RunID          string          `json:"run_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	CorrelationID  string          `json:"correlation_id,omitempty"`
	RequestedBy    string          `json:"requested_by,omitempty"`
	OccurredAt     string          `json:"occurred_at"`
	PayloadVersion string          `json:"payload_version"`
	Payload        json.RawMessage `json:"payload"`
}

type OrchestrationCommandMetaV0 struct {
	CommandID      string
	RunID          string
	IdempotencyKey string
	CorrelationID  string
	RequestedBy    string
	OccurredAt     string
}

type StartRunCommandPayloadV0 struct {
	ProjectRef string `json:"project_ref"`
	AppSpecRef string `json:"app_spec_ref"`
}

type OpenPhaseCommandPayloadV0 struct {
	PhaseID string `json:"phase_id"`
	Reason  string `json:"reason,omitempty"`
}

type ClosePhaseCommandPayloadV0 struct {
	PhaseID      string   `json:"phase_id"`
	ClosureRef   string   `json:"closure_ref"`
	Summary      string   `json:"summary"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type BlockRunCommandPayloadV0 struct {
	BlockerID    string   `json:"blocker_id"`
	ReasonCode   string   `json:"reason_code"`
	Summary      string   `json:"summary"`
	SourceGroup  string   `json:"source_group,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type OrchestrationCommandErrorV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func (err OrchestrationCommandErrorV0) Error() string {
	return err.Code
}

func NewStartRunCommandV0(meta OrchestrationCommandMetaV0, payload StartRunCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandStartRunV0, payload)
}

func NewOpenPhaseCommandV0(meta OrchestrationCommandMetaV0, payload OpenPhaseCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandOpenPhaseV0, payload)
}

func NewClosePhaseCommandV0(meta OrchestrationCommandMetaV0, payload ClosePhaseCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandClosePhaseV0, normalizeClosePhasePayloadV0(payload))
}

func NewBlockRunCommandV0(meta OrchestrationCommandMetaV0, payload BlockRunCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandBlockRunV0, payload)
}

func newOrchestrationCommandV0(meta OrchestrationCommandMetaV0, commandType string, payload any) (OrchestrationCommandV0, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return OrchestrationCommandV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	command := OrchestrationCommandV0{
		CommandID:      strings.TrimSpace(meta.CommandID),
		CommandType:    commandType,
		RunID:          strings.TrimSpace(meta.RunID),
		IdempotencyKey: strings.TrimSpace(meta.IdempotencyKey),
		CorrelationID:  strings.TrimSpace(meta.CorrelationID),
		RequestedBy:    strings.TrimSpace(meta.RequestedBy),
		OccurredAt:     strings.TrimSpace(meta.OccurredAt),
		PayloadVersion: OrchestrationCommandPayloadVersionV0,
		Payload:        payloadJSON,
	}
	if err := ValidateOrchestrationCommandV0(command); err != nil {
		return OrchestrationCommandV0{}, err
	}
	return command, nil
}

func commandErrorV0(code string, field string) OrchestrationCommandErrorV0 {
	return OrchestrationCommandErrorV0{Code: code, Field: field}
}
