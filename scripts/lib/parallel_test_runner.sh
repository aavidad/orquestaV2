#!/usr/bin/env bash

parallel_test_lib_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib/smoke_common.sh
source "$parallel_test_lib_dir/smoke_common.sh"

orquesta_parallel_test_init() {
  local tmp_source="generated"
  if [[ -n "${ORQUESTA_PARALLEL_TEST_TMP:-}" ]]; then
    tmp_source="env:ORQUESTA_PARALLEL_TEST_TMP"
  else
    ORQUESTA_PARALLEL_TEST_TMP="$(mktemp -d)"
  fi
  smoke_temp_root_prepare "$ORQUESTA_PARALLEL_TEST_TMP" "$tmp_source"
  ORQUESTA_PARALLEL_TEST_PIDS=()
  ORQUESTA_PARALLEL_TEST_NAMES=()
  ORQUESTA_PARALLEL_TEST_LOGS=()
}

orquesta_parallel_test_cleanup() {
  if [[ -n "${ORQUESTA_PARALLEL_TEST_TMP:-}" && -d "$ORQUESTA_PARALLEL_TEST_TMP" ]]; then
    smoke_temp_root_cleanup "$ORQUESTA_PARALLEL_TEST_TMP" "${ORQUESTA_KEEP_PARALLEL_TEST_TMP:-${ORQUESTA_KEEP_SMOKE_DIR:-0}}"
  fi
}

orquesta_parallel_test_start() {
  local name="$1"
  shift
  local log="$ORQUESTA_PARALLEL_TEST_TMP/$name.log"
  echo "start $name"
  (
    "$@"
  ) >"$log" 2>&1 &
  ORQUESTA_PARALLEL_TEST_PIDS+=("$!")
  ORQUESTA_PARALLEL_TEST_NAMES+=("$name")
  ORQUESTA_PARALLEL_TEST_LOGS+=("$log")
}

orquesta_parallel_test_wait() {
  local failed=0
  local index
  for index in "${!ORQUESTA_PARALLEL_TEST_PIDS[@]}"; do
    local pid="${ORQUESTA_PARALLEL_TEST_PIDS[$index]}"
    local name="${ORQUESTA_PARALLEL_TEST_NAMES[$index]}"
    local log="${ORQUESTA_PARALLEL_TEST_LOGS[$index]}"
    if wait "$pid"; then
      echo "ok $name"
      continue
    fi
    failed=1
    echo "fail $name" >&2
    sed -n '1,240p' "$log" >&2 || true
  done
  return "$failed"
}
