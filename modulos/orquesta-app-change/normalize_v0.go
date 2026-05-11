package orquestaappchange

import "strings"

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
	request.MetadataRefs = compactAppChangeStringsV0(request.MetadataRefs)
	if request.RequestID == "" {
		request.RequestID = request.ChangeRef
	}
	if request.CorrelationID == "" {
		request.CorrelationID = request.RequestID
	}
	return request
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
