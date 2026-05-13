# Pruebas

- `TestShutdownServerV0SolicitaStopDrenaYQuedaReady`
- `TestShutdownServerV0NoDrenaSiFaltaCheckpoint`
- `TestShutdownServerV0SinRunsQuedaReady`

Evidencia esperada: el caso de uso solicita stop por run, ejecuta supervisor
cuando procede y calcula `shutdown_ready` sin leer runtime ni filesystem.
