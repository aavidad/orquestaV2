-- C4a: una autoridad de lanzamiento solo puede conservar la sesión durable
-- exacta de la ejecución. V32 ya protegía intento, ejecución y fence, pero no
-- enlazaba session_ref con executions.execution_session_ref.
CREATE TABLE microvm_host_launch_session_v33_preflight (
 marker INTEGER PRIMARY KEY CHECK(marker=1)
) STRICT;

CREATE TRIGGER microvm_host_launch_session_v33_preflight_guard
BEFORE INSERT ON microvm_host_launch_session_v33_preflight
WHEN EXISTS (
 SELECT 1
 FROM microvm_host_launch_authorities authority
 LEFT JOIN executions execution ON execution.ref=authority.execution_ref
 WHERE execution.ref IS NULL
  OR execution.execution_session_ref=''
  OR execution.execution_session_ref<>authority.session_ref
)
BEGIN SELECT RAISE(ABORT,'sqlite.microvm_host_launch_authority_session_migration_invalid'); END;

INSERT INTO microvm_host_launch_session_v33_preflight(marker) VALUES(1);
DROP TRIGGER microvm_host_launch_session_v33_preflight_guard;
DROP TABLE microvm_host_launch_session_v33_preflight;

DROP TRIGGER microvm_host_launch_authorities_causal_insert;

CREATE TRIGGER microvm_host_launch_authorities_causal_insert
BEFORE INSERT ON microvm_host_launch_authorities
WHEN NEW.external_ref IS NOT NULL OR NOT EXISTS (
 SELECT 1
 FROM effect_attempts attempt
 JOIN effect_intents intent ON intent.ref=attempt.intent_ref
 JOIN executions execution ON execution.ref=attempt.execution_ref
 WHERE attempt.ref=NEW.effect_attempt_ref
  AND attempt.execution_ref=NEW.execution_ref AND attempt.action_fence=NEW.action_fence
  AND attempt.actor_ref=NEW.actor_ref AND attempt.project_ref=NEW.scope_ref
  AND intent.kind='agent_launch' AND intent.execution_ref=attempt.execution_ref
  AND execution.ref=NEW.execution_ref AND execution.state='dispatching'
  AND execution.execution_session_ref<>''
  AND execution.execution_session_ref=NEW.session_ref
  AND NOT EXISTS (SELECT 1 FROM effect_receipts receipt WHERE receipt.attempt_ref=attempt.ref)
)
BEGIN SELECT RAISE(ABORT,'sqlite.microvm_host_launch_authority_causal_invalid'); END;
