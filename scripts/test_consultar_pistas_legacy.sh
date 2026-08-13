#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
query="${script_dir}/consultar_pistas_legacy.sh"
ledger="${script_dir}/../product/traceability/task_entries.jsonl"

bash -n "$query"

summary="$($query --summary)"
jq -e '
  .schema_version == 1 and
  .authority == "derived_advisory_from_task_entry_ledger" and
  .entries_total == 2108 and
  .unique_entry_refs == 2108 and
  .entries_reviewed == 2108 and
  .semantic_reasons_non_empty == 2108 and
  .capabilities == 184 and
  .matched_entries == 2108 and
  .review_coverage_percent == 100 and
  .semantic_reason_coverage_percent == 100 and
  .dispositions.accepted_pending_reimplementation == 1300 and
  .dispositions.historical_superseded_by_capability == 805 and
  .dispositions.rejected_by_operator == 3 and
  .closure_evidence_states.not_verified == 2108 and
  .depth == "ledger_semantic_lead_not_deep_behavior_assessment" and
  .historical_green == "not_assessed" and
  .function_mapping == "not_assessed" and
  .canonical_state_change == false and
  .claims_historical_success == false and
  .claims_reuse_decision == false and
  (.note | contains("no prueba acierto, fallo, verde histórico, reutilización"))
' <<<"$summary" >/dev/null

first_ref="$(jq -r -s '.[0].entry_ref' "$ledger")"
exact="$($query --entry-ref "$first_ref" --json)"
jq -e --arg first_ref "$first_ref" '
  length == 1 and
  .[0].entry_ref == $first_ref and
  .[0].capability == "ORC-27" and
  (.[0].semantic_reason | length) > 20 and
  .[0].source_ref == "docs/HANDOFF_OPENCLAW_BERSERK_2026-04-02.md" and
  .[0].subject_first_line == 38 and
  .[0].subject_last_line == 38 and
  (.[0].source_sha256 | test("^sha256:[0-9a-f]{64}$")) and
  (.[0].subject_sha256 | test("^sha256:[0-9a-f]{64}$")) and
  .[0].disposition == "historical_superseded_by_capability" and
  .[0].original_state == "legacy_state_unverified_not_closure_evidence" and
  .[0].closure_evidence == "not_verified" and
  (.[0].v2.title | length) > 0 and
  (.[0].v2.status | length) > 0 and
  (.[0].recommended_next_action | length) > 40 and
  .[0].depth == "ledger_semantic_lead_not_deep_behavior_assessment" and
  .[0].historical_green == "not_assessed" and
  .[0].function_mapping == "not_assessed" and
  .[0].claims_historical_success == false and
  .[0].claims_reuse_decision == false
' <<<"$exact" >/dev/null

orc01="$($query --capability ORC-01 --json)"
jq -e '
  length == 20 and all(.[];
    .capability == "ORC-01" and
    .query_total_matches == 73 and
    .query_offset == 0 and .query_limit == 20 and .query_truncated == true)
' <<<"$orc01" >/dev/null

orc01_all="$($query --capability ORC-01 --all --json)"
jq -e '
  length == 73 and all(.[];
    .capability == "ORC-01" and .query_total_matches == 73 and
    .query_truncated == false)
' <<<"$orc01_all" >/dev/null

orc01_page="$($query --capability ORC-01 --offset 20 --limit 10 --json)"
jq -e '
  length == 10 and all(.[];
    .query_total_matches == 73 and .query_offset == 20 and
    .query_limit == 10 and .query_truncated == true)
' <<<"$orc01_page" >/dev/null

text_result="$($query --text 'criterio verificable' --json)"
jq -e '
  length >= 1 and
  any(.[]; .entry_ref == "TASKENTRY-d184bcae775ec14ff044df27")
' <<<"$text_result" >/dev/null

source_result="$($query --source 'HANDOFF_OPENCLAW_BERSERK' --jsonl)"
jq -e -s '
  length > 1 and
  all(.[]; .source_ref == "docs/HANDOFF_OPENCLAW_BERSERK_2026-04-02.md")
' <<<"$source_result" >/dev/null

all_jsonl="$($query --all --jsonl)"
jq -e -s '
  length == 2108 and
  all(.[];
    .depth == "ledger_semantic_lead_not_deep_behavior_assessment" and
    .historical_green == "not_assessed" and
    .function_mapping == "not_assessed" and
    (.v2.title | length) > 0 and
    (.recommended_next_action | length) > 40 and
    (has("semantic_review_verdict") | not) and
    (has("candidate_ref") | not)
  )
' <<<"$all_jsonl" >/dev/null

expect_failure() {
  if "$@" >/dev/null 2>&1; then
    printf 'Se esperaba fallo: %q' "$1" >&2
    shift
    printf ' %q' "$@" >&2
    printf '\n' >&2
    exit 1
  fi
}

expect_failure "$query"
expect_failure "$query" --unknown
expect_failure "$query" --capability
expect_failure "$query" --capability orc-01
expect_failure "$query" --entry-ref TASKENTRY-invalida
expect_failure "$query" --entry-ref TASKENTRY-000000000000000000000000
expect_failure "$query" --all --json --jsonl
expect_failure "$query" --summary --json

fixture_dir="$(mktemp -d)"
valid_fixture="${fixture_dir}/valid.jsonl"
duplicate_fixture="${fixture_dir}/duplicate.jsonl"
bad_shape_fixture="${fixture_dir}/bad-shape.jsonl"
missing_fixture="${fixture_dir}/missing.jsonl"

cleanup() {
  rm -f -- "$valid_fixture" "$duplicate_fixture" "$bad_shape_fixture"
  rmdir -- "$fixture_dir" 2>/dev/null || true
}
trap cleanup EXIT

first_row="$(jq -c -s '.[0]' "$ledger")"
printf '%s\n' "$first_row" >"$valid_fixture"
printf '%s\n%s\n' "$first_row" "$first_row" >"$duplicate_fixture"
jq -c 'del(.semantic_reason)' <<<"$first_row" >"$bad_shape_fixture"

expect_failure env \
  ORQUESTA_LEGACY_TASK_ENTRIES_PATH="$valid_fixture" \
  "$query" --all
fixture_result="$(env \
  ORQUESTA_LEGACY_ALLOW_TEST_FIXTURE=1 \
  ORQUESTA_LEGACY_TASK_ENTRIES_PATH="$valid_fixture" \
  "$query" --all --json)"
jq -e --arg first_ref "$first_ref" '
  length == 1 and .[0].entry_ref == $first_ref
' <<<"$fixture_result" >/dev/null
expect_failure env \
  ORQUESTA_LEGACY_ALLOW_TEST_FIXTURE=1 \
  ORQUESTA_LEGACY_TASK_ENTRIES_PATH="$duplicate_fixture" \
  "$query" --all
expect_failure env \
  ORQUESTA_LEGACY_ALLOW_TEST_FIXTURE=1 \
  ORQUESTA_LEGACY_TASK_ENTRIES_PATH="$bad_shape_fixture" \
  "$query" --all
expect_failure env \
  ORQUESTA_LEGACY_ALLOW_TEST_FIXTURE=1 \
  ORQUESTA_LEGACY_TASK_ENTRIES_PATH="$missing_fixture" \
  "$query" --all

printf '%s\n' \
  'status=passed entries=2108 reviewed=2108 semantic_reasons=2108 capabilities=184 historical_green=not_assessed function_mapping=not_assessed'
