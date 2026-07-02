package orquestadomainworkhttp_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	orquestadomainworkhttp "orquesta/modulos/orquesta-domain-work-http"
)

func TestClientV0RetryPolicyRespetaRetryAfterYBudgetV0(t *testing.T) {
	attempts := 0
	delays := []time.Duration{}
	client, err := orquestadomainworkhttp.NewClientV0(orquestadomainworkhttp.ConfigV0{
		BaseURL:       "http://127.0.0.1:18081",
		CreateJobPath: "/jobs",
		HTTPClient: &http.Client{Transport: roundTripFuncV0(func(r *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return &http.Response{
					StatusCode: http.StatusTooManyRequests,
					Header:     http.Header{"Retry-After": []string{"0"}},
					Body:       io.NopCloser(strings.NewReader(`{"error":"raw body ignored"}`)),
					Request:    r,
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusCreated,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"job_ref":"job-ref-retry-001","status":"accepted"}`)),
				Request:    r,
			}, nil
		})},
		RetryPolicy: orquestadomainworkhttp.RetryPolicyV0{
			MaxAttempts:       2,
			RespectRetryAfter: true,
			Sleep: func(_ context.Context, delay time.Duration) error {
				delays = append(delays, delay)
				return nil
			},
		},
	})
	if err != nil {
		t.Fatalf("NewClientV0: %v", err)
	}

	job, err := client.CreateDomainWorkJobV0(context.Background(), validJobRequestV0())

	if err != nil {
		t.Fatalf("CreateDomainWorkJobV0: %v", err)
	}
	if attempts != 2 || len(delays) != 1 || delays[0] != 0 || job.JobRef != "job-ref-retry-001" {
		t.Fatalf("attempts=%d delays=%v job=%+v", attempts, delays, job)
	}
}

func TestClientV0RetryPolicyBloqueaRetryAfterFueraDeDeadlineV0(t *testing.T) {
	client, err := orquestadomainworkhttp.NewClientV0(orquestadomainworkhttp.ConfigV0{
		BaseURL:       "http://127.0.0.1:18081",
		CreateJobPath: "/jobs",
		HTTPClient: &http.Client{Transport: roundTripFuncV0(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Header:     http.Header{"Retry-After": []string{"10"}},
				Body:       io.NopCloser(strings.NewReader(`temporarily unavailable`)),
				Request:    r,
			}, nil
		})},
		RetryPolicy: orquestadomainworkhttp.RetryPolicyV0{
			MaxAttempts:       2,
			MaxDelay:          time.Second,
			RespectRetryAfter: true,
		},
	})
	if err != nil {
		t.Fatalf("NewClientV0: %v", err)
	}

	_, err = client.CreateDomainWorkJobV0(context.Background(), validJobRequestV0())

	requireErrorCodeV0(t, err, orquestadomainworkhttp.ErrDomainWorkHTTPRetryBudgetExhaustedV0)
}
