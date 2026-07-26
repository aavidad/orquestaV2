package application

import (
	"reflect"
	"testing"

	"orquesta/internal/intake"
)

func TestIntakeFreeTextSurvivesSnapshotRestoreAndAffectsDigest(t *testing.T) {
	state, err := intake.NewState(
		"intake:free-text",
		intake.Policy{MaxQuestionRounds: 2},
	)
	if err != nil {
		t.Fatal(err)
	}
	state, err = intake.Apply(state, freeTextApplicationChange("primer valor"))
	if err != nil {
		t.Fatal(err)
	}
	snapshot := SnapshotIntake(state)
	restored, err := RestoreIntake(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(SnapshotIntake(restored), snapshot) {
		t.Fatalf("restored snapshot=%+v want=%+v", SnapshotIntake(restored), snapshot)
	}
	decision, found := restored.CurrentDecision("intake-question:scope")
	if !found || decision.AnswerText != "primer valor" {
		t.Fatalf("restored decision=%+v found=%v", decision, found)
	}

	changed, err := intake.Apply(state, intake.Change{
		StateRef: "intake:free-text", ExpectedRevision: 2, Origin: intake.OriginForm,
		Choices: []intake.Choice{{
			QuestionRef: "intake-question:scope",
			OptionRef:   "intake-option:scope-other",
			AnswerText:  "segundo valor",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	firstDigest, err := IntakeStateDigest(state)
	if err != nil {
		t.Fatal(err)
	}
	secondDigest, err := IntakeStateDigest(changed)
	if err != nil {
		t.Fatal(err)
	}
	if firstDigest == secondDigest {
		t.Fatalf("answer text did not affect state digest: %s", firstDigest)
	}
}

func freeTextApplicationChange(answer string) intake.Change {
	return intake.Change{
		StateRef: "intake:free-text", ExpectedRevision: 1, Origin: intake.OriginChat,
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
			AnswerText:  answer,
		}},
	}
}
