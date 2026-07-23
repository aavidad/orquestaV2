package bootstrap

import (
	"errors"
	"strings"

	"orquesta/internal/adapters/agent/codex"
	"orquesta/internal/i18n"
)

const codexAgentPromptKey = "prompt.codex.agent"

type catalogCodexPromptRenderer struct {
	catalog *i18n.Catalog
	locale  string
}

func newCatalogCodexPromptRenderer(catalog *i18n.Catalog, locale string) (codex.PromptRenderer, error) {
	if catalog == nil {
		return nil, errors.New("bootstrap.i18n_catalog_unavailable")
	}
	resolution, err := catalog.Resolve(locale)
	if err != nil {
		return nil, err
	}
	return catalogCodexPromptRenderer{catalog: catalog, locale: resolution.Locale}, nil
}

func (renderer catalogCodexPromptRenderer) RenderAgentPrompt(prompt codex.AgentPrompt) (string, error) {
	catalog := renderer.catalog
	template, err := catalog.Text(renderer.locale, codexAgentPromptKey)
	if err != nil {
		return "", err
	}
	return strings.NewReplacer(
		"{project_ref}", prompt.ProjectRef, "{goal_ref}", prompt.GoalRef,
		"{work_item_ref}", prompt.WorkItemRef, "{execution_ref}", prompt.ExecutionRef,
		"{plan_generation}", prompt.PlanGeneration, "{app_spec_generation}", prompt.AppSpecGeneration,
		"{objective}", prompt.Objective, "{phase_ref}", prompt.PhaseRef,
		"{phase_key}", prompt.PhaseKey, "{phase_template_ref}", prompt.PhaseTemplateRef,
		"{phase_input_refs}", prompt.PhaseInputRefs, "{phase_criterion_refs}", prompt.PhaseCriterionRefs,
		"{role_key}", prompt.RoleKey, "{skill_refs}", prompt.SkillRefs,
		"{tool_refs}", prompt.ToolRefs, "{capability_refs}", prompt.CapabilityRefs,
		"{write_set}", prompt.WriteSet, "{output_contract}", prompt.OutputContract,
		"{artifact_media_type}", prompt.ArtifactMediaType,
	).Replace(template), nil
}
