package application

import (
	"context"
	"reflect"
	"testing"

	"orquesta/internal/intake"
)

func TestIntakeServiceAcceptsRecommendationsAtomicallyAndReplaysExactReceipt(
	t *testing.T,
) {
	ctx := context.Background()
	system := newIntakeTestSystem(t)
	created := mustCreateIntake(t, system)
	questions, err := system.service.ApplyIntake(ctx, ApplyIntakeRequest{
		RequestRef: "request:intake:questions",
		ActorRef:   system.actor,
		ProjectRef: system.project,
		Change:     intakeQuestionChange(1, intake.OriginChat),
		AuthorizationReceipt: system.authorizationFor(
			t,
			IntakeOperationApply,
			"request:intake:questions",
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	request := AcceptIntakeRecommendationsRequest{
		RequestRef:       "request:intake:accept-recommendations",
		ActorRef:         system.actor,
		ProjectRef:       system.project,
		StateRef:         questions.Record.State.Ref(),
		ExpectedRevision: questions.Record.State.Revision(),
		Origin:           intake.OriginForm,
		QuestionRound:    1,
		AuthorizationReceipt: system.authorizationFor(
			t,
			IntakeOperationApply,
			"request:intake:accept-recommendations",
		),
	}
	applyCalls := system.store.applyCalls
	accepted, err := system.service.AcceptIntakeRecommendations(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if !accepted.Changed || system.store.applyCalls != applyCalls+1 ||
		accepted.Record.State.Revision() != questions.Record.State.Revision()+1 {
		t.Fatalf("accepted=%+v apply calls=%d/%d",
			accepted, applyCalls, system.store.applyCalls)
	}
	decision, found := accepted.Record.State.CurrentDecision(
		"intake-question:audience",
	)
	if !found || decision.Choice != "intake-option:audience-team" {
		t.Fatalf("accepted decision=%+v found=%v", decision, found)
	}

	later, err := system.service.ApplyIntake(ctx, ApplyIntakeRequest{
		RequestRef: "request:intake:after-recommendations",
		ActorRef:   system.actor,
		ProjectRef: system.project,
		Change: intake.Change{
			StateRef:         accepted.Record.State.Ref(),
			ExpectedRevision: accepted.Record.State.Revision(),
			Origin:           intake.OriginChat,
			Choices: []intake.Choice{{
				QuestionRef: "intake-question:audience",
				OptionRef:   "intake-option:audience-personal",
			}},
		},
		AuthorizationReceipt: system.authorizationFor(
			t,
			IntakeOperationApply,
			"request:intake:after-recommendations",
		),
	})
	if err != nil || !later.Changed {
		t.Fatalf("later mutation=%+v err=%v", later, err)
	}
	if err := ValidateIntakeChain([]IntakeRecord{
		created.Record,
		questions.Record,
		accepted.Record,
		later.Record,
	}); err != nil {
		t.Fatalf("recommendation chain is not recoverable: %v", err)
	}
	replayed, err := system.service.AcceptIntakeRecommendations(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Changed ||
		replayed.Record.Receipt != accepted.Record.Receipt ||
		!reflectIntakeStateEqual(replayed.Record.State, accepted.Record.State) ||
		system.store.applyCalls != applyCalls+2 {
		t.Fatalf("replayed=%+v apply calls=%d", replayed, system.store.applyCalls)
	}

	divergent := request
	divergent.Origin = intake.OriginChat
	if _, err = system.service.AcceptIntakeRecommendations(
		ctx,
		divergent,
	); !IsStateError(err, StateConflict) {
		t.Fatalf("divergent replay error=%v", err)
	}
}

func TestIntakeServiceRecommendationAcceptanceRejectsStaleWithoutWrite(t *testing.T) {
	ctx := context.Background()
	system := newIntakeTestSystem(t)
	mustCreateIntake(t, system)
	questions, err := system.service.ApplyIntake(ctx, ApplyIntakeRequest{
		RequestRef: "request:intake:stale-questions",
		ActorRef:   system.actor,
		ProjectRef: system.project,
		Change:     intakeQuestionChange(1, intake.OriginChat),
		AuthorizationReceipt: system.authorizationFor(
			t,
			IntakeOperationApply,
			"request:intake:stale-questions",
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = system.service.ApplyIntake(ctx, ApplyIntakeRequest{
		RequestRef: "request:intake:stale-later",
		ActorRef:   system.actor,
		ProjectRef: system.project,
		Change: intake.Change{
			StateRef:         questions.Record.State.Ref(),
			ExpectedRevision: questions.Record.State.Revision(),
			Origin:           intake.OriginForm,
			Choices: []intake.Choice{{
				QuestionRef: "intake-question:audience",
				OptionRef:   "intake-option:audience-personal",
			}},
		},
		AuthorizationReceipt: system.authorizationFor(
			t,
			IntakeOperationApply,
			"request:intake:stale-later",
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	applyCalls := system.store.applyCalls
	_, err = system.service.AcceptIntakeRecommendations(
		ctx,
		AcceptIntakeRecommendationsRequest{
			RequestRef:       "request:intake:stale-accept",
			ActorRef:         system.actor,
			ProjectRef:       system.project,
			StateRef:         questions.Record.State.Ref(),
			ExpectedRevision: questions.Record.State.Revision(),
			Origin:           intake.OriginForm,
			QuestionRound:    1,
			AuthorizationReceipt: system.authorizationFor(
				t,
				IntakeOperationApply,
				"request:intake:stale-accept",
			),
		},
	)
	if intake.ErrorCodeOf(err) != intake.ErrorRevisionConflict ||
		system.store.applyCalls != applyCalls {
		t.Fatalf("stale error=%v apply calls=%d/%d",
			err, applyCalls, system.store.applyCalls)
	}
}

func TestIntakeServiceContextIsPureAndBoundToPersistedRevision(t *testing.T) {
	ctx := context.Background()
	system := newIntakeTestSystem(t)
	mustCreateIntake(t, system)
	questions, err := system.service.ApplyIntake(ctx, ApplyIntakeRequest{
		RequestRef: "request:intake:context-questions",
		ActorRef:   system.actor,
		ProjectRef: system.project,
		Change:     intakeQuestionChange(1, intake.OriginChat),
		AuthorizationReceipt: system.authorizationFor(
			t,
			IntakeOperationApply,
			"request:intake:context-questions",
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	before := questions.Record.State
	applyCalls := system.store.applyCalls
	view, err := system.service.GetIntakeContext(ctx, GetIntakeContextRequest{
		ActorRef: system.actor, ProjectRef: system.project,
		Context: intake.ContextRequest{
			StateRef: before.Ref(), ExpectedRevision: before.Revision(),
			Origin: intake.OriginForm, Kind: intake.ContextHelp,
		},
	})
	if err != nil || len(view.Questions) != 1 ||
		view.Questions[0].Question.Ref != "intake-question:audience" ||
		system.store.applyCalls != applyCalls {
		t.Fatalf("context=%+v err=%v apply calls=%d/%d",
			view, err, applyCalls, system.store.applyCalls)
	}
	current, err := system.service.GetIntake(ctx, GetIntakeRequest{
		ActorRef: system.actor, ProjectRef: system.project, StateRef: before.Ref(),
	})
	if err != nil || !reflect.DeepEqual(
		SnapshotIntake(current.State),
		SnapshotIntake(before),
	) {
		t.Fatalf("context mutated state=%+v err=%v", SnapshotIntake(current.State), err)
	}
}
