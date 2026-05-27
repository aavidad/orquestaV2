package orquestamcp

import "testing"

func TestNormalizeMCPPublicMutationIdentityV0DerivaCorrelacionEIdempotencia(t *testing.T) {
	identity := NormalizeMCPPublicMutationIdentityV0(MCPPublicMutationIdentityInputV0{
		RequestID: " request-ref-public-001 ",
		Mutating:  true,
	})

	if identity.RequestID != "request-ref-public-001" ||
		identity.CorrelationID != "request-ref-public-001" ||
		identity.IdempotencyKey != "idem-request-ref-public-001" {
		t.Fatalf("identity=%+v", identity)
	}
	if !identity.CorrelationIDDerived || !identity.IdempotencyKeyDerived || len(identity.Issues) != 0 {
		t.Fatalf("derivacion inesperada: %+v", identity)
	}
}

func TestNormalizeMCPPublicMutationIdentityV0UsaHeadersCanonicos(t *testing.T) {
	identity := NormalizeMCPPublicMutationIdentityV0(MCPPublicMutationIdentityInputV0{
		RequestID:            "request-ref-public-002",
		CorrelationID:        "corr-body",
		IdempotencyKey:       "idem-body",
		HeaderCorrelationID:  "corr-header",
		HeaderIdempotencyKey: "idem-header",
		Mutating:             true,
	})

	if identity.CorrelationID != "corr-header" || identity.IdempotencyKey != "idem-header" {
		t.Fatalf("identity=%+v", identity)
	}
	if identity.CorrelationIDDerived || identity.IdempotencyKeyDerived || len(identity.Issues) != 0 {
		t.Fatalf("flags inesperados: %+v", identity)
	}
}

func TestNormalizeMCPPublicMutationIdentityV0BloqueaMutacionSinKeyEstable(t *testing.T) {
	identity := NormalizeMCPPublicMutationIdentityV0(MCPPublicMutationIdentityInputV0{Mutating: true})

	if len(identity.Issues) != 1 ||
		identity.Issues[0].Code != MCPPublicMutationIssueIdempotencyKeyRequiredV0 {
		t.Fatalf("issues=%+v", identity.Issues)
	}
}
