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

func TestBehaviorCharacterizationAgentBatch13(t *testing.T) {
	fixture := readBehaviorBatch13File(t, behaviorBatch13Path("ORQUESTA_BATCH13_FIXTURE", "product/traceability/fixtures/behavior_characterization_agent_batch_13.jsonl"))
	ledger := readBehaviorBatch13File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch13File(t, "product/roadmap.json")
	if err := validateBehaviorBatch13(fixture, ledger, roadmap, behaviorBatch13PreviousFixtures(t)); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch13RejectsMutations(t *testing.T) {
	fixture := readBehaviorBatch13File(t, behaviorBatch13Path("ORQUESTA_BATCH13_FIXTURE", "product/traceability/fixtures/behavior_characterization_agent_batch_13.jsonl"))
	ledger := readBehaviorBatch13File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch13File(t, "product/roadmap.json")
	previous := behaviorBatch13PreviousFixtures(t)
	mutations := []struct{ name, old, replacement string }{
		{"autoridad canónica", `"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical_ledger"`},
		{"cambio canónico", `"canonical_state_change":false`, `"canonical_state_change":true`},
		{"crea trabajo", `"creates_work_item":false`, `"creates_work_item":true`},
		{"cierra capacidad", `"closes_capability":false`, `"closes_capability":true`},
		{"afirma acreditación", `"claims_accreditation":false`, `"claims_accreditation":true`},
		{"revisión aceptada", `"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`},
		{"disposición evaluada", `"disposition":"not_evaluated"`, `"disposition":"accepted"`},
		{"capability incorrecta", `"capability_id":"ORC-01"`, `"capability_id":"ORC-02"`},
		{"ref de lote previo", `"TASKENTRY-09eb087ac1ca2b7a70615d5b"`, `"TASKENTRY-73299fa04483a25a8f27571e"`},
		{"falsa independencia", `no son evidencia independiente`, `son evidencia independiente`},
		{"ancla alterada", `vista descartable`, `vista inventada`},
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
			if err := validateBehaviorBatch13(changed, ledger, roadmap, previous); err == nil {
				t.Fatal("la mutación autoritativa, duplicada o ambigua fue aceptada")
			}
		})
	}
	for _, size := range []int{3, 5} {
		t.Run(fmt.Sprintf("refs_%d", size), func(t *testing.T) {
			changed := resizeBehaviorBatch13Record(t, fixture, "entrada_tick_derivada_de_snapshot_y_candidatos", size)
			if err := validateBehaviorBatch13(changed, ledger, roadmap, previous); err == nil {
				t.Fatalf("conducta con %d refs aceptada", size)
			}
		})
	}
	t.Run("intercambio semántico", func(t *testing.T) {
		changed := swapBehaviorBatch13Evidence(t, fixture,
			"candidatos_validados_desde_plan_sin_inventar_refs", "ciclo_acotado_por_puertos_y_parada_en_outbox")
		if err := validateBehaviorBatch13(changed, ledger, roadmap, previous); err == nil {
			t.Fatal("el intercambio entre conductas fue aceptado")
		}
	})
	t.Run("LF final ausente", func(t *testing.T) {
		if err := validateBehaviorBatch13(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous); err == nil {
			t.Fatal("fixture sin LF final aceptado")
		}
	})
	t.Run("roadmap degrada ORC-01", func(t *testing.T) {
		changed := mutateBehaviorBatch13RoadmapStatus(t, roadmap, "declared")
		if err := validateBehaviorBatch13(fixture, ledger, changed, previous); err == nil {
			t.Fatal("roadmap mutado fue aceptado")
		}
	})
}

func TestBehaviorCharacterizationAgentBatch13QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(readBehaviorBatch13File(t,
		behaviorBatch13Path("ORQUESTA_BATCH13_FIXTURE", "product/traceability/fixtures/behavior_characterization_agent_batch_13.jsonl")))
	if err != nil {
		t.Fatal(err)
	}
	qualifiers := []string{"declar", "document", "históric", "no consta", "no demuestr", "no registra", "no constituyen", "no equivale", "código histórico", "propia fuente", "reintroduciría", "prohibid"}
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

func TestBehaviorCharacterizationAgentBatch13AdvisoryMappingsMatchSnapshot(t *testing.T) {
	fixturePath := behaviorBatch13Path("ORQUESTA_BATCH13_FIXTURE", "product/traceability/fixtures/behavior_characterization_agent_batch_13.jsonl")
	assessmentPath := behaviorBatch13Path("ORQUESTA_BATCH13_ASSESSMENTS", "product/knowledge/legacy_reuse_assessments_v1.jsonl")
	records, err := decodeBehaviorBatchRecords(readBehaviorBatch13File(t, fixturePath))
	if err != nil {
		t.Fatal(err)
	}
	allAssessments := decodeBehaviorBatch13Maps(t, assessmentPath)
	assessments := make([]map[string]any, 0, 5)
	for _, assessment := range allAssessments {
		if strings.HasPrefix(behaviorBatch13String(t, assessment["characterization_ref"]),
			"BEHAVIOR-AGENT-BATCH-13-") {
			assessments = append(assessments, assessment)
		}
	}
	if len(assessments) != 5 {
		t.Fatalf("assessments=%d", len(assessments))
	}
	index := decodeBehaviorBatch13Maps(t, "product/knowledge/legacy_go_function_snapshot_v1.jsonl")
	indexByOccurrence := make(map[string]map[string]any, len(index))
	for _, item := range index {
		indexByOccurrence[behaviorBatch13String(t, item["occurrence_ref"])] = item
	}
	entriesByBehavior := make(map[string]map[string]bool, len(records))
	for _, record := range records {
		set := make(map[string]bool, len(record.EntryRefs))
		for _, ref := range record.EntryRefs {
			set[ref] = true
		}
		entriesByBehavior[record.Ref] = set
	}
	seenOccurrences := make(map[string]bool)
	linkCount := 0
	for _, assessment := range assessments {
		ref := behaviorBatch13String(t, assessment["characterization_ref"])
		if assessment["authority"] != "advisory_not_capability_state" ||
			assessment["review_state"] != "independent_bootstrap_counterreview_completed_advisory" ||
			assessment["historical_green_state"] != "partial_or_mixed" ||
			assessment["reuse_kind"] != "v2_existing" || assessment["exact_v2_fit"] != "exact_or_primary" {
			t.Fatalf("assessment advisory inválido: %s", ref)
		}
		functionMapping := behaviorBatch13Object(t, assessment["function_mapping"])
		mappingLinks := behaviorBatch13ObjectSlice(t, functionMapping["links"])
		if functionMapping["state"] != "linked_exact" || len(mappingLinks) < 1 || len(mappingLinks) > 3 {
			t.Fatalf("mapping consolidado incoherente: %s", ref)
		}
		for _, link := range mappingLinks {
			occurrence := behaviorBatch13String(t, link["occurrence_ref"])
			indexed, present := indexByOccurrence[occurrence]
			if !present || seenOccurrences[occurrence] {
				t.Fatalf("occurrence ausente o duplicada: %s", occurrence)
			}
			seenOccurrences[occurrence] = true
			for _, field := range []string{"declaration_ref", "variant_ref", "source_path", "git_blob_oid", "blob_digest", "package_path", "package_name", "symbol_kind", "name", "receiver", "signature", "source_sha256", "ast_sha256", "body_sha256", "start_line", "end_line"} {
				if !reflect.DeepEqual(link[field], indexed[field]) {
					t.Fatalf("metadata drift %s %s", occurrence, field)
				}
			}
			if link["review_state"] != "bootstrap_symbol_mapping_reviewed_advisory" || link["body"] != nil || link["canonical_source"] != nil {
				t.Fatalf("link no advisory o contiene cuerpo: %s", occurrence)
			}
			evidenceRefs := behaviorBatch13StringSlice(t, link["evidence_refs"])
			belongs := false
			for _, evidenceRef := range evidenceRefs {
				belongs = belongs || entriesByBehavior[ref][evidenceRef]
			}
			if !belongs {
				t.Fatalf("link sin TASKENTRY de su conducta: %s", occurrence)
			}
			linkCount++
		}
	}
	if linkCount != 13 || len(assessments) != 5 {
		t.Fatalf("conteo de links=%d assessments=%d", linkCount, len(assessments))
	}
}

func behaviorBatch13PreviousFixtures(t *testing.T) [][]byte {
	t.Helper()
	result := make([][]byte, 0, 11)
	for batch := 1; batch <= 11; batch++ {
		result = append(result, readBehaviorBatch13File(t,
			fmt.Sprintf("product/traceability/fixtures/behavior_characterization_agent_batch_%02d.jsonl", batch)))
	}
	return result
}

func behaviorBatch13Path(environment, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(environment)); value != "" {
		return value
	}
	return fallback
}

func readBehaviorBatch13File(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func resizeBehaviorBatch13Record(t *testing.T, raw []byte, behavior string, size int) []byte {
	t.Helper()
	records, err := decodeBehaviorBatchRecords(raw)
	if err != nil {
		t.Fatal(err)
	}
	for index := range records {
		if records[index].Behavior != behavior {
			continue
		}
		if size == 3 {
			records[index].EntryRefs = records[index].EntryRefs[:3]
			records[index].Evidence = records[index].Evidence[:3]
		} else {
			records[index].EntryRefs = append(records[index].EntryRefs, records[index].EntryRefs[0])
			records[index].Evidence = append(records[index].Evidence, records[index].Evidence[0])
		}
		return encodeBehaviorBatch13Records(t, records)
	}
	t.Fatal("conducta ausente")
	return nil
}

func swapBehaviorBatch13Evidence(t *testing.T, raw []byte, leftBehavior, rightBehavior string) []byte {
	t.Helper()
	records, err := decodeBehaviorBatchRecords(raw)
	if err != nil {
		t.Fatal(err)
	}
	left, right := -1, -1
	for index := range records {
		if records[index].Behavior == leftBehavior {
			left = index
		}
		if records[index].Behavior == rightBehavior {
			right = index
		}
	}
	if left < 0 || right < 0 {
		t.Fatal("conductas ausentes")
	}
	records[left].EntryRefs[0], records[right].EntryRefs[0] = records[right].EntryRefs[0], records[left].EntryRefs[0]
	records[left].Evidence[0], records[right].Evidence[0] = records[right].Evidence[0], records[left].Evidence[0]
	return encodeBehaviorBatch13Records(t, records)
}

func encodeBehaviorBatch13Records(t *testing.T, records []behaviorBatchRecord) []byte {
	t.Helper()
	var encoded bytes.Buffer
	for _, record := range records {
		line, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		encoded.Write(line)
		encoded.WriteByte('\n')
	}
	return encoded.Bytes()
}

func mutateBehaviorBatch13RoadmapStatus(t *testing.T, raw []byte, status string) []byte {
	t.Helper()
	var roadmap map[string]any
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		t.Fatal(err)
	}
	for _, value := range behaviorBatch13ObjectSlice(t, roadmap["capability_entries"]) {
		if value["id"] == "ORC-01" {
			value["status"] = status
			changed, err := json.Marshal(roadmap)
			if err != nil {
				t.Fatal(err)
			}
			return changed
		}
	}
	t.Fatal("ORC-01 ausente")
	return nil
}

func decodeBehaviorBatch13Maps(t *testing.T, path string) []map[string]any {
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

func behaviorBatch13ObjectSlice(t *testing.T, value any) []map[string]any {
	t.Helper()
	values, ok := value.([]any)
	if !ok {
		t.Fatalf("array JSON esperado: %T", value)
	}
	result := make([]map[string]any, 0, len(values))
	for _, item := range values {
		result = append(result, behaviorBatch13Object(t, item))
	}
	return result
}

func behaviorBatch13Object(t *testing.T, value any) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("objeto JSON esperado: %T", value)
	}
	return result
}

func behaviorBatch13String(t *testing.T, value any) string {
	t.Helper()
	result, ok := value.(string)
	if !ok || strings.TrimSpace(result) == "" {
		t.Fatalf("string JSON esperada: %T", value)
	}
	return result
}

func behaviorBatch13StringSlice(t *testing.T, value any) []string {
	t.Helper()
	values, ok := value.([]any)
	if !ok {
		t.Fatalf("array string esperado: %T", value)
	}
	result := make([]string, 0, len(values))
	for _, item := range values {
		result = append(result, behaviorBatch13String(t, item))
	}
	return result
}
