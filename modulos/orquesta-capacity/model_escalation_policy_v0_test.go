package orquestacapacity

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeModelEscalationPolicyV0FixturesValidos(t *testing.T) {
	tests := []string{
		"politica_base_por_fase_valida.json",
		"politica_high_por_evidencia_valida.json",
		"politica_xhigh_gate_valida.json",
		"politica_degradacion_quota_home_modelo_valida.json",
	}

	for _, file := range tests {
		t.Run(file, func(t *testing.T) {
			policy, err := DecodeModelEscalationPolicyV0(readModelEscalationPolicyFixtureV0(t, file))
			if err != nil {
				t.Fatalf("fixture valido rechazado: %v", err)
			}
			if policy.SchemaVersion != ModelEscalationPolicySchemaVersionV0 {
				t.Fatalf("schema_version=%q", policy.SchemaVersion)
			}
			if policy.PolicyRef == "" || len(policy.PhaseRules) == 0 || len(policy.DegradationOrder) == 0 {
				t.Fatalf("politica sin reglas minimas: %+v", policy)
			}
		})
	}
}

func TestDecodeModelEscalationPolicyV0FixturesInvalidos(t *testing.T) {
	tests := []struct {
		name string
		file string
		code ModelEscalationPolicyErrorCodeV0
	}{
		{
			name: "xhigh sin gate",
			file: "politica_xhigh_sin_gate_invalida.json",
			code: ErrModelEscalationPolicyXHighGateV0,
		},
		{
			name: "tabla rol modelo",
			file: "politica_tabla_rol_modelo_invalida.json",
			code: ErrModelEscalationPolicyTablaRolModeloV0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeModelEscalationPolicyV0(readModelEscalationPolicyFixtureV0(t, test.file))
			requireModelEscalationPolicyCodeV0(t, err, test.code)
		})
	}
}

func TestDecodeModelEscalationPolicyV0RechazaCamposDesconocidos(t *testing.T) {
	raw := string(readModelEscalationPolicyFixtureV0(t, "politica_base_por_fase_valida.json"))
	raw = strings.Replace(raw, `"policy_ref": "policy-capacity-base:v0",`, `"policy_ref": "policy-capacity-base:v0", "campo_no_contratado": true,`, 1)

	_, err := DecodeModelEscalationPolicyV0([]byte(raw))
	requireModelEscalationPolicyCodeV0(t, err, ErrModelEscalationPolicyJSONInvalidoV0)
}

func TestValidateModelEscalationPolicyV0XHighExigeEvidenciaFresca(t *testing.T) {
	policy := decodeValidModelEscalationPolicyFixtureV0(t, "politica_xhigh_gate_valida.json")
	policy.EvidenceRules[0].Freshness = "stale"

	issues := ValidateModelEscalationPolicyV0(policy)
	requireModelEscalationPolicyIssueCodeV0(t, issues, ErrModelEscalationPolicyXHighGateV0)
}

func TestValidateModelEscalationPolicyV0RechazaRefsNoOpacas(t *testing.T) {
	policy := decodeValidModelEscalationPolicyFixtureV0(t, "politica_base_por_fase_valida.json")
	policy.PolicyRef = "openai-policy:v0"

	issues := ValidateModelEscalationPolicyV0(policy)
	requireModelEscalationPolicyIssueCodeV0(t, issues, ErrModelEscalationPolicyReferenciaNoOpacaV0)
}

func TestValidateModelEscalationPolicyV0RechazaHomeRealEnTexto(t *testing.T) {
	policy := decodeValidModelEscalationPolicyFixtureV0(t, "politica_base_por_fase_valida.json")
	policy.PhaseRules[0].Rationale = "usa /home/alberto como ruta real"

	issues := ValidateModelEscalationPolicyV0(policy)
	requireModelEscalationPolicyIssueCodeV0(t, issues, ErrModelEscalationPolicyInvalidaV0)
}

func TestModelEscalationPolicyV0JSONRoundTripMantieneContrato(t *testing.T) {
	policy := decodeValidModelEscalationPolicyFixtureV0(t, "politica_base_por_fase_valida.json")

	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatalf("marshal policy: %v", err)
	}
	roundTrip, err := DecodeModelEscalationPolicyV0(raw)
	if err != nil {
		t.Fatalf("round-trip invalido: %v", err)
	}
	if !roundTrip.Valid() {
		t.Fatalf("round-trip no valido: %#v", roundTrip.Validate())
	}
}

func decodeValidModelEscalationPolicyFixtureV0(t *testing.T, file string) ModelEscalationPolicyV0 {
	t.Helper()
	policy, err := DecodeModelEscalationPolicyV0(readModelEscalationPolicyFixtureV0(t, file))
	if err != nil {
		t.Fatalf("decode valid fixture %s: %v", file, err)
	}
	return policy
}

func readModelEscalationPolicyFixtureV0(t *testing.T, file string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("docs", "fixtures", "model_escalation_policy_v0", file))
	if err != nil {
		t.Fatalf("read fixture %s: %v", file, err)
	}
	return data
}

func requireModelEscalationPolicyCodeV0(t *testing.T, err error, code ModelEscalationPolicyErrorCodeV0) {
	t.Helper()
	if err == nil {
		t.Fatalf("se esperaba error %q", code)
	}
	var validationErr ModelEscalationPolicyValidationErrorV0
	if !errors.As(err, &validationErr) {
		t.Fatalf("error no es ModelEscalationPolicyValidationErrorV0: %T %v", err, err)
	}
	requireModelEscalationPolicyIssueCodeV0(t, validationErr.Issues, code)
}

func requireModelEscalationPolicyIssueCodeV0(t *testing.T, issues []ModelEscalationPolicyIssueV0, code ModelEscalationPolicyErrorCodeV0) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("no se encontro codigo %q en %#v", code, issues)
}
