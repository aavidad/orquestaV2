package application

import (
	"strings"

	"orquesta/internal/intake"
)

// IntakeAuthorizationRequestRef binds one authorization decision to exactly
// one intake mutation request.
func IntakeAuthorizationRequestRef(
	operation IntakeOperation,
	mutationRequestRef string,
) (string, error) {
	if !validIntakeRequestRef(mutationRequestRef) {
		return "", invalidIntakeRequestRefError()
	}
	switch operation {
	case IntakeOperationCreate, IntakeOperationApply:
		return "authorization-request:intake-" + string(operation) + ":" + mutationRequestRef, nil
	default:
		return "", &intake.DomainError{
			Code: intake.ErrorInvalidArgument, Field: "operation",
		}
	}
}

func validIntakeRequestRef(value string) bool {
	return validApplicationRef(value) && len(value) <= 512 &&
		!strings.ContainsAny(value, "\x00\n\r")
}

func invalidIntakeRequestRefError() error {
	return &intake.DomainError{
		Code: intake.ErrorInvalidArgument, Field: "request_ref",
	}
}
