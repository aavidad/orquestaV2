package orquestagoal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeGoalWorkResultJSONV0CorpusV0(t *testing.T) {
	cases := []struct {
		name        string
		disposition string
		status      string
		missing     bool
	}{
		{"canonical", GoalWorkResultJSONDispositionCanonicalV0, GoalStatusCompleteV0, false},
		{"claude_alias_object_evidence", GoalWorkResultJSONDispositionRepairedV0, GoalStatusCompleteV0, false},
		{"gemini_alias_missing_refs", GoalWorkResultJSONDispositionRepairedV0, GoalStatusCompleteV0, true},
		{"codex_alias_reason", GoalWorkResultJSONDispositionRepairedV0, GoalStatusCompleteV0, false},
		{"invalid_irrecoverable", GoalWorkResultJSONDispositionIrrecoverableV0, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", "goal_result_json", tc.name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := DecodeGoalWorkResultJSONV0(raw)
			if err != nil {
				t.Fatal(err)
			}
			if decoded.Disposition != tc.disposition {
				t.Fatalf("disposition=%q", decoded.Disposition)
			}
			if tc.disposition == GoalWorkResultJSONDispositionIrrecoverableV0 {
				return
			}
			if decoded.Result.Status != tc.status {
				t.Fatalf("status=%q", decoded.Result.Status)
			}
			if tc.missing != (len(decoded.Result.Checklist.MissingRefs) > 0) {
				t.Fatalf("missing_refs=%v", decoded.Result.Checklist.MissingRefs)
			}
			if tc.disposition == GoalWorkResultJSONDispositionRepairedV0 && (decoded.RepairReceipt == nil || decoded.RepairReceipt.OriginalRefHash == "") {
				t.Fatalf("repair_receipt=%+v", decoded.RepairReceipt)
			}
		})
	}
}

func TestDecodeGoalWorkResultJSONV0ProviderEquivalenceV0(t *testing.T) {
	for _, name := range []string{"claude_alias_object_evidence", "codex_alias_reason"} {
		raw, err := os.ReadFile(filepath.Join("testdata", "goal_result_json", name+".json"))
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := DecodeGoalWorkResultJSONV0(raw)
		if err != nil {
			t.Fatal(err)
		}
		if decoded.Result.GoalRef != "goal-ref-f6-corpus-001" || decoded.Result.Status != GoalStatusCompleteV0 {
			t.Fatalf("%s result=%+v", name, decoded.Result)
		}
		if !goalResultTestHasRefV0(decoded.Result.EvidenceRefs, "evidence-ref-result-f6-001") || !goalResultTestHasRefV0(decoded.Result.RequiredTestResults[0].EvidenceRefs, "evidence-ref-test-f6-001") {
			t.Fatalf("%s evidence=%+v", name, decoded.Result)
		}
	}
}

func TestDecodeGoalWorkResultJSONV0NormalizaAliasTipadoDeEntornoDeTestsV0(t *testing.T) {
	for name, raw := range map[string][]byte{
		"reason_code":  []byte(`{"schema_version":"orquesta_goal_result.v0","goal_ref":"goal-ref-test-env-alias","status":"blocked","reason_code":"sandbox_unix_socket_operation_not_permitted"}`),
		"issue_string": []byte(`{"schema_version":"orquesta_goal_result.v0","goal_ref":"goal-ref-test-env-alias","status":"blocked","issues":["external_test_environment_restriction"]}`),
		"issue_object": []byte(`{"schema_version":"orquesta_goal_result.v0","goal_ref":"goal-ref-test-env-alias","status":"blocked","issues":[{"code":"test_environment_unavailable"}]}`),
	} {
		t.Run(name, func(t *testing.T) {
			decoded, err := DecodeGoalWorkResultJSONV0(raw)
			if err != nil {
				t.Fatal(err)
			}
			if decoded.Disposition != GoalWorkResultJSONDispositionRepairedV0 || len(decoded.Result.Issues) != 1 ||
				decoded.Result.Issues[0].Code != GoalIssueRequiredTestsEnvironmentUnavailableV0 {
				t.Fatalf("decoded=%+v", decoded)
			}
		})
	}
}

func TestDecodeGoalWorkResultJSONV0IdempotentEncodeDecodeV0(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "goal_result_json", "claude_alias_object_evidence.json"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := DecodeGoalWorkResultJSONV0(raw)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(first.Result)
	if err != nil {
		t.Fatal(err)
	}
	second, err := DecodeGoalWorkResultJSONV0(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if second.Disposition != GoalWorkResultJSONDispositionCanonicalV0 {
		t.Fatalf("disposition=%q", second.Disposition)
	}
	if second.Result.GoalRef != first.Result.GoalRef || second.Result.Status != first.Result.Status || !goalResultTestHasRefV0(second.Result.EvidenceRefs, "evidence-ref-result-f6-001") {
		t.Fatalf("second=%+v", second.Result)
	}
}

func TestDecodeGoalWorkResultJSONV0MaterializedProcessShapeV0(t *testing.T) {
	raw, err := json.Marshal(GoalWorkResultV0{
		SchemaVersion: GoalWorkResultSchemaV0, Status: GoalStatusCompleteV0, GoalRef: "goal-ref-f6-process-001",
		ArtifactPaths:         []string{"docs/orquesta_goal_result_v0.json"},
		MaterializedArtifacts: []GoalMaterializedArtifactV0{{ArtifactRef: "artifact-ref-f6-process-001", Path: "docs/orquesta_goal_result_v0.json", ArtifactType: "goal_result", Status: GoalMaterializedArtifactStatusValidV0, EvidenceRefs: []string{"evidence-ref-f6-process-artifact"}}},
		Checklist:             GoalWorkChecklistV0{ExpectedRefs: []string{"check-f6-process-001"}, CompletedRefs: []string{"check-f6-process-001"}},
		RequiredTestResults:   []GoalRequiredTestResultV0{{TestRef: "test-ref-f6-process-001", Status: "passed", EvidenceRefs: []string{"evidence-ref-f6-process-test"}}},
		EvidenceRefs:          []string{"evidence-ref-f6-process-result"},
	})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeGoalWorkResultJSONV0(raw)
	if err != nil || decoded.Disposition != GoalWorkResultJSONDispositionCanonicalV0 || decoded.Result.Status != GoalStatusCompleteV0 {
		t.Fatalf("decode=%+v err=%v raw=%s", decoded, err, raw)
	}
}

func TestGoalWorkResultJSONSoftIssueDoesNotBlockCausalClosureV0(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef: "goal-ref-f6-causal-001", Objective: "verify", WriteSet: []GoalWriteScopeV0{{Path: "modulos/orquesta-goal", Purpose: "test"}},
		RequiredTests: []GoalRequiredTestV0{{TestRef: "test-ref-f6-causal-001", EvidenceRefs: []string{"evidence-ref-f6-causal-001"}}},
		ClosurePolicy: GoalClosurePolicyV0{RequireRequiredTests: true},
	}
	result := GoalWorkResultV0{
		Status: GoalStatusCompleteV0, GoalRef: spec.GoalRef,
		RequiredTestResults: []GoalRequiredTestResultV0{{TestRef: "test-ref-f6-causal-001", Status: "passed", EvidenceRefs: []string{"evidence-ref-f6-causal-001"}}},
		Issues:              []GoalWorkIssueV0{{Code: "provider_advisory", Field: "summary"}},
	}
	closure := ValidateGoalWorkClosureV0(spec, result)
	if !closure.Accepted {
		t.Fatalf("closure=%+v", closure)
	}
	result.RequiredTestResults[0].EvidenceRefs = []string{"evidence-ref-f6-causal-receipt-001"}
	if closure = ValidateGoalWorkClosureV0(spec, result); !closure.Accepted {
		t.Fatalf("non-empty causal test evidence rejected: %+v", closure)
	}
	result.RequiredTestResults[0].EvidenceRefs = nil
	if closure = ValidateGoalWorkClosureV0(spec, result); closure.Accepted {
		t.Fatalf("missing causal test evidence accepted: %+v", closure)
	}
}

func TestGoalWorkResultJSONLegacyRequiredTestDoesNotInventEvidenceRequirementV0(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef: "goal-ref-f6-legacy-001", Objective: "verify legacy result", WriteSet: []GoalWriteScopeV0{{Path: "docs"}},
		RequiredTests: []GoalRequiredTestV0{{TestRef: "test-ref-f6-legacy-001"}},
		ClosurePolicy: GoalClosurePolicyV0{RequireRequiredTests: true},
	}
	result := GoalWorkResultV0{
		Status: GoalStatusCompleteV0, GoalRef: spec.GoalRef,
		RequiredTestResults: []GoalRequiredTestResultV0{{TestRef: "test-ref-f6-legacy-001", Status: "passed"}},
	}
	if closure := ValidateGoalWorkClosureV0(spec, result); !closure.Accepted {
		t.Fatalf("legacy closure=%+v", closure)
	}
}

func TestGoalWorkResultJSONClosurePolicyRequiresItsExactEvidenceRefsV0(t *testing.T) {
	spec := GoalWorkSpecV0{
		GoalRef: "goal-ref-f6-policy-001", Objective: "verify closure policy", WriteSet: []GoalWriteScopeV0{{Path: "docs"}},
		RequiredTests: []GoalRequiredTestV0{{TestRef: "test-ref-f6-policy-001", EvidenceRefs: []string{"evidence-ref-f6-test-contract-001"}}},
		ClosurePolicy: GoalClosurePolicyV0{
			RequireRequiredTests: true,
			RequiredEvidenceRefs: []string{"evidence-ref-f6-closure-contract-001"},
		},
	}
	result := GoalWorkResultV0{
		Status: GoalStatusCompleteV0, GoalRef: spec.GoalRef,
		RequiredTestResults: []GoalRequiredTestResultV0{{
			TestRef: "test-ref-f6-policy-001", Status: "passed", EvidenceRefs: []string{"evidence-ref-f6-test-receipt-001"},
		}},
		EvidenceRefs: []string{"evidence-ref-f6-closure-contract-001"},
	}
	if closure := ValidateGoalWorkClosureV0(spec, result); !closure.Accepted {
		t.Fatalf("closure=%+v", closure)
	}
	result.EvidenceRefs = []string{"evidence-ref-f6-test-receipt-001"}
	if closure := ValidateGoalWorkClosureV0(spec, result); closure.Accepted || !hasGoalIssueFieldV0(closure.Issues, "evidence_refs") {
		t.Fatalf("closure=%+v", closure)
	}
}

func TestDecodeGoalWorkResultJSONV0StringObjectNullAndAbsentEvidenceV0(t *testing.T) {
	raw := []byte(`{
		"status":"done",
		"goal_ref":"goal-ref-f6-shapes-001",
		"evidence_refs":"evidence-ref-f6-string-001",
		"checklist":{"evidence_refs":{"ref":"evidence-ref-f6-object-001","description":"must not survive"}},
		"required_tests":[{"test_ref":"test-ref-f6-shapes-001","status":"success","evidence_refs":null}],
		"materialized_artifacts":[{"artifact_ref":"artifact-ref-f6-shapes-001","path":"docs/result.json","status":"valid"}]
	}`)
	decoded, err := DecodeGoalWorkResultJSONV0(raw)
	if err != nil || decoded.Disposition != GoalWorkResultJSONDispositionRepairedV0 {
		t.Fatalf("decode=%+v err=%v", decoded, err)
	}
	if decoded.Result.RequiredTestResults[0].Status != "passed" ||
		!goalResultTestHasRefV0(decoded.Result.EvidenceRefs, "evidence-ref-f6-string-001") ||
		!goalResultTestHasRefV0(decoded.Result.Checklist.EvidenceRefs, "evidence-ref-f6-object-001") ||
		decoded.Result.RequiredTestResults[0].EvidenceRefs != nil {
		t.Fatalf("result=%+v", decoded.Result)
	}
	encoded, err := json.Marshal(decoded.RepairReceipt)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) == "" || containsGoalResultTestTextV0(string(encoded), "must not survive") {
		t.Fatalf("repair receipt leaked description: %s", encoded)
	}
}

func TestDecodeGoalWorkResultJSONV0NullAbsentAndMalformedAreTypedV0(t *testing.T) {
	for _, raw := range [][]byte{[]byte(`null`), []byte(`{}`), []byte(`{"status":`)} {
		decoded, err := DecodeGoalWorkResultJSONV0(raw)
		if decoded.Disposition != GoalWorkResultJSONDispositionIrrecoverableV0 || decoded.RepairReceipt == nil || decoded.RepairReceipt.OriginalRefHash == "" || len(decoded.Issues) == 0 || decoded.Issues[0].Code != ErrGoalResultJSONInvalidV0 {
			t.Fatalf("raw=%q decode=%+v err=%v", raw, decoded, err)
		}
	}
}

func TestDecodeGoalWorkResultJSONV0NullCanonicalUsesAliasAndPreservesIssuesV0(t *testing.T) {
	decoded, err := DecodeGoalWorkResultJSONV0([]byte(`{
		"status":null,
		"estado":"done",
		"issues":"existing_advisory",
		"reason_code":"goal_requires_review"
	}`))
	if err != nil || decoded.Disposition != GoalWorkResultJSONDispositionRepairedV0 || decoded.Result.Status != GoalStatusCompleteV0 {
		t.Fatalf("decode=%+v err=%v", decoded, err)
	}
	if len(decoded.Result.Issues) != 2 || decoded.Result.Issues[0].Code != "existing_advisory" || decoded.Result.Issues[1].Code != "goal_requires_review" {
		t.Fatalf("issues=%+v", decoded.Result.Issues)
	}
}

func containsGoalResultTestTextV0(value, want string) bool {
	for i := 0; i+len(want) <= len(value); i++ {
		if value[i:i+len(want)] == want {
			return true
		}
	}
	return false
}

func goalResultTestHasRefV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
