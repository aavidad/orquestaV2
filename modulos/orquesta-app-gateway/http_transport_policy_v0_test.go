package orquestaappgateway

import (
	"net/http"
	"testing"
	"time"
)

func TestDefaultGatewayHTTPTransportPolicyV0DeclaraInProcessSinProxy(t *testing.T) {
	policy := DefaultGatewayHTTPTransportPolicyV0()
	if policy.Profile != GatewayHTTPTransportProfileInternalInProcessV0 {
		t.Fatalf("profile=%q", policy.Profile)
	}
	if policy.ProxyPolicy != GatewayHTTPProxyPolicyDenyV0 || policy.Network != "none" {
		t.Fatalf("policy=%+v", policy)
	}
	client := newGatewayInProcessHTTPClientV0(ConfigV0{Timeout: time.Second}, http.NewServeMux())
	if _, ok := client.Transport.(InProcessTransportV0); !ok {
		t.Fatalf("internal_inprocess debe usar InProcessTransportV0")
	}
}
