package orquestaweb

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

type RunQueueWebEndpointV0 struct {
	Client RunQueueClientV0
}

type WebRunQueuePageV0 struct {
	SchemaVersion string                 `json:"schema_version"`
	Locale        string                 `json:"locale"`
	Query         WebRunQueueQueryV0     `json:"query"`
	Refresh       WebRunQueueRefreshV0   `json:"refresh"`
	ViewModel     WebRunQueueViewModelV0 `json:"view_model"`
}

type WebRunQueueRefreshV0 struct {
	Enabled        bool   `json:"enabled"`
	Method         string `json:"method,omitempty"`
	Href           string `json:"href,omitempty"`
	IntervalMillis int    `json:"interval_millis,omitempty"`
}

func NewRunQueueWebEndpointV0(client RunQueueClientV0) RunQueueWebEndpointV0 {
	return RunQueueWebEndpointV0{Client: client}
}

func (endpoint RunQueueWebEndpointV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := validateWebPublicQueryV0(r); err != nil {
		writeRunQueuePageV0(w, http.StatusBadRequest, endpoint.pageV0(WebRunQueueQueryV0{},
			NewWebRunQueueErrorViewModelV0("", WebRunQueueErrRespuestaInvalidaV0)))
		return
	}
	switch r.Method {
	case http.MethodGet:
		if webRequestWantsHTMLV0(r) {
			writeWebHTMLStringResponseV0(w, http.StatusOK, runQueueHTMLV0(), "es")
			return
		}
		endpoint.handleRunQueueV0(w, r, runQueueQueryFromURLV0(r))
	case http.MethodPost:
		query, err := decodeRunQueueQueryV0(r)
		if err != nil {
			writeRunQueuePageV0(w, http.StatusBadRequest, endpoint.pageV0(query,
				NewWebRunQueueErrorViewModelV0(query.Locale, WebRunQueueErrRespuestaInvalidaV0)))
			return
		}
		endpoint.handleRunQueueV0(w, r, query)
	case http.MethodOptions:
		handleWebPublicHTTPOptionsV0(w, r, http.MethodGet, http.MethodPost)
	default:
		setWebPublicHTTPAllowV0(w, http.MethodGet, http.MethodPost)
		writeRunQueuePageV0(w, http.StatusMethodNotAllowed, endpoint.pageV0(WebRunQueueQueryV0{},
			NewWebRunQueueErrorViewModelV0("", WebNuevaAppErrMetodoNoSoportadoV0)))
	}
}

func (endpoint RunQueueWebEndpointV0) handleRunQueueV0(
	w http.ResponseWriter,
	r *http.Request,
	query WebRunQueueQueryV0,
) {
	query = normalizeRunQueueQueryV0(query)
	if endpoint.Client == nil {
		writeRunQueuePageV0(w, http.StatusServiceUnavailable, endpoint.pageV0(query,
			NewWebRunQueueErrorViewModelV0(query.Locale, WebNuevaAppErrTransporteNoConfiguradoV0)))
		return
	}
	vm, err := endpoint.Client.ConsultarRunQueue(r.Context(), query)
	if err != nil {
		writeRunQueuePageV0(w, http.StatusBadGateway, endpoint.pageV0(query,
			NewWebRunQueueErrorViewModelV0(query.Locale, WebRunQueueErrTransporteV0)))
		return
	}
	status := http.StatusOK
	if vm.Estado == WebRunQueueEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeRunQueuePageV0(w, status, endpoint.pageV0(query, vm))
}

func (endpoint RunQueueWebEndpointV0) pageV0(
	query WebRunQueueQueryV0,
	vm WebRunQueueViewModelV0,
) WebRunQueuePageV0 {
	return WebRunQueuePageV0{
		SchemaVersion: "web_run_queue_page.v0",
		Locale:        normalizeDirectorStatsLocaleV0(query.Locale),
		Query:         query,
		Refresh:       runQueueRefreshV0(query),
		ViewModel:     vm,
	}
}

func runQueueRefreshV0(query WebRunQueueQueryV0) WebRunQueueRefreshV0 {
	if query.Action != WebRunQueueActionRankV0 {
		return WebRunQueueRefreshV0{}
	}
	values := url.Values{}
	values.Set("action", WebRunQueueActionRankV0)
	values.Set("locale", normalizeDirectorStatsLocaleV0(query.Locale))
	if query.QueueRef != "" {
		values.Set("queue_ref", query.QueueRef)
	}
	if query.Limit > 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}
	return WebRunQueueRefreshV0{
		Enabled:        true,
		Method:         http.MethodGet,
		Href:           WebRunQueuePageEndpointV0 + "?" + values.Encode(),
		IntervalMillis: WebDirectorStatsRefreshIntervalMsV0,
	}
}

func decodeRunQueueQueryV0(r *http.Request) (WebRunQueueQueryV0, error) {
	contentType := r.Header.Get("Content-Type")
	if webControlContentTypeAllowsJSONV0(contentType) {
		var query WebRunQueueQueryV0
		return query, decodeWebControlJSONV0(nil, r, &query)
	}
	if webControlContentTypeAllowsFormV0(contentType) {
		if err := parseWebControlFormV0(nil, r); err != nil {
			return WebRunQueueQueryV0{}, err
		}
		return runQueueQueryFromValuesV0(r.Form), nil
	}
	return WebRunQueueQueryV0{}, errUnsupportedDirectorStatsContentV0{}
}

func writeRunQueuePageV0(w http.ResponseWriter, status int, page WebRunQueuePageV0) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(page)
}
