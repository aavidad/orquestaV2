package orquestaweb

import orquestamcp "orquesta/modulos/orquesta-mcp"

const (
	WebRunQueuePageEndpointV0    = "/run-queue"
	WebRunQueueInboundEndpointV0 = "/api/v0/runs/queue/priority"

	WebRunQueueEstadoOKV0    = "ok"
	WebRunQueueEstadoErrorV0 = "error"
	WebRunQueueActionRankV0  = "rank"
	WebRunQueueActionSetV0   = "set_priority"

	WebRunQueueErrTransporteV0        = "run_queue_error_transporte"
	WebRunQueueErrRespuestaInvalidaV0 = "run_queue_respuesta_invalida"
)

type WebRunQueueQueryV0 struct {
	RequestID      string   `json:"request_id,omitempty"`
	CorrelationID  string   `json:"correlation_id,omitempty"`
	Locale         string   `json:"locale,omitempty"`
	Action         string   `json:"action,omitempty"`
	QueueRef       string   `json:"queue_ref,omitempty"`
	AppRefs        []string `json:"app_refs,omitempty"`
	RunRef         string   `json:"run_ref,omitempty"`
	AppRef         string   `json:"app_ref,omitempty"`
	PriorityScore  int      `json:"priority_score,omitempty"`
	RequestedBy    string   `json:"requested_by,omitempty"`
	Reason         string   `json:"reason,omitempty"`
	IdempotencyKey string   `json:"idempotency_key,omitempty"`
	Limit          int      `json:"limit,omitempty"`
	OccurredAt     string   `json:"occurred_at,omitempty"`
}

type WebRunQueueViewModelV0 struct {
	SchemaVersion   string                     `json:"schema_version"`
	Locale          string                     `json:"locale,omitempty"`
	Estado          string                     `json:"estado"`
	Action          string                     `json:"action,omitempty"`
	QueueRef        string                     `json:"queue_ref,omitempty"`
	Count           int                        `json:"count"`
	Ranked          []WebRunQueueCandidateV0   `json:"ranked,omitempty"`
	Updated         *WebRunQueueCandidateV0    `json:"updated,omitempty"`
	ErroresPublicos []WebRunQueuePublicIssueV0 `json:"errores_publicos,omitempty"`
}

type WebRunQueueCandidateV0 struct {
	Rank             int      `json:"rank"`
	RunRef           string   `json:"run_ref"`
	AppRef           string   `json:"app_ref"`
	Status           string   `json:"status,omitempty"`
	PriorityScore    int      `json:"priority_score"`
	AgingBoost       int      `json:"aging_boost,omitempty"`
	UpdatedAt        string   `json:"updated_at,omitempty"`
	FairnessGroupRef string   `json:"fairness_group_ref,omitempty"`
	EvidenceRefs     []string `json:"evidence_refs,omitempty"`
}

type WebRunQueuePublicIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

func NewWebRunQueueViewModelV0(
	locale string,
	result orquestamcp.MCPRunQueuePriorityToolResultV0,
) WebRunQueueViewModelV0 {
	vm := WebRunQueueViewModelV0{
		SchemaVersion:   "web_run_queue_panel.v0",
		Locale:          normalizeDirectorStatsLocaleV0(locale),
		Estado:          runQueueEstadoV0(result.Estado),
		Action:          trimV0(result.Action),
		QueueRef:        trimV0(result.QueueRef),
		Count:           result.Count,
		Ranked:          webRunQueueCandidatesV0(result.Ranked),
		ErroresPublicos: webRunQueueIssuesV0(result.Errores),
	}
	if result.Updated != nil {
		updated := webRunQueueCandidateV0(*result.Updated)
		vm.Updated = &updated
	}
	return vm
}

func NewWebRunQueueErrorViewModelV0(locale string, code string) WebRunQueueViewModelV0 {
	return WebRunQueueViewModelV0{
		SchemaVersion: "web_run_queue_panel.v0",
		Locale:        normalizeDirectorStatsLocaleV0(locale),
		Estado:        WebRunQueueEstadoErrorV0,
		ErroresPublicos: []WebRunQueuePublicIssueV0{{
			Code:    trimV0(code),
			Message: trimV0(code),
		}},
	}
}

func runQueueEstadoV0(value string) string {
	if trimV0(value) == WebRunQueueEstadoOKV0 {
		return WebRunQueueEstadoOKV0
	}
	return WebRunQueueEstadoErrorV0
}

func webRunQueueCandidatesV0(
	values []orquestamcp.MCPRunQueueRankedCandidateCompactV0,
) []WebRunQueueCandidateV0 {
	out := make([]WebRunQueueCandidateV0, 0, len(values))
	for _, value := range values {
		out = append(out, webRunQueueCandidateV0(value))
	}
	return out
}

func webRunQueueCandidateV0(
	value orquestamcp.MCPRunQueueRankedCandidateCompactV0,
) WebRunQueueCandidateV0 {
	return WebRunQueueCandidateV0{
		Rank:             value.Rank,
		RunRef:           trimV0(value.RunRef),
		AppRef:           trimV0(value.AppRef),
		Status:           trimV0(value.Status),
		PriorityScore:    value.PriorityScore,
		AgingBoost:       value.AgingBoost,
		UpdatedAt:        trimV0(value.UpdatedAt),
		FairnessGroupRef: trimV0(value.FairnessGroupRef),
		EvidenceRefs:     compactStringsV0(value.EvidenceRefs),
	}
}

func webRunQueueIssuesV0(values []orquestamcp.MCPValidationIssueV0) []WebRunQueuePublicIssueV0 {
	out := make([]WebRunQueuePublicIssueV0, 0, len(values))
	for _, value := range values {
		out = append(out, WebRunQueuePublicIssueV0{
			Code:    trimV0(value.Code),
			Field:   trimV0(value.Field),
			Message: trimV0(value.Message),
		})
	}
	return out
}
