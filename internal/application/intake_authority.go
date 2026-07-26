package application

import "errors"

// IntakeAuthorizationRequestRef binds one authorization decision to exactly
// one intake mutation request.
func IntakeAuthorizationRequestRef(
	operation IntakeOperation,
	mutationRequestRef string,
) (string, error) {
	if !validApplicationRef(mutationRequestRef) {
		return "", errors.New("application.request_ref_invalid")
	}
	switch operation {
	case IntakeOperationCreate, IntakeOperationApply:
		return "authorization-request:intake-" + string(operation) + ":" + mutationRequestRef, nil
	default:
		return "", errors.New("application.intake_operation_invalid")
	}
}
