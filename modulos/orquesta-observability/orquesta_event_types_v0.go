package orquestaobservability

import "regexp"

const (
	OrquestaEventSchemaVersionV0 = "orquesta_event.v0"

	OrquestaEventSourceAreaCoreV0     = "core"
	OrquestaEventSourceAreaRuntimeV0  = "runtime"
	OrquestaEventSourceAreaCapacityV0 = "capacity"
	OrquestaEventSourceAreaReviewV0   = "review"

	OrquestaEventSeverityDebugV0   = "debug"
	OrquestaEventSeverityInfoV0    = "info"
	OrquestaEventSeverityWarningV0 = "warning"
	OrquestaEventSeverityErrorV0   = "error"

	OrquestaEventOutcomeObservedV0  = "observed"
	OrquestaEventOutcomeAcceptedV0  = "accepted"
	OrquestaEventOutcomeRejectedV0  = "rejected"
	OrquestaEventOutcomeRunningV0   = "running"
	OrquestaEventOutcomeCompletedV0 = "completed"
	OrquestaEventOutcomeFailedV0    = "failed"
	OrquestaEventOutcomeDegradedV0  = "degraded"
	OrquestaEventOutcomeSkippedV0   = "skipped"

	OrquestaEventPrivacyPublicOperationalV0   = "public_operational"
	OrquestaEventPrivacyInternalOperationalV0 = "internal_operational"
	OrquestaEventPrivacyRedactionNoneV0       = "none"
	OrquestaEventPrivacyMetadataOnlyV0        = "metadata_only"
	OrquestaEventPrivacySummarizedV0          = "summarized"
	OrquestaEventPrivacyRedactedV0            = "redacted"

	ErrOrquestaEventInvalidoV0   = "orquesta_event_invalido"
	ErrEventoDemasiadoExtensoV0  = "evento_demasiado_extenso"
	ErrSecretoDetectadoV0        = "secreto_detectado"
	ErrTranscriptNoPermitidoV0   = "transcript_no_permitido"
	ErrCorrelationIDRequeridoV0  = "correlation_id_requerido"
	ErrIdempotencyKeyRequeridaV0 = "idempotency_key_requerida"
	ErrSourceAreaNoSoportadaV0   = "source_area_no_soportada"
	ErrEventTypeIncompatibleV0   = "event_type_incompatible"
	ErrSinkNoDisponibleV0        = "sink_no_disponible"
	ErrDuplicadoIdempotenteV0    = "duplicado_idempotente"
)

const (
	maxEventJSONBytesV0       = 8192
	maxPayloadJSONBytesV0     = 4096
	maxPayloadPropertiesV0    = 16
	maxPayloadArrayItemsV0    = 20
	maxPayloadStringRunesV0   = 512
	maxPayloadDepthV0         = 4
	maxPayloadPropertyRunesV0 = 64
	maxSummaryRunesV0         = 280
	maxLinksV0                = 12
	maxEventTypeRunesV0       = 96
	maxOpaqueIDRunesV0        = 96
	maxProducerTokenRunesV0   = 80
	maxSubjectVersionRunesV0  = 32
	minEventTypeRunesV0       = 5
	minOpaqueIDRunesV0        = 8
	minPayloadPropertyRunesV0 = 1
	minProducerTokenRunesV0   = 1
	minSubjectVersionRunesV0  = 1
)

var (
	eventTypePatternV0      = regexp.MustCompile(`^(core|runtime|capacity|review)\.[a-z0-9_]+(\.[a-z0-9_]+)?$`)
	opaqueIDPatternV0       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]*(-[A-Za-z0-9_.:-]+)*$`)
	payloadKeyPatternV0     = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	producerTokenPatternV0  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.:-]*$`)
	subjectVersionPatternV0 = regexp.MustCompile(`^[A-Za-z0-9_.:-]+$`)
	forbiddenPayloadKeysV0  = []string{
		"secret", "secreto", "token", "password", "credential", "credencial", "api_key", "oauth",
		"transcript", "prompt", "completion", "raw_text", "full_text", "sql", "dsn",
		"connection", "conexion", "table", "tabla", "provider", "proveedor", "model_name",
		"home_path", "home",
	}
	secretPayloadKeyPartsV0  = []string{"secret", "secreto", "token", "password", "credential", "credencial", "api_key", "oauth"}
	allowedSourceAreasV0     = setV0(OrquestaEventSourceAreaCoreV0, OrquestaEventSourceAreaRuntimeV0, OrquestaEventSourceAreaCapacityV0, OrquestaEventSourceAreaReviewV0)
	allowedSeveritiesV0      = setV0(OrquestaEventSeverityDebugV0, OrquestaEventSeverityInfoV0, OrquestaEventSeverityWarningV0, OrquestaEventSeverityErrorV0)
	allowedOutcomesV0        = setV0(OrquestaEventOutcomeObservedV0, OrquestaEventOutcomeAcceptedV0, OrquestaEventOutcomeRejectedV0, OrquestaEventOutcomeRunningV0, OrquestaEventOutcomeCompletedV0, OrquestaEventOutcomeFailedV0, OrquestaEventOutcomeDegradedV0, OrquestaEventOutcomeSkippedV0)
	allowedSubjectKindsV0    = setV0("project", "task", "agent_run", "capacity_decision", "review", "merge", "artifact", "system")
	allowedProducerModulesV0 = setV0("orquesta-core", "orquesta-runtime", "orquesta-capacity", "orquesta-review")
	allowedClassificationsV0 = setV0(OrquestaEventPrivacyPublicOperationalV0, OrquestaEventPrivacyInternalOperationalV0)
	allowedRedactionLevelsV0 = setV0(OrquestaEventPrivacyRedactionNoneV0, OrquestaEventPrivacyMetadataOnlyV0, OrquestaEventPrivacySummarizedV0, OrquestaEventPrivacyRedactedV0)
	allowedLinkRelsV0        = setV0("caused_by", "part_of", "related_to", "artifact", "run", "parent", "follow_up")
	allowedLinkTargetsV0     = setV0("event", "project", "task", "agent_run", "capacity_decision", "review", "artifact")
)

type PublishOrquestaEventRequestV0 struct {
	RequestID      string          `json:"request_id,omitempty"`
	CorrelationID  string          `json:"correlation_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	Event          OrquestaEventV0 `json:"event"`
	DryRun         bool            `json:"dry_run,omitempty"`
}

type OrquestaEventV0 struct {
	SchemaVersion string                     `json:"schema_version"`
	EventID       string                     `json:"event_id"`
	EventType     string                     `json:"event_type"`
	SourceArea    string                     `json:"source_area"`
	OccurredAt    string                     `json:"occurred_at"`
	Severity      string                     `json:"severity"`
	Outcome       string                     `json:"outcome"`
	Correlation   OrquestaEventCorrelationV0 `json:"correlation"`
	Subject       OrquestaEventSubjectV0     `json:"subject"`
	Producer      OrquestaEventProducerV0    `json:"producer"`
	Privacy       OrquestaEventPrivacyV0     `json:"privacy"`
	Summary       string                     `json:"summary"`
	Links         []OrquestaEventLinkV0      `json:"links,omitempty"`
	Payload       map[string]any             `json:"payload"`
}

type OrquestaEventCorrelationV0 struct {
	CorrelationID string `json:"correlation_id"`
	RequestID     string `json:"request_id,omitempty"`
	CausationID   string `json:"causation_id,omitempty"`
	TraceID       string `json:"trace_id,omitempty"`
}

type OrquestaEventSubjectV0 struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
}

type OrquestaEventProducerV0 struct {
	Module  string `json:"module"`
	Port    string `json:"port"`
	Adapter string `json:"adapter,omitempty"`
}

type OrquestaEventPrivacyV0 struct {
	ContainsSecret     bool   `json:"contains_secret"`
	ContainsTranscript bool   `json:"contains_transcript"`
	Classification     string `json:"classification"`
	RedactionLevel     string `json:"redaction_level"`
}

type OrquestaEventLinkV0 struct {
	Rel        string `json:"rel"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
}

type PublishOrquestaEventAcceptedV0 struct {
	Accepted      bool   `json:"accepted"`
	EventID       string `json:"event_id"`
	CorrelationID string `json:"correlation_id"`
	SinkReceiptID string `json:"sink_receipt_id,omitempty"`
	Stored        bool   `json:"stored"`
}

type OrquestaEventValidationIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

type OrquestaEventValidationErrorV0 struct {
	Issues []OrquestaEventValidationIssueV0 `json:"issues"`
}
