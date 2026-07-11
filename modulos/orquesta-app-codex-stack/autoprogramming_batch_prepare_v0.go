package orquestaappcodexstack

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

const (
	autoprogrammingBatchContextKindV0     = "autoprogramming_batch"
	autoprogrammingBatchTaskContextKindV0 = "autoprogramming_batch_task"
)

type autoprogrammingBatchGoalLauncherV0 struct {
	Delegate orquestagoal.GoalWorkLauncherPortV0
	Store    orquestaautoprogramming.AutoprogrammingBatchStorePortV0
	BatchRef string
}

func (stack StackV0) prepareAutoprogrammingBatchV0(
	ctx context.Context,
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
) (orquestaautoprogramming.AutoprogrammingProgrammableWorkV0, orquestaautoprogramming.AutoprogrammingBatchV0, []orquestaautoprogramming.AutoprogrammingRequestIssueV0) {
	store := stack.Stores.AutoprogrammingBatchStore
	if store == nil {
		return work, orquestaautoprogramming.AutoprogrammingBatchV0{}, []orquestaautoprogramming.AutoprogrammingRequestIssueV0{{
			Code: "autoprogramming_batch_store_required", Field: "autoprogramming_batch_store",
			Message: "batch multi-goal requiere persistencia durable antes del primer launcher",
		}}
	}
	members := make([]orquestaautoprogramming.AutoprogrammingBatchMemberV0, 0, len(work.GoalSpecs))
	baseRevision := ""
	for index, spec := range work.GoalSpecs {
		spec = autoprogrammingBridgeGoalSpecForRunRefV0(work, spec, autoprogrammingBridgeGoalRunRefForSpecV0(work, index))
		workspace, issues := stack.resolveAutoprogrammingBatchWorkspaceV0(ctx, work, spec)
		if len(issues) > 0 {
			return work, orquestaautoprogramming.AutoprogrammingBatchV0{}, issues
		}
		if baseRevision == "" {
			baseRevision = strings.TrimSpace(workspace.BaseRevision)
		} else if baseRevision != strings.TrimSpace(workspace.BaseRevision) {
			return work, orquestaautoprogramming.AutoprogrammingBatchV0{}, []orquestaautoprogramming.AutoprogrammingRequestIssueV0{{
				Code: "autoprogramming_batch_base_revision_mismatch", Field: "base_revision",
				Message: "todos los workspaces del batch deben partir de la misma revision",
			}}
		}
		members = append(members, orquestaautoprogramming.AutoprogrammingBatchMemberV0{
			TaskRef:      strings.TrimPrefix(strings.TrimSpace(spec.GoalRef), "goal-ref-"),
			GoalRef:      strings.TrimSpace(spec.GoalRef),
			RunRef:       strings.TrimSpace(spec.RunRef),
			WorkspaceRef: strings.TrimSpace(workspace.WorkspaceID),
			WriteSet:     autoprogrammingPromotionGoalWriteSetV0(spec.WriteSet),
		})
	}
	tests := make([]orquestaautoprogramming.AutoprogrammingBatchTestV0, 0, len(work.BatchRequiredTests))
	for _, command := range compactStringsV0(work.BatchRequiredTests) {
		sum := sha256.Sum256([]byte(command))
		tests = append(tests, orquestaautoprogramming.AutoprogrammingBatchTestV0{Command: command, SHA256: hex.EncodeToString(sum[:])})
	}
	identity := []string{strings.TrimSpace(work.RequestRef), strings.TrimSpace(work.ProjectRef), baseRevision}
	identityMembers := append([]orquestaautoprogramming.AutoprogrammingBatchMemberV0(nil), members...)
	sort.SliceStable(identityMembers, func(left, right int) bool { return identityMembers[left].TaskRef < identityMembers[right].TaskRef })
	for _, member := range identityMembers {
		identity = append(identity, member.TaskRef, member.GoalRef, member.RunRef, member.WorkspaceRef, strings.Join(member.WriteSet, "\x00"))
	}
	identityTests := append([]orquestaautoprogramming.AutoprogrammingBatchTestV0(nil), tests...)
	sort.SliceStable(identityTests, func(left, right int) bool {
		return identityTests[left].Command+identityTests[left].SHA256 < identityTests[right].Command+identityTests[right].SHA256
	})
	for _, test := range identityTests {
		identity = append(identity, test.Command, test.SHA256)
	}
	created := orquestaautoprogramming.NewAutoprogrammingBatchV0(orquestaautoprogramming.AutoprogrammingBatchPlanV0{
		BatchRef:     codexStackDeterministicRefV0("batch-ref-autoprogramming-", identity...),
		RequestRef:   strings.TrimSpace(work.RequestRef),
		ProjectRef:   strings.TrimSpace(work.ProjectRef),
		BaseRevision: baseRevision,
		Members:      members,
		FrozenTests:  tests,
	})
	if !created.Accepted {
		return work, created.Batch, created.Issues
	}
	batch, err := persistAutoprogrammingBatchPlanV0(ctx, store, created.Batch)
	if err != nil {
		return work, created.Batch, []orquestaautoprogramming.AutoprogrammingRequestIssueV0{{
			Code: "autoprogramming_batch_plan_persist_failed", Field: "autoprogramming_batch_store", Message: err.Error(),
		}}
	}
	return autoprogrammingBridgeWorkWithBatchRefsV0(work, batch), batch, nil
}

func (stack StackV0) resolveAutoprogrammingBatchWorkspaceV0(
	ctx context.Context,
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestaruntimeworktree.GoalWorkspaceV0, []orquestaautoprogramming.AutoprogrammingRequestIssueV0) {
	provisioner := stack.AutoprogrammingPromotion.GoalWorkspaceProvisioner
	workspaceRef := autoprogrammingPromotionGoalContextRefV0(spec.ContextRefs, "goal_workspace", "")
	if provisioner == nil || workspaceRef == "" {
		return orquestaruntimeworktree.GoalWorkspaceV0{}, []orquestaautoprogramming.AutoprogrammingRequestIssueV0{{
			Code: "autoprogramming_batch_workspace_required", Field: "goal_workspace", Message: "workspace fisico persistido requerido para miembro batch",
		}}
	}
	workspace, issues := provisioner.ResolveGoalWorkspaceV0(ctx, orquestaruntimeworktree.GoalWorkspaceRequestV0{
		RunRef: strings.TrimSpace(work.RequestRef), GoalRef: strings.TrimSpace(spec.GoalRef), ProjectRef: strings.TrimSpace(work.ProjectRef),
		WorktreeRef: strings.TrimSpace(work.WorktreeRef), SourceWorkDir: strings.TrimSpace(stack.Codex.ProjectWorkDir),
		WorkspaceRoot: strings.TrimSpace(stack.AutoprogrammingPromotion.GoalWorkspaceRoot),
	})
	if len(issues) > 0 || strings.TrimSpace(workspace.WorkspaceID) != workspaceRef || strings.TrimSpace(workspace.BaseRevision) == "" {
		return orquestaruntimeworktree.GoalWorkspaceV0{}, []orquestaautoprogramming.AutoprogrammingRequestIssueV0{{
			Code: "autoprogramming_batch_workspace_unavailable", Field: "goal_workspace", Message: "workspace batch no resoluble o sin revision base",
		}}
	}
	return workspace, nil
}

func persistAutoprogrammingBatchPlanV0(
	ctx context.Context,
	store orquestaautoprogramming.AutoprogrammingBatchStorePortV0,
	want orquestaautoprogramming.AutoprogrammingBatchV0,
) (orquestaautoprogramming.AutoprogrammingBatchV0, error) {
	if existing, err := store.LoadAutoprogrammingBatchV0(ctx, want.BatchRef); err == nil {
		if existing.PlanHash != want.PlanHash {
			return orquestaautoprogramming.AutoprogrammingBatchV0{}, fmt.Errorf("autoprogramming_batch_plan_conflict")
		}
		return existing, nil
	}
	created, err := store.CompareAndSwapAutoprogrammingBatchV0(ctx, 0, want)
	if err == nil {
		return created, nil
	}
	existing, loadErr := store.LoadAutoprogrammingBatchV0(ctx, want.BatchRef)
	if loadErr != nil || existing.PlanHash != want.PlanHash {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, err
	}
	return existing, nil
}

func (launcher autoprogrammingBatchGoalLauncherV0) LaunchGoalWorkV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	taskRef := autoprogrammingBatchGoalContextRefV0(spec.ContextRefs, autoprogrammingBatchTaskContextKindV0)
	if launcher.Store == nil || launcher.Delegate == nil || taskRef == "" {
		return orquestagoal.GoalLaunchReceiptV0{}, fmt.Errorf("autoprogramming_batch_prelaunch_unavailable")
	}
	receipt, err := launcher.Delegate.LaunchGoalWorkV0(ctx, spec)
	if err != nil || len(orquestagoal.ValidateGoalLaunchReceiptV0(receipt)) > 0 {
		_, blockErr := blockStoredAutoprogrammingBatchV0(ctx, launcher.Store, launcher.BatchRef, "batch-block-ref-launch-"+codexStackOperationalClosureSafeRefV0(taskRef))
		if blockErr != nil {
			return receipt, fmt.Errorf("autoprogramming_batch_launch_block_failed: %v", blockErr)
		}
		if err == nil {
			err = fmt.Errorf("autoprogramming_batch_launch_receipt_invalid")
		}
		return receipt, err
	}
	_, err = transitionStoredAutoprogrammingBatchV0(ctx, launcher.Store, launcher.BatchRef, func(batch orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
		return orquestaautoprogramming.RegisterAutoprogrammingBatchLaunchV0(batch, batch.StoreVersion, launcher.BatchRef+":launch:"+taskRef, taskRef)
	})
	if err != nil {
		_, _ = blockStoredAutoprogrammingBatchV0(ctx, launcher.Store, launcher.BatchRef, "batch-block-ref-launch-receipt-store-"+codexStackOperationalClosureSafeRefV0(taskRef))
	}
	return receipt, err
}

func blockStoredAutoprogrammingBatchV0(
	ctx context.Context,
	store orquestaautoprogramming.AutoprogrammingBatchStorePortV0,
	batchRef string,
	blockRef string,
) (orquestaautoprogramming.AutoprogrammingBatchV0, error) {
	return transitionStoredAutoprogrammingBatchV0(ctx, store, batchRef, func(batch orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
		return orquestaautoprogramming.BlockAutoprogrammingBatchV0(batch, batch.StoreVersion, batchRef+":block:"+blockRef, blockRef)
	})
}

func transitionStoredAutoprogrammingBatchV0(
	ctx context.Context,
	store orquestaautoprogramming.AutoprogrammingBatchStorePortV0,
	batchRef string,
	transition func(orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0,
) (orquestaautoprogramming.AutoprogrammingBatchV0, error) {
	for attempt := 0; attempt < 4; attempt++ {
		current, err := store.LoadAutoprogrammingBatchV0(ctx, batchRef)
		if err != nil {
			return orquestaautoprogramming.AutoprogrammingBatchV0{}, err
		}
		result := transition(current)
		if !result.Accepted {
			code := "autoprogramming_batch_transition_rejected"
			if len(result.Issues) > 0 {
				code = result.Issues[0].Code
			}
			return current, fmt.Errorf("%s", code)
		}
		if result.Replay {
			return result.Batch, nil
		}
		saved, err := store.CompareAndSwapAutoprogrammingBatchV0(ctx, current.StoreVersion, result.Batch)
		if err == nil {
			return saved, nil
		}
	}
	return orquestaautoprogramming.AutoprogrammingBatchV0{}, fmt.Errorf("autoprogramming_batch_cas_exhausted")
}
