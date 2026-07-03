#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
script="$ROOT/scripts/benchmark_context_compression_optin.sh"
workdir="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-context-compression-test.XXXXXX")"
trap 'rm -rf "$workdir"' EXIT

fixture="$workdir/non_critical_context.md"
cat >"$fixture" <<'MD'
# Non critical context sample

This document is a disposable narrative sample for CTX-TASK-801D and T284.
It carries refs that must survive compression:
scan-ref-backlog-d438fd99b7a6
task-instance-ref-backlog-f43e8f42
docs/autoprogramacion_orquesta_pendientes_2026-05-23.md:line:16:sha256:49ddfa8cf3f609e42a9cec0342e28231a4bce588da6c8af8ba14f046c9e78d7a

MD

for i in $(seq 1 80); do
  printf 'Narrative line %03d repeats low priority explanatory material for the benchmark fixture.\n' "$i" >>"$fixture"
done

if bash "$script" --input "$fixture" --output "$workdir/without_confirm.json" 2>"$workdir/missing_confirm.err"; then
  echo "benchmark unexpectedly ran without opt-in confirmation" >&2
  exit 1
fi
grep -q "ORQUESTA_CONTEXT_COMPRESSION_BENCHMARK_CONFIRM" "$workdir/missing_confirm.err"

json_out="$workdir/result.json"
compressed_out="$workdir/compressed.md"
ORQUESTA_CONTEXT_COMPRESSION_BENCHMARK_CONFIRM=CTX_COMPRESSION_BENCHMARK_OPT_IN \
  bash "$script" \
  --input "$fixture" \
  --output "$json_out" \
  --compressed-output "$compressed_out" \
  --target-ratio 0.25

JSON_OUT="$json_out" COMPRESSED_OUT="$compressed_out" python3 - <<'PY'
import json
import os
from pathlib import Path

payload = json.loads(Path(os.environ["JSON_OUT"]).read_text(encoding="utf-8"))
compressed = Path(os.environ["COMPRESSED_OUT"]).read_text(encoding="utf-8")
metrics = payload["metrics"]
if payload["schema_version"] != "orquesta_context_compression_benchmark.v0":
    raise SystemExit("unexpected schema")
if payload["status"] != "valid_benchmark":
    raise SystemExit(f"unexpected status {payload['status']!r}")
if payload["policy"]["productive_adapter_allowed_by_this_script"]:
    raise SystemExit("script must not allow a product adapter")
if metrics["compressed_bytes"] >= metrics["source_bytes"]:
    raise SystemExit(f"bytes did not shrink: {metrics!r}")
if metrics["compressed_tokens_estimated"] >= metrics["source_tokens_estimated"]:
    raise SystemExit(f"tokens did not shrink: {metrics!r}")
if metrics["lost_refs_count"] != 0 or metrics["lost_refs"]:
    raise SystemExit(f"refs lost: {metrics['lost_refs']!r}")
if metrics["elapsed_ms"] < 0:
    raise SystemExit(f"invalid elapsed_ms: {metrics['elapsed_ms']!r}")
for expected in [
    "CTX-TASK-801D",
    "T284",
    "scan-ref-backlog-d438fd99b7a6",
    "task-instance-ref-backlog-f43e8f42",
    "docs/autoprogramacion_orquesta_pendientes_2026-05-23.md:line:16:sha256:49ddfa8cf3f609e42a9cec0342e28231a4bce588da6c8af8ba14f046c9e78d7a",
]:
    if expected not in compressed:
        raise SystemExit(f"compressed output lost {expected!r}")
PY

protected="$workdir/AGENTS.md"
cat >"$protected" <<'MD'
# Contexto para agentes

This file is intentionally protected and must be refused by the benchmark.
MD

if ORQUESTA_CONTEXT_COMPRESSION_BENCHMARK_CONFIRM=CTX_COMPRESSION_BENCHMARK_OPT_IN \
  bash "$script" --input "$protected" --output "$workdir/protected.json" 2>"$workdir/protected.err"; then
  echo "benchmark unexpectedly accepted protected input" >&2
  exit 1
fi
grep -q "protected input refused" "$workdir/protected.err"
