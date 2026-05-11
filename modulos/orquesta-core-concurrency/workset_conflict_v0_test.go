package orquestacoreconcurrency

import (
	"reflect"
	"testing"
)

func TestDetectWorksetConflictsV0DetectaWriteWritePorSolape(t *testing.T) {
	conflicts := DetectWorksetConflictsV0([]WorksetClaimV0{
		conflictClaimV0("claim:b", nil, []string{"modulos/orquesta-core-concurrency"}),
		conflictClaimV0("claim:a", nil, []string{"modulos/orquesta-core-concurrency/workset_claim_v0.go"}),
	})

	if len(conflicts) != 1 {
		t.Fatalf("conflicts=%+v, want 1", conflicts)
	}
	conflict := conflicts[0]
	if conflict.ConflictKind != WorksetConflictKindWriteWriteV0 {
		t.Fatalf("kind=%q, want %q", conflict.ConflictKind, WorksetConflictKindWriteWriteV0)
	}
	if conflict.RecommendedAction != WorksetConflictRecommendedActionSerializeV0 {
		t.Fatalf("action=%q, want %q", conflict.RecommendedAction, WorksetConflictRecommendedActionSerializeV0)
	}
	if !reflect.DeepEqual(conflict.ClaimRefs, []string{"claim:a", "claim:b"}) {
		t.Fatalf("claim_refs=%v", conflict.ClaimRefs)
	}
	requireScopeRefsV0(t, conflict.ScopeRefs, []string{"modulos/orquesta-core-concurrency"})
}

func TestDetectWorksetConflictsV0DetectaReadWriteCruzado(t *testing.T) {
	conflicts := DetectWorksetConflictsV0([]WorksetClaimV0{
		conflictClaimV0("claim:reader", []string{"docs/tareas.md"}, []string{"modulos/reader/out.go"}),
		conflictClaimV0("claim:writer", nil, []string{"docs"}),
	})

	if len(conflicts) != 1 {
		t.Fatalf("conflicts=%+v, want 1", conflicts)
	}
	conflict := conflicts[0]
	if conflict.ConflictKind != WorksetConflictKindReadWriteV0 {
		t.Fatalf("kind=%q, want %q", conflict.ConflictKind, WorksetConflictKindReadWriteV0)
	}
	if !reflect.DeepEqual(conflict.ClaimRefs, []string{"claim:reader", "claim:writer"}) {
		t.Fatalf("claim_refs=%v", conflict.ClaimRefs)
	}
	requireScopeRefsV0(t, conflict.ScopeRefs, []string{"docs"})
}

func TestDetectWorksetConflictsV0NoConfundeScopesHermanos(t *testing.T) {
	conflicts := DetectWorksetConflictsV0([]WorksetClaimV0{
		conflictClaimV0("claim:a", []string{"modulos/x/c.go"}, []string{"modulos/x/a.go"}),
		conflictClaimV0("claim:b", nil, []string{"modulos/x/b.go"}),
	})

	if len(conflicts) != 0 {
		t.Fatalf("conflicts=%+v, want none", conflicts)
	}
}

func TestDetectWorksetConflictsV0EsDeterminista(t *testing.T) {
	claims := []WorksetClaimV0{
		conflictClaimV0("claim:c", []string{"docs/pruebas.md"}, []string{"modulos/c/a.go"}),
		conflictClaimV0("claim:a", nil, []string{"modulos/a"}),
		conflictClaimV0("claim:b", nil, []string{"modulos/a/b.go"}),
		conflictClaimV0("claim:d", nil, []string{"docs"}),
	}

	first := DetectWorksetConflictsV0(claims)
	second := DetectWorksetConflictsV0([]WorksetClaimV0{claims[3], claims[1], claims[0], claims[2]})

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("conflicts no deterministas:\nfirst=%+v\nsecond=%+v", first, second)
	}
	gotRefs := make([]string, 0, len(first))
	for _, conflict := range first {
		gotRefs = append(gotRefs, conflict.ConflictRef)
	}
	wantRefs := []string{
		"conflict:read_write:claim:c+claim:d:docs",
		"conflict:write_write:claim:a+claim:b:modulos/a",
	}
	if !reflect.DeepEqual(gotRefs, wantRefs) {
		t.Fatalf("conflict refs=%v, want %v", gotRefs, wantRefs)
	}
}

func conflictClaimV0(claimRef string, readSet []string, writeSet []string) WorksetClaimV0 {
	readRefs, readIssues := NormalizeScopeRefsV0(readSet)
	if len(readIssues) > 0 {
		panic(readIssues)
	}
	writeRefs, writeIssues := NormalizeScopeRefsV0(writeSet)
	if len(writeIssues) > 0 {
		panic(writeIssues)
	}
	return WorksetClaimV0{
		SchemaVersion: WorksetClaimSchemaVersionV0,
		ClaimRef:      claimRef,
		RunRef:        "run:1",
		TaskRef:       claimRef + ":task",
		ReadSet:       readRefs,
		WriteSet:      writeRefs,
	}
}
