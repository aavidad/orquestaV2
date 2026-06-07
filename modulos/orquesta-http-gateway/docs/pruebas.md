# Pruebas: orquesta-http-gateway

## Comandos

```bash
go test -count=1 ./modulos/orquesta-http-gateway
git diff --check -- modulos/orquesta-http-gateway
```

## Cobertura local

- Registro de cada ruta configurada.
- Preservacion de path y metodo hacia el handler inyectado.
- 404 para rutas no configuradas.
- Barrera de arquitectura para imports productivos prohibidos.
- Manifiesto canonico de rutas: duplicados exactos, prefijos duplicados, shadows
  no declarados y paridad entre rutas registradas y manifest.
- Cobertura de ruta web `/director-stats` separada de API
  `/api/v0/director/stats`.
- Cobertura de ruta web `/run-control` separada de API
  `/api/v0/runs/control`.
- Cobertura de ruta web `/run-queue` separada de API
  `/api/v0/runs/queue/priority`.
- La auditoria de stats se limita al routing: el contenido de
  `DirectorRunStatsV0` pertenece al handler inyectado, no al gateway HTTP fino.
- `PublicRouteMutabilityV0` clasifica las rutas de supervision
  `/api/v0/runs/supervise` y `/api/v0/autoprogramming/supervise` como
  mutaciones, y el guard browser bloquea origen cruzado antes de delegar.
