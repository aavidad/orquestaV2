package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestOperationalClosureSourceV0CierraOPESDomainWorkConReceiptAceptado(t *testing.T) {
	ctx := context.Background()
	fixture := newOPESDomainWorkClosureFixtureForTestV0(t, "accepted")
	record := opesAcceptedSubmissionRecordForFixtureV0(fixture, "accepted")
	if err := fixture.Ledger.RecordDomainWorkArtifactSubmissionV0(ctx, record); err != nil {
		t.Fatalf("record receipt: %v", err)
	}

	got, ok, err := fixture.Source.BuildOperationalDirectorClosureRequestV0(
		ctx,
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{
			Run:        fixture.Run,
			OccurredAt: "2026-05-24T12:00:00Z",
		},
	)
	if err != nil || !ok {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0 ok=%v err=%v request=%+v", ok, err, got)
	}
	if got.TaskID != fixture.Task.TaskID ||
		!codexStackOperationalClosureContainsV0(got.EvidenceRefs, "receipt-ref-opes-closure-accepted") ||
		!codexStackOperationalClosureContainsV0(got.EvidenceRefs, "evidence-ref-opes-domain-work-accepted") {
		t.Fatalf("cierre OPES sin receipt causal: %+v", got)
	}
}

func TestOperationalClosureSourceV0CierraOPESDomainWorkConAliasProyectoCurso(t *testing.T) {
	ctx := context.Background()
	fixture := newOPESDomainWorkClosureFixtureForTestV0(t, "alias-curso")
	record := opesDomainWorkAppChangeRecordForTestV0(fixture.Run.RunID, "opes-job-job-ref-closure-alias-curso")
	record.Request.ExternalWork.ProjectRef = "opes-a1-informatica"
	fixture.Source.AppChangeStore = orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	submission := opesAcceptedSubmissionRecordForFixtureV0(fixture, "alias-curso")
	submission.DomainRef = "opes-a1-informatica"
	if err := fixture.Ledger.RecordDomainWorkArtifactSubmissionV0(ctx, submission); err != nil {
		t.Fatalf("record receipt: %v", err)
	}

	got, ok, err := fixture.Source.BuildOperationalDirectorClosureRequestV0(
		ctx,
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{
			Run:        fixture.Run,
			OccurredAt: "2026-06-12T12:45:00Z",
		},
	)
	if err != nil || !ok {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0 ok=%v err=%v request=%+v", ok, err, got)
	}
	if got.TaskID != fixture.Task.TaskID ||
		!codexStackOperationalClosureContainsV0(got.EvidenceRefs, "receipt-ref-opes-closure-alias-curso") ||
		!codexStackOperationalClosureContainsV0(got.EvidenceRefs, "evidence-ref-opes-domain-work-accepted") {
		t.Fatalf("cierre OPES alias no causal: %+v", got)
	}
}

func TestOperationalClosureSourceV0CierraDomainWorkConNombresNoCanonicos(t *testing.T) {
	ctx := context.Background()
	fixture := newOPESDomainWorkClosureFixtureForTestV0(t, "nombres-no-canonicos")
	record := opesDomainWorkAppChangeRecordForTestV0(fixture.Run.RunID, "opes-job-job-ref-closure-nombres-no-canonicos")
	record.Request.AppRef = "curso_informatica_a1"
	record.Request.ExternalWork.ProjectRef = "temario-informatica-a1"
	fixture.Source.AppChangeStore = orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	submission := opesAcceptedSubmissionRecordForFixtureV0(fixture, "nombres-no-canonicos")
	submission.DomainRef = "nombre-libre-devuelto-por-agente"
	if err := fixture.Ledger.RecordDomainWorkArtifactSubmissionV0(ctx, submission); err != nil {
		t.Fatalf("record receipt: %v", err)
	}

	got, ok, err := fixture.Source.BuildOperationalDirectorClosureRequestV0(
		ctx,
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{
			Run:        fixture.Run,
			OccurredAt: "2026-06-12T13:20:00Z",
		},
	)
	if err != nil || !ok {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0 ok=%v err=%v request=%+v", ok, err, got)
	}
	if got.TaskID != fixture.Task.TaskID ||
		!codexStackOperationalClosureContainsV0(got.EvidenceRefs, "receipt-ref-opes-closure-nombres-no-canonicos") ||
		!codexStackOperationalClosureContainsV0(got.EvidenceRefs, "evidence-ref-domain-work-accepted") ||
		codexStackOperationalClosureContainsV0(got.EvidenceRefs, "evidence-ref-opes-domain-work-accepted") {
		t.Fatalf("cierre DomainWork depende de etiqueta OPES o ignora receipt causal: %+v", got)
	}
}

func TestOperationalClosureSourceV0CierraOPESDomainWorkDesdeProyeccionRunSinEventos(t *testing.T) {
	ctx := context.Background()
	fixture := newOPESDomainWorkClosureFixtureForTestV0(t, "projection-only")
	fixture.Run.Reviews = []string{"review-request-ref-" + fixture.DeliveryRef}
	fixture.Run.ReviewResults = []string{
		"review-result-ref-" + fixture.DeliveryRef +
			"#review_result:accepted#review_request:review-request-ref-" + fixture.DeliveryRef +
			"#delivery:" + fixture.DeliveryRef,
	}
	fixture.Source.EventReader = orquestacionnucleoapp.NewInMemoryEventSinkV0()
	record := opesAcceptedSubmissionRecordForFixtureV0(fixture, "projection-only")
	if err := fixture.Ledger.RecordDomainWorkArtifactSubmissionV0(ctx, record); err != nil {
		t.Fatalf("record receipt: %v", err)
	}
	testEvidenceRef := "test-evidence-ref-" + fixture.DeliveryRef

	got, ok, err := fixture.Source.BuildOperationalDirectorClosureRequestV0(
		ctx,
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{
			Run:                      fixture.Run,
			OccurredAt:               "2026-06-12T12:20:00Z",
			RequiredTestEvidenceRefs: []string{testEvidenceRef},
		},
	)
	if err != nil || !ok {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0 ok=%v err=%v request=%+v", ok, err, got)
	}
	if got.TaskID != fixture.Task.TaskID ||
		got.DeliveryRef != fixture.DeliveryRef ||
		!codexStackOperationalClosureContainsV0(got.RequiredTestEvidenceRefs, testEvidenceRef) ||
		!codexStackOperationalClosureContainsV0(got.EvidenceRefs, "evidence-ref-operational-closure-run-projection-delivery") ||
		!codexStackOperationalClosureContainsV0(got.EvidenceRefs, "evidence-ref-opes-domain-work-accepted") {
		t.Fatalf("cierre OPES desde proyeccion no causal: %+v", got)
	}
}

func TestOperationalClosureSourceV0NoCierraOPESConReceiptNoCausal(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*DomainWorkArtifactSubmissionRecordV0)
	}{
		{
			name: "job-ref-distinto",
			mutate: func(record *DomainWorkArtifactSubmissionRecordV0) {
				record.JobRef = "job-ref-otro"
			},
		},
		{
			name: "artifact-type-distinto",
			mutate: func(record *DomainWorkArtifactSubmissionRecordV0) {
				record.ArtifactType = "visual_asset"
			},
		},
		{
			name: "complete-job-falso",
			mutate: func(record *DomainWorkArtifactSubmissionRecordV0) {
				record.CompleteJob = false
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newOPESDomainWorkClosureFixtureForTestV0(t, tc.name)
			record := opesAcceptedSubmissionRecordForFixtureV0(fixture, tc.name)
			tc.mutate(&record)
			if err := fixture.Ledger.RecordDomainWorkArtifactSubmissionV0(context.Background(), record); err != nil {
				t.Fatalf("record receipt: %v", err)
			}
			_, ok, err := fixture.Source.BuildOperationalDirectorClosureRequestV0(
				context.Background(),
				orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{Run: fixture.Run},
			)
			if err != nil {
				t.Fatalf("BuildOperationalDirectorClosureRequestV0: %v", err)
			}
			if ok {
				t.Fatalf("no debe cerrar OPES con receipt no causal %+v", record)
			}
		})
	}
}

func TestOperationalClosureSourceV0NoCierraOPESSinReceiptAceptado(t *testing.T) {
	fixture := newOPESDomainWorkClosureFixtureForTestV0(t, "missing-receipt")

	_, ok, err := fixture.Source.BuildOperationalDirectorClosureRequestV0(
		context.Background(),
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{Run: fixture.Run},
	)
	if err != nil {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0: %v", err)
	}
	if ok {
		t.Fatalf("no debe cerrar OPES sin receipt aceptado de DomainWork")
	}
}

func TestOperationalClosureSourceV0NoCierraOPESFinalSinMinimosYComunes(t *testing.T) {
	ctx := context.Background()
	fixture := newOPESDomainWorkClosureFixtureForTestV0(t, "final-sin-minimos")
	record := opesDomainWorkAppChangeRecordForTestV0(fixture.Run.RunID, "opes-job-job-ref-closure-final-sin-minimos")
	record.Request.ExternalWork.WorkKind = "finalize_temario_package"
	fixture.Source.AppChangeStore = orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	submission := opesAcceptedSubmissionRecordForFixtureV0(fixture, "final-sin-minimos")
	submission.ArtifactType = "final_domain_package"
	if err := fixture.Ledger.RecordDomainWorkArtifactSubmissionV0(ctx, submission); err != nil {
		t.Fatalf("record receipt: %v", err)
	}

	_, ok, err := fixture.Source.BuildOperationalDirectorClosureRequestV0(
		ctx,
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{Run: fixture.Run},
	)
	if err != nil {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0: %v", err)
	}
	if ok {
		t.Fatalf("no debe cerrar OPES final sin evidencias de extension y comunes")
	}
}

func TestOperationalClosureSourceV0CierraOPESFinalConMinimosYComunes(t *testing.T) {
	ctx := context.Background()
	fixture := newOPESDomainWorkClosureFixtureForTestV0(t, "final-con-minimos")
	record := opesDomainWorkAppChangeRecordForTestV0(fixture.Run.RunID, "opes-job-job-ref-closure-final-con-minimos")
	record.Request.ExternalWork.WorkKind = "finalize_temario_package"
	fixture.Source.AppChangeStore = orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	submission := opesAcceptedSubmissionRecordForFixtureV0(fixture, "final-con-minimos")
	submission.ArtifactType = "final_domain_package"
	submission.EvidenceRefs = append(submission.EvidenceRefs,
		"opes-extension-minima-passed",
		"opes-common-master-not-applicable",
	)
	if err := fixture.Ledger.RecordDomainWorkArtifactSubmissionV0(ctx, submission); err != nil {
		t.Fatalf("record receipt: %v", err)
	}

	got, ok, err := fixture.Source.BuildOperationalDirectorClosureRequestV0(
		ctx,
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{Run: fixture.Run},
	)
	if err != nil || !ok {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0 ok=%v err=%v request=%+v", ok, err, got)
	}
	if !codexStackOperationalClosureContainsV0(got.EvidenceRefs, "opes-extension-minima-passed") ||
		!codexStackOperationalClosureContainsV0(got.EvidenceRefs, "opes-common-master-not-applicable") {
		t.Fatalf("cierre final OPES sin evidencias propagadas: %+v", got)
	}
}

func TestOperationalClosureSourceV0CierraAppChangeExternalWorkLegacyConTestsCausales(t *testing.T) {
	ctx := context.Background()
	runRef := "run-stack-opes-a1-legacy-app-change-closure"
	changeRef := "change-opes-a1-informatica-t058-checkpoint"
	taskRef := orquestaappchangedirectorsource.AppChangeTaskRefV0(changeRef)
	deliveryRef := "ack-ref-app-stack-agent-ref-" + taskRef
	requiredTests := []string{
		"validar criterios de aceptacion del cambio",
		"validar contrato externo de dominio",
	}
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:   orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:          taskRef,
		RunID:           runRef,
		PhaseID:         orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind: orquestacoreworkflow.WorkProfileDomainWorkV0,
		Title:           "Resolver checkpoint OPES A1 Informatica",
		WriteSet: []string{
			"opes-salidas/coordinacion_temarios/a1_maestros_todas_opes_2026-06-11/informatica_a1_72_padres/tema_058",
		},
		AcceptanceCriteria: []string{
			"No crear un temario nuevo ni duplicar A1 Informatica.",
			"Actualizar REGISTRO_TRABAJO_TEMAS_OPES con resumen, hecho, pendiente y evidencia.",
			"operational_director.task_source: director_decision",
		},
		RequiredTests: requiredTests,
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  "contract:function:app-change:legacy:v0",
			FunctionName: "ApplyExternalDomainWorkV0",
		}},
	}
	run := stackOperationalClosureRunForTestV0(runRef, taskRef)
	run.Deliveries = []string{deliveryRef}
	run.DeliveredTasks = []string{taskRef}
	run.DeliveredAgents = []string{orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)}
	run.AcceptedReviews = []string{"accepted-review-ref-" + deliveryRef}
	reader := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	evidenceA := appChangeLegacyRequiredTestEvidenceForTestV0(runRef, taskRef, deliveryRef, requiredTests[0], "criterios")
	evidenceB := appChangeLegacyRequiredTestEvidenceForTestV0(runRef, taskRef, deliveryRef, requiredTests[1], "contrato")
	if err := reader.AppendRunEventsV0(ctx, runRef, []orquestacoreworkflow.OrchestrationEventV0{
		stackOperationalClosureDeliveryEventForTestV0(t, runRef, 1, taskRef, deliveryRef),
		stackOperationalClosureReviewRequestedEventForTestV0(t, runRef, 2, deliveryRef),
		stackOperationalClosureReviewResultEventWithEvidenceRefsForTestV0(
			t,
			runRef,
			3,
			deliveryRef,
			orquestacoreworkflow.ReviewResultStatusAcceptedV0,
			[]string{evidenceA.EvidenceRef, evidenceB.EvidenceRef},
		),
		stackOperationalClosureAcceptedReviewEventForTestV0(t, runRef, 4, deliveryRef),
	}); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	source := codexStackOperationalClosureSourceV0{
		TaskStore:                 orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
		EventReader:               reader,
		RequiredTestEvidenceStore: orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(evidenceA, evidenceB),
		AppChangeStore: orquestaappchange.NewInMemoryAppChangeStoreV0(orquestaappchange.AppChangeRecordV0{
			Request: orquestaappchange.PrepareAppChangeRequestV0(orquestaappchange.AppChangeRequestV0{
				RunRef:    runRef,
				AppRef:    "opes-a1-informatica",
				ChangeRef: changeRef,
				ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
					ProjectRef:    "opes-a1-informatica",
					JobRef:        "job-ref-opes-a1-informatica-t058-checkpoint",
					InterfaceRefs: []string{"opes-local-registry-v1", "orquesta-domain-work-v0"},
					WorkKind:      "review_director_consolidation",
				},
			}),
		}),
	}

	got, ok, err := source.BuildOperationalDirectorClosureRequestV0(
		ctx,
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{
			Run:                      run,
			OccurredAt:               "2026-06-12T12:00:00Z",
			RequiredTestEvidenceRefs: []string{evidenceA.EvidenceRef, evidenceB.EvidenceRef},
		},
	)
	if err != nil || !ok {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0 ok=%v err=%v request=%+v", ok, err, got)
	}
	if got.TaskID != taskRef ||
		got.DeliveryRef != deliveryRef ||
		!codexStackOperationalClosureContainsV0(got.RequiredTestEvidenceRefs, evidenceA.EvidenceRef) ||
		!codexStackOperationalClosureContainsV0(got.RequiredTestEvidenceRefs, evidenceB.EvidenceRef) ||
		codexStackOperationalClosureContainsV0(got.EvidenceRefs, "evidence-ref-opes-domain-work-accepted") {
		t.Fatalf("cierre app-change legacy no causal o mezclado con receipt OPES estricto: %+v", got)
	}
}

func opesAcceptedSubmissionRecordForFixtureV0(
	fixture opesDomainWorkClosureFixtureForTestV0,
	suffix string,
) DomainWorkArtifactSubmissionRecordV0 {
	return DomainWorkArtifactSubmissionRecordV0{
		IdempotencyKey: "idem-opes-closure-" + suffix,
		Status:         DomainWorkArtifactSubmissionStatusAcceptedV0,
		RunRef:         fixture.Run.RunID,
		TaskRef:        fixture.Task.TaskID,
		DeliveryRef:    fixture.DeliveryRef,
		DomainRef:      "opes",
		JobRef:         "job-ref-closure",
		ArtifactRef:    fixture.DeliveryRef,
		ArtifactType:   "content_block",
		CompleteJob:    true,
		ReceiptRef:     "receipt-ref-opes-closure-" + suffix,
		EvidenceRefs:   []string{"opes-artifact-ref-" + suffix},
	}
}

type opesDomainWorkClosureFixtureForTestV0 struct {
	Run         orquestacoreworkflow.OrchestrationRunV0
	Task        orquestacoreworkflow.WorkflowTaskV0
	DeliveryRef string
	Ledger      *InMemoryDomainWorkArtifactSubmissionLedgerV0
	Source      codexStackOperationalClosureSourceV0
}

func newOPESDomainWorkClosureFixtureForTestV0(
	t *testing.T,
	suffix string,
) opesDomainWorkClosureFixtureForTestV0 {
	t.Helper()
	runRef := "run-stack-opes-domain-closure-" + suffix
	changeRef := "opes-job-job-ref-closure-" + suffix
	taskRef := orquestaappchangedirectorsource.AppChangeTaskRefV0(changeRef)
	deliveryRef := "delivery-ref-opes-domain-closure-" + suffix
	testRef := "opes-domain-test-draft_content_block-job-ref-closure-" + suffix
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:   orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:          taskRef,
		RunID:           runRef,
		PhaseID:         orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind: orquestacoreworkflow.WorkProfileDomainWorkV0,
		Title:           "Resolver job OPES con cierre causal",
		WriteSet:        []string{"external/opes/draft_content_block/job-ref-closure-" + suffix},
		AcceptanceCriteria: []string{
			"OPES acepta el artefacto y devuelve receipt por API publica",
		},
		RequiredTests: []string{testRef},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			FunctionName: "ApplyExternalDomainWorkV0",
		}},
	}
	run := stackOperationalClosureRunForTestV0(runRef, taskRef)
	run.Deliveries = []string{deliveryRef}
	run.AcceptedReviews = []string{"accepted-review-ref-" + deliveryRef}
	reader := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := reader.AppendRunEventsV0(context.Background(), runRef, []orquestacoreworkflow.OrchestrationEventV0{
		stackOperationalClosureDeliveryEventForTestV0(t, runRef, 1, taskRef, deliveryRef),
		stackOperationalClosureReviewRequestedEventForTestV0(t, runRef, 2, deliveryRef),
		stackOperationalClosureReviewResultEventForTestV0(t, runRef, 3, deliveryRef, orquestacoreworkflow.ReviewResultStatusAcceptedV0),
		stackOperationalClosureAcceptedReviewEventForTestV0(t, runRef, 4, deliveryRef),
	}); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	return opesDomainWorkClosureFixtureForTestV0{
		Run:         run,
		Task:        task,
		DeliveryRef: deliveryRef,
		Ledger:      ledger,
		Source: codexStackOperationalClosureSourceV0{
			TaskStore:                 orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
			EventReader:               reader,
			RequiredTestEvidenceStore: orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(opesDomainRequiredTestEvidenceForTestV0(runRef, taskRef, deliveryRef, testRef)),
			AppChangeStore:            orquestaappchange.NewInMemoryAppChangeStoreV0(opesDomainWorkAppChangeRecordForTestV0(runRef, changeRef)),
			DomainSubmissionLedger:    ledger,
		},
	}
}

func opesDomainWorkAppChangeRecordForTestV0(
	runRef string,
	changeRef string,
) orquestaappchange.AppChangeRecordV0 {
	return orquestaappchange.AppChangeRecordV0{
		Request: orquestaappchange.PrepareAppChangeRequestV0(orquestaappchange.AppChangeRequestV0{
			RunRef:     runRef,
			AppRef:     "opes",
			ChangeRef:  changeRef,
			UserIntent: "Resolver job OPES por contrato externo.",
			ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
				ProjectRef: "opes",
				JobRef:     "job-ref-closure",
				WorkKind:   "draft_content_block",
			},
		}),
	}
}

func opesDomainRequiredTestEvidenceForTestV0(
	runRef string,
	taskRef string,
	deliveryRef string,
	testRef string,
) orquestacionnucleoapp.RequiredTestEvidenceV0 {
	return orquestacionnucleoapp.RequiredTestEvidenceV0{
		SchemaVersion:     orquestacionnucleoapp.RequiredTestEvidenceSchemaVersionV0,
		EvidenceRef:       "test-evidence-ref-" + deliveryRef,
		RunRef:            runRef,
		TaskRef:           taskRef,
		TestCommand:       testRef,
		Status:            orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
		DeliveryRef:       deliveryRef,
		ReviewRequestID:   "review-request-ref-" + deliveryRef,
		ReviewResultRef:   "review-result-ref-" + deliveryRef,
		AcceptedReviewRef: "accepted-review-ref-" + deliveryRef,
		OccurredAt:        "2026-05-24T12:00:00Z",
		EvidenceRefs:      []string{"domain-test-output-ref-" + deliveryRef},
	}
}

func appChangeLegacyRequiredTestEvidenceForTestV0(
	runRef string,
	taskRef string,
	deliveryRef string,
	testRef string,
	suffix string,
) orquestacionnucleoapp.RequiredTestEvidenceV0 {
	return orquestacionnucleoapp.RequiredTestEvidenceV0{
		SchemaVersion:     orquestacionnucleoapp.RequiredTestEvidenceSchemaVersionV0,
		EvidenceRef:       "test-evidence-ref-app-change-legacy-" + suffix,
		RunRef:            runRef,
		TaskRef:           taskRef,
		TestCommand:       testRef,
		Status:            orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
		DeliveryRef:       deliveryRef,
		ReviewRequestID:   "review-request-ref-" + deliveryRef,
		ReviewResultRef:   "review-result-ref-" + deliveryRef,
		AcceptedReviewRef: "accepted-review-ref-" + deliveryRef,
		OccurredAt:        "2026-06-12T12:00:00Z",
		EvidenceRefs:      []string{"required-test-output-ref-app-change-legacy-" + suffix},
	}
}
