# orquesta-director-candidates

Microproyecto puro para construir `SchedulableWorkCandidateV0` del scheduler desde un DTO compacto y explicito.

Responsabilidad v0:

- recibir refs opacas del trabajo, comandos e idempotency keys;
- recibir claims completos o claims compactos con scopes explicitos;
- construir subcandidates de capacidad, gate y agente;
- derivar un plan neutral de consejo desde complejidad compacta;
- validar el candidate contra contratos publicos antes de devolverlo.

Fuera de alcance:

- persistencia, red, runtime, proveedor, modelo, HOME, OAuth o credenciales;
- lectura de estado durable;
- decision de scheduling o ejecucion de comandos.

Validacion local:

```sh
go test -count=1 ./modulos/orquesta-director-candidates
```
