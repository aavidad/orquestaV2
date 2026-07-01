package orquestaautoprogramming

import "testing"

func TestAutoprogrammingIdleSelfImprovementAPG001ContratoV0(t *testing.T) {
	defaultConfig := ResolveAutoprogrammingIdleSelfImprovementConfigV0(nil)
	if len(defaultConfig.Issues) != 0 ||
		defaultConfig.Config.AfterSeconds != AutoprogrammingIdleSelfImprovementDefaultAfterSecondsV0 {
		t.Fatalf("default_config=%+v", defaultConfig)
	}
	defaultDecision := DecideAutoprogrammingIdleSelfImprovementV0(
		AutoprogrammingIdleSelfImprovementDecisionInputV0{
			Config:         defaultConfig.Config,
			IdleForSeconds: 60,
			QueueSize:      0,
			FreeCapacity:   0,
		},
	)
	if !defaultDecision.Prepare || !defaultDecision.IdleTriggered {
		t.Fatalf("default_decision=%+v", defaultDecision)
	}

	disabledConfig := ResolveAutoprogrammingIdleSelfImprovementConfigV0(map[string]string{
		OrquestaServerIdleSelfImprovementAfterSecondsEnvV0: "0",
	})
	disabledDecision := DecideAutoprogrammingIdleSelfImprovementV0(
		AutoprogrammingIdleSelfImprovementDecisionInputV0{
			Config:         disabledConfig.Config,
			IdleForSeconds: 600,
			QueueSize:      0,
			FreeCapacity:   10,
		},
	)
	if disabledDecision.Prepare || !disabledDecision.Disabled {
		t.Fatalf("disabled_decision=%+v", disabledDecision)
	}

	queueConfig := ResolveAutoprogrammingIdleSelfImprovementConfigV0(map[string]string{
		OrquestaServerIdleSelfImprovementAfterSecondsEnvV0: "600",
		OrquestaServerIdleSelfImprovementTargetQueueEnvV0:  "2",
	})
	queueDecision := DecideAutoprogrammingIdleSelfImprovementV0(
		AutoprogrammingIdleSelfImprovementDecisionInputV0{
			Config:         queueConfig.Config,
			IdleForSeconds: 1,
			QueueSize:      1,
			FreeCapacity:   1,
		},
	)
	if !queueDecision.Prepare || !queueDecision.QueueTriggered {
		t.Fatalf("queue_decision=%+v", queueDecision)
	}

	planner := PlanAutoprogrammingBacklogSelfImprovementV0(
		AutoprogrammingBacklogPlannerInputV0{
			Entries: []AutoprogrammingBacklogPlannerEntryV0{
				{
					TaskRef:    "task-ref-apg001-visible",
					SectionRef: "apg-001",
					Area:       "apg-001",
					Title:      "Trabajo ya visible en cola",
				},
				{
					SectionRef: "estado-actual",
					Title:      "Narrativa vigente",
					Narrative:  true,
				},
				{
					TaskRef:    "task-ref-apg001-new-gap",
					SectionRef: "apg-001-new-gap",
					Area:       "apg-001",
					Title:      "Hueco concreto derivado de fallo observado",
					AcceptanceCriteria: []string{
						"usar evidencia del fallo y corregir la causa general si es posible",
					},
				},
			},
			VisibleQueue: []AutoprogrammingVisibleQueueItemV0{{
				SectionRef: "apg-001",
			}},
			CreateScanner:  true,
			ScannerTaskRef: "task-ref-backlog-scanner",
			BacklogScanRef: "scan-ref-backlog-72c73a98bd54",
			BacklogScan: AutoprogrammingBacklogScanV0{
				Epoch: "backlog-scan-epoch-feee914e7ce2",
				ReservationRefs: []string{
					"reservation-ref-backlog-scan-doc-merge-9b9f5f8061bd",
					"reservation-ref-backlog-task-id-4f2d7716ca0e",
				},
				Documents: []AutoprogrammingBacklogDocumentV0{{
					Path:       "modulos/orquesta-autoprogramming/docs/tareas.md",
					StartLine:  27,
					SHA256:     "55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582",
					SectionRef: "apg-001",
				}},
			},
		},
	)
	if len(planner.Tasks) != 1 || planner.Tasks[0].TaskRef != "task-ref-apg001-new-gap" {
		t.Fatalf("planner_tasks=%+v", planner.Tasks)
	}
	if !hasAutoprogrammingPlannerSkipV0(planner.Skipped, "task-ref-apg001-visible", AutoprogrammingBacklogPlannerSkipVisibleInQueueV0) ||
		!hasAutoprogrammingPlannerSkipV0(planner.Skipped, "", AutoprogrammingBacklogPlannerSkipNarrativeSectionV0) {
		t.Fatalf("planner_skipped=%+v", planner.Skipped)
	}
	if planner.Scanner == nil {
		t.Fatalf("scanner=nil")
	}
	assertAutoprogrammingScannerContextRefV0(t, planner.Scanner.ContextRefs, "backlog_scan_ref:scan-ref-backlog-72c73a98bd54")
	assertAutoprogrammingScannerContextRefV0(t, planner.Scanner.ContextRefs, "backlog_scan_epoch:backlog-scan-epoch-feee914e7ce2")
	assertAutoprogrammingScannerContextRefV0(t, planner.Scanner.ContextRefs, "backlog_scan_reservation_ref:reservation-ref-backlog-scan-doc-merge-9b9f5f8061bd")
	assertAutoprogrammingScannerContextRefV0(t, planner.Scanner.ContextRefs, "backlog_scan_reservation_ref:reservation-ref-backlog-task-id-4f2d7716ca0e")
	assertAutoprogrammingScannerContextRefV0(t, planner.Scanner.ContextRefs, "backlog_scan_doc:modulos/orquesta-autoprogramming/docs/tareas.md:line:27:sha256:55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582")

	assertAutoprogrammingExternalProjectionV0(t,
		ProjectAutoprogrammingExternalWorkV0(AutoprogrammingExternalWorkProjectionInputV0{
			OutboxPendingRefs: []string{"outbox-ref-apg001"},
		}),
		AutoprogrammingExternalProjectionOutboxPendingV0,
	)
	assertAutoprogrammingExternalProjectionV0(t,
		ProjectAutoprogrammingExternalWorkV0(AutoprogrammingExternalWorkProjectionInputV0{
			OutboxPendingRefs: []string{"outbox-ref-apg001"},
			WaitExternalRefs:  []string{"wait-external-ref-apg001"},
		}),
		AutoprogrammingExternalProjectionWaitExternalV0,
	)
	assertAutoprogrammingExternalProjectionV0(t,
		ProjectAutoprogrammingExternalWorkV0(AutoprogrammingExternalWorkProjectionInputV0{
			OutboxPendingRefs:       []string{"outbox-ref-apg001"},
			WaitExternalRefs:        []string{"wait-external-ref-apg001"},
			ExternalProcessRefs:     []string{"process-ref-apg001"},
			ExternalProcessVerified: true,
		}),
		AutoprogrammingExternalProjectionExternalProcessVerifiedV0,
	)
}

func assertAutoprogrammingScannerContextRefV0(t *testing.T, refs []string, want string) {
	t.Helper()
	if !stringsSliceContainsForAutoprogrammingTestV0(refs, want) {
		t.Fatalf("scanner_refs=%v missing=%s", refs, want)
	}
}
