package orquestadirector

import (
	"strings"

	orquestacontext "orquesta/modulos/orquesta-context"
)

func compactLaunchContextStringsV0(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func firstNonEmptyLaunchContextV0(first string, fallback string) string {
	if first = strings.TrimSpace(first); first != "" {
		return first
	}
	return strings.TrimSpace(fallback)
}

func launchContextIssuesFromBundleV0(issues []orquestacontext.ContextBundleIssueV0) []LaunchContextBundleIssueV0 {
	result := make([]LaunchContextBundleIssueV0, 0, len(issues))
	for _, issue := range issues {
		result = append(result, LaunchContextBundleIssueV0{
			Code:    string(issue.Code),
			Field:   issue.Field,
			Message: issue.Message,
		})
	}
	return result
}

func launchContextIssueV0(code string, field string, message string) LaunchContextBundleIssueV0 {
	return LaunchContextBundleIssueV0{Code: code, Field: field, Message: message}
}

func launchContextBundleErrorV0(code string, field string, issues []LaunchContextBundleIssueV0) LaunchContextBundleErrorV0 {
	return LaunchContextBundleErrorV0{Code: code, Field: field, Issues: issues}
}
