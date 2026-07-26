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

func TestQuestionRevisionRejectsDifferentValidCausalDerivation(t *testing.T) {
	creator, err := NewDerivationIdentity(
		"orquesta.test.questions",
		"v1",
		strings.Repeat("d", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	substitute, err := NewDerivationIdentity(
		"orquesta.test.questions",
		"v2",
		strings.Repeat("e", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := Apply(mustState(t, 3), Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues:     []Issue{audienceGap()},
		Questions:  []Question{audienceQuestion(true, false)},
		Derivation: creator,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = Apply(initial, Change{
		StateRef: testStateRef, ExpectedRevision: initial.Revision(),
		Origin: OriginForm, Derivation: substitute,
		QuestionRevisions: []Question{audienceQuestion(false, true)},
	})
	if ErrorCodeOf(err) != ErrorInvalidArgument {
		t.Fatalf("substituted derivation err=%v", err)
	}
	if initial.HasQuestionRevisions() || len(initial.QuestionVersions()) != 1 {
		t.Fatalf("substitution mutated state: %+v", initial)
	}

	_, err = Apply(initial, Change{
		StateRef: testStateRef, ExpectedRevision: initial.Revision(),
		Origin: OriginForm, Derivation: substitute,
		QuestionRetirements: []QuestionRef{"intake-question:audience"},
	})
	if ErrorCodeOf(err) != ErrorInvalidArgument {
		t.Fatalf("substituted retirement derivation err=%v", err)
	}
	retired, err := Apply(initial, Change{
		StateRef: testStateRef, ExpectedRevision: initial.Revision(),
		Origin: OriginForm, Derivation: creator,
		QuestionRetirements: []QuestionRef{"intake-question:audience"},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Apply(retired, Change{
		StateRef: testStateRef, ExpectedRevision: retired.Revision(),
		Origin: OriginChat, Derivation: substitute,
		QuestionRevisions: []Question{audienceQuestion(true, false)},
	})
	if ErrorCodeOf(err) != ErrorInvalidArgument {
		t.Fatalf("substituted restoration derivation err=%v", err)
	}
}

func TestDerivedQuestionRetirementAndRestorationAreOneAppendOnlyVersionChain(
	t *testing.T,
) {
	identity, err := NewDerivationIdentity(
		"orquesta.test.questions",
		"v1",
		strings.Repeat("f", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	root := testQuestion("root", "root", nil)
	child := testQuestion("child", "child", []QuestionRef{root.Ref})
	initial, err := Apply(mustState(t, 3), Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues:    []Issue{testIssue("root"), testIssue("child")},
		Questions: []Question{root, child},
		Choices: []Choice{
			{
				QuestionRef: root.Ref,
				OptionRef:   root.Options[0].Ref,
			},
			{
				QuestionRef: child.Ref,
				OptionRef:   child.Options[0].Ref,
			},
		},
		Derivation: identity,
	})
	if err != nil {
		t.Fatal(err)
	}
	retired, err := Apply(initial, Change{
		StateRef: testStateRef, ExpectedRevision: initial.Revision(),
		Origin: OriginForm, Derivation: identity,
		QuestionRetirements: []QuestionRef{child.Ref},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, found := retired.CurrentDecision(child.Ref); found {
		t.Fatal("retired question retained a current decision")
	}
	for _, question := range retired.Questions() {
		if question.Ref == child.Ref {
			t.Fatal("retired question remained active")
		}
	}
	versions := retired.QuestionVersions()
	latest := versions[len(versions)-1]
	if !latest.Retired || latest.Question.Ref != child.Ref ||
		latest.ReplacesRevision != initial.Revision() ||
		latest.Revision != retired.Revision() {
		t.Fatalf("retirement tombstone=%+v", latest)
	}
	history := retired.History()
	if history[len(history)-1].QuestionsRetired != 1 ||
		history[len(history)-1].QuestionsRevised != 0 ||
		history[len(history)-1].QuestionRound !=
			history[len(history)-2].QuestionRound {
		t.Fatalf("retirement history=%+v", history)
	}
	if len(initial.Questions()) != 2 ||
		initial.QuestionVersions()[len(initial.QuestionVersions())-1].Retired {
		t.Fatal("retirement rewrote prior immutable state")
	}

	restored, err := Apply(retired, Change{
		StateRef: testStateRef, ExpectedRevision: retired.Revision(),
		Origin: OriginChat, Derivation: identity,
		QuestionRevisions: []Question{child},
	})
	if err != nil {
		t.Fatal(err)
	}
	active, found := questionByRef(restored.Questions(), child.Ref)
	if !found || !questionEqual(active, child) {
		t.Fatalf("restored active question=%+v found=%t", active, found)
	}
	versions = restored.QuestionVersions()
	latest = versions[len(versions)-1]
	if latest.Retired ||
		latest.ReplacesRevision != retired.Revision() ||
		latest.Revision != restored.Revision() {
		t.Fatalf("restoration version=%+v", latest)
	}
	if _, found := restored.CurrentDecision(child.Ref); found {
		t.Fatal("historical decision became current after restoration")
	}
}

func questionByRef(
	questions []Question,
	ref QuestionRef,
) (Question, bool) {
	for _, question := range questions {
		if question.Ref == ref {
			return question, true
		}
	}
	return Question{}, false
}

func TestQuestionRetirementRejectsDanglingActiveDependentsAtomically(
	t *testing.T,
) {
	identity, err := NewDerivationIdentity(
		"orquesta.test.questions",
		"v1",
		strings.Repeat("9", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	root := testQuestion("root", "root", nil)
	child := testQuestion("child", "child", []QuestionRef{root.Ref})
	initial, err := Apply(mustState(t, 3), Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues:     []Issue{testIssue("root"), testIssue("child")},
		Questions:  []Question{root, child},
		Derivation: identity,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Apply(initial, Change{
		StateRef: testStateRef, ExpectedRevision: initial.Revision(),
		Origin: OriginForm, Derivation: identity,
		QuestionRetirements: []QuestionRef{root.Ref},
	})
	if ErrorCodeOf(err) != ErrorQuestionNotFound ||
		len(initial.Questions()) != 2 ||
		len(initial.QuestionVersions()) != 2 ||
		len(initial.History()) != 1 {
		t.Fatalf("dangling retirement err=%v state=%+v", err, initial)
	}
}
