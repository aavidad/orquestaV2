package orquestaweb

import (
	"encoding/json"
	"net/http"
)

type RunControlWebEndpointV0 struct {
	Client RunControlClientV0
}

type WebRunControlPageV0 struct {
	SchemaVersion string                   `json:"schema_version"`
	Locale        string                   `json:"locale"`
	Command       WebRunControlCommandV0   `json:"command"`
	ViewModel     WebRunControlViewModelV0 `json:"view_model"`
}

func NewRunControlWebEndpointV0(client RunControlClientV0) RunControlWebEndpointV0 {
	return RunControlWebEndpointV0{Client: client}
}

func (endpoint RunControlWebEndpointV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handleWebPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method == http.MethodGet && webRequestWantsHTMLV0(r) {
		writeWebHTMLStringResponseV0(w, http.StatusOK, runControlHTMLV0(), "es")
		return
	}
	if r.Method != http.MethodPost {
		setWebPublicHTTPAllowV0(w, http.MethodPost)
		writeRunControlPageV0(w, http.StatusMethodNotAllowed, endpoint.pageV0(WebRunControlCommandV0{},
			NewWebRunControlErrorViewModelV0("", WebNuevaAppErrMetodoNoSoportadoV0)))
		return
	}
	command, err := decodeRunControlCommandV0(r)
	if err != nil {
		writeRunControlPageV0(w, http.StatusBadRequest, endpoint.pageV0(command,
			NewWebRunControlErrorViewModelV0(command.Locale, WebRunControlErrRespuestaInvalidaV0)))
		return
	}
	endpoint.handleRunControlV0(w, r, command)
}

func (endpoint RunControlWebEndpointV0) handleRunControlV0(
	w http.ResponseWriter,
	r *http.Request,
	command WebRunControlCommandV0,
) {
	command = normalizeRunControlCommandV0(command)
	if endpoint.Client == nil {
		writeRunControlPageV0(w, http.StatusServiceUnavailable, endpoint.pageV0(command,
			NewWebRunControlErrorViewModelV0(command.Locale, WebNuevaAppErrTransporteNoConfiguradoV0)))
		return
	}
	vm, err := endpoint.Client.EnviarRunControl(r.Context(), command)
	if err != nil {
		writeRunControlPageV0(w, http.StatusBadGateway, endpoint.pageV0(command,
			NewWebRunControlErrorViewModelV0(command.Locale, WebRunControlErrTransporteV0)))
		return
	}
	status := http.StatusOK
	if vm.Estado == WebRunControlEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeRunControlPageV0(w, status, endpoint.pageV0(command, vm))
}

func (endpoint RunControlWebEndpointV0) pageV0(
	command WebRunControlCommandV0,
	vm WebRunControlViewModelV0,
) WebRunControlPageV0 {
	return WebRunControlPageV0{
		SchemaVersion: "web_run_control_page.v0",
		Locale:        normalizeDirectorStatsLocaleV0(command.Locale),
		Command:       command,
		ViewModel:     vm,
	}
}

func decodeRunControlCommandV0(r *http.Request) (WebRunControlCommandV0, error) {
	contentType := r.Header.Get("Content-Type")
	if webControlContentTypeAllowsJSONV0(contentType) {
		var command WebRunControlCommandV0
		return command, decodeWebControlJSONV0(nil, r, &command)
	}
	if webControlContentTypeAllowsFormV0(contentType) {
		if err := parseWebControlFormV0(nil, r); err != nil {
			return WebRunControlCommandV0{}, err
		}
		return runControlCommandFromValuesV0(r.Form), nil
	}
	return WebRunControlCommandV0{}, errUnsupportedDirectorStatsContentV0{}
}

func runControlCommandFromValuesV0(values map[string][]string) WebRunControlCommandV0 {
	return WebRunControlCommandV0{
		RequestID:      formValueV0(values, "request_id"),
		CorrelationID:  formValueV0(values, "correlation_id"),
		Locale:         formValueV0(values, "locale"),
		Action:         formValueV0(values, "action"),
		RunRef:         formValueV0(values, "run_ref"),
		RequestedBy:    formValueV0(values, "requested_by"),
		Reason:         formValueV0(values, "reason"),
		Forced:         formBoolValueV0(values, "forced"),
		IdempotencyKey: formValueV0(values, "idempotency_key"),
		EvidenceRefs:   formValuesV0(values, "evidence_refs"),
	}
}

func writeRunControlPageV0(w http.ResponseWriter, status int, page WebRunControlPageV0) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(page)
}
