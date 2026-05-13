package orquestaappchange

import (
	"context"
	"strings"
)

func ReceiveAppChangeIntentEventV0(
	ctx context.Context,
	event AppChangeIntentEventV0,
	ports AppChangePortsV0,
) (AppChangeResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	event = normalizeAppChangeIntentEventV0(event)
	request := appChangeRequestFromIntentEventV0(event)
	if issues := validateAppChangeIntentEventV0(event); len(issues) > 0 {
		return invalidAppChangeResultV0(request, issues), nil
	}
	if issues := validateAppChangeRequestV0(request); len(issues) > 0 {
		return invalidAppChangeResultV0(request, issues), nil
	}
	record := AppChangeRecordV0{
		Request:     request,
		ReceivedAt:  event.OccurredAt,
		RequestedBy: firstAppChangeValueV0(event.Source, AppChangeDefaultRequestedByV0),
	}
	return requestAppChangeRecordV0(ctx, record, ports)
}

func normalizeAppChangeIntentEventV0(
	event AppChangeIntentEventV0,
) AppChangeIntentEventV0 {
	event.SchemaVersion = strings.TrimSpace(event.SchemaVersion)
	if event.SchemaVersion == "" {
		event.SchemaVersion = AppChangeIntentEventSchemaV0
	}
	event.EventID = strings.TrimSpace(event.EventID)
	event.Source = strings.TrimSpace(event.Source)
	event.OccurredAt = strings.TrimSpace(event.OccurredAt)
	event.RequestID = strings.TrimSpace(event.RequestID)
	event.CorrelationID = strings.TrimSpace(event.CorrelationID)
	event.RunRef = strings.TrimSpace(event.RunRef)
	event.AppRef = strings.TrimSpace(event.AppRef)
	event.ChangeRef = strings.TrimSpace(event.ChangeRef)
	event.ActorRef = strings.TrimSpace(event.ActorRef)
	event.Locale = strings.TrimSpace(event.Locale)
	event.UserIntent = strings.TrimSpace(event.UserIntent)
	event.TargetArea = strings.TrimSpace(event.TargetArea)
	event.CurrentStateRefs = compactAppChangeStringsV0(event.CurrentStateRefs)
	event.Scope = compactAppChangeStringsV0(event.Scope)
	event.AcceptanceCriteria = compactAppChangeStringsV0(event.AcceptanceCriteria)
	event.Constraints = compactAppChangeStringsV0(event.Constraints)
	event.AllowedWriteSet = compactAppChangeStringsV0(event.AllowedWriteSet)
	event.MetadataRefs = compactAppChangeStringsV0(event.MetadataRefs)
	event.ExternalWork = normalizeAppChangeExternalWorkV0(event.ExternalWork)
	if event.ChangeRef == "" && event.EventID != "" {
		event.ChangeRef = "change-ref-" + safeAppChangeRefPartV0(event.EventID)
	}
	if event.RequestID == "" {
		event.RequestID = firstAppChangeValueV0(event.EventID, event.ChangeRef)
	}
	if event.CorrelationID == "" {
		event.CorrelationID = event.RequestID
	}
	return event
}

func validateAppChangeIntentEventV0(
	event AppChangeIntentEventV0,
) []AppChangeIssueV0 {
	var issues []AppChangeIssueV0
	if event.SchemaVersion != AppChangeIntentEventSchemaV0 {
		issues = append(issues, appChangeIssueV0(ErrAppChangeIntentEventSchemaV0, "schema_version"))
	}
	if event.EventID == "" && event.ChangeRef == "" {
		issues = append(issues, appChangeIssueV0(ErrAppChangeIntentEventRefV0, "event_id"))
	}
	if event.EventID != "" && !isCompactAppChangeRefV0(event.EventID) {
		issues = append(issues, appChangeIssueV0(ErrAppChangeIntentEventRefInvalidV0, "event_id"))
	}
	return issues
}

func appChangeRequestFromIntentEventV0(
	event AppChangeIntentEventV0,
) AppChangeRequestV0 {
	return normalizeAppChangeRequestV0(AppChangeRequestV0{
		SchemaVersion:      AppChangeRequestSchemaV0,
		RequestID:          event.RequestID,
		CorrelationID:      event.CorrelationID,
		RunRef:             event.RunRef,
		AppRef:             event.AppRef,
		ChangeRef:          event.ChangeRef,
		ActorRef:           event.ActorRef,
		Locale:             event.Locale,
		UserIntent:         event.UserIntent,
		TargetArea:         event.TargetArea,
		CurrentStateRefs:   event.CurrentStateRefs,
		Scope:              event.Scope,
		AcceptanceCriteria: event.AcceptanceCriteria,
		Constraints:        event.Constraints,
		AllowedWriteSet:    event.AllowedWriteSet,
		MetadataRefs:       append([]string(nil), event.MetadataRefs...),
		ExternalWork:       event.ExternalWork,
	})
}

func safeAppChangeRefPartV0(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var b strings.Builder
	lastDash := false
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
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

func firstAppChangeValueV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
