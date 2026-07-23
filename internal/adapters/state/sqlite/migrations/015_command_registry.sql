-- V20 persists transport-neutral command admission and terminal outcome facts.
-- Payloads and result bodies remain outside durable audit state; exact digests
-- bind replay without creating another lifecycle authority.
CREATE TABLE command_invocations (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),
 command_id TEXT NOT NULL CHECK(length(trim(command_id))>0),
 command_version TEXT NOT NULL CHECK(length(trim(command_version))>0),
 registry_digest TEXT NOT NULL CHECK(length(registry_digest)=71 AND substr(registry_digest,1,7)='sha256:'
  AND substr(registry_digest,8) NOT GLOB '*[^0-9a-f]*'),
 schema_digest TEXT NOT NULL CHECK(length(schema_digest)=64 AND schema_digest NOT GLOB '*[^0-9a-f]*'),
 request_ref TEXT NOT NULL CHECK(length(trim(request_ref))>0),
 input_digest TEXT NOT NULL CHECK(length(input_digest)=64 AND input_digest NOT GLOB '*[^0-9a-f]*'),
 principal_ref TEXT NOT NULL CHECK(length(trim(principal_ref))>0),
 project_ref TEXT NOT NULL CHECK(length(trim(project_ref))>0),
 authenticated_execution_ref TEXT NOT NULL DEFAULT ''
  CHECK(authenticated_execution_ref='' OR length(trim(authenticated_execution_ref))>0),
 replay_mode TEXT NOT NULL CHECK(replay_mode IN ('application_receipt','read_reexecute')),
 status TEXT NOT NULL CHECK(status='admitted'),
 admitted_at INTEGER NOT NULL,
 UNIQUE(principal_ref,project_ref,command_id,command_version,request_ref)
) STRICT;
CREATE INDEX command_invocations_project_idx
 ON command_invocations(project_ref,admitted_at,ref);
CREATE TRIGGER command_invocations_immutable_update
 BEFORE UPDATE ON command_invocations
 BEGIN SELECT RAISE(ABORT,'sqlite.command_invocation_immutable'); END;
CREATE TRIGGER command_invocations_immutable_delete
 BEFORE DELETE ON command_invocations
 BEGIN SELECT RAISE(ABORT,'sqlite.command_invocation_immutable'); END;

CREATE TABLE command_outcomes (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),
 command_invocation_ref TEXT NOT NULL UNIQUE
  REFERENCES command_invocations(ref) ON DELETE RESTRICT,
 status TEXT NOT NULL CHECK(status IN ('completed','rejected','failed')),
 error_code TEXT NOT NULL DEFAULT '',
 output_digest TEXT NOT NULL
  CHECK(length(output_digest)=64 AND output_digest NOT GLOB '*[^0-9a-f]*'),
 completed_at INTEGER NOT NULL,
 CHECK(ref=command_invocation_ref || ':outcome'),
 CHECK(
  (status='completed' AND error_code='') OR
  (status='rejected' AND error_code IN ('invalid_request','unauthenticated','forbidden','not_found','conflict')) OR
  (status='failed' AND error_code IN ('unavailable','internal'))
 )
) STRICT;
CREATE INDEX command_outcomes_completed_idx
 ON command_outcomes(completed_at,ref);
CREATE TRIGGER command_outcomes_admission_guard
 BEFORE INSERT ON command_outcomes
 WHEN NOT EXISTS (
  SELECT 1 FROM command_invocations invocation
  WHERE invocation.ref=NEW.command_invocation_ref
   AND NEW.completed_at>=invocation.admitted_at
 )
 BEGIN SELECT RAISE(ABORT,'sqlite.command_outcome_admission_invalid'); END;
CREATE TRIGGER command_outcomes_immutable_update
 BEFORE UPDATE ON command_outcomes
 BEGIN SELECT RAISE(ABORT,'sqlite.command_outcome_immutable'); END;
CREATE TRIGGER command_outcomes_immutable_delete
 BEFORE DELETE ON command_outcomes
 BEGIN SELECT RAISE(ABORT,'sqlite.command_outcome_immutable'); END;
