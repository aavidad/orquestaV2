package orquestaobservability

import (
	"fmt"
	"strings"
	"time"
)

func validateOperationalStatusQueryV0(query OperationalStatusQueryV0, prefix string, add func(string, string)) {
	if strings.TrimSpace(query.SchemaVersion) != OperationalStatusQuerySchemaVersionV0 {
		add(ErrOperationalStatusQueryInvalidaV0, fieldV0(prefix, "schema_version"))
	}
	validateRequiredOperationalRefV0(query.RequestID, fieldV0(prefix, "request_id"), add)
	validateRequiredOperationalRefV0(query.CorrelationID, fieldV0(prefix, "correlation_id"), add)
	validateOperationalConsumerV0(query.Consumer, fieldV0(prefix, "consumer"), add)
	validateOperationalLocaleV0(query.Locale, fieldV0(prefix, "locale"), add)

	scope := strings.TrimSpace(query.Scope)
	if !allowedV0(allowedOperationalScopesV0, scope) {
		add(ErrScopeNoSoportadoV0, fieldV0(prefix, "scope"))
	}

	validateOptionalOperationalRefV0(query.SubjectRef, fieldV0(prefix, "subject_ref"), add)
	validateOptionalOperationalRefV0(query.TraceRef, fieldV0(prefix, "trace_ref"), add)
	validateOperationalTimeWindowV0(query.TimeWindow, fieldV0(prefix, "time_window"), add)
	validateOperationalSectionsV0(query.IncludeSections, fieldV0(prefix, "include_sections"), add)
	validateOperationalLimitV0(query.Limit, fieldV0(prefix, "limit"), add)
	validateOperationalFreshnessRequestV0(query.Freshness, fieldV0(prefix, "freshness"), add)
	validateOperationalJSONSizeV0(query, maxOperationalQueryJSONBytesV0, fieldV0(prefix, ""), add)
}

func validateOperationalConsumerV0(consumer OperationalStatusConsumerV0, prefix string, add func(string, string)) {
	module := strings.TrimSpace(consumer.Module)
	channel := strings.TrimSpace(consumer.Channel)
	expectedChannel, moduleAllowed := operationalConsumerPairsV0[module]
	if !moduleAllowed || expectedChannel != channel {
		add(ErrConsumidorNoAutorizadoV0, prefix)
	}
	validateOperationalTokenTextV0(module, fieldV0(prefix, "module"), add)
	validateOperationalTokenTextV0(channel, fieldV0(prefix, "channel"), add)
}

func validateOperationalLocaleV0(locale string, field string, add func(string, string)) {
	trimmed := strings.TrimSpace(locale)
	if trimmed == "" || !validSizedPatternV0(trimmed, 2, 8, operationalLocalePatternV0) {
		add(ErrOperationalStatusQueryInvalidaV0, field)
		return
	}
	validateOperationalTextV0(trimmed, field, maxOperationalTokenRunesV0, true, add)
}

func validateOperationalTimeWindowV0(window *OperationalStatusTimeWindowV0, prefix string, add func(string, string)) {
	if window == nil {
		return
	}
	if strings.TrimSpace(window.From) == "" && strings.TrimSpace(window.To) == "" && strings.TrimSpace(window.Preset) == "" {
		add(ErrOperationalStatusQueryInvalidaV0, prefix)
		return
	}
	if strings.TrimSpace(window.From) != "" && !isOccurredAtV0(window.From) {
		add(ErrOperationalStatusQueryInvalidaV0, fieldV0(prefix, "from"))
	}
	if strings.TrimSpace(window.To) != "" && !isOccurredAtV0(window.To) {
		add(ErrOperationalStatusQueryInvalidaV0, fieldV0(prefix, "to"))
	}
	if isOccurredAtV0(window.From) && isOccurredAtV0(window.To) {
		from, _ := time.Parse(time.RFC3339Nano, strings.TrimSpace(window.From))
		to, _ := time.Parse(time.RFC3339Nano, strings.TrimSpace(window.To))
		if from.After(to) {
			add(ErrOperationalStatusQueryInvalidaV0, prefix)
		}
	}
	if strings.TrimSpace(window.Preset) != "" {
		validateOperationalTokenTextV0(window.Preset, fieldV0(prefix, "preset"), add)
	}
}

func validateOperationalSectionsV0(sections []string, prefix string, add func(string, string)) {
	if len(sections) == 0 {
		add(ErrOperationalStatusQueryInvalidaV0, prefix)
		return
	}
	if len(sections) > maxOperationalQuerySectionsV0 {
		add(ErrConsultaDemasiadoAmpliaV0, prefix)
	}
	seen := map[string]bool{}
	for index, section := range sections {
		field := fmt.Sprintf("%s[%d]", prefix, index)
		trimmed := strings.TrimSpace(section)
		if !allowedV0(allowedOperationalSectionsV0, trimmed) {
			add(ErrOperationalStatusQueryInvalidaV0, field)
			continue
		}
		if seen[trimmed] {
			add(ErrOperationalStatusQueryInvalidaV0, field)
		}
		seen[trimmed] = true
	}
}

func validateOperationalLimitV0(limit int, field string, add func(string, string)) {
	if limit <= 0 {
		add(ErrOperationalStatusQueryInvalidaV0, field)
		return
	}
	if limit > maxOperationalLimitV0 {
		add(ErrConsultaDemasiadoAmpliaV0, field)
	}
}

func validateOperationalFreshnessRequestV0(freshness *OperationalStatusFreshnessRequestV0, prefix string, add func(string, string)) {
	if freshness == nil {
		return
	}
	if freshness.MaxAgeSeconds <= 0 && strings.TrimSpace(freshness.WatermarkRef) == "" {
		add(ErrOperationalStatusQueryInvalidaV0, prefix)
	}
	if freshness.MaxAgeSeconds < 0 || freshness.MaxAgeSeconds > maxOperationalFreshnessAgeSecondsV0 {
		add(ErrFrescuraNoGarantizadaV0, fieldV0(prefix, "max_age_seconds"))
	}
	validateOptionalOperationalRefV0(freshness.WatermarkRef, fieldV0(prefix, "watermark_ref"), add)
}
