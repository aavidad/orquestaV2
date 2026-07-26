package application

import (
	"strings"
	"testing"

	"orquesta/internal/intake"
)

func TestIntakeAuthorizationRequestRefBindsOperationAndMutation(t *testing.T) {
	tests := []struct {
		operation  IntakeOperation
		requestRef string
		want       string
	}{
		{
			operation:  IntakeOperationCreate,
			requestRef: "request:intake-create",
			want:       "authorization-request:intake-create:request:intake-create",
		},
		{
			operation:  IntakeOperationApply,
			requestRef: "request:intake-apply",
			want:       "authorization-request:intake-apply:request:intake-apply",
		},
	}
	for _, test := range tests {
		got, err := IntakeAuthorizationRequestRef(test.operation, test.requestRef)
		if err != nil || got != test.want {
			t.Fatalf("ref(%q, %q) = %q, %v; want %q",
				test.operation, test.requestRef, got, err, test.want)
		}
	}
}

func TestIntakeAuthorizationRequestRefRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		operation  IntakeOperation
		requestRef string
	}{
		{operation: IntakeOperation("delete"), requestRef: "request:intake"},
		{operation: IntakeOperationCreate, requestRef: ""},
		{operation: IntakeOperationApply, requestRef: " request:intake"},
		{operation: IntakeOperationApply, requestRef: strings.Repeat("a", 513)},
		{operation: IntakeOperationApply, requestRef: "request:\x00intake"},
		{operation: IntakeOperationApply, requestRef: "request:\nintake"},
		{operation: IntakeOperationApply, requestRef: "request:\rintake"},
	}
	for _, test := range tests {
		if got, err := IntakeAuthorizationRequestRef(test.operation, test.requestRef); err == nil ||
			got != "" || intake.ErrorCodeOf(err) != intake.ErrorInvalidArgument {
			t.Fatalf("ref(%q, %q) = %q, %v", test.operation, test.requestRef, got, err)
		}
	}
}
