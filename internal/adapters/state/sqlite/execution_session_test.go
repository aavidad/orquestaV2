package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

type sqliteExecutionSessionBroker struct {
	at             time.Time
	ensures        int
	revokes        int
	revokeFailures int
	revoked        bool
	replays        int
}

func (broker *sqliteExecutionSessionBroker) Ensure(
	_ context.Context, request ports.ExecutionSessionEnsureRequest,
) (ports.ExecutionSessionReceipt, error) {
	broker.ensures++
	authority, err := application.DeriveExecutionSessionAuthority(request, "execution_token")
	return ports.ExecutionSessionReceipt{Authority: authority, EnsuredAt: broker.at}, err
}
func (broker *sqliteExecutionSessionBroker) Revoke(
	_ context.Context, request ports.ExecutionSessionEnsureRequest,
) error {
	broker.revokes++
	if _, err := application.DeriveExecutionSessionAuthority(request, "execution_token"); err != nil {
		return err
	}
	replay := broker.revoked
	broker.revoked = true
	if broker.revokeFailures > 0 {
		broker.revokeFailures--
		return errors.New("credential store unavailable")
	}
	if replay {
		broker.replays++
	}
	return nil
}
func TestExecutionServicePrincipalExactScopeRevocationAndRestart(t *testing.T) {
	ctx, system := context.Background(), newSQLiteV15System(t, 2)
	broker := &sqliteExecutionSessionBroker{at: system.clock.Now()}
	system.orchestrator = sqliteV22Orchestrator(t, system, broker)
	if persisted, err := sqliteTableHasColumn(ctx, system.repository.db, "executions", "execution_session_ref"); err != nil || !persisted {
		t.Fatalf("execution_session_ref schema persisted=%t err=%v", persisted, err)
	}
	created := system.submit(t, "request:v22-execution-authority")
	queued := created.Record.Executions[0]
	if _, err := system.repository.ExecutionSessionAuthority(ctx, queued.Ref, "execution_token"); !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("queued execution acquired authority err=%v", err)
	}
	if result, err := system.orchestrator.ProcessNext(ctx, "worker:v22-execution-authority"); err != nil || !result.Processed {
		t.Fatalf("ProcessNext result=%+v err=%v", result, err)
	}
	if broker.ensures != 1 {
		t.Fatalf("execution session ensures=%d", broker.ensures)
	}
	var durableSession string
	if err := system.repository.db.QueryRowContext(ctx,
		`SELECT execution_session_ref FROM executions WHERE ref=?`, queued.Ref.String(),
	).Scan(&durableSession); err != nil {
		t.Fatal(err)
	}
	record, err := system.repository.GetGoal(ctx, created.Record.Goal.Ref())
	if err != nil || len(record.Executions) != 1 {
		t.Fatalf("GetGoal executions=%d err=%v", len(record.Executions), err)
	}
	execution := record.Executions[0]
	authority, err := system.repository.ExecutionSessionAuthority(ctx, execution.Ref, "execution_token")
	if err != nil {
		t.Fatal(err)
	}
	if durableSession != authority.SessionRef.String() || execution.ExecutionSessionRef != authority.SessionRef {
		t.Fatalf("durable execution session row=%s record=%s authority=%s",
			durableSession, execution.ExecutionSessionRef, authority.SessionRef)
	}
	if authority.Request != application.ExecutionSessionRequest(record.Goal, execution) {
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
	record, err = system.repository.GetGoal(ctx, created.Record.Goal.Ref())
	if err != nil || len(record.Executions) != 1 {
		t.Fatalf("restart GetGoal executions=%d err=%v", len(record.Executions), err)
	}
	if record.Executions[0].ExecutionSessionRef != authority.SessionRef {
		t.Fatalf("restart durable execution session=%s authority=%s",
			record.Executions[0].ExecutionSessionRef, authority.SessionRef)
	}
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
	successor, err := system.repository.ExecutionSessionAuthority(ctx, successorRef, "execution_token")
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

func TestRevocationSchedulingCoversFailureStopAndReplacement(t *testing.T) {
	const session = "execution-session:sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	for _, state := range []application.ExecutionState{
		application.ExecutionFailed, application.ExecutionStopped, application.ExecutionCanceled,
	} {
		t.Run(string(state), func(t *testing.T) {
			ctx := context.Background()
			system := newSQLiteV15System(t, 2)
			execution := system.submit(t, "request:v22-revoke-"+string(state)).Record.Executions[0]
			if _, err := system.repository.db.ExecContext(ctx, `
UPDATE executions SET state=?,execution_session_ref=?,finished_at=created_at WHERE ref=?`,
				state, session, execution.Ref.String()); err != nil {
				t.Fatal(err)
			}
			tx, err := system.repository.db.BeginTx(ctx, nil)
			if err == nil {
				err = scheduleExecutionSessionRevocation(ctx, tx, execution.GoalRef,
					execution.WorkItemRef, execution.Ref, "", system.clock.Now())
			}
			if err == nil {
				err = scheduleExecutionSessionRevocation(ctx, tx, execution.GoalRef,
					execution.WorkItemRef, execution.Ref, "", system.clock.Now())
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			var actions int
			_ = system.repository.db.QueryRow(`SELECT COUNT(*) FROM outbox
WHERE kind='revoke_execution_session' AND execution_ref=?`, execution.Ref.String()).Scan(&actions)
			if actions != 1 {
				t.Fatalf("state=%s actions=%d", state, actions)
			}
		})
	}
}

func TestTerminalExecutionRevocationRetriesDurablyAcrossRestart(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteV15System(t, 2)
	broker := &sqliteExecutionSessionBroker{at: system.clock.Now(), revokeFailures: 1}
	system.orchestrator = sqliteV22Orchestrator(t, system, broker)
	system.submit(t, "request:v22-credential-revocation")
	for _, want := range []application.ActionKind{application.ActionLaunchAgent, application.ActionObserveAgent} {
		result, err := system.orchestrator.ProcessNext(ctx, "worker:v22-credential-revocation")
		if err != nil || !result.Processed || result.Action != want {
			t.Fatalf("ProcessNext result=%+v want=%s err=%v", result, want, err)
		}
	}
	if broker.ensures != 1 || broker.revokes != 0 {
		t.Fatalf("before revocation ensures=%d revokes=%d", broker.ensures, broker.revokes)
	}
	result, err := system.orchestrator.ProcessNext(ctx, "worker:v22-credential-revocation")
	if err != nil || result.Action != application.ActionRevokeSession || broker.revokes != 1 || !broker.revoked {
		t.Fatalf("failed revoke result=%+v calls=%d err=%v", result, broker.revokes, err)
	}
	var completed, attempts int
	if err := system.repository.db.QueryRow(`SELECT completed_at IS NOT NULL,delivery_attempt FROM outbox
WHERE kind='revoke_execution_session'`).Scan(&completed, &attempts); err != nil ||
		completed != 0 || attempts != 1 {
		t.Fatalf("pending revocation completed=%d attempts=%d err=%v", completed, attempts, err)
	}
	system.clock.Advance(2 * time.Second)
	restartSQLiteV15System(t, system)
	system.orchestrator = sqliteV22Orchestrator(t, system, broker)
	result, err = system.orchestrator.ProcessNext(ctx, "worker:v22-credential-revocation-restart")
	if err != nil || result.Action != application.ActionRevokeSession || broker.revokes != 2 || broker.replays != 1 {
		t.Fatalf("restart revoke result=%+v calls=%d err=%v", result, broker.revokes, err)
	}
	var receipts int
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM action_consumption_receipts
WHERE kind='revoke_execution_session' AND outcome='completed'`).Scan(&receipts); err != nil || receipts != 1 {
		t.Fatalf("revocation receipts=%d err=%v", receipts, err)
	}
	if _, _, err := validateRecoveryDatabase(ctx, system.repository.db); err != nil {
		t.Fatalf("recovery after revocation: %v", err)
	}
}

func sqliteV22Orchestrator(
	t *testing.T, system *sqliteV15System, sessions ports.ExecutionSessionBroker,
) *application.Orchestrator {
	t.Helper()
	_, fuentes := prepararCapacidadSQLiteV15(t, system.repository, system.clock, 1_000)
	orchestrator, err := application.New(application.Dependencies{
		State: system.repository, Access: system.repository, Launcher: system.external,
		Observer: system.external, Controller: system.external, Artifacts: system.external,
		Clock: system.clock, IDs: system.ids, MaxOutputBytes: 1024, MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts: 3, MaxChildrenPerParent: 6, ClaimLease: time.Minute,
		DirectorLeaseDuration: 30 * time.Second, EffectApprovalTTL: system.policy.EffectApprovalTTL,
		BudgetPolicy: system.policy, ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities: sqliteTestCapabilities(), ExecutionSessions: sessions,
		CapacitySources: fuentes, CapacityObservationWait: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	return orchestrator
}
