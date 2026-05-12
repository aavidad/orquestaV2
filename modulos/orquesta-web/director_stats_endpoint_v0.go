package orquestaweb

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
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
	switch r.Method {
	case http.MethodGet:
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
	default:
		w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPost}, ", "))
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
	if trimV0(query.RunRef) == "" {
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
	if query.RunRef != "" {
		query.IncludeAgentProgress = true
	}
	return query
}

func directorStatsRefreshV0(
	locale string,
	query WebDirectorStatsQueryV0,
) WebDirectorStatsRefreshV0 {
	runRef := trimDirectorStatsV0(query.RunRef)
	if runRef == "" {
		return WebDirectorStatsRefreshV0{}
	}
	values := url.Values{}
	values.Set("run_ref", runRef)
	values.Set("locale", normalizeDirectorStatsLocaleV0(firstDirectorStatsNonEmptyV0(query.Locale, locale)))
	values.Set("include_agent_progress", "true")
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
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if contentType == "" || strings.Contains(contentType, "application/json") {
		var query WebDirectorStatsQueryV0
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&query); err != nil {
			return query, err
		}
		return query, nil
	}
	if strings.Contains(contentType, "application/x-www-form-urlencoded") || strings.Contains(contentType, "multipart/form-data") {
		if err := r.ParseForm(); err != nil {
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
