"""Pure APG-006 manifest, blinding, and decision rules. No I/O or subprocesses."""

from __future__ import annotations

import hashlib
import hmac
import json
import os
import re
import statistics
from pathlib import PurePosixPath

MANIFEST = "orquesta.context_rotation.manifest.v1"
DATASET = "orquesta.context_rotation.dataset.v1"
SHA = re.compile(r"^[a-f0-9]{64}$")
REF = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._:/-]{0,199}$")
ROLES = ("director", "workers", "handoffs", "recoveries", "reviewers")
RANDOMIZATION = "sha256-balanced-block-v1"
MINIMUM_PAIRS = 6


def digest(value) -> str:
    raw = json.dumps(value, sort_keys=True, separators=(",", ":")).encode()
    return hashlib.sha256(raw).hexdigest()


def stable_ref(prefix: str, *parts: str) -> str:
    return f"{prefix}-{hashlib.sha256(chr(0).join(parts).encode()).hexdigest()[:20]}"


def _evidence(value) -> bool:
    return isinstance(value, dict) and REF.fullmatch(value.get("ref", "")) is not None and SHA.fullmatch(value.get("sha256", "")) is not None


def _evidence_list(values, *, required=True) -> bool:
    return isinstance(values, list) and (bool(values) or not required) and len(values) <= 100 and all(_evidence(v) for v in values)


def _write_set(values) -> bool:
    if not isinstance(values, list) or not 1 <= len(values) <= 40 or len(values) != len(set(values)):
        return False
    for value in values:
        if not isinstance(value, str) or "\\" in value:
            return False
        path = PurePosixPath(value)
        if path.is_absolute() or value in ("", ".", "..") or ".." in path.parts or path.parts[0] == ".git" or str(path) != value:
            return False
    return True


def _config(value) -> bool:
    if not isinstance(value, dict) or not value.get("model") or not value.get("effort"):
        return False
    tools, budget = value.get("tools"), value.get("budget", {})
    if not isinstance(tools, list) or not 1 <= len(tools) <= 32 or len(tools) != len(set(tools)):
        return False
    required = ("max_total_tokens", "max_cost_micros", "max_duration_ms", "max_tool_calls", "max_handoff_tokens", "max_workspace_bytes")
    if any(not isinstance(budget.get(key), int) or budget[key] <= 0 for key in required):
        return False
    return budget["max_handoff_tokens"] <= budget["max_total_tokens"]


def validate_manifest(manifest: dict) -> list[str]:
    issues = []
    check = lambda ok, issue: None if ok else issues.append(issue)
    check(manifest.get("schema_version") == MANIFEST, "schema_version")
    check(REF.fullmatch(manifest.get("experiment_ref", "")) is not None, "experiment_ref")
    check(re.fullmatch(r"(?:[a-f0-9]{40}|[a-f0-9]{64})", manifest.get("baseline_commit", "")) is not None, "baseline_commit")
    check(isinstance(manifest.get("repository_path"), str) and "\0" not in manifest["repository_path"], "repository_path")
    check(_evidence(manifest.get("snapshot")), "snapshot")
    temp, state = manifest.get("temporary_root", ""), manifest.get("state_root", "")
    check(os.path.isabs(temp) and os.path.normpath(temp) != os.sep, "temporary_root")
    check(os.path.isabs(state) and os.path.commonpath((os.path.normpath(temp), os.path.normpath(state))) == os.path.normpath(temp) and os.path.normpath(state) != os.path.normpath(temp), "state_root")
    check(_config(manifest.get("execution")), "execution")
    randomization = manifest.get("randomization", {})
    check(randomization.get("algorithm") == RANDOMIZATION and isinstance(randomization.get("seed"), str) and len(randomization["seed"]) >= 32 and randomization.get("concurrent_within_pair") is True, "randomization")
    reviewer = manifest.get("reviewer", {})
    check(_config(reviewer) and reviewer.get("blinded") is True and reviewer.get("fresh_context") is True and SHA.fullmatch(reviewer.get("blinding_key_sha256", "")) is not None and bool(reviewer.get("reproduction_test")) and reviewer.get("quality_scale") == "0-100-higher-is-better" and bool(reviewer.get("unnecessary_code_definition")) and SHA.fullmatch(reviewer.get("review_instructions_sha256", "")) is not None, "reviewer")
    retention = manifest.get("retention", {})
    check(retention.get("preserve_receipts") is True and isinstance(retention.get("minimum_retention_hours"), int) and 1 <= retention["minimum_retention_hours"] <= 8760, "retention")
    pairs = manifest.get("pairs")
    check(isinstance(pairs, list) and MINIMUM_PAIRS <= len(pairs) <= 20, "pair_count")
    if not isinstance(pairs, list):
        return sorted(issues)
    pair_refs = set()
    for pair in pairs:
        ref = pair.get("pair_ref", "") if isinstance(pair, dict) else ""
        check(REF.fullmatch(ref) is not None and ref not in pair_refs, "pair_ref")
        pair_refs.add(ref)
        check(bool(pair.get("objective")) and bool(pair.get("acceptance_criteria")) and bool(pair.get("required_tests")), f"{ref}:task_contract")
        check(_write_set(pair.get("write_set")), f"{ref}:write_set")
        check(_evidence_list(pair.get("context_refs")), f"{ref}:context_refs")
        stages, stage_refs = pair.get("treatment_stages"), set()
        check(isinstance(stages, list) and 2 <= len(stages) <= 20, f"{ref}:stages")
        if not isinstance(stages, list):
            continue
        for stage in stages:
            stage_ref = stage.get("stage_ref", "")
            check(REF.fullmatch(stage_ref) is not None and stage_ref not in stage_refs and bool(stage.get("objective")) and bool(stage.get("cause")), f"{ref}:stage")
            stage_refs.add(stage_ref)
            check(_write_set(stage.get("write_set")) and set(stage["write_set"]).issubset(pair["write_set"]), f"{ref}:{stage_ref}:write_set")
            check(_evidence_list(stage.get("context_refs")), f"{ref}:{stage_ref}:context_refs")
    return sorted(set(issues))


def config_hash(manifest: dict, pair: dict) -> str:
    return digest({"execution": manifest["execution"], "pair": pair})


def plan(manifest: dict) -> dict:
    issues = validate_manifest(manifest)
    if issues:
        raise ValueError("invalid manifest: " + ",".join(issues))
    worktrees = []
    seed = manifest["randomization"]["seed"]
    pairs = sorted(manifest["pairs"], key=lambda pair: digest([seed, pair["pair_ref"]]))
    for block, pair in enumerate(pairs):
        cfg = config_hash(manifest, pair)
        modes = ("control", "treatment") if block % 2 == 0 else ("treatment", "control")
        for arm_order, mode in enumerate(modes):
            operation = stable_ref("prepare", manifest["experiment_ref"], pair["pair_ref"], mode, manifest["baseline_commit"], cfg)
            worktrees.append({"operation_ref": operation, "pair_ref": pair["pair_ref"], "mode": mode, "block": block, "arm_order": arm_order, "path": os.path.join(manifest["temporary_root"], "worktrees", manifest["experiment_ref"], pair["pair_ref"], mode), "commit": manifest["baseline_commit"], "config_sha256": cfg})
    return {"schema_version": "orquesta.context_rotation.plan.v1", "experiment_ref": manifest["experiment_ref"], "manifest_sha256": digest(manifest), "randomization": manifest["randomization"], "dry_run": True, "worktrees": worktrees}


def stage_request(manifest: dict, pair_ref: str, mode: str, stage_ref: str) -> tuple[dict, str | None]:
    pair = next((p for p in manifest["pairs"] if p["pair_ref"] == pair_ref), None)
    if pair is None or mode not in ("control", "treatment"):
        raise ValueError("pair/mode not found")
    request = {"schema_version": "orquesta.context_rotation.stage_request.v1", "experiment_ref": manifest["experiment_ref"], "pair_ref": pair_ref, "mode": mode, "stage_ref": stage_ref, "fresh_worker": mode == "treatment", "compact_handoff_only": True, "baseline_commit": manifest["baseline_commit"], "execution": manifest["execution"], "arm_budget_shared_across_stages": True, "acceptance_criteria": pair["acceptance_criteria"], "required_tests": pair["required_tests"]}
    if mode == "control":
        if stage_ref != "continuous-session":
            raise ValueError("control stage must be continuous-session")
        all_context = pair["context_refs"] + [ref for stage in pair["treatment_stages"] for ref in stage["context_refs"]]
        request.update(objective=pair["objective"], cause="paired-control", write_set=pair["write_set"], context_refs=all_context)
        return request, None
    for index, stage in enumerate(pair["treatment_stages"]):
        if stage["stage_ref"] == stage_ref:
            request.update(objective=stage["objective"], cause=stage["cause"], write_set=stage["write_set"], context_refs=pair["context_refs"] + stage["context_refs"])
            previous = None if index == 0 else stable_ref("run", manifest["experiment_ref"], pair_ref, mode, pair["treatment_stages"][index - 1]["stage_ref"])
            return request, previous
    raise ValueError("treatment stage not found")


def _usage(arm: dict) -> tuple[int, int] | None:
    values = arm.get("usage_by_role", {})
    if not isinstance(values, dict) or set(values) != set(ROLES):
        return None
    tokens = cost = 0
    for value in values.values():
        if not isinstance(value, dict):
            return None
        counters = [value.get(k) for k in ("input_uncached", "output", "cache_read", "cache_write", "cost_micros")]
        if any(not isinstance(v, int) or v < 0 for v in counters):
            return None
        tokens += sum(counters[:4]); cost += counters[4]
    return tokens, cost


def _complete(manifest: dict, pair: dict, arm: dict) -> bool:
    usage = _usage(arm)
    metrics, budget = arm.get("metrics", {}), manifest["execution"]["budget"]
    review = arm.get("independent_review", {})
    if arm.get("status") != "completed" or arm.get("config_sha256") != config_hash(manifest, pair) or review.get("completed") is not True or review.get("candidate_ref") != arm.get("blind_candidate_ref"):
        return False
    if not REF.fullmatch(arm.get("candidate_ref", "")) or re.fullmatch(r"candidate-[a-f0-9]{24}", arm.get("blind_candidate_ref", "")) is None or not _evidence(arm.get("diff")) or not _evidence_list(arm.get("artifacts")) or not _evidence_list(arm.get("receipts")) or not _evidence_list(review.get("evidence")) or usage is None:
        return False
    counters = [metrics.get(k) for k in ("duration_ms", "tool_calls", "repeated_reads", "reworks", "failed_attempts", "write_set_conflicts", "human_interventions", "changed_lines", "changed_files")]
    if any(not isinstance(v, int) or v < 0 for v in counters):
        return False
    if not isinstance(review.get("quality_score"), int) or not 0 <= review["quality_score"] <= 100 or not isinstance(review.get("unnecessary_changed_lines"), int) or review["unnecessary_changed_lines"] < 0 or review["unnecessary_changed_lines"] > metrics["changed_lines"]:
        return False
    reviewer = arm["usage_by_role"]["reviewers"]
    return usage[0] <= budget["max_total_tokens"] and usage[1] <= budget["max_cost_micros"] and metrics["duration_ms"] <= budget["max_duration_ms"] and metrics["tool_calls"] <= budget["max_tool_calls"] and arm["usage_by_role"]["handoffs"]["input_uncached"] + arm["usage_by_role"]["handoffs"]["output"] + arm["usage_by_role"]["handoffs"]["cache_read"] + arm["usage_by_role"]["handoffs"]["cache_write"] <= budget["max_handoff_tokens"] and sum(reviewer[k] for k in ("input_uncached", "output", "cache_read", "cache_write")) <= manifest["reviewer"]["budget"]["max_total_tokens"] and reviewer["cost_micros"] <= manifest["reviewer"]["budget"]["max_cost_micros"]


def _quality(pair: dict, control: dict, treatment: dict) -> bool:
    if any(arm.get("independent_review", {}).get("accepted") is not True or arm.get("false_green") or arm.get("causal_loss") or arm.get("unauthorized_external_effects") or arm.get("severe_defects", 0) for arm in (control, treatment)):
        return False
    for arm in (control, treatment):
        tests = arm.get("tests", [])
        if not isinstance(tests, list) or any(not isinstance(test, dict) for test in tests):
            return False
        passed = {test.get("command") for test in tests if test.get("executed") and test.get("passed") and _evidence(test.get("evidence"))}
        if passed != set(pair["required_tests"]):
            return False
    return True


def _handoffs(pair: dict, treatment: dict) -> bool:
    expected = {stage["stage_ref"] for stage in pair["treatment_stages"][1:]}
    handoffs = treatment.get("handoffs", [])
    if not isinstance(handoffs, list):
        return False
    for handoff in handoffs:
        if not isinstance(handoff, dict):
            return False
        if handoff.get("stage_ref") not in expected or handoff.get("continued_from_package_and_refs") is not True or handoff.get("requested_raw_transcript") or handoff.get("out_of_scope_reads") != 0 or not _evidence(handoff.get("package")):
            return False
        expected.remove(handoff["stage_ref"])
    return not expected


def evaluate(manifest: dict, dataset: dict) -> dict:
    issues = validate_manifest(manifest)
    if issues:
        raise ValueError("invalid manifest: " + ",".join(issues))
    incomplete, failed, rows = [], [], []
    expected_hash = digest(manifest)
    if dataset.get("schema_version") != DATASET or dataset.get("experiment_ref") != manifest["experiment_ref"] or dataset.get("manifest_sha256") != expected_hash:
        incomplete.append("dataset_identity_invalid")
    results = {pair.get("pair_ref"): pair for pair in dataset.get("pairs", [])}
    if len(results) != len(dataset.get("pairs", [])) or set(results) - {p["pair_ref"] for p in manifest["pairs"]}:
        incomplete.append("duplicate_or_unknown_pair")
    token_savings, cost_savings, time_ratios = [], [], []
    unnecessary = {"control": 0, "treatment": 0}; unnecessary_improved = 0
    totals = {"control": [0, 0, 0], "treatment": [0, 0, 0]}; improved = 0
    for pair in manifest["pairs"]:
        result = results.get(pair["pair_ref"])
        arms = {arm.get("mode"): arm for arm in result.get("arms", [])} if result else {}
        if not result or len(result.get("arms", [])) != 2 or set(arms) != {"control", "treatment"}:
            incomplete.append(f"invalid_pair:{pair['pair_ref']}"); continue
        control, treatment = arms["control"], arms["treatment"]
        if not _complete(manifest, pair, control) or not _complete(manifest, pair, treatment):
            incomplete.append(f"incomplete_evidence:{pair['pair_ref']}"); continue
        control_use, treatment_use = _usage(control), _usage(treatment)
        if control_use[0] == 0 or control_use[1] == 0 or control["metrics"]["duration_ms"] == 0:
            incomplete.append(f"missing_denominator:{pair['pair_ref']}"); continue
        token_bps = (control_use[0] - treatment_use[0]) * 10000 // control_use[0]
        cost_bps = (control_use[1] - treatment_use[1]) * 10000 // control_use[1]
        control_unnecessary = control["independent_review"]["unnecessary_changed_lines"]
        treatment_unnecessary = treatment["independent_review"]["unnecessary_changed_lines"]
        time_ratio = treatment["metrics"]["duration_ms"] * 10000 // control["metrics"]["duration_ms"]
        rows.append({"pair_ref": pair["pair_ref"], "control_tokens": control_use[0], "treatment_tokens": treatment_use[0], "token_reduction_bps": token_bps, "control_cost_micros": control_use[1], "treatment_cost_micros": treatment_use[1], "cost_reduction_bps": cost_bps, "time_ratio_bps": time_ratio, "control_unnecessary_changed_lines": control_unnecessary, "treatment_unnecessary_changed_lines": treatment_unnecessary, "control_quality_score": control["independent_review"]["quality_score"], "treatment_quality_score": treatment["independent_review"]["quality_score"]})
        token_savings.append(token_bps); cost_savings.append(cost_bps); time_ratios.append(time_ratio)
        improved += treatment_use[0] < control_use[0] and treatment_use[1] < control_use[1]
        unnecessary["control"] += control_unnecessary; unnecessary["treatment"] += treatment_unnecessary
        unnecessary_improved += treatment_unnecessary < control_unnecessary
        if not _quality(pair, control, treatment): failed.append(f"quality_or_tests:{pair['pair_ref']}")
        if treatment["independent_review"]["quality_score"] < control["independent_review"]["quality_score"]: failed.append(f"quality_score_decreased:{pair['pair_ref']}")
        if treatment_unnecessary > control_unnecessary: failed.append(f"unnecessary_code_increased:{pair['pair_ref']}")
        if not _handoffs(pair, treatment): failed.append(f"handoff_continuity:{pair['pair_ref']}")
        if treatment.get("external_effects", 0) > control.get("external_effects", 0): failed.append(f"additional_external_effect:{pair['pair_ref']}")
        for mode, arm in arms.items():
            for index, key in enumerate(("reworks", "failed_attempts", "human_interventions")): totals[mode][index] += arm["metrics"][key]
    median_tokens = int(statistics.median(token_savings)) if len(rows) >= 3 else 0
    median_cost = int(statistics.median(cost_savings)) if len(rows) >= 3 else 0
    median_time = int(statistics.median(time_ratios)) if len(rows) >= 3 else 0
    if len(rows) < MINIMUM_PAIRS: incomplete.append("fewer_than_six_complete_pairs")
    if median_tokens < 2000: failed.append("token_reduction_below_20_percent")
    if median_cost < 2000: failed.append("cost_reduction_below_20_percent")
    minimum_improved = (2 * len(manifest["pairs"]) + 2) // 3
    if improved < minimum_improved: failed.append("fewer_than_two_thirds_pairs_improved")
    if unnecessary["treatment"] >= unnecessary["control"] or unnecessary_improved < minimum_improved: failed.append("unnecessary_code_not_reduced")
    if median_time > 11000: failed.append("time_above_110_percent")
    for index, name in enumerate(("reworks", "failed_attempts", "human_interventions")):
        if totals["treatment"][index] > totals["control"][index]: failed.append(f"{name}_increased")
    review = dataset.get("report_review", {})
    if review.get("completed") is not True or not _evidence_list(review.get("evidence")): incomplete.append("report_review_missing")
    elif review.get("accepted") is not True or review.get("reproduction_passed") is not True: failed.append("report_review_rejected")
    verdict, reasons = ("inconclusive", incomplete) if incomplete else (("keep_control", failed) if failed else ("adopt", ["all_apg_006_thresholds_satisfied"]))
    return {"schema_version": "orquesta.context_rotation.decision.v1", "experiment_ref": manifest["experiment_ref"], "manifest_sha256": expected_hash, "dataset_sha256": digest(dataset), "decision": verdict, "reasons": sorted(set(reasons)), "pair_metrics": rows, "median_token_reduction_bps": median_tokens, "median_cost_reduction_bps": median_cost, "median_time_ratio_bps": median_time, "improved_pairs": improved, "unnecessary_changed_lines": unnecessary, "unnecessary_code_improved_pairs": unnecessary_improved, "thresholds": {"minimum_pairs": MINIMUM_PAIRS, "minimum_token_reduction_bps": 2000, "minimum_cost_reduction_bps": 2000, "maximum_median_time_ratio_bps": 11000, "minimum_improved_pairs": minimum_improved, "quality_score_tolerance": 0, "unnecessary_code_must_decrease": True}}


def blind(manifest: dict, dataset: dict, pair_ref: str, mode: str, key: bytes) -> tuple[dict, dict]:
    if len(key) < 32 or hashlib.sha256(key).hexdigest() != manifest["reviewer"]["blinding_key_sha256"]:
        raise ValueError("blinding key invalid")
    pair = next(p for p in manifest["pairs"] if p["pair_ref"] == pair_ref)
    result = next(p for p in dataset["pairs"] if p["pair_ref"] == pair_ref)
    arm = next(a for a in result["arms"] if a["mode"] == mode)
    candidate = "candidate-" + hmac.new(key, f"{manifest['experiment_ref']}\0{pair_ref}\0{mode}".encode(), hashlib.sha256).hexdigest()[:24]
    if arm.get("blind_candidate_ref") != candidate:
        raise ValueError("dataset blind_candidate_ref mismatch")
    if not _evidence(arm.get("diff")) or not _evidence_list(arm.get("artifacts")) or not arm.get("tests") or any(not _evidence(test.get("evidence")) for test in arm["tests"]):
        raise ValueError("candidate evidence incomplete")
    package = {"schema_version": "orquesta.context_rotation.blind_review.v1", "experiment_ref": manifest["experiment_ref"], "pair_ref": pair_ref, "candidate_ref": candidate, "objective": pair["objective"], "acceptance_criteria": pair["acceptance_criteria"], "required_tests": pair["required_tests"], "review_contract": {"quality_scale": manifest["reviewer"]["quality_scale"], "unnecessary_code_definition": manifest["reviewer"]["unnecessary_code_definition"], "instructions_sha256": manifest["reviewer"]["review_instructions_sha256"]}, "diff": arm["diff"], "artifacts": arm["artifacts"], "test_evidence": [test["evidence"] for test in arm["tests"]]}
    mapping = {"schema_version": "orquesta.context_rotation.blind_mapping.v1", "experiment_ref": manifest["experiment_ref"], "pair_ref": pair_ref, "candidate_ref": candidate, "source_candidate_ref": arm["candidate_ref"], "mode": mode, "package_sha256": digest(package)}
    return package, mapping
