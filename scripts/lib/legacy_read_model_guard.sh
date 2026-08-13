#!/usr/bin/env bash

# Helpers fail-closed para índices advisory. El caller debe usar set -euo pipefail.
legacy_verify_file_manifest() {
  local data_path="$1"
  local manifest_path="$2"
  local expected_kind="$3"
  local expected_sha expected_bytes actual_sha actual_bytes

  for required in "$data_path" "$manifest_path"; do
    if [[ ! -r "$required" || ! -f "$required" ]]; then
      printf 'Read-model o manifiesto no legible: %s\n' "$required" >&2
      return 2
    fi
  done
  if ! jq -e --arg kind "$expected_kind" '
    .schema_version == 1 and .document_kind == $kind and
    (.jsonl_sha256 | test("^sha256:[0-9a-f]{64}$")) and
    (.jsonl_bytes | type) == "number" and .jsonl_bytes > 0
  ' "$manifest_path" >/dev/null; then
    printf 'Manifiesto inválido: %s\n' "$manifest_path" >&2
    return 2
  fi
  expected_sha="$(jq -er '.jsonl_sha256 | sub("^sha256:"; "")' "$manifest_path")"
  expected_bytes="$(jq -er '.jsonl_bytes' "$manifest_path")"
  actual_sha="$(sha256sum "$data_path" | cut -d ' ' -f 1)"
  actual_bytes="$(stat -c '%s' "$data_path")"
  if [[ "$actual_sha" != "$expected_sha" || "$actual_bytes" != "$expected_bytes" ]]; then
    printf 'Read-model no coincide con su manifiesto: %s\n' "$data_path" >&2
    return 2
  fi
}

legacy_validate_reuse_assessments() {
  local data_path="$1"
  local manifest_path="$2"

  legacy_verify_file_manifest \
    "$data_path" "$manifest_path" "legacy_reuse_assessments_manifest" || return
  if ! jq -e -s --slurpfile manifest "$manifest_path" '
    length == $manifest[0].assessment_count and
    (map(.characterization_ref) | unique | length) ==
      $manifest[0].unique_characterization_ref_count and
    ([.[].function_mapping.links[]] | length) ==
      $manifest[0].function_behavior_link_count and
    ([.[].function_mapping.links[].occurrence_ref] | unique | length) ==
      $manifest[0].unique_function_occurrence_count and
    all(.[];
      .schema_version == 1 and
      (.characterization_ref | length) > 0 and
      (.green_rationale | length) > 30 and
      (.agent_recommendation | length) > 30 and
      .review_state == "independent_bootstrap_counterreview_completed_advisory" and
      .authority == "advisory_not_capability_state" and
      (.function_mapping.state == "linked_exact" or
        .function_mapping.state == "reviewed_no_direct_function") and
      .function_mapping.snapshot_census_sha256 ==
        $manifest[0].snapshot_census_sha256 and
      (if .function_mapping.state == "linked_exact" then
        (.function_mapping.links | length) > 0
      else (.function_mapping.links | length) == 0 end) and
      all(.function_mapping.links[];
        (.occurrence_ref | test("^sha256:[0-9a-f]{64}$")) and
        (.source_path | length) > 0 and
        (.signature | startswith("func ")) and
        (.semantic_rationale | length) > 30 and
        .review_state == "bootstrap_symbol_mapping_reviewed_advisory" and
        (has("body") | not) and (has("canonical_source") | not)))
  ' "$data_path" >/dev/null; then
    printf 'Assessments inválidos, truncados, duplicados o incoherentes.\n' >&2
    return 2
  fi
}
