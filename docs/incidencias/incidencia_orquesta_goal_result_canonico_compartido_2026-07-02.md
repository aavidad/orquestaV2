# Incidencia: resultado goal-first compartido se sobrescribe entre goals

Fecha: 2026-07-02.

## Sintoma

La autoprogramacion genero varios resultados utiles por tarea, pero el fichero
`modulos/orquesta-server/docs/orquesta_goal_result_v0.json` se sobrescribio por
goals concurrentes. Un goal podia quedar `blocked` aunque su resultado per-task
existiera y los tests estuvieran pasados, porque el artefacto canonico del
modulo ya contenia el resultado de otro goal.

## Causa

El paquete Codex Goal pedia siempre el mismo resultado durable:
`<write_set>/docs/orquesta_goal_result_v0.json`. Ese nombre es compatible con
un goal aislado, pero no con autoprogramacion paralela sobre el mismo modulo.
El observador y el scanner de artefactos tambien estaban centrados en ese nombre
legacy.

## Cierre

El prompt ahora pide un sidecar unico por `goal_ref`:
`orquesta_goal_result_<goal-ref-seguro>.json`. El observador sigue aceptando el
nombre legacy para compatibilidad, pero tambien escanea los sidecars por goal y
filtra por `goal_ref`. El scanner de refs materializadas reconoce esos sidecars
como receipt terminal, no como artefacto de producto.

Pruebas:

```bash
go test -count=1 ./modulos/orquesta-runtime-codex-goal ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack
go test -count=1 ./...
```
