// Este fichero asigna códigos máquina estables sin revelar rutas físicas.
package main

import (
	"errors"
	"fmt"
)

var (
	errInvalidInput        = errors.New("invalid_input")
	errUnexpectedArguments = errors.New("unexpected_positional_arguments")
	errRootAnchor          = errors.New("root_anchor_failed")
	errPublication         = errors.New("publication_failed")
	errNoAtime             = errors.New("noatime_required")
	errMountIDUnavailable  = errors.New("mount_id_unavailable")
	errOutputDirectory     = errors.New("private_output_directory_required")
	errOutputLocked        = errors.New("output_locked")
	errOutputNameExists    = errors.New("output_name_exists")
	errOutputBudget        = errors.New("output_budget_exhausted")
	errInvalidManifestSeal = errors.New("invalid_manifest_seal")
	errRecoveryConflict    = errors.New("recovery_conflict")
	errStageReplaced       = errors.New("stage_replaced")
	errOutputPath          = errors.New("invalid_output_path")
	errPhysicalOverlap     = errors.New("physical_path_overlap")
)

type recoveryError struct {
	cause, recovery error
	journal         string
}

func (err *recoveryError) Error() string {
	return fmt.Sprintf("recovery_failed journal=%q: %v: %v", err.journal, err.cause, err.recovery)
}
func (err *recoveryError) Unwrap() []error {
	return []error{err.cause, err.recovery}
}
func isRecoveryError(err error) bool {
	var target *recoveryError
	return errors.As(err, &target)
}
