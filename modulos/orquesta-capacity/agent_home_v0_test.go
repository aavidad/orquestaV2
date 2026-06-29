package orquestacapacity

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeAgentHomeV0FixturesValidos(t *testing.T) {
	for _, file := range []string{
		"home_concurrencia_agotada_valido.json",
		"home_cuota_obsoleta_valido.json",
		"home_local_sin_oauth_valido.json",
	} {
		t.Run(file, func(t *testing.T) {
			home, err := DecodeAgentHomeV0(readAgentHomeFixtureV0(t, file))
			if err != nil {
				t.Fatalf("fixture valido rechazado: %v", err)
			}
			if home.SchemaVersion != AgentHomeSchemaVersionV0 || home.HomeRef == "" || home.AccountRef == "" {
				t.Fatalf("home incompleto: %+v", home)
			}
			if home.CredentialKind == "local_runtime" && home.CredentialRef != "" {
				t.Fatalf("home local no debe requerir oauth: %+v", home)
			}
		})
	}
}

func TestDecodeAgentHomeCollectionV0MultiHomeOAuthValido(t *testing.T) {
	collection, err := DecodeAgentHomeCollectionV0(readAgentHomeFixtureV0(t, "multi_home_oauth_valido.json"))
	if err != nil {
		t.Fatalf("collection valida rechazada: %v", err)
	}
	if len(collection.Homes) != 2 {
		t.Fatalf("homes=%d", len(collection.Homes))
	}
	first := collection.Homes[0]
	second := collection.Homes[1]
	if first.LogicalAgentRef != second.LogicalAgentRef ||
		first.ProviderRef != second.ProviderRef ||
		first.HomeRef == second.HomeRef ||
		first.AccountRef == second.AccountRef ||
		first.CredentialRef == second.CredentialRef {
		t.Fatalf("multi-home no separa refs opacas: %+v", collection.Homes)
	}
}

func TestDecodeAgentHomeV0FixturesInvalidos(t *testing.T) {
	tests := []struct {
		name string
		file string
		code CapacityDecisionErrorCodeV0
	}{
		{name: "email y token", file: "home_email_token_invalido.json", code: ErrReferenciaNoOpacaV0},
		{name: "proveedor hardcodeado", file: "home_provider_hardcodeado_invalido.json", code: ErrReferenciaNoOpacaV0},
		{name: "reserved mayor max", file: "home_reserved_mayor_max_invalido.json", code: ErrConcurrenciaHomeAgotadaV0},
		{name: "ruta real", file: "home_ruta_real_invalida.json", code: ErrRutaHomeRealDetectadaV0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeAgentHomeV0(readAgentHomeFixtureV0(t, test.file))
			requireAgentHomeCodeV0(t, err, test.code)
		})
	}
}

func readAgentHomeFixtureV0(t *testing.T, file string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("docs", "fixtures", "agent_home_v0", file))
	if err != nil {
		t.Fatalf("read fixture %s: %v", file, err)
	}
	return data
}

func requireAgentHomeCodeV0(t *testing.T, err error, code CapacityDecisionErrorCodeV0) {
	t.Helper()
	if err == nil {
		t.Fatalf("se esperaba error %q", code)
	}
	var validationErr AgentHomeValidationErrorV0
	if !errors.As(err, &validationErr) {
		t.Fatalf("error no es AgentHomeValidationErrorV0: %T %v", err, err)
	}
	requireCapacityDecisionIssueCodeV0(t, validationErr.Issues, code)
}
