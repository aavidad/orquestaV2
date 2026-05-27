package orquestaweb

import (
	"net/http"
)

type AppChangeWebEndpointV0 struct {
	Client AppChangeClientV0
}

func NewAppChangeWebEndpointV0(client AppChangeClientV0) AppChangeWebEndpointV0 {
	return AppChangeWebEndpointV0{Client: client}
}

func (endpoint AppChangeWebEndpointV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeAppChangeHTMLV0(w, http.StatusOK, appChangePageV0("es", InitialWebAppChangeViewModelV0()))
	case http.MethodPost:
		endpoint.handlePostV0(w, r)
	case http.MethodOptions:
		handleWebPublicHTTPOptionsV0(w, r, http.MethodGet, http.MethodPost)
	default:
		setWebPublicHTTPAllowV0(w, http.MethodGet, http.MethodPost)
		writeAppChangeHTMLV0(w, http.StatusMethodNotAllowed, appChangePageV0("es", WebAppChangeViewModelV0{Estado: WebAppChangeEstadoErrorV0}))
	}
}

func (endpoint AppChangeWebEndpointV0) handlePostV0(w http.ResponseWriter, r *http.Request) {
	form, err := decodeAppChangeFormV0(r)
	if err != nil || endpoint.Client == nil {
		writeAppChangeHTMLV0(w, http.StatusBadRequest, appChangePageV0("es", WebAppChangeViewModelV0{Estado: WebAppChangeEstadoErrorV0}))
		return
	}
	vm, err := endpoint.Client.RequestAppChange(r.Context(), form)
	if err != nil {
		writeAppChangeHTMLV0(w, http.StatusBadGateway, appChangePageV0(form.Locale, WebAppChangeViewModelV0{Estado: WebAppChangeEstadoErrorV0}))
		return
	}
	status := http.StatusOK
	if vm.Estado == WebAppChangeEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeAppChangeHTMLV0(w, status, appChangePageV0(form.Locale, vm))
}

func decodeAppChangeFormV0(r *http.Request) (WebAppChangeFormV0, error) {
	contentType := r.Header.Get("Content-Type")
	if webControlContentTypeAllowsJSONV0(contentType) {
		var form WebAppChangeFormV0
		return form, decodeWebControlJSONV0(nil, r, &form)
	}
	if webControlContentTypeAllowsFormV0(contentType) {
		if err := parseWebControlFormV0(nil, r); err != nil {
			return WebAppChangeFormV0{}, err
		}
		return appChangeFormFromValuesV0(r.Form), nil
	}
	return WebAppChangeFormV0{}, webNuevaAppClientErrorV0(WebAppChangeErrResponseV0, 0)
}

func appChangeFormFromValuesV0(values map[string][]string) WebAppChangeFormV0 {
	return WebAppChangeFormV0{
		RequestID:          formValueV0(values, "request_id"),
		CorrelationID:      formValueV0(values, "correlation_id"),
		Locale:             formValueV0(values, "locale"),
		RunRef:             formValueV0(values, "run_ref"),
		AppRef:             formValueV0(values, "app_ref"),
		ChangeRef:          formValueV0(values, "change_ref"),
		UserIntent:         formValueV0(values, "user_intent"),
		TargetArea:         formValueV0(values, "target_area"),
		CurrentStateRefs:   formValuesV0(values, "current_state_refs"),
		AcceptanceCriteria: formValuesV0(values, "acceptance_criteria"),
		Constraints:        formValuesV0(values, "constraints"),
		AllowedWriteSet:    formValuesV0(values, "allowed_write_set"),
		ExternalProjectRef: formValueV0(values, "external_project_ref"),
		ExternalInterfaces: formValuesV0(values, "external_interface_refs"),
		ExternalWorkKind:   formValueV0(values, "external_work_kind"),
		ExternalWorkRefs:   formValuesV0(values, "external_work_refs"),
	}
}
