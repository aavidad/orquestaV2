#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
manifest_path="${ORQUESTA_GOLDEN_EVALS_MANIFEST:-${repo_root}/docs/evals/orquesta_golden_tasks_v0.json}"

python3 - "$manifest_path" "$repo_root" "$@" <<'PY'
import argparse
import concurrent.futures
import datetime as _dt
import hashlib
import json
import os
from pathlib import Path
import re
import shlex
import subprocess
import sys
import tempfile


MANIFEST_PATH = Path(sys.argv[1])
REPO_ROOT = Path(sys.argv[2])
ARGV = sys.argv[3:]


def utc_now():
    override = os.environ.get("ORQUESTA_GOLDEN_EVALS_NOW", "").strip()
    if override:
        return override
    return _dt.datetime.now(_dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def read_json(path):
    with Path(path).open("r", encoding="utf-8") as handle:
        return json.load(handle)


def write_json(path, data):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as handle:
        json.dump(data, handle, ensure_ascii=False, indent=2, sort_keys=True)
        handle.write("\n")


def sha256_file(path):
    digest = hashlib.sha256()
    with Path(path).open("rb") as handle:
        for chunk in iter(lambda: handle.read(65536), b""):
            digest.update(chunk)
    return digest.hexdigest()


def display_results_dir(path):
    resolved = Path(path).resolve()
    try:
        return str(resolved.relative_to(REPO_ROOT))
    except ValueError:
        return "external-results-dir"


def load_manifest(path):
    manifest = read_json(path)
    issues = []
    if manifest.get("schema_version") != "orquesta_golden_tasks.v0":
        issues.append("manifest_schema_version_invalid")
    tasks = manifest.get("tasks", [])
    if len(tasks) != 5:
        issues.append("manifest_requires_five_tasks")
    classes = [task.get("task_class", "") for task in tasks]
    if len(set(classes)) != len(classes):
        issues.append("manifest_task_classes_not_distinct")
    for task in tasks:
        if not task.get("task_id"):
            issues.append("task_without_id")
        if not task.get("write_set"):
            issues.append(f"{task.get('task_id', 'task')}:missing_write_set")
        if not task.get("expected_files"):
            issues.append(f"{task.get('task_id', 'task')}:missing_expected_files")
        if not task.get("required_tests"):
            issues.append(f"{task.get('task_id', 'task')}:missing_required_tests")
        for verifier in task.get("verifier_refs", []):
            if verifier not in VERIFIERS:
                issues.append(f"{task.get('task_id', 'task')}:unknown_verifier:{verifier}")
    return manifest, issues


def norm_list(value):
    if value is None:
        return []
    if isinstance(value, list):
        return value
    if isinstance(value, dict):
        return list(value.values())
    return [value]


def result_tests(result):
    tests = {}
    raw = result.get("tests", [])
    if isinstance(raw, dict):
        for command, status in raw.items():
            tests[str(command)] = str(status)
        return tests
    for item in norm_list(raw):
        if isinstance(item, str):
            tests[item] = "passed"
        elif isinstance(item, dict):
            command = item.get("command") or item.get("cmd") or item.get("test")
            status = item.get("status") or item.get("result")
            if command:
                tests[str(command)] = str(status or "")
    return tests


def has_passed_test(result, command):
    tests = result_tests(result)
    return tests.get(command) == "passed"


def result_paths(result):
    paths = []
    for key in ("touched_files", "files", "artifact_paths", "artifacts"):
        for item in norm_list(result.get(key)):
            if isinstance(item, str):
                paths.append(item)
            elif isinstance(item, dict):
                path = item.get("path") or item.get("file")
                if path:
                    paths.append(str(path))
    return paths


def metric_int(value):
    if isinstance(value, bool) or value is None:
        return 0
    if isinstance(value, list):
        return len(value)
    if isinstance(value, dict):
        return len(value)
    try:
        return int(value)
    except (TypeError, ValueError):
        try:
            return int(float(value))
        except (TypeError, ValueError):
            return 0


def result_metric_value(raw, key, default=0):
    if isinstance(raw, dict) and key in raw:
        return metric_int(raw.get(key))
    return default


def result_metrics(result):
    raw = result.get("metrics", {})
    if not isinstance(raw, dict):
        raw = {}
    tokens = raw.get("tokens", {})
    if not isinstance(tokens, dict):
        tokens = {}
    touched_files = result_metric_value(raw, "files_touched", len(set(result.get("touched_files", []))))
    scope_reason = raw.get("scope_expansion_reason") or result.get("scope_expansion_reason")
    return {
        "reported": bool(raw),
        "input_tokens": result_metric_value(raw, "input_tokens", metric_int(tokens.get("input"))),
        "output_tokens": result_metric_value(raw, "output_tokens", metric_int(tokens.get("output"))),
        "reasoning_tokens": result_metric_value(raw, "reasoning_tokens", metric_int(tokens.get("reasoning"))),
        "cached_input_tokens": result_metric_value(raw, "cached_input_tokens", metric_int(tokens.get("cached_input"))),
        "total_tokens": result_metric_value(raw, "total_tokens", metric_int(tokens.get("total"))),
        "tool_calls": result_metric_value(raw, "tool_calls"),
        "elapsed_ms": result_metric_value(raw, "elapsed_ms"),
        "files_touched": touched_files,
        "new_files_count": result_metric_value(raw, "new_files_count"),
        "lines_added": result_metric_value(raw, "lines_added"),
        "lines_deleted": result_metric_value(raw, "lines_deleted"),
        "helpers_added": result_metric_value(raw, "helpers_added"),
        "abstractions_added": result_metric_value(raw, "abstractions_added"),
        "rework_count": result_metric_value(raw, "rework_count"),
        "scope_expansions": 1 if str(scope_reason or "").strip() else 0,
    }


def empty_metrics_summary():
    return {
        "tasks_with_metrics": 0,
        "input_tokens_total": 0,
        "output_tokens_total": 0,
        "reasoning_tokens_total": 0,
        "cached_input_tokens_total": 0,
        "total_tokens": 0,
        "tool_calls_total": 0,
        "elapsed_ms_total": 0,
        "files_touched_total": 0,
        "new_files_total": 0,
        "lines_added_total": 0,
        "lines_deleted_total": 0,
        "helpers_added_total": 0,
        "abstractions_added_total": 0,
        "rework_count_total": 0,
        "scope_expansions_total": 0,
    }


def merge_metrics_summary(summary, metrics):
    if metrics.get("reported"):
        summary["tasks_with_metrics"] += 1
    for key in (
        "input_tokens",
        "output_tokens",
        "reasoning_tokens",
        "cached_input_tokens",
        "tool_calls",
        "elapsed_ms",
        "files_touched",
        "new_files_count",
        "lines_added",
        "lines_deleted",
        "helpers_added",
        "abstractions_added",
        "rework_count",
        "scope_expansions",
    ):
        summary_key = {
            "input_tokens": "input_tokens_total",
            "output_tokens": "output_tokens_total",
            "reasoning_tokens": "reasoning_tokens_total",
            "cached_input_tokens": "cached_input_tokens_total",
            "tool_calls": "tool_calls_total",
            "elapsed_ms": "elapsed_ms_total",
            "files_touched": "files_touched_total",
            "new_files_count": "new_files_total",
            "lines_added": "lines_added_total",
            "lines_deleted": "lines_deleted_total",
            "helpers_added": "helpers_added_total",
            "abstractions_added": "abstractions_added_total",
            "rework_count": "rework_count_total",
            "scope_expansions": "scope_expansions_total",
        }[key]
        summary[summary_key] += metric_int(metrics.get(key))
    summary["total_tokens"] += metric_int(metrics.get("total_tokens")) or (
        metric_int(metrics.get("input_tokens"))
        + metric_int(metrics.get("output_tokens"))
        + metric_int(metrics.get("reasoning_tokens"))
    )


def path_is_under(path, prefixes):
    clean = path.strip().replace("\\", "/")
    if not clean or clean.startswith("/") or clean.startswith("../") or "/../" in clean:
        return False
    for prefix in prefixes:
        prefix = str(prefix).strip().replace("\\", "/").rstrip("/")
        if clean == prefix or clean.startswith(prefix + "/"):
            return True
    return False


def expected_file_present(task, result, task_dir, expected):
    if expected in result_paths(result):
        return True
    candidate = task_dir / "worktree" / expected
    return candidate.is_file()


def skip_reason_seen(planner, reason):
    for item in norm_list(planner.get("skipped")):
        if isinstance(item, dict) and item.get("reason") == reason:
            return True
    return False


def check_tests_pass(task, result, task_dir, report):
    issues = []
    for command in task.get("required_tests", []):
        if not has_passed_test(result, command):
            issues.append(f"required_test_not_passed:{command}")
    return issues


def check_expected_files(task, result, task_dir, report):
    return [
        f"expected_file_missing:{expected}"
        for expected in task.get("expected_files", [])
        if not expected_file_present(task, result, task_dir, expected)
    ]


def check_writeset_only(task, result, task_dir, report):
    issues = []
    prefixes = task.get("write_set", [])
    for path in result.get("touched_files", []):
        if not path_is_under(str(path), prefixes):
            issues.append(f"out_of_write_set:{path}")
    for path in result_paths(result):
        if str(path).startswith("/") or str(path).startswith("../") or "/../" in str(path):
            issues.append(f"unsafe_path:{path}")
    return issues


def check_no_opes_productive(task, result, task_dir, report):
    manifest = report["manifest"]
    forbidden = list(manifest.get("global_forbidden_path_prefixes", [])) + list(task.get("forbidden_path_prefixes", []))
    issues = []
    for path in result_paths(result):
        normalized = str(path).lower().replace("\\", "/")
        for prefix in forbidden:
            if normalized.startswith(str(prefix).lower().replace("\\", "/")):
                issues.append(f"forbidden_path:{path}")
    if result.get("opes_productive_touched") is True:
        issues.append("opes_productive_touched")
    return issues


def check_idle_default_60(task, result, task_dir, report):
    evidence = result.get("evidence", {})
    decisions = evidence.get("idle_decisions", {})
    if evidence.get("idle_default_after_seconds") != 60:
        return ["idle_default_after_seconds_not_60"]
    early = decisions.get("59", {})
    ready = decisions.get("60", {})
    if early.get("prepare") is not False or ready.get("prepare") is not True:
        return ["idle_default_60_decision_invalid"]
    return []


def check_idle_zero_disables_clock(task, result, task_dir, report):
    evidence = result.get("evidence", {})
    if evidence.get("idle_zero_clock_disabled") is not True:
        return ["idle_zero_clock_not_disabled"]
    if evidence.get("idle_zero_capacity_prepare") is not True:
        return ["idle_zero_capacity_not_preserved"]
    return []


def check_idle_or_capacity_prepares(task, result, task_dir, report):
    evidence = result.get("evidence", {})
    if evidence.get("idle_prepare") is True and evidence.get("capacity_prepare") is True:
        return []
    return ["idle_or_capacity_prepare_missing"]


def check_planner_skips_visible_queue(task, result, task_dir, report):
    planner = result.get("evidence", {}).get("planner", {})
    if skip_reason_seen(planner, "visible_in_queue"):
        return []
    return ["planner_visible_queue_skip_missing"]


def check_planner_creates_scanner(task, result, task_dir, report):
    scanner = result.get("evidence", {}).get("planner", {}).get("scanner", {})
    if scanner.get("title") == "Escaneo backlog nuevos" or scanner.get("section_ref") == "backlog_scanner":
        return []
    return ["planner_scanner_missing"]


def check_narrative_sections_filtered(task, result, task_dir, report):
    planner = result.get("evidence", {}).get("planner", {})
    tasks = norm_list(planner.get("tasks"))
    if not skip_reason_seen(planner, "narrative_section"):
        return ["narrative_section_skip_missing"]
    for item in tasks:
        if isinstance(item, dict) and item.get("section_ref") == "estado-actual":
            return ["narrative_section_materialized_as_task"]
    return []


def check_scanner_ack(task, result, task_dir, report):
    evidence = result.get("evidence", {}).get("backlog_scan", {})
    issues = []
    expected_ref = task.get("expected_scan_ref")
    if expected_ref and evidence.get("scan_ref") != expected_ref:
        issues.append("backlog_scan_ref_missing")
    docs = evidence.get("documents", [])
    by_key = {
        (doc.get("path"), int(doc.get("line", 0)) if str(doc.get("line", "")).isdigit() else doc.get("line")): doc
        for doc in docs
        if isinstance(doc, dict)
    }
    for expected in task.get("expected_scan_docs", []):
        doc = by_key.get((expected.get("path"), expected.get("line")))
        if not doc:
            issues.append(f"backlog_scan_doc_missing:{expected.get('path')}:{expected.get('line')}")
            continue
        sha = str(doc.get("sha256", ""))
        if not re.fullmatch(r"[0-9a-f]{64}", sha):
            issues.append(f"backlog_scan_doc_hash_invalid:{expected.get('path')}:{expected.get('line')}")
    return issues


def check_scanner_doc_hash_guard(task, result, task_dir, report):
    evidence = result.get("evidence", {}).get("backlog_scan", {})
    if evidence.get("documents_match_snapshot") is True:
        return []
    if evidence.get("proposal_status") == "draft_rebase_pending" and evidence.get("rebase_or_merge_pending") is True:
        return []
    return ["scanner_doc_changed_without_draft_rebase_pending"]


def check_backlog_duplicate_task_id_ambiguous(task, result, task_dir, report):
    evidence = result.get("evidence", {}).get("duplicate_task_id", {})
    if (
        evidence.get("caller_reference") == "T289"
        and int(evidence.get("duplicate_count", 0)) > 1
        and evidence.get("issue_code") == "backlog_duplicate_task_id_ambiguous"
    ):
        return []
    return ["backlog_duplicate_task_id_ambiguous_missing"]


def check_public_projection(task, result, task_dir, report):
    states = set(result.get("evidence", {}).get("public_projection_states", []))
    want = {"outbox_pending", "wait_external", "external_process_verified"}
    missing = sorted(want - states)
    return [f"public_projection_state_missing:{state}" for state in missing]


def check_frozen_files(task, result, task_dir, report):
    touched = set(result.get("touched_files", []))
    changed = sorted(set(task.get("frozen_files", [])) & touched)
    return [f"frozen_file_modified:{path}" for path in changed]


def check_root_cause_corrected(task, result, task_dir, report):
    evidence = result.get("evidence", {})
    if evidence.get("failure_evidence_used") is True and evidence.get("root_cause_corrected") is True:
        return []
    return ["root_cause_not_proven_from_failure_evidence"]


VERIFIERS = {
    "tests_pass": check_tests_pass,
    "expected_files_present": check_expected_files,
    "writeset_only": check_writeset_only,
    "no_opes_productive_touch": check_no_opes_productive,
    "idle_default_60": check_idle_default_60,
    "idle_zero_disables_clock": check_idle_zero_disables_clock,
    "idle_or_capacity_prepares": check_idle_or_capacity_prepares,
    "planner_skips_visible_queue": check_planner_skips_visible_queue,
    "planner_creates_scanner": check_planner_creates_scanner,
    "narrative_sections_filtered": check_narrative_sections_filtered,
    "scanner_ack_cites_ref_lines_hashes": check_scanner_ack,
    "scanner_doc_hash_guard": check_scanner_doc_hash_guard,
    "backlog_duplicate_task_id_ambiguous": check_backlog_duplicate_task_id_ambiguous,
    "public_projection_distinguishes_external_work": check_public_projection,
    "frozen_files_unchanged": check_frozen_files,
    "root_cause_corrected_from_failure_evidence": check_root_cause_corrected,
}


def load_result(results_dir, task):
    task_dir = Path(results_dir) / task["task_id"]
    result_path = task_dir / "result.json"
    if not result_path.is_file():
        return task_dir, None
    return task_dir, read_json(result_path)


def evaluate(results_dir, manifest, manifest_issues):
    generated_at = utc_now()
    report = {
        "schema_version": "orquesta_golden_eval_result.v0",
        "generated_at": generated_at,
        "task_set_ref": manifest.get("task_set_ref"),
        "manifest_sha256": sha256_file(MANIFEST_PATH),
        "results_dir": display_results_dir(results_dir),
        "manifest": manifest,
        "manifest_issues": manifest_issues,
        "summary": {
            "total": len(manifest.get("tasks", [])),
            "passed": 0,
            "failed": 0,
            "score": 0.0,
            "task_classes": sorted({task.get("task_class", "") for task in manifest.get("tasks", [])}),
            "metrics": empty_metrics_summary(),
        },
        "tasks": [],
    }
    for task in manifest.get("tasks", []):
        task_dir, result = load_result(results_dir, task)
        issues = []
        if result is None:
            result = {}
            issues.append("result_missing")
        if result.get("task_id") not in ("", None, task.get("task_id")):
            issues.append("task_id_mismatch")
        if result.get("status") != "complete":
            issues.append("task_status_not_complete")
        for verifier in task.get("verifier_refs", []):
            check = VERIFIERS.get(verifier)
            if check is None:
                issues.append(f"unknown_verifier:{verifier}")
            else:
                issues.extend(check(task, result, task_dir, report))
        metrics = result_metrics(result)
        merge_metrics_summary(report["summary"]["metrics"], metrics)
        passed = len(issues) == 0
        if passed:
            report["summary"]["passed"] += 1
        else:
            report["summary"]["failed"] += 1
        report["tasks"].append({
            "task_id": task.get("task_id"),
            "task_class": task.get("task_class"),
            "status": "passed" if passed else "failed",
            "score": 1.0 if passed else 0.0,
            "issues": issues,
            "verifier_refs": task.get("verifier_refs", []),
            "metrics": metrics,
        })
    total = report["summary"]["total"]
    if total:
        report["summary"]["score"] = round(report["summary"]["passed"] / total, 4)
    if manifest_issues:
        report["summary"]["failed"] = total
        report["summary"]["passed"] = 0
        report["summary"]["score"] = 0.0
    report["status"] = "passed" if report["summary"]["failed"] == 0 and not manifest_issues else "failed"
    del report["manifest"]
    return report


def synthetic_evidence(task):
    refs = set(task.get("verifier_refs", []))
    evidence = {}
    if "idle_default_60" in refs:
        evidence["idle_default_after_seconds"] = 60
        evidence["idle_decisions"] = {
            "59": {"prepare": False},
            "60": {"prepare": True, "reason": "idle_after_seconds_reached"},
        }
    if "idle_zero_disables_clock" in refs:
        evidence["idle_zero_clock_disabled"] = True
        evidence["idle_zero_capacity_prepare"] = True
    if "idle_or_capacity_prepares" in refs:
        evidence["idle_prepare"] = True
        evidence["capacity_prepare"] = True
    if "root_cause_corrected_from_failure_evidence" in refs:
        evidence["failure_evidence_used"] = True
        evidence["root_cause_corrected"] = True
    if refs & {"planner_skips_visible_queue", "planner_creates_scanner", "narrative_sections_filtered"}:
        evidence["planner"] = {
            "skipped": [
                {"task_ref": "task-ref-visible", "reason": "visible_in_queue"},
                {"section_ref": "estado-actual", "reason": "narrative_section"},
            ],
            "tasks": [{"task_ref": "task-ref-new-gap", "section_ref": "t999"}],
            "scanner": {"title": "Escaneo backlog nuevos", "section_ref": "backlog_scanner"},
        }
    if "scanner_ack_cites_ref_lines_hashes" in refs or "scanner_doc_hash_guard" in refs:
        evidence["backlog_scan"] = {
            "scan_ref": task.get("expected_scan_ref", "scan-ref-backlog-f6ae5d12b919"),
            "documents": task.get("expected_scan_docs", []),
            "documents_match_snapshot": False,
            "proposal_status": "draft_rebase_pending",
            "rebase_or_merge_pending": True,
        }
    if "backlog_duplicate_task_id_ambiguous" in refs:
        evidence["duplicate_task_id"] = {
            "caller_reference": "T289",
            "duplicate_count": 2,
            "issue_code": "backlog_duplicate_task_id_ambiguous",
        }
    if "public_projection_distinguishes_external_work" in refs:
        evidence["public_projection_states"] = [
            "outbox_pending",
            "wait_external",
            "external_process_verified",
        ]
    return evidence


def write_synthetic_results(results_dir, manifest):
    for task in manifest.get("tasks", []):
        task_dir = Path(results_dir) / task["task_id"]
        (task_dir / "worktree").mkdir(parents=True, exist_ok=True)
        for expected in task.get("expected_files", []):
            path = task_dir / "worktree" / expected
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text("golden eval synthetic artifact\n", encoding="utf-8")
        result = {
            "task_id": task["task_id"],
            "status": "complete",
            "touched_files": list(task.get("expected_files", [])),
            "artifact_paths": list(task.get("expected_files", [])),
            "tests": [{"command": command, "status": "passed"} for command in task.get("required_tests", [])],
            "evidence": synthetic_evidence(task),
            "metrics": {
                "files_touched": len(task.get("expected_files", [])),
                "new_files_count": len(task.get("expected_files", [])),
                "lines_added": len(task.get("expected_files", [])),
                "lines_deleted": 0,
                "tool_calls": 1,
                "elapsed_ms": 1,
                "total_tokens": 1,
            },
            "opes_productive_touched": False,
        }
        write_json(task_dir / "result.json", result)


def require_confirmation(args):
    confirmed = args.confirm or os.environ.get("ORQUESTA_GOLDEN_EVALS_CONFIRM") == "isolated"
    if not confirmed:
        raise SystemExit("confirmacion requerida: usa --confirm o ORQUESTA_GOLDEN_EVALS_CONFIRM=isolated")
    for key in ("OPES_PRODUCTION", "ORQUESTA_OPES_PRODUCTION"):
        if os.environ.get(key, "").lower() in {"1", "true", "yes", "prod", "production"}:
            raise SystemExit(f"entorno OPES productivo no permitido: {key}")
    if os.environ.get("OPES_BASE_URL") or os.environ.get("ORQUESTA_OPES_BASE_URL"):
        if os.environ.get("ORQUESTA_GOLDEN_EVALS_ALLOW_OPES_ENV") != "1":
            raise SystemExit("variables OPES detectadas; abortado para no tocar OPES productivo")


def default_output_path(prefix):
    stamp = utc_now().replace(":", "").replace("-", "").replace("Z", "Z")
    return REPO_ROOT / "docs/evals/results" / f"{prefix}_{stamp}.json"


def selected_tasks(manifest, task_ids):
    if not task_ids:
        return manifest.get("tasks", [])
    wanted = set(task_ids)
    tasks = [task for task in manifest.get("tasks", []) if task.get("task_id") in wanted]
    missing = wanted - {task.get("task_id") for task in tasks}
    if missing:
        raise SystemExit("tareas no encontradas: " + ", ".join(sorted(missing)))
    return tasks


def manifest_with_tasks(manifest, tasks):
    selected = dict(manifest)
    selected["tasks"] = list(tasks)
    selected["selected_task_ids"] = [task.get("task_id") for task in tasks]
    return selected


def write_requests(run_root, manifest, tasks):
    requests_dir = Path(run_root) / "requests"
    requests_dir.mkdir(parents=True, exist_ok=True)
    for task in tasks:
        packet = {
            "schema_version": "orquesta_golden_task_request.v0",
            "task_set_ref": manifest.get("task_set_ref"),
            "task": task,
            "result_contract": {
                "result_path": f"results/{task['task_id']}/result.json",
                "required_status": "complete",
                "required_checks": task.get("verifier_refs", []),
            },
        }
        write_json(requests_dir / f"{task['task_id']}.json", packet)
    return requests_dir


def launch_task(command, run_root, task):
    request_path = Path(run_root) / "requests" / f"{task['task_id']}.json"
    result_dir = Path(run_root) / "results" / task["task_id"]
    result_dir.mkdir(parents=True, exist_ok=True)
    env = os.environ.copy()
    env["ORQUESTA_GOLDEN_TASK_ID"] = task["task_id"]
    env["ORQUESTA_GOLDEN_TASK_REQUEST"] = str(request_path)
    env["ORQUESTA_GOLDEN_TASK_RESULT_DIR"] = str(result_dir)
    completed = subprocess.run(command, shell=True, cwd=str(REPO_ROOT), env=env)
    if completed.returncode != 0:
        write_json(result_dir / "result.json", {
            "task_id": task["task_id"],
            "status": "failed",
            "launch_exit_code": completed.returncode,
            "touched_files": [],
            "tests": [],
            "evidence": {},
        })
    return completed.returncode


def run_tasks(args, manifest, manifest_issues):
    require_confirmation(args)
    run_root = Path(args.results_dir or default_output_path("orquesta_golden_run").with_suffix("")).resolve()
    tasks = selected_tasks(manifest, args.task)
    write_requests(run_root, manifest, tasks)
    if args.prepare_only:
        payload = {
            "schema_version": "orquesta_golden_eval_prepare.v0",
            "status": "prepared",
            "generated_at": utc_now(),
            "run_root": str(run_root),
            "request_count": len(tasks),
            "request_dir": str(run_root / "requests"),
        }
        output = Path(args.output) if args.output else default_output_path("orquesta_golden_prepare")
        write_json(output, payload)
        print(json.dumps(payload, ensure_ascii=False, sort_keys=True))
        return
    if not args.launcher_command:
        raise SystemExit("--launcher-command es obligatorio para --run sin --prepare-only")
    if args.parallel:
        with concurrent.futures.ThreadPoolExecutor(max_workers=len(tasks) or 1) as pool:
            list(pool.map(lambda task: launch_task(args.launcher_command, run_root, task), tasks))
    else:
        for task in tasks:
            launch_task(args.launcher_command, run_root, task)
    report = evaluate(run_root / "results", manifest_with_tasks(manifest, tasks), manifest_issues)
    output = Path(args.output) if args.output else default_output_path("orquesta_golden_eval")
    write_json(output, report)
    print(json.dumps({"status": report["status"], "output": str(output), "score": report["summary"]["score"]}, sort_keys=True))


def main():
    parser = argparse.ArgumentParser(description="Golden task evaluator for Orquesta agent quality drift.")
    parser.add_argument("--list", action="store_true", help="Listar tareas doradas.")
    parser.add_argument("--self-test", action="store_true", help="Ejecutar verificacion focal con fixtures sinteticos.")
    parser.add_argument("--evaluate", action="store_true", help="Evaluar un directorio de resultados existente.")
    parser.add_argument("--run", action="store_true", help="Preparar y lanzar tareas contra una instancia aislada.")
    parser.add_argument("--prepare-only", action="store_true", help="Solo escribir requests de tareas doradas.")
    parser.add_argument("--parallel", action="store_true", help="Lanzar tareas en paralelo.")
    parser.add_argument("--confirm", action="store_true", help="Confirmar ejecucion opt-in aislada.")
    parser.add_argument("--results-dir", help="Directorio de resultados o run root segun modo.")
    parser.add_argument("--output", help="Ruta del JSON de salida.")
    parser.add_argument("--launcher-command", help="Comando opt-in que consume ORQUESTA_GOLDEN_TASK_REQUEST.")
    parser.add_argument("--task", action="append", default=[], help="task_id a incluir; repetible.")
    args = parser.parse_args(ARGV)

    manifest, manifest_issues = load_manifest(MANIFEST_PATH)
    if args.list:
        for task in manifest.get("tasks", []):
            print(f"{task['task_id']}\t{task['task_class']}")
        return
    if args.self_test:
        with tempfile.TemporaryDirectory(prefix="orquesta-golden-evals-") as tmp:
            results_dir = Path(tmp) / "results"
            write_synthetic_results(results_dir, manifest)
            report = evaluate(results_dir, manifest, manifest_issues)
            output = Path(args.output) if args.output else default_output_path("orquesta_golden_self_test")
            write_json(output, report)
            if report["status"] != "passed":
                print(json.dumps(report, ensure_ascii=False, sort_keys=True))
                raise SystemExit(1)
            print(json.dumps({"status": "passed", "output": str(output), "score": report["summary"]["score"]}, sort_keys=True))
        return
    if args.evaluate:
        if not args.results_dir:
            raise SystemExit("--results-dir es obligatorio con --evaluate")
        tasks = selected_tasks(manifest, args.task)
        report = evaluate(Path(args.results_dir), manifest_with_tasks(manifest, tasks), manifest_issues)
        output = Path(args.output) if args.output else default_output_path("orquesta_golden_eval")
        write_json(output, report)
        print(json.dumps({"status": report["status"], "output": str(output), "score": report["summary"]["score"]}, sort_keys=True))
        if report["status"] != "passed":
            raise SystemExit(1)
        return
    if args.run or args.prepare_only:
        run_tasks(args, manifest, manifest_issues)
        return
    parser.print_help()
    raise SystemExit(2)


if __name__ == "__main__":
    main()
PY
