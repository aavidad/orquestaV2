# Limpieza local de runtime Orquesta

Estado: runbook operativo para `ARCH-ORQ-20260630-003`.

Objetivo: reducir presion de disco/IO de `.orquesta-runtime` y
`.orquesta-purged-*` sin borrar evidencias por defecto.

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
