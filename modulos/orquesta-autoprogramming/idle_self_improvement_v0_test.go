package orquestaautoprogramming

import "testing"

func TestResolveAutoprogrammingIdleSelfImprovementConfigV0DefaultDisparaTras60Segundos(t *testing.T) {
	config := ResolveAutoprogrammingIdleSelfImprovementConfigV0(nil)
	if len(config.Issues) != 0 ||
		config.Config.AfterSeconds != AutoprogrammingIdleSelfImprovementDefaultAfterSecondsV0 {
		t.Fatalf("config=%+v", config)
	}

	early := DecideAutoprogrammingIdleSelfImprovementV0(AutoprogrammingIdleSelfImprovementDecisionInputV0{
		Config:         config.Config,
		IdleForSeconds: 59,
		QueueSize:      1,
		FreeCapacity:   0,
	})
	if early.Prepare {
		t.Fatalf("early=%+v", early)
	}

	ready := DecideAutoprogrammingIdleSelfImprovementV0(AutoprogrammingIdleSelfImprovementDecisionInputV0{
		Config:         config.Config,
		IdleForSeconds: 60,
		QueueSize:      1,
		FreeCapacity:   0,
	})
	if !ready.Prepare || !ready.IdleTriggered || ready.Reason != "idle_after_seconds_reached" {
		t.Fatalf("ready=%+v", ready)
	}
}

func TestResolveAutoprogrammingIdleSelfImprovementConfigV0CeroDesactivaSoloRelojIdle(t *testing.T) {
	config := ResolveAutoprogrammingIdleSelfImprovementConfigV0(map[string]string{
		OrquestaServerIdleSelfImprovementAfterSecondsEnvV0: "0",
	})

	withoutCapacity := DecideAutoprogrammingIdleSelfImprovementV0(AutoprogrammingIdleSelfImprovementDecisionInputV0{
		Config:         config.Config,
		IdleForSeconds: 999,
		QueueSize:      1,
		FreeCapacity:   0,
	})

	if withoutCapacity.Prepare || !withoutCapacity.Disabled || withoutCapacity.Reason != "idle_self_improvement_disabled" {
		t.Fatalf("without_capacity=%+v", withoutCapacity)
	}

	withCapacity := DecideAutoprogrammingIdleSelfImprovementV0(AutoprogrammingIdleSelfImprovementDecisionInputV0{
		Config:         config.Config,
		IdleForSeconds: 999,
		QueueSize:      0,
		FreeCapacity:   1,
	})

	if !withCapacity.Prepare || withCapacity.Disabled || !withCapacity.QueueTriggered ||
		withCapacity.IdleTriggered || withCapacity.Reason != "free_capacity_below_target_queue" {
		t.Fatalf("with_capacity=%+v", withCapacity)
	}
}

func TestDecideAutoprogrammingIdleSelfImprovementV0PreparaConCapacidadLibreBajoTargetQueue(t *testing.T) {
	config := ResolveAutoprogrammingIdleSelfImprovementConfigV0(map[string]string{
		OrquestaServerIdleSelfImprovementAfterSecondsEnvV0: "300",
		OrquestaServerIdleSelfImprovementTargetQueueEnvV0:  "3",
	})

	decision := DecideAutoprogrammingIdleSelfImprovementV0(AutoprogrammingIdleSelfImprovementDecisionInputV0{
		Config:         config.Config,
		IdleForSeconds: 1,
		QueueSize:      2,
		FreeCapacity:   1,
	})

	if !decision.Prepare || !decision.QueueTriggered || decision.Reason != "free_capacity_below_target_queue" {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestPlanAutoprogrammingBacklogSelfImprovementV0SaltaColaNarrativasYCreaScanner(t *testing.T) {
	result := PlanAutoprogrammingBacklogSelfImprovementV0(AutoprogrammingBacklogPlannerInputV0{
		Entries: []AutoprogrammingBacklogPlannerEntryV0{
			{
				TaskRef:    "task-ref-visible",
				SectionRef: "apg-001",
				Area:       "Autoprogramming",
				Title:      "Visible en cola",
			},
			{
				TaskRef:    "task-ref-narrative",
				SectionRef: "estado-actual",
				Narrative:  true,
				Title:      "Seccion narrativa",
			},
			{
				TaskRef:    "task-ref-new-gap",
				SectionRef: "apg-006",
				Area:       "Autoprogramming",
				Title:      "Hueco nuevo",
			},
		},
		VisibleQueue: []AutoprogrammingVisibleQueueItemV0{{
			TaskRef: "task-ref-visible",
		}},
		CreateScanner:  true,
		ScannerTaskRef: "task-ref-backlog-scanner",
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
		BacklogScanRef: "scan-ref-backlog-58c1f26e2898",
	})

	if len(result.Tasks) != 1 || result.Tasks[0].TaskRef != "task-ref-new-gap" {
		t.Fatalf("tasks=%+v", result.Tasks)
	}
	if result.Scanner == nil || result.Scanner.Title != "Escaneo backlog nuevos" {
		t.Fatalf("scanner=%+v", result.Scanner)
	}
	if !stringsSliceContainsForAutoprogrammingTestV0(
		result.Scanner.ContextRefs,
		"backlog_scan_ref:scan-ref-backlog-58c1f26e2898",
	) {
		t.Fatalf("scanner refs=%v", result.Scanner.ContextRefs)
	}
	if !stringsSliceContainsForAutoprogrammingTestV0(
		result.Scanner.ContextRefs,
		"backlog_scan_doc:modulos/orquesta-autoprogramming/docs/tareas.md:line:34:sha256:55c7ad4bcc6cd71d41ec12ebdbdc6e19e54c3c0e47cfd2d3947f302398bc3582",
	) {
		t.Fatalf("scanner refs=%v", result.Scanner.ContextRefs)
	}
	assertAutoprogrammingSkipV0(t, result.Skipped, "task-ref-visible", AutoprogrammingBacklogPlannerSkipVisibleInQueueV0)
	assertAutoprogrammingSkipV0(t, result.Skipped, "task-ref-narrative", AutoprogrammingBacklogPlannerSkipNarrativeSectionV0)
}

func TestPlanAutoprogrammingBacklogSelfImprovementV0NoDuplicaScannerVisiblePorSeccion(t *testing.T) {
	for _, tc := range []struct {
		name           string
		backlogScanRef string
		visibleSection string
	}{
		{
			name:           "seccion_scanner",
			visibleSection: "backlog_scanner",
		},
		{
			name:           "scan_ref",
			backlogScanRef: "scan-ref-backlog-72c73a98bd54",
			visibleSection: "scan-ref-backlog-72c73a98bd54",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := PlanAutoprogrammingBacklogSelfImprovementV0(AutoprogrammingBacklogPlannerInputV0{
				CreateScanner:  true,
				BacklogScanRef: tc.backlogScanRef,
				VisibleQueue: []AutoprogrammingVisibleQueueItemV0{{
					SectionRef: tc.visibleSection,
				}},
			})

			if result.Scanner != nil {
				t.Fatalf("scanner=%+v", result.Scanner)
			}
		})
	}
}

func TestPlanAutoprogrammingBacklogSelfImprovementV0NoMaterializaSeccionSinTaskRefAunqueConserveEvidenciaScanner(t *testing.T) {
	result := PlanAutoprogrammingBacklogSelfImprovementV0(AutoprogrammingBacklogPlannerInputV0{
		Entries: []AutoprogrammingBacklogPlannerEntryV0{{
			SectionRef:         "apg-003",
			Area:               "autoprogramming",
			Title:              "Contrato APG-003 ya documentado",
			WriteSet:           []string{"modulos/orquesta-autoprogramming"},
			RequiredTests:      []string{"go test -count=1 ./modulos/orquesta-autoprogramming"},
			AcceptanceCriteria: []string{"preservar evidencia de entrada publica"},
			ContextRefs: []string{
				"backlog_scan_ref:scan-ref-backlog-5591c1722dcf",
				"backlog_scan_epoch:backlog-scan-epoch-cd0b20695bc4",
			},
		}},
	})

	if len(result.Tasks) != 0 {
		t.Fatalf("tasks=%+v", result.Tasks)
	}
	if !hasAutoprogrammingPlannerSkipSectionV0(
		result.Skipped,
		"apg-003",
		AutoprogrammingBacklogPlannerSkipNarrativeSectionV0,
	) {
		t.Fatalf("skipped=%+v", result.Skipped)
	}
}

func TestProjectAutoprogrammingExternalWorkV0DistingueEstadosPublicos(t *testing.T) {
	cases := []struct {
		name  string
		input AutoprogrammingExternalWorkProjectionInputV0
		want  string
	}{
		{
			name:  "outbox",
			input: AutoprogrammingExternalWorkProjectionInputV0{OutboxPendingRefs: []string{"outbox-ref-001"}},
			want:  AutoprogrammingExternalProjectionOutboxPendingV0,
		},
		{
			name: "wait",
			input: AutoprogrammingExternalWorkProjectionInputV0{
				OutboxPendingRefs: []string{"outbox-ref-001"},
				WaitExternalRefs:  []string{"wait-external-ref-001"},
			},
			want: AutoprogrammingExternalProjectionWaitExternalV0,
		},
		{
			name: "verified",
			input: AutoprogrammingExternalWorkProjectionInputV0{
				OutboxPendingRefs:       []string{"outbox-ref-001"},
				WaitExternalRefs:        []string{"wait-external-ref-001"},
				ExternalProcessRefs:     []string{"process-ref-001"},
				ExternalProcessVerified: true,
			},
			want: AutoprogrammingExternalProjectionExternalProcessVerifiedV0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ProjectAutoprogrammingExternalWorkV0(tc.input)
			if got.State != tc.want || len(got.Refs) == 0 {
				t.Fatalf("got=%+v want=%s", got, tc.want)
			}
		})
	}
}

func assertAutoprogrammingSkipV0(
	t *testing.T,
	skipped []AutoprogrammingBacklogPlannerSkippedV0,
	taskRef string,
	reason string,
) {
	t.Helper()
	for _, skip := range skipped {
		if skip.TaskRef == taskRef && skip.Reason == reason {
			return
		}
	}
	t.Fatalf("skip %s/%s no encontrado en %+v", taskRef, reason, skipped)
}

func hasAutoprogrammingPlannerSkipSectionV0(
	skipped []AutoprogrammingBacklogPlannerSkippedV0,
	sectionRef string,
	reason string,
) bool {
	for _, skip := range skipped {
		if skip.SectionRef == sectionRef && skip.Reason == reason {
			return true
		}
	}
	return false
}
