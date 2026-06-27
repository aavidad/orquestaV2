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
- Validacion estructural de refs, write-set relativo, observaciones, resultados
  y evidencias requeridas.

No incluye:

- runtime real;
- HTTP, MCP, web o CLI;
- OPES, Codex, proveedores, modelos, HOME, OAuth o secretos;
- cierre por heuristica textual.

Validacion:

```bash
go test -count=1 ./modulos/orquesta-goal
```
