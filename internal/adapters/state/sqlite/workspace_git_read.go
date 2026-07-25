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
	"orquesta/internal/ports"
)

func (repository *Repository) ProjectRepository(ctx context.Context, projectRef goal.ProjectRef) (identity.RepositoryRef, error) {
	if projectRef.String() == "" {
		return identity.RepositoryRef{}, invalid(fmt.Errorf("sqlite.project_ref_invalid"))
	}
	tx, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return identity.RepositoryRef{}, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT ref FROM repositories WHERE project_ref=? ORDER BY ref LIMIT 2`, projectRef.String())
	if err != nil {
		return identity.RepositoryRef{}, mapDatabaseError(err)
	}
	defer rows.Close()
	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return identity.RepositoryRef{}, mapDatabaseError(err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return identity.RepositoryRef{}, mapDatabaseError(err)
	}
	if len(values) == 0 {
		return identity.RepositoryRef{}, stateError(application.StateNotFound, fmt.Errorf("sqlite.project_repository_not_found"))
	}
	if len(values) != 1 {
		return identity.RepositoryRef{}, conflict(fmt.Errorf("sqlite.project_repository_ambiguous"))
	}
	ref, err := identity.NewRepositoryRef(values[0])
	if err != nil {
		return identity.RepositoryRef{}, invalid(err)
	}
	if err := commit(tx); err != nil {
		return identity.RepositoryRef{}, err
	}
	return ref, nil
}

type workspaceGitFacts struct {
	bindings     []application.WorkspaceBinding
	changes      []application.ChangeSet
	observations []application.MergeObservation
	receipts     []application.IntegrationReceipt
}

func readWorkspaceGitFacts(ctx context.Context, source queryer, goalValue string) (workspaceGitFacts, error) {
	var facts workspaceGitFacts
	present, err := sqliteTableHasColumn(ctx, source, "executions", "execution_workspace_ref")
	if err != nil {
		return facts, mapDatabaseError(err)
	}
	if !present {
		return facts, nil
	}
	if facts.bindings, err = readWorkspaceBindings(ctx, source, goalValue); err != nil {
		return facts, err
	}
	if facts.changes, err = readChangeSets(ctx, source, goalValue, facts.bindings); err != nil {
		return facts, err
	}
	if facts.observations, err = readMergeObservations(ctx, source, facts.changes); err != nil {
		return facts, err
	}
	facts.receipts, err = readIntegrationReceipts(ctx, source, facts.changes)
	return facts, err
}

func readWorkspaceBindings(ctx context.Context, source queryer, goalValue string) ([]application.WorkspaceBinding, error) {
	rows, err := source.QueryContext(ctx, `SELECT ref,principal_ref,actor_ref,project_ref,repository_ref,goal_ref,work_item_ref,execution_ref,execution_attempt,plan_generation,app_spec_generation,app_spec_hash,write_set_digest,target_ref,base_oid,object_format,adapter_ref,intent_ref,attempt_ref,action_fence,effect_receipt_ref,prepared_at FROM workspace_bindings WHERE goal_ref=? ORDER BY prepared_at,ref`, goalValue)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	return scanWorkspaceBindings(ctx, source, rows)
}

// WorkspaceBinding resolves one exact durable binding for composition-owned
// adapter recovery. It never enumerates goals or guesses a repository from a
// physical worktree.
type WorkspaceBindingRecovery struct {
	Binding               application.WorkspaceBinding
	PrepareIdempotencyKey string
	PreparedReceiptRef    string
}

func (repository *Repository) WorkspaceBinding(
	ctx context.Context,
	ref ports.ExecutionWorkspaceRef,
) (WorkspaceBindingRecovery, bool, error) {
	if ref.String() == "" {
		return WorkspaceBindingRecovery{}, false, invalid(fmt.Errorf("sqlite.workspace_binding_ref_invalid"))
	}
	database, err := repository.database()
	if err != nil {
		return WorkspaceBindingRecovery{}, false, err
	}
	row := database.QueryRowContext(ctx, `SELECT ref,principal_ref,actor_ref,project_ref,repository_ref,goal_ref,work_item_ref,execution_ref,execution_attempt,plan_generation,app_spec_generation,app_spec_hash,write_set_digest,target_ref,base_oid,object_format,adapter_ref,intent_ref,attempt_ref,action_fence,effect_receipt_ref,prepared_at FROM workspace_bindings WHERE ref=?`, ref.String())
	binding, refValue, err := scanWorkspaceBinding(row)
	if errors.Is(err, sql.ErrNoRows) {
		return WorkspaceBindingRecovery{}, false, nil
	}
	if err != nil {
		return WorkspaceBindingRecovery{}, false, err
	}
	binding.WriteSet, err = readWorkspaceWriteSet(ctx, database, refValue)
	if err != nil {
		return WorkspaceBindingRecovery{}, false, err
	}
	if binding.Ref != ref || application.ValidateWorkspaceBinding(binding) != nil {
		return WorkspaceBindingRecovery{}, false, invalid(fmt.Errorf("sqlite.workspace_binding_invalid"))
	}
	var idempotencyKey, preparedReceiptRef string
	if err := database.QueryRowContext(
		ctx, `SELECT intent.idempotency_key,receipt.external_ref
FROM effect_intents intent
JOIN effect_attempts attempt
  ON attempt.ref=?
 AND attempt.intent_ref=intent.ref
 AND attempt.intent_digest=intent.digest
 AND attempt.action_ref=intent.action_ref
 AND attempt.action_fence=?
JOIN effect_receipts receipt
  ON receipt.intent_ref=intent.ref
 AND receipt.intent_digest=intent.digest
 AND receipt.attempt_ref=attempt.ref
 AND receipt.action_ref=intent.action_ref
 AND receipt.action_fence=attempt.action_fence
 AND receipt.idempotency_key=intent.idempotency_key
 AND receipt.ref=?
 AND receipt.status='prepared'
WHERE intent.ref=? AND intent.kind='prepare_workspace'`,
		binding.EffectAttemptRef, binding.EffectFence, binding.ReceiptRef, binding.EffectIntentRef,
	).Scan(&idempotencyKey, &preparedReceiptRef); err != nil {
		return WorkspaceBindingRecovery{}, false, mapDatabaseError(err)
	}
	if idempotencyKey == "" || preparedReceiptRef == "" {
		return WorkspaceBindingRecovery{}, false, invalid(fmt.Errorf("sqlite.workspace_binding_idempotency_invalid"))
	}
	return WorkspaceBindingRecovery{
		Binding: binding, PrepareIdempotencyKey: idempotencyKey, PreparedReceiptRef: preparedReceiptRef,
	}, true, nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanWorkspaceBinding(row rowScanner) (application.WorkspaceBinding, string, error) {
	var b application.WorkspaceBinding
	var ref, principal, actor, project, repository, goalRef, item, execution, format string
	var attempt, plan, spec, fence, prepared int64
	if err := row.Scan(&ref, &principal, &actor, &project, &repository, &goalRef, &item, &execution, &attempt, &plan, &spec, &b.SpecHash, &b.WriteSetDigest, &b.TargetRef, &b.BaseOID, &format, &b.AdapterRef, &b.EffectIntentRef, &b.EffectAttemptRef, &fence, &b.ReceiptRef, &prepared); err != nil {
		return application.WorkspaceBinding{}, "", mapDatabaseError(err)
	}
	var err error
	if b.Ref, err = ports.NewExecutionWorkspaceRef(ref); err != nil {
		return application.WorkspaceBinding{}, "", invalid(err)
	}
	if b.PrincipalRef, err = identity.NewPrincipalRef(principal); err != nil {
		return application.WorkspaceBinding{}, "", invalid(err)
	}
	if b.ActorRef, err = goal.NewActorRef(actor); err != nil {
		return application.WorkspaceBinding{}, "", invalid(err)
	}
	if b.ProjectRef, err = goal.NewProjectRef(project); err != nil {
		return application.WorkspaceBinding{}, "", invalid(err)
	}
	if b.RepositoryRef, err = identity.NewRepositoryRef(repository); err != nil {
		return application.WorkspaceBinding{}, "", invalid(err)
	}
	if b.GoalRef, err = goal.NewGoalRef(goalRef); err != nil {
		return application.WorkspaceBinding{}, "", invalid(err)
	}
	if b.WorkItemRef, err = goal.NewWorkItemRef(item); err != nil {
		return application.WorkspaceBinding{}, "", invalid(err)
	}
	if b.ExecutionRef, err = goal.NewExecutionRef(execution); err != nil {
		return application.WorkspaceBinding{}, "", invalid(err)
	}
	b.ExecutionAttempt = uint64(attempt)
	b.PlanGeneration = goal.PlanGeneration(plan)
	b.AppSpecGeneration = goal.AppSpecGeneration(spec)
	b.ObjectFormat = ports.GitObjectFormat(format)
	b.EffectFence = uint64(fence)
	b.PreparedAt = time.Unix(0, prepared).UTC()
	return b, ref, nil
}

func scanWorkspaceBindings(
	ctx context.Context,
	source queryer,
	rows *sql.Rows,
) ([]application.WorkspaceBinding, error) {
	defer rows.Close()
	var result []application.WorkspaceBinding
	for rows.Next() {
		b, ref, err := scanWorkspaceBinding(rows)
		if err != nil {
			return nil, err
		}
		b.WriteSet, err = readWorkspaceWriteSet(ctx, source, ref)
		if err != nil {
			return nil, err
		}
		if err = application.ValidateWorkspaceBinding(b); err != nil {
			return nil, invalid(err)
		}
		result = append(result, b)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	return result, nil
}

func readWorkspaceWriteSet(ctx context.Context, source queryer, ref string) ([]string, error) {
	rows, err := source.QueryContext(ctx, `SELECT write_scope FROM workspace_binding_write_scopes WHERE workspace_binding_ref=? ORDER BY ordinal`, ref)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var values []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, mapDatabaseError(err)
		}
		values = append(values, v)
	}
	return values, rows.Err()
}

func readChangeSets(ctx context.Context, source queryer, goalValue string, bindings []application.WorkspaceBinding) ([]application.ChangeSet, error) {
	byRef := map[string]application.WorkspaceBinding{}
	for _, b := range bindings {
		byRef[b.Ref.String()] = b
	}
	rows, err := source.QueryContext(ctx, `SELECT ref,workspace_binding_ref,parent_change_ref,principal_ref,actor_ref,project_ref,repository_ref,goal_ref,work_item_ref,execution_ref,execution_attempt,plan_generation,app_spec_generation,app_spec_hash,write_set_digest,base_oid,parent_oid,head_oid,tree_oid,object_format,diff_digest,intent_ref,attempt_ref,action_fence,idempotency_key,adapter_ref,effect_receipt_ref,committed_at FROM change_sets WHERE goal_ref=? ORDER BY committed_at,ref`, goalValue)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []application.ChangeSet
	for rows.Next() {
		var c application.ChangeSet
		var ref, binding, principal, actor, project, repository, gref, item, execution, format string
		var parent sql.NullString
		var attempt, plan, spec, fence, committed int64
		if err := rows.Scan(&ref, &binding, &parent, &principal, &actor, &project, &repository, &gref, &item, &execution, &attempt, &plan, &spec, &c.SpecHash, &c.WriteSetDigest, &c.BaseOID, &c.ParentOID, &c.HeadOID, &c.TreeOID, &format, &c.DiffDigest, &c.EffectIntentRef, &c.EffectAttemptRef, &fence, &c.IdempotencyKey, &c.AdapterRef, &c.ReceiptRef, &committed); err != nil {
			return nil, mapDatabaseError(err)
		}
		var e error
		if c.Ref, e = ports.NewChangeSetRef(ref); e != nil {
			return nil, invalid(e)
		}
		if c.WorkspaceRef = byRef[binding].Ref; c.WorkspaceRef.String() == "" {
			return nil, invalid(fmt.Errorf("sqlite.change_set_binding_missing"))
		}
		if c.PrincipalRef, e = identity.NewPrincipalRef(principal); e != nil {
			return nil, invalid(e)
		}
		if c.ActorRef, e = goal.NewActorRef(actor); e != nil {
			return nil, invalid(e)
		}
		if c.ProjectRef, e = goal.NewProjectRef(project); e != nil {
			return nil, invalid(e)
		}
		if c.RepositoryRef, e = identity.NewRepositoryRef(repository); e != nil {
			return nil, invalid(e)
		}
		if c.GoalRef, e = goal.NewGoalRef(gref); e != nil {
			return nil, invalid(e)
		}
		if c.WorkItemRef, e = goal.NewWorkItemRef(item); e != nil {
			return nil, invalid(e)
		}
		if c.ExecutionRef, e = goal.NewExecutionRef(execution); e != nil {
			return nil, invalid(e)
		}
		if parent.Valid && parent.String != "" {
			if c.ParentChangeRef, e = ports.NewChangeSetRef(parent.String); e != nil {
				return nil, invalid(e)
			}
		}
		c.ExecutionAttempt = uint64(attempt)
		c.PlanGeneration = goal.PlanGeneration(plan)
		c.AppSpecGeneration = goal.AppSpecGeneration(spec)
		c.ObjectFormat = ports.GitObjectFormat(format)
		c.EffectFence = uint64(fence)
		c.CommittedAt = time.Unix(0, committed).UTC()
		c.WriteSet = byRef[binding].WriteSet
		c.ChangedPaths, e = readChangePaths(ctx, source, ref)
		if e != nil {
			return nil, e
		}
		if e = application.ValidateChangeSet(c); e != nil {
			return nil, invalid(e)
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func readChangePaths(ctx context.Context, source queryer, ref string) ([]string, error) {
	rows, err := source.QueryContext(ctx, `SELECT repository_path FROM change_set_paths WHERE change_set_ref=? ORDER BY ordinal`, ref)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var values []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, mapDatabaseError(err)
		}
		values = append(values, v)
	}
	return values, rows.Err()
}

func readMergeObservations(ctx context.Context, source queryer, changes []application.ChangeSet) ([]application.MergeObservation, error) {
	if len(changes) == 0 {
		return nil, nil
	}
	rows, err := source.QueryContext(ctx, `SELECT ref,change_set_ref,repository_ref,source_oid,target_ref,target_oid,object_format,outcome,candidate_tree_oid,conflict_digest,adapter_ref,observed_at FROM merge_observations WHERE change_set_ref IN (SELECT ref FROM change_sets WHERE goal_ref=?) ORDER BY observed_at,ref`, changes[0].GoalRef.String())
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var out []application.MergeObservation
	for rows.Next() {
		var o application.MergeObservation
		var change, repo, format, status string
		var tree, conflict sql.NullString
		var at int64
		if err := rows.Scan(&o.Ref, &change, &repo, &o.SourceOID, &o.TargetRef, &o.TargetOID, &format, &status, &tree, &conflict, &o.AdapterRef, &at); err != nil {
			return nil, mapDatabaseError(err)
		}
		var e error
		if o.ChangeRef, e = ports.NewChangeSetRef(change); e != nil {
			return nil, invalid(e)
		}
		if o.RepositoryRef, e = identity.NewRepositoryRef(repo); e != nil {
			return nil, invalid(e)
		}
		o.ObjectFormat = ports.GitObjectFormat(format)
		o.Status = ports.MergeStatus(status)
		o.CandidateTreeOID = tree.String
		o.ConflictDigest = conflict.String
		o.ObservedAt = time.Unix(0, at).UTC()
		if e = application.ValidateMergeObservation(o); e != nil {
			return nil, invalid(e)
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func readIntegrationReceipts(ctx context.Context, source queryer, changes []application.ChangeSet) ([]application.IntegrationReceipt, error) {
	if len(changes) == 0 {
		return nil, nil
	}
	rows, err := source.QueryContext(ctx, `SELECT ref,change_set_ref,repository_ref,source_oid,target_ref,target_before_oid,target_after_oid,tree_oid,marker_ref,conflict_digest,object_format,status,intent_ref,attempt_ref,action_fence,effect_receipt_ref,adapter_ref,confirmed_at FROM integration_receipts WHERE change_set_ref IN (SELECT ref FROM change_sets WHERE goal_ref=?) ORDER BY confirmed_at,ref`, changes[0].GoalRef.String())
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var out []application.IntegrationReceipt
	for rows.Next() {
		var r application.IntegrationReceipt
		var change, repo, format, status string
		var tree, marker, conflict sql.NullString
		var fence, at int64
		if err := rows.Scan(&r.Ref, &change, &repo, &r.SourceOID, &r.TargetRef, &r.TargetBeforeOID, &r.TargetAfterOID, &tree, &marker, &conflict, &format, &status, &r.EffectIntentRef, &r.EffectAttemptRef, &fence, &r.EffectReceiptRef, &r.AdapterRef, &at); err != nil {
			return nil, mapDatabaseError(err)
		}
		var e error
		if r.ChangeRef, e = ports.NewChangeSetRef(change); e != nil {
			return nil, invalid(e)
		}
		if r.RepositoryRef, e = identity.NewRepositoryRef(repo); e != nil {
			return nil, invalid(e)
		}
		r.TreeOID = tree.String
		r.MarkerRef = marker.String
		r.ConflictDigest = conflict.String
		r.ObjectFormat = ports.GitObjectFormat(format)
		r.Status = ports.IntegrationStatus(status)
		r.EffectFence = uint64(fence)
		r.RecordedAt = time.Unix(0, at).UTC()
		if e = application.ValidateIntegrationReceipt(r); e != nil {
			return nil, invalid(e)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
