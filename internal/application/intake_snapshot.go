package application

import (
	"errors"
	"reflect"

	"orquesta/internal/intake"
)

var errIntakeSnapshotInvalid = errors.New("application.intake_snapshot_invalid")

func SnapshotIntake(state intake.State) IntakeSnapshot {
	issues := state.Issues()
	questions := state.Questions()
	decisions := state.Decisions()
	history := state.History()
	if len(issues) == 0 {
		issues = nil
	}
	if len(questions) == 0 {
		questions = nil
	}
	if len(decisions) == 0 {
		decisions = nil
	}
	if len(history) == 0 {
		history = nil
	}
	return IntakeSnapshot{
		Schema:         state.Schema(),
		Ref:            state.Ref(),
		Revision:       state.Revision(),
		Policy:         state.Policy(),
		QuestionRounds: state.QuestionRounds(),
		Issues:         issues,
		Questions:      questions,
		Decisions:      decisions,
		History:        history,
	}
}

// RestoreIntake replays a persisted snapshot through the pure intake
// transition. Adapters never gain authority to construct private state.
func RestoreIntake(snapshot IntakeSnapshot) (intake.State, error) {
	snapshot = canonicalIntakeSnapshot(snapshot)
	if snapshot.Schema != intake.StateSchema || snapshot.Revision == 0 {
		return intake.State{}, invalidIntakeSnapshot()
	}
	state, err := intake.NewState(snapshot.Ref, snapshot.Policy)
	if err != nil {
		return intake.State{}, invalidIntakeSnapshotWithCause(err)
	}

	issueOffset, questionOffset, decisionOffset := 0, 0, 0
	for _, mutation := range snapshot.History {
		if mutation.Revision != state.Revision()+1 ||
			mutation.IssuesAdded < 0 || mutation.QuestionsAdded < 0 ||
			mutation.ChoicesRecorded < 0 ||
			mutation.IssuesAdded > len(snapshot.Issues)-issueOffset ||
			mutation.QuestionsAdded > len(snapshot.Questions)-questionOffset ||
			mutation.ChoicesRecorded > len(snapshot.Decisions)-decisionOffset {
			return intake.State{}, invalidIntakeSnapshot()
		}
		choices := make([]intake.Choice, mutation.ChoicesRecorded)
		for index := range choices {
			decision := snapshot.Decisions[decisionOffset+index]
			if decision.Revision != mutation.Revision || decision.Origin != mutation.Origin {
				return intake.State{}, invalidIntakeSnapshot()
			}
			choices[index] = intake.Choice{
				QuestionRef: decision.QuestionRef,
				OptionRef:   decision.Choice,
				AnswerText:  decision.AnswerText,
			}
		}
		next, applyErr := intake.Apply(state, intake.Change{
			StateRef:         snapshot.Ref,
			ExpectedRevision: state.Revision(),
			Origin:           mutation.Origin,
			Derivation:       mutation.Derivation,
			Issues: append(
				[]intake.Issue(nil),
				snapshot.Issues[issueOffset:issueOffset+mutation.IssuesAdded]...,
			),
			Questions: cloneSnapshotQuestions(
				snapshot.Questions[questionOffset : questionOffset+mutation.QuestionsAdded],
			),
			Choices: choices,
		})
		if applyErr != nil {
			return intake.State{}, invalidIntakeSnapshotWithCause(applyErr)
		}
		gotHistory := next.History()
		if len(gotHistory) == 0 || !reflect.DeepEqual(gotHistory[len(gotHistory)-1], mutation) {
			return intake.State{}, invalidIntakeSnapshot()
		}
		state = next
		issueOffset += mutation.IssuesAdded
		questionOffset += mutation.QuestionsAdded
		decisionOffset += mutation.ChoicesRecorded
	}
	if issueOffset != len(snapshot.Issues) ||
		questionOffset != len(snapshot.Questions) ||
		decisionOffset != len(snapshot.Decisions) ||
		!reflect.DeepEqual(SnapshotIntake(state), snapshot) {
		return intake.State{}, invalidIntakeSnapshot()
	}
	return state, nil
}

func canonicalIntakeSnapshot(snapshot IntakeSnapshot) IntakeSnapshot {
	if len(snapshot.Issues) == 0 {
		snapshot.Issues = nil
	}
	if len(snapshot.Questions) == 0 {
		snapshot.Questions = nil
	}
	if len(snapshot.Decisions) == 0 {
		snapshot.Decisions = nil
	}
	if len(snapshot.History) == 0 {
		snapshot.History = nil
	}
	return snapshot
}

func cloneSnapshotQuestions(values []intake.Question) []intake.Question {
	result := make([]intake.Question, len(values))
	for index, question := range values {
		result[index] = question
		result[index].DerivedFrom = append([]intake.IssueRef(nil), question.DerivedFrom...)
		result[index].Options = append([]intake.Option(nil), question.Options...)
	}
	return result
}

func invalidIntakeSnapshot() error {
	return &StateError{Code: StateInvalid, Cause: errIntakeSnapshotInvalid}
}

func invalidIntakeSnapshotWithCause(cause error) error {
	return &StateError{Code: StateInvalid, Cause: errors.Join(errIntakeSnapshotInvalid, cause)}
}
