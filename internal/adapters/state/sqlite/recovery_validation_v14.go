package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"orquesta/internal/application"
)

type recoveryControlEffects struct {
	actions int
	effects int
}

func validateRecoveryV14Controls(ctx context.Context, transaction *sql.Tx) error {
	controls, err := readRecoveryV14Controls(ctx, transaction)
	if err != nil {
		return err
	}
	if err := validateRecoveryV14StopActions(ctx, transaction, controls); err != nil {
		return err
	}
	effects, err := validateRecoveryV14StopEffects(ctx, transaction, controls)
	if err != nil {
		return err
	}
	if err := validateRecoveryV14StopSupersessions(ctx, transaction, controls); err != nil {
		return err
	}
	for ref, control := range controls {
		counts := effects[ref]
		switch control.Operation {
		case application.ControlStop:
			if counts.actions != 1 ||
				(control.Status == application.ControlConfirmed && counts.effects != 1) ||
				((control.Status == application.ControlRequested ||
					control.Status == application.ControlSuperseded) && counts.effects != 0) {
				return fmt.Errorf("sqlite.recovery_control_stop_lifecycle_invalid:%s", ref)
			}
		case application.ControlCancel:
			if control.Status == application.ControlRequested && counts.actions == 0 {
				return fmt.Errorf("sqlite.recovery_control_cancel_lifecycle_invalid:%s", ref)
			}
			if control.Status == application.ControlConfirmed && control.Target == application.ControlTargetGoal &&
				control.ReceiptRef != "receipt:"+control.Ref {
				return fmt.Errorf("sqlite.recovery_control_cancel_receipt_invalid:%s", ref)
			}
			if counts.actions == 0 && !localControlConfirmation(control) {
				return fmt.Errorf("sqlite.recovery_control_cancel_local_invalid:%s", ref)
			}
			if err := validateRecoveryV14CancelCoverage(ctx, transaction, control); err != nil {
				return err
			}
		default:
			if counts.actions != 0 || !localControlConfirmation(control) {
				return fmt.Errorf("sqlite.recovery_control_local_invalid:%s", ref)
			}
		}
	}
	return validateRecoveryV14MailboxRetirements(ctx, transaction)
}

func validateRecoveryV14MailboxRetirements(ctx context.Context, transaction *sql.Tx) error {
	var invalidRetirements int
	if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM executions execution
WHERE execution.recipient_mailbox_retired <>
      EXISTS (SELECT 1 FROM mailbox_retirements retirement
              WHERE retirement.recipient_execution_ref = execution.ref)`,
	).Scan(&invalidRetirements); err != nil {
		return err
	}
	if invalidRetirements != 0 {
		return errors.New("sqlite.recovery_execution_mailbox_marker_invalid")
	}
	return nil
}

func localControlConfirmation(control application.ControlRecord) bool {
	return control.Status == application.ControlConfirmed &&
		control.ReceiptRef == "receipt:"+control.Ref &&
		control.ConfirmedAt.Equal(control.RequestedAt)
}
