ALTER TABLE agent_environment_receipts ADD COLUMN physical_manifest_ref TEXT
 CHECK(physical_manifest_ref IS NULL OR (
  typeof(physical_manifest_ref)='text' AND
  length(CAST(physical_manifest_ref AS BLOB)) BETWEEN 1 AND 512 AND
  trim(physical_manifest_ref)=physical_manifest_ref AND
  instr(physical_manifest_ref,char(0))=0
 ));

ALTER TABLE agent_environment_receipts ADD COLUMN physical_manifest_digest TEXT
 CHECK(
  (physical_manifest_ref IS NULL)=(physical_manifest_digest IS NULL) AND
  (physical_manifest_digest IS NULL OR (
   length(physical_manifest_digest)=64 AND
   physical_manifest_digest NOT GLOB '*[^0-9a-f]*'
  ))
 );
