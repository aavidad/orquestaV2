package orquestaopesconnector

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestRESTClientV0SubmitDomainWorkArtifactRechazaReceiptNoCausal(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response map[string]any
	}{
		{
			name: "job-distinto",
			response: map[string]any{
				"id":     "artifact-receipt-ref-001",
				"job_id": "job-ref-otro",
			},
		},
		{
			name: "artifact-type-distinto",
			response: map[string]any{
				"id":     "artifact-receipt-ref-001",
				"job_id": "job-ref-opes-001",
				"artifact": map[string]any{
					"job_id": "job-ref-opes-001",
					"type":   "visual_asset",
				},
			},
		},
		{
			name: "job-no-completado",
			response: map[string]any{
				"id":     "artifact-receipt-ref-001",
				"job_id": "job-ref-opes-001",
				"job": map[string]any{
					"id":     "job-ref-opes-001",
					"status": "pending",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := newRESTClientForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/jobs/job-ref-opes-001/artifacts" {
					t.Fatalf("path=%s", r.URL.Path)
				}
				_ = json.NewEncoder(w).Encode(tc.response)
			}))

			result, err := client.SubmitDomainWorkArtifactV0(context.Background(), opesArtifactSubmissionForTestV0())

			if err != nil {
				t.Fatalf("SubmitDomainWorkArtifactV0: %v", err)
			}
			if result.Status != orquestadomainwork.DomainWorkStatusInvalidV0 || len(result.Issues) == 0 {
				t.Fatalf("receipt debe ser invalido: %+v", result)
			}
		})
	}
}

func TestRESTClientV0SubmitDomainWorkArtifactAceptaEstadosTerminalesNativosOPES(t *testing.T) {
	for _, tc := range []struct {
		name         string
		status       string
		wantAccepted bool
	}{
		{name: "completed", status: "completed", wantAccepted: true},
		{name: "done", status: "done", wantAccepted: true},
		{name: "settled", status: "settled", wantAccepted: true},
		{name: "pending", status: "pending", wantAccepted: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := newRESTClientForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/jobs/job-ref-opes-001/artifacts" {
					t.Fatalf("path=%s", r.URL.Path)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id":     "artifact-receipt-ref-001",
					"job_id": "job-ref-opes-001",
					"artifact": map[string]any{
						"job_id": "job-ref-opes-001",
						"type":   "content_block",
					},
					"job": map[string]any{
						"id":     "job-ref-opes-001",
						"status": tc.status,
					},
				})
			}))

			result, err := client.SubmitDomainWorkArtifactV0(context.Background(), opesArtifactSubmissionForTestV0())

			if err != nil {
				t.Fatalf("SubmitDomainWorkArtifactV0: %v", err)
			}
			if tc.wantAccepted {
				if result.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 ||
					result.JobRef != "job-ref-opes-001" ||
					result.ReceiptRef != "artifact-receipt-ref-001" ||
					len(result.Issues) != 0 {
					t.Fatalf("receipt debe ser aceptado: %+v", result)
				}
				return
			}
			if result.Status != orquestadomainwork.DomainWorkStatusInvalidV0 ||
				len(result.Issues) == 0 ||
				result.Issues[0].Field != "job.status" {
				t.Fatalf("receipt debe ser invalido por job.status: %+v", result)
			}
		})
	}
}

func TestRESTClientV0SubmitDomainWorkArtifactDevuelveReceiptInvalidoConHTTP4xx(t *testing.T) {
	client := newRESTClientForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs/job-ref-opes-001/artifacts" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid job artifact"}`))
	}))

	result, err := client.SubmitDomainWorkArtifactV0(context.Background(), opesArtifactSubmissionForTestV0())

	if err != nil {
		t.Fatalf("SubmitDomainWorkArtifactV0: %v", err)
	}
	if result.Status != orquestadomainwork.DomainWorkStatusInvalidV0 ||
		result.JobRef != "job-ref-opes-001" ||
		result.ArtifactRef != "artifact-ref-001" ||
		len(result.Issues) != 1 ||
		result.Issues[0].Code != ErrOPESHTTPStatusV0+"_400" ||
		result.Issues[0].Field != "artifact_submitter" {
		t.Fatalf("receipt=%+v", result)
	}
}
