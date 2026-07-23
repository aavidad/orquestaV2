package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func (r *Repository) AdmitIntegration(ctx context.Context, s application.AdmitIntegrationState) (application.ActionRecord, bool, error) {
	if err := validateIntegrationAdmissionState(s); err != nil {
		return application.ActionRecord{}, false, invalid(err)
	}
	tx, err := beginTransaction(ctx, r)
	if err != nil {
		return application.ActionRecord{}, false, err
	}
	defer tx.Rollback()
	now, err := r.transactionTime()
	if err != nil {
		return application.ActionRecord{}, false, err
	}
	if s.OperationAt.After(now) {
		return application.ActionRecord{}, false, invalid(errors.New("sqlite.integration_admission_time_future"))
	}
	persisted, persistedFingerprint, found, err := readPersistedIntegrationAction(ctx, tx, s)
	if err != nil {
		return application.ActionRecord{}, false, err
	}
	if found {
		if !integrationAdmissionReplayMatches(persisted, persistedFingerprint, s) {
			return application.ActionRecord{}, false, conflict(errors.New("sqlite.integration_admission_replay_conflict"))
		}
		if _, err := requirePersistedAuthorization(
			ctx, tx, s.AuthorizationReceipt, s.PrincipalRef, s.ProjectRef,
			identity.PermissionChangesIntegrate, s.GoalRef.String(),
		); err != nil {
			return application.ActionRecord{}, false, err
		}
		if err := requireIntegrationPassEvidence(ctx, tx, s); err != nil {
			return application.ActionRecord{}, false, err
		}
		record, readErr := readGoalRecord(ctx, tx, s.GoalRef.String())
		if readErr != nil || application.ValidatePersistedIntegrationReviewGate(record, persisted) != nil {
			return application.ActionRecord{}, false, conflict(errors.New("sqlite.integration_council_resolution_invalid"))
		}
		if err = commit(tx); err != nil {
			return application.ActionRecord{}, false, err
		}
		return persisted, false, nil
	}
	if _, err := requirePersistedAuthorization(
		ctx, tx, s.AuthorizationReceipt, s.PrincipalRef, s.ProjectRef,
		identity.PermissionChangesIntegrate, s.GoalRef.String(),
	); err != nil {
		return application.ActionRecord{}, false, err
	}
	if err := requireIntegrationPassEvidence(ctx, tx, s); err != nil {
		return application.ActionRecord{}, false, err
	}
	if err := requireIntegrationAdmissionFrontier(ctx, tx, s); err != nil {
		return application.ActionRecord{}, false, err
	}
	if err := insertEffectAdmission(ctx, tx, s.Action); err != nil {
		return application.ActionRecord{}, false, err
	}
	if err = insertIntegrationAction(ctx, tx, s); err != nil {
		return application.ActionRecord{}, false, err
	}
	record, readErr := readGoalRecord(ctx, tx, s.GoalRef.String())
	if readErr != nil || application.ValidatePersistedIntegrationReviewGate(record, s.Action) != nil {
		return application.ActionRecord{}, false, conflict(errors.New("sqlite.integration_council_resolution_invalid"))
	}
	if err = commit(tx); err != nil {
		return application.ActionRecord{}, false, err
	}
	return s.Action, true, nil
}

func requireIntegrationPassEvidence(ctx context.Context, tx *sql.Tx, s application.AdmitIntegrationState) error {
	record, err := readGoalRecord(ctx, tx, s.GoalRef.String())
	if err != nil {
		return err
	}
	count := 0
	for _, evidence := range record.Attestations {
		if evidence.Kind == application.AttestationKindRequiredTests && evidence.Verdict == application.AttestationVerdictPassed &&
			evidence.WorkItemRef == s.Action.WorkItemRef && evidence.ExecutionRef == s.Action.ExecutionRef &&
			evidence.ChangeSetRef == s.ChangeRef && evidence.PlanGeneration == s.Action.PlanGeneration &&
			evidence.WorkItemGeneration == s.ExpectedItemRevision {
			count++
		}
	}
	if count != 1 {
		return conflict(errors.New("sqlite.integration_required_tests_pass_missing"))
	}
	return nil
}
func validateIntegrationAdmissionState(s application.AdmitIntegrationState) error {
	action, intent := s.Action, s.Action.EffectIntent
	if !validText(s.RequestRef) || !validCanonicalHash(s.RequestFingerprint) ||
		s.PrincipalRef.String() == "" || s.ProjectRef.String() == "" || s.GoalRef.String() == "" ||
		s.ChangeRef.String() == "" || s.ExpectedGoalRevision == 0 || s.ExpectedItemRevision == 0 ||
		uint64(s.ExpectedGoalRevision) > maxSQLiteInteger || uint64(s.ExpectedItemRevision) > maxSQLiteInteger ||
		s.OperationAt.IsZero() || validateAction(action) != nil {
		return errors.New("sqlite.integration_admission_invalid")
	}
	if action.Kind != application.ActionIntegrateChange ||
		action.GoalRef != s.GoalRef || action.ChangeRef != s.ChangeRef ||
		action.WorkItemGeneration != s.ExpectedItemRevision || action.EffectIntentRef != intent.Ref ||
		!reflect.DeepEqual(action.CouncilResolution, s.CouncilResolution) ||
		!reflect.DeepEqual(intent.CouncilResolution, s.CouncilResolution) ||
		!action.AvailableAt.Equal(intent.CreatedAt) || s.OperationAt.Before(action.AvailableAt) ||
		intent.RequestRef != s.RequestRef || intent.RequestFingerprint != s.RequestFingerprint ||
		intent.ActionRef != action.Ref || intent.ActionKind != application.ActionIntegrateChange ||
		intent.Kind != application.EffectKindIntegrateChange || intent.Subject.ProjectRef != s.ProjectRef ||
		intent.Subject.GoalRef != s.GoalRef || intent.Subject.WorkItemRef != action.WorkItemRef ||
		intent.Subject.ExecutionRef != action.ExecutionRef || intent.Subject.PlanGeneration != action.PlanGeneration ||
		intent.ProposedBy != s.PrincipalRef || intent.Permission != identity.PermissionChangesIntegrate ||
		!sameAuthorizationReceipt(intent.Authority, s.AuthorizationReceipt) ||
		s.OperationAt.Before(s.AuthorizationReceipt.RecordedAt()) || application.ValidateEffectIntent(intent) != nil ||
		action.EffectApproval == nil || application.ValidateEffectApproval(intent, *action.EffectApproval) != nil ||
		action.EffectApproval.Source != application.EffectApprovalSourceIntegrationDecision {
		return errors.New("sqlite.integration_admission_causality_invalid")
	}
	return nil
}

func readPersistedIntegrationAction(
	ctx context.Context, source queryer, state application.AdmitIntegrationState,
) (application.ActionRecord, string, bool, error) {
	var action application.ActionRecord
	var kind, goalValue, itemValue, executionValue, changeValue, intentRef, fingerprint string
	var councilSubject string
	var councilDecisionRef, councilDecisionDigest, councilSkipRef, councilSkipDigest sql.NullString
	var planGeneration, itemGeneration, availableAt int64
	err := source.QueryRowContext(ctx, `
SELECT action.ref,action.kind,action.goal_ref,action.work_item_ref,action.execution_ref,
       action.change_ref,action.expected_target_oid,action.plan_generation,
       action.work_item_generation,action.available_at,action.effect_intent_ref,action.review_gate_digest,
       action.admission_request_fingerprint,action.council_subject_digest,action.council_decision_ref,
       action.council_decision_digest,action.council_skip_ref,action.council_skip_digest
FROM outbox action
JOIN effect_intents intent ON intent.ref=action.effect_intent_ref
WHERE action.kind='integrate_change' AND action.admission_request_ref=?
 AND action.goal_ref=? AND intent.proposed_by_ref=? AND intent.project_ref=?`,
		state.RequestRef, state.GoalRef.String(), state.PrincipalRef.String(), state.ProjectRef.String()).Scan(
		&action.Ref, &kind, &goalValue, &itemValue, &executionValue, &changeValue,
		&action.ExpectedTargetOID, &planGeneration, &itemGeneration, &availableAt,
		&intentRef, &action.ReviewGateDigest, &fingerprint, &councilSubject, &councilDecisionRef,
		&councilDecisionDigest, &councilSkipRef, &councilSkipDigest,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return application.ActionRecord{}, "", false, nil
	}
	if err != nil {
		return application.ActionRecord{}, "", false, mapDatabaseError(err)
	}
	if planGeneration <= 0 || itemGeneration <= 0 {
		return application.ActionRecord{}, "", false, invalid(errors.New("sqlite.integration_action_generation_invalid"))
	}
	action.Kind = application.ActionKind(kind)
	var refErr error
	if action.GoalRef, refErr = goal.NewGoalRef(goalValue); refErr != nil {
		return application.ActionRecord{}, "", false, invalid(refErr)
	}
	if action.WorkItemRef, refErr = goal.NewWorkItemRef(itemValue); refErr != nil {
		return application.ActionRecord{}, "", false, invalid(refErr)
	}
	if action.ExecutionRef, refErr = goal.NewExecutionRef(executionValue); refErr != nil {
		return application.ActionRecord{}, "", false, invalid(refErr)
	}
	if action.ChangeRef, refErr = ports.NewChangeSetRef(changeValue); refErr != nil {
		return application.ActionRecord{}, "", false, invalid(refErr)
	}
	action.PlanGeneration = goal.PlanGeneration(planGeneration)
	action.WorkItemGeneration = goal.Revision(itemGeneration)
	action.AvailableAt = time.Unix(0, availableAt).UTC()
	action.EffectIntentRef = intentRef
	action.CouncilResolution, err = restoreCouncilResolution(
		councilSubject, councilDecisionRef, councilDecisionDigest, councilSkipRef, councilSkipDigest,
	)
	if err != nil {
		return application.ActionRecord{}, "", false, err
	}
	action.EffectIntent, err = readEffectIntent(ctx, source, intentRef)
	if err != nil {
		return application.ActionRecord{}, "", false, err
	}
	approvalRefs, err := readSingleColumn(ctx, source, `SELECT ref FROM effect_approvals WHERE intent_ref=? ORDER BY decided_at,ref`, intentRef)
	if err != nil {
		return application.ActionRecord{}, "", false, err
	}
	if len(approvalRefs) != 1 {
		return application.ActionRecord{}, "", false, invalid(errors.New("sqlite.integration_action_approval_invalid"))
	}
	approval, found, err := readEffectApprovalByRef(ctx, source, approvalRefs[0])
	if err != nil {
		return application.ActionRecord{}, "", false, err
	}
	if !found || application.ValidateEffectApproval(action.EffectIntent, approval) != nil {
		return application.ActionRecord{}, "", false, invalid(errors.New("sqlite.integration_action_approval_invalid"))
	}
	action.EffectApproval = &approval
	if validateAction(action) != nil || action.EffectIntent.ActionRef != action.Ref ||
		action.EffectIntent.ActionKind != action.Kind || action.EffectIntentRef != action.EffectIntent.Ref {
		return application.ActionRecord{}, "", false, invalid(errors.New("sqlite.integration_action_invalid"))
	}
	return action, fingerprint, true, nil
}

func integrationAdmissionReplayMatches(
	persisted application.ActionRecord, persistedFingerprint string, s application.AdmitIntegrationState,
) bool {
	if persistedFingerprint != s.RequestFingerprint || persisted.Ref != s.Action.Ref ||
		persisted.Kind != application.ActionIntegrateChange || persisted.GoalRef != s.GoalRef ||
		persisted.WorkItemRef != s.Action.WorkItemRef || persisted.ExecutionRef != s.Action.ExecutionRef ||
		persisted.ChangeRef != s.ChangeRef || persisted.ExpectedTargetOID != s.Action.ExpectedTargetOID ||
		persisted.ReviewGateDigest != s.Action.ReviewGateDigest ||
		!reflect.DeepEqual(persisted.CouncilResolution, s.Action.CouncilResolution) ||
		persisted.PlanGeneration != s.Action.PlanGeneration ||
		!reflect.DeepEqual(persisted.EffectIntent, s.Action.EffectIntent) ||
		persisted.EffectApproval == nil || s.Action.EffectApproval == nil ||
		*persisted.EffectApproval != *s.Action.EffectApproval {
		return false
	}
	return persisted.EffectIntent.RequestRef == s.RequestRef &&
		persisted.EffectIntent.RequestFingerprint == s.RequestFingerprint &&
		persisted.EffectIntent.ProposedBy == s.PrincipalRef &&
		persisted.EffectIntent.Subject.ProjectRef == s.ProjectRef &&
		persisted.EffectIntent.Subject.GoalRef == s.GoalRef &&
		sameAuthorizationReceipt(persisted.EffectIntent.Authority, s.AuthorizationReceipt)
}

func requireIntegrationAdmissionFrontier(ctx context.Context, tx *sql.Tx, s application.AdmitIntegrationState) error {
	action, intent := s.Action, s.Action.EffectIntent
	var count int
	err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM change_sets change_set
JOIN workspace_bindings binding ON binding.ref=change_set.workspace_binding_ref
JOIN goals goal ON goal.ref=change_set.goal_ref AND goal.project_ref=change_set.project_ref
JOIN work_items item ON item.goal_ref=change_set.goal_ref AND item.ref=change_set.work_item_ref
JOIN executions execution ON execution.goal_ref=change_set.goal_ref
 AND execution.work_item_ref=change_set.work_item_ref AND execution.ref=change_set.execution_ref
WHERE change_set.ref=? AND change_set.project_ref=? AND change_set.goal_ref=?
 AND change_set.work_item_ref=? AND change_set.execution_ref=?
 AND change_set.ref=? AND change_set.plan_generation=?
 AND change_set.app_spec_generation=? AND change_set.app_spec_hash=?
 AND binding.execution_ref=execution.ref AND binding.repository_ref=change_set.repository_ref
 AND binding.goal_ref=change_set.goal_ref AND binding.work_item_ref=change_set.work_item_ref
 AND goal.revision=? AND goal.state='running' AND goal.actor_ref=?
 AND item.revision=? AND item.state='running' AND item.execution_ref=execution.ref
 AND execution.state='awaiting_integration' AND execution.plan_generation=?
 AND execution.app_spec_generation=? AND execution.spec_hash=?
 AND execution.repository_ref=change_set.repository_ref
 AND execution.execution_workspace_ref=change_set.workspace_binding_ref
 AND change_set.committed_at<=?
 AND length(?)=CASE change_set.object_format WHEN 'sha1' THEN 40 ELSE 64 END
 AND ? NOT GLOB '*[^0-9a-f]*'
 AND NOT EXISTS (SELECT 1 FROM integration_receipts receipt
                 WHERE receipt.change_set_ref=change_set.ref AND receipt.status='integrated')
 AND NOT EXISTS (SELECT 1 FROM outbox active
                 WHERE active.kind='integrate_change' AND active.change_ref=change_set.ref
                   AND active.completed_at IS NULL AND active.retired_at IS NULL
                   AND active.quarantined_at IS NULL)`,
		s.ChangeRef.String(), s.ProjectRef.String(), s.GoalRef.String(), action.WorkItemRef.String(),
		action.ExecutionRef.String(), action.ChangeRef.String(), int64(action.PlanGeneration),
		int64(intent.Subject.AppSpecGeneration), intent.Subject.SpecHash,
		int64(s.ExpectedGoalRevision), intent.Subject.ActorRef.String(), int64(s.ExpectedItemRevision),
		int64(action.PlanGeneration), int64(intent.Subject.AppSpecGeneration), intent.Subject.SpecHash,
		requiredTime(action.AvailableAt), action.ExpectedTargetOID, action.ExpectedTargetOID,
	).Scan(&count)
	if err != nil {
		return mapDatabaseError(err)
	}
	if count != 1 {
		return conflict(errors.New("sqlite.integration_admission_frontier_conflict"))
	}
	return nil
}

func insertIntegrationAction(ctx context.Context, tx *sql.Tx, s application.AdmitIntegrationState) error {
	action := s.Action
	subject, decisionRef, decisionDigest, skipRef, skipDigest := storedCouncilResolution(action.CouncilResolution)
	resolutionKind := "accepted_round"
	if action.CouncilResolution != nil && action.CouncilResolution.SkipRef != "" {
		resolutionKind = "skip"
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO outbox(
 ref,kind,goal_ref,work_item_ref,execution_ref,control_ref,change_ref,expected_target_oid,
 admission_request_ref,admission_request_fingerprint,plan_generation,work_item_generation,
 available_at,governance_version,effect_intent_ref,review_gate_digest,council_subject_digest,
 council_resolution_kind,council_decision_ref,council_decision_digest,council_skip_ref,council_skip_digest
) VALUES(?,?,?,?,?,NULL,?,?,?,?,?,?,?,1,?,?,?,?,?,?,?,?)`,
		action.Ref, string(action.Kind), action.GoalRef.String(), action.WorkItemRef.String(),
		action.ExecutionRef.String(), action.ChangeRef.String(), action.ExpectedTargetOID,
		s.RequestRef, s.RequestFingerprint, int64(action.PlanGeneration),
		int64(action.WorkItemGeneration), requiredTime(action.AvailableAt), action.EffectIntentRef, action.ReviewGateDigest,
		subject, resolutionKind, decisionRef, decisionDigest, skipRef, skipDigest,
	)
	return mapDatabaseError(err)
}
