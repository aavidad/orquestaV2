# orquesta-http-gateway

Modulo hexagonal fino para componer rutas HTTP de Orquesta con handlers
inyectados.

Responsabilidades:

- exponer constantes de rutas HTTP estables;
- registrar un `http.Handler` por ruta solo cuando se inyecta;
- preservar path, metodo y request original hacia el handler inyectado;
- devolver 404 para rutas no configuradas.

Rutas v0:

- `/nueva-app`
- `/app-change`
- `/director-stats`
- `/run-control`
- `/run-queue`
- `/api/v0/apps/spec`
- `/api/v0/apps/director`
- `/api/v0/apps/director/goal/observe`
- `/api/v0/apps/{app_ref}/changes`
- `/api/v0/director/stats`
- `/api/v0/autoprogramming/validate-request`
- `/api/v0/autoprogramming/self-improvement`
- `/api/v0/autoprogramming/prepare-run`
- `/api/v0/autoprogramming/status`
- `/api/v0/autoprogramming/supervise`
- `/api/v0/governance/catalog/query`
- `/api/v0/director/human-work/review-plan`
- `/api/v0/runs/control`
- `/api/v0/runs/queue/priority`
- `/api/v0/runs/supervise`
- `/api/v0/server/shutdown`
- `/api/v0/domain-work`
- `/api/v0/external-work/run`

Fuera de alcance:

- crear handlers de negocio;
- conocer web, MCP, factory, runtime, DB, cmd o Codex;
- resolver dependencias concretas o proveedores.

Validacion local:

```bash
go test -count=1 ./modulos/orquesta-http-gateway
git diff --check -- modulos/orquesta-http-gateway
```
