package application

import (
	"errors"
	"strconv"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
)

var (
	ErrProviderManualSelectionInvalid         = errors.New("application.provider_manual_selection_invalid")
	ErrProviderManualSelectionReplayDivergent = errors.New("application.provider_manual_selection_replay_divergent")
)

// ProviderManualSelectionContext binds one query-only decision to the Goal
// snapshot and execution role supplied by its application caller. The caller
// remains responsible for loading the current Goal before invoking the pure
// selector; this type is not another lifecycle authority.
type ProviderManualSelectionContext struct {
	ActorRef          goal.ActorRef
	ProjectRef        goal.ProjectRef
	GoalRef           goal.GoalRef
	GoalRevision      goal.Revision
	PlanGeneration    goal.PlanGeneration
	WorkItemRef       goal.WorkItemRef
	WorkItemRevision  goal.Revision
	AppSpecGeneration goal.AppSpecGeneration
	SpecHash          string
	Role              goal.RoleKey
}

// GoalDirectAuthority carries existing RBAC and Director lease facts. Receipt
// must come from AccessRepository.Authorize. CurrentMembership must be the
// current AccessRepository.Membership observation for project-scoped roles;
// platform_admin is the existing membership-free exception.
type GoalDirectAuthority struct {
	Receipt                    identity.AuthorizationReceipt
	CurrentMembership          identity.Membership
	ExpectedRole               identity.Role
	ExpectedMembershipRevision identity.MembershipRevision
	DirectorLease              DirectorLeaseRecord
}

type ProviderManualSelectionAuthority = GoalDirectAuthority

type ProviderManualSelectionRequest struct {
	RequestRef             string
	Context                ProviderManualSelectionContext
	Candidate              ProviderRouteCandidate
	RequiredCapabilityRefs []string
	ReasoningEffort        governance.ReasoningEffort
	DecidedAt              time.Time
	Authority              ProviderManualSelectionAuthority
	PriorDecision          *ProviderManualSelectionDecision
}

type ProviderManualSelectionReason string

const (
	ProviderManualSelectionSelected                       ProviderManualSelectionReason = "selected"
	ProviderManualSelectionCatalogAbsent                  ProviderManualSelectionReason = "catalog_absent"
	ProviderManualSelectionCatalogContextDivergent        ProviderManualSelectionReason = "catalog_context_divergent"
	ProviderManualSelectionAuthorityMissing               ProviderManualSelectionReason = "authority_missing"
	ProviderManualSelectionAuthorityDenied                ProviderManualSelectionReason = "authority_denied"
	ProviderManualSelectionAuthorityContextDivergent      ProviderManualSelectionReason = "authority_context_divergent"
	ProviderManualSelectionAuthorityRoleDivergent         ProviderManualSelectionReason = "authority_role_divergent"
	ProviderManualSelectionAuthorityRevisionDivergent     ProviderManualSelectionReason = "authority_revision_divergent"
	ProviderManualSelectionAuthorityInactive              ProviderManualSelectionReason = "authority_inactive"
	ProviderManualSelectionAuthorityCurrentnessUnverified ProviderManualSelectionReason = "authority_currentness_unverified"
	ProviderManualSelectionLeaseMissing                   ProviderManualSelectionReason = "lease_missing"
	ProviderManualSelectionLeaseContextDivergent          ProviderManualSelectionReason = "lease_context_divergent"
	ProviderManualSelectionLeaseExpired                   ProviderManualSelectionReason = "lease_expired"
	ProviderManualSelectionProviderRejected               ProviderManualSelectionReason = "provider_rejected"
	ProviderManualSelectionNotExact                       ProviderManualSelectionReason = "selection_not_exact"
)

type ProviderManualSelectionDecision struct {
	RequestRef              string
	RequestFingerprint      string
	Selected                bool
	Reason                  ProviderManualSelectionReason
	RouteReason             ProviderRouteReason
	Context                 ProviderManualSelectionContext
	RequestedCandidate      ProviderRouteCandidate
	SelectedCandidate       ProviderRouteCandidate
	CatalogObservedAt       time.Time
	DecidedAt               time.Time
	AuthorizationReceiptRef string
	AuthorizationRole       identity.Role
	AuthorizationRevision   identity.MembershipRevision
	DirectorLeaseFence      uint64
	ObservedUsage           governance.ResourceUsage
}

type providerManualSelectionTimes struct {
	decidedAt, catalogObservedAt, receiptRecordedAt, authorizationRequestedAt    time.Time
	authorizationDecidedAt, membershipGrantedAt, membershipRevokedAt, leaseUntil time.Time
	providerObservedAt, providerExpiresAt                                        time.Time
}

func canonicalProviderManualSelectionTimes(catalog ProviderCatalog, request ProviderManualSelectionRequest) providerManualSelectionTimes {
	authorization := request.Authority.Receipt.Decision()
	membership := request.Authority.CurrentMembership
	observation, _ := catalog.Provider(request.Candidate.ProviderRef)
	canonical := func(value time.Time) time.Time { return value.Round(0).UTC() }
	return providerManualSelectionTimes{
		decidedAt: canonical(request.DecidedAt), catalogObservedAt: canonical(catalog.ObservedAt()),
		receiptRecordedAt:        canonical(request.Authority.Receipt.RecordedAt()),
		authorizationRequestedAt: canonical(authorization.Request().RequestedAt()), authorizationDecidedAt: canonical(authorization.DecidedAt()),
		membershipGrantedAt: canonical(membership.GrantedAt()), membershipRevokedAt: canonical(membership.RevokedAt()),
		leaseUntil: canonical(request.Authority.DirectorLease.LeaseUntil), providerObservedAt: canonical(observation.ObservedAt),
		providerExpiresAt: canonical(observation.ExpiresAt),
	}
}

// SelectProviderModelForRole validates one exact, already observed provider
// and model. It does not refresh, rank, fall back, persist, launch, reserve
// quota, or mutate Goal lifecycle state.
func SelectProviderModelForRole(
	catalog ProviderCatalog,
	request ProviderManualSelectionRequest,
) (ProviderManualSelectionDecision, error) {
	times := canonicalProviderManualSelectionTimes(catalog, request)
	decision := ProviderManualSelectionDecision{
		RequestRef:         request.RequestRef,
		Reason:             ProviderManualSelectionProviderRejected,
		Context:            request.Context,
		RequestedCandidate: request.Candidate,
		CatalogObservedAt:  times.catalogObservedAt,
		DecidedAt:          times.decidedAt,
	}
	if err := validateProviderManualSelectionRequest(request, times.decidedAt); err != nil {
		return decision, err
	}
	decision.RequestFingerprint = providerManualSelectionFingerprint(catalog, request, times)
	if times.catalogObservedAt.IsZero() {
		decision.Reason = ProviderManualSelectionCatalogAbsent
		return finalizeProviderManualSelectionReplay(decision, request.PriorDecision)
	}
	if times.catalogObservedAt.After(times.decidedAt) {
		decision.Reason = ProviderManualSelectionCatalogContextDivergent
		return finalizeProviderManualSelectionReplay(decision, request.PriorDecision)
	}
	if reason := providerManualSelectionAuthorityReason(request, times); reason != "" {
		decision.Reason = reason
		return finalizeProviderManualSelectionReplay(decision, request.PriorDecision)
	}

	if _, found := catalog.Provider(request.Candidate.ProviderRef); found &&
		!times.decidedAt.Before(times.providerExpiresAt) {
		decision.Reason = ProviderManualSelectionProviderRejected
		decision.RouteReason = ProviderRouteProviderStale
		return finalizeProviderManualSelectionReplay(decision, request.PriorDecision)
	}
	routeRequest := ProviderRouteRequest{
		Candidates:             []ProviderRouteCandidate{request.Candidate},
		RequiredCapabilityRefs: append([]string(nil), request.RequiredCapabilityRefs...),
		ReasoningEffort:        request.ReasoningEffort,
		AllowFallback:          false,
	}
	route, err := RouteProviderModel(catalog, routeRequest)
	if err != nil {
		return decision, err
	}
	if !route.Selected {
		decision.Reason = ProviderManualSelectionProviderRejected
		decision.RouteReason = route.Reason
		if len(route.Rejections) == 1 {
			decision.RouteReason = route.Rejections[0].Reason
		}
		return finalizeProviderManualSelectionReplay(decision, request.PriorDecision)
	}
	if route.UsedFallback || route.Reason != ProviderRouteSelected ||
		route.Candidate != request.Candidate {
		decision.Reason = ProviderManualSelectionNotExact
		return finalizeProviderManualSelectionReplay(decision, request.PriorDecision)
	}

	authorization := request.Authority.Receipt.Decision()
	decision.Selected = true
	decision.Reason = ProviderManualSelectionSelected
	decision.RouteReason = ProviderRouteSelected
	decision.SelectedCandidate = route.Candidate
	decision.AuthorizationReceiptRef = request.Authority.Receipt.Ref()
	decision.AuthorizationRole = authorization.Role()
	decision.AuthorizationRevision = authorization.MembershipRevision()
	decision.DirectorLeaseFence = request.Authority.DirectorLease.Fence
	decision.ObservedUsage = route.ObservedUsage
	return finalizeProviderManualSelectionReplay(decision, request.PriorDecision)
}

func validateProviderManualSelectionRequest(request ProviderManualSelectionRequest, decidedAt time.Time) error {
	context := request.Context
	if !validApplicationRef(request.RequestRef) || context.ActorRef.String() == "" ||
		context.ProjectRef.String() == "" || context.GoalRef.String() == "" ||
		context.GoalRevision == 0 || context.PlanGeneration == 0 ||
		context.WorkItemRef.String() == "" || context.WorkItemRevision == 0 ||
		context.AppSpecGeneration == 0 || !goal.IsCanonicalAppSpecHash(context.SpecHash) ||
		context.Role.String() == "" || decidedAt.IsZero() {
		return ErrProviderManualSelectionInvalid
	}
	if _, err := goal.NewRoleKey(context.Role.String()); err != nil {
		return ErrProviderManualSelectionInvalid
	}
	routeRequest := ProviderRouteRequest{
		Candidates:             []ProviderRouteCandidate{request.Candidate},
		RequiredCapabilityRefs: request.RequiredCapabilityRefs,
		ReasoningEffort:        request.ReasoningEffort,
	}
	if validateProviderRouteRequest(ProviderCatalog{observedAt: decidedAt}, routeRequest) != nil {
		return ErrProviderManualSelectionInvalid
	}
	return nil
}

func providerManualSelectionAuthorityReason(
	request ProviderManualSelectionRequest,
	times providerManualSelectionTimes,
) ProviderManualSelectionReason {
	failure := validateGoalDirectAuthority(goalDirectAuthorityContext{
		ActorRef: request.Context.ActorRef, ProjectRef: request.Context.ProjectRef,
		GoalRef: request.Context.GoalRef, At: times.decidedAt, Times: times,
	}, request.Authority)
	switch failure {
	case goalDirectAuthorityValid:
		return ""
	case goalDirectAuthorityMissing:
		return ProviderManualSelectionAuthorityMissing
	case goalDirectAuthorityDenied:
		return ProviderManualSelectionAuthorityDenied
	case goalDirectAuthorityContextDivergent:
		return ProviderManualSelectionAuthorityContextDivergent
	case goalDirectAuthorityRoleDivergent:
		return ProviderManualSelectionAuthorityRoleDivergent
	case goalDirectAuthorityRevisionDivergent:
		return ProviderManualSelectionAuthorityRevisionDivergent
	case goalDirectAuthorityInactive:
		return ProviderManualSelectionAuthorityInactive
	case goalDirectAuthorityCurrentnessUnverified:
		return ProviderManualSelectionAuthorityCurrentnessUnverified
	case goalDirectAuthorityLeaseMissing:
		return ProviderManualSelectionLeaseMissing
	case goalDirectAuthorityLeaseContextDivergent:
		return ProviderManualSelectionLeaseContextDivergent
	case goalDirectAuthorityLeaseExpired:
		return ProviderManualSelectionLeaseExpired
	default:
		return ProviderManualSelectionAuthorityMissing
	}
}

type goalDirectAuthorityContext struct {
	ActorRef   goal.ActorRef
	ProjectRef goal.ProjectRef
	GoalRef    goal.GoalRef
	At         time.Time
	Times      providerManualSelectionTimes
}

type goalDirectAuthorityFailure string

const (
	goalDirectAuthorityValid                 goalDirectAuthorityFailure = ""
	goalDirectAuthorityMissing               goalDirectAuthorityFailure = "missing"
	goalDirectAuthorityDenied                goalDirectAuthorityFailure = "denied"
	goalDirectAuthorityContextDivergent      goalDirectAuthorityFailure = "context_divergent"
	goalDirectAuthorityRoleDivergent         goalDirectAuthorityFailure = "role_divergent"
	goalDirectAuthorityRevisionDivergent     goalDirectAuthorityFailure = "revision_divergent"
	goalDirectAuthorityInactive              goalDirectAuthorityFailure = "inactive"
	goalDirectAuthorityCurrentnessUnverified goalDirectAuthorityFailure = "currentness_unverified"
	goalDirectAuthorityLeaseMissing          goalDirectAuthorityFailure = "lease_missing"
	goalDirectAuthorityLeaseContextDivergent goalDirectAuthorityFailure = "lease_context_divergent"
	goalDirectAuthorityLeaseExpired          goalDirectAuthorityFailure = "lease_expired"
)

func validateGoalDirectAuthority(
	context goalDirectAuthorityContext,
	authority GoalDirectAuthority,
) goalDirectAuthorityFailure {
	receipt := authority.Receipt
	decision := receipt.Decision()
	authorizationRequest := decision.Request()
	principal := authorizationRequest.Principal()
	if receipt.Ref() == "" || context.Times.receiptRecordedAt.IsZero() || context.Times.authorizationDecidedAt.IsZero() ||
		authorizationRequest.RequestRef() == "" || context.Times.authorizationRequestedAt.IsZero() ||
		identity.ValidatePrincipal(principal) != nil {
		return goalDirectAuthorityMissing
	}
	if decision.Outcome() != identity.AuthorizationAllowed ||
		!identity.RoleAllows(decision.Role(), identity.PermissionGoalsDirect) {
		return goalDirectAuthorityDenied
	}
	if context.Times.receiptRecordedAt.Before(context.Times.authorizationDecidedAt) || context.Times.receiptRecordedAt.After(context.At) ||
		principal.ActorRef != context.ActorRef ||
		authorizationRequest.ProjectRef() != context.ProjectRef ||
		authorizationRequest.Permission() != identity.PermissionGoalsDirect ||
		authorizationRequest.ResourceRef() != context.GoalRef.String() {
		return goalDirectAuthorityContextDivergent
	}
	if identity.ValidateRole(authority.ExpectedRole) != nil ||
		decision.Role() != authority.ExpectedRole {
		return goalDirectAuthorityRoleDivergent
	}
	if (authority.ExpectedRole == identity.RolePlatformAdmin &&
		authority.ExpectedMembershipRevision != 0) ||
		(authority.ExpectedRole != identity.RolePlatformAdmin &&
			authority.ExpectedMembershipRevision == 0) ||
		decision.MembershipRevision() != authority.ExpectedMembershipRevision {
		return goalDirectAuthorityRevisionDivergent
	}
	if decision.Role() != identity.RolePlatformAdmin {
		if decision.Role() == identity.RoleExecutionService {
			return goalDirectAuthorityCurrentnessUnverified
		}
		membership := authority.CurrentMembership
		if membership.PrincipalRef().String() == "" || membership.ProjectRef().String() == "" ||
			context.Times.membershipGrantedAt.After(context.Times.authorizationDecidedAt) {
			return goalDirectAuthorityCurrentnessUnverified
		}
		if membership.PrincipalRef() != principal.Ref || membership.ProjectRef() != context.ProjectRef {
			return goalDirectAuthorityContextDivergent
		}
		if membership.Role() != decision.Role() {
			return goalDirectAuthorityRoleDivergent
		}
		if membership.Revision() != decision.MembershipRevision() {
			return goalDirectAuthorityRevisionDivergent
		}
		if !membership.IsActive() || !identity.RoleAllows(membership.Role(), identity.PermissionGoalsDirect) {
			return goalDirectAuthorityInactive
		}
	}
	lease := authority.DirectorLease
	if !validApplicationRef(lease.Token) || lease.Fence == 0 || context.Times.leaseUntil.IsZero() {
		return goalDirectAuthorityLeaseMissing
	}
	if lease.GoalRef != context.GoalRef || lease.PrincipalRef != principal.Ref {
		return goalDirectAuthorityLeaseContextDivergent
	}
	if !context.At.Before(context.Times.leaseUntil) {
		return goalDirectAuthorityLeaseExpired
	}
	return goalDirectAuthorityValid
}

func providerManualSelectionFingerprint(
	catalog ProviderCatalog,
	request ProviderManualSelectionRequest,
	times providerManualSelectionTimes,
) string {
	context := request.Context
	authority := request.Authority
	authorization := authority.Receipt.Decision()
	authorizationRequest := authorization.Request()
	principal := authorizationRequest.Principal()
	membership := authority.CurrentMembership
	lease := authority.DirectorLease
	fields := []string{
		request.RequestRef, context.ActorRef.String(), context.ProjectRef.String(), context.GoalRef.String(),
		strconv.FormatUint(uint64(context.GoalRevision), 10),
		strconv.FormatUint(uint64(context.PlanGeneration), 10), context.WorkItemRef.String(),
		strconv.FormatUint(uint64(context.WorkItemRevision), 10),
		strconv.FormatUint(uint64(context.AppSpecGeneration), 10), context.SpecHash, context.Role.String(),
		request.Candidate.ProviderRef, request.Candidate.ModelRef, string(request.ReasoningEffort),
		times.decidedAt.Format(time.RFC3339Nano), times.catalogObservedAt.Format(time.RFC3339Nano),
		authority.Receipt.Ref(), times.receiptRecordedAt.Format(time.RFC3339Nano),
		authorizationRequest.RequestRef(), principal.Ref.String(), principal.ActorRef.String(),
		string(principal.Kind), principal.Method,
		times.authorizationRequestedAt.Format(time.RFC3339Nano),
		authorizationRequest.ProjectRef().String(), string(authorizationRequest.Permission()),
		authorizationRequest.ResourceRef(), string(authorization.Outcome()), string(authorization.Role()),
		authorization.ReasonCode(), times.authorizationDecidedAt.Format(time.RFC3339Nano),
		strconv.FormatUint(uint64(authorization.MembershipRevision()), 10),
		string(authority.ExpectedRole), strconv.FormatUint(uint64(authority.ExpectedMembershipRevision), 10),
		membership.PrincipalRef().String(), membership.ProjectRef().String(), string(membership.Role()),
		strconv.FormatUint(uint64(membership.Revision()), 10), string(membership.Status()),
		membership.GrantedBy().String(), times.membershipGrantedAt.Format(time.RFC3339Nano),
		membership.RevokedBy().String(), times.membershipRevokedAt.Format(time.RFC3339Nano),
		lease.GoalRef.String(), lease.PrincipalRef.String(),
		lease.Token, strconv.FormatUint(lease.Fence, 10), times.leaseUntil.Format(time.RFC3339Nano),
		strconv.Itoa(len(request.RequiredCapabilityRefs)),
	}
	fields = append(fields, request.RequiredCapabilityRefs...)
	if observation, found := catalog.Provider(request.Candidate.ProviderRef); found {
		fields = append(fields, "provider_observed", observation.ProviderRef, string(observation.Availability),
			string(observation.Quota), strconv.FormatInt(observation.Usage.Resources.Tokens, 10),
			strconv.FormatInt(observation.Usage.Resources.MoneyMicros, 10),
			string(observation.Usage.Resources.Currency), strconv.FormatInt(observation.Usage.Resources.ActiveTimeNS, 10),
			strconv.FormatInt(observation.Usage.Resources.ProcessSlots, 10),
			strconv.FormatInt(observation.Usage.Resources.DiskBytes, 10),
			strconv.FormatUint(uint64(observation.Usage.Known), 10), string(observation.Usage.Quality),
			times.providerObservedAt.Format(time.RFC3339Nano), times.providerExpiresAt.Format(time.RFC3339Nano),
			strconv.Itoa(len(observation.Models)))
		for _, model := range observation.Models {
			fields = append(fields, model.ProviderRef, model.ModelRef,
				strconv.Itoa(len(model.CapabilityRefs)))
			fields = append(fields, model.CapabilityRefs...)
			fields = append(fields, strconv.Itoa(len(model.ReasoningEfforts)))
			for _, effort := range model.ReasoningEfforts {
				fields = append(fields, string(effort))
			}
		}
	} else if failure, found := catalog.Failure(request.Candidate.ProviderRef); found {
		fields = append(fields, "provider_failure", failure.ProviderRef, string(failure.Code))
	} else {
		fields = append(fields, "provider_absent")
	}
	return fingerprintFields("orquesta.provider-manual-selection.v1", fields...)
}

func finalizeProviderManualSelectionReplay(
	decision ProviderManualSelectionDecision,
	prior *ProviderManualSelectionDecision,
) (ProviderManualSelectionDecision, error) {
	if prior == nil {
		return decision, nil
	}
	if *prior != decision {
		return ProviderManualSelectionDecision{}, ErrProviderManualSelectionReplayDivergent
	}
	return *prior, nil
}
