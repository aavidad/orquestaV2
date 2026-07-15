package sqlite

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func TestRepositoryV10RejectsGoalActorSpoofingAtStateBoundary(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		repository, _ := openTestRepository(t)
		project := mustRef(t, "project:create-actor-binding", goal.NewProjectRef)
		owner := testPrincipal(t, "principal:create-owner", "actor:create-owner", identity.PrincipalKindHuman)
		at := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
		provisionTestAccess(t, repository, owner, project, identity.RoleProjectOwner, at)

		state := newCreateFixture(
			t, "create-actor-binding", "request:create-actor-binding", "fingerprint:create-actor-binding",
			"actor:forged", project.String(),
		)
		state.RequestedBy = owner.Ref
		state.AuthorizationReceipt = authorizeTest(
			t, repository, owner, project, identity.PermissionGoalsCreate,
			project.String(), "auth:create-actor-binding", state.Goal.CreatedAt(),
		)
		if _, _, err := repository.CreateGoal(context.Background(), state); !application.IsStateError(err, application.StateInvalid) {
			t.Fatalf("forged create actor = %v", err)
		}
		if _, err := repository.GetGoal(context.Background(), state.Goal.Ref()); !application.IsStateError(err, application.StateNotFound) {
			t.Fatalf("forged create left goal = %v", err)
		}
	})

	t.Run("amend", func(t *testing.T) {
		repository, _ := openTestRepository(t)
		source := createFailedSourceForAmend(t, repository, "amend-actor-binding-source")
		owner := legacyTestPrincipal(t, source.RequestedBy, source.Goal.Actor())
		state := newAmendFixture(
			t, source, "amend-actor-binding", "request:amend-actor-binding", "fingerprint:amend-actor-binding",
			"forged confirmation", "actor binding",
		)
		forged := mustRef(t, "actor:forged-confirmer", goal.NewActorRef)
		state.Successor = v10ReconfirmSuccessor(t, source, state, forged)
		state.RequestedBy = owner.Ref
		state.AuthorizationReceipt = authorizeTest(
			t, repository, owner, source.Goal.Project(), identity.PermissionGoalsAmend,
			source.Goal.Ref().String(), "auth:amend-actor-binding", state.Successor.CreatedAt(),
		)
		if _, _, err := repository.AmendGoal(context.Background(), state); !application.IsStateError(err, application.StateInvalid) {
			t.Fatalf("forged amendment confirmer = %v", err)
		}
		if _, err := repository.GetGoal(context.Background(), state.Successor.Ref()); !application.IsStateError(err, application.StateNotFound) {
			t.Fatalf("forged amendment left successor = %v", err)
		}
	})
}

func v10ReconfirmSuccessor(
	t *testing.T,
	source application.GoalRecord,
	state application.AmendGoalState,
	confirmedBy goal.ActorRef,
) goal.Goal {
	t.Helper()
	candidate := state.Successor.AppSpec()
	spec, err := source.Goal.AppSpec().Amend(goal.AppSpecInput{
		Ref: candidate.Ref(), Intent: candidate.Intent(), Objective: candidate.Objective(),
		Reason: candidate.Reason(), ConfirmedBy: confirmedBy, ConfirmedAt: candidate.ConfirmedAt(),
	})
	if err != nil {
		t.Fatal(err)
	}
	successor, err := goal.NewSuccessorGoal(state.Successor.Ref(), source.Goal, spec, state.Successor.CreatedAt())
	if err != nil {
		t.Fatal(err)
	}
	return successor
}
