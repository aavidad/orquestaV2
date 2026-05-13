# Pruebas

- `TestShutdownServerV0SolicitaStopDrenaYQuedaReady`
- `TestShutdownServerV0NoDrenaSiFaltaCheckpoint`
- `TestShutdownServerV0PreparaCheckpointAntesDeStopNoForzado`
- `TestShutdownServerV0SinRunsQuedaReady`

Evidencia esperada: el caso de uso solicita stop por run, ejecuta supervisor
cuando procede, registra checkpoint por puerto en modo no forzado y calcula
`shutdown_ready` sin leer runtime ni filesystem.
