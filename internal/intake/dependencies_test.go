package intake

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDecisionChangeReopensDeclaredDependentsTransitively(t *testing.T) {
	state := mustState(t, 3)
	issues := []Issue{testIssue("root"), testIssue("child"), testIssue("leaf")}
	root := testQuestion("root", "root", nil)
	child := testQuestion("child", "child", []QuestionRef{root.Ref})
	leaf := testQuestion("leaf", "leaf", []QuestionRef{child.Ref})

	// Deliberately declare and answer out of dependency order. Refs, not slice
	// order or field-name heuristics, define the graph.
	state, err := Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues: issues, Questions: []Question{leaf, child, root},
		Choices: []Choice{
			{QuestionRef: leaf.Ref, OptionRef: leaf.Options[0].Ref},
			{QuestionRef: root.Ref, OptionRef: root.Options[0].Ref},
			{QuestionRef: child.Ref, OptionRef: child.Options[0].Ref},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	questions := state.Questions()
	questions[0].DependsOn[0] = root.Ref
	if state.Questions()[0].DependsOn[0] != child.Ref {
		t.Fatal("caller mutated dependency graph through Questions accessor")
	}

	changed, err := Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 2, Origin: OriginForm,
		Choices: []Choice{{
			QuestionRef: root.Ref,
			OptionRef:   root.Options[1].Ref,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision, found := changed.CurrentDecision(root.Ref); !found ||
		decision.Choice != root.Options[1].Ref {
		t.Fatalf("root decision = %+v found=%v", decision, found)
	}
	for _, ref := range []QuestionRef{child.Ref, leaf.Ref} {
		if decision, found := changed.CurrentDecision(ref); found {
			t.Fatalf("%s remained current after root changed: %+v", ref, decision)
		}
	}
	reopened := changed.ReopenedDecisions()
	if len(reopened) != 2 ||
		reopened[0].QuestionRef != leaf.Ref ||
		reopened[1].QuestionRef != child.Ref {
		t.Fatalf("reopened order/state = %+v", reopened)
	}
	for _, value := range reopened {
		if len(value.InvalidatedBy) != 1 ||
			value.InvalidatedBy[0] != (DecisionChange{QuestionRef: root.Ref, Revision: 3}) {
			t.Fatalf("reopen cause for %s = %+v", value.QuestionRef, value.InvalidatedBy)
		}
	}

	childReanswered, err := Apply(changed, Change{
		StateRef: testStateRef, ExpectedRevision: 3, Origin: OriginChat,
		Choices: []Choice{{
			QuestionRef: child.Ref,
			OptionRef:   child.Options[0].Ref,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, found := childReanswered.CurrentDecision(child.Ref); !found {
		t.Fatal("reanswered child is not current")
	}
	if _, found := childReanswered.CurrentDecision(leaf.Ref); found {
		t.Fatal("leaf became current without being reanswered")
	}

	fullyReanswered, err := Apply(childReanswered, Change{
		StateRef: testStateRef, ExpectedRevision: 4, Origin: OriginForm,
		Choices: []Choice{{
			QuestionRef: leaf.Ref,
			OptionRef:   leaf.Options[0].Ref,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	reaffirmed, err := Apply(fullyReanswered, Change{
		StateRef: testStateRef, ExpectedRevision: 5, Origin: OriginChat,
		Choices: []Choice{{
			QuestionRef: root.Ref,
			OptionRef:   root.Options[1].Ref,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := reaffirmed.ReopenedDecisions(); len(got) != 0 {
		t.Fatalf("identical root choice reopened dependents: %+v", got)
	}
	for _, ref := range []QuestionRef{root.Ref, child.Ref, leaf.Ref} {
		if _, found := reaffirmed.CurrentDecision(ref); !found {
			t.Fatalf("%s not current after identical reaffirmation", ref)
		}
	}

	atomic, err := Apply(reaffirmed, Change{
		StateRef: testStateRef, ExpectedRevision: 6, Origin: OriginForm,
		Choices: []Choice{
			{QuestionRef: root.Ref, OptionRef: root.Options[0].Ref},
			{QuestionRef: child.Ref, OptionRef: child.Options[0].Ref},
			{QuestionRef: leaf.Ref, OptionRef: leaf.Options[0].Ref},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := atomic.ReopenedDecisions(); len(got) != 0 {
		t.Fatalf("same-change dependent answers stayed reopened: %+v", got)
	}
	for _, ref := range []QuestionRef{root.Ref, child.Ref, leaf.Ref} {
		decision, found := atomic.CurrentDecision(ref)
		if !found || decision.Revision != 7 {
			t.Fatalf("%s atomic decision = %+v found=%v", ref, decision, found)
		}
	}
}

func TestQuestionDependenciesRejectUnknownDuplicateAndCyclesAtomically(t *testing.T) {
	tests := []struct {
		name      string
		questions []Question
		code      ErrorCode
	}{
		{
			name: "unknown",
			questions: []Question{testQuestion(
				"child",
				"child",
				[]QuestionRef{"intake-question:missing"},
			)},
			code: ErrorQuestionNotFound,
		},
		{
			name: "duplicate",
			questions: []Question{
				testQuestion("root", "root", nil),
				testQuestion("child", "child", []QuestionRef{
					"intake-question:root",
					"intake-question:root",
				}),
			},
			code: ErrorDuplicateRef,
		},
		{
			name: "self cycle",
			questions: []Question{testQuestion(
				"root",
				"root",
				[]QuestionRef{"intake-question:root"},
			)},
			code: ErrorDependencyCycle,
		},
		{
			name: "two node cycle",
			questions: []Question{
				testQuestion("root", "root", []QuestionRef{"intake-question:child"}),
				testQuestion("child", "child", []QuestionRef{"intake-question:root"}),
			},
			code: ErrorDependencyCycle,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := mustState(t, 2)
			issues := make([]Issue, 0, len(test.questions))
			for _, question := range test.questions {
				issues = append(issues, testIssue(
					string(question.DerivedFrom[0])[len("intake-issue:"):],
				))
			}
			before := state
			_, err := Apply(state, Change{
				StateRef: testStateRef, ExpectedRevision: 1,
				Origin: OriginChat, Issues: issues, Questions: test.questions,
			})
			if ErrorCodeOf(err) != test.code {
				t.Fatalf("error=%v code=%q want=%q", err, ErrorCodeOf(err), test.code)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatal("invalid dependency graph mutated state")
			}
		})
	}
}

func TestDependentChoiceRequiresCurrentOrSameChangeDependency(t *testing.T) {
	state := mustState(t, 2)
	root := testQuestion("root", "root", nil)
	child := testQuestion("child", "child", []QuestionRef{root.Ref})
	state, err := Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues:    []Issue{testIssue("root"), testIssue("child")},
		Questions: []Question{root, child},
	})
	if err != nil {
		t.Fatal(err)
	}
	before := state
	_, err = Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 2, Origin: OriginForm,
		Choices: []Choice{{
			QuestionRef: child.Ref,
			OptionRef:   child.Options[0].Ref,
		}},
	})
	if ErrorCodeOf(err) != ErrorDependencyPending {
		t.Fatalf("pending dependency error = %v", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatal("pending dependency failure mutated state")
	}

	together, err := Apply(state, Change{
		StateRef: testStateRef, ExpectedRevision: 2, Origin: OriginForm,
		Choices: []Choice{
			{QuestionRef: child.Ref, OptionRef: child.Options[0].Ref},
			{QuestionRef: root.Ref, OptionRef: root.Options[0].Ref},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, found := together.CurrentDecision(child.Ref); !found {
		t.Fatal("same-change dependency did not authorize dependent choice")
	}
}

func TestQuestionDependenciesSurviveCanonicalJSONRoundTrip(t *testing.T) {
	change := Change{
		StateRef: testStateRef, ExpectedRevision: 1, Origin: OriginChat,
		Issues: []Issue{testIssue("root"), testIssue("child")},
		Questions: []Question{
			testQuestion("root", "root", nil),
			testQuestion("child", "child", []QuestionRef{"intake-question:root"}),
		},
	}
	encoded, err := json.Marshal(change)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Change
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(change, decoded) {
		t.Fatalf("dependency JSON round-trip mismatch:\nchange=%+v\ndecoded=%+v", change, decoded)
	}
}
