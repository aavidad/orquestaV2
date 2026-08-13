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

const behaviorBatch23Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_23.jsonl"

func TestBehaviorCharacterizationAgentBatch23(t *testing.T) {
	var previous [][]byte
	for batch := 1; batch <= 20; batch++ {
		previous = append(previous, behaviorBatch23Read(t, fmt.Sprintf("product/traceability/fixtures/behavior_characterization_agent_batch_%02d.jsonl", batch)))
	}
	if err := validateBehaviorBatch23(behaviorBatch23Read(t, behaviorBatch23Fixture), behaviorBatch23Read(t, "product/traceability/task_entries.jsonl"), behaviorBatch23Read(t, "product/roadmap.json"), previous); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch23RejectsMutations(t *testing.T) {
	fixture := behaviorBatch23Read(t, behaviorBatch23Fixture)
	ledger := behaviorBatch23Read(t, "product/traceability/task_entries.jsonl")
	roadmap := behaviorBatch23Read(t, "product/roadmap.json")
	var previous [][]byte
	for batch := 1; batch <= 20; batch++ {
		previous = append(previous, behaviorBatch23Read(t, fmt.Sprintf("product/traceability/fixtures/behavior_characterization_agent_batch_%02d.jsonl", batch)))
	}
	mutations := [][2]string{{`"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical_ledger"`}, {`"claims_accreditation":false`, `"claims_accreditation":true`}, {`"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`}, {`"TASKENTRY-06062bd630685f0c80167a6f"`, `"TASKENTRY-73299fa04483a25a8f27571e"`}, {`{"schema_version":1`, `{"unknown":true,"schema_version":1`}}
	for _, mutation := range mutations {
		changed := bytes.Replace(fixture, []byte(mutation[0]), []byte(mutation[1]), 1)
		if bytes.Equal(changed, fixture) || validateBehaviorBatch23(changed, ledger, roadmap, previous) == nil {
			t.Fatalf("mutación aceptada: %s", mutation[0])
		}
	}
	if validateBehaviorBatch23(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous) == nil {
		t.Fatal("LF final ausente aceptado")
	}
}

func TestBehaviorCharacterizationAgentBatch23MappingsAndAssessments(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(behaviorBatch23Read(t, behaviorBatch23Fixture))
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
	if err := json.Unmarshal(legacyBatchMappingsJSON(t, 23), &root); err != nil {
		t.Fatal(err)
	}
	if root.Authority != "advisory_not_capability_state" || root.Snapshot != "sha256:5e8f30dc3c6640c57f595e3757cc6e07ba4be6eb36a6bcdd094152a2967fddfc" || len(root.Mappings) != 10 {
		t.Fatal("root mappings inválido")
	}
	index := behaviorBatch23JSONL(t, "product/knowledge/legacy_go_function_snapshot_v1.jsonl")
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
		if (state == "linked_exact" && len(links) != 2) || (state == "reviewed_no_direct_function" && len(links) != 0) {
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
	if total != 16 {
		t.Fatalf("links=%d", total)
	}
	expected := map[string][3]string{"BEHAVIOR-AGENT-BATCH-23-EXTERNAL-UPDATE-ROLLBACK": {"partial_or_mixed", "negative_lesson", "not_implemented"}, "BEHAVIOR-AGENT-BATCH-23-FAIR-GLOBAL-QUEUE": {"partial_or_mixed", "v2_existing", "exact_or_primary"}, "BEHAVIOR-AGENT-BATCH-23-PUBLIC-SURFACE-E2E": {"partial_or_mixed", "v2_existing", "exact_or_primary"}, "BEHAVIOR-AGENT-BATCH-23-LOGICAL-AGENT-BINDING": {"partial_or_mixed", "v2_existing", "exact_or_primary"}, "BEHAVIOR-AGENT-BATCH-23-CONVERSATIONAL-WEB-WIZARD": {"partial_or_mixed", "contract", "not_implemented"}, "BEHAVIOR-AGENT-BATCH-23-I18N-SINGLE-OWNER": {"partial_or_mixed", "v2_existing", "exact_or_primary"}, "BEHAVIOR-AGENT-BATCH-23-INDEPENDENT-REVIEW-GATE": {"partial_or_mixed", "v2_existing", "exact_or_primary"}, "BEHAVIOR-AGENT-BATCH-23-DURABLE-SESSION-HANDOFF": {"partial_or_mixed", "contract", "not_implemented"}, "BEHAVIOR-AGENT-BATCH-23-CONSUMER-BACKED-ABSTRACTIONS": {"failed_or_negative", "negative_lesson", "not_implemented"}, "BEHAVIOR-AGENT-BATCH-23-NEUTRAL-DOMAIN-WORK": {"partial_or_mixed", "contract", "not_implemented"}}
	assessments := legacyBatchAssessments(t, 23)
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

func TestBehaviorCharacterizationAgentBatch23QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(behaviorBatch23Read(t, behaviorBatch23Fixture))
	if err != nil {
		t.Fatal(err)
	}
	qualifiers := []string{"declar", "document", "registr", "inform", "históric", "análisis", "auditoría", "cierre", "fuente", "tarea", "política", "corte", "diseño"}
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
func behaviorBatch23Read(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func behaviorBatch23JSONL(t *testing.T, path string) []map[string]any {
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
