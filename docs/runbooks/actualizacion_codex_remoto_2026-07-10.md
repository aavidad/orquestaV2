# Actualizacion Codex remoto 2026-07-10

Estado: actualizacion aplicada en prefijo aislado, sin tocar `uso-app` ni el npm global de `/usr/local`.

- Version anterior global: `/usr/local/bin/codex` -> `codex-cli 0.128.0`.
- Intento de `npm install -g @openai/codex@latest` fallo por `EACCES` sobre `/usr/local/lib/node_modules`; no cambio el binario global.
- Instalacion segura: `npm install -g --prefix /srv/orquesta-self/tools/npm-global @openai/codex@0.144.1`.
- Binario nuevo: `/srv/orquesta-self/tools/npm-global/bin/codex` -> `codex-cli 0.144.1`.
- Backup del prefijo previo: `/srv/orquesta-self/tools/npm-global.backup.20260710T004339Z`.
- Probe con `CODEX_HOME=/srv/orquesta-self/codex-home` y modelo `gpt-5.6-sol`: `codex exec --skip-git-repo-check --sandbox read-only -m gpt-5.6-sol "print ok"` devuelve `ok`.

Decision operativa:

- `scripts/orquesta_server_ctl.sh` arranca Orquesta con `ORQUESTA_CODEX_COMMAND=${ORQUESTA_CODEX_COMMAND:-/srv/orquesta-self/tools/npm-global/bin/codex}`.
- El modelo por defecto remoto se cambia en `/srv/orquesta-self/codex-home/config.toml` a `gpt-5.6-sol` tras backup.
- Rollback rapido: exportar `ORQUESTA_CODEX_COMMAND=/usr/local/bin/codex` al arrancar Orquesta y restaurar el backup de config si `gpt-5.6-sol` falla.

Incidencia residual:

- El probe sigue avisando `failed to clean up stale arg0 temp dirs` y `failed to install system skills` por permisos en el `CODEX_HOME`; no bloquea ejecucion, pero conviene limpiarlo en una tarea gobernada de permisos/higiene.
