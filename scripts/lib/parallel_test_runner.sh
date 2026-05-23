#!/usr/bin/env bash

orquesta_parallel_test_init() {
  ORQUESTA_PARALLEL_TEST_TMP="${ORQUESTA_PARALLEL_TEST_TMP:-$(mktemp -d)}"
  ORQUESTA_PARALLEL_TEST_PIDS=()
  ORQUESTA_PARALLEL_TEST_NAMES=()
  ORQUESTA_PARALLEL_TEST_LOGS=()
}

orquesta_parallel_test_cleanup() {
  if [[ -n "${ORQUESTA_PARALLEL_TEST_TMP:-}" && -d "$ORQUESTA_PARALLEL_TEST_TMP" ]]; then
    rm -rf "$ORQUESTA_PARALLEL_TEST_TMP"
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
