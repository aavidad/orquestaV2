package orquestadomainworkfile

import (
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestFileDomainWorkJobCreatorV0ReparaColisionDeJobRef(t *testing.T) {
	key := domainWorkFileJobKeyV0{
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
	creator := &FileDomainWorkJobCreatorV0{
		jobsByRef: map[string]domainWorkFileJobKeyV0{
			identity.JobRefBase: {
				DomainRef:      "domain-other",
				IdempotencyKey: "idem-other",
			},
		},
	}
	got := creator.nextJobRefLockedV0(key)
	if got == identity.JobRefBase ||
		got != domainWorkFileCollisionJobRefV0(identity.JobRefBase, 2) {
		t.Fatalf("got=%q base=%q", got, identity.JobRefBase)
	}
}
