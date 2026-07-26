package intake

func hasQuestionDependencyCycle(questions map[QuestionRef]Question) bool {
	const (
		unvisited uint8 = iota
		visiting
		visited
	)
	marks := make(map[QuestionRef]uint8, len(questions))
	var visit func(QuestionRef) bool
	visit = func(ref QuestionRef) bool {
		switch marks[ref] {
		case visiting:
			return true
		case visited:
			return false
		}
		marks[ref] = visiting
		for _, dependencyRef := range questions[ref].DependsOn {
			if visit(dependencyRef) {
				return true
			}
		}
		marks[ref] = visited
		return false
	}
	for ref := range questions {
		if visit(ref) {
			return true
		}
	}
	return false
}

func (state State) projectDecisions() (
	map[QuestionRef]Decision,
	map[QuestionRef]ReopenedDecision,
) {
	current := make(map[QuestionRef]Decision, len(state.questions))
	reopened := make(map[QuestionRef]ReopenedDecision)
	questions := make(map[QuestionRef]Question, len(state.questions))
	for _, question := range state.questions {
		questions[question.Ref] = question
	}
	dependencies := transitiveQuestionDependencies(questions)

	for offset := 0; offset < len(state.decisions); {
		end := offset + 1
		for end < len(state.decisions) &&
			state.decisions[end].Revision == state.decisions[offset].Revision {
			end++
		}
		group := state.decisions[offset:end]
		changedByRef := make(map[QuestionRef]DecisionChange, len(group))
		for _, decision := range group {
			previous, active := current[decision.QuestionRef]
			if !active || previous.Choice != decision.Choice ||
				previous.AnswerText != decision.AnswerText {
				changedByRef[decision.QuestionRef] = DecisionChange{
					QuestionRef: decision.QuestionRef,
					Revision:    decision.Revision,
				}
			}
		}
		changed := make([]DecisionChange, 0, len(changedByRef))
		for _, question := range state.questions {
			if value, found := changedByRef[question.Ref]; found {
				changed = append(changed, value)
			}
		}

		for _, question := range state.questions {
			causes := dependencyChangesFor(question.Ref, changed, dependencies)
			if len(causes) == 0 {
				continue
			}
			if previous, active := current[question.Ref]; active {
				reopened[question.Ref] = ReopenedDecision{
					QuestionRef:      question.Ref,
					PreviousDecision: previous,
					InvalidatedBy:    append([]DecisionChange(nil), causes...),
				}
				delete(current, question.Ref)
				continue
			}
			if value, alreadyReopened := reopened[question.Ref]; alreadyReopened {
				value.InvalidatedBy = appendDecisionChanges(value.InvalidatedBy, causes...)
				reopened[question.Ref] = value
			}
		}

		for _, decision := range group {
			current[decision.QuestionRef] = decision
			delete(reopened, decision.QuestionRef)
		}
		offset = end
	}
	return current, reopened
}

func dependencyChangesFor(
	questionRef QuestionRef,
	changes []DecisionChange,
	dependencies map[QuestionRef]map[QuestionRef]struct{},
) []DecisionChange {
	result := make([]DecisionChange, 0, len(changes))
	for _, change := range changes {
		if _, depends := dependencies[questionRef][change.QuestionRef]; depends {
			result = append(result, change)
		}
	}
	return result
}

func transitiveQuestionDependencies(
	questions map[QuestionRef]Question,
) map[QuestionRef]map[QuestionRef]struct{} {
	result := make(map[QuestionRef]map[QuestionRef]struct{}, len(questions))
	var collect func(QuestionRef) map[QuestionRef]struct{}
	collect = func(ref QuestionRef) map[QuestionRef]struct{} {
		if dependencies, found := result[ref]; found {
			return dependencies
		}
		dependencies := make(map[QuestionRef]struct{})
		for _, directRef := range questions[ref].DependsOn {
			dependencies[directRef] = struct{}{}
			for transitiveRef := range collect(directRef) {
				dependencies[transitiveRef] = struct{}{}
			}
		}
		result[ref] = dependencies
		return dependencies
	}
	for ref := range questions {
		collect(ref)
	}
	return result
}

func appendDecisionChanges(
	current []DecisionChange,
	additions ...DecisionChange,
) []DecisionChange {
	for _, addition := range additions {
		duplicate := false
		for _, existing := range current {
			if existing == addition {
				duplicate = true
				break
			}
		}
		if !duplicate {
			current = append(current, addition)
		}
	}
	return current
}
