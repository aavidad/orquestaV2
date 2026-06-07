package orquestaappchange

import (
	"strings"
	"unicode"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func normalizeAppChangeRequestV0(request AppChangeRequestV0) AppChangeRequestV0 {
	request.SchemaVersion = strings.TrimSpace(request.SchemaVersion)
	if request.SchemaVersion == "" {
		request.SchemaVersion = AppChangeRequestSchemaV0
	}
	request.RequestID = strings.TrimSpace(request.RequestID)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.RunRef = normalizeAppChangeRefValueV0(request.RunRef)
	request.AppRef = strings.TrimSpace(request.AppRef)
	request.ChangeRef = normalizeAppChangeRefValueV0(request.ChangeRef)
	request.ActorRef = strings.TrimSpace(request.ActorRef)
	request.Locale = strings.TrimSpace(request.Locale)
	request.UserIntent = strings.TrimSpace(request.UserIntent)
	request.TargetArea = strings.TrimSpace(request.TargetArea)
	request.CurrentStateRefs = compactAppChangeRefsV0(request.CurrentStateRefs)
	request.Scope = compactAppChangeStringsV0(request.Scope)
	request.AcceptanceCriteria = compactAppChangeStringsV0(request.AcceptanceCriteria)
	request.Constraints = compactAppChangeStringsV0(request.Constraints)
	request.AllowedWriteSet = compactAppChangeStringsV0(request.AllowedWriteSet)
	request.RequiredTests = compactAppChangeStringsV0(request.RequiredTests)
	request.MetadataRefs = compactAppChangeRefsV0(request.MetadataRefs)
	request.ExternalWork = normalizeAppChangeExternalWorkV0(request.ExternalWork)
	if request.RequestID == "" {
		request.RequestID = request.ChangeRef
	}
	if request.CorrelationID == "" {
		request.CorrelationID = request.RequestID
	}
	return request
}

func normalizeAppChangeExternalWorkV0(
	work *AppChangeExternalWorkV0,
) *AppChangeExternalWorkV0 {
	if work == nil {
		return nil
	}
	normalized := AppChangeExternalWorkV0{
		ProjectRef:    normalizeAppChangeRefValueV0(work.ProjectRef),
		JobRef:        normalizeAppChangeRefValueV0(work.JobRef),
		InterfaceRefs: compactAppChangeRefsV0(work.InterfaceRefs),
		WorkKind:      normalizeAppChangeRefValueV0(work.WorkKind),
		WorkRefs:      compactAppChangeRefsV0(work.WorkRefs),
		InputFields:   normalizeAppChangeExternalWorkFieldsV0(work.InputFields),
		RequiredTests: normalizeAppChangeExternalWorkRequiredTestsV0(work.RequiredTests),
	}
	if normalized.ProjectRef == "" &&
		normalized.JobRef == "" &&
		len(normalized.InterfaceRefs) == 0 &&
		normalized.WorkKind == "" &&
		len(normalized.WorkRefs) == 0 &&
		len(normalized.InputFields) == 0 &&
		len(normalized.RequiredTests) == 0 {
		return nil
	}
	return &normalized
}

func normalizeAppChangeExternalWorkFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	request := orquestadomainwork.NormalizeDomainWorkJobRequestV0(
		orquestadomainwork.DomainWorkJobRequestV0{
			InputFields: fields,
		},
	)
	for index := range request.InputFields {
		request.InputFields[index].Name = normalizeAppChangeFieldNameV0(request.InputFields[index].Name)
	}
	return request.InputFields
}

func normalizeAppChangeExternalWorkRequiredTestsV0(
	tests []orquestadomainwork.DomainWorkRequiredTestV0,
) []orquestadomainwork.DomainWorkRequiredTestV0 {
	request := orquestadomainwork.NormalizeDomainWorkJobRequestV0(
		orquestadomainwork.DomainWorkJobRequestV0{
			RequiredTests: tests,
		},
	)
	return request.RequiredTests
}

func compactAppChangeRefsV0(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		normalized = append(normalized, normalizeAppChangeRefValueV0(value))
	}
	return compactAppChangeStringsV0(normalized)
}

func normalizeAppChangeRefValueV0(value string) string {
	return normalizeAppChangeCompactTokenV0(value, '-')
}

func normalizeAppChangeFieldNameV0(value string) string {
	return normalizeAppChangeCompactTokenV0(value, '_')
}

func normalizeAppChangeCompactTokenV0(value string, separator rune) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	var builder strings.Builder
	lastSeparator := false
	for _, char := range trimmed {
		switch {
		case char == '/' || char == '\\' || unicode.IsSpace(char):
			if builder.Len() > 0 && !lastSeparator {
				builder.WriteRune(separator)
				lastSeparator = true
			}
		default:
			builder.WriteRune(char)
			lastSeparator = false
		}
	}
	return strings.Trim(string(builder.String()), string(separator))
}

func compactAppChangeStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	if out == nil {
		return []string{}
	}
	return out
}
