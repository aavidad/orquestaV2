package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

// V18 deliberately has no automatic legacy reviewer. An exact V17 PASS that
// is still awaiting integration must be drained under the V17 binary before
// schema upgrade. V18 never guesses, converts, or backfills review approval.
// This keeps pending, claimed and unknown-applied VCS frontiers intact instead
// of guessing that an external merge did not happen.
func validateV18LegacyReviewUpgradeSource(ctx context.Context, tx *sql.Tx) error {
	var pending, claimedOrUnknown, withoutAction int
	err := tx.QueryRowContext(ctx, `
SELECT
 COALESCE(SUM(CASE WHEN action.ref IS NOT NULL AND action.completed_at IS NULL
   AND action.retired_at IS NULL AND action.quarantined_at IS NULL
   AND action.claim_token IS NULL AND NOT EXISTS(
     SELECT 1 FROM effect_attempts attempt WHERE attempt.action_ref=action.ref)
   THEN 1 ELSE 0 END),0),
 COALESCE(SUM(CASE WHEN action.ref IS NOT NULL AND NOT (
   action.completed_at IS NULL AND action.retired_at IS NULL AND action.quarantined_at IS NULL
   AND action.claim_token IS NULL AND NOT EXISTS(
     SELECT 1 FROM effect_attempts attempt WHERE attempt.action_ref=action.ref))
   AND NOT (action.completed_at IS NOT NULL AND action.quarantined_at IS NULL AND EXISTS(
     SELECT 1 FROM integration_receipts integrated
     WHERE integrated.change_set_ref=change_set.ref AND integrated.status='integrated'))
   THEN 1 ELSE 0 END),0),
 COALESCE(SUM(CASE WHEN action.ref IS NULL THEN 1 ELSE 0 END),0)
FROM executions execution
JOIN work_items item ON item.goal_ref=execution.goal_ref AND item.ref=execution.work_item_ref
JOIN change_sets change_set ON change_set.goal_ref=execution.goal_ref
 AND change_set.work_item_ref=execution.work_item_ref AND change_set.execution_ref=execution.ref
JOIN attestations pass ON pass.goal_ref=execution.goal_ref AND pass.work_item_ref=execution.work_item_ref
 AND pass.execution_ref=execution.ref AND pass.change_set_ref=change_set.ref
 AND pass.kind='required_tests' AND pass.verdict='passed'
LEFT JOIN outbox action ON action.goal_ref=execution.goal_ref AND action.work_item_ref=execution.work_item_ref
 AND action.execution_ref=execution.ref AND action.change_ref=change_set.ref AND action.kind='integrate_change'
WHERE execution.state='awaiting_integration'`,
	).Scan(&pending, &claimedOrUnknown, &withoutAction)
	if err != nil {
		return err
	}
	if pending+claimedOrUnknown+withoutAction != 0 {
		return fmt.Errorf("sqlite.v18_upgrade_requires_v17_integration_drain:pending=%d:claimed_or_unknown=%d:without_action=%d",
			pending, claimedOrUnknown, withoutAction)
	}
	return nil
}

func validateRecoveryV18Reviews(ctx context.Context, tx *sql.Tx) error {
	persisted, err := sqliteTableHasColumn(ctx, tx, "review_records", "subject_digest")
	if err != nil || !persisted {
		return mapDatabaseError(err)
	}
	queries := []struct{ code, query string }{
		{"sqlite.recovery_v18_execution_purpose_invalid", `SELECT COUNT(*) FROM executions e WHERE
 (e.purpose IN ('work','author') AND e.review_subject_digest<>'') OR
 (e.purpose IN ('primary_review','adversarial_review') AND
  (length(e.review_subject_digest)<>71 OR substr(e.review_subject_digest,1,7)<>'sha256:'))`},
		{"sqlite.recovery_v18_review_scope_invalid", `SELECT COUNT(*) FROM review_records r
 LEFT JOIN executions e ON e.goal_ref=r.goal_ref AND e.work_item_ref=r.work_item_ref AND e.ref=r.reviewer_execution_ref
 LEFT JOIN change_sets c ON c.ref=r.change_set_ref
 LEFT JOIN artifact_occurrences a ON a.goal_ref=r.goal_ref AND a.artifact_ref=r.assessment_artifact_ref
 LEFT JOIN artifacts artifact ON artifact.goal_ref=r.goal_ref AND artifact.ref=r.assessment_artifact_ref
 LEFT JOIN effect_intents intent ON intent.ref=e.effect_intent_ref
 LEFT JOIN work_item_authorities authority ON authority.goal_ref=r.goal_ref AND authority.work_item_ref=r.work_item_ref
 LEFT JOIN goals g ON g.ref=r.goal_ref
 LEFT JOIN attestations pass ON pass.goal_ref=r.goal_ref AND pass.work_item_ref=r.work_item_ref
  AND pass.execution_ref=c.execution_ref AND pass.change_set_ref=c.ref
  AND pass.kind='required_tests' AND pass.verdict='passed'
 WHERE e.ref IS NULL OR c.ref IS NULL OR a.occurrence_ref IS NULL OR artifact.ref IS NULL
  OR intent.ref IS NULL OR authority.work_item_ref IS NULL OR pass.ref IS NULL OR a.kind<>'review_assessment'
  OR r.ref<>'review:' || e.ref OR r.recorded_at<>e.finished_at OR r.recorded_at<>a.created_at
  OR e.review_subject_digest<>r.subject_digest OR e.attempt_no<>r.reviewer_execution_attempt
  OR e.launch_receipt_ref<>r.launch_receipt_ref OR e.agent_ref<>r.agent_ref OR e.external_ref<>r.external_ref
  OR e.state<>'succeeded' OR c.goal_ref<>r.goal_ref OR c.work_item_ref<>r.work_item_ref
  OR a.work_item_ref<>r.work_item_ref OR a.execution_ref<>e.ref OR a.execution_attempt<>e.attempt_no
  OR a.plan_generation<>e.plan_generation OR a.work_item_generation<>pass.work_item_generation
  OR a.app_spec_generation<>e.app_spec_generation OR a.spec_hash<>e.spec_hash
  OR artifact.digest<>r.assessment_digest
  OR intent.action_kind<>'launch_agent' OR intent.kind<>'agent_launch' OR intent.project_ref<>g.project_ref
  OR intent.goal_ref<>r.goal_ref OR intent.work_item_ref<>r.work_item_ref OR intent.execution_ref<>e.ref
  OR intent.plan_generation<>e.plan_generation OR intent.app_spec_generation<>e.app_spec_generation
  OR intent.spec_hash<>e.spec_hash OR intent.proposed_by_ref<>r.principal_ref
  OR authority.principal_ref<>r.principal_ref OR authority.permission<>intent.permission
  OR authority.authorization_receipt_ref<>intent.authority_receipt_ref
  OR (r.role='primary' AND e.purpose<>'primary_review')
  OR (r.role='adversarial' AND e.purpose<>'adversarial_review')`},
	}
	for _, check := range queries {
		var count int
		if err := tx.QueryRowContext(ctx, check.query).Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return errors.New(check.code)
		}
	}
	return validateRecoveryV18IntegrationGates(ctx, tx)
}

func validateRecoveryV18IntegrationGates(ctx context.Context, tx *sql.Tx) error {
	councilColumns, err := sqliteTableHasColumn(ctx, tx, "outbox", "council_subject_digest")
	if err != nil {
		return err
	}
	query := `
SELECT ref,goal_ref,work_item_ref,execution_ref,change_ref,expected_target_oid,
 review_gate_digest,plan_generation,work_item_generation
FROM outbox WHERE kind='integrate_change' AND completed_at IS NULL
 AND retired_at IS NULL AND quarantined_at IS NULL ORDER BY ref`
	if councilColumns {
		query = `
SELECT ref,goal_ref,work_item_ref,execution_ref,change_ref,expected_target_oid,
 review_gate_digest,plan_generation,work_item_generation,council_subject_digest,
 council_decision_ref,council_decision_digest,council_skip_ref,council_skip_digest
FROM outbox WHERE kind='integrate_change' AND completed_at IS NULL
 AND retired_at IS NULL AND quarantined_at IS NULL ORDER BY ref`
	}
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	type candidate struct {
		action application.ActionRecord
		goal   string
		item   string
		exec   string
		change string
	}
	var candidates []candidate
	for rows.Next() {
		var value candidate
		var plan, itemGeneration int64
		destinations := []any{&value.action.Ref, &value.goal, &value.item, &value.exec,
			&value.change, &value.action.ExpectedTargetOID, &value.action.ReviewGateDigest, &plan, &itemGeneration}
		var subject string
		var decisionRef, decisionDigest, skipRef, skipDigest sql.NullString
		if councilColumns {
			destinations = append(destinations, &subject, &decisionRef, &decisionDigest, &skipRef, &skipDigest)
		}
		if err := rows.Scan(destinations...); err != nil {
			_ = rows.Close()
			return err
		}
		if councilColumns {
			value.action.CouncilResolution, err = restoreCouncilResolution(
				subject, decisionRef, decisionDigest, skipRef, skipDigest)
			if err != nil {
				_ = rows.Close()
				return err
			}
		}
		value.action.Kind = application.ActionIntegrateChange
		value.action.PlanGeneration = goal.PlanGeneration(plan)
		value.action.WorkItemGeneration = goal.Revision(itemGeneration)
		candidates = append(candidates, value)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, candidate := range candidates {
		var refErr error
		if candidate.action.GoalRef, refErr = goal.NewGoalRef(candidate.goal); refErr != nil {
			return refErr
		}
		if candidate.action.WorkItemRef, refErr = goal.NewWorkItemRef(candidate.item); refErr != nil {
			return refErr
		}
		if candidate.action.ExecutionRef, refErr = goal.NewExecutionRef(candidate.exec); refErr != nil {
			return refErr
		}
		if candidate.action.ChangeRef, refErr = ports.NewChangeSetRef(candidate.change); refErr != nil {
			return refErr
		}
		record, readErr := readGoalRecord(ctx, tx, candidate.goal)
		if readErr != nil || application.ValidatePersistedIntegrationReviewGate(record, candidate.action) != nil {
			return errors.New("sqlite.recovery_v18_integration_review_gate_invalid")
		}
	}
	return nil
}
