package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
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

func (repository *Repository) Status(
	ctx context.Context,
	projectRef goal.ProjectRef,
) (application.RepositoryStatus, error) {
	if projectRef.String() == "" {
		return application.RepositoryStatus{}, invalid(errors.New("sqlite.project_ref_invalid"))
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return application.RepositoryStatus{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	var status application.RepositoryStatus
	err = transaction.QueryRowContext(ctx, `
SELECT
	(SELECT COUNT(*) FROM goals WHERE project_ref = ?),
	(SELECT COUNT(*) FROM goals WHERE project_ref = ? AND state = 'running'),
	(SELECT COUNT(*) FROM outbox o JOIN goals g ON g.ref = o.goal_ref
	 WHERE g.project_ref = ? AND o.completed_at IS NULL AND o.retired_at IS NULL AND o.quarantined_at IS NULL),
	(SELECT COUNT(*) FROM outbox o JOIN goals g ON g.ref = o.goal_ref
	 WHERE g.project_ref = ? AND o.quarantined_at IS NOT NULL)`,
		projectRef.String(), projectRef.String(), projectRef.String(), projectRef.String(),
	).Scan(
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
	projectRef goal.ProjectRef,
	limit int,
) ([]application.GoalSummary, error) {
	if projectRef.String() == "" || limit <= 0 {
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
WHERE g.project_ref = ?
ORDER BY g.created_at DESC, g.ref DESC
LIMIT ?`, projectRef.String(), limit)
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
	var requestedByValue string
	var spec goal.AppSpecSnapshot
	var snapshot goal.GoalSnapshot
	var state string
	var revision, planGeneration, generation int64
	var submittedAt, confirmedAt, createdAt int64
	var parentRef, parentHash sql.NullString
	var startedAt, closedAt sql.NullInt64
	err := source.QueryRowContext(ctx, `
SELECT g.request_ref, g.request_fingerprint, g.requested_by_ref,
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
		&requestedByValue,
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
	requestedBy, err := identity.NewPrincipalRef(requestedByValue)
	if err != nil {
		return application.GoalRecord{}, invalid(err)
	}
	snapshot.WorkItems = items
	snapshot.ChildHandoffResolutions, err = readChildHandoffResolutions(ctx, source, goalValue)
	if err != nil {
		return application.GoalRecord{}, err
	}
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
	receipts, err := readConsumptionReceipts(ctx, source, goalValue)
	if err != nil {
		return application.GoalRecord{}, err
	}
	record := application.GoalRecord{
		RequestRef:          requestRef,
		RequestFingerprint:  requestFingerprint,
		RequestedBy:         requestedBy,
		Goal:                aggregate,
		Executions:          executions,
		Artifacts:           artifacts,
		Attestations:        attestations,
		ConsumptionReceipts: receipts,
	}
	if err := validateGoalRecordConsistency(record, goalValue); err != nil {
		return application.GoalRecord{}, invalid(err)
	}
	return record, nil
}

func readWorkItems(
	ctx context.Context,
	source queryer,
	goalValue string,
	artifactRefs map[string][]string,
	attestationRefs map[string][]string,
) ([]goal.WorkItemSnapshot, error) {
	var handoffColumns int
	if err := source.QueryRowContext(ctx, `
SELECT COUNT(*) FROM pragma_table_info('work_items') WHERE name = 'handoff_required'`,
	).Scan(&handoffColumns); err != nil {
		return nil, mapDatabaseError(err)
	}
	if handoffColumns < 0 || handoffColumns > 1 {
		return nil, invalid(fmt.Errorf("sqlite.work_item_handoff_schema_invalid"))
	}
	handoffProjection := "0"
	if handoffColumns == 1 {
		handoffProjection = "handoff_required"
	}
	rows, err := source.QueryContext(ctx, `
SELECT ref, goal_ref, actor_ref, project_ref, objective, state, revision,
       phase_key, role_key, parent_ref, `+handoffProjection+`, output_contract, skip_reason,
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
		var revision, handoffRequired int64
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
			&handoffRequired,
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
		if handoffRequired != 0 && handoffRequired != 1 {
			return nil, invalid(fmt.Errorf("sqlite.work_item_handoff_required_invalid"))
		}
		handoff := handoffRequired == 1
		item.HandoffRequired = &handoff
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
       attempt_no, max_execution_attempts, replaces_execution_ref,
       plan_generation, app_spec_generation, spec_hash,
       max_output_bytes, provider_ref, model_ref, agent_ref,
       external_ref, created_at,
       deadline_at, started_at, provider_accepted_at, last_observed_at,
       provider_observed_at, finished_at, failure_code
FROM executions
WHERE goal_ref = ?
ORDER BY work_item_ref, attempt_no, ref`, goalValue)
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
		var replacesExecutionRef sql.NullString
		var attemptNo, maxExecutionAttempts, planGeneration, appSpecGeneration int64
		if err := rows.Scan(
			&refValue,
			&goalRefValue,
			&workItemRefValue,
			&state,
			&record.ArtifactMediaType,
			&record.IdempotencyKey,
			&attemptNo,
			&maxExecutionAttempts,
			&replacesExecutionRef,
			&planGeneration,
			&appSpecGeneration,
			&record.SpecHash,
			&record.MaxOutputBytes,
			&record.ProviderRef,
			&record.ModelRef,
			&record.AgentRef,
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
		if replacesExecutionRef.Valid {
			if record.ReplacesExecutionRef, refErr = goal.NewExecutionRef(replacesExecutionRef.String); refErr != nil {
				return nil, invalid(refErr)
			}
		}
		record.State = application.ExecutionState(state)
		if attemptNo <= 0 || maxExecutionAttempts <= 0 || planGeneration <= 0 || appSpecGeneration <= 0 {
			return nil, invalid(fmt.Errorf("sqlite.execution_generation_invalid"))
		}
		record.AttemptNo = uint64(attemptNo)
		record.MaxExecutionAttempts = uint64(maxExecutionAttempts)
		record.PlanGeneration = goal.PlanGeneration(planGeneration)
		record.AppSpecGeneration = goal.AppSpecGeneration(appSpecGeneration)
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

func readConsumptionReceipts(
	ctx context.Context,
	source queryer,
	goalValue string,
) ([]application.ActionConsumptionReceipt, error) {
	hasMailbox, err := sqliteTableHasColumn(ctx, source, "action_consumption_receipts", "mailbox_message_ref")
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	mailboxColumn := "NULL"
	if hasMailbox {
		mailboxColumn = "mailbox_message_ref"
	}
	query := `
SELECT action_ref, kind, goal_ref, work_item_ref, execution_ref,
       ` + mailboxColumn + `, plan_generation, work_item_generation, fence, delivery_attempt,
       claim_token, worker_ref, outcome, error_code, consumed_at
FROM action_consumption_receipts
WHERE goal_ref = ?
ORDER BY consumed_at, action_ref`
	rows, err := source.QueryContext(ctx, query, goalValue)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []application.ActionConsumptionReceipt
	for rows.Next() {
		var receipt application.ActionConsumptionReceipt
		var kind, goalRefValue, workItemRefValue, executionRefValue, outcome string
		var mailboxMessageValue sql.NullString
		var planGeneration, itemGeneration, fence, deliveryAttempt, consumedAt int64
		if err := rows.Scan(
			&receipt.ActionRef, &kind, &goalRefValue, &workItemRefValue, &executionRefValue,
			&mailboxMessageValue,
			&planGeneration, &itemGeneration, &fence, &deliveryAttempt,
			&receipt.ClaimToken, &receipt.WorkerRef, &outcome, &receipt.ErrorCode, &consumedAt,
		); err != nil {
			return nil, mapDatabaseError(err)
		}
		if planGeneration <= 0 || itemGeneration <= 0 || fence <= 0 || deliveryAttempt <= 0 {
			return nil, invalid(errors.New("sqlite.receipt_generation_invalid"))
		}
		var refErr error
		receipt.Kind = application.ActionKind(kind)
		if mailboxMessageValue.Valid {
			receipt.MailboxMessageRef, refErr = application.NewMailboxMessageRef(mailboxMessageValue.String)
			if refErr != nil {
				return nil, invalid(refErr)
			}
		}
		if receipt.GoalRef, refErr = goal.NewGoalRef(goalRefValue); refErr != nil {
			return nil, invalid(refErr)
		}
		if receipt.WorkItemRef, refErr = goal.NewWorkItemRef(workItemRefValue); refErr != nil {
			return nil, invalid(refErr)
		}
		if receipt.ExecutionRef, refErr = goal.NewExecutionRef(executionRefValue); refErr != nil {
			return nil, invalid(refErr)
		}
		receipt.PlanGeneration = goal.PlanGeneration(planGeneration)
		receipt.WorkItemGeneration = goal.Revision(itemGeneration)
		receipt.Fence = uint64(fence)
		receipt.DeliveryAttempt = uint64(deliveryAttempt)
		receipt.Outcome = application.ActionConsumptionOutcome(outcome)
		receipt.ConsumedAt = time.Unix(0, consumedAt).UTC()
		result = append(result, receipt)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	return result, nil
}

func readChildHandoffResolutions(
	ctx context.Context,
	source queryer,
	goalValue string,
) ([]goal.ChildHandoffResolutionSnapshot, error) {
	var tableExists int
	if err := source.QueryRowContext(ctx, `
SELECT COUNT(*) FROM sqlite_schema
WHERE type = 'table' AND name = 'goal_child_handoff_resolutions'`).Scan(&tableExists); err != nil {
		return nil, mapDatabaseError(err)
	}
	if tableExists == 0 {
		return nil, nil
	}
	rows, err := source.QueryContext(ctx, `
SELECT parent_work_item_ref, child_work_item_ref, mailbox_message_ref,
       outcome, receipt_ref, resolved_at
FROM goal_child_handoff_resolutions
WHERE goal_ref = ?
ORDER BY resolved_at, parent_work_item_ref, child_work_item_ref`, goalValue)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []goal.ChildHandoffResolutionSnapshot
	for rows.Next() {
		var resolution goal.ChildHandoffResolutionSnapshot
		var resolvedAt int64
		if err := rows.Scan(
			&resolution.ParentRef, &resolution.ChildRef, &resolution.MessageRef,
			&resolution.Outcome, &resolution.ReceiptRef, &resolvedAt,
		); err != nil {
			return nil, mapDatabaseError(err)
		}
		resolution.ResolvedAt = time.Unix(0, resolvedAt).UTC()
		result = append(result, resolution)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	return result, nil
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
