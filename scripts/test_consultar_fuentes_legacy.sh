#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
query="${script_dir}/consultar_fuentes_legacy.sh"

bash -n "$query"

summary="$("$query" --summary)"
jq -e '
  .markdown_sources == 840 and
  .actionable_sources == 339 and
  .context_only_sources == 501 and
  .kinds.bug == 156 and .kinds.skill == 19 and .kinds.task == 164 and
  .kinds.context == 501 and
  .decisions.accepted == 77 and .decisions.conditional == 19 and
  .decisions.historical == 243 and .decisions.context_only == 501 and
  .sources_with_capabilities == 339 and
  .bug_sources_with_pending_invariant_test == 156 and
  .depth == "classified_markdown_source_inventory" and
  .claims_historical_success == false and
  .claims_current_bug == false and
  .claims_reuse_decision == false
' <<<"$summary" >/dev/null

bugs="$("$query" --capability GOV-18 --kind bug --json)"
jq -e '
  length == 3 and all(.[];
    .kind == "bug" and
    (.capability_ids | index("GOV-18")) != null and
    .lesson_state == "pending_invariant_test" and
    .depth == "source_disposition_lead_not_content_assessment" and
    (.recommended_next_action | length) > 40 and
    .claims_current_bug == false)
' <<<"$bugs" >/dev/null

context="$("$query" --source docs/00_INDICE.md --json)"
jq -e '
  length == 1 and .[0].kind == "context" and .[0].actionable == false and
  .[0].classification == "reference" and
  .[0].depth == "classified_context_not_actionable_source" and
  .[0].claims_reuse_decision == false
' <<<"$context" >/dev/null

if "$query" --capability BAD >/dev/null 2>&1; then
  printf '%s\n' 'Se aceptó una capability inválida.' >&2
  exit 1
fi
if "$query" --source ruta/inexistente.md >/dev/null 2>&1; then
  printf '%s\n' 'Una fuente ausente no debe devolver éxito.' >&2
  exit 1
fi

printf '%s\n' 'status=passed markdown_sources=840 actionable=339 context=501'
