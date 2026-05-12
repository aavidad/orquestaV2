package orquestaruntimecodexdelivery

import (
	"context"
	"errors"
	"testing"
)

func TestCodexDeliveryObservationSourceV0OmiteRegistradaAntesDeVerificarWorktree(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	verifier := &failingCodexReceiptWorktreeVerifierV0{}
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-registered-001",
			Spec:          spec,
			AckPath:       writeCodexDeliveryAckForTestV0(t, spec, codexDeliveryAckForTestV0(spec)),
		}},
	}

	observations, err := (CodexDeliveryObservationSourceV0{
		Store:            store,
		WorktreeVerifier: verifier,
	}).BuildAgentDeliveryObservationsV0(
		context.Background(),
		codexDeliveryRequestForTestV0(spec, []string{spec.AgentPacket.DeliveryRefs.AckRef}),
	)
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 0 || verifier.calls != 0 {
		t.Fatalf("observations=%+v verifier_calls=%d", observations, verifier.calls)
	}
}

type failingCodexReceiptWorktreeVerifierV0 struct {
	calls int
}

func (verifier *failingCodexReceiptWorktreeVerifierV0) VerifyCodexReceiptWorktreeV0(
	context.Context,
	CodexReceiptWorktreeVerificationRequestV0,
) error {
	verifier.calls++
	return errors.New("worktree verifier should not run")
}
