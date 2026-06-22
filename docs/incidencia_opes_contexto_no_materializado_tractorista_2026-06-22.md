# Incidencia OPES Contexto No Materializado Tractorista

Fecha: 2026-06-22.

## Contexto

Run OPES:
`run-opes-tractorista-investigacion-bases-20260622`.

Objetivo: localizar bases oficiales de `operario-tractorista-grupo-5` antes de
producir el temario.

El `prepare-run` enviado por el director incluia fuentes oficiales candidatas
en `task.context`:

- pagina de temario de Diputacion de Granada publicada el 27/01/2023;
- BOE-A-2022-781, BOE de 18/01/2022, con remision al BOP Granada 230 de
  01/12/2021;
- administracion.gob.es `idRegistro=195561`;
- BOE-A-2022-17129, BOE de 20/10/2022, con remision al BOP Granada 136 de
  19/07/2022 y correccion BOP 151 de 09/08/2022;
- BOP Granada 18/07/2024 como trazabilidad de personas aprobadas.

## Sintomas

1. `validate-request` acepto el payload y mostro el contexto completo en la
   respuesta.
2. El `agent_packet.json` real entregado al agente externo llego con:
   `context.total_bytes=0`.
3. El paquete conservo una entrada `required=true` y `mode=ref_only`, pero no
   materializo las fuentes ni una accion de lectura concreta util para recuperar
   el contexto original.
4. El objetivo del agente quedo truncado y sin las fuentes candidatas, por lo
   que el agente investigo casi a ciegas y pudo cerrar un falso bloqueo.
5. La API de estado mostro simultaneamente `overall_percentage=100`,
   `completion_percentage=1`, `tasks_open=1`, `agents_in_flight=1` y
   `closure_status=blocked`, una combinacion confusa para direccion OPES.
6. Se pidio `runs/control stop forced=true`; `runs/supervise` devolvio
   `stop_reason=stopped`, pero mantuvo evidencia de `open_tasks=1` y no hubo
   `agent_ack.json` ni checkpoint del agente.
7. En la segunda run, ya con brief fisico y ACK valido, `runs/supervise`
   devolvio `run_supervisor_execute_error` por `transicion_invalida: status` al
   abrir revision. El estado persistido tenia `delivered_tasks=1`,
   `delivered_agents=1` y `deliveries=1`, pero la run quedo `bloqueada` en fase
   `programacion` con un blocker `autoprogramming-open-review` y la cola seguia
   `running`.

## Impacto

- OPES no puede confiar en que las fuentes oficiales pasadas por `prepare-run`
  lleguen al agente.
- El agente puede producir un veredicto incorrecto por falta de contexto, aunque
  la validacion previa pareciera correcta.
- El director humano tiene que crear un brief fisico dentro del proyecto para
  recuperar contexto, lo que deberia ser una excepcion documentada, no el flujo
  normal.
- La lectura de estado no permite distinguir con claridad entre completitud,
  salud operativa y cierre real.

## Evidencia

- Payload del director:
  `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/investigacion_bases_tractorista_2026-06-22/payloads/prepare_run_investigacion_bases.json`.
- Respuesta de validacion con contexto completo:
  `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/investigacion_bases_tractorista_2026-06-22/responses/validate_request.json`.
- Packet real sin contexto:
  `/tmp/orquesta-opes-tractorista-bases-20260622/runtime/run-opes-tractorista-investigacion-bases-20260622/agent-ref-task-autoprogramming-75da65fbb143-g01/agent_packet.json`.
- Stop solicitado:
  `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/investigacion_bases_tractorista_2026-06-22/responses/stop_run_contexto_no_materializado.json`.
- Supervision de stop:
  `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/investigacion_bases_tractorista_2026-06-22/responses/supervise_stop_contexto_no_materializado.json`.
- ACK valido de la segunda run:
  `/tmp/orquesta-opes-tractorista-bases-20260622/runtime/run-opes-tractorista-investigacion-bases-brief-20260622/agent-ref-task-autoprogramming-4be0b9f5a814-g01/agent_ack.json`.
- Supervision con fallo de transicion tras ACK:
  `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/investigacion_bases_tractorista_2026-06-22/responses/supervise_run_brief_after_ack.json`.

## Tareas Tecnicas

- `CTX-TASK-001`: `prepare-run` debe materializar en el `agent_packet` el
  `task.context` aceptado por `validate-request`, o sustituirlo por refs
  resolubles con accion de lectura concreta. No debe degradar contexto critico a
  `total_bytes=0` sin ruta recuperable.
- `CTX-TASK-002`: si una entrada `required ref_only` queda sin contenido y sin
  ref recuperable, la run debe fallar de forma accionable antes de lanzar el
  agente, no pedir al agente que lo resuelva desde un paquete vacio.
- `CTX-TASK-003`: exponer en `/api/v0/autoprogramming/status` una diferencia
  clara entre salud operacional, progreso de artefactos y cierre causal. No
  devolver `overall_percentage=100` cuando `completion_percentage=1` y
  `tasks_open=1`.
- `CTX-TASK-004`: `runs/control stop` debe producir ACK/checkpoint de parada o
  reportar `stop_without_ack` con tarea abierta. `stopped` no debe ocultar que
  no hubo entrega causal.
- `CTX-TASK-005`: si una tarea de autoprogramacion ya tiene ACK entregado, no
  hay agentes pendientes y el unico bloqueo es `autoprogramming-open-review`,
  el supervisor debe reconciliar la run, resolver el blocker tecnico y abrir o
  reintentar la fase `revision` de forma duradera. No debe devolver
  `runtime_error` ni dejar cola `running` sin accion siguiente.

## Criterio De Cierre

Un smoke OPES equivalente debe:

- enviar fuentes candidatas en `task.context`;
- verificar que el agente recibe esas fuentes o refs legibles en
  `agent_packet.json`;
- si el contexto no puede materializarse, bloquear antes del launch con error
  publico accionable;
- al parar una run, mostrar estado final coherente entre proceso, ACK,
  tareas abiertas y porcentaje.
- si una run de autoprogramacion con ACK entregado queda bloqueada en
  `autoprogramming-open-review`, una supervision posterior debe resolver el
  blocker tecnico, abrir `revision` y continuar sin intervencion manual.

## Correccion Parcial Aplicada

Parche local aplicado el 2026-06-22:

- `modulos/orquesta-app-codex-stack/run_coordinator_autoprogramming_review_recovery_v0.go`
  añade reconciliacion duradera para `autoprogramming-open-review`.
- La recuperacion se ejecuta desde supervision enriquecida, drenaje directo y
  recuperacion de cola.
- Pruebas añadidas en `run_delivered_closure_queue_v0_test.go` para el flujo
  `enrich` y para `DrainRunV0`.
- Validacion ejecutada:
  `go test -count=1 ./modulos/orquesta-app-codex-stack`.

Quedan abiertas `CTX-TASK-001` a `CTX-TASK-004`; `CTX-TASK-005` queda cubierta
por este parche y pendiente de smoke OPES real con servidor actualizado.
