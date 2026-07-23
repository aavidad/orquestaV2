package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func (repository *Repository) OpenCouncilRound(
	ctx context.Context, state application.OpenCouncilRoundState,
) (application.CouncilRoundRecord, bool, error) {
	tx, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.CouncilRoundRecord{}, false, err
	}
	defer tx.Rollback()
	now, err := repository.transactionTime()
	if err != nil {
		return application.CouncilRoundRecord{}, false, err
	}
	if state.OperationAt.IsZero() || state.OperationAt.After(now) {
		return application.CouncilRoundRecord{}, false, invalid(errors.New("sqlite.council_operation_time_invalid"))
	}
	record, created, err := openCouncilRoundTx(ctx, tx, state, now)
	if err != nil {
		return application.CouncilRoundRecord{}, false, err
	}
	if err := commit(tx); err != nil {
		return application.CouncilRoundRecord{}, false, err
	}
	return record, created, nil
}

func openCouncilRoundTx(
	ctx context.Context, tx *sql.Tx, state application.OpenCouncilRoundState, now time.Time,
) (application.CouncilRoundRecord, bool, error) {
	_ = now
	current, err := readGoalRecord(ctx, tx, state.GoalRef.String())
	if err != nil {
		return application.CouncilRoundRecord{}, false, err
	}
	item, found := current.Goal.WorkItem(state.Round.WorkItemRef)
	policy, hasPolicy := item.CouncilPolicy()
	if !found || current.Goal.Project() != state.ProjectRef || current.Goal.Revision() != state.ExpectedGoalRevision ||
		item.Revision() != state.ExpectedItemRevision || state.Round.GoalRef != state.GoalRef ||
		state.Round.Subject.GoalRef != state.GoalRef.String() ||
		state.Round.Subject.WorkItemRef != state.Round.WorkItemRef.String() ||
		state.Round.SubjectDigest != application.CouncilSubjectDigest(state.Round.Subject.Digest()) ||
		!hasPolicy || policy != state.Round.Subject.Policy || state.Round.Subject.Policy == council.PolicySkipByOperator ||
		state.Round.OpenedBy != state.PrincipalRef || state.Round.OpenedAt.IsZero() ||
		state.Round.RequestRef != state.RequestRef || state.Round.RequestFingerprint != state.RequestFingerprint ||
		state.Round.AuthorizationReceiptRef != state.AuthorizationReceipt.Ref() ||
		!validText(state.RequestRef) || !validCanonicalHash(state.RequestFingerprint) ||
		!validText(state.Round.Ref) || !validText(state.Round.IdempotencyKey) {
		return application.CouncilRoundRecord{}, false, conflict(errors.New("sqlite.council_round_frontier_invalid"))
	}
	if _, err := council.NewSubject(state.Round.Subject); err != nil {
		return application.CouncilRoundRecord{}, false, invalid(err)
	}
	if err := application.ValidatePersistedCouncilSubject(current, state.Round.Subject); err != nil {
		return application.CouncilRoundRecord{}, false, conflict(err)
	}
	if err := requireCouncilRoundAuthority(ctx, tx, current, state); err != nil {
		return application.CouncilRoundRecord{}, false, err
	}
	for _, prior := range current.CouncilRounds {
		if prior.SubjectDigest != state.Round.SubjectDigest {
			continue
		}
		if !reflect.DeepEqual(prior, state.Round) {
			return application.CouncilRoundRecord{}, false, conflict(errors.New("sqlite.council_round_replay_conflict"))
		}
		return prior, false, nil
	}
	if len(councilFactsFor(current.CouncilFacts, state.Round.SubjectDigest)) != 0 ||
		councilSkipFor(current.CouncilSkips, state.Round.SubjectDigest) != nil ||
		!validCouncilLaunchSet(state.Round, state.Executions, state.Actions) {
		return application.CouncilRoundRecord{}, false, conflict(errors.New("sqlite.council_round_existing_resolution"))
	}
	if err := insertCouncilRound(ctx, tx, state.Round); err != nil {
		return application.CouncilRoundRecord{}, false, err
	}
	if err := insertScheduled(ctx, tx, state.Executions, state.Actions); err != nil {
		return application.CouncilRoundRecord{}, false, err
	}
	if err := insertEvents(ctx, tx, state.Events); err != nil {
		return application.CouncilRoundRecord{}, false, err
	}
	return state.Round, true, nil
}

func requireCouncilRoundAuthority(
	ctx context.Context, tx *sql.Tx, current application.GoalRecord, state application.OpenCouncilRoundState,
) error {
	switch state.Round.Opener {
	case application.CouncilRoundOpenerAuto:
		if state.LeaseToken != "" || state.LeaseFence != 0 || state.Round.DirectorFence != 0 {
			return conflict(errors.New("sqlite.council_auto_authority_invalid"))
		}
		for _, authority := range current.WorkItemAuthorities {
			if authority.WorkItemRef == state.Round.WorkItemRef && authority.PrincipalRef == state.PrincipalRef &&
				sameAuthorizationReceipt(authority.AuthorizationReceipt, state.AuthorizationReceipt) {
				return nil
			}
		}
		return conflict(errors.New("sqlite.council_auto_authority_invalid"))
	case application.CouncilRoundOpenerDirector:
		if state.LeaseToken == "" || state.LeaseFence == 0 || state.Round.DirectorFence != state.LeaseFence {
			return conflict(errors.New("sqlite.council_director_authority_invalid"))
		}
		if _, err := requirePersistedAuthorization(ctx, tx, state.AuthorizationReceipt, state.PrincipalRef,
			state.ProjectRef, identity.PermissionGoalsDirect, state.GoalRef.String()); err != nil {
			return err
		}
		var count int
		err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM director_leases
WHERE goal_ref=? AND project_ref=? AND principal_ref=? AND token=? AND fence=? AND lease_until>?`,
			state.GoalRef.String(), state.ProjectRef.String(), state.PrincipalRef.String(), state.LeaseToken,
			int64(state.LeaseFence), requiredTime(state.OperationAt)).Scan(&count)
		if err != nil {
			return mapDatabaseError(err)
		}
		if count != 1 {
			return conflict(errors.New("sqlite.council_director_lease_conflict"))
		}
		return nil
	default:
		return invalid(errors.New("sqlite.council_round_opener_invalid"))
	}
}

func (repository *Repository) RecordCouncilSkip(
	ctx context.Context, state application.CouncilSkipState,
) (application.CouncilSkipRecord, bool, error) {
	tx, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.CouncilSkipRecord{}, false, err
	}
	defer tx.Rollback()
	now, err := repository.transactionTime()
	if err != nil {
		return application.CouncilSkipRecord{}, false, err
	}
	if state.OperationAt.IsZero() || state.OperationAt.After(now) {
		return application.CouncilSkipRecord{}, false, invalid(errors.New("sqlite.council_operation_time_invalid"))
	}
	if _, err := requirePersistedAuthorization(ctx, tx, state.AuthorizationReceipt, state.PrincipalRef,
		state.ProjectRef, identity.PermissionCouncilSkip, state.GoalRef.String()); err != nil {
		return application.CouncilSkipRecord{}, false, err
	}
	if state.AuthorizationReceipt.Decision().Request().Principal().Kind != identity.PrincipalKindHuman {
		return application.CouncilSkipRecord{}, false, conflict(errors.New("sqlite.council_skip_human_required"))
	}
	itemRef, err := goal.NewWorkItemRef(state.RoundSubject.WorkItemRef)
	if err != nil {
		return application.CouncilSkipRecord{}, false, invalid(err)
	}
	current, err := requireCouncilMutationFrontier(ctx, tx, state.GoalRef,
		itemRef, state.ExpectedGoalRevision, state.ExpectedItemRevision)
	if err != nil {
		return application.CouncilSkipRecord{}, false, err
	}
	item, found := current.Goal.WorkItem(itemRef)
	policy, hasPolicy := item.CouncilPolicy()
	if !found || current.Goal.Project() != state.ProjectRef || !hasPolicy || policy != council.PolicySkipByOperator ||
		state.Record.Subject != state.RoundSubject || state.Record.Skip != state.Skip ||
		state.Record.SubjectDigest != application.CouncilSubjectDigest(state.RoundSubject.Digest()) ||
		state.Record.SkipDigest != application.CouncilSubjectDigest(state.Skip.Digest()) ||
		state.Record.RequestRef != state.RequestRef || state.Record.RequestFingerprint != state.RequestFingerprint ||
		state.Record.AuthorizationReceiptRef != state.AuthorizationReceipt.Ref() ||
		state.PrincipalRef.String() != state.Skip.PrincipalRef ||
		!validText(state.Record.Ref) || !validText(state.RequestRef) ||
		!validCanonicalHash(state.RequestFingerprint) {
		return application.CouncilSkipRecord{}, false, invalid(errors.New("sqlite.council_skip_invalid"))
	}
	if _, err := council.NewSkip(state.RoundSubject, state.Skip); err != nil {
		return application.CouncilSkipRecord{}, false, invalid(err)
	}
	if err := application.ValidatePersistedCouncilSubject(current, state.RoundSubject); err != nil {
		return application.CouncilSkipRecord{}, false, conflict(err)
	}
	for _, prior := range current.CouncilSkips {
		if prior.SubjectDigest != state.Record.SubjectDigest {
			continue
		}
		if !reflect.DeepEqual(prior, state.Record) {
			return application.CouncilSkipRecord{}, false, conflict(errors.New("sqlite.council_skip_replay_conflict"))
		}
		if err := commit(tx); err != nil {
			return application.CouncilSkipRecord{}, false, err
		}
		return prior, false, nil
	}
	if _, found := councilRoundFor(current.CouncilRounds, state.Record.SubjectDigest); found ||
		len(councilFactsFor(current.CouncilFacts, state.Record.SubjectDigest)) != 0 {
		return application.CouncilSkipRecord{}, false, conflict(errors.New("sqlite.council_skip_after_round"))
	}
	if err := insertCouncilSkip(ctx, tx, state.Record); err != nil {
		return application.CouncilSkipRecord{}, false, err
	}
	if err := insertEvents(ctx, tx, state.Events); err != nil {
		return application.CouncilSkipRecord{}, false, err
	}
	if err := commit(tx); err != nil {
		return application.CouncilSkipRecord{}, false, err
	}
	return state.Record, true, nil
}

func insertCouncilRound(ctx context.Context, tx *sql.Tx, record application.CouncilRoundRecord) error {
	subject := record.Subject
	_, err := tx.ExecContext(ctx, `INSERT INTO council_rounds(
ref,goal_ref,work_item_ref,change_set_ref,project_ref,subject_digest,review_subject_digest,review_gate_digest,
policy,spec_hash,plan_generation,work_item_generation,app_spec_generation,opened_by_ref,opener,director_fence,
request_ref,request_fingerprint,authorization_receipt_ref,opened_at,idempotency_key)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		record.Ref, record.GoalRef.String(), record.WorkItemRef.String(), record.ChangeSetRef, subject.ProjectRef,
		string(record.SubjectDigest), subject.ReviewSubjectDigest, subject.ReviewGateDigest, string(subject.Policy),
		subject.SpecHash, int64(subject.PlanGeneration), int64(subject.WorkItemGeneration),
		int64(subject.AppSpecGeneration), record.OpenedBy.String(), string(record.Opener), int64(record.DirectorFence),
		record.RequestRef, record.RequestFingerprint, record.AuthorizationReceiptRef,
		requiredTime(record.OpenedAt), record.IdempotencyKey)
	return mapDatabaseError(err)
}

func insertCouncilSkip(ctx context.Context, tx *sql.Tx, record application.CouncilSkipRecord) error {
	subject, skip := record.Subject, record.Skip
	_, err := tx.ExecContext(ctx, `INSERT INTO council_skips(
ref,goal_ref,work_item_ref,change_set_ref,project_ref,subject_digest,review_subject_digest,review_gate_digest,
policy,plan_generation,work_item_generation,app_spec_generation,principal_ref,reason,spec_hash,idempotency_key,
skip_digest,request_ref,request_fingerprint,authorization_receipt_ref,recorded_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		record.Ref, subject.GoalRef, subject.WorkItemRef, subject.ChangeSetRef, subject.ProjectRef,
		string(record.SubjectDigest), subject.ReviewSubjectDigest, subject.ReviewGateDigest, string(subject.Policy),
		int64(subject.PlanGeneration), int64(subject.WorkItemGeneration), int64(subject.AppSpecGeneration),
		skip.PrincipalRef, skip.Reason, skip.SpecHash, skip.IdempotencyKey, string(record.SkipDigest),
		record.RequestRef, record.RequestFingerprint, record.AuthorizationReceiptRef, requiredTime(record.RecordedAt))
	return mapDatabaseError(err)
}
