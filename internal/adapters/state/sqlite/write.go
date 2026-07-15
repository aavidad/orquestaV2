package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func insertCreateState(ctx context.Context, transaction *sql.Tx, state application.CreateGoalState) error {
	snapshot := state.Goal.Snapshot()
	if err := insertGoalHeader(
		ctx, transaction, state.RequestRef, state.RequestFingerprint, state.RequestedBy, snapshot,
	); err != nil {
		return err
	}
	for position, phase := range snapshot.Phases {
		if _, err := transaction.ExecContext(ctx, `
INSERT INTO goal_phases(goal_ref, ref, phase_key, template_ref, position) VALUES (?, ?, ?, ?, ?)`,
			snapshot.Ref, phase.Ref, phase.Key, phase.TemplateRef, position,
		); err != nil {
			return mapDatabaseError(err)
		}
		if err := insertOrderedContractRefs(ctx, transaction, "goal_phase_contract_refs", snapshot.Ref, phase.Ref, "input", phase.InputRefs); err != nil {
			return err
		}
		if err := insertOrderedContractRefs(ctx, transaction, "goal_phase_contract_refs", snapshot.Ref, phase.Ref, "criterion", phase.CriterionRefs); err != nil {
			return err
		}
	}
	for position, item := range snapshot.WorkItems {
		if _, err := transaction.ExecContext(ctx, `
INSERT INTO work_items(
    ref, goal_ref, actor_ref, project_ref, objective, phase_key, role_key, parent_ref,
    output_contract, skip_reason, state, revision, position,
    created_at, started_at, finished_at, execution_ref
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.Ref,
			item.GoalRef,
			item.ActorRef,
			item.ProjectRef,
			item.Objective,
			item.PhaseKey,
			item.RoleKey,
			nullableString(item.ParentRef),
			string(item.OutputContract),
			string(item.SkipReason),
			string(item.State),
			int64(item.Revision),
			position,
			requiredTime(item.CreatedAt),
			storedTime(item.StartedAt),
			storedTime(item.FinishedAt),
			nullableString(item.ExecutionRef),
		); err != nil {
			return mapDatabaseError(err)
		}
		for dependencyPosition, dependency := range item.DependencyRefs {
			if _, err := transaction.ExecContext(ctx, `
INSERT INTO work_item_dependencies(goal_ref, work_item_ref, dependency_ref, position)
VALUES (?, ?, ?, ?)`, item.GoalRef, item.Ref, dependency, dependencyPosition); err != nil {
				return mapDatabaseError(err)
			}
		}
		for scopePosition, scope := range item.WriteSet {
			if _, err := transaction.ExecContext(ctx, `
INSERT INTO work_item_write_scopes(goal_ref, work_item_ref, scope, position)
VALUES (?, ?, ?, ?)`, item.GoalRef, item.Ref, scope, scopePosition); err != nil {
				return mapDatabaseError(err)
			}
		}
		if err := insertOrderedContractRefs(ctx, transaction, "work_item_requirement_refs", item.GoalRef, item.Ref, "skill", item.SkillRefs); err != nil {
			return err
		}
		if err := insertOrderedContractRefs(ctx, transaction, "work_item_requirement_refs", item.GoalRef, item.Ref, "tool", item.ToolRefs); err != nil {
			return err
		}
		if err := insertOrderedContractRefs(ctx, transaction, "work_item_requirement_refs", item.GoalRef, item.Ref, "capability", item.CapabilityRefs); err != nil {
			return err
		}
	}
	for _, execution := range state.Executions {
		if err := insertExecution(ctx, transaction, execution); err != nil {
			return err
		}
	}
	for _, action := range state.Actions {
		if err := insertAction(ctx, transaction, action); err != nil {
			return err
		}
	}
	return insertEvents(ctx, transaction, state.Events)
}

func insertGoalHeader(
	ctx context.Context,
	transaction *sql.Tx,
	requestRef string,
	requestFingerprint string,
	requestedBy identity.PrincipalRef,
	snapshot goal.GoalSnapshot,
) error {
	intent := snapshot.AppSpec.Intent
	if _, err := transaction.ExecContext(ctx, `
INSERT INTO intents(ref, actor_ref, project_ref, statement, submitted_at, hash)
VALUES (?, ?, ?, ?, ?, ?)`,
		intent.Ref,
		intent.ActorRef,
		intent.ProjectRef,
		intent.Statement,
		requiredTime(intent.SubmittedAt),
		intent.Hash,
	); err != nil {
		return mapDatabaseError(err)
	}
	if err := insertAppSpecSnapshot(ctx, transaction, snapshot.AppSpec); err != nil {
		return err
	}
	if _, err := transaction.ExecContext(ctx, `
INSERT INTO goals(
    ref, request_ref, request_fingerprint, requested_by_ref, app_spec_ref, actor_ref, project_ref, state, revision,
    created_at, started_at, closed_at, plan_generation
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snapshot.Ref,
		requestRef,
		requestFingerprint,
		requestedBy.String(),
		snapshot.AppSpec.Ref,
		snapshot.ActorRef,
		snapshot.ProjectRef,
		string(snapshot.State),
		int64(snapshot.Revision),
		requiredTime(snapshot.CreatedAt),
		storedTime(snapshot.StartedAt),
		storedTime(snapshot.ClosedAt),
		int64(snapshot.PlanGeneration),
	); err != nil {
		return mapDatabaseError(err)
	}
	return nil
}

func insertAppSpecSnapshot(ctx context.Context, transaction *sql.Tx, snapshot goal.AppSpecSnapshot) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO app_specs(
    ref, intent_ref, generation, parent_ref, parent_hash, objective, reason,
    confirmed_by, confirmed_at, hash
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snapshot.Ref,
		snapshot.Intent.Ref,
		int64(snapshot.Generation),
		nullableString(snapshot.ParentRef),
		nullableString(snapshot.ParentHash),
		snapshot.Objective,
		snapshot.Reason,
		snapshot.ConfirmedBy,
		requiredTime(snapshot.ConfirmedAt),
		snapshot.Hash,
	)
	return mapDatabaseError(err)
}

func insertExecution(ctx context.Context, transaction *sql.Tx, execution application.ExecutionRecord) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO executions(
    ref, goal_ref, work_item_ref, attempt_no, max_execution_attempts,
    replaces_execution_ref, plan_generation, app_spec_generation, spec_hash,
    state, artifact_media_type, idempotency_key, max_output_bytes,
    provider_ref, model_ref, agent_ref, external_ref, created_at, deadline_at,
    started_at, provider_accepted_at, last_observed_at, provider_observed_at,
    finished_at, failure_code
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		execution.Ref.String(),
		execution.GoalRef.String(),
		execution.WorkItemRef.String(),
		int64(execution.AttemptNo),
		int64(execution.MaxExecutionAttempts),
		nullableString(execution.ReplacesExecutionRef.String()),
		int64(execution.PlanGeneration),
		int64(execution.AppSpecGeneration),
		execution.SpecHash,
		string(execution.State),
		execution.ArtifactMediaType,
		execution.IdempotencyKey,
		execution.MaxOutputBytes,
		execution.ProviderRef,
		execution.ModelRef,
		execution.AgentRef,
		execution.ExternalRef,
		requiredTime(execution.CreatedAt),
		storedTime(execution.DeadlineAt),
		storedTime(execution.StartedAt),
		storedTime(execution.ProviderAcceptedAt),
		storedTime(execution.LastObservedAt),
		storedTime(execution.ProviderObservedAt),
		storedTime(execution.FinishedAt),
		execution.FailureCode,
	)
	return mapDatabaseError(err)
}

func updateExecutionCAS(
	ctx context.Context,
	transaction *sql.Tx,
	execution application.ExecutionRecord,
	expected application.ExecutionState,
) error {
	if expected == application.ExecutionDispatching && execution.State == application.ExecutionRunning {
		return acceptExecutionCAS(ctx, transaction, execution)
	}
	result, err := transaction.ExecContext(ctx, `
UPDATE executions
SET state = ?, deadline_at = ?, started_at = ?, provider_accepted_at = ?, last_observed_at = ?,
    provider_observed_at = ?, finished_at = ?, failure_code = ?
WHERE ref = ? AND goal_ref = ? AND work_item_ref = ? AND state = ?
  AND provider_ref = ? AND model_ref = ? AND agent_ref = ? AND external_ref = ?
  AND attempt_no = ? AND max_execution_attempts = ?
  AND replaces_execution_ref IS ?
  AND plan_generation = ? AND app_spec_generation = ? AND spec_hash = ?
  AND artifact_media_type = ? AND idempotency_key = ?
  AND max_output_bytes = ? AND created_at = ?`,
		string(execution.State),
		storedTime(execution.DeadlineAt),
		storedTime(execution.StartedAt),
		storedTime(execution.ProviderAcceptedAt),
		storedTime(execution.LastObservedAt),
		storedTime(execution.ProviderObservedAt),
		storedTime(execution.FinishedAt),
		execution.FailureCode,
		execution.Ref.String(),
		execution.GoalRef.String(),
		execution.WorkItemRef.String(),
		string(expected),
		execution.ProviderRef,
		execution.ModelRef,
		execution.AgentRef,
		execution.ExternalRef,
		int64(execution.AttemptNo),
		int64(execution.MaxExecutionAttempts),
		nullableString(execution.ReplacesExecutionRef.String()),
		int64(execution.PlanGeneration),
		int64(execution.AppSpecGeneration),
		execution.SpecHash,
		execution.ArtifactMediaType,
		execution.IdempotencyKey,
		execution.MaxOutputBytes,
		requiredTime(execution.CreatedAt),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}

func acceptExecutionCAS(ctx context.Context, transaction *sql.Tx, execution application.ExecutionRecord) error {
	result, err := transaction.ExecContext(ctx, `
UPDATE executions
SET state = ?, provider_ref = ?, model_ref = ?, agent_ref = ?, external_ref = ?,
    deadline_at = ?, started_at = ?, provider_accepted_at = ?, last_observed_at = ?,
    provider_observed_at = ?, finished_at = ?, failure_code = ?
WHERE ref = ? AND goal_ref = ? AND work_item_ref = ? AND state = 'dispatching'
  AND provider_ref = '' AND model_ref = '' AND agent_ref = '' AND external_ref = ''
  AND attempt_no = ? AND max_execution_attempts = ?
  AND replaces_execution_ref IS ?
  AND plan_generation = ? AND app_spec_generation = ? AND spec_hash = ?
  AND artifact_media_type = ? AND idempotency_key = ?
  AND max_output_bytes = ? AND created_at = ?`,
		string(execution.State),
		execution.ProviderRef,
		execution.ModelRef,
		execution.AgentRef,
		execution.ExternalRef,
		storedTime(execution.DeadlineAt),
		storedTime(execution.StartedAt),
		storedTime(execution.ProviderAcceptedAt),
		storedTime(execution.LastObservedAt),
		storedTime(execution.ProviderObservedAt),
		storedTime(execution.FinishedAt),
		execution.FailureCode,
		execution.Ref.String(),
		execution.GoalRef.String(),
		execution.WorkItemRef.String(),
		int64(execution.AttemptNo),
		int64(execution.MaxExecutionAttempts),
		nullableString(execution.ReplacesExecutionRef.String()),
		int64(execution.PlanGeneration),
		int64(execution.AppSpecGeneration),
		execution.SpecHash,
		execution.ArtifactMediaType,
		execution.IdempotencyKey,
		execution.MaxOutputBytes,
		requiredTime(execution.CreatedAt),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}

func updateGoalCAS(
	ctx context.Context,
	transaction *sql.Tx,
	aggregate goal.Goal,
	expected goal.Revision,
) error {
	snapshot := aggregate.Snapshot()
	result, err := transaction.ExecContext(ctx, `
UPDATE goals
SET state = ?, revision = ?, started_at = ?, closed_at = ?, plan_generation = ?
WHERE ref = ? AND revision = ?`,
		string(snapshot.State),
		int64(snapshot.Revision),
		storedTime(snapshot.StartedAt),
		storedTime(snapshot.ClosedAt),
		int64(snapshot.PlanGeneration),
		snapshot.Ref,
		int64(expected),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}

func updateGoalWorkItems(ctx context.Context, transaction *sql.Tx, aggregate goal.Goal) error {
	for _, item := range aggregate.WorkItems() {
		if err := updateWorkItemSnapshot(ctx, transaction, itemSnapshot(item)); err != nil {
			return err
		}
	}
	return nil
}

func updateWorkItemSnapshot(ctx context.Context, transaction *sql.Tx, snapshot goal.WorkItemSnapshot) error {
	result, err := transaction.ExecContext(ctx, `
UPDATE work_items
SET state = ?, revision = ?, started_at = ?, finished_at = ?, execution_ref = ?, skip_reason = ?
WHERE ref = ? AND goal_ref = ?`,
		string(snapshot.State), int64(snapshot.Revision), storedTime(snapshot.StartedAt),
		storedTime(snapshot.FinishedAt), nullableString(snapshot.ExecutionRef), string(snapshot.SkipReason),
		snapshot.Ref, snapshot.GoalRef,
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}

func updateWorkItemCAS(
	ctx context.Context,
	transaction *sql.Tx,
	item goal.WorkItem,
	expected goal.Revision,
) error {
	snapshot := itemSnapshot(item)
	result, err := transaction.ExecContext(ctx, `
UPDATE work_items
SET state = ?, revision = ?, started_at = ?, finished_at = ?, execution_ref = ?
WHERE ref = ? AND goal_ref = ? AND revision = ?`,
		string(snapshot.State),
		int64(snapshot.Revision),
		storedTime(snapshot.StartedAt),
		storedTime(snapshot.FinishedAt),
		nullableString(snapshot.ExecutionRef),
		snapshot.Ref,
		snapshot.GoalRef,
		int64(expected),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}

func itemSnapshot(item goal.WorkItem) goal.WorkItemSnapshot {
	startedAt, _ := item.StartedAt()
	finishedAt, _ := item.FinishedAt()
	executionRef, hasExecution := item.Execution()
	var executionValue string
	if hasExecution {
		executionValue = executionRef.String()
	}
	parent, hasParent := item.Parent()
	var parentValue string
	if hasParent {
		parentValue = parent.String()
	}
	return goal.WorkItemSnapshot{
		Ref:            item.Ref().String(),
		GoalRef:        item.Goal().String(),
		ActorRef:       item.Actor().String(),
		ProjectRef:     item.Project().String(),
		Objective:      item.Objective(),
		PhaseKey:       item.Phase().String(),
		RoleKey:        item.Role().String(),
		ParentRef:      parentValue,
		OutputContract: item.OutputContract().Kind(),
		SkipReason: func() goal.WorkItemSkipReason {
			reason, _ := item.SkipReason()
			return reason
		}(),
		DependencyRefs: workItemRefStrings(item.Dependencies()),
		WriteSet:       writeScopeStrings(item.WriteSet()),
		SkillRefs:      refStrings(item.SkillRefs()),
		ToolRefs:       refStrings(item.ToolRefs()),
		CapabilityRefs: refStrings(item.CapabilityRefs()),
		State:          item.State(),
		Revision:       item.Revision(),
		CreatedAt:      item.CreatedAt(),
		StartedAt:      startedAt,
		FinishedAt:     finishedAt,
		ExecutionRef:   executionValue,
	}
}

func refStrings[T interface{ String() string }](refs []T) []string {
	values := make([]string, len(refs))
	for index, ref := range refs {
		values[index] = ref.String()
	}
	return values
}

func insertOrderedContractRefs(
	ctx context.Context,
	transaction *sql.Tx,
	table string,
	ownerGoalRef string,
	ownerRef string,
	kind string,
	values []string,
) error {
	var statement string
	switch table {
	case "goal_phase_contract_refs":
		statement = `INSERT INTO goal_phase_contract_refs(goal_ref, phase_ref, kind, value, position) VALUES (?, ?, ?, ?, ?)`
	case "work_item_requirement_refs":
		statement = `INSERT INTO work_item_requirement_refs(goal_ref, work_item_ref, kind, value, position) VALUES (?, ?, ?, ?, ?)`
	default:
		return fmt.Errorf("sqlite.contract_ref_table_invalid")
	}
	for position, value := range values {
		if _, err := transaction.ExecContext(ctx, statement, ownerGoalRef, ownerRef, kind, value, position); err != nil {
			return mapDatabaseError(err)
		}
	}
	return nil
}

func workItemRefStrings(refs []goal.WorkItemRef) []string {
	values := make([]string, len(refs))
	for index, ref := range refs {
		values[index] = ref.String()
	}
	return values
}

func writeScopeStrings(scopes []goal.WriteScope) []string {
	values := make([]string, len(scopes))
	for index, scope := range scopes {
		values[index] = scope.String()
	}
	return values
}

func insertAction(ctx context.Context, transaction *sql.Tx, action application.ActionRecord) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO outbox(
    ref, kind, goal_ref, work_item_ref, execution_ref,
    plan_generation, work_item_generation, available_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		action.Ref,
		string(action.Kind),
		action.GoalRef.String(),
		action.WorkItemRef.String(),
		action.ExecutionRef.String(),
		int64(action.PlanGeneration),
		int64(action.WorkItemGeneration),
		requiredTime(action.AvailableAt),
	)
	return mapDatabaseError(err)
}

func insertEvent(ctx context.Context, transaction *sql.Tx, event application.EventRecord) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO events(ref, kind, goal_ref, work_item_ref, execution_ref, occurred_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		event.Ref,
		event.Kind,
		event.GoalRef.String(),
		nullableString(event.WorkItemRef.String()),
		nullableString(event.ExecutionRef.String()),
		requiredTime(event.OccurredAt),
	)
	return mapDatabaseError(err)
}

func insertEvents(ctx context.Context, transaction *sql.Tx, events []application.EventRecord) error {
	for _, event := range events {
		if err := insertEvent(ctx, transaction, event); err != nil {
			return err
		}
	}
	return nil
}

func insertArtifact(ctx context.Context, transaction *sql.Tx, artifact application.ArtifactRecord) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO artifacts(ref, goal_ref, work_item_ref, digest, media_type, size, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		artifact.Stored.Ref.String(),
		artifact.GoalRef.String(),
		artifact.WorkItemRef.String(),
		artifact.Stored.Digest,
		artifact.Stored.MediaType,
		artifact.Stored.Size,
		requiredTime(artifact.CreatedAt),
	)
	return mapDatabaseError(err)
}

func insertAttestation(ctx context.Context, transaction *sql.Tx, attestation application.AttestationRecord) error {
	_, err := transaction.ExecContext(ctx, `
INSERT INTO attestations(
    ref, goal_ref, work_item_ref, execution_ref, artifact_ref, policy, accepted_at
) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		attestation.Ref.String(),
		attestation.GoalRef.String(),
		attestation.WorkItemRef.String(),
		attestation.ExecutionRef.String(),
		attestation.ArtifactRef.String(),
		attestation.Policy,
		requiredTime(attestation.AcceptedAt),
	)
	return mapDatabaseError(err)
}

func requireOneRow(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return mapDatabaseError(err)
	}
	if rows != 1 {
		return conflict(fmt.Errorf("sqlite.cas_conflict"))
	}
	return nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
