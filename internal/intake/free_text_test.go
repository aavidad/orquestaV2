package intake

import (
	"strings"
	"testing"
)

func TestFreeTextChoiceIsDurableAndReopensDependentsWhenOnlyTextChanges(t *testing.T) {
	state := mustState(t, 2)
	var err error
	state, err = Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues: []Issue{
			audienceGap(),
			{
				Ref: "intake-issue:delivery-gap", Kind: IssueGap,
				Field: "delivery", DetailKey: "intake.issue.delivery.missing",
			},
		},
		Questions: []Question{
			freeTextAudienceQuestion(),
			{
				Ref:         "intake-question:delivery",
				DerivedFrom: []IssueRef{"intake-issue:delivery-gap"},
				DependsOn:   []QuestionRef{"intake-question:audience"},
				PromptKey:   "intake.question.delivery.prompt",
				WhyKey:      "intake.question.delivery.why",
				Options: []Option{
					{
						Ref:          "intake-option:delivery-web",
						LabelKey:     "intake.option.delivery.web.label",
						RationaleKey: "intake.option.delivery.web.rationale",
						Recommended:  true,
					},
					{
						Ref:          "intake-option:delivery-cli",
						LabelKey:     "intake.option.delivery.cli.label",
						RationaleKey: "intake.option.delivery.cli.rationale",
					},
				},
			},
		},
		Choices: []Choice{
			{
				QuestionRef: "intake-question:audience",
				OptionRef:   "intake-option:audience-other",
				AnswerText:  "Equipo interno y clínicas asociadas",
			},
			{
				QuestionRef: "intake-question:delivery",
				OptionRef:   "intake-option:delivery-web",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	decision, found := state.CurrentDecision("intake-question:audience")
	if !found || decision.AnswerText != "Equipo interno y clínicas asociadas" {
		t.Fatalf("free-text decision = %+v, found=%v", decision, found)
	}

	changed, err := Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 2, Origin: OriginForm,
		Choices: []Choice{{
			QuestionRef: "intake-question:audience",
			OptionRef:   "intake-option:audience-other",
			AnswerText:  "Solo el equipo interno",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision, found = changed.CurrentDecision("intake-question:audience"); !found ||
		decision.AnswerText != "Solo el equipo interno" {
		t.Fatalf("changed free-text decision = %+v, found=%v", decision, found)
	}
	if _, found = changed.CurrentDecision("intake-question:delivery"); found {
		t.Fatal("dependent decision stayed active after answer text changed")
	}
	reopened := changed.ReopenedDecisions()
	if len(reopened) != 1 ||
		reopened[0].QuestionRef != "intake-question:delivery" ||
		reopened[0].PreviousDecision.AnswerText != "" ||
		len(reopened[0].InvalidatedBy) != 1 ||
		reopened[0].InvalidatedBy[0].QuestionRef != "intake-question:audience" ||
		reopened[0].InvalidatedBy[0].Revision != 3 {
		t.Fatalf("reopened decisions = %+v", reopened)
	}
}

func TestFreeTextChoiceRequiresExplicitCompatibleOption(t *testing.T) {
	tests := []struct {
		name   string
		choice Choice
		code   ErrorCode
	}{
		{
			name: "missing answer",
			choice: Choice{
				QuestionRef: "intake-question:audience",
				OptionRef:   "intake-option:audience-other",
			},
			code: ErrorAnswerTextRequired,
		},
		{
			name: "blank answer",
			choice: Choice{
				QuestionRef: "intake-question:audience",
				OptionRef:   "intake-option:audience-other",
				AnswerText:  " \n\t ",
			},
			code: ErrorAnswerTextRequired,
		},
		{
			name: "control character",
			choice: Choice{
				QuestionRef: "intake-question:audience",
				OptionRef:   "intake-option:audience-other",
				AnswerText:  "válido\x00oculto",
			},
			code: ErrorAnswerTextRequired,
		},
		{
			name: "answer on closed option",
			choice: Choice{
				QuestionRef: "intake-question:audience",
				OptionRef:   "intake-option:audience-team",
				AnswerText:  "texto no autorizado",
			},
			code: ErrorAnswerTextForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := mustState(t, 1)
			_, err := Apply(state, Change{
				StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
				Issues:    []Issue{audienceGap()},
				Questions: []Question{freeTextAudienceQuestion()},
				Choices:   []Choice{test.choice},
			})
			if ErrorCodeOf(err) != test.code {
				t.Fatalf("error=%v code=%q want=%q", err, ErrorCodeOf(err), test.code)
			}
			if state.Revision() != 1 || len(state.Decisions()) != 0 {
				t.Fatalf("failed free-text mutation changed state: %+v", state)
			}
		})
	}
}

func TestRecommendedOptionCannotRequireUserText(t *testing.T) {
	state := mustState(t, 1)
	question := freeTextAudienceQuestion()
	question.Options[0].Recommended = false
	question.Options[1].Recommended = true
	_, err := Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues: []Issue{audienceGap()}, Questions: []Question{question},
	})
	if ErrorCodeOf(err) != ErrorInvalidArgument {
		t.Fatalf("recommended free-text option error=%v", err)
	}
}

func TestFreeTextContractCountsUnicodeRunesAt4096Boundary(t *testing.T) {
	valid := strings.Repeat("á", MaxAnswerTextRunes)
	tooLong := valid + "界"
	if !ValidAnswerText(valid) {
		t.Fatal("4096 Unicode runes rejected")
	}
	if ValidAnswerText(tooLong) {
		t.Fatal("4097 Unicode runes accepted")
	}

	state := mustState(t, 1)
	applied, err := Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues:    []Issue{audienceGap()},
		Questions: []Question{freeTextAudienceQuestion()},
		Choices: []Choice{{
			QuestionRef: "intake-question:audience",
			OptionRef:   "intake-option:audience-other",
			AnswerText:  valid,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	decision, found := applied.CurrentDecision("intake-question:audience")
	if !found || decision.AnswerText != valid {
		t.Fatalf("boundary decision length=%d found=%v",
			len([]rune(decision.AnswerText)), found)
	}

	_, err = Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues:    []Issue{audienceGap()},
		Questions: []Question{freeTextAudienceQuestion()},
		Choices: []Choice{{
			QuestionRef: "intake-question:audience",
			OptionRef:   "intake-option:audience-other",
			AnswerText:  tooLong,
		}},
	})
	if ErrorCodeOf(err) != ErrorAnswerTextRequired {
		t.Fatalf("4097-rune error=%v", err)
	}
}

func freeTextAudienceQuestion() Question {
	return Question{
		Ref:         "intake-question:audience",
		DerivedFrom: []IssueRef{"intake-issue:audience-gap"},
		PromptKey:   "intake.question.audience.prompt",
		WhyKey:      "intake.question.audience.why",
		Options: []Option{
			{
				Ref:          "intake-option:audience-team",
				LabelKey:     "intake.option.audience.team.label",
				RationaleKey: "intake.option.audience.team.rationale",
				Recommended:  true,
			},
			{
				Ref:          "intake-option:audience-other",
				LabelKey:     "intake.option.audience.other.label",
				RationaleKey: "intake.option.audience.other.rationale",
				AcceptsText:  true,
			},
		},
	}
}
