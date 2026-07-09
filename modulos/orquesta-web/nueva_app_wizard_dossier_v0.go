package orquestaweb

import (
	"strconv"
	"strings"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

const WebNuevaAppWizardDossierSchemaV0 = "web_nueva_app_wizard_dossier.v0"

func BuildWebNuevaAppWizardDossierV0(
	spec orquestafactory.AppSpecRequestV0,
	defaults []WizardDefaultV0,
	contrasts []WizardContrastV0,
	ready bool,
) WizardDossierV0 {
	architecture := wizardDossierArchitectureV0(spec)
	i18n := wizardDossierI18NV0(spec)
	connectors := wizardDossierConnectorsV0(spec)
	documentation := wizardDossierDocumentationV0(spec)
	infographics := wizardDossierInfographicsV0(spec, architecture, connectors)
	decisions := wizardDossierDecisionsV0(spec, defaults, contrasts)
	alternatives := wizardDossierAlternativesV0(spec)
	sections := wizardDossierSectionsV0(spec, architecture, i18n, connectors, documentation, decisions, alternatives)
	diagrams := wizardDossierDiagramsV0(infographics)
	return WizardDossierV0{
		SchemaVersion:          WebNuevaAppWizardDossierSchemaV0,
		DossierRef:             wizardDossierRefV0("wizard_dossier", spec.RequestID+"_"+spec.Nombre+"_"+spec.Objetivo+"_"+strings.Join(spec.Plataformas, "_")),
		Ready:                  ready,
		Summary:                wizardDossierSummaryV0(spec, connectors),
		Markdown:               wizardDossierMarkdownV0(sections, diagrams),
		Sections:               sections,
		Diagrams:               diagrams,
		DecisionRefs:           wizardDossierDecisionRefsV0(decisions),
		RiskRefs:               wizardDossierRiskRefsV0(spec),
		Architecture:           architecture,
		I18N:                   i18n,
		Connectors:             connectors,
		Documentation:          documentation,
		Infographics:           infographics,
		Decisions:              decisions,
		Alternatives:           alternatives,
		AcceptanceBeforeLaunch: wizardDossierAcceptanceBeforeLaunchV0(spec),
	}
}

func wizardDossierSummaryV0(
	spec orquestafactory.AppSpecRequestV0,
	connectors []WizardDossierConnectorV0,
) string {
	name := firstNuevaAppValueV0(spec.Nombre, "app sin nombre")
	appType := firstNuevaAppValueV0(spec.TipoApp, "app")
	objective := firstNuevaAppValueV0(spec.Objetivo, "objetivo pendiente")
	audience := strings.Join(compactStringsV0(spec.UsuariosObjetivo), ", ")
	if audience == "" {
		audience = "audiencia pendiente"
	}
	platforms := strings.Join(compactStringsV0(spec.Plataformas), ", ")
	if platforms == "" {
		platforms = "plataformas pendientes"
	}
	return name + " sera una " + appType + " para " + audience + " en " + platforms +
		". Objetivo: " + objective + ". El contrato previo exige arquitectura hexagonal, i18n por defecto, " +
		strconv.Itoa(len(connectors)) + " conectores gobernados, documentacion extensa e infografias de arquitectura antes de aceptar el lanzamiento."
}

func wizardDossierArchitectureV0(spec orquestafactory.AppSpecRequestV0) WizardDossierArchitectureV0 {
	style := firstNuevaAppValueV0(spec.PreferenciasTecnicas.Arquitectura, "hexagonal")
	return WizardDossierArchitectureV0{
		Style: style,
		Layers: []string{
			"dominio sin dependencias de UI, DB, proveedor, runtime ni transporte",
			"aplicacion con casos de uso y puertos",
			"adaptadores inbound para UI/API/MCP",
			"adaptadores outbound para persistencia, conectores, ficheros y proveedores",
			"bootstrap de composicion con wiring y configuracion",
		},
		Ports: wizardDossierPortsV0(spec),
		Adapters: []string{
			"http_ui_adapter",
			"api_adapter",
			"mcp_adapter",
			"persistence_adapter",
			"domain_connector_adapters",
			"provider_boundary_adapter",
		},
		Boundaries: []string{
			"el dominio externo aporta reglas y validadores",
			"Orquesta aporta direccion por contratos y refs opacas",
			"cada conector cruza por puerto/adaptador autorizado",
			"i18n y documentacion nacen como capacidades transversales",
		},
	}
}

func wizardDossierPortsV0(spec orquestafactory.AppSpecRequestV0) []string {
	ports := []string{"app_usecase_port", "validation_port", "i18n_catalog_port", "documentation_port"}
	if spec.Datos.DBRequired || len(spec.Datos.Storage) > 0 {
		ports = append(ports, "repository_port", "migration_port", "backup_restore_port")
	}
	for _, connector := range spec.Integraciones {
		if connector.Tipo == "" && connector.Nombre == "" {
			continue
		}
		ports = append(ports, wizardDossierRefV0("connector_port", firstNuevaAppValueV0(connector.Tipo, connector.Nombre)))
	}
	return compactStringsV0(ports)
}

func wizardDossierI18NV0(spec orquestafactory.AppSpecRequestV0) WizardDossierI18NV0 {
	enabled := true
	if spec.I18N.Enabled != nil {
		enabled = *spec.I18N.Enabled
	}
	defaultLocale := firstNuevaAppValueV0(spec.I18N.DefaultLocale, spec.Locale, NuevaAppI18nDefaultLocaleV0)
	locales := compactStringsV0(append([]string{defaultLocale}, spec.I18N.Locales...))
	return WizardDossierI18NV0{
		Enabled:       enabled,
		DefaultLocale: defaultLocale,
		Locales:       locales,
		Plan: []string{
			"catalogo de UI desde el primer commit",
			"documentacion de usuario y desarrollo localizada",
			"formatos regionales de fecha, numero y moneda por puerto i18n",
			"tests de cobertura de claves y fallback por locale",
		},
	}
}

func wizardDossierConnectorsV0(spec orquestafactory.AppSpecRequestV0) []WizardDossierConnectorV0 {
	out := make([]WizardDossierConnectorV0, 0, len(spec.Integraciones))
	for idx, connector := range spec.Integraciones {
		if connector.Tipo == "" && connector.Nombre == "" && connector.Proposito == "" {
			continue
		}
		base := firstNuevaAppValueV0(connector.Tipo, connector.Nombre, "connector")
		out = append(out, WizardDossierConnectorV0{
			Tipo:          trimV0(connector.Tipo),
			Nombre:        firstNuevaAppValueV0(connector.Nombre, base),
			Proposito:     firstNuevaAppValueV0(connector.Proposito, "capacidad externa gobernada por puerto"),
			Direccion:     trimV0(connector.Direccion),
			Auth:          trimV0(connector.Auth),
			Criticidad:    trimV0(connector.Criticidad),
			Requerido:     connector.Requerido,
			PortRef:       wizardDossierRefV0("connector_port_"+strconv.Itoa(idx+1), base),
			AdapterRef:    wizardDossierRefV0("connector_adapter_"+strconv.Itoa(idx+1), base),
			Restricciones: compactStringsV0(connector.Restricciones),
		})
	}
	return out
}

func wizardDossierDocumentationV0(spec orquestafactory.AppSpecRequestV0) []WizardDossierDocumentationV0 {
	depth := firstNuevaAppValueV0(spec.Documentacion.Profundidad, "profunda")
	return []WizardDossierDocumentationV0{
		{
			ArtifactRef: "wizard_doc_architecture",
			Title:       "Arquitectura hexagonal y boundaries",
			Purpose:     "Explicar modulos, puertos, adaptadores, persistencia y reglas que no deben cruzar capas.",
			Sections:    []string{"contexto", "dominio", "aplicacion", "puertos", "adaptadores", "bootstrap", "tests de arquitectura", "riesgos"},
		},
		{
			ArtifactRef: "wizard_doc_i18n",
			Title:       "Plan i18n/l10n",
			Purpose:     "Dejar locales, catalogos, formatos regionales y pruebas de cobertura listos para implementacion.",
			Sections:    []string{"locales", "catalogos", "formatos", "fallbacks", "tests", "documentacion localizada"},
		},
		{
			ArtifactRef: "wizard_doc_connectors",
			Title:       "Conectores por puertos/adaptadores",
			Purpose:     "Describir cada integracion sin acoplarla a DB, proveedor o filesystem interno.",
			Sections:    []string{"puertos", "adaptadores", "auth", "criticidad", "contratos", "errores", "observabilidad"},
		},
		{
			ArtifactRef: "wizard_doc_decisions",
			Title:       "Decisiones y alternativas",
			Purpose:     "Registrar defaults, desviaciones del usuario, alternativas descartadas y tradeoffs.",
			Sections:    []string{"decisiones", "alternativas", "coste", "migracion", "operacion", "criterios de aceptacion", "profundidad:" + depth},
		},
	}
}

func wizardDossierInfographicsV0(
	spec orquestafactory.AppSpecRequestV0,
	architecture WizardDossierArchitectureV0,
	connectors []WizardDossierConnectorV0,
) []WizardDossierInfographicV0 {
	connectorNodes := []string{"domain", "application_ports"}
	for _, connector := range connectors {
		connectorNodes = append(connectorNodes, connector.PortRef, connector.AdapterRef)
	}
	return []WizardDossierInfographicV0{
		{
			ArtifactRef: "wizard_infographic_architecture",
			Title:       "Mapa de arquitectura hexagonal",
			Format:      "mermaid_flowchart",
			Nodes:       append([]string{"ui_api_mcp", "application", "domain", "outbound_ports"}, architecture.Adapters...),
			Edges:       []string{"ui_api_mcp->application", "application->domain", "application->outbound_ports", "outbound_ports->adapters"},
		},
		{
			ArtifactRef: "wizard_infographic_connectors",
			Title:       "Mapa de conectores y puertos",
			Format:      "mermaid_flowchart",
			Nodes:       connectorNodes,
			Edges:       wizardDossierConnectorEdgesV0(connectors),
		},
		{
			ArtifactRef: "wizard_infographic_i18n_docs",
			Title:       "Flujo i18n y documentacion",
			Format:      "mermaid_flowchart",
			Nodes:       []string{"spec", "i18n_catalog", "ui_texts", "docs_usuario", "docs_desarrollo", firstNuevaAppValueV0(spec.Locale, "locale")},
			Edges:       []string{"spec->i18n_catalog", "i18n_catalog->ui_texts", "i18n_catalog->docs_usuario", "i18n_catalog->docs_desarrollo"},
		},
	}
}

func wizardDossierConnectorEdgesV0(connectors []WizardDossierConnectorV0) []string {
	if len(connectors) == 0 {
		return []string{"domain->application_ports", "application_ports->connector_port_pending"}
	}
	edges := make([]string, 0, len(connectors)*2+1)
	edges = append(edges, "domain->application_ports")
	for _, connector := range connectors {
		edges = append(edges, "application_ports->"+connector.PortRef, connector.PortRef+"->"+connector.AdapterRef)
	}
	return edges
}

func wizardDossierDecisionsV0(
	spec orquestafactory.AppSpecRequestV0,
	defaults []WizardDefaultV0,
	contrasts []WizardContrastV0,
) []WizardDossierDecisionV0 {
	out := []WizardDossierDecisionV0{
		{Area: "arquitectura", Decision: firstNuevaAppValueV0(spec.PreferenciasTecnicas.Arquitectura, "hexagonal"), Rationale: "mantiene dominio, aplicacion, puertos y adaptadores separados", Source: "default"},
		{Area: "i18n", Decision: strings.Join(wizardDossierI18NV0(spec).Locales, ","), Rationale: "la app y sus documentos nacen localizables", Source: "default"},
		{Area: "documentacion", Decision: firstNuevaAppValueV0(spec.Documentacion.Profundidad, "profunda"), Rationale: "el lanzamiento requiere handoff y referencias extensas", Source: "default"},
	}
	for _, value := range defaults {
		out = append(out, WizardDossierDecisionV0{Area: value.Area, Decision: value.Value, Rationale: value.WhyKey, Source: "engineering_default"})
	}
	for _, contrast := range contrasts {
		out = append(out, WizardDossierDecisionV0{Area: contrast.QuestionRef, Decision: contrast.UserChoice, Rationale: contrast.RationaleKey, Source: "user_override"})
	}
	return out
}

func wizardDossierAlternativesV0(spec orquestafactory.AppSpecRequestV0) []WizardDossierAlternativeV0 {
	return []WizardDossierAlternativeV0{
		{Area: "arquitectura", Option: "capas tradicionales", Tradeoff: "mas rapida de explicar pero tiende a mezclar handlers, persistencia y dominio", WhenToUse: "prototipos descartables sin conectores ni evolucion prevista"},
		{Area: "arquitectura", Option: "microservicios", Tradeoff: "escala equipos y despliegues a cambio de mas coordinacion, observabilidad y coste", WhenToUse: "dominios independientes con ownership y escalado realmente separados"},
		{Area: "persistencia", Option: "sin persistencia", Tradeoff: "reduce operacion pero impide historico, auditoria y colaboracion duradera", WhenToUse: "herramientas efimeras o calculadoras sin datos de usuario"},
		{Area: "i18n", Option: "solo locale base", Tradeoff: "menos catalogo inicial pero deuda inmediata si aparece otro idioma", WhenToUse: "apps internas de vida corta con idioma corporativo unico"},
		{Area: "deploy", Option: firstNuevaAppValueV0(spec.Deploy.Target, "contenedor"), Tradeoff: "se valida contra el tipo de app y puede cambiarse por adaptador de despliegue", WhenToUse: "mantener el despliegue como decision de composicion, no de dominio"},
	}
}

func wizardDossierAcceptanceBeforeLaunchV0(spec orquestafactory.AppSpecRequestV0) []string {
	return []string{
		"resumen amplio revisado por el usuario antes de launch",
		"documentacion extensa planificada: arquitectura, i18n, conectores, decisiones y alternativas",
		"infografias de arquitectura, conectores e i18n/documentacion incluidas en el dossier",
		"hexagonal puro conservado como frontera de implementacion",
		"i18n y conectores definidos por puertos/adaptadores",
		"factory valida AppSpecRequestV0 sin issues bloqueantes para " + firstNuevaAppValueV0(spec.RequestID, "request pendiente"),
	}
}

func wizardDossierSectionsV0(
	spec orquestafactory.AppSpecRequestV0,
	architecture WizardDossierArchitectureV0,
	i18n WizardDossierI18NV0,
	connectors []WizardDossierConnectorV0,
	documentation []WizardDossierDocumentationV0,
	decisions []WizardDossierDecisionV0,
	alternatives []WizardDossierAlternativeV0,
) []WizardDossierSectionV0 {
	connectorNames := make([]string, 0, len(connectors))
	for _, connector := range connectors {
		connectorNames = append(connectorNames, connector.Nombre+" via "+connector.PortRef+"/"+connector.AdapterRef)
	}
	docTitles := make([]string, 0, len(documentation))
	for _, doc := range documentation {
		docTitles = append(docTitles, doc.Title+" ("+doc.ArtifactRef+")")
	}
	decisionTexts := make([]string, 0, len(decisions))
	for _, decision := range decisions {
		decisionTexts = append(decisionTexts, decision.Area+": "+decision.Decision+" ["+decision.Source+"]")
	}
	alternativeTexts := make([]string, 0, len(alternatives))
	for _, alternative := range alternatives {
		alternativeTexts = append(alternativeTexts, alternative.Area+": "+alternative.Option+" - "+alternative.Tradeoff)
	}
	return []WizardDossierSectionV0{
		{SectionRef: "objetivo_alcance", Title: "Objetivo y alcance", Markdown: wizardMarkdownListV0([]string{
			"Nombre: " + firstNuevaAppValueV0(spec.Nombre, "pendiente"),
			"Tipo: " + firstNuevaAppValueV0(spec.TipoApp, "pendiente"),
			"Objetivo: " + firstNuevaAppValueV0(spec.Objetivo, "pendiente"),
			"Usuarios: " + firstNuevaAppValueV0(strings.Join(compactStringsV0(spec.UsuariosObjetivo), ", "), "pendiente"),
			"Plataformas: " + firstNuevaAppValueV0(strings.Join(compactStringsV0(spec.Plataformas), ", "), "pendiente"),
		})},
		{SectionRef: "arquitectura_hexagonal", Title: "Arquitectura hexagonal", Markdown: wizardMarkdownListV0(append(append([]string{"Estilo: " + architecture.Style}, architecture.Layers...), architecture.Ports...))},
		{SectionRef: "i18n_l10n", Title: "i18n/l10n", Markdown: wizardMarkdownListV0(append([]string{"Locales: " + strings.Join(i18n.Locales, ", ")}, i18n.Plan...))},
		{SectionRef: "datos_conectores", Title: "Datos e integraciones", Markdown: wizardMarkdownListV0(append([]string{
			"DB requerida: " + strconv.FormatBool(spec.Datos.DBRequired),
			"Sensibilidad: " + firstNuevaAppValueV0(spec.Datos.Sensibilidad, "pendiente"),
		}, connectorNames...))},
		{SectionRef: "documentacion_infografias", Title: "Documentacion e infografias", Markdown: wizardMarkdownListV0(docTitles)},
		{SectionRef: "decisiones_alternativas", Title: "Decisiones y alternativas", Markdown: wizardMarkdownListV0(append(decisionTexts, alternativeTexts...))},
	}
}

func wizardDossierDiagramsV0(infographics []WizardDossierInfographicV0) []WizardDossierDiagramV0 {
	out := make([]WizardDossierDiagramV0, 0, len(infographics))
	for _, infographic := range infographics {
		source := "flowchart TD\n"
		for _, edge := range infographic.Edges {
			source += "  " + strings.ReplaceAll(edge, "->", " --> ") + "\n"
		}
		out = append(out, WizardDossierDiagramV0{
			DiagramRef: infographic.ArtifactRef,
			Kind:       "mermaid",
			Source:     strings.TrimSpace(source),
			AltText:    infographic.Title + " con " + strconv.Itoa(len(infographic.Nodes)) + " nodos",
		})
	}
	return out
}

func wizardDossierMarkdownV0(sections []WizardDossierSectionV0, diagrams []WizardDossierDiagramV0) string {
	var builder strings.Builder
	builder.WriteString("# Dossier previo a aceptar\n\n")
	for _, section := range sections {
		builder.WriteString("## " + section.Title + "\n\n")
		builder.WriteString(strings.TrimSpace(section.Markdown) + "\n\n")
	}
	if len(diagrams) > 0 {
		builder.WriteString("## Infografias\n\n")
		for _, diagram := range diagrams {
			builder.WriteString("### " + diagram.DiagramRef + "\n\n```mermaid\n" + diagram.Source + "\n```\n\n")
		}
	}
	return strings.TrimSpace(builder.String())
}

func wizardMarkdownListV0(values []string) string {
	values = compactStringsV0(values)
	if len(values) == 0 {
		return "- pendiente"
	}
	var builder strings.Builder
	for _, value := range values {
		builder.WriteString("- " + value + "\n")
	}
	return strings.TrimSpace(builder.String())
}

func wizardDossierDecisionRefsV0(decisions []WizardDossierDecisionV0) []string {
	out := make([]string, 0, len(decisions))
	for index, decision := range decisions {
		out = append(out, wizardDossierRefV0("wizard_decision_"+strconv.Itoa(index+1), decision.Area+"_"+decision.Decision))
	}
	return out
}

func wizardDossierRiskRefsV0(spec orquestafactory.AppSpecRequestV0) []string {
	refs := []string{"wizard_risk_scope_drift"}
	if len(spec.Integraciones) > 0 {
		refs = append(refs, "wizard_risk_connector_auth")
	}
	if spec.Datos.DBRequired || len(spec.Datos.Storage) > 0 {
		refs = append(refs, "wizard_risk_data_migration_backup_restore")
	}
	if spec.I18N.Enabled == nil || *spec.I18N.Enabled {
		refs = append(refs, "wizard_risk_i18n_catalog_coverage")
	}
	return refs
}

func wizardDossierRefV0(prefix string, raw string) string {
	normalized := normalizeGuidedNeedV0(raw)
	normalized = strings.ReplaceAll(normalized, " ", "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		normalized = "pending"
	}
	return prefix + "_" + normalized
}
