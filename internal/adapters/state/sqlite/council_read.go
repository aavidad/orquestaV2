package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type councilGoalRecords struct {
	rounds    []application.CouncilRoundRecord
	facts     []council.ContributionFact
	decisions []application.CouncilDecisionRecord
	skips     []application.CouncilSkipRecord
}

func readCouncilGoalRecords(ctx context.Context, source queryer, goalValue string) (councilGoalRecords, error) {
	persisted, err := sqliteTableHasColumn(ctx, source, "work_items", "council_policy")
	if err != nil || !persisted {
		return councilGoalRecords{}, mapDatabaseError(err)
	}
	var result councilGoalRecords
	result.rounds, err = readCouncilRounds(ctx, source, goalValue)
	if err == nil {
		result.facts, err = readCouncilFacts(ctx, source, goalValue)
	}
	if err == nil {
		result.decisions, err = readCouncilDecisions(ctx, source, goalValue, result.rounds, result.facts)
	}
	if err == nil {
		result.skips, err = readCouncilSkips(ctx, source, goalValue)
	}
	return result, err
}

func readCouncilRounds(
	ctx context.Context, source queryer, goalValue string,
) ([]application.CouncilRoundRecord, error) {
	rows, err := source.QueryContext(ctx, `SELECT ref,goal_ref,work_item_ref,change_set_ref,project_ref,
subject_digest,review_subject_digest,review_gate_digest,policy,spec_hash,plan_generation,
work_item_generation,app_spec_generation,opened_by_ref,opener,director_fence,request_ref,
request_fingerprint,authorization_receipt_ref,opened_at,idempotency_key
FROM council_rounds WHERE goal_ref=? ORDER BY opened_at,ref`, goalValue)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var records []application.CouncilRoundRecord
	for rows.Next() {
		var record application.CouncilRoundRecord
		var goalValue, itemValue, openedBy string
		var policy string
		var plan, itemGeneration, appSpec, fence, openedAt int64
		err := rows.Scan(&record.Ref, &goalValue, &itemValue, &record.ChangeSetRef, &record.Subject.ProjectRef,
			&record.SubjectDigest, &record.Subject.ReviewSubjectDigest, &record.Subject.ReviewGateDigest,
			&policy, &record.Subject.SpecHash, &plan, &itemGeneration, &appSpec, &openedBy, &record.Opener,
			&fence, &record.RequestRef, &record.RequestFingerprint, &record.AuthorizationReceiptRef,
			&openedAt, &record.IdempotencyKey)
		if err != nil {
			return nil, mapDatabaseError(err)
		}
		if record.GoalRef, err = goal.NewGoalRef(goalValue); err != nil {
			return nil, invalid(err)
		}
		if record.WorkItemRef, err = goal.NewWorkItemRef(itemValue); err != nil {
			return nil, invalid(err)
		}
		if record.OpenedBy, err = identity.NewPrincipalRef(openedBy); err != nil {
			return nil, invalid(err)
		}
		if plan <= 0 || itemGeneration <= 0 || appSpec <= 0 || fence < 0 {
			return nil, invalid(errors.New("sqlite.council_round_generation_invalid"))
		}
		record.Subject.Policy = council.Policy(policy)
		record.Subject.GoalRef, record.Subject.WorkItemRef, record.Subject.ChangeSetRef =
			goalValue, itemValue, record.ChangeSetRef
		record.Subject.PlanGeneration, record.Subject.WorkItemGeneration, record.Subject.AppSpecGeneration =
			uint64(plan), uint64(itemGeneration), uint64(appSpec)
		record.DirectorFence, record.OpenedAt = uint64(fence), time.Unix(0, openedAt).UTC()
		if _, err := council.NewSubject(record.Subject); err != nil ||
			record.SubjectDigest != application.CouncilSubjectDigest(record.Subject.Digest()) {
			return nil, invalid(errors.New("sqlite.council_round_subject_invalid"))
		}
		records = append(records, record)
	}
	return records, mapDatabaseError(rows.Err())
}

func readCouncilFacts(
	ctx context.Context, source queryer, goalValue string,
) ([]council.ContributionFact, error) {
	rows, err := source.QueryContext(ctx, `SELECT ref,council_subject_digest,role,ballot,contribution_schema,
body,evidence_json,contribution_digest,execution_ref,execution_attempt,launch_receipt_ref,external_ref,
artifact_ref,artifact_digest,idempotency_key,fact_digest
FROM council_facts WHERE goal_ref=? ORDER BY recorded_at,ref`, goalValue)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []council.ContributionFact
	for rows.Next() {
		var ref, contributionDigest, factDigest, evidenceJSON string
		var attempt int64
		var fact council.ContributionFact
		if err := rows.Scan(&ref, &fact.SubjectDigest, &fact.Role, &fact.Ballot,
			&fact.Contribution.Schema, &fact.Contribution.Body, &evidenceJSON, &contributionDigest,
			&fact.ExecutionRef, &attempt, &fact.LaunchReceiptRef, &fact.ExternalRef, &fact.ArtifactRef,
			&fact.ArtifactDigest, &fact.IdempotencyKey, &factDigest); err != nil {
			return nil, mapDatabaseError(err)
		}
		if attempt <= 0 || ref != "council-fact:"+fact.ExecutionRef {
			return nil, invalid(errors.New("sqlite.council_fact_identity_invalid"))
		}
		fact.ExecutionAttempt = uint64(attempt)
		fact.Contribution.SubjectDigest, fact.Contribution.Role, fact.Contribution.Ballot =
			fact.SubjectDigest, fact.Role, fact.Ballot
		if err := json.Unmarshal([]byte(evidenceJSON), &fact.Contribution.Evidence); err != nil {
			return nil, invalid(errors.New("sqlite.council_fact_evidence_invalid"))
		}
		canonical, err := council.NewContributionFact(fact)
		if err != nil || contributionDigest != canonical.Contribution.Digest() || factDigest != canonical.Digest() {
			return nil, invalid(errors.New("sqlite.council_fact_digest_invalid"))
		}
		result = append(result, canonical)
	}
	return result, mapDatabaseError(rows.Err())
}

func readCouncilDecisions(
	ctx context.Context, source queryer, goalValue string,
	rounds []application.CouncilRoundRecord, facts []council.ContributionFact,
) ([]application.CouncilDecisionRecord, error) {
	rows, err := source.QueryContext(ctx, `SELECT ref,round_ref,subject_digest,outcome,decision_digest,recorded_at
FROM council_decisions WHERE goal_ref=? ORDER BY recorded_at,ref`, goalValue)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []application.CouncilDecisionRecord
	for rows.Next() {
		var record application.CouncilDecisionRecord
		var outcome string
		var recordedAt int64
		if err := rows.Scan(&record.Ref, &record.RoundRef, &record.SubjectDigest,
			&outcome, &record.DecisionDigest, &recordedAt); err != nil {
			return nil, mapDatabaseError(err)
		}
		round, found := councilRoundFor(rounds, record.SubjectDigest)
		if !found || round.Ref != record.RoundRef {
			return nil, invalid(errors.New("sqlite.council_decision_round_invalid"))
		}
		record.Decision, err = council.Evaluate(round.Subject, councilFactsFor(facts, record.SubjectDigest))
		if err != nil || record.Decision.Outcome != council.Outcome(outcome) ||
			record.Decision.Outcome == council.OutcomePending ||
			record.DecisionDigest != application.CouncilSubjectDigest(record.Decision.Digest) {
			return nil, invalid(errors.New("sqlite.council_decision_digest_invalid"))
		}
		record.RecordedAt = time.Unix(0, recordedAt).UTC()
		result = append(result, record)
	}
	return result, mapDatabaseError(rows.Err())
}

func readCouncilSkips(
	ctx context.Context, source queryer, goalValue string,
) ([]application.CouncilSkipRecord, error) {
	rows, err := source.QueryContext(ctx, `SELECT ref,goal_ref,work_item_ref,change_set_ref,project_ref,
subject_digest,review_subject_digest,review_gate_digest,policy,plan_generation,work_item_generation,
app_spec_generation,principal_ref,reason,spec_hash,idempotency_key,skip_digest,request_ref,
request_fingerprint,authorization_receipt_ref,recorded_at
FROM council_skips WHERE goal_ref=? ORDER BY recorded_at,ref`, goalValue)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []application.CouncilSkipRecord
	for rows.Next() {
		var record application.CouncilSkipRecord
		var policy string
		var plan, itemGeneration, appSpec, recordedAt int64
		if err := rows.Scan(&record.Ref, &record.Subject.GoalRef, &record.Subject.WorkItemRef,
			&record.Subject.ChangeSetRef, &record.Subject.ProjectRef, &record.SubjectDigest,
			&record.Subject.ReviewSubjectDigest, &record.Subject.ReviewGateDigest, &policy, &plan,
			&itemGeneration, &appSpec, &record.Skip.PrincipalRef, &record.Skip.Reason, &record.Skip.SpecHash,
			&record.Skip.IdempotencyKey, &record.SkipDigest, &record.RequestRef, &record.RequestFingerprint,
			&record.AuthorizationReceiptRef, &recordedAt); err != nil {
			return nil, mapDatabaseError(err)
		}
		if plan <= 0 || itemGeneration <= 0 || appSpec <= 0 {
			return nil, invalid(errors.New("sqlite.council_skip_generation_invalid"))
		}
		record.Subject.Policy = council.Policy(policy)
		record.Subject.SpecHash = record.Skip.SpecHash
		record.Subject.PlanGeneration, record.Subject.WorkItemGeneration, record.Subject.AppSpecGeneration =
			uint64(plan), uint64(itemGeneration), uint64(appSpec)
		record.RecordedAt = time.Unix(0, recordedAt).UTC()
		record.Skip.RecordedAtUTC = record.RecordedAt
		record.Skip.CouncilSubjectDigest = string(record.SubjectDigest)
		canonical, err := council.NewSkip(record.Subject, record.Skip)
		if err != nil || record.SubjectDigest != application.CouncilSubjectDigest(record.Subject.Digest()) ||
			record.SkipDigest != application.CouncilSubjectDigest(canonical.Digest()) {
			return nil, invalid(errors.New("sqlite.council_skip_digest_invalid"))
		}
		record.Skip = canonical
		result = append(result, record)
	}
	return result, mapDatabaseError(rows.Err())
}
