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
	goalMaterializedRefsMaxFilesV0                     = 64
	goalMaterializedQAScanMaxFilesV0                   = 256
	goalMaterializedQAScanMaxBytesV0                   = 512 * 1024
	goalMaterializedOPESOutOfScopeMaxFilesV0           = 128
	goalMaterializedOPESOutOfScopeMaxRefsV0            = 16
	goalMaterializedWorkDeliveryFileV0                 = "work_delivery.json"
	goalMaterializedOPESReworkDeliveryFileV0           = "opes_topic_rework_delivery.json"
	goalMaterializedGoalResultFileV0                   = "orquesta_goal_result_v0.json"
	goalMaterializedCheckpointFileV0                   = "checkpoint_started.txt"
	goalMaterializedPhase0CheckpointDeliveryFileV0     = "orquesta_phase0_checkpoint_delivery.json"
	goalMaterializedMissingTerminalReceiptEvidence     = "evidence-ref-goal-materialized-missing-terminal-receipt-after-artifacts-pass"
	goalMaterializedArtifactPathsOmittedEvidence       = "evidence-ref-goal-materialized-artifact-paths-omitted"
	goalMaterializedTerminalArtifactMissingEvidence    = "evidence-ref-goal-materialized-terminal-artifact-missing-after-complete"
	goalMaterializedTerminalArtifactNormalizedEvidence = "evidence-ref-goal-materialized-terminal-artifact-path-normalized"
	goalMaterializedOutOfScopeArtifactsEvidence        = "evidence-ref-goal-materialized-out-of-scope-artifacts"
	goalMaterializedQAFailedPublicTextEvidence         = "evidence-ref-goal-materialized-qa-failed-public-text"
	goalMaterializedPartialArtifactsEvidence           = "evidence-ref-goal-materialized-partial-artifacts-written"
	goalMaterializedPhase0NonPublishableEvidence       = "evidence-ref-goal-materialized-phase0-complete-non-publishable"
	goalMaterializedRequiredTestEvidenceMissing        = "evidence-ref-goal-materialized-required-test-evidence-missing"
	goalMaterializedRequiredTestEvidenceDetected       = "evidence-ref-goal-materialized-required-test-evidence-detected"
	goalMaterializedValidArtifactListEvidence          = "evidence-ref-goal-materialized-valid-artifact-list"
	goalMaterializedInvalidArtifactListEvidence        = "evidence-ref-goal-materialized-invalid-artifact-list"
	goalMaterializedTerminalResultEvidence             = "evidence-ref-goal-materialized-terminal-result"
)

var errGoalMaterializedRefsScanDoneV0 = errors.New("goal_materialized_refs_scan_done")

type stackGoalMaterializedRefsSourceV0 struct {
	Config                       ConfigV0
	GoalStateStore               orquestagoal.GoalWorkStateStorePortV0
	GoalClosureValidator         orquestagoal.GoalWorkClosureValidatorPortV0
	RepairMissingTerminalReceipt bool
}

type goalMaterializedRefsScanV0 struct {
	Result                     orquestamcp.MCPDirectorGoalMaterializedRefsV0
	FilesScanned               int
	HasArtifact                bool
	HasCheckpoint              bool
	HasQAPass                  bool
	HasQAFail                  bool
	HasTerminalReceipt         bool
	HasInvalidCanonicalReceipt bool
	HasPhase0Delivery          bool
	TerminalResult             *orquestagoal.GoalWorkResultV0
	ArtifactPaths              []string
	ValidArtifactPaths         []string
	InvalidArtifactPaths       []string
	MissingTestKeys            []string
	PassedTestKeys             []string
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
	canonical, err := source.scanCanonicalGoalMaterializedReceiptV0(projectRoot, state)
	if err != nil {
		return orquestamcp.MCPDirectorGoalMaterializedRefsV0{}, false, err
	}
	scan = mergeGoalMaterializedRefsScanV0(scan, canonical)
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
	if canonical.HasInvalidCanonicalReceipt {
		scan.TerminalResult = nil
	} else if canonical.TerminalResult != nil {
		scan.TerminalResult = canonical.TerminalResult
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
	missing, normalizedPaths := goalMaterializedTerminalArtifactPathStateV0(projectRoot, state, scan.TerminalResult)
	for _, normalized := range normalizedPaths {
		result.EvidenceRefs = append(
			result.EvidenceRefs,
			goalMaterializedTerminalArtifactNormalizedEvidence,
			goalMaterializedTerminalArtifactNormalizedRefV0(state.RunRef, normalized.Declared, normalized.Actual),
		)
	}
	if len(missing) > 0 {
		result.IssueCodes = append(result.IssueCodes, orquestamcp.MCPGoalFirstTerminalArtifactMissingAfterCompleteV0)
		result.EvidenceRefs = append(result.EvidenceRefs, goalMaterializedTerminalArtifactMissingEvidence)
		for _, path := range missing {
			result.EvidenceRefs = append(result.EvidenceRefs, goalMaterializedTerminalArtifactMissingRefV0(state.RunRef, path))
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
	if scan.HasCheckpoint {
		result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-goal-materialized-checkpoint-detected")
		if !scan.HasTerminalReceipt {
			result.IssueCodes = append(result.IssueCodes, "goal_first_materialized_checkpoint_detected")
		}
	}
	result.ExpectedReceiptRefs = append(result.ExpectedReceiptRefs, goalMaterializedExpectedChecklistRefsV0(state)...)
	result.ExpectedReceiptRefs = compactStringsV0(result.ExpectedReceiptRefs)
	result.EvidenceRefs = compactStringsV0(result.EvidenceRefs)
	result.IssueCodes = compactStringsV0(result.IssueCodes)
	return result, true, nil
}

func (source stackGoalMaterializedRefsSourceV0) scanCanonicalGoalMaterializedReceiptV0(
	projectRoot string,
	state orquestagoal.GoalWorkStateV0,
) (goalMaterializedRefsScanV0, error) {
	goalRef := goalMaterializedStateGoalRefV0(state)
	if goalRef == "" {
		return goalMaterializedRefsScanV0{}, nil
	}
	relativeDir := filepath.FromSlash(orquestaruntimecodexgoal.CodexGoalRuntimeReceiptRelativeDirV0(goalRef))
	path := filepath.Join(projectRoot, relativeDir, orquestaruntimecodexgoal.CodexGoalResultFileNameForGoalRefV0(goalRef))
	if !pathWithinRootV0(projectRoot, path) {
		return goalMaterializedRefsScanV0{}, nil
	}
	info, err := os.Lstat(path)
	if err != nil {
		return goalMaterializedRefsScanV0{}, nil
	}
	invalid := goalMaterializedRefsScanV0{FilesScanned: 1, HasTerminalReceipt: true, HasInvalidCanonicalReceipt: true}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > goalMaterializedQAScanMaxBytesV0 {
		return invalid, nil
	}
	resolvedRoot, rootErr := filepath.EvalSymlinks(projectRoot)
	resolvedPath, pathErr := filepath.EvalSymlinks(path)
	if rootErr != nil || pathErr != nil || !pathWithinRootV0(resolvedRoot, resolvedPath) {
		return invalid, nil
	}
	result, ok := goalMaterializedReadValidTerminalGoalResultForStateV0(projectRoot, path, state)
	if !ok || !goalMaterializedTerminalResultExactlyMatchesStateV0(result, state) {
		return invalid, nil
	}
	scan := source.scanGoalMaterializedFileV0(projectRoot, state, path, info)
	scan.TerminalResult = &result
	scan.HasInvalidCanonicalReceipt = false
	return scan, nil
}

func (source stackGoalMaterializedRefsSourceV0) LoadTerminalGoalMaterializedResultV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
) (orquestagoal.GoalWorkResultV0, bool, error) {
	_ = ctx
	projectRoot := strings.TrimSpace(source.Config.Codex.ProjectWorkDir)
	if projectRoot == "" {
		return orquestagoal.GoalWorkResultV0{}, false, nil
	}
	projectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return orquestagoal.GoalWorkResultV0{}, false, nil
	}
	projectRoot = filepath.Clean(projectRoot)
	canonical, err := source.scanCanonicalGoalMaterializedReceiptV0(projectRoot, state)
	if err != nil {
		return orquestagoal.GoalWorkResultV0{}, false, err
	}
	if canonical.HasInvalidCanonicalReceipt {
		return orquestagoal.GoalWorkResultV0{}, false, nil
	}
	if canonical.TerminalResult != nil {
		return *canonical.TerminalResult, true, nil
	}
	for _, scope := range state.Spec.WriteSet {
		result, ok, err := source.loadTerminalGoalMaterializedResultFromScopeV0(projectRoot, state, scope)
		if err != nil || ok {
			return result, ok, err
		}
	}
	return orquestagoal.GoalWorkResultV0{}, false, nil
}

func (source stackGoalMaterializedRefsSourceV0) loadTerminalGoalMaterializedResultFromScopeV0(
	projectRoot string,
	state orquestagoal.GoalWorkStateV0,
	scope orquestagoal.GoalWriteScopeV0,
) (orquestagoal.GoalWorkResultV0, bool, error) {
	relScope := filepath.Clean(filepath.FromSlash(strings.TrimSpace(scope.Path)))
	if relScope == "." || relScope == "" || filepath.IsAbs(relScope) ||
		strings.HasPrefix(relScope, ".."+string(filepath.Separator)) || relScope == ".." {
		return orquestagoal.GoalWorkResultV0{}, false, nil
	}
	target := filepath.Join(projectRoot, relScope)
	if !pathWithinRootV0(projectRoot, target) {
		return orquestagoal.GoalWorkResultV0{}, false, nil
	}
	info, err := os.Stat(target)
	if err != nil {
		return orquestagoal.GoalWorkResultV0{}, false, nil
	}
	if !info.IsDir() {
		result, ok := goalMaterializedReadValidTerminalGoalResultForStateV0(projectRoot, target, state)
		return result, ok, nil
	}
	var found orquestagoal.GoalWorkResultV0
	foundOK := false
	filesScanned := 0
	walkErr := filepath.WalkDir(target, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || foundOK {
			return nil
		}
		if entry != nil && entry.IsDir() {
			if path != target && goalMaterializedResultSkipDirV0(entry.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if filesScanned >= goalMaterializedQAScanMaxFilesV0 {
			return errGoalMaterializedRefsScanDoneV0
		}
		if entry == nil || !pathWithinRootV0(projectRoot, path) {
			return nil
		}
		filesScanned++
		if !goalMaterializedFileIsGoalResultV0(filepath.Base(path)) {
			return nil
		}
		result, ok := goalMaterializedReadValidTerminalGoalResultForStateV0(projectRoot, path, state)
		if !ok {
			return nil
		}
		found = result
		foundOK = true
		return errGoalMaterializedRefsScanDoneV0
	})
	if errors.Is(walkErr, errGoalMaterializedRefsScanDoneV0) {
		walkErr = nil
	}
	return found, foundOK, walkErr
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
		if entry != nil && entry.IsDir() {
			if path != target && goalMaterializedResultSkipDirV0(entry.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if len(scan.Result.DomainReceiptRefs)+len(scan.Result.ArtifactRefs) >= goalMaterializedRefsMaxFilesV0 ||
			scan.FilesScanned >= goalMaterializedQAScanMaxFilesV0 {
			return errGoalMaterializedRefsScanDoneV0
		}
		if entry == nil {
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
	if goalMaterializedFileLooksLikeCheckpointV0(base) {
		scan.HasCheckpoint = true
		scan.Result.ArtifactRefs = []string{goalMaterializedCheckpointRefV0(projectRoot, path, state.RunRef)}
	}
	switch base {
	case goalMaterializedWorkDeliveryFileV0:
		scan.Result.DomainReceiptRefs = []string{goalMaterializedWorkDeliveryRefV0(projectRoot, path, state.RunRef)}
		scan.HasTerminalReceipt = true
	case goalMaterializedOPESReworkDeliveryFileV0:
		scan.HasTerminalReceipt = true
	case goalMaterializedPhase0CheckpointDeliveryFileV0:
		scan.HasPhase0Delivery = true
		scan.Result.ArtifactRefs = []string{goalMaterializedArtifactRefV0(projectRoot, path, state.RunRef)}
	}
	if goalMaterializedFileIsGoalResultV0(base) {
		if result, ok := goalMaterializedReadScannableTerminalGoalResultForStateV0(path, state); ok {
			scan.HasTerminalReceipt = true
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
	qa := goalMaterializedReadQAFileClassificationV0(projectRoot, path, base, info, state)
	if qa.Passed {
		scan.HasQAPass = true
		scan.Result.EvidenceRefs = append(scan.Result.EvidenceRefs, goalMaterializedQAPassRefV0(projectRoot, path, state.RunRef))
	}
	if len(qa.ValidArtifactPaths)+len(qa.InvalidArtifactPaths) > 0 {
		scan.ValidArtifactPaths = append(scan.ValidArtifactPaths, qa.ValidArtifactPaths...)
		scan.InvalidArtifactPaths = append(scan.InvalidArtifactPaths, qa.InvalidArtifactPaths...)
	}
	if qa.Failed {
		scan.HasQAFail = true
		scan.Result.EvidenceRefs = append(scan.Result.EvidenceRefs, goalMaterializedQAFailRefV0(projectRoot, path, state.RunRef))
		scan.Result.IssueCodes = append(scan.Result.IssueCodes, orquestamcp.MCPGoalFirstQAFailedPublicTextV0)
	}
	return scan
}

func goalMaterializedReadScannableTerminalGoalResultForStateV0(
	path string,
	state orquestagoal.GoalWorkStateV0,
) (orquestagoal.GoalWorkResultV0, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return orquestagoal.GoalWorkResultV0{}, false
	}
	result, err := goalMaterializedDecodeGoalWorkResultV0(raw)
	if err != nil {
		return orquestagoal.GoalWorkResultV0{}, false
	}
	if strings.TrimSpace(result.Status) != orquestagoal.GoalStatusCompleteV0 ||
		!goalMaterializedTerminalResultMatchesStateV0(result, state) {
		return orquestagoal.GoalWorkResultV0{}, false
	}
	return result, true
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
	current.HasCheckpoint = current.HasCheckpoint || next.HasCheckpoint
	current.HasQAPass = current.HasQAPass || next.HasQAPass
	current.HasQAFail = current.HasQAFail || next.HasQAFail
	current.HasTerminalReceipt = current.HasTerminalReceipt || next.HasTerminalReceipt
	current.HasInvalidCanonicalReceipt = current.HasInvalidCanonicalReceipt || next.HasInvalidCanonicalReceipt
	current.HasPhase0Delivery = current.HasPhase0Delivery || next.HasPhase0Delivery
	current.MissingTestKeys = compactStringsV0(append(current.MissingTestKeys, next.MissingTestKeys...))
	current.PassedTestKeys = compactStringsV0(append(current.PassedTestKeys, next.PassedTestKeys...))
	if next.TerminalResult != nil {
		current.TerminalResult = next.TerminalResult
	}
	return current
}

func goalMaterializedReadValidTerminalGoalResultForStateV0(
	projectRoot string,
	path string,
	state orquestagoal.GoalWorkStateV0,
) (orquestagoal.GoalWorkResultV0, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return orquestagoal.GoalWorkResultV0{}, false
	}
	result, err := goalMaterializedDecodeGoalWorkResultV0(raw)
	if err != nil {
		return orquestagoal.GoalWorkResultV0{}, false
	}
	if !orquestagoal.GoalWorkResultTerminalV0(result.Status) ||
		!goalMaterializedTerminalResultMatchesStateV0(result, state) ||
		len(orquestagoal.ValidateGoalWorkResultV0(result)) > 0 {
		return orquestagoal.GoalWorkResultV0{}, false
	}
	result.EvidenceRefs = compactStringsV0(append(
		result.EvidenceRefs,
		goalMaterializedTerminalResultRefV0(projectRoot, path, state.RunRef),
	))
	return result, true
}

func goalMaterializedDecodeGoalWorkResultV0(raw []byte) (orquestagoal.GoalWorkResultV0, error) {
	decoded, err := orquestagoal.DecodeGoalWorkResultJSONV0(raw)
	if err != nil {
		return orquestagoal.GoalWorkResultV0{}, err
	}
	if decoded.Disposition == orquestagoal.GoalWorkResultJSONDispositionIrrecoverableV0 {
		return orquestagoal.GoalWorkResultV0{}, errors.New("goal_result_json_irrecoverable")
	}
	return decoded.Result, nil
}

func goalMaterializedTerminalResultMatchesStateV0(
	result orquestagoal.GoalWorkResultV0,
	state orquestagoal.GoalWorkStateV0,
) bool {
	goalRef := goalMaterializedStateGoalRefV0(state)
	resultGoalRef := strings.TrimSpace(result.GoalRef)
	if resultGoalRef == "" {
		return goalRef != ""
	}
	return goalRef != "" && resultGoalRef == goalRef
}

func goalMaterializedTerminalResultExactlyMatchesStateV0(
	result orquestagoal.GoalWorkResultV0,
	state orquestagoal.GoalWorkStateV0,
) bool {
	if strings.TrimSpace(result.GoalRef) == "" ||
		strings.TrimSpace(result.GoalRef) != goalMaterializedStateGoalRefV0(state) {
		return false
	}
	resultExternal := strings.TrimSpace(result.ExternalGoalRef)
	stateExternal := strings.TrimSpace(state.ExternalGoalRef)
	return resultExternal == "" || stateExternal == "" || resultExternal == stateExternal
}

func goalMaterializedStateGoalRefV0(state orquestagoal.GoalWorkStateV0) string {
	for _, goalRef := range []string{state.GoalRef, state.Spec.GoalRef, state.LaunchReceipt.GoalRef} {
		if goalRef = strings.TrimSpace(goalRef); goalRef != "" {
			return goalRef
		}
	}
	return ""
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
	if goalMaterializedFileLooksLikeCheckpointV0(base) {
		return false
	}
	if strings.HasSuffix(strings.ToLower(strings.TrimSpace(base)), "_artifact.txt") {
		return true
	}
	switch strings.ToLower(filepath.Ext(base)) {
	case ".md", ".html", ".json", ".jsonl", ".pdf", ".mp3", ".wav", ".ogg", ".png", ".jpg", ".jpeg", ".webp":
		return true
	default:
		return false
	}
}

func goalMaterializedFileLooksLikeCheckpointV0(base string) bool {
	base = strings.ToLower(strings.TrimSpace(base))
	return base == goalMaterializedCheckpointFileV0 ||
		(strings.HasPrefix(base, "checkpoint_started") && strings.HasSuffix(base, ".txt"))
}

func goalMaterializedFileIsGoalResultV0(base string) bool {
	return orquestaruntimecodexgoal.CodexGoalResultFileNameLooksValidV0(base)
}

func goalMaterializedResultSkipDirV0(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case ".git", ".codex", ".gocache", ".gocache-local", "node_modules", "vendor":
		return true
	default:
		return false
	}
}

func goalMaterializedReadQAFileClassificationV0(
	projectRoot string,
	path string,
	base string,
	info fs.FileInfo,
	state orquestagoal.GoalWorkStateV0,
) goalMaterializedQAClassificationV0 {
	if info == nil || info.IsDir() || info.Size() <= 0 || info.Size() > goalMaterializedQAScanMaxBytesV0 {
		return goalMaterializedQAClassificationV0{}
	}
	if !goalMaterializedPathLooksLikeQAReportV0(projectRoot, path, base) ||
		!strings.EqualFold(filepath.Ext(base), ".json") {
		return goalMaterializedQAClassificationV0{}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return goalMaterializedQAClassificationV0{}
	}
	return goalMaterializedClassifyQAPayloadV0(projectRoot, path, base, state, raw)
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

type goalMaterializedArtifactPathNormalizationV0 struct {
	Declared string
	Actual   string
}

func goalMaterializedTerminalArtifactPathStateV0(
	projectRoot string,
	state orquestagoal.GoalWorkStateV0,
	terminalResult *orquestagoal.GoalWorkResultV0,
) ([]string, []goalMaterializedArtifactPathNormalizationV0) {
	state = orquestagoal.NormalizeGoalWorkStateV0(state)
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
		return nil, nil
	}
	projectRoot = filepath.Clean(projectRoot)
	allowedRoots := goalMaterializedWriteSetAbsRootsV0(projectRoot, state)
	missing := make([]string, 0)
	normalized := make([]goalMaterializedArtifactPathNormalizationV0, 0)
	for _, result := range declaredResults {
		for _, artifactPath := range result.ArtifactPaths {
			rel := filepath.ToSlash(filepath.Clean(strings.TrimSpace(artifactPath)))
			if rel == "" || rel == "." || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, "../") {
				continue
			}
			target := filepath.Join(projectRoot, filepath.FromSlash(rel))
			if !pathWithinRootV0(projectRoot, target) {
				continue
			}
			if len(allowedRoots) > 0 && !goalMaterializedPathWithinAnyRootV0(allowedRoots, target) {
				continue
			}
			if _, err := os.Stat(target); err == nil {
				continue
			} else if os.IsNotExist(err) {
				if actual, ok := goalMaterializedNearbyArtifactPathV0(projectRoot, allowedRoots, target); ok {
					normalized = append(normalized, goalMaterializedArtifactPathNormalizationV0{Declared: rel, Actual: actual})
					continue
				}
				missing = append(missing, rel)
			}
		}
	}
	return compactStringsV0(missing), normalized
}

func goalMaterializedNearbyArtifactPathV0(projectRoot string, allowedRoots []string, target string) (string, bool) {
	directory := filepath.Dir(target)
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", false
	}
	want := filepath.Base(target)
	matches := make([]string, 0, 1)
	for _, entry := range entries {
		name := entry.Name()
		candidate := filepath.Join(directory, name)
		if !goalMaterializedOneSubstitutionV0(name, want) || !pathWithinRootV0(projectRoot, candidate) ||
			(len(allowedRoots) > 0 && !goalMaterializedPathWithinAnyRootV0(allowedRoots, candidate)) || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		matches = append(matches, candidate)
		if len(matches) > 1 {
			return "", false
		}
	}
	if len(matches) != 1 {
		return "", false
	}
	rel := goalMaterializedRelPathV0(projectRoot, matches[0])
	return rel, rel != ""
}

func goalMaterializedOneSubstitutionV0(actual string, expected string) bool {
	if len(actual) != len(expected) || actual == expected {
		return false
	}
	differences := 0
	for index := range actual {
		if actual[index] == expected[index] {
			continue
		}
		differences++
		if differences > 1 {
			return false
		}
	}
	return differences == 1
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

func goalMaterializedTerminalResultRefV0(
	projectRoot string,
	path string,
	runRef string,
) string {
	rel := goalMaterializedRelPathV0(projectRoot, path)
	if rel == "" {
		rel = filepath.Base(path)
	}
	return goalMaterializedTerminalResultEvidence + ":" +
		safeGoalMaterializedRefPartV0(runRef) + ":" +
		safeGoalMaterializedRefPartV0(rel)
}

func goalMaterializedArtifactPathOmittedRefV0(
	runRef string,
	path string,
) string {
	return goalMaterializedArtifactPathsOmittedEvidence + ":" + safeGoalMaterializedRefPartV0(runRef) + ":" + safeGoalMaterializedRefPartV0(path)
}

func goalMaterializedTerminalArtifactMissingRefV0(
	runRef string,
	path string,
) string {
	return goalMaterializedTerminalArtifactMissingEvidence + ":" + safeGoalMaterializedRefPartV0(runRef) + ":" + safeGoalMaterializedRefPartV0(path)
}

func goalMaterializedTerminalArtifactNormalizedRefV0(runRef string, declared string, actual string) string {
	return goalMaterializedTerminalArtifactNormalizedEvidence + ":" + safeGoalMaterializedRefPartV0(runRef) + ":" +
		safeGoalMaterializedRefPartV0(declared) + ":" + safeGoalMaterializedRefPartV0(actual)
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
