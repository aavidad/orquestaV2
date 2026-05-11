#!/usr/bin/env bash
set -euo pipefail

cat <<'MSG'
Contexto local de orquesta-agent-process-registry:
- Lee AGENTS.md, README.md y docs/*.md antes de modificar.
- Mantente en contratos neutrales: sin nucleo, persistence, runtime, DB, HOME, OAuth, provider, prompt ni transcript.
- Trabaja en microtareas y manten ficheros pequenos.
MSG
