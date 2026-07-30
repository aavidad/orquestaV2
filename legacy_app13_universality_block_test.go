// Este contrato documenta el NO-GO de APP-13 sin convertir la guía en
// autoridad de producto ni fingir una universalidad todavía inexistente.
package orquesta_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const app13BlockDocument = "docs/reconstruccion/bloqueo_universalidad_app13_2026-07-30.md"

type app13Capability struct {
	ID                  string   `json:"id"`
	Status              string   `json:"status"`
	AcceptanceContracts []string `json:"acceptance_contracts"`
}
type app13Contract struct {
	ID         string   `json:"id"`
	Status     string   `json:"status"`
	Assertions []string `json:"assertions"`
}
type app13Roadmap struct {
	Capabilities []app13Capability `json:"capability_entries"`
	Contracts    []app13Contract   `json:"acceptance_contracts"`
}
type app13AccreditedLedger struct {
	Capabilities []struct {
		ID            string `json:"id"`
		Status        string `json:"status"`
		AcceptanceRef string `json:"acceptance_ref"`
	} `json:"capabilities"`
}

func TestAPP13UniversalityRemainsNoGo(t *testing.T) {
	var roadmap app13Roadmap
	decodeAPP13JSON(t, "product/roadmap.json", &roadmap)
	capability := uniqueAPP13Capability(t, roadmap)
	if capability.Status != "declared" ||
		!reflect.DeepEqual(capability.AcceptanceContracts, []string{"AC-V33-GENERATED-APPS"}) {
		t.Fatalf("APP-13 ya no conserva el bloqueo declarado: %+v", capability)
	}
	contract := uniqueAPP13Contract(t, roadmap)
	if contract.Status != "planned" {
		t.Fatalf("AC-V33 ya no está planificado: %+v", contract)
	}
	for _, scenario := range []struct{ action, language string }{
		{"creat", "Go"}, {"modif", "Go"},
		{"creat", "non-Go"}, {"modif", "non-Go"},
	} {
		if app13AssertionsFixScenario(contract.Assertions, scenario.action, scenario.language) {
			t.Fatalf("el contrato ya fija %s/%s; sustituir este NO-GO", scenario.action, scenario.language)
		}
	}
	wantAssertions := []string{
		"one Go and one non-Go app pass APP-01 through APP-16",
		"same tree or image passes public E2E accessibility security docs and deploy",
		"generated files alone do not count",
	}
	if !reflect.DeepEqual(contract.Assertions, wantAssertions) {
		t.Fatalf("AC-V33 cambió; resolver el roadmap y sustituir este NO-GO: %+v", contract)
	}
	requireAPP13NotAccredited(t)
	requireApplicationTargetNameCanary(t)
	requireAPP13BlockDocument(t)
}

func decodeAPP13JSON(t *testing.T, path string, target any) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, target); err != nil {
		t.Fatal(err)
	}
}
func uniqueAPP13Capability(t *testing.T, roadmap app13Roadmap) app13Capability {
	t.Helper()
	var matches []app13Capability
	for _, entry := range roadmap.Capabilities {
		if entry.ID == "APP-13" {
			matches = append(matches, entry)
		}
	}
	if len(matches) != 1 {
		t.Fatalf("entradas APP-13=%d", len(matches))
	}
	return matches[0]
}

func uniqueAPP13Contract(t *testing.T, roadmap app13Roadmap) app13Contract {
	t.Helper()
	var matches []app13Contract
	for _, entry := range roadmap.Contracts {
		if entry.ID == "AC-V33-GENERATED-APPS" {
			matches = append(matches, entry)
		}
	}
	if len(matches) != 1 {
		t.Fatalf("contratos AC-V33=%d", len(matches))
	}
	return matches[0]
}

func requireAPP13NotAccredited(t *testing.T) {
	t.Helper()
	var ledger app13AccreditedLedger
	decodeAPP13JSON(t, "product/capabilities.json", &ledger)
	for _, capability := range ledger.Capabilities {
		if capability.ID == "APP-13" ||
			capability.AcceptanceRef == "AC-V33-GENERATED-APPS" {
			t.Fatalf("el ledger acreditado ya contiene APP-13/AC-V33: %+v", capability)
		}
	}
}

func requireApplicationTargetNameCanary(t *testing.T) {
	t.Helper()
	err := filepath.WalkDir("internal", func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(content), "type ApplicationTarget ") {
			t.Fatalf("canario nominal ApplicationTarget apareció en %s; no prueba semántica", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func requireAPP13BlockDocument(t *testing.T) {
	t.Helper()
	content, err := os.ReadFile(app13BlockDocument)
	if err != nil {
		t.Fatal(err)
	}
	normalized := strings.Join(strings.Fields(string(content)), " ")
	required := []string{
		"Estado: NO-GO.", "la guía no amplía el contrato superior",
		"`APP-13` está `declared`", "AC-V33-GENERATED-APPS` está `planned`",
		"docs/reconstruccion/ruta_total_100.md", "toda aplicación creada o modificada por Orquesta",
		"propósito y usuarios", "alcance y exclusiones", "entradas y salidas",
		"arquitectura y módulos", "autoridades escritoras",
		"datos, permisos, secretos y efectos", "arranque, diagnóstico, recuperación y parada",
		"contratos y pruebas", "sección ausente, vacía o mal tipada",
		"configuración regional predeterminada, fallback, catálogos, plurales, formatos y superficies visibles",
		"`ApplicationTarget` conserva solo su referencia y hash", "ApplicationTarget durable en AppSpec",
		"compilePlan + compilePlanExtension + dossier", "crear Go, modificar Go, crear no Go, modificar no Go",
		"no se crea otro registro de aplicaciones", "canario nominal",
	}
	for _, fragment := range required {
		if !strings.Contains(normalized, fragment) {
			t.Errorf("falta el ratchet documental %q", fragment)
		}
	}
	if route, readErr := os.ReadFile("docs/reconstruccion/ruta_total_100.md"); readErr != nil ||
		!strings.Contains(string(route), "Dos apps completas creadas por superficies públicas") {
		t.Fatal("la ruta total ya no conserva el bloqueo de dos aplicaciones creadas")
	}
}

func app13AssertionsFixScenario(assertions []string, action, language string) bool {
	for _, assertion := range assertions {
		lower := strings.ToLower(assertion)
		languageMatch := strings.Contains(lower, strings.ToLower(language))
		if language == "Go" {
			languageMatch = strings.Contains(" "+lower, " go")
		}
		if strings.Contains(lower, strings.ToLower(action)) && languageMatch {
			return true
		}
	}
	return false
}
