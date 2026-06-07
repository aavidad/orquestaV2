package orquestamcp

import (
	"context"
	"strings"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestMCPDomainWorkDescriptorV0EsAdaptadorFino(t *testing.T) {
	descriptor := MCPDomainWorkDescriptorV0()
	if descriptor.Name != MCPDomainWorkToolNameV0 ||
		descriptor.ResourceURI != MCPDomainWorkResourceURIV0 ||
		!strings.Contains(descriptor.InputSchema, MCPDomainWorkActionCreateJobV0) ||
		!strings.Contains(descriptor.InputSchema, MCPDomainWorkActionSubmitArtifactV0) ||
		len(descriptor.Invariantes) == 0 {
		t.Fatalf("descriptor=%+v", descriptor)
	}
}

func TestMCPDomainWorkExecutorV0CreateJobDelegaEnCreator(t *testing.T) {
	creator := &fakeMCPDomainWorkCreatorV0{
		job: orquestadomainwork.DomainWorkJobV0{
			SchemaVersion:  orquestadomainwork.DomainWorkJobSchemaV0,
			Status:         orquestadomainwork.DomainWorkStatusAcceptedV0,
			JobRef:         "job-domain-001",
			DomainRef:      "domain-academic",
			WorkKind:       "syllabus",
			CorrelationID:  "corr-domain-001",
			IdempotencyKey: "idem-domain-001",
		},
	}
	executor := MCPDomainWorkToolExecutorV0{JobCreator: creator}

	result, err := executor.Execute(context.Background(), MCPDomainWorkToolInputV0{
		RequestID:     "req-domain-001",
		CorrelationID: "corr-domain-001",
		Action:        " create_job ",
		JobRequest: orquestadomainwork.DomainWorkJobRequestV0{
			IdempotencyKey: "idem-domain-001",
			RequestedBy:    "director",
			DomainRef:      "domain-academic",
			WorkKind:       "syllabus",
			Objective:      "crear propuesta de temario",
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if result.Estado != MCPDomainWorkEstadoOKV0 ||
		result.Action != MCPDomainWorkActionCreateJobV0 ||
		result.Job == nil ||
		result.Job.JobRef != "job-domain-001" {
		t.Fatalf("result=%+v", result)
	}
	if creator.called != 1 ||
		creator.request.RequestID != "req-domain-001" ||
		creator.request.CorrelationID != "corr-domain-001" ||
		creator.request.DomainRef != "domain-academic" {
		t.Fatalf("creator=%+v", creator)
	}
}

func TestMCPDomainWorkExecutorV0SubmitArtifactDelegaEnSubmitter(t *testing.T) {
	submitter := &fakeMCPDomainWorkSubmitterV0{
		receipt: orquestadomainwork.DomainWorkArtifactReceiptV0{
			SchemaVersion:  orquestadomainwork.DomainWorkArtifactReceiptSchemaV0,
			Status:         orquestadomainwork.DomainWorkStatusAcceptedV0,
			JobRef:         "job-domain-001",
			ArtifactRef:    "artifact-domain-001",
			ReceiptRef:     "receipt-domain-001",
			CorrelationID:  "corr-domain-002",
			IdempotencyKey: "idem-domain-002",
		},
	}
	executor := MCPDomainWorkToolExecutorV0{ArtifactSubmitter: submitter}

	result, err := executor.Execute(context.Background(), MCPDomainWorkToolInputV0{
		RequestID:     "req-domain-002",
		CorrelationID: "corr-domain-002",
		Action:        MCPDomainWorkActionSubmitArtifactV0,
		ArtifactSubmission: orquestadomainwork.DomainWorkArtifactSubmissionV0{
			IdempotencyKey: "idem-domain-002",
			RequestedBy:    "director",
			DomainRef:      "domain-academic",
			JobRef:         "job-domain-001",
			ArtifactRef:    "artifact-domain-001",
			ArtifactType:   "lesson_plan",
			Summary:        "artefacto listo",
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if result.Estado != MCPDomainWorkEstadoOKV0 ||
		result.Action != MCPDomainWorkActionSubmitArtifactV0 ||
		result.Receipt == nil ||
		result.Receipt.ReceiptRef != "receipt-domain-001" {
		t.Fatalf("result=%+v", result)
	}
	if submitter.called != 1 ||
		submitter.submission.RequestID != "req-domain-002" ||
		submitter.submission.CorrelationID != "corr-domain-002" ||
		submitter.submission.ArtifactRef != "artifact-domain-001" {
		t.Fatalf("submitter=%+v", submitter)
	}
}

func TestMCPDomainWorkExecutorV0PropagaReceiptInvalidoSinPuertoCaido(t *testing.T) {
	submitter := &fakeMCPDomainWorkSubmitterV0{
		receipt: orquestadomainwork.DomainWorkArtifactReceiptV0{
			SchemaVersion: orquestadomainwork.DomainWorkArtifactReceiptSchemaV0,
			Status:        orquestadomainwork.DomainWorkStatusInvalidV0,
			JobRef:        "job-domain-001",
			ArtifactRef:   "artifact-domain-001",
			Issues: []orquestadomainwork.DomainWorkIssueV0{{
				Code:  "opes_http_status_400",
				Field: "artifact_submitter",
			}},
		},
	}
	executor := MCPDomainWorkToolExecutorV0{ArtifactSubmitter: submitter}

	result, err := executor.Execute(context.Background(), MCPDomainWorkToolInputV0{
		RequestID: "req-domain-invalid-001",
		Action:    MCPDomainWorkActionSubmitArtifactV0,
		ArtifactSubmission: orquestadomainwork.DomainWorkArtifactSubmissionV0{
			IdempotencyKey: "idem-domain-invalid-001",
			RequestedBy:    "director",
			DomainRef:      "domain-academic",
			JobRef:         "job-domain-001",
			ArtifactRef:    "artifact-domain-001",
			ArtifactType:   "lesson_plan",
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if result.Estado != MCPDomainWorkEstadoErrorV0 ||
		result.Receipt == nil ||
		result.Receipt.Status != orquestadomainwork.DomainWorkStatusInvalidV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "opes_http_status_400" ||
		result.Errores[0].Code == MCPDomainWorkPortUnavailableV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPDomainWorkExecutorV0ValidaActionYPuertos(t *testing.T) {
	result, err := MCPDomainWorkToolExecutorV0{}.Execute(
		context.Background(),
		MCPDomainWorkToolInputV0{Action: "delete_job"},
	)
	if err != nil {
		t.Fatalf("Execute invalid action: %v", err)
	}
	if result.Estado != MCPDomainWorkEstadoErrorV0 ||
		result.Errores[0].Code != MCPDomainWorkActionUnsupportedV0 {
		t.Fatalf("result invalid action=%+v", result)
	}

	result, err = MCPDomainWorkToolExecutorV0{}.Execute(
		context.Background(),
		MCPDomainWorkToolInputV0{Action: MCPDomainWorkActionCreateJobV0},
	)
	if err != nil {
		t.Fatalf("Execute no creator: %v", err)
	}
	if result.Errores[0].Code != MCPDomainWorkCreatorUnavailableV0 {
		t.Fatalf("result no creator=%+v", result)
	}

	result, err = MCPDomainWorkToolExecutorV0{}.Execute(
		context.Background(),
		MCPDomainWorkToolInputV0{Action: MCPDomainWorkActionSubmitArtifactV0},
	)
	if err != nil {
		t.Fatalf("Execute no submitter: %v", err)
	}
	if result.Errores[0].Code != MCPDomainWorkSubmitterUnavailableV0 {
		t.Fatalf("result no submitter=%+v", result)
	}
}

type fakeMCPDomainWorkCreatorV0 struct {
	called  int
	request orquestadomainwork.DomainWorkJobRequestV0
	job     orquestadomainwork.DomainWorkJobV0
}

func (fake *fakeMCPDomainWorkCreatorV0) CreateDomainWorkJobV0(
	_ context.Context,
	request orquestadomainwork.DomainWorkJobRequestV0,
) (orquestadomainwork.DomainWorkJobV0, error) {
	fake.called++
	fake.request = request
	return fake.job, nil
}

type fakeMCPDomainWorkSubmitterV0 struct {
	called     int
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0
	receipt    orquestadomainwork.DomainWorkArtifactReceiptV0
}

func (fake *fakeMCPDomainWorkSubmitterV0) SubmitDomainWorkArtifactV0(
	_ context.Context,
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
) (orquestadomainwork.DomainWorkArtifactReceiptV0, error) {
	fake.called++
	fake.submission = submission
	return fake.receipt, nil
}
