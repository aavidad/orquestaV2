# Tareas

## SRV-001

Crear handler residente con `healthz`, `/api/status` y delegacion al handler de
aplicacion.

## SRV-002

Crear statefile atomico para reconexion tras cortar la sesion de Codex.

## SRV-003

Crear bucle de supervision acotado y testeado.

## SRV-004

Crear comando fino que arranque `run`, `start`, `status` y `stop` sin meter
logica del nucleo en `cmd`.

## SRV-TASK-004: cablear estado durable

Objetivo: el servidor residente debe arrancar con conectores persistentes para
estado operativo, de forma que una ejecucion de autoprogramacion pueda
reanudarse tras corte o reinicio.

Write-set previsto:

- `cmd/orquesta-server/stack.go`
- modulo adaptador file-based bajo `modulos/`
- tests de arranque/recreacion de stores

Validacion:

- los stores recuperan estado al recrear instancia;
- `go test -count=1 ./...`;
- ninguna dependencia de DB concreta queda en el servidor.

## SRV-TASK-005: conector OPES opt-in desde cmd

Objetivo: permitir que el servidor productivo inyecte `domain_work` hacia OPES
sin que el runtime residente conozca OPES.

Estado: hecho.

Validacion:

- `ORQUESTA_OPES_BASE_URL` activa el executor REST OPES;
- sin `ORQUESTA_OPES_BASE_URL`, el executor queda `nil`;
- `go test -count=1 ./cmd/orquesta-server`.

## SRV-TASK-006: automejora por capacidad libre

Estado: hecho.

Objetivo: el servidor residente no debe esperar a estar completamente idle para
seguir pensando trabajo de automejora. Si la cola visible esta por debajo del
objetivo configurado y no hay skips pendientes en el tick, pide al planner nuevas
tareas sin duplicar las que ya estan en cola.

Validacion:

- `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`;
- el planner recibe `known_request_refs`/`known_run_refs`;
- `cmd/orquesta-server` puede generar una tarea scanner para ampliar backlog
  cuando no quedan tareas nuevas concretas.

## SRV-TASK-007: supervisor residente reentrante

Estado: pendiente documentado.

Origen: `task-ref-self-improvement-afbb86d34eb5`. Refs opacas preservadas:
`worktree-ref-orquesta-local-parallel-01`,
`branch-ref-orquesta-local-parallel-01`.

Objetivo: el supervisor residente debe observar el estado disponible, ejecutar
un pulso acotado y volver al bucle tras cada supervision o preparacion de
automejora. No debe quedar retenido esperando agentes largos ni mezclar trabajo
secundario con el trabajo principal.

Validacion prevista:

- `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`;
- un caso focal debe demostrar que el pulso usa limites de espera acotados y que
  la siguiente iteracion del bucle puede ejecutar nueva supervision;
- la automejora de fondo conserva write-set propio, refs opacas y prioridad baja
  sin bloquear runs principales.
