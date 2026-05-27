package orquestadomainworkhttp_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	orquestadomainworkhttp "orquesta/modulos/orquesta-domain-work-http"
)

func TestClientV0AplicaDeadlineConHTTPClientInyectadoSinTimeout(t *testing.T) {
	client, err := orquestadomainworkhttp.NewClientV0(orquestadomainworkhttp.ConfigV0{
		BaseURL:      "http://domain-work-http-test.invalid",
		EgressPolicy: orquestadomainworkhttp.AllowlistEgressPolicyV0("domain-work-http-test.invalid"),
		Timeout:      5 * time.Millisecond,
		HTTPClient: &http.Client{Transport: roundTripFuncV0(func(r *http.Request) (*http.Response, error) {
			<-r.Context().Done()
			return nil, r.Context().Err()
		})},
	})
	if err != nil {
		t.Fatalf("NewClientV0: %v", err)
	}

	_, err = client.CreateDomainWorkJobV0(context.Background(), validJobRequestV0())
	requireErrorCodeV0(t, err, orquestadomainworkhttp.ErrDomainWorkHTTPTimeoutV0)
}
