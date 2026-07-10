package main

import (
	"sort"
	"strings"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacapacity "orquesta/modulos/orquesta-capacity"
)

const (
	codexModelRefLunaV0  = "codex-model-ref-luna-v0"
	codexModelRefTerraV0 = "codex-model-ref-terra-v0"
	codexModelRefSolV0   = "codex-model-ref-sol-v0"
)

type serverProjectConfigCodexModelRoutingV0 struct {
	PolicyRef  *string                                   `json:"policy_ref,omitempty"`
	Strict     *bool                                     `json:"strict,omitempty"`
	Aliases    map[string]string                         `json:"aliases,omitempty"`
	TaskRoutes map[string]serverProjectConfigTaskRouteV0 `json:"task_routes,omitempty"`
}

type serverProjectConfigClaudeModelRoutingV0 struct {
	PolicyRef  *string                                   `json:"policy_ref,omitempty"`
	Strict     *bool                                     `json:"strict,omitempty"`
	Aliases    map[string]string                         `json:"aliases,omitempty"`
	TaskRoutes map[string]serverProjectConfigTaskRouteV0 `json:"task_routes,omitempty"`
}

type serverProjectConfigTaskRouteV0 struct {
	Level                 string   `json:"level,omitempty"`
	Trivial               bool     `json:"trivial,omitempty"`
	ReasonRef             string   `json:"reason_ref,omitempty"`
	EvidenceRefs          []string `json:"evidence_refs,omitempty"`
	RequestedEffort       string   `json:"requested_effort,omitempty"`
	XHighAuthorizationRef string   `json:"xhigh_authorization_ref,omitempty"`
}

func codexModelRoutingFromProjectConfigFileV0(project serverProjectConfigFileV0) orquestaappcodexstack.CodexModelRoutingConfigV0 {
	if project.CodexModelRouting == nil {
		return defaultCodexModelRoutingConfigV0()
	}
	routing := project.CodexModelRouting
	aliases := map[string]string{}
	for ref, model := range routing.Aliases {
		if ref, model = strings.TrimSpace(ref), strings.TrimSpace(model); ref != "" && model != "" {
			aliases[ref] = model
		}
	}
	policyRef := "codex-model-routing-policy-v0"
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
	return orquestaappcodexstack.CodexModelRoutingConfigV0{
		Policy: orquestacapacity.ModelRoutingPolicyV0{
			PolicyRef: policyRef, Strict: strict,
			TrivialModelRef: codexModelRefLunaV0, NormalModelRef: codexModelRefTerraV0, CriticalModelRef: codexModelRefSolV0,
			TrivialEffort: "low", NormalEffort: "medium", ComplexEffort: "high", CriticalEffort: "high",
		},
		ModelAlias: aliases,
		TaskRoutes: routes,
	}
}

func defaultCodexModelRoutingConfigV0() orquestaappcodexstack.CodexModelRoutingConfigV0 {
	return orquestaappcodexstack.DefaultCodexModelRoutingConfigV0()
}

func codexModelRoutingEffortsSummaryV0(config orquestaappcodexstack.CodexModelRoutingConfigV0) string {
	policy := config.Policy
	return "trivial=" + policy.TrivialEffort + ",normal=" + policy.NormalEffort + ",complex=" + policy.ComplexEffort + ",critical=" + policy.CriticalEffort
}

func codexModelRoutingAliasRefsV0(config orquestaappcodexstack.CodexModelRoutingConfigV0) []string {
	refs := make([]string, 0, len(config.ModelAlias))
	for ref := range config.ModelAlias {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	return refs
}

func codexModelRoutingConfigSourceV0(project serverProjectConfigFileV0) string {
	if project.CodexModelRouting != nil {
		return configSettingSourceConfigFileV0
	}
	return "defaulted"
}
