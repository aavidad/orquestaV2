package orquestaopesconnector

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestRESTClientV0RetryPolicyRespetaRetryAfterEnConsultaV0(t *testing.T) {
	attempts := 0
	delays := []time.Duration{}
	client := NewRESTClientV0(RESTClientConfigV0{
		BaseURL: "http://opes.test",
		HTTPClient: &http.Client{Transport: restClientTestTransportV0{t: t, handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts == 1 {
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"error":"raw body ignored"}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"job-ref-opes-001","type":"plan_temario","status":"pending","execution_mode":"external","payload_json":"{}"}]`))
		})}},
		RetryPolicy: RetryPolicyV0{
			MaxAttempts:       2,
			RespectRetryAfter: true,
			Sleep: func(_ context.Context, delay time.Duration) error {
				delays = append(delays, delay)
				return nil
			},
		},
	})

	jobs, err := client.ListExternalJobsV0(context.Background(), ExternalJobQueryV0{Status: "pending"})

	if err != nil {
		t.Fatalf("ListExternalJobsV0: %v", err)
	}
	if attempts != 2 || len(delays) != 1 || delays[0] != 0 || len(jobs) != 1 || jobs[0].ID != "job-ref-opes-001" {
		t.Fatalf("attempts=%d delays=%v jobs=%+v", attempts, delays, jobs)
	}
}

func TestRESTClientV0RetryPolicyBloqueaMutationSinIdempotenciaV0(t *testing.T) {
	attempts := 0
	client := NewRESTClientV0(RESTClientConfigV0{
		BaseURL: "http://opes.test",
		HTTPClient: &http.Client{Transport: restClientTestTransportV0{t: t, handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusTooManyRequests)
		})}},
		RetryPolicy: RetryPolicyV0{MaxAttempts: 2},
	})

	err := client.postJSONV0(context.Background(), "/api/jobs", opesCreateJobRequestV0{}, &opesJobResponseV0{})

	if err == nil || err.Error() != ErrOPESRetryBlockedV0 || attempts != 1 {
		t.Fatalf("attempts=%d err=%v", attempts, err)
	}
}
