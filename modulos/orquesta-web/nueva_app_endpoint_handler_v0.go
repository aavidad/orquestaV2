package orquestaweb

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

func (endpoint NuevaAppWebEndpointV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	catalog := endpoint.catalog()
	if err := validateWebPublicQueryV0(r); err != nil {
		locale := NuevaAppI18nDefaultLocaleV0
		vm := nuevaAppWebPublicErrorViewModelV0("", locale, WebNuevaAppEstadoError, WebNuevaAppErrRespuestaInvalidaV0)
		writeNuevaAppWebPageV0(w, http.StatusBadRequest, endpoint.page(locale, vm))
		return
	}
	switch r.Method {
	case http.MethodGet:
		locale := localeFromNuevaAppRequestV0(r, catalog)
		writeNuevaAppWebPageV0(w, http.StatusOK, endpoint.page(locale, initialNuevaAppViewModelV0(locale)))
	case http.MethodPost:
		endpoint.handlePost(w, r, catalog)
	case http.MethodOptions:
		handleWebPublicHTTPOptionsV0(w, r, http.MethodGet, http.MethodPost)
	default:
		setWebPublicHTTPAllowV0(w, http.MethodGet, http.MethodPost)
		locale := localeFromNuevaAppRequestV0(r, catalog)
		vm := nuevaAppWebPublicErrorViewModelV0("", locale, WebNuevaAppEstadoError, WebNuevaAppErrMetodoNoSoportadoV0)
		writeNuevaAppWebPageV0(w, http.StatusMethodNotAllowed, endpoint.page(locale, vm))
	}
}

func (endpoint NuevaAppWebEndpointV0) handlePost(w http.ResponseWriter, r *http.Request, catalog NuevaAppI18nCatalogV0) {
	status, page := endpoint.postPage(r, catalog)
	writeNuevaAppWebPageV0(w, status, page)
}

func decodeNuevaAppWebFormV0(r *http.Request) (WebNuevaAppFormV0, error) {
	contentType := r.Header.Get("Content-Type")
	if webControlContentTypeAllowsJSONV0(contentType) {
		var form WebNuevaAppFormV0
		if err := decodeWebControlJSONV0(nil, r, &form); err != nil {
			return form, err
		}
		return form, nil
	}
	if webControlContentTypeAllowsFormV0(contentType) {
		if err := parseWebControlFormV0(nil, r); err != nil {
			return WebNuevaAppFormV0{}, err
		}
		return nuevaAppFormFromValuesV0(r.Form), nil
	}
	return WebNuevaAppFormV0{}, errors.New("content_type_no_soportado")
}

func nuevaAppFormFromValuesV0(values map[string][]string) WebNuevaAppFormV0 {
	form := WebNuevaAppFormV0{
		RequestID:             formValueV0(values, "request_id"),
		Locale:                formValueV0(values, "locale"),
		RequestKind:           formValueV0(values, "request_kind"),
		ExecutionMode:         formValueV0(values, "execution_mode"),
		DirectorExecutionMode: formValueV0(values, "director_execution_mode"),
		Nombre:                formValueV0(values, "nombre"),
		Objetivo:              formValueV0(values, "objetivo"),
		Descripcion:           formValueV0(values, "descripcion"),
		TipoApp:               formValueV0(values, "tipo_app"),
		ProjectSource: WebNuevaAppProjectSourceFormV0{
			Kind:       formValueV0(values, "project_source.kind"),
			GitURL:     formValueV0(values, "project_source.git_url"),
			Branch:     formValueV0(values, "project_source.branch"),
			LocalPath:  formValueV0(values, "project_source.local_path"),
			ProjectRef: formValueV0(values, "project_source.project_ref"),
		},
		UsuariosObjetivo: formValuesV0(values, "usuarios_objetivo"),
		Plataformas:      formValuesV0(values, "plataformas"),
		PreferenciasTecnicas: WebNuevaAppPreferenciasFormV0{
			Lenguaje:      formValueV0(values, "preferencias_tecnicas.lenguaje"),
			Framework:     formValueV0(values, "preferencias_tecnicas.framework"),
			Arquitectura:  formValueV0(values, "preferencias_tecnicas.arquitectura"),
			Restricciones: formValuesV0(values, "preferencias_tecnicas.restricciones"),
			Preferencias:  formValuesV0(values, "preferencias_tecnicas.preferencias"),
		},
		Datos: WebNuevaAppDatosFormV0{
			DBRequired:         formBoolValueV0(values, "datos.db_required"),
			NecesidadFuncional: formValueV0(values, "datos.necesidad_funcional"),
			TiposDatos:         formValuesV0(values, "datos.tipos_datos"),
			TiposDetallados:    formDataTypesV0(values, 4),
			Storage:            formDataStorageV0(values, 4),
			Sensibilidad:       formValueV0(values, "datos.sensibilidad"),
			Retencion:          formValueV0(values, "datos.retencion"),
		},
		Deploy: WebNuevaAppDeployFormV0{
			Target:        formValueV0(values, "deploy.target"),
			Restricciones: formValuesV0(values, "deploy.restricciones"),
		},
		Calidad: WebNuevaAppCalidadFormV0{
			Pruebas:               formValueV0(values, "calidad.pruebas"),
			Accesibilidad:         formValueV0(values, "calidad.accesibilidad"),
			AccesibilidadOpciones: formValuesV0(values, "calidad.accesibilidad_opciones"),
			Compliance:            formValuesV0(values, "calidad.compliance"),
			Observabilidad:        formOptionalBoolValueV0(values, "calidad.observabilidad"),
		},
		Documentacion: WebNuevaAppDocumentacionFormV0{
			Usuario:    formOptionalBoolValueV0(values, "documentacion.usuario"),
			Desarrollo: formOptionalBoolValueV0(values, "documentacion.desarrollo"),
			Sistemas:   formOptionalBoolValueV0(values, "documentacion.sistemas"),
			Locales:    formValuesV0(values, "documentacion.locales"),
		},
		I18N: WebNuevaAppI18NFormV0{
			Enabled:       formOptionalBoolValueV0(values, "i18n.enabled"),
			DefaultLocale: formValueV0(values, "i18n.default_locale"),
			Locales:       formValuesV0(values, "i18n.locales"),
			Justificacion: formValueV0(values, "i18n.justificacion"),
		},
		Agentes: WebNuevaAppAgentesFormV0{
			RevisionHumana: formOptionalBoolValueV0(values, "agentes.revision_humana"),
			Autonomia:      formValueV0(values, "agentes.autonomia"),
			Preferencias:   formValuesV0(values, "agentes.preferencias"),
		},
		Restricciones: formValuesV0(values, "restricciones"),
	}
	form.Integraciones = formConnectorsV0(values, 4)
	return form
}

func formDataTypesV0(values map[string][]string, maxRows int) []WebNuevaAppDataTypeFormV0 {
	out := make([]WebNuevaAppDataTypeFormV0, 0, maxRows)
	for index := 0; index < maxRows; index++ {
		prefix := "datos.tipos_detallados." + strconv.Itoa(index) + "."
		row := WebNuevaAppDataTypeFormV0{
			Nombre:        formValueV0(values, prefix+"nombre"),
			Proposito:     formValueV0(values, prefix+"proposito"),
			Sensibilidad:  formValueV0(values, prefix+"sensibilidad"),
			Retencion:     formValueV0(values, prefix+"retencion"),
			Volumen:       formValueV0(values, prefix+"volumen"),
			Restricciones: formValuesV0(values, prefix+"restricciones"),
		}
		if row.Nombre != "" || row.Proposito != "" || row.Sensibilidad != "" || row.Retencion != "" || row.Volumen != "" || len(row.Restricciones) > 0 {
			out = append(out, row)
		}
	}
	if out == nil {
		return []WebNuevaAppDataTypeFormV0{}
	}
	return out
}

func formDataStorageV0(values map[string][]string, maxRows int) []WebNuevaAppDataStorageFormV0 {
	out := make([]WebNuevaAppDataStorageFormV0, 0, maxRows)
	for index := 0; index < maxRows; index++ {
		prefix := "datos.storage." + strconv.Itoa(index) + "."
		row := WebNuevaAppDataStorageFormV0{
			Tipo:          formValueV0(values, prefix+"tipo"),
			Proposito:     formValueV0(values, prefix+"proposito"),
			Requerido:     formBoolValueV0(values, prefix+"requerido"),
			Restricciones: formValuesV0(values, prefix+"restricciones"),
		}
		if row.Tipo != "" || row.Proposito != "" || len(row.Restricciones) > 0 || row.Requerido {
			out = append(out, row)
		}
	}
	if out == nil {
		return []WebNuevaAppDataStorageFormV0{}
	}
	return out
}

func formConnectorsV0(values map[string][]string, maxRows int) []WebNuevaAppConnectorFormV0 {
	out := make([]WebNuevaAppConnectorFormV0, 0, maxRows)
	for index := 0; index < maxRows; index++ {
		prefix := "integraciones." + strconv.Itoa(index) + "."
		row := WebNuevaAppConnectorFormV0{
			Tipo:          formValueV0(values, prefix+"tipo"),
			Nombre:        formValueV0(values, prefix+"nombre"),
			Proposito:     formValueV0(values, prefix+"proposito"),
			Requerido:     formBoolValueV0(values, prefix+"requerido"),
			Restricciones: formValuesV0(values, prefix+"restricciones"),
		}
		if row.Tipo != "" || row.Nombre != "" || row.Proposito != "" || len(row.Restricciones) > 0 || row.Requerido {
			out = append(out, row)
		}
	}
	if out == nil {
		return []WebNuevaAppConnectorFormV0{}
	}
	return out
}

func formValueV0(values map[string][]string, key string) string {
	for _, value := range values[key] {
		if trimmed := trimV0(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func formValuesV0(values map[string][]string, key string) []string {
	var out []string
	for _, value := range values[key] {
		for _, part := range strings.Split(value, ",") {
			if trimmed := trimV0(part); trimmed != "" {
				out = append(out, trimmed)
			}
		}
	}
	return compactStringsV0(out)
}

func formBoolValueV0(values map[string][]string, key string) bool {
	value := formValueV0(values, key)
	if value == "" {
		return false
	}
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
}

func formOptionalBoolValueV0(values map[string][]string, key string) *bool {
	if _, ok := values[key]; !ok {
		return nil
	}
	value := formBoolValueV0(values, key)
	return &value
}
