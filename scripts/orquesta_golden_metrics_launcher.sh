#!/usr/bin/env bash
set -euo pipefail

inner="${ORQUESTA_GOLDEN_METRICS_INNER_LAUNCHER:-}"
if [[ -z "$inner" ]]; then
  echo "ORQUESTA_GOLDEN_METRICS_INNER_LAUNCHER requerido" >&2
  exit 2
fi

task_id="${ORQUESTA_GOLDEN_TASK_ID:?ORQUESTA_GOLDEN_TASK_ID requerido}"
result_dir="${ORQUESTA_GOLDEN_TASK_RESULT_DIR:?ORQUESTA_GOLDEN_TASK_RESULT_DIR requerido}"
result_file="$result_dir/result.json"
mkdir -p "$result_dir"

start_ms="$(python3 - <<'PY'
import time
print(int(time.time() * 1000))
PY
)"

set +e
bash -lc "$inner"
exit_code="$?"
set -e

end_ms="$(python3 - <<'PY'
import time
print(int(time.time() * 1000))
PY
)"
elapsed_ms=$((end_ms - start_ms))

python3 - "$result_file" "$task_id" "$exit_code" "$elapsed_ms" <<'PY'
import json
import os
from pathlib import Path
import subprocess
import sys


result_path = Path(sys.argv[1])
task_id = sys.argv[2]
exit_code = int(sys.argv[3])
elapsed_ms = int(sys.argv[4])
result_dir = result_path.parent


def read_payload(path):
    if not path.is_file():
        return {
            "task_id": task_id,
            "status": "failed" if exit_code else "complete",
            "touched_files": [],
            "tests": [],
            "evidence": {},
        }
    with path.open("r", encoding="utf-8") as handle:
        loaded = json.load(handle)
    if not isinstance(loaded, dict):
        loaded = {}
    loaded.setdefault("task_id", task_id)
    return loaded


def metric_int(value):
    if isinstance(value, bool) or value is None:
        return 0
    if isinstance(value, (list, dict)):
        return len(value)
    try:
        return int(value)
    except (TypeError, ValueError):
        try:
            return int(float(value))
        except (TypeError, ValueError):
            return 0


def result_paths(payload):
    out = []
    for key in ("touched_files", "files", "artifact_paths", "artifacts"):
        raw = payload.get(key)
        values = raw if isinstance(raw, list) else []
        for item in values:
            if isinstance(item, str):
                out.append(item)
            elif isinstance(item, dict):
                path = item.get("path") or item.get("file")
                if path:
                    out.append(str(path))
    return out


def git_metrics(worktree):
    worktree = Path(worktree)
    if not (worktree / ".git").exists():
        return {}
    metrics = {}
    diff = subprocess.run(
        ["git", "-C", str(worktree), "diff", "--numstat"],
        text=True,
        capture_output=True,
        check=False,
    )
    if diff.returncode == 0:
        added = 0
        deleted = 0
        for line in diff.stdout.splitlines():
            parts = line.split("\t")
            if len(parts) < 3:
                continue
            if parts[0].isdigit():
                added += int(parts[0])
            if parts[1].isdigit():
                deleted += int(parts[1])
        metrics["lines_added"] = added
        metrics["lines_deleted"] = deleted
    status = subprocess.run(
        ["git", "-C", str(worktree), "status", "--porcelain"],
        text=True,
        capture_output=True,
        check=False,
    )
    if status.returncode == 0:
        lines = [line for line in status.stdout.splitlines() if line.strip()]
        metrics["files_touched"] = len(lines)
        metrics["new_files_count"] = len([line for line in lines if line.startswith("??") or line[:2].strip() == "A"])
    return metrics


payload = read_payload(result_path)
metrics = payload.get("metrics")
if not isinstance(metrics, dict):
    metrics = {}

paths = set(result_paths(payload))
defaults = {
    "elapsed_ms": elapsed_ms,
    "files_touched": len(paths),
    "new_files_count": 0,
    "lines_added": 0,
    "lines_deleted": 0,
    "helpers_added": 0,
    "abstractions_added": 0,
    "rework_count": 0,
}

worktree = os.environ.get("ORQUESTA_GOLDEN_TASK_WORKTREE") or str(result_dir / "worktree")
defaults.update(git_metrics(worktree))

for key, value in defaults.items():
    metrics.setdefault(key, value)

tokens = metrics.get("tokens")
if isinstance(tokens, dict):
    metrics.setdefault("input_tokens", metric_int(tokens.get("input")))
    metrics.setdefault("output_tokens", metric_int(tokens.get("output")))
    metrics.setdefault("reasoning_tokens", metric_int(tokens.get("reasoning")))
    metrics.setdefault("cached_input_tokens", metric_int(tokens.get("cached_input")))
    metrics.setdefault("total_tokens", metric_int(tokens.get("total")))

if "total_tokens" not in metrics:
    metrics["total_tokens"] = (
        metric_int(metrics.get("input_tokens"))
        + metric_int(metrics.get("output_tokens"))
        + metric_int(metrics.get("reasoning_tokens"))
    )

if exit_code != 0:
    current_status = str(payload.get("status", "")).strip()
    if current_status in {"", "complete"}:
        payload["status"] = "failed"
    payload["launch_exit_code"] = exit_code

metrics["launcher_exit_code"] = exit_code
metrics["metrics_launcher"] = "orquesta_golden_metrics_launcher.v0"
payload["metrics"] = metrics
result_path.parent.mkdir(parents=True, exist_ok=True)
with result_path.open("w", encoding="utf-8") as handle:
    json.dump(payload, handle, ensure_ascii=False, indent=2, sort_keys=True)
    handle.write("\n")
PY

exit "$exit_code"
