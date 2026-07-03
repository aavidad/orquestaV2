package orquestaautoprogramming

import (
	"path"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	FrozenRequiredTestsRuleRefV0                 = "autoprogramming:frozen-required-tests:v0"
	FrozenRequiredTestsContextKindV0             = "frozen_required_test"
	FrozenRequiredTestsPhaseContextKindV0        = "frozen_required_tests_phase"
	FrozenRequiredTestsBaseRequestContextKindV0  = "frozen_required_tests_base_request"
	FrozenRequiredTestsSourceGoalContextKindV0   = "frozen_required_tests_source_goal"
	FrozenRequiredTestsDefinerPhaseV0            = "definer"
	FrozenRequiredTestsImplementerPhaseV0        = "implementer"
	FrozenRequiredTestsModifiedIssueCodeV0       = "frozen_tests_modified"
	FrozenRequiredTestsMissingIssueCodeV0        = "frozen_tests_missing"
	FrozenRequiredTestsInvalidPathIssueCodeV0    = "frozen_tests_invalid_path"
	FrozenRequiredTestsEvidenceVerifiedV0        = "evidence-ref-frozen-required-tests-verified"
	FrozenRequiredTestsEvidenceChangedV0         = "evidence-ref-frozen-required-tests-modified"
	FrozenRequiredTestsEvidenceDefinerLaunchedV0 = "evidence-ref-frozen-required-tests-definer-launched"
	FrozenRequiredTestsEvidenceImplementerV0     = "evidence-ref-frozen-required-tests-implementer-launched"
)

type FrozenRequiredTestV0 struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

func NormalizeFrozenRequiredTestPathV0(value string) (string, bool) {
	value = filepathSlashV0(strings.TrimSpace(value))
	if value == "" || strings.HasPrefix(value, "/") || strings.Contains(value, "\x00") {
		return "", false
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	if !strings.HasSuffix(clean, "_test.go") {
		return "", false
	}
	return clean, true
}

func NormalizeFrozenRequiredTestV0(test FrozenRequiredTestV0) (FrozenRequiredTestV0, bool) {
	rel, ok := NormalizeFrozenRequiredTestPathV0(test.Path)
	if !ok {
		return FrozenRequiredTestV0{}, false
	}
	sha := strings.ToLower(strings.TrimSpace(test.SHA256))
	if !frozenRequiredTestSHA256ValidV0(sha) {
		return FrozenRequiredTestV0{}, false
	}
	return FrozenRequiredTestV0{Path: rel, SHA256: sha}, true
}

func FrozenRequiredTestContextRefV0(
	test FrozenRequiredTestV0,
) (orquestagoal.GoalContextRefV0, bool) {
	normalized, ok := NormalizeFrozenRequiredTestV0(test)
	if !ok {
		return orquestagoal.GoalContextRefV0{}, false
	}
	return orquestagoal.GoalContextRefV0{
		Kind:     FrozenRequiredTestsContextKindV0,
		Ref:      "path=" + normalized.Path + ";sha256=" + normalized.SHA256,
		Required: true,
		Purpose:  "Test requerido congelado antes del goal implementador.",
	}, true
}

func FrozenRequiredTestsPhaseContextRefV0(phase string) orquestagoal.GoalContextRefV0 {
	return orquestagoal.GoalContextRefV0{
		Kind:     FrozenRequiredTestsPhaseContextKindV0,
		Ref:      strings.TrimSpace(phase),
		Required: true,
	}
}

func FrozenRequiredTestsBaseRequestContextRefV0(ref string) orquestagoal.GoalContextRefV0 {
	return orquestagoal.GoalContextRefV0{
		Kind:     FrozenRequiredTestsBaseRequestContextKindV0,
		Ref:      strings.TrimSpace(ref),
		Required: true,
	}
}

func FrozenRequiredTestsSourceGoalContextRefV0(ref string) orquestagoal.GoalContextRefV0 {
	return orquestagoal.GoalContextRefV0{
		Kind:    FrozenRequiredTestsSourceGoalContextKindV0,
		Ref:     strings.TrimSpace(ref),
		Purpose: "Goal definidor que materializo los tests congelados.",
	}
}

func FrozenRequiredTestsRuleV0() orquestagoal.GoalRuleRefV0 {
	return orquestagoal.GoalRuleRefV0{
		Kind:        "frozen_required_tests",
		Ref:         FrozenRequiredTestsRuleRefV0,
		Enforcement: orquestagoal.GoalRuleEnforcementHardV0,
	}
}

func ParseFrozenRequiredTestContextRefV0(
	ref orquestagoal.GoalContextRefV0,
) (FrozenRequiredTestV0, bool) {
	if strings.TrimSpace(ref.Kind) != FrozenRequiredTestsContextKindV0 {
		return FrozenRequiredTestV0{}, false
	}
	fields := map[string]string{}
	for _, part := range strings.Split(ref.Ref, ";") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		fields[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return NormalizeFrozenRequiredTestV0(FrozenRequiredTestV0{
		Path:   fields["path"],
		SHA256: fields["sha256"],
	})
}

func ParseFrozenRequiredTestContextRefsV0(
	refs []orquestagoal.GoalContextRefV0,
) []FrozenRequiredTestV0 {
	out := make([]FrozenRequiredTestV0, 0, len(refs))
	seen := map[string]bool{}
	for _, ref := range refs {
		test, ok := ParseFrozenRequiredTestContextRefV0(ref)
		if !ok || seen[test.Path] {
			continue
		}
		seen[test.Path] = true
		out = append(out, test)
	}
	return out
}

func GoalHasFrozenRequiredTestsPhaseV0(
	spec orquestagoal.GoalWorkSpecV0,
	phase string,
) bool {
	phase = strings.TrimSpace(phase)
	for _, ref := range spec.ContextRefs {
		if strings.TrimSpace(ref.Kind) == FrozenRequiredTestsPhaseContextKindV0 &&
			strings.TrimSpace(ref.Ref) == phase {
			return true
		}
	}
	return false
}

func GoalFrozenRequiredTestsBaseRequestRefV0(spec orquestagoal.GoalWorkSpecV0) string {
	for _, ref := range spec.ContextRefs {
		if strings.TrimSpace(ref.Kind) == FrozenRequiredTestsBaseRequestContextKindV0 {
			return strings.TrimSpace(ref.Ref)
		}
	}
	return ""
}

func frozenRequiredTestSHA256ValidV0(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			continue
		}
		return false
	}
	return true
}

func filepathSlashV0(value string) string {
	return strings.ReplaceAll(value, "\\", "/")
}
