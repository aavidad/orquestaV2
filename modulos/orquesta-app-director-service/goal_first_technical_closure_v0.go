package orquestaappdirectorservice

import (
	"context"
	"path/filepath"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	appDirectorGoalTechnicalLanguageMismatchV0  = "wrong_language_generated"
	appDirectorGoalTechnicalFrameworkMismatchV0 = "goal_technical_framework_mismatch"
)

type appDirectorGoalLanguageManifestRuleV0 struct {
	Canonical            string
	ExpectedManifests    []string
	ConflictingManifests []string
}

type appDirectorGoalClosureValidatorV0 struct {
	Base orquestagoal.GoalWorkClosureValidatorPortV0
}

func (validator appDirectorGoalClosureValidatorV0) ValidateGoalWorkClosureV0(
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
	if issues := appDirectorGoalTechnicalClosureIssuesV0(spec, result); len(issues) > 0 {
		return orquestagoal.GoalClosureValidationV0{
			Status:       orquestagoal.GoalStatusBlockedV0,
			NeedsRework:  true,
			EvidenceRefs: append([]string(nil), closure.EvidenceRefs...),
			Issues:       issues,
		}, nil
	}
	return closure, nil
}

func appDirectorGoalTechnicalClosureIssuesV0(
	spec orquestagoal.GoalWorkSpecV0,
	result orquestagoal.GoalWorkResultV0,
) []orquestagoal.GoalWorkIssueV0 {
	var issues []orquestagoal.GoalWorkIssueV0
	if language := appDirectorGoalRequestedTechnicalTokenV0(spec, "technical-language-"); language != "" &&
		!appDirectorGoalResultMatchesLanguageV0(language, result) {
		issues = append(issues, orquestagoal.GoalWorkIssueV0{
			Code:  appDirectorGoalTechnicalLanguageMismatchV0,
			Field: "technical.language",
		})
	}
	if framework := appDirectorGoalRequestedTechnicalTokenV0(spec, "technical-framework-"); framework != "" &&
		!appDirectorGoalResultMatchesFrameworkV0(framework, result) {
		issues = append(issues, orquestagoal.GoalWorkIssueV0{
			Code:  appDirectorGoalTechnicalFrameworkMismatchV0,
			Field: "technical.framework",
		})
	}
	return issues
}

func appDirectorGoalRequestedTechnicalTokenV0(
	spec orquestagoal.GoalWorkSpecV0,
	prefix string,
) string {
	for _, ref := range spec.ContextRefs {
		if strings.TrimSpace(ref.Kind) != "technical_stack" {
			continue
		}
		value := strings.TrimPrefix(strings.TrimSpace(ref.Ref), prefix)
		if value != strings.TrimSpace(ref.Ref) && value != "" {
			return value
		}
	}
	return ""
}

func appDirectorGoalResultMatchesLanguageV0(
	language string,
	result orquestagoal.GoalWorkResultV0,
) bool {
	if rule, ok := appDirectorGoalLanguageManifestRuleForTokenV0(language); ok {
		if appDirectorGoalResultHasAnyManifestV0(result, rule.ExpectedManifests...) {
			return true
		}
		if appDirectorGoalResultHasAnyManifestV0(result, rule.ConflictingManifests...) {
			return false
		}
		return appDirectorGoalResultTextHasTokenV0(result, rule.Canonical)
	}
	return appDirectorGoalResultTextHasTokenV0(result, language)
}

func appDirectorGoalResultMatchesFrameworkV0(
	framework string,
	result orquestagoal.GoalWorkResultV0,
) bool {
	switch strings.TrimSpace(framework) {
	case "net-http":
		return appDirectorGoalResultTextHasAnyTokenV0(result, "net/http", "net-http", "net_http")
	default:
		return appDirectorGoalResultTextHasTokenV0(result, framework)
	}
}

func appDirectorGoalLanguageManifestRuleForTokenV0(language string) (appDirectorGoalLanguageManifestRuleV0, bool) {
	switch strings.TrimSpace(language) {
	case "go", "golang":
		return appDirectorGoalLanguageManifestRuleV0{
			Canonical:            "go",
			ExpectedManifests:    []string{"go.mod"},
			ConflictingManifests: []string{"pyproject.toml", "setup.py", "package.json", "Cargo.toml"},
		}, true
	case "python", "py":
		return appDirectorGoalLanguageManifestRuleV0{
			Canonical:            "python",
			ExpectedManifests:    []string{"pyproject.toml", "setup.py"},
			ConflictingManifests: []string{"go.mod", "package.json", "Cargo.toml"},
		}, true
	case "node", "nodejs", "javascript", "js", "typescript", "ts":
		return appDirectorGoalLanguageManifestRuleV0{
			Canonical:            "node",
			ExpectedManifests:    []string{"package.json"},
			ConflictingManifests: []string{"go.mod", "pyproject.toml", "setup.py", "Cargo.toml"},
		}, true
	case "rust":
		return appDirectorGoalLanguageManifestRuleV0{
			Canonical:            "rust",
			ExpectedManifests:    []string{"Cargo.toml"},
			ConflictingManifests: []string{"go.mod", "pyproject.toml", "setup.py", "package.json"},
		}, true
	default:
		return appDirectorGoalLanguageManifestRuleV0{}, false
	}
}

func appDirectorGoalResultHasAnyManifestV0(
	result orquestagoal.GoalWorkResultV0,
	manifests ...string,
) bool {
	for _, path := range appDirectorGoalResultPathsV0(result) {
		base := filepath.Base(strings.ToLower(strings.TrimSpace(path)))
		for _, manifest := range manifests {
			if base == strings.ToLower(strings.TrimSpace(manifest)) {
				return true
			}
		}
	}
	return false
}

func appDirectorGoalResultTextHasTokenV0(
	result orquestagoal.GoalWorkResultV0,
	token string,
) bool {
	return appDirectorGoalResultTextHasAnyTokenV0(result, token)
}

func appDirectorGoalResultTextHasAnyTokenV0(
	result orquestagoal.GoalWorkResultV0,
	tokens ...string,
) bool {
	text := strings.ToLower(strings.Join(appDirectorGoalResultTextSignalsV0(result), "\n"))
	for _, token := range tokens {
		if trimmed := strings.ToLower(strings.TrimSpace(token)); trimmed != "" && strings.Contains(text, trimmed) {
			return true
		}
	}
	return false
}

func appDirectorGoalResultTextSignalsV0(result orquestagoal.GoalWorkResultV0) []string {
	signals := []string{result.Summary}
	signals = append(signals, result.ArtifactRefs...)
	signals = append(signals, result.ArtifactPaths...)
	signals = append(signals, result.DomainReceiptRefs...)
	signals = append(signals, result.EvidenceRefs...)
	for _, artifact := range result.MaterializedArtifacts {
		signals = append(signals, artifact.ArtifactRef, artifact.Path, artifact.ArtifactType, artifact.Scope)
		signals = append(signals, artifact.EvidenceRefs...)
	}
	for _, test := range result.RequiredTestResults {
		signals = append(signals, test.TestRef, test.Status)
		signals = append(signals, test.EvidenceRefs...)
	}
	return signals
}

func appDirectorGoalResultPathsV0(result orquestagoal.GoalWorkResultV0) []string {
	paths := append([]string(nil), result.ArtifactPaths...)
	for _, artifact := range result.MaterializedArtifacts {
		if path := strings.TrimSpace(artifact.Path); path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}
