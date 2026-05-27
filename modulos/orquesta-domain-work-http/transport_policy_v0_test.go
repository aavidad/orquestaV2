package orquestadomainworkhttp

import "testing"

func TestDefaultHTTPTransportPolicyV0DeclaraDomainEgressSinProxy(t *testing.T) {
	policy := DefaultHTTPTransportPolicyV0()
	if policy.Profile != HTTPTransportProfileDomainEgressV0 {
		t.Fatalf("profile=%q", policy.Profile)
	}
	if policy.ProxyPolicy != HTTPProxyPolicyDenyV0 {
		t.Fatalf("proxy_policy=%q", policy.ProxyPolicy)
	}
	transport := newDomainWorkHTTPTransportV0(policy)
	if transport.Proxy != nil {
		t.Fatalf("domain_egress no debe heredar proxy del entorno")
	}
	if transport.DialContext == nil || transport.TLSHandshakeTimeout <= 0 || transport.ResponseHeaderTimeout <= 0 {
		t.Fatalf("transporte domain_egress debe declarar dial/tls/headers")
	}
	if transport.MaxIdleConns <= 0 || transport.MaxIdleConnsPerHost <= 0 || transport.IdleConnTimeout <= 0 {
		t.Fatalf("transporte domain_egress debe declarar pool y keepalive")
	}
}
