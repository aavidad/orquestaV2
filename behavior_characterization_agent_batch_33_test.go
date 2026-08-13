// Este contrato conserva una propuesta trazable; no crea trabajo, cierre ni estado canónico.
package orquesta_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

const behaviorBatch33Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_33.jsonl"

func TestBehaviorCharacterizationAgentBatch33(t *testing.T) {
	if err := validateBehaviorBatch33(readBehaviorBatch33File(t, behaviorBatch33Fixture), readBehaviorBatch33File(t, "product/traceability/task_entries.jsonl"), readBehaviorBatch33File(t, "product/roadmap.json"), behaviorBatch33PreviousFixtures(t)); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch33RejectsMutations(t *testing.T) {
	fixture := readBehaviorBatch33File(t, behaviorBatch33Fixture)
	ledger := readBehaviorBatch33File(t, "product/traceability/task_entries.jsonl")
	roadmap := readBehaviorBatch33File(t, "product/roadmap.json")
	previous := behaviorBatch33PreviousFixtures(t)
	mutations := []struct{ name, old, replacement string }{
		{"autoridad", `"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical_ledger"`}, {"cierre", `"closes_capability":false`, `"closes_capability":true`}, {"acreditación", `"claims_accreditation":false`, `"claims_accreditation":true`}, {"revisión", `"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`}, {"capability", `"capability_id":"STG-10"`, `"capability_id":"STG-09"`}, {"ref reutilizada", `TASKENTRY-f4268b3e5636ef27f69805d3`, `TASKENTRY-42d9dc522726fe09795ed722`}, {"ancla", `chat como principal`, `chat autenticado`}, {"campo", `{"schema_version":1`, `{"unknown":true,"schema_version":1`}, {"duplicada", `{"schema_version":1`, `{"schema_version":2,"schema_version":1`}, {"no canónico", `{"schema_version":1`, ` {"schema_version":1`},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed := bytes.Replace(fixture, []byte(mutation.old), []byte(mutation.replacement), 1)
			if bytes.Equal(changed, fixture) {
				t.Fatal("mutación no aplicada")
			}
			if err := validateBehaviorBatch33(changed, ledger, roadmap, previous); err == nil {
				t.Fatal("mutación aceptada")
			}
		})
	}
	if err := validateBehaviorBatch33(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous); err == nil {
		t.Fatal("LF final ausente aceptado")
	}
}

func behaviorBatch33PreviousFixtures(t *testing.T) [][]byte {
	t.Helper()
	result := make([][]byte, 0, 32)
	for index := 1; index <= 32; index++ {
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

func readBehaviorBatch33File(t *testing.T, path string) []byte {
	t.Helper()
	if path == behaviorBatch33Fixture {
		if override := os.Getenv("ORQUESTA_BATCH33_FIXTURE"); override != "" {
			path = override
		}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}
