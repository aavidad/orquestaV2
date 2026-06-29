package orquestafactory

import "strings"

type appSpecNormalizerV0 struct {
	req AppSpecRequestV0
}

func (n appSpecNormalizerV0) requestKind() string {
	return NormalizeRequestKindV0(n.req.RequestKind)
}

func (n appSpecNormalizerV0) executionMode() string {
	return NormalizeExecutionModeV0(n.req.ExecutionMode)
}

func (n appSpecNormalizerV0) projectSource() ProjectSourceSpecV0 {
	return normalizeProjectSourceV0(n.req.ProjectSource)
}

func (n appSpecNormalizerV0) app() AppInfoV0 {
	return AppInfoV0{
		Nombre:           strings.TrimSpace(n.req.Nombre),
		Slug:             slugV0(n.req.Nombre),
		Objetivo:         strings.TrimSpace(n.req.Objetivo),
		Descripcion:      strings.TrimSpace(n.req.Descripcion),
		TipoApp:          strings.TrimSpace(n.req.TipoApp),
		UsuariosObjetivo: emptyStringsV0(compactUniqueV0(n.req.UsuariosObjetivo)),
	}
}

func (n appSpecNormalizerV0) scope() ScopeV0 {
	pattern := architecturePatternOrDefaultV0(n.req.PreferenciasTecnicas.Arquitectura)
	return ScopeV0{
		Objetivos:         []string{strings.TrimSpace(n.req.Objetivo)},
		FueraDeAlcance:    []string{},
		Supuestos:         []string{"Arquitectura " + pattern + ", i18n y documentacion se aplican por defecto y son condiciones de aceptacion de la app generada."},
		PreguntasAbiertas: []string{},
	}
}

func (n appSpecNormalizerV0) architecture() ArchitectureV0 {
	pattern := architecturePatternOrDefaultV0(n.req.PreferenciasTecnicas.Arquitectura)
	return ArchitectureV0{
		Patron:             pattern,
		ModulosIniciales:   architectureModulesV0(pattern),
		Fronteras:          architectureFrontiersV0(pattern),
		ContratosEsperados: architectureContractsV0(pattern),
	}
}

func architectureModulesV0(pattern string) []ModuleBoundaryV0 {
	modules := []ModuleBoundaryV0{
		{Nombre: "domain", Responsabilidad: "Entidades, value objects y reglas puras sin framework, IO ni adaptadores.", Puertos: []string{"domain_services"}},
		{Nombre: "application", Responsabilidad: "Casos de uso, comandos, consultas y DTOs de aplicacion sobre puertos.", Puertos: []string{"input_ports", "output_ports"}},
		{Nombre: "ports", Responsabilidad: "Interfaces de entrada y salida versionadas que conectan aplicacion con adaptadores.", Puertos: []string{"inbound", "outbound"}},
		{Nombre: "adapters", Responsabilidad: "HTTP, CLI, persistencia, cache, colas o clientes externos sin reglas de negocio ni composicion global.", Puertos: []string{"http", "cli", "persistence"}},
		{Nombre: "bootstrap", Responsabilidad: "Composicion de dependencias, configuracion canonica y wiring de adaptadores.", Puertos: []string{"composition_root"}},
		{Nombre: "i18n-docs", Responsabilidad: "Catalogos i18n, loader y documentacion inicial localizada.", Puertos: []string{"i18n_bundle", "docs_bundle"}},
	}
	switch pattern {
	case "event_driven":
		modules = append(modules, ModuleBoundaryV0{Nombre: "events", Responsabilidad: "Eventos de dominio, handlers asincronos y contratos de mensajeria sin proveedor concreto.", Puertos: []string{"event_bus"}})
	case "microservices":
		modules = append(modules, ModuleBoundaryV0{Nombre: "service-boundaries", Responsabilidad: "Cortes de servicio, contratos entre servicios y reglas de propiedad de datos.", Puertos: []string{"service_contracts"}})
	case "serverless":
		modules = append(modules, ModuleBoundaryV0{Nombre: "functions", Responsabilidad: "Funciones de entrada finas que delegan en application y se componen en bootstrap.", Puertos: []string{"function_entrypoints"}})
	case "plugin_based":
		modules = append(modules, ModuleBoundaryV0{Nombre: "plugins", Responsabilidad: "Puntos de extension versionados y aislamiento de plugins mediante puertos.", Puertos: []string{"plugin_contracts"}})
	case "data_pipeline":
		modules = append(modules, ModuleBoundaryV0{Nombre: "pipeline", Responsabilidad: "Ingesta, transformacion y publicacion de datos con pasos versionados e idempotentes.", Puertos: []string{"pipeline_steps"}})
	}
	return modules
}

func architectureFrontiersV0(pattern string) []string {
	return []string{
		"El dominio no importa application, adapters, HTTP, DB, filesystem, runtime, LLM ni framework UI.",
		"Los casos de uso dependen de puertos/interfaces, no de adaptadores concretos.",
		"Los handlers y adaptadores son finos: traducen transporte, validan forma y llaman casos de uso.",
		"La composicion de repositorios, autenticacion, reglas, fixtures y configuracion vive en bootstrap/cmd, no en handlers.",
		"Cada dependencia externa entra como conector versionado.",
		"El patron " + pattern + " guia el corte de modulos, pero no autoriza a mezclar reglas de dominio con transporte, persistencia o proveedores.",
	}
}

func architectureContractsV0(pattern string) []string {
	contracts := []string{"SolicitarNuevaApp v0", "AppSpecV0", "ArquitecturaLimpiaSegunPatron v0"}
	if pattern == "hexagonal" {
		contracts = append(contracts, "ArquitecturaHexagonalEstricta v0")
	}
	return contracts
}

func (n appSpecNormalizerV0) i18n() I18NSpecV0 {
	enabled := true
	if n.req.I18N.Enabled != nil {
		enabled = *n.req.I18N.Enabled
	}
	defaultLocale := firstNonEmptyV0(n.req.I18N.DefaultLocale, n.req.Locale)
	locales := compactUniqueV0(append([]string{defaultLocale}, n.req.I18N.Locales...))
	return I18NSpecV0{
		Enabled:       enabled,
		DefaultLocale: defaultLocale,
		Locales:       emptyStringsV0(locales),
		Justificacion: strings.TrimSpace(n.req.I18N.Justificacion),
	}
}

func (n appSpecNormalizerV0) data() DataSpecV0 {
	dataTypes := normalizeDataTypesV0(n.req.Datos)
	storage := normalizeDataStorageV0(n.req.Datos.Storage)
	data := DataSpecV0{
		PersistenceRequired: n.req.Datos.DBRequired,
		Needs:               emptyStringsV0(dataNeedsV0(n.req.Datos, dataTypes, storage)),
		Types:               dataTypes,
		Storage:             storage,
		Sensitivity:         dataSensitivityV0(n.req.Datos, dataTypes),
		Retention:           strings.TrimSpace(n.req.Datos.Retencion),
	}
	if data.PersistenceRequired {
		data.Connector = "persistence"
	}
	return data
}

func (n appSpecNormalizerV0) connectors() ConnectorsSpecV0 {
	var required []ConnectorSpecV0
	var optional []ConnectorSpecV0
	if n.req.Datos.DBRequired {
		proposito := firstNonEmptyV0(strings.TrimSpace(n.req.Datos.NecesidadFuncional), firstDataNeedV0(n.req.Datos))
		required = append(required, ConnectorSpecV0{
			Nombre:    "persistence",
			Proposito: proposito,
			Contrato:  "PersistenceRepository v0",
		})
	}
	for _, storage := range normalizeDataStorageV0(n.req.Datos.Storage) {
		tipo := strings.TrimSpace(storage.Tipo)
		if tipo == "" || !dataStorageTypeCreatesConnectorV0(tipo) || strings.TrimSpace(storage.Proposito) == "" {
			continue
		}
		spec := ConnectorSpecV0{
			Nombre:    "storage-" + tipo,
			Proposito: strings.TrimSpace(storage.Proposito),
			Contrato:  "PersistenceRepository v0",
		}
		if storage.Requerido {
			required = append(required, spec)
			continue
		}
		optional = append(optional, spec)
	}
	for _, connector := range n.req.Integraciones {
		spec := ConnectorSpecV0{
			Nombre:    strings.TrimSpace(connector.Nombre),
			Proposito: strings.TrimSpace(connector.Proposito),
		}
		if spec.Nombre == "" || spec.Proposito == "" {
			continue
		}
		if connector.Requerido {
			required = append(required, spec)
			continue
		}
		optional = append(optional, spec)
	}
	return ConnectorsSpecV0{Required: emptyConnectorsV0(required), Optional: emptyConnectorsV0(optional)}
}

func dataStorageTypeCreatesConnectorV0(tipo string) bool {
	switch strings.TrimSpace(tipo) {
	case "", "sin_preferencia", "sin_persistencia":
		return false
	default:
		return true
	}
}

func normalizeDataTypesV0(datos DatosRequestV0) []DataTypeSpecV0 {
	out := make([]DataTypeSpecV0, 0, len(datos.TiposDatos)+len(datos.TiposDetallados))
	seen := map[string]bool{}
	for _, name := range compactUniqueV0(datos.TiposDatos) {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, DataTypeSpecV0{Nombre: strings.TrimSpace(name)})
	}
	for _, value := range datos.TiposDetallados {
		spec := DataTypeSpecV0{
			Nombre:        strings.TrimSpace(value.Nombre),
			Proposito:     strings.TrimSpace(value.Proposito),
			Sensibilidad:  strings.TrimSpace(value.Sensibilidad),
			Retencion:     strings.TrimSpace(value.Retencion),
			Volumen:       strings.TrimSpace(value.Volumen),
			Restricciones: emptyStringsV0(compactUniqueV0(value.Restricciones)),
		}
		if spec.Nombre == "" && spec.Proposito == "" {
			continue
		}
		key := strings.ToLower(firstNonEmptyV0(spec.Nombre, spec.Proposito))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, spec)
	}
	if out == nil {
		return []DataTypeSpecV0{}
	}
	return out
}

func normalizeDataStorageV0(values []DataStorageRequestV0) []DataStorageSpecV0 {
	out := make([]DataStorageSpecV0, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		spec := DataStorageSpecV0{
			Tipo:          strings.TrimSpace(value.Tipo),
			Proposito:     strings.TrimSpace(value.Proposito),
			Requerido:     value.Requerido,
			Restricciones: emptyStringsV0(compactUniqueV0(value.Restricciones)),
		}
		if spec.Tipo == "" && spec.Proposito == "" && len(spec.Restricciones) == 0 && !spec.Requerido {
			continue
		}
		key := strings.ToLower(firstNonEmptyV0(spec.Tipo, spec.Proposito))
		if key != "" && seen[key] {
			continue
		}
		if key != "" {
			seen[key] = true
		}
		out = append(out, spec)
	}
	if out == nil {
		return []DataStorageSpecV0{}
	}
	return out
}

func dataNeedsV0(datos DatosRequestV0, dataTypes []DataTypeSpecV0, storage []DataStorageSpecV0) []string {
	values := []string{datos.NecesidadFuncional}
	for _, value := range dataTypes {
		values = append(values, value.Proposito)
	}
	for _, value := range storage {
		values = append(values, value.Proposito)
	}
	return compactUniqueV0(values)
}

func dataSensitivityV0(datos DatosRequestV0, dataTypes []DataTypeSpecV0) string {
	values := []string{strings.TrimSpace(datos.Sensibilidad)}
	for _, value := range dataTypes {
		values = append(values, value.Sensibilidad)
	}
	return strings.Join(compactUniqueV0(values), ", ")
}

func firstDataNeedV0(datos DatosRequestV0) string {
	for _, value := range dataNeedsV0(datos, normalizeDataTypesV0(datos), normalizeDataStorageV0(datos.Storage)) {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "Persistencia de datos de la aplicacion"
}

func (n appSpecNormalizerV0) platforms() []string {
	platforms := compactUniqueV0(n.req.Plataformas)
	if len(platforms) > 0 {
		return platforms
	}
	return defaultPlatformsV0(n.req.TipoApp)
}

func (n appSpecNormalizerV0) deploy() DeploySpecV0 {
	return DeploySpecV0{
		Target:       firstNonEmptyV0(n.req.Deploy.Target, "sin_preferencia"),
		Restrictions: emptyStringsV0(compactUniqueV0(n.req.Deploy.Restricciones)),
	}
}

func (n appSpecNormalizerV0) quality() QualitySpecV0 {
	accessibilityOptions := normalizeAccessibilityOptionsV0(n.req.Calidad)
	return QualitySpecV0{
		Tests:                firstNonEmptyV0(n.req.Calidad.Pruebas, "basica"),
		Accessibility:        accessibilityOptions[0],
		AccessibilityOptions: emptyStringsV0(accessibilityOptions),
		Security:             []string{"sin_secretos_hardcodeados"},
		Compliance:           emptyStringsV0(compactUniqueV0(n.req.Calidad.Compliance)),
		Observability:        boolDefaultV0(n.req.Calidad.Observabilidad, true),
	}
}

func (n appSpecNormalizerV0) docs() DocsSpecV0 {
	return DocsSpecV0{
		User:        boolDefaultV0(n.req.Documentacion.Usuario, true),
		Development: boolDefaultV0(n.req.Documentacion.Desarrollo, true),
		Systems:     boolDefaultV0(n.req.Documentacion.Sistemas, true),
		Depth:       normalizeDocsDepthV0(n.req.Documentacion.Profundidad),
		Locales:     docsLocalesV0(n.req, n.i18n().Locales),
	}
}

func normalizeDocsDepthV0(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "basica", "basic":
		return "basica"
	case "normal", "media", "standard":
		return "normal"
	case "profunda", "deep", "completa":
		return "profunda"
	default:
		return "profunda"
	}
}

func (n appSpecNormalizerV0) agentPreferences() AgentPreferencesV0 {
	return AgentPreferencesV0{
		HumanReview: boolDefaultV0(n.req.Agentes.RevisionHumana, true),
		Autonomy:    firstNonEmptyV0(n.req.Agentes.Autonomia, "media"),
		Notes:       compactUniqueV0(n.req.Agentes.Preferencias),
	}
}
