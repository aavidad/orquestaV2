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

const behaviorBatch28Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_28.jsonl"

func TestBehaviorCharacterizationAgentBatch28(t *testing.T) {
	var previous [][]byte
	for batch := 1; batch <= 27; batch++ {
		previous = append(previous, behaviorBatch28Read(t, fmt.Sprintf("product/traceability/fixtures/behavior_characterization_agent_batch_%02d.jsonl", batch)))
	}
	if err := validateBehaviorBatch28(behaviorBatch28Read(t, behaviorBatch28Fixture), behaviorBatch28Read(t, "product/traceability/task_entries.jsonl"), behaviorBatch28Read(t, "product/roadmap.json"), previous); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch28RejectsMutations(t *testing.T) {
	fixture := behaviorBatch28Read(t, behaviorBatch28Fixture)
	ledger := behaviorBatch28Read(t, "product/traceability/task_entries.jsonl")
	roadmap := behaviorBatch28Read(t, "product/roadmap.json")
	var previous [][]byte
	for batch := 1; batch <= 27; batch++ {
		previous = append(previous, behaviorBatch28Read(t, fmt.Sprintf("product/traceability/fixtures/behavior_characterization_agent_batch_%02d.jsonl", batch)))
	}
	mutations := [][2]string{{`"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical_ledger"`}, {`"claims_accreditation":false`, `"claims_accreditation":true`}, {`"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`}, {`"TASKENTRY-2784cfd194c5069d8fab0ef9"`, `"TASKENTRY-73299fa04483a25a8f27571e"`}, {`{"schema_version":1`, `{"unknown":true,"schema_version":1`}}
	for _, mutation := range mutations {
		changed := bytes.Replace(fixture, []byte(mutation[0]), []byte(mutation[1]), 1)
		if bytes.Equal(changed, fixture) || validateBehaviorBatch28(changed, ledger, roadmap, previous) == nil {
			t.Fatalf("mutación aceptada: %s", mutation[0])
		}
	}
	if validateBehaviorBatch28(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous) == nil {
		t.Fatal("LF final ausente aceptado")
	}
}

func TestBehaviorCharacterizationAgentBatch28MappingsAndAssessments(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(behaviorBatch28Read(t, behaviorBatch28Fixture))
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
	const snapshot = "sha256:5e8f30dc3c6640c57f595e3757cc6e07ba4be6eb36a6bcdd094152a2967fddfc"
	assessments := behaviorBatch28JSONL(t, "product/knowledge/legacy_reuse_assessments_v1.jsonl")
	mappingByRef := map[string]map[string]any{}
	for _, assessment := range assessments {
		ref, _ := assessment["characterization_ref"].(string)
		if belongs[ref] == nil {
			continue
		}
		if mappingByRef[ref] != nil {
			t.Fatalf("mapping duplicado: %s", ref)
		}
		mappingByRef[ref] = assessment["function_mapping"].(map[string]any)
	}
	if len(mappingByRef) != 10 {
		t.Fatalf("mappings del lote=%d", len(mappingByRef))
	}
	index := behaviorBatch28JSONL(t, "product/knowledge/legacy_go_function_snapshot_v1.jsonl")
	byOccurrence := map[string]map[string]any{}
	for _, item := range index {
		byOccurrence[item["occurrence_ref"].(string)] = item
	}
	seen := map[string]bool{}
	total := 0
	for ref, mapping := range mappingByRef {
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
	if total != 11 {
		t.Fatalf("links=%d", total)
	}
	expected := map[string][3]string{
		"BEHAVIOR-AGENT-BATCH-28-GENERATED-APP-DOC-BUNDLE":        {"partial_or_mixed", "contract", "not_implemented"},
		"BEHAVIOR-AGENT-BATCH-28-COMPACT-CAUSAL-HANDOFF":          {"partial_or_mixed", "contract", "not_implemented"},
		"BEHAVIOR-AGENT-BATCH-28-TOKEN-FRUGAL-COMPACT-PROFILE":    {"unknown", "idea", "not_implemented"},
		"BEHAVIOR-AGENT-BATCH-28-REF-ONLY-ARTIFACT-CONTEXT":       {"partial_or_mixed", "contract", "not_implemented"},
		"BEHAVIOR-AGENT-BATCH-28-STABLE-PROMPT-CACHE-PREFIX":      {"partial_or_mixed", "contract", "not_implemented"},
		"BEHAVIOR-AGENT-BATCH-28-DURABLE-BOUNDARY-COMPACTION":     {"partial_or_mixed", "contract", "not_implemented"},
		"BEHAVIOR-AGENT-BATCH-28-RISK-AWARE-CHEAP-ROUTING":        {"partial_or_mixed", "contract", "not_implemented"},
		"BEHAVIOR-AGENT-BATCH-28-EXPLICIT-FORK-SELECTION":         {"unknown", "idea", "not_implemented"},
		"BEHAVIOR-AGENT-BATCH-28-AUTHORIZED-EXTERNAL-EFFECTS":     {"partial_or_mixed", "v2_existing", "exact_or_primary"},
		"BEHAVIOR-AGENT-BATCH-28-OPAQUE-EXTERNAL-DOMAIN-BOUNDARY": {"partial_or_mixed", "contract", "not_implemented"},
	}
	matched := 0
	for _, assessment := range assessments {
		ref := assessment["characterization_ref"].(string)
		want, ok := expected[ref]
		if !ok {
			continue
		}
		matched++
		mapping := assessment["function_mapping"].(map[string]any)
		if assessment["authority"] != "advisory_not_capability_state" || assessment["review_state"] != "independent_bootstrap_counterreview_completed_advisory" || assessment["historical_green_state"] != want[0] || assessment["reuse_kind"] != want[1] || assessment["exact_v2_fit"] != want[2] || mapping["state"] != mappingByRef[ref]["state"] || mapping["rationale"] != mappingByRef[ref]["rationale"] || mapping["snapshot_census_sha256"] != snapshot || !reflect.DeepEqual(mapping["links"], mappingByRef[ref]["links"]) {
			t.Fatalf("assessment: %s", ref)
		}
		delete(expected, ref)
	}
	if matched != 10 || len(expected) != 0 {
		t.Fatal("assessment ausente")
	}
}

func TestBehaviorCharacterizationAgentBatch28QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(behaviorBatch28Read(t, behaviorBatch28Fixture))
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

func TestBehaviorCharacterizationAgentBatch28DocumentsExhaustedDenominators(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(behaviorBatch28Read(t, behaviorBatch28Fixture))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"CTX-03": true, "CTX-04": true, "EVD-08": true}
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
	for _, item := range behaviorBatch28JSONL(t, "product/traceability/task_entries.jsonl") {
		if capability, _ := item["capability_id"].(string); capability == "CTX-03" || capability == "CTX-04" || capability == "EVD-08" {
			counts[capability]++
		}
	}
	if counts["CTX-03"] != 1 || counts["CTX-04"] != 1 || counts["EVD-08"] != 1 {
		t.Fatalf("ledger no agotado: %v", counts)
	}
}
func behaviorBatch28Read(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func behaviorBatch28JSONL(t *testing.T, path string) []map[string]any {
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
