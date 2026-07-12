package orquestamcp

import (
	"fmt"
	"testing"
)

func TestProjectMCPAutoprogrammingStatusListsV0AgrupaDiagnosticosRepetidos(t *testing.T) {
	diagnostics := make([]MCPAutoprogrammingDiagnosticV0, 0, 70)
	for index := 0; index < 68; index++ {
		diagnostics = append(diagnostics, MCPAutoprogrammingDiagnosticV0{
			Code:         "estado_vivo_desconocido",
			Scope:        fmt.Sprintf("run:run-ref-%03d", index),
			Message:      "estado vivo desconocido",
			EvidenceRefs: []string{fmt.Sprintf("evidence-ref-%03d", index)},
		})
	}
	diagnostics = append(diagnostics,
		MCPAutoprogrammingDiagnosticV0{Code: "queue_live", Scope: "queue"},
		MCPAutoprogrammingDiagnosticV0{Code: "goal_first_blocked", Scope: "run:block"},
	)

	result := projectMCPAutoprogrammingStatusListsV0(MCPAutoprogrammingStatusToolResultV0{
		Diagnostics: diagnostics,
	})
	if result.DiagnosticsTotal != 70 || len(result.Diagnostics) != 3 {
		t.Fatalf("total=%d diagnostics=%+v", result.DiagnosticsTotal, result.Diagnostics)
	}
	aggregate := result.Diagnostics[0]
	if aggregate.Code != "estado_vivo_desconocido" || aggregate.Count != 68 ||
		aggregate.Scope != "aggregate:estado_vivo_desconocido" ||
		len(aggregate.SampleRefs) != mcpAutoprogrammingStatusSampleRefsLimitV0 ||
		len(aggregate.EvidenceRefs) != mcpAutoprogrammingStatusSampleRefsLimitV0 {
		t.Fatalf("aggregate=%+v", aggregate)
	}
}

func TestProjectMCPAutoprogrammingStatusListsV0AgrupaYAcotaStaleConTotalReal(t *testing.T) {
	stale := make([]MCPAutoprogrammingActionableRunV0, 0, 59)
	for index := 0; index < 34; index++ {
		stale = append(stale, MCPAutoprogrammingActionableRunV0{
			Code:         "estado_vivo_desconocido",
			RunRef:       fmt.Sprintf("run-ref-unknown-%03d", index),
			Reason:       "estado vivo desconocido",
			EvidenceRefs: []string{fmt.Sprintf("evidence-ref-unknown-%03d", index)},
		})
	}
	for index := 0; index < 25; index++ {
		stale = append(stale, MCPAutoprogrammingActionableRunV0{
			Code:   fmt.Sprintf("unique-code-%03d", index),
			RunRef: fmt.Sprintf("run-ref-unique-%03d", index),
		})
	}

	result := projectMCPAutoprogrammingStatusListsV0(MCPAutoprogrammingStatusToolResultV0{
		StaleRunning: stale,
	})
	if result.StaleRunningTotal != 59 || len(result.StaleRunning) != mcpAutoprogrammingStatusDefaultListLimitV0 {
		t.Fatalf("total=%d returned=%d", result.StaleRunningTotal, len(result.StaleRunning))
	}
	aggregate := result.StaleRunning[0]
	if aggregate.Code != "estado_vivo_desconocido" || aggregate.Count != 34 ||
		aggregate.RunRef != "" || len(aggregate.SampleRefs) != mcpAutoprogrammingStatusSampleRefsLimitV0 ||
		len(aggregate.EvidenceRefs) != mcpAutoprogrammingStatusSampleRefsLimitV0 {
		t.Fatalf("aggregate=%+v", aggregate)
	}
}

func TestMCPAutoprogrammingQueueInputV0UsaLimiteSeguroSoloPorDefecto(t *testing.T) {
	if got := mcpAutoprogrammingQueueInputV0(MCPAutoprogrammingStatusToolInputV0{}).Limit; got != mcpAutoprogrammingStatusDefaultListLimitV0 {
		t.Fatalf("default limit=%d", got)
	}
	if got := mcpAutoprogrammingQueueInputV0(MCPAutoprogrammingStatusToolInputV0{QueueLimit: 7}).Limit; got != 7 {
		t.Fatalf("explicit limit=%d", got)
	}
}
