package intake

// BuildAcceptRecommendationsChange expands every still-pending recommendation
// from one question round into one ordinary atomic Change. It does not apply
// the change, so the application writer remains the only durable authority.
func BuildAcceptRecommendationsChange(
	current State,
	request AcceptRecommendationsRequest,
) (Change, error) {
	if err := validateStateRequest(
		current,
		request.StateRef,
		request.ExpectedRevision,
		request.Origin,
	); err != nil {
		return Change{}, err
	}
	if request.QuestionRound == 0 || request.QuestionRound > current.questionRounds {
		return Change{}, domainError(ErrorQuestionRound, "request.question_round")
	}

	questions := current.questionsInRound(request.QuestionRound)
	choices := make([]Choice, 0, len(questions))
	currentDecisions, _ := current.projectDecisions()
	for _, question := range questions {
		if _, resolved := currentDecisions[question.Ref]; resolved {
			continue
		}
		recommendation, valid := recommendedOption(question)
		if !valid {
			return Change{}, domainError(
				ErrorRecommendationCount,
				"state.questions.options.recommended",
			)
		}
		choices = append(choices, Choice{
			QuestionRef: question.Ref,
			OptionRef:   recommendation.Ref,
		})
	}
	if len(choices) == 0 {
		return Change{}, domainError(
			ErrorRecommendationsDone,
			"request.question_round",
		)
	}
	change := Change{
		StateRef:         request.StateRef,
		ExpectedRevision: request.ExpectedRevision,
		Origin:           request.Origin,
		Choices:          choices,
	}
	if _, err := Apply(current, change); err != nil {
		return Change{}, err
	}
	return change, nil
}

// ReemitContext returns catalog-keyed facts from one exact snapshot. Help and
// clarification never call Apply, increment Revision or append History.
func ReemitContext(current State, request ContextRequest) (Context, error) {
	if err := validateStateRequest(
		current,
		request.StateRef,
		request.ExpectedRevision,
		request.Origin,
	); err != nil {
		return Context{}, err
	}
	if request.Kind != ContextClarification && request.Kind != ContextHelp {
		return Context{}, domainError(ErrorInvalidArgument, "request.kind")
	}

	questionsByRef := make(map[QuestionRef]Question, len(current.questions))
	for _, question := range current.questions {
		questionsByRef[question.Ref] = question
	}
	selected := make([]Question, 0, len(request.QuestionRefs))
	if len(request.QuestionRefs) == 0 {
		selected = cloneQuestions(current.questions)
	} else {
		seen := make(map[QuestionRef]struct{}, len(request.QuestionRefs))
		for index, ref := range request.QuestionRefs {
			if !validRef(string(ref), "intake-question:") {
				return Context{}, domainError(
					ErrorInvalidRef,
					indexedField("request.question_refs", index),
				)
			}
			if _, duplicate := seen[ref]; duplicate {
				return Context{}, domainError(
					ErrorDuplicateRef,
					indexedField("request.question_refs", index),
				)
			}
			question, found := questionsByRef[ref]
			if !found {
				return Context{}, domainError(
					ErrorQuestionNotFound,
					indexedField("request.question_refs", index),
				)
			}
			selected = append(selected, cloneQuestions([]Question{question})[0])
			seen[ref] = struct{}{}
		}
	}

	issueRefs := make(map[IssueRef]struct{})
	for _, question := range selected {
		for _, issueRef := range question.DerivedFrom {
			issueRefs[issueRef] = struct{}{}
		}
	}
	issues := make([]Issue, 0, len(issueRefs))
	for _, issue := range current.issues {
		if _, selectedIssue := issueRefs[issue.Ref]; selectedIssue {
			issues = append(issues, issue)
		}
	}

	decisions, reopened := current.projectDecisions()
	questionContexts := make([]QuestionContext, 0, len(selected))
	for _, question := range selected {
		item := QuestionContext{Question: question}
		if decision, found := decisions[question.Ref]; found {
			value := decision
			item.CurrentDecision = &value
		}
		if decision, found := reopened[question.Ref]; found {
			value := decision
			value.InvalidatedBy = append([]DecisionChange(nil), value.InvalidatedBy...)
			item.ReopenedDecision = &value
		}
		questionContexts = append(questionContexts, item)
	}

	return Context{
		StateRef:  current.ref,
		Revision:  current.revision,
		Origin:    request.Origin,
		Kind:      request.Kind,
		Issues:    issues,
		Questions: questionContexts,
	}, nil
}

func validateStateRequest(
	current State,
	stateRef Ref,
	expectedRevision Revision,
	origin Origin,
) error {
	if !current.initialized() {
		return domainError(ErrorChannelStateCreation, "state")
	}
	if !validRef(string(stateRef), "intake:") {
		return domainError(ErrorInvalidRef, "request.state_ref")
	}
	if stateRef != current.ref {
		return domainError(ErrorStateMismatch, "request.state_ref")
	}
	if expectedRevision != current.revision {
		return domainError(ErrorRevisionConflict, "request.expected_revision")
	}
	if origin != OriginChat && origin != OriginForm {
		return domainError(ErrorInvalidOrigin, "request.origin")
	}
	return nil
}

func (state State) questionsInRound(round uint32) []Question {
	offset := 0
	for _, mutation := range state.history {
		end := offset + mutation.QuestionsAdded
		if mutation.QuestionsAdded > 0 && mutation.QuestionRound == round {
			return cloneQuestions(state.questions[offset:end])
		}
		offset = end
	}
	return nil
}
