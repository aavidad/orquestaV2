#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

source "$ROOT/scripts/lib/go_tool.sh"
orquesta_go_tool_ensure_path

go test -count=1 ./modulos/orquesta-rails
go test -count=1 ./modulos/orquesta-core-workflow -run 'TestRail|TestValidateWorkflowTaskV0AllowsOpaqueOperationalDetails|TestValidateWorkflowTaskV0RejectsForbiddenDetails'
go test -count=1 ./modulos/orquesta-context -run 'TestContext.*DetailRail|TestBuildContextBundleV0RechazaDetallesProhibidos'
go test -count=1 ./modulos/orquesta-director-agent -run 'TestDirectorAgentDetailRail|TestValidateDirectorAgentDecisionV0RechazaDetalleSensible'
go test -count=1 ./cmd/orquesta-server -run 'Test.*DetailRails'
