# orquesta-app-gateway

Raiz de composicion HTTP de Orquesta.

Este modulo ensambla handlers ya existentes:

- web `/nueva-app`;
- web `/director-stats`;
- REST `/api/v0/apps/spec`;
- REST `/api/v0/apps/director`;
- REST `/api/v0/director/stats`.

No es servidor real y no abre sockets. Devuelve un `http.Handler` listo para que
otro adaptador superior lo sirva.

Fuera de alcance:

- DB, migraciones, drivers o stores concretos;
- runtime real, procesos, Codex, HOME, OAuth o modelos;
- `cmd`, flags, env, `ListenAndServe`;
- logica de negocio de director, scheduler, factory o web.

Validacion local:

```bash
go test -count=1 ./modulos/orquesta-app-gateway
git diff --check -- modulos/orquesta-app-gateway
```
