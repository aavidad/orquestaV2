-- ORC-28: fija cada cuerpo fisico antes del UDS dentro del mismo repositorio
-- autoritativo. Las etapas no son lifecycle ni intentos de efecto adicionales.
CREATE TABLE agent_provider_requests (
 execution_ref TEXT NOT NULL,
 action_fence INTEGER NOT NULL CHECK(action_fence>0),
 stage TEXT NOT NULL CHECK(stage IN ('launch','session_start','session_input')),
 effect_attempt_ref TEXT NOT NULL,
 provider_ref TEXT NOT NULL CHECK(length(CAST(provider_ref AS BLOB)) BETWEEN 1 AND 512
  AND trim(provider_ref)=provider_ref AND instr(provider_ref,char(0))=0),
 idempotency_key TEXT NOT NULL CHECK(length(CAST(idempotency_key AS BLOB)) BETWEEN 1 AND 512
  AND trim(idempotency_key)=idempotency_key AND instr(idempotency_key,char(0))=0),
 target_ref TEXT NOT NULL DEFAULT '',
 expected_revision INTEGER NOT NULL DEFAULT 0 CHECK(expected_revision>=0),
 body BLOB NOT NULL CHECK(typeof(body)='blob' AND length(body) BETWEEN 1 AND 2097152),
 body_sha256 TEXT NOT NULL CHECK(length(body_sha256)=64 AND body_sha256 NOT GLOB '*[^0-9a-f]*'),
 launch_binding_ref TEXT,
 launch_binding_revision INTEGER CHECK(launch_binding_revision>0),
 PRIMARY KEY(execution_ref,action_fence,stage),
 UNIQUE(effect_attempt_ref,stage),
 UNIQUE(provider_ref,idempotency_key),
 FOREIGN KEY(effect_attempt_ref,execution_ref,action_fence)
  REFERENCES effect_attempts(ref,execution_ref,action_fence) ON DELETE RESTRICT,
 CHECK(
  (stage='launch' AND target_ref='' AND expected_revision=0
   AND ((launch_binding_ref IS NULL AND launch_binding_revision IS NULL) OR
        (launch_binding_ref IS NOT NULL AND length(CAST(launch_binding_ref AS BLOB)) BETWEEN 1 AND 512
         AND trim(launch_binding_ref)=launch_binding_ref AND instr(launch_binding_ref,char(0))=0
         AND launch_binding_revision IS NOT NULL)))
  OR
  (stage IN ('session_start','session_input') AND length(CAST(target_ref AS BLOB)) BETWEEN 1 AND 512
   AND trim(target_ref)=target_ref AND instr(target_ref,char(0))=0 AND expected_revision>0
   AND launch_binding_ref IS NULL AND launch_binding_revision IS NULL)
 )
) STRICT;

CREATE TRIGGER agent_provider_requests_causal_insert
BEFORE INSERT ON agent_provider_requests
WHEN NOT EXISTS (
 SELECT 1
 FROM effect_attempts attempt
 JOIN effect_intents intent ON intent.ref=attempt.intent_ref
 JOIN executions execution ON execution.ref=attempt.execution_ref
 WHERE attempt.ref=NEW.effect_attempt_ref
  AND attempt.execution_ref=NEW.execution_ref AND attempt.action_fence=NEW.action_fence
  AND intent.kind='agent_launch' AND intent.execution_ref=attempt.execution_ref
  AND execution.state='dispatching'
  AND NOT EXISTS (SELECT 1 FROM effect_receipts receipt WHERE receipt.attempt_ref=attempt.ref)
  AND (
   NEW.stage='launch' OR EXISTS (
    SELECT 1 FROM agent_provider_requests launch
    WHERE launch.execution_ref=NEW.execution_ref AND launch.action_fence=NEW.action_fence
     AND launch.stage='launch' AND launch.effect_attempt_ref=NEW.effect_attempt_ref
     AND launch.provider_ref=NEW.provider_ref
     AND launch.launch_binding_ref=NEW.target_ref
     AND launch.launch_binding_revision=NEW.expected_revision
   )
  )
  AND (
   NEW.stage<>'session_input' OR EXISTS (
    SELECT 1 FROM agent_provider_requests started
    WHERE started.execution_ref=NEW.execution_ref AND started.action_fence=NEW.action_fence
     AND started.stage='session_start' AND started.effect_attempt_ref=NEW.effect_attempt_ref
     AND started.provider_ref=NEW.provider_ref AND started.target_ref=NEW.target_ref
     AND started.expected_revision=NEW.expected_revision
   )
  )
 )
BEGIN SELECT RAISE(ABORT,'sqlite.agent_provider_request_causal_invalid'); END;

CREATE TRIGGER agent_provider_requests_bind_launch_once
BEFORE UPDATE ON agent_provider_requests
WHEN OLD.stage<>'launch' OR OLD.launch_binding_ref IS NOT NULL OR OLD.launch_binding_revision IS NOT NULL
 OR NEW.stage<>OLD.stage OR NEW.execution_ref<>OLD.execution_ref OR NEW.action_fence<>OLD.action_fence
 OR NEW.effect_attempt_ref<>OLD.effect_attempt_ref OR NEW.provider_ref<>OLD.provider_ref
 OR NEW.idempotency_key<>OLD.idempotency_key OR NEW.target_ref<>OLD.target_ref
 OR NEW.expected_revision<>OLD.expected_revision OR NOT (NEW.body IS OLD.body)
 OR NEW.body_sha256<>OLD.body_sha256 OR NEW.launch_binding_ref IS NULL
 OR NEW.launch_binding_revision IS NULL
 OR NOT EXISTS (
  SELECT 1 FROM effect_attempts attempt
  JOIN executions execution ON execution.ref=attempt.execution_ref
  WHERE attempt.ref=OLD.effect_attempt_ref AND execution.state='dispatching'
   AND NOT EXISTS (SELECT 1 FROM effect_receipts receipt WHERE receipt.attempt_ref=attempt.ref)
 )
BEGIN SELECT RAISE(ABORT,'sqlite.agent_provider_request_immutable'); END;

CREATE TRIGGER agent_provider_requests_immutable_delete
BEFORE DELETE ON agent_provider_requests
BEGIN SELECT RAISE(ABORT,'sqlite.agent_provider_request_immutable'); END;
