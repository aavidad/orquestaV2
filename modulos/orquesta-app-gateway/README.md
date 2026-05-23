# orquesta-app-gateway

Raiz de composicion HTTP de Orquesta.

Este modulo ensambla handlers ya existentes:

- web `/nueva-app`;
- web `/app-change`;
- web `/director-stats`;
- web `/run-control`;
- web `/run-queue`;
- REST `/api/v0/apps/spec`;
- REST `/api/v0/apps/director`;
- REST `/api/v0/apps/{app_ref}/changes`;
- REST `/api/v0/director/stats`;
- REST `/api/v0/runs/control`;
- REST `/api/v0/runs/queue/priority`;
- REST `/api/v0/runs/supervise`;
- REST `/api/v0/autoprogramming/validate-request`;
- REST `/api/v0/autoprogramming/prepare-run`;
- REST `/api/v0/autoprogramming/status`;
- REST `/api/v0/autoprogramming/supervise`;
- REST `/api/v0/server/shutdown`.
- REST `/api/v0/domain-work`;
- REST `/api/v0/external-work/run`.

No es servidor real y no abre sockets. Devuelve un `http.Handler` listo para que
otro adaptador superior lo sirva.

Fuera de alcance:

- DB, migraciones, drivers o stores concretos;
- runtime real, procesos, Codex, HOME, OAuth o modelos;
- `cmd`, flags, env, `ListenAndServe`;
- logica de negocio de director, scheduler, factory, cola o web.

Validacion local:

```bash
go test -count=1 ./modulos/orquesta-app-gateway
git diff --check -- modulos/orquesta-app-gateway
```
