#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

source "$ROOT/scripts/lib/parallel_test_runner.sh"

orquesta_parallel_test_init
trap orquesta_parallel_test_cleanup EXIT

orquesta_parallel_test_start stack_closure \
  go test -count=1 ./modulos/orquesta-app-codex-stack \
    -run '^(TestCodexStackAutoprogrammingPrepareRunAPIV0CierraConPlanStateYTestsRealesV0|TestOperationalClosureTaskClassifierV0AceptaMarkerEnContextRefsV0)$'

orquesta_parallel_test_start stack_prepare_queue \
  go test -count=1 ./modulos/orquesta-app-codex-stack \
    -run 'Test(CodexStackAutoprogrammingPrepareRunAPIV0|AutoprogrammingDirectorDecisionSourceV0|StackDrainQueueStatus)'

orquesta_parallel_test_start director_plan_state \
  go test -count=1 ./modulos/orquesta-app-director-service \
    -run 'Test(EnsureContinueOperationalDirectorPlanStateFromWorkflowTasksV0AceptaMarkerEnContextRefsV0|ContinueAppDirectorV0DecisionPlanStateEjecutaRunnerYCierra)'

orquesta_parallel_test_start mcp_web_prepare \
  go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-web \
    -run 'Test.*AutoprogrammingPrepareRun'

orquesta_parallel_test_wait
