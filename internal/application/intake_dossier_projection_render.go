package application

import (
	"html"
	"strconv"
	"strings"

	"orquesta/internal/intake"
)

var intakeDossierProjectionSectionOrder = []IntakeDossierSectionKind{
	IntakeDossierSectionProductScope,
	IntakeDossierSectionUsersRoles,
	IntakeDossierSectionArchitecture,
	IntakeDossierSectionData,
	IntakeDossierSectionIntegrations,
	IntakeDossierSectionSecurityPrivacy,
	IntakeDossierSectionUIUX,
	IntakeDossierSectionI18NL10N,
	IntakeDossierSectionDeployOperations,
	IntakeDossierSectionOrchestrationPlan,
	IntakeDossierSectionRisksOpenIssues,
}

var intakeDossierProjectionDiagramOrder = []IntakeDossierDiagramPurpose{
	IntakeDossierDiagramArchitecture,
	IntakeDossierDiagramUserFlow,
	IntakeDossierDiagramDataIntegrations,
	IntakeDossierDiagramI18N,
	IntakeDossierDiagramDeployment,
}

func renderIntakeDossierProjection(
	spec IntakeDossierProjectionSpec,
) IntakeDossierInput {
	markdown := []string{
		renderProjectionProductScope(spec),
		renderProjectionUsersRoles(spec.UsersRoles),
		renderProjectionArchitecture(spec.Architecture),
		renderProjectionData(spec.Data),
		renderProjectionIntegrations(spec.Integrations),
		renderProjectionSecurityPrivacy(spec.SecurityPrivacy),
		renderProjectionUIUX(spec.UIUX),
		renderProjectionI18N(spec.I18NL10N),
		renderProjectionDeployment(spec.DeployOperations),
		renderProjectionPlan(spec.Plan),
		renderProjectionRisks(spec),
	}
	sections := make([]IntakeDossierSection, len(intakeDossierProjectionSectionOrder))
	for index, kind := range intakeDossierProjectionSectionOrder {
		sections[index] = IntakeDossierSection{
			Ref:      IntakeDossierSectionRef("intake-dossier-section:" + string(kind)),
			Kind:     kind,
			TitleKey: intake.MessageKey("intake.dossier." + string(kind) + ".title"),
			Markdown: markdown[index],
		}
	}

	purposes := append(
		[]IntakeDossierDiagramPurpose(nil),
		intakeDossierProjectionDiagramOrder...,
	)
	sources := []string{
		renderProjectionArchitectureDiagram(spec.Architecture),
		renderProjectionUserFlowDiagram(spec.UsersRoles.MainFlow),
		renderProjectionDataDiagram(spec.Data, spec.Integrations, spec.Architecture),
		renderProjectionI18NDiagram(spec.I18NL10N),
		renderProjectionDeploymentDiagram(spec.DeployOperations),
	}
	if spec.IncludeDecisionsDiagram {
		purposes = append(purposes, IntakeDossierDiagramDecisions)
		sources = append(
			sources, renderProjectionDecisionsDiagram(spec.boundDecisions),
		)
	}
	diagrams := make([]IntakeDossierDiagram, len(purposes))
	for index, purpose := range purposes {
		diagrams[index] = IntakeDossierDiagram{
			Ref:     IntakeDossierDiagramRef("intake-dossier-diagram:" + string(purpose)),
			Purpose: purpose,
			Kind:    IntakeDossierDiagramMermaid,
			Source:  sources[index],
			AltTextKey: intake.MessageKey(
				"intake.dossier.diagram." + string(purpose) + ".alt",
			),
		}
	}

	riskRefs := make([]IntakeRiskRef, len(spec.Risks))
	for index, risk := range spec.Risks {
		riskRefs[index] = risk.Ref
	}
	return IntakeDossierInput{
		Statement: spec.Statement,
		Objective: spec.Objective,
		Sections:  sections,
		Diagrams:  diagrams,
		RiskRefs:  riskRefs,
	}
}

func renderProjectionProductScope(spec IntakeDossierProjectionSpec) string {
	var output strings.Builder
	writeProjectionScalar(&output, "objective", spec.Objective)
	writeProjectionItems(&output, "in_scope", spec.ProductScope.InScope)
	writeProjectionItems(&output, "out_of_scope", spec.ProductScope.OutOfScope)
	writeProjectionDecisions(&output, spec.boundDecisions)
	return strings.TrimSpace(output.String())
}

func renderProjectionUsersRoles(value IntakeDossierUsersRolesProjection) string {
	var output strings.Builder
	writeProjectionItems(&output, "users", value.Users)
	writeProjectionHeading(&output, "roles")
	for _, role := range value.Roles {
		writeProjectionBullet(&output, role.Key, role.Summary)
		for _, permission := range role.PermissionSummaries {
			output.WriteString("  ")
			writeProjectionBullet(&output, permission.Key, permission.Summary)
		}
	}
	writeProjectionHeading(&output, "main_flow")
	for _, step := range value.MainFlow {
		output.WriteString(strconv.FormatUint(uint64(step.Order), 10))
		output.WriteString(". `")
		output.WriteString(step.Key)
		output.WriteString("` — ")
		output.WriteString(projectionMarkdownText(step.Summary))
		output.WriteByte('\n')
	}
	return strings.TrimSpace(output.String())
}

func renderProjectionArchitecture(
	value IntakeDossierArchitectureProjection,
) string {
	var output strings.Builder
	writeProjectionScalar(&output, "rationale", value.Rationale)
	writeProjectionScalar(&output, "domain", value.Domain)
	writeProjectionScalar(&output, "application", value.Application)
	writeProjectionItems(&output, "ports", value.Ports)
	writeProjectionItems(&output, "adapters", value.Adapters)
	writeProjectionScalar(&output, "composition_root", value.CompositionRoot)
	writeProjectionItems(&output, "core_limits", value.CoreLimits)
	return strings.TrimSpace(output.String())
}

func renderProjectionData(value IntakeDossierDataProjection) string {
	var output strings.Builder
	writeProjectionItems(&output, "entities", value.Entities)
	writeProjectionItems(&output, "lifecycles", value.Lifecycles)
	writeProjectionItems(&output, "import_export", value.ImportExport)
	writeProjectionItems(&output, "backups", value.Backups)
	writeProjectionItems(&output, "versioning", value.Versioning)
	return strings.TrimSpace(output.String())
}

func renderProjectionIntegrations(
	value IntakeDossierIntegrationsProjection,
) string {
	var output strings.Builder
	writeProjectionHeading(&output, "connectors")
	for _, connector := range value.Connectors {
		writeProjectionBullet(&output, connector.Key, connector.Summary)
		output.WriteString("  - `authentication` — ")
		output.WriteString(projectionMarkdownText(connector.Authentication))
		output.WriteByte('\n')
		output.WriteString("  - `data_flow` — ")
		output.WriteString(projectionMarkdownText(connector.DataFlow))
		output.WriteByte('\n')
	}
	return strings.TrimSpace(output.String())
}

func renderProjectionSecurityPrivacy(
	value IntakeDossierSecurityPrivacyProjection,
) string {
	var output strings.Builder
	writeProjectionItems(&output, "data_classes", value.DataClasses)
	writeProjectionItems(&output, "authentication", value.Authentication)
	writeProjectionItems(&output, "authorization", value.Authorization)
	writeProjectionItems(&output, "audit", value.Audit)
	writeProjectionItems(&output, "retention", value.Retention)
	writeProjectionItems(&output, "privacy_requirements", value.PrivacyRequirements)
	return strings.TrimSpace(output.String())
}

func renderProjectionUIUX(value IntakeDossierUIUXProjection) string {
	var output strings.Builder
	writeProjectionItems(&output, "views", value.Views)
	writeProjectionItems(&output, "navigation", value.Navigation)
	writeProjectionItems(&output, "empty_states", value.EmptyStates)
	writeProjectionItems(&output, "error_states", value.ErrorStates)
	writeProjectionItems(&output, "accessibility", value.Accessibility)
	writeProjectionItems(&output, "delivery", value.Delivery)
	return strings.TrimSpace(output.String())
}

func renderProjectionI18N(value IntakeDossierI18NL10NProjection) string {
	var output strings.Builder
	writeProjectionHeading(&output, "locales")
	for _, locale := range value.Locales {
		writeProjectionBullet(&output, locale.Tag, locale.Summary)
	}
	writeProjectionScalar(&output, "fallback_locale", value.FallbackLocale)
	writeProjectionScalar(&output, "catalog", value.Catalog)
	writeProjectionItems(&output, "formats", value.Formats)
	writeProjectionItems(&output, "visible_surfaces", value.VisibleSurfaces)
	writeProjectionScalar(&output, "visible_text_rule", value.VisibleTextRule)
	return strings.TrimSpace(output.String())
}

func renderProjectionDeployment(
	value IntakeDossierDeployOperationsProjection,
) string {
	var output strings.Builder
	writeProjectionItems(&output, "environments", value.Environments)
	writeProjectionItems(&output, "deployment_units", value.DeploymentUnits)
	writeProjectionItems(&output, "network", value.Network)
	writeProjectionItems(&output, "observability", value.Observability)
	writeProjectionItems(&output, "backups", value.Backups)
	writeProjectionItems(&output, "updates", value.Updates)
	writeProjectionItems(&output, "rollback", value.Rollback)
	return strings.TrimSpace(output.String())
}

func renderProjectionPlan(plan PlanSpec) string {
	var output strings.Builder
	writeProjectionHeading(&output, "phases")
	for _, phase := range plan.Phases {
		writeProjectionBullet(&output, phase.Key, phase.Ref)
		writeProjectionStringList(&output, "input_refs", phase.InputRefs, "    ")
		writeProjectionStringList(&output, "criterion_refs", phase.CriterionRefs, "    ")
	}
	writeProjectionHeading(&output, "work_items")
	for _, item := range plan.WorkItems {
		writeProjectionBullet(&output, item.Key, item.Objective)
		writeProjectionNestedScalar(&output, "phase", item.Phase)
		writeProjectionNestedScalar(&output, "role", item.Role)
		if item.Parent != "" {
			writeProjectionNestedScalar(&output, "parent", item.Parent)
		}
		writeProjectionStringList(&output, "dependencies", item.Dependencies, "  ")
		writeProjectionStringList(&output, "write_set", item.WriteSet, "  ")
		writeProjectionNestedScalar(
			&output, "output_contract", string(item.OutputContract),
		)
		writeProjectionNestedScalar(
			&output, "council_policy", string(item.CouncilPolicy),
		)
		writeProjectionNestedScalar(
			&output, "security_criticality", string(item.SecurityCriticality),
		)
		writeProjectionNestedScalar(
			&output, "reasoning_effort", string(item.ReasoningEffort),
		)
		writeProjectionStringList(&output, "skill_refs", item.SkillRefs, "  ")
		writeProjectionStringList(&output, "tool_refs", item.ToolRefs, "  ")
		writeProjectionStringList(&output, "capability_refs", item.CapabilityRefs, "  ")
		output.WriteString("  - `handoff_required` — ")
		output.WriteString(strconv.FormatBool(item.HandoffRequired))
		output.WriteByte('\n')
		output.WriteString("  - `required_tests`\n")
		for _, requiredTest := range item.RequiredTests {
			output.WriteString("    - `")
			output.WriteString(projectionMarkdownText(requiredTest.Ref))
			output.WriteString("` — `")
			output.WriteString(projectionMarkdownText(requiredTest.ToolRef))
			output.WriteString("`")
			if requiredTest.WorkingDirectory != "" {
				output.WriteString(" @ `")
				output.WriteString(projectionMarkdownText(requiredTest.WorkingDirectory))
				output.WriteString("`")
			}
			output.WriteByte('\n')
			writeProjectionStringList(
				&output, "arguments", requiredTest.Arguments, "      ",
			)
		}
	}
	return strings.TrimSpace(output.String())
}

func renderProjectionRisks(spec IntakeDossierProjectionSpec) string {
	var output strings.Builder
	writeProjectionHeading(&output, "risks")
	for _, risk := range spec.Risks {
		writeProjectionBullet(&output, string(risk.Ref), risk.Summary)
		output.WriteString("  - `mitigation` — ")
		output.WriteString(projectionMarkdownText(risk.Mitigation))
		output.WriteByte('\n')
		output.WriteString("  - `status_key` — `")
		output.WriteString(string(risk.StatusKey))
		output.WriteString("`\n")
	}
	writeProjectionItems(&output, "open_issues", spec.OpenIssues)
	writeProjectionItems(&output, "deferred_decisions", spec.DeferredDecisions)
	return strings.TrimSpace(output.String())
}

func writeProjectionDecisions(
	output *strings.Builder,
	values []intakeDossierBoundDecisionProjection,
) {
	writeProjectionHeading(output, "decisions")
	for _, bound := range values {
		decision := bound.Decision
		value := bound.View
		writeProjectionBullet(
			output, string(decision.QuestionRef), value.ChoiceSummary,
		)
		writeProjectionNestedScalar(
			output, "choice_ref", string(decision.Choice),
		)
		writeProjectionNestedScalar(
			output, "recommendation_ref", string(decision.Recommendation),
		)
		writeProjectionNestedScalar(
			output, "recommendation", value.RecommendationSummary,
		)
		writeProjectionNestedScalar(
			output, "recommendation_rationale_key",
			string(decision.RecommendationRationale),
		)
		if decision.Choice != decision.Recommendation {
			justified := value.DeviationJustification != ""
			writeProjectionNestedMachineScalar(
				output, "deviation_justified", strconv.FormatBool(justified),
			)
			status := "not_provided"
			if justified {
				status = "provided"
			}
			writeProjectionNestedMachineScalar(
				output, "deviation_justification_status", status,
			)
			if justified {
				writeProjectionNestedScalar(
					output, "deviation_justification",
					value.DeviationJustification,
				)
			}
		}
	}
}

func writeProjectionHeading(output *strings.Builder, key string) {
	if output.Len() > 0 {
		output.WriteByte('\n')
	}
	output.WriteString("### `")
	output.WriteString(key)
	output.WriteString("`\n")
}

func writeProjectionItems(
	output *strings.Builder,
	key string,
	values []IntakeDossierProjectionItem,
) {
	if len(values) == 0 {
		return
	}
	writeProjectionHeading(output, key)
	for _, value := range values {
		writeProjectionBullet(output, value.Key, value.Summary)
	}
}

func writeProjectionScalar(output *strings.Builder, key, value string) {
	writeProjectionHeading(output, key)
	output.WriteString(projectionMarkdownText(value))
	output.WriteByte('\n')
}

func writeProjectionNestedScalar(output *strings.Builder, key, value string) {
	output.WriteString("  - `")
	output.WriteString(key)
	output.WriteString("` — ")
	output.WriteString(projectionMarkdownText(value))
	output.WriteByte('\n')
}

func writeProjectionNestedMachineScalar(
	output *strings.Builder,
	key string,
	value string,
) {
	output.WriteString("  - `")
	output.WriteString(key)
	output.WriteString("` — `")
	output.WriteString(value)
	output.WriteString("`\n")
}

func writeProjectionBullet(output *strings.Builder, key, summary string) {
	output.WriteString("- `")
	output.WriteString(projectionMarkdownText(key))
	output.WriteString("` — ")
	output.WriteString(projectionMarkdownText(summary))
	output.WriteByte('\n')
}

func writeProjectionStringList(
	output *strings.Builder,
	key string,
	values []string,
	indent string,
) {
	if len(values) == 0 {
		return
	}
	output.WriteString(indent)
	output.WriteString("- `")
	output.WriteString(key)
	output.WriteString("` — ")
	for index, value := range values {
		if index > 0 {
			output.WriteString(", ")
		}
		output.WriteString("`")
		output.WriteString(projectionMarkdownText(value))
		output.WriteString("`")
	}
	output.WriteByte('\n')
}

func projectionMarkdownText(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	return strings.NewReplacer(
		"\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_",
		"[", "\\[", "]", "\\]", "<", "&lt;", ">", "&gt;",
		"#", "\\#", "|", "\\|",
	).Replace(value)
}

func renderProjectionArchitectureDiagram(
	value IntakeDossierArchitectureProjection,
) string {
	var output strings.Builder
	output.WriteString("flowchart LR\n")
	writeProjectionDiagramNode(&output, "composition_root", value.CompositionRoot)
	writeProjectionDiagramNode(&output, "adapters", "adapters")
	writeProjectionDiagramNode(&output, "ports", "ports")
	writeProjectionDiagramNode(&output, "application", value.Application)
	writeProjectionDiagramNode(&output, "domain", value.Domain)
	writeProjectionDiagramEdge(&output, "composition_root", "adapters")
	for index, adapter := range value.Adapters {
		ref := "adapter_" + projectionDiagramIndex(index)
		writeProjectionDiagramNode(&output, ref, adapter.Summary)
		writeProjectionDiagramEdge(&output, "adapters", ref)
	}
	writeProjectionDiagramEdge(&output, "adapters", "ports")
	for index, port := range value.Ports {
		ref := "port_" + projectionDiagramIndex(index)
		writeProjectionDiagramNode(&output, ref, port.Summary)
		writeProjectionDiagramEdge(&output, "ports", ref)
	}
	writeProjectionDiagramEdge(&output, "ports", "application")
	writeProjectionDiagramEdge(&output, "application", "domain")
	return strings.TrimSpace(output.String())
}

func renderProjectionUserFlowDiagram(
	values []IntakeDossierProjectionStep,
) string {
	var output strings.Builder
	output.WriteString("flowchart TD\n")
	for index, step := range values {
		ref := "step_" + projectionDiagramIndex(index)
		writeProjectionDiagramNode(&output, ref, step.Summary)
		if index > 0 {
			writeProjectionDiagramEdge(
				&output, "step_"+projectionDiagramIndex(index-1), ref,
			)
		}
	}
	return strings.TrimSpace(output.String())
}

func renderProjectionDataDiagram(
	data IntakeDossierDataProjection,
	integrations IntakeDossierIntegrationsProjection,
	architecture IntakeDossierArchitectureProjection,
) string {
	var output strings.Builder
	output.WriteString("flowchart LR\n")
	writeProjectionDiagramNode(&output, "application", architecture.Application)
	for index, entity := range data.Entities {
		ref := "entity_" + projectionDiagramIndex(index)
		writeProjectionDiagramNode(&output, ref, entity.Summary)
		writeProjectionDiagramEdge(&output, ref, "application")
	}
	for index, connector := range integrations.Connectors {
		ref := "integration_" + projectionDiagramIndex(index)
		writeProjectionDiagramNode(&output, ref, connector.Summary)
		writeProjectionDiagramEdge(&output, "application", ref)
	}
	return strings.TrimSpace(output.String())
}

func renderProjectionI18NDiagram(
	value IntakeDossierI18NL10NProjection,
) string {
	var output strings.Builder
	output.WriteString("flowchart LR\n")
	writeProjectionDiagramNode(&output, "catalog", value.Catalog)
	for index, locale := range value.Locales {
		ref := "locale_" + projectionDiagramIndex(index)
		writeProjectionDiagramNode(
			&output, ref, locale.Tag+": "+locale.Summary,
		)
		writeProjectionDiagramEdge(&output, ref, "catalog")
	}
	for index, surface := range value.VisibleSurfaces {
		ref := "surface_" + projectionDiagramIndex(index)
		writeProjectionDiagramNode(&output, ref, surface.Summary)
		writeProjectionDiagramEdge(&output, "catalog", ref)
	}
	return strings.TrimSpace(output.String())
}

func renderProjectionDeploymentDiagram(
	value IntakeDossierDeployOperationsProjection,
) string {
	var output strings.Builder
	output.WriteString("flowchart TD\n")
	writeProjectionDiagramItems(&output, "environment", value.Environments)
	writeProjectionDiagramItems(&output, "unit", value.DeploymentUnits)
	writeProjectionDiagramItems(&output, "network", value.Network)
	writeProjectionDiagramItems(&output, "observability", value.Observability)
	writeProjectionDiagramItems(&output, "backup", value.Backups)
	writeProjectionDiagramItems(&output, "update", value.Updates)
	writeProjectionDiagramItems(&output, "rollback", value.Rollback)
	writeProjectionDiagramEdge(&output, "environment", "unit")
	writeProjectionDiagramEdge(&output, "unit", "network")
	writeProjectionDiagramEdge(&output, "unit", "observability")
	writeProjectionDiagramEdge(&output, "unit", "backup")
	writeProjectionDiagramEdge(&output, "update", "unit")
	writeProjectionDiagramEdge(&output, "unit", "rollback")
	return strings.TrimSpace(output.String())
}

func renderProjectionDecisionsDiagram(
	values []intakeDossierBoundDecisionProjection,
) string {
	var output strings.Builder
	output.WriteString("flowchart LR\n")
	for index, bound := range values {
		value := bound.View
		recommendation := "recommendation_" + projectionDiagramIndex(index)
		choice := "choice_" + projectionDiagramIndex(index)
		writeProjectionDiagramNode(
			&output, recommendation, value.RecommendationSummary,
		)
		writeProjectionDiagramNode(&output, choice, value.ChoiceSummary)
		writeProjectionDiagramEdge(&output, recommendation, choice)
	}
	return strings.TrimSpace(output.String())
}

func writeProjectionDiagramItems(
	output *strings.Builder,
	prefix string,
	values []IntakeDossierProjectionItem,
) {
	writeProjectionDiagramNode(output, prefix, prefix)
	for index, value := range values {
		ref := prefix + "_" + projectionDiagramIndex(index)
		writeProjectionDiagramNode(
			output, ref, value.Summary,
		)
		writeProjectionDiagramEdge(output, prefix, ref)
	}
}

func writeProjectionDiagramNode(
	output *strings.Builder,
	ref string,
	label string,
) {
	output.WriteString("  ")
	output.WriteString(ref)
	output.WriteString("[\"")
	output.WriteString(projectionDiagramLabel(label))
	output.WriteString("\"]\n")
}

func writeProjectionDiagramEdge(
	output *strings.Builder,
	from string,
	to string,
) {
	output.WriteString("  ")
	output.WriteString(from)
	output.WriteString(" --> ")
	output.WriteString(to)
	output.WriteByte('\n')
}

func projectionDiagramLabel(value string) string {
	return html.EscapeString(strings.Join(strings.Fields(value), " "))
}

func projectionDiagramIndex(index int) string {
	value := strconv.Itoa(index + 1)
	if len(value) >= 4 {
		return value
	}
	return strings.Repeat("0", 4-len(value)) + value
}
