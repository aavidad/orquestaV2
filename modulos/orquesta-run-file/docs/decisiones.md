# Decisiones

## Snapshots separados

Cada puerto escribe su propio JSON. Esto evita acoplar app-change a la cola y
mantiene las responsabilidades del adaptador pequenas.

## Escritura atomica

Cada mutacion escribe un fichero temporal en el mismo directorio y publica con
`os.Rename`. El estado en memoria solo se sustituye si la escritura termina sin
error.

## Paridad con memoria

El adaptador replica las normalizaciones y copias defensivas de
`orquesta-run-memory` e `InMemoryAppChangeStoreV0` para que el cambio de
conector no altere la capa de aplicacion.

## Recarga tras compactacion externa

El servidor puede compactar snapshots durante el autodiagnostico de arranque.
Cuando eso ocurre, `ReloadFromDiskV0` permite sincronizar el estado en memoria
con los JSON compactados sin crear otro adaptador ni saltarse los puertos en el
flujo normal.
