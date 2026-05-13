package orquestaexternalworkrun

import (
	"strings"
	"unicode"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
)

func normalizeStartExternalWorkRunRequestV0(
	request StartExternalWorkRunRequestV0,
	config StartExternalWorkRunConfigV0,
) StartExternalWorkRunRequestV0 {
	config = normalizeStartExternalWorkRunConfigV0(config)
	request.SchemaVersion = strings.TrimSpace(request.SchemaVersion)
	if request.SchemaVersion == "" {
		request.SchemaVersion = StartExternalWorkRunRequestSchemaV0
	}
	request.RequestID = strings.TrimSpace(request.RequestID)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.ProjectRef = strings.TrimSpace(request.ProjectRef)
	request.AppSpecRef = strings.TrimSpace(request.AppSpecRef)
	request.QueueRef = strings.TrimSpace(request.QueueRef)
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.RequestedBy = strings.TrimSpace(request.RequestedBy)

	request.AppChangeRequest = orquestaappchange.PrepareAppChangeRequestV0(request.AppChangeRequest)
	request = fillExternalWorkRunEnvelopeV0(request, config)
	request.AppChangeRequest.RunRef = request.RunRef
	if request.AppChangeRequest.AppRef == "" {
		request.AppChangeRequest.AppRef = request.ProjectRef
	}
	if request.AppChangeRequest.CorrelationID == "" {
		request.AppChangeRequest.CorrelationID = request.CorrelationID
	}
	if request.AppChangeRequest.RequestID == "" {
		request.AppChangeRequest.RequestID = request.RequestID
	}
	request.AppChangeRequest = orquestaappchange.PrepareAppChangeRequestV0(request.AppChangeRequest)
	return request
}

func normalizeStartExternalWorkRunConfigV0(
	config StartExternalWorkRunConfigV0,
) StartExternalWorkRunConfigV0 {
	config.QueueRef = strings.TrimSpace(config.QueueRef)
	if config.QueueRef == "" {
		config.QueueRef = ExternalWorkRunDefaultQueueRefV0
	}
	if config.DefaultPriorityScore <= 0 {
		config.DefaultPriorityScore = ExternalWorkRunDefaultPriorityScoreV0
	}
	config.OccurredAt = strings.TrimSpace(config.OccurredAt)
	config.RequestedBy = strings.TrimSpace(config.RequestedBy)
	if config.RequestedBy == "" {
		config.RequestedBy = ExternalWorkRunDefaultRequestedByV0
	}
	return config
}

func fillExternalWorkRunEnvelopeV0(
	request StartExternalWorkRunRequestV0,
	config StartExternalWorkRunConfigV0,
) StartExternalWorkRunRequestV0 {
	change := request.AppChangeRequest
	if request.RequestID == "" {
		request.RequestID = firstExternalWorkRunStringV0(change.RequestID, change.ChangeRef)
	}
	if request.CorrelationID == "" {
		request.CorrelationID = firstExternalWorkRunStringV0(change.CorrelationID, request.RequestID)
	}
	if request.ProjectRef == "" {
		request.ProjectRef = compactExternalWorkRunRefV0(firstExternalWorkProjectRefV0(change))
	}
	if request.RunRef == "" {
		request.RunRef = "run-external-work-" + compactExternalWorkRunRefV0(
			externalWorkRunRefBasisV0(request, change),
		)
	}
	if request.AppSpecRef == "" {
		request.AppSpecRef = "app-spec-external-work-" + compactExternalWorkRunRefV0(request.ProjectRef)
	}
	if request.QueueRef == "" {
		request.QueueRef = config.QueueRef
	}
	if request.PriorityScore <= 0 {
		request.PriorityScore = config.DefaultPriorityScore
	}
	if request.OccurredAt == "" {
		request.OccurredAt = config.OccurredAt
	}
	if request.RequestedBy == "" {
		request.RequestedBy = config.RequestedBy
	}
	return request
}

func firstExternalWorkProjectRefV0(change orquestaappchange.AppChangeRequestV0) string {
	if change.ExternalWork != nil {
		if value := strings.TrimSpace(change.ExternalWork.ProjectRef); value != "" {
			return value
		}
	}
	return firstExternalWorkRunStringV0(change.AppRef, "external-work")
}

func externalWorkRunRefBasisV0(
	request StartExternalWorkRunRequestV0,
	change orquestaappchange.AppChangeRequestV0,
) string {
	parts := []string{request.ProjectRef}
	if change.ExternalWork != nil {
		if value := strings.TrimSpace(change.ExternalWork.JobRef); value != "" {
			parts = append(parts, value)
		}
	}
	if change.ChangeRef != "" {
		parts = append(parts, change.ChangeRef)
	}
	if len(parts) == 1 {
		parts = append(parts, firstExternalWorkRunStringV0(change.RequestID, request.RequestID, "request"))
	}
	return strings.Join(parts, "-")
}

func firstExternalWorkRunStringV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func compactExternalWorkRunRefV0(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '_', r == '.', r == '-':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "external-work"
	}
	return out
}

func compactExternalWorkRunStringsV0(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if out == nil {
		return []string{}
	}
	return out
}
