-- C4b: partición sin heurísticas de toda autoridad v32. Las filas anteriores
-- quedan exentas de forma explícita; no se les inventan digests criptográficos.
CREATE TABLE microvm_host_launch_runtime_digest_epoch (
 singleton INTEGER PRIMARY KEY CHECK(singleton=1),
 schema_epoch INTEGER NOT NULL CHECK(schema_epoch=39),
 legacy_authority_count INTEGER NOT NULL CHECK(legacy_authority_count>=0)
) STRICT;

INSERT INTO microvm_host_launch_runtime_digest_epoch(singleton,schema_epoch,legacy_authority_count)
SELECT 1,39,COUNT(*) FROM microvm_host_launch_authorities;

CREATE TABLE microvm_host_launch_runtime_digest_legacy_exemptions (
 execution_ref TEXT NOT NULL,
 action_fence INTEGER NOT NULL CHECK(action_fence>0),
 PRIMARY KEY(execution_ref,action_fence),
 FOREIGN KEY(execution_ref,action_fence)
  REFERENCES microvm_host_launch_authorities(execution_ref,action_fence) ON DELETE RESTRICT
) STRICT;

INSERT INTO microvm_host_launch_runtime_digest_legacy_exemptions(execution_ref,action_fence)
SELECT execution_ref,action_fence FROM microvm_host_launch_authorities;

CREATE TABLE microvm_host_launch_runtime_digests (
 execution_ref TEXT NOT NULL,
  action_fence INTEGER NOT NULL CHECK(action_fence>0),
  plan_sha256 TEXT NOT NULL
  CHECK(length(plan_sha256)=64 AND plan_sha256 NOT GLOB '*[^0-9a-f]*'),
  concession_sha256 TEXT NOT NULL
  CHECK(length(concession_sha256)=64 AND concession_sha256 NOT GLOB '*[^0-9a-f]*'),
  kernel_sha256 TEXT NOT NULL
  CHECK(length(kernel_sha256)=64 AND kernel_sha256 NOT GLOB '*[^0-9a-f]*'),
 initramfs_sha256 TEXT NOT NULL
  CHECK(length(initramfs_sha256)=64 AND initramfs_sha256 NOT GLOB '*[^0-9a-f]*'),
 profile_sha256 TEXT NOT NULL
  CHECK(length(profile_sha256)=64 AND profile_sha256 NOT GLOB '*[^0-9a-f]*'),
 PRIMARY KEY(execution_ref,action_fence),
 FOREIGN KEY(execution_ref,action_fence)
  REFERENCES microvm_host_launch_authorities(execution_ref,action_fence)
  ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED
) STRICT;

CREATE TRIGGER microvm_host_launch_authorities_runtime_digest_required
BEFORE INSERT ON microvm_host_launch_authorities
WHEN NOT EXISTS (
 SELECT 1 FROM microvm_host_launch_runtime_digests runtime
 WHERE runtime.execution_ref=NEW.execution_ref AND runtime.action_fence=NEW.action_fence
)
BEGIN SELECT RAISE(ABORT,'sqlite.microvm_host_launch_runtime_digests_required'); END;

CREATE TRIGGER microvm_host_launch_runtime_digests_immutable_update
BEFORE UPDATE ON microvm_host_launch_runtime_digests
BEGIN SELECT RAISE(ABORT,'sqlite.microvm_host_launch_runtime_digests_immutable'); END;

CREATE TRIGGER microvm_host_launch_runtime_digests_reject_legacy
BEFORE INSERT ON microvm_host_launch_runtime_digests
WHEN EXISTS (
 SELECT 1 FROM microvm_host_launch_runtime_digest_legacy_exemptions legacy
 WHERE legacy.execution_ref=NEW.execution_ref AND legacy.action_fence=NEW.action_fence
)
BEGIN SELECT RAISE(ABORT,'sqlite.microvm_host_launch_runtime_digests_legacy_conflict'); END;

CREATE TRIGGER microvm_host_launch_runtime_digests_immutable_delete
BEFORE DELETE ON microvm_host_launch_runtime_digests
BEGIN SELECT RAISE(ABORT,'sqlite.microvm_host_launch_runtime_digests_immutable'); END;

CREATE TRIGGER microvm_host_launch_runtime_digest_legacy_exemptions_immutable_update
BEFORE UPDATE ON microvm_host_launch_runtime_digest_legacy_exemptions
BEGIN SELECT RAISE(ABORT,'sqlite.microvm_host_launch_runtime_digest_legacy_exemption_immutable'); END;

CREATE TRIGGER microvm_host_launch_runtime_digest_legacy_exemptions_closed_insert
BEFORE INSERT ON microvm_host_launch_runtime_digest_legacy_exemptions
BEGIN SELECT RAISE(ABORT,'sqlite.microvm_host_launch_runtime_digest_legacy_exemption_closed'); END;

CREATE TRIGGER microvm_host_launch_runtime_digest_legacy_exemptions_immutable_delete
BEFORE DELETE ON microvm_host_launch_runtime_digest_legacy_exemptions
BEGIN SELECT RAISE(ABORT,'sqlite.microvm_host_launch_runtime_digest_legacy_exemption_immutable'); END;

CREATE TRIGGER microvm_host_launch_runtime_digest_epoch_immutable_update
BEFORE UPDATE ON microvm_host_launch_runtime_digest_epoch
BEGIN SELECT RAISE(ABORT,'sqlite.microvm_host_launch_runtime_digest_epoch_immutable'); END;

CREATE TRIGGER microvm_host_launch_runtime_digest_epoch_closed_insert
BEFORE INSERT ON microvm_host_launch_runtime_digest_epoch
BEGIN SELECT RAISE(ABORT,'sqlite.microvm_host_launch_runtime_digest_epoch_closed'); END;

CREATE TRIGGER microvm_host_launch_runtime_digest_epoch_immutable_delete
BEFORE DELETE ON microvm_host_launch_runtime_digest_epoch
BEGIN SELECT RAISE(ABORT,'sqlite.microvm_host_launch_runtime_digest_epoch_immutable'); END;
