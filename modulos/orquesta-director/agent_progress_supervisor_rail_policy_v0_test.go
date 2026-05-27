package orquestadirector

import (
	"reflect"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestAgentProgressSupervisorRailPolicyV0ConservaRefsOpacasOperativas(t *testing.T) {
	report := orquestaruntime.AgentProgressReportV0{
		ReportID:       "agent-progress-report-ref-provider-runtime-db-sql-home-model-001",
		EvidenceRefs:   []string{"evidence-ref-runtime-provider-model-db-sql-home-001"},
		AgentRequestID: "agent-request-ref-progress-policy-001",
		RunID:          "run-ref-progress-policy-001",
	}

	got := supervisionEvidenceRefsV0(report)
	want := []string{
		"agent-progress-report-ref-provider-runtime-db-sql-home-model-001",
		"evidence-ref-runtime-provider-model-db-sql-home-001",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("refs opacas filtradas: got=%v want=%v", got, want)
	}
}

func TestAgentProgressSupervisorRailPolicyV0FiltraDetalleSensibleEfectivo(t *testing.T) {
	report := orquestaruntime.AgentProgressReportV0{
		ReportID: "agent-progress-report-ref-rail-policy-001",
		EvidenceRefs: []string{
			"evidence-ref-provider-model-db-sql-001",
			"api_key=valor",
			"/home/operador/proyecto",
			"prompt=raw",
			"raw_text=payload-completo",
		},
		AgentRequestID: "agent-request-ref-progress-policy-002",
		RunID:          "run-ref-progress-policy-002",
	}

	got := supervisionEvidenceRefsV0(report)
	want := []string{
		"agent-progress-report-ref-rail-policy-001",
		"evidence-ref-provider-model-db-sql-001",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("refs sensibles no filtradas: got=%v want=%v", got, want)
	}
}
