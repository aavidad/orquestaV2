package sqlite

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/intake"
)

func TestIntakeQuestionRevisionSurvivesRestartAndReplaysExactly(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:intake:question-version:create")
	identity, err := intake.NewDerivationIdentity(
		"orquesta.test.questions",
		"v1",
		strings.Repeat("b", 64),
	)
	sqliteTestNoError(t, err)
	question := sqliteIntakeDependencyQuestion("versioned", nil)
	initialRequestRef := "request:intake:question-version:initial"
	initial, err := system.service.ApplyIntake(ctx, application.ApplyIntakeRequest{
		RequestRef: initialRequestRef,
		ActorRef:   system.principal.ActorRef,
		ProjectRef: system.project,
		Change: intake.Change{
			StateRef:         system.stateRef,
			ExpectedRevision: created.Record.State.Revision(),
			Origin:           intake.OriginChat,
			Issues: []intake.Issue{
				sqliteIntakeDependencyIssue("versioned"),
			},
			Questions: []intake.Question{question},
			Choices: []intake.Choice{{
				QuestionRef: question.Ref,
				OptionRef:   question.Options[0].Ref,
			}},
			Derivation: identity,
		},
		AuthorizationReceipt: system.authorizeIntake(
			t,
			application.IntakeOperationApply,
			initialRequestRef,
		),
	})
	sqliteTestNoError(t, err)

	revisedQuestion := question
	revisedQuestion.Options = append([]intake.Option(nil), question.Options...)
	revisedQuestion.Options[0].Recommended = false
	revisedQuestion.Options[1].Recommended = true
	requestRef := "request:intake:question-version:revise"
	request := application.ApplyIntakeRequest{
		RequestRef: requestRef,
		ActorRef:   system.principal.ActorRef,
		ProjectRef: system.project,
		Change: intake.Change{
			StateRef:         system.stateRef,
			ExpectedRevision: initial.Record.State.Revision(),
			Origin:           intake.OriginForm,
			QuestionRevisions: []intake.Question{
				revisedQuestion,
			},
			Derivation: identity,
		},
		AuthorizationReceipt: system.authorizeIntake(
			t,
			application.IntakeOperationApply,
			requestRef,
		),
	}
	revised, err := system.service.ApplyIntake(ctx, request)
	sqliteTestNoError(t, err)
	if !revised.Changed {
		t.Fatal("question revision was not persisted")
	}

	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteIntakeTestRepository(t, system.path)
	t.Cleanup(func() { _ = system.repository.Close() })
	system.service, err = application.NewIntakeService(system.repository)
	sqliteTestNoError(t, err)

	restarted, err := system.service.GetIntake(ctx, application.GetIntakeRequest{
		ActorRef:   system.principal.ActorRef,
		ProjectRef: system.project,
		StateRef:   system.stateRef,
	})
	sqliteTestNoError(t, err)
	if !reflect.DeepEqual(
		application.SnapshotIntake(restarted.State),
		application.SnapshotIntake(revised.Record.State),
	) {
		t.Fatal("restart changed the versioned intake snapshot")
	}
	if _, found := restarted.State.CurrentDecision(question.Ref); found {
		t.Fatal("restart restored a decision from the superseded question")
	}
	versions := restarted.State.QuestionVersions()
	if len(versions) != 2 ||
		versions[1].ReplacesRevision != versions[0].Revision {
		t.Fatalf("restart versions = %+v", versions)
	}

	replayed, err := system.service.ApplyIntake(ctx, request)
	sqliteTestNoError(t, err)
	if replayed.Changed ||
		replayed.Record.Receipt != revised.Record.Receipt ||
		!reflect.DeepEqual(
			application.SnapshotIntake(replayed.Record.State),
			application.SnapshotIntake(revised.Record.State),
		) {
		t.Fatalf(
			"restart replay changed result: changed=%t receipt=%+v/%+v",
			replayed.Changed,
			replayed.Record.Receipt,
			revised.Record.Receipt,
		)
	}
}

func TestIntakeQuestionRetirementSurvivesRestartAndReplaysExactly(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:intake:question-retirement:create")
	identity, err := intake.NewDerivationIdentity(
		"orquesta.test.questions",
		"v1",
		strings.Repeat("d", 64),
	)
	sqliteTestNoError(t, err)
	question := sqliteIntakeDependencyQuestion("retired", nil)
	initialRequestRef := "request:intake:question-retirement:initial"
	initial, err := system.service.ApplyIntake(ctx, application.ApplyIntakeRequest{
		RequestRef: initialRequestRef,
		ActorRef:   system.principal.ActorRef, ProjectRef: system.project,
		Change: intake.Change{
			StateRef: system.stateRef, ExpectedRevision: created.Record.State.Revision(),
			Origin: intake.OriginChat, Derivation: identity,
			Issues:    []intake.Issue{sqliteIntakeDependencyIssue("retired")},
			Questions: []intake.Question{question},
			Choices: []intake.Choice{{
				QuestionRef: question.Ref,
				OptionRef:   question.Options[0].Ref,
			}},
		},
		AuthorizationReceipt: system.authorizeIntake(
			t,
			application.IntakeOperationApply,
			initialRequestRef,
		),
	})
	sqliteTestNoError(t, err)
	requestRef := "request:intake:question-retirement:retire"
	request := application.ApplyIntakeRequest{
		RequestRef: requestRef,
		ActorRef:   system.principal.ActorRef, ProjectRef: system.project,
		Change: intake.Change{
			StateRef: system.stateRef, ExpectedRevision: initial.Record.State.Revision(),
			Origin: intake.OriginForm, Derivation: identity,
			QuestionRetirements: []intake.QuestionRef{question.Ref},
		},
		AuthorizationReceipt: system.authorizeIntake(
			t,
			application.IntakeOperationApply,
			requestRef,
		),
	}
	retired, err := system.service.ApplyIntake(ctx, request)
	sqliteTestNoError(t, err)
	if !retired.Changed {
		t.Fatal("question retirement was not persisted")
	}

	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteIntakeTestRepository(t, system.path)
	t.Cleanup(func() { _ = system.repository.Close() })
	system.service, err = application.NewIntakeService(system.repository)
	sqliteTestNoError(t, err)

	restarted, err := system.service.GetIntake(ctx, application.GetIntakeRequest{
		ActorRef:   system.principal.ActorRef,
		ProjectRef: system.project,
		StateRef:   system.stateRef,
	})
	sqliteTestNoError(t, err)
	if !reflect.DeepEqual(
		application.SnapshotIntake(restarted.State),
		application.SnapshotIntake(retired.Record.State),
	) {
		t.Fatal("restart changed the retirement tombstone")
	}
	if _, found := restarted.State.CurrentDecision(question.Ref); found {
		t.Fatal("restart restored a decision for retired question")
	}
	versions := restarted.State.QuestionVersions()
	if len(versions) != 2 || !versions[1].Retired ||
		versions[1].ReplacesRevision != versions[0].Revision {
		t.Fatalf("restart retirement versions=%+v", versions)
	}

	replayed, err := system.service.ApplyIntake(ctx, request)
	sqliteTestNoError(t, err)
	if replayed.Changed ||
		replayed.Record.Receipt != retired.Record.Receipt ||
		!reflect.DeepEqual(
			application.SnapshotIntake(replayed.Record.State),
			application.SnapshotIntake(retired.Record.State),
		) {
		t.Fatalf(
			"restart replay changed retirement: changed=%t receipt=%+v/%+v",
			replayed.Changed,
			replayed.Record.Receipt,
			retired.Record.Receipt,
		)
	}
}
