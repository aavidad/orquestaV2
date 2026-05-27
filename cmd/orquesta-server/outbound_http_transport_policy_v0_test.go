package main

import "testing"

func TestCommandHTTPTransportPolicyV0DeclaraPerfilesSinProxy(t *testing.T) {
	for _, profile := range []string{
		commandHTTPTransportProfileLoopbackControlPlaneV0,
		commandHTTPTransportProfileOPESTemporalV0,
	} {
		policy := commandHTTPTransportPolicyV0ForProfile(profile)
		if policy.Profile != profile {
			t.Fatalf("profile=%q want %q", policy.Profile, profile)
		}
		if policy.ProxyPolicy != commandHTTPProxyPolicyDenyV0 {
			t.Fatalf("proxy_policy=%q", policy.ProxyPolicy)
		}
		transport := commandHTTPTransportV0(profile)
		if transport.Proxy != nil {
			t.Fatalf("%s no debe heredar proxy del entorno", profile)
		}
		if transport.DialContext == nil || transport.TLSHandshakeTimeout <= 0 || transport.ResponseHeaderTimeout <= 0 {
			t.Fatalf("%s debe declarar dial/tls/headers", profile)
		}
	}
}
