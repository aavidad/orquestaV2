package main

import "testing"

func TestGuardianCandidateAddressPolicyV0AceptaLoopbackTemporalDinamico(t *testing.T) {
	addr, policy, err := guardianCandidateAddressPolicyV0("")
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if addr != guardianCandidateDynamicAddrV0 || !policy.Dynamic {
		t.Fatalf("addr=%q policy=%+v", addr, policy)
	}
}

func TestGuardianCandidateAddressPolicyV0RechazaExternosYURLs(t *testing.T) {
	for _, raw := range []string{
		"0.0.0.0:8787",
		"192.0.2.10:8787",
		"http://127.0.0.1:8787/api/v0/server/readiness",
		"user@127.0.0.1:8787",
		"127.0.0.1:0",
	} {
		if _, _, err := guardianCandidateAddressPolicyV0(raw); err == nil {
			t.Fatalf("addr aceptado sin ownership: %q", raw)
		}
	}
}

func TestGuardianCandidateAddressPolicyV0AceptaLoopbackDeclarado(t *testing.T) {
	addr, policy, err := guardianCandidateAddressPolicyV0("127.0.0.1:8787")
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if addr != "127.0.0.1:8787" || policy.Dynamic {
		t.Fatalf("addr=%q policy=%+v", addr, policy)
	}
}
