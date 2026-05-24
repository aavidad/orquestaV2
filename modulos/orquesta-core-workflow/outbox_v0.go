package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	OutboxPayloadVersionV0 = "outbox_payload.v0"

	OutboxMessagePersistRunEventsV0        = "PersistRunEvents"
	OutboxMessagePublishOrquestaEventV0    = "PublishOrquestaEvent"
	OutboxMessageRequestCapacityDecisionV0 = "RequestCapacityDecision"
	OutboxMessageLaunchRuntimeAgentV0      = "LaunchRuntimeAgent"
	OutboxMessageStopRuntimeAgentV0        = "StopRuntimeAgent"
	OutboxMessageRequestDeployPlanV0       = "RequestDeployPlan"
	OutboxMessageSendDirectorQuestionV0    = "SendDirectorQuestion"

	OutboxTargetPersistenceV0   = "persistence"
	OutboxTargetObservabilityV0 = "observability"
	OutboxTargetCapacityV0      = "capacity"
	OutboxTargetAgentLauncherV0 = "agent_launcher"
	OutboxTargetDeployPlannerV0 = "deploy_planner"
	OutboxTargetDirectorV0      = "director"

	ErrOutboxTipoNoSoportadoV0  = "outbox_tipo_no_soportado"
	ErrOutboxInvalidoV0         = "outbox_invalido"
	ErrOutboxPayloadInvalidoV0  = "payload_invalido"
	ErrOutboxDetalleProhibidoV0 = "detalle_prohibido"
)

const (
	maxOutboxPayloadBytesV0  = 4096
	maxOutboxStringValueV0   = 1000
	maxOutboxCollectionLenV0 = 20
)

var forbiddenOutboxPayloadKeysV0 = []string{
	"transcript",
	"prompt",
	"completion",
	"raw_text",
	"full_text",
	"full_context",
	"contexto_completo",
	"massive_context",
	"adapter",
	"adaptador",
}

type OutboxMessageV0 struct {
	MessageID        string          `json:"message_id"`
	MessageType      string          `json:"message_type"`
	RunID            string          `json:"run_id"`
	IdempotencyKey   string          `json:"idempotency_key"`
	CorrelationID    string          `json:"correlation_id,omitempty"`
	CausationEventID string          `json:"causation_event_id,omitempty"`
	TargetPort       string          `json:"target_port"`
	PayloadVersion   string          `json:"payload_version"`
	Payload          json.RawMessage `json:"payload"`
}

type OutboxMessageMetaV0 struct {
	MessageID        string
	RunID            string
	IdempotencyKey   string
	CorrelationID    string
	CausationEventID string
}

type OutboxMessageErrorV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func (err OutboxMessageErrorV0) Error() string {
	return codeFieldErrorTextV0(err.Code, err.Field)
}

func NewSendDirectorQuestionOutboxV0(meta OutboxMessageMetaV0, question DirectorQuestionV0) (OutboxMessageV0, error) {
	normalizedQuestion, err := NewDirectorQuestionV0(question)
	if err != nil {
		return OutboxMessageV0{}, err
	}
	payloadJSON, err := json.Marshal(normalizedQuestion)
	if err != nil {
		return OutboxMessageV0{}, outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload")
	}

	message := OutboxMessageV0{
		MessageID:        strings.TrimSpace(meta.MessageID),
		MessageType:      OutboxMessageSendDirectorQuestionV0,
		RunID:            strings.TrimSpace(meta.RunID),
		IdempotencyKey:   strings.TrimSpace(meta.IdempotencyKey),
		CorrelationID:    strings.TrimSpace(meta.CorrelationID),
		CausationEventID: strings.TrimSpace(meta.CausationEventID),
		TargetPort:       OutboxTargetDirectorV0,
		PayloadVersion:   OutboxPayloadVersionV0,
		Payload:          payloadJSON,
	}
	if err := ValidateOutboxMessageV0(message); err != nil {
		return OutboxMessageV0{}, err
	}
	return message, nil
}

func outboxErrorV0(code string, field string) OutboxMessageErrorV0 {
	return OutboxMessageErrorV0{Code: code, Field: field}
}
