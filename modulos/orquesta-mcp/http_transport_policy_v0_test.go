package orquestamcp

import "testing"

func TestDefaultMCPHTTPTransportPolicyV0DeclaraLoopbackSinProxy(t *testing.T) {
	policy := DefaultMCPHTTPTransportPolicyV0()
	if policy.Profile != MCPHTTPTransportProfileLoopbackControlPlaneV0 {
		t.Fatalf("profile=%q", policy.Profile)
	}
	if policy.ProxyPolicy != MCPHTTPProxyPolicyDenyV0 {
		t.Fatalf("proxy_policy=%q", policy.ProxyPolicy)
	}
	transport := newMCPHTTPTransportV0(policy)
	if transport.Proxy != nil {
		t.Fatalf("loopback_control_plane no debe heredar proxy del entorno")
	}
	if transport.DialContext == nil || transport.TLSHandshakeTimeout <= 0 || transport.ResponseHeaderTimeout <= 0 {
		t.Fatalf("transporte mcp debe declarar dial/tls/headers")
	}
}
