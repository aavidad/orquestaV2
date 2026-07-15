package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

// V09 predates requested_by_ref. Recovery validates its exact domain snapshot
// without asking the latest runtime reader for V10-only columns.
func validateRecoveryV09GoalRecords(ctx context.Context, transaction *sql.Tx) error {
	rows, err := transaction.QueryContext(ctx, "SELECT ref FROM goals ORDER BY ref")
	if err != nil {
		return err
	}
	var refs []string
	for rows.Next() {
		var ref string
		if err := rows.Scan(&ref); err != nil {
			_ = rows.Close()
			return err
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, ref := range refs {
		record, err := readRecoveryV09GoalRecord(ctx, transaction, ref)
		if err != nil {
			return err
		}
		if err := validateGoalRecordConsistency(record, ref); err != nil {
			return err
		}
	}
	return nil
}

func readRecoveryV09GoalRecord(
	ctx context.Context,
	source queryer,
	goalValue string,
) (application.GoalRecord, error) {
	var requestRef, requestFingerprint string
	var spec goal.AppSpecSnapshot
	var snapshot goal.GoalSnapshot
	var state string
	var revision, planGeneration, generation int64
	var submittedAt, confirmedAt, createdAt int64
	var parentRef, parentHash sql.NullString
	var startedAt, closedAt sql.NullInt64
	err := source.QueryRowContext(ctx, `
SELECT g.request_ref, g.request_fingerprint,
       i.ref, i.actor_ref, i.project_ref, i.statement, i.submitted_at, i.hash,
       spec.ref, spec.generation, spec.parent_ref, spec.parent_hash, spec.objective,
       spec.reason, spec.confirmed_by, spec.confirmed_at, spec.hash,
       g.ref, g.actor_ref, g.project_ref, g.state, g.revision,
       g.created_at, g.started_at, g.closed_at, g.plan_generation
FROM goals g
JOIN app_specs spec ON spec.ref = g.app_spec_ref
JOIN intents i ON i.ref = spec.intent_ref
WHERE g.ref = ?`, goalValue).Scan(
		&requestRef, &requestFingerprint,
		&spec.Intent.Ref, &spec.Intent.ActorRef, &spec.Intent.ProjectRef,
		&spec.Intent.Statement, &submittedAt, &spec.Intent.Hash,
		&spec.Ref, &generation, &parentRef, &parentHash, &spec.Objective,
		&spec.Reason, &spec.ConfirmedBy, &confirmedAt, &spec.Hash,
		&snapshot.Ref, &snapshot.ActorRef, &snapshot.ProjectRef, &state, &revision,
		&createdAt, &startedAt, &closedAt, &planGeneration,
	)
	if err != nil {
		return application.GoalRecord{}, err
	}
	if revision <= 0 || planGeneration < 0 || generation <= 0 ||
		!validText(requestRef) || !validText(requestFingerprint) {
		return application.GoalRecord{}, errors.New("sqlite.recovery_v09_goal_invalid")
	}
	spec.Intent.SubmittedAt = time.Unix(0, submittedAt).UTC()
	spec.Generation = goal.AppSpecGeneration(generation)
	if parentRef.Valid {
		spec.ParentRef = parentRef.String
	}
	if parentHash.Valid {
		spec.ParentHash = parentHash.String
	}
	spec.ConfirmedAt = time.Unix(0, confirmedAt).UTC()
	snapshot.AppSpec = spec
	snapshot.SchemaVersion = goal.GoalSnapshotSchemaVersion
	snapshot.State = goal.GoalState(state)
	snapshot.Revision = goal.Revision(revision)
	snapshot.CreatedAt = time.Unix(0, createdAt).UTC()
	snapshot.StartedAt = restoredTime(startedAt)
	snapshot.ClosedAt = restoredTime(closedAt)
	snapshot.PlanGeneration = goal.PlanGeneration(planGeneration)
	snapshot.Phases, err = readPhases(ctx, source, goalValue)
	if err != nil {
		return application.GoalRecord{}, err
	}
	artifacts, artifactRefs, err := readArtifacts(ctx, source, goalValue)
	if err != nil {
		return application.GoalRecord{}, err
	}
	attestations, attestationRefs, err := readAttestations(ctx, source, goalValue)
	if err != nil {
		return application.GoalRecord{}, err
	}
	snapshot.WorkItems, err = readWorkItems(ctx, source, goalValue, artifactRefs, attestationRefs)
	if err != nil {
		return application.GoalRecord{}, err
	}
	aggregate, err := goal.RestoreGoal(snapshot)
	if err != nil {
		return application.GoalRecord{}, err
	}
	executions, err := readExecutions(ctx, source, goalValue)
	if err != nil {
		return application.GoalRecord{}, err
	}
	if aggregate.PlanGeneration() > 0 && len(executions) == 0 {
		return application.GoalRecord{}, errors.New("sqlite.recovery_v09_executions_missing")
	}
	receipts, err := readConsumptionReceipts(ctx, source, goalValue)
	if err != nil {
		return application.GoalRecord{}, err
	}
	requestedBy, err := identity.NewPrincipalRef(aggregate.Actor().String())
	if err != nil {
		return application.GoalRecord{}, err
	}
	return application.GoalRecord{
		RequestRef: requestRef, RequestFingerprint: requestFingerprint, RequestedBy: requestedBy,
		Goal: aggregate, Executions: executions, Artifacts: artifacts,
		Attestations: attestations, ConsumptionReceipts: receipts,
	}, nil
}
