package sqlite

import (
	"context"
	"database/sql"
	"errors"
)

// validateRecoveryV16WorkspaceGit keeps the append-only V16 facts tied to the
// one Goal/execution and V15 effect ledger.  It deliberately checks durable
// causal fields rather than filesystem paths, which never enter SQLite.
func validateRecoveryV16WorkspaceGit(ctx context.Context, tx *sql.Tx) error {
	for _, check := range recoveryV16Checks {
		var invalid int
		if err := tx.QueryRowContext(ctx, check.query).Scan(&invalid); err != nil {
			return err
		}
		if invalid != 0 {
			return errors.New(check.code)
		}
	}
	return nil
}

var recoveryV16Checks = []recoveryV15Check{
	{"sqlite.recovery_v16_workspace_binding_invalid", `
SELECT COUNT(*)
FROM workspace_bindings binding
JOIN goals goal ON goal.ref=binding.goal_ref
JOIN executions execution ON execution.goal_ref=binding.goal_ref
 AND execution.work_item_ref=binding.work_item_ref AND execution.ref=binding.execution_ref
JOIN effect_intents intent ON intent.ref=binding.intent_ref
JOIN effect_attempts attempt ON attempt.ref=binding.attempt_ref
JOIN effect_receipts receipt ON receipt.ref=binding.effect_receipt_ref
WHERE binding.project_ref<>goal.project_ref OR binding.actor_ref<>goal.actor_ref
 OR binding.execution_attempt<>execution.attempt_no
 OR binding.plan_generation<>execution.plan_generation
 OR binding.app_spec_generation<>execution.app_spec_generation
 OR binding.app_spec_hash<>execution.spec_hash
 OR intent.goal_ref<>binding.goal_ref OR intent.work_item_ref<>binding.work_item_ref
 OR intent.execution_ref<>binding.execution_ref
 OR intent.kind<>'prepare_workspace' OR intent.action_kind<>'prepare_workspace'
 OR attempt.intent_ref<>binding.intent_ref OR attempt.action_ref<>intent.action_ref
 OR attempt.action_fence<>binding.action_fence
 OR receipt.attempt_ref<>binding.attempt_ref OR receipt.intent_ref<>binding.intent_ref
 OR receipt.action_ref<>intent.action_ref OR receipt.action_fence<>binding.action_fence
 OR receipt.status<>'prepared' OR receipt.confirmed_at<binding.prepared_at
 OR NOT EXISTS (SELECT 1 FROM outbox action
   WHERE action.ref=intent.action_ref AND action.kind='prepare_workspace'
    AND action.effect_intent_ref=intent.ref AND action.goal_ref=binding.goal_ref
    AND action.work_item_ref=binding.work_item_ref AND action.execution_ref=binding.execution_ref
    AND action.fence=binding.action_fence)
 OR NOT EXISTS (SELECT 1 FROM action_consumption_receipts consumed
   WHERE consumed.action_ref=intent.action_ref AND consumed.kind='prepare_workspace'
    AND consumed.goal_ref=binding.goal_ref AND consumed.work_item_ref=binding.work_item_ref
    AND consumed.execution_ref=binding.execution_ref AND consumed.change_ref=''
    AND consumed.fence=binding.action_fence AND consumed.outcome='completed'
    AND consumed.error_code='' AND consumed.effect_receipt_ref=receipt.ref
    AND consumed.consumed_at>=receipt.confirmed_at)`},
	{"sqlite.recovery_v16_workspace_write_set_invalid", `
SELECT COUNT(*) FROM workspace_bindings binding
WHERE NOT EXISTS (SELECT 1 FROM workspace_binding_write_scopes scope
                  WHERE scope.workspace_binding_ref=binding.ref)
   OR EXISTS (
      SELECT 1 FROM workspace_binding_write_scopes scope
      WHERE scope.workspace_binding_ref=binding.ref
      GROUP BY scope.workspace_binding_ref HAVING MIN(scope.ordinal)<>0 OR MAX(scope.ordinal)+1<>COUNT(*)
   )`},
	{"sqlite.recovery_v16_change_set_invalid", `
SELECT COUNT(*)
FROM change_sets change_set
JOIN workspace_bindings binding ON binding.ref=change_set.workspace_binding_ref
JOIN effect_intents intent ON intent.ref=change_set.intent_ref
JOIN effect_attempts attempt ON attempt.ref=change_set.attempt_ref
JOIN effect_receipts receipt ON receipt.ref=change_set.effect_receipt_ref
WHERE change_set.project_ref<>binding.project_ref OR change_set.repository_ref<>binding.repository_ref
 OR change_set.goal_ref<>binding.goal_ref OR change_set.work_item_ref<>binding.work_item_ref
 OR change_set.execution_ref<>binding.execution_ref OR change_set.execution_attempt<>binding.execution_attempt
 OR change_set.plan_generation<>binding.plan_generation
 OR change_set.app_spec_generation<>binding.app_spec_generation
 OR change_set.app_spec_hash<>binding.app_spec_hash OR change_set.write_set_digest<>binding.write_set_digest
 OR change_set.base_oid<>binding.base_oid OR change_set.object_format<>binding.object_format
 OR (change_set.parent_change_ref IS NOT NULL AND NOT EXISTS (
   SELECT 1 FROM change_sets parent WHERE parent.ref=change_set.parent_change_ref
    AND parent.project_ref=change_set.project_ref AND parent.repository_ref=change_set.repository_ref
 ))
 OR intent.goal_ref<>change_set.goal_ref OR intent.work_item_ref<>change_set.work_item_ref
 OR intent.execution_ref<>change_set.execution_ref
 OR intent.kind<>'commit_change' OR intent.action_kind<>'commit_change'
 OR attempt.intent_ref<>change_set.intent_ref OR attempt.action_ref<>intent.action_ref
 OR attempt.action_fence<>change_set.action_fence
 OR receipt.attempt_ref<>change_set.attempt_ref OR receipt.intent_ref<>change_set.intent_ref
 OR receipt.action_ref<>intent.action_ref OR receipt.action_fence<>change_set.action_fence
 OR receipt.status<>'committed' OR receipt.confirmed_at<change_set.committed_at
 OR NOT EXISTS (SELECT 1 FROM outbox action
   WHERE action.ref=intent.action_ref AND action.kind='commit_change'
    AND action.effect_intent_ref=intent.ref AND action.goal_ref=change_set.goal_ref
    AND action.work_item_ref=change_set.work_item_ref AND action.execution_ref=change_set.execution_ref
    AND action.change_ref=change_set.ref AND action.fence=change_set.action_fence)
 OR NOT EXISTS (SELECT 1 FROM action_consumption_receipts consumed
   WHERE consumed.action_ref=intent.action_ref AND consumed.kind='commit_change'
    AND consumed.goal_ref=change_set.goal_ref AND consumed.work_item_ref=change_set.work_item_ref
    AND consumed.execution_ref=change_set.execution_ref AND consumed.change_ref=change_set.ref
    AND consumed.fence=change_set.action_fence AND consumed.outcome='completed'
    AND consumed.error_code='' AND consumed.effect_receipt_ref=receipt.ref
    AND consumed.consumed_at>=receipt.confirmed_at)`},
	{"sqlite.recovery_v16_change_set_paths_invalid", `
SELECT COUNT(*) FROM change_sets change_set
WHERE NOT EXISTS (SELECT 1 FROM change_set_paths path WHERE path.change_set_ref=change_set.ref)
   OR EXISTS (
      SELECT 1 FROM change_set_paths path WHERE path.change_set_ref=change_set.ref
      GROUP BY path.change_set_ref HAVING MIN(path.ordinal)<>0 OR MAX(path.ordinal)+1<>COUNT(*)
   )`},
	{"sqlite.recovery_v16_merge_observation_invalid", `
SELECT COUNT(*)
FROM merge_observations observation
JOIN change_sets change_set ON change_set.ref=observation.change_set_ref
WHERE observation.project_ref<>change_set.project_ref OR observation.repository_ref<>change_set.repository_ref
 OR observation.source_oid<>change_set.head_oid OR observation.object_format<>change_set.object_format
`},
	{"sqlite.recovery_v16_integration_completion_fact_missing", `
SELECT COUNT(*)
FROM action_consumption_receipts consumed
JOIN outbox action ON action.ref=consumed.action_ref
JOIN effect_intents intent ON intent.ref=action.effect_intent_ref
JOIN effect_receipts receipt ON receipt.ref=consumed.effect_receipt_ref
WHERE action.kind='integrate_change' AND consumed.kind='integrate_change'
 AND consumed.outcome='completed' AND consumed.error_code=''
 AND (SELECT COUNT(*)
      FROM integration_receipts integration
      JOIN merge_observations observation ON observation.ref=integration.merge_observation_ref
      WHERE integration.effect_receipt_ref=receipt.ref
       AND integration.intent_ref=intent.ref
       AND integration.attempt_ref=receipt.attempt_ref
       AND integration.action_fence=consumed.fence
       AND integration.change_set_ref=action.change_ref
       AND observation.change_set_ref=action.change_ref)<>1`},
	{"sqlite.recovery_v16_succeeded_workspace_integration_missing", `
SELECT COUNT(*)
FROM executions execution
WHERE execution.execution_workspace_ref<>'' AND execution.state='succeeded'
 AND EXISTS (
  SELECT 1
  FROM outbox action
  JOIN action_consumption_receipts consumed ON consumed.action_ref=action.ref
  WHERE action.kind='integrate_change' AND action.execution_ref=execution.ref
   AND action.goal_ref=execution.goal_ref AND action.work_item_ref=execution.work_item_ref
   AND consumed.kind='integrate_change' AND consumed.outcome='completed'
   AND consumed.error_code='')
 AND NOT EXISTS (
  SELECT 1
  FROM change_sets change_set
  JOIN integration_receipts integration ON integration.change_set_ref=change_set.ref
  JOIN merge_observations observation ON observation.ref=integration.merge_observation_ref
  WHERE change_set.execution_ref=execution.ref
   AND change_set.goal_ref=execution.goal_ref
   AND change_set.work_item_ref=execution.work_item_ref
   AND change_set.workspace_binding_ref=execution.execution_workspace_ref
   AND integration.status='integrated'
   AND observation.change_set_ref=change_set.ref)`},
	{"sqlite.recovery_v16_integration_receipt_invalid", `
SELECT COUNT(*)
FROM integration_receipts integration
JOIN merge_observations observation ON observation.ref=integration.merge_observation_ref
JOIN change_sets change_set ON change_set.ref=integration.change_set_ref
JOIN effect_intents intent ON intent.ref=integration.intent_ref
JOIN effect_attempts attempt ON attempt.ref=integration.attempt_ref
JOIN effect_receipts receipt ON receipt.ref=integration.effect_receipt_ref
WHERE observation.change_set_ref<>integration.change_set_ref
 OR integration.project_ref<>change_set.project_ref OR integration.repository_ref<>change_set.repository_ref
 OR integration.target_ref<>observation.target_ref OR integration.source_oid<>observation.source_oid
 OR integration.target_before_oid<>observation.target_oid OR integration.object_format<>observation.object_format
 OR integration.status<>CASE observation.outcome WHEN 'clean' THEN 'integrated' ELSE observation.outcome END
 OR intent.goal_ref<>change_set.goal_ref OR intent.work_item_ref<>change_set.work_item_ref
 OR intent.execution_ref<>change_set.execution_ref
 OR intent.kind<>'integrate_change' OR intent.action_kind<>'integrate_change'
 OR attempt.intent_ref<>integration.intent_ref OR attempt.action_ref<>intent.action_ref
 OR attempt.action_fence<>integration.action_fence
 OR receipt.attempt_ref<>integration.attempt_ref OR receipt.intent_ref<>integration.intent_ref
 OR receipt.action_ref<>intent.action_ref OR receipt.action_fence<>integration.action_fence
 OR receipt.status<>integration.status OR receipt.confirmed_at<integration.confirmed_at
 OR NOT EXISTS (SELECT 1 FROM outbox action
   WHERE action.ref=intent.action_ref AND action.kind='integrate_change'
    AND action.effect_intent_ref=intent.ref AND action.goal_ref=change_set.goal_ref
    AND action.work_item_ref=change_set.work_item_ref AND action.execution_ref=change_set.execution_ref
    AND action.change_ref=change_set.ref AND action.expected_target_oid=integration.target_before_oid
    AND action.fence=integration.action_fence)
 OR NOT EXISTS (SELECT 1 FROM action_consumption_receipts consumed
   WHERE consumed.action_ref=intent.action_ref AND consumed.kind='integrate_change'
    AND consumed.goal_ref=change_set.goal_ref AND consumed.work_item_ref=change_set.work_item_ref
    AND consumed.execution_ref=change_set.execution_ref AND consumed.change_ref=change_set.ref
    AND consumed.fence=integration.action_fence AND consumed.outcome='completed'
    AND consumed.error_code='' AND consumed.effect_receipt_ref=receipt.ref
    AND consumed.consumed_at>=receipt.confirmed_at)`},
}
