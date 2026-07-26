package intake

import (
	"reflect"
	"testing"
)

func TestBuildAcceptRecommendationsChangeAcceptsEveryPendingQuestionAtomically(t *testing.T) {
	state := mustState(t, 2)
	issues := []Issue{
		audienceGap(),
		testIssue("platform"),
		testIssue("storage"),
	}
	questions := []Question{
		audienceQuestion(true, false),
		testQuestion("platform", "platform", nil),
		testQuestion("storage", "storage", nil),
	}
	state, err := Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues: issues, Questions: questions,
		Choices: []Choice{{
			QuestionRef: questions[0].Ref,
			OptionRef:   questions[0].Options[1].Ref,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	change, err := BuildAcceptRecommendationsChange(state, AcceptRecommendationsRequest{
		StateRef: testStateRef, ExpectedRevision: 2,
		Origin: OriginForm, QuestionRound: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	wantChoices := []Choice{
		{QuestionRef: questions[1].Ref, OptionRef: questions[1].Options[0].Ref},
		{QuestionRef: questions[2].Ref, OptionRef: questions[2].Options[0].Ref},
	}
	if change.StateRef != testStateRef || change.ExpectedRevision != 2 ||
		change.Origin != OriginForm || !reflect.DeepEqual(change.Choices, wantChoices) ||
		len(change.Issues) != 0 || len(change.Questions) != 0 {
		t.Fatalf("compiled change = %+v", change)
	}

	accepted, err := Apply(state, change)
	if err != nil {
		t.Fatal(err)
	}
	history := accepted.History()
	if accepted.Revision() != 3 || accepted.QuestionRounds() != 1 ||
		len(history) != 2 || history[1].ChoicesRecorded != 2 ||
		history[1].QuestionsAdded != 0 || history[1].QuestionRound != 1 {
		t.Fatalf("atomic acceptance = revision %d rounds %d history %+v",
			accepted.Revision(), accepted.QuestionRounds(), history)
	}
	audience, found := accepted.CurrentDecision(questions[0].Ref)
	if !found || audience.Choice != questions[0].Options[1].Ref {
		t.Fatalf("existing user override changed: %+v found=%v", audience, found)
	}
	for _, question := range questions[1:] {
		decision, found := accepted.CurrentDecision(question.Ref)
		if !found || decision.Choice != question.Options[0].Ref ||
			decision.Recommendation != question.Options[0].Ref ||
			decision.Revision != 3 {
			t.Fatalf("recommendation not accepted for %s: %+v found=%v",
				question.Ref, decision, found)
		}
	}

	before := accepted
	_, err = BuildAcceptRecommendationsChange(accepted, AcceptRecommendationsRequest{
		StateRef: testStateRef, ExpectedRevision: 3,
		Origin: OriginChat, QuestionRound: 1,
	})
	if ErrorCodeOf(err) != ErrorRecommendationsDone || !reflect.DeepEqual(accepted, before) {
		t.Fatalf("resolved round error=%v state mutated=%v", err, !reflect.DeepEqual(accepted, before))
	}
}

func TestBuildAcceptRecommendationsChangeRejectsInvalidScopeAndTamperingAtomically(t *testing.T) {
	state := mustState(t, 2)
	state, err := Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues: []Issue{audienceGap()}, Questions: []Question{audienceQuestion(true, false)},
	})
	if err != nil {
		t.Fatal(err)
	}
	before := state
	tests := []struct {
		name    string
		request AcceptRecommendationsRequest
		code    ErrorCode
	}{
		{
			name: "stale",
			request: AcceptRecommendationsRequest{
				StateRef: testStateRef, ExpectedRevision: 1,
				Origin: OriginForm, QuestionRound: 1,
			},
			code: ErrorRevisionConflict,
		},
		{
			name: "unknown round",
			request: AcceptRecommendationsRequest{
				StateRef: testStateRef, ExpectedRevision: 2,
				Origin: OriginForm, QuestionRound: 2,
			},
			code: ErrorQuestionRound,
		},
		{
			name: "zero round",
			request: AcceptRecommendationsRequest{
				StateRef: testStateRef, ExpectedRevision: 2,
				Origin: OriginForm,
			},
			code: ErrorQuestionRound,
		},
		{
			name: "other state",
			request: AcceptRecommendationsRequest{
				StateRef: "intake:other", ExpectedRevision: 2,
				Origin: OriginForm, QuestionRound: 1,
			},
			code: ErrorStateMismatch,
		},
		{
			name: "invalid origin",
			request: AcceptRecommendationsRequest{
				StateRef: testStateRef, ExpectedRevision: 2,
				Origin: "web", QuestionRound: 1,
			},
			code: ErrorInvalidOrigin,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, gotErr := BuildAcceptRecommendationsChange(state, test.request)
			if ErrorCodeOf(gotErr) != test.code {
				t.Fatalf("error=%v code=%q want=%q", gotErr, ErrorCodeOf(gotErr), test.code)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatal("failed recommendation request mutated state")
			}
		})
	}

	change, err := BuildAcceptRecommendationsChange(state, AcceptRecommendationsRequest{
		StateRef: testStateRef, ExpectedRevision: 2,
		Origin: OriginForm, QuestionRound: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	change.Choices = append(change.Choices, Choice{
		QuestionRef: "intake-question:unknown",
		OptionRef:   "intake-option:unknown",
	})
	if _, err = Apply(state, change); ErrorCodeOf(err) != ErrorQuestionNotFound {
		t.Fatalf("tampered atomic change error = %v", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatal("failed atomic acceptance mutated state")
	}
}

func TestReemitContextIsRevisionBoundAndDoesNotMutateCausalState(t *testing.T) {
	state := mustState(t, 2)
	question := audienceQuestion(true, false)
	state, err := Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues: []Issue{audienceGap()}, Questions: []Question{question},
		Choices: []Choice{{
			QuestionRef: question.Ref,
			OptionRef:   question.Options[1].Ref,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	before := state

	context, err := ReemitContext(state, ContextRequest{
		StateRef: testStateRef, ExpectedRevision: 2,
		Origin: OriginForm, Kind: ContextHelp,
	})
	if err != nil {
		t.Fatal(err)
	}
	if context.StateRef != testStateRef || context.Revision != 2 ||
		context.Origin != OriginForm || context.Kind != ContextHelp ||
		len(context.Issues) != 1 || len(context.Questions) != 1 ||
		context.Questions[0].CurrentDecision == nil ||
		context.Questions[0].CurrentDecision.Choice != question.Options[1].Ref {
		t.Fatalf("context = %+v", context)
	}
	if !reflect.DeepEqual(state, before) || state.QuestionRounds() != 1 ||
		len(state.History()) != 1 {
		t.Fatal("context re-emission changed revision, rounds, or causal history")
	}

	context.Issues[0].Field = "mutated"
	context.Questions[0].Question.DerivedFrom[0] = "intake-issue:mutated"
	context.Questions[0].Question.Options[0].Ref = "intake-option:mutated"
	context.Questions[0].CurrentDecision.Choice = "intake-option:mutated"
	again, err := ReemitContext(state, ContextRequest{
		StateRef: testStateRef, ExpectedRevision: 2,
		Origin: OriginChat, Kind: ContextClarification,
		QuestionRefs: []QuestionRef{question.Ref},
	})
	if err != nil {
		t.Fatal(err)
	}
	if again.Issues[0].Field != "audience" ||
		again.Questions[0].Question.DerivedFrom[0] != audienceGap().Ref ||
		again.Questions[0].Question.Options[0].Ref != question.Options[0].Ref ||
		again.Questions[0].CurrentDecision.Choice != question.Options[1].Ref {
		t.Fatal("caller mutated state through re-emitted context")
	}

	for _, test := range []struct {
		name    string
		request ContextRequest
		code    ErrorCode
	}{
		{
			name: "stale",
			request: ContextRequest{
				StateRef: testStateRef, ExpectedRevision: 1,
				Origin: OriginChat, Kind: ContextHelp,
			},
			code: ErrorRevisionConflict,
		},
		{
			name: "invalid kind",
			request: ContextRequest{
				StateRef: testStateRef, ExpectedRevision: 2,
				Origin: OriginChat, Kind: "explain",
			},
			code: ErrorInvalidArgument,
		},
		{
			name: "unknown question",
			request: ContextRequest{
				StateRef: testStateRef, ExpectedRevision: 2,
				Origin: OriginChat, Kind: ContextClarification,
				QuestionRefs: []QuestionRef{"intake-question:unknown"},
			},
			code: ErrorQuestionNotFound,
		},
		{
			name: "duplicate question",
			request: ContextRequest{
				StateRef: testStateRef, ExpectedRevision: 2,
				Origin: OriginChat, Kind: ContextClarification,
				QuestionRefs: []QuestionRef{question.Ref, question.Ref},
			},
			code: ErrorDuplicateRef,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, gotErr := ReemitContext(state, test.request)
			if ErrorCodeOf(gotErr) != test.code {
				t.Fatalf("error=%v code=%q want=%q", gotErr, ErrorCodeOf(gotErr), test.code)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatal("failed context request mutated causal state")
			}
		})
	}
}

func testIssue(name string) Issue {
	return Issue{
		Ref:       IssueRef("intake-issue:" + name),
		Kind:      IssueGap,
		Field:     name,
		DetailKey: MessageKey("intake.issue." + name),
	}
}

func testQuestion(
	name string,
	issue string,
	dependsOn []QuestionRef,
) Question {
	return Question{
		Ref:         QuestionRef("intake-question:" + name),
		DerivedFrom: []IssueRef{IssueRef("intake-issue:" + issue)},
		DependsOn:   append([]QuestionRef(nil), dependsOn...),
		PromptKey:   MessageKey("intake.question." + name + ".prompt"),
		WhyKey:      MessageKey("intake.question." + name + ".why"),
		Options: []Option{
			{
				Ref:          OptionRef("intake-option:" + name + "-recommended"),
				LabelKey:     MessageKey("intake.option." + name + ".recommended.label"),
				RationaleKey: MessageKey("intake.option." + name + ".recommended.rationale"),
				Recommended:  true,
			},
			{
				Ref:          OptionRef("intake-option:" + name + "-alternate"),
				LabelKey:     MessageKey("intake.option." + name + ".alternate.label"),
				RationaleKey: MessageKey("intake.option." + name + ".alternate.rationale"),
			},
		},
	}
}
