package orquestaserver

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path"
	"path/filepath"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	idleSelfImprovementFrozenTestsDefinerSuffixV0     = "-required-tests"
	idleSelfImprovementFrozenTestsImplementerSuffixV0 = "-implementer"
)

func (runtime *RuntimeV0) idleSelfImprovementFrozenTestsRequestsV0(
	requests []IdleSelfImprovementRequestV0,
) []IdleSelfImprovementRequestV0 {
	if runtime == nil || !runtime.config.IdleSelfImprovementFrozenTests || len(requests) == 0 {
		return requests
	}
	out := make([]IdleSelfImprovementRequestV0, 0, len(requests))
	for _, request := range requests {
		if !idleSelfImprovementFrozenTestsEligibleRequestV0(request) {
			out = append(out, request)
			continue
		}
		if runtime.idleSelfImprovementFrozenTestsImplementerAlreadyAcceptedV0(request) {
			continue
		}
		if implementer, ok := runtime.idleSelfImprovementFrozenTestsImplementerRequestV0(request); ok {
			out = append(out, implementer)
			continue
		}
		out = append(out, idleSelfImprovementFrozenTestsDefinerRequestV0(request))
	}
	if out == nil {
		return []IdleSelfImprovementRequestV0{}
	}
	return out
}

func idleSelfImprovementFrozenTestsEligibleRequestV0(request IdleSelfImprovementRequestV0) bool {
	if strings.TrimSpace(request.FrozenRequiredTestsPhase) != "" {
		return false
	}
	return idleSelfImprovementFrozenTestsSingleModuleScopeV0(request.WriteSet)
}

func idleSelfImprovementFrozenTestsSingleModuleScopeV0(writeSet []string) bool {
	scopes := compactConfigStringsV0(writeSet)
	if len(scopes) != 1 {
		return false
	}
	scope := strings.TrimSpace(filepath.ToSlash(scopes[0]))
	if scope == "" || strings.HasPrefix(scope, "/") || strings.Contains(scope, "\x00") {
		return false
	}
	clean := path.Clean(scope)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return false
	}
	parts := strings.Split(clean, "/")
	return len(parts) == 2 && (parts[0] == "modulos" || parts[0] == "cmd")
}

func idleSelfImprovementFrozenTestsDefinerRequestV0(
	request IdleSelfImprovementRequestV0,
) IdleSelfImprovementRequestV0 {
	baseRef := strings.TrimSpace(request.RequestRef)
	out := request
	out.RequestRef = idleSelfImprovementFrozenTestsDerivedRequestRefV0(baseRef, idleSelfImprovementFrozenTestsDefinerSuffixV0)
	out.ActiveAttemptRef = ""
	out.RequiredTests = nil
	out.FrozenRequiredTests = nil
	out.FrozenRequiredTestsPhase = orquestaautoprogramming.FrozenRequiredTestsDefinerPhaseV0
	out.FrozenRequiredTestsBaseRequestRef = baseRef
	out.FrozenRequiredTestsSourceGoalRef = ""
	out.ContextRefs = compactConfigStringsV0(append(out.ContextRefs,
		"frozen_required_tests_phase:"+orquestaautoprogramming.FrozenRequiredTestsDefinerPhaseV0,
		"frozen_required_tests_base_request:"+baseRef,
	))
	out.AcceptanceCriteria = compactConfigStringsV0(append(out.AcceptanceCriteria,
		"modo opt-in tests congelados: el goal definidor solo escribe nuevos casos _test.go dentro del write-set",
		"el goal definidor devuelve artifact_paths y materialized_artifacts para los tests requeridos creados",
	))
	out.EvidenceRefs = compactConfigStringsV0(append(out.EvidenceRefs,
		orquestaautoprogramming.FrozenRequiredTestsEvidenceDefinerLaunchedV0,
	))
	return out
}

func (runtime *RuntimeV0) idleSelfImprovementFrozenTestsImplementerRequestV0(
	request IdleSelfImprovementRequestV0,
) (IdleSelfImprovementRequestV0, bool) {
	state := runtime.tracker.SnapshotV0()
	if state.IdleSelfImprovementGoalSpec == nil ||
		state.IdleSelfImprovementGoalResult == nil ||
		state.IdleSelfImprovementGoalClosure == nil ||
		!state.IdleSelfImprovementGoalClosure.Accepted {
		return IdleSelfImprovementRequestV0{}, false
	}
	spec := *state.IdleSelfImprovementGoalSpec
	if !orquestaautoprogramming.GoalHasFrozenRequiredTestsPhaseV0(
		spec,
		orquestaautoprogramming.FrozenRequiredTestsDefinerPhaseV0,
	) || idleSelfImprovementFrozenTestsBaseRequestRefV0(spec, request) != strings.TrimSpace(request.RequestRef) {
		return IdleSelfImprovementRequestV0{}, false
	}
	tests := runtime.idleSelfImprovementFrozenTestsFromDefinerResultV0(
		request,
		*state.IdleSelfImprovementGoalResult,
	)
	if len(tests) == 0 {
		return IdleSelfImprovementRequestV0{}, false
	}
	baseRef := strings.TrimSpace(request.RequestRef)
	out := request
	out.RequestRef = idleSelfImprovementFrozenTestsDerivedRequestRefV0(baseRef, idleSelfImprovementFrozenTestsImplementerSuffixV0)
	out.ActiveAttemptRef = ""
	out.ParentRunRef = firstNonEmptyIdleSelfImprovementV0(out.ParentRunRef, spec.RunRef, spec.RequestRef)
	out.SupersedesRunRef = firstNonEmptyIdleSelfImprovementV0(out.SupersedesRunRef, spec.RunRef, spec.RequestRef)
	out.RescueReason = firstNonEmptyIdleSelfImprovementV0(out.RescueReason, "frozen_required_tests_defined")
	out.FrozenRequiredTests = tests
	out.FrozenRequiredTestsPhase = orquestaautoprogramming.FrozenRequiredTestsImplementerPhaseV0
	out.FrozenRequiredTestsBaseRequestRef = baseRef
	out.FrozenRequiredTestsSourceGoalRef = spec.GoalRef
	out.ContextRefs = compactConfigStringsV0(append(out.ContextRefs,
		"frozen_required_tests_phase:"+orquestaautoprogramming.FrozenRequiredTestsImplementerPhaseV0,
		"frozen_required_tests_base_request:"+baseRef,
		"frozen_required_tests_source_goal:"+spec.GoalRef,
	))
	out.AcceptanceCriteria = compactConfigStringsV0(append(out.AcceptanceCriteria,
		"el goal implementador no puede modificar los tests congelados por hash",
		"si cambia un test congelado, el cierre queda en rework con frozen_tests_modified",
	))
	out.EvidenceRefs = compactConfigStringsV0(append(out.EvidenceRefs,
		orquestaautoprogramming.FrozenRequiredTestsEvidenceImplementerV0,
	))
	return out, true
}

func (runtime *RuntimeV0) idleSelfImprovementFrozenTestsImplementerAlreadyAcceptedV0(
	request IdleSelfImprovementRequestV0,
) bool {
	state := runtime.tracker.SnapshotV0()
	if state.IdleSelfImprovementGoalSpec == nil ||
		state.IdleSelfImprovementGoalClosure == nil ||
		!state.IdleSelfImprovementGoalClosure.Accepted {
		return false
	}
	spec := *state.IdleSelfImprovementGoalSpec
	return orquestaautoprogramming.GoalHasFrozenRequiredTestsPhaseV0(
		spec,
		orquestaautoprogramming.FrozenRequiredTestsImplementerPhaseV0,
	) && idleSelfImprovementFrozenTestsBaseRequestRefV0(spec, request) == strings.TrimSpace(request.RequestRef)
}

func (runtime *RuntimeV0) idleSelfImprovementFrozenTestsFromDefinerResultV0(
	request IdleSelfImprovementRequestV0,
	result orquestagoal.GoalWorkResultV0,
) []orquestaautoprogramming.FrozenRequiredTestV0 {
	root := strings.TrimSpace(runtime.config.IdleSelfImprovementProjectWorkDir)
	if root == "" {
		root = strings.TrimSpace(runtime.config.ProjectWorkDir)
	}
	if root == "" {
		return nil
	}
	paths := idleSelfImprovementFrozenTestsArtifactPathsV0(result)
	out := make([]orquestaautoprogramming.FrozenRequiredTestV0, 0, len(paths))
	seen := map[string]bool{}
	for _, candidate := range paths {
		rel, ok := orquestaautoprogramming.NormalizeFrozenRequiredTestPathV0(candidate)
		if !ok || seen[rel] || !idleSelfImprovementFrozenTestWithinWriteSetV0(rel, request.WriteSet) {
			continue
		}
		abs, ok := idleSelfImprovementProjectFilePathV0(root, rel)
		if !ok {
			continue
		}
		body, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		sum := sha256.Sum256(body)
		test, ok := orquestaautoprogramming.NormalizeFrozenRequiredTestV0(
			orquestaautoprogramming.FrozenRequiredTestV0{
				Path:   rel,
				SHA256: hex.EncodeToString(sum[:]),
			},
		)
		if !ok {
			continue
		}
		seen[test.Path] = true
		out = append(out, test)
	}
	return out
}

func idleSelfImprovementFrozenTestsArtifactPathsV0(
	result orquestagoal.GoalWorkResultV0,
) []string {
	paths := append([]string(nil), result.ArtifactPaths...)
	for _, artifact := range result.MaterializedArtifacts {
		paths = append(paths, artifact.Path)
	}
	return compactConfigStringsV0(paths)
}

func idleSelfImprovementFrozenTestWithinWriteSetV0(rel string, writeSet []string) bool {
	scopes := compactConfigStringsV0(writeSet)
	if len(scopes) != 1 {
		return false
	}
	scope := strings.TrimSpace(filepath.ToSlash(scopes[0]))
	scope = path.Clean(scope)
	rel = path.Clean(filepath.ToSlash(rel))
	return rel == scope || strings.HasPrefix(rel, scope+"/")
}

func idleSelfImprovementProjectFilePathV0(root string, rel string) (string, bool) {
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
	if err != nil || relToRoot == "." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) || relToRoot == ".." {
		return "", false
	}
	return abs, true
}

func idleSelfImprovementFrozenTestsBaseRequestRefV0(
	spec orquestagoal.GoalWorkSpecV0,
	request IdleSelfImprovementRequestV0,
) string {
	if ref := orquestaautoprogramming.GoalFrozenRequiredTestsBaseRequestRefV0(spec); ref != "" {
		return ref
	}
	return strings.TrimSpace(request.RequestRef)
}

func idleSelfImprovementFrozenTestsDerivedRequestRefV0(baseRef string, suffix string) string {
	baseRef = strings.TrimSpace(baseRef)
	if baseRef == "" {
		baseRef = "request-ref-idle-self-improvement"
	}
	if strings.HasSuffix(baseRef, suffix) {
		return baseRef
	}
	return baseRef + suffix
}

func idleSelfImprovementGoalSpecWithFrozenRequiredTestsV0(
	spec orquestagoal.GoalWorkSpecV0,
	request IdleSelfImprovementRequestV0,
) orquestagoal.GoalWorkSpecV0 {
	phase := strings.TrimSpace(request.FrozenRequiredTestsPhase)
	if phase == "" {
		return spec
	}
	spec.ContextRefs = append(spec.ContextRefs,
		orquestaautoprogramming.FrozenRequiredTestsPhaseContextRefV0(phase),
	)
	if ref := strings.TrimSpace(request.FrozenRequiredTestsBaseRequestRef); ref != "" {
		spec.ContextRefs = append(spec.ContextRefs,
			orquestaautoprogramming.FrozenRequiredTestsBaseRequestContextRefV0(ref),
		)
	}
	if ref := strings.TrimSpace(request.FrozenRequiredTestsSourceGoalRef); ref != "" {
		spec.ContextRefs = append(spec.ContextRefs,
			orquestaautoprogramming.FrozenRequiredTestsSourceGoalContextRefV0(ref),
		)
	}
	for _, test := range request.FrozenRequiredTests {
		if ref, ok := orquestaautoprogramming.FrozenRequiredTestContextRefV0(test); ok {
			spec.ContextRefs = append(spec.ContextRefs, ref)
		}
	}
	spec.RuleRefs = append(spec.RuleRefs, orquestaautoprogramming.FrozenRequiredTestsRuleV0())
	switch phase {
	case orquestaautoprogramming.FrozenRequiredTestsDefinerPhaseV0:
		spec.WorkProfileKind = "required_tests"
		spec.ClosurePolicy.RequireArtifactPaths = true
		spec.ClosurePolicy.RequireMaterializedArtifacts = true
		spec.ClosurePolicy.RequireChecklist = true
	case orquestaautoprogramming.FrozenRequiredTestsImplementerPhaseV0:
		spec.WorkProfileKind = "implementation"
	}
	return spec
}

func (runtime *RuntimeV0) idleSelfImprovementResultWithFrozenTestGuardV0(
	result orquestagoal.GoalWorkResultV0,
) orquestagoal.GoalWorkResultV0 {
	result = orquestagoal.NormalizeGoalWorkResultV0(result)
	if runtime == nil || runtime.tracker == nil || result.Status != orquestagoal.GoalStatusCompleteV0 {
		return result
	}
	state := runtime.tracker.SnapshotV0()
	if state.IdleSelfImprovementGoalSpec == nil {
		return result
	}
	spec := *state.IdleSelfImprovementGoalSpec
	frozen := orquestaautoprogramming.ParseFrozenRequiredTestContextRefsV0(spec.ContextRefs)
	if len(frozen) == 0 {
		return result
	}
	root := strings.TrimSpace(runtime.config.IdleSelfImprovementProjectWorkDir)
	if root == "" {
		root = strings.TrimSpace(runtime.config.ProjectWorkDir)
	}
	issue := idleSelfImprovementFrozenTestsChangedIssueV0(root, frozen)
	if issue.Code == "" {
		result.EvidenceRefs = compactConfigStringsV0(append(
			result.EvidenceRefs,
			orquestaautoprogramming.FrozenRequiredTestsEvidenceVerifiedV0,
		))
		return result
	}
	result.Status = orquestagoal.GoalStatusBlockedV0
	result.Summary = orquestaautoprogramming.FrozenRequiredTestsModifiedIssueCodeV0
	result.Issues = append(result.Issues, issue)
	result.EvidenceRefs = compactConfigStringsV0(append(
		result.EvidenceRefs,
		orquestaautoprogramming.FrozenRequiredTestsEvidenceChangedV0,
	))
	return result
}

func idleSelfImprovementFrozenTestsChangedIssueV0(
	root string,
	tests []orquestaautoprogramming.FrozenRequiredTestV0,
) orquestagoal.GoalWorkIssueV0 {
	root = strings.TrimSpace(root)
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
		abs, ok := idleSelfImprovementProjectFilePathV0(root, expected.Path)
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
