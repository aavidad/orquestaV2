# orquesta-server

Servidor residente de Orquesta.

Este modulo permite ejecutar Orquesta como proceso independiente de la consola
que lo lanza. Expone `healthz`, `/api/status` y delega el resto del trafico al
handler de aplicacion inyectado.

El supervisor global se ejecuta por pulsos acotados mediante un puerto. Si el
usuario cierra la sesion de Codex, el daemon sigue vivo y el operador puede
reengancharse leyendo el statefile y consultando la API.

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
