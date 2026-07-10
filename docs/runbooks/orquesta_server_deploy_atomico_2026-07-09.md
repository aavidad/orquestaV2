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

Actualizacion 2026-07-09 tarde 2: la verificacion de identidad acepta tanto
`binary_sha256` top-level como `runtime_identity.binary_sha256`, que es la forma
publicada por `/api/status` en el servidor real.

Actualizacion 2026-07-09 tarde 3: cuando `ORQUESTA_DEPLOY_REF` es una rama, el
worktree desplegable queda en esa rama con `checkout -B <ref> <sha>`, no en
detached HEAD. Esto evita que un segundo deploy lea una rama local stale aunque
el HEAD del worktree este en el commit correcto.

Actualizacion F5 2026-07-10: `orquesta_server_ctl.sh start` valida antes de
arrancar que `ORQUESTA_CTL_WORKDIR` existe, es un worktree git con `HEAD`
verificable y no esta atrasado ni divergente respecto a su upstream local
conocido. El default remoto del `ctl` pasa a
`/srv/orquesta-self/worktrees/orquesta`; un workdir invalido corta con
`ctl_workdir_invalid`, uno atrasado con `ctl_workdir_stale` y uno divergente con
`ctl_workdir_not_aligned`. El endpoint `prepare-run` del servidor tambien queda
envuelto en composicion para rechazar worktrees invalidos/stale antes de aceptar
trabajo amplio.

Actualizacion F5 2026-07-10 2: el recibo de deploy incluye `remote_url` junto a
`git_sha`, `binary_sha256`, `binary_path` y el `ctl_status` recortado. Tras el
deploy, validar el estado con `orquesta_server_ctl.sh status` o `/api/status` y
comparar `runtime_identity.binary_sha256` con el `binary_sha256` del recibo.

Uso minimo:

```bash
ORQUESTA_DEPLOY_REPO=/ruta/repo \
ORQUESTA_DEPLOY_WORKTREE=/ruta/worktree-deploy \
ORQUESTA_DEPLOY_REF=main \
ORQUESTA_DEPLOY_BINARY=/srv/orquesta-self/runtime/orquesta-server-claude \
ORQUESTA_CTL_HOME=/srv/orquesta-self/claude-director-20260705 \
ORQUESTA_CTL_WORKDIR=/srv/orquesta-self/worktrees/orquesta \
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
  exponen `binary_sha256`, `runtime_identity.binary_sha256`,
  `runtime_binary_sha256` u `orquesta_server_sha256` distinto del binario
  instalado.
- `deploy_runtime_identity_missing` bloquea si ninguna superficie de estado
  expone identidad del binario vivo.
- `deploy_status_failed` bloquea si `scripts/orquesta_server_ctl.sh status`
  falla tras arrancar.
- `deploy_stop_failed` bloquea si `scripts/orquesta_server_ctl.sh stop` no
  puede parar de forma gobernada el servidor vivo antes del swap.
- `deploy_readiness_unreachable` bloquea si una URL configurada de verificacion
  no responde.
- `ctl_workdir_invalid`, `ctl_workdir_stale` y `ctl_workdir_not_aligned`
  bloquean el arranque si el workdir de agentes no es verificable o no esta
  alineado con su upstream local conocido.
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
