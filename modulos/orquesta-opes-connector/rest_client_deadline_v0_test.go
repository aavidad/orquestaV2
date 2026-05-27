package orquestaopesconnector

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestRESTClientV0AplicaDeadlineConHTTPClientInyectadoSinTimeout(t *testing.T) {
	client := NewRESTClientV0(RESTClientConfigV0{
		BaseURL: "http://opes-test.invalid",
		Timeout: 5 * time.Millisecond,
		HTTPClient: &http.Client{Transport: roundTripFuncV0(func(r *http.Request) (*http.Response, error) {
			<-r.Context().Done()
			return nil, r.Context().Err()
		})},
	})

	_, err := client.ListExternalJobsV0(context.Background(), ExternalJobQueryV0{
		ExecutionMode: "external",
		Status:        "pending",
		Limit:         1,
	})
	if err == nil || err.Error() != ErrOPESHTTPTimeoutV0 {
		t.Fatalf("err=%v", err)
	}
}

type roundTripFuncV0 func(*http.Request) (*http.Response, error)

func (fn roundTripFuncV0) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
