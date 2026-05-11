package orquestaapprunner

import (
	"strings"
	"unicode"
)

func normalizePrepareAppOrchestrationRequestV0(
	request PrepareAppOrchestrationRequestV0,
) PrepareAppOrchestrationRequestV0 {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.ProjectRef = strings.TrimSpace(request.ProjectRef)
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.RequestedBy = strings.TrimSpace(request.RequestedBy)
	if request.RunRef == "" && request.AppSpec.SpecID != "" {
		request.RunRef = "run-" + safeAppRunnerRefPartV0(request.AppSpec.SpecID)
	}
	if request.ProjectRef == "" {
		request.ProjectRef = "project-" + safeAppRunnerRefPartV0(firstAppRunnerValueV0(
			request.AppSpec.App.Slug,
			request.AppSpec.SpecID,
		))
	}
	if request.OccurredAt == "" {
		request.OccurredAt = strings.TrimSpace(request.AppSpec.CreatedAt)
	}
	if request.CorrelationID == "" && request.RunRef != "" {
		request.CorrelationID = "corr-" + safeAppRunnerRefPartV0(request.RunRef)
	}
	if request.RequestedBy == "" {
		request.RequestedBy = "orquesta-app-runner"
	}
	return request
}

func safeAppRunnerRefPartV0(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
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

func firstAppRunnerValueV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func compactAppRunnerRefsV0(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
