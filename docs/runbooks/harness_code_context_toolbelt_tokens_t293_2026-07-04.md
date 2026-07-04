# Runbook: harness de tokens para CodeContext toolbelt MCP

Fecha: 2026-07-04.

Scope: task-autonomia-t2-context-broker-mcp-toolbelt-20260704-002.

Objetivo: comparar el uso de tokens de un goal que consulta codigo mediante
`orquesta.codebase.query.v0` contra el baseline T293=164945, sin inflar
contexto crudo y sin usar `codebase-memory-mcp` salvo opt-in central explicito.

## Invariantes

- La consulta de codigo entra por `CodeContextQueryPortV0`.
- La herramienta para el agente Codex es MCP local:
  `orquesta.codebase.query.v0`, `transport=mcp_local_sin_http_localhost`.
- HTTP `/api/v0/codebase/query` queda como compatibilidad del servidor, no como
  requisito del agente.
- No copiar ficheros completos, transcripts ni dumps largos al prompt.
- No arrancar indexadores por agente.

## Harness A/B

Preparar dos goals con mismo HEAD, objetivo y write-set:

- Brazo A: exploracion libre con `rg`/`sed` acotado, sin broker de codigo en el
  toolbelt.
- Brazo B: mismo objetivo, pero indicando al agente que use
  `orquesta.codebase.query.v0` para busquedas de simbolo, arquitectura o
  `repo_map`.

Para cada brazo registrar un JSON compacto:

```json
{
  "schema_version": "orquesta_code_context_toolbelt_token_harness.v0",
  "baseline_ref": "T293",
  "baseline_tokens": 164945,
  "arm": "A|B",
  "goal_ref": "",
  "commit_ref": "",
  "input_tokens": 0,
  "cached_input_tokens": 0,
  "output_tokens": 0,
  "total_tokens": 0,
  "duration_ms": 0,
  "quality_status": "accepted|needs_review|rejected",
  "evidence_refs": []
}
```

Las metricas de tokens deben venir del receipt/observacion del backend cuando
existan (`input_tokens`, `cached_input_tokens`, `total_tokens` o equivalentes).
Si el backend no las expone, usar solo `tokens_estimated` como preflight y
dejar `quality_status=needs_review`; esa estimacion no cierra decision de
producto.

## Calculo

Comparar contra T293:

```text
delta_tokens = total_tokens - 164945
improvement_ratio = (164945 - total_tokens) / 164945
```

Criterio operativo inicial:

- mejora aceptable si el brazo B reduce tokens o duracion al menos 15% frente a
  A y mantiene `quality_status=accepted`;
- si B no mejora o pierde cobertura, no activar el broker por defecto;
- si faltan metricas reales de backend, repetir cuando el backend publique
  usage y conservar solo la evidencia de harness.

## Verificacion local

El cambio de toolbelt queda cubierto por:

```bash
go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-runtime-codex-appserver ./cmd/orquesta-server -run Toolbelt
```

La prueba focal valida que `orquesta.codebase.query.v0` se anuncia como MCP
local, asociado a `CodeContextQueryPortV0`, y que su entrada MCP no depende de
`POST /api/v0/codebase/query`.
