package sqlite

import (
	"context"
	"reflect"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/intake"
)

func TestIntakeSQLitePreservesDependenciesAndReopensAfterRestart(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:intake:dependencies:create")

	rootQuestion := sqliteIntakeDependencyQuestion("root", nil)
	childQuestion := sqliteIntakeDependencyQuestion(
		"child",
		[]intake.QuestionRef{rootQuestion.Ref},
	)
	firstRequestRef := "request:intake:dependencies:first"
	first, err := system.service.ApplyIntake(ctx, application.ApplyIntakeRequest{
		RequestRef: firstRequestRef,
		ActorRef:   system.principal.ActorRef,
		ProjectRef: system.project,
		Change: intake.Change{
			StateRef:         system.stateRef,
			ExpectedRevision: created.Record.State.Revision(),
			Origin:           intake.OriginChat,
			Issues: []intake.Issue{
				sqliteIntakeDependencyIssue("root"),
				sqliteIntakeDependencyIssue("child"),
			},
			Questions: []intake.Question{rootQuestion, childQuestion},
			Choices: []intake.Choice{
				{QuestionRef: rootQuestion.Ref, OptionRef: rootQuestion.Options[0].Ref},
				{QuestionRef: childQuestion.Ref, OptionRef: childQuestion.Options[0].Ref},
			},
		},
		AuthorizationReceipt: system.authorizeIntake(
			t,
			application.IntakeOperationApply,
			firstRequestRef,
		),
	})
	sqliteTestNoError(t, err)
	if !first.Changed {
		t.Fatal("initial dependency state was not persisted")
	}

	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteIntakeTestRepository(t, system.path)
	system.service, err = application.NewIntakeService(system.repository)
	sqliteTestNoError(t, err)

	restarted, err := system.service.GetIntake(ctx, application.GetIntakeRequest{
		ActorRef:   system.principal.ActorRef,
		ProjectRef: system.project,
		StateRef:   system.stateRef,
	})
	sqliteTestNoError(t, err)
	questions := restarted.State.Questions()
	if len(questions) != 2 ||
		!reflect.DeepEqual(questions[1].DependsOn, []intake.QuestionRef{rootQuestion.Ref}) {
		t.Fatalf("dependency graph after restart = %+v", questions)
	}

	secondRequestRef := "request:intake:dependencies:change-root"
	changed, err := system.service.ApplyIntake(ctx, application.ApplyIntakeRequest{
		RequestRef: secondRequestRef,
		ActorRef:   system.principal.ActorRef,
		ProjectRef: system.project,
		Change: intake.Change{
			StateRef:         system.stateRef,
			ExpectedRevision: restarted.State.Revision(),
			Origin:           intake.OriginForm,
			Choices: []intake.Choice{{
				QuestionRef: rootQuestion.Ref,
				OptionRef:   rootQuestion.Options[1].Ref,
			}},
		},
		AuthorizationReceipt: system.authorizeIntake(
			t,
			application.IntakeOperationApply,
			secondRequestRef,
		),
	})
	sqliteTestNoError(t, err)
	if _, found := changed.Record.State.CurrentDecision(childQuestion.Ref); found {
		t.Fatal("dependent decision remained current after persisted root change")
	}
	reopened := changed.Record.State.ReopenedDecisions()
	if len(reopened) != 1 || reopened[0].QuestionRef != childQuestion.Ref ||
		len(reopened[0].InvalidatedBy) != 1 ||
		reopened[0].InvalidatedBy[0].QuestionRef != rootQuestion.Ref {
		t.Fatalf("persisted reopened decision = %+v", reopened)
	}
}

func sqliteIntakeDependencyIssue(name string) intake.Issue {
	return intake.Issue{
		Ref:       intake.IssueRef("intake-issue:" + name),
		Kind:      intake.IssueGap,
		Field:     name,
		DetailKey: intake.MessageKey("intake.issue." + name),
	}
}

func sqliteIntakeDependencyQuestion(
	name string,
	dependsOn []intake.QuestionRef,
) intake.Question {
	return intake.Question{
		Ref:         intake.QuestionRef("intake-question:" + name),
		DerivedFrom: []intake.IssueRef{intake.IssueRef("intake-issue:" + name)},
		DependsOn:   append([]intake.QuestionRef(nil), dependsOn...),
		PromptKey:   intake.MessageKey("intake.question." + name + ".prompt"),
		WhyKey:      intake.MessageKey("intake.question." + name + ".why"),
		Options: []intake.Option{
			{
				Ref:          intake.OptionRef("intake-option:" + name + "-recommended"),
				LabelKey:     intake.MessageKey("intake.option." + name + ".recommended.label"),
				RationaleKey: intake.MessageKey("intake.option." + name + ".recommended.rationale"),
				Recommended:  true,
			},
			{
				Ref:          intake.OptionRef("intake-option:" + name + "-alternate"),
				LabelKey:     intake.MessageKey("intake.option." + name + ".alternate.label"),
				RationaleKey: intake.MessageKey("intake.option." + name + ".alternate.rationale"),
			},
		},
	}
}
