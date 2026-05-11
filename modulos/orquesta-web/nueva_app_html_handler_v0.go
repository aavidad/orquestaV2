package orquestaweb

import (
	"net/http"
	"strings"
)

const NuevaAppHTMLHandlerSchemaV0 = "nueva_app_html_handler.v0"

type NuevaAppHTMLHandlerV0 struct {
	Endpoint NuevaAppWebEndpointV0
}

func NewNuevaAppHTMLHandlerV0(client SolicitarNuevaAppClientV0) NuevaAppHTMLHandlerV0 {
	return NuevaAppHTMLHandlerV0{
		Endpoint: NewNuevaAppWebEndpointV0(client),
	}
}

func (handler NuevaAppHTMLHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	endpoint := handler.endpoint()
	catalog := endpoint.catalog()
	switch r.Method {
	case http.MethodGet:
		locale := localeFromNuevaAppRequestV0(r, catalog)
		page := endpoint.page(locale, initialNuevaAppViewModelV0(locale))
		writeNuevaAppHTMLPageV0(w, http.StatusOK, page)
	case http.MethodPost:
		status, page := endpoint.postPage(r, catalog)
		writeNuevaAppHTMLPageV0(w, status, page)
	default:
		w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPost}, ", "))
		locale := localeFromNuevaAppRequestV0(r, catalog)
		vm := nuevaAppWebPublicErrorViewModelV0("", locale, WebNuevaAppEstadoError, WebNuevaAppErrMetodoNoSoportadoV0)
		writeNuevaAppHTMLPageV0(w, http.StatusMethodNotAllowed, endpoint.page(locale, vm))
	}
}

func (handler NuevaAppHTMLHandlerV0) endpoint() NuevaAppWebEndpointV0 {
	if handler.Endpoint.Catalog.messages == nil || handler.Endpoint.Catalog.defaultLocale == "" {
		handler.Endpoint.Catalog = NewNuevaAppI18nCatalogV0()
	}
	return handler.Endpoint
}
