package orquestadomainwork

import (
	"context"
	"testing"
)

func TestDomainWorkJobRequestV0NormalizaContratoExterno(t *testing.T) {
	request := NormalizeDomainWorkJobRequestV0(DomainWorkJobRequestV0{
		RequestID:     " req-domain-work-001 ",
		DomainRef:     " opes ",
		InterfaceRefs: []string{" mcp-contract-ref-opes-v0 ", "mcp-contract-ref-opes-v0"},
		WorkKind:      " draft_content_block ",
		Objective:     " Crear bloque editorial verificable. ",
		InputFields: []DomainWorkFieldV0{
			{Name: " level ", Value: " A1/A2 "},
			{Name: " source_refs ", Values: []string{" BOE-A-1 ", "BOE-A-1"}},
			{Name: " block_position ", ValueJSON: []byte(`{"block_order":2,"chapter_order":1}`)},
		},
		ExternalRefs: []DomainWorkExternalRefV0{
			{Kind: " run_ref ", Ref: " run-ref-001 "},
			{Kind: " run_ref ", Ref: " run-ref-001 "},
			{Kind: " ", Ref: "ignored"},
		},
	})

	if request.SchemaVersion != DomainWorkJobRequestSchemaV0 ||
		request.RequestedBy != DomainWorkDefaultRequestedByV0 ||
		request.DomainRef != "opes" ||
		request.WorkKind != "draft_content_block" ||
		request.CorrelationID != "req-domain-work-001" ||
		request.IdempotencyKey != "req-domain-work-001" ||
		len(request.InterfaceRefs) != 1 ||
		len(request.InputFields) != 3 ||
		request.InputFields[0].Value != "A1/A2" ||
		len(request.InputFields[1].Values) != 1 ||
		string(request.InputFields[2].ValueJSON) != `{"block_order":2,"chapter_order":1}` ||
		len(request.ExternalRefs) != 1 {
		t.Fatalf("request=%+v", request)
	}
	if issues := ValidateDomainWorkJobRequestV0(request); len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestDomainWorkJobRequestV0RechazaRefsNoCompactas(t *testing.T) {
	request := validDomainWorkJobRequestForTestV0()
	request.WorkRefs = []string{"ruta/local/no-permitida"}

	issues := ValidateDomainWorkJobRequestV0(request)

	if len(issues) != 1 ||
		issues[0].Code != ErrDomainWorkRefInvalidV0 ||
		issues[0].Field != "work_refs" {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestDomainWorkJobRequestV0RechazaJSONInvalido(t *testing.T) {
	request := validDomainWorkJobRequestForTestV0()
	request.InputFields = []DomainWorkFieldV0{
		{Name: "block_position", ValueJSON: []byte(`{"broken"`)},
	}

	issues := ValidateDomainWorkJobRequestV0(request)

	if len(issues) == 0 || issues[0].Code != ErrDomainWorkFieldJSONInvalidV0 {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestDomainWorkArtifactSubmissionV0ValidaEntrega(t *testing.T) {
	submission := validDomainWorkArtifactSubmissionForTestV0()

	if issues := ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}

	submission.ArtifactRef = ""
	issues := ValidateDomainWorkArtifactSubmissionV0(submission)
	if len(issues) == 0 || issues[0].Code != ErrDomainWorkArtifactRefRequiredV0 {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestDomainWorkJobRecordFilterV0NormalizaContratoDeLectura(t *testing.T) {
	filter := NormalizeDomainWorkJobRecordFilterV0(DomainWorkJobRecordFilterV0{
		DomainRef:      " dominio-demo ",
		WorkKind:       " generate_content_package ",
		JobRef:         " job-ref-001 ",
		CorrelationID:  " corr-001 ",
		IdempotencyKey: " idem-001 ",
		Status:         " accepted ",
		ExternalRefs: []DomainWorkExternalRefV0{
			{Kind: " run_ref ", Ref: " run-ref-001 "},
			{Kind: "run_ref", Ref: "run-ref-001"},
			{Kind: "", Ref: "ignored"},
		},
		Limit: -1,
	})

	if filter.DomainRef != "dominio-demo" ||
		filter.WorkKind != "generate_content_package" ||
		filter.JobRef != "job-ref-001" ||
		filter.CorrelationID != "corr-001" ||
		filter.IdempotencyKey != "idem-001" ||
		filter.Status != DomainWorkStatusAcceptedV0 ||
		len(filter.ExternalRefs) != 1 ||
		filter.ExternalRefs[0].Kind != "run_ref" ||
		filter.ExternalRefs[0].Ref != "run-ref-001" ||
		filter.Limit != 0 {
		t.Fatalf("filter=%+v", filter)
	}
}

func TestDomainWorkPortsV0SonInterfacesHexagonales(t *testing.T) {
	creator := fakeDomainWorkConnectorV0{}
	submitter := fakeDomainWorkConnectorV0{}
	var _ DomainWorkJobCreatorPortV0 = creator
	var _ DomainWorkJobRecordSourcePortV0 = creator
	var _ DomainWorkJobRecordStorePortV0 = creator
	var _ DomainWorkArtifactSubmitterPortV0 = submitter
}

func validDomainWorkJobRequestForTestV0() DomainWorkJobRequestV0 {
	return NormalizeDomainWorkJobRequestV0(DomainWorkJobRequestV0{
		RequestID:     "req-domain-work-001",
		DomainRef:     "opes",
		InterfaceRefs: []string{"mcp-contract-ref-opes-v0"},
		WorkKind:      "draft_content_block",
		WorkRefs:      []string{"topic-ref-001", "chapter-ref-001"},
		Objective:     "Crear bloque editorial verificable.",
		InputFields: []DomainWorkFieldV0{
			{Name: "program_id", Value: "program-ref-001"},
			{Name: "language_code", Value: "es"},
			{Name: "source_refs", Values: []string{"boe-ref-001"}},
		},
		ExternalRefs: []DomainWorkExternalRefV0{
			{Kind: "run_ref", Ref: "run-ref-001"},
			{Kind: "task_ref", Ref: "task-ref-001"},
		},
	})
}

func validDomainWorkArtifactSubmissionForTestV0() DomainWorkArtifactSubmissionV0 {
	return NormalizeDomainWorkArtifactSubmissionV0(DomainWorkArtifactSubmissionV0{
		RequestID:    "req-domain-artifact-001",
		DomainRef:    "opes",
		JobRef:       "job-ref-001",
		ArtifactRef:  "artifact-ref-001",
		ArtifactType: "content_block",
		Summary:      "Bloque editorial listo para revision.",
		PayloadFields: []DomainWorkFieldV0{
			{Name: "title", Value: "Bloque 1"},
			{Name: "body", Value: "Contenido producido por Orquesta."},
		},
		PayloadRefs: []string{"payload-ref-001"},
		ExternalRefs: []DomainWorkExternalRefV0{
			{Kind: "run_ref", Ref: "run-ref-001"},
			{Kind: "task_ref", Ref: "task-ref-001"},
		},
		CompleteJob: true,
	})
}

type fakeDomainWorkConnectorV0 struct{}

func (fakeDomainWorkConnectorV0) CreateDomainWorkJobV0(
	context.Context,
	DomainWorkJobRequestV0,
) (DomainWorkJobV0, error) {
	return DomainWorkJobV0{Status: DomainWorkStatusAcceptedV0}, nil
}

func (fakeDomainWorkConnectorV0) SubmitDomainWorkArtifactV0(
	context.Context,
	DomainWorkArtifactSubmissionV0,
) (DomainWorkArtifactReceiptV0, error) {
	return DomainWorkArtifactReceiptV0{Status: DomainWorkStatusAcceptedV0}, nil
}

func (fakeDomainWorkConnectorV0) ListDomainWorkJobRecordsV0(
	context.Context,
	DomainWorkJobRecordFilterV0,
) ([]DomainWorkJobRecordV0, error) {
	return []DomainWorkJobRecordV0{{
		Request: validDomainWorkJobRequestForTestV0(),
		Job:     DomainWorkJobV0{Status: DomainWorkStatusAcceptedV0},
	}}, nil
}
