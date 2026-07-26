package application

import (
	"strings"
	"testing"

	"orquesta/internal/intake"
	"orquesta/internal/wizard/gaps"
)

func TestWizardGapsRetiresAndRestoresQuestionFromExactCausalEdge(t *testing.T) {
	evaluator := wizardGapsV1Evaluator(t)
	empty, err := evaluator.Evaluate(gaps.Input{})
	if err != nil {
		t.Fatal(err)
	}
	team, err := evaluator.Evaluate(gaps.Input{
		Selections: []gaps.Selection{{
			Dimension: gaps.DimensionU1,
			Option:    wizardOptionRef(gaps.DimensionU1, "team"),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	parent := mustWizardDimensionQuestion(t, empty, gaps.DimensionU1)
	child := mustWizardDimensionQuestion(t, team, gaps.DimensionT1)
	issues := wizardIssuesForQuestions(
		t,
		[]gaps.Result{empty, team},
		[]gaps.Question{parent, child},
	)
	state, err := intake.NewState(
		"intake:wizard-retirement",
		intake.Policy{MaxQuestionRounds: 6},
	)
	if err != nil {
		t.Fatal(err)
	}
	state, err = intake.Apply(state, intake.Change{
		StateRef: state.Ref(), ExpectedRevision: state.Revision(),
		Origin: intake.OriginForm, Derivation: evaluator.Identity(),
		Issues: issues,
		Questions: []intake.Question{
			parent.IntakeQuestion(),
			child.IntakeQuestion(),
		},
		Choices: []intake.Choice{
			{
				QuestionRef: intake.QuestionRef(parent.Ref()),
				OptionRef:   intake.OptionRef(wizardOptionRef(gaps.DimensionU1, "team")),
			},
			{
				QuestionRef: intake.QuestionRef(child.Ref()),
				OptionRef:   intake.OptionRef(wizardRecommendedOption(t, child).Ref()),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	state, err = intake.Apply(state, intake.Change{
		StateRef: state.Ref(), ExpectedRevision: state.Revision(),
		Origin: intake.OriginChat,
		Choices: []intake.Choice{{
			QuestionRef: intake.QuestionRef(parent.Ref()),
			OptionRef:   intake.OptionRef(wizardOptionRef(gaps.DimensionU1, "personal")),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	evaluated, err := evaluateWizardGapsDetailed(
		state,
		gaps.Facts{},
		nil,
		evaluator,
	)
	if err != nil {
		t.Fatal(err)
	}
	change, err := wizardGapsChangeWithReconciliation(
		state,
		intake.OriginChat,
		evaluator.Identity(),
		evaluated.result,
		evaluated.reconcilable,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(change.QuestionRetirements) != 1 ||
		change.QuestionRetirements[0] != intake.QuestionRef(child.Ref()) {
		t.Fatalf("retirements=%v", change.QuestionRetirements)
	}
	retired, err := intake.Apply(state, change)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := currentQuestion(
		retired,
		intake.QuestionRef(child.Ref()),
	); found {
		t.Fatal("T1 remained active after personal audience")
	}
	if _, found := retired.CurrentDecision(intake.QuestionRef(child.Ref())); found {
		t.Fatal("T1 decision remained current after retirement")
	}

	changedBack, err := intake.Apply(retired, intake.Change{
		StateRef: retired.Ref(), ExpectedRevision: retired.Revision(),
		Origin: intake.OriginForm,
		Choices: []intake.Choice{{
			QuestionRef: intake.QuestionRef(parent.Ref()),
			OptionRef:   intake.OptionRef(wizardOptionRef(gaps.DimensionU1, "team")),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	evaluated, err = evaluateWizardGapsDetailed(
		changedBack,
		gaps.Facts{},
		nil,
		evaluator,
	)
	if err != nil {
		t.Fatal(err)
	}
	change, err = wizardGapsChangeWithReconciliation(
		changedBack,
		intake.OriginForm,
		evaluator.Identity(),
		evaluated.result,
		evaluated.reconcilable,
	)
	if err != nil {
		t.Fatal(err)
	}
	restoresChild := false
	for _, revision := range change.QuestionRevisions {
		if revision.Ref == intake.QuestionRef(child.Ref()) {
			restoresChild = true
		}
	}
	addsChildAsNew := false
	for _, question := range change.Questions {
		if question.Ref == intake.QuestionRef(child.Ref()) {
			addsChildAsNew = true
		}
	}
	if !restoresChild || addsChildAsNew {
		t.Fatalf(
			"restoration revisions=%v questions=%v",
			change.QuestionRevisions,
			change.Questions,
		)
	}
	restored, err := intake.Apply(changedBack, change)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := currentQuestion(
		restored,
		intake.QuestionRef(child.Ref()),
	); !found {
		t.Fatal("T1 was not restored after team audience")
	}
}

func wizardIssuesForQuestions(
	t *testing.T,
	results []gaps.Result,
	questions []gaps.Question,
) []intake.Issue {
	t.Helper()
	refs := make(map[intake.IssueRef]struct{})
	for _, question := range questions {
		for _, ref := range question.IntakeQuestion().DerivedFrom {
			refs[ref] = struct{}{}
		}
	}
	found := make(map[intake.IssueRef]struct{})
	var result []intake.Issue
	for _, evaluation := range results {
		for _, issue := range evaluation.Issues() {
			projected := issue.IntakeIssue()
			if _, required := refs[projected.Ref]; !required {
				continue
			}
			if _, duplicate := found[projected.Ref]; duplicate {
				continue
			}
			found[projected.Ref] = struct{}{}
			result = append(result, projected)
		}
	}
	if len(found) != len(refs) {
		t.Fatalf("issues found=%d required=%d", len(found), len(refs))
	}
	return result
}

func wizardRecommendedOption(t *testing.T, question gaps.Question) gaps.Option {
	t.Helper()
	option, found := question.RecommendedOption()
	if !found {
		t.Fatalf("%s recommendation missing", question.Ref())
	}
	return option
}

func wizardOptionRef(
	dimension gaps.DimensionRef,
	option string,
) gaps.OptionRef {
	return gaps.OptionRef(
		"intake-option:wizard." + strings.ToLower(string(dimension)) + "." + option,
	)
}
