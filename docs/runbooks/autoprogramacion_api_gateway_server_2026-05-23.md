# Autoprogramacion API gateway y servidor - 2026-05-23

## Alcance

Este runbook valida el contrato externo de autoprogramacion servido por el
gateway HTTP y el servidor residente. No define producto nuevo ni mete logica de
nucleo en HTTP: las rutas solo transportan requests hacia puertos inyectados.

Write-set del corte:

- `modulos/orquesta-http-gateway`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-server`
- `cmd/orquesta-server`

## Contrato publico

Rutas REST estables expuestas por el gateway:

- `POST /api/v0/autoprogramming/validate-request`
- `POST /api/v0/autoprogramming/self-improvement`
- `POST /api/v0/autoprogramming/prepare-run`
- `POST /api/v0/director/human-work/review-plan`
- `POST /api/v0/apps/director`
- `POST /api/v0/apps/{app_ref}/changes`
- `POST /api/v0/director/stats`
- `POST /api/v0/runs/control`
- `POST /api/v0/runs/queue/priority`
- `POST /api/v0/runs/supervise`
- `POST /api/v0/server/shutdown`
- `POST /api/v0/domain-work`
- `POST /api/v0/external-work/run`

Rutas web compatibles:

- `GET|POST /nueva-app`
- `GET|POST /app-change`
- `GET /director-stats`
- `GET|POST /run-control`
- `GET /run-queue`

Rutas propias del servidor residente:

- `GET /healthz`
- `GET /api/status`
- `GET /api/v0/server/status`

## Fronteras

- `orquesta-http-gateway` registra solo handlers inyectados; no importa web,
  MCP, runtime, DB, Codex, OPES ni `cmd`.
- `orquesta-app-gateway` compone web + REST/MCP por puertos; no abre sockets,
  no lee env y no crea proveedores reales.
- `orquesta-server` sirve estado residente y delega el resto en el handler de
  aplicacion; supervisor, startup y state store entran por puertos.
- `cmd/orquesta-server` es el borde opt-in para Codex, state-file, domain-work
  file/HTTP, OPES y runner de tests requeridos.
- Shutdown y run-control son opt-in por rutas/puertos; no tocan OPES productivo
  salvo configuracion explicita del bridge.

## Validacion del corte

Ejecutar desde la raiz del repo:

```bash
go test -count=1 ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-server ./cmd/orquesta-server
```

Criterios de aceptacion manual:

- Gateway sin logica de nucleo ni imports prohibidos.
- `cmd/orquesta-server` queda como wiring fino del servidor real.
- Las rutas web, CLI/MCP y REST usan los mismos contratos publicos.
- `domain-work` y `external-work/run` aceptan refs opacas; el dominio externo
  conserva validadores, persistencia y ensamblado.
- No reaparecen el control-plane heredado ni helpers locales de DB.

## Resultado 2026-05-23

Validado con la bateria focal obligatoria del paquete:

- `go test -count=1 ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-server ./cmd/orquesta-server`
- `validar criterios de aceptacion del cambio`
- `validar contrato externo de dominio`
