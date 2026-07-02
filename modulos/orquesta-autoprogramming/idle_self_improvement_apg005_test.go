package orquestaautoprogramming

import (
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

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
			QueueSize:      1,
			FreeCapacity:   0,
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
			CreateScanner:  true,
			BacklogScanRef: "scan-ref-backlog-368a95054517",
			BacklogScan: AutoprogrammingBacklogScanV0{
				Epoch: "backlog-scan-epoch-8dc06f56b7fc",
				ReservationRefs: []string{
					"reservation-ref-backlog-scan-doc-merge-9e9c403aac4f",
					"reservation-ref-backlog-task-id-911abe614485",
				},
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
		!stringsSliceContainsForAutoprogrammingTestV0(planner.Scanner.ContextRefs, "backlog_scan_ref:scan-ref-backlog-368a95054517") ||
		!stringsSliceContainsForAutoprogrammingTestV0(planner.Scanner.ContextRefs, "backlog_scan_epoch:backlog-scan-epoch-8dc06f56b7fc") ||
		!stringsSliceContainsForAutoprogrammingTestV0(planner.Scanner.ContextRefs, "backlog_scan_reservation_ref:reservation-ref-backlog-scan-doc-merge-9e9c403aac4f") ||
		!stringsSliceContainsForAutoprogrammingTestV0(planner.Scanner.ContextRefs, "backlog_scan_reservation_ref:reservation-ref-backlog-task-id-911abe614485") ||
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

func TestAutoprogrammingGoalFirstSpecsAPG005(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.RequestRef = "request-ref-autoprogramming-backlog-apg-005-0a062281"
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef:   "task-id-ref-backlog-48cf16dc7592",
			Area:      "apg-005",
			Title:     "APG-005 clasificacion y specs goal-first",
			Objective: "Compilar GoalWorkSpecV0 neutral para autoprogramacion goal-first.",
			ContextRefs: []string{
				"goal_migration:goal-first",
				"goal_capability:starter",
				"goal_capability:observer",
				"goal_capability:closure-validator",
				"backlog_scan_ref:scan-ref-backlog-368a95054517",
				"backlog_scan_epoch:backlog-scan-epoch-8dc06f56b7fc",
				"backlog_scan_doc:modulos/orquesta-autoprogramming/docs/tareas.md:line:42:sha256:55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582",
				"backlog_task_id_range:T263",
			},
			AcceptanceCriteria: []string{
				"legacy_loop_compatible, goal_ready, blocked_by_goal_capability, covered_by_goal_first y legacy_loop_required distinguibles",
				"goal_ready compila GoalWorkSpecV0 con write-set, tests y refs opacas",
			},
		}}
		request.WriteSet = []string{"modulos/orquesta-autoprogramming"}
		request.RequiredTests = []string{"go test -count=1 ./modulos/orquesta-autoprogramming"}
		request.BacklogScan = AutoprogrammingBacklogScanV0{
			Epoch: "backlog-scan-epoch-8dc06f56b7fc",
			ReservationRefs: []string{
				"reservation-ref-backlog-scan-doc-merge-9e9c403aac4f",
				"reservation-ref-backlog-task-id-911abe614485",
			},
			Documents: []AutoprogrammingBacklogDocumentV0{{
				Path:       "modulos/orquesta-autoprogramming/docs/tareas.md",
				StartLine:  42,
				SHA256:     "55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582",
				SectionRef: "apg-005",
			}},
		}
	}))

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if result.Work.GoalMigration.Status != AutoprogrammingGoalMigrationGoalReadyV0 ||
		result.Work.GoalMigration.RecommendedAction != AutoprogrammingGoalMigrationActionLaunchGoalV0 ||
		len(result.Work.GoalSpecs) != 1 ||
		len(result.Work.Tasks) != 0 {
		t.Fatalf("work=%+v", result.Work)
	}
	spec := result.Work.GoalSpecs[0]
	if spec.DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 ||
		spec.WorkKind != AutoprogrammingGoalWorkKindV0 ||
		len(spec.WriteSet) != 1 ||
		spec.WriteSet[0].Path != "modulos/orquesta-autoprogramming" ||
		len(spec.RequiredTests) != 1 ||
		spec.RequiredTests[0].Command != "go test -count=1 ./modulos/orquesta-autoprogramming" ||
		!spec.ClosurePolicy.RequireRequiredTests {
		t.Fatalf("spec=%+v", spec)
	}
	if !hasGoalContextRefForAutoprogrammingTestV0(
		spec.ContextRefs,
		"workflow_task_context",
		"backlog_scan_ref:scan-ref-backlog-368a95054517",
	) {
		t.Fatalf("context_refs=%+v", spec.ContextRefs)
	}
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
