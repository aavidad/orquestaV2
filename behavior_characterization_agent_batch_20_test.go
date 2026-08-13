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

const behaviorBatch20Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_20.jsonl"

func TestBehaviorCharacterizationAgentBatch20(t *testing.T) {
	fixture := behaviorBatch20Read(t, behaviorBatch20Fixture)
	var previous [][]byte
	for batch := 1; batch <= 17; batch++ {
		previous = append(previous, behaviorBatch20Read(t, fmt.Sprintf("product/traceability/fixtures/behavior_characterization_agent_batch_%02d.jsonl", batch)))
	}
	if err := validateBehaviorBatch20(fixture, behaviorBatch20Read(t, "product/traceability/task_entries.jsonl"), behaviorBatch20Read(t, "product/roadmap.json"), previous); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch20RejectsAuthorityAndReuse(t *testing.T) {
	fixture := behaviorBatch20Read(t, behaviorBatch20Fixture)
	ledger := behaviorBatch20Read(t, "product/traceability/task_entries.jsonl")
	roadmap := behaviorBatch20Read(t, "product/roadmap.json")
	var previous [][]byte
	for batch := 1; batch <= 17; batch++ {
		previous = append(previous, behaviorBatch20Read(t, fmt.Sprintf("product/traceability/fixtures/behavior_characterization_agent_batch_%02d.jsonl", batch)))
	}
	mutations := [][2]string{{`"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical_ledger"`}, {`"claims_accreditation":false`, `"claims_accreditation":true`}, {`"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`}, {`"TASKENTRY-d9b21766020e25cd53aa64d2"`, `"TASKENTRY-73299fa04483a25a8f27571e"`}, {`{"schema_version":1`, `{"unknown":true,"schema_version":1`}}
	for _, m := range mutations {
		changed := bytes.Replace(fixture, []byte(m[0]), []byte(m[1]), 1)
		if bytes.Equal(changed, fixture) || validateBehaviorBatch20(changed, ledger, roadmap, previous) == nil {
			t.Fatalf("mutación aceptada: %s", m[0])
		}
	}
	if validateBehaviorBatch20(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous) == nil {
		t.Fatal("LF final ausente aceptado")
	}
}

func TestBehaviorCharacterizationAgentBatch20MappingsAndAssessments(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(behaviorBatch20Read(t, behaviorBatch20Fixture))
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
		Authority, Snapshot string
		Mappings            []map[string]any `json:"mappings"`
		SnapshotCensus      string           `json:"snapshot_census_sha256"`
	}
	if err := json.Unmarshal(legacyBatchMappingsJSON(t, 20), &root); err != nil {
		t.Fatal(err)
	}
	if root.Authority != "advisory_not_capability_state" || root.SnapshotCensus != "sha256:5e8f30dc3c6640c57f595e3757cc6e07ba4be6eb36a6bcdd094152a2967fddfc" || len(root.Mappings) != 5 {
		t.Fatal("root mappings inválido")
	}
	index := behaviorBatch20JSONL(t, "product/knowledge/legacy_go_function_snapshot_v1.jsonl")
	byOccurrence := map[string]map[string]any{}
	for _, item := range index {
		byOccurrence[item["occurrence_ref"].(string)] = item
	}
	mapByRef := map[string]map[string]any{}
	seen := map[string]bool{}
	total := 0
	for _, mapping := range root.Mappings {
		ref := mapping["characterization_ref"].(string)
		mapByRef[ref] = mapping
		links := mapping["links"].([]any)
		state := mapping["state"].(string)
		if (state == "linked_exact" && len(links) != 3) || (state == "reviewed_no_direct_function" && len(links) != 0) {
			t.Fatalf("state/links %s", ref)
		}
		for _, raw := range links {
			link := raw.(map[string]any)
			occurrence := link["occurrence_ref"].(string)
			indexed, ok := byOccurrence[occurrence]
			if !ok || seen[occurrence] {
				t.Fatalf("occurrence %s", occurrence)
			}
			seen[occurrence] = true
			for _, field := range []string{"declaration_ref", "variant_ref", "source_root", "source_path", "git_blob_oid", "blob_digest", "package_path", "package_name", "symbol_kind", "name", "receiver", "signature", "source_sha256", "ast_sha256", "body_sha256", "start_line", "end_line"} {
				if !reflect.DeepEqual(link[field], indexed[field]) {
					t.Fatalf("drift %s %s", occurrence, field)
				}
			}
			if link["review_state"] != "bootstrap_symbol_mapping_reviewed_advisory" || link["body"] != nil || link["canonical_source"] != nil {
				t.Fatal("link no advisory")
			}
			okRef := false
			for _, e := range link["evidence_refs"].([]any) {
				okRef = okRef || belongs[ref][e.(string)]
			}
			if !okRef {
				t.Fatal("evidence ajena")
			}
			total++
		}
	}
	if total != 9 {
		t.Fatalf("links=%d", total)
	}
	assessments := legacyBatchAssessments(t, 20)
	expected := map[string][3]string{"BEHAVIOR-AGENT-BATCH-20-CLOSED-EXECUTION-PERIMETER": {"partial_or_mixed", "v2_existing", "exact_or_primary"}, "BEHAVIOR-AGENT-BATCH-20-DOCS-AS-VERIFIED-PROJECTION": {"partial_or_mixed", "contract", "not_implemented"}, "BEHAVIOR-AGENT-BATCH-20-REPRODUCIBLE-CI-GATE": {"failed_or_negative", "contract", "not_implemented"}, "BEHAVIOR-AGENT-BATCH-20-THIN-WEB-ADMIN": {"partial_or_mixed", "contract", "not_implemented"}, "BEHAVIOR-AGENT-BATCH-20-DERIVED-STATE-NO-CLOSURE": {"partial_or_mixed", "v2_existing", "exact_or_primary"}}
	if len(assessments) != 5 {
		t.Fatal("assessments")
	}
	for _, a := range assessments {
		ref := a["characterization_ref"].(string)
		want, ok := expected[ref]
		fm := a["function_mapping"].(map[string]any)
		if !ok || a["authority"] != "advisory_not_capability_state" || a["review_state"] != "independent_bootstrap_counterreview_completed_advisory" || a["historical_green_state"] != want[0] || a["reuse_kind"] != want[1] || a["exact_v2_fit"] != want[2] || fm["state"] != mapByRef[ref]["state"] || fm["rationale"] != mapByRef[ref]["rationale"] || !reflect.DeepEqual(fm["links"], mapByRef[ref]["links"]) {
			t.Fatalf("assessment %s", ref)
		}
		delete(expected, ref)
	}
	if len(expected) != 0 {
		t.Fatal("assessment ausente")
	}
}

func TestBehaviorCharacterizationAgentBatch20QualifiesHistoricalClaims(t *testing.T) {
	records, err := decodeBehaviorBatchRecords(behaviorBatch20Read(t, behaviorBatch20Fixture))
	if err != nil {
		t.Fatal(err)
	}
	qualifiers := []string{"declar", "document", "registr", "inform", "históric", "auditoría", "plan"}
	for _, record := range records {
		for _, claims := range [][]string{record.Worked, record.Failed, record.Attempts} {
			for _, claim := range claims {
				lower := strings.ToLower(claim)
				qualified := false
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

func behaviorBatch20Read(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func behaviorBatch20JSONL(t *testing.T, path string) []map[string]any {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []map[string]any
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64*1024), 1024*1024)
	for s.Scan() {
		var v map[string]any
		if json.Unmarshal(s.Bytes(), &v) != nil {
			t.Fatal("jsonl inválido")
		}
		out = append(out, v)
	}
	if s.Err() != nil {
		t.Fatal(s.Err())
	}
	return out
}
