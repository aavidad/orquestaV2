// Este contrato conserva una propuesta trazable; no crea trabajo, cierre ni estado canónico.
package orquesta_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

const behaviorBatch27Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_27.jsonl"

func TestBehaviorCharacterizationAgentBatch27(t *testing.T) {
	if err := validateBehaviorBatch27(readBehaviorBatch27File(t, behaviorBatch27Fixture), readBehaviorBatch27File(t, "product/traceability/task_entries.jsonl"), readBehaviorBatch27File(t, "product/roadmap.json"), behaviorBatch27PreviousFixtures(t)); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch27RejectsMutations(t *testing.T) {
	fixture := readBehaviorBatch27File(t, behaviorBatch27Fixture)
	ledger := readBehaviorBatch27File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch27File(t, "product/roadmap.json")
	previous := behaviorBatch27PreviousFixtures(t)
	mutations := []struct{ name, old, replacement string }{
		{"autoridad", `"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical_ledger"`}, {"cierre", `"closes_capability":false`, `"closes_capability":true`}, {"acreditación", `"claims_accreditation":false`, `"claims_accreditation":true`}, {"revisión", `"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`}, {"capability", `"capability_id":"AGT-05"`, `"capability_id":"AGT-04"`}, {"ref reutilizada", `TASKENTRY-3781daf1295fb9c441d55ad4`, `TASKENTRY-42d9dc522726fe09795ed722`}, {"ancla", `substring de caller`, `texto del caller`}, {"campo", `{"schema_version":1`, `{"unknown":true,"schema_version":1`}, {"duplicada", `{"schema_version":1`, `{"schema_version":2,"schema_version":1`}, {"no canónico", `{"schema_version":1`, ` {"schema_version":1`},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed := bytes.Replace(fixture, []byte(mutation.old), []byte(mutation.replacement), 1)
			if bytes.Equal(changed, fixture) {
				t.Fatal("mutación no aplicada")
			}
			if err := validateBehaviorBatch27(changed, ledger, roadmap, previous); err == nil {
				t.Fatal("mutación aceptada")
			}
		})
	}
	if err := validateBehaviorBatch27(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous); err == nil {
		t.Fatal("LF final ausente aceptado")
	}
}

func behaviorBatch27PreviousFixtures(t *testing.T) [][]byte {
	t.Helper()
	result := make([][]byte, 0, 26)
	for index := 1; index <= 26; index++ {
		path := fmt.Sprintf("product/traceability/fixtures/behavior_characterization_agent_batch_%02d.jsonl", index)
		content, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		result = append(result, content)
	}
	return result
}

func readBehaviorBatch27File(t *testing.T, path string) []byte {
	t.Helper()
	if path == behaviorBatch27Fixture {
		if override := os.Getenv("ORQUESTA_BATCH27_FIXTURE"); override != "" {
			path = override
		}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}
