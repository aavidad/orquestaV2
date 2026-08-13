// Este contrato fija siete conductas V1 con evidencia; los denominadores cero no inflan cobertura.
package orquesta_test

import (
	"bytes"
	"testing"
)

const behaviorBatch26Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_26.jsonl"

func TestBehaviorCharacterizationAgentBatch26(t *testing.T) {
	fixture := behaviorBatch26Read(t, behaviorBatch26Path("ORQUESTA_BATCH26_FIXTURE", behaviorBatch26Fixture))
	if err := validateBehaviorBatch26(fixture, behaviorBatch26Read(t, "product/traceability/task_entries.jsonl"), behaviorBatch26Read(t, "product/roadmap.json"), behaviorBatch26Previous(t)); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch26Assessments(t *testing.T) {
	fixture := behaviorBatch26Read(t, behaviorBatch26Path("ORQUESTA_BATCH26_FIXTURE", behaviorBatch26Fixture))
	assessments := behaviorBatch26Read(t, behaviorBatch26Path("ORQUESTA_BATCH26_ASSESSMENTS", "product/knowledge/legacy_reuse_assessments_v1.jsonl"))
	if err := validateBehaviorBatch26Assessments(assessments, fixture, behaviorBatch26Read(t, "product/knowledge/legacy_go_function_snapshot_v1.jsonl")); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch26RejectsMutations(t *testing.T) {
	fixture := behaviorBatch26Read(t, behaviorBatch26Path("ORQUESTA_BATCH26_FIXTURE", behaviorBatch26Fixture))
	ledger, roadmap, previous := behaviorBatch26Read(t, "product/traceability/task_entries.jsonl"), behaviorBatch26Read(t, "product/roadmap.json"), behaviorBatch26Previous(t)
	for _, mutation := range []struct{ name, old, replacement string }{
		{"authority", `"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical"`}, {"state", `"canonical_state_change":false`, `"canonical_state_change":true`}, {"review", `"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`}, {"reused", `"TASKENTRY-7998d20ee814e735848ee87f"`, `"TASKENTRY-76c5cb177737ae2b21014d61"`}, {"unknown", `{"schema_version":1`, `{"unknown":true,"schema_version":1`},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			changed := bytes.Replace(fixture, []byte(mutation.old), []byte(mutation.replacement), 1)
			if bytes.Equal(changed, fixture) {
				t.Fatal("mutación inerte")
			}
			if validateBehaviorBatch26(changed, ledger, roadmap, previous) == nil {
				t.Fatal("mutación aceptada")
			}
		})
	}
	if validateBehaviorBatch26(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous) == nil {
		t.Fatal("LF ausente aceptado")
	}
}
