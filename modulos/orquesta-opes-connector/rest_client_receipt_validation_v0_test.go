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
