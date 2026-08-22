package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"math"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

// ResolveAgentHistoricalRuntimeAuthority derives the accepted launch fact from
// the existing ledgers. V40 deliberately owns no row and accepts no digest or
// subject field from its caller.
func (repository *Repository) ResolveAgentHistoricalRuntimeAuthority(
	ctx context.Context,
	key ports.AgentHistoricalRuntimeAuthorityKey,
) (ports.AgentHistoricalRuntimeAuthority, error) {
	if ctx == nil || key.ActionFence == 0 || key.ActionFence > math.MaxInt64 || key.ExecutionRef.String() == "" {
		return ports.AgentHistoricalRuntimeAuthority{}, invalid(
			errors.New("sqlite.agent_historical_runtime_authority_key_invalid"),
		)
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentHistoricalRuntimeAuthority{}, err
	}

	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return ports.AgentHistoricalRuntimeAuthority{}, microVMHostLaunchContextError(ctx, err)
	}
	defer transaction.Rollback()

	var executionRef, goalRef, workItemRef, specHash string
	var providerRef, modelRef, agentRef, externalRef string
	var planSHA256, grantSHA256, kernelSHA256, initramfsSHA256, profileSHA256 string
	var actionFence, planGeneration, appSpecGeneration, executionAttempt int64
	err = transaction.QueryRowContext(ctx, `
SELECT execution.ref,attempt.action_fence,
       attempt.goal_ref,attempt.work_item_ref,attempt.plan_generation,
       attempt.app_spec_generation,execution.attempt_no,attempt.spec_hash,
       execution.provider_ref,execution.model_ref,execution.agent_ref,receipt.external_ref,
       runtime.plan_sha256,runtime.concession_sha256,
       runtime.kernel_sha256,runtime.initramfs_sha256,runtime.profile_sha256
FROM microvm_host_launch_authorities authority
JOIN microvm_host_launch_runtime_digests runtime
  ON runtime.execution_ref=authority.execution_ref
 AND runtime.action_fence=authority.action_fence
 AND runtime.plan_sha256=authority.plan_sha256
 AND runtime.concession_sha256=authority.concession_sha256
JOIN effect_attempts attempt
  ON attempt.ref=authority.effect_attempt_ref
 AND attempt.execution_ref=authority.execution_ref
 AND attempt.action_fence=authority.action_fence
JOIN effect_intents intent
  ON intent.ref=attempt.intent_ref
 AND intent.kind='agent_launch'
 AND intent.execution_ref=attempt.execution_ref
JOIN effect_receipts receipt
  ON receipt.attempt_ref=attempt.ref
 AND receipt.status='accepted'
 AND receipt.intent_ref=attempt.intent_ref
 AND receipt.intent_digest=attempt.intent_digest
 AND receipt.approval_ref=attempt.approval_ref
 AND receipt.project_ref=attempt.project_ref
 AND receipt.goal_ref=attempt.goal_ref
 AND receipt.work_item_ref=attempt.work_item_ref
 AND receipt.execution_ref=attempt.execution_ref
 AND receipt.plan_generation=attempt.plan_generation
 AND receipt.app_spec_generation=attempt.app_spec_generation
 AND receipt.spec_hash=attempt.spec_hash
 AND receipt.actor_ref=attempt.actor_ref
 AND receipt.action_ref=attempt.action_ref
 AND receipt.action_fence=attempt.action_fence
JOIN executions execution
  ON execution.ref=attempt.execution_ref
 AND execution.goal_ref=attempt.goal_ref
 AND execution.work_item_ref=attempt.work_item_ref
 AND execution.plan_generation=attempt.plan_generation
 AND execution.app_spec_generation=attempt.app_spec_generation
 AND execution.spec_hash=attempt.spec_hash
 AND execution.external_ref=receipt.external_ref
WHERE authority.execution_ref=? AND authority.action_fence=?`,
		key.ExecutionRef.String(), key.ActionFence,
	).Scan(
		&executionRef, &actionFence,
		&goalRef, &workItemRef, &planGeneration,
		&appSpecGeneration, &executionAttempt, &specHash,
		&providerRef, &modelRef, &agentRef, &externalRef,
		&planSHA256, &grantSHA256,
		&kernelSHA256, &initramfsSHA256, &profileSHA256,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.AgentHistoricalRuntimeAuthority{}, stateError(
			application.StateNotFound,
			errors.New("sqlite.agent_historical_runtime_authority_not_found"),
		)
	}
	if err != nil {
		return ports.AgentHistoricalRuntimeAuthority{}, microVMHostLaunchDatabaseError(ctx, err)
	}

	execution, executionErr := goal.NewExecutionRef(executionRef)
	goalID, goalErr := goal.NewGoalRef(goalRef)
	workItem, workItemErr := goal.NewWorkItemRef(workItemRef)
	if executionErr != nil || goalErr != nil || workItemErr != nil ||
		actionFence <= 0 || planGeneration <= 0 || appSpecGeneration <= 0 || executionAttempt <= 0 {
		return ports.AgentHistoricalRuntimeAuthority{}, invalid(
			errors.New("sqlite.agent_historical_runtime_authority_corrupt"),
		)
	}
	authority := ports.AgentHistoricalRuntimeAuthority{
		Key: ports.AgentHistoricalRuntimeAuthorityKey{
			ExecutionRef: execution,
			ActionFence:  uint64(actionFence),
		},
		Subject: ports.AgentEnvironmentLifecycleSubject{
			ExecutionRef:      execution,
			GoalRef:           goalID,
			WorkItemRef:       workItem,
			PlanGeneration:    goal.PlanGeneration(planGeneration),
			AppSpecGeneration: goal.AppSpecGeneration(appSpecGeneration),
			ExecutionAttempt:  uint64(executionAttempt),
			SpecHash:          specHash,
			ProviderRef:       providerRef,
			ModelRef:          modelRef,
			AgentRef:          agentRef,
			ExternalRef:       externalRef,
		},
		Digests: ports.AgentHistoricalRuntimeDigests{
			PlanSHA256:      planSHA256,
			GrantSHA256:     grantSHA256,
			KernelSHA256:    kernelSHA256,
			InitramfsSHA256: initramfsSHA256,
			ProfileSHA256:   profileSHA256,
		},
	}
	if authority.Key != key || ports.ValidateAgentHistoricalRuntimeAuthority(authority) != nil {
		return ports.AgentHistoricalRuntimeAuthority{}, invalid(
			errors.New("sqlite.agent_historical_runtime_authority_corrupt"),
		)
	}
	if err := commit(transaction); err != nil {
		return ports.AgentHistoricalRuntimeAuthority{}, microVMHostLaunchContextError(ctx, err)
	}
	return authority, nil
}
