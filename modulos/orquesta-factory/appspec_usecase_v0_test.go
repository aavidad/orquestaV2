package orquestafactory

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSolicitarNuevaAppV0AppliesDefaults(t *testing.T) {
	now := time.Date(2026, 5, 4, 10, 30, 0, 0, time.UTC)
	spec, issues := SolicitarNuevaAppV0(validMinimalRequestV0(), now)
	if len(issues) > 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
	if spec.SchemaVersion != AppSpecSchemaV0 {
		t.Fatalf("schema=%q, want %q", spec.SchemaVersion, AppSpecSchemaV0)
	}
	if spec.SpecID != "spec-panel-de-reservas-req-fty-003-minima" {
		t.Fatalf("spec id=%q", spec.SpecID)
	}
	if spec.CreatedAt != "2026-05-04T10:30:00Z" {
		t.Fatalf("created_at=%q", spec.CreatedAt)
	}
	if spec.App.Slug != "panel-de-reservas" {
		t.Fatalf("slug=%q", spec.App.Slug)
	}
	if spec.Architecture.Patron != "hexagonal" {
		t.Fatalf("architecture=%q", spec.Architecture.Patron)
	}
	if !spec.I18N.Enabled || spec.I18N.DefaultLocale != "es-ES" {
		t.Fatalf("i18n=%+v", spec.I18N)
	}
	if !spec.Docs.User || !spec.Docs.Development || !spec.Docs.Systems || spec.Docs.Depth != "profunda" {
		t.Fatalf("docs defaults not applied: %+v", spec.Docs)
	}
	if spec.Deploy.Target != "sin_preferencia" {
		t.Fatalf("deploy target=%q", spec.Deploy.Target)
	}
	if spec.RequestKind != RequestKindCrearAppCompletaV0 || spec.ExecutionMode != ExecutionModeNormalV0 {
		t.Fatalf("request policy: kind=%q mode=%q", spec.RequestKind, spec.ExecutionMode)
	}
	if spec.ProjectSource.Kind != ProjectSourceKindNewV0 {
		t.Fatalf("project_source=%+v", spec.ProjectSource)
	}
	if spec.Validation.Estado != "valida" {
		t.Fatalf("validation=%+v", spec.Validation)
	}
	if len(spec.DefaultsApplied) == 0 {
		t.Fatalf("expected defaults_applied")
	}
}

func TestSolicitarNuevaAppV0DeclaraHexagonalidadEstricta(t *testing.T) {
	spec, issues := SolicitarNuevaAppV0(validMinimalRequestV0(), time.Date(2026, 6, 19, 9, 0, 0, 0, time.UTC))
	if len(issues) > 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}

	for _, boundary := range []string{"domain", "application", "ports", "adapters", "bootstrap"} {
		if !factoryModuleBoundaryNamedV0(spec.Architecture.ModulosIniciales, boundary) {
			t.Fatalf("missing hexagonal boundary %q in %+v", boundary, spec.Architecture.ModulosIniciales)
		}
	}
	for _, want := range []string{
		"El dominio no importa application, adapters, HTTP, DB, filesystem, runtime, LLM ni framework UI.",
		"Los casos de uso dependen de puertos/interfaces, no de adaptadores concretos.",
		"Los handlers y adaptadores son finos: traducen transporte, validan forma y llaman casos de uso.",
		"La composicion de repositorios, autenticacion, reglas, fixtures y configuracion vive en bootstrap/cmd, no en handlers.",
	} {
		if !factoryStringInSetV0(spec.Architecture.Fronteras, want) {
			t.Fatalf("missing architecture frontier %q in %+v", want, spec.Architecture.Fronteras)
		}
	}
	if !factoryStringInSetV0(spec.Architecture.ContratosEsperados, "ArquitecturaHexagonalEstricta v0") {
		t.Fatalf("missing strict hexagonal contract: %+v", spec.Architecture.ContratosEsperados)
	}
}

func TestSolicitarNuevaAppV0HonorsExplicitOptions(t *testing.T) {
	no := false
	req := validMinimalRequestV0()
	req.Datos = DatosRequestV0{
		DBRequired:         true,
		NecesidadFuncional: "Guardar reservas y cambios de estado.",
		Sensibilidad:       "interna",
		Retencion:          "12 meses",
	}
	req.Calidad.Observabilidad = &no
	req.Documentacion.Usuario = &no
	req.Documentacion.Profundidad = "normal"
	req.Agentes.RevisionHumana = &no
	req.Agentes.Autonomia = "alta"
	req.RequestKind = RequestKindDocumentarAppV0
	req.ExecutionMode = ExecutionModeDebugV0
	req.ProjectSource = ProjectSourceRequestV0{
		Kind:   ProjectSourceKindGitHubV0,
		GitURL: "https://github.com/example/portal.git",
		Branch: "main",
	}

	spec, issues := SolicitarNuevaAppV0(req, time.Date(2026, 5, 4, 11, 0, 0, 0, time.UTC))
	if len(issues) > 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
	if !spec.Data.PersistenceRequired || spec.Data.Connector != "persistence" {
		t.Fatalf("data=%+v", spec.Data)
	}
	if len(spec.Connectors.Required) != 1 || spec.Connectors.Required[0].Nombre != "persistence" {
		t.Fatalf("required connectors=%+v", spec.Connectors.Required)
	}
	if spec.Quality.Observability {
		t.Fatalf("observability should honor explicit false")
	}
	if spec.Docs.User {
		t.Fatalf("docs user should honor explicit false")
	}
	if spec.Docs.Depth != "normal" {
		t.Fatalf("docs depth should honor explicit value: %+v", spec.Docs)
	}
	if spec.AgentPreferences.HumanReview {
		t.Fatalf("human review should honor explicit false")
	}
	if spec.AgentPreferences.Autonomy != "alta" {
		t.Fatalf("autonomy=%q", spec.AgentPreferences.Autonomy)
	}
	if spec.RequestKind != RequestKindDocumentarAppV0 || spec.ExecutionMode != ExecutionModeDebugV0 {
		t.Fatalf("request policy not honored: kind=%q mode=%q", spec.RequestKind, spec.ExecutionMode)
	}
	if spec.ProjectSource.Kind != ProjectSourceKindGitHubV0 ||
		spec.ProjectSource.GitURL != "https://github.com/example/portal.git" ||
		spec.ProjectSource.Branch != "main" {
		t.Fatalf("project_source not honored: %+v", spec.ProjectSource)
	}
}

func TestSolicitarNuevaAppV0PreservaDatosExpertosArquitecturaYAccesibilidadV0(t *testing.T) {
	req := validMinimalRequestV0()
	req.PreferenciasTecnicas.Arquitectura = "event_driven"
	req.Datos = DatosRequestV0{
		DBRequired: true,
		TiposDetallados: []DataTypeRequestV0{
			{
				Nombre:        "Pisos",
				Proposito:     "Mostrar pisos cercanos en alquiler",
				Sensibilidad:  "publica",
				Retencion:     "mientras el anuncio este activo",
				Volumen:       "alto",
				Restricciones: []string{"geolocalizacion aproximada"},
			},
			{
				Nombre:       "Usuarios",
				Proposito:    "Guardar favoritos y alertas",
				Sensibilidad: "personal",
			},
		},
		Storage: []DataStorageRequestV0{
			{
				Tipo:      "relacional",
				Proposito: "Consultas transaccionales de anuncios",
				Requerido: true,
			},
			{
				Tipo:      "vectorial",
				Proposito: "Busqueda semantica de preferencias",
				Requerido: false,
			},
		},
		Fuentes: []DataSourceRequestV0{
			{
				Nombre:     "Catastro publico",
				Tipo:       "api",
				Proposito:  "Cruzar ubicacion y referencia catastral",
				Owner:      "administracion externa",
				Frecuencia: "diaria",
			},
		},
		Operacion: DataOperationRequestV0{
			Criticidad:     "alta",
			Disponibilidad: "horario laboral",
			RPO:            "24h",
			RTO:            "4h",
			Auditoria:      true,
			Restricciones:  []string{"trazabilidad de cambios"},
		},
	}
	req.Integraciones = []ConnectorRequestV0{{
		Tipo:       "api",
		Nombre:     "crm",
		Proposito:  "Sincronizar oportunidades",
		Direccion:  "bidireccional",
		Auth:       "oauth",
		DataScope:  "contactos y favoritos",
		Criticidad: "alta",
		Requerido:  true,
	}}
	req.Calidad.Accesibilidad = "normal"
	req.Calidad.AccesibilidadOpciones = []string{"normal", "wcag_aa", "teclado", "lectores_pantalla", "contraste_alto", "movimiento_reducido", "subtitulos_transcripciones"}

	spec, issues := SolicitarNuevaAppV0(req, time.Date(2026, 6, 25, 11, 0, 0, 0, time.UTC))
	if len(issues) > 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
	if spec.Architecture.Patron != "event_driven" ||
		!factoryModuleBoundaryNamedV0(spec.Architecture.ModulosIniciales, "events") ||
		!factoryStringInSetV0(spec.Architecture.ContratosEsperados, "ArquitecturaLimpiaSegunPatron v0") ||
		factoryStringInSetV0(spec.Architecture.ContratosEsperados, "ArquitecturaHexagonalEstricta v0") {
		t.Fatalf("architecture=%+v", spec.Architecture)
	}
	if len(spec.Data.Types) != 2 ||
		spec.Data.Types[0].Nombre != "Pisos" ||
		spec.Data.Types[1].Sensibilidad != "personal" ||
		len(spec.Data.Storage) != 2 ||
		spec.Data.Storage[0].Tipo != "relacional" ||
		!spec.Data.Storage[0].Requerido {
		t.Fatalf("data experto no normalizado: %+v", spec.Data)
	}
	if !factoryStringInSetV0(spec.Data.Needs, "Mostrar pisos cercanos en alquiler") ||
		!factoryStringInSetV0(spec.Data.Needs, "Busqueda semantica de preferencias") ||
		!factoryStringInSetV0(spec.Data.Needs, "Cruzar ubicacion y referencia catastral") ||
		spec.Data.Sensitivity != "publica, personal" {
		t.Fatalf("data needs/sensitivity=%+v", spec.Data)
	}
	if len(spec.Data.Sources) != 1 ||
		spec.Data.Sources[0].Nombre != "Catastro publico" ||
		spec.Data.Sources[0].Frecuencia != "diaria" ||
		spec.Data.Operation.Criticidad != "alta" ||
		spec.Data.Operation.RTO != "4h" ||
		!spec.Data.Operation.Auditoria {
		t.Fatalf("data sources/operation=%+v", spec.Data)
	}
	if spec.Quality.Accessibility != "normal" ||
		len(spec.Quality.AccessibilityOptions) != 7 ||
		spec.Quality.AccessibilityOptions[1] != "wcag_aa" ||
		spec.Quality.AccessibilityOptions[2] != "teclado" ||
		spec.Quality.AccessibilityOptions[6] != "subtitulos_transcripciones" {
		t.Fatalf("quality=%+v", spec.Quality)
	}
	if !factoryConnectorNamedV0(spec.Connectors.Required, "storage-relacional") ||
		!factoryConnectorNamedV0(spec.Connectors.Optional, "storage-vectorial") {
		t.Fatalf("connectors=%+v", spec.Connectors)
	}
	crm := factoryConnectorByNameV0(spec.Connectors.Required, "crm")
	if crm.Tipo != "api" ||
		crm.Direccion != "bidireccional" ||
		crm.Auth != "oauth" ||
		crm.DataScope != "contactos y favoritos" ||
		crm.Criticidad != "alta" {
		t.Fatalf("connector experto=%+v", crm)
	}
}

func TestSolicitarNuevaAppV0NormalizaAliasAccesibilidadExpertaV0(t *testing.T) {
	req := validMinimalRequestV0()
	req.Calidad.Accesibilidad = "wcag"
	req.Calidad.AccesibilidadOpciones = []string{
		"keyboard",
		"screen readers",
		"high contrast",
		"reduced motion",
		"captions",
	}

	spec, issues := SolicitarNuevaAppV0(req, time.Date(2026, 6, 30, 11, 0, 0, 0, time.UTC))
	if len(issues) > 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
	want := []string{"wcag_aa", "teclado", "lectores_pantalla", "contraste_alto", "movimiento_reducido", "subtitulos_transcripciones"}
	if len(spec.Quality.AccessibilityOptions) != len(want) {
		t.Fatalf("accessibility options=%+v", spec.Quality.AccessibilityOptions)
	}
	for index, value := range want {
		if spec.Quality.AccessibilityOptions[index] != value {
			t.Fatalf("option %d=%q want %q in %+v", index, spec.Quality.AccessibilityOptions[index], value, spec.Quality.AccessibilityOptions)
		}
	}
}

func TestValidateAppSpecRequestV0RechazaAccesibilidadExpertaDesconocidaV0(t *testing.T) {
	req := validMinimalRequestV0()
	req.Calidad.AccesibilidadOpciones = []string{"normal", "proveedor_auditoria_x"}

	issues := ValidateAppSpecRequestV0(req)

	if !hasIssueFieldV0(issues, "calidad.accesibilidad_opciones.1") {
		t.Fatalf("expected accessibility option issue, got %+v", issues)
	}
}

func TestSolicitarNuevaAppV0NoCreaConectorStorageParaSinPreferenciaOPersistencia(t *testing.T) {
	req := validMinimalRequestV0()
	req.Datos.Storage = []DataStorageRequestV0{
		{
			Tipo:      "sin_preferencia",
			Proposito: "El operador delega la decision en Orquesta",
			Requerido: true,
		},
		{
			Tipo:      "sin_persistencia",
			Proposito: "La app no guarda estado durable propio",
			Requerido: true,
		},
	}

	spec, issues := SolicitarNuevaAppV0(req, time.Date(2026, 6, 27, 11, 0, 0, 0, time.UTC))
	if len(issues) > 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
	if factoryConnectorNamedV0(spec.Connectors.Required, "storage-sin_preferencia") ||
		factoryConnectorNamedV0(spec.Connectors.Required, "storage-sin_persistencia") ||
		factoryConnectorNamedV0(spec.Connectors.Optional, "storage-sin_preferencia") ||
		factoryConnectorNamedV0(spec.Connectors.Optional, "storage-sin_persistencia") {
		t.Fatalf("connectors=%+v", spec.Connectors)
	}
}

func TestSolicitarNuevaAppV0DocumentationTypeUsesDocumentationPlatform(t *testing.T) {
	req := validMinimalRequestV0()
	req.TipoApp = "documentacion"
	req.RequestKind = RequestKindDocumentarAppV0

	spec, issues := SolicitarNuevaAppV0(req, time.Date(2026, 5, 15, 9, 0, 0, 0, time.UTC))
	if len(issues) > 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
	if len(spec.Platforms) != 1 || spec.Platforms[0] != "documentation" {
		t.Fatalf("platforms=%+v", spec.Platforms)
	}
}

func TestSolicitarNuevaAppV0ReturnsValidationIssues(t *testing.T) {
	req := validMinimalRequestV0()
	req.Nombre = ""
	_, issues := SolicitarNuevaAppV0(req, time.Time{})
	if !hasIssueCodeV0(issues, ErrAppSpecInvalida) {
		t.Fatalf("expected validation issue, got %+v", issues)
	}
}

func TestSolicitarNuevaAppV0BuildsASCIISlug(t *testing.T) {
	req := validMinimalRequestV0()
	req.Nombre = "Aplicaci\u00f3n de Tesoreria"
	spec, issues := SolicitarNuevaAppV0(req, time.Date(2026, 5, 4, 11, 30, 0, 0, time.UTC))
	if len(issues) > 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
	if spec.App.Slug != "aplicacion-de-tesoreria" {
		t.Fatalf("slug=%q", spec.App.Slug)
	}
}

func TestDecodeAppSpecRequestV0AcceptsSpanishJSONTags(t *testing.T) {
	raw := []byte(`{
		"schema_version":"app_spec_request.v0",
		"request_id":"req-fty-004-tags",
		"source":"orquesta-web",
		"locale":"es-ES",
		"nombre":"Panel de reservas",
		"objetivo":"Gestionar reservas.",
		"tipo_app":"web",
		"calidad":{"accesibilidad":"wcag_aa","observabilidad":false},
		"documentacion":{"usuario":false,"desarrollo":true,"sistemas":true,"profundidad":"profunda"},
		"agentes":{"revision_humana":false,"autonomia":"alta","preferencias":["review externo"]}
	}`)
	req, issues := DecodeAppSpecRequestV0(raw)
	if len(issues) > 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
	if req.Documentacion.Usuario == nil || *req.Documentacion.Usuario {
		t.Fatalf("documentacion.usuario not decoded: %+v", req.Documentacion)
	}
	if req.Documentacion.Profundidad != "profunda" {
		t.Fatalf("documentacion.profundidad not decoded: %+v", req.Documentacion)
	}
	if req.Agentes.RevisionHumana == nil || *req.Agentes.RevisionHumana {
		t.Fatalf("agentes.revision_humana not decoded: %+v", req.Agentes)
	}
	if req.Calidad.Observabilidad == nil || *req.Calidad.Observabilidad {
		t.Fatalf("calidad.observabilidad not decoded: %+v", req.Calidad)
	}
}

func TestValidateAppSpecRequestV0RejectsUnsupportedDeployTarget(t *testing.T) {
	req := validMinimalRequestV0()
	req.Deploy.Target = "mainframe"
	issues := ValidateAppSpecRequestV0(req)
	if !hasIssueCodeV0(issues, ErrTargetNoSoportado) {
		t.Fatalf("expected %s, got %+v", ErrTargetNoSoportado, issues)
	}
}

func TestValidateAppSpecRequestV0RejectsDirectConnectorProvider(t *testing.T) {
	req := validMinimalRequestV0()
	req.Integraciones = []ConnectorRequestV0{{
		Nombre:    "postgres",
		Proposito: "Guardar datos",
		Requerido: true,
	}}
	issues := ValidateAppSpecRequestV0(req)
	if !hasIssueCodeV0(issues, ErrConectorRequeridoNoDisponible) {
		t.Fatalf("expected %s, got %+v", ErrConectorRequeridoNoDisponible, issues)
	}
}

func TestValidateAppSpecRequestV0RejectsUnsupportedRequestPolicy(t *testing.T) {
	req := validMinimalRequestV0()
	req.RequestKind = "hacer_lo_que_sea"
	req.ExecutionMode = "rapido"

	issues := ValidateAppSpecRequestV0(req)

	if !hasIssueFieldV0(issues, "request_kind") || !hasIssueFieldV0(issues, "execution_mode") {
		t.Fatalf("expected request policy issues, got %+v", issues)
	}
}

func TestSolicitarNuevaAppV0InfersLocalProjectSource(t *testing.T) {
	req := validMinimalRequestV0()
	req.RequestKind = RequestKindSeguridadV0
	req.ProjectSource.LocalPath = "/srv/apps/agenda"

	spec, issues := SolicitarNuevaAppV0(req, time.Date(2026, 5, 4, 13, 0, 0, 0, time.UTC))
	if len(issues) > 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
	if spec.ProjectSource.Kind != ProjectSourceKindLocalPathV0 ||
		spec.ProjectSource.LocalPath != "/srv/apps/agenda" {
		t.Fatalf("project_source=%+v", spec.ProjectSource)
	}
}

func TestSolicitarNuevaAppV0InfersGitHubProjectSource(t *testing.T) {
	req := validMinimalRequestV0()
	req.RequestKind = RequestKindRevisarCodigoV0
	req.ProjectSource.GitURL = "https://github.com/example/agenda.git"

	spec, issues := SolicitarNuevaAppV0(req, time.Date(2026, 5, 4, 13, 30, 0, 0, time.UTC))
	if len(issues) > 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
	if spec.ProjectSource.Kind != ProjectSourceKindGitHubV0 ||
		spec.ProjectSource.GitURL != "https://github.com/example/agenda.git" {
		t.Fatalf("project_source=%+v", spec.ProjectSource)
	}
}

func TestAppSpecV0SerializesPublicIssueShape(t *testing.T) {
	spec, issues := SolicitarNuevaAppV0(validMinimalRequestV0(), time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC))
	if len(issues) > 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal spec: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode marshaled spec: %v", err)
	}
	if decoded["schema_version"] != AppSpecSchemaV0 {
		t.Fatalf("schema_version missing from JSON: %s", data)
	}
	assertJSONArrayV0(t, decoded, "defaults_applied")
	assertJSONArrayV0(t, decoded["validation"].(map[string]any), "warnings")
	assertJSONArrayV0(t, decoded["validation"].(map[string]any), "errores")
	assertJSONArrayV0(t, decoded["connectors"].(map[string]any), "required")
	assertJSONArrayV0(t, decoded["connectors"].(map[string]any), "optional")
	assertJSONArrayV0(t, decoded["app"].(map[string]any), "usuarios_objetivo")
	projectSource, ok := decoded["project_source"].(map[string]any)
	if !ok || projectSource["kind"] != ProjectSourceKindNewV0 {
		t.Fatalf("project_source missing from JSON: %s", data)
	}
}

func assertJSONArrayV0(t *testing.T, object map[string]any, key string) {
	t.Helper()
	if _, ok := object[key].([]any); !ok {
		t.Fatalf("%s should be JSON array, got %#v", key, object[key])
	}
}

func factoryModuleBoundaryNamedV0(boundaries []ModuleBoundaryV0, name string) bool {
	for _, boundary := range boundaries {
		if boundary.Nombre == name {
			return true
		}
	}
	return false
}

func factoryStringInSetV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func factoryConnectorNamedV0(values []ConnectorSpecV0, name string) bool {
	for _, value := range values {
		if value.Nombre == name {
			return true
		}
	}
	return false
}

func factoryConnectorByNameV0(values []ConnectorSpecV0, name string) ConnectorSpecV0 {
	for _, value := range values {
		if value.Nombre == name {
			return value
		}
	}
	return ConnectorSpecV0{}
}
