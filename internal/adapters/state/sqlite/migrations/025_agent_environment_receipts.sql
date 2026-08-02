CREATE TABLE agent_environment_receipts (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),idempotency_key TEXT NOT NULL UNIQUE CHECK(length(trim(idempotency_key))>0),
 project_ref TEXT NOT NULL,goal_ref TEXT NOT NULL,work_item_ref TEXT NOT NULL,execution_ref TEXT NOT NULL UNIQUE,
 workspace_ref TEXT NOT NULL UNIQUE,workspace_binding_digest TEXT NOT NULL CHECK(length(workspace_binding_digest)=64 AND workspace_binding_digest NOT GLOB '*[^0-9a-f]*'),
 base_oid TEXT NOT NULL,object_format TEXT NOT NULL CHECK(object_format IN ('sha1','sha256')),change_set_ref TEXT,change_digest TEXT,
 state TEXT NOT NULL CHECK(state='preserved_pending_review'),execution_attempt INTEGER NOT NULL CHECK(execution_attempt>0),external_ref TEXT NOT NULL CHECK(length(trim(external_ref))>0),fence INTEGER NOT NULL CHECK(fence>0),
 bundle_ref TEXT NOT NULL,bundle_digest TEXT NOT NULL,inventory_ref TEXT NOT NULL,inventory_digest TEXT NOT NULL,
 configuration_digest TEXT NOT NULL,rootfs_digest TEXT NOT NULL,seal_digest TEXT NOT NULL,provider_receipt_ref TEXT NOT NULL CHECK(length(trim(provider_receipt_ref))>0),
 sealed_at INTEGER NOT NULL,preserved_at INTEGER NOT NULL,recorded_at INTEGER NOT NULL,
 FOREIGN KEY(goal_ref,project_ref) REFERENCES goals(ref,project_ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref) REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(workspace_ref) REFERENCES workspace_bindings(execution_workspace_ref) ON DELETE RESTRICT,
 FOREIGN KEY(change_set_ref) REFERENCES change_sets(ref) ON DELETE RESTRICT,
 CHECK(length(base_oid)=CASE object_format WHEN 'sha1' THEN 40 ELSE 64 END AND base_oid NOT GLOB '*[^0-9a-f]*'),
 CHECK((change_set_ref IS NULL)=(change_digest IS NULL)),
 CHECK(bundle_ref='artifact:sha256:'||bundle_digest AND inventory_ref='artifact:sha256:'||inventory_digest),
 CHECK(length(bundle_digest)=64 AND bundle_digest NOT GLOB '*[^0-9a-f]*' AND length(inventory_digest)=64 AND inventory_digest NOT GLOB '*[^0-9a-f]*'),
 CHECK(length(configuration_digest)=64 AND configuration_digest NOT GLOB '*[^0-9a-f]*' AND length(rootfs_digest)=64 AND rootfs_digest NOT GLOB '*[^0-9a-f]*' AND length(seal_digest)=64 AND seal_digest NOT GLOB '*[^0-9a-f]*'),
 CHECK(change_digest IS NULL OR (length(change_digest)=64 AND change_digest NOT GLOB '*[^0-9a-f]*')),
 CHECK(sealed_at<=preserved_at AND preserved_at<=recorded_at)
) STRICT;
CREATE TRIGGER agent_environment_receipts_immutable_update BEFORE UPDATE ON agent_environment_receipts BEGIN SELECT RAISE(ABORT,'sqlite.agent_environment_receipt_immutable'); END;
CREATE TRIGGER agent_environment_receipts_immutable_delete BEFORE DELETE ON agent_environment_receipts BEGIN SELECT RAISE(ABORT,'sqlite.agent_environment_receipt_immutable'); END;
