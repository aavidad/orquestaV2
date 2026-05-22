package orquestadomainworkhttp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestadomainworkhttp "orquesta/modulos/orquesta-domain-work-http"
)

func TestClientV0CreaJobYEnviaArtefactoPorContratoNeutral(t *testing.T) {
	var gotJobRequest orquestadomainwork.DomainWorkJobRequestV0
	var gotSubmission orquestadomainwork.DomainWorkArtifactSubmissionV0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jobs":
			if r.Method != http.MethodPost {
				t.Fatalf("method=%s", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&gotJobRequest); err != nil {
				t.Fatalf("decode job: %v", err)
			}
			_ = json.NewEncoder(w).Encode(orquestadomainwork.DomainWorkJobV0{
				SchemaVersion:  orquestadomainwork.DomainWorkJobSchemaV0,
				Status:         orquestadomainwork.DomainWorkStatusAcceptedV0,
				JobRef:         "job-ref-non-opes-001",
				DomainRef:      gotJobRequest.DomainRef,
				WorkKind:       gotJobRequest.WorkKind,
				CorrelationID:  gotJobRequest.CorrelationID,
				IdempotencyKey: gotJobRequest.IdempotencyKey,
				ExternalRefs:   gotJobRequest.ExternalRefs,
			})
		case "/artifacts":
			if r.Method != http.MethodPost {
				t.Fatalf("method=%s", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&gotSubmission); err != nil {
				t.Fatalf("decode submission: %v", err)
			}
			_ = json.NewEncoder(w).Encode(orquestadomainwork.DomainWorkArtifactReceiptV0{
				SchemaVersion:  orquestadomainwork.DomainWorkArtifactReceiptSchemaV0,
				Status:         orquestadomainwork.DomainWorkStatusAcceptedV0,
				JobRef:         gotSubmission.JobRef,
				ArtifactRef:    gotSubmission.ArtifactRef,
				ReceiptRef:     "receipt-ref-non-opes-001",
				CorrelationID:  gotSubmission.CorrelationID,
				IdempotencyKey: gotSubmission.IdempotencyKey,
				ExternalRefs:   gotSubmission.ExternalRefs,
			})
		default:
			t.Fatalf("path=%s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := orquestadomainworkhttp.NewClientV0(orquestadomainworkhttp.ConfigV0{
		BaseURL:            server.URL,
		CreateJobPath:      "/jobs",
		SubmitArtifactPath: "/artifacts",
	})
	if err != nil {
		t.Fatalf("NewClientV0: %v", err)
	}

	job, err := client.CreateDomainWorkJobV0(context.Background(), validJobRequestV0())
	if err != nil {
		t.Fatalf("CreateDomainWorkJobV0: %v", err)
	}
	if job.JobRef != "job-ref-non-opes-001" ||
		job.DomainRef != "domain-ref-non-opes-001" ||
		gotJobRequest.ExternalRefs[0].Ref != "entity-ref-non-opes-001" {
		t.Fatalf("job=%+v got=%+v", job, gotJobRequest)
	}

	receipt, err := client.SubmitDomainWorkArtifactV0(context.Background(), validArtifactSubmissionV0(job.JobRef))
	if err != nil {
		t.Fatalf("SubmitDomainWorkArtifactV0: %v", err)
	}
	if receipt.ReceiptRef != "receipt-ref-non-opes-001" ||
		gotSubmission.JobRef != job.JobRef ||
		gotSubmission.ExternalRefs[0].Ref != "run-ref-non-opes-001" {
		t.Fatalf("receipt=%+v got=%+v", receipt, gotSubmission)
	}
}

func TestClientV0RechazaBaseURLInvalida(t *testing.T) {
	_, err := orquestadomainworkhttp.NewClientV0(orquestadomainworkhttp.ConfigV0{})
	if err == nil || err.Error() != orquestadomainworkhttp.ErrDomainWorkHTTPBaseURLRequiredV0 {
		t.Fatalf("err=%v", err)
	}
}

func TestClientV0NoLlamaHTTPConSubmissionInvalida(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		t.Fatalf("HTTP no esperado")
	}))
	defer server.Close()
	client, err := orquestadomainworkhttp.NewClientV0(orquestadomainworkhttp.ConfigV0{
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("NewClientV0: %v", err)
	}
	receipt, err := client.SubmitDomainWorkArtifactV0(context.Background(), orquestadomainwork.DomainWorkArtifactSubmissionV0{})
	if err != nil {
		t.Fatalf("SubmitDomainWorkArtifactV0: %v", err)
	}
	if receipt.Status != orquestadomainwork.DomainWorkStatusInvalidV0 ||
		len(receipt.Issues) == 0 ||
		calls != 0 {
		t.Fatalf("receipt=%+v calls=%d", receipt, calls)
	}
}

func validJobRequestV0() orquestadomainwork.DomainWorkJobRequestV0 {
	return orquestadomainwork.DomainWorkJobRequestV0{
		SchemaVersion:  orquestadomainwork.DomainWorkJobRequestSchemaV0,
		RequestID:      "request-ref-non-opes-001",
		CorrelationID:  "corr-non-opes-001",
		IdempotencyKey: "idem-non-opes-001",
		RequestedBy:    "test",
		DomainRef:      "domain-ref-non-opes-001",
		WorkKind:       "compose_external_summary",
		Objective:      "crear job en app externa no OPES",
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{{
			Kind: "entity_ref",
			Ref:  "entity-ref-non-opes-001",
		}},
	}
}

func validArtifactSubmissionV0(jobRef string) orquestadomainwork.DomainWorkArtifactSubmissionV0 {
	return orquestadomainwork.DomainWorkArtifactSubmissionV0{
		SchemaVersion:  orquestadomainwork.DomainWorkArtifactSubmissionSchemaV0,
		RequestID:      "request-ref-non-opes-submit-001",
		CorrelationID:  "corr-non-opes-001",
		IdempotencyKey: "idem-non-opes-submit-001",
		RequestedBy:    "test",
		DomainRef:      "domain-ref-non-opes-001",
		JobRef:         jobRef,
		ArtifactRef:    "artifact-ref-non-opes-001",
		ArtifactType:   "work_delivery",
		Summary:        "entrega no OPES",
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{{
			Kind: "run_ref",
			Ref:  "run-ref-non-opes-001",
		}},
	}
}
