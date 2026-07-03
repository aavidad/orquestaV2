#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
confirm_value="${ORQUESTA_MUTATION_PILOT_CONFIRM_VALUE:-MUTATION_PILOT_OPT_IN}"

usage() {
  cat >&2 <<'USAGE'
Uso:
  ORQUESTA_MUTATION_PILOT_CONFIRM=MUTATION_PILOT_OPT_IN scripts/orquesta_mutation_pilot.sh

Este piloto es opt-in y no acepta argumentos. El alcance esta fijado a:
  ./modulos/orquesta-estado-vivo
  ./modulos/orquesta-goal
USAGE
}

if [[ "$#" -ne 0 ]]; then
  usage
  exit 2
fi

if [[ "${ORQUESTA_MUTATION_PILOT_CONFIRM:-}" != "$confirm_value" ]]; then
  echo "falta confirmacion opt-in: exporta ORQUESTA_MUTATION_PILOT_CONFIRM=$confirm_value" >&2
  exit 2
fi

if [[ -z "${ORQUESTA_NIGHTLY_RESULTS_DIR:-}" && -z "${HOME:-}" ]]; then
  echo "ORQUESTA_NIGHTLY_RESULTS_DIR o HOME son necesarios para escribir resultado JSON" >&2
  exit 2
fi

target_packages=(
  "./modulos/orquesta-estado-vivo"
  "./modulos/orquesta-goal"
)

for package in "${target_packages[@]}"; do
  if [[ ! -d "$repo_root/${package#./}" ]]; then
    echo "paquete objetivo no existe: $package" >&2
    exit 2
  fi
done

tool_candidate="${ORQUESTA_MUTATION_PILOT_TOOL:-go-mutesting}"
if [[ "$tool_candidate" == */* ]]; then
  if [[ "$tool_candidate" == /* ]]; then
    tool_path="$tool_candidate"
  else
    tool_path="$(cd "$(dirname "$tool_candidate")" && pwd)/$(basename "$tool_candidate")"
  fi
  if [[ ! -x "$tool_path" ]]; then
    echo "herramienta de mutation testing no ejecutable: $tool_candidate" >&2
    exit 2
  fi
else
  tool_path="$(command -v "$tool_candidate" || true)"
  if [[ -z "$tool_path" ]]; then
    echo "herramienta de mutation testing no encontrada: $tool_candidate" >&2
    echo "instala go-mutesting o define ORQUESTA_MUTATION_PILOT_TOOL=/ruta/herramienta" >&2
    exit 2
  fi
fi

results_root="${ORQUESTA_NIGHTLY_RESULTS_DIR:-$HOME/.orquesta-nightly}"
results_dir="${ORQUESTA_MUTATION_PILOT_RESULTS_DIR:-$results_root/mutation-testing-pilot}"
logs_dir="$results_dir/logs"
date_id="${ORQUESTA_MUTATION_PILOT_DATE_ID:-${ORQUESTA_NIGHTLY_DATE_ID:-$(date -u +%Y%m%d)}}"
started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
started_epoch="$(date -u +%s)"
run_id="mutation-pilot-$date_id-$(date -u +%H%M%S)"
result_file="$results_dir/resultado_mutation_${date_id}.json"
log_file="$logs_dir/${run_id}.log"
acceptable_score="${ORQUESTA_MUTATION_PILOT_ACCEPTABLE_SCORE:-0.70}"
top_survivors="${ORQUESTA_MUTATION_PILOT_TOP_SURVIVORS:-5}"

if [[ ! "$top_survivors" =~ ^[0-9]+$ || "$top_survivors" -lt 1 ]]; then
  echo "ORQUESTA_MUTATION_PILOT_TOP_SURVIVORS debe ser entero positivo" >&2
  exit 2
fi

mkdir -p "$results_dir" "$logs_dir"

{
  printf 'run_id=%s\n' "$run_id"
  printf 'started_at_utc=%s\n' "$started_at"
  printf 'tool=%s\n' "$tool_candidate"
  printf 'target_packages=%s\n' "${target_packages[*]}"
} >"$log_file"

package_result_args=()
for package in "${target_packages[@]}"; do
  safe_package="${package#./}"
  safe_package="${safe_package//\//_}"
  package_log="$logs_dir/${run_id}_${safe_package}.log"

  {
    printf '\n== package %s ==\n' "$package"
    printf 'package_log=%s\n' "$package_log"
  } >>"$log_file"

  set +e
  (
    cd "$repo_root"
    "$tool_path" "$package"
  ) >"$package_log" 2>&1
  package_exit_code="$?"
  set -e

  {
    printf 'package=%s\n' "$package"
    printf 'tool_exit_code=%s\n' "$package_exit_code"
  } >>"$log_file"

  package_result_args+=("$package" "$package_log" "$package_exit_code")
done

finished_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
finished_epoch="$(date -u +%s)"
duration_seconds="$((finished_epoch - started_epoch))"

set +e
python3 - \
  "$result_file" \
  "$date_id" \
  "$run_id" \
  "$started_at" \
  "$finished_at" \
  "$duration_seconds" \
  "$acceptable_score" \
  "$top_survivors" \
  "$tool_candidate" \
  "$tool_path" \
  "$log_file" \
  "${package_result_args[@]}" <<'PY'
import json
import os
import re
import sys

(
    result_file,
    date_id,
    run_id,
    started_at,
    finished_at,
    duration_seconds,
    acceptable_score,
    top_survivors,
    tool_candidate,
    tool_path,
    log_file,
    *package_args,
) = sys.argv[1:]

try:
    threshold = float(acceptable_score)
except ValueError as exc:
    raise SystemExit(f"ORQUESTA_MUTATION_PILOT_ACCEPTABLE_SCORE invalido: {acceptable_score}") from exc

if threshold < 0 or threshold > 1:
    raise SystemExit("ORQUESTA_MUTATION_PILOT_ACCEPTABLE_SCORE debe estar entre 0 y 1")

top_limit = int(top_survivors)

if len(package_args) % 3 != 0:
    raise SystemExit("argumentos internos invalidos para paquetes")

score_patterns = (
    re.compile(r"\bmutation[_ -]?score\b\s*(?:is|=|:)?\s*([0-9]+(?:\.[0-9]+)?)\s*(%)?", re.I),
    re.compile(r"\bscore\b\s*(?:is|=|:)\s*([0-9]+(?:\.[0-9]+)?)\s*(%)?", re.I),
)

counter_patterns = {
    "mutations_total": (
        re.compile(r"\bmutations?_total\b\s*[:=]\s*(\d+)", re.I),
        re.compile(r"\btotal_mutants\b\s*[:=]\s*(\d+)", re.I),
        re.compile(r"\btotal\s+(?:is\s+)?(\d+)\b", re.I),
    ),
    "mutants_killed": (
        re.compile(r"\bmutants?_killed\b\s*[:=]\s*(\d+)", re.I),
        re.compile(r"\bkilled\b\s*[:=]\s*(\d+)", re.I),
    ),
    "mutants_survived": (
        re.compile(r"\bmutants?_survived\b\s*[:=]\s*(\d+)", re.I),
        re.compile(r"\bsurvivors?\b\s*[:=]\s*(\d+)", re.I),
        re.compile(r"\bsurvived\b\s*[:=]\s*(\d+)", re.I),
    ),
}

survivor_line_pattern = re.compile(
    r"(^|\b)(SURVIVED:|surviving mutant|survivor:|not killed\b)",
    re.I,
)
survivor_summary_pattern = re.compile(
    r"\b(mutants?_survived|survivors?|survived)\b\s*[:=]\s*\d+",
    re.I,
)


def clamp_score(value: float) -> float:
    if value > 1:
        value = value / 100
    return max(0.0, min(1.0, value))


def first_counter(text: str, key: str):
    for pattern in counter_patterns[key]:
        match = pattern.search(text)
        if match:
            return int(match.group(1))
    return None


def parse_package_log(text: str):
    score = None
    for pattern in score_patterns:
        match = pattern.search(text)
        if match:
            raw = float(match.group(1))
            if match.group(2) == "%":
                raw = raw / 100
            score = clamp_score(raw)
            break

    total = first_counter(text, "mutations_total")
    killed = first_counter(text, "mutants_killed")
    survived = first_counter(text, "mutants_survived")

    if killed is None and total is not None and survived is not None:
        killed = max(total - survived, 0)
    if survived is None and total is not None and killed is not None:
        survived = max(total - killed, 0)
    if score is None and total:
        if killed is not None:
            score = killed / total
        elif survived is not None:
            score = (total - survived) / total

    survivors = []
    for raw_line in text.splitlines():
        line = raw_line.strip()
        if not line:
            continue
        if survivor_line_pattern.search(line) and not survivor_summary_pattern.search(line):
            survivors.append(line[:500])

    return score, total, killed, survived, survivors[:top_limit]


packages = []
global_survivors = []
all_scored = True
below_threshold = []

for index in range(0, len(package_args), 3):
    package, package_log, exit_code_raw = package_args[index : index + 3]
    exit_code = int(exit_code_raw)
    try:
        with open(package_log, encoding="utf-8", errors="replace") as fh:
            text = fh.read()
    except OSError:
        text = ""

    score, total, killed, survived, survivors = parse_package_log(text)
    scored = score is not None
    all_scored = all_scored and scored
    if scored and score < threshold:
        below_threshold.append(package)

    for survivor in survivors:
        global_survivors.append({"package": package, "mutant": survivor})

    packages.append(
        {
            "package": package,
            "command": [tool_candidate, package],
            "tool_exit_code": exit_code,
            "status": "scored" if scored else "unscored",
            "score": round(score, 6) if scored else None,
            "score_percent": round(score * 100, 3) if scored else None,
            "mutations_total": total,
            "mutants_killed": killed,
            "mutants_survived": survived,
            "surviving_mutants_top": survivors,
            "log_file": package_log,
        }
    )

payload = {
    "schema_version": "orquesta_mutation_pilot.v0",
    "result_ref": f"orquesta-mutation-pilot-{date_id}",
    "run_id": run_id,
    "date_utc": date_id,
    "started_at_utc": started_at,
    "finished_at_utc": finished_at,
    "duration_seconds": int(duration_seconds),
    "status": "valid_pilot_result" if all_scored else "partial_pilot_result",
    "policy": {
        "opt_in_env": "ORQUESTA_MUTATION_PILOT_CONFIRM",
        "opt_in_value": "MUTATION_PILOT_OPT_IN",
        "ci_default": False,
        "go_mod_dependency_required": False,
        "target_packages_locked": True,
    },
    "tool": {
        "requested": tool_candidate,
        "resolved_path": tool_path,
    },
    "threshold": {
        "acceptable_score": threshold,
        "below_threshold_packages": below_threshold,
    },
    "target_packages": [package["package"] for package in packages],
    "packages": packages,
    "surviving_mutants_top": global_survivors[:top_limit],
    "log_file": log_file,
    "notes": [
        "manual/nightly extended opt-in only",
        "only the two configured target packages are executed",
        "surviving_mutants_top entries are candidates for new tests",
    ],
}

tmp = result_file + ".tmp"
with open(tmp, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, indent=2, sort_keys=True)
    fh.write("\n")
os.replace(tmp, result_file)

if not all_scored:
    raise SystemExit(1)
PY
json_status="$?"
set -e

echo "mutation_pilot_result=$result_file"
echo "mutation_pilot_log=$log_file"

exit "$json_status"
