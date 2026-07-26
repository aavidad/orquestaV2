package application

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strconv"

	"orquesta/internal/intake"
)

var errIntakeChainInvalid = errors.New("application.intake_chain_invalid")

// ValidateIntakeChain validates one ordered actor/project/state history.
// Recovery adapters pass every immutable receipt snapshot for the scope,
// ordered by revision. The application replays each domain transition instead
// of trusting adapter-owned prefix or delta checks.
func ValidateIntakeChain(records []IntakeRecord) error {
	if len(records) == 0 {
		return invalidIntakeChain(nil)
	}

	first := records[0]
	if err := validateIntakeScope(first.ActorRef, first.ProjectRef); err != nil {
		return invalidIntakeChain(err)
	}
	pristine, err := intake.NewState(first.State.Ref(), first.State.Policy())
	if err != nil || !reflectIntakeStateEqual(first.State, pristine) {
		return invalidIntakeChain(err)
	}
	createFingerprint := fingerprintFields(
		"orquesta.intake.create.v1",
		first.ActorRef.String(),
		first.ProjectRef.String(),
		string(first.State.Ref()),
		strconv.FormatUint(uint64(first.State.Policy().MaxQuestionRounds), 10),
		first.Receipt.AuthorizationReceiptRef,
	)
	if err := validateIntakeChainRecord(
		first,
		IntakeOperationCreate,
		0,
		createFingerprint,
	); err != nil {
		return invalidIntakeChain(err)
	}

	requestRefs := map[string]struct{}{first.Receipt.RequestRef: {}}
	previous := first
	for _, record := range records[1:] {
		if previous.State.Revision() == intake.Revision(math.MaxUint64) {
			return invalidIntakeChain(nil)
		}
		if record.ActorRef != first.ActorRef ||
			record.ProjectRef != first.ProjectRef ||
			record.State.Ref() != first.State.Ref() ||
			record.State.Revision() != previous.State.Revision()+1 {
			return invalidIntakeChain(nil)
		}
		if _, duplicate := requestRefs[record.Receipt.RequestRef]; duplicate {
			return invalidIntakeChain(nil)
		}

		change, err := intakeChainChange(previous.State, record.State)
		if err != nil {
			return invalidIntakeChain(err)
		}
		replayed, err := intake.Apply(previous.State, change)
		if err != nil || !reflectIntakeStateEqual(replayed, record.State) {
			return invalidIntakeChain(err)
		}
		encoded, err := json.Marshal(change)
		if err != nil {
			return invalidIntakeChain(err)
		}
		applyFingerprint := fingerprintFields(
			"orquesta.intake.apply.v1",
			record.ActorRef.String(),
			record.ProjectRef.String(),
			string(encoded),
			record.Receipt.AuthorizationReceiptRef,
		)
		if err := validateIntakeChainRecord(
			record,
			IntakeOperationApply,
			previous.State.Revision(),
			applyFingerprint,
		); err != nil {
			return invalidIntakeChain(err)
		}

		requestRefs[record.Receipt.RequestRef] = struct{}{}
		previous = record
	}
	return nil
}

func validateIntakeChainRecord(
	record IntakeRecord,
	operation IntakeOperation,
	previousRevision intake.Revision,
	requestFingerprint string,
) error {
	receipt := record.Receipt
	if !validIntakeRequestRef(receipt.RequestRef) ||
		receipt.Operation != operation ||
		receipt.PreviousRevision != previousRevision ||
		receipt.Revision != record.State.Revision() ||
		receipt.RequestFingerprint != requestFingerprint {
		return errIntakeChainInvalid
	}
	if err := validateStoredIntakeRecord(
		record.ActorRef,
		record.ProjectRef,
		record.State.Ref(),
		record,
	); err != nil {
		return err
	}
	return nil
}

func intakeChainChange(previous, next intake.State) (intake.Change, error) {
	previousSnapshot := SnapshotIntake(previous)
	nextSnapshot := SnapshotIntake(next)
	if previousSnapshot.Schema != nextSnapshot.Schema ||
		previousSnapshot.Ref != nextSnapshot.Ref ||
		previousSnapshot.Policy != nextSnapshot.Policy ||
		len(nextSnapshot.History) <= len(previousSnapshot.History) ||
		len(nextSnapshot.History)-len(previousSnapshot.History) != 1 ||
		!intakeChainPrefix(nextSnapshot.History, previousSnapshot.History) ||
		!intakeChainPrefix(nextSnapshot.Issues, previousSnapshot.Issues) ||
		!intakeChainQuestionPrefix(nextSnapshot.Questions, previousSnapshot.Questions) ||
		!intakeChainPrefix(nextSnapshot.Decisions, previousSnapshot.Decisions) {
		return intake.Change{}, errIntakeChainInvalid
	}

	mutation := nextSnapshot.History[len(nextSnapshot.History)-1]
	issuesAdded := len(nextSnapshot.Issues) - len(previousSnapshot.Issues)
	questionsAdded := len(nextSnapshot.Questions) - len(previousSnapshot.Questions)
	choicesRecorded := len(nextSnapshot.Decisions) - len(previousSnapshot.Decisions)
	if mutation.Revision != nextSnapshot.Revision ||
		mutation.IssuesAdded != issuesAdded ||
		mutation.QuestionsAdded != questionsAdded ||
		mutation.ChoicesRecorded != choicesRecorded {
		return intake.Change{}, errIntakeChainInvalid
	}

	decisions := nextSnapshot.Decisions[len(previousSnapshot.Decisions):]
	choices := make([]intake.Choice, len(decisions))
	for index, decision := range decisions {
		choices[index] = intake.Choice{
			QuestionRef: decision.QuestionRef,
			OptionRef:   decision.Choice,
			AnswerText:  decision.AnswerText,
		}
	}
	return intake.Change{
		StateRef:         previousSnapshot.Ref,
		ExpectedRevision: previousSnapshot.Revision,
		Origin:           mutation.Origin,
		Issues: append(
			[]intake.Issue(nil),
			nextSnapshot.Issues[len(previousSnapshot.Issues):]...,
		),
		Questions: cloneSnapshotQuestions(
			nextSnapshot.Questions[len(previousSnapshot.Questions):],
		),
		Choices: choices,
	}, nil
}

func intakeChainPrefix[T comparable](values, prefix []T) bool {
	if len(values) < len(prefix) {
		return false
	}
	for index := range prefix {
		if values[index] != prefix[index] {
			return false
		}
	}
	return true
}

func intakeChainQuestionPrefix(values, prefix []intake.Question) bool {
	if len(values) < len(prefix) {
		return false
	}
	for index := range prefix {
		if !reflect.DeepEqual(values[index], prefix[index]) {
			return false
		}
	}
	return true
}

func invalidIntakeChain(cause error) error {
	if cause == nil {
		cause = errIntakeChainInvalid
	} else {
		cause = errors.Join(errIntakeChainInvalid, cause)
	}
	return &StateError{Code: StateInvalid, Cause: cause}
}
