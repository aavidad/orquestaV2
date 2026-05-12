# Apagado controlado de servidor y agentes

Estado: runbook operativo v0. Documenta lo que Orquesta hace hoy y el hueco
pendiente para un cierre graceful completo.

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

Cuando todos los runs activos estan drenados:

```bash
orquesta-server stop
```

El comando envia `os.Interrupt` al proceso servidor. El runtime del servidor:

- cierra el HTTP server con `Shutdown`;
- persiste estado `stopped`;
- termina el loop supervisor.

Este paso no debe usarse como sustituto del cierre de runs.

## Limitacion actual

Orquesta todavia no tiene implementado un protocolo completo de checkpoint
antes de apagar agentes.

Existe estado `CheckpointRecorded` en RunControl, pero falta la operacion
end-to-end:

- enviar orden `prepare_shutdown` o equivalente al agente;
- pedir resumen/checkpoint de continuidad;
- recibir ACK durable del checkpoint;
- esperar hasta deadline;
- parar proceso solo despues del ACK o por timeout controlado.

Por tanto, hoy hay dos modos:

- `forced=true`: drena y detiene agentes sin exigir checkpoint previo;
- `forced=false`: puede requerir checkpoint, pero no hay todavia pipeline
  completo para solicitarlo y verificarlo automaticamente.

## Regla operativa hasta cerrar el hueco

Para trabajo importante, antes de parar:

- pedir al director o al operador que fuerce una entrega/checkpoint documental;
- comprobar stats y artefactos;
- despues ejecutar `POST /api/v0/runs/control action=stop forced=true`;
- no asumir continuidad si no existe ACK/checkpoint durable.

## Mejora pendiente

Implementar `orquesta.server.shutdown.v0` como caso de uso hexagonal:

- puerto inbound REST/MCP para shutdown;
- puerto de lectura de runs activos;
- puerto `PrepareAgentShutdownPortV0`;
- puerto `CheckpointStorePortV0`;
- politica de deadline por agente;
- parada via `StopAgentPortV0`;
- stats de cierre por run y agente;
- modo `graceful` y modo `forced`.

El servidor debe aceptar una orden unica de apagado y coordinar la secuencia:
quiesce, checkpoint, stop, confirmacion y cierre final.
