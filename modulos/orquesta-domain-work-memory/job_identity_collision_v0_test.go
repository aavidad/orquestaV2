package orquestadomainworkmemory

import (
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestInMemoryDomainWorkJobCreatorV0ReparaColisionDeJobRef(t *testing.T) {
	key := domainWorkMemoryJobKeyV0{
		DomainRef:      "domain-target",
		IdempotencyKey: "idem-target",
	}
	identity, err := orquestadomainwork.BuildDomainWorkJobIdentityV0(
		orquestadomainwork.DomainWorkJobRequestV0{
			DomainRef:      key.DomainRef,
			IdempotencyKey: key.IdempotencyKey,
		},
	)
	if err != nil {
		t.Fatalf("BuildDomainWorkJobIdentityV0: %v", err)
	}
	creator := NewInMemoryDomainWorkJobCreatorV0()
	creator.jobsByRef[identity.JobRefBase] = domainWorkMemoryJobKeyV0{
		DomainRef:      "domain-other",
		IdempotencyKey: "idem-other",
	}
	got := creator.nextJobRefLockedV0(key)
	if got == identity.JobRefBase ||
		got != domainWorkMemoryCollisionJobRefV0(identity.JobRefBase, 2) {
		t.Fatalf("got=%q base=%q", got, identity.JobRefBase)
	}
}
