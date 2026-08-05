-- C4a: autoridad durable preparada antes del lanzamiento físico. La clave
-- compuesta del padre existe solo para que SQLite pueda comprobar de forma
-- declarativa que ref, ejecución y fence pertenecen al mismo intento.
CREATE UNIQUE INDEX effect_attempts_microvm_host_launch_scope_idx
ON effect_attempts(ref, execution_ref, action_fence);

CREATE TABLE microvm_host_launch_authorities (
 execution_ref TEXT NOT NULL
  CHECK(length(execution_ref) BETWEEN 1 AND 512 AND trim(execution_ref)=execution_ref
   AND instr(execution_ref,char(0))=0 AND instr(execution_ref,char(10))=0 AND instr(execution_ref,char(13))=0
   AND substr(execution_ref,1,1) NOT IN (char(9),char(10),char(11),char(12),char(13),char(32))
   AND substr(execution_ref,-1,1) NOT IN (char(9),char(10),char(11),char(12),char(13),char(32))),
 action_fence INTEGER NOT NULL CHECK(action_fence>0),
 effect_attempt_ref TEXT NOT NULL UNIQUE
  CHECK(length(effect_attempt_ref) BETWEEN 1 AND 512 AND trim(effect_attempt_ref)=effect_attempt_ref
   AND instr(effect_attempt_ref,char(0))=0 AND instr(effect_attempt_ref,char(10))=0 AND instr(effect_attempt_ref,char(13))=0
   AND substr(effect_attempt_ref,1,1) NOT IN (char(9),char(10),char(11),char(12),char(13),char(32))
   AND substr(effect_attempt_ref,-1,1) NOT IN (char(9),char(10),char(11),char(12),char(13),char(32))),
 session_ref TEXT NOT NULL
  CHECK(length(session_ref) BETWEEN 19 AND 512 AND substr(session_ref,1,18)='execution-session:'
   AND trim(session_ref)=session_ref AND instr(session_ref,char(0))=0 AND instr(session_ref,char(10))=0
   AND instr(session_ref,char(13))=0 AND instr(session_ref,'/')=0 AND instr(session_ref,'\')=0
   AND substr(session_ref,1,1) NOT IN (char(9),char(10),char(11),char(12),char(13),char(32))
   AND substr(session_ref,-1,1) NOT IN (char(9),char(10),char(11),char(12),char(13),char(32))),
 plan_sha256 TEXT NOT NULL
  CHECK(length(plan_sha256)=64 AND instr(plan_sha256,char(0))=0 AND plan_sha256 NOT GLOB '*[^0-9a-f]*'),
 concession_sha256 TEXT NOT NULL
  CHECK(length(concession_sha256)=64 AND instr(concession_sha256,char(0))=0
   AND concession_sha256 NOT GLOB '*[^0-9a-f]*'),
 control_service_ref TEXT NOT NULL
  CHECK(length(control_service_ref) BETWEEN 10 AND 160 AND substr(control_service_ref,1,9)='servicio:'
   AND instr(control_service_ref,char(0))=0 AND substr(control_service_ref,10) NOT GLOB '*[^a-z0-9_-]*'),
 control_port INTEGER NOT NULL CHECK(control_port BETWEEN 1 AND 4294967295),
 control_identity_ref TEXT NOT NULL
  CHECK(length(control_identity_ref) BETWEEN 20 AND 160 AND substr(control_identity_ref,1,19)='identidad-servicio:'
   AND instr(control_identity_ref,char(0))=0 AND substr(control_identity_ref,20) NOT GLOB '*[^a-z0-9_-]*'),
 control_identity_sha256 TEXT NOT NULL
  CHECK(length(control_identity_sha256)=64 AND instr(control_identity_sha256,char(0))=0
   AND control_identity_sha256 NOT GLOB '*[^0-9a-f]*'),
 proxy_service_ref TEXT
  CHECK(proxy_service_ref IS NULL OR (length(proxy_service_ref) BETWEEN 10 AND 160
   AND substr(proxy_service_ref,1,9)='servicio:' AND instr(proxy_service_ref,char(0))=0
   AND substr(proxy_service_ref,10) NOT GLOB '*[^a-z0-9_-]*')),
 proxy_port INTEGER CHECK(proxy_port IS NULL OR proxy_port BETWEEN 1 AND 4294967295),
 proxy_identity_ref TEXT
  CHECK(proxy_identity_ref IS NULL OR (length(proxy_identity_ref) BETWEEN 20 AND 160
   AND substr(proxy_identity_ref,1,19)='identidad-servicio:'
   AND instr(proxy_identity_ref,char(0))=0 AND substr(proxy_identity_ref,20) NOT GLOB '*[^a-z0-9_-]*')),
 proxy_identity_sha256 TEXT
  CHECK(proxy_identity_sha256 IS NULL OR (length(proxy_identity_sha256)=64
   AND instr(proxy_identity_sha256,char(0))=0 AND proxy_identity_sha256 NOT GLOB '*[^0-9a-f]*')),
 external_ref TEXT UNIQUE
  CHECK(external_ref IS NULL OR (length(external_ref) BETWEEN 11 AND 96
   AND substr(external_ref,1,10)='ejecucion:' AND instr(external_ref,char(0))=0
   AND substr(external_ref,11) NOT GLOB '*[^a-z0-9_-]*')),
 credential_ref TEXT NOT NULL
  CHECK(length(credential_ref) BETWEEN 12 AND 139 AND substr(credential_ref,1,11)='credential:'
   AND instr(credential_ref,char(0))=0 AND substr(credential_ref,12,1) GLOB '[a-z0-9]'
   AND substr(credential_ref,13) NOT GLOB '*[^a-z0-9._-]*'),
 owner_ref TEXT NOT NULL
  CHECK(length(owner_ref) BETWEEN 1 AND 512 AND trim(owner_ref)=owner_ref
   AND instr(owner_ref,char(0))=0 AND instr(owner_ref,char(10))=0 AND instr(owner_ref,char(13))=0
   AND substr(owner_ref,1,1) NOT IN (char(9),char(10),char(11),char(12),char(13),char(32))
   AND substr(owner_ref,-1,1) NOT IN (char(9),char(10),char(11),char(12),char(13),char(32))),
 scope_ref TEXT NOT NULL
  CHECK(length(scope_ref) BETWEEN 1 AND 512 AND trim(scope_ref)=scope_ref
   AND instr(scope_ref,char(0))=0 AND instr(scope_ref,char(10))=0 AND instr(scope_ref,char(13))=0
   AND substr(scope_ref,1,1) NOT IN (char(9),char(10),char(11),char(12),char(13),char(32))
   AND substr(scope_ref,-1,1) NOT IN (char(9),char(10),char(11),char(12),char(13),char(32))),
 purpose_ref TEXT NOT NULL
  CHECK(length(purpose_ref) BETWEEN 1 AND 512 AND trim(purpose_ref)=purpose_ref
   AND instr(purpose_ref,char(0))=0 AND instr(purpose_ref,char(10))=0 AND instr(purpose_ref,char(13))=0
   AND substr(purpose_ref,1,1) NOT IN (char(9),char(10),char(11),char(12),char(13),char(32))
   AND substr(purpose_ref,-1,1) NOT IN (char(9),char(10),char(11),char(12),char(13),char(32))),
 credential_version INTEGER NOT NULL CHECK(credential_version>0),
 actor_ref TEXT NOT NULL
  CHECK(length(actor_ref) BETWEEN 1 AND 512 AND trim(actor_ref)=actor_ref
   AND instr(actor_ref,char(0))=0 AND instr(actor_ref,char(10))=0 AND instr(actor_ref,char(13))=0
   AND substr(actor_ref,1,1) NOT IN (char(9),char(10),char(11),char(12),char(13),char(32))
   AND substr(actor_ref,-1,1) NOT IN (char(9),char(10),char(11),char(12),char(13),char(32))),
 request_ref TEXT NOT NULL UNIQUE
  CHECK(length(request_ref)=108
   AND substr(request_ref,1,44)='request:microvm-host-launch-one-shot:sha256:'
   AND instr(request_ref,char(0))=0 AND substr(request_ref,45) NOT GLOB '*[^0-9a-f]*'),
 PRIMARY KEY(execution_ref,action_fence),
 FOREIGN KEY(effect_attempt_ref,execution_ref,action_fence)
  REFERENCES effect_attempts(ref,execution_ref,action_fence) ON DELETE RESTRICT,
 CHECK((proxy_service_ref IS NULL AND proxy_port IS NULL AND proxy_identity_ref IS NULL AND proxy_identity_sha256 IS NULL)
  OR (proxy_service_ref IS NOT NULL AND proxy_port IS NOT NULL
   AND proxy_identity_ref IS NOT NULL AND proxy_identity_sha256 IS NOT NULL)),
 CHECK(proxy_service_ref IS NULL OR (proxy_service_ref<>control_service_ref AND proxy_port<>control_port))
) STRICT;

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
  AND NOT EXISTS (SELECT 1 FROM effect_receipts receipt WHERE receipt.attempt_ref=attempt.ref)
)
BEGIN SELECT RAISE(ABORT,'sqlite.microvm_host_launch_authority_causal_invalid'); END;

CREATE TRIGGER microvm_host_launch_authorities_bind_once
BEFORE UPDATE ON microvm_host_launch_authorities
WHEN NOT (
 OLD.external_ref IS NULL AND NEW.external_ref IS NOT NULL
 AND NEW.execution_ref=OLD.execution_ref AND NEW.action_fence=OLD.action_fence
 AND NEW.effect_attempt_ref=OLD.effect_attempt_ref AND NEW.session_ref=OLD.session_ref
 AND NEW.plan_sha256=OLD.plan_sha256 AND NEW.concession_sha256=OLD.concession_sha256
 AND NEW.control_service_ref=OLD.control_service_ref AND NEW.control_port=OLD.control_port
 AND NEW.control_identity_ref=OLD.control_identity_ref AND NEW.control_identity_sha256=OLD.control_identity_sha256
 AND NEW.proxy_service_ref IS OLD.proxy_service_ref AND NEW.proxy_port IS OLD.proxy_port
 AND NEW.proxy_identity_ref IS OLD.proxy_identity_ref AND NEW.proxy_identity_sha256 IS OLD.proxy_identity_sha256
 AND NEW.credential_ref=OLD.credential_ref AND NEW.owner_ref=OLD.owner_ref
 AND NEW.scope_ref=OLD.scope_ref AND NEW.purpose_ref=OLD.purpose_ref
 AND NEW.credential_version=OLD.credential_version AND NEW.actor_ref=OLD.actor_ref
 AND NEW.request_ref=OLD.request_ref
)
BEGIN SELECT RAISE(ABORT,'sqlite.microvm_host_launch_authority_immutable'); END;

CREATE TRIGGER microvm_host_launch_authorities_immutable_delete
BEFORE DELETE ON microvm_host_launch_authorities
BEGIN SELECT RAISE(ABORT,'sqlite.microvm_host_launch_authority_immutable'); END;
