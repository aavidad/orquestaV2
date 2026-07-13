package orquestaweb

import (
	"encoding/json"
	"net/http"
)

const WebAutoprogrammingPageEndpointV0 = "/autoprogramming"

type AutoprogrammingWebEndpointV0 struct {
	StatusClient AutoprogrammingStatusClientV0
}

func NewAutoprogrammingWebEndpointV0(client AutoprogrammingStatusClientV0) AutoprogrammingWebEndpointV0 {
	return AutoprogrammingWebEndpointV0{StatusClient: client}
}

func (endpoint AutoprogrammingWebEndpointV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := validateWebPublicQueryV0(r); err != nil {
		writeAutoprogrammingStatusV0(w, http.StatusBadRequest,
			newWebAutoprogrammingStatusErrorV0("", WebAutoprogrammingStatusErrRespuestaInvalidaV0))
		return
	}
	if handleWebPublicHTTPOptionsV0(w, r, http.MethodGet) {
		return
	}
	if r.Method != http.MethodGet {
		setWebPublicHTTPAllowV0(w, http.MethodGet)
		http.Error(w, "metodo no permitido", http.StatusMethodNotAllowed)
		return
	}
	if !webRequestWantsHTMLV0(r) && r.URL != nil && r.URL.RawQuery != "" {
		endpoint.handleStatusV0(w, r, autoprogrammingStatusQueryFromURLV0(r))
		return
	}
	writeWebHTMLStringResponseV0(w, http.StatusOK, autoprogrammingHTMLV0(), "es")
}

func (endpoint AutoprogrammingWebEndpointV0) handleStatusV0(
	w http.ResponseWriter,
	r *http.Request,
	query WebAutoprogrammingStatusQueryV0,
) {
	if endpoint.StatusClient == nil {
		writeAutoprogrammingStatusV0(w, http.StatusServiceUnavailable,
			newWebAutoprogrammingStatusErrorV0(query.Locale, WebAutoprogrammingStatusErrTransporteV0))
		return
	}
	viewModel, err := endpoint.StatusClient.ConsultarAutoprogrammingStatus(r.Context(), query)
	if err != nil {
		writeAutoprogrammingStatusV0(w, http.StatusBadGateway,
			newWebAutoprogrammingStatusErrorV0(query.Locale, WebAutoprogrammingStatusErrTransporteV0))
		return
	}
	status := http.StatusOK
	if viewModel.Estado == WebAutoprogrammingPrepareRunEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeAutoprogrammingStatusV0(w, status, viewModel)
}

func autoprogrammingStatusQueryFromURLV0(r *http.Request) WebAutoprogrammingStatusQueryV0 {
	values := webPublicQueryValuesV0(r)
	return WebAutoprogrammingStatusQueryV0{
		RequestID:            values.Get("request_id"),
		CorrelationID:        values.Get("correlation_id"),
		Locale:               values.Get("locale"),
		OccurredAt:           values.Get("occurred_at"),
		RunRef:               values.Get("run_ref"),
		AppRef:               values.Get("app_ref"),
		ExternalJobRef:       values.Get("external_job_ref"),
		QueueRef:             values.Get("queue_ref"),
		AppRefs:              queryValuesV0(r, "app_refs"),
		QueueLimit:           intQueryValueV0(values.Get("queue_limit")),
		IncludeProcessRefs:   boolQueryValueV0(values.Get("include_process_refs")),
		IncludeAgentProgress: boolQueryValueV0(values.Get("include_agent_progress")),
		IncludeAgentUsage:    boolQueryValueV0(values.Get("include_agent_usage")),
		ScopeMode:            values.Get("scope_mode"),
		Scope:                values.Get("scope"),
	}
}

func newWebAutoprogrammingStatusErrorV0(locale string, code string) WebAutoprogrammingStatusViewModelV0 {
	return WebAutoprogrammingStatusViewModelV0{
		SchemaVersion: "web_autoprogramming_status.v0",
		Locale:        normalizeDirectorStatsLocaleV0(locale),
		Estado:        WebAutoprogrammingPrepareRunEstadoErrorV0,
		ErroresPublicos: []WebAutoprogrammingPrepareRunPublicIssueV0{
			{Code: code},
		},
	}
}

func writeAutoprogrammingStatusV0(w http.ResponseWriter, status int, viewModel WebAutoprogrammingStatusViewModelV0) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(viewModel)
}
