package orquestai18ndocs

import "testing"

func TestPublicErrorCatalogV0DeclaraMetadataYCodigosUnicos(t *testing.T) {
	seen := map[string]bool{}
	for _, entry := range PublicErrorCatalogV0() {
		if entry.Code == "" || entry.I18nKey == "" || entry.Boundary == "" || entry.Adapter == "" || entry.Severity == "" {
			t.Fatalf("entrada incompleta: %+v", entry)
		}
		if seen[entry.Code] {
			t.Fatalf("codigo duplicado: %s", entry.Code)
		}
		seen[entry.Code] = true
		if entry.HTTPStatus < 0 || entry.JSONRPCCode > 0 {
			t.Fatalf("mapping invalido: %+v", entry)
		}
	}
	for _, code := range []string{
		PublicErrorMethodNotAllowedV0,
		PublicErrorBodyInvalidV0,
		PublicErrorMCPTransportUnboundV0,
		PublicErrorOperatorPortV0,
		PublicErrorTransportV0,
		PublicErrorRunRefRequiredV0,
	} {
		if !seen[code] {
			t.Fatalf("catalogo no contiene %s", code)
		}
	}
}

func TestNormalizePublicErrorCodeV0NoPropagaErroresNoCatalogados(t *testing.T) {
	if got := NormalizePublicErrorCodeV0(PublicErrorBodyTooLargeV0, PublicErrorExecutorV0); got != PublicErrorBodyTooLargeV0 {
		t.Fatalf("codigo catalogado=%q", got)
	}
	if got := NormalizePublicErrorCodeV0("db password=/tmp/private", PublicErrorExecutorV0); got != PublicErrorExecutorV0 {
		t.Fatalf("codigo desconocido=%q", got)
	}
	if got := NormalizePublicErrorCodeV0("", "fallback_no_catalogado"); got != PublicErrorExecutorV0 {
		t.Fatalf("fallback desconocido=%q", got)
	}
}
