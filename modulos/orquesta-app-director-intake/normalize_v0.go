package orquestaappdirectorintake

import "strings"

func normalizePrepareAppDirectorIntakeRequestV0(
	request PrepareAppDirectorIntakeRequestV0,
) PrepareAppDirectorIntakeRequestV0 {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.ProjectRef = strings.TrimSpace(request.ProjectRef)
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.RequestedBy = strings.TrimSpace(request.RequestedBy)
	if request.RunRef == "" && request.AppSpec.SpecID != "" {
		request.RunRef = "run-" + safeDirectorIntakeRefPartV0(request.AppSpec.SpecID)
	}
	if request.ProjectRef == "" {
		request.ProjectRef = "project-" + safeDirectorIntakeRefPartV0(firstDirectorIntakeValueV0(
			request.AppSpec.App.Slug,
			request.AppSpec.SpecID,
		))
	}
	if request.OccurredAt == "" {
		request.OccurredAt = strings.TrimSpace(request.AppSpec.CreatedAt)
	}
	if request.CorrelationID == "" && request.RunRef != "" {
		request.CorrelationID = "corr-" + safeDirectorIntakeRefPartV0(request.RunRef)
	}
	if request.RequestedBy == "" {
		request.RequestedBy = "orquesta-app-director-intake"
	}
	return request
}

func safeDirectorIntakeRefPartV0(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var b strings.Builder
	lastDash := false
	for i := 0; i < len(value); i++ {
		c := value[i]
		if isDirectorIntakeASCIILetterV0(c) || (c >= '0' && c <= '9') {
			b.WriteByte(c)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "ref"
	}
	return out
}

func firstDirectorIntakeValueV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func compactDirectorIntakeStringsV0(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		result = append(result, trimmed)
	}
	return result
}

func isDirectorIntakeASCIILetterV0(value byte) bool {
	return (value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z')
}
