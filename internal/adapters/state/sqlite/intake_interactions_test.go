package sqlite

import (
	"context"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/intake"
)

func TestIntakeRecommendationAcceptanceReplaysAfterSQLiteRestart(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:intake:accept:create")

	questionChange := sqliteIntakeAudienceChange(
		created.Record.State,
		system.stateRef,
		intake.OriginChat,
		"intake-option:audience-team",
	)
	questionChange.Choices = nil
	questionRequestRef := "request:intake:accept:questions"
	questions, err := system.service.ApplyIntake(ctx, application.ApplyIntakeRequest{
		RequestRef: questionRequestRef,
		ActorRef:   system.principal.ActorRef,
		ProjectRef: system.project,
		Change:     questionChange,
		AuthorizationReceipt: system.authorizeIntake(
			t,
			application.IntakeOperationApply,
			questionRequestRef,
		),
	})
	sqliteTestNoError(t, err)

	acceptRequestRef := "request:intake:accept:recommendations"
	acceptRequest := application.AcceptIntakeRecommendationsRequest{
		RequestRef:       acceptRequestRef,
		ActorRef:         system.principal.ActorRef,
		ProjectRef:       system.project,
		StateRef:         system.stateRef,
		ExpectedRevision: questions.Record.State.Revision(),
		Origin:           intake.OriginForm,
		QuestionRound:    1,
		AuthorizationReceipt: system.authorizeIntake(
			t,
			application.IntakeOperationApply,
			acceptRequestRef,
		),
	}
	accepted, err := system.service.AcceptIntakeRecommendations(ctx, acceptRequest)
	sqliteTestNoError(t, err)
	if !accepted.Changed {
		t.Fatal("recommendation acceptance was not persisted")
	}
	later := system.apply(
		t,
		accepted.Record.State,
		"request:intake:accept:later",
		intake.OriginChat,
		"intake-option:audience-personal",
	)

	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteIntakeTestRepository(t, system.path)
	system.service, err = application.NewIntakeService(system.repository)
	sqliteTestNoError(t, err)

	current, err := system.service.GetIntake(ctx, application.GetIntakeRequest{
		ActorRef: system.principal.ActorRef, ProjectRef: system.project,
		StateRef: system.stateRef,
	})
	sqliteTestNoError(t, err)
	if current.State.Revision() != later.Record.State.Revision() {
		t.Fatalf("current revision after restart=%d want=%d",
			current.State.Revision(), later.Record.State.Revision())
	}
	replayed, err := system.service.AcceptIntakeRecommendations(ctx, acceptRequest)
	sqliteTestNoError(t, err)
	if replayed.Changed ||
		replayed.Record.Receipt != accepted.Record.Receipt ||
		replayed.Record.State.Revision() != accepted.Record.State.Revision() {
		t.Fatalf("replay after restart=%+v accepted=%+v", replayed, accepted)
	}
}
