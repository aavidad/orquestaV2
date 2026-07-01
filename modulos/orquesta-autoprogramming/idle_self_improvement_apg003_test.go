package orquestaautoprogramming

import "testing"

func TestAutoprogrammingIdleSelfImprovementAPG003ContratoPuroYEvidencia(t *testing.T) {
	source := validAutoprogrammingRequestSourceV0(func(source *AutoprogrammingRequestSourceV0) {
		source.SourceRef = "source-ref-apg003-web"
		source.SourceSurface = "web"
		source.Transport = "http"
		source.RequestID = "request-ref-autoprogramming-backlog-apg-003-dd90d015"
		source.CorrelationID = "scan-ref-backlog-58c1f26e2898"
		source.RequestedBy = "operator-ref-autoprogramming"
		source.PriorityScore = 3
		for i := range source.AutoprogrammingRequest.Tasks {
			source.AutoprogrammingRequest.Tasks[i].ContextRefs = append(
				source.AutoprogrammingRequest.Tasks[i].ContextRefs[:0],
				AutoprogrammingRequestSourceRefsV0(*source)...,
			)
		}
	})
	sourceValidation := ValidateAutoprogrammingRequestSourceV0(source)
	if !sourceValidation.Accepted || !sourceValidation.RequestValidation.Accepted {
		t.Fatalf("source_validation=%+v request=%+v", sourceValidation.Issues, sourceValidation.RequestValidation.Issues)
	}
	assertAutoprogrammingScannerContextRefV0(t, sourceValidation.SourceRefs, "source_ref:source-ref-apg003-web")
	assertAutoprogrammingScannerContextRefV0(t, sourceValidation.SourceRefs, "source_surface:web")
	assertAutoprogrammingScannerContextRefV0(t, sourceValidation.SourceRefs, "source_transport:http")

	defaultConfig := ResolveAutoprogrammingIdleSelfImprovementConfigV0(nil)
	defaultDecision := DecideAutoprogrammingIdleSelfImprovementV0(
		AutoprogrammingIdleSelfImprovementDecisionInputV0{
			Config:         defaultConfig.Config,
			IdleForSeconds: 60,
			QueueSize:      0,
			FreeCapacity:   0,
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
					TaskRef:    "task-ref-apg003-visible",
					SectionRef: "apg-003",
					Area:       "apg-003",
					Title:      "Trabajo ya visible",
				},
				{
					SectionRef: "estado-actual",
					Title:      "Narrativa vigente",
					Narrative:  true,
				},
				{
					TaskRef:    "task-ref-apg003-failure-general-cause",
					SectionRef: "apg-003-new-gap",
					Area:       "apg-003",
					Title:      "Corregir causa general desde evidencia de fallo",
					AcceptanceCriteria: []string{
						"usar evidencia del fallo y corregir la causa general si es posible",
					},
				},
			},
			VisibleQueue: []AutoprogrammingVisibleQueueItemV0{{
				SectionRef: "apg-003",
			}},
			CreateScanner:  true,
			ScannerTaskRef: "task-ref-backlog-scanner",
			BacklogScanRef: "scan-ref-backlog-58c1f26e2898",
			BacklogScan: AutoprogrammingBacklogScanV0{
				Epoch: "backlog-scan-epoch-68612dec3688",
				ReservationRefs: []string{
					"reservation-ref-backlog-scan-doc-merge-55b2a657b4e5",
					"reservation-ref-backlog-task-id-2fba20b72349",
				},
				Documents: []AutoprogrammingBacklogDocumentV0{{
					Path:       "modulos/orquesta-autoprogramming/docs/tareas.md",
					StartLine:  34,
					SHA256:     "55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582",
					SectionRef: "apg-003",
				}},
			},
		},
	)
	if len(planner.Tasks) != 1 || planner.Tasks[0].TaskRef != "task-ref-apg003-failure-general-cause" {
		t.Fatalf("planner_tasks=%+v", planner.Tasks)
	}
	assertAutoprogrammingSkipV0(t, planner.Skipped, "task-ref-apg003-visible", AutoprogrammingBacklogPlannerSkipVisibleInQueueV0)
	if !hasAutoprogrammingPlannerSkipV0(planner.Skipped, "", AutoprogrammingBacklogPlannerSkipNarrativeSectionV0) {
		t.Fatalf("planner_skipped=%+v", planner.Skipped)
	}
	if planner.Scanner == nil || planner.Scanner.Title != "Escaneo backlog nuevos" {
		t.Fatalf("scanner=%+v", planner.Scanner)
	}
	for _, want := range []string{
		"backlog_scan_ref:scan-ref-backlog-58c1f26e2898",
		"backlog_scan_epoch:backlog-scan-epoch-68612dec3688",
		"backlog_scan_reservation_ref:reservation-ref-backlog-scan-doc-merge-55b2a657b4e5",
		"backlog_scan_reservation_ref:reservation-ref-backlog-task-id-2fba20b72349",
		"backlog_scan_doc:modulos/orquesta-autoprogramming/docs/tareas.md:line:34:sha256:55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582",
	} {
		assertAutoprogrammingScannerContextRefV0(t, planner.Scanner.ContextRefs, want)
	}

	for _, tc := range []struct {
		name  string
		input AutoprogrammingExternalWorkProjectionInputV0
		want  string
	}{
		{
			name:  "outbox",
			input: AutoprogrammingExternalWorkProjectionInputV0{OutboxPendingRefs: []string{"outbox-ref-apg003"}},
			want:  AutoprogrammingExternalProjectionOutboxPendingV0,
		},
		{
			name: "wait",
			input: AutoprogrammingExternalWorkProjectionInputV0{
				OutboxPendingRefs: []string{"outbox-ref-apg003"},
				WaitExternalRefs:  []string{"wait-external-ref-apg003"},
			},
			want: AutoprogrammingExternalProjectionWaitExternalV0,
		},
		{
			name: "verified",
			input: AutoprogrammingExternalWorkProjectionInputV0{
				OutboxPendingRefs:       []string{"outbox-ref-apg003"},
				WaitExternalRefs:        []string{"wait-external-ref-apg003"},
				ExternalProcessRefs:     []string{"process-ref-apg003"},
				ExternalProcessVerified: true,
			},
			want: AutoprogrammingExternalProjectionExternalProcessVerifiedV0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertAutoprogrammingExternalProjectionV0(t, ProjectAutoprogrammingExternalWorkV0(tc.input), tc.want)
		})
	}
}
