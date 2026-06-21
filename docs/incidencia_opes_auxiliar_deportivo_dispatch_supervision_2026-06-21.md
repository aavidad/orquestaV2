# Incidencia OPES - Auxiliar Deportivo aceptado sin despacho verificable

Fecha: 2026-06-21
Origen: OPES director externo desde `/home/alberto/Trabajo/OPES`
Curso: `Auxiliar Deportivo`
Course ID OPES: `dipgra-auxiliar-deportivo-17`

## Resumen

Se han enviado 17 trabajos `external-work/run`, uno por cada tema oficial del curso `Auxiliar Deportivo`, usando el patrón OPES de padre por tema con 6 subroles. Orquesta respondió `estado=ok` en los 17 payloads y añadió evidencia `evidence-ref-external-work-run-queued`.

El problema inicial observado desde Orquesta fue de observabilidad y supervisión: después del envío, Orquesta mostraba parte de los trabajos como `running` y parte como `ready`, pero no aparecían carpetas en `/home/alberto/Trabajo/orquesta/.orquesta-runtime`. La comprobación posterior mostró que la ejecución real sí estaba arrancando en `/home/alberto/Trabajo/OPES/.orquesta-runtime`. Las llamadas manuales de supervisión siguen haciendo timeout o devolviendo diagnósticos incompletos.

## Evidencia

Payloads y respuestas OPES:

- `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/auxiliar_deportivo_2026-06-21/orquesta_payloads/payload_t001.json` ... `payload_t017.json`
- `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/auxiliar_deportivo_2026-06-21/orquesta_responses/response_t001.json` ... `response_t017.json`
- Matriz de reutilización: `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/auxiliar_deportivo_2026-06-21/MATRIZ_REUTILIZACION_INICIAL.md`

Estado observado por `/api/v0/autoprogramming/status`:

- `t001`, `t002`, `t003`, `t004`, `t005`, `t006`, `t007`, `t008`: `running`
- `t009` a `t017`: `ready`
- No se encontraron directorios con `aux-deportivo` en `/home/alberto/Trabajo/orquesta/.orquesta-runtime`.
- Sí se encontraron procesos reales `codex` y scripts `orquesta_codex_exec_v0.sh` en `/home/alberto/Trabajo/OPES/.orquesta-runtime/run-external-work-opes-job-opes-aux-deportivo-*`.
- En el momento corregido ya estaban activos padres de los temas `001` a `010`; `011` a `017` permanecían `ready`.
- Aún no se habían encontrado ACKs finales ni artefactos editoriales en los write-sets del curso.

Readiness:

- `GET /api/v0/server/readiness` devuelve `ready=true`.
- `external_bridge_status=disabled`.
- `external_bridge_ready=true`.

Timeouts:

- `POST /api/v0/runs/supervise` con varios `run_refs` de Auxiliar Deportivo: timeout a 10 s sin respuesta.
- `POST /api/v0/autoprogramming/supervise`: timeout a 20 s sin respuesta.

## Riesgo

Hay riesgo de diagnóstico erróneo por runtime no evidente desde el repo de Orquesta: el estado de cola puede ser correcto, pero las rutas de ejecución no son visibles donde el operador espera. Además, los timeouts de supervisión impiden saber por API si un `running` tiene proceso vivo, ACK pendiente o bloqueo. Esto puede llevar a relanzar trabajos duplicados o a documentar falsos negativos.

## Actualización 2026-06-21 13:45

La ejecución real sí produjo artefactos y ACKs en
`/home/alberto/Trabajo/OPES/.orquesta-runtime`, pero
`POST /api/v0/autoprogramming/status` seguía devolviendo `running` para runs que
ya tenían `agent_ack.json` y `codex_last_message.txt`.

Ejemplos verificados:

- `t002`: `agent_ack.json` con `status=blocked`, archivos de coordinación
  creados y bloqueo real por prueba obligatoria inexistente en `PATH`.
- `t003`: `agent_ack.json` con `status=completed`, prueba local pasada y
  borrador OPES no publicable creado.
- `t010`: `agent_ack.json` con `status=completed`, coordinación creada y nota
  de prueba obligatoria inexistente en `PATH`.

Además se observó un patrón de prueba obligatoria frágil:

- Algunos agentes reciben como prueba obligatoria un comando con nombre
  `opes-domain-test-topic_parent_with_subagents-...`.
- En `t002` y `t010` ese comando no existía en `PATH`, por lo que el agente tuvo
  que dejar el resultado como `command_not_found` o anotarlo como no superado.
- En `t003`, `t006` y `t012` los agentes empezaron a crear validadores locales
  con ese nombre dentro de su write-set para poder dejar evidencia ejecutable.

Esto no debe resolverse con convenciones improvisadas por cada agente. Orquesta
debe materializar el test requerido en una ruta ejecutable conocida, pasar la
ruta exacta como contrato o no exigir un comando inexistente.

## Reproducción

1. Desde OPES, enviar cualquiera de los payloads:
   `POST http://127.0.0.1:18789/api/v0/external-work/run`
2. Comprobar que devuelve `estado=ok` y `evidence-ref-external-work-run-queued`.
3. Consultar:
   `POST http://127.0.0.1:18789/api/v0/autoprogramming/status`
4. Buscar runtime en ambas raíces:
   `find /home/alberto/Trabajo/orquesta/.orquesta-runtime -maxdepth 4 -type d | rg 'aux-deportivo|auxiliar-deportivo|opes-aux'`
   `find /home/alberto/Trabajo/OPES/.orquesta-runtime -maxdepth 4 -type d | rg 'aux-deportivo|auxiliar-deportivo|opes-aux'`
5. Intentar supervisión:
   `POST /api/v0/runs/supervise`
   `POST /api/v0/autoprogramming/supervise`

## Criterio de arreglo

Orquesta debe cumplir al menos una de estas salidas de forma verificable:

- Si un run está `running`, debe existir proceso/carpeta/ACK o evidencia de ejecución enlazada al run, indicando la raíz runtime real usada.
- Si no puede despachar por `external_bridge_status=disabled`, el run debe quedar `blocked_waiting_external_bridge` o equivalente, no `running`.
- La supervisión no debe hacer timeout silencioso; debe devolver diagnóstico accionable por run.
- El supervisor debe poder promocionar `ready` a ejecución real o explicar por qué no.
- El estado debe distinguir `queued`, `ready`, `running_real`, `running_sin_proceso`, `blocked_bridge_disabled` y `completed`.
- La API de estado debe exponer `runtime_root` o `agent_control_dir` por run para evitar búsquedas manuales en rutas equivocadas.
- La ingesta debe reflejar ACKs ya escritos en disco y no mantener `running`
  indefinidamente cuando existe `agent_ack.json`.
- Las pruebas obligatorias deben llegar como rutas ejecutables o contratos
  materializados; no como nombres sueltos que no existen en `PATH`.

## Restricciones

No tocar reglas OPES de dominio dentro del núcleo genérico de Orquesta. El arreglo debe ser de estado, despacho, supervisión y evidencias, no una excepción específica para Auxiliar Deportivo.

## Actualización 2026-06-21 14:10

Se observaron tres fallos adicionales durante el rework del mapa BOP40 de
Auxiliar Deportivo:

1. Run de mapa con proceso vivo pero sin progreso observable:
   `run-external-work-opes-job-opes-aux-deportivo-rework-mapa-bop40-20260621-change-opes-aux-deportivo-rework-mapa-bop40-20260621`
   quedó varios minutos sin actualizar `codex_stderr.log`, sin `agent_ack.json`
   y sin archivos en `mapa_rework`.
2. Al escribir `orquesta_shutdown_request.json`, el agente despertó, terminó de
   materializar los artefactos de mapa y pasó el test local, pero detectó la
   petición de apagado justo antes del ACK terminal. Escribió
   `agent_shutdown_checkpoint_ack.json` y salió sin `agent_ack.json`, aunque
   `codex_last_message.txt` indicaba que el ACK se había escrito.
3. La nueva run `bop40_ola0_fuentes_manifest` quedó inicialmente en `ready` sin
   proceso. Los endpoints `POST /api/v0/autoprogramming/supervise` y
   `POST /api/v0/runs/supervise` hicieron timeout a 30 s, pero tras el timeout
   apareció proceso real en `/home/alberto/Trabajo/OPES/.orquesta-runtime`.

Evidencia OPES:

- Artefactos creados por el rework:
  `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/auxiliar_deportivo_2026-06-21/mapa_rework/`
- Validador local pasado:
  `opes-domain-test-course_map_rework-job-opes-aux-deportivo-rework-mapa-bop40-20260621`
- Control file de shutdown tardío:
  `/home/alberto/Trabajo/OPES/.orquesta-runtime/run-external-work-opes-job-opes-aux-deportivo-rework-mapa-bop40-20260621-change-opes-aux-deportivo-rework-mapa-bop40-20260621/agent-ref-task-ref-app-change-appchange-26e67c2d81ae203cc8c86bd7f9feb6a2/agent_shutdown_checkpoint_ack.json`

## Tareas de arreglo para Orquesta

- `ORQ-OPES-AUX-001`: reconciliar entrega materializada cuando existe
  `codex_last_message.txt` con "ACK" pero falta `agent_ack.json`; el estado no
  puede quedar como completado, pero debe exponer `completed_without_terminal_ack`
  con paths de artefactos y test receipts detectables.
- `ORQ-OPES-AUX-002`: impedir que una petición de shutdown tardía anule un ACK
  terminal si el agente ya ha pasado pruebas y está en la fase explícita de
  escribir `agent_ack.json`; debe preferirse ACK terminal o un checkpoint que
  incluya `completed_candidate_files` y `completed_candidate_tests`.
- `ORQ-OPES-AUX-003`: `runs/supervise` y `autoprogramming/supervise` no deben
  hacer timeout silencioso; si despachan después del timeout HTTP, debe quedar
  evento observable `dispatch_started_after_http_timeout`.
- `ORQ-OPES-AUX-004`: exponer en estado el `runtime_root` real y reconciliar
  runs antiguas que siguen como `running` aunque ya tengan ACK, checkpoint o no
  tengan proceso vivo.
- `ORQ-OPES-AUX-005`: materializar pruebas obligatorias como rutas ejecutables
  conocidas; no delegar en cada agente la creación improvisada de un comando con
  nombre lógico.

## Actualización 2026-06-21 14:45 - corte por cuota Codex

Durante la ola BOP40 de Auxiliar Deportivo se observó un bloqueo externo de
cuota Codex dentro de agentes reales:

```text
ERROR: You've hit your usage limit. Visit https://chatgpt.com/codex/settings/usage to purchase more credits or try again at 6:24 PM.
```

Runs afectados con evidencia:

- `bop40-t019`: proceso principal terminó sin `agent_ack.json`, pero dejó
  artefactos parciales útiles en OPES (`04_markdown`, `05_html`, `08_assets`,
  `10_tutor_rag` y `temas_bop`). Orquesta debe exponer
  `interrupted_quota_with_partial_artifacts` y abrir rework causal que reutilice
  esos archivos, no relanzar desde cero ni duplicar trabajo.
- `bop40-t037`: proceso principal terminó sin `agent_ack.json` y sin artefactos
  útiles detectados en el árbol del curso. Orquesta debe marcarlo como
  `interrupted_quota_no_artifacts` y reanudar el mismo run/payload cuando vuelva
  la cuota.
- `bop40-t027`: un `required-tests-retry` se cortó por cuota, aunque el padre
  principal ya tenía ACK `completed`. La reanudación debe repetir solo el retry
  de prueba, no recrear el tema.

También quedaron runs BOP40 aceptadas pero sin despacho visible en ese momento:
`006`, `007`, `008`, `009`, `010`, `011`, `030`, `031`, `032`, `033`, `039` y
`040`.

Tareas adicionales:

- `ORQ-OPES-AUX-006`: detectar `usage limit` en logs de agentes y convertirlo en
  estado durable `interrupted_quota`, con `retry_after` extraído del mensaje si
  existe.
- `ORQ-OPES-AUX-007`: reconciliar artefactos parciales cuando falta ACK terminal
  por cuota, separando `partial_artifacts_reusable` de `no_artifacts`.
- `ORQ-OPES-AUX-008`: al volver la cuota, reanudar automáticamente las runs
  interrumpidas y los retries pendientes sin exigir relanzamiento manual ni
  crear payloads duplicados.
- `ORQ-OPES-AUX-009`: cuando un `required-tests-retry` falle por cuota, no
  degradar el padre principal completado; repetir solo la prueba o registrar
  `test_retry_interrupted_quota`.

## Actualización 2026-06-21 14:50 - cola estabilizada

Al estabilizarse la ola no quedaban procesos `aux-deportivo-bop40-tNNN-padre`
activos. Estado OPES observado:

- 26 padres BOP40 con ACK principal, de los cuales 25 `completed` y `003`
  recuperado por `review-rework completed`.
- 14 `required-tests-retry` completados.
- Reintentos cortados por cuota: `027` y `038`.
- Sin ACK principal y pendientes de reanudación: `006`, `007`, `008`, `009`,
  `010`, `011`, `019`, `030`, `031`, `032`, `033`, `037`, `039`, `040`.
- `019` tiene artefactos parciales reutilizables; `037` no tiene artefactos
  útiles detectados.

Este estado confirma que Orquesta no debe depender de relanzamiento manual tras
el corte de cuota. Debe reanudar desde los runs aceptados, conservar parciales
útiles y distinguir `retry_test_interrupted_quota` de fallo del padre principal.

## Actualización 2026-06-21 23:56 - reanudación completada con anomalías de supervisión

Tras volver la cuota se arrancó un servidor Orquesta aislado para OPES en
`127.0.0.1:18789`, con `ORQUESTA_CODEX_PROJECT_WORKDIR=/home/alberto/Trabajo/OPES`
y runtime `/home/alberto/Trabajo/OPES/.orquesta-runtime`. Se reinyectaron solo
los temas pendientes/incompletos:
`006`, `007`, `008`, `009`, `010`, `011`, `019`, `030`, `031`, `032`, `033`,
`037`, `039` y `040`.

Resultado OPES:

- `40/40` temas BOP con artefactos no vacíos en `04_markdown`, `09_tests`,
  `10_tutor_rag`, `11_validacion`, `08_assets`, `05_html` y `temas_bop`.
- `40/40` runs BOP con `agent_ack.json` y `status=completed`.
- Sin procesos Codex activos al cierre.
- Checkpoint OPES:
  `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/auxiliar_deportivo_2026-06-21/CHECKPOINT_BOP40_OLA1_COMPLETADA_2026-06-21_2356.md`

Anomalías nuevas:

1. `POST /api/v0/runs/supervise` devolvió HTTP 500 para `007` y `030`, pero los
   runs acabaron despachando y entregando artefactos/ACK.
2. `POST /api/v0/runs/supervise` agotó timeout HTTP para `033`, `037`, `039` y
   `040`; después aparecieron procesos reales o entregas en runtime.
3. Los runs `009`, `010`, `033`, `037` y `039` no generaron
   `codex_last_message.txt`, aunque sí generaron `agent_ack.json completed` con
   ficheros y test receipts.
4. Una supervisión final informó `last=stopped` para `009`, `010`, `033` y
   `037`, pese a que el ACK terminal ya estaba escrito en disco.

Tareas adicionales:

- `ORQ-OPES-AUX-010`: reconciliar `agent_ack.json completed` como fuente
  terminal aunque falte `codex_last_message.txt`; el estado público no debe
  quedarse en `stopped` si existe ACK válido.
- `ORQ-OPES-AUX-011`: `runs/supervise` debe devolver respuesta parcial con
  `dispatch_started`, `process_ref` o `reconcile_pending` antes de agotar el
  timeout HTTP; no debe obligar al director a inferir el estado por filesystem.
- `ORQ-OPES-AUX-012`: cuando una llamada de supervisión devuelve HTTP 500 pero
  el proceso se lanza igualmente, registrar evento durable
  `supervision_http_error_but_dispatch_started` con `run_ref` y `process_ref`.
- `ORQ-OPES-AUX-013`: el estado público debe exponer, por run, conteo de
  `agent_ack.json`, `codex_last_message.txt`, proceso vivo y última escritura
  de artefacto, para distinguir bloqueo real de entrega completada.

## Actualización 2026-06-21 - ola 2 cierre textual BOP40

Durante la ola de cierre textual de los 40 temas BOP se repitieron anomalías de
supervisión:

1. Los temas `037`, `039` y `040` devolvieron timeout HTTP en
   `POST /api/v0/runs/supervise`, pero después aparecieron procesos Codex reales
   ejecutándose para esos mismos `run_ref`.
2. El tema `035` devolvió `estado=error`, `stop_reason=runtime_error`,
   `last.status=failed`, con evidencias `operational-director-plan-state:blocked`,
   `external-wait-exhausted`,
   `evidence-ref-app-director-operational-plan-state-wait-expired-v0` y
   `evidence-ref-app-director-wait-subagents-expired-v0`. La proyección indicaba
   una tarea abierta pero `requested_agents=0`, por lo que no llegó a lanzar
   agente real.
3. Una auditoría independiente de `Oficial de Servicios Múltiples` quedó primero
   en `waiting_outbox` tras `max_ticks` y solo arrancó cuando bajó la presión de
   concurrencia de Auxiliar Deportivo.

Tareas adicionales:

- `ORQ-OPES-AUX-014`: si `runs/supervise` agota timeout pero deja proceso real
  lanzado, responder con estado durable `dispatch_started_after_timeout` o
  permitir consulta posterior por `run_ref` sin inspección manual de procesos.
- `ORQ-OPES-AUX-015`: si el plan queda bloqueado con tareas abiertas y
  `requested_agents=0`, Orquesta debe replanificar o marcar `needs_replan` con
  causa clara; no debe quedar como fallo opaco de `operational_director_plan_state.active_step`.
- `ORQ-OPES-AUX-016`: el supervisor debe exponer cola y presión de concurrencia
  por proyecto para distinguir `waiting_outbox` normal de bloqueo real.

## Actualización 2026-06-22 - corrección parcial programada

Se programó una corrección en Orquesta para los fallos observados durante la
ola 2 de Auxiliar Deportivo:

- Si `SuperviseCodexV0` detecta un proceso real vivo (`running_live`), corta la
  supervisión con `stop_reason=dispatch_started` en vez de seguir ticks hasta
  `max_ticks`. Esto evita que una llamada HTTP larga agote timeout mientras el
  agente ya ha quedado lanzado.
- La salida MCP/HTTP de `runs/supervise` añade `next_actions` para
  `dispatch_started`, `waiting_outbox` y `needs_replan`, de modo que el operador
  no tenga que inferir el siguiente paso por `pgrep` o por inspección manual del
  runtime.
- Cuando el snapshot queda en `waiting_outbox`, la salida añade diagnóstico
  `run_supervisor_queue_pressure` con `queue_ref`, total de candidatos,
  ejecutables y conteos `ready`, `running`, `delivered`, `stopped` y `closed`.
- Si un drain devuelve plan operativo bloqueado por `external-wait-exhausted`,
  con tareas abiertas y `requested_agents=0`, el snapshot público pasa a
  `needs_replan` y conserva evidencia `evidence-ref-codex-supervisor-operational-plan-needs-replan`
  en lugar de quedar como `failed` opaco por
  `operational_director_plan_state.active_step`.

Archivos tocados:

- `modulos/orquesta-app-codex-stack/codex_supervisor_v0.go`
- `modulos/orquesta-app-codex-stack/run_supervisor_mcp_executor_v0.go`
- `modulos/orquesta-app-codex-stack/codex_supervisor_stack_lifecycle_v0.go`
- pruebas asociadas en `codex_supervisor_v0_test.go` y
  `codex_supervisor_stack_lifecycle_v0_test.go`
- prueba de diagnóstico de cola en
  `run_supervisor_mcp_executor_error_diagnostics_v0_test.go`

Pruebas ejecutadas:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexSupervisorV0|TestCodexSupervisorStackLifecycleV0SupervisaRunExistenteSinCanalParaleloV0|TestCodexSupervisorSnapshotNeedsOperationalReplanV0DetectaWaitSinAgentesV0|TestCodexStackRunSupervisorAPIV0EmpujaRunExistenteSinRelanzarAgentes|TestCodexStackRunSupervisorQueueDiagnosticsMCPV0ExponePresionWaitingOutbox'`
- `go test -count=1 ./modulos/orquesta-run-supervisor`
- `go test -count=1 ./modulos/orquesta-app-codex-stack`
- `go test -count=1 ./modulos/orquesta-mcp`
- `go test -count=1 ./cmd/orquesta-server`
