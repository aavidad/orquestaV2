package orquestaappcodexstack

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestFrozenRequiredTestsClosureValidatorV0BloqueaTestCongeladoModificadoV0(t *testing.T) {
	projectDir := t.TempDir()
	rel := "modulos/example/frozen_actor_test.go"
	writeFrozenRequiredTestForStackTestV0(t, projectDir, rel, "package example\n\nfunc TestFrozenActor(t *testing.T) {}\n")
	expectedSHA := sha256FileForStackTestV0(t, filepath.Join(projectDir, filepath.FromSlash(rel)))
	frozenRef, ok := orquestaautoprogramming.FrozenRequiredTestContextRefV0(
		orquestaautoprogramming.FrozenRequiredTestV0{Path: rel, SHA256: expectedSHA},
	)
	if !ok {
		t.Fatalf("frozen ref invalida")
	}
	spec := orquestagoal.GoalWorkSpecV0{
		GoalRef:         "goal-ref-frozen-required-tests",
		RequestRef:      "request-ref-frozen-required-tests",
		RunRef:          "run-ref-frozen-required-tests",
		ProjectRef:      "project-ref-frozen-required-tests",
		WorkKind:        "idle_self_improvement",
		WorkProfileKind: "implementation",
		Objective:       "implementar sin tocar tests congelados",
		DirectorKind:    orquestagoal.GoalDirectorKindCodexGoalV0,
		ContextRefs: []orquestagoal.GoalContextRefV0{
			orquestaautoprogramming.FrozenRequiredTestsPhaseContextRefV0(orquestaautoprogramming.FrozenRequiredTestsImplementerPhaseV0),
			frozenRef,
		},
		RuleRefs: []orquestagoal.GoalRuleRefV0{orquestaautoprogramming.FrozenRequiredTestsRuleV0()},
		WriteSet: []orquestagoal.GoalWriteScopeV0{{Path: "modulos/example"}},
		RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef: "required-test-ref-example",
			Command: "go test -count=1 ./modulos/example",
		}},
		ClosurePolicy: orquestagoal.GoalClosurePolicyV0{RequireRequiredTests: true},
	}
	result := orquestagoal.GoalWorkResultV0{
		SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
		GoalRef:       spec.GoalRef,
		Status:        orquestagoal.GoalStatusCompleteV0,
		Summary:       "implementado",
		RequiredTestResults: []orquestagoal.GoalRequiredTestResultV0{{
			TestRef:      "required-test-ref-example",
			Status:       "passed",
			EvidenceRefs: []string{"evidence-ref-required-test-passed"},
		}},
		EvidenceRefs: []string{"evidence-ref-implementer"},
	}
	validator := frozenRequiredTestsClosureValidatorV0{
		Base:           orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		ProjectWorkDir: projectDir,
	}
	closure, err := validator.ValidateGoalWorkClosureV0(context.Background(), spec, result)
	if err != nil || !closure.Accepted {
		t.Fatalf("closure inicial err=%v closure=%+v", err, closure)
	}

	writeFrozenRequiredTestForStackTestV0(t, projectDir, rel, "package example\n\nfunc TestFrozenActorModified(t *testing.T) {}\n")
	closure, err = validator.ValidateGoalWorkClosureV0(context.Background(), spec, result)
	if err != nil ||
		closure.Accepted ||
		!closure.NeedsRework ||
		!goalClosureHasIssueCodeForStackTestV0(closure, orquestaautoprogramming.FrozenRequiredTestsModifiedIssueCodeV0) {
		t.Fatalf("closure modificada err=%v closure=%+v", err, closure)
	}
}

func writeFrozenRequiredTestForStackTestV0(t *testing.T, projectDir string, rel string, body string) {
	t.Helper()
	path := filepath.Join(projectDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func sha256FileForStackTestV0(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	sum := sha256.Sum256(body)
	return fmt.Sprintf("%x", sum[:])
}

func goalClosureHasIssueCodeForStackTestV0(
	closure orquestagoal.GoalClosureValidationV0,
	code string,
) bool {
	for _, issue := range closure.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
