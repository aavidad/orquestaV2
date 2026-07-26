package sqlite

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/intake"
)

func TestV23RecoveryAcceptsDurableIntakeChain(t *testing.T) {
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:v23-recovery-create")
	first := system.apply(
		t, created.Record.State, "request:v23-recovery-first", intake.OriginChat,
		intake.OptionRef("intake-option:audience-personal"),
	)
	_ = system.apply(
		t, first.Record.State, "request:v23-recovery-second", intake.OriginForm,
		intake.OptionRef("intake-option:audience-team"),
	)

	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("valid V23 intake recovery: %s", intakeTestErrorChain(err))
	}
}

func TestV23RecoveryRejectsDivergentIntakeBranch(t *testing.T) {
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:v23-branch-create")
	first := system.apply(
		t, created.Record.State, "request:v23-branch-first", intake.OriginChat,
		intake.OptionRef("intake-option:audience-personal"),
	)
	_ = system.apply(
		t, first.Record.State, "request:v23-branch-second", intake.OriginForm,
		intake.OptionRef("intake-option:audience-team"),
	)

	change := sqliteIntakeAudienceChange(
		created.Record.State, system.stateRef, intake.OriginChat,
		intake.OptionRef("intake-option:audience-team"),
	)
	alternative, err := intake.Apply(created.Record.State, change)
	sqliteTestNoError(t, err)
	encodedChange, err := json.Marshal(change)
	sqliteTestNoError(t, err)
	alternativeReceipt := first.Record.Receipt
	alternativeReceipt.RequestFingerprint = canonicalFingerprint(
		"orquesta.intake.apply.v1",
		system.principal.ActorRef.String(),
		system.project.String(),
		string(encodedChange),
		alternativeReceipt.AuthorizationReceiptRef,
	)
	alternativeReceipt.StateDigest, err = application.IntakeStateDigest(alternative)
	sqliteTestNoError(t, err)
	alternativeReceipt.Ref = v23TestIntakeReceiptRef(alternativeReceipt)
	alternativeSnapshot, err := encodeIntakeState(alternative)
	sqliteTestNoError(t, err)

	rewriteRecoveryTrigger(
		t, system.repository.db, "intake_receipts_immutable_update", func() {
			mustV10Exec(t, system.repository.db, `
UPDATE intake_receipts
SET ref=?, request_fingerprint=?, state_digest=?, snapshot_json=?
WHERE ref=?`,
				alternativeReceipt.Ref, alternativeReceipt.RequestFingerprint,
				alternativeReceipt.StateDigest, string(alternativeSnapshot),
				first.Record.Receipt.Ref,
			)
		},
	)
	requireV23IntakeRecoveryError(t, system, "sqlite.recovery_v23_intake_chain_invalid")
}

func TestV23RecoveryRejectsRewrittenIntakeRequestFingerprint(t *testing.T) {
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:v23-fingerprint-create")
	first := system.apply(
		t, created.Record.State, "request:v23-fingerprint-first", intake.OriginChat,
		intake.OptionRef("intake-option:audience-personal"),
	)
	_ = system.apply(
		t, first.Record.State, "request:v23-fingerprint-second", intake.OriginForm,
		intake.OptionRef("intake-option:audience-team"),
	)

	rewritten := first.Record.Receipt
	rewritten.RequestFingerprint = strings.Repeat("0", 64)
	rewritten.Ref = v23TestIntakeReceiptRef(rewritten)
	rewriteRecoveryTrigger(
		t, system.repository.db, "intake_receipts_immutable_update", func() {
			mustV10Exec(t, system.repository.db, `
UPDATE intake_receipts
SET ref=?, request_fingerprint=?
WHERE ref=?`,
				rewritten.Ref, rewritten.RequestFingerprint, first.Record.Receipt.Ref,
			)
		},
	)
	requireV23IntakeRecoveryError(t, system, "sqlite.recovery_v23_intake_chain_invalid")
}

func TestV23RecoveryRejectsSubstitutedIntakeAuthorization(t *testing.T) {
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:v23-auth-create")
	first := system.apply(
		t, created.Record.State, "request:v23-auth-first", intake.OriginChat,
		intake.OptionRef("intake-option:audience-personal"),
	)
	_ = system.apply(
		t, first.Record.State, "request:v23-auth-second", intake.OriginForm,
		intake.OptionRef("intake-option:audience-team"),
	)
	substitute := system.authorizeIntake(
		t, application.IntakeOperationApply, "request:v23-auth-foreign",
	)
	change := sqliteIntakeAudienceChange(
		created.Record.State, system.stateRef, intake.OriginChat,
		intake.OptionRef("intake-option:audience-personal"),
	)
	encodedChange, err := json.Marshal(change)
	sqliteTestNoError(t, err)
	rewritten := first.Record.Receipt
	rewritten.AuthorizationReceiptRef = substitute.Ref()
	rewritten.RequestFingerprint = canonicalFingerprint(
		"orquesta.intake.apply.v1",
		system.principal.ActorRef.String(),
		system.project.String(),
		string(encodedChange),
		rewritten.AuthorizationReceiptRef,
	)
	rewritten.Ref = v23TestIntakeReceiptRef(rewritten)
	rewriteRecoveryTrigger(
		t, system.repository.db, "intake_receipts_immutable_update", func() {
			mustV10Exec(t, system.repository.db, `
UPDATE intake_receipts
SET ref=?, request_fingerprint=?, authorization_receipt_ref=?
WHERE ref=?`,
				rewritten.Ref, rewritten.RequestFingerprint,
				rewritten.AuthorizationReceiptRef, first.Record.Receipt.Ref,
			)
		},
	)
	requireV23IntakeRecoveryError(
		t, system, "sqlite.recovery_v23_intake_authorization_invalid",
	)
}

func TestV23RecoveryRejectsIntakeRevisionGap(t *testing.T) {
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:v23-gap-create")
	first := system.apply(
		t, created.Record.State, "request:v23-gap-first", intake.OriginChat,
		intake.OptionRef("intake-option:audience-personal"),
	)
	_ = system.apply(
		t, first.Record.State, "request:v23-gap-second", intake.OriginForm,
		intake.OptionRef("intake-option:audience-team"),
	)

	rewriteRecoveryTrigger(
		t, system.repository.db, "intake_receipts_immutable_delete", func() {
			mustV10Exec(
				t, system.repository.db, `DELETE FROM intake_receipts WHERE ref=?`,
				first.Record.Receipt.Ref,
			)
		},
	)
	requireV23IntakeRecoveryError(t, system, "sqlite.recovery_v23_intake_chain_invalid")
}

func requireV23IntakeRecoveryError(
	t *testing.T,
	system *sqliteIntakeTestSystem,
	code string,
) {
	t.Helper()
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil ||
		!recoveryErrorContains(err, code) {
		t.Fatalf("invalid V23 intake passed recovery: %s", intakeTestErrorChain(err))
	}
}

func v23TestIntakeReceiptRef(receipt application.IntakeReceipt) string {
	return intakeReceiptPrefix + canonicalFingerprint(
		"orquesta.intake.receipt.v1",
		string(receipt.Operation),
		receipt.ActorRef.String(),
		receipt.ProjectRef.String(),
		receipt.RequestRef,
		receipt.RequestFingerprint,
		string(receipt.StateRef),
		strconv.FormatUint(uint64(receipt.PreviousRevision), 10),
		strconv.FormatUint(uint64(receipt.Revision), 10),
		receipt.StateDigest,
		receipt.AuthorizationReceiptRef,
	)
}
