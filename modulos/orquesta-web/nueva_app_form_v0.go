package orquestaweb

import (
	"strings"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

const (
	WebNuevaAppSourceV0        = "orquesta-web"
	WebNuevaAppDefaultSchemaV0 = orquestafactory.AppSpecRequestSchemaV0
)

type WebNuevaAppFormV0 struct {
	Action                string                         `json:"nueva_app_action,omitempty"`
	RequestID             string                         `json:"request_id"`
	Locale                string                         `json:"locale"`
	RequestKind           string                         `json:"request_kind,omitempty"`
	ExecutionMode         string                         `json:"execution_mode,omitempty"`
	DirectorExecutionMode string                         `json:"director_execution_mode,omitempty"`
	Nombre                string                         `json:"nombre"`
	Objetivo              string                         `json:"objetivo"`
	Descripcion           string                         `json:"descripcion,omitempty"`
	TipoApp               string                         `json:"tipo_app"`
	ProjectSource         WebNuevaAppProjectSourceFormV0 `json:"project_source,omitempty"`
	UsuariosObjetivo      []string                       `json:"usuarios_objetivo,omitempty"`
	Plataformas           []string                       `json:"plataformas,omitempty"`
	Integraciones         []WebNuevaAppConnectorFormV0   `json:"integraciones,omitempty"`
	PreferenciasTecnicas  WebNuevaAppPreferenciasFormV0  `json:"preferencias_tecnicas,omitempty"`
	Datos                 WebNuevaAppDatosFormV0         `json:"datos,omitempty"`
	Deploy                WebNuevaAppDeployFormV0        `json:"deploy,omitempty"`
	Calidad               WebNuevaAppCalidadFormV0       `json:"calidad,omitempty"`
	Documentacion         WebNuevaAppDocumentacionFormV0 `json:"documentacion,omitempty"`
	I18N                  WebNuevaAppI18NFormV0          `json:"i18n,omitempty"`
	Agentes               WebNuevaAppAgentesFormV0       `json:"agentes,omitempty"`
	Restricciones         []string                       `json:"restricciones,omitempty"`
}

type WebNuevaAppProjectSourceFormV0 struct {
	Kind       string `json:"kind,omitempty"`
	GitURL     string `json:"git_url,omitempty"`
	Branch     string `json:"branch,omitempty"`
	LocalPath  string `json:"local_path,omitempty"`
	ProjectRef string `json:"project_ref,omitempty"`
}

type WebNuevaAppConnectorFormV0 struct {
	Tipo          string   `json:"tipo,omitempty"`
	Nombre        string   `json:"nombre,omitempty"`
	Proposito     string   `json:"proposito,omitempty"`
	Direccion     string   `json:"direccion,omitempty"`
	Auth          string   `json:"auth,omitempty"`
	DataScope     string   `json:"data_scope,omitempty"`
	Criticidad    string   `json:"criticidad,omitempty"`
	Requerido     bool     `json:"requerido,omitempty"`
	Restricciones []string `json:"restricciones,omitempty"`
}

type WebNuevaAppPreferenciasFormV0 struct {
	Lenguaje      string   `json:"lenguaje,omitempty"`
	Framework     string   `json:"framework,omitempty"`
	Arquitectura  string   `json:"arquitectura,omitempty"`
	Restricciones []string `json:"restricciones,omitempty"`
	Preferencias  []string `json:"preferencias,omitempty"`
}

type WebNuevaAppDatosFormV0 struct {
	DBRequired         bool                           `json:"db_required,omitempty"`
	NecesidadFuncional string                         `json:"necesidad_funcional,omitempty"`
	TiposDatos         []string                       `json:"tipos_datos,omitempty"`
	TiposDetallados    []WebNuevaAppDataTypeFormV0    `json:"tipos_detallados,omitempty"`
	Fuentes            []WebNuevaAppDataSourceFormV0  `json:"fuentes,omitempty"`
	Storage            []WebNuevaAppDataStorageFormV0 `json:"storage,omitempty"`
	Operacion          WebNuevaAppDataOperationFormV0 `json:"operacion,omitempty"`
	Sensibilidad       string                         `json:"sensibilidad,omitempty"`
	Retencion          string                         `json:"retencion,omitempty"`
}

type WebNuevaAppDataTypeFormV0 struct {
	Nombre        string   `json:"nombre,omitempty"`
	Proposito     string   `json:"proposito,omitempty"`
	Sensibilidad  string   `json:"sensibilidad,omitempty"`
	Retencion     string   `json:"retencion,omitempty"`
	Volumen       string   `json:"volumen,omitempty"`
	Restricciones []string `json:"restricciones,omitempty"`
}

type WebNuevaAppDataStorageFormV0 struct {
	Tipo          string   `json:"tipo,omitempty"`
	Proposito     string   `json:"proposito,omitempty"`
	Requerido     bool     `json:"requerido,omitempty"`
	Restricciones []string `json:"restricciones,omitempty"`
}

type WebNuevaAppDataSourceFormV0 struct {
	Nombre        string   `json:"nombre,omitempty"`
	Tipo          string   `json:"tipo,omitempty"`
	Proposito     string   `json:"proposito,omitempty"`
	Owner         string   `json:"owner,omitempty"`
	Frecuencia    string   `json:"frecuencia,omitempty"`
	Restricciones []string `json:"restricciones,omitempty"`
}

type WebNuevaAppDataOperationFormV0 struct {
	Criticidad     string   `json:"criticidad,omitempty"`
	Disponibilidad string   `json:"disponibilidad,omitempty"`
	RPO            string   `json:"rpo,omitempty"`
	RTO            string   `json:"rto,omitempty"`
	Auditoria      bool     `json:"auditoria,omitempty"`
	Restricciones  []string `json:"restricciones,omitempty"`
}

type WebNuevaAppDeployFormV0 struct {
	Target        string   `json:"target,omitempty"`
	Restricciones []string `json:"restricciones,omitempty"`
}

type WebNuevaAppCalidadFormV0 struct {
	Pruebas               string   `json:"pruebas,omitempty"`
	Accesibilidad         string   `json:"accesibilidad,omitempty"`
	AccesibilidadOpciones []string `json:"accesibilidad_opciones,omitempty"`
	Compliance            []string `json:"compliance,omitempty"`
	Observabilidad        *bool    `json:"observabilidad,omitempty"`
}

type WebNuevaAppDocumentacionFormV0 struct {
	Usuario     *bool    `json:"usuario,omitempty"`
	Desarrollo  *bool    `json:"desarrollo,omitempty"`
	Sistemas    *bool    `json:"sistemas,omitempty"`
	Profundidad string   `json:"profundidad,omitempty"`
	Locales     []string `json:"locales,omitempty"`
}

type WebNuevaAppI18NFormV0 struct {
	Enabled       *bool    `json:"enabled,omitempty"`
	DefaultLocale string   `json:"default_locale,omitempty"`
	Locales       []string `json:"locales,omitempty"`
	Justificacion string   `json:"justificacion,omitempty"`
}

type WebNuevaAppAgentesFormV0 struct {
	RevisionHumana *bool    `json:"revision_humana,omitempty"`
	Autonomia      string   `json:"autonomia,omitempty"`
	Preferencias   []string `json:"preferencias,omitempty"`
}

func (form WebNuevaAppFormV0) ToAppSpecRequestV0() orquestafactory.AppSpecRequestV0 {
	return orquestafactory.AppSpecRequestV0{
		SchemaVersion:        WebNuevaAppDefaultSchemaV0,
		RequestID:            trimV0(form.RequestID),
		Source:               WebNuevaAppSourceV0,
		Locale:               trimV0(form.Locale),
		RequestKind:          orquestafactory.NormalizeRequestKindV0(form.RequestKind),
		ExecutionMode:        orquestafactory.NormalizeExecutionModeV0(form.ExecutionMode),
		Nombre:               trimV0(form.Nombre),
		Objetivo:             trimV0(form.Objetivo),
		Descripcion:          trimV0(form.Descripcion),
		TipoApp:              trimV0(form.TipoApp),
		ProjectSource:        mapProjectSourceFormV0(form.ProjectSource),
		UsuariosObjetivo:     compactStringsV0(form.UsuariosObjetivo),
		Plataformas:          compactStringsV0(form.Plataformas),
		Integraciones:        mapConnectorFormsV0(form.Integraciones),
		PreferenciasTecnicas: mapPreferenciasFormV0(form.PreferenciasTecnicas),
		Datos:                mapDatosFormV0(form.Datos),
		Deploy:               mapDeployFormV0(form.Deploy),
		Calidad:              mapCalidadFormV0(form.Calidad),
		Documentacion:        mapDocumentacionFormV0(form.Documentacion),
		I18N:                 mapI18NFormV0(form.Locale, form.I18N),
		Agentes:              mapAgentesFormV0(form.Agentes),
		Restricciones:        compactStringsV0(form.Restricciones),
	}
}

func mapConnectorFormsV0(values []WebNuevaAppConnectorFormV0) []orquestafactory.ConnectorRequestV0 {
	out := make([]orquestafactory.ConnectorRequestV0, 0, len(values))
	for _, value := range values {
		out = append(out, orquestafactory.ConnectorRequestV0{
			Tipo:          trimV0(value.Tipo),
			Nombre:        trimV0(value.Nombre),
			Proposito:     trimV0(value.Proposito),
			Direccion:     trimV0(value.Direccion),
			Auth:          trimV0(value.Auth),
			DataScope:     trimV0(value.DataScope),
			Criticidad:    trimV0(value.Criticidad),
			Requerido:     value.Requerido,
			Restricciones: compactStringsV0(value.Restricciones),
		})
	}
	if out == nil {
		return []orquestafactory.ConnectorRequestV0{}
	}
	return out
}

func mapPreferenciasFormV0(value WebNuevaAppPreferenciasFormV0) orquestafactory.PreferenciasTecnicasV0 {
	arquitectura := trimV0(value.Arquitectura)
	if arquitectura == "" {
		arquitectura = "hexagonal"
	}
	return orquestafactory.PreferenciasTecnicasV0{
		Lenguaje:      trimV0(value.Lenguaje),
		Framework:     trimV0(value.Framework),
		Arquitectura:  arquitectura,
		Restricciones: compactStringsV0(value.Restricciones),
		Preferencias:  compactStringsV0(value.Preferencias),
	}
}

func mapDatosFormV0(value WebNuevaAppDatosFormV0) orquestafactory.DatosRequestV0 {
	return orquestafactory.DatosRequestV0{
		DBRequired:         value.DBRequired,
		NecesidadFuncional: trimV0(value.NecesidadFuncional),
		TiposDatos:         compactStringsV0(value.TiposDatos),
		TiposDetallados:    mapDataTypeFormsV0(value.TiposDetallados),
		Fuentes:            mapDataSourceFormsV0(value.Fuentes),
		Storage:            mapDataStorageFormsV0(value.Storage),
		Operacion:          mapDataOperationFormV0(value.Operacion),
		Sensibilidad:       trimV0(value.Sensibilidad),
		Retencion:          trimV0(value.Retencion),
	}
}

func mapDataTypeFormsV0(values []WebNuevaAppDataTypeFormV0) []orquestafactory.DataTypeRequestV0 {
	out := make([]orquestafactory.DataTypeRequestV0, 0, len(values))
	for _, value := range values {
		out = append(out, orquestafactory.DataTypeRequestV0{
			Nombre:        trimV0(value.Nombre),
			Proposito:     trimV0(value.Proposito),
			Sensibilidad:  trimV0(value.Sensibilidad),
			Retencion:     trimV0(value.Retencion),
			Volumen:       trimV0(value.Volumen),
			Restricciones: compactStringsV0(value.Restricciones),
		})
	}
	if out == nil {
		return []orquestafactory.DataTypeRequestV0{}
	}
	return out
}

func mapDataStorageFormsV0(values []WebNuevaAppDataStorageFormV0) []orquestafactory.DataStorageRequestV0 {
	out := make([]orquestafactory.DataStorageRequestV0, 0, len(values))
	for _, value := range values {
		out = append(out, orquestafactory.DataStorageRequestV0{
			Tipo:          trimV0(value.Tipo),
			Proposito:     trimV0(value.Proposito),
			Requerido:     value.Requerido,
			Restricciones: compactStringsV0(value.Restricciones),
		})
	}
	if out == nil {
		return []orquestafactory.DataStorageRequestV0{}
	}
	return out
}

func mapDataSourceFormsV0(values []WebNuevaAppDataSourceFormV0) []orquestafactory.DataSourceRequestV0 {
	out := make([]orquestafactory.DataSourceRequestV0, 0, len(values))
	for _, value := range values {
		out = append(out, orquestafactory.DataSourceRequestV0{
			Nombre:        trimV0(value.Nombre),
			Tipo:          trimV0(value.Tipo),
			Proposito:     trimV0(value.Proposito),
			Owner:         trimV0(value.Owner),
			Frecuencia:    trimV0(value.Frecuencia),
			Restricciones: compactStringsV0(value.Restricciones),
		})
	}
	if out == nil {
		return []orquestafactory.DataSourceRequestV0{}
	}
	return out
}

func mapDataOperationFormV0(value WebNuevaAppDataOperationFormV0) orquestafactory.DataOperationRequestV0 {
	return orquestafactory.DataOperationRequestV0{
		Criticidad:     trimV0(value.Criticidad),
		Disponibilidad: trimV0(value.Disponibilidad),
		RPO:            trimV0(value.RPO),
		RTO:            trimV0(value.RTO),
		Auditoria:      value.Auditoria,
		Restricciones:  compactStringsV0(value.Restricciones),
	}
}

func mapDeployFormV0(value WebNuevaAppDeployFormV0) orquestafactory.DeployRequestV0 {
	return orquestafactory.DeployRequestV0{
		Target:        trimV0(value.Target),
		Restricciones: compactStringsV0(value.Restricciones),
	}
}

func mapCalidadFormV0(value WebNuevaAppCalidadFormV0) orquestafactory.CalidadRequestV0 {
	return orquestafactory.CalidadRequestV0{
		Pruebas:               trimV0(value.Pruebas),
		Accesibilidad:         trimV0(value.Accesibilidad),
		AccesibilidadOpciones: compactStringsV0(value.AccesibilidadOpciones),
		Compliance:            compactStringsV0(value.Compliance),
		Observabilidad:        value.Observabilidad,
	}
}

func mapDocumentacionFormV0(value WebNuevaAppDocumentacionFormV0) orquestafactory.DocumentacionV0 {
	return orquestafactory.DocumentacionV0{
		Usuario:     value.Usuario,
		Desarrollo:  value.Desarrollo,
		Sistemas:    value.Sistemas,
		Profundidad: trimV0(value.Profundidad),
		Locales:     compactStringsV0(value.Locales),
	}
}

func mapI18NFormV0(locale string, value WebNuevaAppI18NFormV0) orquestafactory.I18NRequestV0 {
	defaultLocale := trimV0(value.DefaultLocale)
	if defaultLocale == "" {
		defaultLocale = trimV0(locale)
	}
	return orquestafactory.I18NRequestV0{
		Enabled:       value.Enabled,
		DefaultLocale: defaultLocale,
		Locales:       compactStringsV0(value.Locales),
		Justificacion: trimV0(value.Justificacion),
	}
}

func mapAgentesFormV0(value WebNuevaAppAgentesFormV0) orquestafactory.AgentesRequestV0 {
	return orquestafactory.AgentesRequestV0{
		RevisionHumana: value.RevisionHumana,
		Autonomia:      trimV0(value.Autonomia),
		Preferencias:   compactStringsV0(value.Preferencias),
	}
}

func mapProjectSourceFormV0(value WebNuevaAppProjectSourceFormV0) orquestafactory.ProjectSourceRequestV0 {
	return orquestafactory.ProjectSourceRequestV0{
		Kind:       trimV0(value.Kind),
		GitURL:     trimV0(value.GitURL),
		Branch:     trimV0(value.Branch),
		LocalPath:  trimV0(value.LocalPath),
		ProjectRef: trimV0(value.ProjectRef),
	}
}

func trimV0(value string) string {
	return strings.TrimSpace(value)
}

func compactStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := trimV0(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	if out == nil {
		return []string{}
	}
	return out
}
