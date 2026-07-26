package application

import "testing"

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
	}
	for _, test := range tests {
		if got, err := IntakeAuthorizationRequestRef(test.operation, test.requestRef); err == nil || got != "" {
			t.Fatalf("ref(%q, %q) = %q, %v", test.operation, test.requestRef, got, err)
		}
	}
}
