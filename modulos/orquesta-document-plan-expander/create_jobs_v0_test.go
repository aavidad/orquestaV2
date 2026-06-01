package orquestadocumentplanexpander

import (
	"context"
	"errors"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestCreateDomainDocumentPlanDerivedJobsV0CreaJobsPorPuerto(t *testing.T) {
	creator := &fakeDocumentPlanJobCreatorV0{}
	result, err := CreateDomainDocumentPlanDerivedJobsV0(
		context.Background(),
		DomainDocumentPlanExpansionRequestV0{
			Plan:          validDocumentPlanForExpanderTestV0(),
			CorrelationID: "corr-docplan-crear-001",
			RequestedBy:   "test",
		},
		DomainDocumentPlanDerivedJobsCreationPortsV0{JobCreator: creator},
	)
	if err != nil {
		t.Fatalf("CreateDomainDocumentPlanDerivedJobsV0: %v", err)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if result.Status != DomainDocumentPlanDerivedJobsCreationStatusAcceptedV0 ||
		result.CorrelationID != "corr-docplan-crear-001" {
		t.Fatalf("result=%+v", result)
	}
	wantJobs := len(validDocumentPlanForExpanderTestV0().Sections) +
		len(validDocumentPlanForExpanderTestV0().Visuals) +
		len(validDocumentPlanForExpanderTestV0().ReviewSteps)
	if len(result.RequestedJobs) != wantJobs ||
		len(result.CreatedJobs) != wantJobs ||
		len(creator.requests) != wantJobs {
		t.Fatalf("requested=%d created=%d calls=%d", len(result.RequestedJobs), len(result.CreatedJobs), len(creator.requests))
	}
	for i, job := range result.CreatedJobs {
		if job.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 ||
			job.JobRef == "" ||
			job.WorkKind != result.RequestedJobs[i].WorkKind {
			t.Fatalf("created[%d]=%+v requested=%+v", i, job, result.RequestedJobs[i])
		}
	}
}

func TestCreateDomainDocumentPlanDerivedJobsV0MantieneIdempotenciaEstable(t *testing.T) {
	request := DomainDocumentPlanExpansionRequestV0{
		Plan:          validDocumentPlanForExpanderTestV0(),
		CorrelationID: "corr-docplan-idem-001",
		RequestedBy:   "test",
	}
	firstCreator := &fakeDocumentPlanJobCreatorV0{}
	first, err := CreateDomainDocumentPlanDerivedJobsV0(
		context.Background(),
		request,
		DomainDocumentPlanDerivedJobsCreationPortsV0{JobCreator: firstCreator},
	)
	if err != nil || len(first.Issues) != 0 {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	secondCreator := &fakeDocumentPlanJobCreatorV0{}
	second, err := CreateDomainDocumentPlanDerivedJobsV0(
		context.Background(),
		request,
		DomainDocumentPlanDerivedJobsCreationPortsV0{JobCreator: secondCreator},
	)
	if err != nil || len(second.Issues) != 0 {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	if len(firstCreator.requests) != len(secondCreator.requests) {
		t.Fatalf("calls first=%d second=%d", len(firstCreator.requests), len(secondCreator.requests))
	}
	for i := range firstCreator.requests {
		firstRequest := firstCreator.requests[i]
		secondRequest := secondCreator.requests[i]
		if firstRequest.RequestID == "" ||
			firstRequest.RequestID != firstRequest.IdempotencyKey ||
			firstRequest.RequestID != secondRequest.RequestID ||
			firstRequest.IdempotencyKey != secondRequest.IdempotencyKey ||
			first.CreatedJobs[i].JobRef != second.CreatedJobs[i].JobRef {
			t.Fatalf("i=%d first=%+v second=%+v firstJob=%+v secondJob=%+v",
				i,
				firstRequest,
				secondRequest,
				first.CreatedJobs[i],
				second.CreatedJobs[i],
			)
		}
	}
}

func TestCreateDomainDocumentPlanDerivedJobsV0RequiereCreator(t *testing.T) {
	result, err := CreateDomainDocumentPlanDerivedJobsV0(
		context.Background(),
		DomainDocumentPlanExpansionRequestV0{Plan: validDocumentPlanForExpanderTestV0()},
		DomainDocumentPlanDerivedJobsCreationPortsV0{},
	)
	if err != nil {
		t.Fatalf("CreateDomainDocumentPlanDerivedJobsV0: %v", err)
	}
	if len(result.Issues) != 1 ||
		result.Issues[0].Code != ErrDomainDocumentPlanJobCreatorRequiredV0 ||
		result.Status != DomainDocumentPlanDerivedJobsCreationStatusInvalidV0 ||
		len(result.RequestedJobs) != 0 ||
		len(result.CreatedJobs) != 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestCreateDomainDocumentPlanDerivedJobsV0NoCreaSiExpansionInvalida(t *testing.T) {
	plan := validDocumentPlanForExpanderTestV0()
	plan.Sections = nil
	creator := &fakeDocumentPlanJobCreatorV0{}

	result, err := CreateDomainDocumentPlanDerivedJobsV0(
		context.Background(),
		DomainDocumentPlanExpansionRequestV0{Plan: plan},
		DomainDocumentPlanDerivedJobsCreationPortsV0{JobCreator: creator},
	)
	if err != nil {
		t.Fatalf("CreateDomainDocumentPlanDerivedJobsV0: %v", err)
	}
	if len(result.Issues) == 0 ||
		len(result.RequestedJobs) != 0 ||
		len(result.CreatedJobs) != 0 ||
		len(creator.requests) != 0 {
		t.Fatalf("result=%+v calls=%d", result, len(creator.requests))
	}
}

func TestCreateDomainDocumentPlanDerivedJobsV0DevuelveErrorDePuertoConProgreso(t *testing.T) {
	creator := &fakeDocumentPlanJobCreatorV0{
		errAt: 3,
		err:   errors.New("creator unavailable"),
	}
	result, err := CreateDomainDocumentPlanDerivedJobsV0(
		context.Background(),
		DomainDocumentPlanExpansionRequestV0{Plan: validDocumentPlanForExpanderTestV0()},
		DomainDocumentPlanDerivedJobsCreationPortsV0{JobCreator: creator},
	)
	if !errors.Is(err, creator.err) {
		t.Fatalf("err=%v", err)
	}
	wantJobs := len(validDocumentPlanForExpanderTestV0().Sections) +
		len(validDocumentPlanForExpanderTestV0().Visuals) +
		len(validDocumentPlanForExpanderTestV0().ReviewSteps)
	if len(result.RequestedJobs) != wantJobs ||
		len(result.CreatedJobs) != 3 ||
		len(creator.requests) != 4 {
		t.Fatalf("result=%+v calls=%d", result, len(creator.requests))
	}
}

func TestCreateDomainDocumentPlanDerivedJobsV0DetieneSiPuertoRechazaJob(t *testing.T) {
	creator := &fakeDocumentPlanJobCreatorV0{
		invalid:   true,
		invalidAt: 2,
	}
	result, err := CreateDomainDocumentPlanDerivedJobsV0(
		context.Background(),
		DomainDocumentPlanExpansionRequestV0{Plan: validDocumentPlanForExpanderTestV0()},
		DomainDocumentPlanDerivedJobsCreationPortsV0{JobCreator: creator},
	)
	if err != nil {
		t.Fatalf("CreateDomainDocumentPlanDerivedJobsV0: %v", err)
	}
	if len(result.Issues) == 0 ||
		result.Issues[len(result.Issues)-1].Code != ErrDomainDocumentPlanDerivedJobRejectedV0 ||
		len(result.CreatedJobs) != 2 ||
		len(creator.requests) != 3 {
		t.Fatalf("result=%+v calls=%d", result, len(creator.requests))
	}
}

func TestCreateDomainDocumentPlanDerivedJobsV0RequiereJobRefAceptado(t *testing.T) {
	creator := &fakeDocumentPlanJobCreatorV0{
		emptyJobRef:   true,
		emptyJobRefAt: 0,
	}
	result, err := CreateDomainDocumentPlanDerivedJobsV0(
		context.Background(),
		DomainDocumentPlanExpansionRequestV0{Plan: validDocumentPlanForExpanderTestV0()},
		DomainDocumentPlanDerivedJobsCreationPortsV0{JobCreator: creator},
	)
	if err != nil {
		t.Fatalf("CreateDomainDocumentPlanDerivedJobsV0: %v", err)
	}
	if len(result.Issues) != 1 ||
		result.Issues[0].Code != ErrDomainDocumentPlanDerivedJobRefRequiredV0 ||
		len(result.CreatedJobs) != 0 ||
		len(creator.requests) != 1 {
		t.Fatalf("result=%+v calls=%d", result, len(creator.requests))
	}
}

type fakeDocumentPlanJobCreatorV0 struct {
	requests      []orquestadomainwork.DomainWorkJobRequestV0
	errAt         int
	err           error
	invalid       bool
	invalidAt     int
	emptyJobRef   bool
	emptyJobRefAt int
}

func (fake *fakeDocumentPlanJobCreatorV0) CreateDomainWorkJobV0(
	_ context.Context,
	request orquestadomainwork.DomainWorkJobRequestV0,
) (orquestadomainwork.DomainWorkJobV0, error) {
	callIndex := len(fake.requests)
	fake.requests = append(fake.requests, request)
	if fake.err != nil && callIndex == fake.errAt {
		return orquestadomainwork.DomainWorkJobV0{}, fake.err
	}
	if fake.invalid && callIndex == fake.invalidAt {
		return orquestadomainwork.DomainWorkJobV0{
			SchemaVersion:  orquestadomainwork.DomainWorkJobSchemaV0,
			Status:         orquestadomainwork.DomainWorkStatusInvalidV0,
			DomainRef:      request.DomainRef,
			WorkKind:       request.WorkKind,
			CorrelationID:  request.CorrelationID,
			IdempotencyKey: request.IdempotencyKey,
			Issues: []orquestadomainwork.DomainWorkIssueV0{{
				Code:  "fake_invalid_job",
				Field: "fake",
			}},
		}, nil
	}
	jobRef := "job-created-" + request.RequestID
	if fake.emptyJobRef && callIndex == fake.emptyJobRefAt {
		jobRef = ""
	}
	return orquestadomainwork.DomainWorkJobV0{
		SchemaVersion:  orquestadomainwork.DomainWorkJobSchemaV0,
		Status:         orquestadomainwork.DomainWorkStatusAcceptedV0,
		JobRef:         jobRef,
		DomainRef:      request.DomainRef,
		WorkKind:       request.WorkKind,
		CorrelationID:  request.CorrelationID,
		IdempotencyKey: request.IdempotencyKey,
		ExternalRefs:   append([]orquestadomainwork.DomainWorkExternalRefV0(nil), request.ExternalRefs...),
		EvidenceRefs:   []string{"fake-created-" + request.RequestID},
	}, nil
}
