package orquesta_test

import (
	"bytes"
	"testing"
)

const behaviorBatch22Fixture = "product/traceability/fixtures/behavior_characterization_agent_batch_22.jsonl"

func TestBehaviorCharacterizationAgentBatch22(t *testing.T) {
	fixture := behaviorBatch22Read(t, behaviorBatch22Path("ORQUESTA_BATCH22_FIXTURE", behaviorBatch22Fixture))
	if err := validateBehaviorBatch22(fixture, behaviorBatch22Read(t, "product/traceability/task_entries.jsonl"), behaviorBatch22Read(t, "product/roadmap.json"), behaviorBatch22Previous(t)); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch22Assessments(t *testing.T) {
	fixture := behaviorBatch22Read(t, behaviorBatch22Path("ORQUESTA_BATCH22_FIXTURE", behaviorBatch22Fixture))
	assessments := behaviorBatch22Read(t, behaviorBatch22Path("ORQUESTA_BATCH22_ASSESSMENTS", "product/knowledge/legacy_reuse_assessments_v1.jsonl"))
	if err := validateBehaviorBatch22Assessments(assessments, fixture, behaviorBatch22Read(t, "product/knowledge/legacy_go_function_snapshot_v1.jsonl")); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorCharacterizationAgentBatch22RejectsMutations(t *testing.T) {
	fixture := behaviorBatch22Read(t, behaviorBatch22Path("ORQUESTA_BATCH22_FIXTURE", behaviorBatch22Fixture))
	ledger, roadmap, previous := behaviorBatch22Read(t, "product/traceability/task_entries.jsonl"), behaviorBatch22Read(t, "product/roadmap.json"), behaviorBatch22Previous(t)
	for _, mutation := range []struct{ name, old, replacement string }{
		{"authority", `"authority":"proposal_fixture_not_canonical_ledger"`, `"authority":"canonical"`},
		{"state", `"canonical_state_change":false`, `"canonical_state_change":true`},
		{"review", `"review_state":"bootstrap_first_review_pending_independent_counterreview"`, `"review_state":"accepted"`},
		{"reused", `"TASKENTRY-064674b17518e99c35254a41"`, `"TASKENTRY-76c5cb177737ae2b21014d61"`},
		{"unknown", `{"schema_version":1`, `{"unknown":true,"schema_version":1`},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			changed := bytes.Replace(fixture, []byte(mutation.old), []byte(mutation.replacement), 1)
			if bytes.Equal(changed, fixture) {
				t.Fatal("mutación inerte")
			}
			if validateBehaviorBatch22(changed, ledger, roadmap, previous) == nil {
				t.Fatal("mutación aceptada")
			}
		})
	}
	if validateBehaviorBatch22(bytes.TrimSuffix(fixture, []byte("\n")), ledger, roadmap, previous) == nil {
		t.Fatal("LF ausente aceptado")
	}
}
