// Este contrato conserva una propuesta trazable; no crea trabajo, cierre ni estado canónico.
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

const behaviorBatch25Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_25.jsonl"

func TestBehaviorCharacterizationAgentBatch25(t *testing.T) {
	var previous [][]byte
	for batch := 1; batch <= 23; batch++ {
		if batch == 22 {
			continue
		}
		previous = append(previous, behaviorBatch25Read(t, fmt.Sprintf("product/traceability/fixtures/behavior_characterization_agent_batch_%02d.jsonl", batch)))
	}
	if err := validateBehaviorBatch25(behaviorBatch25Read(t, behaviorBatch25Fixture), behaviorBatch25Read(t, "product/traceability/task_entries.jsonl"), behaviorBatch25Read(t, "product/roadmap.json"), previous); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch25RejectsMutations(t *testing.T) {
	fixture := behaviorBatch25Read(t, behaviorBatch25Fixture)
	ledger := behaviorBatch25Read(t, "product/traceability/task_entries.jsonl")
	roadmap := behaviorBatch25Read(t, "product/roadmap.json")
	var previous [][]byte
	for batch := 1; batch <= 23; batch++ {
		if batch == 22 {
			continue
		}
		previous = append(previous, behaviorBatch25Read(t, fmt.Sprintf("product/traceability/fixtures/behavior_characterization_agent_batch_%02d.jsonl", batch)))
	}
	mutations := [][2]string{{`"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical_ledger"`}, {`"claims_accreditation":false`, `"claims_accreditation":true`}, {`"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`}, {`"TASKENTRY-a3906b0e1758f2d8edd1a592"`, `"TASKENTRY-73299fa04483a25a8f27571e"`}, {`{"schema_version":1`, `{"unknown":true,"schema_version":1`}}
	for _, mutation := range mutations {
		changed := bytes.Replace(fixture, []byte(mutation[0]), []byte(mutation[1]), 1)
		if bytes.Equal(changed, fixture) || validateBehaviorBatch25(changed, ledger, roadmap, previous) == nil {
			t.Fatalf("mutación aceptada: %s", mutation[0])
		}
	}
	if validateBehaviorBatch25(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous) == nil {
		t.Fatal("LF final ausente aceptado")
	}
}

func TestBehaviorCharacterizationAgentBatch25MappingsAndAssessments(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(behaviorBatch25Read(t, behaviorBatch25Fixture))
	if err != nil {
		t.Fatal(err)
	}
	belongs := map[string]map[string]bool{}
	for _, record := range records {
		belongs[record.Ref] = map[string]bool{}
		for _, ref := range record.EntryRefs {
			belongs[record.Ref][ref] = true
		}
	}
	var root struct {
		Authority string           `json:"authority"`
		Snapshot  string           `json:"snapshot_census_sha256"`
		Mappings  []map[string]any `json:"mappings"`
	}
	if err := json.Unmarshal(legacyBatchMappingsJSON(t, 25), &root); err != nil {
		t.Fatal(err)
	}
	if root.Authority != "advisory_not_capability_state" || root.Snapshot != "sha256:5e8f30dc3c6640c57f595e3757cc6e07ba4be6eb36a6bcdd094152a2967fddfc" || len(root.Mappings) != 10 {
		t.Fatal("root mappings inválido")
	}
	index := behaviorBatch25JSONL(t, "product/knowledge/legacy_go_function_snapshot_v1.jsonl")
	byOccurrence := map[string]map[string]any{}
	for _, item := range index {
		byOccurrence[item["occurrence_ref"].(string)] = item
	}
	mappingByRef, seen := map[string]map[string]any{}, map[string]bool{}
	total := 0
	for _, mapping := range root.Mappings {
		ref := mapping["characterization_ref"].(string)
		mappingByRef[ref] = mapping
		links := mapping["links"].([]any)
		state := mapping["state"].(string)
		if (state == "linked_exact" && (len(links) < 1 || len(links) > 2)) || (state == "reviewed_no_direct_function" && len(links) != 0) {
			t.Fatalf("state/links: %s", ref)
		}
		for _, raw := range links {
			link := raw.(map[string]any)
			occurrence := link["occurrence_ref"].(string)
			indexed, ok := byOccurrence[occurrence]
			if !ok || seen[occurrence] {
				t.Fatalf("occurrence: %s", occurrence)
			}
			seen[occurrence] = true
			for _, field := range []string{"declaration_ref", "variant_ref", "source_root", "source_path", "git_blob_oid", "blob_digest", "package_path", "package_name", "symbol_kind", "name", "receiver", "signature", "source_sha256", "ast_sha256", "body_sha256", "start_line", "end_line"} {
				if !reflect.DeepEqual(link[field], indexed[field]) {
					t.Fatalf("metadata drift %s %s", occurrence, field)
				}
			}
			if link["review_state"] != "bootstrap_symbol_mapping_reviewed_advisory" || link["body"] != nil || link["canonical_source"] != nil {
				t.Fatal("link no advisory")
			}
			related := false
			for _, evidence := range link["evidence_refs"].([]any) {
				related = related || belongs[ref][evidence.(string)]
			}
			if !related {
				t.Fatal("evidence ajena")
			}
			total++
		}
	}
	if total != 11 {
		t.Fatalf("links=%d", total)
	}
	expected := map[string][3]string{"BEHAVIOR-AGENT-BATCH-25-PORTABLE-DEPLOY-TARGETS": {"partial_or_mixed", "contract", "not_implemented"}, "BEHAVIOR-AGENT-BATCH-25-CAUSAL-REWORK-REPLAN": {"partial_or_mixed", "v2_existing", "exact_or_primary"}, "BEHAVIOR-AGENT-BATCH-25-EVAL-BEFORE-OPTIMIZE": {"partial_or_mixed", "contract", "not_implemented"}, "BEHAVIOR-AGENT-BATCH-25-DURABLE-COUNCIL-POLICY": {"failed_or_negative", "v2_existing", "exact_or_primary"}, "BEHAVIOR-AGENT-BATCH-25-TYPED-CONFIG-REGISTRY": {"failed_or_negative", "v2_existing", "exact_or_primary"}, "BEHAVIOR-AGENT-BATCH-25-REDACTED-EFFECTIVE-CONFIG": {"partial_or_mixed", "v2_existing", "exact_or_primary"}, "BEHAVIOR-AGENT-BATCH-25-ATOMIC-INSTALL-UPDATE-ROLLBACK": {"failed_or_negative", "contract", "not_implemented"}, "BEHAVIOR-AGENT-BATCH-25-EXECUTION-WORK-PROFILE": {"partial_or_mixed", "v2_existing", "exact_or_primary"}, "BEHAVIOR-AGENT-BATCH-25-SECURITY-CRITICALITY-FLOOR": {"failed_or_negative", "v2_existing", "exact_or_primary"}, "BEHAVIOR-AGENT-BATCH-25-LOCAL-OPENAI-COMPATIBLE": {"unknown", "idea", "not_implemented"}}
	assessments := legacyBatchAssessments(t, 25)
	if len(assessments) != 10 {
		t.Fatal("assessments")
	}
	for _, assessment := range assessments {
		ref := assessment["characterization_ref"].(string)
		want, ok := expected[ref]
		mapping := assessment["function_mapping"].(map[string]any)
		if !ok || assessment["authority"] != "advisory_not_capability_state" || assessment["review_state"] != "independent_bootstrap_counterreview_completed_advisory" || assessment["historical_green_state"] != want[0] || assessment["reuse_kind"] != want[1] || assessment["exact_v2_fit"] != want[2] || mapping["state"] != mappingByRef[ref]["state"] || mapping["rationale"] != mappingByRef[ref]["rationale"] || !reflect.DeepEqual(mapping["links"], mappingByRef[ref]["links"]) {
			t.Fatalf("assessment: %s", ref)
		}
		delete(expected, ref)
	}
	if len(expected) != 0 {
		t.Fatal("assessment ausente")
	}
}

func TestBehaviorCharacterizationAgentBatch25QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(behaviorBatch25Read(t, behaviorBatch25Fixture))
	if err != nil {
		t.Fatal(err)
	}
	qualifiers := []string{"declar", "document", "registr", "inform", "históric", "análisis", "auditoría", "cierre", "fuente", "tarea", "política", "corte", "diseño", "legacy", "guía", "roadmap"}
	for _, record := range records {
		for _, claims := range [][]string{record.Worked, record.Failed, record.Attempts} {
			for _, claim := range claims {
				lower, qualified := strings.ToLower(claim), false
				for _, q := range qualifiers {
					qualified = qualified || strings.Contains(lower, q)
				}
				if !qualified {
					t.Fatalf("claim no calificado: %s", claim)
				}
			}
		}
	}
}

func TestBehaviorCharacterizationAgentBatch25DocumentsAGT08DenominatorExhausted(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(behaviorBatch25Read(t, behaviorBatch25Fixture))
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range records {
		if record.CapabilityID == "AGT-08" {
			if len(record.EntryRefs) != 1 || !strings.Contains(strings.Join(record.Uncertainties, "\n"), "denominator_exhausted") {
				t.Fatal("AGT-08 no documenta 1/1 denominator_exhausted")
			}
			return
		}
	}
	t.Fatal("AGT-08 ausente")
}
func behaviorBatch25Read(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func behaviorBatch25JSONL(t *testing.T, path string) []map[string]any {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []map[string]any
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var item map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &item); err != nil {
			t.Fatal(err)
		}
		out = append(out, item)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
