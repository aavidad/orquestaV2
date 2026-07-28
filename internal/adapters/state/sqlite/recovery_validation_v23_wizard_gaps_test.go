package sqlite

import (
	"context"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/intake"
)

func TestV23WizardGapsRecoveryRejectsCorruptedNoOpOutcomes(t *testing.T) {
	tests := []struct {
		name     string
		wantCode string
		mutate   func(*testing.T, *sqliteIntakeTestSystem, application.ApplyWizardGapsResult)
	}{
		{
			name:     "outcome ref",
			wantCode: "sqlite.recovery_v23_wizard_gaps_outcome_invalid",
			mutate: func(
				t *testing.T,
				system *sqliteIntakeTestSystem,
				noOp application.ApplyWizardGapsResult,
			) {
				t.Helper()
				altered := mutateWizardGapsOutcomeRef(
					noOp.RequestOutcome.ReceiptRef,
				)
				mustV10Exec(
					t,
					system.repository.db,
					`UPDATE wizard_gaps_outcomes SET ref=? WHERE ref=?`,
					altered,
					noOp.RequestOutcome.ReceiptRef,
				)
			},
		},
		{
			name:     "evaluation digest",
			wantCode: "sqlite.recovery_v23_wizard_gaps_outcome_invalid",
			mutate: func(
				t *testing.T,
				system *sqliteIntakeTestSystem,
				noOp application.ApplyWizardGapsResult,
			) {
				t.Helper()
				mustV10Exec(
					t,
					system.repository.db,
					`UPDATE wizard_gaps_outcomes SET evaluation_digest=? WHERE ref=?`,
					strings.Repeat("0", 64),
					noOp.RequestOutcome.ReceiptRef,
				)
			},
		},
		{
			name:     "authorization binding",
			wantCode: "sqlite.recovery_v23_wizard_gaps_binding_invalid",
			mutate: func(
				t *testing.T,
				system *sqliteIntakeTestSystem,
				noOp application.ApplyWizardGapsResult,
			) {
				t.Helper()
				foreign := system.authorizeIntake(
					t,
					application.IntakeOperationApply,
					"request:wizard-gaps-recovery-foreign-authorization",
				)
				mustV10Exec(
					t,
					system.repository.db,
					`UPDATE wizard_gaps_outcomes
SET authorization_receipt_ref=?
WHERE ref=?`,
					foreign.Ref(),
					noOp.RequestOutcome.ReceiptRef,
				)
			},
		},
		{
			name:     "source receipt binding",
			wantCode: "sqlite.recovery_v23_wizard_gaps_binding_invalid",
			mutate: func(
				t *testing.T,
				system *sqliteIntakeTestSystem,
				noOp application.ApplyWizardGapsResult,
			) {
				t.Helper()
				mustV10Exec(
					t,
					system.repository.db,
					`UPDATE wizard_gaps_outcomes
SET source_intake_receipt_ref=(
    SELECT ref
    FROM intake_receipts
    WHERE actor_ref=? AND project_ref=? AND state_ref=? AND revision=1
)
WHERE ref=?`,
					system.principal.ActorRef.String(),
					system.project.String(),
					system.stateRef,
					noOp.RequestOutcome.ReceiptRef,
				)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			system, service, seed := newSQLiteWizardGapsNoOpTestSystem(t)
			noOp, err := service.ApplyWizardGaps(
				context.Background(),
				sqliteWizardGapsRequest(
					t,
					system,
					"request:wizard-gaps-recovery-noop",
					seed.Record.State.Revision(),
					intake.OriginForm,
				),
			)
			sqliteTestNoError(t, err)
			if noOp.RequestOutcome.Kind !=
				application.WizardGapsRequestOutcomeNoOp {
				t.Fatalf("no-op=%+v", noOp)
			}
			rewriteRecoveryTrigger(
				t,
				system.repository.db,
				"wizard_gaps_outcomes_immutable_update",
				func() { test.mutate(t, system, noOp) },
			)
			if _, _, err := validateRecoveryDatabase(
				context.Background(),
				system.repository.db,
			); err == nil || !recoveryErrorContains(err, test.wantCode) {
				t.Fatalf(
					"corruption passed recovery: want=%s err=%s",
					test.wantCode,
					intakeTestErrorChain(err),
				)
			}
		})
	}
}

func mutateWizardGapsOutcomeRef(value string) string {
	last := value[len(value)-1]
	if last == '0' {
		return value[:len(value)-1] + "1"
	}
	return value[:len(value)-1] + "0"
}
