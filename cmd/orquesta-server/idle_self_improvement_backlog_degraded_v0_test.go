package main

import (
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestIdleSelfImprovementBacklogPlannerV0DegradadoConCapacidadYTrabajoConocidoNoCreaScannerV0(t *testing.T) {
	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: t.TempDir()}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests:      1,
			Trigger:          "capacity_free",
			KnownRequestRefs: []string{"request-ref-autoprogramming-backlog-t135-idle-self-improvement-planner-fallback-safety"},
			BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
				RequestRef:    "request-ref-base",
				CorrelationID: "corr-request-ref-base",
				ProjectRef:    "project-ref-orquesta",
				WriteSet: []string{
					"cmd/orquesta-server",
					"modulos/orquesta-server",
				},
				RequiredTests: []string{"go test -count=1 ./cmd/orquesta-server"},
			},
		},
	)
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	if len(result.Requests) != 0 ||
		result.Message != "backlog_planner_degraded_known_work:autoprogramming_backlog_doc_unavailable" ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-backlog-known-work") ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-backlog-planner-fallback") {
		t.Fatalf("result=%+v", result)
	}
}
