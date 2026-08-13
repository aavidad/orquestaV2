// Este contrato fija diez conductas V1 de operaciones, integración y colaboración.
package orquesta_test

import (
	"bytes"
	"testing"
)

const behaviorBatch31Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_31.jsonl"

func TestBehaviorCharacterizationAgentBatch31(t *testing.T) {
	fixture := behaviorBatch31Read(t, behaviorBatch31Path("ORQUESTA_BATCH31_FIXTURE", behaviorBatch31Fixture))
	if err := validateBehaviorBatch31(fixture, behaviorBatch31Read(t, "product/traceability/task_entries.jsonl"), behaviorBatch31Read(t, "product/roadmap.json"), behaviorBatch31Previous(t)); err != nil {
		t.Fatal(err)
	}
}
func TestBehaviorCharacterizationAgentBatch31Assessments(t *testing.T) {
	fixture := behaviorBatch31Read(t, behaviorBatch31Path("ORQUESTA_BATCH31_FIXTURE", behaviorBatch31Fixture))
	assessments := behaviorBatch31Read(t, behaviorBatch31Path("ORQUESTA_BATCH31_ASSESSMENTS", "product/knowledge/legacy_reuse_assessments_v1.jsonl"))
	if err := validateBehaviorBatch31Assessments(assessments, fixture, behaviorBatch31Read(t, "product/knowledge/legacy_go_function_snapshot_v1.jsonl")); err != nil {
		t.Fatal(err)
	}
}
func TestBehaviorCharacterizationAgentBatch31RejectsMutations(t *testing.T) {
	fixture := behaviorBatch31Read(t, behaviorBatch31Path("ORQUESTA_BATCH31_FIXTURE", behaviorBatch31Fixture))
	ledger, roadmap, previous := behaviorBatch31Read(t, "product/traceability/task_entries.jsonl"), behaviorBatch31Read(t, "product/roadmap.json"), behaviorBatch31Previous(t)
	for _, mutation := range []struct{ name, old, replacement string }{{"authority", `"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical"`}, {"state", `"canonical_state_change":false`, `"canonical_state_change":true`}, {"review", `"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`}, {"reused", `"TASKENTRY-63c7afcc9eb0927c5fe616bc"`, `"TASKENTRY-3896b0f650809633434958bc"`}, {"denominator", `denominator_exhausted`, `denominator_available`}, {"overlap", `source_overlap_collapsed`, `source_overlap_available`}, {"rejected_capability", `"capability_id":"ORC-18"`, `"capability_id":"ORC-15"`}, {"unknown", `{"schema_version":1`, `{"unknown":true,"schema_version":1`}} {
		t.Run(mutation.name, func(t *testing.T) {
			changed := bytes.Replace(fixture, []byte(mutation.old), []byte(mutation.replacement), 1)
			if bytes.Equal(changed, fixture) {
				t.Fatal("mutación inerte")
			}
			if validateBehaviorBatch31(changed, ledger, roadmap, previous) == nil {
				t.Fatal("mutación aceptada")
			}
		})
	}
	if validateBehaviorBatch31(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous) == nil {
		t.Fatal("LF ausente aceptado")
	}
}
