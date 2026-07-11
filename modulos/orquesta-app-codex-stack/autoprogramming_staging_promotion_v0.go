package orquestaappcodexstack

import (
	"context"
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
	Enabled                  bool
	Port                     orquestaautoprogramming.AutoprogrammingStagingPromotionPortV0
	GoalFirstSnapshotStore   orquestaruntimeworktree.WorktreeSnapshotStorePortV0
	GoalWorkspaceProvisioner orquestaruntimeworktree.GoalWorkspaceProvisionerPortV0
	GoalWorkspaceRoot        string
	AppRef                   string
	RepoRef                  string
	CommitMessage            string
}

func (stack StackV0) maybePromoteClosedAutoprogrammingRunV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, []string, error) {
	config := stack.AutoprogrammingPromotion
	if !config.Enabled || config.Port == nil ||
		run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
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
	if state, goalFirst, err := stack.autoprogrammingPromotionGoalStateV0(ctx, run); err != nil {
		return false, nil, err
	} else if goalFirst {
		if refs, verified := stack.autoprogrammingPromotionGoalFirstWriteSetVerifiedV0(ctx, state, request); !verified {
			return false, refs, nil
		}
	}
	decision := orquestaautoprogramming.EvaluateAutoprogrammingStagingPromotionV0(request)
	if !decision.Ready {
		return false, decision.EvidenceRefs, nil
	}
	promoted, err := config.Port.PromoteAutoprogrammingStagingV0(ctx, decision.PromotionCommand)
	if err != nil {
		return false, promoted.EvidenceRefs, err
	}
	promoted = autoprogrammingPromotionEffectWithIntegrationStatusV0(promoted)
	refs := compactStringsV0(append(decision.EvidenceRefs, autoprogrammingPromotionEffectEvidenceRefsV0(promoted)...))
	if !autoprogrammingPromotionEffectCompleteV0(promoted.Status) {
		return false, refs, nil
	}
	archived, err := config.Port.ArchiveAutoprogrammingStagingV0(ctx, decision.CleanupCommand)
	if err != nil {
		return false, compactStringsV0(append(refs, archived.EvidenceRefs...)), err
	}
	refs = compactStringsV0(append(refs, archived.EvidenceRefs...))
	return archived.Status == orquestaautoprogramming.AutoprogrammingStagingEffectArchivedV0, refs, nil
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
	projectWorkDir := strings.TrimSpace(stack.Codex.ProjectWorkDir)
	if projectWorkDir == "" {
		return []string{evidencePrefix, evidencePrefix + "-project-work-dir-missing", baselineRef}, false
	}
	result, issues := orquestaruntimeworktree.VerifyWorktreeWriteSetV0(ctx, orquestaruntimeworktree.WorktreeVerifyRequestV0{
		Baseline:             baseline,
		ProjectWorkDir:       projectWorkDir,
		WriteSet:             request.WriteSet,
		IgnorePrefixes:       codexStackWorktreeIgnorePrefixesV0(),
		AllowPartialSnapshot: false,
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
	for _, ref := range refs {
		if value, ok := strings.CutPrefix(strings.TrimSpace(ref.Ref), legacyPrefix); ok {
			return strings.TrimSpace(value)
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
	if state.LastResult == nil {
		return nil
	}
	commandsByRef := make(map[string]string, len(state.Spec.RequiredTests))
	for _, test := range state.Spec.RequiredTests {
		commandsByRef[strings.TrimSpace(test.TestRef)] = autoprogrammingPromotionGoalRequiredTestCommandV0(test)
	}
	var out []orquestaautoprogramming.AutoprogrammingRequiredTestEvidenceV0
	for _, result := range state.LastResult.RequiredTestResults {
		command := strings.TrimSpace(commandsByRef[strings.TrimSpace(result.TestRef)])
		if command == "" {
			command = strings.TrimSpace(result.TestRef)
		}
		for _, evidenceRef := range result.EvidenceRefs {
			if strings.TrimSpace(evidenceRef) == "" {
				continue
			}
			out = append(out, orquestaautoprogramming.AutoprogrammingRequiredTestEvidenceV0{
				EvidenceRef: strings.TrimSpace(evidenceRef),
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

func autoprogrammingPromotionEffectCompleteV0(status string) bool {
	switch strings.TrimSpace(status) {
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
			result.IntegrationStatus = orquestaautoprogramming.AutoprogrammingStagingIntegrationStatusIntegratedV0
		case orquestaautoprogramming.AutoprogrammingStagingEffectPendingPushV0:
			result.IntegrationStatus = orquestaautoprogramming.AutoprogrammingStagingIntegrationStatusPendingIntegrationV0
		case orquestaautoprogramming.AutoprogrammingStagingEffectBlockedV0:
			result.IntegrationStatus = orquestaautoprogramming.AutoprogrammingStagingIntegrationStatusBlockedPushV0
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
	}
	return compactStringsV0(refs)
}
