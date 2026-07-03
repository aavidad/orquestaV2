package orquestaappcodexstack

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path"
	"path/filepath"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

type frozenRequiredTestsClosureValidatorV0 struct {
	Base           orquestagoal.GoalWorkClosureValidatorPortV0
	ProjectWorkDir string
}

func (validator frozenRequiredTestsClosureValidatorV0) ValidateGoalWorkClosureV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
	result orquestagoal.GoalWorkResultV0,
) (orquestagoal.GoalClosureValidationV0, error) {
	base := validator.Base
	if base == nil {
		base = orquestagoal.DefaultGoalWorkClosureValidatorV0{}
	}
	closure, err := base.ValidateGoalWorkClosureV0(ctx, spec, result)
	if err != nil || !closure.Accepted {
		return closure, err
	}
	if issue := frozenRequiredTestsDefinerIssueV0(spec, result); issue.Code != "" {
		return frozenRequiredTestsBlockedClosureV0(closure, issue), nil
	}
	frozen := orquestaautoprogramming.ParseFrozenRequiredTestContextRefsV0(spec.ContextRefs)
	if len(frozen) == 0 {
		return closure, nil
	}
	issue := validator.frozenRequiredTestsChangedIssueV0(frozen)
	if issue.Code != "" {
		return frozenRequiredTestsBlockedClosureV0(
			closure,
			issue,
			orquestaautoprogramming.FrozenRequiredTestsEvidenceChangedV0,
		), nil
	}
	closure.EvidenceRefs = compactStringsV0(append(
		closure.EvidenceRefs,
		orquestaautoprogramming.FrozenRequiredTestsEvidenceVerifiedV0,
	))
	return closure, nil
}

func frozenRequiredTestsDefinerIssueV0(
	spec orquestagoal.GoalWorkSpecV0,
	result orquestagoal.GoalWorkResultV0,
) orquestagoal.GoalWorkIssueV0 {
	if !orquestaautoprogramming.GoalHasFrozenRequiredTestsPhaseV0(
		spec,
		orquestaautoprogramming.FrozenRequiredTestsDefinerPhaseV0,
	) {
		return orquestagoal.GoalWorkIssueV0{}
	}
	paths := append([]string(nil), result.ArtifactPaths...)
	for _, artifact := range result.MaterializedArtifacts {
		paths = append(paths, artifact.Path)
	}
	paths = compactStringsV0(paths)
	if len(paths) == 0 {
		return orquestagoal.GoalWorkIssueV0{
			Code:  orquestaautoprogramming.FrozenRequiredTestsMissingIssueCodeV0,
			Field: "artifact_paths",
		}
	}
	for _, candidate := range paths {
		if _, ok := orquestaautoprogramming.NormalizeFrozenRequiredTestPathV0(candidate); !ok {
			return orquestagoal.GoalWorkIssueV0{
				Code:   orquestaautoprogramming.FrozenRequiredTestsInvalidPathIssueCodeV0,
				Field:  "artifact_paths",
				Detail: "path=" + strings.TrimSpace(candidate),
			}
		}
	}
	return orquestagoal.GoalWorkIssueV0{}
}

func (validator frozenRequiredTestsClosureValidatorV0) frozenRequiredTestsChangedIssueV0(
	tests []orquestaautoprogramming.FrozenRequiredTestV0,
) orquestagoal.GoalWorkIssueV0 {
	root := strings.TrimSpace(validator.ProjectWorkDir)
	if root == "" {
		return orquestagoal.GoalWorkIssueV0{
			Code:  orquestaautoprogramming.FrozenRequiredTestsMissingIssueCodeV0,
			Field: "project_work_dir",
		}
	}
	for _, expected := range tests {
		expected, ok := orquestaautoprogramming.NormalizeFrozenRequiredTestV0(expected)
		if !ok {
			return orquestagoal.GoalWorkIssueV0{
				Code:  orquestaautoprogramming.FrozenRequiredTestsInvalidPathIssueCodeV0,
				Field: "frozen_required_tests",
			}
		}
		abs, ok := frozenRequiredTestsProjectFilePathV0(root, expected.Path)
		if !ok {
			return orquestagoal.GoalWorkIssueV0{
				Code:   orquestaautoprogramming.FrozenRequiredTestsInvalidPathIssueCodeV0,
				Field:  "frozen_required_tests.path",
				Detail: "path=" + expected.Path,
			}
		}
		body, err := os.ReadFile(abs)
		if err != nil {
			return orquestagoal.GoalWorkIssueV0{
				Code:   orquestaautoprogramming.FrozenRequiredTestsMissingIssueCodeV0,
				Field:  "frozen_required_tests.path",
				Detail: "path=" + expected.Path,
			}
		}
		sum := sha256.Sum256(body)
		if actual := hex.EncodeToString(sum[:]); actual != expected.SHA256 {
			return orquestagoal.GoalWorkIssueV0{
				Code:   orquestaautoprogramming.FrozenRequiredTestsModifiedIssueCodeV0,
				Field:  "frozen_required_tests.sha256",
				Detail: "path=" + expected.Path,
			}
		}
	}
	return orquestagoal.GoalWorkIssueV0{}
}

func frozenRequiredTestsBlockedClosureV0(
	closure orquestagoal.GoalClosureValidationV0,
	issue orquestagoal.GoalWorkIssueV0,
	evidenceRefs ...string,
) orquestagoal.GoalClosureValidationV0 {
	closure.Status = orquestagoal.GoalStatusBlockedV0
	closure.Accepted = false
	closure.NeedsRework = true
	closure.Issues = append(closure.Issues, issue)
	closure.EvidenceRefs = compactStringsV0(append(closure.EvidenceRefs, evidenceRefs...))
	return closure
}

func frozenRequiredTestsProjectFilePathV0(root string, rel string) (string, bool) {
	root = strings.TrimSpace(root)
	rel = strings.TrimSpace(filepath.ToSlash(rel))
	if root == "" || rel == "" || strings.HasPrefix(rel, "/") {
		return "", false
	}
	clean := path.Clean(rel)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	abs := filepath.Join(rootAbs, filepath.FromSlash(clean))
	relToRoot, err := filepath.Rel(rootAbs, abs)
	if err != nil || relToRoot == "." || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", false
	}
	return abs, true
}
