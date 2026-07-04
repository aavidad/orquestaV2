package orquestaopesdirector

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestadomainworkmemory "orquesta/modulos/orquesta-domain-work-memory"
	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
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
		EvidenceRefs:  []string{"opes-final-evidence:topic_quality_contract_pass"},
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-sintetico-a1"},
			{Name: "topic_id", Value: "tema-001"},
			{Name: "source_work_kind", Value: "draft_content_block"},
			{Name: "canonical_word_count", Value: "12000"},
			{Name: "topic_quality_status", Value: "passed"},
			{Name: "topic_text", Value: "Contenido publico del tema con desarrollo normativo, ejemplos profesionales y explicacion didactica sin instrucciones internas."},
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
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", topicRegistryStatusTextSettledPendingDerivativesV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "operational_status", "waiting") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "operational_status_contract", "working|waiting|needs_rework|blocked|complete") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_status", topicRegistrySettlementTextV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_scope", "topic_text") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_reason", "topic_quality_contract_passed") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settled_refs", "opes-final-evidence:topic_quality_contract_pass") ||
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
		EvidenceRefs: []string{"opes-final-evidence:topic_quality_contract_pass"},
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-retry"},
			{Name: "topic_id", Value: "tema-001"},
			{Name: "source_work_kind", Value: "draft_content_block"},
			{Name: "canonical_word_count", Value: "12000"},
			{Name: "topic_quality_status", Value: "passed"},
			{Name: "topic_text", Value: "Contenido publico del tema con desarrollo normativo, ejemplos profesionales y explicacion didactica sin instrucciones internas."},
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
	followup, ok := requestedWorkKindForTestV0(result.RequestedJobs, "finalize_temario_package")
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "followup_ref", "final-package-manifest-closure-evidence-required") ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "expected_artifact_type", orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0) {
		t.Fatalf("followup=%+v ok=%v result=%+v", followup, ok, result)
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
	followup, ok := requestedWorkKindForTestV0(result.RequestedJobs, "finalize_temario_package")
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "followup_ref", "final-package-manifest-closure-evidence-required") ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "expected_artifact_type", orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0) {
		t.Fatalf("followup=%+v ok=%v result=%+v", followup, ok, result)
	}
}

func TestProduceOPESCausalJobsV0PaqueteFinalCompleteConManifestYQAGenericaNoLiberaRegistro(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-final-generic-qa-001",
		ArtifactRef:  "artifact-final-generic-qa-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0,
		ReceiptRef:   "receipt-final-generic-qa-001",
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
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "registry_action", "update") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", "pendiente_validacion_paquete_final") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", "final-package-manifest-closure-evidence-required") {
		t.Fatalf("request=%+v ok=%v result=%+v", request, ok, result)
	}
	followup, ok := requestedWorkKindForTestV0(result.RequestedJobs, "finalize_temario_package")
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "followup_ref", "final-package-manifest-closure-evidence-required") ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "expected_artifact_type", orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0) {
		t.Fatalf("followup=%+v ok=%v result=%+v", followup, ok, result)
	}
}

func TestProduceOPESCausalJobsV0PaqueteFinalSinBancoYTutorNoLiberaRegistroV0(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-final-sin-banco-tutor-001",
		ArtifactRef:  "artifact-final-sin-banco-tutor-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0,
		ReceiptRef:   "receipt-final-sin-banco-tutor-001",
		CompleteJob:  true,
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-final"},
			{Name: "topic_id", Value: "tema-004"},
			{Name: "manifest_cierre", ValueJSON: json.RawMessage(`{
				"schema_version":"opes_final_package_evidence_manifest.v0",
				"package_ref":"package-ref-final-001",
				"manifest_ref":"manifest-cierre-ref-final-001",
				"checksum_refs":["checksum-ref-final-001"],
				"validation_report_ref":"validation-report-ref-final-001",
				"review_matrix_ref":"review-matrix-ref-final-001",
				"qa_passes":{
					"extension_pass":true,
					"official_text_qa_pass":true,
					"strict_editorial_qa_pass":true
				},
				"qa_report_refs":{
					"extension":"09_validacion/informe_extension_temario.json",
					"official_text":"09_validacion/informe_texto_publico.json",
					"strict_editorial":"09_validacion/informe_editorial.json"
				},
				"required_evidence_refs":{
					"html":"opes-final-evidence:html:001",
					"rag":"opes-final-evidence:rag:001",
					"audio":"opes-final-evidence:audio:001",
					"tests":"opes-final-evidence:tests:001",
					"visual":"opes-final-evidence:visual:001",
					"qa":"opes-final-evidence:qa:001"
				}
			}`)},
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

func TestProduceOPESCausalJobsV0PaqueteFinalSinTopicQualityRefsNoLiberaRegistroV0(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-final-sin-topic-quality-001",
		ArtifactRef:  "artifact-final-sin-topic-quality-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0,
		ReceiptRef:   "receipt-final-sin-topic-quality-001",
		CompleteJob:  true,
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-final"},
			{Name: "topic_id", Value: "tema-004"},
			{Name: "manifest_cierre", ValueJSON: json.RawMessage(`{
				"schema_version":"opes_final_package_evidence_manifest.v0",
				"package_ref":"package-ref-final-001",
				"manifest_ref":"manifest-cierre-ref-final-001",
				"checksum_refs":["checksum-ref-final-001"],
				"validation_report_ref":"validation-report-ref-final-001",
				"review_matrix_ref":"review-matrix-ref-final-001",
				"qa_passes":{
					"extension_pass":true,
					"official_text_qa_pass":true,
					"strict_editorial_qa_pass":true,
					"question_bank_publicable":true,
					"tutor_assets_publicable":true
				},
				"qa_report_refs":{
					"extension":"09_validacion/informe_extension_temario.json",
					"official_text":[
						"09_validacion/informe_texto_publico_sin_notas_autor.json",
						"09_validacion/informe_texto_publico_sin_metacomentarios_examen.json"
					],
					"strict_editorial":"09_validacion/informe_texto_publico_sin_andamiaje_interno.json",
					"question_bank_publicable":"09_validacion/informe_question_bank_publicable.json",
					"tutor_assets_publicable":"09_validacion/informe_tutor_assets_publicable.json"
				},
				"required_evidence_refs":{
					"html":"opes-final-evidence:html:001",
					"rag":"opes-final-evidence:rag:001",
					"audio":"opes-final-evidence:audio:001",
					"tests":"opes-final-evidence:tests:001",
					"tutor":"opes-final-evidence:tutor:001",
					"visual":"opes-final-evidence:visual:001",
					"qa":"opes-final-evidence:qa:001"
				}
			}`)},
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
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", "final-package-manifest-closure-evidence-required") ||
		domainWorkFieldValueForDirectorTestV0(request.InputFields, "registry_action", "release") {
		t.Fatalf("request=%+v ok=%v result=%+v", request, ok, result)
	}
	followup, ok := requestedWorkKindForTestV0(result.RequestedJobs, "finalize_temario_package")
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "followup_ref", "final-package-manifest-closure-evidence-required") ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "expected_artifact_type", orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0) {
		t.Fatalf("followup=%+v ok=%v result=%+v", followup, ok, result)
	}
}

func TestProduceOPESCausalJobsV0PaqueteFinalCompleteConManifestCompatibleYQATernaLiberaRegistro(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-final-evidence-001",
		ArtifactRef:  "artifact-final-evidence-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0,
		ReceiptRef:   "receipt-final-evidence-001",
		CompleteJob:  true,
		EvidenceRefs: []string{"topic-quality-contract-result-ref-final-004"},
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-final"},
			{Name: "topic_id", Value: "tema-004"},
			{Name: "manifest_cierre", ValueJSON: json.RawMessage(`{
				"schema_version":"opes_final_package_evidence_manifest.v0",
				"package_ref":"package-ref-final-001",
				"manifest_ref":"manifest-cierre-ref-final-001",
				"checksum_refs":["checksum-ref-final-001"],
				"validation_report_ref":"validation-report-ref-final-001",
				"review_matrix_ref":"review-matrix-ref-final-001",
				"topic_quality_contract_result_refs":{
					"tema_004":"topic-quality-contract-result-ref-final-004"
				},
				"qa_passes":{
					"extension_pass":true,
					"official_text_qa_pass":true,
					"strict_editorial_qa_pass":true,
					"question_bank_publicable":true,
					"tutor_assets_publicable":true
				},
				"qa_report_refs":{
					"extension":"09_validacion/informe_extension_temario.json",
					"official_text":[
						"09_validacion/informe_texto_publico_sin_notas_autor.json",
						"09_validacion/informe_texto_publico_sin_metacomentarios_examen.json"
					],
					"strict_editorial":"09_validacion/informe_texto_publico_sin_andamiaje_interno.json",
					"question_bank_publicable":"09_validacion/informe_question_bank_publicable.json",
					"tutor_assets_publicable":"09_validacion/informe_tutor_assets_publicable.json"
				},
				"required_evidence_refs":{
					"html":"opes-final-evidence:html:001",
					"rag":"opes-final-evidence:rag:001",
					"audio":"opes-final-evidence:audio:001",
					"tests":"opes-final-evidence:tests:001",
					"tutor":"opes-final-evidence:tutor:001",
					"visual":"opes-final-evidence:visual:001",
					"qa":"opes-final-evidence:qa:001"
				}
			}`)},
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
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_status", topicRegistrySettlementFinalV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_scope", "final_package") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_reason", "final_package_closure_evidence_complete") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settled_refs", "topic-quality-contract-result-ref-final-004") ||
		domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", "final-package-manifest-closure-evidence-required") {
		t.Fatalf("request=%+v ok=%v result=%+v", request, ok, result)
	}
}

func TestProduceOPESCausalJobsV0GoalFirstTextoQAPassSinCheckpointNoAsientaTemaV0(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-topic-goal-first-no-checkpoint-001",
		ArtifactRef:  "artifact-topic-goal-first-no-checkpoint-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
		ReceiptRef:   "receipt-topic-goal-first-no-checkpoint-001",
		EvidenceRefs: []string{"evidence-ref-external-work-goal-first-result"},
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-grupo-b"},
			{Name: "topic_id", Value: "tema-032"},
			{Name: "source_work_kind", Value: "draft_content_block"},
			{Name: "director_execution_mode", Value: "goal_first"},
			{Name: "goal_first_heartbeat_refs", Values: []string{"heartbeat-ref-goal-first-tema-032"}},
			{Name: "status", Value: "ready"},
			{Name: "canonical_word_count", Value: "12000"},
			{Name: "topic_quality_status", Value: "passed"},
			{Name: "topic_text", Value: "Contenido publico del tema con desarrollo normativo, ejemplos profesionales y explicacion didactica sin instrucciones internas."},
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
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", "pendiente_continuar") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "operational_status", "waiting") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", topicRegistryGoalFirstCheckpointRequiredRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "goal_first_lifecycle_status", topicRegistryLifecycleHeartbeatOnlyV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "goal_first_heartbeat_refs", "heartbeat-ref-goal-first-tema-032") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_status", topicRegistrySettlementNotSettledV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_scope", "goal_first_lifecycle") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_reason", "goal_first_checkpoint_required") {
		t.Fatalf("request=%+v ok=%v result=%+v", request, ok, result)
	}
	followup, ok := requestedWorkKindForTestV0(result.RequestedJobs, "review_director_consolidation")
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "followup_ref", topicRegistryGoalFirstCheckpointRequiredRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "topic_id", "tema-032") {
		t.Fatalf("followup=%+v ok=%v result=%+v", followup, ok, result)
	}
}

func TestProduceOPESCausalJobsV0GoalFirstTextoQAPassConCheckpointAsientaTemaV0(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-topic-goal-first-checkpoint-001",
		ArtifactRef:  "artifact-topic-goal-first-checkpoint-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
		ReceiptRef:   "receipt-topic-goal-first-checkpoint-001",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-grupo-b"},
			{Name: "topic_id", Value: "tema-033"},
			{Name: "source_work_kind", Value: "draft_content_block"},
			{Name: "director_execution_mode", Value: "goal_first"},
			{Name: "goal_first_checkpoint_refs", Values: []string{"checkpoint-ref-goal-first-tema-033"}},
			{Name: "status", Value: "ready"},
			{Name: "canonical_word_count", Value: "12000"},
			{Name: "topic_quality_status", Value: "passed"},
			{Name: "topic_text", Value: "Contenido publico del tema con desarrollo normativo, ejemplos profesionales y explicacion didactica sin instrucciones internas."},
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
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", topicRegistryStatusTextSettledPendingDerivativesV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "operational_status", "waiting") ||
		domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", topicRegistryGoalFirstCheckpointRequiredRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "goal_first_lifecycle_status", topicRegistryLifecycleCheckpointRecordedV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "goal_first_checkpoint_refs", "checkpoint-ref-goal-first-tema-033") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_status", topicRegistrySettlementTextV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_reason", "topic_quality_contract_passed") {
		t.Fatalf("request=%+v ok=%v result=%+v", request, ok, result)
	}
	if _, ok := requestedWorkKindForTestV0(result.RequestedJobs, "review_director_consolidation"); ok {
		t.Fatalf("no debe crear followup de checkpoint con checkpoint durable: result=%+v", result)
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
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", partialStatus) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "operational_status", "waiting") {
		t.Fatalf("request=%+v ok=%v result=%+v", request, ok, result)
	}
}

func TestTopicRegistryOperationalStatusV0NormalizaAliasesCanonicosV0(t *testing.T) {
	cases := map[string]string{
		"working":                         "working",
		"en_progreso_orquesta":            "working",
		"waiting":                         "waiting",
		"pendiente_continuar":             "waiting",
		"needs_rework":                    "needs_rework",
		"pendiente_rework_editorial":      "needs_rework",
		"blocked":                         "blocked",
		"stale_lock_no_process":           "blocked",
		"complete":                        "complete",
		"paquete_final_local_verificable": "complete",
	}
	for status, want := range cases {
		t.Run(status, func(t *testing.T) {
			record := OPESCausalArtifactRecordV0{
				ArtifactType: orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
				PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
					{Name: "status", Value: status},
				},
			}
			if got := topicRegistryOperationalStatusForRecordV0(record); got != want {
				t.Fatalf("status=%q got=%q want=%q", status, got, want)
			}
		})
	}
}

func TestProduceOPESCausalJobsV0BloqueaRegistroPorQATemaFallidaV0(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-topic-quality-fails-001",
		ArtifactRef:  "artifact-topic-quality-fails-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
		ReceiptRef:   "receipt-topic-quality-fails-001",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-grupo-b"},
			{Name: "topic_id", Value: "tema-029"},
			{Name: "source_work_kind", Value: "draft_content_block"},
			{Name: "status", Value: "ready"},
			{Name: "canonical_word_count", Value: "12000"},
			{Name: "topic_quality_status", Value: "passed"},
			{Name: "topic_text", Value: "Preguntas De Recuperacion\nRepaso Espaciado\nLa respuesta debe empezar con una definicion."},
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
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", "pendiente_rework_editorial") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", topicRegistryQualityNeedsReworkRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", "topic-quality-"+safeRefV0(ErrOPESTopicQualityStudyScaffoldingV0)) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "topic_quality_status", OPESTopicQualityStatusNeedsReworkV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "topic_quality_issue_refs", ErrOPESTopicQualityStudyScaffoldingV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_status", topicRegistrySettlementNeedsReworkV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_reason", "topic_quality_contract_failed") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "next_required_work_kinds", "review_director_consolidation") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "operational_status", "needs_rework") {
		t.Fatalf("request=%+v ok=%v result=%+v", request, ok, result)
	}
	followup, ok := requestedWorkKindForTestV0(result.RequestedJobs, "review_director_consolidation")
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "followup_ref", topicRegistryQualityNeedsReworkRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "topic_id", "tema-029") {
		t.Fatalf("followup=%+v ok=%v result=%+v", followup, ok, result)
	}
}

func TestProduceOPESCausalJobsV0BloqueaRegistroPorMojibakePreAudioV0(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-topic-quality-mojibake-001",
		ArtifactRef:  "artifact-topic-quality-mojibake-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
		ReceiptRef:   "receipt-topic-quality-mojibake-001",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-integrador-social"},
			{Name: "topic_id", Value: "tema-audio-001"},
			{Name: "source_work_kind", Value: "draft_content_block"},
			{Name: "status", Value: "ready"},
			{Name: "canonical_word_count", Value: "12000"},
			{Name: "topic_quality_status", Value: "passed"},
			{Name: "topic_text", Value: "Texto p\u00c3\u00bablico con designaci\u00c3\u00b3n corrupta antes de generar audio."},
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
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", "pendiente_rework_editorial") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", topicRegistryQualityNeedsReworkRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", "topic-quality-"+safeRefV0(ErrOPESTopicQualityPublicMojibakeV0)) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "topic_quality_issue_refs", ErrOPESTopicQualityPublicMojibakeV0) {
		t.Fatalf("request=%+v ok=%v result=%+v", request, ok, result)
	}
	followup, ok := requestedWorkKindForTestV0(result.RequestedJobs, "review_director_consolidation")
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "followup_ref", topicRegistryQualityNeedsReworkRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "topic_id", "tema-audio-001") {
		t.Fatalf("followup=%+v ok=%v result=%+v", followup, ok, result)
	}
}

func TestProduceOPESCausalJobsV0BloqueaRegistroPorContaminacionEstructuralV0(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-topic-quality-structural-fails-001",
		ArtifactRef:  "artifact-topic-quality-structural-fails-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
		ReceiptRef:   "receipt-topic-quality-structural-fails-001",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-grupo-b"},
			{Name: "topic_id", Value: "tema-046"},
			{Name: "source_work_kind", Value: "draft_content_block"},
			{Name: "status", Value: "ready"},
			{Name: "canonical_word_count", Value: "12000"},
			{Name: "topic_quality_status", Value: "passed"},
			{Name: "topic_text", Value: "Bloque ajeno del tema_017 con referencia interna a canon maestro y figura .svg visible."},
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
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", "pendiente_rework_editorial") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", topicRegistryQualityNeedsReworkRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", "topic-quality-"+safeRefV0(ErrOPESTopicQualityStructuralContaminationV0)) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "topic_quality_issue_refs", ErrOPESTopicQualityStructuralContaminationV0) {
		t.Fatalf("request=%+v ok=%v result=%+v", request, ok, result)
	}
	followup, ok := requestedWorkKindForTestV0(result.RequestedJobs, "review_director_consolidation")
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "followup_ref", topicRegistryQualityNeedsReworkRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "topic_id", "tema-046") {
		t.Fatalf("followup=%+v ok=%v result=%+v", followup, ok, result)
	}
}

func TestProduceOPESCausalJobsV0NoBloqueaRegistroConQATemaCompletaV0(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-topic-quality-pass-001",
		ArtifactRef:  "artifact-topic-quality-pass-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
		ReceiptRef:   "receipt-topic-quality-pass-001",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-grupo-b"},
			{Name: "topic_id", Value: "tema-030"},
			{Name: "source_work_kind", Value: "draft_content_block"},
			{Name: "status", Value: "ready"},
			{Name: "canonical_word_count", Value: "12000"},
			{Name: "topic_quality_status", Value: "passed"},
			{Name: "topic_text", Value: "Contenido publico del tema con desarrollo normativo, ejemplos profesionales y explicacion didactica sin instrucciones internas."},
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
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", topicRegistryStatusTextSettledPendingDerivativesV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "operational_status", "waiting") ||
		domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", topicRegistryQualityNeedsReworkRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "topic_quality_status", OPESTopicQualityStatusCompleteV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_status", topicRegistrySettlementTextV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_scope", "topic_text") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_reason", "topic_quality_contract_passed") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settled_refs", "artifact-topic-quality-pass-001") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "next_required_work_kinds", "generate_audio_asset") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "next_required_work_kinds", "finalize_temario_package") {
		t.Fatalf("request=%+v ok=%v result=%+v", request, ok, result)
	}
	if _, ok := requestedWorkKindForTestV0(result.RequestedJobs, "review_director_consolidation"); ok {
		t.Fatalf("no debe crear followup de rework: result=%+v", result)
	}
}

func TestProduceOPESCausalJobsV0NoBloqueaRegistroConVisualDidacticoDeclaradoV0(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		Status:       "accepted",
		DomainRef:    "opes",
		JobRef:       "job-topic-quality-visual-pass-001",
		ArtifactRef:  "artifact-topic-quality-visual-pass-001",
		ArtifactType: orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
		ReceiptRef:   "receipt-topic-quality-visual-pass-001",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-grupo-b"},
			{Name: "topic_id", Value: "tema-031"},
			{Name: "source_work_kind", Value: "draft_content_block"},
			{Name: "status", Value: "ready"},
			{Name: "canonical_word_count", Value: "12000"},
			{Name: "topic_quality_status", Value: "passed"},
			{Name: "require_didactic_visual", Value: "true"},
			{Name: "topic_text", Value: "Contenido publico del tema con desarrollo normativo y explicacion didactica sin instrucciones internas."},
			{Name: "visuals", ValueJSON: json.RawMessage(`[
				{
					"visual_ref":"visual-ref-flujo-tramite",
					"raster":true,
					"anchor_ref":"apartado-ref-flujo",
					"didactic_function":"diagrama de pasos, responsables y evidencias evaluables",
					"evidence_refs":["evidence-ref-visual-reviewed"]
				}
			]`)},
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
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", topicRegistryStatusTextSettledPendingDerivativesV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "operational_status", "waiting") ||
		domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", topicRegistryQualityNeedsReworkRefV0) ||
		domainWorkFieldValueForDirectorTestV0(request.InputFields, "topic_quality_issue_refs", ErrOPESTopicQualityDidacticVisualRequiredV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "topic_quality_status", OPESTopicQualityStatusCompleteV0) {
		t.Fatalf("request=%+v ok=%v result=%+v", request, ok, result)
	}
	if _, ok := requestedWorkKindForTestV0(result.RequestedJobs, "review_director_consolidation"); ok {
		t.Fatalf("no debe crear followup de rework: result=%+v", result)
	}
}

func TestProduceOPESCausalJobsV0BloqueaDerivadoOPESSinEvidenciaMinimaV0(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		IdempotencyKey: "idem-opes-html-no-evidence",
		Status:         "accepted",
		CorrelationID:  "corr-opes-html-no-evidence",
		DomainRef:      OPESCausalProducerDefaultDomainRefV0,
		JobRef:         "job-ref-opes-html-no-evidence",
		ArtifactRef:    "artifact-ref-opes-html-no-evidence",
		ArtifactType:   orquestadomainwork.DomainWorkArtifactTypeLocalHTMLSiteV0,
		CompleteJob:    true,
		ReceiptRef:     "receipt-ref-opes-html-no-evidence",
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-html-no-evidence"},
			{Name: "topic_id", Value: "tema-html-no-evidence"},
			{Name: "source_work_kind", Value: "generate_html_site"},
			{Name: "status", Value: "complete"},
		},
	}}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{
		DomainRef:     OPESCausalProducerDefaultDomainRefV0,
		CorrelationID: "corr-opes-html-no-evidence",
	}, OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator})
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	request, ok := requestedWorkKindForTestV0(result.RequestedJobs, opesTopicRegistryUpdateWorkKindV0)
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", "pendiente_rework_evidencia_minima") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "operational_status", "needs_rework") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", "required-evidence-html-site-publicable") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_status", topicRegistrySettlementNeedsReworkV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_scope", "required_evidence") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_reason", "required_evidence_missing") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "next_required_work_kinds", "review_director_consolidation") {
		t.Fatalf("registry_update=%+v ok=%v result=%+v", request, ok, result)
	}
	followup, ok := requestedWorkKindForTestV0(result.RequestedJobs, "review_director_consolidation")
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "followup_ref", "required-evidence-html-site-publicable") ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "rework_reason", "required_evidence_missing") ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "required_evidence_missing_refs", "required-evidence-html-site-publicable") ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "publication_status", "not_publicable_without_required_evidence") ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "recommended_action", "review_required_evidence") ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "topic_id", "tema-html-no-evidence") {
		t.Fatalf("followup=%+v ok=%v result=%+v", followup, ok, result)
	}
}

func TestProduceOPESCausalJobsV0BloqueaSecuenciaOPESCompletaSinEvidenciaMinimaV0(t *testing.T) {
	for _, workKind := range orquestaopesbridge.OPESFullTemarioJobTypeSequenceV0() {
		if workKind == orquestadomainwork.DomainWorkKindPlanSyllabusV0 ||
			workKind == "finalize_temario_package" ||
			workKind == opesTopicRegistryUpdateWorkKindV0 {
			continue
		}
		t.Run(workKind, func(t *testing.T) {
			artifactType := opesSequenceArtifactTypeForEvidenceGateTestV0(workKind)
			requirements := topicRegistryRequiredEvidenceRequirementsV0(workKind, artifactType)
			if len(requirements) == 0 {
				t.Fatalf("work_kind=%s artifact=%s sin requisito de evidencia minima", workKind, artifactType)
			}
			pendingRef := requirements[0].PendingRef
			ref := safeRefV0(workKind)
			source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
				IdempotencyKey: "idem-opes-required-evidence-" + ref,
				Status:         "accepted",
				CorrelationID:  "corr-opes-required-evidence-" + ref,
				DomainRef:      OPESCausalProducerDefaultDomainRefV0,
				JobRef:         "job-ref-opes-required-evidence-" + ref,
				ArtifactRef:    "artifact-ref-opes-required-evidence-" + ref,
				ArtifactType:   artifactType,
				CompleteJob:    true,
				ReceiptRef:     "receipt-ref-opes-required-evidence-" + ref,
				PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
					{Name: "course_id", Value: "curso-required-evidence"},
					{Name: "topic_id", Value: "tema-" + ref},
					{Name: "source_work_kind", Value: workKind},
					{Name: "status", Value: "complete"},
				},
			}}}
			creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
			result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{
				DomainRef:     OPESCausalProducerDefaultDomainRefV0,
				CorrelationID: "corr-opes-required-evidence-" + ref,
			}, OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator})
			if err != nil {
				t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
			}
			request, ok := requestedWorkKindForTestV0(result.RequestedJobs, opesTopicRegistryUpdateWorkKindV0)
			if !ok ||
				!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", "pendiente_rework_evidencia_minima") ||
				!domainWorkFieldValueForDirectorTestV0(request.InputFields, "operational_status", "needs_rework") ||
				!domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", pendingRef) ||
				!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_status", topicRegistrySettlementNeedsReworkV0) ||
				!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_scope", "required_evidence") ||
				!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_reason", "required_evidence_missing") ||
				!domainWorkFieldValueForDirectorTestV0(request.InputFields, "next_required_work_kinds", "review_director_consolidation") {
				t.Fatalf("registry_update=%+v ok=%v result=%+v", request, ok, result)
			}
			followup, ok := requestedWorkKindForTestV0(result.RequestedJobs, "review_director_consolidation")
			if !ok ||
				!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "followup_ref", pendingRef) ||
				!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "rework_reason", "required_evidence_missing") ||
				!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "required_evidence_missing_refs", pendingRef) ||
				!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "publication_status", "not_publicable_without_required_evidence") ||
				!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "recommended_action", "review_required_evidence") ||
				!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "topic_id", "tema-"+ref) {
				t.Fatalf("followup=%+v ok=%v result=%+v", followup, ok, result)
			}
		})
	}
}

func TestProduceOPESCausalJobsV0DerivadoOPESConEvidenciaMinimaNoCreaReworkV0(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		IdempotencyKey: "idem-opes-html-with-evidence",
		Status:         "accepted",
		CorrelationID:  "corr-opes-html-with-evidence",
		DomainRef:      OPESCausalProducerDefaultDomainRefV0,
		JobRef:         "job-ref-opes-html-with-evidence",
		ArtifactRef:    "artifact-ref-opes-html-with-evidence",
		ArtifactType:   orquestadomainwork.DomainWorkArtifactTypeLocalHTMLSiteV0,
		CompleteJob:    true,
		ReceiptRef:     "receipt-ref-opes-html-with-evidence",
		EvidenceRefs:   []string{"opes-final-evidence:html_site_publicable"},
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-html-with-evidence"},
			{Name: "topic_id", Value: "tema-html-with-evidence"},
			{Name: "source_work_kind", Value: "generate_html_site"},
			{Name: "status", Value: "complete"},
			{Name: "html_topic_pages_manifest", Value: "html/topic_pages_manifest.json"},
			{Name: "html_validation_report", Value: "validacion/html_links.json"},
		},
	}}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{
		DomainRef:     OPESCausalProducerDefaultDomainRefV0,
		CorrelationID: "corr-opes-html-with-evidence",
	}, OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator})
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	request, ok := requestedWorkKindForTestV0(result.RequestedJobs, opesTopicRegistryUpdateWorkKindV0)
	if !ok ||
		domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", "required-evidence-html-site-publicable") ||
		domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", topicRegistryArtifactQualityNeedsReworkRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "artifact_quality_status", OPESArtifactQualityStatusCompleteV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_reason", "artifact_phase_not_terminal") {
		t.Fatalf("registry_update=%+v ok=%v result=%+v", request, ok, result)
	}
	if _, ok := requestedWorkKindForTestV0(result.RequestedJobs, "review_director_consolidation"); ok {
		t.Fatalf("no debe crear rework con evidencia minima: result=%+v", result)
	}
}

func TestProduceOPESCausalJobsV0QuestionBankConEvidenceRefPeroContratoFallidoCreaReworkV0(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		IdempotencyKey: "idem-opes-question-bank-bad-contract",
		Status:         "accepted",
		CorrelationID:  "corr-opes-question-bank-bad-contract",
		DomainRef:      OPESCausalProducerDefaultDomainRefV0,
		JobRef:         "job-ref-opes-question-bank-bad-contract",
		ArtifactRef:    "artifact-ref-opes-question-bank-bad-contract",
		ArtifactType:   orquestadomainwork.DomainWorkArtifactTypeQuestionBankV0,
		CompleteJob:    true,
		ReceiptRef:     "receipt-ref-opes-question-bank-bad-contract",
		EvidenceRefs:   []string{"opes-final-evidence:question_bank_publicable"},
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-question-bank"},
			{Name: "topic_id", Value: "tema-question-bank-bad"},
			{Name: "source_work_kind", Value: "generate_question_bank"},
			{Name: "status", Value: "complete"},
			{Name: "question_bank", ValueJSON: json.RawMessage(`{
				"questions":[{
					"stem":"Pregunta pobre",
					"options":["A","B"],
					"correct_answers":["A","B"],
					"explanation":"breve"
				}]
			}`)},
		},
	}}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{
		DomainRef:     OPESCausalProducerDefaultDomainRefV0,
		CorrelationID: "corr-opes-question-bank-bad-contract",
	}, OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator})
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	request, ok := requestedWorkKindForTestV0(result.RequestedJobs, opesTopicRegistryUpdateWorkKindV0)
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", "pendiente_rework_tests") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "operational_status", "needs_rework") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", topicRegistryQuestionBankQualityNeedsReworkRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "question_bank_quality_status", OPESQuestionBankQualityStatusNeedsReworkV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "question_bank_quality_issue_refs", ErrOPESQuestionBankQuestionCountBelowMinV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "question_bank_quality_issue_refs", ErrOPESQuestionBankOptionsInvalidV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "question_bank_quality_issue_refs", ErrOPESQuestionBankCorrectAnswerInvalidV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_status", topicRegistrySettlementNeedsReworkV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_scope", "question_bank_quality") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_reason", "question_bank_quality_contract_failed") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "next_required_work_kinds", "review_director_consolidation") {
		t.Fatalf("registry_update=%+v ok=%v result=%+v", request, ok, result)
	}
	followup, ok := requestedWorkKindForTestV0(result.RequestedJobs, "review_director_consolidation")
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "followup_ref", topicRegistryQuestionBankQualityNeedsReworkRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "rework_reason", "question_bank_quality_contract_failed") ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "question_bank_quality_missing_refs", topicRegistryQuestionBankQualityNeedsReworkRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "publication_status", "not_publicable_question_bank_quality_failed") ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "recommended_action", "review_question_bank_quality") ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "topic_id", "tema-question-bank-bad") {
		t.Fatalf("followup=%+v ok=%v result=%+v", followup, ok, result)
	}
}

func TestProduceOPESCausalJobsV0QuestionBankConContratoPassNoCreaReworkV0(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		IdempotencyKey: "idem-opes-question-bank-pass-contract",
		Status:         "accepted",
		CorrelationID:  "corr-opes-question-bank-pass-contract",
		DomainRef:      OPESCausalProducerDefaultDomainRefV0,
		JobRef:         "job-ref-opes-question-bank-pass-contract",
		ArtifactRef:    "artifact-ref-opes-question-bank-pass-contract",
		ArtifactType:   orquestadomainwork.DomainWorkArtifactTypeQuestionBankV0,
		CompleteJob:    true,
		ReceiptRef:     "receipt-ref-opes-question-bank-pass-contract",
		EvidenceRefs: []string{
			"opes-final-evidence:question_bank_publicable",
			"question_bank_structural_report",
			"question_bank_difficulty_report",
			"question_bank_three_model_review",
		},
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-question-bank"},
			{Name: "topic_id", Value: "tema-question-bank-pass"},
			{Name: "source_work_kind", Value: "generate_question_bank"},
			{Name: "status", Value: "complete"},
			{Name: "question_bank", ValueJSON: validQuestionBankRawForDirectorTestV0(t, 50)},
		},
	}}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{
		DomainRef:     OPESCausalProducerDefaultDomainRefV0,
		CorrelationID: "corr-opes-question-bank-pass-contract",
	}, OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator})
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	request, ok := requestedWorkKindForTestV0(result.RequestedJobs, opesTopicRegistryUpdateWorkKindV0)
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "question_bank_quality_status", OPESQuestionBankQualityStatusCompleteV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "question_bank_question_count", "50") ||
		domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", topicRegistryQuestionBankQualityNeedsReworkRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_scope", "question_bank") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_reason", "question_bank_quality_contract_passed") {
		t.Fatalf("registry_update=%+v ok=%v result=%+v", request, ok, result)
	}
	if _, ok := requestedWorkKindForTestV0(result.RequestedJobs, "review_director_consolidation"); ok {
		t.Fatalf("no debe crear rework con banco publicable: result=%+v", result)
	}
}

func TestTopicRegistryRequiredEvidencePolicyV0CubreSecuenciaOPESCompletaV0(t *testing.T) {
	for _, workKind := range orquestaopesbridge.OPESFullTemarioJobTypeSequenceV0() {
		t.Run(workKind, func(t *testing.T) {
			if workKind == "finalize_temario_package" || workKind == opesTopicRegistryUpdateWorkKindV0 {
				return
			}
			artifactType := opesSequenceArtifactTypeForEvidenceGateTestV0(workKind)
			record := OPESCausalArtifactRecordV0{
				ArtifactType: artifactType,
				PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
					{Name: "source_work_kind", Value: workKind},
				},
			}
			if len(topicRegistryRequiredEvidenceRequirementsV0(workKind, artifactType)) == 0 &&
				len(topicRegistryRequiredEvidencePendingRefsForRecordV0(record)) == 0 {
				t.Fatalf("work_kind=%s artifact=%s sin gate de evidencia minima", workKind, artifactType)
			}
		})
	}
}

func TestProduceOPESCausalJobsV0VisualConEvidenceRefPeroContratoFallidoCreaReworkV0(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		IdempotencyKey: "idem-opes-visual-bad-contract",
		Status:         "accepted",
		CorrelationID:  "corr-opes-visual-bad-contract",
		DomainRef:      OPESCausalProducerDefaultDomainRefV0,
		JobRef:         "job-ref-opes-visual-bad-contract",
		ArtifactRef:    "artifact-ref-opes-visual-bad-contract",
		ArtifactType:   orquestadomainwork.DomainWorkArtifactTypeVisualAssetV0,
		CompleteJob:    true,
		ReceiptRef:     "receipt-ref-opes-visual-bad-contract",
		EvidenceRefs:   []string{"opes-final-evidence:visual_didactic_publicable"},
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-visual"},
			{Name: "topic_id", Value: "tema-visual-bad"},
			{Name: "source_work_kind", Value: "generate_visual_asset"},
			{Name: "status", Value: "complete"},
		},
	}}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{
		DomainRef:     OPESCausalProducerDefaultDomainRefV0,
		CorrelationID: "corr-opes-visual-bad-contract",
	}, OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator})
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	request, ok := requestedWorkKindForTestV0(result.RequestedJobs, opesTopicRegistryUpdateWorkKindV0)
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "proposed_status", "pendiente_rework_artifact_quality") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "operational_status", "needs_rework") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", topicRegistryArtifactQualityNeedsReworkRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "artifact_quality_status", OPESArtifactQualityStatusNeedsReworkV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "artifact_quality_issue_refs", ErrOPESArtifactQualityVisualDidacticFunctionRequiredV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "artifact_quality_issue_refs", ErrOPESArtifactQualityVisualPlacementRequiredV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_status", topicRegistrySettlementNeedsReworkV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_scope", "artifact_quality") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_reason", "artifact_quality_contract_failed") ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "next_required_work_kinds", "review_director_consolidation") {
		t.Fatalf("registry_update=%+v ok=%v result=%+v", request, ok, result)
	}
	followup, ok := requestedWorkKindForTestV0(result.RequestedJobs, "review_director_consolidation")
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "followup_ref", topicRegistryArtifactQualityNeedsReworkRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "rework_reason", "artifact_quality_contract_failed") ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "artifact_quality_missing_refs", topicRegistryArtifactQualityNeedsReworkRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "artifact_quality_issue_refs", ErrOPESArtifactQualityVisualAltTextRequiredV0) ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "publication_status", "not_publicable_artifact_quality_failed") ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "recommended_action", "review_artifact_quality") ||
		!domainWorkFieldValueForDirectorTestV0(followup.InputFields, "topic_id", "tema-visual-bad") {
		t.Fatalf("followup=%+v ok=%v result=%+v", followup, ok, result)
	}
}

func TestProduceOPESCausalJobsV0HTMLConContratoArtifactQualityPassNoCreaReworkV0(t *testing.T) {
	source := fakeArtifactSourceV0{records: []OPESCausalArtifactRecordV0{{
		IdempotencyKey: "idem-opes-html-good-contract",
		Status:         "accepted",
		CorrelationID:  "corr-opes-html-good-contract",
		DomainRef:      OPESCausalProducerDefaultDomainRefV0,
		JobRef:         "job-ref-opes-html-good-contract",
		ArtifactRef:    "artifact-ref-opes-html-good-contract",
		ArtifactType:   orquestadomainwork.DomainWorkArtifactTypeLocalHTMLSiteV0,
		CompleteJob:    true,
		ReceiptRef:     "receipt-ref-opes-html-good-contract",
		EvidenceRefs:   []string{"opes-final-evidence:html_site_publicable"},
		PayloadFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "course_id", Value: "curso-html"},
			{Name: "topic_id", Value: "tema-html-good"},
			{Name: "source_work_kind", Value: "generate_html_site"},
			{Name: "status", Value: "complete"},
			{Name: "html_topic_pages_manifest", Value: "html/topic_pages_manifest.json"},
			{Name: "html_validation_report", Value: "validacion/html_links.json"},
		},
	}}}
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	result, err := ProduceOPESCausalJobsV0(context.Background(), OPESCausalProducerRequestV0{
		DomainRef:     OPESCausalProducerDefaultDomainRefV0,
		CorrelationID: "corr-opes-html-good-contract",
	}, OPESCausalProducerPortsV0{ArtifactSource: source, JobCreator: creator})
	if err != nil {
		t.Fatalf("ProduceOPESCausalJobsV0: %v", err)
	}
	request, ok := requestedWorkKindForTestV0(result.RequestedJobs, opesTopicRegistryUpdateWorkKindV0)
	if !ok ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "artifact_quality_status", OPESArtifactQualityStatusCompleteV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "artifact_quality_issue_count", "0") ||
		domainWorkFieldValueForDirectorTestV0(request.InputFields, "pending_refs", topicRegistryArtifactQualityNeedsReworkRefV0) ||
		!domainWorkFieldValueForDirectorTestV0(request.InputFields, "settlement_reason", "artifact_phase_not_terminal") {
		t.Fatalf("registry_update=%+v ok=%v result=%+v", request, ok, result)
	}
	if _, ok := requestedWorkKindForTestV0(result.RequestedJobs, "review_director_consolidation"); ok {
		t.Fatalf("no debe crear rework con HTML estructurado: result=%+v", result)
	}
}

func opesSequenceArtifactTypeForEvidenceGateTestV0(workKind string) string {
	switch workKind {
	case "review_codex", "review_gemini", "review_claude":
		return orquestadomainwork.DomainWorkArtifactTypeAgentReviewReportV0
	case "review_pair_codex_gemini", "review_pair_codex_claude", "review_pair_gemini_claude":
		return orquestadomainwork.DomainWorkArtifactTypeAgentPairReviewReportV0
	case "review_director_consolidation":
		return orquestadomainwork.DomainWorkArtifactTypeDirectorReviewMatrixV0
	case "generate_learning_games":
		return "learning_games_package"
	case "generate_help_manual_assets":
		return "help_manual_package"
	case "visual_asset_reuse":
		return "visual_reuse_manifest"
	default:
		return orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(workKind)
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

func validQuestionBankRawForDirectorTestV0(t *testing.T, count int) json.RawMessage {
	t.Helper()
	questions := make([]map[string]any, 0, count)
	for i := 0; i < count; i++ {
		questions = append(questions, map[string]any{
			"stem": "Pregunta publicable sobre contenido canonico del tema",
			"options": []string{
				"Respuesta correcta desarrollada",
				"Distractor plausible relacionado",
				"Distractor parcial razonable",
				"Distractor tecnico alternativo",
			},
			"correct_answer": "A",
			"explanation":    "Explicacion tutor para aciertos y fallos",
		})
	}
	raw, err := json.Marshal(map[string]any{"questions": questions})
	if err != nil {
		t.Fatalf("marshal question bank fixture: %v", err)
	}
	return raw
}
