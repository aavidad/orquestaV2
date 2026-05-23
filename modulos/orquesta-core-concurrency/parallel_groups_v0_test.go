package orquestacoreconcurrency

import (
	"reflect"
	"strings"
	"testing"
)

func TestEvaluateParallelGroupsV0CombinaDependenciasYConflictos(t *testing.T) {
	claims := []WorksetClaimV0{
		parallelGroupClaimV0("claim:safe", nil, []string{"modulos/safe/a.go"}),
		parallelGroupClaimV0("claim:a", nil, []string{"docs/tareas.md"}),
		parallelGroupClaimV0("claim:b", nil, []string{"docs"}),
		parallelGroupClaimV0("claim:wait", []string{"claim:safe"}, []string{"modulos/wait/a.go"}),
	}

	plan := EvaluateParallelGroupsV0(claims)

	if !strings.HasPrefix(plan.PlanRef, "parallel_group_plan:run:1:") || len(plan.PlanRef) > 64 {
		t.Fatalf("plan_ref=%q", plan.PlanRef)
	}
	if plan.RunRef != "run:1" {
		t.Fatalf("run_ref=%q", plan.RunRef)
	}
	if !reflect.DeepEqual(plan.ReadyClaimRefs, []string{"claim:safe"}) {
		t.Fatalf("ready=%v", plan.ReadyClaimRefs)
	}
	if !reflect.DeepEqual(plan.BlockedClaimRefs, []string{"claim:a", "claim:b", "claim:wait"}) {
		t.Fatalf("blocked=%v", plan.BlockedClaimRefs)
	}
	if !reflect.DeepEqual(plan.ConflictRefs, []string{"conflict:write_write:claim:a+claim:b:docs"}) {
		t.Fatalf("conflict_refs=%v", plan.ConflictRefs)
	}
	if !reflect.DeepEqual(plan.RepairableConflictRefs, []string{"conflict:write_write:claim:a+claim:b:docs"}) {
		t.Fatalf("repairable_conflict_refs=%v", plan.RepairableConflictRefs)
	}
	if !reflect.DeepEqual(plan.SequenceClaimRefs, []string{"claim:a", "claim:b"}) {
		t.Fatalf("sequence_claim_refs=%v", plan.SequenceClaimRefs)
	}
	if plan.Summary != "parallel_group_plan ready=1 blocked=3 conflicts=1 repairable_conflicts=1 hard_blocked=0 dependency_issues=0" {
		t.Fatalf("summary=%q", plan.Summary)
	}
}

func TestEvaluateParallelGroupsV0EsDeterminista(t *testing.T) {
	claims := []WorksetClaimV0{
		parallelGroupClaimV0("claim:c", nil, []string{"modulos/c/a.go"}),
		parallelGroupClaimV0("claim:a", nil, []string{"modulos/a"}),
		parallelGroupClaimV0("claim:b", nil, []string{"modulos/a/b.go"}),
		parallelGroupClaimV0("claim:d", []string{"claim:c"}, []string{"docs/d.md"}),
	}

	first := EvaluateParallelGroupsV0(claims)
	second := EvaluateParallelGroupsV0([]WorksetClaimV0{claims[3], claims[1], claims[0], claims[2]})

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("plan no determinista:\nfirst=%+v\nsecond=%+v", first, second)
	}
}

func TestEvaluateParallelGroupsV0BloqueaConflictoConClaimNoListoSinElegirGanador(t *testing.T) {
	plan := EvaluateParallelGroupsV0([]WorksetClaimV0{
		parallelGroupClaimV0("claim:ready", nil, []string{"docs/a.md"}),
		parallelGroupClaimV0("claim:future", []string{"claim:missing"}, []string{"docs"}),
		parallelGroupClaimV0("claim:safe", nil, []string{"modulos/safe/a.go"}),
	})

	if !reflect.DeepEqual(plan.ReadyClaimRefs, []string{"claim:safe"}) {
		t.Fatalf("ready=%v", plan.ReadyClaimRefs)
	}
	if !reflect.DeepEqual(plan.BlockedClaimRefs, []string{"claim:future", "claim:ready"}) {
		t.Fatalf("blocked=%v", plan.BlockedClaimRefs)
	}
	if !reflect.DeepEqual(plan.ConflictRefs, []string{"conflict:write_write:claim:future+claim:ready:docs"}) {
		t.Fatalf("conflict_refs=%v", plan.ConflictRefs)
	}
}

func TestEvaluateParallelGroupsV0NoDeclaraReadyClaimsConWriteSetInvalido(t *testing.T) {
	plan := EvaluateParallelGroupsV0([]WorksetClaimV0{
		parallelGroupRawClaimV0("claim:home", nil, []string{"$HOME/proyecto/a.go"}),
		parallelGroupRawClaimV0("claim:db", nil, []string{"db/tables/work_items"}),
		parallelGroupRawClaimV0("claim:runtime", nil, []string{"runtime/process"}),
		parallelGroupRawClaimV0("claim:provider", nil, []string{"provider/openai"}),
		parallelGroupRawClaimV0("claim:oauth", nil, []string{"oauth/tokens"}),
		parallelGroupRawClaimV0("claim:modelo", nil, []string{"modelo/gpt"}),
		parallelGroupRawClaimV0("claim:secretos", nil, []string{"config/secretos"}),
		parallelGroupClaimV0("claim:safe", nil, []string{"modulos/orquesta-core-concurrency/parallel_groups_v0.go"}),
	})

	if !reflect.DeepEqual(plan.ReadyClaimRefs, []string{"claim:safe"}) {
		t.Fatalf("ready=%v", plan.ReadyClaimRefs)
	}
	if !reflect.DeepEqual(plan.BlockedClaimRefs, []string{
		"claim:db",
		"claim:home",
		"claim:modelo",
		"claim:oauth",
		"claim:provider",
		"claim:runtime",
		"claim:secretos",
	}) {
		t.Fatalf("blocked=%v", plan.BlockedClaimRefs)
	}
	if len(plan.ConflictRefs) != 0 {
		t.Fatalf("conflict_refs=%v", plan.ConflictRefs)
	}
	if !reflect.DeepEqual(plan.HardBlockedClaimRefs, []string{
		"claim:db",
		"claim:home",
		"claim:modelo",
		"claim:oauth",
		"claim:provider",
		"claim:runtime",
		"claim:secretos",
	}) {
		t.Fatalf("hard_blocked=%v", plan.HardBlockedClaimRefs)
	}
	if len(plan.RepairableConflictRefs) != 0 || len(plan.SequenceClaimRefs) != 0 {
		t.Fatalf("repairable=%v sequence=%v", plan.RepairableConflictRefs, plan.SequenceClaimRefs)
	}
}

func TestEvaluateParallelGroupsV0ConservaEvidenciaCompactaParaConflictoReparable(t *testing.T) {
	first := parallelGroupClaimV0("claim:a", nil, []string{"docs/tareas.md"})
	first.EvidenceRefs = []string{" evidence:claim-a ", "evidence:shared"}
	second := parallelGroupClaimV0("claim:b", nil, []string{"docs"})
	second.EvidenceRefs = []string{"evidence:claim-b", "evidence:shared"}

	plan := EvaluateParallelGroupsV0([]WorksetClaimV0{first, second})

	if !reflect.DeepEqual(plan.RepairableConflictRefs, []string{"conflict:write_write:claim:a+claim:b:docs"}) {
		t.Fatalf("repairable=%v", plan.RepairableConflictRefs)
	}
	if !reflect.DeepEqual(plan.SequenceClaimRefs, []string{"claim:a", "claim:b"}) {
		t.Fatalf("sequence=%v", plan.SequenceClaimRefs)
	}
	if !reflect.DeepEqual(plan.EvidenceRefs, []string{"evidence:claim-a", "evidence:claim-b", "evidence:shared"}) {
		t.Fatalf("evidence=%v", plan.EvidenceRefs)
	}
}

func parallelGroupClaimV0(claimRef string, dependsOn []string, writeSet []string) WorksetClaimV0 {
	writeRefs, writeIssues := NormalizeScopeRefsV0(writeSet)
	if len(writeIssues) > 0 {
		panic(writeIssues)
	}
	return WorksetClaimV0{
		SchemaVersion: WorksetClaimSchemaVersionV0,
		ClaimRef:      claimRef,
		RunRef:        "run:1",
		TaskRef:       claimRef + ":task",
		WriteSet:      writeRefs,
		DependsOn:     dependsOn,
	}
}

func parallelGroupRawClaimV0(claimRef string, dependsOn []string, writeSet []string) WorksetClaimV0 {
	refs := make([]ScopeRefV0, 0, len(writeSet))
	for _, ref := range writeSet {
		refs = append(refs, ScopeRefV0{Ref: ref})
	}
	return WorksetClaimV0{
		SchemaVersion: WorksetClaimSchemaVersionV0,
		ClaimRef:      claimRef,
		RunRef:        "run:1",
		TaskRef:       claimRef + ":task",
		WriteSet:      refs,
		DependsOn:     dependsOn,
	}
}
