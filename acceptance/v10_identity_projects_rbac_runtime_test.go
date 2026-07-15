package acceptance_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func v10AssertRealCollaborativeRBAC(t *testing.T, fixture v10Fixture) {
	t.Helper()
	ctx := context.Background()
	system := v10NewRealSystem(t, fixture)
	t.Cleanup(func() {
		if system.repository != nil {
			_ = system.repository.Close()
		}
	})
	projectA := system.projectA.ProjectRef()
	projectB := system.projectB.ProjectRef()
	ownerAccess := v10Access(t, system.owner, projectA)
	contributorAccess := v10Access(t, system.contributor, projectA)
	viewerAccess := v10Access(t, system.viewer, projectA)
	serviceAccess := v10Access(t, system.service, projectA)
	outsiderAAccess := v10Access(t, system.outsider, projectA)
	outsiderBAccess := v10Access(t, system.outsider, projectB)

	contributorGrant := v10GrantActive(
		t, system, system.owner, ownerAccess, system.contributor,
		identity.RoleContributor, "membership-request:v10-contributor",
	)
	viewerGrant := v10GrantActive(
		t, system, system.owner, ownerAccess, system.viewer,
		identity.RoleViewer, "membership-request:v10-viewer",
	)
	serviceGrant := v10GrantActive(
		t, system, system.owner, ownerAccess, system.service,
		identity.RoleContributor, "membership-request:v10-service",
	)
	if contributorGrant.audit.Ref() == viewerGrant.audit.Ref() ||
		contributorGrant.audit.Ref() == serviceGrant.audit.Ref() || viewerGrant.audit.Ref() == serviceGrant.audit.Ref() {
		t.Fatal("membership grants reused an audit receipt")
	}
	system.clock.Advance(time.Second)
	replayedMembership, replayedAudit, created, err := system.orchestrator.GrantMembership(
		ctx, ownerAccess, serviceGrant.request, system.service,
	)
	if err != nil || created || replayedMembership.Snapshot() != serviceGrant.membership.Snapshot() ||
		replayedAudit.Ref() != serviceGrant.audit.Ref() {
		t.Fatalf("service grant replay: membership=%+v audit=%+v created=%v err=%v",
			replayedMembership.Snapshot(), replayedAudit, created, err)
	}
	staleGrant := v10GrantRequest(
		t, "membership-request:v10-service-stale", system.owner, system.service.Ref,
		projectA, identity.RoleViewer, 0, system.clock.Now(),
	)
	if _, _, _, err := system.orchestrator.GrantMembership(
		ctx, ownerAccess, staleGrant, system.service,
	); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("stale membership grant accepted: %v", err)
	}

	v10AssertRolePolicy(
		t, fixture, system.repository, projectA, identity.RoleContributor,
		[]identity.Principal{system.contributor, system.service}, system.clock.Now(),
	)
	v10AssertRolePolicy(
		t, fixture, system.repository, projectA, identity.RoleViewer,
		[]identity.Principal{system.viewer}, system.clock.Now(),
	)

	sourceResult, err := system.orchestrator.Submit(ctx, ownerAccess, application.SubmitRequest{
		RequestRef: "request:v10-shared-source", Statement: "shared project evidence", Confirm: true,
	})
	if err != nil || !sourceResult.Created || sourceResult.Record.RequestedBy != system.owner.Ref {
		t.Fatalf("owner source create: %+v err=%v", sourceResult, err)
	}
	source := v10CloseGoal(t, system, sourceResult.Record.Goal.Ref())
	if source.Goal.Actor() != system.owner.ActorRef || len(source.Artifacts) != 1 ||
		source.Goal.State() != goal.GoalStateSucceeded {
		t.Fatalf("shared source did not close with evidence: %+v", source)
	}
	sharedByContributor, err := system.orchestrator.GetGoal(ctx, contributorAccess, source.Goal.Ref())
	if err != nil || sharedByContributor.Goal.Actor() != system.owner.ActorRef {
		t.Fatalf("contributor cannot share owner Goal: %+v err=%v", sharedByContributor, err)
	}
	ownerArtifact, err := system.orchestrator.GetArtifact(
		ctx, ownerAccess, source.Goal.Ref(), source.Artifacts[0].Stored.Ref,
	)
	if err != nil {
		t.Fatalf("owner artifact read: %v", err)
	}
	contributorArtifact, err := system.orchestrator.GetArtifact(
		ctx, contributorAccess, source.Goal.Ref(), source.Artifacts[0].Stored.Ref,
	)
	if err != nil || !reflect.DeepEqual(contributorArtifact, ownerArtifact) {
		t.Fatalf("artifact is not shared by project membership: %+v err=%v", contributorArtifact, err)
	}
	viewerArtifact, err := system.orchestrator.GetArtifact(
		ctx, viewerAccess, source.Goal.Ref(), source.Artifacts[0].Stored.Ref,
	)
	if err != nil || !reflect.DeepEqual(viewerArtifact, ownerArtifact) {
		t.Fatalf("viewer exact artifact read: %+v err=%v", viewerArtifact, err)
	}

	amended, err := system.orchestrator.Amend(ctx, contributorAccess, application.AmendRequest{
		RequestRef: "request:v10-collaborative-amend", SourceGoalRef: source.Goal.Ref(),
		ExpectedSourceRevision: source.Goal.Revision(), ExpectedSourceSpecHash: source.Goal.SpecHash(),
		Statement: "collaborator-confirmed successor", NormalizedObjective: "shared successor",
		Reason: "operator.collaborative_amendment", Confirm: true,
	})
	if err != nil || !amended.Created || amended.Record.RequestedBy != system.contributor.Ref ||
		amended.Record.Goal.Actor() != system.owner.ActorRef ||
		amended.Record.Goal.AppSpec().Intent().Actor() != system.owner.ActorRef ||
		amended.Record.Goal.AppSpec().ConfirmedBy() != system.contributor.ActorRef {
		t.Fatalf("collaborative attribution crossed creator/caller: %+v err=%v", amended, err)
	}
	if got, err := system.orchestrator.GetGoal(ctx, contributorAccess, amended.Record.Goal.Ref()); err != nil || got.RequestedBy != system.contributor.Ref {
		t.Fatalf("collaborator cannot read attributed successor: %+v err=%v", got, err)
	}

	beforeViewer, err := system.repository.Status(ctx, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := system.orchestrator.Submit(ctx, viewerAccess, application.SubmitRequest{
		RequestRef: "request:v10-viewer-write", Statement: "must remain denied", Confirm: true,
	}); !v10Forbidden(err) {
		t.Fatalf("viewer Goal write was not forbidden: %v", err)
	}
	if _, err := system.orchestrator.Amend(ctx, viewerAccess, application.AmendRequest{
		RequestRef: "request:v10-viewer-amend", SourceGoalRef: source.Goal.Ref(),
		ExpectedSourceRevision: source.Goal.Revision(), ExpectedSourceSpecHash: source.Goal.SpecHash(),
		Statement: "must remain denied", Reason: "operator.viewer_denied", Confirm: true,
	}); !v10Forbidden(err) {
		t.Fatalf("viewer amendment was not forbidden: %v", err)
	}
	viewerGrantRequest := v10GrantRequest(
		t, "membership-request:v10-viewer-denied", system.viewer, system.outsider.Ref,
		projectA, identity.RoleViewer, 0, system.clock.Now(),
	)
	if _, _, _, err := system.orchestrator.GrantMembership(
		ctx, viewerAccess, viewerGrantRequest, system.outsider,
	); !v10Forbidden(err) {
		t.Fatalf("viewer membership management was not forbidden: %v", err)
	}
	if listed, err := system.orchestrator.ListGoals(ctx, viewerAccess, 20); err != nil || len(listed) != 2 {
		t.Fatalf("viewer project list: goals=%+v err=%v", listed, err)
	}
	if status, err := system.orchestrator.Status(ctx, viewerAccess); err != nil || status.Goals != 2 {
		t.Fatalf("viewer project status: status=%+v err=%v", status, err)
	}
	afterViewer, err := system.repository.Status(ctx, projectA)
	if err != nil || afterViewer != beforeViewer {
		t.Fatalf("viewer denial changed lifecycle state: before=%+v after=%+v err=%v", beforeViewer, afterViewer, err)
	}

	if _, err := system.orchestrator.GetGoal(ctx, serviceAccess, source.Goal.Ref()); err != nil {
		t.Fatalf("service contributor could not read shared Goal: %v", err)
	}
	serviceGoal, err := system.orchestrator.Submit(ctx, serviceAccess, application.SubmitRequest{
		RequestRef: "request:v10-service-goal", Statement: "service follows contributor policy", Confirm: true,
	})
	if err != nil || !serviceGoal.Created || serviceGoal.Record.RequestedBy != system.service.Ref ||
		serviceGoal.Record.Goal.Actor() != system.service.ActorRef {
		t.Fatalf("service contributor create: %+v err=%v", serviceGoal, err)
	}
	system.clock.Advance(time.Second)
	revokeRequest := v10RevokeRequest(
		t, "membership-request:v10-service-revoke", system.owner, system.service.Ref,
		projectA, 1, system.clock.Now(),
	)
	revoked, revokeAudit, changed, err := system.orchestrator.RevokeMembership(ctx, ownerAccess, revokeRequest)
	if err != nil || !changed || revoked.Status() != identity.MembershipRevoked || revoked.Revision() != 2 ||
		revokeAudit.Action() != identity.MembershipAuditRevoked || revokeAudit.PreviousRevision() != 1 {
		t.Fatalf("service revoke: membership=%+v audit=%+v changed=%v err=%v",
			revoked.Snapshot(), revokeAudit, changed, err)
	}
	system.clock.Advance(time.Second)
	replayedRevoke, replayedRevokeAudit, changed, err := system.orchestrator.RevokeMembership(
		ctx, ownerAccess, revokeRequest,
	)
	if err != nil || changed || replayedRevoke.Snapshot() != revoked.Snapshot() ||
		replayedRevokeAudit.Ref() != revokeAudit.Ref() {
		t.Fatalf("service revoke replay: membership=%+v audit=%+v changed=%v err=%v",
			replayedRevoke.Snapshot(), replayedRevokeAudit, changed, err)
	}
	staleRevoke := v10RevokeRequest(
		t, "membership-request:v10-service-stale-revoke", system.owner, system.service.Ref,
		projectA, 1, system.clock.Now(),
	)
	if _, _, _, err := system.orchestrator.RevokeMembership(
		ctx, ownerAccess, staleRevoke,
	); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("stale revoke accepted: %v", err)
	}
	beforeRevokedWrite, err := system.repository.Status(ctx, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := system.orchestrator.Submit(ctx, serviceAccess, application.SubmitRequest{
		RequestRef: "request:v10-revoked-service", Statement: "must remain revoked", Confirm: true,
	}); !v10Forbidden(err) {
		t.Fatalf("revoked service write was not forbidden: %v", err)
	}
	if _, err := system.orchestrator.GetGoal(ctx, serviceAccess, source.Goal.Ref()); !v10Forbidden(err) {
		t.Fatalf("revoked known service read was not forbidden: %v", err)
	}
	afterRevokedWrite, err := system.repository.Status(ctx, projectA)
	if err != nil || afterRevokedWrite != beforeRevokedWrite {
		t.Fatalf("revoked service changed lifecycle: before=%+v after=%+v err=%v",
			beforeRevokedWrite, afterRevokedWrite, err)
	}

	beforeOutsider, err := system.repository.Status(ctx, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := system.orchestrator.GetGoal(ctx, outsiderAAccess, source.Goal.Ref()); !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("outsider enumerated project A Goal: %v", err)
	}
	if _, err := system.orchestrator.GetArtifact(
		ctx, outsiderAAccess, source.Goal.Ref(), source.Artifacts[0].Stored.Ref,
	); !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("outsider enumerated project A artifact: %v", err)
	}
	if _, err := system.orchestrator.ListGoals(ctx, outsiderAAccess, 20); !v10Forbidden(err) {
		t.Fatalf("outsider project A list was not forbidden: %v", err)
	}
	if _, err := system.orchestrator.Status(ctx, outsiderAAccess); !v10Forbidden(err) {
		t.Fatalf("outsider project A status was not forbidden: %v", err)
	}
	if _, err := system.orchestrator.Submit(ctx, outsiderAAccess, application.SubmitRequest{
		RequestRef: "request:v10-outsider-write", Statement: "must not cross project", Confirm: true,
	}); !v10Forbidden(err) {
		t.Fatalf("outsider project A write was not forbidden: %v", err)
	}
	if listed, err := system.orchestrator.ListGoals(ctx, outsiderBAccess, 20); err != nil || len(listed) != 0 {
		t.Fatalf("project B list leaked project A: goals=%+v err=%v", listed, err)
	}
	if status, err := system.orchestrator.Status(ctx, outsiderBAccess); err != nil || status != (application.RepositoryStatus{}) {
		t.Fatalf("project B status leaked project A: status=%+v err=%v", status, err)
	}
	afterOutsider, err := system.repository.Status(ctx, projectA)
	if err != nil || afterOutsider != beforeOutsider {
		t.Fatalf("outsider probes changed lifecycle: before=%+v after=%+v err=%v",
			beforeOutsider, afterOutsider, err)
	}

	allowedRequest := v10AuthorizationRequest(
		t, "authorization-request:v10-owner-replay", system.owner, projectA,
		identity.PermissionGoalsGet, source.Goal.Ref().String(), system.clock.Now(),
	)
	allowed, err := system.repository.Authorize(ctx, allowedRequest)
	if err != nil || allowed.Decision().Outcome() != identity.AuthorizationAllowed {
		t.Fatalf("allowed authorization receipt: %+v err=%v", allowed, err)
	}
	allowedReplay, err := system.repository.Authorize(ctx, allowedRequest)
	if err != nil || allowedReplay.Ref() != allowed.Ref() ||
		!allowedReplay.RecordedAt().Equal(allowed.RecordedAt()) {
		t.Fatalf("allowed authorization replay: %+v err=%v", allowedReplay, err)
	}
	deniedRequest := v10AuthorizationRequest(
		t, "authorization-request:v10-outsider-replay", system.outsider, projectA,
		identity.PermissionGoalsGet, source.Goal.Ref().String(), system.clock.Now(),
	)
	denied, err := system.repository.Authorize(ctx, deniedRequest)
	if err != nil || denied.Decision().Outcome() != identity.AuthorizationDenied || denied.Decision().Role() != "" {
		t.Fatalf("denied authorization receipt: %+v err=%v", denied, err)
	}
	deniedReplay, err := system.repository.Authorize(ctx, deniedRequest)
	if err != nil || deniedReplay.Ref() != denied.Ref() ||
		!deniedReplay.RecordedAt().Equal(denied.RecordedAt()) {
		t.Fatalf("denied authorization replay: %+v err=%v", deniedReplay, err)
	}

	v10AssertSingleDatabaseAudit(
		t, system, source, amended.Record, serviceGoal.Record,
		serviceGrant.audit, revokeAudit, allowed, denied,
	)
	v10AssertRestart(t, system, source, amended.Record, serviceGoal.Record)
}
