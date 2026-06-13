# Incidencia OPES: reconciliacion de ACKs en runs externos A2

Fecha: 2026-06-13.

## Contexto

Instancia OPES temporal:

```text
http://127.0.0.1:8793
ORQUESTA_CODEX_PROJECT_WORKDIR=/home/alberto/Trabajo/OPES
runtime=/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/a2_informatica_2026-06-13/orquesta_opes_rework_2026-06-13/runtime
state=/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/a2_informatica_2026-06-13/orquesta_opes_rework_2026-06-13/state
```

Curso:

```text
informatica-a2-analista-programador-60
```

Se lanzaron 60 trabajos externos, uno por tema, con padres Codex para cierre/rework causal. Varios padres escribieron `agent_ack.json` con `status=completed`.

## Sintoma

Algunos runs con ACK completado y review aceptada quedan en `/api/v0/director/stats` como:

```text
status=activa
tasks_delivered=1
agents_delivered=1
accepted_reviews=1
tasks_closed=0
closures=0
```

La cola puede mostrar esos runs como `running` o `ready` aunque ya exista ACK en disco. Ejemplos observados:

```text
run-opes-a2-informatica-t001-cierre-20260613
run-opes-a2-informatica-t006-cierre-20260613
run-opes-a2-informatica-t007-cierre-20260613
run-opes-a2-informatica-t008-cierre-20260613
run-opes-a2-informatica-t013-cierre-20260613
```

Ademas, una supervision dirigida secuencial sobre `t001` devolvio HTTP 500:

```text
POST /api/v0/runs/supervise
run_ref=run-opes-a2-informatica-t001-cierre-20260613
response=500
stop_reason=runtime_error
last.status=failed
evidence_refs incluye:
- operational-director-plan-state:blocked
- operational-closure-run-not-active
- evidence-ref-app-director-operational-plan-state-review-accepted-v0
- evidence-ref-app-director-operational-plan-state-required-tests-passed-v0
- evidence-ref-app-director-operational-plan-state-closure-blocked-v0
- evidence-ref-app-director-operational-plan-state-closure-prerequisite-ready-v0
```

`t002` agoto tiempo de cliente en una supervision dirigida posterior. `t003`, `t004`, `t006`, `t007` y `t013` devolvieron `200 ok stopped`.

### Sintoma adicional: bucle de supervisor por `active_step`

Despues de avanzar la ola y recibir ACKs de varios temas, el audit empezo a
registrar errores repetidos:

```text
event=supervisor_tick_error
status=error
error=app_director_service_invalido: operational_director_plan_state.active_step
stop_reason=tick_error
total_executions=0
queue_size=0
```

En paralelo, `resident_director_tick_result` repetia sobre el mismo run ya
completado:

```text
run_ref=run-opes-a2-informatica-t012-cierre-20260613
status=completed
executed_actions=22
```

Esto convive con procesos Codex vivos que si hacen trabajo real en otros temas
(`039`, `046`, `048` en el corte observado), por lo que no se debe matar la ola
sin antes preservar sus ACKs y artefactos.

### Sintoma adicional: shutdown no cierra instancia sin agentes vivos

Tras terminar los procesos Codex vivos, el shutdown ordenado dejo la instancia
en `waiting_checkpoint` aunque `ps` ya no mostraba procesos Codex del runtime
OPES. El endpoint informo 5 agentes en vuelo por falta de
`agent_shutdown_checkpoint_ack.json` en runs ya terminados o con ACK de
contenido.

Un segundo intento con `forced=true` devolvio HTTP 500:

```text
estado=error
shutdown_ready=false
runs_requested=0
runs_stopped=0
agents_in_flight=0
checkpoints_pending=0
errores_publicos[0].code=server_shutdown_http_error
errores_publicos[0].message=server_shutdown_executor_error
```

Este caso obliga al operador a cerrar por senal un servidor que ya no tiene
agentes vivos para evitar que siga en bucle de supervisor.

## Impacto

El trabajo de contenido no se pierde: los ACKs existen en runtime y son recuperables. El problema es de reconciliacion/cierre de Orquesta:

- el director residente no siempre materializa `tasks_closed` y `closures` tras ACK y review aceptada;
- la cola no siempre refleja el estado real de entrega;
- `runs/supervise` puede devolver 500 para un run ya entregado en lugar de reconciliarlo o informar un estado recuperable.

Esto obliga al director OPES a actualizar el registro por tema desde fuera y puede dejar una ola amplia con estados ambiguos.

## Reproduccion focal

1. Arrancar instancia OPES con cola amplia y proyecto OPES.
2. Enviar trabajos externos `run-opes-a2-informatica-t001...`.
3. Esperar a que el agente escriba `agent_ack.json` con `status=completed`.
4. Consultar:

```bash
curl -fsS http://127.0.0.1:8793/api/v0/director/stats \
  -H 'Content-Type: application/json' \
  -d '{"request_id":"req-stats-a2-t001-20260613","run_ref":"run-opes-a2-informatica-t001-cierre-20260613"}'
```

5. Ejecutar:

```bash
curl -fsS http://127.0.0.1:8793/api/v0/runs/supervise \
  -H 'Content-Type: application/json' \
  -d '{"request_id":"req-director-reconcile-a2-t001-20260613","run_ref":"run-opes-a2-informatica-t001-cierre-20260613","max_ticks":1,"max_bursts":4,"max_steps_per_burst":8,"max_dispatches_per_wait":4,"max_commands":8,"max_outbox_per_cycle":8,"max_decision_cycles":4,"max_external_waits":1}'
```

## Criterio de arreglo

- Si hay ACK `completed`, entrega ingerida, review aceptada y pruebas requeridas pasadas, Orquesta debe cerrar la task padre o dejar un bloqueo causal explicito y recuperable, no un 500.
- `/api/v0/runs/supervise` debe ser idempotente para runs ya entregados o parcialmente cerrados.
- La cola debe reconciliar `ready/running/delivered` con ACKs y procesos vivos sin marcar falsos activos permanentes.
- El registro OPES debe poder actualizarse desde el conector oficial sin depender de una actualizacion manual del director.
- El supervisor residente no debe entrar en bucle sobre un run ya completado ni
  convertir un `active_step` recuperable en `tick_error` permanente con
  `total_executions=0`.
- El shutdown forzado debe poder cerrar una instancia sin agentes vivos aunque
  queden checkpoints de apagado obsoletos por una reconciliacion previa rota.

## Tarea tecnica pendiente para Orquesta

Titulo: reconciliar ACKs externos, checkpoints de apagado y cierre de runs OPES
sin falsos activos.

Alcance:

- implementar una reconciliacion idempotente entre `agent_ack.json`,
  reviews aceptadas, estado de cola, procesos vivos y cierres de workflow;
- convertir checkpoints de apagado obsoletos en estado recuperable cuando no
  existan procesos vivos;
- hacer que `forced=true` cierre el servidor o devuelva una causa accionable
  sin dejar el puerto abierto;
- evitar bucles del supervisor residente sobre runs ya completados;
- exponer evidencia suficiente para que OPES no tenga que actualizar el
  registro desde fuera.

Criterios de aceptacion:

- una ola OPES con ACKs completados queda con `tasks_closed` y `closures`
  coherentes tras una llamada repetible a supervise;
- `/api/v0/server/shutdown` normal no espera checkpoints inexistentes cuando no
  hay procesos vivos;
- `/api/v0/server/shutdown forced=true` libera el puerto en una instancia sin
  agentes reales;
- hay pruebas de regresion para ACK completado, cola inflada, checkpoint stale
  y shutdown forzado;
- los errores publicos incluyen `run_ref`, `agent_ref`, causa y accion de
  recuperacion.

## Evidencias locales

```text
/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/a2_informatica_2026-06-13/orquesta_opes_rework_2026-06-13/state/audit/audit.jsonl
/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/a2_informatica_2026-06-13/orquesta_opes_rework_2026-06-13/state/run-state/queue_v0.json
/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/a2_informatica_2026-06-13/orquesta_opes_rework_2026-06-13/runtime/run-opes-a2-informatica-t001-cierre-20260613/**/agent_ack.json
```

## Reproduccion adicional en instancia de reanudacion 8794

Fecha: 2026-06-14.

Instancia OPES temporal:

```text
http://127.0.0.1:8794
ORQUESTA_CODEX_PROJECT_WORKDIR=/home/alberto/Trabajo/OPES
runtime=/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/a2_informatica_2026-06-13/orquesta_opes_rework_reanudacion_2026-06-13/runtime
state=/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/a2_informatica_2026-06-13/orquesta_opes_rework_reanudacion_2026-06-13/state
```

Resultado de contenido:

```text
course_id=informatica-a2-analista-programador-60
registro OPES: 60 temas en rework_orquesta_ack_recibido_no_publicable
ACKs de la instancia 8794: 40/40 temas esperados
procesos Codex activos del runtime de reanudacion: 0
```

El apagado ordenado por `POST /api/v0/server/shutdown` devolvio
`waiting_checkpoint` con 35 checkpoints pendientes aunque no existian procesos
Codex activos de la reanudacion. El apagado forzado devolvio:

```text
estado=error
shutdown_ready=false
runs_requested=0
runs_stopped=0
agents_in_flight=0
checkpoints_pending=0
errores_publicos[0].code=server_shutdown_http_error
errores_publicos[0].message=server_shutdown_executor_error
```

El servidor seguia escuchando en `127.0.0.1:8794` con PID `2468698` y el
proceso padre `go run` PID `2468639`, por lo que el director OPES tuvo que
cerrar la instancia por senal `TERM` tras preservar ACKs, registro y evidencias.
