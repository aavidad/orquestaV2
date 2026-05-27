package orquestaopesconnector

import "testing"

func TestDefaultHTTPTransportPolicyV0DeclaraOPESTemporalSinProxy(t *testing.T) {
	policy := DefaultHTTPTransportPolicyV0()
	if policy.Profile != HTTPTransportProfileOPESTemporalV0 {
		t.Fatalf("profile=%q", policy.Profile)
	}
	if policy.ProxyPolicy != HTTPProxyPolicyDenyV0 {
		t.Fatalf("proxy_policy=%q", policy.ProxyPolicy)
	}
	transport := newOPESHTTPTransportV0(policy)
	if transport.Proxy != nil {
		t.Fatalf("opes_temporal no debe heredar proxy del entorno")
	}
	if transport.DialContext == nil || transport.TLSHandshakeTimeout <= 0 || transport.ResponseHeaderTimeout <= 0 {
		t.Fatalf("transporte opes_temporal debe declarar dial/tls/headers")
	}
	if transport.MaxIdleConns <= 0 || transport.MaxIdleConnsPerHost <= 0 || transport.IdleConnTimeout <= 0 {
		t.Fatalf("transporte opes_temporal debe declarar pool y keepalive")
	}
}
