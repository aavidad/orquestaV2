package orquestaweb

import "testing"

func TestDefaultWebHTTPTransportPolicyV0DeclaraLoopbackSinProxy(t *testing.T) {
	policy := DefaultWebHTTPTransportPolicyV0()
	if policy.Profile != WebHTTPTransportProfileLoopbackControlPlaneV0 {
		t.Fatalf("profile=%q", policy.Profile)
	}
	if policy.ProxyPolicy != WebHTTPProxyPolicyDenyV0 {
		t.Fatalf("proxy_policy=%q", policy.ProxyPolicy)
	}
	transport := newWebHTTPTransportV0(policy)
	if transport.Proxy != nil {
		t.Fatalf("loopback_control_plane no debe heredar proxy del entorno")
	}
	if transport.DialContext == nil || transport.TLSHandshakeTimeout <= 0 || transport.ResponseHeaderTimeout <= 0 {
		t.Fatalf("transporte web debe declarar dial/tls/headers")
	}
}
