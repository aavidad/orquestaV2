package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

func (repository *Repository) GetGoal(ctx context.Context, goalRef goal.GoalRef) (application.GoalRecord, error) {
	if goalRef.String() == "" {
		return application.GoalRecord{}, invalid(errors.New("sqlite.goal_ref_invalid"))
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return application.GoalRecord{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	record, err := readGoalRecord(ctx, transaction, goalRef.String())
	if err != nil {
		return application.GoalRecord{}, err
	}
	if err := commit(transaction); err != nil {
		return application.GoalRecord{}, err
	}
	return record, nil
}

func (repository *Repository) Status(ctx context.Context) (application.RepositoryStatus, error) {
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return application.RepositoryStatus{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	var status application.RepositoryStatus
	err = transaction.QueryRowContext(ctx, `
SELECT
    (SELECT COUNT(*) FROM goals),
    (SELECT COUNT(*) FROM goals WHERE state = 'running'),
    (SELECT COUNT(*) FROM outbox WHERE completed_at IS NULL AND quarantined_at IS NULL),
    (SELECT COUNT(*) FROM outbox WHERE quarantined_at IS NOT NULL)`).Scan(
		&status.Goals,
		&status.RunningGoals,
		&status.PendingActions,
		&status.QuarantinedActions,
	)
	if err != nil {
		return application.RepositoryStatus{}, mapDatabaseError(err)
	}
	if err := commit(transaction); err != nil {
		return application.RepositoryStatus{}, err
	}
	return status, nil
}

func (repository *Repository) ListGoals(
	ctx context.Context,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	limit int,
) ([]application.GoalSummary, error) {
	if actorRef.String() == "" || projectRef.String() == "" || limit <= 0 {
		return nil, invalid(errors.New("sqlite.list_query_invalid"))
	}
	database, err := repository.database()
	if err != nil {
		return nil, err
	}
	rows, err := database.QueryContext(ctx, `
SELECT g.ref,
       i.ref, i.actor_ref, i.project_ref, i.statement, i.submitted_at, i.hash,
       spec.ref, spec.generation, spec.parent_ref, spec.parent_hash, spec.objective,
       spec.reason, spec.confirmed_by, spec.confirmed_at, spec.hash,
       g.actor_ref, g.project_ref,
       g.state, g.revision, g.created_at, g.closed_at,
       (SELECT COUNT(*) FROM artifacts a WHERE a.goal_ref = g.ref)
FROM goals g
JOIN app_specs spec ON spec.ref = g.app_spec_ref
JOIN intents i ON i.ref = spec.intent_ref
WHERE g.actor_ref = ? AND g.project_ref = ?
ORDER BY g.created_at DESC, g.ref DESC
LIMIT ?`, actorRef.String(), projectRef.String(), limit)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()

	result := make([]application.GoalSummary, 0)
	for rows.Next() {
		var summary application.GoalSummary
		var goalValue, goalActorValue, goalProjectValue string
		var specSnapshot goal.AppSpecSnapshot
		var parentRef, parentHash sql.NullString
		var state string
		var revision, generation int64
		var submittedAt, confirmedAt, createdAt int64
		var closedAt sql.NullInt64
		if err := rows.Scan(
			&goalValue,
			&specSnapshot.Intent.Ref,
			&specSnapshot.Intent.ActorRef,
			&specSnapshot.Intent.ProjectRef,
			&specSnapshot.Intent.Statement,
			&submittedAt,
			&specSnapshot.Intent.Hash,
			&specSnapshot.Ref,
			&generation,
			&parentRef,
			&parentHash,
			&specSnapshot.Objective,
			&specSnapshot.Reason,
			&specSnapshot.ConfirmedBy,
			&confirmedAt,
			&specSnapshot.Hash,
			&goalActorValue,
			&goalProjectValue,
			&state,
			&revision,
			&createdAt,
			&closedAt,
			&summary.ArtifactCount,
		); err != nil {
			return nil, mapDatabaseError(err)
		}
		if revision <= 0 || generation <= 0 {
			return nil, invalid(fmt.Errorf("sqlite.revision_invalid"))
		}
		specSnapshot.Intent.SubmittedAt = time.Unix(0, submittedAt).UTC()
		specSnapshot.Generation = goal.AppSpecGeneration(generation)
		if parentRef.Valid {
			specSnapshot.ParentRef = parentRef.String
		}
		if parentHash.Valid {
			specSnapshot.ParentHash = parentHash.String
		}
		specSnapshot.ConfirmedAt = time.Unix(0, confirmedAt).UTC()
		spec, restoreErr := goal.RestoreAppSpec(specSnapshot)
		if restoreErr != nil {
			return nil, invalid(restoreErr)
		}
		if spec.Intent().Actor().String() != goalActorValue || spec.Intent().Project().String() != goalProjectValue {
			return nil, invalid(fmt.Errorf("sqlite.goal_app_spec_invalid"))
		}
		var refErr error
		if summary.Ref, refErr = goal.NewGoalRef(goalValue); refErr != nil {
			return nil, invalid(refErr)
		}
		summary.IntentRef = spec.Intent().Ref()
		summary.AppSpecRef = spec.Ref()
		summary.AppSpecGeneration = spec.Generation()
		summary.SpecHash = spec.Hash()
		summary.ActorRef = spec.Intent().Actor()
		summary.ProjectRef = spec.Intent().Project()
		summary.Statement = spec.Intent().Statement()
		summary.State = goal.GoalState(state)
		summary.Revision = goal.Revision(revision)
		summary.CreatedAt = time.Unix(0, createdAt).UTC()
		summary.ClosedAt = restoredTime(closedAt)
		result = append(result, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	return result, nil
}

func readGoalRecord(ctx context.Context, source queryer, goalValue string) (application.GoalRecord, error) {
	var requestRef string
	var requestFingerprint string
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
		&requestRef,
		&requestFingerprint,
		&spec.Intent.Ref,
		&spec.Intent.ActorRef,
		&spec.Intent.ProjectRef,
		&spec.Intent.Statement,
		&submittedAt,
		&spec.Intent.Hash,
		&spec.Ref,
		&generation,
		&parentRef,
		&parentHash,
		&spec.Objective,
		&spec.Reason,
		&spec.ConfirmedBy,
		&confirmedAt,
		&spec.Hash,
		&snapshot.Ref,
		&snapshot.ActorRef,
		&snapshot.ProjectRef,
		&state,
		&revision,
		&createdAt,
		&startedAt,
		&closedAt,
		&planGeneration,
	)
	if err != nil {
		return application.GoalRecord{}, mapDatabaseError(err)
	}
	if revision <= 0 || planGeneration < 0 || generation <= 0 {
		return application.GoalRecord{}, invalid(fmt.Errorf("sqlite.revision_invalid"))
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
	items, err := readWorkItems(ctx, source, goalValue, artifactRefs, attestationRefs)
	if err != nil {
		return application.GoalRecord{}, err
	}
	if !validText(requestRef) || !validText(requestFingerprint) {
		return application.GoalRecord{}, invalid(fmt.Errorf("sqlite.request_identity_invalid"))
	}
	snapshot.WorkItems = items
	aggregate, err := goal.RestoreGoal(snapshot)
	if err != nil {
		return application.GoalRecord{}, invalid(err)
	}
	executions, err := readExecutions(ctx, source, goalValue)
	if err != nil {
		return application.GoalRecord{}, err
	}
	if aggregate.PlanGeneration() > 0 && len(executions) == 0 {
		return application.GoalRecord{}, invalid(fmt.Errorf("sqlite.executions_missing"))
	}
	return application.GoalRecord{
		RequestRef:         requestRef,
		RequestFingerprint: requestFingerprint,
		Goal:               aggregate,
		Executions:         executions,
		Artifacts:          artifacts,
		Attestations:       attestations,
	}, nil
}

func readWorkItems(
	ctx context.Context,
	source queryer,
	goalValue string,
	artifactRefs map[string][]string,
	attestationRefs map[string][]string,
) ([]goal.WorkItemSnapshot, error) {
	rows, err := source.QueryContext(ctx, `
SELECT ref, goal_ref, actor_ref, project_ref, objective, state, revision,
       phase_key, role_key, parent_ref, output_contract, skip_reason,
       created_at, started_at, finished_at, execution_ref
FROM work_items
WHERE goal_ref = ?
ORDER BY position`, goalValue)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []goal.WorkItemSnapshot
	for rows.Next() {
		var item goal.WorkItemSnapshot
		var state, outputContract, skipReason string
		var revision int64
		var createdAt int64
		var startedAt, finishedAt sql.NullInt64
		var parentRef, executionRef sql.NullString
		if err := rows.Scan(
			&item.Ref,
			&item.GoalRef,
			&item.ActorRef,
			&item.ProjectRef,
			&item.Objective,
			&state,
			&revision,
			&item.PhaseKey,
			&item.RoleKey,
			&parentRef,
			&outputContract,
			&skipReason,
			&createdAt,
			&startedAt,
			&finishedAt,
			&executionRef,
		); err != nil {
			return nil, mapDatabaseError(err)
		}
		if revision <= 0 {
			return nil, invalid(fmt.Errorf("sqlite.revision_invalid"))
		}
		item.State = goal.WorkItemState(state)
		item.OutputContract = goal.OutputContractKind(outputContract)
		item.SkipReason = goal.WorkItemSkipReason(skipReason)
		item.Revision = goal.Revision(revision)
		item.CreatedAt = time.Unix(0, createdAt).UTC()
		item.StartedAt = restoredTime(startedAt)
		item.FinishedAt = restoredTime(finishedAt)
		if executionRef.Valid {
			item.ExecutionRef = executionRef.String
		}
		if parentRef.Valid {
			item.ParentRef = parentRef.String
		}
		item.ArtifactRefs = append([]string(nil), artifactRefs[item.Ref]...)
		item.AttestationRefs = append([]string(nil), attestationRefs[item.Ref]...)
		item.DependencyRefs, err = readOrderedStrings(ctx, source, `
SELECT dependency_ref FROM work_item_dependencies WHERE work_item_ref = ? ORDER BY position`, item.Ref)
		if err != nil {
			return nil, err
		}
		item.WriteSet, err = readOrderedStrings(ctx, source, `
SELECT scope FROM work_item_write_scopes WHERE work_item_ref = ? ORDER BY position`, item.Ref)
		if err != nil {
			return nil, err
		}
		item.SkillRefs, err = readContractRefs(ctx, source, `
SELECT value FROM work_item_requirement_refs
WHERE goal_ref = ? AND work_item_ref = ? AND kind = ? ORDER BY position`, goalValue, item.Ref, "skill")
		if err != nil {
			return nil, err
		}
		item.ToolRefs, err = readContractRefs(ctx, source, `
SELECT value FROM work_item_requirement_refs
WHERE goal_ref = ? AND work_item_ref = ? AND kind = ? ORDER BY position`, goalValue, item.Ref, "tool")
		if err != nil {
			return nil, err
		}
		item.CapabilityRefs, err = readContractRefs(ctx, source, `
SELECT value FROM work_item_requirement_refs
WHERE goal_ref = ? AND work_item_ref = ? AND kind = ? ORDER BY position`, goalValue, item.Ref, "capability")
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	return result, nil
}

func readExecutions(ctx context.Context, source queryer, goalValue string) ([]application.ExecutionRecord, error) {
	rows, err := source.QueryContext(ctx, `
SELECT ref, goal_ref, work_item_ref, state, artifact_media_type, idempotency_key,
       max_output_bytes, max_attempts, provider_ref, external_ref, created_at,
       deadline_at, started_at, provider_accepted_at, last_observed_at,
       provider_observed_at, finished_at, failure_code
FROM executions
WHERE goal_ref = ?
ORDER BY created_at, ref`, goalValue)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var records []application.ExecutionRecord
	for rows.Next() {
		var record application.ExecutionRecord
		var refValue, goalRefValue, workItemRefValue, state string
		var createdAt int64
		var deadlineAt, startedAt, providerAcceptedAt, observedAt, providerObservedAt, finishedAt sql.NullInt64
		var maxAttempts int64
		if err := rows.Scan(
			&refValue,
			&goalRefValue,
			&workItemRefValue,
			&state,
			&record.ArtifactMediaType,
			&record.IdempotencyKey,
			&record.MaxOutputBytes,
			&maxAttempts,
			&record.ProviderRef,
			&record.ExternalRef,
			&createdAt,
			&deadlineAt,
			&startedAt,
			&providerAcceptedAt,
			&observedAt,
			&providerObservedAt,
			&finishedAt,
			&record.FailureCode,
		); err != nil {
			return nil, mapDatabaseError(err)
		}
		var refErr error
		if record.Ref, refErr = goal.NewExecutionRef(refValue); refErr != nil {
			return nil, invalid(refErr)
		}
		if record.GoalRef, refErr = goal.NewGoalRef(goalRefValue); refErr != nil {
			return nil, invalid(refErr)
		}
		if record.WorkItemRef, refErr = goal.NewWorkItemRef(workItemRefValue); refErr != nil {
			return nil, invalid(refErr)
		}
		record.State = application.ExecutionState(state)
		if maxAttempts <= 0 {
			return nil, invalid(fmt.Errorf("sqlite.max_attempts_invalid"))
		}
		record.MaxAttempts = uint64(maxAttempts)
		record.CreatedAt = time.Unix(0, createdAt).UTC()
		record.DeadlineAt = restoredTime(deadlineAt)
		record.StartedAt = restoredTime(startedAt)
		record.ProviderAcceptedAt = restoredTime(providerAcceptedAt)
		record.LastObservedAt = restoredTime(observedAt)
		record.ProviderObservedAt = restoredTime(providerObservedAt)
		record.FinishedAt = restoredTime(finishedAt)
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	return records, nil
}

func readPhases(ctx context.Context, source queryer, goalValue string) ([]goal.PhaseInstanceSnapshot, error) {
	rows, err := source.QueryContext(ctx, `
SELECT ref, phase_key, template_ref FROM goal_phases WHERE goal_ref = ? ORDER BY position`, goalValue)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []goal.PhaseInstanceSnapshot
	for rows.Next() {
		var phase goal.PhaseInstanceSnapshot
		if err := rows.Scan(&phase.Ref, &phase.Key, &phase.TemplateRef); err != nil {
			return nil, mapDatabaseError(err)
		}
		phase.InputRefs, err = readContractRefs(ctx, source, `
SELECT value FROM goal_phase_contract_refs
WHERE goal_ref = ? AND phase_ref = ? AND kind = ? ORDER BY position`, goalValue, phase.Ref, "input")
		if err != nil {
			return nil, err
		}
		phase.CriterionRefs, err = readContractRefs(ctx, source, `
SELECT value FROM goal_phase_contract_refs
WHERE goal_ref = ? AND phase_ref = ? AND kind = ? ORDER BY position`, goalValue, phase.Ref, "criterion")
		if err != nil {
			return nil, err
		}
		result = append(result, phase)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	return result, nil
}

func readContractRefs(ctx context.Context, source queryer, query string, args ...any) ([]string, error) {
	rows, err := source.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, mapDatabaseError(err)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	return result, nil
}

func readOrderedStrings(ctx context.Context, source queryer, query string, value string) ([]string, error) {
	rows, err := source.QueryContext(ctx, query, value)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			return nil, mapDatabaseError(err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	return result, nil
}

func readArtifacts(
	ctx context.Context,
	source queryer,
	goalValue string,
) ([]application.ArtifactRecord, map[string][]string, error) {
	rows, err := source.QueryContext(ctx, `
SELECT ref, goal_ref, work_item_ref, digest, media_type, size, created_at
FROM artifacts
WHERE goal_ref = ?
ORDER BY created_at, ref`, goalValue)
	if err != nil {
		return nil, nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []application.ArtifactRecord
	refs := make(map[string][]string)
	for rows.Next() {
		var record application.ArtifactRecord
		var refValue, goalRefValue, workItemRefValue string
		var createdAt int64
		if err := rows.Scan(
			&refValue,
			&goalRefValue,
			&workItemRefValue,
			&record.Stored.Digest,
			&record.Stored.MediaType,
			&record.Stored.Size,
			&createdAt,
		); err != nil {
			return nil, nil, mapDatabaseError(err)
		}
		var refErr error
		if record.Stored.Ref, refErr = goal.NewArtifactRef(refValue); refErr != nil {
			return nil, nil, invalid(refErr)
		}
		if record.GoalRef, refErr = goal.NewGoalRef(goalRefValue); refErr != nil {
			return nil, nil, invalid(refErr)
		}
		if record.WorkItemRef, refErr = goal.NewWorkItemRef(workItemRefValue); refErr != nil {
			return nil, nil, invalid(refErr)
		}
		record.CreatedAt = time.Unix(0, createdAt).UTC()
		refs[workItemRefValue] = append(refs[workItemRefValue], refValue)
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, mapDatabaseError(err)
	}
	return result, refs, nil
}

func readAttestations(
	ctx context.Context,
	source queryer,
	goalValue string,
) ([]application.AttestationRecord, map[string][]string, error) {
	rows, err := source.QueryContext(ctx, `
SELECT ref, goal_ref, work_item_ref, execution_ref, artifact_ref, policy, accepted_at
FROM attestations
WHERE goal_ref = ?
ORDER BY accepted_at, ref`, goalValue)
	if err != nil {
		return nil, nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []application.AttestationRecord
	refs := make(map[string][]string)
	for rows.Next() {
		var record application.AttestationRecord
		var refValue, goalRefValue, workItemRefValue, executionRefValue, artifactRefValue string
		var acceptedAt int64
		if err := rows.Scan(
			&refValue,
			&goalRefValue,
			&workItemRefValue,
			&executionRefValue,
			&artifactRefValue,
			&record.Policy,
			&acceptedAt,
		); err != nil {
			return nil, nil, mapDatabaseError(err)
		}
		var refErr error
		if record.Ref, refErr = goal.NewAttestationRef(refValue); refErr != nil {
			return nil, nil, invalid(refErr)
		}
		if record.GoalRef, refErr = goal.NewGoalRef(goalRefValue); refErr != nil {
			return nil, nil, invalid(refErr)
		}
		if record.WorkItemRef, refErr = goal.NewWorkItemRef(workItemRefValue); refErr != nil {
			return nil, nil, invalid(refErr)
		}
		if record.ExecutionRef, refErr = goal.NewExecutionRef(executionRefValue); refErr != nil {
			return nil, nil, invalid(refErr)
		}
		if record.ArtifactRef, refErr = goal.NewArtifactRef(artifactRefValue); refErr != nil {
			return nil, nil, invalid(refErr)
		}
		record.AcceptedAt = time.Unix(0, acceptedAt).UTC()
		refs[workItemRefValue] = append(refs[workItemRefValue], refValue)
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, mapDatabaseError(err)
	}
	return result, refs, nil
}
