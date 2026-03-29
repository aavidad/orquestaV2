package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestConstruirResumePayloadAdoptadoNoReanidaEnvelopePrevio(t *testing.T) {
	prev := `{"project_context":{"slug":"demo"},"governance_catalog":{"hash":"abc123"}}`
	payload, err := construirResumePayloadAdoptado(prev, apiAgenteAdoptarContextoRequest{
		CWD:               "/tmp/demo",
		Branch:            "feature/demo",
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-1",
		Nota:              "ignora la ruta historica",
	}, &db.Proyecto{
		ID:      7,
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: "/tmp/orquesta",
	}, map[string]any{
		"tipo_agente": "programador",
		"hash":        "def456",
	})
	if err != nil {
		t.Fatalf("construirResumePayloadAdoptado: %v", err)
	}
	if strings.Contains(payload, `"resume_payload_prev"`) || strings.Contains(payload, `"resume_previo"`) {
		t.Fatalf("el payload adoptado no deberia reanidarse: %s", payload)
	}
	var envelope map[string]any
	if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
		t.Fatalf("payload invalido: %v", err)
	}
	if _, ok := envelope["project_context"].(map[string]any); !ok {
		t.Fatalf("falta project_context: %+v", envelope)
	}
	governanceCatalog, ok := envelope["governance_catalog"].(map[string]any)
	if !ok || governanceCatalog["hash"] != "def456" {
		t.Fatalf("governance_catalog inesperado: %+v", envelope)
	}
	adopted, ok := envelope["adopted_context"].(map[string]any)
	if !ok || adopted["branch"] != "feature/demo" {
		t.Fatalf("adopted_context inesperado: %+v", envelope)
	}
}

func TestConstruirResumePayloadAdoptadoDescartaContextoAjeno(t *testing.T) {
	prev := `{"project_context":{"slug":"orquestador","ruta_abs":"/tmp/orquestador"},"adopted_context":{"cwd":"/tmp/orquestador"},"checkpoint":{"cwd":"/tmp/orquestador"},"persist":"ok"}`
	payload, err := construirResumePayloadAdoptado(prev, apiAgenteAdoptarContextoRequest{
		CWD:         "/tmp/orquesta",
		Herramienta: "codex-cli",
		Nota:        "adopcion limpia",
	}, &db.Proyecto{
		ID:      9,
		Slug:    "orquesta",
		Nombre:  "Orquesta",
		RutaAbs: "/tmp/orquesta",
	}, map[string]any{
		"tipo_agente": "programador",
		"hash":        "xyz789",
	})
	if err != nil {
		t.Fatalf("construirResumePayloadAdoptado: %v", err)
	}
	if strings.Contains(payload, "orquestador") || strings.Contains(payload, `"checkpoint"`) {
		t.Fatalf("el payload adoptado no deberia conservar contexto ajeno: %s", payload)
	}
	if !strings.Contains(payload, `"slug":"orquesta"`) || !strings.Contains(payload, `"persist":"ok"`) {
		t.Fatalf("payload adoptado incompleto: %s", payload)
	}
}

func TestConstruirResumenContextoAdoptadoDescartaContinuidadAjena(t *testing.T) {
	req := apiAgenteAdoptarContextoRequest{
		CWD:  "/tmp/orquesta",
		Nota: "adopcion limpia",
	}
	got := construirResumenContextoAdoptado(
		"Retoma el trabajo del agente Codex4. en el proyecto orquestador. Contexto adoptado por Orquesta sobre orquestador (/tmp/orquestador). 4 propuesta(s) abiertas.",
		"Frente asignado: humo E2E",
		"Catálogo efectivo abc123 (3 reglas, 1 skills, 1 workflows)",
		&db.Proyecto{Slug: "orquesta", RutaAbs: "/tmp/orquesta"},
		req,
	)
	if strings.Contains(got, "orquestador") {
		t.Fatalf("resumen adoptado contaminado: %s", got)
	}
	for _, token := range []string{
		"Frente asignado: humo E2E",
		"Contexto adoptado por Orquesta sobre orquesta",
		"Catálogo efectivo abc123",
	} {
		if !strings.Contains(got, token) {
			t.Fatalf("falta %s en el resumen adoptado: %s", token, got)
		}
	}
}

func TestConstruirResumePayloadAdoptadoPriorizaRutaActualSobreRutaHistorica(t *testing.T) {
	rutaActual := filepath.Join(t.TempDir(), "orquesta")
	if err := os.MkdirAll(filepath.Join(rutaActual, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir ruta actual: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaActual, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	payload, err := construirResumePayloadAdoptado("", apiAgenteAdoptarContextoRequest{
		CWD:         filepath.Join(rutaActual, "cmd"),
		Herramienta: "codex-cli",
		Nota:        "ruta actual",
	}, &db.Proyecto{
		ID:      7,
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: "/tmp/historico/orquestador",
	}, nil)
	if err != nil {
		t.Fatalf("construirResumePayloadAdoptado: %v", err)
	}
	var envelope map[string]any
	if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
		t.Fatalf("payload invalido: %v", err)
	}
	projectContext, ok := envelope["project_context"].(map[string]any)
	if !ok {
		t.Fatalf("project_context ausente: %+v", envelope)
	}
	if got, _ := projectContext["ruta_abs"].(string); got != rutaActual {
		t.Fatalf("ruta_abs inesperada: got=%q want=%q", got, rutaActual)
	}
}
