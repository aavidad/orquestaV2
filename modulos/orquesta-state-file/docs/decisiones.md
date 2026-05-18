# Decisiones

## STF-001: conector file-based

El estado operativo durable entra como conector file-based para evitar acoplar el
servidor o el nucleo a una base de datos concreta. Backends futuros deben
implementar los mismos puertos.

## STF-002: JSON por agregado

Cada agregado se guarda en un documento JSON separado. Las escrituras usan
temporal, `sync`, `rename` y sincronizacion del directorio.

## STF-003: claims de outbox no sobreviven sin ACK

Un `claim` de outbox es una reserva local de ejecucion, no una verdad durable.
Si el servidor cae despues de reclamar un mensaje y antes de registrar `ack`, el
adaptador de fichero debe reabrirlo al cargar el ledger para que el dispatcher
lo reintente idempotentemente. Un `ack` si es durable y cierra el mensaje.

Esto evita que `StopRuntimeAgent` o `LaunchRuntimeAgent` queden bloqueados tras
reinicio, que fue uno de los patrones de bucle detectados en pruebas reales
OPES-Orquesta.

## STF-004: waits del Director como agregado propio

`WorkflowTaskWaitStateV0` se persiste separado de `runs/` y
`workflow_tasks/`. El run conserva refs compactas y las tasks conservan metadata
operativa; el wait state conserva la observacion viva del Director: causa,
scope, agentes objetivo, agentes pendientes, intento y deadline. Un replay de
eventos no debe fingir que puede reconstruir esta foto sin task store o sin una
rematerializacion idempotente.
