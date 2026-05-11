package orquestacoreconcurrency

import (
	"encoding/json"
	"strings"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildWorksetClaimFromWorkflowTaskV0UsesPublicTaskAndContextRefs(t *testing.T) {
	claim, issues := BuildWorksetClaimFromWorkflowTaskV0(
		validConcurrencyWorkflowTaskV0(),
		validConcurrencyContextBundleV0(),
	)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %#v", issues)
	}
	if claim.ClaimRef != "claim-task-concurrency-001" ||
		claim.RunRef != "run-concurrency-001" ||
		claim.TaskRef != "task-concurrency-001" {
		t.Fatalf("unexpected identity: %#v", claim)
	}
	if !scopeRefsContainV0(claim.ReadSet, "modulos/orquesta-core-concurrency/workset_claim_v0.go") {
		t.Fatalf("expected read-set from context bundle, got %#v", claim.ReadSet)
	}
	if !scopeRefsContainV0(claim.WriteSet, "modulos/orquesta-core-concurrency/workflow_claim_mapper_v0.go") {
		t.Fatalf("expected write-set from workflow task/context, got %#v", claim.WriteSet)
	}
	if !stringSliceContainsV0(claim.EvidenceRefs, "context-bundle-concurrency-001") {
		t.Fatalf("expected bundle ref as evidence, got %#v", claim.EvidenceRefs)
	}
}

func TestBuildWorksetClaimFromWorkflowTaskV0RejectsInvalidBundleScopes(t *testing.T) {
	bundle := validConcurrencyContextBundleV0()
	bundle.Entries = append(bundle.Entries, orquestacontext.ContextBundleEntryV0{
		EntryRef: "entry-home-leak", Kind: orquestacontext.ContextEntryReadRefV0,
		SourceRef: "$HOME/.codex", Required: true,
	})

	_, issues := BuildWorksetClaimFromWorkflowTaskV0(validConcurrencyWorkflowTaskV0(), bundle)
	if !hasWorksetClaimIssueCodeV0(issues, ErrScopeRefHomeV0) {
		t.Fatalf("expected HOME scope issue, got %#v", issues)
	}
}

func TestBuildWorksetClaimFromWorkflowTaskV0DoesNotLeakAdaptersV0(t *testing.T) {
	claim, issues := BuildWorksetClaimFromWorkflowTaskV0(
		validConcurrencyWorkflowTaskV0(),
		validConcurrencyContextBundleV0(),
	)
	if len(issues) != 0 {
		t.Fatalf("expected valid claim, got %#v", issues)
	}
	raw, err := json.Marshal(claim)
	if err != nil {
		t.Fatalf("marshal claim: %v", err)
	}
	lower := strings.ToLower(string(raw))
	for _, forbidden := range []string{"sqlite", "postgres", "oauth", "/home/", "provider", "model", "token"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("claim leaked forbidden marker %q: %s", forbidden, raw)
		}
	}
}

func validConcurrencyWorkflowTaskV0() orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:        "task-concurrency-001",
		RunID:         "run-concurrency-001",
		PhaseID:       "programacion",
		Title:         "Construir claim de concurrencia desde contratos publicos",
		Summary:       "Mapper puro sin inspeccionar ficheros reales.",
		WriteSet: []string{
			"modulos/orquesta-core-concurrency/workflow_claim_mapper_v0.go",
			"modulos/orquesta-core-concurrency/workflow_claim_mapper_v0_test.go",
		},
		AcceptanceCriteria: []string{"El claim se valida con refs compactas."},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  "contract-workset-claim-v0",
			FunctionName: "BuildWorksetClaimFromWorkflowTaskV0",
		}},
	}
}

func validConcurrencyContextBundleV0() orquestacontext.ContextBundleV0 {
	return orquestacontext.BuildContextBundleV0(orquestacontext.ContextBundleRequestV0{
		SchemaVersion: orquestacontext.ContextBundleRequestSchemaVersionV0,
		BundleRef:     "context-bundle-concurrency-001",
		WorkOrderRef:  "work-order-concurrency-001",
		TargetModule:  "orquesta-core-concurrency",
		Phase:         "programacion",
		TaskKind:      "microtarea_codigo",
		Objective:     "Construir claim desde tarea y contexto pequeno.",
		CapacityLevel: "medium",
		ReadSet:       []string{"modulos/orquesta-core-concurrency/workset_claim_v0.go"},
		WriteSet:      []string{"modulos/orquesta-core-concurrency/workflow_claim_mapper_v0.go"},
		ContractRefs:  []string{"WorksetClaimV0", "WorkflowTaskV0", "ContextBundleV0"},
		EvidenceRefs:  []string{"evidence-concurrency-001"},
		MaxEntries:    20,
		MaxTotalBytes: 16000,
	})
}

func scopeRefsContainV0(refs []ScopeRefV0, expected string) bool {
	for _, ref := range refs {
		if ref.Ref == expected {
			return true
		}
	}
	return false
}

func stringSliceContainsV0(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func hasWorksetClaimIssueCodeV0(
	issues []WorksetClaimIssueV0,
	code WorksetClaimIssueCodeV0,
) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
