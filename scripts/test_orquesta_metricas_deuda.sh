#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
script="$ROOT/scripts/orquesta_metricas_deuda.sh"

workdir="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-metricas-deuda-test.XXXXXX")"
trap 'rm -rf "$workdir"' EXIT

assert_positive_key_value_output() {
  local output="$1"
  OUTPUT="$output" python3 - <<'PY'
import os
import re

expected_keys = [
    "env_vars_orquesta",
    "env_vars_orquesta_test_only",
    "endpoints_status",
    "interfaces_estado",
    "modulos_director",
]

lines = os.environ["OUTPUT"].splitlines()
if len(lines) != len(expected_keys):
    raise SystemExit(f"expected {len(expected_keys)} lines, got {len(lines)}: {lines!r}")

values = {}
for expected_key, line in zip(expected_keys, lines):
    if "=" not in line:
        raise SystemExit(f"missing key=value separator in {line!r}")
    key, value = line.split("=", 1)
    if key != expected_key:
        raise SystemExit(f"expected key {expected_key!r}, got {key!r}")
    if not re.fullmatch(r"[0-9]+", value) or (key != "env_vars_orquesta_test_only" and value == "0"):
        raise SystemExit(f"expected valid integer for {key}, got {value!r}")
    values[key] = int(value)
PY
}

assert_json_matches_key_value_output() {
  local plain="$1"
  local json="$2"
  PLAIN_OUTPUT="$plain" JSON_OUTPUT="$json" python3 - <<'PY'
import json
import os

expected_keys = [
    "env_vars_orquesta",
    "env_vars_orquesta_test_only",
    "endpoints_status",
    "interfaces_estado",
    "modulos_director",
]

plain_values = {}
for line in os.environ["PLAIN_OUTPUT"].splitlines():
    key, value = line.split("=", 1)
    plain_values[key] = int(value)

payload = json.loads(os.environ["JSON_OUTPUT"])
if list(payload.keys()) != expected_keys:
    raise SystemExit(f"unexpected JSON keys/order: {list(payload.keys())!r}")
if payload != plain_values:
    raise SystemExit(f"JSON values do not match key=value output: {payload!r} != {plain_values!r}")
PY
}

repo_output="$(cd "$ROOT" && "$script")"
assert_positive_key_value_output "$repo_output"

repo_json_output="$(cd "$ROOT" && "$script" --json)"
assert_json_matches_key_value_output "$repo_output" "$repo_json_output"

# Use a synthetic unavailable locale so this case is portable without es_ES.
# The shims simulate locale-sensitive sort/comm and reject calls unless the
# metric script has overridden the inherited locale with LC_ALL=C.
locale_shims="$workdir/locale-shims"
mkdir -p "$locale_shims"
system_sort="$(command -v sort)"
system_comm="$(command -v comm)"
printf '#!/usr/bin/env bash\n[ "${LC_ALL:-}" = C ] || { echo "sort requires LC_ALL=C" >&2; exit 1; }\nexec %q "$@"\n' \
  "$system_sort" >"$locale_shims/sort"
printf '#!/usr/bin/env bash\n[ "${LC_ALL:-}" = C ] || { echo "comm requires LC_ALL=C" >&2; exit 1; }\nexec %q "$@"\n' \
  "$system_comm" >"$locale_shims/comm"
chmod +x "$locale_shims/sort" "$locale_shims/comm"

locale_simulated_output="$(cd "$ROOT" && LC_ALL=orquesta_test_non_c PATH="$locale_shims:$PATH" "$script" --json 2>"$workdir/locale-warning")"
assert_json_matches_key_value_output "$repo_output" "$locale_simulated_output"

fixture="$workdir/repo"
mkdir -p \
  "$fixture/cmd/app" \
  "$fixture/modulos/orquesta-director-one" \
  "$fixture/modulos/director-two" \
  "$fixture/modulos/plain/nested-director" \
  "$fixture/modulos/pkg"

cat >"$fixture/cmd/app/main.go" <<'GO'
package main

const alpha = "ORQUESTA_ALPHA"
const alphaAgain = "ORQUESTA_ALPHA"
const beta = "ORQUESTA_BETA_2"
const routeStatus = "/api/v0/server/status"
const routeStats = "/api/v0/server/stats"
const routeReadiness = "/api/v0/server/readiness"
const routeIgnored = "/api/v0/server/tasks"

type FooStore interface{}
type Plain interface{}
type queueState interface{}
GO

cat >"$fixture/modulos/pkg/more.go" <<'GO'
package pkg

const gamma = "ORQUESTA_GAMMA_EXTRA"
const lowerIgnored = "orquesta_lower_ignored"
const routeObserve = "/api/v0/observe/events"
const routeControl = "/api/v0/server/control"
const routeHealth = "/api/v0/health"
const routeDuplicate = "/api/v0/server/status"

type SnapshotRegistry interface {
}
type MarkerQueue interface{}
type ledger interface{}
GO

expected_fixture_output=$'env_vars_orquesta=3\nenv_vars_orquesta_test_only=0\nendpoints_status=6\ninterfaces_estado=5\nmodulos_director=2'
fixture_output="$(cd "$fixture" && "$script")"
if [ "$fixture_output" != "$expected_fixture_output" ]; then
  echo "unexpected fixture metrics" >&2
  printf 'expected:\n%s\nactual:\n%s\n' "$expected_fixture_output" "$fixture_output" >&2
  exit 1
fi

fixture_json_output="$(cd "$fixture" && "$script" --json)"
assert_json_matches_key_value_output "$fixture_output" "$fixture_json_output"

before_files="$(cd "$fixture" && find . -type f -printf '%P\n' | sort)"
(cd "$fixture" && "$script" >/dev/null)
after_files="$(cd "$fixture" && find . -type f -printf '%P\n' | sort)"
if [ "$before_files" != "$after_files" ]; then
  echo "metric script modified fixture files" >&2
  exit 1
fi

invalid_root="$workdir/not-a-repo"
mkdir -p "$invalid_root"
set +e
invalid_output="$(cd "$invalid_root" && "$script" 2>&1 >/dev/null)"
invalid_status=$?
set -e
if [ "$invalid_status" -ne 2 ]; then
  echo "expected invalid root exit 2, got $invalid_status" >&2
  exit 1
fi
grep -qi 'repo root containing modulos/ and cmd/' <<<"$invalid_output" || {
  echo "invalid root message was not clear: $invalid_output" >&2
  exit 1
}

echo "orquesta_metricas_deuda_ok=true"
