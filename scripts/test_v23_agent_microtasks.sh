#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "${script_dir}/.." && pwd)"
manifest="${repo_root}/docs/reconstruccion/worksets/v23_agent_microtasks_v1.json"
emitter="${repo_root}/scripts/emitir_plan_v23_microtareas.sh"

fail() {
  printf 'status=failed check=%s\n' "$1" >&2
  exit 1
}

jq -e . "${manifest}" >/dev/null || fail json

jq -e '
  .schema_version == 1 and
  .document_kind == "orquesta_agent_microtask_plan" and
  .vertical == "V23" and
  .contract == "AC-V23-WIZARD" and
  (.phases | length) == 4 and
  (.tasks | length) == 13 and
  ([.tasks[].order] | sort) == [range(1; 14)] and
  ([.tasks[].work_item_key] | unique | length) == 13
' "${manifest}" >/dev/null || fail identity_and_counts

jq -e '
  all(.tasks[];
    .status == "planned" and
    (.title | length) > 0 and
    (.objective | length) > 0 and
    (.target.file | length) > 0 and
    (.target.symbol | length) > 0 and
    (.preconditions | length) > 0 and
    (.postconditions | length) > 0 and
    (.read_set | length) > 0 and
    (.write_set | length) > 0 and
    (.required_tests | length) > 0 and
    all(.required_tests[];
      (.ref | startswith("required-test:v23-")) and
      .tool_ref == "tool:go" and
      (.arguments | length) > 0 and
      (.working_directory | length) > 0
    ) and
    (.negative_cases | length) > 0 and
    (.output_artifacts | length) > 0 and
    .max_files_changed >= 1 and
    .max_files_changed <= 5 and
    (.write_set | length) <= .max_files_changed and
    .max_loc >= 1 and .max_loc <= 300 and
    .max_minutes >= 1 and .max_minutes <= 90 and
    (.preflight.capability | length) > 0 and
    (.preflight.path | length) > 0 and
    (.preflight.operation | length) > 0
  )
' "${manifest}" >/dev/null || fail task_contracts

jq -e '
  .tasks as $tasks |
  all($tasks[];
    . as $task |
    all($task.dependencies[];
      . as $dependency |
      any($tasks[];
        .work_item_key == $dependency and .order < $task.order
      )
    )
  )
' "${manifest}" >/dev/null || fail dag_dependencies

jq -e '
  (
    [.current_baseline.deferred_scopes[]] | sort
  ) == (
    [.tasks[].closes_scopes[]] | unique | sort
  ) and
  (
    [.current_baseline.deferred_scopes[]] | sort
  ) == [
    "dossier_generation",
    "roadmap_promotion",
    "seal_and_receipt",
    "web_surface",
    "wizard_gap_exact_evaluation_snapshot_and_replay",
    "wizard_help_surface"
  ]
' "${manifest}" >/dev/null || fail deferred_scope_coverage

jq -e '
  (.scope_transfers | length) == 3 and
  (
    [.scope_transfers[] | [.capability_id, .from, .to]] | sort
  ) == (
    [
      ["UI-05", "V23", "V24"],
      ["WIZ-10", "V23", "V28"],
      ["WIZ-13", "V23", "V24"]
    ] | sort
  )
' "${manifest}" >/dev/null || fail scope_transfers

jq -e '
  [.tasks[] | select(.parallel_group != "")] as $parallel |
  all(range(0; ($parallel | length));
    . as $left |
    all(range($left + 1; ($parallel | length));
      . as $right |
      if $parallel[$left].parallel_group == $parallel[$right].parallel_group
      then
        ([
          $parallel[$left].write_set[] as $path |
          select($parallel[$right].write_set | index($path))
        ] | length) == 0
      else true
      end
    )
  )
' "${manifest}" >/dev/null || fail parallel_write_set_overlap

"${emitter}" |
  jq -e '
    (.phases | length) == 4 and
    (.work_items | length) == 13 and
    all(.work_items[];
      (.key | startswith("v23_")) and
      (.dependencies | type) == "array" and
      (.write_set | length) > 0 and
      (.required_tests | length) > 0 and
      .output_contract == "evidence_bundle" and
      (.tool_refs | length) > 0 and
      all(.tool_refs[]; . == "tool:go") and
      all(.capability_refs[]; startswith("capability:"))
    )
  ' >/dev/null || fail emitted_plan

printf 'status=passed tasks=13 phases=4 parallel_waves=2 deferred_scopes=6 transfers=3\n'
