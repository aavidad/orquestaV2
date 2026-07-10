# Handoff F6: normalizador neutral de resultados Goal

Fecha: 2026-07-10.

## Resultado

`orquesta-goal` incorpora `DecodeGoalWorkResultJSONV0`, contrato neutral para
recibos JSON terminales. Acepta el contrato canónico y recupera aliases de
Claude, Gemini y Codex sin introducir proveedores en el módulo puro.

- El resultado de decode declara `disposition` (`canonical`, `repaired` o
  `irrecoverable`), issues tipadas y `repair_receipt` durable.
- El receipt conserva sólo versión de esquema original, hash SHA-256 del input,
  transformaciones compactas y evidence refs; no conserva payload, descripción
  de evidencia ni texto sensible.
- Se normalizan aliases de schema/status/goal/required tests, `missing_refs`,
  `reason_code`, issues string y `evidence_refs` string u objeto
  `{ref,description}`. La descripción se descarta deliberadamente.
- Claude y Gemini usan el decoder compartido; el reader materializado del stack
  y los parsers de marcador/fichero de Codex app-server también pasan por él.
  El saneamiento de salida propio de app-server permanece como frontera de
  egress, después de la normalización neutral.
- La validación de required tests mantiene compatibilidad con resultados
  históricos sin evidencia declarada en el spec. Cuando un `GoalRequiredTestV0`
  exige `evidence_refs`, sólo pasa si el resultado aporta evidencia causal no
  vacía; no exige inventar igualdad con las refs del spec. Las refs exactas de
  cierre las gobierna `ClosurePolicy.RequiredEvidenceRefs`. Issues advisory no
  se convierten en veto de cierre.
- Un JSON presente pero irrecuperable se proyecta como resultado terminal
  `invalid`, nunca como `running`/`pending`. Claude y Gemini ya no reintentan
  releer el fichero con `Gosched`; el writer debe publicar un recibo completo.

## Corpus y verificación

El corpus está en `modulos/orquesta-goal/testdata/goal_result_json/` e incluye
canónico, aliases Claude/Gemini/Codex, objetos de evidencia BUG-170,
checklist/missing refs recuperables e input irrecuperable. Las pruebas cubren
equivalencia de forma proveedor, decode-encode-decode y frontera causal.

Ejecutado con cachés fuera del worktree:

```bash
TMPDIR=/tmp/f6-goal-result GOTMPDIR=/tmp/f6-goal-result \
GOCACHE=/tmp/f6-goal-result/gocache \
go test -count=1 ./modulos/orquesta-goal ./modulos/orquesta-runtime-claude \
  ./modulos/orquesta-runtime-gemini ./modulos/orquesta-runtime-codex-goal \
  ./modulos/orquesta-runtime-codex-appserver ./modulos/orquesta-app-codex-stack
```

También pasaron `go test -count=20 ./modulos/orquesta-runtime-claude` y, dos
veces, los focos locales de cola/restart, watcher, observe y secuencia OPES
goal-first del stack. No hubo rojos externos.

Integración: el branch F6 fue rebasado localmente sobre `6a8cb3e066`; HEAD
actual `3a232ebf2` preserva el WIP previo y hay cambios sin commit para esta
corrección final. No se usó SSH, remoto, deploy ni procesos residentes.

estado_final: ready_for_operator_f6_no_commit
