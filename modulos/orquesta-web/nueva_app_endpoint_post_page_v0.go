package orquestaweb

import (
	"errors"
	"net/http"
	"strings"
)

func (endpoint NuevaAppWebEndpointV0) postPage(r *http.Request, catalog NuevaAppI18nCatalogV0) (int, NuevaAppWebPageV0) {
	form, err := decodeNuevaAppWebFormV0(r)
	locale := firstNuevaAppLocaleV0(form.Locale, localeFromNuevaAppRequestV0(r, catalog))
	if err != nil {
		vm := nuevaAppWebPublicErrorViewModelV0(form.RequestID, locale, WebNuevaAppEstadoInvalida, WebNuevaAppErrFormIncompletoV0)
		return http.StatusBadRequest, endpoint.page(locale, vm)
	}
	if isNuevaAppPreviewActionV0(form.Action) {
		return endpoint.postDirectorPreviewPage(r, form, locale)
	}
	if endpoint.DirectorClient != nil {
		return endpoint.postDirectorPage(r, form, locale)
	}
	if endpoint.Client == nil {
		vm := nuevaAppWebPublicErrorViewModelV0(form.RequestID, locale, WebNuevaAppEstadoError, WebNuevaAppErrTransporteNoConfiguradoV0)
		return http.StatusServiceUnavailable, endpoint.page(locale, vm)
	}

	vm, err := endpoint.Client.SolicitarNuevaApp(r.Context(), form)
	if err != nil {
		code := WebNuevaAppErrTransporteV0
		var clientErr WebNuevaAppClientErrorV0
		if errors.As(err, &clientErr) && strings.TrimSpace(clientErr.Code) != "" {
			code = clientErr.Code
		}
		vm = nuevaAppWebPublicErrorViewModelV0(form.RequestID, locale, WebNuevaAppEstadoError, code)
		return http.StatusBadGateway, endpoint.page(locale, vm)
	}
	if vm.Locale != "" {
		locale = vm.Locale
	}
	return http.StatusOK, endpoint.page(locale, vm)
}

func (endpoint NuevaAppWebEndpointV0) postDirectorPreviewPage(
	r *http.Request,
	form WebNuevaAppFormV0,
	locale string,
) (int, NuevaAppWebPageV0) {
	if endpoint.DirectorPreviewClient == nil {
		vm := nuevaAppWebPublicErrorViewModelV0(form.RequestID, locale, WebNuevaAppEstadoError, WebNuevaAppErrTransporteNoConfiguradoV0)
		return http.StatusServiceUnavailable, endpoint.page(locale, vm)
	}
	vm, err := endpoint.DirectorPreviewClient.PreviewDirectorApp(r.Context(), form)
	if err != nil {
		code := WebNuevaAppErrTransporteV0
		var clientErr WebNuevaAppClientErrorV0
		if errors.As(err, &clientErr) && strings.TrimSpace(clientErr.Code) != "" {
			code = clientErr.Code
		}
		vm = nuevaAppWebPublicErrorViewModelV0(form.RequestID, locale, WebNuevaAppEstadoError, code)
		return http.StatusBadGateway, endpoint.page(locale, vm)
	}
	if vm.Locale != "" {
		locale = vm.Locale
	}
	return http.StatusOK, endpoint.page(locale, vm)
}

func (endpoint NuevaAppWebEndpointV0) postDirectorPage(
	r *http.Request,
	form WebNuevaAppFormV0,
	locale string,
) (int, NuevaAppWebPageV0) {
	vm, err := endpoint.DirectorClient.ArrancarDirectorApp(r.Context(), form)
	if err != nil {
		code := WebNuevaAppErrTransporteV0
		var clientErr WebNuevaAppClientErrorV0
		if errors.As(err, &clientErr) && strings.TrimSpace(clientErr.Code) != "" {
			code = clientErr.Code
		}
		vm = nuevaAppWebPublicErrorViewModelV0(form.RequestID, locale, WebNuevaAppEstadoError, code)
		return http.StatusBadGateway, endpoint.page(locale, vm)
	}
	if vm.Locale != "" {
		locale = vm.Locale
	}
	return http.StatusOK, endpoint.page(locale, vm)
}

func isNuevaAppPreviewActionV0(action string) bool {
	switch strings.TrimSpace(action) {
	case "preview_goal", "preview":
		return true
	default:
		return false
	}
}
