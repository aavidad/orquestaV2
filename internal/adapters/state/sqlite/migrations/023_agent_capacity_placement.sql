-- V38-Q2: cuota por colocación y binding inmutable con la reserva física.
CREATE TABLE agent_quota_observations (
 ref TEXT PRIMARY KEY CHECK(ref<>'' AND ref=trim(ref) AND instr(ref,char(0))=0),
 placement_ref TEXT NOT NULL CHECK(placement_ref<>'' AND placement_ref=trim(placement_ref) AND instr(placement_ref,char(0))=0),
 window_ref TEXT NOT NULL CHECK(window_ref<>'' AND window_ref=trim(window_ref) AND instr(window_ref,char(0))=0),
 status TEXT NOT NULL CHECK(status IN ('available','exhausted','unknown')),
 quality TEXT NOT NULL CHECK(quality IN ('unknown','estimated','measured','exact')),
 observed_at INTEGER NOT NULL,expires_at INTEGER NOT NULL CHECK(expires_at>observed_at),
 reset_at INTEGER CHECK(reset_at>observed_at),retry_at INTEGER CHECK(retry_at>observed_at),
 evidence_ref TEXT NOT NULL DEFAULT '' CHECK(evidence_ref='' OR evidence_ref=trim(evidence_ref) AND instr(evidence_ref,char(0))=0),
 idempotency_key TEXT NOT NULL CHECK(idempotency_key<>'' AND idempotency_key=trim(idempotency_key) AND instr(idempotency_key,char(0))=0),
 expected_revision INTEGER NOT NULL CHECK(expected_revision>=0),
 revision INTEGER NOT NULL CHECK(revision>0 AND revision=expected_revision+1),
 UNIQUE(placement_ref,revision),UNIQUE(placement_ref,idempotency_key),
 UNIQUE(ref,revision,placement_ref)
) STRICT;
CREATE INDEX agent_quota_observations_current_idx ON agent_quota_observations(placement_ref,revision DESC);
CREATE TRIGGER agent_quota_observations_revision_guard BEFORE INSERT ON agent_quota_observations
WHEN NEW.expected_revision<>COALESCE((SELECT MAX(revision) FROM agent_quota_observations
 WHERE placement_ref=NEW.placement_ref),0)
BEGIN SELECT RAISE(ABORT,'sqlite.agent_quota_observation_revision_conflict'); END;
CREATE TRIGGER agent_quota_observations_immutable_update BEFORE UPDATE ON agent_quota_observations
BEGIN SELECT RAISE(ABORT,'sqlite.agent_quota_observation_immutable'); END;
CREATE TRIGGER agent_quota_observations_immutable_delete BEFORE DELETE ON agent_quota_observations
BEGIN SELECT RAISE(ABORT,'sqlite.agent_quota_observation_immutable'); END;

CREATE TABLE agent_placement_bindings (
 reservation_ref TEXT PRIMARY KEY REFERENCES agent_capacity_reservations(ref)
  ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED,
 placement_ref TEXT NOT NULL CHECK(placement_ref<>'' AND placement_ref=trim(placement_ref) AND instr(placement_ref,char(0))=0),
 quota_observation_ref TEXT NOT NULL,
 quota_observation_revision INTEGER NOT NULL CHECK(quota_observation_revision>0),
 FOREIGN KEY(quota_observation_ref,quota_observation_revision,placement_ref)
  REFERENCES agent_quota_observations(ref,revision,placement_ref) ON DELETE RESTRICT
) STRICT;
CREATE TRIGGER agent_placement_bindings_current_guard BEFORE INSERT ON agent_placement_bindings
WHEN NOT EXISTS (SELECT 1 FROM agent_quota_observations observation
 WHERE observation.ref=NEW.quota_observation_ref
  AND observation.revision=NEW.quota_observation_revision
  AND observation.placement_ref=NEW.placement_ref
  AND observation.revision=(SELECT MAX(current.revision) FROM agent_quota_observations current
   WHERE current.placement_ref=observation.placement_ref))
BEGIN SELECT RAISE(ABORT,'sqlite.agent_placement_binding_quota_invalid'); END;
CREATE TRIGGER agent_placement_bindings_immutable_update BEFORE UPDATE ON agent_placement_bindings
BEGIN SELECT RAISE(ABORT,'sqlite.agent_placement_binding_immutable'); END;
CREATE TRIGGER agent_placement_bindings_immutable_delete BEFORE DELETE ON agent_placement_bindings
BEGIN SELECT RAISE(ABORT,'sqlite.agent_placement_binding_immutable'); END;
