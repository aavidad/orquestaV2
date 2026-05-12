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

Gap que bloquea autoprogramacion robusta:

- el servidor residente monta todavia parte del estado operativo en memoria:
  runs, eventos, outbox, tareas, cambios de app, registro de procesos, control y
  cola de runs;
- si el daemon cae o se reinicia, puede perder cola y contexto operativo aunque
  el statefile del servidor siga existiendo;
- por tanto Orquesta puede hacer pruebas reales, pero aun no es aceptable para
  una ejecucion larga de autoprogramacion sin persistencia de puertos.

## Decision de arquitectura

La persistencia operativa entra como conectores file-based reemplazables, no
como base de datos hardcodeada.

El core conserva solo puertos. El servidor decide por configuracion que
adaptador usar. La implementacion inicial sera local-first y JSON atomico
porque sirve para desarrollo, pruebas y recuperacion sin acoplar SQLite,
Postgres ni otro backend.

## Trabajo repartible

1. `state-file core`: conector durable para `RunStore`, `EventSink`,
   `WorkflowTaskStore/Writer` y `AgentProcessRegistry`.
2. `state-file outbox`: conector durable para pending, claim y ack del outbox.
3. `state-file control`: conector durable para `RunControl`, `RunQueue` y
   `AppChangeRecordStore`.
4. `mcp/autoprogramacion`: contrato REST/MCP para pedir mejora de app existente
   o autoprogramacion, validando alcance, write-set y tests requeridos.
5. `server wiring`: reemplazar memoria por conectores persistentes cuando el
   servidor residente arranca en modo durable.
6. `smoke real`: lanzar una orden sobre el propio repo con director y agentes
   reales, comprobar estadisticas, progreso, parada y continuidad.

## Criterios de aceptacion

- `go test -count=1 ./...` pasa.
- Al recrear cada conector file-based se recupera el estado escrito.
- `cmd/orquesta-server` no instancia memoria para estado productivo durable por
  defecto salvo fallback explicito de test/debug.
- El endpoint/API de trabajo acepta `request_kind=modificar_app_existente` y
  no filtra detalles internos de runtime, HOME, DB o proveedor al core.
- La prueba real se considera valida solo si los agentes son arrancados por
  Orquesta y no por la sesion humana.

