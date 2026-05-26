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
objetivo configurado, hay capacidad libre y existe planner inyectado, pide nuevas
tareas sin duplicar las que ya estan en cola. Los skips no terminales del tick
cuentan como presion de cola, pero no bloquean por si solos la planificacion si
queda hueco bajo el objetivo.

Validacion:

- `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`;
- el planner recibe `known_request_refs`/`known_run_refs`;
- `cmd/orquesta-server` puede generar una tarea scanner para ampliar backlog
  cuando no quedan tareas nuevas concretas.

## SRV-TASK-007: supervisor residente reentrante

Estado: hecho local.

Origen: `task-ref-self-improvement-afbb86d34eb5`. Refs opacas preservadas:
`worktree-ref-orquesta-local-parallel-01`,
`branch-ref-orquesta-local-parallel-01`.

Objetivo: el supervisor residente debe observar el estado disponible, ejecutar
un pulso acotado y volver al bucle tras cada supervision o preparacion de
automejora. No debe quedar retenido esperando agentes largos ni mezclar trabajo
secundario con el trabajo principal.

Validacion:

- 2026-05-26, `go test -count=1 ./modulos/orquesta-server`.
- `TestRuntimeV0SupervisorAsyncCoalesceaUnTickPendienteV0` demuestra que el
  pulso residente no se solapa, coalescea un unico tick pendiente durante una
  supervision activa y libera el slot al terminar.
- `TestRuntimeV0SupervisorPreparaAutomejoraTrasIdleV0` demuestra que preparar
  automejora en segundo plano no bloquea el siguiente pulso del supervisor.
- `TestRuntimeV0SupervisorPreparaAutomejoraConCapacidadLibreV0` y
  `TestRuntimeV0SupervisorPreparaAutomejoraConCapacidadLibreAunqueHayaSkipsV0`
  cubren automejora por capacidad libre con cola visible y planner inyectado.
- `TestIdleSelfImprovementAuditPayloadRedactaMensajeDeBlockerV0` cubre que la
  auditoria de blockers conserva refs/evidencias compactas y redacta mensajes
  con rutas, HOME, tokens, prompts o transcripts.

## SRV-TASK-008: auditoria startup compacta

Estado: hecho local.

Objetivo: la auditoria JSONL del autodiagnostico de arranque no debe persistir
payloads crudos con paths de proyecto, runtime o state; debe conservar solo una
traza compacta para operador y Director.

Validacion:

- 2026-05-26, `go test -count=1 ./modulos/orquesta-server -run 'TestRuntimeV0PrepareStartup'`.
- `TestRuntimeV0PrepareStartupAuditaSummarySinPathsV0` demuestra que
  `startup_check_ready` usa `command_summary`/`result_summary`, no `command` ni
  `result` crudos, y no serializa paths sensibles.

## SRV-TASK-009: visibilidad de fallo de persistencia de estado

Estado: hecho local.

Origen: T95 `server-state-persist-failure-visibility`.

Objetivo: los fallos no fatales del `StateStorePortV0` posteriores al arranque
no deben quedar ocultos. El status residente debe exponer una proyeccion
compacta y redactada con contador, codigo publico, transicion afectada y ultimo
instante observado.

Validacion:

- `go test -count=1 ./modulos/orquesta-server`;
- `TestRuntimeV0PersistStateFailureVisibleSinFiltrarDetallesV0` cubre
  `state_persist_failed` en memoria/status sin rutas locales ni error crudo del
  store;
- `TestRuntimeV0PersistStateConfirmedRecuperaEstadoDegradadoV0` cubre que una
  persistencia posterior correcta marca `state_persist_status=ok` y conserva el
  contador historico de fallos.
