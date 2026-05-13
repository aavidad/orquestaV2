# Apagado controlado de servidor y agentes

Estado: runbook operativo v0. Documenta el cierre controlado disponible y el
hueco pendiente para un protocolo graceful con checkpoint completo.

## Principio

Parar el servidor no equivale a cerrar de forma segura el trabajo de los
agentes. El cierre seguro debe tratar primero los runs activos y despues parar
el proceso servidor.

Orden correcto:

1. bloquear o evitar nuevas solicitudes;
2. pedir parada de cada run activo por RunControl;
3. esperar parada confirmada de agentes;
4. parar el servidor.

## Cierre de un run

Entrada canonica REST/MCP:

```http
POST /api/v0/runs/control
```

Payload minimo para parada controlada forzada:

```json
{
  "action": "stop",
  "run_ref": "run-ref",
  "requested_by": "operator",
  "reason": "apagado controlado",
  "forced": true,
  "idempotency_key": "idem-shutdown-run-ref"
}
```

Efecto esperado:

- `RunControl` deja el run en `stop_requested`;
- el coordinador/drainer no agenda trabajo nuevo;
- Orquesta materializa `StopAgent`;
- se genera outbox `StopRuntimeAgent`;
- el runtime llama `StopV0(process_ref)`;
- el proceso recibe `os.Interrupt`;
- si el runtime confirma parada, el core registra `AgentStopConfirmed`;
- stats debe terminar con `agents_in_flight=0`.

Verificacion por stats:

```http
POST /api/v0/director/stats
```

Condiciones de cierre del run:

- `agents_stop_confirmed == agents_stop_requested`;
- `agents_in_flight == 0`;
- no hay procesos runtime vivos asociados al run;
- si hay outbox pendiente de stop, el run no se considera cerrado.

## Cierre del servidor

Entrada canonica de cierre de servidor:

```http
POST /api/v0/server/shutdown
```

Payload minimo para cierre operativo forzado:

```json
{
  "requested_by": "operator",
  "reason": "apagado controlado",
  "forced": true,
  "idempotency_key": "idem-server-shutdown"
}
```

Efecto esperado:

- el caso de uso lista runs activos por `RunQueueReaderPortV0`;
- si `forced=false`, intenta preparar checkpoint por
  `PrepareAgentShutdownPortV0` y registra ACK durable por
  `RunControlCheckpointWriterPortV0`;
- solicita `stop` por `RunControlWriterPortV0`;
- ejecuta el supervisor global para drenar stop/outbox/confirmaciones;
- lee stats de cierre por run;
- devuelve `shutdown_ready=true` solo si no quedan agentes en vuelo ni
  checkpoints pendientes en los runs objetivo.

Respuesta relevante:

```json
{
  "estado": "ok",
  "status": "ready",
  "shutdown_ready": true,
  "runs_requested": 1,
  "runs_stopped": 1,
  "agents_in_flight": 0,
  "checkpoints_pending": 0
}
```

Cuando el endpoint devuelve `shutdown_ready=true`:

```bash
orquesta-server stop
```

El comando CLI llama primero a `/api/v0/server/shutdown` con `forced=true`.
Solo si el endpoint responde `shutdown_ready=true` envia `os.Interrupt` al
proceso servidor. El runtime del servidor:

- cierra el HTTP server con `Shutdown`;
- persiste estado `stopped`;
- termina el loop supervisor.

Este paso no debe usarse como sustituto de stats ni de RunControl; el endpoint
es la frontera unica de cierre coordinado.

## Limitacion actual

Orquesta todavia no tiene implementado un protocolo completo de checkpoint
antes de apagar agentes.

Existe `RecordRunCheckpointV0` en RunControl y `PrepareAgentShutdownPortV0` en
shutdown. El stack actual lo usa de forma conservadora: si no hay agentes en
vuelo puede registrar checkpoint y continuar; si hay agentes vivos devuelve
`waiting_checkpoint`.

Falta la operacion interactiva end-to-end para agentes vivos:

- enviar orden `prepare_shutdown` o equivalente al agente;
- pedir resumen/checkpoint de continuidad;
- recibir ACK durable del checkpoint;
- esperar hasta deadline;
- parar proceso solo despues del ACK o por timeout controlado.

Por tanto, hoy hay dos modos:

- `forced=true`: drena y detiene agentes sin exigir checkpoint previo;
- `forced=false`: exige checkpoint durable; hoy queda en `waiting_checkpoint`
  si hay agentes vivos porque falta conector interactivo fiable.

## Regla operativa hasta cerrar el hueco

Para trabajo importante, antes de parar:

- pedir al director o al operador que fuerce una entrega/checkpoint documental;
- comprobar stats y artefactos;
- despues ejecutar `POST /api/v0/runs/control action=stop forced=true`;
- no asumir continuidad si no existe ACK/checkpoint durable.

## Mejora pendiente

`orquesta.server.shutdown.v0` ya existe como caso de uso hexagonal para
checkpoint conservador, stop, drain y stats. Falta elevarlo a protocolo
graceful completo para agentes vivos:

- conector real de `PrepareAgentShutdownPortV0` por runtime/proveedor;
- politica de deadline por agente;
- orden `prepare_shutdown` antes de stop cuando `forced=false`;
- ACK durable de checkpoint;
- parada via runtime solo despues de ACK o deadline controlado;
- stats de cierre por run y agente con razon de cierre.

La secuencia objetivo completa queda:
quiesce, prepare checkpoint, ACK durable, stop, confirmacion y cierre final.
