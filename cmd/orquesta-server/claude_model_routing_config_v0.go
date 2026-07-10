package main

import (
	"fmt"
	"sort"
	"strings"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacapacity "orquesta/modulos/orquesta-capacity"
)

const (
	claudeModelRefHaikuV0  = "claude-model-ref-haiku-v0"
	claudeModelRefSonnetV0 = "claude-model-ref-sonnet-v0"
	claudeModelRefFableV0  = "claude-model-ref-fable-v0"
)

func claudeModelRoutingFromProjectConfigFileV0(project serverProjectConfigFileV0) orquestaappcodexstack.ClaudeModelRoutingConfigV0 {
	if project.ClaudeModelRouting == nil {
		return defaultClaudeModelRoutingConfigV0()
	}
	routing := project.ClaudeModelRouting
	aliases := map[string]string{}
	for ref, model := range routing.Aliases {
		if ref, model = strings.TrimSpace(ref), strings.TrimSpace(model); ref != "" && model != "" && !strings.Contains(strings.ToLower(model), "opus") {
			aliases[ref] = model
		}
	}
	policyRef := "claude-model-routing-policy-v0"
	if routing.PolicyRef != nil {
		policyRef = strings.TrimSpace(*routing.PolicyRef)
	}
	strict := true
	if routing.Strict != nil {
		strict = *routing.Strict
	}
	routes := map[string]orquestacapacity.ModelRoutingRequestV0{}
	for taskRef, route := range routing.TaskRoutes {
		taskRef = strings.TrimSpace(taskRef)
		routes[taskRef] = orquestacapacity.ModelRoutingRequestV0{
			TaskRef: taskRef, Level: orquestacapacity.ModelRoutingLevelV0(strings.TrimSpace(route.Level)), Trivial: route.Trivial,
			ReasonRef: strings.TrimSpace(route.ReasonRef), EvidenceRefs: append([]string(nil), route.EvidenceRefs...),
			RequestedEffort: strings.TrimSpace(route.RequestedEffort), XHighAuthorizationRef: strings.TrimSpace(route.XHighAuthorizationRef),
		}
	}
	return orquestaappcodexstack.ClaudeModelRoutingConfigV0{
		Policy: orquestacapacity.ModelRoutingPolicyV0{
			PolicyRef: policyRef, Strict: strict,
			TrivialModelRef: claudeModelRefHaikuV0, NormalModelRef: claudeModelRefSonnetV0, CriticalModelRef: claudeModelRefFableV0,
			TrivialEffort: "low", NormalEffort: "medium", ComplexEffort: "high", CriticalEffort: "high",
		},
		ModelAlias: aliases,
		TaskRoutes: routes,
	}
}

func defaultClaudeModelRoutingConfigV0() orquestaappcodexstack.ClaudeModelRoutingConfigV0 {
	return orquestaappcodexstack.DefaultClaudeModelRoutingConfigV0()
}

func claudeModelRoutingEffortsSummaryV0(config orquestaappcodexstack.ClaudeModelRoutingConfigV0) string {
	policy := config.Policy
	return "trivial=" + policy.TrivialEffort + ",normal=" + policy.NormalEffort + ",complex=" + policy.ComplexEffort + ",critical=" + policy.CriticalEffort
}

func claudeModelRoutingAliasRefsV0(config orquestaappcodexstack.ClaudeModelRoutingConfigV0) []string {
	refs := make([]string, 0, len(config.ModelAlias))
	for ref := range config.ModelAlias {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	return refs
}

func claudeModelRoutingConfigSourceV0(project serverProjectConfigFileV0) string {
	if project.ClaudeModelRouting != nil {
		return configSettingSourceConfigFileV0
	}
	return "defaulted"
}

func claudeGoalModelRouteV0(config orquestaappcodexstack.ClaudeModelRoutingConfigV0) (orquestacapacity.ModelRoutingDecisionV0, string, error) {
	decision := orquestacapacity.ResolveModelRoutingV0(config.Policy, orquestacapacity.ModelRoutingRequestV0{TaskRef: "task-ref-claude-goal", Level: orquestacapacity.ModelRoutingLevelNormalV0})
	model := strings.TrimSpace(config.ModelAlias[decision.SelectedModelRef])
	if decision.Rejected || model == "" || strings.Contains(strings.ToLower(model), "opus") || !validClaudeGoalRoutingEffortV0(decision.ReasoningEffort) {
		return decision, "", fmt.Errorf("claude_model_routing_goal_rejected")
	}
	return decision, model, nil
}

func validClaudeGoalRoutingEffortV0(effort string) bool {
	return effort == "low" || effort == "medium" || effort == "high"
}
