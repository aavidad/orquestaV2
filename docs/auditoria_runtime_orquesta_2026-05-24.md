# Auditoria runtime Orquesta - 2026-05-24

Estado: activo en el servidor residente.

## Log principal

El servidor escribe auditoria JSONL en:

```text
${ORQUESTA_SERVER_STATE_DIR}/audit/orquesta_server_audit_v0.jsonl
```

En la composicion local actual:

```text
/home/alberto/Trabajo/.orquesta-control/orquesta/state/audit/orquesta_server_audit_v0.jsonl
```

Cada linea es un evento JSON independiente con `schema_version`,
`event`, `occurred_at`, `status`, `error` y `payload`.

## Eventos cubiertos

- `startup_check_start`, `startup_check_ready`, `startup_check_blocked`,
  `startup_check_error`: arranque, purga logica y bloqueos de startup.
- `supervisor_tick_start`, `supervisor_tick_result`,
  `supervisor_tick_error`, `supervisor_tick_panic`: cada tick residente del
  supervisor global.
- `idle_self_improvement_check`, `idle_self_improvement_scheduled`,
  `idle_self_improvement_plan_result`, `idle_self_improvement_prepare_start`,
  `idle_self_improvement_prepare_result`,
  `idle_self_improvement_prepare_error`,
  `idle_self_improvement_prepare_batch`: ciclo de automejora idle.
- `http_request`: metodo, ruta, query, estado HTTP, bytes, content-length,
  duracion y origen remoto de cada peticion al servidor.

## Diagnosticos de drain

`RunDrainResultV0` y `RunExecutionSummaryV0` conservan `diagnostics`.
Cuando el supervisor ejecuta una run, el JSONL incluye:

- `drain_attempt`: estado del intento, pasos ejecutados y outbox pendiente.
- `director_burst`: acciones del director dentro del intento.
- `outbox_batch_dispatch`: target port, tipo de mensaje, planificados,
  ACKs, pendientes, fallidos e issues.
- `outbox_dispatch`: target port, tipo de mensaje, message id, dispatch ref e
  issues.
- `external_wait`: esperas externas.
- `drain_final`: estado final del drain con pending refs.

Esto evita que estados agregados como `wait_unhandled_outbox` o `blocked`
queden sin detalle operacional.

## Timeline global

Desde el 2026-05-25 la auditoria queda disponible como una fuente declarable de
`WorkspaceTimelineQueryV0`. La timeline global no lee el JSONL por su cuenta ni
recompone shell: consume un puerto de observabilidad y marca `audit` como
`not_available` cuando la composicion no inyecta adaptador de lectura.

## Evidencia local

El arranque con la version nueva dejo eventos de startup y peticiones HTTP.
El supervisor residente dejo ejecuciones reales con diagnostics para runs de
autoprogramacion, por ejemplo:

- `request-ref-autoprogramming-backlog-t09-local-sensitive-data-sanitizer-5f3fb8a9`
- `request-ref-autoprogramming-backlog-t10-flaky-tests-observability-ec93fa54`
- `request-ref-autoprogramming-backlog-t11-rails-blandos-y-falsos-positivos-2e7401b7`
- `request-ref-autoprogramming-backlog-t12-opes-consumer-smoke-real-opt-in-4c9edcf7`

El estado actual ya expone tambien errores concretos como
`transicion_invalida: payload.agent_request_id`, sin convertirlos en un mensaje
opaco.

## Configuracion

- `ORQUESTA_SERVER_AUDIT_FILE`: nombre del fichero `.jsonl` dentro de
  `${ORQUESTA_SERVER_STATE_DIR}/audit/`.
- `ORQUESTA_SERVER_AUDIT_DISABLED=true`: desactiva la auditoria del servidor
  solo si se necesita en pruebas especificas.

## Pendiente

- Anadir visor web filtrable sobre el JSONL: fecha, event, run_ref, outcome,
  status, target_port y message_type.
- Anadir rotacion/retencion configurable para produccion.
- Anadir captura local opt-in de payloads HTTP completos si hace falta depurar
  cuerpos de peticiones. Por defecto solo se registra metadata para no duplicar
  cuerpos grandes ni convertir la auditoria en persistencia paralela.
