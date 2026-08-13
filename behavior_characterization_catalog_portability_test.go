// Estas utilidades hacen portables los contratos de lotes contra el catálogo consolidado.
package orquesta_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const legacyBehaviorSnapshotSHA = "sha256:5e8f30dc3c6640c57f595e3757cc6e07ba4be6eb36a6bcdd094152a2967fddfc"

func legacyBatchAssessments(t *testing.T, batch int) []map[string]any {
	t.Helper()
	raw, err := os.ReadFile("product/knowledge/legacy_reuse_assessments_v1.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	prefix := fmt.Sprintf("BEHAVIOR-AGENT-BATCH-%02d-", batch)
	var result []map[string]any
	for _, line := range bytes.Split(bytes.TrimSuffix(raw, []byte("\n")), []byte("\n")) {
		var item map[string]any
		if err := json.Unmarshal(line, &item); err != nil {
			t.Fatal(err)
		}
		ref, _ := item["characterization_ref"].(string)
		if strings.HasPrefix(ref, prefix) {
			result = append(result, item)
		}
	}
	return result
}

func TestLegacyGeneralFunctionCatalogIsPortableAndConsumerNeutral(t *testing.T) {
	script, err := filepath.Abs("scripts/consultar_catalogo_funciones_legacy_v1.sh")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(script, "--name", "HandleCommandV0", "--json")
	command.Dir = t.TempDir()
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("consultar catálogo desde cwd externo: %v\n%s", err, output)
	}
	var entries []map[string]any
	if err := json.Unmarshal(output, &entries); err != nil {
		t.Fatalf("decodificar catálogo portable: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("se esperaba una función exacta; got=%d", len(entries))
	}
	entry := entries[0]
	function, ok := entry["function"].(map[string]any)
	if !ok || function["name"] != "HandleCommandV0" ||
		!strings.HasPrefix(function["signature"].(string), "func HandleCommandV0") {
		t.Fatalf("identidad o firma inesperadas: %#v", function)
	}
	if entry["contains_body"] != false || entry["code_reuse_authorized"] != false ||
		entry["consumer_application_decision"] != nil {
		t.Fatalf("el catálogo general no debe incluir cuerpo ni autorizar/decidir: %#v", entry)
	}
	if _, exists := entry["capability_id"]; exists {
		t.Fatal("el catálogo general no debe exigir una capability consumidora")
	}
	if _, exists := entry["v2"]; exists {
		t.Fatal("el catálogo general no debe imponer el estado de OrquestaV2")
	}
	license, ok := entry["license"].(map[string]any)
	if !ok || license["permits_code_reuse"] != false || license["author"] != nil {
		t.Fatalf("licencia o autoría no deben fabricarse: %#v", license)
	}
	behaviors, ok := entry["behavior_assessments"].([]any)
	if !ok || len(behaviors) == 0 {
		t.Fatal("la función enlazada debe exponer mecanismo, aciertos y fallos")
	}
}

func legacyBatchMappingsJSON(t *testing.T, batch int) []byte {
	t.Helper()
	assessments := legacyBatchAssessments(t, batch)
	mappings := make([]map[string]any, 0, len(assessments))
	for _, assessment := range assessments {
		mapping := make(map[string]any)
		for key, value := range assessment["function_mapping"].(map[string]any) {
			mapping[key] = value
		}
		mapping["characterization_ref"] = assessment["characterization_ref"]
		mappings = append(mappings, mapping)
	}
	root := map[string]any{
		"schema_version":         1,
		"batch_ref":              fmt.Sprintf("BEHAVIOR-AGENT-BATCH-%02d", batch),
		"authority":              "advisory_not_capability_state",
		"snapshot_census_sha256": legacyBehaviorSnapshotSHA,
		"mappings":               mappings,
	}
	raw, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
