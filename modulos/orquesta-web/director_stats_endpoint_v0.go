package orquestaweb

import (
	"encoding/json"
	"net/http"
	"net/url"
)

type DirectorStatsWebEndpointV0 struct {
	Client  DirectorStatsClientV0
	Catalog NuevaAppI18nCatalogV0
}

type WebDirectorStatsPageV0 struct {
	SchemaVersion string                      `json:"schema_version"`
	Locale        string                      `json:"locale"`
	Query         WebDirectorStatsQueryV0     `json:"query"`
	Refresh       WebDirectorStatsRefreshV0   `json:"refresh"`
	ViewModel     WebDirectorStatsViewModelV0 `json:"view_model"`
}

type WebDirectorStatsRefreshV0 struct {
	Enabled        bool   `json:"enabled"`
	Method         string `json:"method,omitempty"`
	Href           string `json:"href,omitempty"`
	IntervalMillis int    `json:"interval_millis,omitempty"`
}

func NewDirectorStatsWebEndpointV0(client DirectorStatsClientV0) DirectorStatsWebEndpointV0 {
	return DirectorStatsWebEndpointV0{
		Client:  client,
		Catalog: NewNuevaAppI18nCatalogV0(),
	}
}

func (endpoint DirectorStatsWebEndpointV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	catalog := endpoint.catalog()
	if err := validateWebPublicQueryV0(r); err != nil {
		writeDirectorStatsPageV0(w, http.StatusBadRequest, endpoint.page(NuevaAppI18nDefaultLocaleV0, WebDirectorStatsQueryV0{},
			NewWebDirectorStatsErrorViewModelV0("", WebDirectorStatsErrRespuestaInvalidaV0)))
		return
	}
	switch r.Method {
	case http.MethodGet:
		if webRequestWantsHTMLV0(r) {
			writeWebHTMLStringResponseV0(w, http.StatusOK, directorStatsHTMLV0(), "es")
			return
		}
		endpoint.handleDirectorStats(w, r, directorStatsQueryFromURLV0(r))
	case http.MethodPost:
		query, err := decodeDirectorStatsQueryV0(r)
		if err != nil {
			locale := localeFromNuevaAppRequestV0(r, catalog)
			writeDirectorStatsPageV0(w, http.StatusBadRequest, endpoint.page(locale, query,
				NewWebDirectorStatsErrorViewModelV0(query.RunRef, WebDirectorStatsErrRespuestaInvalidaV0)))
			return
		}
		endpoint.handleDirectorStats(w, r, query)
	case http.MethodOptions:
		handleWebPublicHTTPOptionsV0(w, r, http.MethodGet, http.MethodPost)
	default:
		setWebPublicHTTPAllowV0(w, http.MethodGet, http.MethodPost)
		locale := localeFromNuevaAppRequestV0(r, catalog)
		writeDirectorStatsPageV0(w, http.StatusMethodNotAllowed, endpoint.page(locale, WebDirectorStatsQueryV0{},
			NewWebDirectorStatsErrorViewModelV0("", WebNuevaAppErrMetodoNoSoportadoV0)))
	}
}

func (endpoint DirectorStatsWebEndpointV0) handleDirectorStats(
	w http.ResponseWriter,
	r *http.Request,
	query WebDirectorStatsQueryV0,
) {
	locale := localeFromNuevaAppRequestV0(r, endpoint.catalog())
	if trimV0(query.RunRef) == "" && trimV0(query.ExternalJobRef) == "" {
		vm := NewWebDirectorStatsErrorViewModelV0(query.RunRef, WebDirectorStatsErrRunRefRequeridoV0)
		writeDirectorStatsPageV0(w, http.StatusBadRequest, endpoint.page(locale, query, vm))
		return
	}
	query = directorStatsQueryWithAgentProgressV0(query)
	if endpoint.Client == nil {
		vm := NewWebDirectorStatsErrorViewModelV0(query.RunRef, WebNuevaAppErrTransporteNoConfiguradoV0)
		writeDirectorStatsPageV0(w, http.StatusServiceUnavailable, endpoint.page(locale, query, vm))
		return
	}
	vm, err := endpoint.Client.ConsultarDirectorStats(r.Context(), query)
	if err != nil {
		vm = NewWebDirectorStatsErrorViewModelV0(query.RunRef, WebDirectorStatsErrTransporteV0)
		writeDirectorStatsPageV0(w, http.StatusBadGateway, endpoint.page(locale, query, vm))
		return
	}
	writeDirectorStatsPageV0(w, http.StatusOK, endpoint.page(locale, query, vm))
}

func (endpoint DirectorStatsWebEndpointV0) page(
	locale string,
	query WebDirectorStatsQueryV0,
	vm WebDirectorStatsViewModelV0,
) WebDirectorStatsPageV0 {
	return WebDirectorStatsPageV0{
		SchemaVersion: WebDirectorStatsPageSchemaV0,
		Locale:        endpoint.catalog().normalizeLocale(locale),
		Query:         query,
		Refresh:       directorStatsRefreshV0(locale, query),
		ViewModel:     vm,
	}
}

func directorStatsQueryWithAgentProgressV0(query WebDirectorStatsQueryV0) WebDirectorStatsQueryV0 {
	query.RunRef = trimDirectorStatsV0(query.RunRef)
	query.AppRef = trimDirectorStatsV0(query.AppRef)
	query.ExternalJobRef = trimDirectorStatsV0(query.ExternalJobRef)
	if query.RunRef != "" || query.ExternalJobRef != "" {
		query.IncludeProcessRefs = true
		query.IncludeAgentProgress = true
	}
	return query
}

func directorStatsRefreshV0(
	locale string,
	query WebDirectorStatsQueryV0,
) WebDirectorStatsRefreshV0 {
	runRef := trimDirectorStatsV0(query.RunRef)
	externalJobRef := trimDirectorStatsV0(query.ExternalJobRef)
	if runRef == "" && externalJobRef == "" {
		return WebDirectorStatsRefreshV0{}
	}
	values := url.Values{}
	if runRef != "" {
		values.Set("run_ref", runRef)
	}
	values.Set("locale", normalizeDirectorStatsLocaleV0(firstDirectorStatsNonEmptyV0(query.Locale, locale)))
	values.Set("include_process_refs", "true")
	values.Set("include_agent_progress", "true")
	if query.AppRef != "" {
		values.Set("app_ref", trimDirectorStatsV0(query.AppRef))
	}
	if externalJobRef != "" {
		values.Set("external_job_ref", externalJobRef)
	}
	if query.IncludeAgentUsage {
		values.Set("include_agent_usage", "true")
	}
	return WebDirectorStatsRefreshV0{
		Enabled:        true,
		Method:         http.MethodGet,
		Href:           WebDirectorStatsPageEndpointV0 + "?" + values.Encode(),
		IntervalMillis: WebDirectorStatsRefreshIntervalMsV0,
	}
}

func (endpoint DirectorStatsWebEndpointV0) catalog() NuevaAppI18nCatalogV0 {
	if endpoint.Catalog.messages == nil || endpoint.Catalog.defaultLocale == "" {
		return NewNuevaAppI18nCatalogV0()
	}
	return endpoint.Catalog
}

func decodeDirectorStatsQueryV0(r *http.Request) (WebDirectorStatsQueryV0, error) {
	contentType := r.Header.Get("Content-Type")
	if webControlContentTypeAllowsJSONV0(contentType) {
		var query WebDirectorStatsQueryV0
		if err := decodeWebControlJSONV0(nil, r, &query); err != nil {
			return query, err
		}
		return query, nil
	}
	if webControlContentTypeAllowsFormV0(contentType) {
		if err := parseWebControlFormV0(nil, r); err != nil {
			return WebDirectorStatsQueryV0{}, err
		}
		return directorStatsQueryFromValuesV0(r.Form), nil
	}
	return WebDirectorStatsQueryV0{}, errUnsupportedDirectorStatsContentV0{}
}

func writeDirectorStatsPageV0(w http.ResponseWriter, status int, page WebDirectorStatsPageV0) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(page)
}

type errUnsupportedDirectorStatsContentV0 struct{}

func (errUnsupportedDirectorStatsContentV0) Error() string {
	return "content_type_no_soportado"
}
