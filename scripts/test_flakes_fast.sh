#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

COUNT="${ORQUESTA_FLAKE_COUNT:-20}"

go test ./cmd/orquesta-server \
  -run '^TestCodexLaunchDirectorWaveCommandV0RecursiveFakeRuntimeEjecutableConLinaje$' \
  -count="$COUNT" -shuffle=on -timeout=180s -v

go test ./modulos/orquesta-runtime \
  -run 'Test(RuntimeFakeLifecycleV0|ExternalAgentProcessBatchV0)' \
  -count="$COUNT" -shuffle=on -timeout=90s -v
