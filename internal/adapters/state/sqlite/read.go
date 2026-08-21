package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
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
	header, err := readGoalHeader(ctx, source, goalValue)
	if err != nil {
		return application.GoalRecord{}, err
	}
	data, err := readGoalData(ctx, source, goalValue, header)
	if err != nil {
		return application.GoalRecord{}, err
	}
	governanceData, err := readGoalGovernance(ctx, source, header.snapshot.ProjectRef, goalValue)
	if err != nil {
		return application.GoalRecord{}, err
	}
	workspaceFacts, err := readWorkspaceGitFacts(ctx, source, goalValue)
	if err != nil {
		return application.GoalRecord{}, err
	}
	reviews, err := readReviewRecords(ctx, source, goalValue)
	if err != nil {
		return application.GoalRecord{}, err
	}
	mailboxes, err := readGoalMailboxRecords(ctx, source, goalValue)
	if err != nil {
		return application.GoalRecord{}, err
	}
	councilRecords, err := readCouncilGoalRecords(ctx, source, goalValue)
	if err != nil {
		return application.GoalRecord{}, err
	}
	record := application.GoalRecord{
		RequestRef: header.requestRef, RequestFingerprint: header.requestFingerprint,
		RequestedBy: data.requestedBy, Goal: data.goal, Executions: data.executions,
		Artifacts: data.artifacts, Attestations: data.attestations, Controls: data.controls,
		BudgetEnvelopes: governanceData.envelopes, BudgetReservations: governanceData.reservations,
		BudgetSettlements: governanceData.settlements, WorkItemAuthorities: governanceData.authorities,
		EffectIntents: governanceData.intents, EffectApprovals: governanceData.approvals,
		EffectAttempts: governanceData.attempts, EffectAttemptOutcomes: governanceData.outcomes,
		EffectReceipts:      governanceData.receipts,
		ConsumptionReceipts: data.consumptionReceipts,
		WorkspaceBindings:   workspaceFacts.bindings, ChangeSets: workspaceFacts.changes,
		MergeObservations: workspaceFacts.observations, IntegrationReceipts: workspaceFacts.receipts,
		Reviews: reviews, Mailboxes: mailboxes,
		CouncilRounds: councilRecords.rounds, CouncilFacts: councilRecords.facts,
		CouncilDecisions: councilRecords.decisions, CouncilSkips: councilRecords.skips,
	}
	if err := validateGoalRecordConsistency(record, goalValue); err != nil {
		return application.GoalRecord{}, invalid(err)
	}
	return record, nil
}

func readReviewRecords(ctx context.Context, source queryer, goalValue string) ([]application.ReviewRecord, error) {
	persisted, err := sqliteTableHasColumn(ctx, source, "review_records", "subject_digest")
	if err != nil || !persisted {
		return nil, mapDatabaseError(err)
	}
	rows, err := source.QueryContext(ctx, `SELECT ref,goal_ref,work_item_ref,change_set_ref,subject_digest,
role,verdict,reviewer_execution_ref,reviewer_execution_attempt,launch_receipt_ref,principal_ref,agent_ref,
external_ref,assessment_artifact_ref,assessment_digest,recorded_at FROM review_records WHERE goal_ref=? ORDER BY recorded_at,ref`, goalValue)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var records []application.ReviewRecord
	for rows.Next() {
		var record application.ReviewRecord
		var goalValue, itemValue, changeValue, executionValue, principalValue string
		var attempt, recorded int64
		if err := rows.Scan(&record.Ref, &goalValue, &itemValue, &changeValue, &record.SubjectDigest,
			&record.Role, &record.Verdict, &executionValue, &attempt, &record.LaunchReceiptRef,
			&principalValue, &record.AgentRef, &record.ExternalRef, &record.AssessmentArtifactRef, &record.AssessmentDigest, &recorded); err != nil {
			return nil, mapDatabaseError(err)
		}
		if attempt <= 0 {
			return nil, invalid(errors.New("sqlite.review_attempt_invalid"))
		}
		if record.GoalRef, err = goal.NewGoalRef(goalValue); err != nil {
			return nil, invalid(err)
		}
		if record.WorkItemRef, err = goal.NewWorkItemRef(itemValue); err != nil {
			return nil, invalid(err)
		}
		if record.ChangeSetRef, err = ports.NewChangeSetRef(changeValue); err != nil {
			return nil, invalid(err)
		}
		if record.ReviewerExecutionRef, err = goal.NewExecutionRef(executionValue); err != nil {
			return nil, invalid(err)
		}
		if record.PrincipalRef, err = identity.NewPrincipalRef(principalValue); err != nil {
			return nil, invalid(err)
		}
		record.ReviewerExecutionAttempt, record.RecordedAt = uint64(attempt), time.Unix(0, recorded).UTC()
		if _, err := record.Assessment(); err != nil {
			return nil, invalid(err)
		}
		records = append(records, record)
	}
	return records, mapDatabaseError(rows.Err())
}

type storedGoalHeader struct {
	requestRef, requestFingerprint, requestedBy string
	spec                                        goal.AppSpecSnapshot
	snapshot                                    goal.GoalSnapshot
	state                                       string
	revision, plan, generation                  int64
	paused, cancelRequested, controlSequence    int64
	submitted, confirmed, created               int64
	parentRef, parentHash                       sql.NullString
	started, closed                             sql.NullInt64
}

type goalHeader struct {
	requestRef, requestFingerprint, requestedBy string
	snapshot                                    goal.GoalSnapshot
}

func readGoalHeader(ctx context.Context, source queryer, goalValue string) (goalHeader, error) {
	controls, err := sqliteTableHasColumn(ctx, source, "goals", "control_sequence")
	if err != nil {
		return goalHeader{}, mapDatabaseError(err)
	}
	projection := "0, 0, 0"
	if controls {
		projection = "g.paused, g.cancel_requested, g.control_sequence"
	}
	var v storedGoalHeader
	err = source.QueryRowContext(ctx, `SELECT g.request_ref,g.request_fingerprint,g.requested_by_ref,
i.ref,i.actor_ref,i.project_ref,i.statement,i.submitted_at,i.hash,spec.ref,spec.generation,
spec.parent_ref,spec.parent_hash,spec.objective,spec.reason,spec.confirmed_by,spec.confirmed_at,spec.hash,
g.ref,g.actor_ref,g.project_ref,g.state,g.revision,`+projection+`,g.created_at,g.started_at,g.closed_at,g.plan_generation
FROM goals g JOIN app_specs spec ON spec.ref=g.app_spec_ref JOIN intents i ON i.ref=spec.intent_ref
WHERE g.ref=?`, goalValue).Scan(&v.requestRef, &v.requestFingerprint, &v.requestedBy,
		&v.spec.Intent.Ref, &v.spec.Intent.ActorRef, &v.spec.Intent.ProjectRef, &v.spec.Intent.Statement,
		&v.submitted, &v.spec.Intent.Hash, &v.spec.Ref, &v.generation, &v.parentRef, &v.parentHash,
		&v.spec.Objective, &v.spec.Reason, &v.spec.ConfirmedBy, &v.confirmed, &v.spec.Hash,
		&v.snapshot.Ref, &v.snapshot.ActorRef, &v.snapshot.ProjectRef, &v.state, &v.revision,
		&v.paused, &v.cancelRequested, &v.controlSequence, &v.created, &v.started, &v.closed, &v.plan)
	if err != nil {
		return goalHeader{}, mapDatabaseError(err)
	}
	return restoreGoalHeader(v)
}

func restoreGoalHeader(v storedGoalHeader) (goalHeader, error) {
	if v.revision <= 0 || v.plan < 0 || v.generation <= 0 ||
		(v.paused != 0 && v.paused != 1) || (v.cancelRequested != 0 && v.cancelRequested != 1) || v.controlSequence < 0 {
		return goalHeader{}, invalid(fmt.Errorf("sqlite.revision_invalid"))
	}
	v.spec.Intent.SubmittedAt = time.Unix(0, v.submitted).UTC()
	v.spec.Generation = goal.AppSpecGeneration(v.generation)
	if v.parentRef.Valid {
		v.spec.ParentRef = v.parentRef.String
	}
	if v.parentHash.Valid {
		v.spec.ParentHash = v.parentHash.String
	}
	v.spec.ConfirmedAt = time.Unix(0, v.confirmed).UTC()
	v.snapshot.AppSpec, v.snapshot.SchemaVersion = v.spec, goal.GoalSnapshotSchemaVersion
	v.snapshot.State, v.snapshot.Revision = goal.GoalState(v.state), goal.Revision(v.revision)
	v.snapshot.Paused, v.snapshot.CancelRequested = v.paused == 1, v.cancelRequested == 1
	v.snapshot.ControlSequence, v.snapshot.PlanGeneration = uint64(v.controlSequence), goal.PlanGeneration(v.plan)
	v.snapshot.CreatedAt, v.snapshot.StartedAt = time.Unix(0, v.created).UTC(), restoredTime(v.started)
	v.snapshot.ClosedAt = restoredTime(v.closed)
	return goalHeader{v.requestRef, v.requestFingerprint, v.requestedBy, v.snapshot}, nil
}

type goalData struct {
	requestedBy         identity.PrincipalRef
	goal                goal.Goal
	executions          []application.ExecutionRecord
	artifacts           []application.ArtifactRecord
	attestations        []application.AttestationRecord
	controls            []application.ControlRecord
	consumptionReceipts []application.ActionConsumptionReceipt
}

func readGoalData(ctx context.Context, source queryer, goalValue string, header goalHeader) (goalData, error) {
	var data goalData
	var err error
	header.snapshot.Phases, err = readPhases(ctx, source, goalValue)
	if err != nil {
		return data, err
	}
	var artifactRefs, attestationRefs map[string][]string
	data.artifacts, artifactRefs, err = readArtifacts(ctx, source, goalValue)
	if err == nil {
		data.attestations, attestationRefs, err = readAttestations(ctx, source, goalValue)
	}
	if err == nil {
		header.snapshot.WorkItems, err = readWorkItems(ctx, source, goalValue, artifactRefs, attestationRefs)
	}
	if err != nil {
		return data, err
	}
	if !validText(header.requestRef) || !validText(header.requestFingerprint) {
		return data, invalid(fmt.Errorf("sqlite.request_identity_invalid"))
	}
	data.requestedBy, err = identity.NewPrincipalRef(header.requestedBy)
	if err != nil {
		return data, invalid(err)
	}
	header.snapshot.ChildHandoffResolutions, err = readChildHandoffResolutions(ctx, source, goalValue)
	if err != nil {
		return data, err
	}
	data.goal, err = goal.RestoreGoal(header.snapshot)
	if err != nil {
		return data, invalid(err)
	}
	data.executions, err = readExecutions(ctx, source, goalValue)
	if err != nil {
		return data, err
	}
	if data.goal.PlanGeneration() > 0 && len(data.executions) == 0 {
		return data, invalid(fmt.Errorf("sqlite.executions_missing"))
	}
	data.consumptionReceipts, err = readConsumptionReceipts(ctx, source, goalValue)
	if err == nil {
		data.controls, err = readControls(ctx, source, goalValue)
	}
	return data, err
}

type goalGovernance struct {
	envelopes    []governance.BudgetEnvelope
	reservations []governance.BudgetReservation
	settlements  []governance.BudgetSettlement
	authorities  []application.WorkItemAuthority
	intents      []application.EffectIntent
	approvals    []application.EffectApproval
	attempts     []application.EffectAttempt
	outcomes     []application.EffectAttemptOutcome
	receipts     []application.EffectReceipt
}

func readGoalGovernance(ctx context.Context, source queryer, projectRef, goalValue string) (goalGovernance, error) {
	var data goalGovernance
	persisted, err := sqliteTableHasColumn(ctx, source, "outbox", "governance_version")
	if err != nil {
		return data, mapDatabaseError(err)
	}
	if !persisted {
		return data, nil
	}
	data.envelopes, err = readBudgetEnvelopesForGoal(ctx, source, projectRef, goalValue)
	if err == nil {
		data.reservations, err = readBudgetReservationsForGoal(ctx, source, goalValue)
	}
	if err == nil {
		data.settlements, err = readBudgetSettlementsForGoal(ctx, source, goalValue)
	}
	if err == nil {
		data.authorities, err = readWorkItemAuthoritiesForGoal(ctx, source, goalValue)
	}
	if err == nil {
		data.intents, err = readEffectIntentsForGoal(ctx, source, goalValue)
	}
	if err == nil {
		data.approvals, err = readEffectApprovalsForGoal(ctx, source, goalValue)
	}
	if err == nil {
		data.attempts, err = readEffectAttemptsForGoal(ctx, source, goalValue)
	}
	if err == nil {
		data.outcomes, err = readEffectAttemptOutcomesForGoal(ctx, source, goalValue)
	}
	if err == nil {
		data.receipts, err = readEffectReceiptsForGoal(ctx, source, goalValue)
	}
	return data, err
}

func readWorkItems(
	ctx context.Context,
	source queryer,
	goalValue string,
	artifactRefs map[string][]string,
	attestationRefs map[string][]string,
) ([]goal.WorkItemSnapshot, error) {
	schema, err := readWorkItemSchema(ctx, source)
	if err != nil {
		return nil, err
	}
	handoffProjection := "0"
	if schema.handoff {
		handoffProjection = "handoff_required"
	}
	controlProjection := "'', NULL, 0, 0, 0"
	interruptedProjection := "NULL"
	if schema.controls {
		controlProjection = "interrupt_cause, rework_of, paused, cancel_requested, control_sequence"
		interruptedProjection = "interrupted_at"
	}
	governanceProjection := "0, '', 0, 0, '', 0, 0, 0, 'normal', 'medium'"
	if schema.governance {
		governanceProjection = `governance_version, budget_demand_ref, budget_tokens,
            budget_money_micros, budget_currency, budget_active_time_ns,
            budget_process_slots, budget_disk_bytes, security_criticality, reasoning_effort`
	}
	councilProjection := "''"
	if schema.council {
		councilProjection = "council_policy"
	}
	rows, err := source.QueryContext(ctx, `
SELECT ref, goal_ref, actor_ref, project_ref, objective, state, revision,
       phase_key, role_key, parent_ref, `+handoffProjection+`, output_contract, skip_reason,
	       `+controlProjection+`, created_at, started_at, `+interruptedProjection+`, finished_at, execution_ref,
       `+governanceProjection+`, `+councilProjection+`
FROM work_items
WHERE goal_ref = ?
ORDER BY position`, goalValue)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []goal.WorkItemSnapshot
	for rows.Next() {
		stored, err := scanWorkItem(rows)
		if err != nil {
			return nil, err
		}
		item, err := restoreWorkItem(stored)
		if err != nil {
			return nil, err
		}
		if err := readWorkItemRelations(ctx, source, goalValue, &item, artifactRefs, attestationRefs); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	return result, nil
}

type storedWorkItem struct {
	item                                                                goal.WorkItemSnapshot
	state, outputContract, skipReason, interruptCause, councilPolicy    string
	demandRef, currency, criticality, effort                            string
	revision, handoff, paused, cancel, control, governanceVersion       int64
	tokens, moneyMicros, activeTimeNS, processSlots, diskBytes, created int64
	started, interrupted, finished                                      sql.NullInt64
	parent, reworkOf, execution                                         sql.NullString
}

func scanWorkItem(rows *sql.Rows) (storedWorkItem, error) {
	var v storedWorkItem
	err := rows.Scan(&v.item.Ref, &v.item.GoalRef, &v.item.ActorRef, &v.item.ProjectRef,
		&v.item.Objective, &v.state, &v.revision, &v.item.PhaseKey, &v.item.RoleKey,
		&v.parent, &v.handoff, &v.outputContract, &v.skipReason, &v.interruptCause,
		&v.reworkOf, &v.paused, &v.cancel, &v.control, &v.created, &v.started,
		&v.interrupted, &v.finished, &v.execution, &v.governanceVersion, &v.demandRef,
		&v.tokens, &v.moneyMicros, &v.currency, &v.activeTimeNS, &v.processSlots,
		&v.diskBytes, &v.criticality, &v.effort, &v.councilPolicy)
	if err != nil {
		return storedWorkItem{}, mapDatabaseError(err)
	}
	return v, nil
}

func restoreWorkItem(v storedWorkItem) (goal.WorkItemSnapshot, error) {
	item := v.item
	if v.revision <= 0 || (v.paused != 0 && v.paused != 1) ||
		(v.cancel != 0 && v.cancel != 1) || v.control < 0 {
		return item, invalid(fmt.Errorf("sqlite.revision_invalid"))
	}
	if v.handoff != 0 && v.handoff != 1 {
		return item, invalid(fmt.Errorf("sqlite.work_item_handoff_required_invalid"))
	}
	if v.governanceVersion != 0 && v.governanceVersion != 1 {
		return item, invalid(fmt.Errorf("sqlite.work_item_governance_version_invalid"))
	}
	if v.governanceVersion == 0 {
		item.BudgetDemand = governance.BudgetDemand{Ref: "budget-demand:" + item.Ref}
		item.SecurityCriticality = governance.SecurityCriticalityNormal
		item.ReasoningEffort = governance.ReasoningEffortMedium
	} else {
		item.BudgetDemand = governance.BudgetDemand{Ref: v.demandRef, Resources: governance.ResourceVector{
			Tokens: v.tokens, MoneyMicros: v.moneyMicros, Currency: governance.Currency(v.currency),
			ActiveTimeNS: v.activeTimeNS, ProcessSlots: v.processSlots, DiskBytes: v.diskBytes}}
		item.SecurityCriticality = governance.SecurityCriticality(v.criticality)
		item.ReasoningEffort = governance.ReasoningEffort(v.effort)
		if governance.ValidateBudgetDemand(item.BudgetDemand) != nil ||
			governance.ValidateSecurityCriticality(item.SecurityCriticality) != nil ||
			governance.ValidateReasoningEffort(item.ReasoningEffort) != nil {
			return item, invalid(fmt.Errorf("sqlite.work_item_governance_invalid"))
		}
	}
	handoff := v.handoff == 1
	item.HandoffRequired, item.State, item.Revision = &handoff, goal.WorkItemState(v.state), goal.Revision(v.revision)
	item.OutputContract, item.SkipReason = goal.OutputContractKind(v.outputContract), goal.WorkItemSkipReason(v.skipReason)
	item.CouncilPolicy = council.Policy(v.councilPolicy)
	item.InterruptCause, item.Paused, item.CancelRequested = goal.WorkItemInterruptCause(v.interruptCause), v.paused == 1, v.cancel == 1
	item.ControlSequence, item.CreatedAt = uint64(v.control), time.Unix(0, v.created).UTC()
	item.StartedAt, item.InterruptedAt, item.FinishedAt = restoredTime(v.started), restoredTime(v.interrupted), restoredTime(v.finished)
	if v.parent.Valid {
		item.ParentRef = v.parent.String
	}
	if v.reworkOf.Valid {
		item.ReworkOf = v.reworkOf.String
	}
	if v.execution.Valid {
		item.ExecutionRef = v.execution.String
	}
	return item, nil
}

func readWorkItemRelations(
	ctx context.Context, source queryer, goalValue string, item *goal.WorkItemSnapshot,
	artifactRefs, attestationRefs map[string][]string,
) error {
	// Workspace-backed executions persist their output before commit/integration,
	// while the WorkItem deliberately remains running. Those staged records are
	// GoalRecord facts, not WorkItem outputs, until integration succeeds.
	if item.State == goal.WorkItemStateSucceeded {
		item.ArtifactRefs = append([]string(nil), artifactRefs[item.Ref]...)
		item.AttestationRefs = append([]string(nil), attestationRefs[item.Ref]...)
	}
	var err error
	item.DependencyRefs, err = readContractRefs(ctx, source,
		`SELECT dependency_ref FROM work_item_dependencies WHERE work_item_ref = ? ORDER BY position`, item.Ref)
	if err == nil {
		item.WriteSet, err = readContractRefs(ctx, source,
			`SELECT scope FROM work_item_write_scopes WHERE work_item_ref = ? ORDER BY position`, item.Ref)
	}
	if err == nil {
		item.RequiredTests, err = readRequiredTestSpecs(ctx, source, goalValue, item.Ref)
	}
	requirements := []struct {
		kind   string
		target *[]string
	}{{"skill", &item.SkillRefs}, {"tool", &item.ToolRefs}, {"capability", &item.CapabilityRefs}}
	for _, requirement := range requirements {
		if err != nil {
			return err
		}
		*requirement.target, err = readContractRefs(ctx, source, `SELECT value FROM work_item_requirement_refs
WHERE goal_ref = ? AND work_item_ref = ? AND kind = ? ORDER BY position`, goalValue, item.Ref, requirement.kind)
	}
	return err
}

func readExecutions(ctx context.Context, source queryer, goalValue string) ([]application.ExecutionRecord, error) {
	schema, err := readExecutionSchema(ctx, source)
	if err != nil {
		return nil, err
	}
	markerProjection := "0"
	if schema.mailbox {
		markerProjection = "recipient_mailbox_retired"
	}
	governanceProjection := "0, NULL, NULL, NULL"
	if schema.governance {
		governanceProjection = "governance_version, budget_reservation_ref, effect_intent_ref, launch_receipt_ref"
	}
	workspaceProjection := "'', ''"
	if schema.workspace {
		workspaceProjection = "repository_ref, execution_workspace_ref"
	}
	reviewProjection := "'work', '', ''"
	if schema.reviews {
		reviewProjection = "purpose, review_subject_digest, ''"
	}
	if schema.council {
		reviewProjection = "purpose, review_subject_digest, council_subject_digest"
	}
	sessionProjection := "''"
	if schema.session {
		sessionProjection = "execution_session_ref"
	}
	preservacionProjection := "0"
	if schema.preservacion {
		preservacionProjection = "environment_preservation_required"
	}
	rows, err := source.QueryContext(ctx, `
SELECT ref, goal_ref, work_item_ref, state, artifact_media_type, idempotency_key,
       attempt_no, max_execution_attempts, replaces_execution_ref,
       plan_generation, app_spec_generation, spec_hash,
       max_output_bytes, provider_ref, model_ref, agent_ref,
	       external_ref, created_at,
       deadline_at, started_at, provider_accepted_at, last_observed_at,
       provider_observed_at, finished_at, failure_code, `+markerProjection+`,
	       `+governanceProjection+`, `+workspaceProjection+`, `+reviewProjection+`, `+sessionProjection+`, `+preservacionProjection+`
FROM executions
WHERE goal_ref = ?
ORDER BY (
    SELECT item.position FROM work_items item
    WHERE item.goal_ref = executions.goal_ref AND item.ref = executions.work_item_ref
), attempt_no, ref`, goalValue)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var records []application.ExecutionRecord
	for rows.Next() {
		stored, err := scanExecution(rows)
		if err != nil {
			return nil, err
		}
		record, err := restoreExecution(stored)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	return records, nil
}

type storedExecution struct {
	record                                                                        application.ExecutionRecord
	ref, goalRef, workItemRef, state                                              string
	created                                                                       int64
	deadline, started, accepted, observed, providerObserved, finished             sql.NullInt64
	replaces, reservation, intent, receipt, repository, workspace                 sql.NullString
	purpose, reviewSubject, councilSubject, session                               string
	attempt, maxAttempts, plan, appSpec, mailbox, governanceVersion, preservacion int64
}

func scanExecution(rows *sql.Rows) (storedExecution, error) {
	var v storedExecution
	err := rows.Scan(&v.ref, &v.goalRef, &v.workItemRef, &v.state, &v.record.ArtifactMediaType,
		&v.record.IdempotencyKey, &v.attempt, &v.maxAttempts, &v.replaces, &v.plan, &v.appSpec,
		&v.record.SpecHash, &v.record.MaxOutputBytes, &v.record.ProviderRef, &v.record.ModelRef,
		&v.record.AgentRef, &v.record.ExternalRef, &v.created, &v.deadline, &v.started,
		&v.accepted, &v.observed, &v.providerObserved, &v.finished, &v.record.FailureCode,
		&v.mailbox, &v.governanceVersion, &v.reservation, &v.intent, &v.receipt, &v.repository, &v.workspace,
		&v.purpose, &v.reviewSubject, &v.councilSubject, &v.session, &v.preservacion)
	if err != nil {
		return storedExecution{}, mapDatabaseError(err)
	}
	return v, nil
}

func restoreExecution(v storedExecution) (application.ExecutionRecord, error) {
	record := v.record
	var err error
	if record.Ref, err = goal.NewExecutionRef(v.ref); err != nil {
		return record, invalid(err)
	}
	if record.GoalRef, err = goal.NewGoalRef(v.goalRef); err != nil {
		return record, invalid(err)
	}
	if record.WorkItemRef, err = goal.NewWorkItemRef(v.workItemRef); err != nil {
		return record, invalid(err)
	}
	if v.replaces.Valid {
		if record.ReplacesExecutionRef, err = goal.NewExecutionRef(v.replaces.String); err != nil {
			return record, invalid(err)
		}
	}
	if v.attempt <= 0 || v.maxAttempts <= 0 || v.plan <= 0 || v.appSpec <= 0 ||
		(v.mailbox != 0 && v.mailbox != 1) || (v.governanceVersion != 0 && v.governanceVersion != 1) ||
		(v.preservacion != 0 && v.preservacion != 1) {
		return record, invalid(fmt.Errorf("sqlite.execution_generation_invalid"))
	}
	record.State, record.AttemptNo, record.MaxExecutionAttempts = application.ExecutionState(v.state), uint64(v.attempt), uint64(v.maxAttempts)
	record.Purpose, record.ReviewSubjectDigest = application.ExecutionPurpose(v.purpose), v.reviewSubject
	record.CouncilSubjectDigest = application.CouncilSubjectDigest(v.councilSubject)
	record.PlanGeneration, record.AppSpecGeneration = goal.PlanGeneration(v.plan), goal.AppSpecGeneration(v.appSpec)
	record.CreatedAt, record.DeadlineAt = time.Unix(0, v.created).UTC(), restoredTime(v.deadline)
	record.StartedAt, record.ProviderAcceptedAt = restoredTime(v.started), restoredTime(v.accepted)
	record.LastObservedAt, record.ProviderObservedAt = restoredTime(v.observed), restoredTime(v.providerObserved)
	record.FinishedAt, record.RecipientMailboxRetired = restoredTime(v.finished), v.mailbox == 1
	record.RequierePreservacionEntorno = v.preservacion == 1
	if v.reservation.Valid {
		record.BudgetReservationRef = v.reservation.String
	}
	if v.intent.Valid {
		record.EffectIntentRef = v.intent.String
	}
	if v.receipt.Valid {
		record.LaunchReceiptRef = v.receipt.String
	}
	if v.repository.Valid && v.repository.String != "" {
		if record.RepositoryRef, err = identity.NewRepositoryRef(v.repository.String); err != nil {
			return record, invalid(err)
		}
	}
	if v.workspace.Valid && v.workspace.String != "" {
		if record.ExecutionWorkspaceRef, err = ports.NewExecutionWorkspaceRef(v.workspace.String); err != nil {
			return record, invalid(err)
		}
	}
	if v.session != "" {
		if record.ExecutionSessionRef, err = ports.NewExecutionSessionRef(v.session); err != nil {
			return record, invalid(err)
		}
	}
	return record, nil
}

func readConsumptionReceipts(
	ctx context.Context,
	source queryer,
	goalValue string,
) ([]application.ActionConsumptionReceipt, error) {
	mailboxColumn, changeColumn, effectColumn, err := consumptionReceiptColumns(ctx, source)
	if err != nil {
		return nil, err
	}
	query := `
SELECT action_ref, kind, goal_ref, work_item_ref, execution_ref,
       ` + mailboxColumn + `, ` + changeColumn + `, plan_generation, work_item_generation, fence, delivery_attempt,
       claim_token, worker_ref, outcome, error_code, ` + effectColumn + `, consumed_at
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
		var changeRefValue sql.NullString
		var effectReceipt sql.NullString
		var planGeneration, itemGeneration, fence, deliveryAttempt, consumedAt int64
		if err := rows.Scan(
			&receipt.ActionRef, &kind, &goalRefValue, &workItemRefValue, &executionRefValue,
			&mailboxMessageValue, &changeRefValue,
			&planGeneration, &itemGeneration, &fence, &deliveryAttempt,
			&receipt.ClaimToken, &receipt.WorkerRef, &outcome, &receipt.ErrorCode,
			&effectReceipt, &consumedAt,
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
		if changeRefValue.Valid && changeRefValue.String != "" {
			if receipt.ChangeRef, refErr = ports.NewChangeSetRef(changeRefValue.String); refErr != nil {
				return nil, invalid(refErr)
			}
		}
		receipt.PlanGeneration = goal.PlanGeneration(planGeneration)
		receipt.WorkItemGeneration = goal.Revision(itemGeneration)
		receipt.Fence = uint64(fence)
		receipt.DeliveryAttempt = uint64(deliveryAttempt)
		receipt.Outcome = application.ActionConsumptionOutcome(outcome)
		if effectReceipt.Valid {
			receipt.EffectReceiptRef = effectReceipt.String
		}
		receipt.ConsumedAt = time.Unix(0, consumedAt).UTC()
		result = append(result, receipt)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	return result, nil
}

func consumptionReceiptColumns(ctx context.Context, source queryer) (string, string, string, error) {
	columns := []string{"NULL", "NULL", "NULL"}
	for position, name := range []string{"mailbox_message_ref", "change_ref", "effect_receipt_ref"} {
		found, err := sqliteTableHasColumn(ctx, source, "action_consumption_receipts", name)
		if err != nil {
			return "", "", "", mapDatabaseError(err)
		}
		if found {
			columns[position] = name
		}
	}
	return columns[0], columns[1], columns[2], nil
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

func readArtifacts(
	ctx context.Context,
	source queryer,
	goalValue string,
) ([]application.ArtifactRecord, map[string][]string, error) {
	persisted, err := sqliteTableHasColumn(ctx, source, "artifact_occurrences", "occurrence_ref")
	if err != nil {
		return nil, nil, mapDatabaseError(err)
	}
	query := `SELECT occurrence.occurrence_ref,occurrence.kind,artifact.ref,occurrence.goal_ref,
       occurrence.work_item_ref,occurrence.execution_ref,occurrence.execution_attempt,
       occurrence.plan_generation,occurrence.work_item_generation,occurrence.app_spec_generation,
       occurrence.spec_hash,artifact.digest,artifact.media_type,artifact.size,occurrence.created_at
FROM artifact_occurrences occurrence JOIN artifacts artifact
 ON artifact.goal_ref=occurrence.goal_ref AND artifact.ref=occurrence.artifact_ref
WHERE occurrence.goal_ref=? ORDER BY occurrence.created_at,occurrence.occurrence_ref`
	if !persisted {
		query = `SELECT 'artifact-occurrence:migrated-v16:'||ref,'agent_output',ref,goal_ref,work_item_ref,'',0,0,0,0,'',digest,media_type,size,created_at FROM artifacts WHERE goal_ref=? ORDER BY created_at,ref`
	}
	rows, err := source.QueryContext(ctx, query, goalValue)
	if err != nil {
		return nil, nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []application.ArtifactRecord
	refs := make(map[string][]string)
	for rows.Next() {
		var record application.ArtifactRecord
		var refValue, goalRefValue, workItemRefValue, executionRefValue, kind string
		var createdAt, executionAttempt, planGeneration, itemGeneration, appGeneration int64
		if err := rows.Scan(
			&record.OccurrenceRef, &kind, &refValue, &goalRefValue, &workItemRefValue, &executionRefValue,
			&executionAttempt, &planGeneration, &itemGeneration, &appGeneration, &record.SpecHash,
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
		if persisted {
			if record.ExecutionRef, refErr = goal.NewExecutionRef(executionRefValue); refErr != nil {
				return nil, nil, invalid(refErr)
			}
		}
		if persisted && (executionAttempt <= 0 || planGeneration <= 0 || itemGeneration <= 0 || appGeneration <= 0) {
			return nil, nil, invalid(errors.New("sqlite.artifact_occurrence_generation_invalid"))
		}
		record.Kind = application.ArtifactKind(kind)
		record.ExecutionAttempt = uint64(executionAttempt)
		record.PlanGeneration = goal.PlanGeneration(planGeneration)
		record.WorkItemGeneration = goal.Revision(itemGeneration)
		record.AppSpecGeneration = goal.AppSpecGeneration(appGeneration)
		record.CreatedAt = time.Unix(0, createdAt).UTC()
		if record.Kind == application.ArtifactKindAgentOutput &&
			!slices.Contains(refs[workItemRefValue], refValue) {
			refs[workItemRefValue] = append(refs[workItemRefValue], refValue)
		}
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
	typed, err := sqliteTableHasColumn(ctx, source, "attestations", "kind")
	if err != nil {
		return nil, nil, mapDatabaseError(err)
	}
	query := `SELECT ref,kind,verdict,goal_ref,work_item_ref,execution_ref,execution_attempt,
       plan_generation,work_item_generation,app_spec_generation,spec_hash,artifact_ref,
       subject_digest,workspace_binding_digest,change_set_ref,change_set_digest,
       manifest_artifact_ref,report_artifact_ref,attestor_ref,receipt_ref,policy_ref,
       required_tests_digest,policy_digest,effect_intent_ref,effect_attempt_ref,effect_fence,
       effect_receipt_ref,started_at,finished_at,policy,accepted_at
FROM attestations WHERE goal_ref=? ORDER BY accepted_at,ref`
	if !typed {
		query = `SELECT ref,'artifact_provenance','observed',goal_ref,work_item_ref,execution_ref,0,0,0,0,'',artifact_ref,'','',NULL,'',NULL,NULL,'','',policy,'','',NULL,NULL,0,NULL,accepted_at,accepted_at,policy,accepted_at FROM attestations WHERE goal_ref=? ORDER BY accepted_at,ref`
	}
	rows, err := source.QueryContext(ctx, query, goalValue)
	if err != nil {
		return nil, nil, mapDatabaseError(err)
	}
	var result []application.AttestationRecord
	refs := make(map[string][]string)
	for rows.Next() {
		record, workItemRefValue, scanErr := scanAttestation(rows)
		if scanErr != nil {
			_ = rows.Close()
			return nil, nil, scanErr
		}
		refs[workItemRefValue] = append(refs[workItemRefValue], record.Ref.String())
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, nil, mapDatabaseError(err)
	}
	if err := rows.Close(); err != nil {
		return nil, nil, mapDatabaseError(err)
	}
	for index := range result {
		result[index].Tests, err = readAttestationOutcomes(ctx, source, result[index].Ref.String())
		if err != nil {
			return nil, nil, err
		}
	}
	return result, refs, nil
}

func scanAttestation(row interface{ Scan(...any) error }) (application.AttestationRecord, string, error) {
	var record application.AttestationRecord
	var refValue, goalValue, itemValue, executionValue, artifactValue, kind, verdict string
	var changeRef, manifestRef, reportRef, effectIntent, effectAttempt, effectReceipt sql.NullString
	var acceptedAt, startedAt, finishedAt, attempt, plan, itemGeneration, appGeneration, effectFence int64
	if err := row.Scan(&refValue, &kind, &verdict, &goalValue, &itemValue, &executionValue, &attempt, &plan, &itemGeneration, &appGeneration, &record.SpecHash, &artifactValue, &record.SubjectDigest, &record.WorkspaceBindingDigest, &changeRef, &record.ChangeSetDigest, &manifestRef, &reportRef, &record.AttestorRef, &record.ReceiptRef, &record.PolicyRef, &record.RequiredTestsDigest, &record.PolicyDigest, &effectIntent, &effectAttempt, &effectFence, &effectReceipt, &startedAt, &finishedAt, &record.Policy, &acceptedAt); err != nil {
		return record, "", mapDatabaseError(err)
	}
	var err error
	if record.Ref, err = goal.NewAttestationRef(refValue); err == nil {
		record.GoalRef, err = goal.NewGoalRef(goalValue)
	}
	if err == nil {
		record.WorkItemRef, err = goal.NewWorkItemRef(itemValue)
	}
	if err == nil {
		record.ExecutionRef, err = goal.NewExecutionRef(executionValue)
	}
	if err == nil {
		record.ArtifactRef, err = goal.NewArtifactRef(artifactValue)
	}
	if err == nil && changeRef.Valid {
		record.ChangeSetRef, err = ports.NewChangeSetRef(changeRef.String)
	}
	if err == nil && manifestRef.Valid {
		record.ManifestArtifactRef, err = goal.NewArtifactRef(manifestRef.String)
	}
	if err == nil && reportRef.Valid {
		record.ReportArtifactRef, err = goal.NewArtifactRef(reportRef.String)
	}
	if err != nil {
		return record, "", invalid(err)
	}
	record.Kind, record.Verdict = application.AttestationKind(kind), application.AttestationVerdict(verdict)
	record.ExecutionAttempt, record.PlanGeneration = uint64(attempt), goal.PlanGeneration(plan)
	record.WorkItemGeneration, record.AppSpecGeneration = goal.Revision(itemGeneration), goal.AppSpecGeneration(appGeneration)
	record.EffectIntentRef, record.EffectAttemptRef, record.EffectReceiptRef = effectIntent.String, effectAttempt.String, effectReceipt.String
	record.EffectFence = uint64(effectFence)
	record.StartedAt, record.FinishedAt, record.AcceptedAt = time.Unix(0, startedAt).UTC(), time.Unix(0, finishedAt).UTC(), time.Unix(0, acceptedAt).UTC()
	return record, itemValue, nil
}
