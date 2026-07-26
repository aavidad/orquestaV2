package intake

import (
	"math"
	"strconv"
	"strings"
	"unicode"
)

// State is an immutable intake snapshot. All slices are private and accessors
// return defensive copies.
type State struct {
	ref            Ref
	revision       Revision
	policy         Policy
	questionRounds uint32
	issues         []Issue
	questions      []Question
	decisions      []Decision
	history        []Mutation
}

func NewState(ref Ref, policy Policy) (State, error) {
	if !validRef(string(ref), "intake:") {
		return State{}, domainError(ErrorInvalidRef, "state_ref")
	}
	if policy.MaxQuestionRounds == 0 {
		return State{}, domainError(ErrorInvalidArgument, "policy.max_question_rounds")
	}
	return State{ref: ref, revision: 1, policy: policy}, nil
}

func (state State) Schema() string         { return StateSchema }
func (state State) Ref() Ref               { return state.ref }
func (state State) Revision() Revision     { return state.revision }
func (state State) Policy() Policy         { return state.policy }
func (state State) QuestionRounds() uint32 { return state.questionRounds }
func (state State) Issues() []Issue        { return cloneIssues(state.issues) }
func (state State) Questions() []Question  { return cloneQuestions(state.questions) }
func (state State) Decisions() []Decision  { return append([]Decision(nil), state.decisions...) }
func (state State) History() []Mutation    { return append([]Mutation(nil), state.history...) }

func (state State) CurrentDecision(ref QuestionRef) (Decision, bool) {
	decisions, _ := state.projectDecisions()
	decision, found := decisions[ref]
	return decision, found
}

func (state State) ReopenedDecisions() []ReopenedDecision {
	_, reopened := state.projectDecisions()
	result := make([]ReopenedDecision, 0, len(reopened))
	for _, question := range state.questions {
		if decision, found := reopened[question.Ref]; found {
			decision.InvalidatedBy = append([]DecisionChange(nil), decision.InvalidatedBy...)
			result = append(result, decision)
		}
	}
	return result
}

// Apply validates the complete change before allocating the next immutable
// snapshot. On error the caller's current State cannot have been modified.
func Apply(current State, change Change) (State, error) {
	if !current.initialized() {
		return State{}, domainError(ErrorChannelStateCreation, "state")
	}
	if !validRef(string(change.StateRef), "intake:") {
		return State{}, domainError(ErrorInvalidRef, "change.state_ref")
	}
	if change.StateRef != current.ref {
		return State{}, domainError(ErrorStateMismatch, "change.state_ref")
	}
	if change.ExpectedRevision != current.revision {
		return State{}, domainError(ErrorRevisionConflict, "change.expected_revision")
	}
	if change.Origin != OriginChat && change.Origin != OriginForm {
		return State{}, domainError(ErrorInvalidOrigin, "change.origin")
	}
	if len(change.Issues) == 0 && len(change.Questions) == 0 && len(change.Choices) == 0 {
		return State{}, domainError(ErrorInvalidArgument, "change")
	}
	if current.revision == Revision(math.MaxUint64) {
		return State{}, domainError(ErrorInvalidArgument, "state.revision")
	}

	issueRefs := make(map[IssueRef]struct{}, len(current.issues)+len(change.Issues))
	for _, issue := range current.issues {
		issueRefs[issue.Ref] = struct{}{}
	}
	for index, issue := range change.Issues {
		if err := validateIssue(issue, index); err != nil {
			return State{}, err
		}
		if _, exists := issueRefs[issue.Ref]; exists {
			return State{}, domainError(ErrorDuplicateRef, "change.issues.ref")
		}
		issueRefs[issue.Ref] = struct{}{}
	}

	questionsByRef := make(map[QuestionRef]Question, len(current.questions)+len(change.Questions))
	optionRefs := make(map[OptionRef]struct{})
	for _, question := range current.questions {
		questionsByRef[question.Ref] = question
		for _, option := range question.Options {
			optionRefs[option.Ref] = struct{}{}
		}
	}
	for index, question := range change.Questions {
		if err := validateQuestion(question, index, issueRefs); err != nil {
			return State{}, err
		}
		if _, exists := questionsByRef[question.Ref]; exists {
			return State{}, domainError(ErrorDuplicateRef, "change.questions.ref")
		}
		for _, option := range question.Options {
			if _, exists := optionRefs[option.Ref]; exists {
				return State{}, domainError(ErrorDuplicateRef, "change.questions.options.ref")
			}
			optionRefs[option.Ref] = struct{}{}
		}
		questionsByRef[question.Ref] = question
	}
	for index, question := range change.Questions {
		if err := validateQuestionDependencies(question, index, questionsByRef); err != nil {
			return State{}, err
		}
	}
	if hasQuestionDependencyCycle(questionsByRef) {
		return State{}, domainError(ErrorDependencyCycle, "change.questions.depends_on")
	}
	if len(change.Questions) > 0 && current.questionRounds >= current.policy.MaxQuestionRounds {
		return State{}, domainError(ErrorRoundLimit, "policy.max_question_rounds")
	}

	choiceQuestions := make(map[QuestionRef]struct{}, len(change.Choices))
	type resolvedChoice struct {
		choice         Choice
		recommendation Option
	}
	resolved := make([]resolvedChoice, 0, len(change.Choices))
	for index, choice := range change.Choices {
		if !validRef(string(choice.QuestionRef), "intake-question:") {
			return State{}, domainError(ErrorInvalidRef, indexedField("change.choices.question_ref", index))
		}
		if !validRef(string(choice.OptionRef), "intake-option:") {
			return State{}, domainError(ErrorInvalidRef, indexedField("change.choices.option_ref", index))
		}
		if _, duplicate := choiceQuestions[choice.QuestionRef]; duplicate {
			return State{}, domainError(ErrorChoiceConflict, "change.choices.question_ref")
		}
		choiceQuestions[choice.QuestionRef] = struct{}{}
		question, exists := questionsByRef[choice.QuestionRef]
		if !exists {
			return State{}, domainError(ErrorQuestionNotFound, indexedField("change.choices.question_ref", index))
		}
		if !questionHasOption(question, choice.OptionRef) {
			return State{}, domainError(ErrorOptionNotFound, indexedField("change.choices.option_ref", index))
		}
		recommendation, ok := recommendedOption(question)
		if !ok {
			return State{}, domainError(ErrorRecommendationCount, "change.questions.options.recommended")
		}
		resolved = append(resolved, resolvedChoice{choice: choice, recommendation: recommendation})
	}
	currentDecisions, _ := current.projectDecisions()
	for index, item := range resolved {
		question := questionsByRef[item.choice.QuestionRef]
		for _, dependencyRef := range question.DependsOn {
			if _, included := choiceQuestions[dependencyRef]; included {
				continue
			}
			if _, decided := currentDecisions[dependencyRef]; !decided {
				return State{}, domainError(
					ErrorDependencyPending,
					indexedField("change.choices.question_ref", index),
				)
			}
		}
	}

	updated := current.clone()
	updated.issues = append(updated.issues, cloneIssues(change.Issues)...)
	updated.questions = append(updated.questions, cloneQuestions(change.Questions)...)
	if len(change.Questions) > 0 {
		updated.questionRounds++
	}
	updated.revision++
	for _, item := range resolved {
		updated.decisions = append(updated.decisions, Decision{
			QuestionRef:             item.choice.QuestionRef,
			Choice:                  item.choice.OptionRef,
			Recommendation:          item.recommendation.Ref,
			RecommendationRationale: item.recommendation.RationaleKey,
			Origin:                  change.Origin,
			Revision:                updated.revision,
		})
	}
	updated.history = append(updated.history, Mutation{
		Origin: change.Origin, Revision: updated.revision,
		IssuesAdded: len(change.Issues), QuestionsAdded: len(change.Questions),
		ChoicesRecorded: len(change.Choices), QuestionRound: updated.questionRounds,
	})
	return updated, nil
}

func (state State) initialized() bool {
	return validRef(string(state.ref), "intake:") && state.revision > 0 &&
		state.policy.MaxQuestionRounds > 0
}

func (state State) clone() State {
	state.issues = cloneIssues(state.issues)
	state.questions = cloneQuestions(state.questions)
	state.decisions = append([]Decision(nil), state.decisions...)
	state.history = append([]Mutation(nil), state.history...)
	return state
}

func validateIssue(issue Issue, index int) error {
	if !validRef(string(issue.Ref), "intake-issue:") {
		return domainError(ErrorInvalidRef, indexedField("change.issues.ref", index))
	}
	if issue.Kind != IssueGap && issue.Kind != IssueContradiction {
		return domainError(ErrorInvalidArgument, indexedField("change.issues.kind", index))
	}
	if !validMachineKey(issue.Field) {
		return domainError(ErrorInvalidArgument, indexedField("change.issues.field", index))
	}
	if !validMessageKey(issue.DetailKey) {
		return domainError(ErrorMessageKeyInvalid, indexedField("change.issues.detail_key", index))
	}
	return nil
}

func validateQuestion(question Question, index int, issues map[IssueRef]struct{}) error {
	if !validRef(string(question.Ref), "intake-question:") {
		return domainError(ErrorInvalidRef, indexedField("change.questions.ref", index))
	}
	if len(question.DerivedFrom) == 0 {
		return domainError(ErrorIssueNotFound, indexedField("change.questions.derived_from", index))
	}
	derivedRefs := make(map[IssueRef]struct{}, len(question.DerivedFrom))
	for _, ref := range question.DerivedFrom {
		if !validRef(string(ref), "intake-issue:") {
			return domainError(ErrorInvalidRef, indexedField("change.questions.derived_from", index))
		}
		if _, duplicate := derivedRefs[ref]; duplicate {
			return domainError(ErrorDuplicateRef, indexedField("change.questions.derived_from", index))
		}
		if _, exists := issues[ref]; !exists {
			return domainError(ErrorIssueNotFound, indexedField("change.questions.derived_from", index))
		}
		derivedRefs[ref] = struct{}{}
	}
	if !validMessageKey(question.PromptKey) {
		return domainError(ErrorMessageKeyInvalid, indexedField("change.questions.prompt_key", index))
	}
	if !validMessageKey(question.WhyKey) {
		return domainError(ErrorMessageKeyInvalid, indexedField("change.questions.why_key", index))
	}
	recommendations := 0
	options := make(map[OptionRef]struct{}, len(question.Options))
	for optionIndex, option := range question.Options {
		if !validRef(string(option.Ref), "intake-option:") {
			return domainError(ErrorInvalidRef, indexedField("change.questions.options.ref", optionIndex))
		}
		if _, duplicate := options[option.Ref]; duplicate {
			return domainError(ErrorDuplicateRef, "change.questions.options.ref")
		}
		if !validMessageKey(option.LabelKey) {
			return domainError(ErrorMessageKeyInvalid, indexedField("change.questions.options.label_key", optionIndex))
		}
		if !validMessageKey(option.RationaleKey) {
			return domainError(ErrorMessageKeyInvalid, indexedField("change.questions.options.rationale_key", optionIndex))
		}
		if option.Recommended {
			recommendations++
		}
		options[option.Ref] = struct{}{}
	}
	if recommendations != 1 {
		return domainError(ErrorRecommendationCount, indexedField("change.questions.options.recommended", index))
	}
	return nil
}

func validateQuestionDependencies(
	question Question,
	index int,
	questions map[QuestionRef]Question,
) error {
	dependencies := make(map[QuestionRef]struct{}, len(question.DependsOn))
	for _, ref := range question.DependsOn {
		if !validRef(string(ref), "intake-question:") {
			return domainError(
				ErrorInvalidRef,
				indexedField("change.questions.depends_on", index),
			)
		}
		if _, duplicate := dependencies[ref]; duplicate {
			return domainError(
				ErrorDuplicateRef,
				indexedField("change.questions.depends_on", index),
			)
		}
		if _, exists := questions[ref]; !exists {
			return domainError(
				ErrorQuestionNotFound,
				indexedField("change.questions.depends_on", index),
			)
		}
		dependencies[ref] = struct{}{}
	}
	return nil
}

func recommendedOption(question Question) (Option, bool) {
	var found Option
	count := 0
	for _, option := range question.Options {
		if option.Recommended {
			found, count = option, count+1
		}
	}
	return found, count == 1
}

func questionHasOption(question Question, ref OptionRef) bool {
	for _, option := range question.Options {
		if option.Ref == ref {
			return true
		}
	}
	return false
}

func validRef(value, prefix string) bool {
	if len(value) <= len(prefix) || len(value) > 512 || !strings.HasPrefix(value, prefix) ||
		strings.TrimSpace(value) != value {
		return false
	}
	for _, char := range value {
		if unicode.IsControl(char) || unicode.IsSpace(char) {
			return false
		}
	}
	return true
}

func validMachineKey(value string) bool {
	if value == "" || len(value) > 256 || strings.TrimSpace(value) != value ||
		strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") || strings.Contains(value, "..") {
		return false
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') &&
			char != '.' && char != '_' && char != '-' {
			return false
		}
	}
	return true
}

func validMessageKey(key MessageKey) bool {
	value := string(key)
	return strings.Contains(value, ".") && validMachineKey(value)
}

func indexedField(field string, index int) string {
	return field + "[" + strconv.Itoa(index) + "]"
}

func cloneIssues(values []Issue) []Issue {
	return append([]Issue(nil), values...)
}

func cloneQuestions(values []Question) []Question {
	out := make([]Question, len(values))
	for index, question := range values {
		out[index] = question
		out[index].DerivedFrom = append([]IssueRef(nil), question.DerivedFrom...)
		out[index].DependsOn = append([]QuestionRef(nil), question.DependsOn...)
		out[index].Options = append([]Option(nil), question.Options...)
	}
	return out
}
