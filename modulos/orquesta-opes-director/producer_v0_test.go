package orquestaopesdirector

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestadomainworkmemory "orquesta/modulos/orquesta-domain-work-memory"
	orquestaopestopicregistry "orquesta/modulos/orquesta-opes-topic-registry"
)

func TestProduceOPESCausalJobsV0ExpandeDocumentPlanYEsIdempotente(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{
		validDocumentPlanArtifactRecordForTestV0(),
	}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{},
		OPESCausalProducerPortsV0{
			ArtifactSource: source,
			JobCreator:     creator,
			JobRecords:     creator,
		})
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	if result.Status != OPESCausalProducerStatusCompletedV0 ||
		len(result.CreatedJobs) < 2 {
		t.Fatalf("result=%+v", result)
	}
	if !createdWorkKindForTestV0(result.CreatedJobs, "draft_content_block") ||
		!createdWorkKindForTestV0(result.CreatedJobs, "plan_temario") {
		t.Fatalf("created_jobs=%+v", result.CreatedJobs)
	}

	second, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{},
		OPESCausalProducerPortsV0{
			ArtifactSource: source,
			JobCreator:     creator,
			JobRecords:     creator,
		})
	if err != nil {
		t.Fatalf("second ProduceOPESCausalJobsV0: %v", err)
	}
	if len(second.CreatedJobs) != 0 || len(second.SkippedRefs) == 0 {
		t.Fatalf("second=%+v", second)
	}
}

func TestProduceOPESCausalJobsV0PaquetePendienteCreaFollowupSinCerrar(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-finalize-001",
		ArtifactRef:  "artifact-final-package-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0,
		ReceiptRef:   "receipt-final-package-001",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "status", Value: "pendiente_continuar"},
			{Name: "followup_refs", Values: []string{"needs-audio-generation-and-audio-qa"}},
			{Name: "target_artifact_type", Value: orquestadomainwork.DomainWorkArtifactTypeAudioAssetV0},
		},
	}}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{},
		OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator})
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	if len(result.CreatedJobs) != 1 ||
		result.CreatedJobs[0].WorkKind != "generate_audio_asset" ||
		result.CreatedJobs[0].Status != orquestadomainwork.DomainWorkStatusAcceptedV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestProduceOPESCausalJobsV0RejectedCreaCorreccionMismaFase(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "rejected",
		DomainRef:    "opes",
		JobRef:       "job-tests-001",
		ArtifactRef:  "artifact-tests-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeQuestionBankV0,
		IssueRefs:    []string{"domain-work-submit-artifact-rejected"},
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "source_work_kind", Value: "generate_question_bank"},
		},
	}}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{},
		OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator})
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	if len(result.CreatedJobs) != 1 ||
		result.CreatedJobs[0].WorkKind != "generate_question_bank" {
		t.Fatalf("result=%+v", result)
	}
}

func TestProduceOPESCausalJobsV0RejectedTransitorioNoCreaCorreccionPrematura(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "rejected",
		DomainRef:    "opes",
		JobRef:       "job-tests-transient-001",
		ArtifactRef:  "artifact-tests-transient-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeQuestionBankV0,
		IssueRefs:    []string{"domain-work-submit-artifact-rejected", "opes_http_status_429"},
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "source_work_kind", Value: "generate_question_bank"},
		},
	}}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{},
		OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator})
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	if len(result.CreatedJobs) != 0 || len(result.RequestedJobs) != 0 ||
		!domainWorkIssueCodeForDirectorTestV0(result.Issues, ErrOPESCausalRejectedTransientRetryV0) {
		t.Fatalf("result=%+v", result)
	}
}

func TestProduceOPESCausalJobsV0PendingRefsAliasCreaFollowup(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-final-pending-refs-001",
		ArtifactRef:  "artifact-final-pending-refs-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0,
		ReceiptRef:   "receipt-final-pending-refs-001",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "pending_refs", Values: []string{"needs-audio-qa"}},
			{Name: "target_artifact_type", Value: orquestadomainwork.DomainWorkArtifactTypeAudioAssetV0},
		},
	}}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{},
		OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator})
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	if len(result.CreatedJobs) != 1 ||
		result.CreatedJobs[0].WorkKind != "generate_audio_asset" {
		t.Fatalf("result=%+v", result)
	}
	request, ok := requestedWorkKindForTestV0(result.RequestedJobs, "generate_audio_asset")
	if !ok || !domainWorkFieldValueForDirectorTestV0(request.InputFields, "followup_ref", "needs-audio-qa") {
		t.Fatalf("request=%+v ok=%v result=%+v", request, ok, result)
	}
}

func TestProduceOPESCausalJobsV0CreaActualizacionRegistroPorTema(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:        "accepted",
		DomainRef:     "opes",
		JobRef:        "job-topic-synthetic-001",
		ArtifactRef:   "artifact-topic-synthetic-001",
		ArtifactType:  orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
		ReceiptRef:    "receipt-topic-synthetic-001",
		CorrelationID: "corr-topic-synthetic-001",
		Summary:       "Entrega sintética de tema para registrar avance.",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-sintetico-a1"},
			{Name: "topic_id", Value: "tema-001"},
			{Name: "source_work_kind", Value: "draft_content_block"},
		},
	}}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	updater := &fakeTopicRegistryUpdaterV0{}
	result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{},
		OPESCausalProducerPortsV0{
			ArtifactSource:       source,
			JobCreator:           creator,
			JobRecords:           creator,
			TopicRegistryUpdater: updater,
		})
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	if len(result.CreatedJobs) != 1 || result.CreatedJobs[0].WorkKind != opesTopicRegistryUpdateWorkKindV0 {
		t.Fatalf("result=%+v", result)
	}
	request, ok := requestedWorkKindForTestV0(result.RequestedJobs, opesTopicRegistryUpdateWorkKindV0)
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "course_id", "curso-sintetico-a1") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "topic_id", "tema-001") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "registry_action", "update") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", "en_progreso_orquesta") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "expected_artifact_type", orquestadomainwork.DomainWorkArtifactTypeTopicRegistryUpdateV0) {
		t.Fatalf("request=%+v ok=%v", request, ok)
	}
	if len(updater.requests) != 1 ||
		updater.requests[0].WorkKind != opesTopicRegistryUpdateWorkKindV0 ||
		len(result.TopicRegistryUpdates) != 1 ||
		result.TopicRegistryUpdates[0].Status != orquestaopestopicregistry.TopicRegistryUpdateStatusAppliedV0 {
		t.Fatalf("updater=%+v result=%+v", updater.requests, result.TopicRegistryUpdates)
	}

	second, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{},
		OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator, JobRecords: creator})
	if err != nil {
		t.Fatalf("second ProduceOPESCausalJobsV0: %v", err)
	}
	if len(second.CreatedJobs) != 0 || len(second.SkippedRefs) != 1 {
		t.Fatalf("second=%+v", second)
	}
}

func TestProduceOPESCausalJobsV0ReintentaTopicRegistryUpdaterTrasFallo(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-topic-retry-001",
		ArtifactRef:  "artifact-topic-retry-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
		ReceiptRef:   "receipt-topic-retry-001",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-retry"},
			{Name: "topic_id", Value: "tema-001"},
			{Name: "source_work_kind", Value: "draft_content_block"},
		},
	}}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	failingUpdater := &fakeTopicRegistryUpdaterV0{
		status: orquestaopestopicregistry.TopicRegistryUpdateStatusFailedV0,
		err:    errors.New("tool unavailable"),
	}
	first, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{},
		OPESCausalProducerPortsV0{
			ArtifactSource:       source,
			JobCreator:           creator,
			JobRecords:           creator,
			TopicRegistryUpdater: failingUpdater,
		})
	if err != nil {
		t.Fatalf("first ProduceOPESCausalJobsV0: %v", err)
	}
	if len(first.CreatedJobs) != 0 ||
		len(failingUpdater.requests) != 1 ||
		!domainWorkIssueCodeForDirectorTestV0(first.Issues, ErrOPESCausalTopicRegistryFailedV0) {
		t.Fatalf("first=%+v updater=%+v", first, failingUpdater.requests)
	}

	successUpdater := &fakeTopicRegistryUpdaterV0{}
	second, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{},
		OPESCausalProducerPortsV0{
			ArtifactSource:       source,
			JobCreator:           creator,
			JobRecords:           creator,
			TopicRegistryUpdater: successUpdater,
		})
	if err != nil {
		t.Fatalf("second ProduceOPESCausalJobsV0: %v", err)
	}
	if len(second.CreatedJobs) != 1 ||
		second.CreatedJobs[0].WorkKind != opesTopicRegistryUpdateWorkKindV0 ||
		!stringsInSetV0(second.CreatedJobs[0].EvidenceRefs, opesTopicRegistryAppliedEvidenceV0) ||
		len(successUpdater.requests) != 1 {
		t.Fatalf("second=%+v updater=%+v", second, successUpdater.requests)
	}
}

func TestProduceOPESCausalJobsV0PaqueteFinalSinCompleteJobNoLiberaRegistro(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-final-no-complete-001",
		ArtifactRef:  "artifact-final-no-complete-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0,
		ReceiptRef:   "receipt-final-no-complete-001",
		CompleteJob:  false,
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-final"},
			{Name: "topic_id", Value: "tema-002"},
		},
	}}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{},
		OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator})
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	request, ok := requestedWorkKindForTestV0(result.RequestedJobs, opesTopicRegistryUpdateWorkKindV0)
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "registry_action", "update") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", "pendiente_continuar") {
		t.Fatalf("request=%+v ok=%v result=%+v", request, ok, result)
	}
}

func TestProduceOPESCausalJobsV0PaqueteFinalCompleteSinEvidenciaNoLiberaRegistro(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-final-no-evidence-001",
		ArtifactRef:  "artifact-final-no-evidence-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0,
		ReceiptRef:   "receipt-final-no-evidence-001",
		CompleteJob:  true,
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-final"},
			{Name: "topic_id", Value: "tema-003"},
		},
	}}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{},
		OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator})
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	request, ok := requestedWorkKindForTestV0(result.RequestedJobs, opesTopicRegistryUpdateWorkKindV0)
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "registry_action", "update") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", "pendiente_validacion_paquete_final") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", "final-package-manifest-closure-evidence-required") {
		t.Fatalf("request=%+v ok=%v result=%+v", request, ok, result)
	}
}

func TestProduceOPESCausalJobsV0PaqueteFinalCompleteConEvidenciaAntiguaNoLiberaRegistro(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-final-old-evidence-001",
		ArtifactRef:  "artifact-final-old-evidence-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0,
		ReceiptRef:   "receipt-final-old-evidence-001",
		CompleteJob:  true,
		EvidenceRefs: []string{"evidence-ref-opes-finalpkg-deterministic-package-contract"},
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-final"},
			{Name: "topic_id", Value: "tema-004"},
		},
	}}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{},
		OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator})
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	request, ok := requestedWorkKindForTestV0(result.RequestedJobs, opesTopicRegistryUpdateWorkKindV0)
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "registry_action", "update") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", "pendiente_validacion_paquete_final") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", "final-package-manifest-closure-evidence-required") {
		t.Fatalf("request=%+v ok=%v result=%+v", request, ok, result)
	}
}

func TestProduceOPESCausalJobsV0PaqueteFinalCompleteConManifestYEvidenciasLiberaRegistro(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-final-evidence-001",
		ArtifactRef:  "artifact-final-evidence-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0,
		ReceiptRef:   "receipt-final-evidence-001",
		CompleteJob:  true,
		EvidenceRefs: []string{
			"manifest_cierre.json",
			"opes-final-evidence:html:001",
			"opes-final-evidence:rag:001",
			"opes-final-evidence:audio:001",
			"opes-final-evidence:tests:001",
			"opes-final-evidence:visual:001",
			"opes-final-evidence:qa:001",
		},
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-final"},
			{Name: "topic_id", Value: "tema-004"},
		},
	}}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{},
		OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator})
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	request, ok := requestedWorkKindForTestV0(result.RequestedJobs, opesTopicRegistryUpdateWorkKindV0)
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "registry_action", "release") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", "paquete_final_local_verificable") ||
		domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", "final-package-manifest-closure-evidence-required") {
		t.Fatalf("request=%+v ok=%v result=%+v", request, ok, result)
	}
}

func TestProduceOPESCausalJobsV0FiltraPorCorrelationID(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{
		{
			Status:        "accepted",
			DomainRef:     "opes",
			JobRef:        "job-topic-corr-a",
			ArtifactRef:   "artifact-topic-corr-a",
			ArtifactType:  orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
			ReceiptRef:    "receipt-topic-corr-a",
			CorrelationID: "corr-a",
			PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "course_id", Value: "curso-corr"},
				{Name: "topic_id", Value: "tema-a"},
			},
		},
		{
			Status:        "accepted",
			DomainRef:     "opes",
			JobRef:        "job-topic-corr-b",
			ArtifactRef:   "artifact-topic-corr-b",
			ArtifactType:  orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
			ReceiptRef:    "receipt-topic-corr-b",
			CorrelationID: "corr-b",
			PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
				{Name: "course_id", Value: "curso-corr"},
				{Name: "topic_id", Value: "tema-b"},
			},
		},
	}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	result, err := ProduceOPESCausalJobsV0(
		context.Background(),
		OPESCausalProducerRequestV0{CorrelationID: "corr-a"},
		OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator},
	)
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	if len(result.CreatedJobs) != 1 ||
		len(result.ProcessedRefs) != 1 ||
		result.CreatedJobs[0].CorrelationID != "corr-a" {
		t.Fatalf("result=%+v", result)
	}
}

func TestProduceOPESCausalJobsV0PreservaEstadoTextoMinimoPendiente(t *testing.T) {
	const partialStatus = "texto_minimo_B_ok_pendiente_assets_html_tests_rag_audio_qa"
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-topic-partial-001",
		ArtifactRef:  "artifact-topic-partial-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
		ReceiptRef:   "receipt-topic-partial-001",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-integracion-social-b"},
			{Name: "topic_id", Value: "tema-023"},
			{Name: "source_work_kind", Value: "draft_content_block"},
			{Name: "status", Value: partialStatus},
		},
	}}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{},
		OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator, JobRecords: creator})
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	request, ok := requestedWorkKindForTestV0(result.RequestedJobs, opesTopicRegistryUpdateWorkKindV0)
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", partialStatus) {
		t.Fatalf("request=%+v ok=%v result=%+v", request, ok, result)
	}
}

func TestProduceOPESCausalJobsV0NoRepiteRegistroDesdeRegistro(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-registry-synthetic-001",
		ArtifactRef:  "artifact-registry-synthetic-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeTopicRegistryUpdateV0,
		ReceiptRef:   "receipt-registry-synthetic-001",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-sintetico-a1"},
			{Name: "topic_id", Value: "tema-001"},
			{Name: "source_work_kind", Value: opesTopicRegistryUpdateWorkKindV0},
		},
	}}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{},
		OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator, JobRecords: creator})
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	if len(result.CreatedJobs) != 0 {
		t.Fatalf("result=%+v", result)
	}
}

type fakeArtifactSourceV0 struct {
	records []OPESCausalArtifactRecordV0
}

func (source fakeArtifactSourceV0) ListOPESCausalArtifactRecordsV0(
	_ context.Context,
	filter OPESCausalArtifactRecordFilterV0,
) ([]OPESCausalArtifactRecordV0, error) {
	var out []OPESCausalArtifactRecordV0
	for _, record := range source.records {
		if filter.DomainRef != "" && record.DomainRef != filter.DomainRef {
			continue
		}
		if filter.CorrelationID != "" && record.CorrelationID != filter.CorrelationID {
			continue
		}
		out = append(out, record)
	}
	return out, nil
}

func validDocumentPlanArtifactRecordForTestV0() OPESCausalArtifactRecordV0 {
	sections, _ := json.Marshal([]orquestadomainwork.DomainDocumentPlanSectionV0{{
		SectionRef: "section-topic-001",
		Order:      1,
		Title:      "Tema 1",
		Objective:  "Redactar el tema base.",
		WorkKind:   "draft_content_block",
	}})
	deliverables, _ := json.Marshal([]orquestadomainwork.DomainDocumentPlanDeliverableV0{{
		DeliverableRef: "deliverable-topic-001",
		ArtifactType:   "assembled_topic",
		Title:          "Tema ensamblado",
		Required:       true,
	}})
	return OPESCausalArtifactRecordV0{
		Status:        "accepted",
		DomainRef:     "opes",
		JobRef:        "job-plan-temario-001",
		ArtifactRef:   "artifact-plan-temario-001",
		ArtifactType:  orquestadomainwork.DomainDocumentPlanArtifactTypeV0,
		ReceiptRef:    "receipt-plan-temario-001",
		CorrelationID: "corr-plan-temario-001",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "schema_version", Value: orquestadomainwork.DomainDocumentPlanSchemaV0},
			{Name: "plan_ref", Value: "plan-temario-001"},
			{Name: "domain_ref", Value: "opes"},
			{Name: "work_kind", Value: "plan_temario"},
			{Name: "document_kind", Value: "temario_oposicion"},
			{Name: "language_code", Value: "es"},
			{Name: "title", Value: "Temario de prueba"},
			{Name: "objective", Value: "Planificar un temario completo OPES."},
			{Name: "sections", ValueJSON: sections},
			{Name: "deliverables", ValueJSON: deliverables},
		},
	}
}

func createdWorkKindForTestV0(jobs []orquestadomainwork.DomainWorkJobV0, workKind string) bool {
	for _, job := range jobs {
		if job.WorkKind == workKind {
			return true
		}
	}
	return false
}

func requestedWorkKindForTestV0(
	jobs []orquestadomainwork.DomainWorkJobRequestV0,
	workKind string,
) (orquestadomainwork.DomainWorkJobRequestV0, bool) {
	for _, job := range jobs {
		if job.WorkKind == workKind {
			return job, true
		}
	}
	return orquestadomainwork.DomainWorkJobRequestV0{}, false
}

func domainWorkFieldValueForDirectorTestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
	value string,
) bool {
	for _, field := range fields {
		if field.Name == name && field.Value == value {
			return true
		}
		for _, item := range field.Values {
			if field.Name == name && item == value {
				return true
			}
		}
	}
	return false
}

type fakeTopicRegistryUpdaterV0 struct {
	requests []orquestadomainwork.DomainWorkJobRequestV0
	status   string
	err      error
}

func (updater *fakeTopicRegistryUpdaterV0) ApplyOPESCausalTopicRegistryUpdateV0(
	_ context.Context,
	request orquestadomainwork.DomainWorkJobRequestV0,
) (orquestaopestopicregistry.TopicRegistryUpdateResultV0, error) {
	updater.requests = append(updater.requests, request)
	status := updater.status
	if status == "" {
		status = orquestaopestopicregistry.TopicRegistryUpdateStatusAppliedV0
	}
	return orquestaopestopicregistry.TopicRegistryUpdateResultV0{
		SchemaVersion: orquestaopestopicregistry.TopicRegistryUpdateResultSchemaV0,
		Status:        status,
		EvidenceRefs:  []string{"evidence-ref-topic-registry-applied"},
	}, updater.err
}

func domainWorkIssueCodeForDirectorTestV0(
	issues []orquestadomainwork.DomainWorkIssueV0,
	code string,
) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
