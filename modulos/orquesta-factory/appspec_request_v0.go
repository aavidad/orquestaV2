package orquestafactory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/text/language"
)

const (
	AppSpecRequestSchemaV0 = "app_spec_request.v0"

	ErrAppSpecInvalida               = "app_spec_invalida"
	ErrOpcionIncompatible            = "opcion_incompatible"
	ErrTargetNoSoportado             = "target_no_soportado"
	ErrIdiomaInvalido                = "idioma_invalido"
	ErrConectorRequeridoNoDisponible = "conector_requerido_no_disponible"
)

var supportedDataStorageTypesV0 = []string{
	"sin_preferencia",
	"sin_persistencia",
	"relacional",
	"documental",
	"vectorial",
	"objetos",
	"objetos_blob",
	"clave_valor",
	"clave_valor_cache",
	"series_temporales",
	"grafo",
	"cache",
	"busqueda",
	"eventos_auditoria",
	"mixta",
}

func SupportedDataStorageTypesV0() []string {
	return append([]string(nil), supportedDataStorageTypesV0...)
}

func DataStorageTypeSupportedV0(value string) bool {
	return containsV0(strings.TrimSpace(value), supportedDataStorageTypesV0...)
}

type AppSpecRequestV0 struct {
	SchemaVersion        string                 `json:"schema_version"`
	RequestID            string                 `json:"request_id"`
	Source               string                 `json:"source"`
	Locale               string                 `json:"locale"`
	RequestKind          string                 `json:"request_kind,omitempty"`
	ExecutionMode        string                 `json:"execution_mode,omitempty"`
	ProjectSource        ProjectSourceRequestV0 `json:"project_source,omitempty"`
	Nombre               string                 `json:"nombre"`
	Objetivo             string                 `json:"objetivo"`
	Descripcion          string                 `json:"descripcion,omitempty"`
	TipoApp              string                 `json:"tipo_app"`
	UsuariosObjetivo     []string               `json:"usuarios_objetivo,omitempty"`
	Plataformas          []string               `json:"plataformas,omitempty"`
	Integraciones        []ConnectorRequestV0   `json:"integraciones,omitempty"`
	PreferenciasTecnicas PreferenciasTecnicasV0 `json:"preferencias_tecnicas,omitempty"`
	Datos                DatosRequestV0         `json:"datos,omitempty"`
	Deploy               DeployRequestV0        `json:"deploy,omitempty"`
	Calidad              CalidadRequestV0       `json:"calidad,omitempty"`
	Documentacion        DocumentacionV0        `json:"documentacion,omitempty"`
	I18N                 I18NRequestV0          `json:"i18n,omitempty"`
	Agentes              AgentesRequestV0       `json:"agentes,omitempty"`
	Restricciones        []string               `json:"restricciones,omitempty"`
}

type ConnectorRequestV0 struct {
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

type ProjectSourceRequestV0 struct {
	Kind       string `json:"kind,omitempty"`
	GitURL     string `json:"git_url,omitempty"`
	Branch     string `json:"branch,omitempty"`
	LocalPath  string `json:"local_path,omitempty"`
	ProjectRef string `json:"project_ref,omitempty"`
}

type PreferenciasTecnicasV0 struct {
	Lenguaje      string   `json:"lenguaje,omitempty"`
	Framework     string   `json:"framework,omitempty"`
	Arquitectura  string   `json:"arquitectura,omitempty"`
	Restricciones []string `json:"restricciones,omitempty"`
	Preferencias  []string `json:"preferencias,omitempty"`
}

type DatosRequestV0 struct {
	DBRequired         bool                   `json:"db_required,omitempty"`
	NecesidadFuncional string                 `json:"necesidad_funcional,omitempty"`
	TiposDatos         []string               `json:"tipos_datos,omitempty"`
	TiposDetallados    []DataTypeRequestV0    `json:"tipos_detallados,omitempty"`
	Fuentes            []DataSourceRequestV0  `json:"fuentes,omitempty"`
	Storage            []DataStorageRequestV0 `json:"storage,omitempty"`
	Operacion          DataOperationRequestV0 `json:"operacion,omitempty"`
	Sensibilidad       string                 `json:"sensibilidad,omitempty"`
	Retencion          string                 `json:"retencion,omitempty"`
}

type DataTypeRequestV0 struct {
	Nombre        string   `json:"nombre,omitempty"`
	Proposito     string   `json:"proposito,omitempty"`
	Sensibilidad  string   `json:"sensibilidad,omitempty"`
	Retencion     string   `json:"retencion,omitempty"`
	Volumen       string   `json:"volumen,omitempty"`
	Restricciones []string `json:"restricciones,omitempty"`
}

type DataStorageRequestV0 struct {
	Tipo          string   `json:"tipo,omitempty"`
	Proposito     string   `json:"proposito,omitempty"`
	Requerido     bool     `json:"requerido,omitempty"`
	Restricciones []string `json:"restricciones,omitempty"`
}

type DataSourceRequestV0 struct {
	Nombre        string   `json:"nombre,omitempty"`
	Tipo          string   `json:"tipo,omitempty"`
	Proposito     string   `json:"proposito,omitempty"`
	Owner         string   `json:"owner,omitempty"`
	Frecuencia    string   `json:"frecuencia,omitempty"`
	Restricciones []string `json:"restricciones,omitempty"`
}

type DataOperationRequestV0 struct {
	Criticidad     string   `json:"criticidad,omitempty"`
	Disponibilidad string   `json:"disponibilidad,omitempty"`
	RPO            string   `json:"rpo,omitempty"`
	RTO            string   `json:"rto,omitempty"`
	Auditoria      bool     `json:"auditoria,omitempty"`
	Restricciones  []string `json:"restricciones,omitempty"`
}

type DeployRequestV0 struct {
	Target        string   `json:"target,omitempty"`
	Restricciones []string `json:"restricciones,omitempty"`
}

type CalidadRequestV0 struct {
	Pruebas               string   `json:"pruebas,omitempty"`
	Accesibilidad         string   `json:"accesibilidad,omitempty"`
	AccesibilidadOpciones []string `json:"accesibilidad_opciones,omitempty"`
	Compliance            []string `json:"compliance,omitempty"`
	Observabilidad        *bool    `json:"observabilidad,omitempty"`
}

type DocumentacionV0 struct {
	Usuario     *bool    `json:"usuario,omitempty"`
	Desarrollo  *bool    `json:"desarrollo,omitempty"`
	Sistemas    *bool    `json:"sistemas,omitempty"`
	Profundidad string   `json:"profundidad,omitempty"`
	Locales     []string `json:"locales,omitempty"`
}

type I18NRequestV0 struct {
	Enabled       *bool    `json:"enabled,omitempty"`
	DefaultLocale string   `json:"default_locale,omitempty"`
	Locales       []string `json:"locales,omitempty"`
	Justificacion string   `json:"justificacion,omitempty"`
}

type AgentesRequestV0 struct {
	RevisionHumana *bool    `json:"revision_humana,omitempty"`
	Autonomia      string   `json:"autonomia,omitempty"`
	Preferencias   []string `json:"preferencias,omitempty"`
}

type ValidationIssue struct {
	Code    string `json:"code"`
	Field   string `json:"path,omitempty"`
	Message string `json:"message"`
}

func DecodeAppSpecRequestV0(data []byte) (AppSpecRequestV0, []ValidationIssue) {
	var req AppSpecRequestV0
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return req, []ValidationIssue{{
			Code:    ErrAppSpecInvalida,
			Field:   "",
			Message: fmt.Sprintf("request JSON invalido: %v", err),
		}}
	}
	return req, ValidateAppSpecRequestV0(req)
}

func ValidateAppSpecRequestV0(req AppSpecRequestV0) []ValidationIssue {
	var issues []ValidationIssue
	issues = append(issues, validateRequiredV0(req)...)
	issues = append(issues, validateEnumsV0(req)...)
	issues = append(issues, validateProjectSourceV0(req.ProjectSource)...)
	issues = append(issues, validateRequestKindProjectSourceV0(req)...)
	issues = append(issues, validateLocalesV0(req)...)
	issues = append(issues, validateI18NV0(req)...)
	issues = append(issues, validateDatosV0(req)...)
	return issues
}

func validateRequiredV0(req AppSpecRequestV0) []ValidationIssue {
	var issues []ValidationIssue
	required := map[string]string{
		"schema_version": req.SchemaVersion,
		"request_id":     req.RequestID,
		"source":         req.Source,
		"locale":         req.Locale,
		"nombre":         req.Nombre,
		"objetivo":       req.Objetivo,
		"tipo_app":       req.TipoApp,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			issues = append(issues, issue(ErrAppSpecInvalida, field, "campo obligatorio"))
		}
	}
	if req.SchemaVersion != "" && req.SchemaVersion != AppSpecRequestSchemaV0 {
		issues = append(issues, issue(ErrAppSpecInvalida, "schema_version", "schema_version no soportado"))
	}
	return issues
}

func validateEnumsV0(req AppSpecRequestV0) []ValidationIssue {
	var issues []ValidationIssue
	if req.Source != "" && !containsV0(req.Source, "orquesta-web", "orquesta-mcp", "orquesta-cli") {
		issues = append(issues, issue(ErrAppSpecInvalida, "source", "source no soportado"))
	}
	if req.TipoApp != "" && !containsV0(req.TipoApp, "web", "api", "cli", "desktop", "mobile", "automation", "data", "plugin", "mixed", "documentacion", "documentation") {
		issues = append(issues, issue(ErrAppSpecInvalida, "tipo_app", "tipo_app no soportado"))
	}
	if req.RequestKind != "" && !RequestKindSupportedV0(req.RequestKind) {
		issues = append(issues, issue(ErrAppSpecInvalida, "request_kind", "tipo de peticion no soportado"))
	}
	if req.ExecutionMode != "" && !ExecutionModeSupportedV0(req.ExecutionMode) {
		issues = append(issues, issue(ErrAppSpecInvalida, "execution_mode", "modo de ejecucion no soportado"))
	}
	if target := req.Deploy.Target; target != "" && !containsV0(target, "sin_preferencia", "local", "contenedor", "paas", "serverless", "kubernetes", "desktop", "mobile_store") {
		issues = append(issues, issue(ErrTargetNoSoportado, "deploy.target", "target de deploy no soportado"))
	}
	if pruebas := req.Calidad.Pruebas; pruebas != "" && !containsV0(pruebas, "basica", "media", "alta") {
		issues = append(issues, issue(ErrAppSpecInvalida, "calidad.pruebas", "nivel de pruebas no soportado"))
	}
	if accesibilidad := req.Calidad.Accesibilidad; accesibilidad != "" && !AccessibilityLevelSupportedV0(accesibilidad) {
		issues = append(issues, issue(ErrAppSpecInvalida, "calidad.accesibilidad", "nivel de accesibilidad no soportado"))
	}
	for index, accesibilidad := range req.Calidad.AccesibilidadOpciones {
		if strings.TrimSpace(accesibilidad) != "" && !AccessibilityOptionSupportedV0(accesibilidad) {
			issues = append(issues, issue(ErrAppSpecInvalida, fmt.Sprintf("calidad.accesibilidad_opciones.%d", index), "opcion de accesibilidad no soportada"))
		}
	}
	if autonomia := req.Agentes.Autonomia; autonomia != "" && !containsV0(autonomia, "baja", "media", "alta") {
		issues = append(issues, issue(ErrAppSpecInvalida, "agentes.autonomia", "autonomia no soportada"))
	}
	if arch := req.PreferenciasTecnicas.Arquitectura; strings.TrimSpace(arch) != "" && !ArchitecturePatternSupportedV0(arch) {
		issues = append(issues, issue(ErrOpcionIncompatible, "preferencias_tecnicas.arquitectura", "arquitectura no soportada"))
	}
	issues = append(issues, validateConnectorNamesV0(req)...)
	return issues
}

func validateLocalesV0(req AppSpecRequestV0) []ValidationIssue {
	var issues []ValidationIssue
	for _, candidate := range append([]string{req.Locale, req.I18N.DefaultLocale}, req.I18N.Locales...) {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if _, err := language.Parse(candidate); err != nil {
			issues = append(issues, issue(ErrIdiomaInvalido, "locale", "locale BCP 47 invalido"))
		}
	}
	return issues
}

func validateI18NV0(req AppSpecRequestV0) []ValidationIssue {
	if req.I18N.Enabled == nil || *req.I18N.Enabled {
		return nil
	}
	if strings.TrimSpace(req.I18N.Justificacion) == "" {
		return []ValidationIssue{issue(ErrOpcionIncompatible, "i18n.justificacion", "i18n.enabled=false requiere justificacion")}
	}
	return nil
}

func validateDatosV0(req AppSpecRequestV0) []ValidationIssue {
	var issues []ValidationIssue
	for index, storage := range req.Datos.Storage {
		tipo := strings.TrimSpace(storage.Tipo)
		if tipo != "" && !DataStorageTypeSupportedV0(tipo) {
			issues = append(issues, issue(ErrAppSpecInvalida, fmt.Sprintf("datos.storage.%d.tipo", index), "tipo de almacenamiento no soportado"))
		}
	}
	if !req.Datos.DBRequired {
		return issues
	}
	if strings.TrimSpace(req.Datos.NecesidadFuncional) == "" && !hasDetailedDataNeedV0(req.Datos) {
		issues = append(issues, issue(ErrAppSpecInvalida, "datos.necesidad_funcional", "db_required requiere necesidad funcional"))
	}
	return issues
}

func hasDetailedDataNeedV0(datos DatosRequestV0) bool {
	for _, tipo := range datos.TiposDetallados {
		if strings.TrimSpace(tipo.Nombre) != "" || strings.TrimSpace(tipo.Proposito) != "" {
			return true
		}
	}
	for _, storage := range datos.Storage {
		if strings.TrimSpace(storage.Tipo) != "" || strings.TrimSpace(storage.Proposito) != "" {
			return true
		}
	}
	return false
}

func issue(code, field, message string) ValidationIssue {
	return ValidationIssue{Code: code, Field: field, Message: message}
}

func containsV0(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
