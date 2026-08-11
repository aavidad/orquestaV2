-- V23: conserva el resultado canónico exacto sin reescribir el receipt de
-- inputs histórico. El store lo insertará en la misma transacción causal.
CREATE TABLE wizard_gaps_result_snapshots (
 ref TEXT PRIMARY KEY CHECK(
  length(ref)=92 AND substr(ref,1,28)='wizard-gaps-result-snapshot:'
  AND substr(ref,29) NOT GLOB '*[^0-9a-f]*'),
 input_receipt_ref TEXT NOT NULL UNIQUE
  REFERENCES wizard_gaps_input_receipts(ref) ON DELETE RESTRICT,
 expected_revision INTEGER NOT NULL CHECK(expected_revision>0),
 evaluator_schema TEXT NOT NULL CHECK(
  length(trim(evaluator_schema))>0 AND evaluator_schema=trim(evaluator_schema)),
 evaluator_version TEXT NOT NULL CHECK(
  length(trim(evaluator_version))>0 AND evaluator_version=trim(evaluator_version)),
 evaluator_semantic_digest TEXT NOT NULL CHECK(
  length(evaluator_semantic_digest)=64
  AND evaluator_semantic_digest NOT GLOB '*[^0-9a-f]*'),
 snapshot_schema TEXT NOT NULL CHECK(
  snapshot_schema='orquesta.wizard.gaps.result-snapshot.v1'),
 snapshot_digest TEXT NOT NULL CHECK(
  length(snapshot_digest)=64 AND snapshot_digest NOT GLOB '*[^0-9a-f]*'),
 snapshot_bytes BLOB NOT NULL CHECK(
  typeof(snapshot_bytes)='blob' AND length(snapshot_bytes) BETWEEN 1 AND 1048576)
) STRICT;

CREATE TRIGGER wizard_gaps_result_snapshots_causal_guard
BEFORE INSERT ON wizard_gaps_result_snapshots
WHEN NOT EXISTS (
 SELECT 1 FROM wizard_gaps_input_receipts input
 WHERE input.ref=NEW.input_receipt_ref
   AND input.expected_revision=NEW.expected_revision
   AND input.evaluator_schema=NEW.evaluator_schema
   AND input.evaluator_version=NEW.evaluator_version
   AND input.evaluator_semantic_digest=NEW.evaluator_semantic_digest
)
BEGIN
 SELECT RAISE(ABORT, 'sqlite.wizard_gaps_result_snapshot_causal_invalid');
END;

CREATE TRIGGER wizard_gaps_result_snapshots_immutable_update
BEFORE UPDATE ON wizard_gaps_result_snapshots
BEGIN
 SELECT RAISE(ABORT, 'sqlite.wizard_gaps_result_snapshot_immutable');
END;

CREATE TRIGGER wizard_gaps_result_snapshots_immutable_delete
BEFORE DELETE ON wizard_gaps_result_snapshots
BEGIN
 SELECT RAISE(ABORT, 'sqlite.wizard_gaps_result_snapshot_immutable');
END;
