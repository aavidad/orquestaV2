package application

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/intake"
)

func TestValidateIntakeChainReplaysExactOrderedHistory(t *testing.T) {
	records := validIntakeChain(t)
	if err := ValidateIntakeChain(records); err != nil {
		t.Fatal(err)
	}
}

func TestValidateIntakeChainRejectsHistoricalBranch(t *testing.T) {
	records := validIntakeChain(t)
	alternateChange := intakeQuestionChange(1, intake.OriginForm)
	alternate, err := intake.Apply(records[0].State, alternateChange)
	if err != nil {
		t.Fatal(err)
	}
	records[1] = intakeChainApplyRecord(
		t,
		records[0],
		alternate,
		alternateChange,
		records[1].Receipt.RequestRef,
		records[1].Receipt.AuthorizationReceiptRef,
	)

	if err := ValidateIntakeChain(records); !IsStateError(err, StateInvalid) {
		t.Fatalf("historical branch err = %v", err)
	}
}

func TestValidateIntakeChainRejectsRewrittenFingerprintAndReceiptRef(t *testing.T) {
	records := validIntakeChain(t)
	tampered := records[1]
	tampered.Receipt.RequestFingerprint = strings.Repeat("a", 64)
	receipt, err := buildIntakeReceipt(
		tampered.Receipt.Operation,
		tampered.Receipt.RequestRef,
		tampered.Receipt.RequestFingerprint,
		tampered.ActorRef,
		tampered.ProjectRef,
		tampered.State,
		tampered.Receipt.PreviousRevision,
		tampered.Receipt.AuthorizationReceiptRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	tampered.Receipt = receipt
	records[1] = tampered

	if err := ValidateIntakeChain(records); !IsStateError(err, StateInvalid) {
		t.Fatalf("rewritten fingerprint and receipt err = %v", err)
	}
}

func TestValidateIntakeChainRejectsEmptyDuplicateAndOverflow(t *testing.T) {
	if err := ValidateIntakeChain(nil); !IsStateError(err, StateInvalid) {
		t.Fatalf("empty chain err = %v", err)
	}

	records := validIntakeChain(t)
	records[0].ActorRef = goal.ActorRef{}
	if err := ValidateIntakeChain(records); !IsStateError(err, StateInvalid) {
		t.Fatalf("empty actor scope err = %v", err)
	}

	records = validIntakeChain(t)
	records[2].Receipt.RequestRef = records[1].Receipt.RequestRef
	if err := ValidateIntakeChain(records); !IsStateError(err, StateInvalid) {
		t.Fatalf("duplicate request err = %v", err)
	}

	records = validIntakeChain(t)
	records[1].Receipt.PreviousRevision = intake.Revision(math.MaxUint64)
	if err := ValidateIntakeChain(records); !IsStateError(err, StateInvalid) {
		t.Fatalf("overflow revision err = %v", err)
	}
}

func validIntakeChain(t *testing.T) []IntakeRecord {
	t.Helper()
	system := newIntakeTestSystem(t)
	created := mustCreateIntake(t, system)
	firstRequest := ApplyIntakeRequest{
		RequestRef:           "request:intake-chain-chat",
		ActorRef:             system.actor,
		ProjectRef:           system.project,
		Change:               intakeQuestionChange(1, intake.OriginChat),
		AuthorizationReceipt: system.authorization,
	}
	first, err := system.service.ApplyIntake(context.Background(), firstRequest)
	if err != nil {
		t.Fatal(err)
	}
	second, err := system.service.ApplyIntake(context.Background(), ApplyIntakeRequest{
		RequestRef: "request:intake-chain-form",
		ActorRef:   system.actor,
		ProjectRef: system.project,
		Change: intake.Change{
			StateRef:         first.Record.State.Ref(),
			ExpectedRevision: first.Record.State.Revision(),
			Origin:           intake.OriginForm,
			Choices: []intake.Choice{{
				QuestionRef: "intake-question:audience",
				OptionRef:   "intake-option:audience-personal",
			}},
		},
		AuthorizationReceipt: system.authorization,
	})
	if err != nil {
		t.Fatal(err)
	}
	return []IntakeRecord{created.Record, first.Record, second.Record}
}

func intakeChainApplyRecord(
	t *testing.T,
	previous IntakeRecord,
	state intake.State,
	change intake.Change,
	requestRef string,
	authorizationReceiptRef string,
) IntakeRecord {
	t.Helper()
	encoded, err := json.Marshal(change)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint := fingerprintFields(
		"orquesta.intake.apply.v1",
		previous.ActorRef.String(),
		previous.ProjectRef.String(),
		string(encoded),
		authorizationReceiptRef,
	)
	receipt, err := buildIntakeReceipt(
		IntakeOperationApply,
		requestRef,
		fingerprint,
		previous.ActorRef,
		previous.ProjectRef,
		state,
		previous.State.Revision(),
		authorizationReceiptRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	return IntakeRecord{
		ActorRef:   previous.ActorRef,
		ProjectRef: previous.ProjectRef,
		State:      state,
		Receipt:    receipt,
	}
}
