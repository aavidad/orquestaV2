#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "${script_dir}/.." && pwd)"
catalog="${repo_root}/product/knowledge/tooling_adoption_v1.json"
roadmap="${repo_root}/product/roadmap.json"
query="${repo_root}/scripts/consultar_herramientas.sh"

fail() {
  printf 'status=failed check=%s\n' "$1" >&2
  exit 1
}

jq -e . "${catalog}" >/dev/null || fail json

jq -e '
  .schema_version == 1 and
  .document_kind == "tooling_adoption_read_model" and
  (.entries | length) >= 25 and
  ([.entries[].key] | unique | length) == (.entries | length) and
  all(.entries[];
    (.key | length) > 0 and
    (.name | length) > 0 and
    (.capability_ids | length) > 0 and
    (
      .status == "accredited" or
      .status == "partial" or
      .status == "candidate" or
      .status == "conditional" or
      .status == "debt"
    ) and
    (.v2_evidence | length) > 0 and
    (.owner_vertical | length) > 0 and
    (.decision | length) > 0 and
    (.next_gate | length) > 0
  )
' "${catalog}" >/dev/null || fail catalog_contract

jq -n -e \
  --slurpfile catalog "${catalog}" \
  --slurpfile roadmap "${roadmap}" '
    ([
      $roadmap[0] |
      .. |
      objects |
      select(.id? != null) |
      .id
    ] | unique) as $known |
    all($catalog[0].entries[].capability_ids[]; . as $id | ($known | index($id)) != null)
  ' >/dev/null || fail roadmap_capabilities

jq -e '
  [.entries[] | select(.key == "firecracker_microvm")] as $firecracker |
  ($firecracker | length) == 1 and
  $firecracker[0].status == "candidate" and
  $firecracker[0].owner_vertical == "agent_runtime_elastic" and
  $firecracker[0].capability_ids == ["EVD-04", "EVD-13", "EXT-21", "ORC-28"] and
  $firecracker[0].technical_decision == {
    "id": "agent_microvm_network",
    "status": "planned_not_applied"
  } and
  $firecracker[0].next_gate == "A+B+C sobre el mismo candidato: A núcleo elástico neutral sin KVM ni Firecracker; B adaptador Firecracker opt-in sin fallback y una microVM por agente; C ola física 1/5/10/16/20; solo A+B+C permiten acreditar V38."
' "${catalog}" >/dev/null || fail firecracker_contract

all_count="$("${query}" --all | wc -l)"
catalog_count="$(jq '.entries | length' "${catalog}")"
[[ "${all_count}" == "${catalog_count}" ]] || fail all_query

"${query}" --status debt | rg -q '^skill_creator[[:space:]]' ||
  fail status_query
"${query}" --capability CTX-03 | rg -q '^caveman_compact_profile[[:space:]]' ||
  fail capability_query
"${query}" --term firecracker | rg -q '^firecracker_microvm[[:space:]]' ||
  fail term_query
"${query}" --capability EXT-10 --json |
  jq -e 'length == 1 and .[0].key == "git_worktrees"' >/dev/null ||
  fail json_query

if "${query}" --term no_existe_esta_herramienta >/dev/null 2>&1; then
  fail no_match_exit
fi

printf 'status=passed entries=%s queries=5 roadmap_refs=valid\n' "${catalog_count}"
