package main

import (
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func (planner idleSelfImprovementBacklogPlannerV0) degradedBacklogPlanResultV0(
	base orquestaserver.IdleSelfImprovementRequestV0,
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
	reason string,
) orquestaserver.IdleSelfImprovementPlanResultV0 {
	if strings.TrimSpace(request.Trigger) == "capacity_free" &&
		idleSelfImprovementHasKnownBacklogWorkV0(idleSelfImprovementExcludedRequestRefsV0(request)) {
		return orquestaserver.IdleSelfImprovementPlanResultV0{
			Requests: []orquestaserver.IdleSelfImprovementRequestV0{},
			EvidenceRefs: []string{
				"evidence-ref-autoprogramming-backlog-known-work",
				"evidence-ref-autoprogramming-backlog-planner-fallback",
			},
			Message: "backlog_planner_degraded_known_work:" + strings.TrimSpace(reason),
		}
	}
	fallback := idleSelfImprovementBacklogFallbackRequestV0(base, request, reason)
	fallback = planner.withBacklogScannerMergeLeaseV0(fallback)
	return orquestaserver.IdleSelfImprovementPlanResultV0{
		Requests: []orquestaserver.IdleSelfImprovementRequestV0{fallback},
		EvidenceRefs: []string{
			"evidence-ref-autoprogramming-backlog-planner-fallback",
			"evidence-ref-autoprogramming-backlog-scanner",
		},
		Message: strings.TrimSpace(reason),
	}
}
