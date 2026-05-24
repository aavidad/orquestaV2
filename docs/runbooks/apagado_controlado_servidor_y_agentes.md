# Apagado controlado de servidor y agentes

Estado: runbook operativo v0. Documenta el cierre controlado disponible y el
protocolo cooperativo de checkpoint para agentes Codex arrancados con prompt
compatible.

## Principio

Parar el servidor no equivale a cerrar de forma segura el trabajo de los
agentes. El cierre seguro debe tratar primero los runs activos y despues parar
el proceso servidor.

Orden correcto:

1. bloquear o evitar nuevas solicitudes;
2. pedir parada de cada run activo por RunControl;
3. esperar parada confirmada de agentes;
4. parar el servidor.

Autoridad: solo el Director puede iniciar shutdown. Los agentes no solicitan
apagado; reciben la orden, preparan checkpoint/ACK y cierran dentro del plazo
cooperativo.

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
- solicita `stop` por `RunControlWriterPortV0`;
- si `forced=false`, esa solicitud deja la run en `stop_requested` para
  bloquear trabajo nuevo y despues intenta preparar checkpoint por
  `PrepareAgentShutdownPortV0`;
- registra ACK durable por `RunControlCheckpointWriterPortV0` cuando existe;
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

## Checkpoint cooperativo

Para agentes Codex arrancados con el prompt actual, el stack usa un protocolo
por ficheros de control:

1. `PrepareAgentShutdownPortV0` lee stats y descriptors del run por puertos.
2. Si hay agentes en vuelo, escribe `orquesta_shutdown_request.json` en el
   runtime dir de cada agente.
3. El agente debe detectar esa request antes de bloques largos, parar en punto
   consistente y escribir `agent_shutdown_checkpoint_ack.json`.
4. Orquesta valida `codex_shutdown_checkpoint_ack.v0` y exige correlacion de
   `run_ref`, `agent_ref` y `checkpoint_ref`.
5. Solo cuando todos los agentes en vuelo han respondido se llama a
   `RecordRunCheckpointV0`.
6. Despues el supervisor puede drenar stop/outbox y confirmar parada.

Si falta descriptor, runtime dir o ACK valido, el endpoint devuelve
`waiting_checkpoint`, `checkpoint_agents_pending`, `pending_checkpoint_agent_refs`
por run y `checkpoint_evidence_refs` compactas. No se inventa checkpoint.

Modos operativos:

- `forced=true`: drena y detiene agentes sin exigir checkpoint previo;
- `forced=false`: exige checkpoint durable por ACK cooperativo o queda en
  `waiting_checkpoint`.

## Regla operativa

Para trabajo importante, antes de parar:

- usar `POST /api/v0/server/shutdown forced=false`;
- comprobar si quedan checkpoints pendientes;
- si un agente no responde, decidir si se espera, se pide intervencion al
  director o se fuerza el cierre;
- no asumir continuidad si no existe ACK/checkpoint durable.

## Mejora pendiente

Queda elevar el protocolo a todos los runtimes/proveedores, no solo Codex
cooperativo por prompt:

- conectores equivalentes para Claude, Gemini, agentes API o runtimes locales;
- politica de deadline por agente;
- exposicion REST/MCP/web de `pending_agent_refs` y razones sanitizadas;
- parada via runtime solo despues de ACK o deadline controlado.

La secuencia objetivo completa queda:
quiesce, prepare checkpoint, ACK durable, stop, confirmacion y cierre final.
