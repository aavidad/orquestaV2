package orquestaweb

import (
	"encoding/json"
	"net/http"
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
	ViewModel     WebDirectorStatsViewModelV0 `json:"view_model"`
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
		ViewModel:     vm,
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
