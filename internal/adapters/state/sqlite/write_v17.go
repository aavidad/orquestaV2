package sqlite

import (
	"context"
	"database/sql"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

func insertWorkItemRelations(ctx context.Context, tx *sql.Tx, item goal.WorkItemSnapshot) error {
	for position, dependency := range item.DependencyRefs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO work_item_dependencies(goal_ref,work_item_ref,dependency_ref,position) VALUES (?,?,?,?)`, item.GoalRef, item.Ref, dependency, position); err != nil {
			return mapDatabaseError(err)
		}
	}
	for position, scope := range item.WriteSet {
		if _, err := tx.ExecContext(ctx, `INSERT INTO work_item_write_scopes(goal_ref,work_item_ref,scope,position) VALUES (?,?,?,?)`, item.GoalRef, item.Ref, scope, position); err != nil {
			return mapDatabaseError(err)
		}
	}
	for position, required := range item.RequiredTests {
		if _, err := tx.ExecContext(ctx, `INSERT INTO work_item_required_tests(goal_ref,work_item_ref,position,ref,tool_ref,working_directory) VALUES (?,?,?,?,?,?)`, item.GoalRef, item.Ref, position, required.Ref, required.ToolRef, required.WorkingDirectory); err != nil {
			return mapDatabaseError(err)
		}
		for argumentPosition, argument := range required.Arguments {
			if _, err := tx.ExecContext(ctx, `INSERT INTO work_item_required_test_arguments(goal_ref,work_item_ref,required_test_ref,position,value) VALUES (?,?,?,?,?)`, item.GoalRef, item.Ref, required.Ref, argumentPosition, argument); err != nil {
				return mapDatabaseError(err)
			}
		}
	}
	for _, refs := range []struct {
		kind string
		refs []string
	}{{"skill", item.SkillRefs}, {"tool", item.ToolRefs}, {"capability", item.CapabilityRefs}} {
		if err := insertOrderedContractRefs(ctx, tx, "work_item_requirement_refs", item.GoalRef, item.Ref, refs.kind, refs.refs); err != nil {
			return err
		}
	}
	return nil
}

func insertAttestation(ctx context.Context, tx *sql.Tx, a application.AttestationRecord) error {
	if err := validateAttestationRecord(a); err != nil {
		return invalid(err)
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO attestations(ref,kind,verdict,goal_ref,work_item_ref,execution_ref,execution_attempt,plan_generation,work_item_generation,app_spec_generation,spec_hash,artifact_ref,subject_digest,workspace_binding_digest,change_set_ref,change_set_digest,manifest_artifact_ref,report_artifact_ref,attestor_ref,receipt_ref,policy_ref,required_tests_digest,policy_digest,effect_intent_ref,effect_attempt_ref,effect_fence,effect_receipt_ref,started_at,finished_at,policy,accepted_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.Ref.String(), string(a.Kind), string(a.Verdict), a.GoalRef.String(), a.WorkItemRef.String(), a.ExecutionRef.String(), int64(a.ExecutionAttempt), int64(a.PlanGeneration), int64(a.WorkItemGeneration), int64(a.AppSpecGeneration), a.SpecHash, a.ArtifactRef.String(), a.SubjectDigest, a.WorkspaceBindingDigest, nullableString(a.ChangeSetRef.String()), a.ChangeSetDigest, nullableString(a.ManifestArtifactRef.String()), nullableString(a.ReportArtifactRef.String()), a.AttestorRef, a.ReceiptRef, a.PolicyRef, a.RequiredTestsDigest, a.PolicyDigest, nullableString(a.EffectIntentRef), nullableString(a.EffectAttemptRef), int64(a.EffectFence), nullableString(a.EffectReceiptRef), requiredTime(a.StartedAt), requiredTime(a.FinishedAt), a.Policy, requiredTime(a.AcceptedAt))
	if err != nil {
		return mapDatabaseError(err)
	}
	for position, outcome := range a.Tests {
		if _, err := tx.ExecContext(ctx, `INSERT INTO attestation_test_outcomes(attestation_ref,goal_ref,work_item_ref,position,required_test_ref,exit_code,output_digest) VALUES (?,?,?,?,?,?,?)`, a.Ref.String(), a.GoalRef.String(), a.WorkItemRef.String(), position, outcome.RequiredTestRef.String(), outcome.ExitCode, outcome.OutputDigest); err != nil {
			return mapDatabaseError(err)
		}
	}
	return nil
}
