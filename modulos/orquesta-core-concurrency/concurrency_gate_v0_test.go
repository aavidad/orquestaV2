package orquestacoreconcurrency

import (
	"reflect"
	"testing"
)

func TestEvaluateConcurrencyGateV0PermiteSoloClaimReady(t *testing.T) {
	evaluation := EvaluateConcurrencyGateV0([]WorksetClaimV0{
		parallelGroupClaimV0("claim:ready", nil, []string{"modulos/a/a.go"}),
		parallelGroupClaimV0("claim:blocked", []string{"claim:ready"}, []string{"modulos/b/b.go"}),
	}, []string{" claim:ready ", "claim:ready"})

	if evaluation.SchemaVersion != ConcurrencyGateSchemaVersionV0 {
		t.Fatalf("schema=%q", evaluation.SchemaVersion)
	}
	if evaluation.Decision != ConcurrencyGateDecisionAllowRequestAgentV0 {
		t.Fatalf("decision=%q", evaluation.Decision)
	}
	if !reflect.DeepEqual(evaluation.SubjectClaimRefs, []string{"claim:ready"}) {
		t.Fatalf("subjects=%v", evaluation.SubjectClaimRefs)
	}
	if !reflect.DeepEqual(evaluation.ReadyClaimRefs, []string{"claim:ready"}) {
		t.Fatalf("ready=%v", evaluation.ReadyClaimRefs)
	}
	if !reflect.DeepEqual(evaluation.BlockedClaimRefs, []string{"claim:blocked"}) {
		t.Fatalf("blocked=%v", evaluation.BlockedClaimRefs)
	}
	if evaluation.Summary != "concurrency_gate decision=allow_request_agent subjects=1 ready=1 blocked=1 conflicts=0" {
		t.Fatalf("summary=%q", evaluation.Summary)
	}
}

func TestEvaluateConcurrencyGateV0BloqueaClaimEnConflicto(t *testing.T) {
	evaluation := EvaluateConcurrencyGateV0([]WorksetClaimV0{
		parallelGroupClaimV0("claim:a", nil, []string{"docs/tareas.md"}),
		parallelGroupClaimV0("claim:b", nil, []string{"docs"}),
		parallelGroupClaimV0("claim:safe", nil, []string{"modulos/safe/a.go"}),
	}, []string{"claim:a"})

	if evaluation.Decision != ConcurrencyGateDecisionBlockRequestAgentV0 {
		t.Fatalf("decision=%q", evaluation.Decision)
	}
	if !reflect.DeepEqual(evaluation.ReadyClaimRefs, []string{"claim:safe"}) {
		t.Fatalf("ready=%v", evaluation.ReadyClaimRefs)
	}
	if !reflect.DeepEqual(evaluation.BlockedClaimRefs, []string{"claim:a", "claim:b"}) {
		t.Fatalf("blocked=%v", evaluation.BlockedClaimRefs)
	}
	if !reflect.DeepEqual(evaluation.ConflictRefs, []string{"conflict:write_write:claim:a+claim:b:docs"}) {
		t.Fatalf("conflicts=%v", evaluation.ConflictRefs)
	}
}

func TestEvaluateConcurrencyGateV0ConsultaDirectorSiClaimNoPerteneceAlPlan(t *testing.T) {
	evaluation := EvaluateConcurrencyGateV0([]WorksetClaimV0{
		parallelGroupClaimV0("claim:ready", nil, []string{"modulos/a/a.go"}),
	}, []string{"claim:unknown"})

	if evaluation.Decision != ConcurrencyGateDecisionAskDirectorV0 {
		t.Fatalf("decision=%q", evaluation.Decision)
	}
}

func TestEvaluateConcurrencyGateV0EsDeterminista(t *testing.T) {
	claims := []WorksetClaimV0{
		parallelGroupClaimV0("claim:b", nil, []string{"modulos/b/b.go"}),
		parallelGroupClaimV0("claim:a", nil, []string{"modulos/a/a.go"}),
	}

	first := EvaluateConcurrencyGateV0(claims, []string{"claim:b", "claim:a"})
	second := EvaluateConcurrencyGateV0([]WorksetClaimV0{claims[1], claims[0]}, []string{"claim:a", "claim:b"})

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("gate no determinista:\nfirst=%+v\nsecond=%+v", first, second)
	}
}
