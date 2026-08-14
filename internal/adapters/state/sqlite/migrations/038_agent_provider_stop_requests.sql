-- ORC-28: fija el cuerpo fisico de Stop despues de su EffectAttempt durable y
-- antes del provider. No crea lifecycle, cola ni autoridad alternativa.
CREATE TABLE agent_provider_stop_requests (
 execution_ref TEXT NOT NULL,
 launch_action_fence INTEGER NOT NULL CHECK(launch_action_fence>0),
 stop_action_fence INTEGER NOT NULL CHECK(stop_action_fence>0),
 stop_effect_attempt_ref TEXT NOT NULL,
 provider_ref TEXT NOT NULL CHECK(length(CAST(provider_ref AS BLOB)) BETWEEN 1 AND 512
  AND trim(provider_ref)=provider_ref AND instr(provider_ref,char(0))=0),
 idempotency_key TEXT NOT NULL CHECK(length(CAST(idempotency_key AS BLOB)) BETWEEN 1 AND 512
  AND trim(idempotency_key)=idempotency_key AND instr(idempotency_key,char(0))=0),
 target_ref TEXT NOT NULL CHECK(length(CAST(target_ref AS BLOB)) BETWEEN 1 AND 512
  AND trim(target_ref)=target_ref AND instr(target_ref,char(0))=0),
 expected_revision INTEGER NOT NULL CHECK(expected_revision>0),
 body BLOB NOT NULL CHECK(typeof(body)='blob' AND length(body) BETWEEN 1 AND 2097152),
 body_sha256 TEXT NOT NULL CHECK(length(body_sha256)=64 AND body_sha256 NOT GLOB '*[^0-9a-f]*'),
 PRIMARY KEY(execution_ref,launch_action_fence,stop_action_fence),
 UNIQUE(stop_effect_attempt_ref),
 UNIQUE(provider_ref,idempotency_key),
 FOREIGN KEY(stop_effect_attempt_ref,execution_ref,stop_action_fence)
  REFERENCES effect_attempts(ref,execution_ref,action_fence) ON DELETE RESTRICT
) STRICT;

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
  AND attempt.idempotency_key=NEW.idempotency_key
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

CREATE TRIGGER agent_provider_stop_requests_immutable_update
BEFORE UPDATE ON agent_provider_stop_requests
BEGIN SELECT RAISE(ABORT,'sqlite.agent_provider_stop_request_immutable'); END;

CREATE TRIGGER agent_provider_stop_requests_immutable_delete
BEFORE DELETE ON agent_provider_stop_requests
BEGIN SELECT RAISE(ABORT,'sqlite.agent_provider_stop_request_immutable'); END;
