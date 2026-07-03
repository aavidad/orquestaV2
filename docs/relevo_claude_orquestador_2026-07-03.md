# Relevo para Claude orquestador - 2026-07-03

## Corte operativo

- Subagentes locales de Codex cerrados por orden del operador.
- `orquesta-server run` local de prueba parado.
- Remoto `berserk@uso.dipgra.cloud`: sin `codex`, `tmux` ni `orquesta-server`
  vivos observados; worktree `/srv/orquesta-self/worktrees/orquesta` sigue en
  HEAD `1667a411d4` con WIP sucio de 71 ficheros. No integrar completo.
- No hay `codebase-memory-mcp` vivo observado en local.

## Cambio guardado para revision

Se deja un avance parcial de T-PER-301/T-PER-302:

- Nuevo modulo `modulos/orquesta-runtime-codex-appserver`.
- `cmd/orquesta-server/codex_goal_backend_env_v0.go` instancia el backend
  principal desde el modulo nuevo.
- El modulo ya no importa `orquesta/modulos/orquesta-server`; `cmd` adapta su
  `ConfigV0` al `ConfigV0` neutral del modulo.
- `architecture_boundaries_test.go` protege que el modulo no dependa de `cmd`,
  stores de producto ni `orquesta-server`.
- `estado_backend_v0.go` introduce `EstadoBackendAppServerV0`,
  `ObservacionBackendV0` y `TransicionBackendV0` pura con tests.

## Desviacion importante

No marcar T-PER-301 ni T-PER-302 como cerradas todavia.

La implementacion legacy `cmd/orquesta-server/codex_goal_app_server*.go` sigue
existiendo y conserva tests/logica principal. Esto deja dos superficies donde
podrian caer futuros fixes: el modulo nuevo y `cmd`. La maquina de estados
existe, pero aun no gobierna `EnsureV0`, shutdown, cleanup ni active-work.

Incidencia canonica: `BUG-ORQ-20260703-150`.

## Verificacion hecha antes del corte

```bash
go test -count=1 ./modulos/orquesta-runtime-codex-appserver
go test -count=1 ./cmd/orquesta-server -run 'Test(CodexAppServer|CleanupCodex|WaitForStateHealthy|SmokeGoalFirstScript|SmokeCommon)'
go test -count=1 . -run 'Test(CodexAppServerAdapterDoesNotImportCmdOrProductStorage|NeutralOrchestrationPackagesDoNot(Import|DependOn)ProductAdapters)'
```

Pendiente para cerrar antes de claim 100%:

- `go test -count=1 ./...`
- `go build ./...`
- mover/reducir legacy `cmd/orquesta-server/codex_goal_app_server*.go` a
  fachadas finas o completar migracion de tests al modulo.
- crear recolector unico de observaciones y consumir `TransicionBackendV0`.

