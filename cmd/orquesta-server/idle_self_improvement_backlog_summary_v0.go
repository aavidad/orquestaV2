package main

import "strings"

func idleSelfImprovementBacklogSummaryV0(section idleSelfImprovementBacklogSectionV0) string {
	objective := strings.TrimSpace(section.Objective)
	if objective == "" {
		objective = "cerrar seccion pendiente de autoprogramacion"
	}
	return "backlog pendiente " + section.Heading + ": " + objective
}

func idleSelfImprovementRequestRefForBacklogSectionV0(
	section idleSelfImprovementBacklogSectionV0,
) string {
	refSuffix := section.Ref + "-" + idleSelfImprovementBacklogHashV0(
		idleSelfImprovementBacklogSectionFingerprintV0(section),
	)
	return "request-ref-autoprogramming-backlog-" + refSuffix
}
