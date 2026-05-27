package publicidentity

import "testing"

func TestNormalizePublicIdentityV0DistingueReadOnlySinIdempotency(t *testing.T) {
	identity := NormalizePublicIdentityV0(PublicIdentityInputV0{
		Mode:          PublicIdentityModeReadOnlyV0,
		RequestID:     " req-read-001 ",
		Mutating:      true,
		CorrelationID: " corr-read-body ",
	})

	if identity.Mode != PublicIdentityModeReadOnlyV0 ||
		identity.CorrelationID != "corr-read-body" ||
		identity.IdempotencyKey != "" ||
		identity.IdempotencyKeySource != PublicIdentitySourceEmptyV0 ||
		len(identity.Issues) != 0 {
		t.Fatalf("identity=%+v", identity)
	}
}

func TestNormalizePublicIdentityV0PriorizaHeadersYMarcaFuentes(t *testing.T) {
	identity := NormalizePublicIdentityV0(PublicIdentityInputV0{
		Mode:                 PublicIdentityModeMutationIdempotentV0,
		RequestID:            "req-public-001",
		CorrelationID:        "corr-body",
		IdempotencyKey:       "idem-body",
		HeaderCorrelationID:  " corr-header ",
		HeaderIdempotencyKey: " idem-header ",
	})

	if identity.CorrelationID != "corr-header" ||
		identity.IdempotencyKey != "idem-header" ||
		identity.CorrelationSource != PublicIdentitySourceHeaderV0 ||
		identity.IdempotencyKeySource != PublicIdentitySourceHeaderV0 ||
		len(identity.Issues) != 0 {
		t.Fatalf("identity=%+v", identity)
	}
}

func TestNormalizePublicIdentityV0BloqueaMutacionNonIdempotentSinKey(t *testing.T) {
	identity := NormalizePublicIdentityV0(PublicIdentityInputV0{
		Mode:      PublicIdentityModeMutationNonIdempotentV0,
		RequestID: "req-public-002",
	})

	if identity.IdempotencyKey != "" ||
		identity.IdempotencyKeySource != PublicIdentitySourceMissingBlockedV0 ||
		len(identity.Issues) != 1 ||
		identity.Issues[0].Code != PublicIdentityIssueIdempotencyKeyRequiredV0 {
		t.Fatalf("identity=%+v", identity)
	}
}
