package bubblewrap

import "testing"

func TestErrorExposesStableCauseCode(t *testing.T) {
	err := &Error{Code: CodeSnapshotLimit}
	if err.Error() != CodeSnapshotLimit || err.CauseCode() != CodeSnapshotLimit {
		t.Fatalf("error=%q cause=%q", err.Error(), err.CauseCode())
	}
	var nilError *Error
	if nilError.CauseCode() != "" {
		t.Fatalf("nil cause=%q", nilError.CauseCode())
	}
}
