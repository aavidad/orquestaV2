package orquestaappcodexstack

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

type AutoprogrammingPromotionConfigV0 struct {
	Enabled                    bool
	Port                       orquestaautoprogramming.AutoprogrammingStagingPromotionPortV0
	GoalFirstSnapshotStore     orquestaruntimeworktree.WorktreeSnapshotStorePortV0
	GoalWorkspaceProvisioner   orquestaruntimeworktree.GoalWorkspaceProvisionerPortV0
	GoalWorkspaceIntegration   orquestaruntimeworktree.GoalWorkspaceIntegrationPortV0
	BatchTestRunner            AutoprogrammingBatchTestRunnerPortV0
	BatchPromotionFinalizer    AutoprogrammingBatchPromotionFinalizerPortV0
	BatchPromotionReconciler   AutoprogrammingBatchPromotionClaimReconcilerPortV0
	CanonicalWorkDir           string
	GoalWorkspaceRoot          string
	BatchIntegrationReceiptDir string
	BatchPromotionReceiptDir   string
	AppRef                     string
	RepoRef                    string
	CommitMessage              string
}

const autoprogrammingGoalFirstPromotionCompleteEvidenceV0 = "evidence-ref-codex-stack-autoprogramming-goal-first-promotion-complete:"
const autoprogrammingGoalFirstPromotionPendingEvidenceV0 = "evidence-ref-codex-stack-autoprogramming-goal-first-promotion-pending"
const autoprogrammingGoalFirstPromotionRefEvidenceV0 = "evidence-ref-codex-stack-autoprogramming-goal-first-promotion-ref:"
const autoprogrammingGoalFirstIntegrationReceiptEvidenceV0 = "evidence-ref-codex-stack-autoprogramming-goal-first-integration-receipt:"
const autoprogrammingGoalFirstCommitEvidenceV0 = "evidence-ref-codex-stack-autoprogramming-goal-first-commit:"
const autoprogrammingGoalFirstArchiveEvidenceV0 = "evidence-ref-codex-stack-autoprogramming-goal-first-archive:"

func (stack *StackV0) maybePromoteClosedAutoprogrammingRunV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, []string, error) {
	coordinator := stack.goalFirstPromotionCoordinatorV0()
	if coordinator == nil {
		return false, nil, fmt.Errorf("autoprogramming_goal_first_promotion_coordinator_unavailable")
	}
	release, err := coordinator.acquireV0(ctx, run.RunID)
	if err != nil {
		return false, nil, err
	}
	defer release()
	return stack.maybePromoteClosedAutoprogrammingRunSerializedV0(ctx, run)
}

func (stack *StackV0) maybePromoteClosedAutoprogrammingRunSerializedV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, []string, error) {
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		handled, complete, refs, err := stack.maybeFinalizeClosedAutoprogrammingBatchV0(ctx, run)
		if handled || err != nil {
			return complete, refs, err
		}
	}
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		return true, nil, nil
	}
	goalState, goalFirst, err := stack.autoprogrammingPromotionGoalStateV0(ctx, run)
	if err != nil {
		return false, nil, err
	}
	if goalFirst {
		if autoprogrammingGoalFirstPromotionCompletionVerifiedV0(goalState, run.RunID) {
			return true, compactStringsV0(goalState.EvidenceRefs), nil
		}
		if err := stack.persistAutoprogrammingGoalFirstPromotionEvidenceV0(ctx, run.RunID, []string{
			autoprogrammingGoalFirstPromotionPendingEvidenceV0,
		}); err != nil {
			return false, nil, err
		}
	}
	config := stack.AutoprogrammingPromotion
	if !config.Enabled || config.Port == nil {
		if goalFirst {
			return false, []string{
				autoprogrammingGoalFirstPromotionPendingEvidenceV0,
				"evidence-ref-codex-stack-autoprogramming-promotion-port-unavailable",
			}, nil
		}
		return true, nil, nil
	}
	request, ok, err := stack.autoprogrammingPromotionRequestV0(ctx, run)
	if err != nil {
		return false, nil, err
	}
	if !ok {
		if refs, missing := stack.autoprogrammingPromotionMissingGoalStateEvidenceRefsV0(ctx, run); missing {
			return false, refs, nil
		}
		return true, nil, nil
	}
	if goalFirst {
		if refs, verified := stack.autoprogrammingPromotionGoalFirstWriteSetVerifiedV0(ctx, goalState, request); !verified {
			if err := stack.persistAutoprogrammingGoalFirstPromotionEvidenceV0(ctx, run.RunID, append(refs, autoprogrammingGoalFirstPromotionPendingEvidenceV0)); err != nil {
				return false, refs, err
			}
			return false, refs, nil
		}
	}
	decision := orquestaautoprogramming.EvaluateAutoprogrammingStagingPromotionV0(request)
	if !decision.Ready {
		if goalFirst {
			if err := stack.persistAutoprogrammingGoalFirstPromotionEvidenceV0(ctx, run.RunID, append(decision.EvidenceRefs, autoprogrammingGoalFirstPromotionPendingEvidenceV0)); err != nil {
				return false, decision.EvidenceRefs, err
			}
		}
		return false, decision.EvidenceRefs, nil
	}
	promoted, err := config.Port.PromoteAutoprogrammingStagingV0(ctx, decision.PromotionCommand)
	if err != nil {
		return false, promoted.EvidenceRefs, err
	}
	promoted = autoprogrammingPromotionEffectWithIntegrationStatusV0(promoted)
	refs := compactStringsV0(append(decision.EvidenceRefs, autoprogrammingPromotionEffectEvidenceRefsV0(promoted)...))
	if !autoprogrammingPromotionEffectCompleteV0(promoted) {
		if goalFirst {
			if err := stack.persistAutoprogrammingGoalFirstPromotionEvidenceV0(ctx, run.RunID, append(refs, autoprogrammingGoalFirstPromotionPendingEvidenceV0)); err != nil {
				return false, refs, err
			}
		}
		return false, refs, nil
	}
	archived, err := config.Port.ArchiveAutoprogrammingStagingV0(ctx, decision.CleanupCommand)
	if err != nil {
		return false, compactStringsV0(append(refs, archived.EvidenceRefs...)), err
	}
	refs = compactStringsV0(append(refs, archived.EvidenceRefs...))
	complete := archived.Status == orquestaautoprogramming.AutoprogrammingStagingEffectArchivedV0
	if complete && goalFirst {
		completionRefs := autoprogrammingGoalFirstPromotionCompletionEvidenceRefsV0(run.RunID, goalState.GoalRef, promoted, archived)
		refs = compactStringsV0(append(refs, completionRefs...))
		if err := stack.persistAutoprogrammingGoalFirstPromotionEvidenceV0(ctx, run.RunID, refs); err != nil {
			return false, refs, err
		}
	}
	return complete, refs, nil
}

func autoprogrammingGoalFirstPromotionCompletionEvidenceRefsV0(
	runRef string,
	goalRef string,
	promoted orquestaautoprogramming.AutoprogrammingStagingEffectResultV0,
	archived orquestaautoprogramming.AutoprogrammingStagingEffectResultV0,
) []string {
	promotionRef := strings.TrimSpace(promoted.PromotionRef)
	integrationReceiptRef := strings.TrimSpace(promoted.IntegrationReceiptRef)
	commitRef := strings.TrimSpace(promoted.CommitRef)
	archiveRef := strings.TrimSpace(archived.ArchiveRef)
	digest := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(runRef), strings.TrimSpace(goalRef), promotionRef,
		integrationReceiptRef, commitRef, archiveRef,
	}, "\x00")))
	return compactStringsV0([]string{
		autoprogrammingGoalFirstPromotionRefEvidenceV0 + promotionRef,
		autoprogrammingGoalFirstIntegrationReceiptEvidenceV0 + integrationReceiptRef,
		autoprogrammingGoalFirstCommitEvidenceV0 + commitRef,
		autoprogrammingGoalFirstArchiveEvidenceV0 + archiveRef,
		autoprogrammingGoalFirstPromotionCompleteEvidenceV0 + hex.EncodeToString(digest[:16]),
	})
}

func autoprogrammingGoalFirstPromotionCompletionVerifiedV0(
	state orquestagoal.GoalWorkStateV0,
	runRef string,
) bool {
	promotionRef := goalFirstEvidenceSuffixV0(state.EvidenceRefs, autoprogrammingGoalFirstPromotionRefEvidenceV0)
	integrationReceiptRef := goalFirstEvidenceSuffixV0(state.EvidenceRefs, autoprogrammingGoalFirstIntegrationReceiptEvidenceV0)
	commitRef := goalFirstEvidenceSuffixV0(state.EvidenceRefs, autoprogrammingGoalFirstCommitEvidenceV0)
	archiveRef := goalFirstEvidenceSuffixV0(state.EvidenceRefs, autoprogrammingGoalFirstArchiveEvidenceV0)
	if integrationReceiptRef == "" || archiveRef == "" {
		return false
	}
	digest := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(runRef), strings.TrimSpace(state.GoalRef), promotionRef,
		integrationReceiptRef, commitRef, archiveRef,
	}, "\x00")))
	expectedMarker := hex.EncodeToString(digest[:16])
	for _, ref := range state.EvidenceRefs {
		if marker, ok := strings.CutPrefix(strings.TrimSpace(ref), autoprogrammingGoalFirstPromotionCompleteEvidenceV0); ok && strings.TrimSpace(marker) == expectedMarker {
			return true
		}
	}
	return false
}

func goalFirstEvidenceSuffixV0(refs []string, prefix string) string {
	for _, ref := range refs {
		if value, ok := strings.CutPrefix(strings.TrimSpace(ref), prefix); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (stack *StackV0) persistAutoprogrammingGoalFirstPromotionEvidenceV0(
	ctx context.Context,
	runRef string,
	refs []string,
) error {
	store := stack.Ports.GoalStateStore
	if store == nil {
		store = stack.Stores.AppGoalStateStore
	}
	if store == nil {
		return fmt.Errorf("autoprogramming_goal_first_promotion_state_store_required")
	}
	for attempt := 0; attempt < 3; attempt++ {
		state, err := store.LoadGoalWorkStateV0(ctx, runRef)
		if err != nil {
			return err
		}
		if autoprogrammingGoalFirstPromotionRefsContainedV0(state.EvidenceRefs, refs) {
			return nil
		}
		state.EvidenceRefs = compactStringsV0(append(state.EvidenceRefs, refs...))
		state, err = orquestagoal.NewGoalWorkStateV0(state)
		if err != nil {
			return err
		}
		if cas, ok := store.(orquestagoal.GoalWorkStateCASStorePortV0); ok {
			if _, err = cas.CompareAndSwapGoalWorkStateV0(ctx, state.StoreVersion, state); err == nil {
				return nil
			}
			var conflict orquestagoal.GoalWorkStateCASConflictErrorV0
			if !errors.As(err, &conflict) {
				return err
			}
			continue
		}
		return store.SaveGoalWorkStateV0(ctx, state)
	}
	return fmt.Errorf("autoprogramming_goal_first_promotion_state_conflict")
}

func autoprogrammingGoalFirstPromotionRefsContainedV0(current, required []string) bool {
	for _, ref := range compactStringsV0(required) {
		if !goalFirstStringSliceContainsV0(current, ref) {
			return false
		}
	}
	return true
}

func (stack StackV0) autoprogrammingPromotionGoalFirstWriteSetVerifiedV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
	request orquestaautoprogramming.AutoprogrammingStagingPromotionRequestV0,
) ([]string, bool) {
	const evidencePrefix = "evidence-ref-codex-stack-autoprogramming-goal-first-worktree-verify"
	baselineRef := autoprogrammingPromotionGoalContextRefV0(
		state.Spec.ContextRefs,
		"worktree_baseline",
		"",
	)
	if baselineRef == "" {
		return []string{evidencePrefix, evidencePrefix + "-baseline-ref-missing"}, false
	}
	if stack.AutoprogrammingPromotion.GoalFirstSnapshotStore == nil {
		return []string{evidencePrefix, evidencePrefix + "-snapshot-store-missing", baselineRef}, false
	}
	baseline, err := stack.AutoprogrammingPromotion.GoalFirstSnapshotStore.LoadWorktreeSnapshotV0(ctx, baselineRef)
	if err != nil {
		return []string{evidencePrefix, evidencePrefix + "-baseline-unavailable", baselineRef}, false
	}
	if len(baseline.OmittedPaths) > 0 {
		refs := []string{evidencePrefix, evidencePrefix + "-baseline-partial", baselineRef}
		for _, path := range baseline.OmittedPaths {
			refs = append(refs, evidencePrefix+"-baseline-omitted-path:"+strings.TrimSpace(path))
		}
		return compactStringsV0(refs), false
	}
	projectWorkDir, projectWorkDirErr := stack.autoprogrammingGoalProjectWorkDirV0(ctx, state)
	if projectWorkDirErr != nil {
		return []string{evidencePrefix, evidencePrefix + "-project-work-dir-missing", baselineRef}, false
	}
	result, issues := orquestaruntimeworktree.VerifyWorktreeWriteSetV0(ctx, orquestaruntimeworktree.WorktreeVerifyRequestV0{
		Baseline:                  baseline,
		ProjectWorkDir:            projectWorkDir,
		WriteSet:                  request.WriteSet,
		DestructiveAuthorizations: autoprogrammingWorktreeDestructiveAuthorizationsV0(state.Spec.DestructiveAuthorizations),
		IgnorePrefixes:            codexStackWorktreeIgnorePrefixesV0(),
		AllowPartialSnapshot:      false,
	})
	refs := autoprogrammingPromotionGoalFirstWriteSetEvidenceRefsV0(evidencePrefix, baselineRef, result, issues)
	return refs, len(issues) == 0 && result.OK
}

func autoprogrammingPromotionGoalFirstWriteSetEvidenceRefsV0(
	prefix string,
	baselineRef string,
	result orquestaruntimeworktree.WorktreeVerifyResultV0,
	issues []orquestaruntimeworktree.WorktreeIssueV0,
) []string {
	refs := []string{prefix, baselineRef}
	for _, path := range result.ChangedPaths {
		refs = append(refs, prefix+"-changed-path:"+strings.TrimSpace(path))
	}
	for _, path := range result.OutsideWriteSet {
		refs = append(refs, prefix+"-outside-write-set:"+strings.TrimSpace(path))
	}
	for _, issue := range issues {
		refs = append(refs, prefix+"-issue:"+string(issue.Code))
		for _, evidence := range issue.Evidence {
			refs = append(refs, prefix+"-issue-evidence:"+strings.TrimSpace(evidence))
		}
	}
	return compactStringsV0(refs)
}

func (stack StackV0) autoprogrammingPromotionRequestV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestaautoprogramming.AutoprogrammingStagingPromotionRequestV0, bool, error) {
	tasks, err := stack.autoprogrammingPromotionTasksV0(ctx, run)
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingStagingPromotionRequestV0{}, false, err
	}
	if len(tasks) == 0 {
		return stack.autoprogrammingPromotionGoalStateRequestV0(ctx, run)
	}
	evidence, err := stack.autoprogrammingPromotionRequiredTestEvidenceV0(ctx, run)
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingStagingPromotionRequestV0{}, false, err
	}
	return orquestaautoprogramming.AutoprogrammingStagingPromotionRequestV0{
		RequestRef:           run.AppSpecRef,
		RunRef:               run.RunID,
		ProjectRef:           run.ProjectRef,
		WorktreeRef:          autoprogrammingPromotionTaskUniqueContextRefV0(tasks, "worktree_ref:"),
		BranchRef:            autoprogrammingPromotionTaskUniqueContextRefV0(tasks, "branch_ref:"),
		RunClosed:            run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0,
		WriteSet:             autoprogrammingPromotionTaskWriteSetV0(tasks),
		RequiredTests:        autoprogrammingPromotionTaskRequiredTestsV0(tasks),
		ClosedTaskRefs:       append([]string(nil), run.ClosedTasks...),
		AcceptedReviewRefs:   append([]string(nil), run.AcceptedReviews...),
		RequiredTestEvidence: evidence,
		LiveWorks:            stack.autoprogrammingPromotionLiveWorksV0(ctx, run.RunID),
		EvidenceRefs:         []string{"evidence-ref-codex-stack-autoprogramming-promotion"},
	}, true, nil
}

func (stack StackV0) autoprogrammingPromotionTasksV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) ([]orquestacoreworkflow.WorkflowTaskV0, error) {
	if len(compactStringsV0(run.Tasks)) == 0 {
		return nil, nil
	}
	if stack.Stores.TaskStore == nil {
		return nil, fmt.Errorf("autoprogramming_promotion_task_store_required")
	}
	tasks, err := stack.Stores.TaskStore.LoadWorkflowTasksV0(ctx, run.RunID, run.Tasks)
	if err != nil {
		return nil, err
	}
	out := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(tasks))
	for _, task := range tasks {
		if autoprogrammingPromotionTaskV0(task) {
			out = append(out, task)
		}
	}
	return out, nil
}

func autoprogrammingPromotionTaskV0(task orquestacoreworkflow.WorkflowTaskV0) bool {
	for _, ref := range task.ContextRefs {
		if strings.TrimSpace(ref) == autoprogrammingBridgeOperationalTaskSourceRefV0 {
			return true
		}
	}
	return false
}

func (stack StackV0) autoprogrammingPromotionGoalStateRequestV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestaautoprogramming.AutoprogrammingStagingPromotionRequestV0, bool, error) {
	state, ok, err := stack.autoprogrammingPromotionGoalStateV0(ctx, run)
	if err != nil || !ok {
		return orquestaautoprogramming.AutoprogrammingStagingPromotionRequestV0{}, false, err
	}
	return orquestaautoprogramming.AutoprogrammingStagingPromotionRequestV0{
		RequestRef:           firstNonEmptyQueuedSourceV0(state.Spec.RequestRef, run.AppSpecRef),
		RunRef:               run.RunID,
		GoalRef:              state.GoalRef,
		ProjectRef:           firstNonEmptyQueuedSourceV0(state.Spec.ProjectRef, run.ProjectRef),
		WorktreeRef:          autoprogrammingPromotionGoalContextRefV0(state.Spec.ContextRefs, "worktree", "worktree_ref:"),
		BranchRef:            autoprogrammingPromotionGoalContextRefV0(state.Spec.ContextRefs, "branch", "branch_ref:"),
		RunClosed:            run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0,
		WriteSet:             autoprogrammingPromotionGoalWriteSetV0(state.Spec.WriteSet),
		RequiredTests:        autoprogrammingPromotionGoalRequiredTestsV0(state.Spec.RequiredTests),
		ClosedTaskRefs:       autoprogrammingPromotionGoalClosedRefsV0(state),
		AcceptedReviewRefs:   autoprogrammingPromotionGoalAcceptedReviewRefsV0(state),
		RequiredTestEvidence: autoprogrammingPromotionGoalRequiredTestEvidenceV0(state),
		LiveWorks:            stack.autoprogrammingPromotionLiveWorksV0(ctx, run.RunID),
		EvidenceRefs: compactStringsV0(append(
			[]string{"evidence-ref-codex-stack-autoprogramming-goal-first-promotion"},
			state.EvidenceRefs...,
		)),
	}, true, nil
}

func (stack StackV0) autoprogrammingPromotionGoalStateV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestagoal.GoalWorkStateV0, bool, error) {
	store := stack.Ports.GoalStateStore
	if store == nil {
		store = stack.Stores.AppGoalStateStore
	}
	if store == nil {
		return orquestagoal.GoalWorkStateV0{}, false, nil
	}
	state, err := store.LoadGoalWorkStateV0(ctx, run.RunID)
	if err != nil {
		return orquestagoal.GoalWorkStateV0{}, false, nil
	}
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return orquestagoal.GoalWorkStateV0{}, false, err
	}
	if strings.TrimSpace(state.Spec.WorkKind) != orquestaautoprogramming.AutoprogrammingGoalWorkKindV0 {
		return orquestagoal.GoalWorkStateV0{}, false, nil
	}
	return state, true, nil
}

func (stack StackV0) autoprogrammingPromotionMissingGoalStateEvidenceRefsV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) ([]string, bool) {
	if strings.TrimSpace(run.RunID) == "" {
		return nil, false
	}
	if !autoprogrammingPromotionGoalFirstContainerWithoutStateV0(run) {
		return nil, false
	}
	refs := []string{
		"evidence-ref-codex-stack-autoprogramming-goal-first-state-missing",
		"evidence-ref-codex-stack-autoprogramming-goal-first-container",
		run.AppSpecRef,
	}
	if marker, ok := stack.goalFirstRunMarkerWithoutStateV0(ctx, run.RunID); ok {
		return compactStringsV0(append(
			append(refs, "evidence-ref-codex-stack-autoprogramming-goal-first-marker"),
			marker.EvidenceRefs...,
		)), true
	}
	return compactStringsV0(refs), true
}

func autoprogrammingPromotionGoalFirstContainerWithoutStateV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	return strings.HasPrefix(strings.TrimSpace(run.AppSpecRef), "app-spec-ref-autoprogramming-") &&
		len(compactStringsV0(run.Tasks)) == 0 &&
		len(compactStringsV0(run.FunctionContracts)) == 0
}

func autoprogrammingPromotionGoalContextRefV0(
	refs []orquestagoal.GoalContextRefV0,
	kind string,
	legacyPrefix string,
) string {
	for _, ref := range refs {
		if strings.TrimSpace(ref.Kind) == kind && strings.TrimSpace(ref.Ref) != "" {
			return strings.TrimSpace(ref.Ref)
		}
	}
	legacyPrefix = strings.TrimSpace(legacyPrefix)
	if legacyPrefix != "" {
		for _, ref := range refs {
			if value, ok := strings.CutPrefix(strings.TrimSpace(ref.Ref), legacyPrefix); ok {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func autoprogrammingPromotionGoalWriteSetV0(scopes []orquestagoal.GoalWriteScopeV0) []string {
	out := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		out = append(out, scope.Path)
	}
	return compactStringsV0(out)
}

func autoprogrammingPromotionGoalRequiredTestsV0(tests []orquestagoal.GoalRequiredTestV0) []string {
	out := make([]string, 0, len(tests))
	for _, test := range tests {
		out = append(out, autoprogrammingPromotionGoalRequiredTestCommandV0(test))
	}
	return compactStringsV0(out)
}

func autoprogrammingPromotionGoalRequiredTestCommandV0(test orquestagoal.GoalRequiredTestV0) string {
	for _, value := range []string{test.Command, test.CommandRef, test.TestRef} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func autoprogrammingPromotionGoalClosedRefsV0(state orquestagoal.GoalWorkStateV0) []string {
	if state.LastClosure == nil {
		return nil
	}
	return compactStringsV0([]string{state.GoalRef})
}

func autoprogrammingPromotionGoalAcceptedReviewRefsV0(state orquestagoal.GoalWorkStateV0) []string {
	if state.LastClosure == nil || !state.LastClosure.Accepted {
		return nil
	}
	return compactStringsV0(append(
		[]string{"goal-closure-ref-" + state.GoalRef},
		state.LastClosure.EvidenceRefs...,
	))
}

func autoprogrammingPromotionGoalRequiredTestEvidenceV0(
	state orquestagoal.GoalWorkStateV0,
) []orquestaautoprogramming.AutoprogrammingRequiredTestEvidenceV0 {
	if state.LastClosure == nil || !state.LastClosure.Accepted {
		return nil
	}
	commandsByRef := make(map[string]string, len(state.Spec.RequiredTests))
	for _, test := range state.Spec.RequiredTests {
		testRef := strings.TrimSpace(test.TestRef)
		if testRef != "" {
			commandsByRef[testRef] = autoprogrammingPromotionGoalRequiredTestCommandV0(test)
		}
	}
	var out []orquestaautoprogramming.AutoprogrammingRequiredTestEvidenceV0
	seen := make(map[string]struct{}, len(state.LastClosure.AttestationVerifications))
	for _, verification := range state.LastClosure.AttestationVerifications {
		testRef := strings.TrimSpace(verification.TestRef)
		command, required := commandsByRef[testRef]
		evidenceRef := strings.TrimSpace(verification.AttestationRef)
		if !required || !verification.Verified || !verification.Independent || evidenceRef == "" {
			continue
		}
		key := testRef + "\x00" + evidenceRef
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, orquestaautoprogramming.AutoprogrammingRequiredTestEvidenceV0{
			EvidenceRef: evidenceRef,
			TaskRef:     state.GoalRef,
			TestCommand: command,
			Status:      orquestagoal.GoalRequiredTestAttestationStatusPassedV0,
		})
	}
	if state.Spec.ClosurePolicy.RequireIndependentRequiredTestAttestation || state.LastResult == nil {
		return out
	}
	// Compatibilidad con contratos que no exigen atestacion independiente:
	// solo esos contratos pueden usar evidencia declarada en el resultado.
	for _, result := range state.LastResult.RequiredTestResults {
		testRef := strings.TrimSpace(result.TestRef)
		command, required := commandsByRef[testRef]
		if !required {
			continue
		}
		for _, evidenceRef := range result.EvidenceRefs {
			evidenceRef = strings.TrimSpace(evidenceRef)
			if evidenceRef == "" {
				continue
			}
			key := testRef + "\x00" + evidenceRef
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, orquestaautoprogramming.AutoprogrammingRequiredTestEvidenceV0{
				EvidenceRef: evidenceRef,
				TaskRef:     state.GoalRef,
				TestCommand: command,
				Status:      result.Status,
			})
		}
	}
	return out
}

func (stack StackV0) autoprogrammingPromotionRequiredTestEvidenceV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) ([]orquestaautoprogramming.AutoprogrammingRequiredTestEvidenceV0, error) {
	if stack.Stores.OperationalPlanStateStore == nil || stack.Stores.RequiredTestEvidenceStore == nil {
		return nil, nil
	}
	state, err := stack.Stores.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(
		ctx,
		run.RunID,
		autoprogrammingPromotionPlanRefV0(run.RunID),
	)
	if err != nil {
		return nil, nil
	}
	refs := autoprogrammingPromotionPlanTestEvidenceRefsV0(state)
	items, err := stack.Stores.RequiredTestEvidenceStore.LoadRequiredTestEvidenceV0(ctx, run.RunID, refs)
	if err != nil {
		return nil, err
	}
	out := make([]orquestaautoprogramming.AutoprogrammingRequiredTestEvidenceV0, 0, len(items))
	for _, item := range items {
		out = append(out, orquestaautoprogramming.AutoprogrammingRequiredTestEvidenceV0{
			EvidenceRef: item.EvidenceRef,
			TaskRef:     item.TaskRef,
			TestCommand: item.TestCommand,
			Status:      string(item.Status),
		})
	}
	return out, nil
}

func autoprogrammingPromotionPlanTestEvidenceRefsV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) []string {
	var refs []string
	for _, step := range state.Steps {
		refs = append(refs, step.RequiredTestEvidenceRefs...)
	}
	return compactStringsV0(refs)
}

func (stack StackV0) autoprogrammingPromotionLiveWorksV0(
	ctx context.Context,
	currentRunRef string,
) []orquestaautoprogramming.AutoprogrammingLiveWorkV0 {
	if stack.Stores.RunQueue == nil || stack.Stores.RunStore == nil {
		return nil
	}
	candidates, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{QueueRef: normalizeRunQueueConfigV0(stack.RunQueue).QueueRef},
	)
	if err != nil {
		return nil
	}
	out := make([]orquestaautoprogramming.AutoprogrammingLiveWorkV0, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.RunRef) == strings.TrimSpace(currentRunRef) {
			continue
		}
		live, ok := stack.autoprogrammingPromotionLiveWorkV0(ctx, candidate)
		if ok {
			out = append(out, live)
		}
	}
	return out
}

func (stack StackV0) autoprogrammingPromotionLiveWorkV0(
	ctx context.Context,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) (orquestaautoprogramming.AutoprogrammingLiveWorkV0, bool) {
	run, err := stack.Stores.RunStore.LoadRunV0(ctx, candidate.RunRef)
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingLiveWorkV0{}, false
	}
	var writeSet []string
	if len(compactStringsV0(run.Tasks)) > 0 && stack.Stores.TaskStore != nil {
		tasks, err := stack.Stores.TaskStore.LoadWorkflowTasksV0(ctx, run.RunID, run.Tasks)
		if err != nil {
			return orquestaautoprogramming.AutoprogrammingLiveWorkV0{}, false
		}
		writeSet = autoprogrammingPromotionTaskWriteSetV0(tasks)
	}
	if len(writeSet) == 0 {
		if state, ok, err := stack.autoprogrammingPromotionGoalStateV0(ctx, run); err == nil && ok {
			writeSet = autoprogrammingPromotionGoalWriteSetV0(state.Spec.WriteSet)
		}
	}
	if len(writeSet) == 0 {
		return orquestaautoprogramming.AutoprogrammingLiveWorkV0{}, false
	}
	return orquestaautoprogramming.AutoprogrammingLiveWorkV0{
		WorkRef:  candidate.RunRef,
		Status:   candidate.Status,
		WriteSet: writeSet,
	}, true
}

func autoprogrammingPromotionTaskContextRefV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	prefix string,
) string {
	for _, task := range tasks {
		for _, ref := range task.ContextRefs {
			if value, ok := strings.CutPrefix(strings.TrimSpace(ref), prefix); ok {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func autoprogrammingPromotionTaskUniqueContextRefV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	prefix string,
) string {
	seen := ""
	for _, task := range tasks {
		value := autoprogrammingPromotionTaskContextRefV0([]orquestacoreworkflow.WorkflowTaskV0{task}, prefix)
		if value == "" {
			continue
		}
		if seen != "" && seen != value {
			return ""
		}
		seen = value
	}
	return seen
}

func autoprogrammingPromotionTaskWriteSetV0(tasks []orquestacoreworkflow.WorkflowTaskV0) []string {
	var out []string
	for _, task := range tasks {
		out = append(out, task.WriteSet...)
	}
	return compactStringsV0(out)
}

func autoprogrammingPromotionTaskRequiredTestsV0(tasks []orquestacoreworkflow.WorkflowTaskV0) []string {
	var out []string
	for _, task := range tasks {
		out = append(out, task.RequiredTests...)
	}
	return compactStringsV0(out)
}

func autoprogrammingPromotionPlanRefV0(runRef string) string {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" {
		return ""
	}
	safe := strings.NewReplacer("\\", "-", "/", "-", " ", "-").Replace(runRef)
	return "operational-director-plan-director-decisions-" + safe
}

func autoprogrammingPromotionEffectCompleteV0(result orquestaautoprogramming.AutoprogrammingStagingEffectResultV0) bool {
	if strings.TrimSpace(result.IntegrationStatus) != orquestaautoprogramming.AutoprogrammingStagingIntegrationStatusIntegratedV0 ||
		strings.TrimSpace(result.IntegrationReceiptRef) == "" {
		return false
	}
	switch strings.TrimSpace(result.Status) {
	case orquestaautoprogramming.AutoprogrammingStagingEffectPromotedV0,
		orquestaautoprogramming.AutoprogrammingStagingEffectCleanV0:
		return true
	default:
		return false
	}
}

func autoprogrammingPromotionEffectWithIntegrationStatusV0(
	result orquestaautoprogramming.AutoprogrammingStagingEffectResultV0,
) orquestaautoprogramming.AutoprogrammingStagingEffectResultV0 {
	result.IntegrationReceiptRef = strings.TrimSpace(result.IntegrationReceiptRef)
	result.IntegrationStatus = strings.TrimSpace(result.IntegrationStatus)
	if result.IntegrationStatus == "" {
		switch strings.TrimSpace(result.Status) {
		case orquestaautoprogramming.AutoprogrammingStagingEffectPromotedV0,
			orquestaautoprogramming.AutoprogrammingStagingEffectCleanV0:
			result.IntegrationStatus = orquestaautoprogramming.AutoprogrammingStagingIntegrationStatusPendingIntegrationV0
		case orquestaautoprogramming.AutoprogrammingStagingEffectPendingPushV0:
			result.IntegrationStatus = orquestaautoprogramming.AutoprogrammingStagingIntegrationStatusPendingIntegrationV0
		case orquestaautoprogramming.AutoprogrammingStagingEffectBlockedV0:
			result.IntegrationStatus = orquestaautoprogramming.AutoprogrammingStagingIntegrationStatusBlockedIntegrationV0
		}
	}
	if result.IntegrationReceiptRef == "" &&
		result.IntegrationStatus == orquestaautoprogramming.AutoprogrammingStagingIntegrationStatusIntegratedV0 {
		result.IntegrationReceiptRef = "integration-receipt-ref-" + codexStackOperationalClosureSafeRefV0(
			firstNonEmptyQueuedSourceV0(result.RunRef, result.PromotionRef),
		) + "-" + codexStackOperationalClosureSafeRefV0(
			firstNonEmptyQueuedSourceV0(result.CommitShortRef, result.CommitRef, result.Status),
		)
	}
	return result
}

func autoprogrammingPromotionEffectEvidenceRefsV0(
	result orquestaautoprogramming.AutoprogrammingStagingEffectResultV0,
) []string {
	refs := append([]string{}, result.EvidenceRefs...)
	switch strings.TrimSpace(result.IntegrationStatus) {
	case orquestaautoprogramming.AutoprogrammingStagingIntegrationStatusIntegratedV0:
		refs = append(refs, "evidence-ref-autoprogramming-integration-receipt")
		if result.IntegrationReceiptRef != "" {
			refs = append(refs, result.IntegrationReceiptRef)
		}
	case orquestaautoprogramming.AutoprogrammingStagingIntegrationStatusPendingIntegrationV0:
		refs = append(refs, "evidence-ref-autoprogramming-pending-integration")
	case orquestaautoprogramming.AutoprogrammingStagingIntegrationStatusBlockedPushV0:
		refs = append(refs, "evidence-ref-autoprogramming-blocked-push")
	case orquestaautoprogramming.AutoprogrammingStagingIntegrationStatusBlockedIntegrationV0:
		refs = append(refs, "evidence-ref-autoprogramming-blocked-integration")
	}
	return compactStringsV0(refs)
}
