# orquesta-goal

Contrato neutral para el modo goal-first de Orquesta.

En este modelo Orquesta no fuerza un loop interno de `wait/review/replan/close`
para cada paso. Orquesta compila un `GoalWorkSpecV0` con objetivo, reglas,
contexto, write-set, tests requeridos, artefactos esperados y politica de
cierre. Un runtime externo ejecuta el goal y actua como Director operativo del
trabajo hasta `complete` o `blocked`.

El modulo no conoce Codex. Codex Goal entra por un adaptador opt-in en
`modulos/orquesta-runtime-codex-goal`.

Incluye:

- `GoalWorkSpecV0`: contrato de entrada para lanzar un goal gobernado por
  Orquesta.
- `GoalWorkResultV0`: resultado observable de un goal.
- Puertos de lanzamiento, observacion y validacion.
- Lifecycle neutral `StartGoalWorkV0`/`ObserveGoalWorkV0` sobre puertos:
  lanza, persiste estado, observa y valida cierre sin tocar runs, HTTP,
  filesystem, Codex ni OPES.
- Observacion batch neutral `ObserveActiveGoalWorksV0`: lista estados goal
  activos por puerto opcional y reutiliza `ObserveGoalWorkV0` para una pasada
  residente sin reconstruir el loop director historico.
- Validacion estructural de refs, write-set relativo, observaciones, resultados
  y evidencias requeridas.
- Contrato nativo de artefactos materializados:
  `materialized_artifacts` declara cada fichero/artefacto producido con path,
  tipo, estado (`valid`, `invalid`, `partial`, `non_publishable`), evidencias e
  issues; `checklist` declara expectativas completadas/faltantes; y
  `rework_plan_refs` enlaza el plan causal de reparacion cuando el cierre no es
  publicable.

Regla de cierre:

- Un resultado `complete` con cualquier `materialized_artifacts.status` distinto
  de `valid` queda bloqueado y necesita rework.
- Si la politica exige checklist, no se acepta cierre sin
  `checklist.expected_refs` o con `checklist.missing_refs`.
- Si hay artefactos parciales y la politica exige plan de rework, no se acepta
  cierre sin `rework_plan_refs`.
- Los `artifact_paths` y los paths de `materialized_artifacts` se validan contra
  el `write_set`; el scanner de filesystem puede ayudar a recuperar evidencias,
  pero el contrato canonico de cierre vive en `GoalWorkResultV0`.

No incluye:

- runtime real;
- HTTP, MCP, web o CLI;
- OPES, Codex, proveedores, modelos, HOME, OAuth o secretos;
- cierre por heuristica textual.

Validacion:

```bash
go test -count=1 ./modulos/orquesta-goal
```
