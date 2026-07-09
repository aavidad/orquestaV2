#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
uso: scripts/orquesta_golden_ab_launcher.sh --baseline-command CMD --variant-command CMD [--primary-arm baseline|variant] [--variant-skill-ref REF]

Launcher A/B opt-in para golden tasks. Consume ORQUESTA_GOLDEN_TASK_* del harness
existente y escribe un result.json unico con metricas comparables por brazo.
USAGE
}

baseline_command=""
variant_command=""
primary_arm="variant"
variant_skill_refs=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --baseline-command)
      baseline_command="${2:-}"
      shift 2
      ;;
    --variant-command)
      variant_command="${2:-}"
      shift 2
      ;;
    --primary-arm)
      primary_arm="${2:-}"
      shift 2
      ;;
    --variant-skill-ref)
      variant_skill_refs+=("${2:-}")
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "argumento no soportado: $1" >&2
      usage
      exit 2
      ;;
  esac
done

if [[ -z "$baseline_command" || -z "$variant_command" ]]; then
  usage
  exit 2
fi

case "$primary_arm" in
  baseline|variant) ;;
  *)
    echo "--primary-arm debe ser baseline o variant" >&2
    exit 2
    ;;
esac

task_id="${ORQUESTA_GOLDEN_TASK_ID:?ORQUESTA_GOLDEN_TASK_ID requerido}"
result_dir="${ORQUESTA_GOLDEN_TASK_RESULT_DIR:?ORQUESTA_GOLDEN_TASK_RESULT_DIR requerido}"
result_file="$result_dir/result.json"
arms_dir="$result_dir/arms"
mkdir -p "$arms_dir"

now_ms() {
  python3 - <<'PY'
import time
print(int(time.time() * 1000))
PY
}

run_arm() {
  local arm="$1"
  local command="$2"
  local arm_dir="$arms_dir/$arm"
  mkdir -p "$arm_dir"
  local start_ms
  local end_ms
  start_ms="$(now_ms)"
  set +e
  if [[ "$arm" == "variant" && "${#variant_skill_refs[@]}" -gt 0 ]]; then
    GOLDEN_AB_ARM="$arm" \
    GOLDEN_AB_PRIMARY_ARM="$primary_arm" \
    GOLDEN_AGENT_SKILL_REFS="$(IFS=,; echo "${variant_skill_refs[*]}")" \
    ORQUESTA_GOLDEN_TASK_RESULT_DIR="$arm_dir" \
    bash -lc "$command"
  else
    GOLDEN_AB_ARM="$arm" \
    GOLDEN_AB_PRIMARY_ARM="$primary_arm" \
    ORQUESTA_GOLDEN_TASK_RESULT_DIR="$arm_dir" \
    bash -lc "$command"
  fi
  local exit_code="$?"
  set -e
  end_ms="$(now_ms)"
  printf '%s\n' "$exit_code" > "$arm_dir/.exit_code"
  printf '%s\n' "$((end_ms - start_ms))" > "$arm_dir/.elapsed_ms"
}

run_arm "baseline" "$baseline_command"
run_arm "variant" "$variant_command"

python3 - "$result_file" "$task_id" "$primary_arm" "${variant_skill_refs[*]-}" <<'PY'
import json
from pathlib import Path
import sys


result_path = Path(sys.argv[1])
task_id = sys.argv[2]
primary_arm = sys.argv[3]
variant_skill_refs = [value for value in sys.argv[4].split() if value]
arms_dir = result_path.parent / "arms"


def read_int(path):
    try:
        return int(Path(path).read_text(encoding="utf-8").strip())
    except (OSError, ValueError):
        return 0


def read_payload(path, task_id, exit_code):
    path = Path(path)
    if not path.is_file():
        return {
            "task_id": task_id,
            "status": "failed" if exit_code else "complete",
            "touched_files": [],
            "artifact_paths": [],
            "tests": [],
            "evidence": {},
            "metrics": {},
        }
    try:
        loaded = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        loaded = {}
    if not isinstance(loaded, dict):
        loaded = {}
    loaded.setdefault("task_id", task_id)
    loaded.setdefault("status", "failed" if exit_code else "complete")
    loaded.setdefault("touched_files", [])
    loaded.setdefault("artifact_paths", [])
    loaded.setdefault("tests", [])
    loaded.setdefault("evidence", {})
    loaded.setdefault("metrics", {})
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


def normalized_metrics(payload, elapsed_ms, exit_code):
    raw = payload.get("metrics")
    metrics = dict(raw) if isinstance(raw, dict) else {}
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
    metrics.setdefault("elapsed_ms", elapsed_ms)
    metrics.setdefault("files_touched", len(payload.get("touched_files") or []))
    metrics.setdefault("new_files_count", 0)
    metrics.setdefault("lines_added", 0)
    metrics.setdefault("lines_deleted", 0)
    metrics.setdefault("helpers_added", 0)
    metrics.setdefault("abstractions_added", 0)
    metrics.setdefault("rework_count", 0)
    metrics["launcher_exit_code"] = exit_code
    return metrics


def arm_payload(arm):
    arm_dir = arms_dir / arm
    exit_code = read_int(arm_dir / ".exit_code")
    elapsed_ms = read_int(arm_dir / ".elapsed_ms")
    payload = read_payload(arm_dir / "result.json", task_id, exit_code)
    metrics = normalized_metrics(payload, elapsed_ms, exit_code)
    return {
        "arm": arm,
        "result_path": str(arm_dir / "result.json"),
        "status": payload.get("status"),
        "exit_code": exit_code,
        "metrics": metrics,
        "touched_files": payload.get("touched_files") or [],
        "artifact_paths": payload.get("artifact_paths") or [],
        "tests": payload.get("tests") or [],
        "evidence": payload.get("evidence") if isinstance(payload.get("evidence"), dict) else {},
        "payload": payload,
    }


arms = {arm: arm_payload(arm) for arm in ("baseline", "variant")}
primary = arms[primary_arm]
failed_arms = [
    arm for arm, payload in arms.items()
    if payload["exit_code"] != 0 or payload.get("status") != "complete"
]

baseline_metrics = arms["baseline"]["metrics"]
variant_metrics = arms["variant"]["metrics"]
metrics = dict(primary["metrics"])
metrics.update({
    "ab_launcher": "orquesta_golden_ab_launcher.v0",
    "ab_primary_arm": primary_arm,
    "ab_failed_arms_count": len(failed_arms),
    "baseline_launcher_exit_code": arms["baseline"]["exit_code"],
    "variant_launcher_exit_code": arms["variant"]["exit_code"],
    "baseline_total_tokens": metric_int(baseline_metrics.get("total_tokens")),
    "variant_total_tokens": metric_int(variant_metrics.get("total_tokens")),
    "token_delta": metric_int(variant_metrics.get("total_tokens")) - metric_int(baseline_metrics.get("total_tokens")),
    "baseline_elapsed_ms": metric_int(baseline_metrics.get("elapsed_ms")),
    "variant_elapsed_ms": metric_int(variant_metrics.get("elapsed_ms")),
    "elapsed_ms_delta": metric_int(variant_metrics.get("elapsed_ms")) - metric_int(baseline_metrics.get("elapsed_ms")),
    "baseline_files_touched": metric_int(baseline_metrics.get("files_touched")),
    "variant_files_touched": metric_int(variant_metrics.get("files_touched")),
    "files_touched_delta": metric_int(variant_metrics.get("files_touched")) - metric_int(baseline_metrics.get("files_touched")),
    "baseline_new_files_count": metric_int(baseline_metrics.get("new_files_count")),
    "variant_new_files_count": metric_int(variant_metrics.get("new_files_count")),
    "new_files_delta": metric_int(variant_metrics.get("new_files_count")) - metric_int(baseline_metrics.get("new_files_count")),
    "baseline_lines_added": metric_int(baseline_metrics.get("lines_added")),
    "variant_lines_added": metric_int(variant_metrics.get("lines_added")),
    "lines_added_delta": metric_int(variant_metrics.get("lines_added")) - metric_int(baseline_metrics.get("lines_added")),
    "baseline_rework_count": metric_int(baseline_metrics.get("rework_count")),
    "variant_rework_count": metric_int(variant_metrics.get("rework_count")),
    "rework_delta": metric_int(variant_metrics.get("rework_count")) - metric_int(baseline_metrics.get("rework_count")),
})

evidence = dict(primary["evidence"])
evidence["ab_comparison"] = {
    "baseline_status": arms["baseline"]["status"],
    "variant_status": arms["variant"]["status"],
    "failed_arms": failed_arms,
    "variant_skill_refs": variant_skill_refs,
    "token_delta": metrics["token_delta"],
    "files_touched_delta": metrics["files_touched_delta"],
    "lines_added_delta": metrics["lines_added_delta"],
    "rework_delta": metrics["rework_delta"],
}

result = {
    "schema_version": "orquesta_golden_ab_result.v0",
    "task_id": task_id,
    "status": "complete" if not failed_arms else "failed",
    "primary_arm": primary_arm,
    "arms": {
        "baseline": {key: value for key, value in arms["baseline"].items() if key != "payload"},
        "variant": {key: value for key, value in arms["variant"].items() if key != "payload"},
    },
    "touched_files": primary["touched_files"],
    "artifact_paths": primary["artifact_paths"],
    "tests": primary["tests"],
    "evidence": evidence,
    "metrics": metrics,
}
if failed_arms:
    result["launch_exit_code"] = max(arms[arm]["exit_code"] for arm in failed_arms)

result_path.parent.mkdir(parents=True, exist_ok=True)
result_path.write_text(json.dumps(result, ensure_ascii=False, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY
