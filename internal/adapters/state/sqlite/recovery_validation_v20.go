package sqlite

import (
	"context"
	"database/sql"
	"errors"
)

func validateRecoveryV20CommandAudit(ctx context.Context, tx *sql.Tx) error {
	checks := []struct {
		code  string
		query string
	}{
		{
			code: "sqlite.recovery_v20_command_invocation_invalid",
			query: `SELECT COUNT(*) FROM command_invocations invocation WHERE
 length(trim(invocation.ref))=0 OR length(trim(invocation.command_id))=0 OR
 length(trim(invocation.command_version))=0 OR
 length(invocation.registry_digest)<>71 OR substr(invocation.registry_digest,1,7)<>'sha256:' OR
 substr(invocation.registry_digest,8) GLOB '*[^0-9a-f]*' OR
 length(invocation.schema_digest)<>64 OR invocation.schema_digest GLOB '*[^0-9a-f]*' OR
 length(trim(invocation.request_ref))=0 OR
 length(invocation.input_digest)<>64 OR invocation.input_digest GLOB '*[^0-9a-f]*' OR
 length(trim(invocation.principal_ref))=0 OR length(trim(invocation.project_ref))=0 OR
 (invocation.authenticated_execution_ref<>'' AND length(trim(invocation.authenticated_execution_ref))=0) OR
 invocation.replay_mode NOT IN ('application_receipt','read_reexecute') OR
 invocation.status<>'admitted'`,
		},
		{
			code: "sqlite.recovery_v20_command_outcome_invalid",
			query: `SELECT COUNT(*) FROM command_outcomes outcome
LEFT JOIN command_invocations invocation ON invocation.ref=outcome.command_invocation_ref
WHERE invocation.ref IS NULL OR outcome.ref<>outcome.command_invocation_ref || ':outcome' OR
 length(outcome.output_digest)<>64 OR outcome.output_digest GLOB '*[^0-9a-f]*' OR
 outcome.completed_at<invocation.admitted_at OR
 (outcome.status='completed' AND outcome.error_code<>'') OR
 (outcome.status='rejected' AND outcome.error_code NOT IN
  ('invalid_request','unauthenticated','forbidden','not_found','conflict')) OR
 (outcome.status='failed' AND outcome.error_code NOT IN ('unavailable','internal')) OR
 outcome.status NOT IN ('completed','rejected','failed')`,
		},
		{
			code: "sqlite.recovery_v20_command_governance_cross_link_invalid",
			query: `SELECT COUNT(*)
FROM command_invocations invocation
JOIN command_outcomes outcome ON outcome.command_invocation_ref=invocation.ref
WHERE outcome.status='completed' AND (
 (invocation.command_id='orquesta.director.plan.propose' AND NOT EXISTS (
  SELECT 1 FROM director_decisions decision
  JOIN authorization_receipts authorization ON authorization.ref=decision.authorization_receipt_ref
  WHERE decision.request_ref=invocation.request_ref
   AND decision.principal_ref=invocation.principal_ref
   AND decision.project_ref=invocation.project_ref
   AND authorization.principal_ref=invocation.principal_ref
   AND authorization.project_ref=invocation.project_ref
 )) OR
 (invocation.command_id='orquesta.effects.decide' AND NOT EXISTS (
  SELECT 1 FROM effect_approvals approval
  JOIN authorization_receipts authorization ON authorization.ref=approval.authorization_receipt_ref
  WHERE approval.request_ref=invocation.request_ref
   AND approval.decided_by_ref=invocation.principal_ref
   AND approval.project_ref=invocation.project_ref
   AND authorization.principal_ref=invocation.principal_ref
   AND authorization.project_ref=invocation.project_ref
 )) OR
 (invocation.command_id='orquesta.goals.control' AND NOT EXISTS (
  SELECT 1 FROM controls control
  JOIN authorization_receipts authorization ON authorization.ref=control.authorization_receipt_ref
  WHERE control.request_ref=invocation.request_ref
   AND control.principal_ref=invocation.principal_ref
   AND control.project_ref=invocation.project_ref
   AND authorization.principal_ref=invocation.principal_ref
   AND authorization.project_ref=invocation.project_ref
 )) OR
 (invocation.command_id='orquesta.council.round.open' AND NOT EXISTS (
  SELECT 1 FROM council_rounds round
  JOIN authorization_receipts authorization ON authorization.ref=round.authorization_receipt_ref
  WHERE round.request_ref=invocation.request_ref
   AND round.opened_by_ref=invocation.principal_ref
   AND round.project_ref=invocation.project_ref
   AND authorization.principal_ref=invocation.principal_ref
   AND authorization.project_ref=invocation.project_ref
 )) OR
 (invocation.command_id='orquesta.council.skip' AND NOT EXISTS (
  SELECT 1 FROM council_skips skip
  JOIN authorization_receipts authorization ON authorization.ref=skip.authorization_receipt_ref
  WHERE skip.request_ref=invocation.request_ref
   AND skip.principal_ref=invocation.principal_ref
   AND skip.project_ref=invocation.project_ref
   AND authorization.principal_ref=invocation.principal_ref
   AND authorization.project_ref=invocation.project_ref
 ))
)`,
		},
	}
	for _, check := range checks {
		var count int
		if err := tx.QueryRowContext(ctx, check.query).Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return errors.New(check.code)
		}
	}
	return nil
}
