#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

source "$ROOT/scripts/lib/parallel_test_runner.sh"

orquesta_parallel_test_init
trap orquesta_parallel_test_cleanup EXIT

orquesta_parallel_test_start rails \
  "$ROOT/scripts/test_rails_fast.sh"

orquesta_parallel_test_start autoprogramming \
  "$ROOT/scripts/test_autoprogramming_fast.sh"

orquesta_parallel_test_start architecture_boundary \
  go test -count=1 . \
    -run '^TestNeutralOrchestrationPackagesDoNotImportProductAdapters$'

orquesta_parallel_test_wait
