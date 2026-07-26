package application

import (
	"strings"

	"orquesta/internal/intake"
	"orquesta/internal/wizard/catalog"
	"orquesta/internal/wizard/gaps"
)

type wizardGapsPreflight struct {
	dimensions          map[intake.QuestionRef]gaps.DimensionDescriptor
	issueDerivations    map[intake.IssueRef]intake.DerivationIdentity
	questionDerivations map[intake.QuestionRef]intake.DerivationIdentity
	issues              map[intake.IssueRef]intake.Issue
	questions           map[intake.QuestionRef]intake.Question
}

type wizardGapsIssueContract struct {
	field        string
	detailPrefix string
	order        int
}

func preflightWizardGapsQuestions(
	state intake.State,
	evaluator wizardGapsEvaluator,
) (wizardGapsPreflight, error) {
	issueDerivations, questionDerivations, err := wizardGapsArtifactDerivations(state)
	if err != nil {
		return wizardGapsPreflight{}, err
	}
	identity := evaluator.Identity()
	dimensions := evaluator.Dimensions()
	rules := evaluator.Rules()
	contracts := make(map[intake.QuestionRef]gaps.DimensionDescriptor, len(dimensions))
	optionOwners := make(map[intake.OptionRef]intake.QuestionRef)
	issueContracts := make(map[intake.IssueRef]wizardGapsIssueContract)
	dependencies := make(map[intake.QuestionRef]map[intake.QuestionRef]struct{})
	dependencyOrder := make(map[intake.QuestionRef]int, len(dimensions))
	for index, dimension := range dimensions {
		questionRef := wizardDimensionQuestionRef(dimension.Ref())
		contracts[questionRef] = dimension
		dependencyOrder[questionRef] = index
		dependencies[questionRef] = make(map[intake.QuestionRef]struct{})
		for _, dependency := range dimension.DependsOn() {
			if dependency != dimension.Ref() {
				dependencies[questionRef][wizardDimensionQuestionRef(dependency)] = struct{}{}
			}
		}
		for _, option := range dimension.Options() {
			optionOwners[intake.OptionRef(option.Ref())] = questionRef
		}
		key := strings.ToLower(string(dimension.Ref()))
		issueContracts[intake.IssueRef("intake-issue:wizard.dimension."+key)] =
			wizardGapsIssueContract{
				field:        string(dimension.Slot()),
				detailPrefix: "wizard.gaps.dimension." + key + ".issue.",
				order:        0,
			}
	}
	for _, rule := range rules {
		ruleKey := strings.ToLower(string(rule.Ref()))
		for _, target := range rule.Targets() {
			questionRef := wizardDimensionQuestionRef(target)
			dimension, found := contracts[questionRef]
			if !found {
				return wizardGapsPreflight{}, wizardGapsProjectionConflict(
					"rule-target",
					string(target),
				)
			}
			issueContracts[intake.IssueRef("intake-issue:wizard.rule."+ruleKey)] =
				wizardGapsIssueContract{
					field:        string(dimension.Slot()),
					detailPrefix: "wizard.gaps.rule." + ruleKey + ".",
					order:        rule.Ordinal() + 1,
				}
			for _, dependency := range rule.DependsOn() {
				if dependency != target {
					dependencies[questionRef][wizardDimensionQuestionRef(dependency)] =
						struct{}{}
				}
			}
		}
	}

	issues := make(map[intake.IssueRef]intake.Issue)
	for _, issue := range state.Issues() {
		issues[issue.Ref] = issue
		contract, canonical := issueContracts[issue.Ref]
		if !canonical {
			continue
		}
		if issueDerivations[issue.Ref] != identity ||
			!wizardGapsIssuePayloadPermitted(issue, contract) {
			return wizardGapsPreflight{}, wizardGapsProjectionConflict(
				"issue",
				string(issue.Ref),
			)
		}
	}

	questions := make(map[intake.QuestionRef]intake.Question)
	for _, question := range state.Questions() {
		questions[question.Ref] = question
		if dimension, canonical := contracts[question.Ref]; canonical {
			if questionDerivations[question.Ref] != identity ||
				!wizardDimensionQuestionPayloadPermitted(
					question,
					dimension,
					issueContracts,
					issues,
					dependencies[question.Ref],
					dependencyOrder,
				) {
				return wizardGapsPreflight{}, wizardGapsProjectionConflict(
					"question",
					string(question.Ref),
				)
			}
		}
		for _, option := range question.Options {
			owner, canonical := optionOwners[option.Ref]
			if canonical && owner != question.Ref {
				return wizardGapsPreflight{}, wizardGapsProjectionConflict(
					"option",
					string(option.Ref),
				)
			}
		}
	}
	return wizardGapsPreflight{
		dimensions: contracts, issueDerivations: issueDerivations,
		questionDerivations: questionDerivations, issues: issues,
		questions: questions,
	}, nil
}

func preflightWizardGapsDimensionPayloads(
	preflight wizardGapsPreflight,
	evaluator wizardGapsEvaluator,
	facts gaps.Facts,
	selections []gaps.Selection,
	packRefs []catalog.PackRef,
) error {
	for _, dimension := range evaluator.Dimensions() {
		questionRef := wizardDimensionQuestionRef(dimension.Ref())
		actual, found := preflight.questions[questionRef]
		if !found {
			continue
		}
		withoutDimension := make([]gaps.Selection, 0, len(selections))
		for _, selection := range selections {
			if selection.Dimension != dimension.Ref() {
				withoutDimension = append(withoutDimension, selection)
			}
		}
		evaluation, err := evaluator.Evaluate(gaps.Input{
			Facts: facts, Selections: withoutDimension, PackRefs: packRefs,
		})
		if err != nil {
			return err
		}
		var expected intake.Question
		for _, question := range evaluation.Questions() {
			if question.Ref() == gaps.QuestionRef(questionRef) {
				expected = question.IntakeQuestion()
				break
			}
		}
		if expected.Ref == "" ||
			preflight.questionDerivations[questionRef] != evaluator.Identity() ||
			!wizardGapsQuestionPayloadEqual(actual, expected) {
			return wizardGapsProjectionConflict("question", string(questionRef))
		}
	}
	return nil
}

func preflightWizardGapsSupplementalQuestions(
	preflight wizardGapsPreflight,
	evaluator wizardGapsEvaluator,
	evaluation gaps.Result,
) error {
	identity := evaluator.Identity()
	optionOwners := make(map[intake.OptionRef]intake.QuestionRef)
	expected := make(map[intake.QuestionRef]intake.Question)
	expectedIssues := make(map[intake.IssueRef]intake.Issue)
	supplementalIssueRefs := make(map[intake.IssueRef]struct{})
	for _, issue := range evaluation.Issues() {
		expectedIssues[intake.IssueRef(issue.Ref())] = issue.IntakeIssue()
	}
	for _, question := range evaluation.Questions() {
		if question.Dimension() != "" {
			continue
		}
		projected := question.IntakeQuestion()
		expected[projected.Ref] = projected
		for _, ref := range projected.DerivedFrom {
			supplementalIssueRefs[ref] = struct{}{}
		}
		for _, option := range projected.Options {
			optionOwners[option.Ref] = projected.Ref
		}
	}
	for ref := range supplementalIssueRefs {
		actual, found := preflight.issues[ref]
		if !found {
			continue
		}
		want, expected := expectedIssues[ref]
		if !expected || preflight.issueDerivations[ref] != identity || actual != want {
			return wizardGapsProjectionConflict("issue", string(ref))
		}
	}
	for _, question := range preflight.questions {
		for _, option := range question.Options {
			owner, canonical := optionOwners[option.Ref]
			if canonical && owner != question.Ref {
				return wizardGapsProjectionConflict("option", string(option.Ref))
			}
		}
		want, canonical := expected[question.Ref]
		if !canonical {
			continue
		}
		if preflight.questionDerivations[question.Ref] != identity ||
			!wizardGapsQuestionPayloadEqual(question, want) {
			return wizardGapsProjectionConflict("question", string(question.Ref))
		}
		for _, ref := range want.DerivedFrom {
			actualIssue, found := preflight.issues[ref]
			expectedIssue, expected := expectedIssues[ref]
			if !found || !expected ||
				preflight.issueDerivations[ref] != identity ||
				actualIssue != expectedIssue {
				return wizardGapsProjectionConflict("issue", string(ref))
			}
		}
	}
	return nil
}

func wizardGapsArtifactDerivations(
	state intake.State,
) (
	map[intake.IssueRef]intake.DerivationIdentity,
	map[intake.QuestionRef]intake.DerivationIdentity,
	error,
) {
	issues := state.Issues()
	questions := state.Questions()
	issueDerivations := make(map[intake.IssueRef]intake.DerivationIdentity, len(issues))
	questionDerivations := make(
		map[intake.QuestionRef]intake.DerivationIdentity,
		len(questions),
	)
	issueOffset, questionOffset := 0, 0
	for _, mutation := range state.History() {
		if mutation.IssuesAdded < 0 || mutation.QuestionsAdded < 0 ||
			mutation.IssuesAdded > len(issues)-issueOffset ||
			mutation.QuestionsAdded > len(questions)-questionOffset {
			return nil, nil, wizardGapsProjectionConflict(
				"history",
				string(state.Ref()),
			)
		}
		for _, issue := range issues[issueOffset : issueOffset+mutation.IssuesAdded] {
			issueDerivations[issue.Ref] = mutation.Derivation
		}
		for _, question := range questions[questionOffset : questionOffset+mutation.QuestionsAdded] {
			questionDerivations[question.Ref] = mutation.Derivation
		}
		issueOffset += mutation.IssuesAdded
		questionOffset += mutation.QuestionsAdded
	}
	if issueOffset != len(issues) || questionOffset != len(questions) {
		return nil, nil, wizardGapsProjectionConflict("history", string(state.Ref()))
	}
	return issueDerivations, questionDerivations, nil
}

func wizardDimensionQuestionPayloadPermitted(
	question intake.Question,
	dimension gaps.DimensionDescriptor,
	issueContracts map[intake.IssueRef]wizardGapsIssueContract,
	issues map[intake.IssueRef]intake.Issue,
	allowedDependencies map[intake.QuestionRef]struct{},
	dependencyOrder map[intake.QuestionRef]int,
) bool {
	if question.Ref != wizardDimensionQuestionRef(dimension.Ref()) ||
		question.PromptKey != intake.MessageKey(dimension.PromptKey()) ||
		question.WhyKey != intake.MessageKey(dimension.WhyKey()) ||
		len(question.DerivedFrom) == 0 {
		return false
	}
	expectedOptions := dimension.Options()
	if len(question.Options) != len(expectedOptions) {
		return false
	}
	recommended := 0
	for index, expected := range expectedOptions {
		option := question.Options[index]
		if option.Ref != intake.OptionRef(expected.Ref()) ||
			option.LabelKey != intake.MessageKey(expected.LabelKey()) ||
			option.RationaleKey != intake.MessageKey(expected.RationaleKey()) ||
			option.AcceptsText != (expected.Kind() == gaps.OptionFreeText) {
			return false
		}
		if option.Recommended {
			recommended++
		}
	}
	if recommended != 1 {
		return false
	}
	seenIssues := make(map[intake.IssueRef]struct{}, len(question.DerivedFrom))
	lastIssueOrder := -1
	for _, ref := range question.DerivedFrom {
		contract, permitted := issueContracts[ref]
		issue, found := issues[ref]
		if !permitted || !found || !wizardGapsIssuePayloadPermitted(issue, contract) {
			return false
		}
		if _, duplicate := seenIssues[ref]; duplicate {
			return false
		}
		if contract.order <= lastIssueOrder {
			return false
		}
		lastIssueOrder = contract.order
		seenIssues[ref] = struct{}{}
	}
	seenDependencies := make(map[intake.QuestionRef]struct{}, len(question.DependsOn))
	lastDependencyOrder := -1
	for _, ref := range question.DependsOn {
		if _, permitted := allowedDependencies[ref]; !permitted {
			return false
		}
		if _, duplicate := seenDependencies[ref]; duplicate {
			return false
		}
		order, found := dependencyOrder[ref]
		if !found || order <= lastDependencyOrder {
			return false
		}
		lastDependencyOrder = order
		seenDependencies[ref] = struct{}{}
	}
	return true
}

func wizardGapsIssuePayloadPermitted(
	issue intake.Issue,
	contract wizardGapsIssueContract,
) bool {
	if issue.Field != contract.field {
		return false
	}
	switch issue.Kind {
	case intake.IssueGap:
		return issue.DetailKey == intake.MessageKey(contract.detailPrefix+"gap")
	case intake.IssueContradiction:
		return issue.DetailKey == intake.MessageKey(contract.detailPrefix+"contradiction")
	default:
		return false
	}
}

func wizardDimensionQuestionRef(ref gaps.DimensionRef) intake.QuestionRef {
	return intake.QuestionRef(
		"intake-question:wizard." + strings.ToLower(string(ref)),
	)
}
