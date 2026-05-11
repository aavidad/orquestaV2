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

type AppSpecRequestV0 struct {
	SchemaVersion        string                 `json:"schema_version"`
	RequestID            string                 `json:"request_id"`
	Source               string                 `json:"source"`
	Locale               string                 `json:"locale"`
	RequestKind          string                 `json:"request_kind,omitempty"`
	ExecutionMode        string                 `json:"execution_mode,omitempty"`
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
	Requerido     bool     `json:"requerido,omitempty"`
	Restricciones []string `json:"restricciones,omitempty"`
}

type PreferenciasTecnicasV0 struct {
	Lenguaje      string   `json:"lenguaje,omitempty"`
	Framework     string   `json:"framework,omitempty"`
	Arquitectura  string   `json:"arquitectura,omitempty"`
	Restricciones []string `json:"restricciones,omitempty"`
	Preferencias  []string `json:"preferencias,omitempty"`
}

type DatosRequestV0 struct {
	DBRequired         bool     `json:"db_required,omitempty"`
	NecesidadFuncional string   `json:"necesidad_funcional,omitempty"`
	TiposDatos         []string `json:"tipos_datos,omitempty"`
	Sensibilidad       string   `json:"sensibilidad,omitempty"`
	Retencion          string   `json:"retencion,omitempty"`
}

type DeployRequestV0 struct {
	Target        string   `json:"target,omitempty"`
	Restricciones []string `json:"restricciones,omitempty"`
}

type CalidadRequestV0 struct {
	Pruebas        string   `json:"pruebas,omitempty"`
	Accesibilidad  string   `json:"accesibilidad,omitempty"`
	Compliance     []string `json:"compliance,omitempty"`
	Observabilidad *bool    `json:"observabilidad,omitempty"`
}

type DocumentacionV0 struct {
	Usuario    *bool    `json:"usuario,omitempty"`
	Desarrollo *bool    `json:"desarrollo,omitempty"`
	Sistemas   *bool    `json:"sistemas,omitempty"`
	Locales    []string `json:"locales,omitempty"`
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
	if req.TipoApp != "" && !containsV0(req.TipoApp, "web", "api", "cli", "desktop", "mobile", "automation", "data", "plugin", "mixed") {
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
	if accesibilidad := req.Calidad.Accesibilidad; accesibilidad != "" && !containsV0(accesibilidad, "no_aplica", "basica", "wcag_aa") {
		issues = append(issues, issue(ErrAppSpecInvalida, "calidad.accesibilidad", "nivel de accesibilidad no soportado"))
	}
	if autonomia := req.Agentes.Autonomia; autonomia != "" && !containsV0(autonomia, "baja", "media", "alta") {
		issues = append(issues, issue(ErrAppSpecInvalida, "agentes.autonomia", "autonomia no soportada"))
	}
	if arch := req.PreferenciasTecnicas.Arquitectura; arch != "" && arch != "hexagonal" {
		issues = append(issues, issue(ErrOpcionIncompatible, "preferencias_tecnicas.arquitectura", "solo hexagonal esta permitido en v0"))
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
	if !req.Datos.DBRequired {
		return nil
	}
	if strings.TrimSpace(req.Datos.NecesidadFuncional) == "" {
		return []ValidationIssue{issue(ErrAppSpecInvalida, "datos.necesidad_funcional", "db_required requiere necesidad funcional")}
	}
	return nil
}

func validateConnectorNamesV0(req AppSpecRequestV0) []ValidationIssue {
	var issues []ValidationIssue
	for index, connector := range req.Integraciones {
		nombre := strings.TrimSpace(strings.ToLower(connector.Nombre))
		if nombre == "" {
			continue
		}
		if containsV0(nombre, "postgres", "postgresql", "mysql", "sqlite", "mongodb", "redis", "runtime", "filesystem", "fs", "llm", "cache", "queue", "cola", "deploy", "database", "db") {
			issues = append(issues, issue(ErrConectorRequeridoNoDisponible, fmt.Sprintf("integraciones.%d.nombre", index), "la integracion debe nombrar una capacidad, no un proveedor directo"))
		}
	}
	return issues
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
