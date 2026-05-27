package orquestacli

import "testing"

func TestDefaultCliHTTPTransportPolicyV0DeclaraLoopbackSinProxy(t *testing.T) {
	policy := DefaultCliHTTPTransportPolicyV0()
	if policy.Profile != CliHTTPTransportProfileLoopbackControlPlaneV0 {
		t.Fatalf("profile=%q", policy.Profile)
	}
	if policy.ProxyPolicy != CliHTTPProxyPolicyDenyV0 {
		t.Fatalf("proxy_policy=%q", policy.ProxyPolicy)
	}
	transport := newCLIHTTPTransportV0(policy)
	if transport.Proxy != nil {
		t.Fatalf("loopback_control_plane no debe heredar proxy del entorno")
	}
	if transport.DialContext == nil || transport.TLSHandshakeTimeout <= 0 || transport.ResponseHeaderTimeout <= 0 {
		t.Fatalf("transporte cli debe declarar dial/tls/headers")
	}
}
