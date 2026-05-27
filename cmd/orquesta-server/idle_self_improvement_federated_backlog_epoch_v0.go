package main

import "strings"

func (planner idleSelfImprovementBacklogPlannerV0) federatedBacklogNonExecutableContextRefsV0() []string {
	var refs []string
	for _, source := range planner.federatedBacklogSourcesV0() {
		if idleSelfImprovementFederatedBacklogSourceRunnableV0(source) {
			continue
		}
		refs = appendFederatedBacklogContextRefV0(
			refs,
			"federated_backlog_source_not_executable:",
			idleSelfImprovementFederatedBacklogSourceTokenV0(source),
		)
	}
	return compactServerStackStringsV0(refs)
}

func idleSelfImprovementFederatedBacklogSourceTokenV0(source idleSelfImprovementFederatedBacklogSourceV0) string {
	parts := []string{
		"path=" + strings.TrimSpace(source.SourcePath),
		"state=" + strings.TrimSpace(source.State),
	}
	if source.RelatedTXX != "" {
		parts = append(parts, "related_txx="+strings.TrimSpace(source.RelatedTXX))
	}
	return strings.Join(parts, ";")
}
