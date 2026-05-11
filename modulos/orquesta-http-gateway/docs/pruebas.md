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
- Cobertura de ruta web `/director-stats` separada de API
  `/api/v0/director/stats`.
- La auditoria de stats se limita al routing: el contenido de
  `DirectorRunStatsV0` pertenece al handler inyectado, no al gateway HTTP fino.
