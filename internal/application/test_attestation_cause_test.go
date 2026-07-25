package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"orquesta/internal/ports"
)

type codedTestAttestorFailure struct {
	message string
	code    string
}

func (err *codedTestAttestorFailure) Error() string     { return err.message }
func (err *codedTestAttestorFailure) CauseCode() string { return err.code }

func TestAttestFailurePreservesTypedCauseWithoutChangingDurableQuarantine(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	raw := &codedTestAttestorFailure{
		message: "private attestor detail /tmp/must-not-leak",
		code:    "test_attestor.snapshot_limit_exceeded",
	}
	system.attestor.override = func(ports.TestAttestationRequest) (ports.TestAttestationResult, error) {
		return ports.TestAttestationResult{}, raw
	}

	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:test-attestation-cause")
	if err == nil || err.Error() != effectUnknownAppliedCode ||
		!result.Processed || result.Action != ActionAttestTest {
		t.Fatalf("attestation failure result=%+v err=%v", result, err)
	}
	var caused interface{ CauseCode() string }
	if !errors.As(err, &caused) || caused.CauseCode() != raw.code {
		t.Fatalf("typed cause missing: err=%T %v cause=%v", err, err, caused)
	}
	if errors.Is(err, raw) || strings.Contains(err.Error(), raw.message) {
		t.Fatalf("private adapter error escaped: %T %v", err, err)
	}

	record := system.record(t)
	assertUnknownAppliedConsumption(t, record, ActionAttestTest)
	for _, receipt := range record.ConsumptionReceipts {
		if receipt.Kind == ActionAttestTest && receipt.ErrorCode != effectUnknownAppliedCode {
			t.Fatalf("durable cause changed: %+v", receipt)
		}
	}
	for _, attestation := range record.Attestations {
		if attestation.Kind == AttestationKindRequiredTests {
			t.Fatalf("failed physical attestation produced terminal evidence: %+v", attestation)
		}
	}
}

func TestTestAttestorCauseCodeAcceptsOnlyTypedSafeCodes(t *testing.T) {
	valid := &codedTestAttestorFailure{message: "opaque", code: "test_attestor.snapshot_limit_exceeded"}
	if got := testAttestorCauseCode(errors.Join(errors.New("outer"), valid)); got != valid.code {
		t.Fatalf("wrapped typed cause=%q want=%q", got, valid.code)
	}
	for _, err := range []error{
		errors.New("test_attestor.snapshot_limit_exceeded"),
		&codedTestAttestorFailure{message: "opaque", code: "test_attestor.bad\nsecret"},
		&codedTestAttestorFailure{message: "opaque", code: "other.snapshot_limit_exceeded"},
	} {
		if got := testAttestorCauseCode(err); got != "" {
			t.Fatalf("unsafe cause accepted: %q from %T", got, err)
		}
	}
}
