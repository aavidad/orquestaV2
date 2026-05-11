package orquestaobservability

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"
)

func validateOrquestaEventV0(event OrquestaEventV0, prefix string, add func(string, string)) {
	if strings.TrimSpace(event.SchemaVersion) != OrquestaEventSchemaVersionV0 {
		add(ErrOrquestaEventInvalidoV0, fieldV0(prefix, "schema_version"))
	}
	if !isOpaqueIDV0(event.EventID) {
		add(ErrOrquestaEventInvalidoV0, fieldV0(prefix, "event_id"))
	}

	sourceArea := strings.TrimSpace(event.SourceArea)
	eventType := strings.TrimSpace(event.EventType)
	if !allowedV0(allowedSourceAreasV0, sourceArea) {
		add(ErrSourceAreaNoSoportadaV0, fieldV0(prefix, "source_area"))
	}
	if !validSizedPatternV0(eventType, minEventTypeRunesV0, maxEventTypeRunesV0, eventTypePatternV0) {
		add(ErrOrquestaEventInvalidoV0, fieldV0(prefix, "event_type"))
	} else if allowedV0(allowedSourceAreasV0, sourceArea) && !strings.HasPrefix(eventType, sourceArea+".") {
		add(ErrEventTypeIncompatibleV0, fieldV0(prefix, "event_type"))
	}

	if !isOccurredAtV0(event.OccurredAt) {
		add(ErrOrquestaEventInvalidoV0, fieldV0(prefix, "occurred_at"))
	}
	if !allowedV0(allowedSeveritiesV0, strings.TrimSpace(event.Severity)) {
		add(ErrOrquestaEventInvalidoV0, fieldV0(prefix, "severity"))
	}
	if !allowedV0(allowedOutcomesV0, strings.TrimSpace(event.Outcome)) {
		add(ErrOrquestaEventInvalidoV0, fieldV0(prefix, "outcome"))
	}

	validateCorrelationV0(event.Correlation, fieldV0(prefix, "correlation"), add)
	validateSubjectV0(event.Subject, fieldV0(prefix, "subject"), add)
	validateProducerV0(event.Producer, fieldV0(prefix, "producer"), add)
	validatePrivacyV0(event.Privacy, fieldV0(prefix, "privacy"), add)
	validateSummaryV0(event.Summary, fieldV0(prefix, "summary"), add)
	validateLinksV0(event.Links, fieldV0(prefix, "links"), add)
	validatePayloadV0(event.Payload, fieldV0(prefix, "payload"), add)
	validateEventSizeV0(event, prefix, add)
}

func validateCorrelationV0(correlation OrquestaEventCorrelationV0, prefix string, add func(string, string)) {
	if !isOpaqueIDV0(correlation.CorrelationID) {
		add(ErrCorrelationIDRequeridoV0, fieldV0(prefix, "correlation_id"))
	}
	validateOptionalOpaqueIDV0(correlation.RequestID, fieldV0(prefix, "request_id"), add)
	validateOptionalOpaqueIDV0(correlation.CausationID, fieldV0(prefix, "causation_id"), add)
	validateOptionalOpaqueIDV0(correlation.TraceID, fieldV0(prefix, "trace_id"), add)
}

func validateSubjectV0(subject OrquestaEventSubjectV0, prefix string, add func(string, string)) {
	if !allowedV0(allowedSubjectKindsV0, strings.TrimSpace(subject.Kind)) {
		add(ErrOrquestaEventInvalidoV0, fieldV0(prefix, "kind"))
	}
	if !isOpaqueIDV0(subject.ID) {
		add(ErrOrquestaEventInvalidoV0, fieldV0(prefix, "id"))
	}
	if strings.TrimSpace(subject.Version) != "" && !validSizedPatternV0(strings.TrimSpace(subject.Version), minSubjectVersionRunesV0, maxSubjectVersionRunesV0, subjectVersionPatternV0) {
		add(ErrOrquestaEventInvalidoV0, fieldV0(prefix, "version"))
	}
}

func validateProducerV0(producer OrquestaEventProducerV0, prefix string, add func(string, string)) {
	if !allowedV0(allowedProducerModulesV0, strings.TrimSpace(producer.Module)) {
		add(ErrOrquestaEventInvalidoV0, fieldV0(prefix, "module"))
	}
	if !validSizedPatternV0(strings.TrimSpace(producer.Port), minProducerTokenRunesV0, maxProducerTokenRunesV0, producerTokenPatternV0) {
		add(ErrOrquestaEventInvalidoV0, fieldV0(prefix, "port"))
	}
	if strings.TrimSpace(producer.Adapter) != "" && !validSizedPatternV0(strings.TrimSpace(producer.Adapter), minProducerTokenRunesV0, maxProducerTokenRunesV0, producerTokenPatternV0) {
		add(ErrOrquestaEventInvalidoV0, fieldV0(prefix, "adapter"))
	}
}

func validatePrivacyV0(privacy OrquestaEventPrivacyV0, prefix string, add func(string, string)) {
	if privacy.ContainsSecret {
		add(ErrSecretoDetectadoV0, fieldV0(prefix, "contains_secret"))
	}
	if privacy.ContainsTranscript {
		add(ErrTranscriptNoPermitidoV0, fieldV0(prefix, "contains_transcript"))
	}
	if !allowedV0(allowedClassificationsV0, strings.TrimSpace(privacy.Classification)) {
		add(ErrOrquestaEventInvalidoV0, fieldV0(prefix, "classification"))
	}
	if !allowedV0(allowedRedactionLevelsV0, strings.TrimSpace(privacy.RedactionLevel)) {
		add(ErrOrquestaEventInvalidoV0, fieldV0(prefix, "redaction_level"))
	}
}

func validateSummaryV0(summary string, field string, add func(string, string)) {
	trimmed := strings.TrimSpace(summary)
	if trimmed == "" {
		add(ErrOrquestaEventInvalidoV0, field)
		return
	}
	if utf8.RuneCountInString(summary) > maxSummaryRunesV0 {
		add(ErrEventoDemasiadoExtensoV0, field)
	}
}

func validateLinksV0(links []OrquestaEventLinkV0, prefix string, add func(string, string)) {
	if len(links) > maxLinksV0 {
		add(ErrEventoDemasiadoExtensoV0, prefix)
	}
	for index, link := range links {
		item := fmt.Sprintf("%s[%d]", prefix, index)
		if !allowedV0(allowedLinkRelsV0, strings.TrimSpace(link.Rel)) {
			add(ErrOrquestaEventInvalidoV0, fieldV0(item, "rel"))
		}
		if !allowedV0(allowedLinkTargetsV0, strings.TrimSpace(link.TargetType)) {
			add(ErrOrquestaEventInvalidoV0, fieldV0(item, "target_type"))
		}
		if !isOpaqueIDV0(link.TargetID) {
			add(ErrOrquestaEventInvalidoV0, fieldV0(item, "target_id"))
		}
	}
}

func validatePayloadV0(payload map[string]any, field string, add func(string, string)) {
	if payload == nil {
		add(ErrOrquestaEventInvalidoV0, field)
		return
	}
	validateCompactMapV0(field, reflect.ValueOf(payload), 0, add)
	data, err := json.Marshal(payload)
	if err != nil {
		add(ErrOrquestaEventInvalidoV0, field)
		return
	}
	if len(data) > maxPayloadJSONBytesV0 {
		add(ErrEventoDemasiadoExtensoV0, field)
	}
}

func validateEventSizeV0(event OrquestaEventV0, prefix string, add func(string, string)) {
	data, err := json.Marshal(event)
	if err != nil {
		add(ErrOrquestaEventInvalidoV0, prefix)
		return
	}
	if len(data) > maxEventJSONBytesV0 {
		add(ErrEventoDemasiadoExtensoV0, prefix)
	}
}
