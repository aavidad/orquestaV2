# Incidencia: el recibo goal-first debe declarar artifact_paths

Fecha: 2026-07-02

## Sintoma

Un goal-first OPES pudo escribir artefactos fuera del `write_set` estrecho y
cerrar con un JSON terminal que declaraba solo la fase prevista. El trabajo
extra quedo oculto para cierre causal y QA.

## Riesgo

Si el recibo terminal no lista rutas de ficheros creados, modificados o
verificados, Orquesta solo puede validar `artifact_refs` opacas y no puede
aplicar el bloqueo ya existente para `artifact_paths` fuera del `write_set`.

## Avance aplicado

- El prompt Codex Goal incluye `artifact_paths` en la forma obligatoria del
  `ORQUESTA_GOAL_RESULT_V0`.
- Instruye listar todas las rutas relativas de ficheros creados, modificados o
  verificados para cierre, incluido el JSON durable.
- Si el agente escribio fuera del `write_set`, debe declararlo y cerrar como
  `blocked` con `summary=out_of_scope_artifacts`, no ocultarlo.
- `orquesta-goal` ya bloquea cierre cuando `artifact_paths` declara rutas fuera
  del `write_set`; este avance conecta ese gate con el contrato del agente.

## Evidencia

Tests:

```bash
go test -count=1 ./modulos/orquesta-runtime-codex-goal ./modulos/orquesta-goal
git diff --check
```

## Residual

Sigue pendiente el cierre fuerte de `BUG-ORQ-20260701-085`: comparar el
write-set con un snapshot/manifest independiente de filesystem para detectar
rutas omitidas en el recibo terminal.
