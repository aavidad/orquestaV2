package commands

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

type auditEntry struct {
	record    CommandAuditRecord
	terminals map[string]CommandAuditTerminal
}
type memoryAudit struct {
	mu      sync.Mutex
	entries map[string]auditEntry
	admits  int
	now     time.Time
}

func newMemoryAudit() *memoryAudit {
	return &memoryAudit{entries: make(map[string]auditEntry), now: time.Date(2026, 7, 22, 20, 0, 0, 0, time.UTC)}
}
func (audit *memoryAudit) set(value time.Time) { audit.mu.Lock(); audit.now = value; audit.mu.Unlock() }
func (audit *memoryAudit) Begin(_ context.Context, record CommandAuditRecord) (AuditSession, error) {
	audit.mu.Lock()
	defer audit.mu.Unlock()
	audit.admits++
	entry, exists := audit.entries[record.Ref]
	if exists && !sameAdmissionFacts(record, entry.record) {
		return AuditSession{}, ErrAuditConflict
	}
	if !exists {
		record.AdmittedAt = audit.now
		entry = auditEntry{record: record, terminals: make(map[string]CommandAuditTerminal)}
		audit.entries[record.Ref] = entry
	}
	outcomeRef := record.Ref + ":outcome"
	terminal, completed := entry.terminals[outcomeRef]
	session := AuditSession{Record: entry.record, OutcomeRef: outcomeRef, AdmissionCreated: !exists}
	if completed {
		session.Terminal = &terminal
	}
	return session, nil
}
func (audit *memoryAudit) Complete(_ context.Context, request AuditCompletionRequest) (AuditCompletion, error) {
	audit.mu.Lock()
	defer audit.mu.Unlock()
	entry, ok := audit.entries[request.RecordRef]
	if !ok {
		return AuditCompletion{}, errors.New("missing")
	}
	if terminal, exists := entry.terminals[request.Terminal.OutcomeRef]; exists {
		return AuditCompletion{Terminal: terminal}, nil
	}
	terminal := request.Terminal
	terminal.CompletedAt = audit.now
	entry.terminals[terminal.OutcomeRef] = terminal
	audit.entries[request.RecordRef] = entry
	return AuditCompletion{Terminal: terminal, Created: true}, nil
}

type fakeApplication struct {
	applicationAPI
	mu          sync.Mutex
	calls       map[string]int
	statusGoals int64
	statusErr   error
	submissions []application.SubmitRequest
	proposals   []application.ProposeDirectorPlanRequest
}

func newFakeApplication() *fakeApplication { return &fakeApplication{calls: make(map[string]int)} }
func (api *fakeApplication) called(name string) int {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.calls[name]++
	return api.calls[name]
}
func (api *fakeApplication) Submit(_ context.Context, _ application.Access, request application.SubmitRequest) (application.SubmitResult, error) {
	created := api.called("Submit") == 1
	api.mu.Lock()
	api.submissions = append(api.submissions, request)
	api.mu.Unlock()
	return application.SubmitResult{Created: created}, nil
}
func (api *fakeApplication) Amend(context.Context, application.Access, application.AmendRequest) (application.AmendResult, error) {
	return application.AmendResult{Created: api.called("Amend") == 1}, nil
}
func (api *fakeApplication) GetGoal(context.Context, application.Access, goal.GoalRef) (application.GoalRecord, error) {
	api.called("GetGoal")
	return application.GoalRecord{}, nil
}
func (api *fakeApplication) ListGoals(context.Context, application.Access, int) ([]application.GoalSummary, error) {
	api.called("ListGoals")
	return []application.GoalSummary{}, nil
}
func (api *fakeApplication) GetArtifact(context.Context, application.Access, goal.GoalRef, goal.ArtifactRef) (ports.ArtifactContent, error) {
	api.called("GetArtifact")
	return ports.ArtifactContent{}, nil
}
func (api *fakeApplication) Status(context.Context, application.Access) (application.RepositoryStatus, error) {
	api.called("Status")
	api.mu.Lock()
	goals := api.statusGoals
	api.mu.Unlock()
	return application.RepositoryStatus{Goals: goals}, api.statusErr
}
func (api *fakeApplication) GrantMembership(_ context.Context, _ application.Access, request identity.MembershipGrantRequest, _ identity.Principal) (identity.Membership, identity.MembershipAuditReceipt, bool, error) {
	created := api.called("GrantMembership") == 1
	membershipRevision := request.ExpectedRevision() + 1
	if !created {
		membershipRevision++
	}
	membership, _ := identity.NewMembership(identity.MembershipInput{
		PrincipalRef: request.TargetRef(), ProjectRef: request.ProjectRef(), Role: request.Role(),
		Revision: membershipRevision, Status: identity.MembershipActive,
		GrantedBy: request.Actor().Ref, GrantedAt: request.RequestedAt(),
	})
	receipt, _ := identity.NewMembershipAuditReceipt(identity.MembershipAuditReceiptInput{
		Ref: "membership-audit:grant", RequestRef: request.RequestRef(), Action: identity.MembershipAuditGranted,
		ActorRef: request.Actor().Ref, TargetRef: request.TargetRef(), ProjectRef: request.ProjectRef(), Role: request.Role(),
		PreviousRevision: request.ExpectedRevision(), Revision: request.ExpectedRevision() + 1, OccurredAt: request.RequestedAt(),
	})
	return membership, receipt, created, nil
}
func (api *fakeApplication) RevokeMembership(_ context.Context, _ application.Access, request identity.MembershipRevokeRequest) (identity.Membership, identity.MembershipAuditReceipt, bool, error) {
	revoked := api.called("RevokeMembership") == 1
	membership, _ := identity.NewMembership(identity.MembershipInput{
		PrincipalRef: request.TargetRef(), ProjectRef: request.ProjectRef(), Role: identity.RoleContributor,
		Revision: request.ExpectedRevision() + 1, Status: identity.MembershipRevoked,
		GrantedBy: request.Actor().Ref, GrantedAt: request.RequestedAt().Add(-time.Second),
		RevokedBy: request.Actor().Ref, RevokedAt: request.RequestedAt(),
	})
	receipt, _ := identity.NewMembershipAuditReceipt(identity.MembershipAuditReceiptInput{
		Ref: "membership-audit:revoke", RequestRef: request.RequestRef(), Action: identity.MembershipAuditRevoked,
		ActorRef: request.Actor().Ref, TargetRef: request.TargetRef(), ProjectRef: request.ProjectRef(), Role: identity.RoleContributor,
		PreviousRevision: request.ExpectedRevision(), Revision: request.ExpectedRevision() + 1, OccurredAt: request.RequestedAt(),
	})
	return membership, receipt, revoked, nil
}
func (api *fakeApplication) ClaimDirector(context.Context, application.Access, application.ClaimDirectorRequest) (application.DirectorLeaseResult, error) {
	changed := api.called("ClaimDirector") == 1
	principalRef, _ := identity.NewPrincipalRef("principal:test")
	goalRef, _ := goal.NewGoalRef("goal:test")
	return application.DirectorLeaseResult{Lease: application.DirectorLeaseRecord{
		GoalRef: goalRef, PrincipalRef: principalRef, Token: "director-lease:test", Fence: 1,
		LeaseUntil: time.Date(2026, 7, 22, 21, 0, 0, 0, time.UTC),
	}, Changed: changed}, nil
}
func (api *fakeApplication) RenewDirector(context.Context, application.Access, application.RenewDirectorRequest) (application.DirectorLeaseResult, error) {
	changed := api.called("RenewDirector") == 1
	principalRef, _ := identity.NewPrincipalRef("principal:test")
	goalRef, _ := goal.NewGoalRef("goal:test")
	return application.DirectorLeaseResult{Lease: application.DirectorLeaseRecord{
		GoalRef: goalRef, PrincipalRef: principalRef, Token: "director-lease:test", Fence: 1,
		LeaseUntil: time.Date(2026, 7, 22, 21, 0, 0, 0, time.UTC),
	}, Changed: changed}, nil
}
func (api *fakeApplication) ProposeDirectorPlan(_ context.Context, _ application.Access, request application.ProposeDirectorPlanRequest) (application.DirectorPlanResult, error) {
	created := api.called("ProposeDirectorPlan") == 1
	api.mu.Lock()
	api.proposals = append(api.proposals, request)
	api.mu.Unlock()
	goalRef, _ := goal.NewGoalRef("goal:test")
	return application.DirectorPlanResult{Decision: application.DirectorDecisionRecord{
		Ref: "director-decision:test", GoalRef: goalRef, LeaseFence: 1,
		AppliedGoalRevision: 2, AppliedPlanGeneration: 2,
	}, Created: created}, nil
}
func (api *fakeApplication) Control(context.Context, application.Access, application.ControlRequest) (application.ControlResult, error) {
	created := api.called("Control") == 1
	status := application.ControlRequested
	if !created {
		status = application.ControlConfirmed
	}
	return application.ControlResult{Control: application.ControlRecord{Ref: "control:test", Status: status}, Created: created}, nil
}
func (api *fakeApplication) DecideEffect(context.Context, application.Access, application.DecideEffectRequest) (application.DecideEffectResult, error) {
	return application.DecideEffectResult{Approval: application.EffectApproval{
		Ref: "effect-approval:test", IntentRef: "effect-intent:test", Decision: application.EffectApproved,
	}, Created: api.called("DecideEffect") == 1}, nil
}
func (api *fakeApplication) ListPendingChanges(context.Context, application.Access, application.ListPendingChangesRequest) (application.ListPendingChangesResult, error) {
	api.called("ListPendingChanges")
	return application.ListPendingChangesResult{}, nil
}
func (api *fakeApplication) IntegrateChange(context.Context, application.Access, application.IntegrateChangeRequest) (application.IntegrateChangeResult, error) {
	return application.IntegrateChangeResult{Action: application.ActionRecord{Ref: "action:integrate:test"}, Created: api.called("IntegrateChange") == 1}, nil
}
func (api *fakeApplication) AdmitMailbox(_ context.Context, _ application.Access, request application.AdmitMailboxRequest) (application.MailboxAdmissionResult, error) {
	created := api.called("AdmitMailbox") == 1
	messageRef, _ := application.NewMailboxMessageRef("mailbox-message:test")
	state := application.MailboxStateAdmitted
	if !created {
		state = application.MailboxStateDelivered
	}
	record := application.MailboxRecord{
		Envelope:  application.MailboxEnvelope{Ref: messageRef, GoalRef: request.GoalRef},
		Admission: application.MailboxAdmissionReceipt{Ref: "mailbox-admission:test", MessageRef: messageRef},
		State:     state,
	}
	return application.MailboxAdmissionResult{Record: record, Created: created}, nil
}
func (api *fakeApplication) ClaimMailbox(_ context.Context, _ application.Access, request application.ClaimMailboxRequest) (application.MailboxClaimResult, error) {
	claimed := api.called("ClaimMailbox") == 1
	principalRef, _ := identity.NewPrincipalRef("principal:test")
	claim := application.MailboxClaim{Attempt: application.MailboxDeliveryAttempt{
		MessageRef: request.MessageRef,
		Recipient:  application.MailboxEndpoint{PrincipalRef: principalRef, WorkItemRef: request.RecipientWorkItemRef, ExecutionRef: request.RecipientExecutionRef},
		ClaimToken: "mailbox-claim:test", Fence: 1,
	}}
	if !claimed {
		claim.Record.State = application.MailboxStateDelivered
	}
	return application.MailboxClaimResult{
		Claim: claim, GoalRevision: 1, PlanGeneration: 1, Claimed: claimed,
	}, nil
}
func (api *fakeApplication) MarkMailboxDelivered(_ context.Context, _ application.Access, request application.MarkMailboxDeliveredRequest) (application.MailboxMutationResult, error) {
	changed := api.called("MarkMailboxDelivered") == 1
	record := mailboxMutationRecord(request.MessageRef, request.RecipientWorkItemRef, request.RecipientExecutionRef, !changed)
	return application.MailboxMutationResult{Record: record, Changed: changed}, nil
}
func (api *fakeApplication) ConsumeMailbox(_ context.Context, _ application.Access, request application.ConsumeMailboxRequest) (application.MailboxMutationResult, error) {
	changed := api.called("ConsumeMailbox") == 1
	record := mailboxMutationRecord(request.MessageRef, request.RecipientWorkItemRef, request.RecipientExecutionRef, !changed)
	return application.MailboxMutationResult{Record: record, Changed: changed}, nil
}
func (api *fakeApplication) GetMailbox(context.Context, application.Access, application.GetMailboxRequest) (application.MailboxRecord, error) {
	api.called("GetMailbox")
	return application.MailboxRecord{}, nil
}
func (api *fakeApplication) ListMailbox(context.Context, application.Access, application.ListMailboxRequest) (application.ListMailboxResult, error) {
	api.called("ListMailbox")
	return application.ListMailboxResult{}, nil
}
func (api *fakeApplication) AcknowledgeMailbox(_ context.Context, _ application.Access, request application.AcknowledgeMailboxRequest) (application.MailboxResolutionResult, error) {
	return application.MailboxResolutionResult{Acknowledgement: application.MailboxAcknowledgement{
		Ref: "mailbox-acknowledgement:test", MessageRef: request.MessageRef, Outcome: application.MailboxOutcomeAcknowledged,
	}, Created: api.called("AcknowledgeMailbox") == 1}, nil
}
func (api *fakeApplication) BlockMailbox(_ context.Context, _ application.Access, request application.BlockMailboxRequest) (application.MailboxResolutionResult, error) {
	return application.MailboxResolutionResult{Acknowledgement: application.MailboxAcknowledgement{
		Ref: "mailbox-block:test", MessageRef: request.MessageRef, Outcome: application.MailboxOutcomeBlocked,
	}, Created: api.called("BlockMailbox") == 1}, nil
}

func (api *fakeApplication) OpenCouncilRound(_ context.Context, _ application.Access, request application.OpenCouncilRoundRequest) (application.OpenCouncilRoundResult, error) {
	created := api.called("OpenCouncilRound") == 1
	workItemRef, _ := goal.NewWorkItemRef("work-item:council")
	return application.OpenCouncilRoundResult{Round: application.CouncilRoundRecord{
		Ref: "council-round:test", GoalRef: request.GoalRef, WorkItemRef: workItemRef,
		ChangeSetRef:  request.ChangeRef.String(),
		SubjectDigest: application.CouncilSubjectDigest("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		Opener:        application.CouncilRoundOpenerDirector, DirectorFence: request.LeaseFence,
		OpenedAt: time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC),
	}, Created: created}, nil
}

func (api *fakeApplication) SkipCouncil(_ context.Context, _ application.Access, request application.SkipCouncilRequest) (application.SkipCouncilResult, error) {
	created := api.called("SkipCouncil") == 1
	return application.SkipCouncilResult{Skip: application.CouncilSkipRecord{
		Ref: "council-skip:test",
		Subject: council.Subject{
			GoalRef: "goal:test", WorkItemRef: "work-item:council", ChangeSetRef: request.ChangeRef.String(),
		},
		SubjectDigest: application.CouncilSubjectDigest("sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		Skip:          council.Skip{PrincipalRef: "principal:test", Reason: request.Reason},
		RecordedAt:    time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC),
	}, Created: created}, nil
}

func mailboxMutationRecord(messageRef application.MailboxMessageRef, workRef goal.WorkItemRef, executionRef goal.ExecutionRef, progressed bool) application.MailboxRecord {
	principalRef, _ := identity.NewPrincipalRef("principal:test")
	record := application.MailboxRecord{
		Envelope: application.MailboxEnvelope{Ref: messageRef},
		Attempts: []application.MailboxDeliveryAttempt{{
			MessageRef: messageRef, Recipient: application.MailboxEndpoint{PrincipalRef: principalRef, WorkItemRef: workRef, ExecutionRef: executionRef},
			ClaimToken: "claim:t", Fence: 1, DeliveryRef: "mailbox-delivery:test", ConsumptionRef: "mailbox-consumption:test",
		}},
	}
	if progressed {
		record.State = application.MailboxStateConsumed
		record.Attempts = append(record.Attempts, application.MailboxDeliveryAttempt{
			MessageRef: messageRef, Recipient: record.Attempts[0].Recipient,
			ClaimToken: "claim:later", Fence: 2, DeliveryRef: "mailbox-delivery:later", ConsumptionRef: "mailbox-consumption:later",
		})
	}
	return record
}

func testPrincipal(t *testing.T) identity.Principal {
	t.Helper()
	principalRef, err := identity.NewPrincipalRef("principal:test")
	if err != nil {
		t.Fatal(err)
	}
	actorRef, err := goal.NewActorRef("actor:test")
	if err != nil {
		t.Fatal(err)
	}
	principal, err := identity.NewPrincipal(principalRef, actorRef, identity.PrincipalKindHuman, "local_token")
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func testExecutionServicePrincipal(t *testing.T) identity.Principal {
	t.Helper()
	principalRef, _ := identity.NewPrincipalRef("principal:execution:test")
	actorRef, _ := goal.NewActorRef("actor:execution:test")
	principal, err := identity.NewPrincipal(principalRef, actorRef, identity.PrincipalKindService, "opaque_authn")
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

type testExecutionAuthority struct {
	mu                sync.Mutex
	expectedPrincipal identity.PrincipalRef
	expectedProject   goal.ProjectRef
	expectedExecution goal.ExecutionRef
	resolvedExecution goal.ExecutionRef
	err               error
	calls             int
}

func (authority *testExecutionAuthority) ResolveExecution(
	_ context.Context,
	principal identity.Principal,
	projectRef goal.ProjectRef,
	claimed goal.ExecutionRef,
) (goal.ExecutionRef, error) {
	authority.mu.Lock()
	defer authority.mu.Unlock()
	authority.calls++
	if authority.err != nil {
		return goal.ExecutionRef{}, authority.err
	}
	if authority.expectedPrincipal.String() != "" && principal.Ref != authority.expectedPrincipal {
		return goal.ExecutionRef{}, errors.New("execution authority principal denied")
	}
	if authority.expectedProject.String() != "" && projectRef != authority.expectedProject {
		return goal.ExecutionRef{}, errors.New("execution authority project denied")
	}
	if authority.expectedExecution.String() != "" && claimed != authority.expectedExecution {
		return goal.ExecutionRef{}, errors.New("execution authority claim denied")
	}
	if authority.resolvedExecution.String() != "" {
		return authority.resolvedExecution, nil
	}
	return claimed, nil
}

func (authority *testExecutionAuthority) ClassifyExecutionPrincipal(
	_ context.Context, principal identity.Principal, projectRef goal.ProjectRef,
) (bool, bool, goal.ExecutionRef, error) {
	if principal.Ref.String() != "principal:execution:test" {
		return false, false, goal.ExecutionRef{}, nil
	}
	executionRef, _ := goal.NewExecutionRef("execution:test")
	return true, projectRef.String() == "project:test", executionRef, nil
}

func exactTestExecutionAuthority(t *testing.T) *testExecutionAuthority {
	t.Helper()
	projectRef, err := goal.NewProjectRef("project:test")
	if err != nil {
		t.Fatal(err)
	}
	return &testExecutionAuthority{
		expectedPrincipal: testPrincipal(t).Ref,
		expectedProject:   projectRef,
	}
}

func testDispatcher(t *testing.T) (*Dispatcher, *fakeApplication, *memoryAudit) {
	t.Helper()
	api, audit := newFakeApplication(), newMemoryAudit()
	dispatcher, err := newDispatcher(
		api, audit, APILimits{MaxRequestBytes: 1 << 20, MaxListLimit: 100},
		exactTestExecutionAuthority(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	return dispatcher, api, audit
}

func invoke(t *testing.T, dispatcher *Dispatcher, id, requestRef string, payload any, execution bool) Result {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	invocation := Invocation{CommandID: id, CommandVersion: "1", RequestRef: requestRef, ProjectRef: "project:test", Principal: testPrincipal(t), Payload: encoded}
	if execution {
		invocation.ClaimedExecutionRef = "execution:test"
	}
	return dispatcher.Dispatch(context.Background(), invocation)
}

func TestCommandDispatcherRejectsUnknownCommandAndStrictSchemaViolations(t *testing.T) {
	dispatcher, _, audit := testDispatcher(t)
	unknown := invoke(t, dispatcher, "orquesta.unknown", "request:unknown", map[string]any{}, false)
	if unknown.Failure == nil || unknown.Failure.Code != CodeInvalidRequest {
		t.Fatalf("unknown=%+v", unknown)
	}
	invalid := invoke(t, dispatcher, "orquesta.system.status", "request:schema", map[string]any{"project_ref": "spoof"}, false)
	if invalid.Failure == nil || invalid.Failure.Code != CodeInvalidRequest || audit.admits != 0 {
		t.Fatalf("invalid=%+v admits=%d", invalid, audit.admits)
	}
	concatenated := Invocation{CommandID: "orquesta.system.status", CommandVersion: "1", RequestRef: "request:concat", ProjectRef: "project:test", Principal: testPrincipal(t), Payload: json.RawMessage(`{} {}`)}
	if result := dispatcher.Dispatch(context.Background(), concatenated); result.Failure == nil || result.Failure.Code != CodeInvalidRequest || audit.admits != 0 {
		t.Fatalf("concatenated=%+v admits=%d", result, audit.admits)
	}
}

func TestTerminalFailureReplayShortCircuitsHandlerAndValidatesDigest(t *testing.T) {
	dispatcher, api, audit := testDispatcher(t)
	results := make(map[string]Result)
	for _, test := range []struct {
		name, code string
		err        error
	}{
		{name: "rejected", code: CodeForbidden, err: application.ErrForbidden},
		{name: "failed", code: CodeUnavailable, err: context.DeadlineExceeded},
	} {
		before := api.calls["Status"]
		api.statusErr = test.err
		first := invoke(t, dispatcher, "orquesta.system.status", "request:terminal-"+test.name, map[string]any{}, false)
		api.statusErr = nil
		replay := invoke(t, dispatcher, "orquesta.system.status", "request:terminal-"+test.name, map[string]any{}, false)
		if first.Failure == nil || first.Failure.Code != test.code || replay.Failure == nil || replay.Failure.Code != test.code ||
			replay.AuditRef != first.AuditRef || api.calls["Status"] != before+1 {
			t.Fatalf("%s first=%+v replay=%+v calls=%v", test.name, first, replay, api.calls)
		}
		results[test.name] = first
	}

	first := results["rejected"]
	audit.mu.Lock()
	entry := audit.entries[first.AuditRef]
	terminal := entry.terminals[first.AuditRef+":outcome"]
	terminal.OutputDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	entry.terminals[first.AuditRef+":outcome"] = terminal
	audit.entries[first.AuditRef] = entry
	audit.mu.Unlock()
	corrupt := invoke(t, dispatcher, "orquesta.system.status", "request:terminal-rejected", map[string]any{}, false)
	if corrupt.Failure == nil || corrupt.Failure.Code != CodeUnavailable || api.calls["Status"] != 2 {
		t.Fatalf("corrupt=%+v calls=%v", corrupt, api.calls)
	}
}

func TestSchemaValidationPreservesGenericStringWhitespace(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"value":{"type":"string"}},"required":["value"],"additionalProperties":false}`)
	got, err := validatePayload(schema, json.RawMessage(`{"value":"  domain text  "}`))
	if err != nil || string(got) != `{"value":"  domain text  "}` {
		t.Fatalf("got=%s err=%v", got, err)
	}
}

func TestCommandDispatcherBindsPrincipalProjectAndExecutionOutsidePayload(t *testing.T) {
	payload := map[string]any{"goal_ref": "goal:test", "message_ref": "mailbox-message:test", "recipient_work_item_ref": "work-item:test"}
	encoded, _ := json.Marshal(payload)
	invocation := Invocation{
		CommandID: "orquesta.mailbox.claim", CommandVersion: "1",
		RequestRef: "request:execution-authority", ProjectRef: "project:test",
		ClaimedExecutionRef: "execution:test", Principal: testPrincipal(t), Payload: encoded,
	}

	t.Run("nil resolver fails before audit", func(t *testing.T) {
		api, audit := newFakeApplication(), newMemoryAudit()
		dispatcher, err := newDispatcher(api, audit, APILimits{MaxRequestBytes: 1 << 20, MaxListLimit: 100})
		if err != nil {
			t.Fatal(err)
		}
		result := dispatcher.Dispatch(context.Background(), invocation)
		if result.Failure == nil || result.Failure.Code != CodeForbidden ||
			result.AuditRef != "" || audit.admits != 0 || api.calls["ClaimMailbox"] != 0 {
			t.Fatalf("result=%+v admits=%d calls=%v", result, audit.admits, api.calls)
		}
	})

	t.Run("exact authority succeeds", func(t *testing.T) {
		dispatcher, api, audit := testDispatcher(t)
		result := dispatcher.Dispatch(context.Background(), invocation)
		if result.Failure != nil || result.AuditRef == "" ||
			audit.admits != 1 || api.calls["ClaimMailbox"] != 1 {
			t.Fatalf("result=%+v admits=%d calls=%v", result, audit.admits, api.calls)
		}
	})

	t.Run("resolver cannot substitute successor", func(t *testing.T) {
		api, audit := newFakeApplication(), newMemoryAudit()
		authority := exactTestExecutionAuthority(t)
		authority.resolvedExecution, _ = goal.NewExecutionRef("execution:successor")
		dispatcher, err := newDispatcher(
			api, audit, APILimits{MaxRequestBytes: 1 << 20, MaxListLimit: 100}, authority,
		)
		if err != nil {
			t.Fatal(err)
		}
		result := dispatcher.Dispatch(context.Background(), invocation)
		if result.Failure == nil || result.Failure.Code != CodeForbidden ||
			result.AuditRef != "" || audit.admits != 0 || api.calls["ClaimMailbox"] != 0 {
			t.Fatalf("result=%+v admits=%d calls=%v", result, audit.admits, api.calls)
		}
	})

	t.Run("other project fails before audit", func(t *testing.T) {
		api, audit := newFakeApplication(), newMemoryAudit()
		authority := exactTestExecutionAuthority(t)
		dispatcher, err := newDispatcher(
			api, audit, APILimits{MaxRequestBytes: 1 << 20, MaxListLimit: 100}, authority,
		)
		if err != nil {
			t.Fatal(err)
		}
		foreign := invocation
		foreign.ProjectRef = "project:other"
		result := dispatcher.Dispatch(context.Background(), foreign)
		if result.Failure == nil || result.Failure.Code != CodeForbidden ||
			result.AuditRef != "" || audit.admits != 0 || api.calls["ClaimMailbox"] != 0 {
			t.Fatalf("result=%+v admits=%d calls=%v", result, audit.admits, api.calls)
		}
	})

	t.Run("other principal fails before audit", func(t *testing.T) {
		api, audit := newFakeApplication(), newMemoryAudit()
		authority := exactTestExecutionAuthority(t)
		dispatcher, err := newDispatcher(
			api, audit, APILimits{MaxRequestBytes: 1 << 20, MaxListLimit: 100}, authority,
		)
		if err != nil {
			t.Fatal(err)
		}
		foreign := invocation
		principalRef, _ := identity.NewPrincipalRef("principal:other")
		actorRef, _ := goal.NewActorRef("actor:other")
		foreign.Principal, _ = identity.NewPrincipal(
			principalRef, actorRef, identity.PrincipalKindService, "test_service",
		)
		result := dispatcher.Dispatch(context.Background(), foreign)
		if result.Failure == nil || result.Failure.Code != CodeForbidden ||
			result.AuditRef != "" || audit.admits != 0 || api.calls["ClaimMailbox"] != 0 {
			t.Fatalf("result=%+v admits=%d calls=%v", result, audit.admits, api.calls)
		}
	})
}

func TestCallerClaimedExecutionNeverBecomesAuthority(t *testing.T) {
	dispatcher, _, audit := testDispatcher(t)
	result := invoke(t, dispatcher, "orquesta.system.status", "request:principal", map[string]any{}, true)
	if result.Failure == nil || result.Failure.Code != CodeInvalidRequest || audit.admits != 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestExecutionServicePrincipalCannotEnterPrincipalGoalCommandsBeforeAudit(t *testing.T) {
	dispatcher, api, audit := testDispatcher(t)
	invocation := Invocation{
		CommandID: "orquesta.goals.get", CommandVersion: "1", RequestRef: "request:execution-service-goal-get",
		ProjectRef: "project:test", Principal: testExecutionServicePrincipal(t), Payload: json.RawMessage(`{"goal_ref":"goal:test"}`),
	}
	result := dispatcher.Dispatch(context.Background(), invocation)
	if result.Failure == nil || result.Failure.Code != CodeForbidden || result.AuditRef != "" ||
		audit.admits != 0 || api.calls["GetGoal"] != 0 {
		t.Fatalf("goals.get result=%+v admits=%d calls=%v", result, audit.admits, api.calls)
	}

	invocation.CommandID, invocation.RequestRef = "orquesta.artifacts.read", "request:execution-service-artifact-read"
	invocation.Payload = json.RawMessage(`{"goal_ref":"goal:test","artifact_ref":"artifact:sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`)
	result = dispatcher.Dispatch(context.Background(), invocation)
	if result.Failure != nil || result.AuditRef == "" || audit.admits != 1 || api.calls["GetArtifact"] != 1 {
		t.Fatalf("artifacts.read result=%+v admits=%d calls=%v", result, audit.admits, api.calls)
	}
	generic := testExecutionServicePrincipal(t)
	generic.Ref, _ = identity.NewPrincipalRef("principal:generic-service")
	invocation.CommandID, invocation.RequestRef = "orquesta.system.status", "request:generic-service-status"
	invocation.Principal, invocation.Payload = generic, json.RawMessage(`{}`)
	result = dispatcher.Dispatch(context.Background(), invocation)
	if result.Failure != nil || audit.admits != 2 || api.calls["Status"] != 1 {
		t.Fatalf("generic service result=%+v admits=%d calls=%v", result, audit.admits, api.calls)
	}
}

func TestUnauthenticatedRequestNeverCreatesCommandAudit(t *testing.T) {
	dispatcher, _, audit := testDispatcher(t)
	result := dispatcher.Dispatch(context.Background(), Invocation{CommandID: "orquesta.system.status", CommandVersion: "1", RequestRef: "request:no-auth", ProjectRef: "project:test", Payload: json.RawMessage(`{}`)})
	if result.Failure == nil || result.Failure.Code != CodeUnauthenticated || audit.admits != 0 {
		t.Fatalf("result=%+v admits=%d", result, audit.admits)
	}
}

func TestCommandReplayUsesOriginalAdmissionAcrossClockChanges(t *testing.T) {
	dispatcher, api, audit := testDispatcher(t)
	payload := map[string]any{"statement": "build", "confirm": true}
	first := invoke(t, dispatcher, "orquesta.goals.create", "request:replay", payload, false)
	audit.set(time.Date(2026, 7, 22, 21, 0, 0, 0, time.UTC))
	second := invoke(t, dispatcher, "orquesta.goals.create", "request:replay", payload, false)
	if first.Failure != nil || second.Failure != nil || string(first.Data) != string(second.Data) || first.AuditRef != second.AuditRef || api.calls["Submit"] != 2 {
		t.Fatalf("first=%+v second=%+v calls=%v", first, second, api.calls)
	}
}

func TestQueryReplayUsesStableIdentityAndRejectsChangedObservation(t *testing.T) {
	dispatcher, api, audit := testDispatcher(t)
	api.statusGoals = 1
	first := invoke(t, dispatcher, "orquesta.system.status", "request:query", map[string]any{}, false)
	api.statusGoals = 2
	audit.set(time.Date(2026, 7, 22, 21, 0, 0, 0, time.UTC))
	second := invoke(t, dispatcher, "orquesta.system.status", "request:query", map[string]any{}, false)
	if first.Failure != nil || second.Failure == nil || second.Failure.Code != CodeConflict || second.Data != nil ||
		first.AuditRef != second.AuditRef || api.calls["Status"] != 2 || audit.admits != 2 {
		t.Fatalf("first=%s second=%s refs=%q/%q", first.Data, second.Data, first.AuditRef, second.AuditRef)
	}
}

func TestApplicationHandlersEnforceDescriptorPermissionAndExistingUseCaseAuthorization(t *testing.T) {
	for _, definition := range compiledDefinitions {
		if expectedHandlerPermissions[definition.Handler] != definition.Permission {
			t.Fatalf("%s permission=%s", definition.ID, definition.Permission)
		}
		mutated := definition
		mutated.Permission = "goals.create"
		if mutated.Permission != definition.Permission && validateDefinition(mutated) == nil {
			t.Fatalf("mutated permission accepted for %s", definition.ID)
		}
	}
}
