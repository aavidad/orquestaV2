package application

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/intake"
)

func TestIntakeDerivationSurvivesSnapshotReplayAndValidatedChain(t *testing.T) {
	system := newIntakeTestSystem(t)
	created := mustCreateIntake(t, system)
	identity, err := intake.NewDerivationIdentity(
		"orquesta.test.deriver",
		"v1",
		strings.Repeat("b", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	request := ApplyIntakeRequest{
		RequestRef: "request:intake-derived",
		ActorRef:   system.actor,
		ProjectRef: system.project,
		Change:     intakeQuestionChange(1, intake.OriginChat),
		AuthorizationReceipt: system.authorizationFor(
			t, IntakeOperationApply, "request:intake-derived",
		),
	}
	request.Change.Derivation = identity
	first, err := system.service.ApplyIntake(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := SnapshotIntake(first.Record.State)
	restored, err := RestoreIntake(snapshot)
	if err != nil || !reflect.DeepEqual(SnapshotIntake(restored), snapshot) {
		t.Fatalf("restore err=%v restored=%+v", err, SnapshotIntake(restored))
	}
	history := restored.History()
	if len(history) != 1 || history[0].Derivation != identity {
		t.Fatalf("history=%+v", history)
	}
	if err = ValidateIntakeChain([]IntakeRecord{
		created.Record,
		first.Record,
	}); err != nil {
		t.Fatal(err)
	}

	replayed, err := system.service.ApplyIntake(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Changed || replayed.Record.Receipt != first.Record.Receipt {
		t.Fatalf(
			"first=%+v replayed=%+v",
			first.Record.Receipt,
			replayed.Record.Receipt,
		)
	}
}

func TestIntakeChainRejectsRewrittenDerivationIdentity(t *testing.T) {
	system := newIntakeTestSystem(t)
	created := mustCreateIntake(t, system)
	firstIdentity, err := intake.NewDerivationIdentity(
		"orquesta.test.deriver",
		"v1",
		strings.Repeat("c", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	change := intakeQuestionChange(1, intake.OriginChat)
	change.Derivation = firstIdentity
	first, err := system.service.ApplyIntake(context.Background(), ApplyIntakeRequest{
		RequestRef: "request:intake-derived-tamper",
		ActorRef:   system.actor,
		ProjectRef: system.project,
		Change:     change,
		AuthorizationReceipt: system.authorizationFor(
			t, IntakeOperationApply, "request:intake-derived-tamper",
		),
	})
	if err != nil {
		t.Fatal(err)
	}

	tamperedSnapshot := SnapshotIntake(first.Record.State)
	secondIdentity, err := intake.NewDerivationIdentity(
		"orquesta.test.deriver",
		"v2",
		strings.Repeat("d", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	tamperedSnapshot.History[0].Derivation = secondIdentity
	tamperedState, err := RestoreIntake(tamperedSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	tampered := first.Record
	tampered.State = tamperedState
	if err = ValidateIntakeChain([]IntakeRecord{
		created.Record,
		tampered,
	}); !IsStateError(err, StateInvalid) {
		t.Fatalf("rewritten derivation err=%v", err)
	}
}
