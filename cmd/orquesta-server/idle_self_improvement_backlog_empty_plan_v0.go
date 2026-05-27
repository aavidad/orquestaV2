package main

import orquestaserver "orquesta/modulos/orquesta-server"

func (planner idleSelfImprovementBacklogPlannerV0) emptyBacklogSectionsPlanResultV0(
	base orquestaserver.IdleSelfImprovementRequestV0,
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
	excluded map[string]bool,
) orquestaserver.IdleSelfImprovementPlanResultV0 {
	if idleSelfImprovementHasKnownBacklogWorkV0(excluded) {
		return orquestaserver.IdleSelfImprovementPlanResultV0{
			Requests: []orquestaserver.IdleSelfImprovementRequestV0{},
			EvidenceRefs: []string{
				"evidence-ref-autoprogramming-backlog-known-work",
				"evidence-ref-autoprogramming-backlog-planner-fallback",
			},
			Message: "backlog_tareas_ya_visibles_en_cola",
		}
	}
	fallback := idleSelfImprovementBacklogFallbackRequestV0(
		base,
		request,
		"backlog_sin_tareas_pendientes_detectadas",
	)
	fallback = planner.withBacklogScannerMergeLeaseV0(fallback)
	return orquestaserver.IdleSelfImprovementPlanResultV0{
		Requests: []orquestaserver.IdleSelfImprovementRequestV0{fallback},
		EvidenceRefs: []string{
			"evidence-ref-autoprogramming-backlog-planner-fallback",
			"evidence-ref-autoprogramming-backlog-scanner",
		},
		Message: "backlog_sin_tareas_pendientes_detectadas",
	}
}
