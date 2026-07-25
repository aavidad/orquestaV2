package ports

import "testing"

func TestTestAttestorContractErrorExposesStableCauseCode(t *testing.T) {
	err := &TestAttestorContractError{Code: "test_attestor.subject_digest_mismatch"}
	if err.Error() != err.Code || err.CauseCode() != err.Code {
		t.Fatalf("error=%q cause=%q", err.Error(), err.CauseCode())
	}
	var nilError *TestAttestorContractError
	if nilError.CauseCode() != "" {
		t.Fatalf("nil cause=%q", nilError.CauseCode())
	}
}
