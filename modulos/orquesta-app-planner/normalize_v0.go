package orquestaappplanner

import "strings"

func normalizeAppPlanRequestV0(request AppPlanRequestV0) AppPlanRequestV0 {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.AppRef = strings.TrimSpace(request.AppRef)
	request.AppName = strings.TrimSpace(request.AppName)
	request.AppKind = strings.TrimSpace(request.AppKind)
	request.Scale = strings.TrimSpace(request.Scale)
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.RequestedBy = strings.TrimSpace(request.RequestedBy)
	request.Locale = strings.TrimSpace(request.Locale)
	if request.AppKind == "" {
		request.AppKind = "go_api_web"
	}
	if request.RequestedBy == "" {
		request.RequestedBy = "orquesta-app-planner"
	}
	if request.Locale == "" {
		request.Locale = "es-ES"
	}
	if request.Scale == "" {
		request.Scale = AppPlanScaleStandardV0
	}
	return request
}

func compactAppPlannerStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func appPlannerStringInSetV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}
