# Autoprogramacion: sincronizacion de estado de backlog

Fecha: 2026-05-24.

## Contrato cerrado

- El planner residente consume `Estado:` canonico del backlog y docs locales de
  modulo como cierre documental.
- Los ACKs de `.orquesta-runtime` siguen entrando solo como refs correladas por
  el lector estricto existente; no se usa Git ni worktrees como fuente de verdad.
- Si una seccion no tiene estado canonico pero muestra evidencia ambigua, el
  planner crea una revision documental acotada a docs en vez de abrir codigo.
- Si la unica tarea viable ya esta visible en cola, no crea fallback ni scanner
  duplicado y devuelve `backlog_tareas_ya_visibles_en_cola`.

## Validacion

```bash
go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server
```
