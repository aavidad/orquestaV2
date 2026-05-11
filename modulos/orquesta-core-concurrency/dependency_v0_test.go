package orquestacoreconcurrency

import (
	"reflect"
	"testing"
)

func TestEvaluateWorksetDependenciesV0DetectaMissingYSelf(t *testing.T) {
	evaluation := EvaluateWorksetDependenciesV0([]WorksetClaimV0{
		dependencyClaimV0("claim:ready", nil),
		dependencyClaimV0("claim:missing", []string{"claim:absent"}),
		dependencyClaimV0("claim:self", []string{"claim:self"}),
	})

	if !reflect.DeepEqual(evaluation.ReadyClaimRefs, []string{"claim:ready"}) {
		t.Fatalf("ready=%v", evaluation.ReadyClaimRefs)
	}
	if !reflect.DeepEqual(evaluation.BlockedClaimRefs, []string{"claim:missing", "claim:self"}) {
		t.Fatalf("blocked=%v", evaluation.BlockedClaimRefs)
	}
	requireDependencyIssueV0(t, evaluation.Issues, WorksetDependencyIssueKindMissingV0, "dependency:dependency_missing:claim:missing:claim:absent")
	requireDependencyIssueV0(t, evaluation.Issues, WorksetDependencyIssueKindSelfV0, "dependency:dependency_self:claim:self:claim:self")
}

func TestEvaluateWorksetDependenciesV0DetectaCicloDeterminista(t *testing.T) {
	claims := []WorksetClaimV0{
		dependencyClaimV0("claim:c", []string{"claim:a"}),
		dependencyClaimV0("claim:a", []string{"claim:b"}),
		dependencyClaimV0("claim:b", []string{"claim:c"}),
		dependencyClaimV0("claim:ready", nil),
	}

	first := EvaluateWorksetDependenciesV0(claims)
	second := EvaluateWorksetDependenciesV0([]WorksetClaimV0{claims[3], claims[1], claims[0], claims[2]})

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("evaluacion no determinista:\nfirst=%+v\nsecond=%+v", first, second)
	}
	requireDependencyIssueV0(t, first.Issues, WorksetDependencyIssueKindCycleV0, "dependency:dependency_cycle:claim:a+claim:b+claim:c:claim:a+claim:b+claim:c")
	if !reflect.DeepEqual(first.ReadyClaimRefs, []string{"claim:ready"}) {
		t.Fatalf("ready=%v", first.ReadyClaimRefs)
	}
	if !reflect.DeepEqual(first.BlockedClaimRefs, []string{"claim:a", "claim:b", "claim:c"}) {
		t.Fatalf("blocked=%v", first.BlockedClaimRefs)
	}
}

func TestEvaluateWorksetDependenciesV0DetectaDependenciaNoCerrada(t *testing.T) {
	evaluation := EvaluateWorksetDependenciesV0([]WorksetClaimV0{
		dependencyClaimV0("claim:a", []string{"claim:b"}),
		dependencyClaimV0("claim:b", []string{"claim:external"}),
		dependencyClaimV0("claim:ready", nil),
	})

	requireDependencyIssueV0(t, evaluation.Issues, WorksetDependencyIssueKindMissingV0, "dependency:dependency_missing:claim:b:claim:external")
	requireDependencyIssueV0(t, evaluation.Issues, WorksetDependencyIssueKindNotClosedV0, "dependency:dependency_not_closed:claim:a+claim:b:claim:external")
	if !reflect.DeepEqual(evaluation.ReadyClaimRefs, []string{"claim:ready"}) {
		t.Fatalf("ready=%v", evaluation.ReadyClaimRefs)
	}
	if !reflect.DeepEqual(evaluation.BlockedClaimRefs, []string{"claim:a", "claim:b"}) {
		t.Fatalf("blocked=%v", evaluation.BlockedClaimRefs)
	}
}

func TestEvaluateWorksetDependenciesV0AceptaCadenaCerrada(t *testing.T) {
	evaluation := EvaluateWorksetDependenciesV0([]WorksetClaimV0{
		dependencyClaimV0("claim:a", []string{"claim:b"}),
		dependencyClaimV0("claim:b", []string{"claim:c"}),
		dependencyClaimV0("claim:c", nil),
	})

	if len(evaluation.Issues) != 0 {
		t.Fatalf("issues=%+v, want none", evaluation.Issues)
	}
	if !reflect.DeepEqual(evaluation.ReadyClaimRefs, []string{"claim:c"}) {
		t.Fatalf("ready=%v", evaluation.ReadyClaimRefs)
	}
	if !reflect.DeepEqual(evaluation.BlockedClaimRefs, []string{"claim:a", "claim:b"}) {
		t.Fatalf("blocked=%v", evaluation.BlockedClaimRefs)
	}
}

func dependencyClaimV0(claimRef string, dependsOn []string) WorksetClaimV0 {
	return WorksetClaimV0{
		SchemaVersion: WorksetClaimSchemaVersionV0,
		ClaimRef:      claimRef,
		RunRef:        "run:1",
		TaskRef:       claimRef + ":task",
		WriteSet:      []ScopeRefV0{{Ref: "modulos/orquesta-core-concurrency/" + claimRef}},
		DependsOn:     dependsOn,
		EvidenceRefs:  []string{"evidence:" + claimRef},
	}
}

func requireDependencyIssueV0(t *testing.T, issues []WorksetDependencyIssueV0, kind WorksetDependencyIssueKindV0, issueRef string) {
	t.Helper()
	for _, issue := range issues {
		if issue.IssueKind == kind && issue.IssueRef == issueRef {
			return
		}
	}
	t.Fatalf("issues=%+v no contienen %q %q", issues, kind, issueRef)
}
