package orquestaautoprogramming

import "testing"

func TestBuildAutoprogrammingSelfImprovementRequestV0ReconciliacionAPG004T208(t *testing.T) {
	proposal := AutoprogrammingSelfImprovementProposalV0{
		ProjectRef:        "project-ref-orquesta",
		WorktreeRef:       "worktree-ref-orquesta-server-idle-self-improvement",
		WorktreeIsolated:  true,
		BranchRef:         "branch-ref-orquesta-server-idle-self-improvement",
		ObservedBy:        "guardian-autoprogramming",
		SourceTaskRef:     "task-ref-self-improvement-b8946563e08f",
		FailureKind:       "documentacion",
		FailureSummary:    "backlog pendiente APG-004 reconciliacion documental T208, 2026-05-27",
		SuggestedArea:     "autoprogramming",
		SuggestedWriteSet: []string{"modulos/orquesta-autoprogramming"},
		RequiredTests:     []string{"go test -count=1 ./modulos/orquesta-autoprogramming"},
		AcceptanceCriteria: []string{
			"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0 desactiva automejora idle",
			"el planner salta tareas ya visibles en cola y puede crear una tarea scanner para descubrir nuevos huecos",
			"las secciones narrativas del backlog se filtran como contexto y no como pendientes ejecutables",
		},
		ContextRefs: []string{
			"backlog_scan_ref:scan-ref-backlog-2dce8e490e64",
			"source_task_ref:task-ref-self-improvement-b8946563e08f",
		},
		EvidenceRefs: []string{"evidence-ref-apg004-t208-reconciliado"},
		BacklogScan: AutoprogrammingBacklogScanV0{
			Epoch:           "backlog-scan-epoch-2d6bb44d338a",
			ReservationRefs: []string{"reservation-ref-apg004-t208"},
			Documents: []AutoprogrammingBacklogDocumentV0{{
				Path:       "modulos/orquesta-autoprogramming/docs/tareas.md",
				StartLine:  32,
				SHA256:     "sha256-ref-apg004-t208",
				SectionRef: "apg-004",
			}},
		},
	}

	result := BuildAutoprogrammingSelfImprovementRequestV0(proposal)

	if !result.Accepted || !result.Background {
		t.Fatalf("result=%+v", result)
	}
	request := result.Request
	if request.WorktreeRef != proposal.WorktreeRef || request.BranchRef != proposal.BranchRef {
		t.Fatalf("refs opacas no preservadas: %+v", request)
	}
	if len(request.WriteSet) != 1 || request.WriteSet[0] != "modulos/orquesta-autoprogramming" {
		t.Fatalf("write_set=%v", request.WriteSet)
	}

	work := BuildAutoprogrammingProgrammableWorkV0(request)
	if !work.Accepted || len(work.Work.Tasks) != 1 {
		t.Fatalf("work=%+v", work)
	}
	task := work.Work.Tasks[0]
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "source_task_ref:task-ref-self-improvement-b8946563e08f")
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "backlog_scan_epoch:backlog-scan-epoch-2d6bb44d338a")
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "backlog_scan_ref:scan-ref-backlog-2dce8e490e64")
	if !stringsSliceContainsForAutoprogrammingTestV0(task.RequiredTests, "go test -count=1 ./modulos/orquesta-autoprogramming") {
		t.Fatalf("required_tests=%v", task.RequiredTests)
	}
}
