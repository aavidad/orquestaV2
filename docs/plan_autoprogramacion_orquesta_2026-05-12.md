# Plan de autoprogramacion de Orquesta

Fecha: 2026-05-12.

## Objetivo

Orquesta debe poder recibir una orden de mejora sobre su propio repositorio,
arrancar un director, repartir trabajo entre agentes reales, observar progreso,
aceptar cambios validados y continuar aunque se corte la sesion humana.

La sesion humana solo debe actuar como operador/revisor. No debe decidir cuantos
agentes hacen falta ni dirigir manualmente cada fase cuando el contrato de
entrada ya contiene objetivo, alcance, restricciones y write-set.

## Estado actual

Ya existe:

- servidor residente con `start`, `status`, `stop`, `/healthz` y `/api/status`;
- API REST para arrancar director de app, pedir cambios, pausar/reanudar/parar
  runs y consultar estadisticas de director;
- contratos `request_kind` para crear, modificar, documentar, analizar,
  revisar, validar, seguridad, deploy, i18n y refactor;
- director con cierre por fases, rework, split de tareas y estadisticas
  compactas;
- runtime Codex por conector, con control de progreso, ACK, review gate y
  parada de agentes;
- politica de no copiar legacy en bloque.

Actualizacion 2026-05-13:

- `cmd/orquesta-server` ya cablea conectores file-based para `RunStore`,
  `EventSink`, `WorkflowTaskStore/Writer`, `AgentProcessRegistry`,
  outbox, `RunControl`, `RunQueue` y `AppChangeRecordStore`;
- el gap ya no es "todo el estado operativo vive en memoria" en el servidor
  residente por defecto;
- el siguiente bloqueo serio es demostrar continuidad E2E de una ejecucion
  larga: reinicio de daemon, rehidratacion de cola/outbox/procesos, supervision
  posterior y cierre de entregas sin intervencion humana.

Gap que bloquea autoprogramacion robusta:

- falta una prueba real temporal sobre el propio repositorio que sobreviva a
  reinicio controlado del daemon y continue hasta delivery/review;
- falta asegurar que todos los puertos necesarios para una ejecucion larga se
  rehidratan desde conectores, sin fallback silencioso a memoria;
- falta politica de parada cooperativa con deadline cuando los agentes siguen
  progresando pero no han emitido checkpoint final.

## Decision de arquitectura

La persistencia operativa entra como conectores file-based reemplazables, no
como base de datos hardcodeada.

El core conserva solo puertos. El servidor decide por configuracion que
adaptador usar. La implementacion inicial sera local-first y JSON atomico
porque sirve para desarrollo, pruebas y recuperacion sin acoplar SQLite,
Postgres ni otro backend.

## Trabajo repartible

1. `state-file core`: cerrado en version inicial. Conector durable para
   `RunStore`, `EventSink`, `WorkflowTaskStore/Writer` y
   `AgentProcessRegistry`.
2. `state-file outbox`: cerrado en version inicial. Conector durable para
   pending, claim y ack del outbox.
3. `state-file control`: cerrado en version inicial. Conector durable para
   `RunControl`, `RunQueue` y `AppChangeRecordStore`.
4. `mcp/autoprogramacion`: version inicial disponible para validar requests de
   mejora de app existente o autoprogramacion. Queda ampliar pruebas reales y
   politica de aceptacion.
5. `server wiring`: cerrado en version inicial. El servidor residente usa los
   conectores file-based indicados arriba.
6. `smoke continuidad servidor`: cerrado en version inicial sin agentes reales.
   `scripts/smoke_orquesta_server_restart_state.sh` valida que RunQueue
   persiste y reabre tras reinicio con estado temporal.
7. `smoke real`: lanzar una orden sobre el propio repo con director y agentes
   reales, comprobar estadisticas, progreso, parada, reinicio y continuidad.

Trabajo futuro, no cerrado:

1. `mcp/autoprogramacion`: endurecer contrato REST/MCP para pedir mejora de app
   existente o autoprogramacion, validando alcance, write-set y tests
   requeridos.
2. `continuidad daemon larga`: test temporal con estado aislado que arranca una
   orden real con agentes, reinicia daemon y verifica que no se pierden cola,
   outbox, procesos ni entregas.
3. `shutdown cooperativo largo`: validar deadline en smoke real y proyectar
   razon de cierre por agente para runs con progreso real.

## Criterios de aceptacion

- `go test -count=1 ./...` pasa.
- Al recrear cada conector file-based se recupera el estado escrito.
- `cmd/orquesta-server` no instancia memoria para estado productivo durable por
  defecto salvo fallback explicito de test/debug.
- El endpoint/API de trabajo acepta `request_kind=modificar_app_existente` y
  no filtra detalles internos de runtime, HOME, DB o proveedor al core.
- La prueba real se considera valida solo si los agentes son arrancados por
  Orquesta y no por la sesion humana.
