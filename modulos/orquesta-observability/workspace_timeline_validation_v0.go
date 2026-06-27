package orquestaobservability

import (
	"fmt"
	"math"
	"strings"
)

func ValidateWorkspaceTimelineQueryV0(query WorkspaceTimelineQueryV0) error {
	var issues []OperationalStatusValidationIssueV0
	add := addOperationalStatusIssueFuncV0(&issues)
	if strings.TrimSpace(query.SchemaVersion) != WorkspaceTimelineQuerySchemaVersionV0 {
		add(ErrWorkspaceTimelineQueryInvalidaV0, "schema_version")
	}
	if strings.TrimSpace(query.RequestID) == "" {
		add(ErrWorkspaceTimelineQueryInvalidaV0, "request_id")
	}
	if strings.TrimSpace(query.CorrelationID) == "" {
		add(ErrWorkspaceTimelineQueryInvalidaV0, "correlation_id")
	}
	if scope := strings.TrimSpace(query.Scope); scope != "" && scope != WorkspaceTimelineScopeWorkspaceV0 {
		add(ErrScopeNoSoportadoV0, "scope")
	}
	validateWorkspaceTimelineTimeWindowV0(query.TimeWindow, "time_window", add)
	validateWorkspaceTimelineSourcesV0(workspaceTimelineQuerySourcesV0(query), "sources", add)
	validateOperationalLimitV0(workspaceTimelineQueryLimitV0(query), "limit", add)
	if len(issues) > 0 {
		return OperationalStatusValidationErrorV0{Issues: issues}
	}
	return nil
}

func ValidateWorkspaceTimelineProjectionV0(projection WorkspaceTimelineProjectionV0) error {
	return ValidateWorkspaceTimelineV0(projection)
}

func ValidateWorkspaceTimelineV0(timeline WorkspaceTimelineV0) error {
	var issues []OperationalStatusValidationIssueV0
	add := addOperationalStatusIssueFuncV0(&issues)
	if strings.TrimSpace(timeline.SchemaVersion) != WorkspaceTimelineSchemaVersionV0 {
		add(ErrWorkspaceTimelineQueryInvalidaV0, "schema_version")
	}
	validateRequiredOperationalRefV0(firstNonEmptyOperationalStatusV0(timeline.TimelineRef, timeline.TimelineID), "timeline_ref", add)
	if !isOccurredAtV0(timeline.GeneratedAt) {
		add(ErrWorkspaceTimelineQueryInvalidaV0, "generated_at")
	}
	validateRequiredOperationalRefV0(timeline.CorrelationID, "correlation_id", add)
	validateOptionalOperationalRefV0(timeline.WorkspaceRef, "workspace_ref", add)
	validateOptionalOperationalRefV0(timeline.ProjectRef, "project_ref", add)
	validateOptionalOperationalRefV0(timeline.AgentRef, "agent_ref", add)
	validateOptionalOperationalRefV0(timeline.TaskRef, "task_ref", add)
	validateOptionalOperationalRefV0(timeline.RunRef, "run_ref", add)
	if workspaceTimelineHasFreshnessV0(timeline.Freshness) {
		if strings.TrimSpace(timeline.Freshness.WatermarkRef) != "" ||
			timeline.Freshness.MaxAgeSeconds != 0 ||
			timeline.Freshness.Partial ||
			timeline.Freshness.Stale {
			validateDiagnosticoFreshnessV0(timeline.Freshness, "freshness", add)
		}
	}
	validateWorkspaceTimelineSourceStatusesV0(timeline.Sources, "sources", add)
	validateWorkspaceTimelineItemsV0(timeline.Items, "items", add)
	validateDiagnosticoContadoresV0(timeline.Counters, "counters", add)
	validateDiagnosticoWarningsV0(timeline.Warnings, "warnings", add)
	validateDiagnosticoPrivacyV0(timeline.Privacy, "privacy", add)
	validateOperationalJSONSizeV0(timeline, maxDiagnosticoJSONBytesV0, "", add)
	if len(issues) > 0 {
		return OperationalStatusValidationErrorV0{Issues: issues}
	}
	return nil
}

func workspaceTimelineQuerySourcesV0(query WorkspaceTimelineQueryV0) []string {
	if len(query.Sources) > 0 {
		return query.Sources
	}
	return query.IncludeSources
}

func workspaceTimelineQueryLimitV0(query WorkspaceTimelineQueryV0) int {
	if query.Page.Limit > 0 {
		return query.Page.Limit
	}
	return query.Limit
}

func validateWorkspaceTimelineTimeWindowV0(
	window OperationalStatusTimeWindowV0,
	prefix string,
	add func(string, string),
) {
	if strings.TrimSpace(window.From) != "" && !isOccurredAtV0(window.From) {
		add(ErrWorkspaceTimelineQueryInvalidaV0, fieldV0(prefix, "from"))
	}
	if strings.TrimSpace(window.To) != "" && !isOccurredAtV0(window.To) {
		add(ErrWorkspaceTimelineQueryInvalidaV0, fieldV0(prefix, "to"))
	}
	if code := forbiddenOperationalTextCodeV0(window.Preset); code != "" {
		add(code, fieldV0(prefix, "preset"))
	}
}

func workspaceTimelineHasFreshnessV0(freshness DiagnosticoFreshnessV0) bool {
	return strings.TrimSpace(freshness.WatermarkRef) != "" ||
		freshness.MaxAgeSeconds != 0 ||
		freshness.Partial ||
		freshness.Stale
}

func validateWorkspaceTimelineSourcesV0(sources []string, prefix string, add func(string, string)) {
	if len(sources) == 0 {
		add(ErrWorkspaceTimelineQueryInvalidaV0, prefix)
		return
	}
	if len(sources) > maxWorkspaceTimelineSourcesV0 {
		add(ErrConsultaDemasiadoAmpliaV0, prefix)
	}
	seen := map[string]bool{}
	for index, source := range sources {
		field := fmt.Sprintf("%s[%d]", prefix, index)
		trimmed := strings.TrimSpace(source)
		if !allowedV0(allowedWorkspaceTimelineSourcesV0, trimmed) || seen[trimmed] {
			add(ErrWorkspaceTimelineQueryInvalidaV0, field)
		}
		seen[trimmed] = true
	}
}

func validateWorkspaceTimelineSourceStatusesV0(
	statuses []WorkspaceTimelineSourceStatusV0,
	prefix string,
	add func(string, string),
) {
	for index, status := range statuses {
		field := fmt.Sprintf("%s[%d]", prefix, index)
		if !allowedV0(allowedWorkspaceTimelineSourcesV0, strings.TrimSpace(status.Source)) {
			add(ErrWorkspaceTimelineQueryInvalidaV0, fieldV0(field, "source"))
		}
		if !allowedV0(allowedWorkspaceTimelineSourceStatusesV0, strings.TrimSpace(status.Status)) {
			add(ErrWorkspaceTimelineQueryInvalidaV0, fieldV0(field, "status"))
		}
		validateOperationalTextV0(status.ReasonCode, fieldV0(field, "reason_code"), maxOperationalTokenRunesV0, false, add)
		validateOperationalRefListV0(status.EvidenceRefs, maxOperationalEvidenceRefsV0, fieldV0(field, "evidence_refs"), add)
	}
}

func validateWorkspaceTimelineItemsV0(items []WorkspaceTimelineItemV0, prefix string, add func(string, string)) {
	if len(items) > maxWorkspaceTimelineEventsV0 {
		add(ErrConsultaDemasiadoAmpliaV0, prefix)
	}
	for index, item := range items {
		field := fmt.Sprintf("%s[%d]", prefix, index)
		validateRequiredOperationalRefV0(item.EventRef, fieldV0(field, "event_ref"), add)
		if !isOccurredAtV0(item.OccurredAt) {
			add(ErrWorkspaceTimelineQueryInvalidaV0, fieldV0(field, "occurred_at"))
		}
		if !allowedV0(allowedWorkspaceTimelineSourcesV0, strings.TrimSpace(item.Source)) {
			add(ErrWorkspaceTimelineQueryInvalidaV0, fieldV0(field, "source"))
		}
		if !allowedV0(allowedWorkspaceTimelineCategoriesV0, strings.TrimSpace(item.Category)) {
			add(ErrWorkspaceTimelineQueryInvalidaV0, fieldV0(field, "category"))
		}
		validateOperationalTextV0(item.Status, fieldV0(field, "status"), maxOperationalTokenRunesV0, false, add)
		validateOperationalI18nKeyV0(item.SummaryKey, fieldV0(field, "summary_key"), add)
		validateOperationalTextV0(item.Summary, fieldV0(field, "summary"), maxOperationalTextRunesV0, false, add)
		validateWorkspaceTimelineRefsV0(item.Refs, fieldV0(field, "refs"), add)
		validateWorkspaceTranscriptClassificationV0(item.Transcript, fieldV0(field, "transcript"), add)
		validateOperationalRefListV0(item.EvidenceRefs, maxOperationalEvidenceRefsV0, fieldV0(field, "evidence_refs"), add)
	}
}

func validateWorkspaceTimelineRefsV0(refs WorkspaceTimelineRefsV0, prefix string, add func(string, string)) {
	validateOptionalOperationalRefV0(refs.RunRef, fieldV0(prefix, "run_ref"), add)
	validateOptionalOperationalRefV0(refs.TaskRef, fieldV0(prefix, "task_ref"), add)
	validateOptionalOperationalRefV0(refs.AgentRef, fieldV0(prefix, "agent_ref"), add)
	validateOptionalOperationalRefV0(refs.ProjectRef, fieldV0(prefix, "project_ref"), add)
	validateOptionalOperationalRefV0(refs.WorktreeRef, fieldV0(prefix, "worktree_ref"), add)
	validateOptionalOperationalRefV0(refs.AuditRef, fieldV0(prefix, "audit_ref"), add)
}

func validateWorkspaceTranscriptClassificationV0(
	classification WorkspaceTranscriptClassificationV0,
	prefix string,
	add func(string, string),
) {
	if !allowedV0(allowedWorkspaceTranscriptClassesV0, strings.TrimSpace(classification.Class)) {
		add(ErrWorkspaceTimelineQueryInvalidaV0, fieldV0(prefix, "class"))
	}
	if math.IsNaN(classification.Confidence) || math.IsInf(classification.Confidence, 0) ||
		classification.Confidence < 0 || classification.Confidence > 1 {
		add(ErrWorkspaceTimelineQueryInvalidaV0, fieldV0(prefix, "confidence"))
	}
}

func firstNonEmptyOperationalStatusV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
