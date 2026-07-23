package sqlite

import (
	"context"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestGovernanceAuditProjectionReadsExistingCausalFactsWithoutOwningApplicationWrites(t *testing.T) {
	t.Run("director", func(t *testing.T) {
		system := newSQLiteDirectorSystem(t)
		claim, err := system.orchestrator.ClaimDirector(
			context.Background(), system.ownerAccess,
			application.ClaimDirectorRequest{
				RequestRef: "request:v20-projection-director-claim",
				GoalRef:    system.goal.Goal.Ref(),
			},
		)
		sqliteTestNoError(t, err)
		current, err := system.repository.GetGoal(context.Background(), system.goal.Goal.Ref())
		sqliteTestNoError(t, err)
		decision, err := system.orchestrator.ProposeDirectorPlan(
			context.Background(), system.ownerAccess, application.ProposeDirectorPlanRequest{
				RequestRef: "request:v20-projection-director", GoalRef: current.Goal.Ref(),
				ExpectedGoalRevision:   current.Goal.Revision(),
				ExpectedPlanGeneration: current.Goal.PlanGeneration(),
				LeaseToken:             claim.Lease.Token, LeaseFence: claim.Lease.Fence,
				Reason: "project the exact director chain",
				Plan: application.PlanSpec{WorkItems: []application.WorkItemSpec{{
					Key: "work:v20-projected", Objective: "produce projected evidence",
					Phase: goal.DefaultPhaseKey().String(), Role: "role:worker",
					OutputContract: goal.OutputContractEvidenceBundle,
				}}},
			},
		)
		sqliteTestNoError(t, err)
		auditRef := v20RecordCompletedCommand(t, system.repository, "director",
			"orquesta.director.plan.propose", decision.Decision.RequestRef,
			decision.Decision.PrincipalRef.String(), system.project.String())
		projection := v20Projection(t, system.repository, auditRef)
		v20RequireFacts(t, projection,
			ports.CommandGovernanceAuthorization, ports.CommandGovernanceDirectorDecision,
			ports.CommandGovernanceEffectIntent, ports.CommandGovernanceCausalEvent,
		)
	})

	t.Run("effect intent attempt receipt", func(t *testing.T) {
		system, created, principal := v20SensitiveGoal(t, false)
		intent := created.Record.EffectIntents[0]
		decision, err := system.orchestrator.DecideEffect(
			context.Background(), system.access, application.DecideEffectRequest{
				RequestRef: "request:v20-projection-effect", GoalRef: created.Record.Goal.Ref(),
				IntentRef: intent.Ref, ExpectedIntentDigest: intent.Digest,
				Decision: application.EffectApproved, Reason: "approve one bounded physical effect",
			},
		)
		sqliteTestNoError(t, err)
		processed, err := system.orchestrator.ProcessNext(context.Background(), "worker:v20-projection")
		if err != nil || !processed.Processed || processed.Action != application.ActionLaunchAgent {
			t.Fatalf("process=%+v err=%v", processed, err)
		}
		auditRef := v20RecordCompletedCommand(t, system.repository, "effect",
			"orquesta.effects.decide", decision.Approval.RequestRef,
			principal.Ref.String(), system.project.String())
		projection := v20Projection(t, system.repository, auditRef)
		v20RequireFacts(t, projection,
			ports.CommandGovernanceAuthorization, ports.CommandGovernanceEffectDecision,
			ports.CommandGovernanceEffectIntent, ports.CommandGovernanceEffectAttempt,
			ports.CommandGovernanceEffectReceipt, ports.CommandGovernanceCausalEvent,
		)
	})

	t.Run("control", func(t *testing.T) {
		system := newSQLiteV15System(t, 1)
		created := system.submit(t, "request:v20-projection-control-goal")
		principal := testPrincipal(t, "principal:v15-owner", "actor:v15-owner", identity.PrincipalKindHuman)
		current, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
		sqliteTestNoError(t, err)
		control, err := system.orchestrator.Control(
			context.Background(), system.access, application.ControlRequest{
				RequestRef: "request:v20-projection-control", Operation: application.ControlCancel,
				Target: application.ControlTargetGoal, GoalRef: current.Goal.Ref(),
				ExpectedGoalRevision:      current.Goal.Revision(),
				ExpectedPlanGeneration:    current.Goal.PlanGeneration(),
				ExpectedAppSpecGeneration: current.Goal.AppSpec().Generation(),
				ExpectedSpecHash:          current.Goal.SpecHash(), Reason: "close governed work",
			},
		)
		sqliteTestNoError(t, err)
		auditRef := v20RecordCompletedCommand(t, system.repository, "control",
			"orquesta.goals.control", control.Control.RequestRef,
			principal.Ref.String(), system.project.String())
		projection := v20Projection(t, system.repository, auditRef)
		v20RequireFacts(t, projection,
			ports.CommandGovernanceAuthorization, ports.CommandGovernanceClosureControl,
			ports.CommandGovernanceCausalEvent, ports.CommandGovernanceTerminalEvent,
		)
	})
}

func TestGovernanceAuditProjectionReadsCouncilRootsAndDescendants(t *testing.T) {
	t.Run("round", func(t *testing.T) {
		system, goalRef := seedSQLiteV19Council(t, council.PolicyRequired, false)
		principal := testPrincipal(t, "principal:v15-owner", "actor:v15-owner", identity.PrincipalKindHuman)
		record, err := system.repository.GetGoal(context.Background(), goalRef)
		sqliteTestNoError(t, err)
		claim, err := system.orchestrator.ClaimDirector(
			context.Background(), system.access,
			application.ClaimDirectorRequest{RequestRef: "request:v20-council-claim", GoalRef: goalRef},
		)
		sqliteTestNoError(t, err)
		round, err := system.orchestrator.OpenCouncilRound(
			context.Background(), system.access, application.OpenCouncilRoundRequest{
				RequestRef: "request:v20-council-round", GoalRef: goalRef,
				ChangeRef:            record.ChangeSets[0].Ref,
				ExpectedGoalRevision: record.Goal.Revision(),
				ExpectedItemRevision: record.Goal.WorkItems()[0].Revision(),
				LeaseToken:           claim.Lease.Token, LeaseFence: claim.Lease.Fence,
			},
		)
		sqliteTestNoError(t, err)
		auditRef := v20RecordCompletedCommand(t, system.repository, "council-round",
			"orquesta.council.round.open", round.Round.RequestRef,
			principal.Ref.String(), system.project.String())
		projection := v20Projection(t, system.repository, auditRef)
		v20RequireFacts(t, projection,
			ports.CommandGovernanceAuthorization, ports.CommandGovernanceCouncilRound,
			ports.CommandGovernanceEffectIntent,
		)
	})

	t.Run("skip", func(t *testing.T) {
		system, goalRef := seedSQLiteV19Council(t, council.PolicySkipByOperator, false)
		principal := testPrincipal(t, "principal:v15-owner", "actor:v15-owner", identity.PrincipalKindHuman)
		record, err := system.repository.GetGoal(context.Background(), goalRef)
		sqliteTestNoError(t, err)
		skip, err := system.orchestrator.SkipCouncil(
			context.Background(), system.access, application.SkipCouncilRequest{
				RequestRef: "request:v20-council-skip", GoalRef: goalRef,
				ChangeRef:            record.ChangeSets[0].Ref,
				ExpectedGoalRevision: record.Goal.Revision(),
				ExpectedItemRevision: record.Goal.WorkItems()[0].Revision(),
				Reason:               "operator accepted the reviewed bounded change",
			},
		)
		sqliteTestNoError(t, err)
		auditRef := v20RecordCompletedCommand(t, system.repository, "council-skip",
			"orquesta.council.skip", skip.Skip.RequestRef,
			principal.Ref.String(), system.project.String())
		projection := v20Projection(t, system.repository, auditRef)
		v20RequireFacts(t, projection,
			ports.CommandGovernanceAuthorization, ports.CommandGovernanceCouncilSkip,
		)
	})
}

func TestGovernanceAuditProjectionReadsTerminalAttestationChain(t *testing.T) {
	system, _, principal := v20SensitiveGoal(t, true)
	v20ApproveEffect(t, system, principal, application.EffectKindPrepareWorkspace, "prepare")
	processSQLiteV16Actions(t, system, application.ActionPrepareWorkspace)
	v20ApproveEffect(t, system, principal, application.EffectKindAgentLaunch, "launch")
	processSQLiteV16Actions(t, system, application.ActionLaunchAgent, application.ActionObserveAgent)
	v20ApproveEffect(t, system, principal, application.EffectKindCommitChange, "commit")
	processSQLiteV16Actions(t, system, application.ActionCommitChange)
	auditRef := v20ApproveEffect(t, system, principal, application.EffectKindAttestTest, "attest")
	processSQLiteV16Actions(t, system, application.ActionAttestTest)

	projection := v20Projection(t, system.repository, auditRef)
	v20RequireFacts(t, projection,
		ports.CommandGovernanceAuthorization, ports.CommandGovernanceEffectDecision,
		ports.CommandGovernanceEffectIntent, ports.CommandGovernanceEffectAttempt,
		ports.CommandGovernanceEffectReceipt, ports.CommandGovernanceCausalEvent,
		ports.CommandGovernanceAttestation,
	)
}

func v20SensitiveGoal(
	t *testing.T,
	withTests bool,
) (*sqliteV15System, application.SubmitResult, identity.Principal) {
	t.Helper()
	system := newSQLiteV15System(t, 4)
	if withTests {
		system.orchestrator = newSQLiteV16OrchestratorWithAttestor(t, system, &sqliteTestAttestor{})
	}
	principal := testPrincipal(t, "principal:v15-owner", "actor:v15-owner", identity.PrincipalKindHuman)
	work := application.WorkItemSpec{
		Key: "work:v20-effect", Objective: "produce governed effect", Phase: "phase:v20-effect",
		Role: "role:worker", OutputContract: goal.OutputContractEvidenceBundle,
		SecurityCriticality: governance.SecurityCriticalitySensitive,
		ReasoningEffort:     governance.ReasoningEffortMedium,
	}
	if withTests {
		work.WriteSet = []string{"internal/v20"}
		work.RequiredTests = sqliteRequiredTestSpecs("required-test:v20-projection")
		work.CouncilPolicy = council.PolicyAuto
	}
	created, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
		RequestRef: "request:v20-effect-goal", Statement: "explicit sensitive launch", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:v20-effect", Key: "phase:v20-effect",
				TemplateRef: "phase-template:v20-effect",
			}},
			WorkItems: []application.WorkItemSpec{work},
		},
	})
	sqliteTestNoError(t, err)
	return system, created, principal
}

func v20RecordCompletedCommand(
	t *testing.T,
	repository *Repository,
	suffix, commandID, requestRef, principalRef, projectRef string,
) string {
	t.Helper()
	record := commandAuditTestRecord("projection-" + suffix)
	record.CommandID = commandID
	record.RequestRef = requestRef
	record.PrincipalRef = principalRef
	record.ProjectRef = projectRef
	record.ReplayMode = ports.CommandReplayApplicationReceipt
	session, err := repository.Begin(context.Background(), record)
	if err != nil || !session.AdmissionCreated {
		t.Fatalf("audit admission=%+v err=%v", session, err)
	}
	completion, err := repository.Complete(context.Background(), ports.CommandAuditCompletionRequest{
		RecordRef: record.Ref,
		Terminal: ports.CommandAuditTerminal{
			OutcomeRef: session.OutcomeRef, Status: "completed",
			OutputDigest: strings.Repeat("d", 64),
		},
	})
	if err != nil || !completion.Created || completion.Terminal.Status != "completed" {
		t.Fatalf("audit completion=%+v err=%v", completion, err)
	}
	return record.Ref
}

func v20ApproveEffect(
	t *testing.T,
	system *sqliteV15System,
	principal identity.Principal,
	kind application.EffectKind,
	suffix string,
) string {
	t.Helper()
	var intentRef, digest, goalRef string
	err := system.repository.db.QueryRow(`
SELECT intent.ref,intent.digest,intent.goal_ref
FROM effect_intents intent
WHERE intent.kind=? AND NOT EXISTS(
 SELECT 1 FROM effect_approvals approval WHERE approval.intent_ref=intent.ref
)
ORDER BY intent.created_at,intent.ref LIMIT 1`, string(kind)).Scan(&intentRef, &digest, &goalRef)
	sqliteTestNoError(t, err)
	decision, err := system.orchestrator.DecideEffect(
		context.Background(), system.access, application.DecideEffectRequest{
			RequestRef: "request:v20-projection-" + suffix,
			GoalRef:    mustRef(t, goalRef, goal.NewGoalRef),
			IntentRef:  intentRef, ExpectedIntentDigest: digest,
			Decision: application.EffectApproved, Reason: "approve bounded " + suffix + " effect",
		},
	)
	sqliteTestNoError(t, err)
	return v20RecordCompletedCommand(t, system.repository, "approval-"+suffix,
		"orquesta.effects.decide", decision.Approval.RequestRef,
		principal.Ref.String(), system.project.String())
}

func v20Projection(
	t *testing.T,
	repository *Repository,
	invocationRef string,
) ports.CommandGovernanceProjection {
	t.Helper()
	projection, err := repository.ProjectCommandGovernance(context.Background(), invocationRef)
	if err != nil || projection.InvocationRef != invocationRef ||
		projection.OutcomeRef != invocationRef+":outcome" || projection.OutcomeStatus != "completed" ||
		projection.ErrorCode != "" || projection.AuthorizationReceiptRef == "" ||
		projection.FactRef == "" || len(projection.Facts) < 2 {
		t.Fatalf("projection=%+v err=%v", projection, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err != nil {
		t.Fatalf("projected governance recovery=%v", err)
	}
	return projection
}

func v20RequireFacts(
	t *testing.T,
	projection ports.CommandGovernanceProjection,
	kinds ...ports.CommandGovernanceFactKind,
) {
	t.Helper()
	seen := make(map[ports.CommandGovernanceFactKind]bool, len(projection.Facts))
	for _, fact := range projection.Facts {
		if fact.Ref == "" || fact.CausalRef == "" {
			t.Fatalf("incomplete causal fact=%+v projection=%+v", fact, projection)
		}
		seen[fact.Kind] = true
	}
	for _, kind := range kinds {
		if !seen[kind] {
			t.Fatalf("missing fact kind=%s projection=%+v", kind, projection)
		}
	}
}
