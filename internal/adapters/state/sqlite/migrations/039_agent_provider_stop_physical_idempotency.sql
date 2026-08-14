-- ORC-28: la autoridad Stop sigue ligada al EffectAttempt por ref, execution y
-- fence. La idempotencia fisica del provider es un identificador derivado y
-- acotado independiente de la clave opaca de application.
DROP TRIGGER agent_provider_stop_requests_causal_insert;

CREATE TRIGGER agent_provider_stop_requests_causal_insert
BEFORE INSERT ON agent_provider_stop_requests
WHEN NOT EXISTS (
 SELECT 1
 FROM effect_attempts attempt
 JOIN effect_intents intent ON intent.ref=attempt.intent_ref
 JOIN executions execution ON execution.ref=attempt.execution_ref
 WHERE attempt.ref=NEW.stop_effect_attempt_ref
  AND attempt.execution_ref=NEW.execution_ref
  AND attempt.action_fence=NEW.stop_action_fence
  AND intent.kind='agent_stop' AND intent.execution_ref=attempt.execution_ref
  AND execution.state='running' AND execution.provider_ref=NEW.provider_ref
  AND execution.external_ref=NEW.target_ref
  AND NOT EXISTS (SELECT 1 FROM effect_receipts receipt WHERE receipt.attempt_ref=attempt.ref)
  AND EXISTS (
   SELECT 1 FROM agent_provider_requests launch
   WHERE launch.execution_ref=NEW.execution_ref
    AND launch.action_fence=NEW.launch_action_fence AND launch.stage='launch'
    AND launch.provider_ref=NEW.provider_ref
    AND launch.launch_binding_ref=NEW.target_ref
    AND launch.launch_binding_revision=NEW.expected_revision
  )
 )
BEGIN SELECT RAISE(ABORT,'sqlite.agent_provider_stop_request_causal_invalid'); END;
