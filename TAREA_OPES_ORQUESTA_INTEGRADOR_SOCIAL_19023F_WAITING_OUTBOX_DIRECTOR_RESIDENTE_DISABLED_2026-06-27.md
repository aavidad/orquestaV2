# Tarea Orquesta: Integrador Social 19023f queda parcialmente dependiente de supervisión

Fecha: 2026-06-27.

## Contexto

Desde OPES se lanzó una primera ola real para el curso:

- `course_id`: `auxiliar-tecnico-superior-integracion-social-b-dipgra-2026`
- curso: Auxiliar Técnico/a Superior de Integración Social
- temas: `001` a `006`
- servidor Orquesta: `127.0.0.1:19023`
- ola: `20260627-19023f`

Los payloads se generaron desde los requests 19023e ya validados y se corrigieron
para usar `20260627-19023f` y `00_control/director_20260627`. Antes de lanzar se
comprobó que cada fichero solo mencionaba su propio `topic_id`.

## Evidencia

Requests OPES:

```text
/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/auxiliar-tecnico-superior-de-integracion-social/00_control/director_20260627/orquesta_requests/external_work_tema_001_rework_productivo_request_19023f.json
/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/auxiliar-tecnico-superior-de-integracion-social/00_control/director_20260627/orquesta_requests/external_work_tema_002_rework_productivo_request_19023f.json
/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/auxiliar-tecnico-superior-de-integracion-social/00_control/director_20260627/orquesta_requests/external_work_tema_003_rework_productivo_request_19023f.json
/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/auxiliar-tecnico-superior-de-integracion-social/00_control/director_20260627/orquesta_requests/external_work_tema_004_rework_productivo_request_19023f.json
/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/auxiliar-tecnico-superior-de-integracion-social/00_control/director_20260627/orquesta_requests/external_work_tema_005_rework_productivo_request_19023f.json
/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/auxiliar-tecnico-superior-de-integracion-social/00_control/director_20260627/orquesta_requests/external_work_tema_006_rework_productivo_request_19023f.json
```

Responses HTTP 200:

```text
/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/auxiliar-tecnico-superior-de-integracion-social/00_control/director_20260627/orquesta_responses/external_work_tema_001_rework_productivo_response_19023f.json
...
/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/auxiliar-tecnico-superior-de-integracion-social/00_control/director_20260627/orquesta_responses/external_work_tema_006_rework_productivo_response_19023f.json
```

Los seis responses devuelven `estado: ok` y `evidence-ref-external-work-run-queued`.

Primer `runs/supervise` de T001:

```json
{
  "estado": "ok",
  "run_ref": "run-opes-integracion-social-b-t001-rework-productivo-20260627-19023f",
  "stop_reason": "max_ticks",
  "last": {
    "status": "waiting_outbox",
    "evidence_refs": [
      "wait_unhandled_outbox",
      "evidence-ref-codex-supervisor-drain-projection-tasks-7",
      "evidence-ref-codex-supervisor-drain-projection-open-tasks-7",
      "evidence-ref-codex-supervisor-drain-projection-requested-agents-0"
    ]
  },
  "diagnostics": [
    {
      "code": "run_supervisor_queue_pressure",
      "message": "queue_ref=global total=36 executable=6 ready=6 running=0 delivered=0 stopped=30 closed=0"
    }
  ]
}
```

Después, `/api/v0/runs/queue/priority` mostró:

```text
T001: running
T002-T006: ready
```

Actualización posterior en la misma sesión:

```text
T001-T006: running
```

También aparecieron carpetas runtime con padre y subroles reales bajo:

```text
/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/auxiliar-tecnico-superior-de-integracion-social/00_control/director_20260625/orquesta_server/19023_runtime/run-opes-integracion-social-b-t00*-rework-productivo-20260627-19023f/
```

Esto acota el fallo: no es una imposibilidad total de lanzamiento, sino una
mezcla de `waiting_outbox`, ticks residentes poco claros y observabilidad
insuficiente para saber si la ola terminará sola.

El `server/status` de 19023 indica:

```text
startup_ready=true
resident_director_status=disabled
external_bridge_status=disabled
ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false
```

## Problema

Orquesta acepta los trabajos OPES, pero la autonomía práctica queda incompleta:

1. El director residente está desactivado en la instancia usada para OPES.
2. Un run puede mostrar `waiting_outbox` antes de pasar a ejecución real, sin que
   quede claro si requiere supervisión manual, espera o solo más ticks.
3. La evidencia pública mezcla `ready`, `running` y `waiting_outbox`, pero no deja
   claro al operador si la ola terminará sola o requiere intervención.
4. `/api/v0/director/stats` sin `run_ref` devuelve `run_ref_requerido`, lo que
   dificulta una vista global simple de cola viva para OPES.

## Resultado esperado

Para trabajos OPES lanzados por `external-work/run`, Orquesta debería:

- activar o exponer claramente el modo autónomo necesario para que las runs ready
  se materialicen sin llamadas manuales;
- drenar outbox y despachar agentes de forma residente cuando haya capacidad;
- devolver un estado global humano de cola con `ready`, `running`, `waiting_outbox`,
  `stalled`, `done` y causa pública accionable;
- no requerir que OPES haga polling manual de cada `run_ref` para saber si una ola
  sigue viva;
- conservar compatibilidad con `runs/supervise` para intervención puntual.

## No tocar desde esta tarea

No se debe cambiar código de OPES para rodear este problema. La solución pertenece
a Orquesta: supervisor residente, outbox/dispatch, estado global y autonomía de
external-work.

## Actualización 2026-06-27: ola 19023g

Al lanzar la siguiente ola de Integración Social B (`t007` a `t012`) desde OPES:

- `POST /api/v0/external-work/run` aceptó los seis trabajos con HTTP 200.
- `runs/supervise` de `t007` devolvió `dispatch_started` y materializó agentes.
- `runs/supervise` de `t008` también devolvió `dispatch_started` y materializó
  padre/subroles.
- Las llamadas paralelas a `runs/supervise` de `t009`, `t010`, `t011` y `t012`
  quedaron más de 90 segundos sin cuerpo JSON. Se interrumpieron los clientes
  `curl` para no dejar procesos de observación colgados.
- Durante ese bloqueo solo existían carpetas runtime reales para `t007` y
  `t008`; `t009`-`t012` aparecían solo en los procesos `curl`/payload, no como
  agentes materializados.

Estado observado:

```text
run-opes-integracion-social-b-t007-rework-productivo-20260627-19023g: runtime creado
run-opes-integracion-social-b-t008-rework-productivo-20260627-19023g: runtime creado
run-opes-integracion-social-b-t009..t012-rework-productivo-20260627-19023g: aceptados, sin runtime en el momento de la observación
```

Impacto operativo:

- OPES puede seguir trabajando con `t007` y `t008`, pero no debe dar por vivos
  `t009`-`t012` solo porque `external-work/run` haya respondido `ok`.
- La API necesita diferenciar con claridad `accepted`, `queued_not_dispatched`,
  `dispatch_started`, `running_live` y `supervise_hung`.
- `runs/supervise` debería responder con timeout controlado y cuerpo JSON aunque
  no pueda despachar por presión de cola, capacidad o lock interno.

Actualización posterior de la misma ola: tras interrumpir los clientes HTTP
colgados, aparecieron carpetas runtime y procesos reales también para
`t009`-`t012`. Por tanto, el fallo confirmado no fue solo ausencia de dispatch,
sino falta de respuesta HTTP/observabilidad durante un dispatch tardío. El
operador no puede distinguir si debe esperar, cortar el cliente o reintentar sin
riesgo de duplicar.

## Actualización 2026-06-27 14:46: ola limpia 19023i

Tras descartar la ola contaminada `19023h`, OPES lanzó payloads limpios
`19023i` para Integración Social B, temas `t013` a `t018`.

Observado:

- `POST /api/v0/external-work/run` respondió HTTP 200 para los seis runs.
- El registro OPES quedó con seis temas en estado `en_orquesta`.
- `POST /api/v0/runs/supervise` con cuerpo `{}` quedó abierto más de 180
  segundos sin devolver JSON al cliente.
- Durante ese bloqueo sí se materializó runtime real para
  `run-opes-integracion-social-b-t013-rework-productivo-20260627-19023i`, con
  padre y subroles `fuentes`, `reutilizacion`, `redaccion`, `visuales`,
  `tests-tutor` y `html-rag-audio-qa`.
- En el momento de cortar el cliente HTTP, no había evidencia equivalente para
  `t014`-`t018`.

Impacto operativo:

- El cliente `curl` de supervisión tuvo que interrumpirse para no dejar una
  sesión colgada, sin detener Orquesta ni los agentes ya lanzados.
- Orquesta vuelve a necesitar una respuesta HTTP acotada que distinga:
  `accepted`, `queued_not_dispatched`, `dispatch_started`, `running_live`,
  `waiting_external` y `supervise_timeout_no_body`.
- Si el supervisor decide despachar solo un run por ciclo, debe explicarlo en
  la respuesta y no mantener la conexión abierta sin cuerpo.

### Supervisión dirigida 014-018

Después de materializar solo `t013`, OPES lanzó cinco llamadas dirigidas a
`/api/v0/runs/supervise`, una por cada run `t014` a `t018`, con `run_ref`
explícito y timeout de cliente de 90 segundos.

Resultado:

```text
t014: timeout_or_error, sin JSON útil, sin runtime observado
t015: timeout_or_error, sin JSON útil, sin runtime observado
t016: timeout_or_error, sin JSON útil, sin runtime observado
t017: timeout_or_error, sin JSON útil, sin runtime observado
t018: timeout_or_error, sin JSON útil, sin runtime observado
```

El runtime visible de la ola `19023i` seguía siendo solo `t013`. Esto confirma
que el problema no es que falten `run_ref` explícitos en la llamada de OPES: aun
con supervisión dirigida, Orquesta puede dejar el cliente sin respuesta y sin
materializar la run solicitada.

### Run contaminado reaparece como `running`

`POST /api/v0/autoprogramming/status` mostró además
`run-opes-integracion-social-b-t014-rework-productivo-20260627-19023h` en
estado `running`, pese a que la ola `19023h` fue descartada por payload
contaminado y no había procesos Codex vivos de ese run.

Acción operativa desde OPES:

- Se verificó que no había procesos vivos para `19023h/t014`.
- Se llamó a `POST /api/v0/runs/control` con `action=stop`, `forced=true`,
  `run_ref=run-opes-integracion-social-b-t014-rework-productivo-20260627-19023h`.
- La API devolvió HTTP 200, `status=stop_requested`,
  `checkpoint_recorded=true` y evidencia
  `evidence-ref-mcp-run-control-checkpoint-recorded`.

Pendiente para Orquesta:

- Reconciliar runs contaminados/stale sin procesos vivos para que no vuelvan a
  competir con la ola limpia.
- Evitar que `autoprogramming/status` muestre como `running` una run sin
  procesos vivos ni ACK pendiente real.
- En `external-work/run`/supervisor, impedir que una ola descartada siga con
  prioridad superior a la ola de reemplazo limpia.

### `autoprogramming/supervise` tras cerrar t013

Después de cerrar `t013`, OPES llamó a la acción segura recomendada por
`autoprogramming/status`:

```text
POST /api/v0/autoprogramming/supervise
request_id=req-autoprogramming-supervise-integracion-social-19023i-after-t013-20260627
queue_ref=global
max_dispatches=24
max_outbox=42
include_process_refs=true
```

Resultado:

- El cliente agotó timeout de 90 segundos sin cuerpo JSON útil.
- Aun así, Orquesta materializó runtime para
  `run-opes-integracion-social-b-t016-rework-productivo-20260627-19023i`.
- En el momento de la observación, `t016` tenía seis directorios de agente y
  procesos Codex vivos; `t014`, `t015`, `t017` y `t018` seguían sin runtime
  visible.

Pendiente para Orquesta: `autoprogramming/supervise` debe devolver respuesta
controlada aunque haya despachado parcialmente. El operador necesita saber qué
runs fueron materializados, cuáles quedaron en cola y cuál es la próxima acción
segura.

### `autoprogramming/supervise` tras liberar t015

Fecha de observación: 2026-06-27T15:20:42+02:00.

Contexto operativo:

- La ola limpia `19023i` tenía `external-work/run` aceptado para `t014` y
  `t018`.
- Las respuestas de creación indicaban `estado=ok`, `external-work-run-queued`
  y `run_ref` correctos:
  `run-opes-integracion-social-b-t014-rework-productivo-20260627-19023i` y
  `run-opes-integracion-social-b-t018-rework-productivo-20260627-19023i`.
- El registro OPES quedó con `t014` y `t018` como `en_orquesta`, mientras
  `t013`, `t015`, `t016` y `t017` estaban liberados como
  `pendiente_integracion_qa`.

Acción ejecutada:

```text
POST /api/v0/autoprogramming/supervise
request_id=req-autoprogramming-supervise-integracion-social-19023i-after-t015-20260627
queue_ref=global
max_dispatches=24
max_outbox=42
include_process_refs=true
timeout_cliente=90s
```

Resultado:

- El cliente agotó timeout de 90 segundos sin cuerpo JSON útil
  (`supervise_code=timeout_or_error`).
- No se materializó runtime para `t014` ni para `t018`.
- `GET /api/v0/autoprogramming/status` devolvió `metodo_no_permitido`; al
  usar `POST /api/v0/autoprogramming/status` sin timeout, la llamada se quedó
  bloqueada y hubo que cortarla manualmente.
- `POST /api/v0/runs/queue/priority` también quedó bloqueado sin respuesta útil
  cuando se consultó sin timeout.

Pendiente para Orquesta:

- Las APIs de estado y cola deben ser siempre acotadas en tiempo y devolver una
  respuesta diagnóstica, aunque no puedan despachar.
- Debe existir una consulta fiable para saber si un `run_ref` aceptado está
  `queued`, `blocked_by_capacity`, `blocked_by_stale_run`, `waiting_supervisor`,
  `dispatching`, `running` o `failed_to_materialize`.
- Si `external-work/run` acepta una run pero no la materializa después de varias
  supervisiones, Orquesta debe exponer la causa y la acción segura: reintentar,
  reencolar, cancelar stale, reiniciar supervisor residente o reparar runtime.

### Estado de cola con `alive_percentage=0`

Fecha de observación: 2026-06-27T15:21:xx+02:00.

Con `curl --max-time 10 -X POST /api/v0/autoprogramming/status -d '{}'`, la
API llegó a devolver HTTP 200, pero después de agotar el timeout del cliente y
con salida parcial. El contenido útil mostró:

```text
queue_health.running_stale=6
efficiency_summary.overall_percentage=85
efficiency_summary.alive_percentage=0
operator.active_runs=6
recommended_action=supervise:queue
diagnostics=run_stats_required_for_safe_supervision
```

Runs listadas como `running` pese a no tener agentes vivos observados:

```text
run-opes-integracion-social-b-t014-rework-productivo-20260627-19023h
run-opes-integracion-social-b-t014-rework-productivo-20260627-19023i
run-opes-integracion-social-b-t015-rework-productivo-20260627-19023i
run-opes-integracion-social-b-t016-rework-productivo-20260627-19023i
run-opes-integracion-social-b-t017-rework-productivo-20260627-19023i
run-opes-integracion-social-b-t018-rework-productivo-20260627-19023i
```

Problema:

- `t015`, `t016` y `t017` ya tenían ACK y fueron liberadas por OPES, pero
  Orquesta seguía mostrándolas como `running`.
- `t014h` pertenecía a la ola contaminada descartada y reapareció como
  `running` pese a haberse pedido `stop`.
- `t014i` y `t018i` aparecían como `running` aunque no existía runtime
  materializado para esas refs.

Pendiente para Orquesta:

- El reconciliador de runs debe cerrar o degradar automáticamente las runs
  `running` sin procesos vivos ni ACK pendiente real.
- El `status` debe distinguir `running_live` de `running_stale`; no deben
  aparecer ambas situaciones bajo el mismo estado operativo.
- Cuando una run se marca `stop_requested` con checkpoint, no debe volver a
  encabezar la cola sin una causa explícita.

### `stop_requested` seguido de materialización tardía

Fecha de observación: 2026-06-27T15:23:xx+02:00.

OPES pidió `POST /api/v0/runs/control` con `action=stop`, `forced=true` para
varias runs stale o ya integradas. Resultados observados:

```text
t014h: HTTP 200, stop_requested, checkpoint_recorded=true
t015i: HTTP 200, stop_requested, checkpoint_recorded=true
t016i: timeout del cliente sin cuerpo
t017i: timeout del cliente sin cuerpo
t014i: HTTP 200, stop_requested, checkpoint_recorded=true
t018i: HTTP 200, stop_requested, checkpoint_recorded=true
```

Después de que `t014i` devolviera `stop_requested`, apareció runtime real en:

```text
run-opes-integracion-social-b-t014-rework-productivo-20260627-19023i
```

y se materializaron padre + seis subroles con procesos Codex vivos.

Decisión OPES:

- No se mataron esos agentes porque ya estaban produciendo trabajo útil y
  trazable.
- Se mantiene la observación desde OPES hasta ACK o bloqueo.

Pendiente para Orquesta:

- Un `stop_requested` aceptado no debe competir con una materialización tardía
  sin reconciliación explícita.
- Si el control llega cuando el despacho ya está en curso, la respuesta debe
  indicarlo como `stop_pending_but_dispatch_in_progress` o similar.
- El estado humano debe aclarar si el agente vivo prevalece, si se cancelará
  cooperativamente o si se debe conservar la entrega.

### Entrega útil sin `agent_ack.json` final

Fecha de observación: 2026-06-27T15:32:xx+02:00.

En `run-opes-integracion-social-b-t014-rework-productivo-20260627-19023i`,
Orquesta materializó padre + seis subroles. Cinco subroles acabaron con ACK
exacto:

```text
S1 fuentes: completed
S2 reutilización: completed
S4 visuales: completed
S5 tests/tutor: completed
S6 HTML/RAG/audio/QA: completed
```

Pero S3 redacción y el padre quedaron sin `agent_ack.json`, pese a dejar
artefactos y log final útil. En S3, el log terminó con:

```text
Validación focal lista: 4 artefactos existen, bloque publicable sin patrones
internos, ampliado existente 11.327 palabras > mínimo B 10.800. Escribo ACK de
control como última acción.
```

No quedaron procesos vivos para esa run. OPES decidió conservar el trabajo y
liberar el tema como `pendiente_integracion_qa`, no como `ready`, con nota de
causalidad pendiente.

Pendiente para Orquesta:

- Si el agente termina después de anunciar escritura de ACK, el wrapper debe
  verificar que `agent_ack.json` existe y, si no existe, emitir un estado
  `completed_without_ack_file` o `ack_write_failed` con evidencia.
- El padre no debe cerrar basándose en una matriz desactualizada cuando ya
  existen ACK hijos posteriores.
- La reconciliación debería reconstruir el estado desde archivos presentes,
  procesos vivos y últimos mensajes, evitando que una entrega útil quede como
  fallo opaco.
