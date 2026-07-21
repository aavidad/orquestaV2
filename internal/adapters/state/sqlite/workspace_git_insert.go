package sqlite

import (
	"context"
	"database/sql"

	"orquesta/internal/application"
)

func insertWorkspaceBinding(ctx context.Context, tx *sql.Tx, b application.WorkspaceBinding) error {
	_, e := tx.ExecContext(ctx, `INSERT INTO workspace_bindings(ref,execution_workspace_ref,principal_ref,actor_ref,project_ref,repository_ref,goal_ref,work_item_ref,execution_ref,execution_attempt,plan_generation,app_spec_generation,app_spec_hash,write_set_digest,target_ref,base_oid,object_format,adapter_ref,intent_ref,attempt_ref,action_fence,effect_receipt_ref,prepared_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, b.Ref.String(), b.Ref.String(), b.PrincipalRef.String(), b.ActorRef.String(), b.ProjectRef.String(), b.RepositoryRef.String(), b.GoalRef.String(), b.WorkItemRef.String(), b.ExecutionRef.String(), b.ExecutionAttempt, uint64(b.PlanGeneration), uint64(b.AppSpecGeneration), b.SpecHash, b.WriteSetDigest, b.TargetRef, b.BaseOID, string(b.ObjectFormat), b.AdapterRef, b.EffectIntentRef, b.EffectAttemptRef, b.EffectFence, b.ReceiptRef, requiredTime(b.PreparedAt))
	if e != nil {
		return mapDatabaseError(e)
	}
	for i, v := range b.WriteSet {
		if _, e = tx.ExecContext(ctx, `INSERT INTO workspace_binding_write_scopes(workspace_binding_ref,ordinal,write_scope)VALUES(?,?,?)`, b.Ref.String(), i, v); e != nil {
			return mapDatabaseError(e)
		}
	}
	return nil
}
func insertChangeSet(ctx context.Context, tx *sql.Tx, c application.ChangeSet) error {
	_, e := tx.ExecContext(ctx, `INSERT INTO change_sets(ref,workspace_binding_ref,parent_change_ref,principal_ref,actor_ref,project_ref,repository_ref,goal_ref,work_item_ref,execution_ref,execution_attempt,plan_generation,app_spec_generation,app_spec_hash,write_set_digest,base_oid,parent_oid,head_oid,tree_oid,object_format,diff_digest,intent_ref,attempt_ref,action_fence,idempotency_key,adapter_ref,effect_receipt_ref,committed_at)VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, c.Ref.String(), c.WorkspaceRef.String(), nullableString(c.ParentChangeRef.String()), c.PrincipalRef.String(), c.ActorRef.String(), c.ProjectRef.String(), c.RepositoryRef.String(), c.GoalRef.String(), c.WorkItemRef.String(), c.ExecutionRef.String(), c.ExecutionAttempt, uint64(c.PlanGeneration), uint64(c.AppSpecGeneration), c.SpecHash, c.WriteSetDigest, c.BaseOID, c.ParentOID, c.HeadOID, c.TreeOID, string(c.ObjectFormat), c.DiffDigest, c.EffectIntentRef, c.EffectAttemptRef, c.EffectFence, c.IdempotencyKey, c.AdapterRef, c.ReceiptRef, requiredTime(c.CommittedAt))
	if e != nil {
		return mapDatabaseError(e)
	}
	for i, v := range c.ChangedPaths {
		if _, e = tx.ExecContext(ctx, `INSERT INTO change_set_paths(change_set_ref,ordinal,repository_path)VALUES(?,?,?)`, c.Ref.String(), i, v); e != nil {
			return mapDatabaseError(e)
		}
	}
	return nil
}
func insertMergeObservation(ctx context.Context, tx *sql.Tx, o application.MergeObservation) error {
	result, e := tx.ExecContext(ctx, `INSERT INTO merge_observations(ref,change_set_ref,project_ref,repository_ref,target_ref,source_oid,target_oid,object_format,outcome,candidate_tree_oid,conflict_digest,adapter_ref,observed_at) SELECT ?,?,project_ref,?,?,?,?,?,?,?,?,?,? FROM change_sets WHERE ref=?`, o.Ref, o.ChangeRef.String(), o.RepositoryRef.String(), o.TargetRef, o.SourceOID, o.TargetOID, string(o.ObjectFormat), string(o.Status), nullableString(o.CandidateTreeOID), nullableString(o.ConflictDigest), o.AdapterRef, requiredTime(o.ObservedAt), o.ChangeRef.String())
	if e == nil {
		e = requireOneRow(result)
	}
	return mapDatabaseError(e)
}
func insertIntegrationReceipt(ctx context.Context, tx *sql.Tx, i application.IntegrationReceipt, observationRef string) error {
	result, e := tx.ExecContext(ctx, `INSERT INTO integration_receipts(ref,change_set_ref,merge_observation_ref,project_ref,repository_ref,target_ref,source_oid,target_before_oid,target_after_oid,tree_oid,marker_ref,conflict_digest,object_format,status,intent_ref,attempt_ref,action_fence,effect_receipt_ref,adapter_ref,confirmed_at) SELECT ?,?,?,project_ref,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,? FROM change_sets WHERE ref=?`, i.Ref, i.ChangeRef.String(), observationRef, i.RepositoryRef.String(), i.TargetRef, i.SourceOID, i.TargetBeforeOID, i.TargetAfterOID, nullableString(i.TreeOID), nullableString(i.MarkerRef), nullableString(i.ConflictDigest), string(i.ObjectFormat), string(i.Status), i.EffectIntentRef, i.EffectAttemptRef, i.EffectFence, i.EffectReceiptRef, i.AdapterRef, requiredTime(i.RecordedAt), i.ChangeRef.String())
	if e == nil {
		e = requireOneRow(result)
	}
	return mapDatabaseError(e)
}
