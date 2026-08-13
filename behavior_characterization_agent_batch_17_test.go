// Este contrato conserva una propuesta trazable; no crea trabajo, cierre, admisión ni estado canónico.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

const behaviorBatch17Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_17.jsonl"

func TestBehaviorCharacterizationAgentBatch17(t *testing.T) {
	fixture := readBehaviorBatch17File(t, behaviorBatch17Path("ORQUESTA_BATCH17_FIXTURE", behaviorBatch17Fixture))
	ledger := readBehaviorBatch17File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch17File(t, "product/roadmap.json")
	if err := validateBehaviorBatch17(fixture, ledger, roadmap, behaviorBatch17PreviousFixtures(t)); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch17RejectsMutations(t *testing.T) {
	fixture := readBehaviorBatch17File(t, behaviorBatch17Path("ORQUESTA_BATCH17_FIXTURE", behaviorBatch17Fixture))
	ledger := readBehaviorBatch17File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch17File(t, "product/roadmap.json")
	previous := behaviorBatch17PreviousFixtures(t)
	mutations := []struct{ name, old, replacement string }{
		{"autoridad canónica", `"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical_ledger"`},
		{"cambio canónico", `"canonical_state_change":false`, `"canonical_state_change":true`},
		{"crea trabajo", `"creates_work_item":false`, `"creates_work_item":true`},
		{"cierra capability", `"closes_capability":false`, `"closes_capability":true`},
		{"afirma acreditación", `"claims_accreditation":false`, `"claims_accreditation":true`},
		{"revisión aceptada", `"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`},
		{"capability incorrecta", `"capability_id":"UI-02"`, `"capability_id":"UI-03"`},
		{"entrada de lote previo", `"TASKENTRY-e0f2506885afda6217d4334c"`, `"TASKENTRY-73299fa04483a25a8f27571e"`},
		{"falsa independencia", `no son evidencia independiente`, `son evidencia independiente`},
		{"campo desconocido", `{"schema_version":1`, `{"unknown":true,"schema_version":1`},
		{"clave duplicada", `{"schema_version":1`, `{"schema_version":2,"schema_version":1`},
		{"valor posterior", `}` + "\n", `} {}` + "\n"},
		{"JSON no canónico", `{"schema_version":1`, ` {"schema_version":1`},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed := bytes.Replace(fixture, []byte(mutation.old), []byte(mutation.replacement), 1)
			if bytes.Equal(changed, fixture) {
				t.Fatal("la mutación no cambió el fixture")
			}
			if err := validateBehaviorBatch17(changed, ledger, roadmap, previous); err == nil {
				t.Fatal("la mutación autoritativa, reutilizada o ambigua fue aceptada")
			}
		})
	}
	t.Run("LF final ausente", func(t *testing.T) {
		if err := validateBehaviorBatch17(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous); err == nil {
			t.Fatal("fixture sin LF final aceptado")
		}
	})
	t.Run("roadmap acredita TLS-01 sin evidencia", func(t *testing.T) {
		changed := mutateBehaviorBatch17RoadmapStatus(t, roadmap, "TLS-01", "accredited")
		if err := validateBehaviorBatch17(fixture, ledger, changed, previous); err == nil {
			t.Fatal("roadmap mutado fue aceptado")
		}
	})
	t.Run("roadmap degrada ORC-14", func(t *testing.T) {
		changed := mutateBehaviorBatch17RoadmapStatus(t, roadmap, "ORC-14", "declared")
		if err := validateBehaviorBatch17(fixture, ledger, changed, previous); err == nil {
			t.Fatal("roadmap mutado fue aceptado")
		}
	})
}

func TestBehaviorCharacterizationAgentBatch17QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(readBehaviorBatch17File(t,
		behaviorBatch17Path("ORQUESTA_BATCH17_FIXTURE", behaviorBatch17Fixture)))
	if err != nil {
		t.Fatal(err)
	}
	qualifiers := []string{"declar", "document", "registr", "inform", "históric", "no consta", "código legacy", "auditoría"}
	for _, record := range records {
		for _, claims := range [][]string{record.Worked, record.Failed, record.Attempts} {
			for _, claim := range claims {
				normalized := strings.ToLower(claim)
				qualified := false
				for _, qualifier := range qualifiers {
					qualified = qualified || strings.Contains(normalized, qualifier)
				}
				if !qualified {
					t.Errorf("afirmación histórica sin calificar: %q", claim)
				}
			}
		}
	}
}

func TestBehaviorCharacterizationAgentBatch17CounterreviewedMappingsMatchSnapshot(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(readBehaviorBatch17File(t,
		behaviorBatch17Path("ORQUESTA_BATCH17_FIXTURE", behaviorBatch17Fixture)))
	if err != nil {
		t.Fatal(err)
	}
	assessments := decodeBehaviorBatch17JSONLMaps(t,
		behaviorBatch17Path("ORQUESTA_BATCH17_ASSESSMENTS", "product/knowledge/legacy_reuse_assessments_v1.jsonl"))
	if os.Getenv("ORQUESTA_BATCH17_ASSESSMENTS") == "" {
		assessments = legacyBatchAssessments(t, 17)
	}
	if len(assessments) != 5 {
		t.Fatalf("assessments=%d", len(assessments))
	}
	var linksRoot map[string]any
	linksRaw := legacyBatchMappingsJSON(t, 17)
	if override := strings.TrimSpace(os.Getenv("ORQUESTA_BATCH17_LINKS")); override != "" {
		linksRaw = readBehaviorBatch17File(t, override)
	}
	if err := json.Unmarshal(linksRaw, &linksRoot); err != nil {
		t.Fatal(err)
	}
	mappings := behaviorBatch17ObjectSlice(t, linksRoot["mappings"])
	if len(mappings) != 5 || linksRoot["authority"] != "advisory_not_capability_state" ||
		linksRoot["snapshot_census_sha256"] != "sha256:5e8f30dc3c6640c57f595e3757cc6e07ba4be6eb36a6bcdd094152a2967fddfc" {
		t.Fatal("root de mappings inválido")
	}
	index := decodeBehaviorBatch17JSONLMaps(t, "product/knowledge/legacy_go_function_snapshot_v1.jsonl")
	indexByOccurrence := make(map[string]map[string]any, len(index))
	for _, item := range index {
		indexByOccurrence[behaviorBatch17String(t, item["occurrence_ref"])] = item
	}
	entriesByBehavior := make(map[string]map[string]bool, len(records))
	for _, record := range records {
		set := make(map[string]bool, len(record.EntryRefs)+len(record.Evidence))
		for _, ref := range record.EntryRefs {
			set[ref] = true
		}
		for _, evidence := range record.Evidence {
			set[evidence.SourceRef] = true
		}
		entriesByBehavior[record.Ref] = set
	}
	mappingByRef := make(map[string]map[string]any, len(mappings))
	for _, mapping := range mappings {
		mappingByRef[behaviorBatch17String(t, mapping["characterization_ref"])] = mapping
	}
	expectedAssessment := map[string][3]string{
		"BEHAVIOR-AGENT-BATCH-17-REAL-MCP-SERVER":        {"partial_or_mixed", "v2_existing", "exact_or_primary"},
		"BEHAVIOR-AGENT-BATCH-17-CAUSAL-MAILBOX":         {"partial_or_mixed", "v2_existing", "exact_or_primary"},
		"BEHAVIOR-AGENT-BATCH-17-CANONICAL-TOOL-SPEC":    {"partial_or_mixed", "contract", "not_implemented"},
		"BEHAVIOR-AGENT-BATCH-17-MINIMAL-CONTEXT-BUNDLE": {"partial_or_mixed", "contract", "not_implemented"},
		"BEHAVIOR-AGENT-BATCH-17-ISOLATED-REAL-SMOKE":    {"partial_or_mixed", "contract", "not_implemented"},
	}
	seenOccurrences := make(map[string]bool)
	linkCount := 0
	for _, assessment := range assessments {
		ref := behaviorBatch17String(t, assessment["characterization_ref"])
		want, exists := expectedAssessment[ref]
		if !exists || assessment["authority"] != "advisory_not_capability_state" ||
			assessment["review_state"] != "independent_bootstrap_counterreview_completed_advisory" ||
			assessment["historical_green_state"] != want[0] || assessment["reuse_kind"] != want[1] || assessment["exact_v2_fit"] != want[2] {
			t.Fatalf("assessment inválido: %s", ref)
		}
		mapping, mapped := mappingByRef[ref]
		if !mapped {
			t.Fatalf("mapping ausente: %s", ref)
		}
		functionMapping := behaviorBatch17Object(t, assessment["function_mapping"])
		assessmentLinks := behaviorBatch17ObjectSlice(t, functionMapping["links"])
		mappingLinks := behaviorBatch17ObjectSlice(t, mapping["links"])
		if functionMapping["state"] != "linked_exact" || len(mappingLinks) < 1 || len(mappingLinks) > 3 ||
			!reflect.DeepEqual(assessmentLinks, mappingLinks) || functionMapping["rationale"] != mapping["rationale"] {
			t.Fatalf("mapping incoherente: %s", ref)
		}
		for _, link := range mappingLinks {
			occurrence := behaviorBatch17String(t, link["occurrence_ref"])
			indexed, present := indexByOccurrence[occurrence]
			if !present || seenOccurrences[occurrence] {
				t.Fatalf("occurrence ausente o duplicada: %s", occurrence)
			}
			seenOccurrences[occurrence] = true
			for _, field := range []string{"declaration_ref", "variant_ref", "source_root", "source_path", "git_blob_oid", "blob_digest", "package_path", "package_name", "symbol_kind", "name", "receiver", "signature", "source_sha256", "ast_sha256", "body_sha256", "start_line", "end_line"} {
				if !reflect.DeepEqual(link[field], indexed[field]) {
					t.Fatalf("metadata drift %s %s", occurrence, field)
				}
			}
			if link["review_state"] != "bootstrap_symbol_mapping_reviewed_advisory" || link["body"] != nil || link["canonical_source"] != nil {
				t.Fatalf("link no advisory o contiene cuerpo: %s", occurrence)
			}
			relation := behaviorBatch17String(t, link["relation"])
			if relation != "primary_implementation" && relation != "supporting_mechanism" && relation != "negative_example" {
				t.Fatalf("relation inválida: %s", relation)
			}
			belongs := false
			for _, evidenceRef := range behaviorBatch17StringSlice(t, link["evidence_refs"]) {
				belongs = belongs || entriesByBehavior[ref][evidenceRef]
			}
			if !belongs {
				t.Fatalf("link sin evidencia de su conducta: %s", occurrence)
			}
			linkCount++
		}
		delete(expectedAssessment, ref)
	}
	if len(expectedAssessment) != 0 || len(mappingByRef) != 5 || linkCount != 11 {
		t.Fatalf("cobertura mapping inesperada: pending=%d mappings=%d links=%d", len(expectedAssessment), len(mappingByRef), linkCount)
	}
}

func behaviorBatch17PreviousFixtures(t *testing.T) [][]byte {
	t.Helper()
	result := make([][]byte, 0, 14)
	for batch := 1; batch <= 14; batch++ {
		result = append(result, readBehaviorBatch17File(t,
			fmt.Sprintf("product/traceability/fixtures/behavior_characterization_agent_batch_%02d.jsonl", batch)))
	}
	return result
}

func behaviorBatch17Path(environment, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(environment)); value != "" {
		return value
	}
	return fallback
}

func readBehaviorBatch17File(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func mutateBehaviorBatch17RoadmapStatus(t *testing.T, raw []byte, capabilityID, status string) []byte {
	t.Helper()
	var roadmap map[string]any
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		t.Fatal(err)
	}
	for _, value := range behaviorBatch17ObjectSlice(t, roadmap["capability_entries"]) {
		if value["id"] == capabilityID {
			value["status"] = status
			changed, err := json.Marshal(roadmap)
			if err != nil {
				t.Fatal(err)
			}
			return changed
		}
	}
	t.Fatalf("capability ausente: %s", capabilityID)
	return nil
}

func decodeBehaviorBatch17JSONLMaps(t *testing.T, path string) []map[string]any {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var result []map[string]any
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var item map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &item); err != nil {
			t.Fatal(err)
		}
		result = append(result, item)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return result
}

func behaviorBatch17ObjectSlice(t *testing.T, value any) []map[string]any {
	t.Helper()
	values, ok := value.([]any)
	if !ok {
		t.Fatalf("array JSON esperado: %T", value)
	}
	result := make([]map[string]any, 0, len(values))
	for _, item := range values {
		result = append(result, behaviorBatch17Object(t, item))
	}
	return result
}

func behaviorBatch17Object(t *testing.T, value any) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("objeto JSON esperado: %T", value)
	}
	return result
}

func behaviorBatch17String(t *testing.T, value any) string {
	t.Helper()
	result, ok := value.(string)
	if !ok || strings.TrimSpace(result) == "" {
		t.Fatalf("string JSON esperada: %T", value)
	}
	return result
}

func behaviorBatch17StringSlice(t *testing.T, value any) []string {
	t.Helper()
	values, ok := value.([]any)
	if !ok {
		t.Fatalf("array string esperado: %T", value)
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, behaviorBatch17String(t, value))
	}
	return result
}
