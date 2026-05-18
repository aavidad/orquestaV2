package orquestafactory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type requestFixtureV0 struct {
	Case     string          `json:"case"`
	Request  json.RawMessage `json:"request"`
	Expected struct {
		Valid       bool   `json:"valid"`
		PublicError string `json:"public_error"`
	} `json:"expected"`
}

func TestValidateAppSpecRequestV0Fixtures(t *testing.T) {
	fixtures := []string{
		"request_minima_valida.json",
		"request_i18n_invalida.json",
		"request_db_directa_invalida.json",
	}
	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			fixture := readRequestFixtureV0(t, name)
			_, issues := DecodeAppSpecRequestV0(fixture.Request)
			if got := len(issues) == 0; got != fixture.Expected.Valid {
				t.Fatalf("valid=%v, want %v, issues=%+v", got, fixture.Expected.Valid, issues)
			}
			if fixture.Expected.PublicError != "" && !hasIssueCodeV0(issues, fixture.Expected.PublicError) {
				t.Fatalf("missing public error %q in %+v", fixture.Expected.PublicError, issues)
			}
		})
	}
}

func TestValidateAppSpecRequestV0RejectsNonHexagonal(t *testing.T) {
	req := validMinimalRequestV0()
	req.PreferenciasTecnicas.Arquitectura = "capas"
	issues := ValidateAppSpecRequestV0(req)
	if !hasIssueCodeV0(issues, ErrOpcionIncompatible) {
		t.Fatalf("expected %s, got %+v", ErrOpcionIncompatible, issues)
	}
}

func TestValidateAppSpecRequestV0RejectsInvalidLocale(t *testing.T) {
	req := validMinimalRequestV0()
	req.Locale = "no es locale"
	issues := ValidateAppSpecRequestV0(req)
	if !hasIssueCodeV0(issues, ErrIdiomaInvalido) {
		t.Fatalf("expected %s, got %+v", ErrIdiomaInvalido, issues)
	}
}

func TestValidateAppSpecRequestV0AcceptsDocumentationType(t *testing.T) {
	req := validMinimalRequestV0()
	req.TipoApp = "documentacion"
	req.RequestKind = RequestKindDocumentarAppV0

	if issues := ValidateAppSpecRequestV0(req); len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
}

func validMinimalRequestV0() AppSpecRequestV0 {
	return AppSpecRequestV0{
		SchemaVersion: AppSpecRequestSchemaV0,
		RequestID:     "req-fty-003-minima",
		Source:        "orquesta-web",
		Locale:        "es-ES",
		Nombre:        "Panel de reservas",
		Objetivo:      "Gestionar solicitudes de reserva.",
		TipoApp:       "web",
	}
}

func readRequestFixtureV0(t *testing.T, name string) requestFixtureV0 {
	t.Helper()
	path := filepath.Join("docs", "fixtures", "app_spec_v0", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var fixture requestFixtureV0
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("decode fixture %s: %v", name, err)
	}
	return fixture
}

func hasIssueCodeV0(issues []ValidationIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func hasIssueFieldV0(issues []ValidationIssue, field string) bool {
	for _, issue := range issues {
		if issue.Field == field {
			return true
		}
	}
	return false
}
