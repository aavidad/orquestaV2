package orquestadomainworkhttp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestClientV0DevuelveErrorConStatusNo2xx(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		call     func(orquestadomainworkhttp.ClientV0) error
		wantCode string
	}{
		{
			name: "create job",
			path: "/jobs",
			call: func(client orquestadomainworkhttp.ClientV0) error {
				_, err := client.CreateDomainWorkJobV0(context.Background(), validJobRequestV0())
				return err
			},
			wantCode: orquestadomainworkhttp.ErrDomainWorkHTTPStatusFailedV0 + "_503",
		},
		{
			name: "submit artifact",
			path: "/artifacts",
			call: func(client orquestadomainworkhttp.ClientV0) error {
				_, err := client.SubmitDomainWorkArtifactV0(context.Background(), validArtifactSubmissionV0("job-ref-non-opes-001"))
				return err
			},
			wantCode: orquestadomainworkhttp.ErrDomainWorkHTTPStatusFailedV0 + "_409",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statusByPath := map[string]int{
				"/jobs":      http.StatusServiceUnavailable,
				"/artifacts": http.StatusConflict,
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.path {
					t.Fatalf("path=%s", r.URL.Path)
				}
				http.Error(w, "external adapter failed", statusByPath[tt.path])
			}))
			defer server.Close()
			client := mustNewClientV0(t, server.URL)
			err := tt.call(client)
			requireErrorCodeV0(t, err, tt.wantCode)
		})
	}
}

func TestClientV0DevuelveErrorConTimeoutOCancelacion(t *testing.T) {
	client, err := orquestadomainworkhttp.NewClientV0(orquestadomainworkhttp.ConfigV0{
		BaseURL: "http://domain-work-http-test.invalid",
		HTTPClient: &http.Client{Transport: roundTripFuncV0(func(r *http.Request) (*http.Response, error) {
			<-r.Context().Done()
			return nil, r.Context().Err()
		})},
	})
	if err != nil {
		t.Fatalf("NewClientV0: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err = client.CreateDomainWorkJobV0(ctx, validJobRequestV0())
	requireErrorCodeV0(t, err, orquestadomainworkhttp.ErrDomainWorkHTTPRequestFailedV0)
}

func TestClientV0DevuelveErrorConPayloadInvalido(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		body     string
		call     func(orquestadomainworkhttp.ClientV0) error
		wantCode string
	}{
		{
			name: "create job invalid json",
			path: "/jobs",
			body: "{",
			call: func(client orquestadomainworkhttp.ClientV0) error {
				_, err := client.CreateDomainWorkJobV0(context.Background(), validJobRequestV0())
				return err
			},
			wantCode: orquestadomainworkhttp.ErrDomainWorkHTTPResponseDecodeFailedV0,
		},
		{
			name: "create job missing ref",
			path: "/jobs",
			body: `{"status":"accepted"}`,
			call: func(client orquestadomainworkhttp.ClientV0) error {
				_, err := client.CreateDomainWorkJobV0(context.Background(), validJobRequestV0())
				return err
			},
			wantCode: orquestadomainworkhttp.ErrDomainWorkHTTPResponseJobMissingV0,
		},
		{
			name: "submit artifact invalid json",
			path: "/artifacts",
			body: "{",
			call: func(client orquestadomainworkhttp.ClientV0) error {
				_, err := client.SubmitDomainWorkArtifactV0(context.Background(), validArtifactSubmissionV0("job-ref-non-opes-001"))
				return err
			},
			wantCode: orquestadomainworkhttp.ErrDomainWorkHTTPResponseDecodeFailedV0,
		},
		{
			name: "submit artifact missing receipt",
			path: "/artifacts",
			body: `{"status":"accepted"}`,
			call: func(client orquestadomainworkhttp.ClientV0) error {
				_, err := client.SubmitDomainWorkArtifactV0(context.Background(), validArtifactSubmissionV0("job-ref-non-opes-001"))
				return err
			},
			wantCode: orquestadomainworkhttp.ErrDomainWorkHTTPResponseReceiptMissingV0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.path {
					t.Fatalf("path=%s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			client := mustNewClientV0(t, server.URL)
			err := tt.call(client)
			requireErrorCodeV0(t, err, tt.wantCode)
		})
	}
}

func TestClientV0MantieneRefsOpacasEnJobsYArtefactos(t *testing.T) {
	jobRequest := validJobRequestV0()
	jobRequest.DomainRef = "tenant:alpha.topic#42"
	jobRequest.WorkRefs = []string{"workflow:case#99"}
	jobRequest.InputRefs = []string{"input:opaque#payload"}
	jobRequest.EvidenceRefs = []string{"evidence:opaque#job"}
	jobRequest.ExternalRefs = []orquestadomainwork.DomainWorkExternalRefV0{{
		Kind: "external_ticket",
		Ref:  "ticket:nonopes#abc.123",
	}}
	artifactSubmission := validArtifactSubmissionV0("job:opaque#created")
	artifactSubmission.ArtifactRef = "artifact:opaque#delivery.1"
	artifactSubmission.PayloadRefs = []string{"payload:opaque#artifact"}
	artifactSubmission.EvidenceRefs = []string{"evidence:opaque#artifact"}
	artifactSubmission.ExternalRefs = []orquestadomainwork.DomainWorkExternalRefV0{{
		Kind: "external_run",
		Ref:  "run:nonopes#xyz.789",
	}}

	var gotJobRequest orquestadomainwork.DomainWorkJobRequestV0
	var gotSubmission orquestadomainwork.DomainWorkArtifactSubmissionV0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jobs":
			if err := json.NewDecoder(r.Body).Decode(&gotJobRequest); err != nil {
				t.Fatalf("decode job: %v", err)
			}
			_ = json.NewEncoder(w).Encode(orquestadomainwork.DomainWorkJobV0{
				SchemaVersion:  orquestadomainwork.DomainWorkJobSchemaV0,
				Status:         orquestadomainwork.DomainWorkStatusAcceptedV0,
				JobRef:         "job:opaque#created",
				DomainRef:      gotJobRequest.DomainRef,
				WorkKind:       gotJobRequest.WorkKind,
				CorrelationID:  gotJobRequest.CorrelationID,
				IdempotencyKey: gotJobRequest.IdempotencyKey,
				ExternalRefs:   gotJobRequest.ExternalRefs,
				EvidenceRefs:   gotJobRequest.EvidenceRefs,
			})
		case "/artifacts":
			if err := json.NewDecoder(r.Body).Decode(&gotSubmission); err != nil {
				t.Fatalf("decode submission: %v", err)
			}
			_ = json.NewEncoder(w).Encode(orquestadomainwork.DomainWorkArtifactReceiptV0{
				SchemaVersion:  orquestadomainwork.DomainWorkArtifactReceiptSchemaV0,
				Status:         orquestadomainwork.DomainWorkStatusAcceptedV0,
				JobRef:         gotSubmission.JobRef,
				ArtifactRef:    gotSubmission.ArtifactRef,
				ReceiptRef:     "receipt:opaque#accepted.1",
				CorrelationID:  gotSubmission.CorrelationID,
				IdempotencyKey: gotSubmission.IdempotencyKey,
				ExternalRefs:   gotSubmission.ExternalRefs,
				EvidenceRefs:   gotSubmission.EvidenceRefs,
			})
		default:
			t.Fatalf("path=%s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := mustNewClientV0(t, server.URL)
	job, err := client.CreateDomainWorkJobV0(context.Background(), jobRequest)
	if err != nil {
		t.Fatalf("CreateDomainWorkJobV0: %v", err)
	}
	receipt, err := client.SubmitDomainWorkArtifactV0(context.Background(), artifactSubmission)
	if err != nil {
		t.Fatalf("SubmitDomainWorkArtifactV0: %v", err)
	}
	if job.JobRef != "job:opaque#created" ||
		job.DomainRef != "tenant:alpha.topic#42" ||
		job.ExternalRefs[0].Ref != "ticket:nonopes#abc.123" ||
		job.EvidenceRefs[0] != "evidence:opaque#job" ||
		gotJobRequest.WorkRefs[0] != "workflow:case#99" ||
		gotJobRequest.InputRefs[0] != "input:opaque#payload" {
		t.Fatalf("job=%+v got=%+v", job, gotJobRequest)
	}
	if receipt.JobRef != "job:opaque#created" ||
		receipt.ArtifactRef != "artifact:opaque#delivery.1" ||
		receipt.ReceiptRef != "receipt:opaque#accepted.1" ||
		receipt.ExternalRefs[0].Ref != "run:nonopes#xyz.789" ||
		receipt.EvidenceRefs[0] != "evidence:opaque#artifact" ||
		gotSubmission.PayloadRefs[0] != "payload:opaque#artifact" {
		t.Fatalf("receipt=%+v got=%+v", receipt, gotSubmission)
	}
}

func mustNewClientV0(t *testing.T, baseURL string) orquestadomainworkhttp.ClientV0 {
	t.Helper()
	client, err := orquestadomainworkhttp.NewClientV0(orquestadomainworkhttp.ConfigV0{
		BaseURL:            baseURL,
		CreateJobPath:      "/jobs",
		SubmitArtifactPath: "/artifacts",
	})
	if err != nil {
		t.Fatalf("NewClientV0: %v", err)
	}
	return client
}

func requireErrorCodeV0(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil || err.Error() != code {
		t.Fatalf("err=%v want=%s", err, code)
	}
}

type roundTripFuncV0 func(*http.Request) (*http.Response, error)

func (fn roundTripFuncV0) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
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
