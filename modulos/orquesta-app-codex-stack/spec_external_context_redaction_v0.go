package orquestaappcodexstack

import (
	"fmt"
	"strings"
	"unicode/utf8"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestarails "orquesta/modulos/orquesta-rails"
)

func redactExternalWorkSensitiveFieldV0(
	field orquestadomainwork.DomainWorkFieldV0,
) orquestadomainwork.DomainWorkFieldV0 {
	reviewRequired := externalWorkFieldNeedsSanitizationReviewV0(field)
	if externalContextRefPartHasSensitiveNameV0(field.Name) {
		field.Value = "redacted-sensitive-field"
		if reviewRequired {
			field.Value = "non_public_context_review_required"
		}
		field.Values = nil
		field.ValueJSON = nil
		return field
	}
	if externalWorkFieldHasPrivateMaterialV0(field) {
		field.Value = "prompt completo redacted-sensitive-material"
		field.Values = nil
		field.ValueJSON = nil
		return field
	}
	field.Value, _ = orquestarails.RedactOperationalTextForFieldV0(
		"codex_stack_external_context",
		field.Name,
		field.Value,
	)
	for index, value := range field.Values {
		field.Values[index], _ = orquestarails.RedactOperationalTextForFieldV0(
			"codex_stack_external_context",
			field.Name,
			value,
		)
	}
	if len(field.ValueJSON) > 0 {
		redacted, _ := orquestarails.RedactOperationalTextForFieldV0(
			"codex_stack_external_context",
			field.Name,
			string(field.ValueJSON),
		)
		field.ValueJSON = []byte(redacted)
	}
	if reviewRequired {
		field.Value = "non_public_context_review_required"
		field.Values = nil
		field.ValueJSON = nil
	}
	return field
}

func externalWorkFieldNeedsSanitizationReviewV0(field orquestadomainwork.DomainWorkFieldV0) bool {
	if localSensitiveDataSanitizerNeedsReviewV0(field.Value) ||
		localSensitiveDataSanitizerNeedsReviewV0(string(field.ValueJSON)) {
		return true
	}
	for _, value := range field.Values {
		if localSensitiveDataSanitizerNeedsReviewV0(value) {
			return true
		}
	}
	return false
}

func externalWorkFieldHasPrivateMaterialV0(field orquestadomainwork.DomainWorkFieldV0) bool {
	for _, value := range append(append([]string{field.Value}, field.Values...), string(field.ValueJSON)) {
		lower := strings.ToLower(value)
		if strings.Contains(lower, "-----begin ") ||
			strings.Contains(lower, "private key-----") {
			return true
		}
	}
	return false
}

func boundedExternalWorkContextContentV0(value string, maxBytes int) (string, bool) {
	value = strings.TrimSpace(value)
	if maxBytes <= 0 {
		return "", true
	}
	if len(value) <= maxBytes {
		return value, false
	}
	return truncateExternalWorkContextByBytesV0(value, maxBytes), true
}

func sanitizeExternalWorkContextContentV0(value string) string {
	replacer := strings.NewReplacer(
		"://", "_url_",
		"/home/", "/home-redacted/",
		"/Users/", "/users-redacted/",
		"/users/", "/users-redacted/",
		"$HOME", "HOME_REF",
		"~/", "HOME_REF/",
	)
	return replacer.Replace(strings.TrimSpace(value))
}

func externalContextRefPartV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer(
		" ", "-",
		"\t", "-",
		"\n", "-",
		"\r", "-",
		"/", "-",
		"\\", "-",
	).Replace(value)
	value = strings.Trim(value, "-")
	if value == "" || externalContextRefPartHasSensitiveNameV0(value) {
		return "field"
	}
	return value
}

func externalContextRefPartHasSensitiveNameV0(value string) bool {
	value = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "_", "-")
	for _, marker := range []string{
		"access-token", "refresh-token", "api-key", "secret", "secreto",
		"password", "credential", "credencial", "token", "prompt",
		"transcript", "completion",
	} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func externalWorkContextBudgetEntryV0(
	area string,
	taskRef string,
	omitted int,
) orquestacontext.ContextMaterializedEntryV0 {
	refSuffix := packetRefSuffixV0(taskRef, area, "context-budget", fmt.Sprintf("%02d", omitted))
	content := fmt.Sprintf(
		`{"name":"external_context_budget","value":"omitted_input_fields","omitted_fields":%d}`,
		omitted,
	)
	return orquestacontext.ContextMaterializedEntryV0{
		EntryRef:  "entry-ref-app-stack-external-" + refSuffix,
		Layer:     orquestacontext.ContextLayerTaskContextV0,
		Kind:      orquestacontext.ContextEntryDocRefV0,
		SourceRef: "source-ref-app-stack-external-" + refSuffix,
		Mode:      orquestacontext.ContextMaterializationModeContentV0,
		Content:   content,
		Bytes:     len(content),
		Truncated: true,
		Required:  false,
	}
}

func minIntV0(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncateExternalWorkContextByBytesV0(value string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(value) <= maxBytes {
		return value
	}
	for maxBytes > 0 && !utf8.ValidString(value[:maxBytes]) {
		maxBytes--
	}
	return value[:maxBytes]
}
