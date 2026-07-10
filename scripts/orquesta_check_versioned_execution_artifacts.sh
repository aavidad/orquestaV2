#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: scripts/orquesta_check_versioned_execution_artifacts.sh [--root DIR] [--audit FILE] [--json]" >&2
}

root=""
audit=""
json=false

while [ "$#" -gt 0 ]; do
  case "$1" in
    --root)
      root="${2:-}"
      shift 2
      ;;
    --audit)
      audit="${2:-}"
      shift 2
      ;;
    --json)
      json=true
      shift
      ;;
    *)
      usage
      exit 2
      ;;
  esac
done

if [ -z "$root" ]; then
  root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
fi
if [ -z "$root" ] || { [ ! -d "$root/.git" ] && [ ! -f "$root/.git" ]; }; then
  echo "orquesta_check_versioned_execution_artifacts: root must be a Git worktree" >&2
  exit 2
fi

root="$(cd "$root" && pwd)"
if [ -z "$audit" ]; then
  audit="$root/docs/auditorias/s13_artefactos_ejecucion_versionados_2026-07-10.json"
fi
if [ ! -f "$audit" ]; then
  echo "orquesta_check_versioned_execution_artifacts: audit file not found" >&2
  exit 2
fi

python3 - "$root" "$audit" "$json" <<'PY'
import json
import subprocess
import sys

root, audit_path, json_mode = sys.argv[1:]
try:
    with open(audit_path, encoding="utf-8") as fh:
        audit = json.load(fh)
except (OSError, json.JSONDecodeError) as exc:
    raise SystemExit(f"orquesta_check_versioned_execution_artifacts: audit_invalid:{exc}")

if audit.get("schema_version") != "orquesta_s13_versioned_execution_artifact_audit.v0":
    raise SystemExit("orquesta_check_versioned_execution_artifacts: audit_schema_invalid")

entries = audit.get("entries")
if not isinstance(entries, list):
    raise SystemExit("orquesta_check_versioned_execution_artifacts: audit_entries_invalid")

allowed_categories = {
    "historical_evidence",
    "test_fixture",
    "movable_runtime_artifact",
    "retain_pending_reference",
}
audited = {}
for entry in entries:
    if not isinstance(entry, dict):
        raise SystemExit("orquesta_check_versioned_execution_artifacts: audit_entry_invalid")
    path = entry.get("path")
    category = entry.get("category")
    if not isinstance(path, str) or not path or category not in allowed_categories or path in audited:
        raise SystemExit("orquesta_check_versioned_execution_artifacts: audit_entry_invalid")
    audited[path] = category

tracked = subprocess.check_output(["git", "-C", root, "ls-files"], text=True).splitlines()
def is_execution_artifact(path):
    return (
        "checkpoint_started_goal-ref" in path
        or "orquesta_goal_result_goal-ref" in path
        or "goal-ref-task-autoprogramming" in path
        or "goal-ref-autoprogramming-backlog" in path
    )

candidates = {path for path in tracked if is_execution_artifact(path)}
unclassified = sorted(candidates - set(audited))
stale_audit = sorted(set(audited) - candidates)
categories = {category: 0 for category in sorted(allowed_categories)}
for path in candidates:
    category = audited.get(path)
    if category:
        categories[category] += 1

report = {
    "schema_version": "orquesta_versioned_execution_artifact_guard.v0",
    "status": "passed" if not unclassified and not stale_audit else "failed",
    "tracked_candidate_count": len(candidates),
    "audited_entry_count": len(audited),
    "categories": categories,
    "unclassified_paths": unclassified,
    "stale_audit_paths": stale_audit,
}
if json_mode == "true":
    print(json.dumps(report, separators=(",", ":"), sort_keys=True))
else:
    print(f"versioned_execution_artifacts_status={report['status']}")
    print(f"versioned_execution_artifacts_tracked={report['tracked_candidate_count']}")
    print(f"versioned_execution_artifacts_unclassified={len(unclassified)}")
    print(f"versioned_execution_artifacts_stale_audit={len(stale_audit)}")

if report["status"] != "passed":
    raise SystemExit(1)
PY
