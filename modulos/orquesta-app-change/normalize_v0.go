package orquestaappchange

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func normalizeAppChangeRequestV0(request AppChangeRequestV0) AppChangeRequestV0 {
	request.SchemaVersion = strings.TrimSpace(request.SchemaVersion)
	if request.SchemaVersion == "" {
		request.SchemaVersion = AppChangeRequestSchemaV0
	}
	request.RequestID = strings.TrimSpace(request.RequestID)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.AppRef = strings.TrimSpace(request.AppRef)
	request.ChangeRef = strings.TrimSpace(request.ChangeRef)
	request.ActorRef = strings.TrimSpace(request.ActorRef)
	request.Locale = strings.TrimSpace(request.Locale)
	request.UserIntent = strings.TrimSpace(request.UserIntent)
	request.TargetArea = strings.TrimSpace(request.TargetArea)
	request.CurrentStateRefs = compactAppChangeStringsV0(request.CurrentStateRefs)
	request.Scope = compactAppChangeStringsV0(request.Scope)
	request.AcceptanceCriteria = compactAppChangeStringsV0(request.AcceptanceCriteria)
	request.Constraints = compactAppChangeStringsV0(request.Constraints)
	request.AllowedWriteSet = compactAppChangeStringsV0(request.AllowedWriteSet)
	request.RequiredTests = compactAppChangeStringsV0(request.RequiredTests)
	request.MetadataRefs = compactAppChangeStringsV0(request.MetadataRefs)
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
		ProjectRef:    strings.TrimSpace(work.ProjectRef),
		JobRef:        strings.TrimSpace(work.JobRef),
		InterfaceRefs: compactAppChangeStringsV0(work.InterfaceRefs),
		WorkKind:      strings.TrimSpace(work.WorkKind),
		WorkRefs:      compactAppChangeStringsV0(work.WorkRefs),
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
