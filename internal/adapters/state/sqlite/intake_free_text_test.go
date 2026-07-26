package sqlite

import (
	"context"
	"reflect"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/intake"
)

func TestIntakeSQLiteRestartsAndReplaysExactFreeText(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:intake:free-text-create")
	requestRef := "request:intake:free-text-apply"
	change := intake.Change{
		StateRef:         system.stateRef,
		ExpectedRevision: created.Record.State.Revision(),
		Origin:           intake.OriginChat,
		Issues: []intake.Issue{{
			Ref: "intake-issue:scope", Kind: intake.IssueGap,
			Field: "scope", DetailKey: "intake.issue.scope.missing",
		}},
		Questions: []intake.Question{{
			Ref:         "intake-question:scope",
			DerivedFrom: []intake.IssueRef{"intake-issue:scope"},
			PromptKey:   "intake.question.scope.prompt",
			WhyKey:      "intake.question.scope.why",
			Options: []intake.Option{
				{
					Ref:          "intake-option:scope-web",
					LabelKey:     "intake.option.scope.web.label",
					RationaleKey: "intake.option.scope.web.rationale",
					Recommended:  true,
				},
				{
					Ref:          "intake-option:scope-other",
					LabelKey:     "intake.option.scope.other.label",
					RationaleKey: "intake.option.scope.other.rationale",
					AcceptsText:  true,
				},
			},
		}},
		Choices: []intake.Choice{{
			QuestionRef: "intake-question:scope",
			OptionRef:   "intake-option:scope-other",
			AnswerText:  "Ámbito exacto\ncon segunda línea",
		}},
	}
	applied, err := system.service.ApplyIntake(ctx, application.ApplyIntakeRequest{
		RequestRef: requestRef,
		ActorRef:   system.principal.ActorRef,
		ProjectRef: system.project,
		Change:     change,
		AuthorizationReceipt: system.authorizeIntake(
			t, application.IntakeOperationApply, requestRef,
		),
	})
	sqliteTestNoError(t, err)

	if err := system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	system.repository = openSQLiteIntakeTestRepository(t, system.path)
	system.service, err = application.NewIntakeService(system.repository)
	sqliteTestNoError(t, err)

	current, err := system.service.GetIntake(ctx, application.GetIntakeRequest{
		ActorRef:   system.principal.ActorRef,
		ProjectRef: system.project,
		StateRef:   system.stateRef,
	})
	sqliteTestNoError(t, err)
	if !reflect.DeepEqual(
		application.SnapshotIntake(current.State),
		application.SnapshotIntake(applied.Record.State),
	) {
		t.Fatalf("restart state=%+v want=%+v", current.State.Decisions(), applied.Record.State.Decisions())
	}
	decision, found := current.State.CurrentDecision("intake-question:scope")
	if !found || decision.AnswerText != "Ámbito exacto\ncon segunda línea" {
		t.Fatalf("restart decision=%+v found=%v", decision, found)
	}

	replayed, found, err := system.repository.ReplayIntake(
		ctx,
		application.IntakeReplayRequest{
			RequestRef:              requestRef,
			RequestFingerprint:      applied.Record.Receipt.RequestFingerprint,
			Operation:               applied.Record.Receipt.Operation,
			ActorRef:                system.principal.ActorRef,
			ProjectRef:              system.project,
			StateRef:                system.stateRef,
			AuthorizationReceiptRef: applied.Record.Receipt.AuthorizationReceiptRef,
		},
	)
	sqliteTestNoError(t, err)
	if !found || replayed.Receipt != applied.Record.Receipt ||
		!reflect.DeepEqual(
			application.SnapshotIntake(replayed.State),
			application.SnapshotIntake(applied.Record.State),
		) {
		t.Fatalf("historical free-text replay found=%v record=%+v", found, replayed)
	}
}
