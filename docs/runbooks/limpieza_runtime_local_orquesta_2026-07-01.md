# Limpieza de runtime Orquesta

Estado: runbook operativo para `ARCH-ORQ-20260630-003`.

Objetivo: reducir presion de disco/IO de `.orquesta-runtime` y
`.orquesta-purged-*`, y evitar colapsos de disco en runtimes remotos aislados,
sin borrar evidencias por defecto.

Comando de inventario seguro:

```bash
scripts/orquesta_runtime_retention.sh --min-age-days 14
```

El comando anterior solo informa. No borra la raiz `.orquesta-runtime`, no borra
contenedores globales `codex-waves`/`waves` y solo enumera hijos antiguos o
`.orquesta-purged-*` en la raiz del repo. Si un candidato contiene
`agent_ack.json`, `director_decisions.json`, `orquesta_shutdown_request.json`
sin checkpoint, `outbox`, `plan_state` o un `codex_wave_registry_v0.json` con
agentes `running`/`stop_requested` o PID vivo, queda bloqueado incluso en modo
`--delete`. Los hijos de `codex-waves`/`waves` solo se consideran borrables si
tienen `codex_wave_registry_v0.json` directo; contenedores intermedios de dominio
quedan bloqueados.

Borrado explicito, solo tras revisar el inventario:

```bash
scripts/orquesta_runtime_retention.sh --min-age-days 14 --delete --confirm-delete orquesta-runtime-retention
```

Reglas:

- No ejecutar en produccion ni sobre rutas de OPES productivo.
- No usar con `--delete` si hay agentes vivos usando ese repo.
- Si hay duda sobre una evidencia, mover/copiar fuera del repo antes de borrar.
- Si se necesita limpiar un runtime de ola concreta, preferir el mecanismo
  existente `--purge-runtime --confirm-purge-runtime=<wave_ref>`.

## Norma remota

En servidor remoto aislado, cada agente debe tratar el espacio como recurso de
produccion aunque el runtime sea de pruebas:

- Antes y despues de builds, smokes largos, indexacion, subagentes o goals
  masivos, revisar `df -h`, `du -xh --max-depth=1` del runtime y procesos vivos.
- Enviar temporales a rutas aisladas y conocidas: `TMPDIR`, `GOTMPDIR`,
  `GOCACHE`, `GOMODCACHE`, `GOPATH`, `ORQUESTA_FLAKY_HARNESS_CACHE_ROOT` y
  runtime propio bajo `/srv/orquesta-self/runtime`.
- Limpiar al terminar solo lo que el agente haya creado y este probado como
  inactivo: caches Go, `go-build`, bundles, copias `audit-*` sin worktree vivo,
  smokes temporales no citados como evidencia, logs rotados y directorios de
  indexadores parados.
- No borrar nunca a ciegas worktrees Git, `server-latest`, runtimes con procesos
  vivos, sesiones tmux, sockets activos, markers de owner, `agent_ack`,
  `director_decisions`, `outbox`, `plan_state`, `plan.json`,
  `artifact_manifest.json`, `artifacts_manifest.json` ni evidencias referidas
  desde inventario/incidencias.
- Si una carpeta parece obsoleta pero podria ser evidencia, primero escribir un
  resumen compacto o manifest de retencion y dejarla pendiente de limpieza
  gobernada. No convertir una limpieza manual urgente en politica normal.
