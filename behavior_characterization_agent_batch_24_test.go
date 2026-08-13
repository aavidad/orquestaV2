// Este contrato conserva una propuesta trazable; no crea trabajo, cierre ni estado canónico.
package orquesta_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

const behaviorBatch24Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_24.jsonl"

func TestBehaviorCharacterizationAgentBatch24(t *testing.T) {
	if err := validateBehaviorBatch24(readBehaviorBatch24File(t, behaviorBatch24Fixture), readBehaviorBatch24File(t, "product/traceability/task_entries.jsonl"), readBehaviorBatch24File(t, "product/roadmap.json"), behaviorBatch24PreviousFixtures(t)); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch24RejectsMutations(t *testing.T) {
	fixture := readBehaviorBatch24File(t, behaviorBatch24Fixture)
	ledger := readBehaviorBatch24File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch24File(t, "product/roadmap.json")
	previous := behaviorBatch24PreviousFixtures(t)
	mutations := []struct{ name, old, replacement string }{
		{"autoridad", `"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical_ledger"`}, {"cierre", `"closes_capability":false`, `"closes_capability":true`}, {"acreditación", `"claims_accreditation":false`, `"claims_accreditation":true`}, {"revisión", `"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`}, {"capability", `"capability_id":"APP-04"`, `"capability_id":"APP-03"`}, {"ref reutilizada", `TASKENTRY-66f3e416a87aa14c774d917b`, `TASKENTRY-42d9dc522726fe09795ed722`}, {"ancla", `pending_restart informa sin fingir activación inmediata`, `pending_restart finge activación`}, {"campo", `{"schema_version":1`, `{"unknown":true,"schema_version":1`}, {"duplicada", `{"schema_version":1`, `{"schema_version":2,"schema_version":1`}, {"no canónico", `{"schema_version":1`, ` {"schema_version":1`},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed := bytes.Replace(fixture, []byte(mutation.old), []byte(mutation.replacement), 1)
			if bytes.Equal(changed, fixture) {
				t.Fatal("mutación no aplicada")
			}
			if err := validateBehaviorBatch24(changed, ledger, roadmap, previous); err == nil {
				t.Fatal("mutación aceptada")
			}
		})
	}
	if err := validateBehaviorBatch24(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous); err == nil {
		t.Fatal("LF final ausente aceptado")
	}
}

func behaviorBatch24PreviousFixtures(t *testing.T) [][]byte {
	t.Helper()
	result := make([][]byte, 0, 21)
	for index := 1; index <= 21; index++ {
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

func readBehaviorBatch24File(t *testing.T, path string) []byte {
	t.Helper()
	if path == behaviorBatch24Fixture {
		if override := os.Getenv("ORQUESTA_BATCH24_FIXTURE"); override != "" {
			path = override
		}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}
