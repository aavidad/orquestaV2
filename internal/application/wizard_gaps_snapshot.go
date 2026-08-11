package application

import (
	"bytes"
	"errors"

	"orquesta/internal/wizard/gaps"
)

const wizardGapsResultSnapshotBindingSchema = "orquesta.wizard.gaps.result-snapshot-binding.v1"

// WizardGapsResultSnapshot binds canonical evaluation bytes to an immutable
// input receipt. The binding points to the receipt, avoiding a digest cycle
// and preserving historical input receipt hashes.
type WizardGapsResultSnapshot struct {
	Ref             string
	InputReceiptRef string
	Digest          string
	Bytes           []byte
}

func buildWizardGapsResultSnapshot(inputReceiptRef string, evaluation gaps.Result) (WizardGapsResultSnapshot, error) {
	if inputReceiptRef == "" {
		return WizardGapsResultSnapshot{}, invalidWizardGapsResultSnapshot("input_receipt_ref", nil)
	}
	encoded, digest, err := gaps.MarshalResultSnapshot(evaluation)
	if err != nil {
		return WizardGapsResultSnapshot{}, invalidWizardGapsResultSnapshot("snapshot", err)
	}
	snapshot := WizardGapsResultSnapshot{
		InputReceiptRef: inputReceiptRef,
		Digest:          digest,
		Bytes:           append([]byte(nil), encoded...),
	}
	snapshot.Ref = wizardGapsResultSnapshotRef(snapshot.InputReceiptRef, snapshot.Digest)
	if _, err := validateWizardGapsResultSnapshot(inputReceiptRef, snapshot, evaluation); err != nil {
		return WizardGapsResultSnapshot{}, err
	}
	return snapshot, nil
}

func validateWizardGapsResultSnapshot(
	inputReceiptRef string,
	snapshot WizardGapsResultSnapshot,
	expected gaps.Result,
) (gaps.Result, error) {
	if inputReceiptRef == "" || snapshot.InputReceiptRef != inputReceiptRef ||
		!validWizardGapsCanonicalHash(snapshot.Digest) || len(snapshot.Bytes) == 0 ||
		snapshot.Ref != wizardGapsResultSnapshotRef(inputReceiptRef, snapshot.Digest) {
		return gaps.Result{}, invalidWizardGapsResultSnapshot("binding", nil)
	}
	restored, err := gaps.RestoreResultSnapshot(snapshot.Bytes, snapshot.Digest)
	if err != nil {
		return gaps.Result{}, invalidWizardGapsResultSnapshot("snapshot", err)
	}
	expectedBytes, expectedDigest, err := gaps.MarshalResultSnapshot(expected)
	if err != nil || expectedDigest != snapshot.Digest || !bytes.Equal(expectedBytes, snapshot.Bytes) {
		return gaps.Result{}, invalidWizardGapsResultSnapshot("evaluation", err)
	}
	return restored, nil
}

func wizardGapsResultSnapshotRef(inputReceiptRef, digest string) string {
	return "wizard-gaps-result-snapshot:" + fingerprintFields(wizardGapsResultSnapshotBindingSchema, inputReceiptRef, digest)
}

func cloneWizardGapsResultSnapshot(snapshot WizardGapsResultSnapshot) WizardGapsResultSnapshot {
	clone := snapshot
	clone.Bytes = append([]byte(nil), snapshot.Bytes...)
	return clone
}

func wizardGapsResultSnapshotEmpty(snapshot WizardGapsResultSnapshot) bool {
	return snapshot.Ref == "" && snapshot.InputReceiptRef == "" &&
		snapshot.Digest == "" && len(snapshot.Bytes) == 0
}

func invalidWizardGapsResultSnapshot(stage string, cause error) error {
	err := errors.New("application.wizard_gaps_result_snapshot_" + stage + "_invalid")
	if cause != nil {
		return errors.Join(err, cause)
	}
	return err
}
