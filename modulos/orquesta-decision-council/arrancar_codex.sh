#!/usr/bin/env bash
set -euo pipefail

cat <<'TXT'
Contexto local recomendado para orquesta-decision-council:
- AGENTS.md
- README.md
- docs/contratos.md
- docs/tareas.md
- docs/pruebas.md
- docs/decisiones.md
- ../../modulos/CONTRATOS.md si la tarea afecta a otros modulos

Reglas clave:
- Consejo multiagente puro: propuesta, critica, voto y quorum.
- Sin DB, runtime, proveedor, HOME, OAuth, secretos, prompts completos ni transcripts.
- Familias de agente como refs opacas; los proveedores concretos son configuracion externa.
- Si falta contrato de otro modulo: CONSULTA AL DIRECTOR.
TXT
