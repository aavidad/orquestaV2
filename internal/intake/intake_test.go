package intake

import (
	"errors"
	"reflect"
	"testing"
)

const testStateRef Ref = "intake:test"

func TestChatAndFormMutateOneVersionedState(t *testing.T) {
	state := mustState(t, 3)
	chat, err := Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues:    []Issue{audienceGap()},
		Questions: []Question{audienceQuestion(true, false)},
	})
	if err != nil {
		t.Fatal(err)
	}
	form, err := Apply(chat, Change{
		StateRef: testStateRef, ExpectedRevision: 2, Origin: OriginForm,
		Choices: []Choice{{
			QuestionRef: "intake-question:audience",
			OptionRef:   "intake-option:audience-personal",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if state.Revision() != 1 || chat.Revision() != 2 || form.Revision() != 3 ||
		state.Ref() != chat.Ref() || chat.Ref() != form.Ref() || form.Schema() != StateSchema {
		t.Fatalf("identity/revisions = %q %d/%d/%d schema=%q",
			form.Ref(), state.Revision(), chat.Revision(), form.Revision(), form.Schema())
	}
	history := form.History()
	if len(history) != 2 || history[0].Origin != OriginChat || history[1].Origin != OriginForm ||
		history[0].Revision != 2 || history[1].Revision != 3 {
		t.Fatalf("history = %+v", history)
	}
	decision, ok := form.CurrentDecision("intake-question:audience")
	if !ok || decision.Choice != "intake-option:audience-personal" ||
		decision.Recommendation != "intake-option:audience-team" ||
		decision.RecommendationRationale != "intake.option.audience.team.rationale" ||
		decision.Origin != OriginForm || decision.Revision != 3 {
		t.Fatalf("decision = %+v, found=%v", decision, ok)
	}
}

func TestQuestionMustDeriveFromIssueAndHaveExactlyOneRecommendation(t *testing.T) {
	tests := []struct {
		name     string
		question Question
		code     ErrorCode
	}{
		{
			name: "not derived",
			question: func() Question {
				value := audienceQuestion(true, false)
				value.DerivedFrom = nil
				return value
			}(),
			code: ErrorIssueNotFound,
		},
		{name: "zero recommendations", question: audienceQuestion(false, false), code: ErrorRecommendationCount},
		{name: "multiple recommendations", question: audienceQuestion(true, true), code: ErrorRecommendationCount},
		{
			name: "missing rationale key",
			question: func() Question {
				value := audienceQuestion(true, false)
				value.Options[0].RationaleKey = ""
				return value
			}(),
			code: ErrorMessageKeyInvalid,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := mustState(t, 2)
			_, err := Apply(state, Change{
				StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
				Issues: []Issue{audienceGap()}, Questions: []Question{test.question},
			})
			if ErrorCodeOf(err) != test.code {
				t.Fatalf("error = %v, code=%q want=%q", err, ErrorCodeOf(err), test.code)
			}
			if state.Revision() != 1 || len(state.Issues()) != 0 || len(state.Questions()) != 0 ||
				len(state.History()) != 0 {
				t.Fatalf("failed mutation changed state: %+v", state)
			}
		})
	}
}

func TestApplyRejectsStaleInvalidRefsAndChannelCreationWithoutPartialMutation(t *testing.T) {
	state := mustState(t, 2)
	valid := Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues: []Issue{audienceGap()}, Questions: []Question{audienceQuestion(true, false)},
	}
	updated, err := Apply(state, valid)
	if err != nil {
		t.Fatal(err)
	}

	stale := valid
	stale.Origin = OriginForm
	_, err = Apply(updated, stale)
	if ErrorCodeOf(err) != ErrorRevisionConflict {
		t.Fatalf("stale error = %v", err)
	}
	if updated.Revision() != 2 || len(updated.History()) != 1 {
		t.Fatalf("stale mutation changed current state: %+v", updated)
	}

	invalid := valid
	invalid.StateRef = "form:private-copy"
	_, err = Apply(state, invalid)
	if ErrorCodeOf(err) != ErrorInvalidRef {
		t.Fatalf("invalid ref error = %v", err)
	}

	_, err = Apply(State{}, valid)
	if ErrorCodeOf(err) != ErrorChannelStateCreation {
		t.Fatalf("channel creation error = %v", err)
	}
	var domainErr *DomainError
	if !errors.As(err, &domainErr) || domainErr.Field != "state" {
		t.Fatalf("typed error = %#v", err)
	}
}

func TestRoundPolicyIsExplicitAndSharedByBothOrigins(t *testing.T) {
	if _, err := NewState(testStateRef, Policy{}); ErrorCodeOf(err) != ErrorInvalidArgument {
		t.Fatalf("zero policy error = %v", err)
	}
	state := mustState(t, 1)
	first, err := Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues: []Issue{audienceGap()}, Questions: []Question{audienceQuestion(true, false)},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Apply(first, Change{
		StateRef: testStateRef, ExpectedRevision: 2, Origin: OriginForm,
		Issues: []Issue{{
			Ref: "intake-issue:access-conflict", Kind: IssueContradiction,
			Field: "access", DetailKey: "intake.issue.access.contradiction",
		}},
		Questions: []Question{{
			Ref: "intake-question:access", DerivedFrom: []IssueRef{"intake-issue:access-conflict"},
			PromptKey: "intake.question.access.prompt", WhyKey: "intake.question.access.why",
			Options: []Option{
				{Ref: "intake-option:access-local", LabelKey: "intake.option.access.local.label", RationaleKey: "intake.option.access.local.rationale", Recommended: true},
				{Ref: "intake-option:access-oidc", LabelKey: "intake.option.access.oidc.label", RationaleKey: "intake.option.access.oidc.rationale"},
			},
		}},
	})
	if ErrorCodeOf(err) != ErrorRoundLimit {
		t.Fatalf("shared round limit error = %v", err)
	}
	if first.QuestionRounds() != 1 || first.Revision() != 2 || len(first.Issues()) != 1 {
		t.Fatalf("round-limit failure changed state: %+v", first)
	}
}

func TestStateAccessorsReturnDefensiveCopies(t *testing.T) {
	state := mustState(t, 2)
	state, err := Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues: []Issue{audienceGap()}, Questions: []Question{audienceQuestion(true, false)},
	})
	if err != nil {
		t.Fatal(err)
	}
	issues, questions, history := state.Issues(), state.Questions(), state.History()
	issues[0].Field = "changed"
	questions[0].DerivedFrom[0] = "intake-issue:changed"
	questions[0].Options[0].Ref = "intake-option:changed"
	history[0].Origin = OriginForm
	if state.Issues()[0].Field != "audience" ||
		state.Questions()[0].DerivedFrom[0] != "intake-issue:audience-gap" ||
		state.Questions()[0].Options[0].Ref != "intake-option:audience-team" ||
		state.History()[0].Origin != OriginChat {
		t.Fatal("caller mutated immutable state through an accessor")
	}
}

func TestSnapshotRoundTripIsCompleteDefensiveAndContinuesRevisionSequence(t *testing.T) {
	state := stateWithDecision(t)
	exported := state.Snapshot()
	restored, err := Restore(exported)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored.Snapshot(), state.Snapshot()) {
		t.Fatalf("round trip mismatch:\nrestored=%+v\nwant=%+v", restored.Snapshot(), state.Snapshot())
	}

	exported.Issues[0].Field = "mutated"
	exported.Questions[0].DerivedFrom[0] = "intake-issue:mutated"
	exported.Questions[0].Options[0].Ref = "intake-option:mutated"
	exported.Decisions[0].Choice = "intake-option:mutated"
	exported.History[0].Origin = OriginForm
	if state.Issues()[0].Field != "audience" ||
		restored.Questions()[0].DerivedFrom[0] != "intake-issue:audience-gap" ||
		restored.Questions()[0].Options[0].Ref != "intake-option:audience-team" ||
		restored.Decisions()[0].Choice != "intake-option:audience-personal" ||
		restored.History()[0].Origin != OriginChat {
		t.Fatal("snapshot or restored state aliases caller-owned memory")
	}

	next, err := Apply(restored, Change{
		StateRef: testStateRef, ExpectedRevision: 3, Origin: OriginChat,
		Choices: []Choice{{
			QuestionRef: "intake-question:audience",
			OptionRef:   "intake-option:audience-team",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if next.Revision() != 4 || len(next.History()) != 3 || len(next.Decisions()) != 2 {
		t.Fatalf("restored sequence did not continue: %+v", next.Snapshot())
	}
	if _, err = Apply(restored, Change{
		StateRef: testStateRef, ExpectedRevision: 2, Origin: OriginChat,
		Choices: []Choice{{
			QuestionRef: "intake-question:audience",
			OptionRef:   "intake-option:audience-team",
		}},
	}); ErrorCodeOf(err) != ErrorRevisionConflict {
		t.Fatalf("restored state accepted stale revision: %v", err)
	}
}

func TestRestoreRejectsImpossibleSnapshots(t *testing.T) {
	valid := stateWithDecision(t).Snapshot()
	tests := []struct {
		name   string
		mutate func(*Snapshot)
	}{
		{name: "schema", mutate: func(value *Snapshot) { value.Schema = "orquesta.intake.state.v2" }},
		{name: "ref", mutate: func(value *Snapshot) { value.Ref = "form:private" }},
		{name: "policy", mutate: func(value *Snapshot) { value.Policy.MaxQuestionRounds = 0 }},
		{name: "revision without history", mutate: func(value *Snapshot) { value.Revision++ }},
		{name: "round total", mutate: func(value *Snapshot) { value.QuestionRounds = 0 }},
		{name: "round beyond policy", mutate: func(value *Snapshot) { value.QuestionRounds = 4 }},
		{name: "history revision", mutate: func(value *Snapshot) { value.History[0].Revision = 3 }},
		{name: "history origin", mutate: func(value *Snapshot) { value.History[0].Origin = "private" }},
		{name: "negative count", mutate: func(value *Snapshot) { value.History[0].IssuesAdded = -1 }},
		{name: "unaccounted issue", mutate: func(value *Snapshot) { value.History[0].IssuesAdded = 0 }},
		{name: "question before issue", mutate: func(value *Snapshot) {
			value.History[0].IssuesAdded = 0
			value.History[1].IssuesAdded = 1
		}},
		{name: "invalid issue", mutate: func(value *Snapshot) { value.Issues[0].Kind = "unknown" }},
		{name: "duplicate question ref", mutate: func(value *Snapshot) {
			value.Questions = append(value.Questions, cloneQuestions(value.Questions)...)
			value.History[1].QuestionsAdded = 1
		}},
		{name: "decision unknown choice", mutate: func(value *Snapshot) {
			value.Decisions[0].Choice = "intake-option:unknown"
		}},
		{name: "decision false recommendation", mutate: func(value *Snapshot) {
			value.Decisions[0].Recommendation = "intake-option:audience-personal"
		}},
		{name: "decision false rationale", mutate: func(value *Snapshot) {
			value.Decisions[0].RecommendationRationale = "intake.option.false.rationale"
		}},
		{name: "decision wrong origin", mutate: func(value *Snapshot) {
			value.Decisions[0].Origin = OriginChat
		}},
		{name: "decision wrong revision", mutate: func(value *Snapshot) {
			value.Decisions[0].Revision = 2
		}},
		{name: "unaccounted decision", mutate: func(value *Snapshot) {
			value.History[1].ChoicesRecorded = 0
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := cloneSnapshot(valid)
			test.mutate(&candidate)
			if state, err := Restore(candidate); err == nil {
				t.Fatalf("impossible snapshot restored: %+v", state.Snapshot())
			}
		})
	}
}

func stateWithDecision(t *testing.T) State {
	t.Helper()
	state := mustState(t, 3)
	state, err := Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues:    []Issue{audienceGap()},
		Questions: []Question{audienceQuestion(true, false)},
	})
	if err != nil {
		t.Fatal(err)
	}
	state, err = Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 2, Origin: OriginForm,
		Choices: []Choice{{
			QuestionRef: "intake-question:audience",
			OptionRef:   "intake-option:audience-personal",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func cloneSnapshot(value Snapshot) Snapshot {
	value.Issues = cloneIssues(value.Issues)
	value.Questions = cloneQuestions(value.Questions)
	value.Decisions = append([]Decision(nil), value.Decisions...)
	value.History = append([]Mutation(nil), value.History...)
	return value
}

func mustState(t *testing.T, maxRounds uint32) State {
	t.Helper()
	state, err := NewState(testStateRef, Policy{MaxQuestionRounds: maxRounds})
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func audienceGap() Issue {
	return Issue{
		Ref: "intake-issue:audience-gap", Kind: IssueGap,
		Field: "audience", DetailKey: "intake.issue.audience.missing",
	}
}

func audienceQuestion(recommendTeam, recommendPersonal bool) Question {
	return Question{
		Ref: "intake-question:audience", DerivedFrom: []IssueRef{"intake-issue:audience-gap"},
		PromptKey: "intake.question.audience.prompt", WhyKey: "intake.question.audience.why",
		Options: []Option{
			{
				Ref: "intake-option:audience-team", LabelKey: "intake.option.audience.team.label",
				RationaleKey: "intake.option.audience.team.rationale", Recommended: recommendTeam,
			},
			{
				Ref: "intake-option:audience-personal", LabelKey: "intake.option.audience.personal.label",
				RationaleKey: "intake.option.audience.personal.rationale", Recommended: recommendPersonal,
			},
		},
	}
}
