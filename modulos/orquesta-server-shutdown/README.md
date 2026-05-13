# orquesta-server-shutdown

Caso de uso para coordinar un apagado controlado del servidor Orquesta sin
acoplarse al runtime ni a la persistencia concreta.

El modulo no mata procesos. Solicita parada de runs por `RunControl`, ejecuta
ticks de supervisor para que el nucleo drene `StopRuntimeAgent` por sus puertos
inyectados y verifica si cada run queda listo para apagar el proceso servidor.

Validacion local:

```bash
go test -count=1 ./modulos/orquesta-server-shutdown
```
