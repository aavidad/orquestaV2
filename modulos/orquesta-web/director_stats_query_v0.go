package orquestaweb

import (
	"net/http"
)

func directorStatsQueryFromURLV0(r *http.Request) WebDirectorStatsQueryV0 {
	return WebDirectorStatsQueryV0{
		RequestID:            r.URL.Query().Get("request_id"),
		CorrelationID:        r.URL.Query().Get("correlation_id"),
		Locale:               r.URL.Query().Get("locale"),
		RunRef:               r.URL.Query().Get("run_ref"),
		OccurredAt:           r.URL.Query().Get("occurred_at"),
		IncludeProcessRefs:   boolQueryValueV0(r.URL.Query().Get("include_process_refs")),
		IncludeAgentProgress: boolQueryValueV0(r.URL.Query().Get("include_agent_progress")),
		IncludeAgentUsage:    boolQueryValueV0(r.URL.Query().Get("include_agent_usage")),
	}
}

func directorStatsQueryFromValuesV0(values map[string][]string) WebDirectorStatsQueryV0 {
	return WebDirectorStatsQueryV0{
		RequestID:            formValueV0(values, "request_id"),
		CorrelationID:        formValueV0(values, "correlation_id"),
		Locale:               formValueV0(values, "locale"),
		RunRef:               formValueV0(values, "run_ref"),
		OccurredAt:           formValueV0(values, "occurred_at"),
		IncludeProcessRefs:   formBoolValueV0(values, "include_process_refs"),
		IncludeAgentProgress: formBoolValueV0(values, "include_agent_progress"),
		IncludeAgentUsage:    formBoolValueV0(values, "include_agent_usage"),
	}
}

func boolQueryValueV0(value string) bool {
	return value == "true" || value == "1" || value == "yes"
}
