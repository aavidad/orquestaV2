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
	return ScopeV0{
		Objetivos:         []string{strings.TrimSpace(n.req.Objetivo)},
		FueraDeAlcance:    []string{},
		Supuestos:         []string{"Arquitectura hexagonal estricta, i18n y documentacion se aplican por defecto y son condiciones de aceptacion de la app generada."},
		PreguntasAbiertas: []string{},
	}
}

func (n appSpecNormalizerV0) architecture() ArchitectureV0 {
	return ArchitectureV0{
		Patron: "hexagonal",
		ModulosIniciales: []ModuleBoundaryV0{
			{Nombre: "domain", Responsabilidad: "Entidades, value objects y reglas puras sin framework, IO ni adaptadores.", Puertos: []string{"domain_services"}},
			{Nombre: "application", Responsabilidad: "Casos de uso, comandos, consultas y DTOs de aplicacion sobre puertos.", Puertos: []string{"input_ports", "output_ports"}},
			{Nombre: "ports", Responsabilidad: "Interfaces de entrada y salida versionadas que conectan aplicacion con adaptadores.", Puertos: []string{"inbound", "outbound"}},
			{Nombre: "adapters", Responsabilidad: "HTTP, CLI, persistencia, cache, colas o clientes externos sin reglas de negocio ni composicion global.", Puertos: []string{"http", "cli", "persistence"}},
			{Nombre: "bootstrap", Responsabilidad: "Composicion de dependencias, configuracion canonica y wiring de adaptadores.", Puertos: []string{"composition_root"}},
			{Nombre: "i18n-docs", Responsabilidad: "Catalogos i18n, loader y documentacion inicial localizada.", Puertos: []string{"i18n_bundle", "docs_bundle"}},
		},
		Fronteras: []string{
			"El dominio no importa application, adapters, HTTP, DB, filesystem, runtime, LLM ni framework UI.",
			"Los casos de uso dependen de puertos/interfaces, no de adaptadores concretos.",
			"Los handlers y adaptadores son finos: traducen transporte, validan forma y llaman casos de uso.",
			"La composicion de repositorios, autenticacion, reglas, fixtures y configuracion vive en bootstrap/cmd, no en handlers.",
			"Cada dependencia externa entra como conector versionado.",
		},
		ContratosEsperados: []string{"SolicitarNuevaApp v0", "AppSpecV0", "ArquitecturaHexagonalEstricta v0"},
	}
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
	data := DataSpecV0{
		PersistenceRequired: n.req.Datos.DBRequired,
		Needs:               emptyStringsV0(compactUniqueV0([]string{n.req.Datos.NecesidadFuncional})),
		Sensitivity:         strings.TrimSpace(n.req.Datos.Sensibilidad),
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
		required = append(required, ConnectorSpecV0{
			Nombre:    "persistence",
			Proposito: strings.TrimSpace(n.req.Datos.NecesidadFuncional),
			Contrato:  "PersistenceRepository v0",
		})
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
	return QualitySpecV0{
		Tests:         firstNonEmptyV0(n.req.Calidad.Pruebas, "basica"),
		Accessibility: firstNonEmptyV0(n.req.Calidad.Accesibilidad, "basica"),
		Security:      []string{"sin_secretos_hardcodeados"},
		Compliance:    emptyStringsV0(compactUniqueV0(n.req.Calidad.Compliance)),
		Observability: boolDefaultV0(n.req.Calidad.Observabilidad, true),
	}
}

func (n appSpecNormalizerV0) docs() DocsSpecV0 {
	return DocsSpecV0{
		User:        boolDefaultV0(n.req.Documentacion.Usuario, true),
		Development: boolDefaultV0(n.req.Documentacion.Desarrollo, true),
		Systems:     boolDefaultV0(n.req.Documentacion.Sistemas, true),
		Locales:     docsLocalesV0(n.req, n.i18n().Locales),
	}
}

func (n appSpecNormalizerV0) agentPreferences() AgentPreferencesV0 {
	return AgentPreferencesV0{
		HumanReview: boolDefaultV0(n.req.Agentes.RevisionHumana, true),
		Autonomy:    firstNonEmptyV0(n.req.Agentes.Autonomia, "media"),
		Notes:       compactUniqueV0(n.req.Agentes.Preferencias),
	}
}
