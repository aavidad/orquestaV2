// Este contrato fija diez conductas V1 para OPES, configuración, estado y retención.
package orquesta_test

import (
	"bytes"
	"testing"
)

const behaviorBatch30Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_30.jsonl"

func TestBehaviorCharacterizationAgentBatch30(t *testing.T) {
	fixture := behaviorBatch30Read(t, behaviorBatch30Path("ORQUESTA_BATCH30_FIXTURE", behaviorBatch30Fixture))
	if err := validateBehaviorBatch30(fixture, behaviorBatch30Read(t, "product/traceability/task_entries.jsonl"), behaviorBatch30Read(t, "product/roadmap.json"), behaviorBatch30Previous(t)); err != nil {
		t.Fatal(err)
	}
}
func TestBehaviorCharacterizationAgentBatch30Assessments(t *testing.T) {
	fixture := behaviorBatch30Read(t, behaviorBatch30Path("ORQUESTA_BATCH30_FIXTURE", behaviorBatch30Fixture))
	assessments := behaviorBatch30Read(t, behaviorBatch30Path("ORQUESTA_BATCH30_ASSESSMENTS", "product/knowledge/legacy_reuse_assessments_v1.jsonl"))
	if err := validateBehaviorBatch30Assessments(assessments, fixture, behaviorBatch30Read(t, "product/knowledge/legacy_go_function_snapshot_v1.jsonl")); err != nil {
		t.Fatal(err)
	}
}
func TestBehaviorCharacterizationAgentBatch30RejectsMutations(t *testing.T) {
	fixture := behaviorBatch30Read(t, behaviorBatch30Path("ORQUESTA_BATCH30_FIXTURE", behaviorBatch30Fixture))
	ledger, roadmap, previous := behaviorBatch30Read(t, "product/traceability/task_entries.jsonl"), behaviorBatch30Read(t, "product/roadmap.json"), behaviorBatch30Previous(t)
	for _, mutation := range []struct{ name, old, replacement string }{{"authority", `"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical"`}, {"state", `"canonical_state_change":false`, `"canonical_state_change":true`}, {"review", `"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`}, {"reused", `"TASKENTRY-da20bc78070bde861549ce23"`, `"TASKENTRY-76c5cb177737ae2b21014d61"`}, {"denominator", `denominator_exhausted`, `denominator_available`}, {"unknown", `{"schema_version":1`, `{"unknown":true,"schema_version":1`}} {
		t.Run(mutation.name, func(t *testing.T) {
			changed := bytes.Replace(fixture, []byte(mutation.old), []byte(mutation.replacement), 1)
			if bytes.Equal(changed, fixture) {
				t.Fatal("mutación inerte")
			}
			if validateBehaviorBatch30(changed, ledger, roadmap, previous) == nil {
				t.Fatal("mutación aceptada")
			}
		})
	}
	if validateBehaviorBatch30(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous) == nil {
		t.Fatal("LF ausente aceptado")
	}
}
