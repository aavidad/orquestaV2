package orquestaweb

import (
	"strings"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

const WebWorkspaceTimelineSchemaV0 = "web_workspace_timeline.v0"

type WebWorkspaceTimelineViewModelV0 struct {
	SchemaVersion  string                               `json:"schema_version"`
	Locale         string                               `json:"locale"`
	TimelineID     string                               `json:"timeline_id"`
	Scope          string                               `json:"scope"`
	AgentRef       string                               `json:"agent_ref,omitempty"`
	ProjectRef     string                               `json:"project_ref,omitempty"`
	TaskRef        string                               `json:"task_ref,omitempty"`
	Items          []WebWorkspaceTimelineItemV0         `json:"items"`
	Sources        []WebWorkspaceTimelineSourceStatusV0 `json:"sources"`
	Frescura       WebWorkspaceTimelinePageV0           `json:"page"`
	PrivacyOK      bool                                 `json:"privacy_ok"`
	RedactionLevel string                               `json:"redaction_level"`
}

type WebWorkspaceTimelineItemV0 struct {
	Ref                      string `json:"ref"`
	OccurredAt               string `json:"occurred_at"`
	Source                   string `json:"source"`
	Category                 string `json:"category"`
	Summary                  string `json:"summary"`
	RunRef                   string `json:"run_ref,omitempty"`
	AgentRef                 string `json:"agent_ref,omitempty"`
	TranscriptClassification string `json:"transcript_classification"`
}

type WebWorkspaceTimelineSourceStatusV0 struct {
	Source     string `json:"source"`
	Status     string `json:"status"`
	ReasonCode string `json:"reason_code,omitempty"`
}

type WebWorkspaceTimelinePageV0 struct {
	Limit     int    `json:"limit"`
	CursorRef string `json:"cursor_ref,omitempty"`
	HasMore   bool   `json:"has_more"`
}

func NewWebWorkspaceTimelineV0(
	locale string,
	timeline orquestaobservability.WorkspaceTimelineV0,
) WebWorkspaceTimelineViewModelV0 {
	return WebWorkspaceTimelineViewModelV0{
		SchemaVersion:  WebWorkspaceTimelineSchemaV0,
		Locale:         strings.TrimSpace(locale),
		TimelineID:     firstWebWorkspaceTimelineV0(timeline.TimelineID, timeline.TimelineRef),
		Scope:          firstWebWorkspaceTimelineV0(timeline.Scope, orquestaobservability.WorkspaceTimelineScopeWorkspaceV0),
		AgentRef:       firstWebWorkspaceTimelineV0(timeline.Filters.AgentRef, timeline.AgentRef),
		ProjectRef:     firstWebWorkspaceTimelineV0(timeline.Filters.ProjectRef, timeline.ProjectRef),
		TaskRef:        firstWebWorkspaceTimelineV0(timeline.Filters.TaskRef, timeline.TaskRef),
		Items:          webWorkspaceTimelineItemsV0(timeline.Items),
		Sources:        webWorkspaceTimelineSourcesV0(timeline.Sources),
		Frescura:       webWorkspaceTimelinePageV0(timeline.Page),
		PrivacyOK:      webOperationalStatusPrivacyOKV0(timeline.Privacy),
		RedactionLevel: webOperationalStatusRedactionLevelV0(timeline.Privacy),
	}
}

func webWorkspaceTimelineItemsV0(
	items []orquestaobservability.WorkspaceTimelineItemV0,
) []WebWorkspaceTimelineItemV0 {
	out := make([]WebWorkspaceTimelineItemV0, 0, len(items))
	for _, item := range items {
		out = append(out, WebWorkspaceTimelineItemV0{
			Ref:                      firstWebWorkspaceTimelineV0(item.EventRef),
			OccurredAt:               strings.TrimSpace(item.OccurredAt),
			Source:                   strings.TrimSpace(item.Source),
			Category:                 strings.TrimSpace(item.Category),
			Summary:                  strings.TrimSpace(item.Summary),
			RunRef:                   strings.TrimSpace(item.Refs.RunRef),
			AgentRef:                 strings.TrimSpace(item.Refs.AgentRef),
			TranscriptClassification: strings.TrimSpace(item.Transcript.Class),
		})
	}
	return out
}

func firstWebWorkspaceTimelineV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func webWorkspaceTimelineSourcesV0(
	sources []orquestaobservability.WorkspaceTimelineSourceStatusV0,
) []WebWorkspaceTimelineSourceStatusV0 {
	out := make([]WebWorkspaceTimelineSourceStatusV0, 0, len(sources))
	for _, source := range sources {
		out = append(out, WebWorkspaceTimelineSourceStatusV0{
			Source:     strings.TrimSpace(source.Source),
			Status:     strings.TrimSpace(source.Status),
			ReasonCode: strings.TrimSpace(source.ReasonCode),
		})
	}
	return out
}

func webWorkspaceTimelinePageV0(page orquestaobservability.WorkspaceTimelinePageV0) WebWorkspaceTimelinePageV0 {
	return WebWorkspaceTimelinePageV0{
		Limit:     page.Limit,
		CursorRef: strings.TrimSpace(page.CursorRef),
		HasMore:   page.HasMore,
	}
}
