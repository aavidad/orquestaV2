package orquestadomainwork

import (
	"strings"
	"testing"
)

func TestBuildDomainWorkJobIdentityV0CanonicalizaSinRequestID(t *testing.T) {
	request := validDomainWorkJobRequestForTestV0()
	first, err := BuildDomainWorkJobIdentityV0(request)
	if err != nil {
		t.Fatalf("BuildDomainWorkJobIdentityV0: %v", err)
	}
	request.RequestID = "req-distinta"
	second, err := BuildDomainWorkJobIdentityV0(request)
	if err != nil {
		t.Fatalf("BuildDomainWorkJobIdentityV0 retry: %v", err)
	}
	if first != second {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	if !strings.HasPrefix(first.Fingerprint, "sha256-") ||
		!strings.HasPrefix(first.JobRefBase, "domain-work-job-sha256-") {
		t.Fatalf("identity=%+v", first)
	}
}

func TestBuildDomainWorkJobIdentityV0DistingueRequiredTests(t *testing.T) {
	request := validDomainWorkJobRequestForTestV0()
	base, err := BuildDomainWorkJobIdentityV0(request)
	if err != nil {
		t.Fatalf("base identity: %v", err)
	}
	request.RequiredTests = []DomainWorkRequiredTestV0{{
		TestRef: "domain-test-ref-001",
	}}
	withTest, err := BuildDomainWorkJobIdentityV0(request)
	if err != nil {
		t.Fatalf("with test identity: %v", err)
	}
	if withTest.Fingerprint == base.Fingerprint {
		t.Fatalf("fingerprint no distingue required_tests: %s", base.Fingerprint)
	}
	if withTest.JobRefBase != base.JobRefBase {
		t.Fatalf("job ref base debe depender solo de clave idempotente")
	}
}
