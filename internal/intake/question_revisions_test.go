package intake

import (
	"strings"
	"testing"
)

func TestDerivedQuestionRevisionPreservesVersionsAndReopensDecision(t *testing.T) {
	identity, err := NewDerivationIdentity(
		"orquesta.test.questions",
		"v1",
		strings.Repeat("a", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := Apply(mustState(t, 3), Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues:    []Issue{audienceGap()},
		Questions: []Question{audienceQuestion(true, false)},
		Choices: []Choice{{
			QuestionRef: "intake-question:audience",
			OptionRef:   "intake-option:audience-personal",
		}},
		Derivation: identity,
	})
	if err != nil {
		t.Fatal(err)
	}
	revisedQuestion := audienceQuestion(false, true)
	revised, err := Apply(initial, Change{
		StateRef: testStateRef, ExpectedRevision: initial.Revision(),
		Origin: OriginForm, QuestionRevisions: []Question{revisedQuestion},
		Derivation: identity,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, found := revised.CurrentDecision("intake-question:audience"); found {
		t.Fatal("decision from the replaced question version remained current")
	}
	reopened := revised.ReopenedDecisions()
	if len(reopened) != 1 ||
		reopened[0].QuestionRef != "intake-question:audience" ||
		len(reopened[0].InvalidatedBy) != 1 ||
		reopened[0].InvalidatedBy[0].Revision != revised.Revision() {
		t.Fatalf("reopened decision = %+v", reopened)
	}
	versions := revised.QuestionVersions()
	if len(versions) != 2 ||
		versions[0].Revision != initial.Revision() ||
		versions[0].ReplacesRevision != 0 ||
		versions[1].Revision != revised.Revision() ||
		versions[1].ReplacesRevision != initial.Revision() ||
		!questionEqual(versions[0].Question, audienceQuestion(true, false)) ||
		!questionEqual(versions[1].Question, revisedQuestion) {
		t.Fatalf("question versions = %+v", versions)
	}
	if !questionEqual(
		initial.Questions()[0],
		audienceQuestion(true, false),
	) {
		t.Fatal("reconciliation rewrote the prior immutable state")
	}
	history := revised.History()
	if len(history) != 2 || history[1].QuestionsAdded != 0 ||
		history[1].QuestionsRevised != 1 ||
		history[1].QuestionRound != history[0].QuestionRound {
		t.Fatalf("history = %+v", history)
	}
}

func TestQuestionRevisionFailsClosedWithoutCausalDerivation(t *testing.T) {
	initial, err := Apply(mustState(t, 3), Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues:    []Issue{audienceGap()},
		Questions: []Question{audienceQuestion(true, false)},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Apply(initial, Change{
		StateRef: testStateRef, ExpectedRevision: initial.Revision(),
		Origin:            OriginForm,
		QuestionRevisions: []Question{audienceQuestion(false, true)},
	})
	if ErrorCodeOf(err) != ErrorInvalidArgument ||
		initial.HasQuestionRevisions() ||
		len(initial.QuestionVersions()) != 1 {
		t.Fatalf("uncausal revision err=%v state=%+v", err, initial)
	}
}
