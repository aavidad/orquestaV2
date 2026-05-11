package orquestacapacity

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeCapacityDecisionV0FixturesValidos(t *testing.T) {
	tests := []string{
		"decision_minima_valida.json",
		"decision_xhigh_evidencia_valida.json",
		"decision_cuota_obsoleta_handoff_valida.json",
	}

	for _, file := range tests {
		t.Run(file, func(t *testing.T) {
			decision, err := DecodeCapacityDecisionV0(readCapacityDecisionFixtureV0(t, file))
			if err != nil {
				t.Fatalf("fixture valido rechazado: %v", err)
			}
			if decision.SchemaVersion != CapacityDecisionSchemaVersionV0 {
				t.Fatalf("schema_version=%q", decision.SchemaVersion)
			}
			if decision.Response.Pool.PoolID == "" || decision.Response.Modelo.ModelRef == "" {
				t.Fatalf("decision sin pool/modelo opaco: %+v", decision.Response)
			}
		})
	}
}

func TestDecodeCapacityDecisionV0FixturesInvalidos(t *testing.T) {
	tests := []struct {
		name string
		file string
		code CapacityDecisionErrorCodeV0
	}{
		{
			name: "xhigh sin gate",
			file: "decision_xhigh_sin_evidencia_invalida.json",
			code: ErrEvidenciaInsuficienteParaXHighV0,
		},
		{
			name: "local sin score probado",
			file: "decision_local_sin_score_invalida.json",
			code: ErrLocalSinScoreSuficienteV0,
		},
		{
			name: "proveedor o modelo hardcodeado",
			file: "decision_proveedor_hardcodeado_invalida.json",
			code: ErrReferenciaNoOpacaV0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeCapacityDecisionV0(readCapacityDecisionFixtureV0(t, test.file))
			requireCapacityDecisionCodeV0(t, err, test.code)
		})
	}
}

func TestDecodeCapacityDecisionV0RechazaCamposDesconocidos(t *testing.T) {
	raw := string(readCapacityDecisionFixtureV0(t, "decision_minima_valida.json"))
	raw = strings.Replace(raw, `"schema_version": "capacity_decision.v0",`, `"schema_version": "capacity_decision.v0", "campo_no_contratado": true,`, 1)

	_, err := DecodeCapacityDecisionV0([]byte(raw))
	requireCapacityDecisionCodeV0(t, err, ErrCapacityDecisionJSONInvalidoV0)
}

func TestValidateCapacityDecisionV0CuotaObsoletaFuerzaDegradacionOHandoff(t *testing.T) {
	decision := decodeValidCapacityDecisionFixtureV0(t, "decision_minima_valida.json")
	decision.Response.Cuota.Freshness = "obsolete"
	decision.Response.Handoff = "none"
	decision.Response.Degradacion.Action = "none"
	decision.Response.Degradacion.Reason = "sin bloqueo declarado"

	issues := ValidateCapacityDecisionV0(decision)
	requireCapacityDecisionIssueCodeV0(t, issues, ErrCuotaObsoletaV0)
}

func TestValidateCapacityDecisionV0VentanaInsuficienteExigeHandoff(t *testing.T) {
	decision := decodeValidCapacityDecisionFixtureV0(t, "decision_minima_valida.json")
	remaining := 10
	decision.Response.Cuota.RemainingSeconds = &remaining
	decision.Response.Handoff = "none"
	decision.Response.Degradacion.Action = "none"
	decision.Response.Degradacion.Reason = "ventana ignorada"

	issues := ValidateCapacityDecisionV0(decision)
	requireCapacityDecisionIssueCodeV0(t, issues, ErrHandoffRequeridoV0)
}

func TestCapacityDecisionV0JSONRoundTripMantieneContrato(t *testing.T) {
	decision := decodeValidCapacityDecisionFixtureV0(t, "decision_minima_valida.json")

	raw, err := json.Marshal(decision)
	if err != nil {
		t.Fatalf("marshal decision: %v", err)
	}
	roundTrip, err := DecodeCapacityDecisionV0(raw)
	if err != nil {
		t.Fatalf("round-trip invalido: %v", err)
	}
	if !roundTrip.Valid() {
		t.Fatalf("round-trip no valido: %#v", roundTrip.Validate())
	}
}

func decodeValidCapacityDecisionFixtureV0(t *testing.T, file string) CapacityDecisionV0 {
	t.Helper()
	decision, err := DecodeCapacityDecisionV0(readCapacityDecisionFixtureV0(t, file))
	if err != nil {
		t.Fatalf("decode valid fixture %s: %v", file, err)
	}
	return decision
}

func readCapacityDecisionFixtureV0(t *testing.T, file string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("docs", "fixtures", "capacity_decision_v0", file))
	if err != nil {
		t.Fatalf("read fixture %s: %v", file, err)
	}
	return data
}

func requireCapacityDecisionCodeV0(t *testing.T, err error, code CapacityDecisionErrorCodeV0) {
	t.Helper()
	if err == nil {
		t.Fatalf("se esperaba error %q", code)
	}
	var validationErr CapacityDecisionValidationErrorV0
	if !errors.As(err, &validationErr) {
		t.Fatalf("error no es CapacityDecisionValidationErrorV0: %T %v", err, err)
	}
	requireCapacityDecisionIssueCodeV0(t, validationErr.Issues, code)
}

func requireCapacityDecisionIssueCodeV0(t *testing.T, issues []CapacityDecisionIssueV0, code CapacityDecisionErrorCodeV0) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("no se encontro codigo %q en %#v", code, issues)
}
