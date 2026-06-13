package orquestaopesdirector

import (
	"context"
	"encoding/json"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestadomainworkmemory "orquesta/modulos/orquesta-domain-work-memory"
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
