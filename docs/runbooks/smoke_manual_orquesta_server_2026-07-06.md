# Smoke manual Orquesta server 2026-07-06

Objetivo: validar el binario compilado local antes de pruebas mas amplias, sin
tocar remoto ni produccion.

Binario probado:

- `/tmp/orquesta-builds/orquesta-server-221a4f07d`
- sha256:
  `da86ac68d25be8405b094a7561710e3607baca4bf1a9b4ca643d20ea79f8f541`

Entorno:

- `ORQUESTA_SERVER_ADDR=127.0.0.1:0`
- `ORQUESTA_SERVER_STATE_DIR=/tmp/orquesta-manual-smoke-final-u8Vhtu/state`
- `ORQUESTA_CODEX_PROJECT_WORKDIR=/tmp/orquesta-manual-smoke-final-u8Vhtu/project`
- `ORQUESTA_CODEX_RUNTIME_WORKDIR=/tmp/orquesta-manual-smoke-final-u8Vhtu/runtime`
- `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DISABLED=true`
- `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false`
- `ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED=false`

Resultado final:

- `GET /healthz`: HTTP 200, `{"status":"ok"}`
- `GET /api/status`: HTTP 200, `status=running`,
  `addr=127.0.0.1:36887`
- `GET /api/v0/server/readiness`: HTTP 200, `ready=true`,
  `startup_status=startup_ready`, `diagnostics=[]`
- `POST /api/v0/server/shutdown`: HTTP 200, `status=ready`,
  `shutdown_ready=true`, `runs_stopped/runs_requested=0/0`,
  `agents_in_flight=0`
- Proceso cerrado limpio: `SERVER_EXITED_CLEAN=1`
- Sin procesos residuales del binario probado al cierre.

Hallazgos durante la prueba:

- Sin `ORQUESTA_CODEX_GOAL_BACKEND`, readiness devuelve HTTP 503 con
  diagnostico esperado `external_work_goal_backend_required`; no es crash.
- El endpoint de shutdown exige `idempotency_key` y autoridad de Director
  (`requested_by=orquesta-director`). Payloads manuales sin esos campos
  devolvieron `idempotency_key_requerida` y `requester_not_authorized`.
- `scripts/lib/smoke_common.sh` tenia el payload comun desactualizado: enviaba
  shutdown sin `idempotency_key`. Cerrado en
  `BUG-ORQ-20260706-SMOKE-SHUTDOWN-PAYLOAD-STALE`.

Verificacion posterior al fix del helper:

```bash
go test -count=1 ./cmd/orquesta-server -run 'TestSmokeCommonShutdownCleanupEnviaContratoDirectorV0|TestSmokeCommonShutdownCleanupMataAppServerPropioSinBaseURLV0|TestSmokeCommonReadiness'
go test -count=1 ./cmd/orquesta-server
git diff --check
```

Estado: smoke local de arranque/readiness/shutdown verde. No se lanzaron goals
reales ni consumo de proveedor.

## Revalidacion post-commit e1a0fa4ef

Tras commitear el fix del helper de shutdown, se ejecuto el corte prudente
solicitado: test completo, build nuevo y smoke local del binario actualizado.

Verificacion:

```bash
go test -count=1 ./...
go build -trimpath -o /tmp/orquesta-builds/orquesta-server-e1a0fa4ef ./cmd/orquesta-server
sha256sum /tmp/orquesta-builds/orquesta-server-e1a0fa4ef
```

Resultado:

- `go test -count=1 ./...`: verde.
- Binario: `/tmp/orquesta-builds/orquesta-server-e1a0fa4ef`
- sha256:
  `49ef50c2b63dcb10aabf4d28f63fea11bda43a4e202902fe1a981b64f4788bae`
- Smoke local aislado:
  - runtime: `/tmp/orquesta-manual-smoke-e1a0fa4ef-M8wPCq`
  - addr: `127.0.0.1:42923`
  - `GET /healthz`: HTTP 200
  - `GET /api/status`: HTTP 200, `status=running`
  - `GET /api/v0/server/readiness`: HTTP 200, `ready=true`,
    `startup_status=startup_ready`, `diagnostics=[]`
  - `POST /api/v0/server/shutdown`: HTTP 200, `status=ready`,
    `shutdown_ready=true`
  - cierre: `SERVER_EXITED_CLEAN=1`

Estado: nucleo compilable y smoke de servidor local verde en `e1a0fa4ef`.
Siguiente prueba no cubierta aqui: goal real minimo con proveedor
`app_server_tmux`.
