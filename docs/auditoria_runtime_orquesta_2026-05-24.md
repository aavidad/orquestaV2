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
- `http_request`: metodo, ruta, query keys redactadas, estado HTTP, bytes,
  content-length, duracion e identidad de cliente redactada por categoria.

## Identidad de cliente HTTP

Desde el cierre T187 del 2026-05-26, `http_request` no persiste `RemoteAddr`
crudo, IP, puerto efimero ni valores de cabeceras `Forwarded`,
`X-Forwarded-For`, `X-Forwarded-Host` o `X-Real-IP`.

El payload publica `client_identity` con schema
`orquesta_http_client_identity.v0`, `category` (`loopback`, `private`,
`external` o `unknown`), `redaction=category_only`,
`raw_address_persisted=false`, `forwarded_header_policy=ignored_untrusted` y
`authorization_scope` derivado del bind efectivo (`loopback_bind` o
`remote_bind_opt_in`). Si hay cabeceras de forwarding, solo se guarda la lista
de nombres presentes; los valores se ignoran por defecto y no son identidad
real ni evidencia de autorizacion.

La decision de bind, permiso, token o acceso remoto sigue en el control plane
del servidor. Una peticion local no se vuelve remota segura ni autorizada por
cabeceras declaradas por el cliente.

Revalidacion 2026-05-27: el contrato sigue acotado a `client_identity`
redactada por categoria y a nombres de headers de forwarding presentes, sin
valores, IP:puerto ni `RemoteAddr` durable.

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

## Fallo de escritura

Desde el cierre T94 del 2026-05-26, si el sink JSONL no puede escribir, el
servidor no intenta registrar ese fallo en la misma auditoria rota. Lo proyecta
en el estado residente como `audit_status=degraded`, `audit_failures`,
`audit_last_code=audit_write_failed`, `audit_last_event`,
`audit_last_severity` y `audit_last_failed_at`.

La proyeccion es compacta y publica: no incluye rutas locales, HOME, permisos
exactos, payloads HTTP, prompts, transcripts, tokens ni el error crudo del sink.
La liveness puede seguir respondiendo; readiness y estado operativo pueden
degradar mientras la auditoria obligatoria siga fallando.

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
Tambien expone `audit_write_failed` cuando el sink de auditoria falla, separado
de `state_persist_failed` para no confundir auditoria rota con store de estado
roto.

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
- Definir por composicion si eventos concretos de auditoria degradada deben
  bloquear mutaciones externas ademas de degradar readiness.
