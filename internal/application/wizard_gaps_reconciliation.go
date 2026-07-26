package application

import (
	"orquesta/internal/intake"
	"orquesta/internal/wizard/catalog"
	"orquesta/internal/wizard/gaps"
)

type wizardGapsDecisionTransition struct {
	current     intake.Decision
	previous    intake.Decision
	hasPrevious bool
}

// wizardGapsCounterfactualReconciliation keeps request-scoped facts and packs
// fixed while replacing one latest durable decision with its causal predecessor.
// A projection is reconcilable only when that isolated decision change alters
// its payload or emission and postdates its latest immutable version.
func wizardGapsCounterfactualReconciliation(
	state intake.State,
	facts gaps.Facts,
	selections []gaps.Selection,
	packRefs []catalog.PackRef,
	evaluator wizardGapsEvaluator,
	current gaps.Result,
) (map[intake.QuestionRef]struct{}, error) {
	result := make(map[intake.QuestionRef]struct{})
	latestVersions := make(map[intake.QuestionRef]intake.QuestionVersion)
	for _, version := range state.QuestionVersions() {
		latestVersions[version.Question.Ref] = version
	}
	dimensions := make(map[intake.QuestionRef]gaps.DimensionDescriptor)
	for _, dimension := range evaluator.Dimensions() {
		dimensions[wizardDimensionQuestionRef(dimension.Ref())] = dimension
	}
	currentQuestions := wizardGapsResultQuestions(current)
	for ref, transition := range wizardGapsDecisionTransitions(state.Decisions()) {
		active, found := state.CurrentDecision(ref)
		if !found ||
			active.Choice != transition.current.Choice ||
			active.AnswerText != transition.current.AnswerText {
			continue
		}
		dimension, isDimension := dimensions[ref]
		if !isDimension {
			continue
		}
		counterfactualSelections := make(
			[]gaps.Selection,
			0,
			len(selections),
		)
		for _, selection := range selections {
			if selection.Dimension != dimension.Ref() {
				counterfactualSelections = append(
					counterfactualSelections,
					selection,
				)
			}
		}
		if transition.hasPrevious {
			counterfactualSelections = append(
				counterfactualSelections,
				gaps.Selection{
					Dimension: dimension.Ref(),
					Option:    gaps.OptionRef(transition.previous.Choice),
					FreeText:  transition.previous.AnswerText,
				},
			)
		}
		counterfactual, err := evaluator.Evaluate(gaps.Input{
			Facts: facts, Selections: counterfactualSelections,
			PackRefs: packRefs,
		})
		if err != nil {
			return nil, err
		}
		previousQuestions := wizardGapsResultQuestions(counterfactual)
		for questionRef, version := range latestVersions {
			if transition.current.Revision <= version.Revision {
				continue
			}
			actual, actualFound := currentQuestions[questionRef]
			previous, previousFound := previousQuestions[questionRef]
			if actualFound != previousFound ||
				(actualFound &&
					!wizardGapsQuestionPayloadEqual(actual, previous)) {
				result[questionRef] = struct{}{}
			}
		}
	}
	for _, reopened := range state.ReopenedDecisions() {
		result[reopened.QuestionRef] = struct{}{}
	}
	return result, nil
}

func wizardGapsDecisionTransitions(
	decisions []intake.Decision,
) map[intake.QuestionRef]wizardGapsDecisionTransition {
	latest := make(map[intake.QuestionRef]intake.Decision)
	transitions := make(
		map[intake.QuestionRef]wizardGapsDecisionTransition,
	)
	for _, decision := range decisions {
		previous, found := latest[decision.QuestionRef]
		if !found ||
			previous.Choice != decision.Choice ||
			previous.AnswerText != decision.AnswerText {
			transitions[decision.QuestionRef] = wizardGapsDecisionTransition{
				current: decision, previous: previous, hasPrevious: found,
			}
		}
		latest[decision.QuestionRef] = decision
	}
	return transitions
}

func wizardGapsResultQuestions(
	result gaps.Result,
) map[intake.QuestionRef]intake.Question {
	questions := make(map[intake.QuestionRef]intake.Question)
	for _, question := range result.Questions() {
		projected := question.IntakeQuestion()
		questions[projected.Ref] = projected
	}
	return questions
}

// wizardGapsReconcilableQuestions is the durable-edge fallback used by pure
// projection tests. Production additionally supplies the counterfactual set
// above, which proves behavior rather than trusting a broad reopened flag.
func wizardGapsReconcilableQuestions(
	state intake.State,
) map[intake.QuestionRef]struct{} {
	result := make(map[intake.QuestionRef]struct{})
	for _, reopened := range state.ReopenedDecisions() {
		result[reopened.QuestionRef] = struct{}{}
	}

	latestVersions := make(map[intake.QuestionRef]intake.QuestionVersion)
	for _, version := range state.QuestionVersions() {
		latestVersions[version.Question.Ref] = version
	}
	latestDecisionChanges := wizardGapsLatestDecisionChanges(state.Decisions())
	dependencies := make(
		map[intake.QuestionRef]map[intake.QuestionRef]struct{},
		len(latestVersions),
	)
	var collectDependencies func(intake.QuestionRef) map[intake.QuestionRef]struct{}
	collectDependencies = func(
		ref intake.QuestionRef,
	) map[intake.QuestionRef]struct{} {
		if known, found := dependencies[ref]; found {
			return known
		}
		known := make(map[intake.QuestionRef]struct{})
		dependencies[ref] = known
		for _, direct := range latestVersions[ref].Question.DependsOn {
			known[direct] = struct{}{}
			for transitive := range collectDependencies(direct) {
				known[transitive] = struct{}{}
			}
		}
		return known
	}
	for ref, version := range latestVersions {
		for dependency := range collectDependencies(ref) {
			if latestDecisionChanges[dependency] > version.Revision {
				result[ref] = struct{}{}
				break
			}
		}
	}
	return result
}

func wizardGapsLatestDecisionChanges(
	decisions []intake.Decision,
) map[intake.QuestionRef]intake.Revision {
	latest := make(map[intake.QuestionRef]intake.Decision)
	changes := make(map[intake.QuestionRef]intake.Revision)
	for _, decision := range decisions {
		previous, found := latest[decision.QuestionRef]
		if !found ||
			previous.Choice != decision.Choice ||
			previous.AnswerText != decision.AnswerText {
			changes[decision.QuestionRef] = decision.Revision
		}
		latest[decision.QuestionRef] = decision
	}
	return changes
}
