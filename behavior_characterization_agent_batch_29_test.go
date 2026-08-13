// Este contrato conserva una propuesta trazable; no crea trabajo, cierre ni estado canónico.
package orquesta_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

const behaviorBatch29Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_29.jsonl"

func TestBehaviorCharacterizationAgentBatch29(t *testing.T) {
	if err := validateBehaviorBatch29(readBehaviorBatch29File(t, behaviorBatch29Fixture), readBehaviorBatch29File(t, "product/traceability/task_entries.jsonl"), readBehaviorBatch29File(t, "product/roadmap.json"), behaviorBatch29PreviousFixtures(t)); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch29RejectsMutations(t *testing.T) {
	fixture := readBehaviorBatch29File(t, behaviorBatch29Fixture)
	ledger := readBehaviorBatch29File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch29File(t, "product/roadmap.json")
	previous := behaviorBatch29PreviousFixtures(t)
	mutations := []struct{ name, old, replacement string }{
		{"autoridad", `"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical_ledger"`}, {"cierre", `"closes_capability":false`, `"closes_capability":true`}, {"acreditación", `"claims_accreditation":false`, `"claims_accreditation":true`}, {"revisión", `"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`}, {"capability", `"capability_id":"EXT-05"`, `"capability_id":"EXT-04"`}, {"ref reutilizada", `TASKENTRY-912c4b4949d38a3f9f2d38a7`, `TASKENTRY-42d9dc522726fe09795ed722`}, {"ancla", `voter_ref autocontenido`, `voter_ref confiable`}, {"campo", `{"schema_version":1`, `{"unknown":true,"schema_version":1`}, {"duplicada", `{"schema_version":1`, `{"schema_version":2,"schema_version":1`}, {"no canónico", `{"schema_version":1`, ` {"schema_version":1`},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed := bytes.Replace(fixture, []byte(mutation.old), []byte(mutation.replacement), 1)
			if bytes.Equal(changed, fixture) {
				t.Fatal("mutación no aplicada")
			}
			if err := validateBehaviorBatch29(changed, ledger, roadmap, previous); err == nil {
				t.Fatal("mutación aceptada")
			}
		})
	}
	if err := validateBehaviorBatch29(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous); err == nil {
		t.Fatal("LF final ausente aceptado")
	}
}

func behaviorBatch29PreviousFixtures(t *testing.T) [][]byte {
	t.Helper()
	result := make([][]byte, 0, 28)
	for index := 1; index <= 28; index++ {
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

func readBehaviorBatch29File(t *testing.T, path string) []byte {
	t.Helper()
	if path == behaviorBatch29Fixture {
		if override := os.Getenv("ORQUESTA_BATCH29_FIXTURE"); override != "" {
			path = override
		}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}
