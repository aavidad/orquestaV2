# Deploy atomico local de orquesta-server

Fecha: 2026-07-09.

`scripts/orquesta_server_deploy.sh` prepara un despliegue local sin tocar
servidores remotos. Sincroniza un worktree por ref con fast-forward obligatorio,
compila desde un arbol exportado, calcula `sha256`, conserva backup del binario
previo, hace swap atomico por `mv`, arranca solo mediante
`scripts/orquesta_server_ctl.sh start` y deja recibo durable en el state dir.

Actualizacion 2026-07-09: el recibo tambien cubre fallos no manejados por fase
(`deploy_unhandled_failure`). La verificacion runtime ya no es opcional: `ctl
status` debe responder, y `status`, readiness o supervisor deben exponer un
`sha256` del binario vivo que coincida con el instalado. Si se configura una URL
de verificacion, una respuesta vacia, inalcanzable o sin readiness positiva
bloquea el despliegue.

Actualizacion 2026-07-09 tarde: antes del swap atomico el deploy ejecuta
`orquesta_server_ctl.sh stop`. Si la parada gobernada falla, aborta con
`deploy_stop_failed` y no sustituye el binario. Esto evita que un servidor vivo
con binario viejo convierta `ctl start` en no-op y termine verificando contra
un proceso stale.

Uso minimo:

```bash
ORQUESTA_DEPLOY_REPO=/ruta/repo \
ORQUESTA_DEPLOY_WORKTREE=/ruta/worktree-deploy \
ORQUESTA_DEPLOY_REF=main \
ORQUESTA_DEPLOY_BINARY=/srv/orquesta-self/runtime/orquesta-server-claude \
ORQUESTA_CTL_HOME=/srv/orquesta-self/claude-director-20260705 \
ORQUESTA_CTL_WORKDIR=/srv/orquesta-self/worktrees/pilot-remoto-1 \
ORQUESTA_DEPLOY_REQUIRE_CONFIG=1 \
  scripts/orquesta_server_deploy.sh
```

Contrato operativo:

- `deploy_not_fast_forward` bloquea si el worktree desplegable no puede avanzar
  por fast-forward hacia la ref objetivo.
- `deploy_config_missing` bloquea cuando `ORQUESTA_DEPLOY_REQUIRE_CONFIG=1` y
  no existe `ORQUESTA_CTL_CONFIG` ni
  `$ORQUESTA_CTL_WORKDIR/orquesta.config.json` legible.
- `deploy_runtime_identity_mismatch` bloquea si `status`, readiness o supervisor
  exponen `binary_sha256`, `runtime_binary_sha256` u
  `orquesta_server_sha256` distinto del binario instalado.
- `deploy_runtime_identity_missing` bloquea si ninguna superficie de estado
  expone identidad del binario vivo.
- `deploy_status_failed` bloquea si `scripts/orquesta_server_ctl.sh status`
  falla tras arrancar.
- `deploy_stop_failed` bloquea si `scripts/orquesta_server_ctl.sh stop` no
  puede parar de forma gobernada el servidor vivo antes del swap.
- `deploy_readiness_unreachable` bloquea si una URL configurada de verificacion
  no responde.
- El recibo queda en
  `$ORQUESTA_DEPLOY_STATE_DIR/orquesta_server_deploy_receipt_v0.json` o, si no
  se define, en `$ORQUESTA_CTL_HOME/state/`.

El swap de binario es atomico; el despliegue completo no es una transaccion con
rollback automatico. Si `start` o la verificacion final fallan, queda backup del
binario previo y recibo `failed` para que el operador restaure de forma
gobernada.

No ejecutar comandos remotos desde este script. La promocion a un host real debe
inyectar rutas locales ya montadas o ejecutarse dentro del host objetivo con
confirmacion operativa externa.
