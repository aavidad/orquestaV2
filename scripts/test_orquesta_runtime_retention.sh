#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
script="$ROOT/scripts/orquesta_runtime_retention.sh"

workdir="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-runtime-retention-test.XXXXXX")"
trap 'rm -rf "$workdir"' EXIT

mkdir -p \
  "$workdir/.orquesta-runtime/old-run" \
  "$workdir/.orquesta-runtime/new-run" \
  "$workdir/.orquesta-runtime/old-blocked" \
  "$workdir/.orquesta-runtime/old-plan" \
  "$workdir/.orquesta-runtime/old-registry-running" \
  "$workdir/.orquesta-runtime/codex-waves/old-wave" \
  "$workdir/.orquesta-runtime/codex-waves/domain-container" \
  "$workdir/.orquesta-runtime/codex-waves/domain-container/old-nested-wave" \
  "$workdir/.orquesta-runtime/codex-waves/new-wave" \
  "$workdir/.orquesta-runtime/waves/old-director-wave" \
  "$workdir/.orquesta-purged-old" \
  "$workdir/.orquesta-purged-new"

touch -d '20 days ago' \
  "$workdir/.orquesta-runtime/old-run" \
  "$workdir/.orquesta-runtime/old-blocked" \
  "$workdir/.orquesta-runtime/old-registry-running" \
  "$workdir/.orquesta-runtime/codex-waves/old-wave" \
  "$workdir/.orquesta-runtime/codex-waves/domain-container" \
  "$workdir/.orquesta-runtime/codex-waves/domain-container/old-nested-wave" \
  "$workdir/.orquesta-runtime/waves/old-director-wave" \
  "$workdir/.orquesta-purged-old"
touch -d '1 day ago' \
  "$workdir/.orquesta-runtime/new-run" \
  "$workdir/.orquesta-runtime/codex-waves/new-wave" \
  "$workdir/.orquesta-purged-new"
touch "$workdir/.orquesta-runtime/old-blocked/agent_ack.json"
touch -d '20 days ago' "$workdir/.orquesta-runtime/old-blocked/agent_ack.json"
touch -d '20 days ago' "$workdir/.orquesta-runtime/old-blocked"
printf '{"schema_version":"orquesta_plan.v0"}\n' >"$workdir/.orquesta-runtime/old-plan/plan.json"
printf '{"artifacts":[]}\n' >"$workdir/.orquesta-runtime/old-plan/artifacts_manifest.json"
touch -d '20 days ago' \
  "$workdir/.orquesta-runtime/old-plan/plan.json" \
  "$workdir/.orquesta-runtime/old-plan/artifacts_manifest.json" \
  "$workdir/.orquesta-runtime/old-plan"
cat >"$workdir/.orquesta-runtime/old-registry-running/codex_wave_registry_v0.json" <<'JSON'
{"agents":[{"pid":999999,"status":"running"}]}
JSON
touch -d '20 days ago' "$workdir/.orquesta-runtime/old-registry-running/codex_wave_registry_v0.json"
touch -d '20 days ago' "$workdir/.orquesta-runtime/old-registry-running"
cat >"$workdir/.orquesta-runtime/codex-waves/old-wave/codex_wave_registry_v0.json" <<'JSON'
{"agents":[{"pid":0,"status":"completed"}]}
JSON
cat >"$workdir/.orquesta-runtime/waves/old-director-wave/codex_wave_registry_v0.json" <<'JSON'
{"agents":[{"pid":0,"status":"completed"}]}
JSON
touch -d '20 days ago' \
  "$workdir/.orquesta-runtime/codex-waves/old-wave/codex_wave_registry_v0.json" \
  "$workdir/.orquesta-runtime/waves/old-director-wave/codex_wave_registry_v0.json" \
  "$workdir/.orquesta-runtime/codex-waves/old-wave" \
  "$workdir/.orquesta-runtime/waves/old-director-wave"

dry_run_output="$("$script" --repo "$workdir" --min-age-days 7)"
grep -q 'mode=dry-run' <<<"$dry_run_output"
grep -q 'old-run' <<<"$dry_run_output"
grep -q 'old-blocked' <<<"$dry_run_output"
grep -q 'action=blocked' <<<"$dry_run_output"
grep -q 'agent_ack_unreconciled' <<<"$dry_run_output"
grep -q 'old-plan' <<<"$dry_run_output"
grep -q 'durable_plan_or_artifact_manifest' <<<"$dry_run_output"
grep -q 'old-registry-running' <<<"$dry_run_output"
grep -q 'agent_live' <<<"$dry_run_output"
grep -q 'old-wave' <<<"$dry_run_output"
grep -q 'old-director-wave' <<<"$dry_run_output"
grep -q 'domain-container' <<<"$dry_run_output"
grep -q 'wave_registry_missing' <<<"$dry_run_output"
grep -q '.orquesta-purged-old' <<<"$dry_run_output"
if grep -q 'new-run\|new-wave\|.orquesta-purged-new' <<<"$dry_run_output"; then
  echo "dry-run included recent runtime entries" >&2
  exit 1
fi

for path in \
  "$workdir/.orquesta-runtime/old-run" \
  "$workdir/.orquesta-runtime/codex-waves/old-wave" \
  "$workdir/.orquesta-runtime/old-blocked" \
  "$workdir/.orquesta-runtime/old-plan" \
  "$workdir/.orquesta-runtime/old-registry-running" \
  "$workdir/.orquesta-runtime/codex-waves/domain-container" \
  "$workdir/.orquesta-purged-old"; do
  [ -d "$path" ] || {
    echo "dry-run deleted $path" >&2
    exit 1
  }
done

if "$script" --repo "$workdir" --min-age-days 7 --delete >/dev/null 2>&1; then
  echo "delete without confirmation succeeded" >&2
  exit 1
fi

"$script" --repo "$workdir" --min-age-days 7 --delete --confirm-delete orquesta-runtime-retention >/dev/null

for removed in \
  "$workdir/.orquesta-runtime/old-run" \
  "$workdir/.orquesta-runtime/codex-waves/old-wave" \
  "$workdir/.orquesta-runtime/waves/old-director-wave" \
  "$workdir/.orquesta-purged-old"; do
  [ ! -e "$removed" ] || {
    echo "candidate was not deleted: $removed" >&2
    exit 1
  }
done

for kept in \
  "$workdir/.orquesta-runtime" \
  "$workdir/.orquesta-runtime/codex-waves" \
  "$workdir/.orquesta-runtime/waves" \
  "$workdir/.orquesta-runtime/old-blocked" \
  "$workdir/.orquesta-runtime/old-plan" \
  "$workdir/.orquesta-runtime/old-registry-running" \
  "$workdir/.orquesta-runtime/codex-waves/domain-container" \
  "$workdir/.orquesta-runtime/codex-waves/domain-container/old-nested-wave" \
  "$workdir/.orquesta-runtime/new-run" \
  "$workdir/.orquesta-runtime/codex-waves/new-wave" \
  "$workdir/.orquesta-purged-new"; do
  [ -d "$kept" ] || {
    echo "safe/recent path was deleted: $kept" >&2
    exit 1
  }
done

echo "orquesta_runtime_retention_ok=true"
