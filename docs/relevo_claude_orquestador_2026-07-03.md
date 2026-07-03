# Relevo para Claude orquestador - 2026-07-03

> Estado vigente: este relevo describe un corte intermedio. Queda supersedido
> por la integracion T270 de la misma noche, documentada en
> `docs/bitacora_correccion_pericial_2026-07-03.md` y en
> `docs/inventario_bugs_orquesta_2026-06-30.md` como cierre de
> `BUG-ORQ-20260703-150`. No reabrir T-PER-301/T-PER-302 salvo regresion del
> recolector unico o reintroduccion de implementacion tmux/app-server dentro de
> `cmd/orquesta-server`.

## Corte operativo

- Subagentes locales de Codex cerrados por orden del operador.
- `orquesta-server run` local de prueba parado.
- Remoto `berserk@uso.dipgra.cloud`: sin `codex`, `tmux` ni `orquesta-server`
  vivos observados; worktree `/srv/orquesta-self/worktrees/orquesta` sigue en
  HEAD `1667a411d4` con WIP sucio de 71 ficheros. No integrar completo.
- No hay `codebase-memory-mcp` vivo observado en local.

## Cambio guardado para revision en el momento del corte

En el momento de este relevo se dejaba un avance parcial de
T-PER-301/T-PER-302:

- Nuevo modulo `modulos/orquesta-runtime-codex-appserver`.
- `cmd/orquesta-server/codex_goal_backend_env_v0.go` instancia el backend
  principal desde el modulo nuevo.
- El modulo ya no importa `orquesta/modulos/orquesta-server`; `cmd` adapta su
  `ConfigV0` al `ConfigV0` neutral del modulo.
- `architecture_boundaries_test.go` protege que el modulo no dependa de `cmd`,
  stores de producto ni `orquesta-server`.
- `estado_backend_v0.go` introduce `EstadoBackendAppServerV0`,
  `ObservacionBackendV0` y `TransicionBackendV0` pura con tests.

## Desviacion historica del corte

En este punto todavia no debian marcarse T-PER-301 ni T-PER-302 como cerradas.
Ese estado ya no es vigente tras T270.

En ese momento, la implementacion legacy
`cmd/orquesta-server/codex_goal_app_server*.go` seguia existiendo y conservaba
tests/logica principal. Esto dejaba dos superficies donde podian caer futuros
fixes: el modulo nuevo y `cmd`. La maquina de estados existia, pero todavia no
gobernaba `EnsureV0`, shutdown, cleanup ni active-work.

Incidencia canonica: `BUG-ORQ-20260703-150`.

Cierre posterior T270: 19 ficheros legacy se retiraron o redujeron a fachada
fina; en `cmd/orquesta-server` queda `codex_goal_app_server_v0.go` como
reexport/wiring hacia `modulos/orquesta-runtime-codex-appserver` y su test de
wiring. El modulo runtime contiene el recolector unico
`recolector_observacion_backend_v0.go`, y `TransicionBackendV0` gobierna
`EnsureV0`, shutdown, cleanup y active-work. La fila vigente de inventario marca
`BUG-ORQ-20260703-150` como cerrado.

## Verificacion hecha antes del corte

```bash
go test -count=1 ./modulos/orquesta-runtime-codex-appserver
go test -count=1 ./cmd/orquesta-server -run 'Test(CodexAppServer|CleanupCodex|WaitForStateHealthy|SmokeGoalFirstScript|SmokeCommon)'
go test -count=1 . -run 'Test(CodexAppServerAdapterDoesNotImportCmdOrProductStorage|NeutralOrchestrationPackagesDoNot(Import|DependOn)ProductAdapters)'
```

Pendiente historico ya cerrado por T270:

- `go test -count=1 ./...`
- `go build ./...`
- mover/reducir legacy `cmd/orquesta-server/codex_goal_app_server*.go` a
  fachadas finas o completar migracion de tests al modulo.
- crear recolector unico de observaciones y consumir `TransicionBackendV0`.

Verificacion posterior reejecutada sin servidores durante la reconciliacion
documental:

```bash
go test -count=1 ./modulos/orquesta-runtime-codex-appserver ./cmd/orquesta-server -run 'Test(TransicionBackendV0|RecolectarObservacionBackendV0|CodexAppServerTmuxBackendV0EnsureShutdownCleanupMigradoV0|ServerCodexGoalBackendFromEnvV0TmuxNoArrancaAppServerEnConstruccionV0|ServerSupervisorWithCodexGoalBackendV0|ServerGoalWorkPortsFromBackendV0|ResidualGoFileBudgetT90V0)'
go test -count=1 . -run 'Test(CodexAppServerAdapterDoesNotImportCmdOrProductStorage|NeutralOrchestrationPackagesDoNot(Import|DependOn)ProductAdapters)'
```
