package orquestacore

import (
	"context"
	"strconv"
	"strings"
)

const (
	OrquestaEventContractVersionV0                   = "orquesta_event.v0"
	OrquestaEventSourceAreaCoreV0                    = "core"
	OrquestaEventTypeProyectoRegistradoEnBorradorV0  = "core.proyecto_registrado_en_borrador.v0"
	OrquestaEventSummaryProyectoRegistradoBorradorV0 = "proyecto registrado en borrador"
	OrquestaEventSeverityInfoV0                      = "info"
	OrquestaEventOutcomeAcceptedV0                   = "accepted"
	OrquestaEventSubjectKindProjectV0                = "project"
	OrquestaEventProducerModuleCoreV0                = "orquesta-core"
	OrquestaEventProducerPortCoreV0                  = "OrquestaEventPublisherPortV0"
	OrquestaEventPrivacyInternalOperationalV0        = "internal_operational"
	OrquestaEventPrivacyRedactionSummarizedV0        = "summarized"
)

type OrquestaEventPublisherPortV0 interface {
	PublicarOrquestaEventV0(context.Context, PublishOrquestaEventRequestV0) (PublishOrquestaEventAcceptedV0, error)
}

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

type OrquestaEventPublishErrorV0 struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	Field         string `json:"field,omitempty"`
	Retryable     bool   `json:"retryable"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

func (err OrquestaEventPublishErrorV0) Error() string {
	return err.Code
}

func NewPublishOrquestaEventRequestsV0(cmd RegistrarProyectoDesdeAppSpecCommandV0, accepted RegistroProyectoAceptadoV0, dryRun bool) []PublishOrquestaEventRequestV0 {
	requests := make([]PublishOrquestaEventRequestV0, 0, len(accepted.EventosDominio))
	for index, event := range accepted.EventosDominio {
		correlationID := firstNonEmptyV0(event.CorrelationID, cmd.CorrelationID)
		idempotencySource := strings.Join([]string{
			cmd.IdempotencyKey,
			accepted.RegistroID,
			event.Tipo,
			event.ProyectoID,
			strconv.Itoa(index),
		}, "|")
		requestID := firstNonEmptyV0(cmd.RequestID, cmd.AppSpec.RequestID)
		eventID := stableIDV0("event", idempotencySource)
		requests = append(requests, PublishOrquestaEventRequestV0{
			RequestID:      requestID,
			CorrelationID:  correlationID,
			IdempotencyKey: stableIDV0("orquesta_event", idempotencySource),
			Event:          MapEventoDominioCoreAOrquestaEventV0(event, correlationID, requestID, eventID),
			DryRun:         dryRun,
		})
	}
	return requests
}

func MapEventoDominioCoreAOrquestaEventV0(event EventoDominioCoreV0, correlationID, requestID, eventID string) OrquestaEventV0 {
	return OrquestaEventV0{
		SchemaVersion: OrquestaEventContractVersionV0,
		EventID:       strings.TrimSpace(eventID),
		EventType:     orquestaEventTypeV0(event.Tipo),
		SourceArea:    OrquestaEventSourceAreaCoreV0,
		OccurredAt:    strings.TrimSpace(event.OccurredAt),
		Severity:      OrquestaEventSeverityInfoV0,
		Outcome:       OrquestaEventOutcomeAcceptedV0,
		Correlation: OrquestaEventCorrelationV0{
			CorrelationID: strings.TrimSpace(correlationID),
			RequestID:     strings.TrimSpace(requestID),
		},
		Subject: OrquestaEventSubjectV0{
			Kind:    OrquestaEventSubjectKindProjectV0,
			ID:      strings.TrimSpace(event.ProyectoID),
			Version: ProyectoPlanBorradorPayloadVersionV0,
		},
		Producer: OrquestaEventProducerV0{
			Module: OrquestaEventProducerModuleCoreV0,
			Port:   OrquestaEventProducerPortCoreV0,
		},
		Privacy: OrquestaEventPrivacyV0{
			ContainsSecret:     false,
			ContainsTranscript: false,
			Classification:     OrquestaEventPrivacyInternalOperationalV0,
			RedactionLevel:     OrquestaEventPrivacyRedactionSummarizedV0,
		},
		Summary: orquestaEventSummaryV0(event.Tipo),
		Payload: copyAnyMapV0(event.Payload),
	}
}

func orquestaEventTypeV0(domainType string) string {
	switch strings.TrimSpace(domainType) {
	case EventoProyectoRegistradoEnBorradorV0:
		return OrquestaEventTypeProyectoRegistradoEnBorradorV0
	default:
		return "core.evento_dominio.v0"
	}
}

func orquestaEventSummaryV0(domainType string) string {
	switch strings.TrimSpace(domainType) {
	case EventoProyectoRegistradoEnBorradorV0:
		return OrquestaEventSummaryProyectoRegistradoBorradorV0
	default:
		return strings.TrimSpace(domainType)
	}
}

func copyAnyMapV0(values map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		result[key] = value
	}
	return result
}
