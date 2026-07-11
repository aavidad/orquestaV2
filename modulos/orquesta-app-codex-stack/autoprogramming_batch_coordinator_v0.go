package orquestaappcodexstack

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

type AutoprogrammingBatchTestRunRequestV0 struct {
	BatchRef         string
	GateGeneration   uint64
	Revision         string
	Test             orquestaautoprogramming.AutoprogrammingBatchTestV0
	TestHash         string
	ClaimRef         string
	CanonicalWorkDir string
}

type AutoprogrammingBatchTestRunResultV0 struct {
	ReceiptRef   string
	Status       string
	EvidenceRefs []string
}

type AutoprogrammingBatchTestRunnerPortV0 interface {
	RunAutoprogrammingBatchTestV0(context.Context, AutoprogrammingBatchTestRunRequestV0) (AutoprogrammingBatchTestRunResultV0, error)
}

type AutoprogrammingBatchTestClaimReconcilerPortV0 interface {
	ReconcileAutoprogrammingBatchTestClaimV0(context.Context, AutoprogrammingBatchTestRunRequestV0) (AutoprogrammingBatchTestRunResultV0, bool, error)
}

type AutoprogrammingBatchPromotionRequestV0 struct {
	BatchRef           string   `json:"batch_ref"`
	GateGeneration     uint64   `json:"gate_generation"`
	ClaimRef           string   `json:"claim_ref"`
	IntegratedRevision string   `json:"integrated_revision"`
	CanonicalWorkDir   string   `json:"canonical_work_dir"`
	ReceiptDir         string   `json:"receipt_dir"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
}

type AutoprogrammingBatchPromotionResultV0 struct {
	BatchRef           string   `json:"batch_ref"`
	GateGeneration     uint64   `json:"gate_generation"`
	ClaimRef           string   `json:"claim_ref"`
	IntegratedRevision string   `json:"integrated_revision"`
	CanonicalClean     bool     `json:"canonical_clean"`
	ReceiptRef         string   `json:"receipt_ref"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
}

type AutoprogrammingBatchPromotionFinalizerPortV0 interface {
	FinalizeAutoprogrammingBatchPromotionV0(context.Context, AutoprogrammingBatchPromotionRequestV0) (AutoprogrammingBatchPromotionResultV0, error)
}

type AutoprogrammingBatchPromotionClaimReconcilerPortV0 interface {
	ReconcileAutoprogrammingBatchPromotionV0(context.Context, AutoprogrammingBatchPromotionRequestV0) (AutoprogrammingBatchPromotionResultV0, bool, error)
}

func (stack StackV0) maybeFinalizeClosedAutoprogrammingBatchV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, bool, []string, error) {
	state, goalFirst, err := stack.autoprogrammingPromotionGoalStateV0(ctx, run)
	if err != nil || !goalFirst {
		return false, false, nil, err
	}
	batchRef := autoprogrammingBatchGoalContextRefV0(state.Spec.ContextRefs, autoprogrammingBatchContextKindV0)
	taskRef := autoprogrammingBatchGoalContextRefV0(state.Spec.ContextRefs, autoprogrammingBatchTaskContextKindV0)
	if batchRef == "" && taskRef == "" {
		return false, false, nil, nil
	}
	refs := compactStringsV0([]string{"evidence-ref-autoprogramming-batch-v0", batchRef, taskRef})
	store := stack.Stores.AutoprogrammingBatchStore
	if store == nil || batchRef == "" || taskRef == "" {
		return true, false, append(refs, "evidence-ref-autoprogramming-batch-store-missing"), nil
	}
	batch, err := store.LoadAutoprogrammingBatchV0(ctx, batchRef)
	if err != nil {
		return true, false, refs, err
	}
	if !autoprogrammingBatchMemberMatchesRunV0(batch, taskRef, state.GoalRef, run.RunID) {
		return true, false, refs, fmt.Errorf("autoprogramming_batch_member_run_mismatch")
	}
	batch, err = transitionStoredAutoprogrammingBatchV0(ctx, store, batchRef, func(current orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
		return orquestaautoprogramming.RegisterAutoprogrammingBatchFocalCloseV0(current, current.StoreVersion, batchRef+":focal-close:"+taskRef, taskRef)
	})
	if err != nil {
		return true, false, refs, err
	}
	batch, advanceRefs, err := stack.advanceAutoprogrammingBatchV0(ctx, batch)
	refs = compactStringsV0(append(refs, advanceRefs...))
	if err != nil {
		return true, false, refs, err
	}
	return true, batch.Status == orquestaautoprogramming.AutoprogrammingBatchStatusClosedV0, refs, nil
}

func autoprogrammingBatchMemberMatchesRunV0(
	batch orquestaautoprogramming.AutoprogrammingBatchV0,
	taskRef string,
	goalRef string,
	runRef string,
) bool {
	for _, member := range batch.Members {
		if member.TaskRef == strings.TrimSpace(taskRef) {
			return member.GoalRef == strings.TrimSpace(goalRef) && member.RunRef == strings.TrimSpace(runRef)
		}
	}
	return false
}

func (stack StackV0) advanceAutoprogrammingBatchV0(
	ctx context.Context,
	batch orquestaautoprogramming.AutoprogrammingBatchV0,
) (orquestaautoprogramming.AutoprogrammingBatchV0, []string, error) {
	store := stack.Stores.AutoprogrammingBatchStore
	refs := []string{batch.BatchRef, "evidence-ref-autoprogramming-batch-status:" + batch.Status}
	if batch.Status == orquestaautoprogramming.AutoprogrammingBatchStatusPendingIntegrationV0 {
		var err error
		batch, refs, err = stack.integrateAutoprogrammingBatchMembersV0(ctx, store, batch, refs)
		if err != nil || batch.Status == orquestaautoprogramming.AutoprogrammingBatchStatusBlockedV0 {
			return batch, refs, err
		}
	}
	if batch.Status == orquestaautoprogramming.AutoprogrammingBatchStatusPendingBatchGateV0 ||
		batch.Status == orquestaautoprogramming.AutoprogrammingBatchStatusBatchGateRunningV0 {
		var err error
		batch, refs, err = stack.runAutoprogrammingBatchGateV0(ctx, store, batch, refs)
		if err != nil || batch.Status == orquestaautoprogramming.AutoprogrammingBatchStatusBlockedV0 ||
			batch.Status == orquestaautoprogramming.AutoprogrammingBatchStatusReworkPendingV0 {
			return batch, refs, err
		}
	}
	if batch.Status == orquestaautoprogramming.AutoprogrammingBatchStatusBatchGatePassedV0 ||
		batch.Status == orquestaautoprogramming.AutoprogrammingBatchStatusPromotionPendingV0 {
		var err error
		batch, refs, err = stack.promoteAutoprogrammingBatchV0(ctx, store, batch, refs)
		if err != nil || batch.Status != orquestaautoprogramming.AutoprogrammingBatchStatusPromotionPendingV0 ||
			strings.TrimSpace(batch.PromotionReceipt.ReceiptRef) == "" {
			return batch, refs, err
		}
	}
	if batch.Status == orquestaautoprogramming.AutoprogrammingBatchStatusPromotionPendingV0 {
		closed, err := transitionStoredAutoprogrammingBatchV0(ctx, store, batch.BatchRef, func(current orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
			return orquestaautoprogramming.CloseAutoprogrammingBatchV0(current, current.StoreVersion, batch.BatchRef+":close")
		})
		if err != nil {
			return batch, refs, err
		}
		batch = closed
		refs = append(refs, "evidence-ref-autoprogramming-batch-closed")
	}
	return batch, compactStringsV0(refs), nil
}

func (stack StackV0) integrateAutoprogrammingBatchMembersV0(
	ctx context.Context,
	store orquestaautoprogramming.AutoprogrammingBatchStorePortV0,
	batch orquestaautoprogramming.AutoprogrammingBatchV0,
	refs []string,
) (orquestaautoprogramming.AutoprogrammingBatchV0, []string, error) {
	config := stack.AutoprogrammingPromotion
	if config.GoalWorkspaceIntegration == nil || strings.TrimSpace(config.CommitMessage) == "" {
		return stack.blockAutoprogrammingBatchV0(ctx, store, batch, "batch-block-ref-integration-port-missing", refs)
	}
	receiptDir := strings.TrimSpace(config.BatchIntegrationReceiptDir)
	if receiptDir == "" && strings.TrimSpace(config.GoalWorkspaceRoot) != "" {
		receiptDir = filepath.Join(strings.TrimSpace(config.GoalWorkspaceRoot), "autoprogramming-integration-receipts-v0")
	}
	if receiptDir == "" {
		return stack.blockAutoprogrammingBatchV0(ctx, store, batch, "batch-block-ref-integration-receipt-dir-missing", refs)
	}
	for _, member := range batch.Members {
		if member.IntegrationStatus == orquestaautoprogramming.AutoprogrammingBatchIntegrationIntegratedV0 {
			continue
		}
		state, err := stack.autoprogrammingBatchGoalStateV0(ctx, member.RunRef)
		if err != nil {
			return batch, refs, err
		}
		workspaceDir, err := stack.autoprogrammingGoalProjectWorkDirV0(ctx, state)
		if err != nil {
			return stack.blockAutoprogrammingBatchV0(ctx, store, batch, "batch-block-ref-workspace-unavailable-"+codexStackOperationalClosureSafeRefV0(member.TaskRef), refs)
		}
		parentRevision := autoprogrammingBatchExpectedParentRevisionV0(batch)
		claimRef, claimed, claimMatches := autoprogrammingBatchIntegrationClaimRefV0(batch, member.TaskRef, parentRevision)
		if claimed && !claimMatches {
			return stack.blockAutoprogrammingBatchV0(ctx, store, batch, "batch-block-ref-orphan-integration-claim", refs)
		}
		if !claimed {
			claimRef = codexStackDeterministicRefV0("batch-integration-claim-ref-", batch.BatchRef, fmt.Sprint(batch.GateGeneration), member.TaskRef, parentRevision)
			batch, err = transitionStoredAutoprogrammingBatchV0(ctx, store, batch.BatchRef, func(current orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
				return orquestaautoprogramming.ClaimAutoprogrammingBatchIntegrationV0(current, current.StoreVersion, batch.BatchRef+":claim-integration:"+fmt.Sprint(batch.GateGeneration)+":"+member.TaskRef, claimRef, member.TaskRef, parentRevision)
			})
			if err != nil {
				return batch, refs, err
			}
		}
		integrationRef := codexStackDeterministicRefV0("integration-ref-autoprogramming-batch-", batch.BatchRef, fmt.Sprint(batch.GateGeneration), member.TaskRef, parentRevision)
		result, issues := config.GoalWorkspaceIntegration.IntegrateGoalWorkspaceV0(ctx, orquestaruntimeworktree.GoalWorkspaceIntegrationRequestV0{
			IntegrationRef: integrationRef, SourceWorkspaceDir: workspaceDir, CanonicalWorkDir: strings.TrimSpace(stack.Codex.ProjectWorkDir),
			BaseRevision: batch.BaseRevision, ExpectedParentRevision: parentRevision,
			WriteSet: member.WriteSet, CommitMessage: config.CommitMessage, ReceiptDir: receiptDir,
		})
		if len(issues) > 0 || (result.Status != orquestaruntimeworktree.GoalWorkspaceIntegrationStatusIntegratedV0 && result.Status != orquestaruntimeworktree.GoalWorkspaceIntegrationStatusReplayedV0) ||
			strings.TrimSpace(result.SourceCommit) == "" || strings.TrimSpace(result.IntegratedCommit) == "" || strings.TrimSpace(result.IntegrationRef) == "" ||
			strings.TrimSpace(result.ExpectedParentRevision) != parentRevision {
			return stack.blockAutoprogrammingBatchV0(ctx, store, batch, "batch-block-ref-integration-"+codexStackOperationalClosureSafeRefV0(member.TaskRef), append(refs, result.EvidenceRefs...))
		}
		batch, err = transitionStoredAutoprogrammingBatchV0(ctx, store, batch.BatchRef, func(current orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
			return orquestaautoprogramming.RegisterAutoprogrammingBatchIntegrationV0(current, current.StoreVersion, batch.BatchRef+":integrate:"+fmt.Sprint(batch.GateGeneration)+":"+member.TaskRef, claimRef, member.TaskRef, result.SourceCommit, parentRevision, result.IntegratedCommit, result.IntegrationRef)
		})
		if err != nil {
			return batch, refs, err
		}
		refs = compactStringsV0(append(refs, append(result.EvidenceRefs, result.IntegrationRef, result.IntegratedCommit)...))
	}
	return batch, refs, nil
}

func (stack StackV0) runAutoprogrammingBatchGateV0(
	ctx context.Context,
	store orquestaautoprogramming.AutoprogrammingBatchStorePortV0,
	batch orquestaautoprogramming.AutoprogrammingBatchV0,
	refs []string,
) (orquestaautoprogramming.AutoprogrammingBatchV0, []string, error) {
	runner := stack.AutoprogrammingPromotion.BatchTestRunner
	if runner == nil {
		return stack.blockAutoprogrammingBatchV0(ctx, store, batch, "batch-block-ref-test-runner-missing", refs)
	}
	for _, test := range batch.FrozenTests {
		testHash := orquestaautoprogramming.AutoprogrammingBatchTestHashV0(test)
		if autoprogrammingBatchHasTestReceiptV0(batch, batch.IntegratedRevision, testHash) {
			continue
		}
		claimRef, claimed := autoprogrammingBatchClaimRefV0(batch, batch.GateGeneration, batch.IntegratedRevision, testHash)
		request := AutoprogrammingBatchTestRunRequestV0{BatchRef: batch.BatchRef, GateGeneration: batch.GateGeneration, Revision: batch.IntegratedRevision, Test: test, TestHash: testHash, ClaimRef: claimRef, CanonicalWorkDir: strings.TrimSpace(stack.Codex.ProjectWorkDir)}
		if claimed {
			reconciler, ok := runner.(AutoprogrammingBatchTestClaimReconcilerPortV0)
			if !ok {
				return stack.blockAutoprogrammingBatchV0(ctx, store, batch, "batch-block-ref-orphan-claim-"+codexStackDeterministicDigestV0(batch.IntegratedRevision, testHash), append(refs, claimRef))
			}
			reconciled, found, err := reconciler.ReconcileAutoprogrammingBatchTestClaimV0(ctx, request)
			if err != nil || !found {
				return stack.blockAutoprogrammingBatchV0(ctx, store, batch, "batch-block-ref-orphan-claim-"+codexStackDeterministicDigestV0(batch.IntegratedRevision, testHash), append(refs, claimRef))
			}
			batch, refs, err = recordAutoprogrammingBatchTestResultV0(ctx, store, batch, request, reconciled, refs)
			if err != nil {
				return batch, refs, err
			}
			continue
		}
		claimRef = codexStackDeterministicRefV0("batch-test-claim-ref-", batch.BatchRef, fmt.Sprint(batch.GateGeneration), batch.IntegratedRevision, testHash)
		var err error
		batch, err = transitionStoredAutoprogrammingBatchV0(ctx, store, batch.BatchRef, func(current orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
			return orquestaautoprogramming.ClaimAutoprogrammingBatchTestV0(current, current.StoreVersion, batch.BatchRef+":claim:"+fmt.Sprint(batch.GateGeneration)+":"+batch.IntegratedRevision+":"+testHash, batch.IntegratedRevision, testHash, claimRef)
		})
		if err != nil {
			return batch, refs, err
		}
		request.ClaimRef = claimRef
		result, err := runner.RunAutoprogrammingBatchTestV0(ctx, request)
		if err != nil {
			return stack.blockAutoprogrammingBatchV0(ctx, store, batch, "batch-block-ref-test-execution-"+codexStackDeterministicDigestV0(batch.IntegratedRevision, testHash), append(refs, claimRef))
		}
		batch, refs, err = recordAutoprogrammingBatchTestResultV0(ctx, store, batch, request, result, refs)
		if err != nil || batch.Status == orquestaautoprogramming.AutoprogrammingBatchStatusReworkPendingV0 {
			return batch, refs, err
		}
	}
	return batch, refs, nil
}

func recordAutoprogrammingBatchTestResultV0(
	ctx context.Context,
	store orquestaautoprogramming.AutoprogrammingBatchStorePortV0,
	batch orquestaautoprogramming.AutoprogrammingBatchV0,
	request AutoprogrammingBatchTestRunRequestV0,
	result AutoprogrammingBatchTestRunResultV0,
	refs []string,
) (orquestaautoprogramming.AutoprogrammingBatchV0, []string, error) {
	status, receiptRef := strings.TrimSpace(result.Status), strings.TrimSpace(result.ReceiptRef)
	if receiptRef == "" || (status != orquestaautoprogramming.AutoprogrammingBatchTestReceiptPassedV0 && status != orquestaautoprogramming.AutoprogrammingBatchTestReceiptFailedV0) {
		return batch, refs, fmt.Errorf("autoprogramming_batch_test_receipt_invalid")
	}
	saved, err := transitionStoredAutoprogrammingBatchV0(ctx, store, batch.BatchRef, func(current orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
		return orquestaautoprogramming.RecordAutoprogrammingBatchTestReceiptV0(current, current.StoreVersion, batch.BatchRef+":test-receipt:"+fmt.Sprint(batch.GateGeneration)+":"+request.Revision+":"+request.TestHash, request.Revision, request.TestHash, request.ClaimRef, receiptRef, status)
	})
	return saved, compactStringsV0(append(refs, append(result.EvidenceRefs, receiptRef)...)), err
}

func (stack StackV0) promoteAutoprogrammingBatchV0(
	ctx context.Context,
	store orquestaautoprogramming.AutoprogrammingBatchStorePortV0,
	batch orquestaautoprogramming.AutoprogrammingBatchV0,
	refs []string,
) (orquestaautoprogramming.AutoprogrammingBatchV0, []string, error) {
	config := stack.AutoprogrammingPromotion
	if strings.TrimSpace(batch.PromotionReceipt.ReceiptRef) != "" {
		return batch, refs, nil
	}
	claimRef, claimed := autoprogrammingBatchPromotionClaimRefV0(batch)
	if claimed && config.BatchPromotionReconciler == nil {
		return stack.blockAutoprogrammingBatchV0(ctx, store, batch, "batch-block-ref-promotion-reconciler-missing", append(refs, claimRef))
	}
	if !claimed && config.BatchPromotionFinalizer == nil {
		return stack.blockAutoprogrammingBatchV0(ctx, store, batch, "batch-block-ref-promotion-finalizer-missing", refs)
	}
	if strings.TrimSpace(stack.Codex.ProjectWorkDir) == "" || strings.TrimSpace(config.BatchPromotionReceiptDir) == "" {
		return stack.blockAutoprogrammingBatchV0(ctx, store, batch, "batch-block-ref-promotion-paths-missing", refs)
	}
	if !claimed {
		claimRef = codexStackDeterministicRefV0("batch-promotion-claim-ref-", batch.BatchRef, fmt.Sprint(batch.GateGeneration), batch.IntegratedRevision)
		var err error
		batch, err = transitionStoredAutoprogrammingBatchV0(ctx, store, batch.BatchRef, func(current orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
			return orquestaautoprogramming.ClaimAutoprogrammingBatchPromotionV0(current, current.StoreVersion, batch.BatchRef+":claim-promotion:"+fmt.Sprint(batch.GateGeneration), claimRef, batch.IntegratedRevision)
		})
		if err != nil {
			return batch, refs, err
		}
	}
	request := AutoprogrammingBatchPromotionRequestV0{
		BatchRef: batch.BatchRef, GateGeneration: batch.GateGeneration, ClaimRef: claimRef,
		IntegratedRevision: batch.IntegratedRevision, CanonicalWorkDir: strings.TrimSpace(stack.Codex.ProjectWorkDir),
		ReceiptDir: strings.TrimSpace(config.BatchPromotionReceiptDir), EvidenceRefs: compactStringsV0(append(refs, batch.IntegratedRevision)),
	}
	var result AutoprogrammingBatchPromotionResultV0
	var err error
	if claimed {
		var found bool
		result, found, err = config.BatchPromotionReconciler.ReconcileAutoprogrammingBatchPromotionV0(ctx, request)
		if err != nil || !found {
			return stack.blockAutoprogrammingBatchV0(ctx, store, batch, "batch-block-ref-orphan-promotion-claim", append(refs, claimRef))
		}
	} else {
		result, err = config.BatchPromotionFinalizer.FinalizeAutoprogrammingBatchPromotionV0(ctx, request)
		if err != nil {
			return batch, compactStringsV0(append(refs, result.EvidenceRefs...)), err
		}
	}
	refs = compactStringsV0(append(refs, result.EvidenceRefs...))
	if !autoprogrammingBatchPromotionResultMatchesV0(request, result) {
		return stack.blockAutoprogrammingBatchV0(ctx, store, batch, "batch-block-ref-promotion-result-invalid", refs)
	}
	saved, err := transitionStoredAutoprogrammingBatchV0(ctx, store, batch.BatchRef, func(current orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
		return orquestaautoprogramming.RegisterAutoprogrammingBatchPromotionV0(current, current.StoreVersion, batch.BatchRef+":promotion:"+fmt.Sprint(batch.GateGeneration), claimRef, batch.IntegratedRevision, result.ReceiptRef)
	})
	return saved, refs, err
}

func autoprogrammingBatchPromotionResultMatchesV0(request AutoprogrammingBatchPromotionRequestV0, result AutoprogrammingBatchPromotionResultV0) bool {
	return result.BatchRef == request.BatchRef && result.GateGeneration == request.GateGeneration &&
		result.ClaimRef == request.ClaimRef && result.IntegratedRevision == request.IntegratedRevision &&
		result.CanonicalClean && strings.TrimSpace(result.ReceiptRef) != ""
}

func (stack StackV0) autoprogrammingBatchGoalStateV0(ctx context.Context, runRef string) (orquestagoal.GoalWorkStateV0, error) {
	store := stack.Ports.GoalStateStore
	if store == nil {
		store = stack.Stores.AppGoalStateStore
	}
	if store == nil {
		return orquestagoal.GoalWorkStateV0{}, fmt.Errorf("autoprogramming_batch_goal_state_store_missing")
	}
	return store.LoadGoalWorkStateV0(ctx, strings.TrimSpace(runRef))
}

func (stack StackV0) blockAutoprogrammingBatchV0(
	ctx context.Context,
	store orquestaautoprogramming.AutoprogrammingBatchStorePortV0,
	batch orquestaautoprogramming.AutoprogrammingBatchV0,
	blockRef string,
	refs []string,
) (orquestaautoprogramming.AutoprogrammingBatchV0, []string, error) {
	blocked, err := transitionStoredAutoprogrammingBatchV0(ctx, store, batch.BatchRef, func(current orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
		return orquestaautoprogramming.BlockAutoprogrammingBatchV0(current, current.StoreVersion, batch.BatchRef+":block:"+blockRef, blockRef)
	})
	return blocked, compactStringsV0(append(refs, blockRef)), err
}

func autoprogrammingBatchClaimRefV0(batch orquestaautoprogramming.AutoprogrammingBatchV0, generation uint64, revision, testHash string) (string, bool) {
	for _, claim := range batch.TestClaims {
		if claim.GateGeneration == generation && claim.Revision == revision && claim.TestHash == testHash {
			return claim.ClaimRef, true
		}
	}
	return "", false
}

func autoprogrammingBatchHasTestReceiptV0(batch orquestaautoprogramming.AutoprogrammingBatchV0, revision, testHash string) bool {
	for _, receipt := range batch.TestReceipts {
		if receipt.GateGeneration == batch.GateGeneration && receipt.Revision == revision && receipt.TestHash == testHash {
			return true
		}
	}
	return false
}

func autoprogrammingBatchExpectedParentRevisionV0(batch orquestaautoprogramming.AutoprogrammingBatchV0) string {
	if strings.TrimSpace(batch.IntegrationHeadRevision) != "" {
		return strings.TrimSpace(batch.IntegrationHeadRevision)
	}
	return strings.TrimSpace(batch.BaseRevision)
}

func autoprogrammingBatchIntegrationClaimRefV0(
	batch orquestaautoprogramming.AutoprogrammingBatchV0,
	taskRef string,
	parentRevision string,
) (string, bool, bool) {
	for _, claim := range batch.IntegrationClaims {
		if claim.GateGeneration != batch.GateGeneration || claim.Status != orquestaautoprogramming.AutoprogrammingBatchClaimStatusClaimedV0 {
			continue
		}
		return claim.ClaimRef, true, claim.TaskRef == taskRef && claim.ParentRevision == parentRevision
	}
	return "", false, false
}

func autoprogrammingBatchPromotionClaimRefV0(batch orquestaautoprogramming.AutoprogrammingBatchV0) (string, bool) {
	for _, claim := range batch.PromotionClaims {
		if claim.GateGeneration == batch.GateGeneration && claim.Revision == batch.IntegratedRevision && claim.Status == orquestaautoprogramming.AutoprogrammingBatchClaimStatusClaimedV0 {
			return claim.ClaimRef, true
		}
	}
	return "", false
}
