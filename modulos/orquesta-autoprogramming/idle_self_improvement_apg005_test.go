package orquestaautoprogramming

import "testing"

func TestAutoprogrammingIdleSelfImprovementAPG005(t *testing.T) {
	defaultConfig := ResolveAutoprogrammingIdleSelfImprovementConfigV0(nil)
	if len(defaultConfig.Issues) != 0 ||
		defaultConfig.Config.AfterSeconds != AutoprogrammingIdleSelfImprovementDefaultAfterSecondsV0 {
		t.Fatalf("default_config=%+v", defaultConfig)
	}
	defaultDecision := DecideAutoprogrammingIdleSelfImprovementV0(
		AutoprogrammingIdleSelfImprovementDecisionInputV0{
			Config:         defaultConfig.Config,
			IdleForSeconds: 60,
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
			FreeCapacity:   6,
		},
	)
	if disabledDecision.Prepare || !disabledDecision.Disabled {
		t.Fatalf("disabled_decision=%+v", disabledDecision)
	}

	queueConfig := ResolveAutoprogrammingIdleSelfImprovementConfigV0(map[string]string{
		OrquestaServerIdleSelfImprovementTargetQueueEnvV0: "3",
	})
	queueDecision := DecideAutoprogrammingIdleSelfImprovementV0(
		AutoprogrammingIdleSelfImprovementDecisionInputV0{
			Config:       queueConfig.Config,
			QueueSize:    1,
			FreeCapacity: 2,
		},
	)
	if !queueDecision.Prepare || !queueDecision.QueueTriggered {
		t.Fatalf("queue_decision=%+v", queueDecision)
	}

	planner := PlanAutoprogrammingBacklogSelfImprovementV0(
		AutoprogrammingBacklogPlannerInputV0{
			Entries: []AutoprogrammingBacklogPlannerEntryV0{
				{
					TaskRef:    "task-ref-backlog-visible",
					SectionRef: "apg-visible",
					Area:       "apg-005",
					Title:      "Tarea ya visible",
				},
				{
					SectionRef: "estado-actual",
					Title:      "Estado actual",
					Narrative:  true,
				},
				{
					TaskRef:            "task-ref-backlog-new",
					SectionRef:         "apg-new",
					Area:               "apg-005",
					Title:              "Tarea nueva",
					AcceptanceCriteria: []string{"usar evidencia del fallo y corregir la causa general si es posible"},
				},
			},
			VisibleQueue: []AutoprogrammingVisibleQueueItemV0{{
				TaskRef: "task-ref-backlog-visible",
			}},
			CreateScanner: true,
			BacklogScan: AutoprogrammingBacklogScanV0{
				Epoch:           "backlog-scan-epoch-8dc06f56b7fc",
				ReservationRefs: []string{"reservation-ref-backlog-scan-doc-merge-9e9c403aac4f"},
				Documents: []AutoprogrammingBacklogDocumentV0{{
					Path:       "modulos/orquesta-autoprogramming/docs/tareas.md",
					StartLine:  42,
					SHA256:     "55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582",
					SectionRef: "apg-005",
				}},
			},
		},
	)
	if len(planner.Tasks) != 1 || planner.Tasks[0].TaskRef != "task-ref-backlog-new" {
		t.Fatalf("planner_tasks=%+v", planner.Tasks)
	}
	if !hasAutoprogrammingPlannerSkipV0(planner.Skipped, "task-ref-backlog-visible", AutoprogrammingBacklogPlannerSkipVisibleInQueueV0) ||
		!hasAutoprogrammingPlannerSkipV0(planner.Skipped, "", AutoprogrammingBacklogPlannerSkipNarrativeSectionV0) {
		t.Fatalf("planner_skipped=%+v", planner.Skipped)
	}
	if planner.Scanner == nil ||
		!stringsSliceContainsForAutoprogrammingTestV0(planner.Scanner.ContextRefs, "backlog_scan_epoch:backlog-scan-epoch-8dc06f56b7fc") ||
		!stringsSliceContainsForAutoprogrammingTestV0(planner.Scanner.ContextRefs, "backlog_scan_doc:modulos/orquesta-autoprogramming/docs/tareas.md:line:42:sha256:55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582") {
		t.Fatalf("scanner=%+v", planner.Scanner)
	}

	assertAutoprogrammingExternalProjectionV0(t,
		ProjectAutoprogrammingExternalWorkV0(AutoprogrammingExternalWorkProjectionInputV0{
			OutboxPendingRefs: []string{"outbox-ref-001"},
		}),
		AutoprogrammingExternalProjectionOutboxPendingV0,
	)
	assertAutoprogrammingExternalProjectionV0(t,
		ProjectAutoprogrammingExternalWorkV0(AutoprogrammingExternalWorkProjectionInputV0{
			WaitExternalRefs: []string{"wait-ref-001"},
		}),
		AutoprogrammingExternalProjectionWaitExternalV0,
	)
	assertAutoprogrammingExternalProjectionV0(t,
		ProjectAutoprogrammingExternalWorkV0(AutoprogrammingExternalWorkProjectionInputV0{
			ExternalProcessRefs:     []string{"process-ref-001"},
			ExternalProcessVerified: true,
		}),
		AutoprogrammingExternalProjectionExternalProcessVerifiedV0,
	)
}

func hasAutoprogrammingPlannerSkipV0(
	values []AutoprogrammingBacklogPlannerSkippedV0,
	taskRef string,
	reason string,
) bool {
	for _, value := range values {
		if value.TaskRef == taskRef && value.Reason == reason {
			return true
		}
	}
	return false
}

func assertAutoprogrammingExternalProjectionV0(
	t *testing.T,
	projection AutoprogrammingExternalWorkProjectionV0,
	state string,
) {
	t.Helper()
	if projection.State != state {
		t.Fatalf("projection=%+v want_state=%s", projection, state)
	}
}
