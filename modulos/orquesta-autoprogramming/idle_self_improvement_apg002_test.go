package orquestaautoprogramming

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestAutoprogrammingIdleSelfImprovementAPG002(t *testing.T) {
	work := BuildAutoprogrammingProgrammableWorkV1(validAutoprogrammingRequestV1(func(request *AutoprogrammingRequestV1) {
		request.RequestRef = "request-ref-autoprogramming-backlog-apg-002-97827de1"
		request.Tasks[0].TaskRef = "task-id-ref-backlog-10352ecd2077"
		request.Tasks[0].Area = "apg-002"
		request.Tasks[0].ContextRefs = append(request.Tasks[0].ContextRefs,
			"backlog_scan_ref:scan-ref-backlog-ef70c10fd351",
			"backlog_scan_epoch:backlog-scan-epoch-7e607dc35fdb",
			"backlog_scan_doc:modulos/orquesta-autoprogramming/docs/tareas.md:line:30:sha256:55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582",
			"backlog_task_id_ref:task-id-ref-backlog-10352ecd2077",
			"backlog_task_id_range:T263",
		)
		request.WorkProfiles = []AutoprogrammingWorkProfileV1{
			{
				AppKind:     "web_application",
				Area:        "apg-002",
				ProfileKind: orquestacoreworkflow.WorkProfileDocumentationV0,
			},
			{
				AppKind:     "web_application",
				TaskRef:     "task-id-ref-backlog-10352ecd2077",
				ProfileKind: orquestacoreworkflow.WorkProfileImplementationV0,
			},
		}
	}))
	if !work.Accepted {
		t.Fatalf("issues=%+v", work.Issues)
	}
	if got := work.Work.Base.Tasks[0].WorkProfileKind; got != orquestacoreworkflow.WorkProfileImplementationV0 {
		t.Fatalf("work_profile_kind=%s", got)
	}
	if source := work.Work.ProfileBindings[0].Source; source != "task_ref:task-id-ref-backlog-10352ecd2077" {
		t.Fatalf("profile_bindings=%+v", work.Work.ProfileBindings)
	}

	defaultConfig := ResolveAutoprogrammingIdleSelfImprovementConfigV0(nil)
	defaultDecision := DecideAutoprogrammingIdleSelfImprovementV0(
		AutoprogrammingIdleSelfImprovementDecisionInputV0{
			Config:         defaultConfig.Config,
			IdleForSeconds: 60,
		},
	)
	if defaultConfig.Config.AfterSeconds != 60 ||
		!defaultDecision.Prepare ||
		!defaultDecision.IdleTriggered {
		t.Fatalf("default_config=%+v default_decision=%+v", defaultConfig, defaultDecision)
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
		OrquestaServerIdleSelfImprovementAfterSecondsEnvV0: "600",
		OrquestaServerIdleSelfImprovementTargetQueueEnvV0:  "2",
	})
	queueDecision := DecideAutoprogrammingIdleSelfImprovementV0(
		AutoprogrammingIdleSelfImprovementDecisionInputV0{
			Config:       queueConfig.Config,
			QueueSize:    1,
			FreeCapacity: 1,
		},
	)
	if !queueDecision.Prepare || !queueDecision.QueueTriggered {
		t.Fatalf("queue_decision=%+v", queueDecision)
	}

	planner := PlanAutoprogrammingBacklogSelfImprovementV0(
		AutoprogrammingBacklogPlannerInputV0{
			Entries: []AutoprogrammingBacklogPlannerEntryV0{
				{
					TaskRef:    "task-id-ref-backlog-10352ecd2077",
					SectionRef: "apg-002",
					Area:       "apg-002",
					Title:      "Trabajo ya visible",
				},
				{
					SectionRef: "estado-actual",
					Title:      "Narrativa vigente",
					Narrative:  true,
				},
				{
					TaskRef:            "task-ref-apg002-new-gap",
					SectionRef:         "apg-002-new-gap",
					Area:               "apg-002",
					Title:              "Hueco concreto derivado de fallo observado",
					AcceptanceCriteria: []string{"usar evidencia del fallo y corregir la causa general si es posible"},
				},
			},
			VisibleQueue: []AutoprogrammingVisibleQueueItemV0{{
				SectionRef: "apg-002",
			}},
			CreateScanner:  true,
			ScannerTaskRef: "task-ref-backlog-scanner",
			BacklogScanRef: "scan-ref-backlog-ef70c10fd351",
			BacklogScan: AutoprogrammingBacklogScanV0{
				Epoch: "backlog-scan-epoch-7e607dc35fdb",
				ReservationRefs: []string{
					"reservation-ref-backlog-scan-doc-merge-ad53830da699",
					"reservation-ref-backlog-task-id-899db4424236",
				},
				Documents: []AutoprogrammingBacklogDocumentV0{{
					Path:       "modulos/orquesta-autoprogramming/docs/tareas.md",
					StartLine:  30,
					SHA256:     "55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582",
					SectionRef: "apg-002",
				}},
			},
		},
	)
	if len(planner.Tasks) != 1 || planner.Tasks[0].TaskRef != "task-ref-apg002-new-gap" {
		t.Fatalf("planner_tasks=%+v", planner.Tasks)
	}
	assertAutoprogrammingSkipV0(t, planner.Skipped, "task-id-ref-backlog-10352ecd2077", AutoprogrammingBacklogPlannerSkipVisibleInQueueV0)
	if !hasAutoprogrammingPlannerSkipV0(planner.Skipped, "", AutoprogrammingBacklogPlannerSkipNarrativeSectionV0) {
		t.Fatalf("planner_skipped=%+v", planner.Skipped)
	}
	if planner.Scanner == nil || planner.Scanner.Title != "Escaneo backlog nuevos" {
		t.Fatalf("scanner=%+v", planner.Scanner)
	}
	for _, want := range []string{
		"backlog_scan_ref:scan-ref-backlog-ef70c10fd351",
		"backlog_scan_epoch:backlog-scan-epoch-7e607dc35fdb",
		"backlog_scan_reservation_ref:reservation-ref-backlog-scan-doc-merge-ad53830da699",
		"backlog_scan_reservation_ref:reservation-ref-backlog-task-id-899db4424236",
		"backlog_scan_doc:modulos/orquesta-autoprogramming/docs/tareas.md:line:30:sha256:55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582",
	} {
		assertAutoprogrammingScannerContextRefV0(t, planner.Scanner.ContextRefs, want)
	}

	assertAutoprogrammingExternalProjectionV0(t,
		ProjectAutoprogrammingExternalWorkV0(AutoprogrammingExternalWorkProjectionInputV0{
			OutboxPendingRefs: []string{"outbox-ref-apg002"},
		}),
		AutoprogrammingExternalProjectionOutboxPendingV0,
	)
	assertAutoprogrammingExternalProjectionV0(t,
		ProjectAutoprogrammingExternalWorkV0(AutoprogrammingExternalWorkProjectionInputV0{
			WaitExternalRefs: []string{"wait-external-ref-apg002"},
		}),
		AutoprogrammingExternalProjectionWaitExternalV0,
	)
	assertAutoprogrammingExternalProjectionV0(t,
		ProjectAutoprogrammingExternalWorkV0(AutoprogrammingExternalWorkProjectionInputV0{
			ExternalProcessRefs:     []string{"process-ref-apg002"},
			ExternalProcessVerified: true,
		}),
		AutoprogrammingExternalProjectionExternalProcessVerifiedV0,
	)
}
