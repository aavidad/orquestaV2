package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const (
	DirectorAgentUsageQuotaUnknownV0       = "unknown"
	DirectorAgentUsageQuotaNotConfiguredV0 = "not_configured"
	DirectorAgentUsageQuotaAvailableV0     = "available"
	DirectorAgentUsageQuotaLimitedV0       = "limited"
	DirectorAgentUsageQuotaExhaustedV0     = "exhausted"
)

type AgentUsageStatsProviderPortV0 interface {
	BuildAgentUsageStatsV0(
		context.Context,
		AgentUsageStatsRequestV0,
	) ([]AgentUsageStatsObservationV0, error)
}

type AgentUsageStatsRequestV0 struct {
	Run           orquestacoreworkflow.OrchestrationRunV0
	CorrelationID string
	EvidenceRefs  []string
}

type AgentUsageStatsObservationV0 struct {
	AgentRequestID   string   `json:"agent_request_id"`
	RuntimeKind      string   `json:"runtime_kind,omitempty"`
	ConnectorRef     string   `json:"connector_ref,omitempty"`
	ProfileRef       string   `json:"profile_ref,omitempty"`
	ModelAlias       string   `json:"model_alias,omitempty"`
	CapacityLevel    string   `json:"capacity_level,omitempty"`
	ReasoningEffort  string   `json:"reasoning_effort,omitempty"`
	QuotaStatus      string   `json:"quota_status,omitempty"`
	QuotaRemaining   int64    `json:"quota_remaining,omitempty"`
	QuotaLimit       int64    `json:"quota_limit,omitempty"`
	QuotaResetAt     string   `json:"quota_reset_at,omitempty"`
	PromptTokens     int64    `json:"prompt_tokens,omitempty"`
	CompletionTokens int64    `json:"completion_tokens,omitempty"`
	TotalTokens      int64    `json:"total_tokens,omitempty"`
	CostMicros       int64    `json:"cost_micros,omitempty"`
	EvidenceRefs     []string `json:"evidence_refs,omitempty"`
}

type DirectorAgentUsageStatsV0 struct {
	RuntimeKind      string   `json:"runtime_kind,omitempty"`
	ConnectorRef     string   `json:"connector_ref,omitempty"`
	ProfileRef       string   `json:"profile_ref,omitempty"`
	ModelAlias       string   `json:"model_alias,omitempty"`
	CapacityLevel    string   `json:"capacity_level,omitempty"`
	ReasoningEffort  string   `json:"reasoning_effort,omitempty"`
	QuotaStatus      string   `json:"quota_status,omitempty"`
	QuotaRemaining   int64    `json:"quota_remaining,omitempty"`
	QuotaLimit       int64    `json:"quota_limit,omitempty"`
	QuotaResetAt     string   `json:"quota_reset_at,omitempty"`
	PromptTokens     int64    `json:"prompt_tokens,omitempty"`
	CompletionTokens int64    `json:"completion_tokens,omitempty"`
	TotalTokens      int64    `json:"total_tokens,omitempty"`
	CostMicros       int64    `json:"cost_micros,omitempty"`
	EvidenceRefs     []string `json:"evidence_refs,omitempty"`
}

func ApplyDirectorAgentUsageStatsV0(
	stats *DirectorRunStatsV0,
	observations []AgentUsageStatsObservationV0,
) {
	if stats == nil || len(observations) == 0 {
		return
	}
	byAgent := latestAgentUsageStatsByAgentV0(observations)
	for index := range stats.Agents {
		observation, ok := byAgent[stats.Agents[index].AgentRequestID]
		if !ok {
			continue
		}
		usage := directorAgentUsageStatsFromObservationV0(observation)
		stats.Agents[index].Usage = &usage
	}
}

func latestAgentUsageStatsByAgentV0(
	observations []AgentUsageStatsObservationV0,
) map[string]AgentUsageStatsObservationV0 {
	out := map[string]AgentUsageStatsObservationV0{}
	for _, observation := range observations {
		agentRef := strings.TrimSpace(observation.AgentRequestID)
		if agentRef != "" {
			out[agentRef] = observation
		}
	}
	return out
}

func directorAgentUsageStatsFromObservationV0(
	observation AgentUsageStatsObservationV0,
) DirectorAgentUsageStatsV0 {
	return DirectorAgentUsageStatsV0{
		RuntimeKind:      strings.TrimSpace(observation.RuntimeKind),
		ConnectorRef:     strings.TrimSpace(observation.ConnectorRef),
		ProfileRef:       strings.TrimSpace(observation.ProfileRef),
		ModelAlias:       strings.TrimSpace(observation.ModelAlias),
		CapacityLevel:    strings.TrimSpace(observation.CapacityLevel),
		ReasoningEffort:  strings.TrimSpace(observation.ReasoningEffort),
		QuotaStatus:      normalizeDirectorAgentQuotaStatusV0(observation.QuotaStatus),
		QuotaRemaining:   nonNegativeInt64V0(observation.QuotaRemaining),
		QuotaLimit:       nonNegativeInt64V0(observation.QuotaLimit),
		QuotaResetAt:     strings.TrimSpace(observation.QuotaResetAt),
		PromptTokens:     nonNegativeInt64V0(observation.PromptTokens),
		CompletionTokens: nonNegativeInt64V0(observation.CompletionTokens),
		TotalTokens:      nonNegativeInt64V0(observation.TotalTokens),
		CostMicros:       nonNegativeInt64V0(observation.CostMicros),
		EvidenceRefs:     compactStringsV0(observation.EvidenceRefs),
	}
}

func normalizeDirectorAgentQuotaStatusV0(status string) string {
	switch strings.TrimSpace(status) {
	case DirectorAgentUsageQuotaAvailableV0,
		DirectorAgentUsageQuotaLimitedV0,
		DirectorAgentUsageQuotaExhaustedV0,
		DirectorAgentUsageQuotaNotConfiguredV0:
		return strings.TrimSpace(status)
	default:
		return DirectorAgentUsageQuotaUnknownV0
	}
}

func nonNegativeInt64V0(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
