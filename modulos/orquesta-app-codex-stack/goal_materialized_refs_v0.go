package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

const (
	goalMaterializedRefsMaxFilesV0                 = 64
	goalMaterializedQAScanMaxFilesV0               = 256
	goalMaterializedQAScanMaxBytesV0               = 512 * 1024
	goalMaterializedOPESOutOfScopeMaxFilesV0       = 128
	goalMaterializedOPESOutOfScopeMaxRefsV0        = 16
	goalMaterializedWorkDeliveryFileV0             = "work_delivery.json"
	goalMaterializedOPESReworkDeliveryFileV0       = "opes_topic_rework_delivery.json"
	goalMaterializedGoalResultFileV0               = "orquesta_goal_result_v0.json"
	goalMaterializedCheckpointFileV0               = "checkpoint_started.txt"
	goalMaterializedPhase0CheckpointDeliveryFileV0 = "orquesta_phase0_checkpoint_delivery.json"
	goalMaterializedMissingTerminalReceiptEvidence = "evidence-ref-goal-materialized-missing-terminal-receipt-after-artifacts-pass"
	goalMaterializedArtifactPathsOmittedEvidence   = "evidence-ref-goal-materialized-artifact-paths-omitted"
	goalMaterializedOutOfScopeArtifactsEvidence    = "evidence-ref-goal-materialized-out-of-scope-artifacts"
	goalMaterializedQAFailedPublicTextEvidence     = "evidence-ref-goal-materialized-qa-failed-public-text"
	goalMaterializedPartialArtifactsEvidence       = "evidence-ref-goal-materialized-partial-artifacts-written"
	goalMaterializedPhase0NonPublishableEvidence   = "evidence-ref-goal-materialized-phase0-complete-non-publishable"
	goalMaterializedRequiredTestEvidenceMissing    = "evidence-ref-goal-materialized-required-test-evidence-missing"
	goalMaterializedRequiredTestEvidenceDetected   = "evidence-ref-goal-materialized-required-test-evidence-detected"
	goalMaterializedValidArtifactListEvidence      = "evidence-ref-goal-materialized-valid-artifact-list"
	goalMaterializedInvalidArtifactListEvidence    = "evidence-ref-goal-materialized-invalid-artifact-list"
)

var errGoalMaterializedRefsScanDoneV0 = errors.New("goal_materialized_refs_scan_done")

type stackGoalMaterializedRefsSourceV0 struct {
	Config                       ConfigV0
	GoalStateStore               orquestagoal.GoalWorkStateStorePortV0
	GoalClosureValidator         orquestagoal.GoalWorkClosureValidatorPortV0
	RepairMissingTerminalReceipt bool
}

type goalMaterializedRefsScanV0 struct {
	Result               orquestamcp.MCPDirectorGoalMaterializedRefsV0
	FilesScanned         int
	HasArtifact          bool
	HasQAPass            bool
	HasQAFail            bool
	HasTerminalReceipt   bool
	HasPhase0Delivery    bool
	TerminalResult       *orquestagoal.GoalWorkResultV0
	ArtifactPaths        []string
	ValidArtifactPaths   []string
	InvalidArtifactPaths []string
	MissingTestKeys      []string
	PassedTestKeys       []string
}

func (source stackGoalMaterializedRefsSourceV0) ResolveDirectorGoalMaterializedRefsV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
) (orquestamcp.MCPDirectorGoalMaterializedRefsV0, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	projectRoot := strings.TrimSpace(source.Config.Codex.ProjectWorkDir)
	if projectRoot == "" {
		return orquestamcp.MCPDirectorGoalMaterializedRefsV0{}, false, nil
	}
	projectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return orquestamcp.MCPDirectorGoalMaterializedRefsV0{}, false, nil
	}
	scan := goalMaterializedRefsScanV0{}
	for _, scope := range state.Spec.WriteSet {
		if len(scan.Result.DomainReceiptRefs)+len(scan.Result.ArtifactRefs) >= goalMaterializedRefsMaxFilesV0 ||
			scan.FilesScanned >= goalMaterializedQAScanMaxFilesV0 {
			break
		}
		next, err := source.scanGoalWriteScopeForMaterializedRefsV0(projectRoot, state, scope)
		if err != nil {
			return orquestamcp.MCPDirectorGoalMaterializedRefsV0{}, false, err
		}
		scan = mergeGoalMaterializedRefsScanV0(scan, next)
	}
	result := scan.Result
	if source.RepairMissingTerminalReceipt &&
		source.GoalStateStore != nil &&
		scan.TerminalResult != nil {
		repaired, repairErr := repairGoalFirstReceiptFromMaterializedResultV0(
			ctx,
			state,
			*scan.TerminalResult,
			source.GoalStateStore,
			source.GoalClosureValidator,
		)
		if repairErr == nil && repaired.Repaired {
			source.RepairMissingTerminalReceipt = false
			return source.ResolveDirectorGoalMaterializedRefsV0(ctx, repaired.State)
		}
	}
	if !scan.HasTerminalReceipt &&
		scan.HasArtifact &&
		scan.HasQAPass &&
		!scan.HasQAFail &&
		!goalMaterializedMissingTerminalReceiptHandledV0(state) {
		result.IssueCodes = append(result.IssueCodes, orquestamcp.MCPGoalFirstMissingTerminalReceiptAfterArtifactsPassV0)
		result.EvidenceRefs = append(result.EvidenceRefs, goalMaterializedMissingTerminalReceiptEvidence)
		result.ExpectedReceiptRefs = append(result.ExpectedReceiptRefs, goalMaterializedExpectedTerminalReceiptRefsV0(state)...)
	}
	if scan.HasArtifact && !scan.HasTerminalReceipt {
		result.IssueCodes = append(result.IssueCodes, orquestamcp.MCPGoalFirstPartialArtifactsWrittenV0)
		result.EvidenceRefs = append(result.EvidenceRefs, goalMaterializedPartialArtifactsEvidence)
	}
	if scan.HasPhase0Delivery {
		result.IssueCodes = append(result.IssueCodes, orquestamcp.MCPGoalFirstPhase0CompleteNonPublishableV0)
		result.EvidenceRefs = append(result.EvidenceRefs, goalMaterializedPhase0NonPublishableEvidence)
	}
	if len(goalMaterializedRequiredTestEvidenceMissingKeysV0(scan)) > 0 {
		result.IssueCodes = append(result.IssueCodes, orquestamcp.MCPGoalFirstRequiredTestEvidenceMissingV0)
		result.EvidenceRefs = append(result.EvidenceRefs, goalMaterializedRequiredTestEvidenceMissing)
	}
	if omitted := goalMaterializedArtifactPathsOmittedV0(state, scan.TerminalResult, scan.ArtifactPaths); len(omitted) > 0 {
		result.IssueCodes = append(result.IssueCodes, orquestamcp.MCPGoalFirstArtifactPathsOmittedMaterializedV0)
		result.EvidenceRefs = append(result.EvidenceRefs, goalMaterializedArtifactPathsOmittedEvidence)
		for _, path := range omitted {
			result.EvidenceRefs = append(result.EvidenceRefs, goalMaterializedArtifactPathOmittedRefV0(state.RunRef, path))
		}
	}
	if outOfScope := goalMaterializedOPESOutOfScopeArtifactsV0(projectRoot, state); len(outOfScope) > 0 {
		result.IssueCodes = append(result.IssueCodes, orquestamcp.MCPGoalFirstOutOfScopeMaterializedArtifactsV0)
		result.EvidenceRefs = append(result.EvidenceRefs, goalMaterializedOutOfScopeArtifactsEvidence)
		for _, path := range outOfScope {
			result.EvidenceRefs = append(result.EvidenceRefs, goalMaterializedOutOfScopeArtifactRefV0(state.RunRef, path))
		}
	}
	for _, path := range scan.ValidArtifactPaths {
		result.ArtifactRefs = append(result.ArtifactRefs, goalMaterializedValidArtifactRefV0(state.RunRef, path))
	}
	if len(scan.ValidArtifactPaths) > 0 {
		result.EvidenceRefs = append(result.EvidenceRefs, goalMaterializedValidArtifactListEvidence)
	}
	for _, path := range scan.InvalidArtifactPaths {
		result.ArtifactRefs = append(result.ArtifactRefs, goalMaterializedInvalidArtifactRefV0(state.RunRef, path))
	}
	if len(scan.InvalidArtifactPaths) > 0 {
		result.EvidenceRefs = append(result.EvidenceRefs, goalMaterializedInvalidArtifactListEvidence)
	}
	if source.RepairMissingTerminalReceipt &&
		source.GoalStateStore != nil &&
		goalFirstStringSliceContainsV0(result.IssueCodes, orquestamcp.MCPGoalFirstMissingTerminalReceiptAfterArtifactsPassV0) {
		repaired, repairErr := repairGoalFirstReceiptFromMaterializedRefsV0(
			ctx,
			state,
			result,
			source.GoalStateStore,
			source.GoalClosureValidator,
		)
		if repairErr == nil && repaired.Repaired {
			source.RepairMissingTerminalReceipt = false
			return source.ResolveDirectorGoalMaterializedRefsV0(ctx, repaired.State)
		}
	}
	if len(result.DomainReceiptRefs) == 0 &&
		len(result.ArtifactRefs) == 0 &&
		len(result.IssueCodes) == 0 &&
		len(result.EvidenceRefs) == 0 {
		return orquestamcp.MCPDirectorGoalMaterializedRefsV0{}, false, nil
	}
	if len(result.DomainReceiptRefs) > 0 {
		result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-goal-materialized-work-delivery-detected")
		result.IssueCodes = append(result.IssueCodes, "goal_first_materialized_work_delivery_detected")
	}
	if len(result.ArtifactRefs) > 0 {
		result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-goal-materialized-checkpoint-detected")
		result.IssueCodes = append(result.IssueCodes, "goal_first_materialized_checkpoint_detected")
	}
	result.ExpectedReceiptRefs = append(result.ExpectedReceiptRefs, goalMaterializedExpectedChecklistRefsV0(state)...)
	result.ExpectedReceiptRefs = compactStringsV0(result.ExpectedReceiptRefs)
	result.EvidenceRefs = compactStringsV0(result.EvidenceRefs)
	result.IssueCodes = compactStringsV0(result.IssueCodes)
	return result, true, nil
}

func (source stackGoalMaterializedRefsSourceV0) scanGoalWriteScopeForMaterializedRefsV0(
	projectRoot string,
	state orquestagoal.GoalWorkStateV0,
	scope orquestagoal.GoalWriteScopeV0,
) (goalMaterializedRefsScanV0, error) {
	relScope := filepath.Clean(filepath.FromSlash(strings.TrimSpace(scope.Path)))
	if relScope == "." || relScope == "" || filepath.IsAbs(relScope) || strings.HasPrefix(relScope, ".."+string(filepath.Separator)) || relScope == ".." {
		return goalMaterializedRefsScanV0{}, nil
	}
	target := filepath.Join(projectRoot, relScope)
	if !pathWithinRootV0(projectRoot, target) {
		return goalMaterializedRefsScanV0{}, nil
	}
	info, err := os.Stat(target)
	if err != nil {
		return goalMaterializedRefsScanV0{}, nil
	}
	if !info.IsDir() {
		return source.scanGoalMaterializedFileV0(projectRoot, state, target, info), nil
	}
	scan := goalMaterializedRefsScanV0{}
	walkErr := filepath.WalkDir(target, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if len(scan.Result.DomainReceiptRefs)+len(scan.Result.ArtifactRefs) >= goalMaterializedRefsMaxFilesV0 ||
			scan.FilesScanned >= goalMaterializedQAScanMaxFilesV0 {
			return errGoalMaterializedRefsScanDoneV0
		}
		if entry == nil || entry.IsDir() {
			return nil
		}
		if !pathWithinRootV0(projectRoot, path) {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return nil
		}
		scan = mergeGoalMaterializedRefsScanV0(
			scan,
			source.scanGoalMaterializedFileV0(projectRoot, state, path, info),
		)
		return nil
	})
	if errors.Is(walkErr, errGoalMaterializedRefsScanDoneV0) {
		walkErr = nil
	}
	return scan, walkErr
}

func (source stackGoalMaterializedRefsSourceV0) scanGoalMaterializedFileV0(
	projectRoot string,
	state orquestagoal.GoalWorkStateV0,
	path string,
	info fs.FileInfo,
) goalMaterializedRefsScanV0 {
	scan := goalMaterializedRefsScanV0{FilesScanned: 1}
	base := strings.ToLower(strings.TrimSpace(filepath.Base(path)))
	switch base {
	case goalMaterializedWorkDeliveryFileV0:
		scan.Result.DomainReceiptRefs = []string{goalMaterializedWorkDeliveryRefV0(projectRoot, path, state.RunRef)}
		scan.HasTerminalReceipt = true
	case goalMaterializedOPESReworkDeliveryFileV0:
		scan.HasTerminalReceipt = true
	case goalMaterializedCheckpointFileV0:
		scan.Result.ArtifactRefs = []string{goalMaterializedCheckpointRefV0(projectRoot, path, state.RunRef)}
	case goalMaterializedPhase0CheckpointDeliveryFileV0:
		scan.HasPhase0Delivery = true
		scan.Result.ArtifactRefs = []string{goalMaterializedArtifactRefV0(projectRoot, path, state.RunRef)}
	}
	if goalMaterializedFileIsGoalResultV0(base) {
		scan.HasTerminalReceipt = true
		if result, ok := goalMaterializedReadTerminalGoalResultV0(path); ok {
			scan.TerminalResult = &result
			scan.HasArtifact = scan.HasArtifact || len(result.ArtifactRefs) > 0 || len(result.ArtifactPaths) > 0
			scan.HasQAPass = scan.HasQAPass || goalMaterializedGoalResultRequiredTestsPassedV0(result)
			scan.MissingTestKeys = append(scan.MissingTestKeys, goalMaterializedGoalResultRequiredTestEvidenceMissingKeysFromResultV0(result)...)
			scan.Result.ArtifactRefs = append(scan.Result.ArtifactRefs, result.ArtifactRefs...)
			scan.Result.DomainReceiptRefs = append(scan.Result.DomainReceiptRefs, result.DomainReceiptRefs...)
			scan.Result.EvidenceRefs = append(scan.Result.EvidenceRefs, result.EvidenceRefs...)
			scan.ArtifactPaths = append(scan.ArtifactPaths, result.ArtifactPaths...)
		}
	}
	if evidence, ok := goalMaterializedReadRequiredTestEvidenceV0(path, info, state); ok {
		scan.Result.EvidenceRefs = append(
			scan.Result.EvidenceRefs,
			goalMaterializedRequiredTestEvidenceFileRefV0(projectRoot, path, state.RunRef),
			evidence.EvidenceRef,
			goalMaterializedRequiredTestEvidenceDetected,
		)
		scan.Result.EvidenceRefs = append(scan.Result.EvidenceRefs, evidence.EvidenceRefs...)
		if evidence.Status == orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 {
			scan.PassedTestKeys = append(scan.PassedTestKeys, goalMaterializedRequiredTestEvidenceKeysV0(state, evidence)...)
		}
		return scan
	}
	if goalMaterializedFileLooksLikeArtifactV0(projectRoot, path, base) {
		scan.HasArtifact = true
		scan.Result.ArtifactRefs = append(scan.Result.ArtifactRefs, goalMaterializedArtifactRefV0(projectRoot, path, state.RunRef))
		if rel := goalMaterializedRelPathV0(projectRoot, path); rel != "" {
			scan.ArtifactPaths = append(scan.ArtifactPaths, rel)
		}
	}
	if goalMaterializedFileLooksLikeQAReportV0(projectRoot, path, base, info, state) {
		scan.HasQAPass = true
		scan.Result.EvidenceRefs = append(scan.Result.EvidenceRefs, goalMaterializedQAPassRefV0(projectRoot, path, state.RunRef))
	}
	if validPaths, invalidPaths, ok := goalMaterializedReadQAArtifactPathListsV0(projectRoot, path, base, info, state); ok {
		scan.ValidArtifactPaths = append(scan.ValidArtifactPaths, validPaths...)
		scan.InvalidArtifactPaths = append(scan.InvalidArtifactPaths, invalidPaths...)
	}
	if goalMaterializedFileLooksLikeQAFailReportV0(projectRoot, path, base, info, state) {
		scan.HasQAFail = true
		scan.Result.EvidenceRefs = append(scan.Result.EvidenceRefs, goalMaterializedQAFailRefV0(projectRoot, path, state.RunRef))
		scan.Result.IssueCodes = append(scan.Result.IssueCodes, orquestamcp.MCPGoalFirstQAFailedPublicTextV0)
	}
	return scan
}

func goalMaterializedOPESOutOfScopeArtifactsV0(
	projectRoot string,
	state orquestagoal.GoalWorkStateV0,
) []string {
	state = orquestagoal.NormalizeGoalWorkStateV0(state)
	if !goalMaterializedStateLooksLikeOPESV0(state) ||
		state.LastResult == nil ||
		strings.TrimSpace(state.LastResult.Status) != orquestagoal.GoalStatusCompleteV0 {
		return nil
	}
	topicRoots := goalMaterializedOPESTopicRootsForStateV0(projectRoot, state)
	if len(topicRoots) == 0 {
		return nil
	}
	allowedRoots := goalMaterializedWriteSetAbsRootsV0(projectRoot, state)
	out := make([]string, 0)
	filesScanned := 0
	for _, topicRoot := range topicRoots {
		if len(out) >= goalMaterializedOPESOutOfScopeMaxRefsV0 ||
			filesScanned >= goalMaterializedOPESOutOfScopeMaxFilesV0 {
			break
		}
		walkErr := filepath.WalkDir(topicRoot, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry == nil {
				return nil
			}
			if !pathWithinRootV0(projectRoot, path) {
				return nil
			}
			if entry.IsDir() {
				if filepath.Clean(path) != filepath.Clean(topicRoot) &&
					goalMaterializedPathWithinAnyRootV0(allowedRoots, path) {
					return fs.SkipDir
				}
				switch strings.ToLower(strings.TrimSpace(entry.Name())) {
				case ".git", "node_modules", "vendor":
					return fs.SkipDir
				default:
					return nil
				}
			}
			filesScanned++
			if filesScanned > goalMaterializedOPESOutOfScopeMaxFilesV0 {
				return errGoalMaterializedRefsScanDoneV0
			}
			if goalMaterializedPathWithinAnyRootV0(allowedRoots, path) {
				return nil
			}
			info, infoErr := entry.Info()
			if infoErr != nil {
				return nil
			}
			base := strings.ToLower(strings.TrimSpace(filepath.Base(path)))
			if !goalMaterializedFileLooksLikeArtifactV0(projectRoot, path, base) &&
				!goalMaterializedPathLooksLikeQAReportV0(projectRoot, path, base) &&
				!goalMaterializedFileIsGoalResultV0(base) {
				return nil
			}
			if info.Size() <= 0 {
				return nil
			}
			if rel := goalMaterializedRelPathV0(projectRoot, path); rel != "" {
				out = append(out, rel)
			}
			if len(out) >= goalMaterializedOPESOutOfScopeMaxRefsV0 {
				return errGoalMaterializedRefsScanDoneV0
			}
			return nil
		})
		if walkErr != nil && !errors.Is(walkErr, errGoalMaterializedRefsScanDoneV0) {
			continue
		}
	}
	return compactStringsV0(out)
}

func goalMaterializedOPESTopicRootsForStateV0(
	projectRoot string,
	state orquestagoal.GoalWorkStateV0,
) []string {
	roots := make([]string, 0)
	for _, scope := range state.Spec.WriteSet {
		relScope := filepath.Clean(filepath.FromSlash(strings.TrimSpace(scope.Path)))
		if relScope == "." || relScope == "" || filepath.IsAbs(relScope) ||
			strings.HasPrefix(relScope, ".."+string(filepath.Separator)) || relScope == ".." {
			continue
		}
		parts := strings.Split(relScope, string(filepath.Separator))
		for i, part := range parts {
			if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(part)), "tema_") {
				continue
			}
			rootRel := filepath.Join(parts[:i+1]...)
			if rootRel == relScope {
				continue
			}
			root := filepath.Join(projectRoot, rootRel)
			if !pathWithinRootV0(projectRoot, root) {
				continue
			}
			if info, err := os.Stat(root); err == nil && info.IsDir() {
				roots = append(roots, filepath.Clean(root))
			}
			break
		}
	}
	return compactStringsV0(roots)
}

func goalMaterializedWriteSetAbsRootsV0(
	projectRoot string,
	state orquestagoal.GoalWorkStateV0,
) []string {
	roots := make([]string, 0, len(state.Spec.WriteSet))
	for _, scope := range state.Spec.WriteSet {
		relScope := filepath.Clean(filepath.FromSlash(strings.TrimSpace(scope.Path)))
		if relScope == "." || relScope == "" || filepath.IsAbs(relScope) ||
			strings.HasPrefix(relScope, ".."+string(filepath.Separator)) || relScope == ".." {
			continue
		}
		root := filepath.Join(projectRoot, relScope)
		if pathWithinRootV0(projectRoot, root) {
			roots = append(roots, filepath.Clean(root))
		}
	}
	return compactStringsV0(roots)
}

func goalMaterializedPathWithinAnyRootV0(roots []string, path string) bool {
	for _, root := range roots {
		if pathWithinRootV0(root, path) {
			return true
		}
	}
	return false
}

func mergeGoalMaterializedRefsScanV0(
	current goalMaterializedRefsScanV0,
	next goalMaterializedRefsScanV0,
) goalMaterializedRefsScanV0 {
	current.Result.DomainReceiptRefs = compactStringsV0(append(current.Result.DomainReceiptRefs, next.Result.DomainReceiptRefs...))
	current.Result.ArtifactRefs = compactStringsV0(append(current.Result.ArtifactRefs, next.Result.ArtifactRefs...))
	current.Result.ExpectedReceiptRefs = compactStringsV0(append(current.Result.ExpectedReceiptRefs, next.Result.ExpectedReceiptRefs...))
	current.Result.EvidenceRefs = compactStringsV0(append(current.Result.EvidenceRefs, next.Result.EvidenceRefs...))
	current.Result.IssueCodes = compactStringsV0(append(current.Result.IssueCodes, next.Result.IssueCodes...))
	current.ArtifactPaths = compactStringsV0(append(current.ArtifactPaths, next.ArtifactPaths...))
	current.ValidArtifactPaths = compactStringsV0(append(current.ValidArtifactPaths, next.ValidArtifactPaths...))
	current.InvalidArtifactPaths = compactStringsV0(append(current.InvalidArtifactPaths, next.InvalidArtifactPaths...))
	current.FilesScanned += next.FilesScanned
	current.HasArtifact = current.HasArtifact || next.HasArtifact
	current.HasQAPass = current.HasQAPass || next.HasQAPass
	current.HasQAFail = current.HasQAFail || next.HasQAFail
	current.HasTerminalReceipt = current.HasTerminalReceipt || next.HasTerminalReceipt
	current.HasPhase0Delivery = current.HasPhase0Delivery || next.HasPhase0Delivery
	current.MissingTestKeys = compactStringsV0(append(current.MissingTestKeys, next.MissingTestKeys...))
	current.PassedTestKeys = compactStringsV0(append(current.PassedTestKeys, next.PassedTestKeys...))
	if next.TerminalResult != nil {
		current.TerminalResult = next.TerminalResult
	}
	return current
}

func goalMaterializedReadTerminalGoalResultV0(path string) (orquestagoal.GoalWorkResultV0, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return orquestagoal.GoalWorkResultV0{}, false
	}
	var result orquestagoal.GoalWorkResultV0
	if json.Unmarshal(raw, &result) != nil {
		return orquestagoal.GoalWorkResultV0{}, false
	}
	result = orquestagoal.NormalizeGoalWorkResultV0(result)
	if strings.TrimSpace(result.Status) != orquestagoal.GoalStatusCompleteV0 {
		return orquestagoal.GoalWorkResultV0{}, false
	}
	return result, true
}

func goalMaterializedGoalResultRequiredTestsPassedV0(result orquestagoal.GoalWorkResultV0) bool {
	if len(result.RequiredTestResults) == 0 {
		return false
	}
	for _, test := range result.RequiredTestResults {
		status := strings.ToLower(strings.TrimSpace(test.Status))
		if status != "passed" && status != "pass" && status != "ok" && status != "success" {
			return false
		}
		if len(compactStringsV0(test.EvidenceRefs)) == 0 {
			return false
		}
	}
	return true
}

func goalMaterializedGoalResultRequiredTestEvidenceMissingKeysFromResultV0(result orquestagoal.GoalWorkResultV0) []string {
	keys := make([]string, 0, len(result.RequiredTestResults))
	for _, test := range result.RequiredTestResults {
		status := strings.ToLower(strings.TrimSpace(test.Status))
		if (status == "passed" || status == "pass" || status == "ok" || status == "success") &&
			len(compactStringsV0(test.EvidenceRefs)) == 0 {
			keys = append(keys, goalMaterializedRequiredTestResultKeysV0(test)...)
		}
	}
	return compactStringsV0(keys)
}

func goalMaterializedRequiredTestEvidenceMissingKeysV0(scan goalMaterializedRefsScanV0) []string {
	missing := make([]string, 0, len(scan.MissingTestKeys))
	for _, key := range scan.MissingTestKeys {
		if strings.TrimSpace(key) == "" || goalFirstStringSliceContainsV0(scan.PassedTestKeys, key) {
			continue
		}
		missing = append(missing, key)
	}
	return compactStringsV0(missing)
}

func goalMaterializedRequiredTestResultKeysV0(test orquestagoal.GoalRequiredTestResultV0) []string {
	keys := make([]string, 0, 1)
	if ref := strings.TrimSpace(test.TestRef); ref != "" {
		keys = append(keys, "test-ref:"+ref)
	} else {
		keys = append(keys, "test-ref:missing")
	}
	return compactStringsV0(keys)
}

func goalMaterializedReadRequiredTestEvidenceV0(
	path string,
	info fs.FileInfo,
	state orquestagoal.GoalWorkStateV0,
) (orquestacionnucleoapp.RequiredTestEvidenceV0, bool) {
	if info == nil || info.IsDir() || info.Size() <= 0 || info.Size() > goalMaterializedQAScanMaxBytesV0 {
		return orquestacionnucleoapp.RequiredTestEvidenceV0{}, false
	}
	if !strings.EqualFold(filepath.Ext(path), ".json") {
		return orquestacionnucleoapp.RequiredTestEvidenceV0{}, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return orquestacionnucleoapp.RequiredTestEvidenceV0{}, false
	}
	var evidence orquestacionnucleoapp.RequiredTestEvidenceV0
	if json.Unmarshal(raw, &evidence) != nil {
		return orquestacionnucleoapp.RequiredTestEvidenceV0{}, false
	}
	evidence, err = orquestacionnucleoapp.NewRequiredTestEvidenceV0(evidence)
	if err != nil {
		return orquestacionnucleoapp.RequiredTestEvidenceV0{}, false
	}
	if !goalMaterializedRequiredTestEvidenceBelongsToStateV0(state, evidence) {
		return orquestacionnucleoapp.RequiredTestEvidenceV0{}, false
	}
	return evidence, true
}

func goalMaterializedRequiredTestEvidenceBelongsToStateV0(
	state orquestagoal.GoalWorkStateV0,
	evidence orquestacionnucleoapp.RequiredTestEvidenceV0,
) bool {
	evidenceRunRef := strings.TrimSpace(evidence.RunRef)
	if evidenceRunRef == "" {
		return false
	}
	for _, runRef := range []string{state.RunRef, state.Spec.RunRef} {
		if strings.TrimSpace(runRef) != "" && strings.TrimSpace(runRef) == evidenceRunRef {
			return true
		}
	}
	return false
}

func goalMaterializedRequiredTestEvidenceKeysV0(
	state orquestagoal.GoalWorkStateV0,
	evidence orquestacionnucleoapp.RequiredTestEvidenceV0,
) []string {
	keys := make([]string, 0, 2)
	if command := strings.TrimSpace(evidence.TestCommand); command != "" {
		keys = append(keys, "test-command:"+command)
	}
	for _, test := range state.Spec.RequiredTests {
		if strings.TrimSpace(test.Command) == strings.TrimSpace(evidence.TestCommand) &&
			strings.TrimSpace(test.TestRef) != "" {
			keys = append(keys, "test-ref:"+strings.TrimSpace(test.TestRef))
		}
	}
	return compactStringsV0(keys)
}

func goalMaterializedRequiredTestEvidenceFileRefV0(projectRoot, path, runRef string) string {
	rel := goalMaterializedRelPathV0(projectRoot, path)
	if rel == "" {
		rel = filepath.Base(path)
	}
	return "evidence-ref-materialized-required-test:" +
		safeGoalMaterializedRefPartV0(runRef) + ":" +
		safeGoalMaterializedRefPartV0(rel)
}

func goalMaterializedExpectedChecklistRefsV0(state orquestagoal.GoalWorkStateV0) []string {
	refs := make([]string, 0, len(state.Spec.ArtifactContracts)+len(state.Spec.RequiredTests)+len(state.Spec.ClosurePolicy.RequiredEvidenceRefs))
	for _, contract := range state.Spec.ArtifactContracts {
		if ref := strings.TrimSpace(contract.ArtifactRef); ref != "" {
			refs = append(refs, "expected-artifact-ref:"+safeGoalMaterializedRefPartV0(ref))
		}
	}
	for _, test := range state.Spec.RequiredTests {
		if ref := strings.TrimSpace(test.TestRef); ref != "" {
			refs = append(refs, "expected-required-test-ref:"+safeGoalMaterializedRefPartV0(ref))
		}
	}
	for _, ref := range state.Spec.ClosurePolicy.RequiredEvidenceRefs {
		if ref = strings.TrimSpace(ref); ref != "" {
			refs = append(refs, "expected-evidence-ref:"+safeGoalMaterializedRefPartV0(ref))
		}
	}
	return compactStringsV0(refs)
}

func goalMaterializedExpectedTerminalReceiptRefsV0(state orquestagoal.GoalWorkStateV0) []string {
	return compactStringsV0([]string{
		"expected-terminal-receipt:" + safeGoalMaterializedRefPartV0(goalMaterializedGoalResultFileV0),
		"expected-terminal-receipt:" + safeGoalMaterializedRefPartV0(orquestaruntimecodexgoal.CodexGoalResultFileNameForGoalRefV0(state.GoalRef)),
		"expected-terminal-receipt:" + safeGoalMaterializedRefPartV0(goalMaterializedOPESReworkDeliveryFileV0),
	})
}

func goalMaterializedFileLooksLikeArtifactV0(
	projectRoot string,
	path string,
	base string,
) bool {
	switch strings.ToLower(strings.TrimSpace(base)) {
	case goalMaterializedWorkDeliveryFileV0,
		goalMaterializedOPESReworkDeliveryFileV0:
		return false
	}
	if goalMaterializedFileIsGoalResultV0(base) {
		return false
	}
	if goalMaterializedPathLooksLikeQAReportV0(projectRoot, path, base) {
		return false
	}
	switch strings.ToLower(filepath.Ext(base)) {
	case ".md", ".html", ".json", ".jsonl", ".pdf", ".mp3", ".wav", ".ogg", ".png", ".jpg", ".jpeg", ".webp":
		return true
	default:
		return false
	}
}

func goalMaterializedFileIsGoalResultV0(base string) bool {
	return orquestaruntimecodexgoal.CodexGoalResultFileNameLooksValidV0(base)
}

func goalMaterializedFileLooksLikeQAReportV0(
	projectRoot string,
	path string,
	base string,
	info fs.FileInfo,
	state orquestagoal.GoalWorkStateV0,
) bool {
	if info == nil || info.IsDir() || info.Size() <= 0 || info.Size() > goalMaterializedQAScanMaxBytesV0 {
		return false
	}
	if !goalMaterializedPathLooksLikeQAReportV0(projectRoot, path, base) {
		return false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	if strings.EqualFold(filepath.Ext(base), ".json") {
		var payload any
		if json.Unmarshal(raw, &payload) == nil &&
			goalMaterializedJSONLooksLikeQAPassForContextV0(projectRoot, path, base, state, payload) {
			return true
		}
	}
	return false
}

func goalMaterializedFileLooksLikeQAFailReportV0(
	projectRoot string,
	path string,
	base string,
	info fs.FileInfo,
	state orquestagoal.GoalWorkStateV0,
) bool {
	if info == nil || info.IsDir() || info.Size() <= 0 || info.Size() > goalMaterializedQAScanMaxBytesV0 {
		return false
	}
	if !goalMaterializedPathLooksLikeQAReportV0(projectRoot, path, base) {
		return false
	}
	if !strings.EqualFold(filepath.Ext(base), ".json") {
		return false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var payload any
	if json.Unmarshal(raw, &payload) != nil {
		return false
	}
	return goalMaterializedJSONLooksLikeQAFailForContextV0(projectRoot, path, base, state, payload)
}

func goalMaterializedReadQAArtifactPathListsV0(
	projectRoot string,
	path string,
	base string,
	info fs.FileInfo,
	state orquestagoal.GoalWorkStateV0,
) ([]string, []string, bool) {
	if info == nil || info.IsDir() || info.Size() <= 0 || info.Size() > goalMaterializedQAScanMaxBytesV0 {
		return nil, nil, false
	}
	if !goalMaterializedPathLooksLikeQAReportV0(projectRoot, path, base) ||
		!strings.EqualFold(filepath.Ext(base), ".json") {
		return nil, nil, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, false
	}
	var payload any
	if json.Unmarshal(raw, &payload) != nil {
		return nil, nil, false
	}
	if !goalMaterializedStateOrReportLooksLikeOPESV0(projectRoot, path, base, state, payload) &&
		!goalMaterializedJSONLooksLikeQAPassV0(payload) &&
		!goalMaterializedJSONLooksLikeQAFailV0(payload) {
		return nil, nil, false
	}
	valid, invalid := goalMaterializedCollectQAArtifactPathListsV0(payload)
	valid = goalMaterializedNormalizeQAArtifactPathsV0(valid)
	invalid = goalMaterializedNormalizeQAArtifactPathsV0(invalid)
	return valid, invalid, len(valid)+len(invalid) > 0
}

func goalMaterializedCollectQAArtifactPathListsV0(value any) ([]string, []string) {
	valid := []string{}
	invalid := []string{}
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			key = goalMaterializedCanonicalQAKeyV0(key)
			switch {
			case goalMaterializedQAArtifactListKeyV0(key, true):
				valid = append(valid, goalMaterializedStringsFromQAArtifactValueV0(item)...)
			case goalMaterializedQAArtifactListKeyV0(key, false):
				invalid = append(invalid, goalMaterializedStringsFromQAArtifactValueV0(item)...)
			default:
				if path := goalMaterializedPathFromQAArtifactObjectV0(typed); path != "" {
					if goalMaterializedQAArtifactObjectValidV0(typed) {
						valid = append(valid, path)
					} else if goalMaterializedQAArtifactObjectInvalidV0(typed) {
						invalid = append(invalid, path)
					}
				}
			}
			nestedValid, nestedInvalid := goalMaterializedCollectQAArtifactPathListsV0(item)
			valid = append(valid, nestedValid...)
			invalid = append(invalid, nestedInvalid...)
		}
	case []any:
		for _, item := range typed {
			nestedValid, nestedInvalid := goalMaterializedCollectQAArtifactPathListsV0(item)
			valid = append(valid, nestedValid...)
			invalid = append(invalid, nestedInvalid...)
		}
	}
	return compactStringsV0(valid), compactStringsV0(invalid)
}

func goalMaterializedQAArtifactListKeyV0(key string, valid bool) bool {
	validKeys := map[string]bool{
		"valid_artifact_paths": true, "valid_artifacts": true, "valid_files": true,
		"artifact_paths_valid": true, "artifact_refs_valid": true,
		"artefactos_validos": true, "archivos_validos": true, "ficheros_validos": true,
	}
	invalidKeys := map[string]bool{
		"invalid_artifact_paths": true, "invalid_artifacts": true, "invalid_files": true,
		"artifact_paths_invalid": true, "artifact_refs_invalid": true,
		"rejected_artifact_paths": true, "rejected_artifacts": true,
		"artefactos_invalidos": true, "archivos_invalidos": true, "ficheros_invalidos": true,
		"artefactos_rechazados": true, "archivos_rechazados": true,
	}
	if valid {
		return validKeys[key]
	}
	return invalidKeys[key]
}

func goalMaterializedStringsFromQAArtifactValueV0(value any) []string {
	out := []string{}
	switch typed := value.(type) {
	case string:
		out = append(out, typed)
	case []any:
		for _, item := range typed {
			out = append(out, goalMaterializedStringsFromQAArtifactValueV0(item)...)
		}
	case map[string]any:
		if path := goalMaterializedPathFromQAArtifactObjectV0(typed); path != "" {
			out = append(out, path)
		}
	}
	return compactStringsV0(out)
}

func goalMaterializedPathFromQAArtifactObjectV0(value map[string]any) string {
	for _, key := range []string{"path", "artifact_path", "file", "file_path", "rel_path", "relative_path", "ruta", "fichero"} {
		if text, ok := value[key].(string); ok && strings.TrimSpace(text) != "" {
			return text
		}
	}
	return ""
}

func goalMaterializedQAArtifactObjectValidV0(value map[string]any) bool {
	for _, key := range []string{"valid", "passed", "ok"} {
		if item, ok := value[key]; ok && goalMaterializedQAPassValueTruthyV0(item) {
			return true
		}
	}
	if status, ok := value["status"]; ok && goalMaterializedQAPassValueTruthyV0(status) {
		return true
	}
	return false
}

func goalMaterializedQAArtifactObjectInvalidV0(value map[string]any) bool {
	for _, key := range []string{"valid", "passed", "ok"} {
		if item, ok := value[key]; ok && goalMaterializedQAPassValueFalseyV0(item) {
			return true
		}
	}
	if status, ok := value["status"]; ok && goalMaterializedQAPassValueFalseyV0(status) {
		return true
	}
	return false
}

func goalMaterializedNormalizeQAArtifactPathsV0(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
		if path == "" || path == "." || strings.HasPrefix(path, "../") || filepath.IsAbs(path) {
			continue
		}
		out = append(out, path)
	}
	return compactStringsV0(out)
}

func goalMaterializedPathLooksLikeQAReportV0(
	projectRoot string,
	path string,
	base string,
) bool {
	rel, err := filepath.Rel(filepath.Clean(projectRoot), filepath.Clean(path))
	if err != nil {
		rel = path
	}
	rel = strings.ToLower(filepath.ToSlash(filepath.Clean(rel)))
	base = strings.ToLower(strings.TrimSpace(base))
	return strings.Contains(rel, "09_validacion/") ||
		strings.Contains(rel, "/validacion/") ||
		strings.Contains(base, "qa") ||
		strings.Contains(base, "validacion") ||
		strings.Contains(base, "validation") ||
		strings.Contains(base, "informe")
}

func goalMaterializedJSONLooksLikeQAPassV0(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			key = strings.ToLower(strings.TrimSpace(key))
			switch v := item.(type) {
			case bool:
				if v && (key == "passed" || key == "ok" || strings.HasSuffix(key, "_pass")) {
					return true
				}
			case string:
				text := strings.ToLower(strings.TrimSpace(v))
				if (key == "status" || key == "estado" || strings.HasSuffix(key, "_status")) &&
					(text == "pass" || text == "passed" || text == "ok" || text == "success") {
					return true
				}
				if strings.HasSuffix(key, "_pass") && (text == "true" || text == "ok" || text == "passed") {
					return true
				}
			}
			if goalMaterializedJSONLooksLikeQAPassV0(item) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if goalMaterializedJSONLooksLikeQAPassV0(item) {
				return true
			}
		}
	}
	return false
}

func goalMaterializedJSONLooksLikeQAPassForContextV0(
	projectRoot string,
	path string,
	base string,
	state orquestagoal.GoalWorkStateV0,
	value any,
) bool {
	if goalMaterializedStateOrReportLooksLikeOPESV0(projectRoot, path, base, state, value) {
		return goalMaterializedJSONLooksLikeOPESQAPassV0(value)
	}
	return goalMaterializedJSONLooksLikeQAPassV0(value)
}

func goalMaterializedJSONLooksLikeQAFailForContextV0(
	projectRoot string,
	path string,
	base string,
	state orquestagoal.GoalWorkStateV0,
	value any,
) bool {
	if goalMaterializedStateOrReportLooksLikeOPESV0(projectRoot, path, base, state, value) {
		return goalMaterializedJSONLooksLikeOPESQAFailV0(value)
	}
	return goalMaterializedJSONLooksLikeQAFailV0(value)
}

func goalMaterializedJSONLooksLikeQAFailV0(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			key = goalMaterializedCanonicalQAKeyV0(key)
			switch v := item.(type) {
			case bool:
				if !v && (key == "passed" || key == "ok" || strings.HasSuffix(key, "_pass")) {
					return true
				}
			case string:
				text := strings.ToLower(strings.TrimSpace(v))
				if (key == "status" || key == "estado" || strings.HasSuffix(key, "_status")) &&
					(text == "fail" || text == "failed" || text == "error" || text == "blocked") {
					return true
				}
				if strings.HasSuffix(key, "_pass") && (text == "false" || text == "fail" || text == "failed" || text == "error") {
					return true
				}
			case float64:
				if v == 0 && strings.HasSuffix(key, "_pass") {
					return true
				}
			}
			if goalMaterializedJSONLooksLikeQAFailV0(item) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if goalMaterializedJSONLooksLikeQAFailV0(item) {
				return true
			}
		}
	}
	return false
}

func goalMaterializedStateOrReportLooksLikeOPESV0(
	projectRoot string,
	path string,
	base string,
	state orquestagoal.GoalWorkStateV0,
	value any,
) bool {
	if goalMaterializedTextLooksLikeOPESV0(base) || goalMaterializedTextLooksLikeOPESV0(path) {
		return true
	}
	if rel, err := filepath.Rel(filepath.Clean(projectRoot), filepath.Clean(path)); err == nil &&
		goalMaterializedTextLooksLikeOPESV0(rel) {
		return true
	}
	if goalMaterializedStateLooksLikeOPESV0(state) {
		return true
	}
	return goalMaterializedJSONLooksLikeOPESReportV0(value)
}

func goalMaterializedStateLooksLikeOPESV0(state orquestagoal.GoalWorkStateV0) bool {
	values := []string{
		state.RunRef,
		state.GoalRef,
		state.ExternalGoalRef,
		state.Spec.GoalRef,
		state.Spec.RequestRef,
		state.Spec.RunRef,
		state.Spec.ProjectRef,
		state.Spec.DomainRef,
		state.Spec.WorkKind,
		state.Spec.WorkProfileKind,
		state.Spec.Objective,
	}
	values = append(values, state.EvidenceRefs...)
	values = append(values, state.LaunchReceipt.EvidenceRefs...)
	values = append(values, state.Spec.SkillRefs...)
	values = append(values, state.Spec.AcceptanceCriteria...)
	values = append(values, state.Spec.EvidenceRefs...)
	values = append(values, state.Spec.ClosurePolicy.RequiredEvidenceRefs...)
	for _, ref := range state.Spec.ContextRefs {
		values = append(values, ref.Kind, ref.Ref, ref.Purpose)
	}
	for _, ref := range state.Spec.RuleRefs {
		values = append(values, ref.Kind, ref.Ref, ref.Enforcement)
	}
	for _, scope := range state.Spec.WriteSet {
		values = append(values, scope.Path, scope.Purpose)
	}
	for _, test := range state.Spec.RequiredTests {
		values = append(values, test.TestRef, test.CommandRef, test.Command)
		values = append(values, test.AcceptanceCriteria...)
		values = append(values, test.AcceptanceCriteriaRefs...)
		values = append(values, test.EvidenceRefs...)
	}
	for _, contract := range state.Spec.ArtifactContracts {
		values = append(values, contract.ArtifactRef, contract.ArtifactType)
		values = append(values, contract.EvidenceRefs...)
	}
	if state.LastResult != nil {
		values = append(values,
			state.LastResult.ArtifactRefs...,
		)
		values = append(values, state.LastResult.ArtifactPaths...)
		values = append(values, state.LastResult.DomainReceiptRefs...)
		values = append(values, state.LastResult.EvidenceRefs...)
	}
	for _, value := range values {
		if goalMaterializedTextLooksLikeOPESV0(value) {
			return true
		}
	}
	return false
}

func goalMaterializedJSONLooksLikeOPESReportV0(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if goalMaterializedTextLooksLikeOPESV0(key) {
				return true
			}
			if text, ok := item.(string); ok && goalMaterializedTextLooksLikeOPESV0(text) {
				return true
			}
			if goalMaterializedJSONLooksLikeOPESReportV0(item) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if goalMaterializedJSONLooksLikeOPESReportV0(item) {
				return true
			}
		}
	}
	return false
}

func goalMaterializedTextLooksLikeOPESV0(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return false
	}
	normalized := goalMaterializedCanonicalQAKeyV0(value)
	return normalized == "opes" ||
		strings.HasPrefix(normalized, "opes_") ||
		strings.HasSuffix(normalized, "_opes") ||
		strings.Contains(normalized, "_opes_") ||
		strings.Contains(normalized, "plan_temario")
}

type goalMaterializedOPESQAPassesV0 struct {
	ExtensionPass         bool
	OfficialTextQAPass    bool
	StrictEditorialQAPass bool
}

func goalMaterializedJSONLooksLikeOPESQAPassV0(value any) bool {
	passes := goalMaterializedOPESQAPassesV0{}
	goalMaterializedCollectOPESQAPassesV0(value, &passes)
	return passes.ExtensionPass && passes.OfficialTextQAPass && passes.StrictEditorialQAPass
}

func goalMaterializedJSONLooksLikeOPESQAFailV0(value any) bool {
	return goalMaterializedCollectOPESQAFailV0(value)
}

func goalMaterializedCollectOPESQAFailV0(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if passKind := goalMaterializedOPESQAPassKindForKeyV0(key); passKind != "" &&
				goalMaterializedQAPassValueFalseyV0(item) {
				return true
			}
			if goalMaterializedCollectOPESQAFailV0(item) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if goalMaterializedCollectOPESQAFailV0(item) {
				return true
			}
		}
	}
	return false
}

func goalMaterializedCollectOPESQAPassesV0(value any, passes *goalMaterializedOPESQAPassesV0) {
	if passes == nil || (passes.ExtensionPass && passes.OfficialTextQAPass && passes.StrictEditorialQAPass) {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if passKind := goalMaterializedOPESQAPassKindForKeyV0(key); passKind != "" &&
				(goalMaterializedQAPassValueTruthyV0(item) || goalMaterializedJSONLooksLikeQAPassV0(item)) {
				goalMaterializedMarkOPESQAPassV0(passes, passKind)
			}
			goalMaterializedCollectOPESQAPassesV0(item, passes)
		}
	case []any:
		for _, item := range typed {
			goalMaterializedCollectOPESQAPassesV0(item, passes)
		}
	}
}

func goalMaterializedOPESQAPassKindForKeyV0(key string) string {
	switch goalMaterializedCanonicalQAKeyV0(key) {
	case "extension", "extension_pass", "extension_qa", "extension_qa_pass", "content_extension_pass":
		return "extension"
	case "official_text", "official_text_pass", "official_text_qa", "official_text_qa_pass",
		"officialtextpass", "officialtextqapass", "texto_oficial_pass", "texto_oficial_qa_pass":
		return "official_text"
	case "strict_editorial", "strict_editorial_pass", "strict_editorial_qa", "strict_editorial_qa_pass",
		"stricteditorialpass", "stricteditorialqapass", "editorial_estricta_pass", "editorial_estricta_qa_pass":
		return "strict_editorial"
	default:
		return ""
	}
}

func goalMaterializedMarkOPESQAPassV0(passes *goalMaterializedOPESQAPassesV0, passKind string) {
	switch passKind {
	case "extension":
		passes.ExtensionPass = true
	case "official_text":
		passes.OfficialTextQAPass = true
	case "strict_editorial":
		passes.StrictEditorialQAPass = true
	}
}

func goalMaterializedQAPassValueTruthyV0(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		text := strings.ToLower(strings.TrimSpace(typed))
		return text == "true" || text == "pass" || text == "passed" || text == "ok" || text == "success"
	case float64:
		return typed == 1
	default:
		return false
	}
}

func goalMaterializedQAPassValueFalseyV0(value any) bool {
	switch typed := value.(type) {
	case bool:
		return !typed
	case string:
		text := strings.ToLower(strings.TrimSpace(typed))
		return text == "false" || text == "fail" || text == "failed" || text == "error" || text == "blocked"
	case float64:
		return typed == 0
	default:
		return false
	}
}

func goalMaterializedCanonicalQAKeyV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastSep := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z',
			r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastSep = false
		default:
			if !lastSep {
				builder.WriteByte('_')
				lastSep = true
			}
		}
	}
	return strings.Trim(builder.String(), "_")
}

func goalMaterializedMissingTerminalReceiptHandledV0(state orquestagoal.GoalWorkStateV0) bool {
	if state.LastClosure != nil &&
		(state.LastClosure.Accepted || strings.TrimSpace(state.LastClosure.Status) == orquestagoal.GoalStatusAcceptedV0) {
		return true
	}
	return state.LastResult != nil &&
		state.LastClosure != nil &&
		goalFirstStringSliceContainsV0(state.LastResult.EvidenceRefs, goalFirstRepairReceiptAttemptedEvidenceRefV0)
}

func goalMaterializedArtifactPathsOmittedV0(
	state orquestagoal.GoalWorkStateV0,
	terminalResult *orquestagoal.GoalWorkResultV0,
	materialized []string,
) []string {
	state = orquestagoal.NormalizeGoalWorkStateV0(state)
	if !state.Spec.ClosurePolicy.RequireArtifactPaths {
		return nil
	}
	declaredResults := make([]orquestagoal.GoalWorkResultV0, 0, 2)
	if state.LastResult != nil &&
		strings.TrimSpace(state.LastResult.Status) == orquestagoal.GoalStatusCompleteV0 {
		declaredResults = append(declaredResults, *state.LastResult)
	}
	if terminalResult != nil &&
		strings.TrimSpace(terminalResult.Status) == orquestagoal.GoalStatusCompleteV0 {
		declaredResults = append(declaredResults, *terminalResult)
	}
	if len(declaredResults) == 0 {
		return nil
	}
	declared := map[string]bool{}
	for _, result := range declaredResults {
		for _, path := range result.ArtifactPaths {
			path = filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
			if path != "" && path != "." {
				declared[path] = true
			}
		}
	}
	omitted := make([]string, 0)
	for _, path := range compactStringsV0(materialized) {
		path = filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
		if path == "" || path == "." || declared[path] {
			continue
		}
		omitted = append(omitted, path)
	}
	return compactStringsV0(omitted)
}

func goalFirstStringSliceContainsV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func goalMaterializedRelPathV0(
	projectRoot string,
	path string,
) string {
	rel, err := filepath.Rel(filepath.Clean(projectRoot), filepath.Clean(path))
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." || strings.HasPrefix(rel, "../") || rel == ".." {
		return ""
	}
	return rel
}

func goalMaterializedQAPassRefV0(
	projectRoot string,
	path string,
	runRef string,
) string {
	rel, err := filepath.Rel(filepath.Clean(projectRoot), filepath.Clean(path))
	if err != nil {
		rel = filepath.Base(path)
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	return "evidence-ref-goal-materialized-qa-pass:" + safeGoalMaterializedRefPartV0(runRef) + ":" + safeGoalMaterializedRefPartV0(rel)
}

func goalMaterializedArtifactPathOmittedRefV0(
	runRef string,
	path string,
) string {
	return goalMaterializedArtifactPathsOmittedEvidence + ":" + safeGoalMaterializedRefPartV0(runRef) + ":" + safeGoalMaterializedRefPartV0(path)
}

func goalMaterializedOutOfScopeArtifactRefV0(
	runRef string,
	path string,
) string {
	return goalMaterializedOutOfScopeArtifactsEvidence + ":" + safeGoalMaterializedRefPartV0(runRef) + ":" + safeGoalMaterializedRefPartV0(path)
}

func goalMaterializedValidArtifactRefV0(
	runRef string,
	path string,
) string {
	return "artifact-ref-materialized-valid:" + safeGoalMaterializedRefPartV0(runRef) + ":" + safeGoalMaterializedRefPartV0(path)
}

func goalMaterializedInvalidArtifactRefV0(
	runRef string,
	path string,
) string {
	return "artifact-ref-materialized-invalid:" + safeGoalMaterializedRefPartV0(runRef) + ":" + safeGoalMaterializedRefPartV0(path)
}

func goalMaterializedQAFailRefV0(
	projectRoot string,
	path string,
	runRef string,
) string {
	rel, err := filepath.Rel(filepath.Clean(projectRoot), filepath.Clean(path))
	if err != nil {
		rel = filepath.Base(path)
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	return goalMaterializedQAFailedPublicTextEvidence + ":" + safeGoalMaterializedRefPartV0(runRef) + ":" + safeGoalMaterializedRefPartV0(rel)
}

func pathWithinRootV0(root string, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func goalMaterializedWorkDeliveryRefV0(
	projectRoot string,
	path string,
	runRef string,
) string {
	rel, err := filepath.Rel(filepath.Clean(projectRoot), filepath.Clean(path))
	if err != nil {
		rel = filepath.Base(path)
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	return "domain-receipt-ref-work-delivery:" + safeGoalMaterializedRefPartV0(runRef) + ":" + safeGoalMaterializedRefPartV0(rel)
}

func goalMaterializedArtifactRefV0(
	projectRoot string,
	path string,
	runRef string,
) string {
	rel, err := filepath.Rel(filepath.Clean(projectRoot), filepath.Clean(path))
	if err != nil {
		rel = filepath.Base(path)
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	return "artifact-ref-materialized:" + safeGoalMaterializedRefPartV0(runRef) + ":" + safeGoalMaterializedRefPartV0(rel)
}

func goalMaterializedCheckpointRefV0(
	projectRoot string,
	path string,
	runRef string,
) string {
	rel, err := filepath.Rel(filepath.Clean(projectRoot), filepath.Clean(path))
	if err != nil {
		rel = filepath.Base(path)
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	return "artifact-ref-checkpoint:" + safeGoalMaterializedRefPartV0(runRef) + ":" + safeGoalMaterializedRefPartV0(rel)
}

func safeGoalMaterializedRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	var builder strings.Builder
	lastSep := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastSep = false
		default:
			if !lastSep {
				builder.WriteByte('-')
				lastSep = true
			}
		}
	}
	out := strings.Trim(builder.String(), "-")
	if out == "" {
		return "unknown"
	}
	return out
}
