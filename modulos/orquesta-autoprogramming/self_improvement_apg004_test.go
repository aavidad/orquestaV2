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
			"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS por defecto dispara tras 60 segundos sin ejecuciones",
			"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0 desactiva automejora idle",
			"el servidor prepara automejora cuando hay idle o capacidad libre por debajo de ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE",
			"el planner salta tareas ya visibles en cola y puede crear una tarea scanner para descubrir nuevos huecos",
			"las secciones narrativas del backlog se filtran como contexto y no como pendientes ejecutables",
			"la proyeccion publica distingue outbox pendiente, wait_external y proceso externo verificado",
		},
		ContextRefs: []string{
			"backlog_scan_ref:scan-ref-backlog-2e2bfd30581c",
			"backlog_task_id_ref:task-id-ref-backlog-5279219cc308",
			"backlog_task_id_range:T263",
			"source_task_ref:task-ref-self-improvement-b8946563e08f",
		},
		EvidenceRefs: []string{"evidence-ref-apg004-t208-reconciliado"},
		BacklogScan: AutoprogrammingBacklogScanV0{
			Epoch: "backlog-scan-epoch-7d58c5b5552a",
			ReservationRefs: []string{
				"reservation-ref-backlog-scan-doc-merge-540fce675096",
				"reservation-ref-backlog-task-id-02081ab71912",
			},
			Documents: []AutoprogrammingBacklogDocumentV0{{
				Path:       "modulos/orquesta-autoprogramming/docs/tareas.md",
				StartLine:  38,
				SHA256:     "55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582",
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
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "backlog_scan_epoch:backlog-scan-epoch-7d58c5b5552a")
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "backlog_scan_ref:scan-ref-backlog-2e2bfd30581c")
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "backlog_scan_reservation_ref:reservation-ref-backlog-scan-doc-merge-540fce675096")
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "backlog_scan_reservation_ref:reservation-ref-backlog-task-id-02081ab71912")
	assertAutoprogrammingContextRefPrefixV0(t, task.ContextRefs, "context_ref:backlog_scan_doc-")
	if !stringsSliceContainsForAutoprogrammingTestV0(task.RequiredTests, "go test -count=1 ./modulos/orquesta-autoprogramming") {
		t.Fatalf("required_tests=%v", task.RequiredTests)
	}

	planner := PlanAutoprogrammingBacklogSelfImprovementV0(AutoprogrammingBacklogPlannerInputV0{
		Entries: []AutoprogrammingBacklogPlannerEntryV0{{
			TaskRef:    "task-ref-apg004-visible",
			SectionRef: "apg-004",
			Area:       "apg-004",
			Title:      "APG-004 visible en cola",
		}},
		VisibleQueue: []AutoprogrammingVisibleQueueItemV0{{
			SectionRef: "apg-004",
		}},
		CreateScanner:  true,
		ScannerTaskRef: "task-ref-backlog-scanner",
		BacklogScanRef: "scan-ref-backlog-2e2bfd30581c",
		BacklogScan:    proposal.BacklogScan,
	})
	if len(planner.Tasks) != 0 ||
		!hasAutoprogrammingPlannerSkipV0(planner.Skipped, "task-ref-apg004-visible", AutoprogrammingBacklogPlannerSkipVisibleInQueueV0) {
		t.Fatalf("planner=%+v", planner)
	}
	if planner.Scanner == nil || planner.Scanner.Title != "Escaneo backlog nuevos" {
		t.Fatalf("scanner=%+v", planner.Scanner)
	}
	assertAutoprogrammingScannerContextRefV0(t, planner.Scanner.ContextRefs, "backlog_scan_ref:scan-ref-backlog-2e2bfd30581c")
	assertAutoprogrammingScannerContextRefV0(t, planner.Scanner.ContextRefs, "backlog_scan_epoch:backlog-scan-epoch-7d58c5b5552a")
	assertAutoprogrammingScannerContextRefV0(t, planner.Scanner.ContextRefs, "backlog_scan_reservation_ref:reservation-ref-backlog-scan-doc-merge-540fce675096")
	assertAutoprogrammingScannerContextRefV0(t, planner.Scanner.ContextRefs, "backlog_scan_reservation_ref:reservation-ref-backlog-task-id-02081ab71912")
	assertAutoprogrammingScannerContextRefV0(t, planner.Scanner.ContextRefs, "backlog_scan_doc:modulos/orquesta-autoprogramming/docs/tareas.md:line:38:sha256:55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582")
}
