package acceptance_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestAcceptanceV25ProviderManualSelectionIsExactContextBoundAndFailClosed(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	actorRef := v25ManualMust(goal.NewActorRef("actor:operator"))
	projectRef := v25ManualMust(goal.NewProjectRef("project:v25"))
	goalRef := v25ManualMust(goal.NewGoalRef("goal:v25-provider-selection"))
	principalRef := v25ManualMust(identity.NewPrincipalRef("principal:operator"))
	principal := v25ManualMust(identity.NewPrincipal(principalRef, actorRef, identity.PrincipalKindHuman, "acceptance"))
	authorizationRequest := v25ManualMust(identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: "authorization-request:v25-manual", Principal: principal, ProjectRef: projectRef,
		Permission: identity.PermissionGoalsDirect, ResourceRef: goalRef.String(), RequestedAt: now.Add(-3 * time.Minute),
	}))
	authorizationDecision := v25ManualMust(identity.NewAuthorizationDecision(identity.AuthorizationDecisionInput{
		Request: authorizationRequest, Outcome: identity.AuthorizationAllowed, Role: identity.RolePlatformAdmin,
		ReasonCode: "acceptance", DecidedAt: now.Add(-2 * time.Minute),
	}))
	receipt := v25ManualMust(identity.NewAuthorizationReceipt(identity.AuthorizationReceiptInput{
		Ref: "authorization-receipt:v25-manual", Decision: authorizationDecision, RecordedAt: now.Add(-time.Minute),
	}))
	request := application.ProviderManualSelectionRequest{
		RequestRef: "provider-manual-selection:v25",
		Context: application.ProviderManualSelectionContext{
			ActorRef: actorRef, ProjectRef: projectRef, GoalRef: goalRef, GoalRevision: 10, PlanGeneration: 5,
			WorkItemRef: v25ManualMust(goal.NewWorkItemRef("work-item:v25-provider-selection")), WorkItemRevision: 4,
			AppSpecGeneration: 3, SpecHash: strings.Repeat("a", 64), Role: v25ManualMust(goal.NewRoleKey("role:director")),
		},
		Candidate:              application.ProviderRouteCandidate{ProviderRef: "provider:target", ModelRef: "model:target"},
		RequiredCapabilityRefs: []string{"capability:edit"}, ReasoningEffort: governance.ReasoningEffortMedium,
		DecidedAt: now,
		Authority: application.ProviderManualSelectionAuthority{
			Receipt: receipt, ExpectedRole: identity.RolePlatformAdmin,
			DirectorLease: application.DirectorLeaseRecord{GoalRef: goalRef, PrincipalRef: principalRef,
				Token: "director-lease:v25-manual", Fence: 17, LeaseUntil: now.Add(time.Minute)},
		},
	}
	observation := ports.ProviderCatalogObservation{
		ProviderRef: "provider:target", Availability: ports.ProviderAvailabilityAvailable, Quota: ports.ProviderQuotaAvailable,
		Usage:      governance.ResourceUsage{Quality: governance.UsageQualityUnknown},
		ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute),
		Models: []ports.ProviderModel{{ProviderRef: "provider:target", ModelRef: "model:target",
			CapabilityRefs: []string{"capability:edit"}, ReasoningEfforts: []governance.ReasoningEffort{governance.ReasoningEffortMedium}}},
	}
	catalog := v25ManualMust(application.ObserveProviderCatalog(context.Background(), now, []application.ProviderCatalogSource{
		v25ManualSelectionSource{observation},
	}))
	selected, err := application.SelectProviderModelForRole(catalog, request)
	if err != nil || !selected.Selected || selected.Reason != application.ProviderManualSelectionSelected ||
		selected.SelectedCandidate != request.Candidate || selected.Context != request.Context ||
		selected.AuthorizationReceiptRef != receipt.Ref() || selected.AuthorizationRole != identity.RolePlatformAdmin ||
		selected.AuthorizationRevision != 0 || selected.DirectorLeaseFence != 17 {
		t.Fatalf("exact manual selection=%+v err=%v", selected, err)
	}
	replayCatalog := v25ManualMust(application.ObserveProviderCatalog(context.Background(), time.Now(), nil))
	first, err := application.SelectProviderModelForRole(replayCatalog, request)
	persisted, replayRequest := first, request
	persisted.CatalogObservedAt, persisted.DecidedAt = first.CatalogObservedAt.Round(0).UTC(), first.DecidedAt.Round(0).UTC()
	replayRequest.PriorDecision = &persisted
	replayed, replayErr := application.SelectProviderModelForRole(replayCatalog, replayRequest)
	if err != nil || replayErr != nil || replayed != persisted {
		t.Fatalf("canonical persisted replay=%+v want=%+v first_err=%v replay_err=%v", replayed, persisted, err, replayErr)
	}
	observation.ExpiresAt = now
	staleCatalog := v25ManualMust(application.ObserveProviderCatalog(context.Background(), now, []application.ProviderCatalogSource{
		v25ManualSelectionSource{observation},
	}))
	stale, err := application.SelectProviderModelForRole(staleCatalog, request)
	if err != nil || stale.Selected || stale.Reason != application.ProviderManualSelectionProviderRejected ||
		stale.RouteReason != application.ProviderRouteProviderStale || stale.AuthorizationReceiptRef != "" || stale.DirectorLeaseFence != 0 {
		t.Fatalf("stale selection=%+v err=%v", stale, err)
	}
}

type v25ManualSelectionSource struct {
	observation ports.ProviderCatalogObservation
}

func (source v25ManualSelectionSource) ProviderRef() string { return source.observation.ProviderRef }
func (source v25ManualSelectionSource) ObserveProviderCatalog(context.Context) (ports.ProviderCatalogObservation, error) {
	return source.observation, nil
}

func v25ManualMust[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}
