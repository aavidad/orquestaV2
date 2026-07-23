package sqlite

import (
	"context"
	"database/sql"
	"errors"
)

func validateRecoveryV19Council(ctx context.Context, tx *sql.Tx) error {
	checks := []struct{ code, query string }{
		{"sqlite.recovery_v19_council_policy_invalid", `SELECT COUNT(*) FROM work_items item WHERE
 (item.council_policy='' AND EXISTS(SELECT 1 FROM work_item_write_scopes scope
  WHERE scope.goal_ref=item.goal_ref AND scope.work_item_ref=item.ref)
  AND item.state NOT IN ('succeeded','failed','skipped','canceled','superseded'))`},
		{"sqlite.recovery_v19_execution_scope_invalid", `SELECT COUNT(*) FROM executions execution
LEFT JOIN council_rounds round ON round.goal_ref=execution.goal_ref AND round.work_item_ref=execution.work_item_ref
 AND round.subject_digest=execution.council_subject_digest
WHERE (execution.purpose IN ('council_proposer','council_critic','council_arbiter') AND
 (execution.review_subject_digest<>'' OR round.ref IS NULL)) OR
 (execution.purpose NOT IN ('council_proposer','council_critic','council_arbiter') AND execution.council_subject_digest<>'')`},
		{"sqlite.recovery_v19_round_scope_invalid", `SELECT COUNT(*) FROM council_rounds round
LEFT JOIN goals goal ON goal.ref=round.goal_ref
LEFT JOIN work_items item ON item.goal_ref=round.goal_ref AND item.ref=round.work_item_ref
LEFT JOIN change_sets change_set ON change_set.ref=round.change_set_ref
WHERE goal.ref IS NULL OR item.ref IS NULL OR change_set.ref IS NULL OR round.project_ref<>goal.project_ref
 OR round.policy<>item.council_policy OR change_set.goal_ref<>round.goal_ref
 OR change_set.work_item_ref<>round.work_item_ref OR change_set.plan_generation<>round.plan_generation
 OR change_set.app_spec_generation<>round.app_spec_generation OR change_set.app_spec_hash<>round.spec_hash
 OR round.work_item_generation>item.revision`},
		{"sqlite.recovery_v19_fact_scope_invalid", `SELECT COUNT(*) FROM council_facts fact
LEFT JOIN council_rounds round ON round.ref=fact.round_ref
LEFT JOIN executions execution ON execution.goal_ref=fact.goal_ref AND execution.work_item_ref=fact.work_item_ref AND execution.ref=fact.execution_ref
LEFT JOIN artifact_occurrences occurrence ON occurrence.occurrence_ref=fact.artifact_occurrence_ref
LEFT JOIN artifacts artifact ON artifact.goal_ref=fact.goal_ref AND artifact.ref=fact.artifact_ref
WHERE round.ref IS NULL OR execution.ref IS NULL OR occurrence.occurrence_ref IS NULL OR artifact.ref IS NULL
 OR fact.council_subject_digest<>round.subject_digest OR execution.council_subject_digest<>fact.council_subject_digest
 OR execution.review_subject_digest<>'' OR execution.state<>'succeeded' OR execution.attempt_no<>fact.execution_attempt
 OR execution.launch_receipt_ref<>fact.launch_receipt_ref OR execution.external_ref<>fact.external_ref
 OR occurrence.kind<>'council_contribution' OR occurrence.execution_ref<>fact.execution_ref
 OR occurrence.artifact_ref<>fact.artifact_ref OR artifact.digest<>fact.artifact_digest
 OR (fact.role='proposer' AND execution.purpose<>'council_proposer')
 OR (fact.role='critic' AND execution.purpose<>'council_critic')
 OR (fact.role='arbiter' AND execution.purpose<>'council_arbiter')`},
		{"sqlite.recovery_v19_decision_scope_invalid", `SELECT COUNT(*) FROM council_decisions decision
LEFT JOIN council_rounds round ON round.ref=decision.round_ref
WHERE round.ref IS NULL OR decision.subject_digest<>round.subject_digest OR
 (SELECT COUNT(*) FROM council_facts fact WHERE fact.round_ref=round.ref)<>3`},
		{"sqlite.recovery_v19_skip_scope_invalid", `SELECT COUNT(*) FROM council_skips skip
LEFT JOIN work_items item ON item.goal_ref=skip.goal_ref AND item.ref=skip.work_item_ref
WHERE item.ref IS NULL OR item.council_policy<>'skip_by_operator' OR
 EXISTS(SELECT 1 FROM council_rounds round WHERE round.subject_digest=skip.subject_digest) OR
 EXISTS(SELECT 1 FROM council_facts fact WHERE fact.council_subject_digest=skip.subject_digest)`},
		{"sqlite.recovery_v19_outbox_resolution_invalid", `SELECT COUNT(*) FROM outbox action
LEFT JOIN work_items item ON item.goal_ref=action.goal_ref AND item.ref=action.work_item_ref
LEFT JOIN council_decisions decision ON decision.ref=action.council_decision_ref
LEFT JOIN council_skips skip ON skip.ref=action.council_skip_ref
WHERE (action.kind<>'integrate_change' AND action.council_subject_digest<>'') OR
 (action.kind='integrate_change' AND item.council_policy<>'' AND
  ((action.council_resolution_kind='accepted_round' AND (decision.ref IS NULL OR decision.outcome<>'accepted'
    OR decision.subject_digest<>action.council_subject_digest OR decision.decision_digest<>action.council_decision_digest))
   OR (action.council_resolution_kind='skip' AND (skip.ref IS NULL OR skip.subject_digest<>action.council_subject_digest
    OR skip.skip_digest<>action.council_skip_digest))
   OR action.council_resolution_kind='')) OR
 (action.kind='integrate_change' AND item.council_policy='' AND action.council_resolution_kind<>'')`},
		{"sqlite.recovery_v19_effect_resolution_invalid", `SELECT COUNT(*) FROM effect_intents intent
LEFT JOIN outbox action ON action.ref=intent.action_ref
WHERE (intent.council_subject_digest<>action.council_subject_digest) OR
 COALESCE(intent.council_decision_ref,'')<>COALESCE(action.council_decision_ref,'') OR
 COALESCE(intent.council_decision_digest,'')<>COALESCE(action.council_decision_digest,'') OR
 COALESCE(intent.council_skip_ref,'')<>COALESCE(action.council_skip_ref,'') OR
 COALESCE(intent.council_skip_digest,'')<>COALESCE(action.council_skip_digest,'')`},
		{"sqlite.recovery_v19_director_council_invalid", `SELECT COUNT(*) FROM director_decisions director
LEFT JOIN council_decisions decision ON decision.ref=director.council_decision_ref
WHERE (director.council_subject_digest='' AND
 (director.council_decision_ref IS NOT NULL OR director.council_decision_digest IS NOT NULL)) OR
 (director.council_subject_digest<>'' AND (decision.ref IS NULL OR decision.subject_digest<>director.council_subject_digest
 OR decision.decision_digest<>director.council_decision_digest OR decision.outcome='accepted'))`},
	}
	for _, check := range checks {
		var count int
		if err := tx.QueryRowContext(ctx, check.query).Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return errors.New(check.code)
		}
	}
	return nil
}
