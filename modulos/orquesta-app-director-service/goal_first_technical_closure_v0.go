package orquestaappdirectorservice

import (
	"context"
	"path/filepath"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	appDirectorGoalTechnicalLanguageMismatchV0  = "goal_technical_language_mismatch"
	appDirectorGoalTechnicalFrameworkMismatchV0 = "goal_technical_framework_mismatch"
)

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
	switch strings.TrimSpace(language) {
	case "go", "golang":
		return appDirectorGoalResultHasGoSignalsV0(result)
	default:
		return appDirectorGoalResultTextHasTokenV0(result, language)
	}
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

func appDirectorGoalResultHasGoSignalsV0(result orquestagoal.GoalWorkResultV0) bool {
	for _, path := range appDirectorGoalResultPathsV0(result) {
		base := strings.ToLower(filepath.Base(path))
		if strings.HasSuffix(strings.ToLower(path), ".go") || base == "go.mod" || base == "go.sum" {
			return true
		}
	}
	return appDirectorGoalResultTextHasAnyTokenV0(result, "go.mod", ".go", "golang")
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
