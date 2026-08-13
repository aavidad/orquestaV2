// Este contrato conserva una propuesta trazable; no crea trabajo, cierre ni estado canónico.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const behaviorBatch35Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_35.jsonl"

func TestBehaviorCharacterizationAgentBatch35(t *testing.T) {
	previous := behaviorBatch35PreviousFixtures(t)
	if err := validateBehaviorBatch35(behaviorBatch35Read(t, behaviorBatch35Fixture), behaviorBatch35Read(t, "product/traceability/task_entries.jsonl"), behaviorBatch35Read(t, "product/roadmap.json"), previous); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch35RejectsMutations(t *testing.T) {
	fixture := behaviorBatch35Read(t, behaviorBatch35Fixture)
	ledger := behaviorBatch35Read(t, "product/traceability/task_entries.jsonl")
	roadmap := behaviorBatch35Read(t, "product/roadmap.json")
	previous := behaviorBatch35PreviousFixtures(t)
	mutations := [][2]string{{`"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical_ledger"`}, {`"claims_accreditation":false`, `"claims_accreditation":true`}, {`"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`}, {`"TASKENTRY-3917736c004c97117565a75e"`, `"TASKENTRY-73299fa04483a25a8f27571e"`}, {`{"schema_version":1`, `{"unknown":true,"schema_version":1`}}
	for _, mutation := range mutations {
		changed := bytes.Replace(fixture, []byte(mutation[0]), []byte(mutation[1]), 1)
		if bytes.Equal(changed, fixture) || validateBehaviorBatch35(changed, ledger, roadmap, previous) == nil {
			t.Fatalf("mutación aceptada: %s", mutation[0])
		}
	}
	if validateBehaviorBatch35(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous) == nil {
		t.Fatal("LF final ausente aceptado")
	}
}

func TestBehaviorCharacterizationAgentBatch35MappingsAndAssessments(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(behaviorBatch35Read(t, behaviorBatch35Fixture))
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
		SchemaVersion int              `json:"schema_version"`
		BatchRef      string           `json:"batch_ref"`
		Authority     string           `json:"authority"`
		Snapshot      string           `json:"snapshot_census_sha256"`
		Mappings      []map[string]any `json:"mappings"`
	}
	if err := json.Unmarshal(legacyBatchMappingsJSON(t, 35), &root); err != nil {
		t.Fatal(err)
	}
	if root.SchemaVersion != 1 || root.BatchRef != "BEHAVIOR-AGENT-BATCH-35" || root.Authority != "advisory_not_capability_state" || root.Snapshot != "sha256:5e8f30dc3c6640c57f595e3757cc6e07ba4be6eb36a6bcdd094152a2967fddfc" || len(root.Mappings) != 4 {
		t.Fatal("root mappings inválido")
	}
	index := behaviorBatch35JSONL(t, "product/knowledge/legacy_go_function_snapshot_v1.jsonl")
	byOccurrence := map[string]map[string]any{}
	for _, item := range index {
		byOccurrence[item["occurrence_ref"].(string)] = item
	}
	mappingByRef, seen := map[string]map[string]any{}, map[string]bool{}
	total := 0
	for _, mapping := range root.Mappings {
		ref := mapping["characterization_ref"].(string)
		if mappingByRef[ref] != nil || belongs[ref] == nil {
			t.Fatalf("mapping duplicado o ajeno: %s", ref)
		}
		mappingByRef[ref] = mapping
		links := mapping["links"].([]any)
		state := mapping["state"].(string)
		if (state == "linked_exact" && (len(links) < 1 || len(links) > 3)) || (state == "reviewed_no_direct_function" && len(links) != 0) {
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
			for _, evidence := range link["evidence_refs"].([]any) {
				if !belongs[ref][evidence.(string)] {
					t.Fatalf("evidence ajena: %s", evidence)
				}
			}
			total++
		}
	}
	if total != 8 {
		t.Fatalf("links=%d", total)
	}
	expected := map[string][3]string{
		"BEHAVIOR-AGENT-BATCH-35-CONTEXTUAL-HELP-EXPLAIN":   {"partial_or_mixed", "contract", "not_implemented"},
		"BEHAVIOR-AGENT-BATCH-35-DETERMINISTIC-WIZARD-BOT":  {"partial_or_mixed", "contract", "not_implemented"},
		"BEHAVIOR-AGENT-BATCH-35-DEPENDENT-DECISION-REOPEN": {"failed_or_negative", "contract", "not_implemented"},
		"BEHAVIOR-AGENT-BATCH-35-FINAL-LAUNCH-DOSSIER":      {"partial_or_mixed", "contract", "not_implemented"},
	}
	assessments := legacyBatchAssessments(t, 35)
	if len(assessments) != 4 {
		t.Fatal("assessments")
	}
	for _, assessment := range assessments {
		ref := assessment["characterization_ref"].(string)
		want, ok := expected[ref]
		mapping := assessment["function_mapping"].(map[string]any)
		if !ok || assessment["authority"] != "advisory_not_capability_state" || assessment["review_state"] != "independent_bootstrap_counterreview_completed_advisory" || assessment["historical_green_state"] != want[0] || assessment["reuse_kind"] != want[1] || assessment["exact_v2_fit"] != want[2] || mapping["state"] != mappingByRef[ref]["state"] || mapping["rationale"] != mappingByRef[ref]["rationale"] || mapping["snapshot_census_sha256"] != root.Snapshot || !reflect.DeepEqual(mapping["links"], mappingByRef[ref]["links"]) {
			t.Fatalf("assessment: %s", ref)
		}
		delete(expected, ref)
	}
	if len(expected) != 0 {
		t.Fatal("assessment ausente")
	}
}

func TestBehaviorCharacterizationAgentBatch35QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(behaviorBatch35Read(t, behaviorBatch35Fixture))
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

func TestBehaviorCharacterizationAgentBatch35DocumentsExhaustedDenominators(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(behaviorBatch35Read(t, behaviorBatch35Fixture))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"WIZ-19": true, "WIZ-20": true}
	for _, record := range records {
		if want[record.CapabilityID] {
			if len(record.EntryRefs) != 1 || !strings.Contains(strings.Join(record.Uncertainties, "\n"), "denominator_exhausted") {
				t.Fatalf("%s no documenta 1/1 denominator_exhausted", record.CapabilityID)
			}
			delete(want, record.CapabilityID)
		}
	}
	if len(want) != 0 {
		t.Fatalf("denominadores ausentes: %v", want)
	}
	counts := map[string]int{}
	for _, item := range behaviorBatch35JSONL(t, "product/traceability/task_entries.jsonl") {
		if capability, _ := item["capability_id"].(string); capability == "WIZ-19" || capability == "WIZ-20" {
			counts[capability]++
		}
	}
	if counts["WIZ-19"] != 1 || counts["WIZ-20"] != 1 {
		t.Fatalf("ledger no agotado: %v", counts)
	}
}

func behaviorBatch35PreviousFixtures(t *testing.T) [][]byte {
	t.Helper()
	var previous [][]byte
	paths, err := filepath.Glob("product/traceability/fixtures/behavior_characterization_agent_batch_*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		if filepath.Base(path) == "behavior_characterization_agent_batch_35.jsonl" {
			continue
		}
		previous = append(previous, behaviorBatch35Read(t, path))
	}
	return previous
}

func behaviorBatch35Read(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func behaviorBatch35JSONL(t *testing.T, path string) []map[string]any {
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
