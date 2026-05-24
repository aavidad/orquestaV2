package orquestaautoprogramming

import "testing"

func TestBuildAutoprogrammingSelfImprovementRequestV0CreaTrabajoDeBajaPrioridad(t *testing.T) {
	result := BuildAutoprogrammingSelfImprovementRequestV0(validSelfImprovementProposalForTestV0())

	if !result.Accepted ||
		!result.Background ||
		result.PriorityScore != AutoprogrammingSelfImprovementDefaultPriorityScoreV0 ||
		result.Request.RequestRef == "" ||
		len(result.Request.Tasks) != 1 ||
		result.Request.Tasks[0].Area != "mcp" ||
		len(result.Request.WriteSet) != 1 ||
		len(result.Request.RequiredTests) != 1 {
		t.Fatalf("result=%+v", result)
	}
	if result.Request.WorktreeRef != "worktree-ref-self-improvement-001" ||
		!result.Request.WorktreeIsolated ||
		result.Request.BranchRef != "branch-ref-self-improvement-001" {
		t.Fatalf("request=%+v", result.Request)
	}
	if !stringSliceContainsSelfImprovementTestV0(result.NextActions, "prepare_run_with_low_priority") {
		t.Fatalf("next_actions=%v", result.NextActions)
	}
	if stringSliceContainsSelfImprovementTestV0(result.Request.Tasks[0].Context, "source_run_ref:") {
		t.Fatalf("contexto contiene ref vacia: %v", result.Request.Tasks[0].Context)
	}
	validation := ValidateAutoprogrammingRequestV0(result.Request)
	if !validation.Accepted {
		t.Fatalf("request invalida: %+v", validation.Issues)
	}
}

func TestBuildAutoprogrammingSelfImprovementRequestV0NoTiraTrabajoSiFaltanRefs(t *testing.T) {
	proposal := validSelfImprovementProposalForTestV0()
	proposal.WorktreeRef = " "
	proposal.BranchRef = " "
	proposal.WorktreeIsolated = false

	result := BuildAutoprogrammingSelfImprovementRequestV0(proposal)

	if result.Accepted || !result.Background || result.Request.RequestRef != "" {
		t.Fatalf("result=%+v", result)
	}
	if !stringSliceContainsSelfImprovementTestV0(result.NextActions, "preserve_failure_evidence") ||
		!stringSliceContainsSelfImprovementTestV0(result.NextActions, "complete_isolated_worktree_before_prepare_run") {
		t.Fatalf("next_actions=%v", result.NextActions)
	}
	if len(result.Issues) < 3 {
		t.Fatalf("issues=%+v", result.Issues)
	}
}

func TestBuildAutoprogrammingSelfImprovementRequestV0RespetaPrioridadExplicita(t *testing.T) {
	proposal := validSelfImprovementProposalForTestV0()
	proposal.PriorityScore = 7

	result := BuildAutoprogrammingSelfImprovementRequestV0(proposal)

	if !result.Accepted || result.PriorityScore != 7 {
		t.Fatalf("result=%+v", result)
	}
}

func TestBuildAutoprogrammingSelfImprovementRequestV0TransportaBacklogScanV0(t *testing.T) {
	proposal := validSelfImprovementProposalForTestV0()
	proposal.BacklogScan = AutoprogrammingBacklogScanV0{
		Epoch:           "backlog-scan-epoch-001",
		ReservationRefs: []string{"reservation-ref-backlog-scan-doc-merge-001"},
		Documents: []AutoprogrammingBacklogDocumentV0{{
			Path:      "docs/autoprogramacion_orquesta_pendientes_2026-05-23.md",
			StartLine: 10,
			SHA256:    "abc123",
		}},
	}

	result := BuildAutoprogrammingSelfImprovementRequestV0(proposal)

	if !result.Accepted || result.Request.BacklogScan.Epoch != proposal.BacklogScan.Epoch {
		t.Fatalf("result=%+v", result)
	}
	work := BuildAutoprogrammingProgrammableWorkV0(result.Request)
	if !work.Accepted ||
		!stringsSliceContainsForAutoprogrammingTestV0(work.Work.Tasks[0].ContextRefs, "backlog_scan_epoch:backlog-scan-epoch-001") {
		t.Fatalf("work=%+v", work)
	}
}

func validSelfImprovementProposalForTestV0() AutoprogrammingSelfImprovementProposalV0 {
	return AutoprogrammingSelfImprovementProposalV0{
		ProjectRef:         "project-ref-orquesta",
		WorktreeRef:        "worktree-ref-self-improvement-001",
		WorktreeIsolated:   true,
		BranchRef:          "branch-ref-self-improvement-001",
		ObservedBy:         "director",
		SourceRunRef:       "run-ref-primary-001",
		SourceTaskRef:      "task-ref-primary-001",
		FailureKind:        "api",
		FailureSummary:     "El endpoint ocultaba errores publicos reparables.",
		SuggestedArea:      "MCP",
		SuggestedWriteSet:  []string{"modulos/orquesta-mcp/arrancar_director_app_http_v0.go"},
		RequiredTests:      []string{"go test -count=1 ./modulos/orquesta-mcp"},
		AcceptanceCriteria: []string{"el fallo queda visible como error publico reparable"},
		CompactRules:       []string{"no bloquear el trabajo principal"},
		ContextRefs:        []string{"doc-ref:principio-orquesta-piensa-director"},
		EvidenceRefs:       []string{"evidence-ref-director-error-hidden"},
	}
}

func stringSliceContainsSelfImprovementTestV0(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
