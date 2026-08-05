ALTER TABLE work_item_authorities ADD COLUMN egress_policy_ref TEXT
    CHECK (
        egress_policy_ref IS NULL OR (
            typeof(egress_policy_ref) = 'text' AND
            length(egress_policy_ref) BETWEEN 1 AND 512 AND
            trim(egress_policy_ref) = egress_policy_ref AND
            instr(egress_policy_ref, char(0)) = 0
        )
    );

ALTER TABLE work_item_authorities ADD COLUMN egress_policy_payload_sha256 TEXT
    CHECK (
        egress_policy_payload_sha256 IS NULL OR (
            typeof(egress_policy_payload_sha256) = 'text' AND
            length(egress_policy_payload_sha256) = 64 AND
            egress_policy_payload_sha256 NOT GLOB '*[^0-9a-f]*'
        )
    );

ALTER TABLE work_item_authorities ADD COLUMN egress_policy_canonical_payload BLOB
    CHECK (
        egress_policy_canonical_payload IS NULL OR (
            typeof(egress_policy_canonical_payload) = 'blob' AND
            length(egress_policy_canonical_payload) BETWEEN 1 AND 65536
        )
    );

CREATE TRIGGER work_item_authorities_egress_shape_guard
BEFORE INSERT ON work_item_authorities
WHEN NOT (
    (NEW.egress_policy_ref IS NULL AND
     NEW.egress_policy_payload_sha256 IS NULL AND
     NEW.egress_policy_canonical_payload IS NULL)
    OR
    (NEW.egress_policy_ref IS NOT NULL AND
     NEW.egress_policy_payload_sha256 IS NOT NULL AND
     NEW.egress_policy_canonical_payload IS NOT NULL)
)
BEGIN SELECT RAISE(ABORT, 'sqlite.work_item_egress_authority_invalid'); END;
