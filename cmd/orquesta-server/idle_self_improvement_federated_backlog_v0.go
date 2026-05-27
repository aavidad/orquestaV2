package main

import (
	"strings"
)

type idleSelfImprovementFederatedBacklogSourceV0 struct {
	SourcePath, SourceKind string
	Owner, State           string
	RelatedTXX             string
	Aliases, Tests         []string
	SourceLine             int
}

func (planner idleSelfImprovementBacklogPlannerV0) loadFederatedBacklogSectionsV0() []idleSelfImprovementBacklogSectionV0 {
	if strings.TrimSpace(planner.ProjectWorkDir) == "" {
		return nil
	}
	sources := planner.federatedBacklogSourcesV0()
	var sections []idleSelfImprovementBacklogSectionV0
	for _, source := range sources {
		if !idleSelfImprovementFederatedBacklogSourceRunnableV0(source) {
			continue
		}
		body, err := planner.readBacklogScanDocumentTextV0(source.SourcePath)
		if err != nil {
			continue
		}
		sections = append(sections, idleSelfImprovementParseFederatedLocalSectionsV0(body, source)...)
	}
	return sections
}

func (planner idleSelfImprovementBacklogPlannerV0) federatedBacklogSourceRefsV0() []string {
	var refs []string
	for _, source := range planner.federatedBacklogSourcesV0() {
		if !idleSelfImprovementFederatedBacklogSourceRunnableV0(source) {
			continue
		}
		refs = append(refs, source.SourcePath)
	}
	return compactServerStackStringsV0(refs)
}

func (planner idleSelfImprovementBacklogPlannerV0) federatedBacklogSourcesV0() []idleSelfImprovementFederatedBacklogSourceV0 {
	if strings.TrimSpace(planner.ProjectWorkDir) == "" {
		return nil
	}
	body, err := planner.readBacklogScanDocumentTextV0(idleSelfImprovementBacklogDocRelV0)
	if err != nil {
		return nil
	}
	return idleSelfImprovementParseFederatedBacklogSourcesV0(body)
}
