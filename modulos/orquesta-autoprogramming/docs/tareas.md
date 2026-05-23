# Tareas: orquesta-autoprogramming

## Estado actual

- Extraido desde `orquesta-orchestration-core` para separar reglas de
  programacion del nucleo generico.
- MCP usa este modulo como preflight.
- `orquesta-runtime-codex-delivery` usa este modulo para evaluar ACKs de codigo.
- La automejora secundaria ya queda modelada como contrato puro:
  `BuildAutoprogrammingSelfImprovementRequestV0` produce una request de baja
  prioridad para que el adaptador la prepare/encole despues.

## Backlog

- APG-001: ampliar limites por contrato si OPES o apps grandes necesitan tareas
  largas sin microfragmentar contenido.
- APG-002: versionar contratos `v1` cuando existan perfiles de trabajo por tipo
  de app, sin romper `v0`.
- APG-003: anadir fixtures de solicitud real desde web/MCP cuando el wizard
  guiado este estabilizado.
