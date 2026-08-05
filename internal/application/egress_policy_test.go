package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type egressPolicyResolverStub struct {
	authority EgressPolicyAuthority
	err       error
	refs      []EgressPolicyRef
}

type sequentialEgressPolicyResolver struct {
	authorities []EgressPolicyAuthority
	refs        []EgressPolicyRef
}

func (resolver *sequentialEgressPolicyResolver) ResolveEgressPolicy(
	_ context.Context,
	ref EgressPolicyRef,
) (EgressPolicyAuthority, error) {
	resolver.refs = append(resolver.refs, ref)
	index := len(resolver.refs) - 1
	if index >= len(resolver.authorities) {
		return EgressPolicyAuthority{}, errors.New("resolver sequence exhausted")
	}
	return resolver.authorities[index], nil
}

type mappedEgressPolicyResolver struct {
	authorities map[EgressPolicyRef]EgressPolicyAuthority
	refs        []EgressPolicyRef
}

func (resolver *mappedEgressPolicyResolver) ResolveEgressPolicy(
	_ context.Context,
	ref EgressPolicyRef,
) (EgressPolicyAuthority, error) {
	resolver.refs = append(resolver.refs, ref)
	authority, found := resolver.authorities[ref]
	if !found {
		return EgressPolicyAuthority{}, errors.New("policy missing")
	}
	return authority, nil
}

type egressAuthorityMutatingRepository struct {
	StateRepository
	replacement EgressPolicyAuthority
}

func (repository egressAuthorityMutatingRepository) ReplayGoalSubmission(
	ctx context.Context,
	request GoalSubmissionReplayRequest,
) (GoalRecord, bool, error) {
	reader, available := repository.StateRepository.(GoalSubmissionReplayReader)
	if !available {
		return GoalRecord{}, false, errors.New("test replay reader unavailable")
	}
	return reader.ReplayGoalSubmission(ctx, request)
}

func (repository egressAuthorityMutatingRepository) CreateGoal(
	ctx context.Context,
	state CreateGoalState,
) (GoalRecord, bool, error) {
	record, created, err := repository.StateRepository.CreateGoal(ctx, state)
	if err == nil && created && len(record.WorkItemAuthorities) != 0 {
		record.WorkItemAuthorities[0].EgressPolicy = repository.replacement
	}
	return record, created, err
}

func (repository *memoryRepository) ReplayGoalSubmission(
	_ context.Context,
	request GoalSubmissionReplayRequest,
) (GoalRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	key := request.RequestedBy.String() + "\x00" + request.ProjectRef.String() + "\x00" + request.RequestRef
	goalRef, found := repository.requests[key]
	if !found {
		return GoalRecord{}, false, nil
	}
	record := repository.records[goalRef]
	if record.RequestFingerprint != request.RequestFingerprint || record.RequestedBy != request.RequestedBy ||
		record.Goal.Project() != request.ProjectRef {
		return GoalRecord{}, false, &StateError{Code: StateConflict}
	}
	return cloneGoalRecord(record), true, nil
}

func (resolver *egressPolicyResolverStub) ResolveEgressPolicy(
	_ context.Context,
	ref EgressPolicyRef,
) (EgressPolicyAuthority, error) {
	resolver.refs = append(resolver.refs, ref)
	return resolver.authority, resolver.err
}

func TestSubmitPersistsExactResolvedEgressPolicyInWorkItemAuthority(t *testing.T) {
	clock := &mutableClock{now: time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	orchestrator, _ := newTestOrchestrator(t, repository, clock, &scriptedAgent{now: clock.Now})
	policy := testEgressPolicyAuthority(t, "egress-policy:web-search", `{"destinations":["example.org"]}`)
	resolver := &egressPolicyResolverStub{authority: policy}
	orchestrator.egressPolicies = resolver
	actor, project := testScope(t)

	result, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:egress-authority", Statement: "research exact public source", Confirm: true,
		Plan: &PlanSpec{
			Phases: []PhaseSpec{{
				Ref: "phase-instance:egress", Key: goal.DefaultPhaseKey().String(), TemplateRef: "phase-template:egress",
			}},
			WorkItems: []WorkItemSpec{{
				Key: "work", Objective: "research exact public source", Phase: goal.DefaultPhaseKey().String(),
				Role: goal.DefaultRoleKey().String(), OutputContract: goal.OutputContractEvidenceBundle,
				EgressPolicyRef: policy.PolicyRef.String(),
			}},
		},
	})
	if err != nil {
		t.Fatalf("submit egress policy: %v", err)
	}
	if len(resolver.refs) != 1 || resolver.refs[0] != policy.PolicyRef ||
		len(result.Record.WorkItemAuthorities) != 1 ||
		result.Record.WorkItemAuthorities[0].EgressPolicy != policy {
		t.Fatalf("resolved authority mismatch: refs=%v authorities=%+v", resolver.refs, result.Record.WorkItemAuthorities)
	}
}

func TestSubmitResolvesRepeatedEgressPolicyRefOnceAndReusesExactAuthority(t *testing.T) {
	clock := &mutableClock{now: time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	orchestrator, _ := newTestOrchestrator(t, repository, clock, &scriptedAgent{now: clock.Now})
	first := testEgressPolicyAuthority(t, "egress-policy:shared", `{"revision":1}`)
	divergent := testEgressPolicyAuthority(t, first.PolicyRef.String(), `{"revision":2}`)
	resolver := &sequentialEgressPolicyResolver{authorities: []EgressPolicyAuthority{first, divergent}}
	orchestrator.egressPolicies = resolver
	actor, project := testScope(t)

	result, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:egress-shared-once", Statement: "two items share one authority", Confirm: true,
		Plan: &PlanSpec{
			Phases: []PhaseSpec{{
				Ref: "phase-instance:egress-shared", Key: goal.DefaultPhaseKey().String(), TemplateRef: "phase-template:egress",
			}},
			WorkItems: []WorkItemSpec{
				{Key: "first", Objective: "first", Phase: goal.DefaultPhaseKey().String(), Role: goal.DefaultRoleKey().String(), OutputContract: goal.OutputContractEvidenceBundle, EgressPolicyRef: first.PolicyRef.String()},
				{Key: "second", Objective: "second", Phase: goal.DefaultPhaseKey().String(), Role: goal.DefaultRoleKey().String(), OutputContract: goal.OutputContractEvidenceBundle, EgressPolicyRef: first.PolicyRef.String()},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resolver.refs) != 1 || len(result.Record.WorkItemAuthorities) != 2 {
		t.Fatalf("resolver calls=%v authorities=%+v", resolver.refs, result.Record.WorkItemAuthorities)
	}
	for _, authority := range result.Record.WorkItemAuthorities {
		if authority.EgressPolicy != first {
			t.Fatalf("shared policy resolution diverged: %+v", result.Record.WorkItemAuthorities)
		}
	}
}

func TestSubmitEgressReplayUsesHistoricalAuthorityWithoutResolver(t *testing.T) {
	clock := &mutableClock{now: time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	orchestrator, _ := newTestOrchestrator(t, repository, clock, &scriptedAgent{now: clock.Now})
	historical := testEgressPolicyAuthority(t, "egress-policy:historical", `{"revision":1}`)
	request := SubmitRequest{
		RequestRef: "request:egress-historical-replay", Statement: "replay historical authority", Confirm: true,
		Plan: &PlanSpec{
			Phases: []PhaseSpec{{
				Ref: "phase-instance:egress-historical", Key: goal.DefaultPhaseKey().String(), TemplateRef: "phase-template:egress",
			}},
			WorkItems: []WorkItemSpec{{
				Key: "work", Objective: "replay historical authority", Phase: goal.DefaultPhaseKey().String(),
				Role: goal.DefaultRoleKey().String(), OutputContract: goal.OutputContractEvidenceBundle,
				EgressPolicyRef: historical.PolicyRef.String(),
			}},
		},
	}
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	orchestrator.egressPolicies = &egressPolicyResolverStub{authority: historical}
	created, err := orchestrator.Submit(context.Background(), access, request)
	if err != nil || !created.Created {
		t.Fatalf("initial submit=%+v err=%v", created, err)
	}

	rotated := &egressPolicyResolverStub{authority: testEgressPolicyAuthority(t, historical.PolicyRef.String(), `{"revision":2}`)}
	orchestrator.egressPolicies = rotated
	rotatedReplay, err := orchestrator.Submit(context.Background(), access, request)
	if err != nil || rotatedReplay.Created || len(rotated.refs) != 0 ||
		rotatedReplay.Record.WorkItemAuthorities[0].EgressPolicy != historical {
		t.Fatalf("rotated replay=%+v calls=%v err=%v", rotatedReplay, rotated.refs, err)
	}

	failing := &egressPolicyResolverStub{err: errors.New("catalog unavailable")}
	orchestrator.egressPolicies = failing
	failedReplay, err := orchestrator.Submit(context.Background(), access, request)
	if err != nil || failedReplay.Created || len(failing.refs) != 0 ||
		failedReplay.Record.WorkItemAuthorities[0].EgressPolicy != historical {
		t.Fatalf("failed resolver replay=%+v calls=%v err=%v", failedReplay, failing.refs, err)
	}

	orchestrator.egressPolicies = nil
	retiredReplay, err := orchestrator.Submit(context.Background(), access, request)
	if err != nil || retiredReplay.Created || retiredReplay.Record.WorkItemAuthorities[0].EgressPolicy != historical {
		t.Fatalf("retired policy replay=%+v err=%v", retiredReplay, err)
	}
}

func TestSubmitEgressReplayRejectsCrossWorkItemPolicySwap(t *testing.T) {
	clock := &mutableClock{now: time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	orchestrator, _ := newTestOrchestrator(t, repository, clock, &scriptedAgent{now: clock.Now})
	first := testEgressPolicyAuthority(t, "egress-policy:first-item", `{"item":1}`)
	second := testEgressPolicyAuthority(t, "egress-policy:second-item", `{"item":2}`)
	resolver := &mappedEgressPolicyResolver{authorities: map[EgressPolicyRef]EgressPolicyAuthority{
		first.PolicyRef: first, second.PolicyRef: second,
	}}
	orchestrator.egressPolicies = resolver
	request := SubmitRequest{
		RequestRef: "request:egress-cross-item", Statement: "keep policies bound to their items", Confirm: true,
		Plan: &PlanSpec{
			Phases: []PhaseSpec{{
				Ref: "phase-instance:egress-cross-item", Key: goal.DefaultPhaseKey().String(), TemplateRef: "phase-template:egress",
			}},
			WorkItems: []WorkItemSpec{
				{Key: "first", Objective: "first", Phase: goal.DefaultPhaseKey().String(), Role: goal.DefaultRoleKey().String(), OutputContract: goal.OutputContractEvidenceBundle, EgressPolicyRef: first.PolicyRef.String()},
				{Key: "second", Objective: "second", Phase: goal.DefaultPhaseKey().String(), Role: goal.DefaultRoleKey().String(), OutputContract: goal.OutputContractEvidenceBundle, EgressPolicyRef: second.PolicyRef.String()},
			},
		},
	}
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	created, err := orchestrator.Submit(context.Background(), access, request)
	if err != nil {
		t.Fatal(err)
	}
	repository.mu.Lock()
	record := repository.records[created.Record.Goal.Ref()]
	record.WorkItemAuthorities[0].EgressPolicy, record.WorkItemAuthorities[1].EgressPolicy =
		record.WorkItemAuthorities[1].EgressPolicy, record.WorkItemAuthorities[0].EgressPolicy
	repository.records[created.Record.Goal.Ref()] = record
	repository.mu.Unlock()
	orchestrator.egressPolicies = nil

	if _, err := orchestrator.Submit(context.Background(), access, request); !IsStateError(err, StateConflict) {
		t.Fatalf("cross-work-item policy swap replayed: %v", err)
	}
}

func TestSubmitRejectsStateRepositoryThatSubstitutesResolvedEgressAuthority(t *testing.T) {
	clock := &mutableClock{now: time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	orchestrator, _ := newTestOrchestrator(t, repository, clock, &scriptedAgent{now: clock.Now})
	requested := testEgressPolicyAuthority(t, "egress-policy:requested", `{"destinations":["example.org"]}`)
	replacement := testEgressPolicyAuthority(t, "egress-policy:replacement", `{"destinations":["other.example"]}`)
	orchestrator.egressPolicies = &egressPolicyResolverStub{authority: requested}
	orchestrator.state = egressAuthorityMutatingRepository{
		StateRepository: repository,
		replacement:     replacement,
	}
	actor, project := testScope(t)

	_, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:egress-authority-substitution", Statement: "research exact public source", Confirm: true,
		Plan: &PlanSpec{
			Phases: []PhaseSpec{{
				Ref: "phase-instance:egress-substitution", Key: goal.DefaultPhaseKey().String(), TemplateRef: "phase-template:egress",
			}},
			WorkItems: []WorkItemSpec{{
				Key: "work", Objective: "research exact public source", Phase: goal.DefaultPhaseKey().String(),
				Role: goal.DefaultRoleKey().String(), OutputContract: goal.OutputContractEvidenceBundle,
				EgressPolicyRef: requested.PolicyRef.String(),
			}},
		},
	})
	if !IsStateError(err, StateConflict) {
		t.Fatalf("substituted persisted egress authority accepted: %v", err)
	}
}

func TestDirectorPlanPersistsResolvedEgressPolicyOnlyForNewWorkItem(t *testing.T) {
	ctx := context.Background()
	system := newDirectorTestSystem(t)
	policy := testEgressPolicyAuthority(t, "egress-policy:director-search", `{"destinations":["example.org"]}`)
	system.orchestrator.egressPolicies = &egressPolicyResolverStub{authority: policy}
	lease := claimDirectorForTest(t, system, system.ownerAccess, "director-claim:egress-policy")
	source := system.goal
	result, err := system.orchestrator.ProposeDirectorPlan(ctx, system.ownerAccess, ProposeDirectorPlanRequest{
		RequestRef: "director-plan:egress-policy", GoalRef: source.Goal.Ref(),
		ExpectedGoalRevision: source.Goal.Revision(), ExpectedPlanGeneration: source.Goal.PlanGeneration(),
		LeaseToken: lease.Token, LeaseFence: lease.Fence, Reason: "add exact governed research",
		Plan: PlanSpec{
			Phases: []PhaseSpec{{
				Ref: "phase-instance:director-egress", Key: "phase:director-egress",
				TemplateRef: "phase-template:director-egress",
			}},
			WorkItems: []WorkItemSpec{{
				Key: "work:director-egress", Objective: "research exact public source",
				Phase: "phase:director-egress", Role: "role:researcher",
				OutputContract: goal.OutputContractEvidenceBundle, EgressPolicyRef: policy.PolicyRef.String(),
			}},
		},
	})
	if err != nil || !result.Created {
		t.Fatalf("director egress plan result=%+v err=%v", result, err)
	}
	record, err := system.orchestrator.GetGoal(ctx, system.ownerAccess, source.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if len(record.WorkItemAuthorities) != len(source.WorkItemAuthorities)+1 ||
		record.WorkItemAuthorities[len(record.WorkItemAuthorities)-1].EgressPolicy != policy {
		t.Fatalf("director egress authority mismatch: %+v", record.WorkItemAuthorities)
	}
	for _, authority := range record.WorkItemAuthorities[:len(source.WorkItemAuthorities)] {
		if authority.EgressPolicy != (EgressPolicyAuthority{}) {
			t.Fatalf("director rewrote prior authority: %+v", authority)
		}
	}
}

func TestResolvePlanEgressPoliciesFailsClosedWithoutExactBoundedResolution(t *testing.T) {
	requested, err := NewEgressPolicyRef("egress-policy:bounded")
	if err != nil {
		t.Fatal(err)
	}
	valid := testEgressPolicyAuthority(t, requested.String(), `{"destinations":[]}`)
	tests := map[string]struct {
		resolver EgressPolicyResolver
		ref      string
	}{
		"resolver missing": {ref: requested.String()},
		"requested ref invalid": {
			resolver: &egressPolicyResolverStub{authority: valid}, ref: " egress-policy:bounded",
		},
		"requested ref invalid UTF-8": {
			resolver: &egressPolicyResolverStub{authority: valid}, ref: string([]byte{0xff}),
		},
		"resolution error": {
			resolver: &egressPolicyResolverStub{err: errors.New("sensitive remote detail")}, ref: requested.String(),
		},
		"crossed ref": {
			resolver: &egressPolicyResolverStub{authority: testEgressPolicyAuthority(t, "egress-policy:other", `{"destinations":[]}`)},
			ref:      requested.String(),
		},
		"digest mismatch": {
			resolver: &egressPolicyResolverStub{authority: EgressPolicyAuthority{
				PolicyRef: requested, PayloadSHA256: strings.Repeat("a", 64), CanonicalPayload: valid.CanonicalPayload,
			}},
			ref: requested.String(),
		},
		"payload oversized": {
			resolver: &egressPolicyResolverStub{authority: EgressPolicyAuthority{
				PolicyRef: requested, CanonicalPayload: strings.Repeat("x", maxEgressPolicyCanonicalPayloadBytes+1),
				PayloadSHA256: egressPolicyPayloadSHA256(strings.Repeat("x", maxEgressPolicyCanonicalPayloadBytes+1)),
			}},
			ref: requested.String(),
		},
		"payload invalid UTF-8": {
			resolver: &egressPolicyResolverStub{authority: EgressPolicyAuthority{
				PolicyRef: requested, CanonicalPayload: string([]byte{0xff}),
				PayloadSHA256: egressPolicyPayloadSHA256(string([]byte{0xff})),
			}},
			ref: requested.String(),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			orchestrator := &Orchestrator{egressPolicies: test.resolver}
			resolved, resolveErr := orchestrator.resolvePlanEgressPolicies(context.Background(), &PlanSpec{
				WorkItems: []WorkItemSpec{{EgressPolicyRef: test.ref}},
			})
			if resolveErr == nil || resolved != nil {
				t.Fatalf("invalid egress resolution accepted: resolved=%+v err=%v", resolved, resolveErr)
			}
		})
	}
}

func TestPlanFingerprintCoversRequestedEgressPolicyWithoutChangingAbsentPlan(t *testing.T) {
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	base := SubmitRequest{RequestRef: "request:egress-fingerprint", Statement: "same", Confirm: true,
		Plan: &PlanSpec{WorkItems: []WorkItemSpec{{Key: "work"}}}}
	absent := submissionFingerprint(access, base)
	withPolicy := base
	withPolicy.Plan = clonePlanSpec(base.Plan)
	withPolicy.Plan.WorkItems[0].EgressPolicyRef = "egress-policy:first"
	first := submissionFingerprint(access, withPolicy)
	withPolicy.Plan.WorkItems[0].EgressPolicyRef = "egress-policy:second"
	second := submissionFingerprint(access, withPolicy)
	if absent == first || first == second || absent == second {
		t.Fatalf("egress policy refs were not isolated in plan fingerprint")
	}
}

func TestAbsentEgressPolicyNeedsNoResolver(t *testing.T) {
	orchestrator := &Orchestrator{}
	resolved, err := orchestrator.resolvePlanEgressPolicies(context.Background(), &PlanSpec{
		WorkItems: []WorkItemSpec{{Key: "work"}},
	})
	if err != nil || len(resolved) != 1 || resolved[0] != (EgressPolicyAuthority{}) {
		t.Fatalf("absent egress policy resolution=%+v err=%v", resolved, err)
	}
}

func TestAuthorLaunchBindsDurableEgressAndRejectsMutationBeforeProvider(t *testing.T) {
	tests := map[string]struct {
		mutate      func(*GoalRecord, EgressPolicyAuthority)
		wantLaunch  bool
		wantErrCode string
	}{
		"exact durable authority": {wantLaunch: true},
		"valid authority substituted": {
			mutate: func(record *GoalRecord, _ EgressPolicyAuthority) {
				record.WorkItemAuthorities[0].EgressPolicy = testEgressPolicyAuthority(
					t, "egress-policy:substituted", `{"destinations":["other.example"]}`,
				)
			},
			wantErrCode: "application.effect_target_mismatch",
		},
		"payload tampered": {
			mutate: func(record *GoalRecord, _ EgressPolicyAuthority) {
				record.WorkItemAuthorities[0].EgressPolicy.CanonicalPayload += " "
			},
			wantErrCode: "application.agent_launch_egress_authority_invalid",
		},
	}
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			clock := &mutableClock{now: time.Date(2026, 8, 5, 13, 0, 0, 0, time.UTC)}
			repository := newMemoryRepository()
			agent := &scriptedAgent{now: clock.Now}
			orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
			policy := testEgressPolicyAuthority(t, "egress-policy:dispatch", `{"destinations":["example.org"]}`)
			resolver := &egressPolicyResolverStub{authority: policy}
			orchestrator.egressPolicies = resolver
			actor, project := testScope(t)
			submitted, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
				RequestRef: "request:egress-dispatch:" + strings.ReplaceAll(name, " ", "-"),
				Statement:  "dispatch with exact durable egress authority", Confirm: true,
				Plan: &PlanSpec{
					Phases: []PhaseSpec{{
						Ref: "phase-instance:egress-dispatch", Key: goal.DefaultPhaseKey().String(),
						TemplateRef: "phase-template:egress-dispatch",
					}},
					WorkItems: []WorkItemSpec{{
						Key: "work", Objective: "use governed network", Phase: goal.DefaultPhaseKey().String(),
						Role: goal.DefaultRoleKey().String(), OutputContract: goal.OutputContractEvidenceBundle,
						EgressPolicyRef: policy.PolicyRef.String(),
					}},
				},
			})
			if err != nil {
				t.Fatalf("submit egress launch: %v", err)
			}
			if testCase.mutate != nil {
				repository.mu.Lock()
				record := repository.records[submitted.Record.Goal.Ref()]
				testCase.mutate(&record, policy)
				repository.records[submitted.Record.Goal.Ref()] = record
				repository.mu.Unlock()
			}
			unavailable := &egressPolicyResolverStub{err: errors.New("catalog must not be read during dispatch")}
			orchestrator.egressPolicies = unavailable
			processed, processErr := orchestrator.ProcessNext(context.Background(), "worker:egress-dispatch")
			if !processed.Processed || processed.Action != ActionLaunchAgent || len(unavailable.refs) != 0 {
				t.Fatalf("dispatch result=%+v catalog refs=%v err=%v", processed, unavailable.refs, processErr)
			}
			if testCase.wantLaunch {
				if processErr != nil || agent.launches != 1 || len(agent.launchRequests) != 1 {
					t.Fatalf("launches=%d requests=%d err=%v", agent.launches, len(agent.launchRequests), processErr)
				}
				want := ports.AgentLaunchEgressAuthority{PolicyRef: policy.PolicyRef.String(),
					PayloadSHA256: policy.PayloadSHA256, CanonicalPayload: policy.CanonicalPayload}
				if agent.launchRequests[0].EgressAuthority != want {
					t.Fatalf("launch egress=%+v want=%+v", agent.launchRequests[0].EgressAuthority, want)
				}
				return
			}
			if processErr == nil || processErr.Error() != testCase.wantErrCode || agent.launches != 0 {
				t.Fatalf("mutation crossed provider: launches=%d err=%v", agent.launches, processErr)
			}
		})
	}
}

func testEgressPolicyAuthority(t *testing.T, rawRef, payload string) EgressPolicyAuthority {
	t.Helper()
	ref, err := NewEgressPolicyRef(rawRef)
	if err != nil {
		t.Fatal(err)
	}
	return EgressPolicyAuthority{
		PolicyRef: ref, PayloadSHA256: egressPolicyPayloadSHA256(payload), CanonicalPayload: payload,
	}
}
