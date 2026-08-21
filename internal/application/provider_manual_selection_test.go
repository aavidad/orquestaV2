package application

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestProviderManualSelectionSelectsOnlyExactAuthorizedPair(t *testing.T) {
	fixture := newProviderManualSelectionFixture()
	decision, err := SelectProviderModelForRole(fixture.catalog(), fixture.request)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Selected || decision.Reason != ProviderManualSelectionSelected ||
		decision.RouteReason != ProviderRouteSelected || decision.SelectedCandidate != fixture.request.Candidate ||
		decision.Context != fixture.request.Context || decision.RequestFingerprint == "" ||
		decision.AuthorizationReceiptRef != fixture.request.Authority.Receipt.Ref() ||
		decision.AuthorizationRole != identity.RoleProjectOwner || decision.AuthorizationRevision != 7 ||
		decision.DirectorLeaseFence != fixture.request.Authority.DirectorLease.Fence ||
		decision.ObservedUsage.Resources.Tokens != 11 {
		t.Fatalf("manual selection=%+v", decision)
	}
}
func TestProviderManualSelectionReplayIsExactConcurrentAndAuthorityBound(t *testing.T) {
	fixture := newProviderManualSelectionFixture()
	catalog := providerManualMust(ObserveProviderCatalog(context.Background(), time.Now(), nil))
	first, err := SelectProviderModelForRole(catalog, fixture.request)
	if err != nil {
		t.Fatal(err)
	}
	persisted, replay := first, fixture.request
	persistedCatalog := catalog
	persistedCatalog.observedAt = catalog.observedAt.Round(0).UTC()
	persisted.CatalogObservedAt, persisted.DecidedAt = first.CatalogObservedAt.Round(0).UTC(), first.DecidedAt.Round(0).UTC()
	replay.DecidedAt, replay.Authority.DirectorLease.LeaseUntil = replay.DecidedAt.Round(0).UTC(), replay.Authority.DirectorLease.LeaseUntil.Round(0).UTC()
	replay.PriorDecision = &persisted
	results := make(chan ProviderManualSelectionDecision, 16)
	var group sync.WaitGroup
	for range 16 {
		group.Add(1)
		go func() {
			defer group.Done()
			decision, replayErr := SelectProviderModelForRole(persistedCatalog, replay)
			if replayErr != nil {
				t.Errorf("concurrent replay err=%v", replayErr)
			}
			results <- decision
		}()
	}
	group.Wait()
	close(results)
	for decision := range results {
		if decision != persisted {
			t.Fatalf("concurrent replay=%+v want=%+v", decision, persisted)
		}
	}
	tests := []struct {
		name   string
		mutate func(*ProviderManualSelectionRequest)
	}{
		{"candidate", func(request *ProviderManualSelectionRequest) { request.Candidate.ModelRef = "model:other" }},
		{"principal kind", func(request *ProviderManualSelectionRequest) {
			request.Authority = providerManualAuthority(*request, providerManualAuthorityOptions{principalKind: identity.PrincipalKindService})
		}},
		{"principal method", func(request *ProviderManualSelectionRequest) {
			request.Authority = providerManualAuthority(*request, providerManualAuthorityOptions{principalMethod: "other"})
		}},
		{"authorization reason", func(request *ProviderManualSelectionRequest) {
			request.Authority = providerManualAuthority(*request, providerManualAuthorityOptions{reasonCode: "other"})
		}},
		{"expected role", func(request *ProviderManualSelectionRequest) {
			request.Authority.ExpectedRole = identity.RoleOperator
		}},
		{"expected revision", func(request *ProviderManualSelectionRequest) {
			request.Authority.ExpectedMembershipRevision++
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			divergent := replay
			test.mutate(&divergent)
			if providerManualSelectionFingerprint(persistedCatalog, divergent, canonicalProviderManualSelectionTimes(persistedCatalog, divergent)) == first.RequestFingerprint {
				t.Fatal("authority fact absent from fingerprint")
			}
			decision, replayErr := SelectProviderModelForRole(persistedCatalog, divergent)
			if !errors.Is(replayErr, ErrProviderManualSelectionReplayDivergent) ||
				decision != (ProviderManualSelectionDecision{}) {
				t.Fatalf("divergent replay=%+v err=%v", decision, replayErr)
			}
		})
	}
}
func TestProviderManualSelectionRejectsCatalogAndEligibilityFailures(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*providerManualSelectionFixture)
		noCatalog  bool
		wantReason ProviderManualSelectionReason
		wantRoute  ProviderRouteReason
	}{
		{name: "catalog absent", noCatalog: true, wantReason: ProviderManualSelectionCatalogAbsent},
		{name: "catalog observed after decision", mutate: func(f *providerManualSelectionFixture) {
			f.request.DecidedAt = f.now.Add(-time.Second)
		}, wantReason: ProviderManualSelectionCatalogContextDivergent},
		{name: "provider absent", mutate: func(f *providerManualSelectionFixture) {
			f.request.Candidate.ProviderRef = "provider:absent"
		}, wantReason: ProviderManualSelectionProviderRejected, wantRoute: ProviderRouteProviderAbsent},
		{name: "provider stale", mutate: func(f *providerManualSelectionFixture) {
			f.target.observation.ExpiresAt = f.now
		}, wantReason: ProviderManualSelectionProviderRejected, wantRoute: ProviderRouteProviderStale},
		{name: "provider unknown", mutate: func(f *providerManualSelectionFixture) {
			f.target.observation.Availability = ports.ProviderAvailabilityUnknown
		}, wantReason: ProviderManualSelectionProviderRejected, wantRoute: ProviderRouteProviderUnknown},
		{name: "provider unavailable", mutate: func(f *providerManualSelectionFixture) {
			f.target.observation.Availability = ports.ProviderAvailabilityUnavailable
		}, wantReason: ProviderManualSelectionProviderRejected, wantRoute: ProviderRouteProviderUnavailable},
		{name: "quota unknown", mutate: func(f *providerManualSelectionFixture) {
			f.target.observation.Quota = ports.ProviderQuotaUnknown
		}, wantReason: ProviderManualSelectionProviderRejected, wantRoute: ProviderRouteQuotaUnknown},
		{name: "quota exhausted", mutate: func(f *providerManualSelectionFixture) {
			f.target.observation.Quota = ports.ProviderQuotaExhausted
		}, wantReason: ProviderManualSelectionProviderRejected, wantRoute: ProviderRouteQuotaExhausted},
		{name: "model absent", mutate: func(f *providerManualSelectionFixture) {
			f.request.Candidate.ModelRef = "model:absent"
		}, wantReason: ProviderManualSelectionProviderRejected, wantRoute: ProviderRouteModelAbsent},
		{name: "capability missing", mutate: func(f *providerManualSelectionFixture) {
			f.request.RequiredCapabilityRefs = []string{"capability:missing"}
		}, wantReason: ProviderManualSelectionProviderRejected, wantRoute: ProviderRouteCapabilityMissing},
		{name: "effort unsupported", mutate: func(f *providerManualSelectionFixture) {
			f.request.ReasoningEffort = governance.ReasoningEffortHigh
		}, wantReason: ProviderManualSelectionProviderRejected, wantRoute: ProviderRouteEffortUnsupported},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newProviderManualSelectionFixture()
			if test.mutate != nil {
				test.mutate(&fixture)
			}
			catalog := ProviderCatalog{}
			if !test.noCatalog {
				catalog = fixture.catalog()
			}
			decision, err := SelectProviderModelForRole(catalog, fixture.request)
			if err != nil {
				t.Fatal(err)
			}
			assertProviderManualSelectionRejected(t, decision, test.wantReason, test.wantRoute)
		})
	}
}
func TestProviderManualSelectionRejectsDivergentRBACAndLease(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*providerManualSelectionFixture)
		want   ProviderManualSelectionReason
	}{
		{"missing authority", func(f *providerManualSelectionFixture) { f.request.Authority = ProviderManualSelectionAuthority{} }, ProviderManualSelectionAuthorityMissing},
		{"denied", func(f *providerManualSelectionFixture) {
			f.request.Authority = providerManualAuthority(f.request, providerManualAuthorityOptions{outcome: identity.AuthorizationDenied})
		}, ProviderManualSelectionAuthorityDenied},
		{"actor", func(f *providerManualSelectionFixture) {
			f.request.Authority = providerManualAuthority(f.request, providerManualAuthorityOptions{actorRef: "actor:other"})
		}, ProviderManualSelectionAuthorityContextDivergent},
		{"project", func(f *providerManualSelectionFixture) {
			f.request.Authority = providerManualAuthority(f.request, providerManualAuthorityOptions{projectRef: "project:other"})
		}, ProviderManualSelectionAuthorityContextDivergent},
		{"Goal", func(f *providerManualSelectionFixture) {
			f.request.Authority = providerManualAuthority(f.request, providerManualAuthorityOptions{resourceRef: "goal:other"})
		}, ProviderManualSelectionAuthorityContextDivergent},
		{"permission", func(f *providerManualSelectionFixture) {
			f.request.Authority = providerManualAuthority(f.request, providerManualAuthorityOptions{permission: identity.PermissionGoalsGet})
		}, ProviderManualSelectionAuthorityContextDivergent},
		{"receipt after selection", func(f *providerManualSelectionFixture) {
			f.request.Authority = providerManualAuthority(f.request, providerManualAuthorityOptions{receiptAt: f.now.Add(time.Second)})
		}, ProviderManualSelectionAuthorityContextDivergent},
		{"expected role", func(f *providerManualSelectionFixture) { f.request.Authority.ExpectedRole = identity.RoleOperator }, ProviderManualSelectionAuthorityRoleDivergent},
		{"expected revision", func(f *providerManualSelectionFixture) { f.request.Authority.ExpectedMembershipRevision++ }, ProviderManualSelectionAuthorityRevisionDivergent},
		{"membership role", func(f *providerManualSelectionFixture) {
			f.request.Authority = providerManualAuthority(f.request, providerManualAuthorityOptions{membershipRole: identity.RoleOperator})
		}, ProviderManualSelectionAuthorityRoleDivergent},
		{"membership revision", func(f *providerManualSelectionFixture) {
			f.request.Authority = providerManualAuthority(f.request, providerManualAuthorityOptions{membershipRevision: 8})
		}, ProviderManualSelectionAuthorityRevisionDivergent},
		{"membership revoked", func(f *providerManualSelectionFixture) {
			f.request.Authority = providerManualAuthority(f.request, providerManualAuthorityOptions{membershipStatus: identity.MembershipRevoked})
		}, ProviderManualSelectionAuthorityInactive},
		{"membership absent", func(f *providerManualSelectionFixture) { f.request.Authority.CurrentMembership = identity.Membership{} }, ProviderManualSelectionAuthorityCurrentnessUnverified},
		{"execution membership unverifiable", func(f *providerManualSelectionFixture) {
			f.request.Authority = providerManualAuthority(f.request, providerManualAuthorityOptions{
				principalKind: identity.PrincipalKindService, role: identity.RoleExecutionService, revision: 3, omitMembership: true,
			})
		}, ProviderManualSelectionAuthorityCurrentnessUnverified},
		{"membership granted after authorization", func(f *providerManualSelectionFixture) {
			f.request.Authority = providerManualAuthority(f.request, providerManualAuthorityOptions{membershipGrantedAt: f.now.Add(-time.Minute)})
		}, ProviderManualSelectionAuthorityCurrentnessUnverified},
		{"lease token", func(f *providerManualSelectionFixture) { f.request.Authority.DirectorLease.Token = "" }, ProviderManualSelectionLeaseMissing},
		{"lease fence", func(f *providerManualSelectionFixture) { f.request.Authority.DirectorLease.Fence = 0 }, ProviderManualSelectionLeaseMissing},
		{"lease Goal", func(f *providerManualSelectionFixture) {
			f.request.Authority.DirectorLease.GoalRef = providerManualMust(goal.NewGoalRef("goal:other"))
		}, ProviderManualSelectionLeaseContextDivergent},
		{"lease principal", func(f *providerManualSelectionFixture) {
			f.request.Authority.DirectorLease.PrincipalRef = providerManualMust(identity.NewPrincipalRef("principal:other"))
		}, ProviderManualSelectionLeaseContextDivergent},
		{"lease expired", func(f *providerManualSelectionFixture) { f.request.Authority.DirectorLease.LeaseUntil = f.now }, ProviderManualSelectionLeaseExpired},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newProviderManualSelectionFixture()
			test.mutate(&fixture)
			decision, err := SelectProviderModelForRole(fixture.catalog(), fixture.request)
			if err != nil {
				t.Fatal(err)
			}
			assertProviderManualSelectionRejected(t, decision, test.want, "")
		})
	}
}
func TestProviderManualSelectionRejectsInvalidContext(t *testing.T) {
	for name, mutate := range map[string]func(*ProviderManualSelectionRequest){
		"request ref": func(request *ProviderManualSelectionRequest) { request.RequestRef = "" },
		"generation":  func(request *ProviderManualSelectionRequest) { request.Context.PlanGeneration = 0 },
		"Goal":        func(request *ProviderManualSelectionRequest) { request.Context.GoalRevision = 0 },
		"WorkItem":    func(request *ProviderManualSelectionRequest) { request.Context.WorkItemRef = goal.WorkItemRef{} },
		"role":        func(request *ProviderManualSelectionRequest) { request.Context.Role = goal.RoleKey{} },
		"candidate":   func(request *ProviderManualSelectionRequest) { request.Candidate.ModelRef = "" },
		"effort":      func(request *ProviderManualSelectionRequest) { request.ReasoningEffort = "auto" },
	} {
		t.Run(name, func(t *testing.T) {
			fixture := newProviderManualSelectionFixture()
			mutate(&fixture.request)
			if _, err := SelectProviderModelForRole(fixture.catalog(), fixture.request); !errors.Is(err, ErrProviderManualSelectionInvalid) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

type providerManualSelectionFixture struct {
	now     time.Time
	target  providerManualCatalogSource
	request ProviderManualSelectionRequest
}

func newProviderManualSelectionFixture() providerManualSelectionFixture {
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	request := ProviderManualSelectionRequest{
		RequestRef: "provider-manual-selection:agt-11",
		Context: ProviderManualSelectionContext{
			ActorRef: providerManualMust(goal.NewActorRef("actor:operator")), ProjectRef: providerManualMust(goal.NewProjectRef("project:orquesta")),
			GoalRef: providerManualMust(goal.NewGoalRef("goal:agt-11")), GoalRevision: 9, PlanGeneration: 4,
			WorkItemRef: providerManualMust(goal.NewWorkItemRef("work-item:agt-11")), WorkItemRevision: 6,
			AppSpecGeneration: 2, SpecHash: strings.Repeat("a", 64), Role: providerManualMust(goal.NewRoleKey("role:worker")),
		},
		Candidate:              ProviderRouteCandidate{ProviderRef: "provider:target", ModelRef: "model:target"},
		RequiredCapabilityRefs: []string{"capability:edit"}, ReasoningEffort: governance.ReasoningEffortMedium, DecidedAt: now,
	}
	request.Authority = providerManualAuthority(request, providerManualAuthorityOptions{})
	return providerManualSelectionFixture{now: now, target: providerManualSource(now, "provider:target", "model:target", 11), request: request}
}
func (fixture providerManualSelectionFixture) catalog() ProviderCatalog {
	return providerManualMust(ObserveProviderCatalog(context.Background(), fixture.now, []ProviderCatalogSource{
		providerManualSource(fixture.now, "provider:alternate", "model:alternate", 3), fixture.target,
	}))
}

type providerManualCatalogSource struct {
	observation ports.ProviderCatalogObservation
}

func (source providerManualCatalogSource) ProviderRef() string { return source.observation.ProviderRef }
func (source providerManualCatalogSource) ObserveProviderCatalog(context.Context) (ports.ProviderCatalogObservation, error) {
	return source.observation, nil
}
func providerManualSource(now time.Time, providerRef, modelRef string, tokens int64) providerManualCatalogSource {
	return providerManualCatalogSource{observation: ports.ProviderCatalogObservation{
		ProviderRef: providerRef, Availability: ports.ProviderAvailabilityAvailable, Quota: ports.ProviderQuotaAvailable,
		Usage:      governance.ResourceUsage{Resources: governance.ResourceVector{Tokens: tokens}, Known: governance.ResourceTokens, Quality: governance.UsageQualityMeasured},
		ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute),
		Models: []ports.ProviderModel{{ProviderRef: providerRef, ModelRef: modelRef, CapabilityRefs: []string{"capability:edit"}, ReasoningEfforts: []governance.ReasoningEffort{governance.ReasoningEffortMedium}}},
	}}
}

type providerManualAuthorityOptions struct {
	actorRef, projectRef, resourceRef, principalMethod, reasonCode string
	principalKind                                                  identity.PrincipalKind
	permission                                                     identity.Permission
	outcome                                                        identity.AuthorizationOutcome
	role, membershipRole                                           identity.Role
	revision, membershipRevision                                   identity.MembershipRevision
	membershipStatus                                               identity.MembershipStatus
	membershipGrantedAt, receiptAt                                 time.Time
	omitMembership                                                 bool
}

func providerManualAuthority(request ProviderManualSelectionRequest, options providerManualAuthorityOptions) ProviderManualSelectionAuthority {
	actorRef, projectRef, resourceRef := request.Context.ActorRef, request.Context.ProjectRef, request.Context.GoalRef.String()
	if options.actorRef != "" {
		actorRef = providerManualMust(goal.NewActorRef(options.actorRef))
	}
	if options.projectRef != "" {
		projectRef = providerManualMust(goal.NewProjectRef(options.projectRef))
	}
	if options.resourceRef != "" {
		resourceRef = options.resourceRef
	}
	if options.principalKind == "" {
		options.principalKind = identity.PrincipalKindHuman
	}
	if options.principalMethod == "" {
		options.principalMethod = "test"
	}
	if options.permission == "" {
		options.permission = identity.PermissionGoalsDirect
	}
	if options.outcome == "" {
		options.outcome = identity.AuthorizationAllowed
	}
	if options.role == "" {
		options.role = identity.RoleProjectOwner
	}
	if options.revision == 0 && options.role != identity.RolePlatformAdmin {
		options.revision = 7
	}
	if options.reasonCode == "" {
		options.reasonCode = "test"
	}
	principalRef := providerManualMust(identity.NewPrincipalRef("principal:operator"))
	principal := providerManualMust(identity.NewPrincipal(principalRef, actorRef, options.principalKind, options.principalMethod))
	authorizationRequest := providerManualMust(identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: "authorization-request:agt-11", Principal: principal, ProjectRef: projectRef,
		Permission: options.permission, ResourceRef: resourceRef, RequestedAt: request.DecidedAt.Add(-3 * time.Minute),
	}))
	authorizationDecision := providerManualMust(identity.NewAuthorizationDecision(identity.AuthorizationDecisionInput{
		Request: authorizationRequest, Outcome: options.outcome, Role: options.role,
		MembershipRevision: options.revision, ReasonCode: options.reasonCode, DecidedAt: request.DecidedAt.Add(-2 * time.Minute),
	}))
	if options.receiptAt.IsZero() {
		options.receiptAt = request.DecidedAt.Add(-time.Minute)
	}
	authority := ProviderManualSelectionAuthority{
		Receipt: providerManualMust(identity.NewAuthorizationReceipt(identity.AuthorizationReceiptInput{
			Ref: "authorization-receipt:agt-11", Decision: authorizationDecision, RecordedAt: options.receiptAt,
		})),
		ExpectedRole: options.role, ExpectedMembershipRevision: options.revision,
		DirectorLease: DirectorLeaseRecord{GoalRef: request.Context.GoalRef, PrincipalRef: principalRef,
			Token: "director-lease:agt-11", Fence: 11, LeaseUntil: request.DecidedAt.Add(time.Minute)},
	}
	if options.omitMembership || options.role == identity.RolePlatformAdmin || options.role == identity.RoleExecutionService {
		return authority
	}
	if options.membershipRole == "" {
		options.membershipRole = options.role
	}
	if options.membershipRevision == 0 {
		options.membershipRevision = options.revision
	}
	if options.membershipStatus == "" {
		options.membershipStatus = identity.MembershipActive
	}
	if options.membershipGrantedAt.IsZero() {
		options.membershipGrantedAt = request.DecidedAt.Add(-time.Hour)
	}
	membership := identity.MembershipInput{
		PrincipalRef: principalRef, ProjectRef: projectRef, Role: options.membershipRole,
		Revision: options.membershipRevision, Status: options.membershipStatus,
		GrantedBy: providerManualMust(identity.NewPrincipalRef("principal:grantor")), GrantedAt: options.membershipGrantedAt,
	}
	if options.membershipStatus == identity.MembershipRevoked {
		membership.RevokedBy = providerManualMust(identity.NewPrincipalRef("principal:revoker"))
		membership.RevokedAt = request.DecidedAt.Add(-time.Second)
	}
	authority.CurrentMembership = providerManualMust(identity.NewMembership(membership))
	return authority
}

func assertProviderManualSelectionRejected(t *testing.T, decision ProviderManualSelectionDecision, wantReason ProviderManualSelectionReason, wantRoute ProviderRouteReason) {
	t.Helper()
	if decision.Selected || decision.Reason != wantReason || decision.RouteReason != wantRoute ||
		decision.SelectedCandidate != (ProviderRouteCandidate{}) || decision.AuthorizationReceiptRef != "" ||
		decision.AuthorizationRole != "" || decision.AuthorizationRevision != 0 || decision.DirectorLeaseFence != 0 {
		t.Fatalf("rejected selection=%+v want_reason=%s want_route=%s", decision, wantReason, wantRoute)
	}
}

func providerManualMust[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}
