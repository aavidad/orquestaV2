package intake

import "math"

// Snapshot is the complete, transport-neutral representation of an intake
// State. Persistence adapters may encode it, but encoding is deliberately not
// part of the domain contract.
type Snapshot struct {
	Schema         string
	Ref            Ref
	Revision       Revision
	Policy         Policy
	QuestionRounds uint32
	Issues         []Issue
	Questions      []Question
	Decisions      []Decision
	History        []Mutation
}

// Snapshot exports a complete defensive copy of state.
func (state State) Snapshot() Snapshot {
	return Snapshot{
		Schema:         StateSchema,
		Ref:            state.ref,
		Revision:       state.revision,
		Policy:         state.policy,
		QuestionRounds: state.questionRounds,
		Issues:         cloneIssues(state.issues),
		Questions:      cloneQuestions(state.questions),
		Decisions:      append([]Decision(nil), state.decisions...),
		History:        append([]Mutation(nil), state.history...),
	}
}

// Restore rebuilds a State only when the complete snapshot could have been
// produced by NewState followed by successful Apply transitions.
func Restore(snapshot Snapshot) (State, error) {
	if snapshot.Schema != StateSchema {
		return State{}, domainError(ErrorInvalidArgument, "snapshot.schema")
	}
	if !validRef(string(snapshot.Ref), "intake:") {
		return State{}, domainError(ErrorInvalidRef, "snapshot.ref")
	}
	if snapshot.Policy.MaxQuestionRounds == 0 {
		return State{}, domainError(ErrorInvalidArgument, "snapshot.policy.max_question_rounds")
	}
	if snapshot.Revision == 0 ||
		snapshot.Revision != Revision(len(snapshot.History))+1 {
		return State{}, domainError(ErrorInvalidArgument, "snapshot.revision")
	}
	if snapshot.QuestionRounds > snapshot.Policy.MaxQuestionRounds {
		return State{}, domainError(ErrorRoundLimit, "snapshot.question_rounds")
	}
	if err := validateSnapshotHistory(snapshot); err != nil {
		return State{}, err
	}
	return State{
		ref:            snapshot.Ref,
		revision:       snapshot.Revision,
		policy:         snapshot.Policy,
		questionRounds: snapshot.QuestionRounds,
		issues:         cloneIssues(snapshot.Issues),
		questions:      cloneQuestions(snapshot.Questions),
		decisions:      append([]Decision(nil), snapshot.Decisions...),
		history:        append([]Mutation(nil), snapshot.History...),
	}, nil
}

func validateSnapshotHistory(snapshot Snapshot) error {
	issueRefs := make(map[IssueRef]struct{}, len(snapshot.Issues))
	questionsByRef := make(map[QuestionRef]Question, len(snapshot.Questions))
	optionRefs := make(map[OptionRef]struct{})
	issueIndex, questionIndex, decisionIndex := 0, 0, 0
	var questionRounds uint32

	for historyIndex, mutation := range snapshot.History {
		expectedRevision := Revision(historyIndex) + 2
		if mutation.Revision != expectedRevision {
			return domainError(ErrorInvalidArgument, "snapshot.history.revision")
		}
		if mutation.Origin != OriginChat && mutation.Origin != OriginForm {
			return domainError(ErrorInvalidOrigin, "snapshot.history.origin")
		}
		if mutation.IssuesAdded < 0 || mutation.QuestionsAdded < 0 ||
			mutation.ChoicesRecorded < 0 {
			return domainError(ErrorInvalidArgument, "snapshot.history.count")
		}
		if mutation.IssuesAdded == 0 && mutation.QuestionsAdded == 0 &&
			mutation.ChoicesRecorded == 0 {
			return domainError(ErrorInvalidArgument, "snapshot.history")
		}
		if mutation.IssuesAdded > len(snapshot.Issues)-issueIndex ||
			mutation.QuestionsAdded > len(snapshot.Questions)-questionIndex ||
			mutation.ChoicesRecorded > len(snapshot.Decisions)-decisionIndex {
			return domainError(ErrorInvalidArgument, "snapshot.history.count")
		}

		nextIssueIndex := issueIndex + mutation.IssuesAdded
		for index := issueIndex; index < nextIssueIndex; index++ {
			issue := snapshot.Issues[index]
			if err := validateIssue(issue, index); err != nil {
				return snapshotDomainError(err)
			}
			if _, duplicate := issueRefs[issue.Ref]; duplicate {
				return domainError(ErrorDuplicateRef, "snapshot.issues.ref")
			}
			issueRefs[issue.Ref] = struct{}{}
		}
		issueIndex = nextIssueIndex

		nextQuestionIndex := questionIndex + mutation.QuestionsAdded
		for index := questionIndex; index < nextQuestionIndex; index++ {
			question := snapshot.Questions[index]
			if err := validateQuestion(question, index, issueRefs); err != nil {
				return snapshotDomainError(err)
			}
			if _, duplicate := questionsByRef[question.Ref]; duplicate {
				return domainError(ErrorDuplicateRef, "snapshot.questions.ref")
			}
			for _, option := range question.Options {
				if _, duplicate := optionRefs[option.Ref]; duplicate {
					return domainError(ErrorDuplicateRef, "snapshot.questions.options.ref")
				}
				optionRefs[option.Ref] = struct{}{}
			}
			questionsByRef[question.Ref] = question
		}
		questionIndex = nextQuestionIndex

		if mutation.QuestionsAdded > 0 {
			if questionRounds == math.MaxUint32 {
				return domainError(ErrorInvalidArgument, "snapshot.question_rounds")
			}
			questionRounds++
			if questionRounds > snapshot.Policy.MaxQuestionRounds {
				return domainError(ErrorRoundLimit, "snapshot.question_rounds")
			}
		}
		if mutation.QuestionRound != questionRounds {
			return domainError(ErrorInvalidArgument, "snapshot.history.question_round")
		}

		nextDecisionIndex := decisionIndex + mutation.ChoicesRecorded
		choiceQuestions := make(map[QuestionRef]struct{}, mutation.ChoicesRecorded)
		for index := decisionIndex; index < nextDecisionIndex; index++ {
			decision := snapshot.Decisions[index]
			if err := validateSnapshotDecision(
				decision, mutation, questionsByRef, choiceQuestions,
			); err != nil {
				return err
			}
		}
		decisionIndex = nextDecisionIndex
	}

	if issueIndex != len(snapshot.Issues) ||
		questionIndex != len(snapshot.Questions) ||
		decisionIndex != len(snapshot.Decisions) {
		return domainError(ErrorInvalidArgument, "snapshot.history.count")
	}
	if questionRounds != snapshot.QuestionRounds {
		return domainError(ErrorInvalidArgument, "snapshot.question_rounds")
	}
	return nil
}

func validateSnapshotDecision(
	decision Decision,
	mutation Mutation,
	questionsByRef map[QuestionRef]Question,
	choiceQuestions map[QuestionRef]struct{},
) error {
	if !validRef(string(decision.QuestionRef), "intake-question:") {
		return domainError(ErrorInvalidRef, "snapshot.decisions.question_ref")
	}
	if !validRef(string(decision.Choice), "intake-option:") {
		return domainError(ErrorInvalidRef, "snapshot.decisions.choice")
	}
	if !validRef(string(decision.Recommendation), "intake-option:") {
		return domainError(ErrorInvalidRef, "snapshot.decisions.recommendation")
	}
	if _, duplicate := choiceQuestions[decision.QuestionRef]; duplicate {
		return domainError(ErrorChoiceConflict, "snapshot.decisions.question_ref")
	}
	choiceQuestions[decision.QuestionRef] = struct{}{}
	question, exists := questionsByRef[decision.QuestionRef]
	if !exists {
		return domainError(ErrorQuestionNotFound, "snapshot.decisions.question_ref")
	}
	if !questionHasOption(question, decision.Choice) {
		return domainError(ErrorOptionNotFound, "snapshot.decisions.choice")
	}
	recommendation, valid := recommendedOption(question)
	if !valid || decision.Recommendation != recommendation.Ref {
		return domainError(ErrorRecommendationCount, "snapshot.decisions.recommendation")
	}
	if decision.RecommendationRationale != recommendation.RationaleKey {
		return domainError(ErrorInvalidArgument, "snapshot.decisions.recommendation_rationale_key")
	}
	if decision.Origin != mutation.Origin {
		return domainError(ErrorInvalidOrigin, "snapshot.decisions.origin")
	}
	if decision.Revision != mutation.Revision {
		return domainError(ErrorInvalidArgument, "snapshot.decisions.revision")
	}
	return nil
}

func snapshotDomainError(err error) error {
	domainErr, ok := err.(*DomainError)
	if !ok {
		return err
	}
	field := domainErr.Field
	const changePrefix = "change."
	if len(field) >= len(changePrefix) && field[:len(changePrefix)] == changePrefix {
		field = "snapshot." + field[len(changePrefix):]
	}
	return domainError(domainErr.Code, field)
}
