package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"orquesta/internal/application"
	"orquesta/internal/wizard/gaps"
)

func insertWizardGapsResultSnapshot(
	ctx context.Context,
	transaction *sql.Tx,
	receipt application.WizardGapsInputReceipt,
) error {
	snapshot := receipt.ResultSnapshot
	_, err := transaction.ExecContext(ctx, `
INSERT INTO wizard_gaps_result_snapshots(
 ref,input_receipt_ref,expected_revision,
 evaluator_schema,evaluator_version,evaluator_semantic_digest,
 snapshot_schema,snapshot_digest,snapshot_bytes
) VALUES(?,?,?,?,?,?,?,?,?)`,
		snapshot.Ref, receipt.Ref, int64(receipt.ExpectedRevision),
		receipt.EvaluatorIdentity.Schema, receipt.EvaluatorIdentity.Version,
		receipt.EvaluatorIdentity.SemanticDigest, gaps.ResultSnapshotSchema,
		snapshot.Digest, snapshot.Bytes,
	)
	return mapDatabaseError(err)
}

func readWizardGapsResultSnapshot(
	ctx context.Context,
	source queryer,
	receipt application.WizardGapsInputReceipt,
) (application.WizardGapsResultSnapshot, error) {
	var snapshot application.WizardGapsResultSnapshot
	var inputReceiptRef, evaluatorSchema, evaluatorVersion string
	var evaluatorDigest, snapshotSchema string
	var expectedRevision int64
	err := source.QueryRowContext(ctx, `
SELECT ref,input_receipt_ref,expected_revision,
       evaluator_schema,evaluator_version,evaluator_semantic_digest,
       snapshot_schema,snapshot_digest,snapshot_bytes
FROM wizard_gaps_result_snapshots WHERE input_receipt_ref=?`, receipt.Ref).Scan(
		&snapshot.Ref, &inputReceiptRef, &expectedRevision,
		&evaluatorSchema, &evaluatorVersion, &evaluatorDigest,
		&snapshotSchema, &snapshot.Digest, &snapshot.Bytes,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return application.WizardGapsResultSnapshot{}, nil
	}
	if err != nil {
		return application.WizardGapsResultSnapshot{}, mapDatabaseError(err)
	}
	snapshot.InputReceiptRef = inputReceiptRef
	refDigest := strings.TrimPrefix(snapshot.Ref, "wizard-gaps-result-snapshot:")
	if snapshot.InputReceiptRef != receipt.Ref ||
		expectedRevision != int64(receipt.ExpectedRevision) ||
		evaluatorSchema != receipt.EvaluatorIdentity.Schema ||
		evaluatorVersion != receipt.EvaluatorIdentity.Version ||
		evaluatorDigest != receipt.EvaluatorIdentity.SemanticDigest ||
		snapshotSchema != gaps.ResultSnapshotSchema ||
		!validCanonicalHash(refDigest) || !validCanonicalHash(snapshot.Digest) {
		return application.WizardGapsResultSnapshot{}, invalid(
			errors.New("sqlite.wizard_gaps_result_snapshot_binding_invalid"),
		)
	}
	if _, err := gaps.RestoreResultSnapshot(snapshot.Bytes, snapshot.Digest); err != nil {
		return application.WizardGapsResultSnapshot{}, invalid(
			errors.Join(errors.New("sqlite.wizard_gaps_result_snapshot_invalid"), err),
		)
	}
	return snapshot, nil
}
