#!/usr/bin/env python3
"""Fail-closed guard for the advisory V38 non-physical status read-model."""

import argparse
import json
import sys
from pathlib import Path


EXPECTED_SCHEMA = "orquesta.v38.nonphysical_status.v1"
EXPECTED_BASE = "b2eabc98ef347e5dd422a4f62caef7ba6ec55132"
EXPECTED_ITEMS = {
    "039_launch_digest_supplement",
    "040_historical_authority",
    "b12_preserve_recover",
    "b12_stop_seal_persist_close",
    "gate_a_nonphysical_candidate",
    "gate_b_nonphysical_binding",
    "b12_nonphysical_acceptance_task08",
}
EXPECTED_ITEM_STATES = {
    "039_launch_digest_supplement": "implemented_unaccredited",
    "040_historical_authority": "implemented_unaccredited",
    "b12_preserve_recover": "implemented_unaccredited",
    "b12_stop_seal_persist_close": "implemented_unaccredited",
    "gate_a_nonphysical_candidate": "implemented_unaccredited",
    "gate_b_nonphysical_binding": "implemented_no_go",
    "b12_nonphysical_acceptance_task08": "incomplete",
}
EXPECTED_ITEM_BLOCKERS = {
    "039_launch_digest_supplement": ["accreditation_receipt"],
    "040_historical_authority": ["accreditation_receipt"],
    "b12_preserve_recover": ["task08_nonphysical_acceptance", "accreditation_receipt"],
    "b12_stop_seal_persist_close": ["task08_nonphysical_acceptance", "accreditation_receipt"],
    "gate_a_nonphysical_candidate": ["accreditation_receipt"],
    "gate_b_nonphysical_binding": ["physical_gate_b_receipt"],
    "b12_nonphysical_acceptance_task08": ["implementation_and_exercise"],
}


class GuardError(Exception):
    def __init__(self, code, detail):
        super().__init__(detail)
        self.code = code


def load_json(path, code):
    def unique_object(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise GuardError(code, f"duplicate JSON key: {key}")
            result[key] = value
        return result

    try:
        with Path(path).open("r", encoding="utf-8") as stream:
            return json.load(stream, object_pairs_hook=unique_object)
    except GuardError:
        raise
    except (OSError, json.JSONDecodeError) as exc:
        raise GuardError(code, str(exc)) from exc


def require(condition, code, detail):
    if not condition:
        raise GuardError(code, detail)


def validate_status(status):
    require(isinstance(status, dict), "STATUS_SCHEMA_INVALID", "status root must be an object")
    require(status.get("schema") == EXPECTED_SCHEMA and status.get("schema_version") == 1,
            "STATUS_SCHEMA_INVALID", "unexpected status schema")
    require(status.get("authority") == "advisory_read_model_no_canonical_state_change",
            "STATUS_AUTHORITY_INVALID", "read-model must remain advisory")
    subject = status.get("subject", {})
    require(subject == {
        "base_commit": EXPECTED_BASE,
        "vertical_id": "agent_runtime_elastic",
        "capability_id": "ORC-28",
        "acceptance_contract_id": "AC-V38-AGENT-RUNTIME-ELASTIC",
    }, "STATUS_SUBJECT_INVALID", "status subject identity drifted")
    snapshot = status.get("canonical_snapshot", {})
    require(snapshot == {
        "roadmap_status": "declared",
        "capabilities_entry": "absent",
        "accreditation_evidence": "absent",
    }, "STATUS_PROMOTION_CLAIM", "canonical snapshot overstates V38")

    items = status.get("work_items")
    require(isinstance(items, list), "STATUS_ITEMS_INVALID", "work_items must be a list")
    require({item.get("id") for item in items if isinstance(item, dict)} == EXPECTED_ITEMS
            and len(items) == len(EXPECTED_ITEMS),
            "STATUS_ITEMS_INVALID", "work item set must be exact")
    for item in items:
        item_id = item.get("id")
        state = item.get("state")
        refs = item.get("exercise_refs")
        require(item.get("blocked_by") == EXPECTED_ITEM_BLOCKERS[item_id],
                "STATUS_BLOCKERS_INVALID",
                f"{item_id}: blockers drifted from the frozen snapshot")
        require(isinstance(refs, list), "STATUS_EXERCISE_REFS_INVALID",
                f"{item_id}: exercise_refs must be a list")
        if state == "exercised" and not refs:
            raise GuardError("STATUS_EXERCISE_RECEIPT_MISSING",
                             f"{item_id}: exercised requires an exact bound receipt ref")
        require(not refs, "STATUS_EXERCISE_REFS_INVALID",
                f"{item_id}: the frozen non-physical snapshot has no bound exercise receipts")
        require(state == EXPECTED_ITEM_STATES[item_id], "STATUS_STATE_INVALID",
                f"{item_id}: expected {EXPECTED_ITEM_STATES[item_id]!r}, got {state!r}")

    gate_c = status.get("physical_gate_c", {})
    require(gate_c.get("state") == "parked", "PHYSICAL_GATE_C_UNPARKED",
            "physical Gate C must remain parked")
    claims = status.get("claims", {})
    expected_claims = {
        "canonical_state_change", "capability_promotion", "accreditation",
        "gate_a_accredits_v38", "gate_b_accredits_v38", "physical_execution",
    }
    require(isinstance(claims, dict) and set(claims) == expected_claims
            and all(value is False for value in claims.values()),
            "STATUS_PROMOTION_CLAIM", "all promotion and accreditation claims must be false")
    require(status.get("evidence_refs") == [], "STATUS_EVIDENCE_FORBIDDEN",
            "advisory status cannot publish accreditation evidence")


def validate_authorities(roadmap, capabilities):
    require(isinstance(roadmap, dict), "ROADMAP_JSON_INVALID", "roadmap root must be an object")
    require(isinstance(capabilities, dict), "CAPABILITIES_JSON_INVALID",
            "capabilities root must be an object")
    entries = roadmap.get("capability_entries", [])
    require(isinstance(entries, list), "ROADMAP_ORC28_INVALID",
            "roadmap capability_entries must be a list")
    orc28 = [entry for entry in entries if entry.get("id") == "ORC-28"]
    require(len(orc28) == 1, "ROADMAP_ORC28_INVALID", "roadmap must contain one ORC-28")
    require(orc28[0].get("status") == "declared" and orc28[0].get("evidence_refs") == [],
            "ROADMAP_V38_PROMOTED", "ORC-28 must remain declared without evidence")
    require(orc28[0].get("owner_context") == "agent_runtime_elastic",
            "ROADMAP_ORC28_INVALID", "ORC-28 owner drifted")
    contracts = roadmap.get("acceptance_contracts", [])
    require(isinstance(contracts, list), "ROADMAP_V38_PROMOTED",
            "roadmap acceptance_contracts must be a list")
    v38 = [contract for contract in contracts if contract.get("id") == "AC-V38-AGENT-RUNTIME-ELASTIC"]
    require(len(v38) == 1 and v38[0].get("status") == "planned",
            "ROADMAP_V38_PROMOTED", "V38 acceptance contract must remain planned")
    published = capabilities.get("capabilities", [])
    require(isinstance(published, list), "CAPABILITIES_JSON_INVALID",
            "release capabilities must be a list")
    require(not any(entry.get("id") == "ORC-28" for entry in published),
            "CAPABILITIES_V38_PROMOTED", "ORC-28 must not enter release capabilities")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--status", required=True)
    parser.add_argument("--roadmap", required=True)
    parser.add_argument("--capabilities", required=True)
    args = parser.parse_args()
    try:
        status = load_json(args.status, "STATUS_JSON_INVALID")
        roadmap = load_json(args.roadmap, "ROADMAP_JSON_INVALID")
        capabilities = load_json(args.capabilities, "CAPABILITIES_JSON_INVALID")
        validate_status(status)
        validate_authorities(roadmap, capabilities)
    except GuardError as exc:
        print(f"{exc.code}: {exc}", file=sys.stderr)
        return 1
    print("V38_NONPHYSICAL_STATUS_OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
