package sqlite

import (
	"context"
	"database/sql"
	"errors"
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
	if err := insertGoalPhases(ctx, transaction, snapshot, 0); err != nil {
		return err
	}
	governed := len(state.WorkItemAuthorities) == len(snapshot.WorkItems) && len(state.BudgetEnvelopes) == 3
	if err := insertWorkItems(ctx, transaction, snapshot, 0, governed); err != nil {
		return err
	}
	if err := insertWorkItemAuthorities(ctx, transaction, snapshot.Ref, state.WorkItemAuthorities); err != nil {
		return err
	}
	if len(state.BudgetEnvelopes) == 0 {
		for _, action := range state.Actions {
			if action.EffectIntentRef != "" {
				return invalid(fmt.Errorf("sqlite.budget_envelopes_missing"))
			}
		}
	}
	if err := insertBudgetEnvelopes(ctx, transaction, state.BudgetEnvelopes); err != nil {
		return err
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

func insertGoalPhases(
	ctx context.Context,
	transaction *sql.Tx,
	snapshot goal.GoalSnapshot,
	start int,
) error {
	if start < 0 || start > len(snapshot.Phases) {
		return invalid(fmt.Errorf("sqlite.goal_phase_start_invalid:%d", start))
	}
	for position := start; position < len(snapshot.Phases); position++ {
		phase := snapshot.Phases[position]
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
	return nil
}

func insertWorkItems(
	ctx context.Context,
	transaction *sql.Tx,
	snapshot goal.GoalSnapshot,
	start int,
	governed bool,
) error {
	if start < 0 || start > len(snapshot.WorkItems) {
		return invalid(fmt.Errorf("sqlite.work_item_start_invalid:%d", start))
	}
	schema, err := readWorkItemSchema(ctx, transaction)
	if err != nil {
		return err
	}
	for position := start; position < len(snapshot.WorkItems); position++ {
		if err := insertWorkItem(ctx, transaction, snapshot.WorkItems[position], position, schema, governed); err != nil {
			return err
		}
	}
	for _, item := range snapshot.WorkItems[start:] {
		if err := insertWorkItemRelations(ctx, transaction, item); err != nil {
			return err
		}
	}
	return nil
}

type workItemSchema struct{ handoff, controls, governance, council bool }

func readWorkItemSchema(ctx context.Context, source queryer) (workItemSchema, error) {
	var schema workItemSchema
	columns := []struct {
		name  string
		value *bool
	}{{"handoff_required", &schema.handoff}, {"control_sequence", &schema.controls},
		{"governance_version", &schema.governance}, {"council_policy", &schema.council}}
	for _, column := range columns {
		found, err := sqliteTableHasColumn(ctx, source, "work_items", column.name)
		if err != nil {
			return workItemSchema{}, mapDatabaseError(err)
		}
		*column.value = found
	}
	return schema, nil
}

func insertWorkItem(
	ctx context.Context, tx *sql.Tx, item goal.WorkItemSnapshot, position int, schema workItemSchema, governed bool,
) error {
	if item.HandoffRequired == nil {
		return invalid(fmt.Errorf("sqlite.work_item_handoff_required_missing:%s", item.Ref))
	}
	if !schema.handoff && *item.HandoffRequired {
		return invalid(fmt.Errorf("sqlite.work_item_handoff_schema_unsupported:%s", item.Ref))
	}
	governed = governed && schema.governance
	if governed && (!schema.handoff || !schema.controls) {
		return invalid(fmt.Errorf("sqlite.work_item_governance_schema_unsupported:%s", item.Ref))
	}
	query, arguments := workItemInsert(item, position, schema, governed)
	_, err := tx.ExecContext(ctx, query, arguments...)
	return mapDatabaseError(err)
}

func workItemInsert(item goal.WorkItemSnapshot, position int, schema workItemSchema, governed bool) (string, []any) {
	query := workItemControlledInsert
	arguments := []any{
		item.Ref, item.GoalRef, item.ActorRef, item.ProjectRef, item.Objective, item.PhaseKey,
		item.RoleKey, nullableString(item.ParentRef), string(item.OutputContract), string(item.SkipReason),
		string(item.InterruptCause), nullableString(item.ReworkOf), string(item.State), int64(item.Revision),
		storedBool(item.Paused), storedBool(item.CancelRequested), int64(item.ControlSequence), position,
		requiredTime(item.CreatedAt), storedTime(item.StartedAt), storedTime(item.InterruptedAt),
		storedTime(item.FinishedAt), nullableString(item.ExecutionRef),
	}
	if !schema.controls {
		query = workItemLegacyInsert
		arguments = []any{item.Ref, item.GoalRef, item.ActorRef, item.ProjectRef, item.Objective,
			item.PhaseKey, item.RoleKey, nullableString(item.ParentRef), string(item.OutputContract),
			string(item.SkipReason), string(item.State), int64(item.Revision), position,
			requiredTime(item.CreatedAt), storedTime(item.StartedAt), storedTime(item.FinishedAt), nullableString(item.ExecutionRef)}
	}
	if schema.handoff {
		query = map[bool]string{true: workItemControlledHandoffInsert, false: workItemLegacyHandoffInsert}[schema.controls]
		arguments = append(arguments, storedBool(*item.HandoffRequired))
	}
	if governed {
		resources := item.BudgetDemand.Resources
		query = workItemGovernedInsert
		arguments = append(arguments, item.BudgetDemand.Ref, resources.Tokens, resources.MoneyMicros,
			string(resources.Currency), resources.ActiveTimeNS, resources.ProcessSlots, resources.DiskBytes,
			string(item.SecurityCriticality), string(item.ReasoningEffort))
	}
	if schema.council {
		if governed {
			query = workItemGovernedCouncilInsert
		} else {
			query = workItemControlledHandoffCouncilInsert
		}
		arguments = append(arguments, string(item.CouncilPolicy))
	}
	return query, arguments
}

const workItemControlledInsert = `INSERT INTO work_items(
ref,goal_ref,actor_ref,project_ref,objective,phase_key,role_key,parent_ref,output_contract,skip_reason,
interrupt_cause,rework_of,state,revision,paused,cancel_requested,control_sequence,position,created_at,
started_at,interrupted_at,finished_at,execution_ref) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
const workItemControlledHandoffInsert = `INSERT INTO work_items(
ref,goal_ref,actor_ref,project_ref,objective,phase_key,role_key,parent_ref,output_contract,skip_reason,
interrupt_cause,rework_of,state,revision,paused,cancel_requested,control_sequence,position,created_at,
started_at,interrupted_at,finished_at,execution_ref,handoff_required) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
const workItemControlledHandoffCouncilInsert = `INSERT INTO work_items(
ref,goal_ref,actor_ref,project_ref,objective,phase_key,role_key,parent_ref,output_contract,skip_reason,
interrupt_cause,rework_of,state,revision,paused,cancel_requested,control_sequence,position,created_at,
started_at,interrupted_at,finished_at,execution_ref,handoff_required,council_policy) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
const workItemLegacyInsert = `INSERT INTO work_items(
ref,goal_ref,actor_ref,project_ref,objective,phase_key,role_key,parent_ref,output_contract,skip_reason,
state,revision,position,created_at,started_at,finished_at,execution_ref) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
const workItemLegacyHandoffInsert = `INSERT INTO work_items(
ref,goal_ref,actor_ref,project_ref,objective,phase_key,role_key,parent_ref,output_contract,skip_reason,
state,revision,position,created_at,started_at,finished_at,execution_ref,handoff_required) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
const workItemGovernedInsert = `INSERT INTO work_items(
ref,goal_ref,actor_ref,project_ref,objective,phase_key,role_key,parent_ref,output_contract,skip_reason,
interrupt_cause,rework_of,state,revision,paused,cancel_requested,control_sequence,position,created_at,
started_at,interrupted_at,finished_at,execution_ref,handoff_required,governance_version,budget_demand_ref,
budget_tokens,budget_money_micros,budget_currency,budget_active_time_ns,budget_process_slots,budget_disk_bytes,
security_criticality,reasoning_effort) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,1,?,?,?,?,?,?,?,?,?)`
const workItemGovernedCouncilInsert = `INSERT INTO work_items(
ref,goal_ref,actor_ref,project_ref,objective,phase_key,role_key,parent_ref,output_contract,skip_reason,
interrupt_cause,rework_of,state,revision,paused,cancel_requested,control_sequence,position,created_at,
started_at,interrupted_at,finished_at,execution_ref,handoff_required,governance_version,budget_demand_ref,
budget_tokens,budget_money_micros,budget_currency,budget_active_time_ns,budget_process_slots,budget_disk_bytes,
security_criticality,reasoning_effort,council_policy) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,1,?,?,?,?,?,?,?,?,?,?)`

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
	controlsPersisted, err := sqliteTableHasColumn(ctx, transaction, "goals", "control_sequence")
	if err != nil {
		return mapDatabaseError(err)
	}
	query := `
INSERT INTO goals(
    ref, request_ref, request_fingerprint, requested_by_ref, app_spec_ref, actor_ref, project_ref, state, revision,
    paused, cancel_requested, control_sequence, created_at, started_at, closed_at, plan_generation
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	arguments := []any{
		snapshot.Ref,
		requestRef,
		requestFingerprint,
		requestedBy.String(),
		snapshot.AppSpec.Ref,
		snapshot.ActorRef,
		snapshot.ProjectRef,
		string(snapshot.State),
		int64(snapshot.Revision),
		storedBool(snapshot.Paused),
		storedBool(snapshot.CancelRequested),
		int64(snapshot.ControlSequence),
		requiredTime(snapshot.CreatedAt),
		storedTime(snapshot.StartedAt),
		storedTime(snapshot.ClosedAt),
		int64(snapshot.PlanGeneration),
	}
	if !controlsPersisted {
		query = `
INSERT INTO goals(
    ref, request_ref, request_fingerprint, requested_by_ref, app_spec_ref,
    actor_ref, project_ref, state, revision, created_at, started_at, closed_at, plan_generation
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		arguments = []any{
			snapshot.Ref, requestRef, requestFingerprint, requestedBy.String(), snapshot.AppSpec.Ref,
			snapshot.ActorRef, snapshot.ProjectRef, string(snapshot.State), int64(snapshot.Revision),
			requiredTime(snapshot.CreatedAt), storedTime(snapshot.StartedAt), storedTime(snapshot.ClosedAt),
			int64(snapshot.PlanGeneration),
		}
	}
	if _, err := transaction.ExecContext(ctx, query, arguments...); err != nil {
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
	schema, err := readExecutionSchema(ctx, transaction)
	if err != nil {
		return err
	}
	query := `
INSERT INTO executions(
    ref, goal_ref, work_item_ref, attempt_no, max_execution_attempts,
    replaces_execution_ref, plan_generation, app_spec_generation, spec_hash,
    state, artifact_media_type, idempotency_key, max_output_bytes,
    provider_ref, model_ref, agent_ref, external_ref, created_at, deadline_at,
    started_at, provider_accepted_at, last_observed_at, provider_observed_at,
    finished_at, failure_code, recipient_mailbox_retired
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	arguments := executionInsertArguments(execution)
	if !schema.mailbox {
		query = `
INSERT INTO executions(
    ref, goal_ref, work_item_ref, attempt_no, max_execution_attempts,
    replaces_execution_ref, plan_generation, app_spec_generation, spec_hash,
    state, artifact_media_type, idempotency_key, max_output_bytes,
    provider_ref, model_ref, agent_ref, external_ref, created_at, deadline_at,
    started_at, provider_accepted_at, last_observed_at, provider_observed_at,
    finished_at, failure_code
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		arguments = arguments[:len(arguments)-1]
	}
	version := int64(0)
	if schema.governance {
		if execution.BudgetReservationRef != "" || execution.EffectIntentRef != "" || execution.LaunchReceiptRef != "" {
			version = 1
		}
		query = `
INSERT INTO executions(
    ref, goal_ref, work_item_ref, attempt_no, max_execution_attempts,
    replaces_execution_ref, plan_generation, app_spec_generation, spec_hash,
    state, artifact_media_type, idempotency_key, max_output_bytes,
    provider_ref, model_ref, agent_ref, external_ref, created_at, deadline_at,
    started_at, provider_accepted_at, last_observed_at, provider_observed_at,
    finished_at, failure_code, recipient_mailbox_retired, governance_version,
    budget_reservation_ref, effect_intent_ref, launch_receipt_ref
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		arguments = append(arguments, version, nullableString(execution.BudgetReservationRef),
			nullableString(execution.EffectIntentRef), nullableString(execution.LaunchReceiptRef))
	}
	if schema.workspace {
		query = `
INSERT INTO executions(
    ref, goal_ref, work_item_ref, attempt_no, max_execution_attempts,
    replaces_execution_ref, plan_generation, app_spec_generation, spec_hash,
    repository_ref, execution_workspace_ref, state, artifact_media_type, idempotency_key, max_output_bytes,
    provider_ref, model_ref, agent_ref, external_ref, governance_version, budget_reservation_ref,
    effect_intent_ref, launch_receipt_ref, created_at, deadline_at, started_at, provider_accepted_at,
    last_observed_at, provider_observed_at, finished_at, failure_code, recipient_mailbox_retired
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		arguments = []any{execution.Ref.String(), execution.GoalRef.String(), execution.WorkItemRef.String(),
			int64(execution.AttemptNo), int64(execution.MaxExecutionAttempts), nullableString(execution.ReplacesExecutionRef.String()),
			int64(execution.PlanGeneration), int64(execution.AppSpecGeneration), execution.SpecHash,
			execution.RepositoryRef.String(), execution.ExecutionWorkspaceRef.String(), string(execution.State), execution.ArtifactMediaType,
			execution.IdempotencyKey, execution.MaxOutputBytes, execution.ProviderRef, execution.ModelRef, execution.AgentRef, execution.ExternalRef,
			version, nullableString(execution.BudgetReservationRef), nullableString(execution.EffectIntentRef), nullableString(execution.LaunchReceiptRef),
			requiredTime(execution.CreatedAt), storedTime(execution.DeadlineAt), storedTime(execution.StartedAt), storedTime(execution.ProviderAcceptedAt),
			storedTime(execution.LastObservedAt), storedTime(execution.ProviderObservedAt), storedTime(execution.FinishedAt), execution.FailureCode,
			storedBool(execution.RecipientMailboxRetired)}
	}
	if schema.reviews {
		query = `
INSERT INTO executions(
 ref,goal_ref,work_item_ref,attempt_no,max_execution_attempts,replaces_execution_ref,
 plan_generation,app_spec_generation,spec_hash,repository_ref,execution_workspace_ref,state,purpose,review_subject_digest,
 artifact_media_type,idempotency_key,max_output_bytes,provider_ref,model_ref,agent_ref,external_ref,
 governance_version,budget_reservation_ref,effect_intent_ref,launch_receipt_ref,created_at,deadline_at,started_at,
 provider_accepted_at,last_observed_at,provider_observed_at,finished_at,failure_code,recipient_mailbox_retired)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
		arguments = []any{execution.Ref.String(), execution.GoalRef.String(), execution.WorkItemRef.String(),
			int64(execution.AttemptNo), int64(execution.MaxExecutionAttempts), nullableString(execution.ReplacesExecutionRef.String()),
			int64(execution.PlanGeneration), int64(execution.AppSpecGeneration), execution.SpecHash,
			execution.RepositoryRef.String(), execution.ExecutionWorkspaceRef.String(), string(execution.State),
			executionPurposeValue(execution), execution.ReviewSubjectDigest, execution.ArtifactMediaType, execution.IdempotencyKey,
			execution.MaxOutputBytes, execution.ProviderRef, execution.ModelRef, execution.AgentRef, execution.ExternalRef,
			version, nullableString(execution.BudgetReservationRef), nullableString(execution.EffectIntentRef),
			nullableString(execution.LaunchReceiptRef), requiredTime(execution.CreatedAt), storedTime(execution.DeadlineAt),
			storedTime(execution.StartedAt), storedTime(execution.ProviderAcceptedAt), storedTime(execution.LastObservedAt),
			storedTime(execution.ProviderObservedAt), storedTime(execution.FinishedAt), execution.FailureCode,
			storedBool(execution.RecipientMailboxRetired)}
	}
	if schema.council {
		query = `
INSERT INTO executions(
 ref,goal_ref,work_item_ref,attempt_no,max_execution_attempts,replaces_execution_ref,
 plan_generation,app_spec_generation,spec_hash,repository_ref,execution_workspace_ref,state,purpose,review_subject_digest,council_subject_digest,
 artifact_media_type,idempotency_key,max_output_bytes,provider_ref,model_ref,agent_ref,external_ref,
 governance_version,budget_reservation_ref,effect_intent_ref,launch_receipt_ref,created_at,deadline_at,started_at,
 provider_accepted_at,last_observed_at,provider_observed_at,finished_at,failure_code,recipient_mailbox_retired)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
		arguments = []any{execution.Ref.String(), execution.GoalRef.String(), execution.WorkItemRef.String(),
			int64(execution.AttemptNo), int64(execution.MaxExecutionAttempts), nullableString(execution.ReplacesExecutionRef.String()),
			int64(execution.PlanGeneration), int64(execution.AppSpecGeneration), execution.SpecHash,
			execution.RepositoryRef.String(), execution.ExecutionWorkspaceRef.String(), string(execution.State),
			executionPurposeValue(execution), execution.ReviewSubjectDigest, string(execution.CouncilSubjectDigest),
			execution.ArtifactMediaType, execution.IdempotencyKey, execution.MaxOutputBytes, execution.ProviderRef,
			execution.ModelRef, execution.AgentRef, execution.ExternalRef, version, nullableString(execution.BudgetReservationRef),
			nullableString(execution.EffectIntentRef), nullableString(execution.LaunchReceiptRef), requiredTime(execution.CreatedAt),
			storedTime(execution.DeadlineAt), storedTime(execution.StartedAt), storedTime(execution.ProviderAcceptedAt),
			storedTime(execution.LastObservedAt), storedTime(execution.ProviderObservedAt), storedTime(execution.FinishedAt),
			execution.FailureCode, storedBool(execution.RecipientMailboxRetired)}
	}
	_, err = transaction.ExecContext(ctx, query, arguments...)
	return mapDatabaseError(err)
}

func executionInsertArguments(execution application.ExecutionRecord) []any {
	return []any{
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
		storedBool(execution.RecipientMailboxRetired),
	}
}

type executionSchema struct{ mailbox, governance, workspace, reviews, council, session bool }

func readExecutionSchema(ctx context.Context, source queryer) (executionSchema, error) {
	var schema executionSchema
	for _, column := range []struct {
		name  string
		value *bool
	}{{"recipient_mailbox_retired", &schema.mailbox}, {"governance_version", &schema.governance},
		{"execution_workspace_ref", &schema.workspace}, {"purpose", &schema.reviews},
		{"council_subject_digest", &schema.council}, {"execution_session_ref", &schema.session}} {
		found, err := sqliteTableHasColumn(ctx, source, "executions", column.name)
		if err != nil {
			return executionSchema{}, mapDatabaseError(err)
		}
		*column.value = found
	}
	return schema, nil
}

func updateExecutionCAS(
	ctx context.Context,
	transaction *sql.Tx,
	execution application.ExecutionRecord,
	expected application.ExecutionState,
) error {
	if err := requireExecutionReviewIdentity(ctx, transaction, execution); err != nil {
		return err
	}
	governancePersisted, err := sqliteTableHasColumn(ctx, transaction, "executions", "governance_version")
	if err != nil {
		return mapDatabaseError(err)
	}
	if !governancePersisted {
		return updateLegacyExecutionCAS(ctx, transaction, execution, expected)
	}
	if expected == application.ExecutionDispatching && execution.State == application.ExecutionRunning {
		return acceptExecutionCAS(ctx, transaction, execution)
	}
	result, err := transaction.ExecContext(ctx, `
UPDATE executions
SET state = ?, deadline_at = ?, started_at = ?, provider_accepted_at = ?, last_observed_at = ?,
    provider_observed_at = ?, finished_at = ?, failure_code = ?, recipient_mailbox_retired = ?,
    governance_version = CASE WHEN ? IS NULL AND ? IS NULL AND ? IS NULL THEN 0 ELSE 1 END,
    budget_reservation_ref = ?, effect_intent_ref = ?, launch_receipt_ref = ?
WHERE ref = ? AND goal_ref = ? AND work_item_ref = ? AND state = ?
  AND provider_ref = ? AND model_ref = ? AND agent_ref = ? AND external_ref = ?
  AND attempt_no = ? AND max_execution_attempts = ?
  AND replaces_execution_ref IS ?
  AND plan_generation = ? AND app_spec_generation = ? AND spec_hash = ?
  AND artifact_media_type = ? AND idempotency_key = ?
  AND max_output_bytes = ? AND created_at = ?
  AND (budget_reservation_ref IS NULL OR budget_reservation_ref IS ?)
  AND (effect_intent_ref IS NULL OR effect_intent_ref IS ?)
  AND (launch_receipt_ref IS NULL OR launch_receipt_ref IS ?)`,
		string(execution.State),
		storedTime(execution.DeadlineAt),
		storedTime(execution.StartedAt),
		storedTime(execution.ProviderAcceptedAt),
		storedTime(execution.LastObservedAt),
		storedTime(execution.ProviderObservedAt),
		storedTime(execution.FinishedAt),
		execution.FailureCode,
		storedBool(execution.RecipientMailboxRetired),
		nullableString(execution.BudgetReservationRef), nullableString(execution.EffectIntentRef),
		nullableString(execution.LaunchReceiptRef), nullableString(execution.BudgetReservationRef),
		nullableString(execution.EffectIntentRef), nullableString(execution.LaunchReceiptRef),
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
		nullableString(execution.BudgetReservationRef), nullableString(execution.EffectIntentRef),
		nullableString(execution.LaunchReceiptRef),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return err
	}
	if err := updateExecutionWorkspaceCAS(ctx, transaction, execution); err != nil {
		return err
	}
	return updateExecutionSessionCAS(ctx, transaction, execution)
}

func requireExecutionReviewIdentity(ctx context.Context, source queryer, execution application.ExecutionRecord) error {
	persisted, err := sqliteTableHasColumn(ctx, source, "executions", "purpose")
	if err != nil || !persisted {
		return mapDatabaseError(err)
	}
	councilPersisted, err := sqliteTableHasColumn(ctx, source, "executions", "council_subject_digest")
	if err != nil {
		return mapDatabaseError(err)
	}
	query := `SELECT COUNT(*) FROM executions WHERE ref=? AND purpose=? AND review_subject_digest=?`
	arguments := []any{execution.Ref.String(), executionPurposeValue(execution), execution.ReviewSubjectDigest}
	if councilPersisted {
		query += ` AND council_subject_digest=?`
		arguments = append(arguments, string(execution.CouncilSubjectDigest))
	}
	var count int
	if err := source.QueryRowContext(ctx, query, arguments...).Scan(&count); err != nil {
		return mapDatabaseError(err)
	}
	if count != 1 {
		return conflict(errors.New("sqlite.execution_review_identity_mismatch"))
	}
	return nil
}

func executionPurposeValue(execution application.ExecutionRecord) string {
	if execution.Purpose == "" {
		return string(application.ExecutionPurposeWork)
	}
	return string(execution.Purpose)
}

func updateExecutionWorkspaceCAS(
	ctx context.Context, transaction *sql.Tx, execution application.ExecutionRecord,
) error {
	workspacePersisted, err := sqliteTableHasColumn(ctx, transaction, "executions", "execution_workspace_ref")
	if err != nil {
		return mapDatabaseError(err)
	}
	if !workspacePersisted {
		return nil
	}
	updated, err := transaction.ExecContext(ctx, `
UPDATE executions SET repository_ref=?, execution_workspace_ref=?
WHERE ref=? AND goal_ref=? AND work_item_ref=? AND state=?`,
		execution.RepositoryRef.String(), execution.ExecutionWorkspaceRef.String(), execution.Ref.String(),
		execution.GoalRef.String(), execution.WorkItemRef.String(), string(execution.State))
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(updated)
}

func updateExecutionSessionCAS(
	ctx context.Context, transaction *sql.Tx, execution application.ExecutionRecord,
) error {
	persisted, err := sqliteTableHasColumn(ctx, transaction, "executions", "execution_session_ref")
	if err != nil || !persisted {
		return mapDatabaseError(err)
	}
	updated, err := transaction.ExecContext(ctx, `
UPDATE executions SET execution_session_ref=?
WHERE ref=? AND goal_ref=? AND work_item_ref=? AND state=?
 AND (execution_session_ref='' OR execution_session_ref=?)`,
		execution.ExecutionSessionRef.String(), execution.Ref.String(), execution.GoalRef.String(),
		execution.WorkItemRef.String(), string(execution.State), execution.ExecutionSessionRef.String())
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(updated)
}

func acceptExecutionCAS(ctx context.Context, transaction *sql.Tx, execution application.ExecutionRecord) error {
	result, err := transaction.ExecContext(ctx, `
UPDATE executions
SET state = ?, provider_ref = ?, model_ref = ?, agent_ref = ?, external_ref = ?,
    deadline_at = ?, started_at = ?, provider_accepted_at = ?, last_observed_at = ?,
    provider_observed_at = ?, finished_at = ?, failure_code = ?, recipient_mailbox_retired = ?,
    governance_version = CASE WHEN budget_reservation_ref IS NULL AND effect_intent_ref IS NULL THEN 0 ELSE 1 END, launch_receipt_ref = ?
WHERE ref = ? AND goal_ref = ? AND work_item_ref = ? AND state = 'dispatching'
  AND provider_ref = '' AND model_ref = '' AND agent_ref = '' AND external_ref = ''
  AND attempt_no = ? AND max_execution_attempts = ?
  AND replaces_execution_ref IS ?
  AND plan_generation = ? AND app_spec_generation = ? AND spec_hash = ?
  AND artifact_media_type = ? AND idempotency_key = ?
  AND max_output_bytes = ? AND created_at = ?
  AND budget_reservation_ref IS ? AND effect_intent_ref IS ?
  AND launch_receipt_ref IS NULL`,
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
		storedBool(execution.RecipientMailboxRetired),
		nullableString(execution.LaunchReceiptRef),
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
		nullableString(execution.BudgetReservationRef), nullableString(execution.EffectIntentRef),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return err
	}
	return updateExecutionSessionCAS(ctx, transaction, execution)
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
SET state = ?, revision = ?, paused = ?, cancel_requested = ?, control_sequence = ?,
    started_at = ?, closed_at = ?, plan_generation = ?
WHERE ref = ? AND revision = ?`,
		string(snapshot.State),
		int64(snapshot.Revision),
		storedBool(snapshot.Paused),
		storedBool(snapshot.CancelRequested),
		int64(snapshot.ControlSequence),
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
SET state = ?, revision = ?, paused = ?, cancel_requested = ?, control_sequence = ?,
    started_at = ?, interrupted_at = ?, finished_at = ?, execution_ref = ?,
    skip_reason = ?, interrupt_cause = ?, rework_of = ?
WHERE ref = ? AND goal_ref = ?`,
		string(snapshot.State), int64(snapshot.Revision), storedBool(snapshot.Paused),
		storedBool(snapshot.CancelRequested), int64(snapshot.ControlSequence), storedTime(snapshot.StartedAt),
		storedTime(snapshot.InterruptedAt), storedTime(snapshot.FinishedAt), nullableString(snapshot.ExecutionRef),
		string(snapshot.SkipReason), string(snapshot.InterruptCause), nullableString(snapshot.ReworkOf),
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
SET state = ?, revision = ?, paused = ?, cancel_requested = ?, control_sequence = ?,
    started_at = ?, interrupted_at = ?, finished_at = ?, execution_ref = ?,
    skip_reason = ?, interrupt_cause = ?, rework_of = ?
WHERE ref = ? AND goal_ref = ? AND revision = ?`,
		string(snapshot.State),
		int64(snapshot.Revision),
		storedBool(snapshot.Paused),
		storedBool(snapshot.CancelRequested),
		int64(snapshot.ControlSequence),
		storedTime(snapshot.StartedAt),
		storedTime(snapshot.InterruptedAt),
		storedTime(snapshot.FinishedAt),
		nullableString(snapshot.ExecutionRef),
		string(snapshot.SkipReason),
		string(snapshot.InterruptCause),
		nullableString(snapshot.ReworkOf),
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
	interruptedAt, _ := item.InterruptedAt()
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
	handoffRequired := item.HandoffRequired()
	interruptCause, _ := item.InterruptCause()
	reworkOf, hasRework := item.ReworkOf()
	var reworkValue string
	if hasRework {
		reworkValue = reworkOf.String()
	}
	return goal.WorkItemSnapshot{
		Ref:             item.Ref().String(),
		GoalRef:         item.Goal().String(),
		ActorRef:        item.Actor().String(),
		ProjectRef:      item.Project().String(),
		Objective:       item.Objective(),
		PhaseKey:        item.Phase().String(),
		RoleKey:         item.Role().String(),
		ParentRef:       parentValue,
		HandoffRequired: &handoffRequired,
		OutputContract:  item.OutputContract().Kind(),
		BudgetDemand:    item.BudgetDemand(), SecurityCriticality: item.SecurityCriticality(),
		ReasoningEffort: item.ReasoningEffort(),
		SkipReason: func() goal.WorkItemSkipReason {
			reason, _ := item.SkipReason()
			return reason
		}(),
		InterruptCause:  interruptCause,
		ReworkOf:        reworkValue,
		DependencyRefs:  workItemRefStrings(item.Dependencies()),
		WriteSet:        writeScopeStrings(item.WriteSet()),
		SkillRefs:       refStrings(item.SkillRefs()),
		ToolRefs:        refStrings(item.ToolRefs()),
		CapabilityRefs:  refStrings(item.CapabilityRefs()),
		State:           item.State(),
		Revision:        item.Revision(),
		Paused:          item.Paused(),
		CancelRequested: item.CancelRequested(),
		ControlSequence: item.ControlSequence(),
		CreatedAt:       item.CreatedAt(),
		StartedAt:       startedAt,
		InterruptedAt:   interruptedAt,
		FinishedAt:      finishedAt,
		ExecutionRef:    executionValue,
	}
}

func storedBool(value bool) int64 {
	if value {
		return 1
	}
	return 0
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
	controlColumn, err := sqliteTableHasColumn(ctx, transaction, "outbox", "control_ref")
	if err != nil {
		return mapDatabaseError(err)
	}
	if !controlColumn {
		if action.ControlRef != "" {
			return invalid(fmt.Errorf("sqlite.action_control_schema_unsupported"))
		}
		_, err = transaction.ExecContext(ctx, `
INSERT INTO outbox(
    ref, kind, goal_ref, work_item_ref, execution_ref,
    plan_generation, work_item_generation, available_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			action.Ref, string(action.Kind), action.GoalRef.String(), action.WorkItemRef.String(),
			action.ExecutionRef.String(), int64(action.PlanGeneration),
			int64(action.WorkItemGeneration), requiredTime(action.AvailableAt),
		)
		return mapDatabaseError(err)
	}
	governanceColumn, err := sqliteTableHasColumn(ctx, transaction, "outbox", "governance_version")
	if err != nil {
		return mapDatabaseError(err)
	}
	if governanceColumn {
		version := int64(0)
		if action.EffectIntentRef != "" {
			version = 1
			if err := insertEffectAdmission(ctx, transaction, action); err != nil {
				return err
			}
		}
		workspaceColumns, columnErr := sqliteTableHasColumn(ctx, transaction, "outbox", "change_ref")
		if columnErr != nil {
			return mapDatabaseError(columnErr)
		}
		if workspaceColumns {
			councilColumns, councilErr := sqliteTableHasColumn(ctx, transaction, "outbox", "council_subject_digest")
			if councilErr != nil {
				return mapDatabaseError(councilErr)
			}
			if councilColumns {
				subject, decisionRef, decisionDigest, skipRef, skipDigest := storedCouncilResolution(action.CouncilResolution)
				resolutionKind := ""
				if action.CouncilResolution != nil {
					resolutionKind = "accepted_round"
					if action.CouncilResolution.SkipRef != "" {
						resolutionKind = "skip"
					}
				}
				_, err = transaction.ExecContext(ctx, `
INSERT INTO outbox(
    ref, kind, goal_ref, work_item_ref, execution_ref, control_ref, change_ref, expected_target_oid,
    plan_generation, work_item_generation, available_at, governance_version, effect_intent_ref, review_gate_digest,
    council_subject_digest,council_resolution_kind,council_decision_ref,council_decision_digest,council_skip_ref,council_skip_digest
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					action.Ref, string(action.Kind), action.GoalRef.String(), action.WorkItemRef.String(), action.ExecutionRef.String(),
					nullableString(action.ControlRef), action.ChangeRef.String(), action.ExpectedTargetOID,
					int64(action.PlanGeneration), int64(action.WorkItemGeneration), requiredTime(action.AvailableAt),
					version, nullableString(action.EffectIntentRef), action.ReviewGateDigest, subject, resolutionKind,
					decisionRef, decisionDigest, skipRef, skipDigest)
				return mapDatabaseError(err)
			}
			_, err = transaction.ExecContext(ctx, `
INSERT INTO outbox(
    ref, kind, goal_ref, work_item_ref, execution_ref, control_ref, change_ref, expected_target_oid,
    plan_generation, work_item_generation, available_at, governance_version, effect_intent_ref, review_gate_digest
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				action.Ref, string(action.Kind), action.GoalRef.String(), action.WorkItemRef.String(), action.ExecutionRef.String(),
				nullableString(action.ControlRef), action.ChangeRef.String(), action.ExpectedTargetOID, int64(action.PlanGeneration), int64(action.WorkItemGeneration),
				requiredTime(action.AvailableAt), version, nullableString(action.EffectIntentRef), action.ReviewGateDigest)
			return mapDatabaseError(err)
		}
		_, err = transaction.ExecContext(ctx, `
INSERT INTO outbox(
    ref, kind, goal_ref, work_item_ref, execution_ref, control_ref,
    plan_generation, work_item_generation, available_at,
    governance_version, effect_intent_ref
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			action.Ref, string(action.Kind), action.GoalRef.String(), action.WorkItemRef.String(),
			action.ExecutionRef.String(), nullableString(action.ControlRef), int64(action.PlanGeneration),
			int64(action.WorkItemGeneration), requiredTime(action.AvailableAt), version,
			nullableString(action.EffectIntentRef),
		)
		return mapDatabaseError(err)
	}
	_, err = transaction.ExecContext(ctx, `
INSERT INTO outbox(
    ref, kind, goal_ref, work_item_ref, execution_ref, control_ref,
    plan_generation, work_item_generation, available_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		action.Ref,
		string(action.Kind),
		action.GoalRef.String(),
		action.WorkItemRef.String(),
		action.ExecutionRef.String(),
		nullableString(action.ControlRef),
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
	if err := validateArtifactRecord(artifact); err != nil {
		return invalid(err)
	}
	if !validCanonicalHash(artifact.Stored.Digest) ||
		artifact.Stored.Ref.String() != "artifact:sha256:"+artifact.Stored.Digest {
		return invalid(fmt.Errorf("sqlite.artifact_cas_identity_invalid"))
	}
	_, err := transaction.ExecContext(ctx, `
INSERT INTO artifacts(ref, goal_ref, work_item_ref, digest, media_type, size, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(goal_ref,ref) DO NOTHING`,
		artifact.Stored.Ref.String(),
		artifact.GoalRef.String(),
		artifact.WorkItemRef.String(),
		artifact.Stored.Digest,
		artifact.Stored.MediaType,
		artifact.Stored.Size,
		requiredTime(artifact.CreatedAt),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	var digest, mediaType string
	var size int64
	if err := transaction.QueryRowContext(ctx, `
SELECT digest,media_type,size FROM artifacts WHERE goal_ref=? AND ref=?`,
		artifact.GoalRef.String(), artifact.Stored.Ref.String()).Scan(&digest, &mediaType, &size); err != nil {
		return mapDatabaseError(err)
	}
	if digest != artifact.Stored.Digest || mediaType != artifact.Stored.MediaType || size != artifact.Stored.Size {
		return invalid(fmt.Errorf("sqlite.artifact_cas_identity_conflict"))
	}
	var duplicate int
	if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*) FROM artifact_occurrences
WHERE kind=? AND goal_ref=? AND work_item_ref=? AND execution_ref=? AND artifact_ref=?
 AND execution_attempt=? AND plan_generation=? AND work_item_generation=?
 AND app_spec_generation=? AND spec_hash=?`,
		string(artifact.Kind), artifact.GoalRef.String(), artifact.WorkItemRef.String(),
		artifact.ExecutionRef.String(), artifact.Stored.Ref.String(), int64(artifact.ExecutionAttempt),
		int64(artifact.PlanGeneration), int64(artifact.WorkItemGeneration),
		int64(artifact.AppSpecGeneration), artifact.SpecHash).Scan(&duplicate); err != nil {
		return mapDatabaseError(err)
	}
	if duplicate != 0 {
		return conflict(fmt.Errorf("sqlite.artifact_occurrence_duplicate"))
	}
	_, err = transaction.ExecContext(ctx, `
INSERT INTO artifact_occurrences(
 occurrence_ref,kind,goal_ref,work_item_ref,execution_ref,artifact_ref,
 execution_attempt,plan_generation,work_item_generation,app_spec_generation,spec_hash,created_at
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		artifact.OccurrenceRef, string(artifact.Kind), artifact.GoalRef.String(),
		artifact.WorkItemRef.String(), artifact.ExecutionRef.String(), artifact.Stored.Ref.String(),
		int64(artifact.ExecutionAttempt), int64(artifact.PlanGeneration), int64(artifact.WorkItemGeneration),
		int64(artifact.AppSpecGeneration), artifact.SpecHash, requiredTime(artifact.CreatedAt))
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
