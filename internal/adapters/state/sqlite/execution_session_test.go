package sqlite

import (
	"context"
	"errors"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func TestExecutionServicePrincipalExactScopeRevocationAndRestart(t *testing.T) {
	ctx, system := context.Background(), newSQLiteV15System(t, 2)
	created := system.submit(t, "request:v22-execution-authority")
	queued := created.Record.Executions[0]
	if _, err := system.repository.ExecutionSessionAuthority(ctx, queued.Ref, "opaque_authn"); !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("queued execution acquired authority err=%v", err)
	}
	if result, err := system.orchestrator.ProcessNext(ctx, "worker:v22-execution-authority"); err != nil || !result.Processed {
		t.Fatalf("ProcessNext result=%+v err=%v", result, err)
	}
	record, err := system.repository.GetGoal(ctx, created.Record.Goal.Ref())
	if err != nil || len(record.Executions) != 1 {
		t.Fatalf("GetGoal executions=%d err=%v", len(record.Executions), err)
	}
	execution := record.Executions[0]
	authority, err := system.repository.ExecutionSessionAuthority(ctx, execution.Ref, "opaque_authn")
	if err != nil {
		t.Fatal(err)
	}
	if authority.Request.ProjectRef != system.project || authority.Request.GoalRef != record.Goal.Ref() ||
		authority.Request.WorkItemRef != execution.WorkItemRef || authority.Request.ExecutionAttempt != execution.AttemptNo ||
		authority.Request.PlanGeneration != execution.PlanGeneration ||
		authority.Request.AppSpecGeneration != execution.AppSpecGeneration ||
		authority.Request.SpecHash != execution.SpecHash {
		t.Fatalf("authority tuple=%+v execution=%+v", authority, execution)
	}
	resolved, err := system.repository.ResolveExecution(ctx, authority.ServicePrincipal, system.project, execution.Ref)
	if err != nil || resolved != execution.Ref {
		t.Fatalf("ResolveExecution resolved=%s err=%v", resolved, err)
	}
	if found, active, ref, err := system.repository.ClassifyExecutionPrincipal(ctx, authority.ServicePrincipal, system.project); err != nil || !found || !active || ref != execution.Ref {
		t.Fatalf("active classification found=%v active=%v ref=%s err=%v", found, active, ref, err)
	}
	wrongProject, _ := goal.NewProjectRef("project:wrong")
	if found, active, ref, err := system.repository.ClassifyExecutionPrincipal(ctx, authority.ServicePrincipal, wrongProject); err != nil || !found || active || ref != execution.Ref {
		t.Fatalf("foreign classification found=%v active=%v ref=%s err=%v", found, active, ref, err)
	}
	if _, err := system.repository.ResolveExecution(ctx, authority.ServicePrincipal, wrongProject, execution.Ref); !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("wrong project err=%v", err)
	}
	human, _ := identity.NewPrincipal(authority.ServicePrincipal.Ref, authority.ServicePrincipal.ActorRef, identity.PrincipalKindHuman, authority.ServicePrincipal.Method)
	if _, err := system.repository.ResolveExecution(ctx, human, system.project, execution.Ref); !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("human substitution err=%v", err)
	}

	restartSQLiteV15System(t, system)
	resolved, err = system.repository.ResolveExecution(ctx, authority.ServicePrincipal, system.project, execution.Ref)
	if err != nil || resolved != execution.Ref {
		t.Fatalf("restart ResolveExecution resolved=%s err=%v", resolved, err)
	}
	owner := testPrincipal(t, "principal:v15-owner", "actor:v15-owner", identity.PrincipalKindHuman)
	grantTestMembership(t, system.repository, owner, authority.ServicePrincipal, system.project, identity.RoleContributor, "grant:v22-execution-membership", system.clock.Now())
	if _, err := system.repository.db.ExecContext(ctx, `UPDATE executions SET state='failed',failure_code='test.terminal',finished_at=created_at WHERE ref=?`, execution.Ref.String()); err != nil {
		t.Fatal(err)
	}
	successorRef, _ := goal.NewExecutionRef("execution:v22-successor")
	if _, err := system.repository.db.ExecContext(ctx, `
INSERT INTO executions(
 ref,goal_ref,work_item_ref,attempt_no,max_execution_attempts,replaces_execution_ref,plan_generation,app_spec_generation,spec_hash,
 repository_ref,execution_workspace_ref,state,purpose,review_subject_digest,council_subject_digest,artifact_media_type,
 idempotency_key,max_output_bytes,provider_ref,model_ref,agent_ref,external_ref,governance_version,budget_reservation_ref,
 effect_intent_ref,launch_receipt_ref,created_at,deadline_at,started_at,provider_accepted_at,last_observed_at,provider_observed_at,
 finished_at,failure_code,recipient_mailbox_retired)
SELECT ?,goal_ref,work_item_ref,attempt_no+1,max_execution_attempts,ref,plan_generation,app_spec_generation,spec_hash,
 repository_ref,execution_workspace_ref,'dispatching',purpose,review_subject_digest,council_subject_digest,artifact_media_type,
 ?,max_output_bytes,'','','','',governance_version,NULL,NULL,NULL,finished_at+1,NULL,NULL,NULL,NULL,NULL,NULL,'',0
FROM executions WHERE ref=?`,
		successorRef.String(), "execution:"+successorRef.String(), execution.Ref.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := system.repository.db.ExecContext(ctx, `UPDATE work_items SET execution_ref=? WHERE goal_ref=? AND ref=?`,
		successorRef.String(), execution.GoalRef.String(), execution.WorkItemRef.String()); err != nil {
		t.Fatal(err)
	}
	if found, active, ref, err := system.repository.ClassifyExecutionPrincipal(ctx, authority.ServicePrincipal, system.project); err != nil || !found || active || ref != execution.Ref {
		t.Fatalf("replaced classification found=%v active=%v ref=%s err=%v", found, active, ref, err)
	}
	staleRequest, _ := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: "authorization:v22-replaced", Principal: authority.ServicePrincipal,
		ProjectRef: system.project, Permission: identity.PermissionGoalsGet,
		ResourceRef: record.Goal.Ref().String(), RequestedAt: system.clock.Now(),
	})
	if _, err := system.repository.Authorize(ctx, staleRequest); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("replaced principal degraded to membership: %v", err)
	}
	if _, err := system.repository.ResolveExecution(ctx, authority.ServicePrincipal, system.project, execution.Ref); !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("replaced execution retained authority err=%v", err)
	}
	successor, err := system.repository.ExecutionSessionAuthority(ctx, successorRef, "opaque_authn")
	if err != nil || successor.Request.ReplacesExecutionRef != execution.Ref ||
		successor.Request.ExecutionAttempt != execution.AttemptNo+1 {
		t.Fatalf("successor authority=%+v err=%v", successor, err)
	}
	if resolved, err := system.repository.ResolveExecution(ctx, successor.ServicePrincipal, system.project, successorRef); err != nil || resolved != successorRef {
		t.Fatalf("successor ResolveExecution=%s err=%v", resolved, err)
	}
	if _, err := system.repository.db.ExecContext(ctx, `UPDATE executions SET state='failed',failure_code='test.terminal',finished_at=created_at WHERE ref=?`, successorRef.String()); err != nil {
		t.Fatal(err)
	}
	if found, active, ref, err := system.repository.ClassifyExecutionPrincipal(ctx, successor.ServicePrincipal, system.project); err != nil || !found || active || ref != successorRef {
		t.Fatalf("terminal classification found=%v active=%v ref=%s err=%v", found, active, ref, err)
	}
}
