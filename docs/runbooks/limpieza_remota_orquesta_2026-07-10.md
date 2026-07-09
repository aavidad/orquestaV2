# Limpieza remota Orquesta - 2026-07-10

## Alcance

Limpieza operativa en el servidor `berserk@uso.dipgra.cloud`, limitada a
`/srv/orquesta-self`.

Exclusion explicita del operador: `uso-app` no se puede tocar. No se reinicio,
paro, borro ni modifico ningun servicio de produccion, systemd, nginx, Docker,
OPES, Hermes ni rutas externas a Orquesta.

## Backup previo

Backup completo creado antes de borrar:

```text
/srv/orquesta-self/backups/cleanup-20260709T221749Z
```

Contenido principal:

- `orquesta-all-refs.bundle`: bundle Git con todas las refs.
- `worktree-*.tgz`: copia completa de cada worktree temporal limpiado.
- `diff-*.patch`: diff binario/textual de cada worktree.
- `status-*.txt` y `untracked-*.txt`: estado Git y untracked por worktree.
- `runtime-escalation-director.tgz`: logs/prompts del agente externo.
- `operator-requests.tgz`: requests de pruebas remotas del 2026-07-09.
- `SHA256SUMS`: hashes de archivos principales.
- `cleanup-actions.txt`: log de acciones de limpieza y verificacion final.

Permisos del backup al cierre: directorio `700`, ficheros `600`.

## Procesos parados

Se pararon solo procesos temporales de Orquesta:

- agente externo Codex de reparacion en
  `/srv/orquesta-self/worktrees/orquesta-external-repair-safe-20260709T220747Z`;
- watcher temporal de validacion `validate_checkpoint`;
- poller Telegram de Orquesta en `pilot-remoto-1` ya no estaba vivo al
  verificar antes de limpiar.

El servidor Orquesta principal quedo vivo:

```text
/srv/orquesta-self/runtime/orquesta-server-claude run --config /srv/orquesta-self/secrets/orquesta.config.json
```

## Worktrees y ramas retiradas

Se eliminaron tras backup:

- `/srv/orquesta-self/worktrees/orquesta-autorepair-20260709T215103Z`
  (`tmp/orquesta-autorepair-20260709T215103Z`)
- `/srv/orquesta-self/worktrees/orquesta-repair-checkpoint-20260709T215549Z`
  (`tmp/orquesta-repair-checkpoint-20260709T215549Z`)
- `/srv/orquesta-self/worktrees/orquesta-external-repair-20260709T220341Z`
  (`tmp/orquesta-external-repair-20260709T220341Z`)
- `/srv/orquesta-self/worktrees/orquesta-external-repair-safe-20260709T220747Z`
  (`tmp/orquesta-external-repair-safe-20260709T220747Z`)
- `/srv/orquesta-self/worktrees/pilot-remoto-1`
  (`pilot-remoto-1`)

Tambien se limpiaron archivos stale de
`/srv/orquesta-self/runtime/escalation-director`, requests temporales de
`/srv/orquesta-self/operator-requests/20260709` y caches Go temporales:

- `/tmp/orquesta-external-repair-go-cache`
- `/tmp/orquesta-external-repair-go-tmp`

## Verificacion final

Estado remoto tras limpieza:

```text
worktree /srv/orquesta-self/worktrees/orquesta
branch refs/heads/trabajo/plataforma-agentes
```

No quedan ramas `tmp/orquesta-*` ni `pilot-remoto-1`.

No quedan procesos:

- `codex exec`
- `orquesta-external-repair`
- `orquesta_telegram_operator_poller`
- `go test -count`
- `validate_checkpoint`

`uso-app` solo aparecio en lectura como conexion Postgres idle:

```text
postgres: uso_app_9da1d59534 uso ... idle
```

No se actuo sobre ella.

## Material recuperable

Los worktrees retirados contenian trabajo parcial respaldado en el backup:

- `orquesta-autorepair`: parche parcial para autorreparacion goal-first.
- `orquesta-repair-checkpoint`: parche focal para no tratar
  `checkpoint_started` como issue terminal cuando es progreso temprano.
- `orquesta-external-repair-safe`: avance de puerto/helper MCP para
  `repair_run_refs`, con tests focales MCP verdes antes de cortar.
- `pilot-remoto-1`: poller Telegram opt-in, config real no commiteada,
  nightly/metricas y auditorias parciales.

No integrar esos parches por cherry-pick bruto. Si se recuperan, hacerlo desde
`diff-*.patch` o los `worktree-*.tgz`, revisando conflictos contra
`trabajo/plataforma-agentes` y manteniendo secretos fuera de Git.

## Estado pendiente

- Reintegrar selectivamente lo valido de los parches respaldados.
- Rehacer Telegram desde ubicacion canonica y config fuera del repo si se
  retoma F2.
- Implementar F8: dossier final pre-lanzamiento del wizard antes de crear apps.
