# orquesta-server

Servidor residente de Orquesta.

Este modulo permite ejecutar Orquesta como proceso independiente de la consola
que lo lanza. Expone `/healthz` como liveness del proceso,
`/api/v0/server/readiness` como readiness operativa,
`/api/v0/server/status` como estado publico y delega el resto del trafico al
handler de aplicacion inyectado. `/api/status` queda solo como alias legacy con
headers de deprecacion/canonical.

El supervisor global se ejecuta por pulsos acotados mediante un puerto. Si el
usuario cierra la sesion de Codex, el daemon sigue vivo y el operador puede
reengancharse leyendo el statefile y consultando la API.

## Uso local y supervision

Para trabajo local, arranca el residente desde la raiz con
`go run ./cmd/orquesta-server run`. `/healthz` solo confirma socket/proceso
vivo; antes de lanzar Codex, OPES, `domain_work` o automejora usa
`GET /api/v0/server/readiness` y exige `ready=true`. La supervision operativa
debe hacerse contra las APIs publicas del servidor, por ejemplo
`GET /api/v0/server/status`, `POST /api/v0/autoprogramming/status` y
`POST /api/v0/autoprogramming/supervise`; el supervisor avanza por pulsos
acotados y no debe sustituirse por acceso directo a stores, runtime, filesystem
o procesos internos.

La composicion productiva configura presupuesto de progreso para agentes Codex
por entorno:

- `ORQUESTA_CODEX_MAX_EXPECTED_SECONDS`: tiempo esperado antes de pedir decision.
- `ORQUESTA_CODEX_NO_ACTIVITY_SECONDS`: ventana sin actividad antes de marcar
  riesgo operativo.
- `ORQUESTA_CODEX_STALLED_TICKS`: muestras sin cambio compacto antes de pedir
  atencion del director. Por defecto son 300 ticks con intervalo de 2s.
- `ORQUESTA_CODEX_LOOP_TICKS`: muestras repetidas antes de marcar posible
  bucle. Por defecto son 300 ticks con intervalo de 2s.

Estos valores alimentan estadisticas de director y no pertenecen al nucleo.

La proyeccion viva de progreso cerrada por T210 se transporta desde
`DirectorRunStatsV0`: entregas, agentes iniciados y proceso registrado pueden
elevar `percent_complete`, pero `TasksClosed` permanece ligado a cierre/review
aceptada. El servidor residente no debe rellenar 0% por defecto si la fuente
trae una senal viva ni recalcular esa semantica desde runtime o filesystem.

## Bridge OPES residente

`cmd/orquesta-server run` puede arrancar el bridge OPES por opt-in:

```sh
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_OPES_BRIDGE_ENABLED=1 \
ORQUESTA_OPES_BRIDGE_CONFIRM=1 \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE=draft_content_block,generate_visual_asset,review_legal,review_pedagogical,review_quality,validate_topic,assemble_topic \
go run ./cmd/orquesta-server run
```

La secuencia de tipos es composicion OPES, no contrato del runtime residente.
Cada tick usa el loop generico y drena solo el primer tipo que siga pendiente.

## Guardian de autoprogramacion

La promocion residente puede exigir guardian externo por opt-in. En ese modo el
servidor consume `orquesta_guardian_result.v0`, trata bloqueos del guardian como
retryables y no sustituye el binario vivo si el candidato no queda promovido.
T208 queda reconciliado como umbrella historico; nuevos huecos deben abrir owner
focal.
