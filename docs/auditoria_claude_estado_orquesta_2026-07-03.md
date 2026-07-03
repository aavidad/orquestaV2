# Informe para revision externa Claude: estado Orquesta y conectores

Fecha: 2026-07-03.
Repo/base de trabajo: `origin/trabajo/plataforma-agentes`.
Objetivo de la revision: detectar errores de arquitectura, falsos verdes y
bloqueos reales para llegar a Orquesta + conectores al 100%, sin tocar
produccion ni OPES productivo.

## Resumen ejecutivo

Orquesta ya opera en modo `goal_first` con Codex Goal como loop interno, y el
loop legacy queda solo con opt-in explicito. El eje que sigue generando fallos
recurrentes no es "un bug suelto", sino el contrato de estado vivo entre:

- `run_ref`
- `goal_ref` / `external_goal_ref`
- backend `app_server_tmux`
- proceso tmux/app-server
- artefactos/checkpoints/receipts
- cierre terminal aceptado o rework

Ese eje aparece en `shutdown`, `observe_goal`, `autoprogramming/status`,
`domain-work/status`, Nueva App y OPES. El agente remoto esta cerrando piezas de
contrato MCP/status/shutdown. En local se ha cerrado un bloqueo T90 y se ha
encontrado un fallo real del smoke Nueva App.

## Estado de sincronizacion

Remoto visto por SSH en `berserk@uso.dipgra.cloud`:

- CWD remoto del agente: `/srv/orquesta-self/runtime/audit-13611445`.
- Rama remota/GitHub observada: `trabajo/plataforma-agentes`.
- Ultimo commit remoto integrado: `7fb0568 Resuelve Codex con path proyectado`.
- Avance desde `9cbb1a0`: 7 commits, 13 ficheros, 190 inserciones y 18
  borrados.
- Areas tocadas por el remoto: `modulos/orquesta-mcp`,
  `modulos/orquesta-mcp/docs/contratos.md`,
  `docs/inventario_bugs_orquesta_2026-06-30.md` y
  `cmd/orquesta-server/codex_env_v0.go`.
- En la ultima comprobacion no habia proceso Codex/tmux de trabajo activo; el
  remoto conserva 12 artefactos/docs sin trackear derivados de runs de
  autoprogramacion, no codigo aplicado.

Commits remotos relevantes de la tanda observada:

- `8639d3a` Declara status domain-work
- `39cdbad` Declara diagnosticos run control
- `0eaf83d` Declara diagnosticos run supervisor
- `a8015c5` Registra runtime Orquesta aislado caido
- `0c3dd56` Declara diagnosticos autoprogramming supervise
- `ea607a4` Declara diagnosticos observe active goals
- `7fb0568` Resuelve Codex con path proyectado
- `eb8e4a0` Declara accion en observe goal
- `7c904db` Declara observaciones batch goal
- `812f9a7` Declara acciones en status MCP
- `5550527` Declara refs en shutdown MCP
- `9cbb1a0` Declara checkpoints en shutdown MCP

## Write-set local actual

Cambios locales integrados en commit rebaseado sobre `7fb0568`:

- `cmd/orquesta-server/codex_goal_app_server_v0.go`
- `cmd/orquesta-server/codex_goal_app_server_rpc_v0.go`
- `cmd/orquesta-server/codex_goal_backend_env_v0.go`
- `cmd/orquesta-server/codex_goal_app_server_tmux_v0_test.go`
- `cmd/orquesta-server/smoke_goal_first_script_guard_v0_test.go`
- `scripts/smoke_goal_first_app_server_real.sh`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- este informe

Commit local antes de push: `c1a5a7b3 Cierra T90 y smoke Nueva App tmux`.

## Hallazgos cerrados en local

### BUG-ORQ-20260703-142: T90 cmd/orquesta-server

Sintoma:

`go test -count=1 ./cmd/orquesta-server -run 'TestResidualGoFileBudgetT90V0'`
fallaba porque `cmd/orquesta-server/codex_goal_app_server_v0.go` tenia 928
lineas y superaba el limite duro de 900.

Cambio:

- Se extrajeron helpers JSON-RPC a
  `cmd/orquesta-server/codex_goal_app_server_rpc_v0.go`.
- `cmd/orquesta-server/codex_goal_app_server_v0.go` queda en 788 lineas.
- No se toco `cmd/orquesta-server/codex_goal_app_server_tmux_v0.go`.

Evidencia:

```bash
go test -count=1 ./cmd/orquesta-server -run 'TestResidualGoFileBudgetT90V0'
go test -count=1 ./cmd/orquesta-server -run 'TestCodexAppServer.*RPC|TestDecodeCodexAppServerRPC|TestServerCodexAppServerGoalBackend'
go test -count=1 ./cmd/orquesta-server
```

Resultado observado: verde.

### Smoke Nueva App: auth de tmux en startup

Sintoma:

`scripts/smoke_goal_first_app_server_real.sh` fallaba antes de crear app:

```text
readiness_http_status=503
readiness_startup_status=startup_degraded_codex_goal_backend_degraded
readiness_diagnostic_0_code=codex_goal_backend_degraded
...
codex_app_server_auth_missing
```

Reproduccion conservada:

- `/tmp/orquesta-goal-first-app-server.u5E6QB`
- `/tmp/orquesta-goal-first-app-server.3gIS4w`
- `/tmp/orquesta-goal-first-app-server.0Z4Gg4`

Causa tecnica:

El backend `app_server_tmux` validaba primero fuente + destino de credenciales,
pero despues volvia a validar solo el `CODEX_HOME` aislado antes de que
`Ensure` lo hubiese preparado. Eso convertia un entorno con fuente valida en
`codex_app_server_auth_missing`.

Cambios:

- En `serverCodexGoalBackendFromEnvForWorkDirV0`, la validacion de auth de tmux
  queda marcada como hecha al comprobar fuente + destino.
- Ya no se exige el code-home aislado antes de `Ensure`.
- El smoke detecta `~/.codex` como fuente por defecto si hay `auth.json` y
  `config.toml`, sin imprimir la ruta ni contenido sensible.

Evidencia focal:

```bash
bash -n scripts/smoke_goal_first_app_server_real.sh
go test -count=1 ./cmd/orquesta-server -run 'TestServerCodexGoalBackendFromEnvV0Tmux(NoExigeCodeHomeAisladoAntesDeEnsure|SinAuthDegradaYConservaShutdown|PreflightOK)V0|TestCodexAppServerTmuxBackendV0PreparaCodeHomeDesdeCODEXHOMEResueltoV0'
go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirstAppServerRealProjectsCodeHomeFromCodexHomeV0|TestSmokeGoalFirstAppServerRealDiagnosticaReadinessStartupV0'
```

Resultado observado: verde.

### BUG-ORQ-20260703-147: catalogo MCP demasiado grande tras sincronizar remoto

Sintoma:

Tras rebasear sobre `7fb0568`, `go test -count=1 ./modulos/orquesta-mcp`
fallaba en `TestRegisterMCPTransportV0ExponeOperacionesExistentes` porque el
JSON del catalogo completo de tools subio a 33.259 bytes frente al limite
31.800.

Causa tecnica:

Los contratos completos `InputShape`/`OutputShape` seguian siendo correctos como
campos internos, pero al serializar el registro entero se duplicaban esquemas
largos de varias herramientas MCP. El remoto habia anadido diagnosticos y
acciones utiles, pero el indice compacto dejo de ser compacto.

Cambio:

- `MCPTransportToolEnvelopeV0.MarshalJSON` compacta shapes largas como
  `shape_ref:<resource>#input|output` solo en la serializacion JSON del catalogo.
- Los campos Go completos no se mutan.
- El MCP real sigue construyendo schemas desde `InputShape` y `OutputShape`.

Evidencia:

```bash
go test -count=1 ./modulos/orquesta-mcp
go test -count=1 ./cmd/orquesta-server
```

Resultado observado: verde.

## Smoke Nueva App real ejecutado

Comando lanzado:

```bash
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 \
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 \
ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux \
ORQUESTA_KEEP_SMOKE_DIR=1 \
./scripts/smoke_goal_first_app_server_real.sh
```

Resultado funcional observado:

- Readiness ya pasa tras el fix de auth.
- `POST /api/v0/apps/director -> HTTP 200`.
- `run_ref=run-spec-smoke-goal-first-req-smoke-goal-first-a63806e510f99a8d68fd767946b14709`
- `goal_ref=goal-ref-app-director-run-spec-smoke-goal-first-req-smoke-goal-first-a63806e510f99a8d68fd767946b14709`
- `external_goal_ref=019f270d-7f56-72b0-9bb4-16834ee466ed`
- Directorio: `/tmp/orquesta-goal-first-app-server.a4QRfx`
- `goal_status=complete`.
- `run_status=cerrada`.
- `closure_status=accepted`.
- `closure_accepted=true`.
- `recommended_action=no_action_closed`.
- `artifact_refs=11`.
- `evidence_refs=11`.
- 28 ficheros materializados en
  `/tmp/orquesta-goal-first-app-server.a4QRfx/project/generated-apps/smoke-goal-first`.

Artefactos observados:

- `README.md`
- `pyproject.toml`
- `run.py`
- `source_tree.md`
- `handoff_report.md`
- `docs/manual_desarrollo.md`
- `docs/manual_sistemas.md`
- `docs/manual_usuario.md`
- `src/smoke_goal_first/domain/...`
- `src/smoke_goal_first/application/...`
- `src/smoke_goal_first/ports/...`
- `src/smoke_goal_first/adapters/...`
- `src/smoke_goal_first/bootstrap/app.py`
- `src/smoke_goal_first/web/index.html`
- `tests/test_http_api.py`
- `tests/test_note_service.py`
- `docs/orquesta_goal_result_goal-ref-...json`

Resultado del receipt final:

`observe_response.json` publica `estado=ok`, cierre aceptado, refs no vacias y
sin `issues`. Por tanto el problema de receipt `status=invalid` observado en
polls intermedios no es el estado final de este smoke.

Residuales observados:

- La solicitud pedia Go/net/http, pero el agente genero Python. Revisar si el
  contrato Nueva App valida lenguaje/framework o si acepta artefactos utiles sin
  rework.
- El script salio con codigo 1 despues del cierre funcional porque
  `/api/v0/server/shutdown` devolvio `shutdown_ready=true`, pero el wrapper
  todavia observo el proceso servidor vivo durante su comprobacion inmediata:
  `el servidor siguio vivo tras shutdown_ready=true`.
- `shutdown_response.json` declaraba `status=ready`, `runs_requested=0`,
  `runs_stopped=0`, `agents_in_flight=0`, `checkpoints_pending=0` y evidencias
  `evidence-ref-codex-app-server-tmux-cleanup-requested`,
  `evidence-ref-codex-app-server-tmux-configured-cleaned` y
  `evidence-ref-shutdown-goal-backend-cleanup-requested`.
- Tras el `trap` del smoke no quedaron procesos observables asociados a
  `orquesta-goal-first-app-server.a4QRfx` ni a la sesion
  `orquesta-goal-4a2ea0ed70880029`. El fallo es de contrato temporal del
  shutdown/wrapper, no de cierre funcional de la app.
- Actualizacion posterior: `BUG-ORQ-20260703-146` queda cerrado en wrapper. Si
  `shutdown_ready=true` indica drenaje operativo pero el proceso temporal sigue
  vivo, el smoke envia senal cooperativa local mediante
  `smoke_shutdown_orquesta_server "$server_pid" "" 0 25 "$runtime_dir"` y solo
  falla si el proceso continua vivo despues.

## Bugs abiertos de arquitectura a revisar

Prioridad 1: lifecycle goal-first / estado vivo / shutdown.

IDs principales:

- `BUG-ORQ-20260701-065`
- `BUG-ORQ-20260701-066`
- `BUG-ORQ-20260701-073`
- `BUG-ORQ-20260701-076`
- `BUG-ORQ-20260701-077`
- `BUG-ORQ-20260701-079`
- `BUG-ORQ-20260701-088`

Hipotesis:

Hay demasiadas fuentes de verdad parciales para saber si un trabajo esta vivo,
cerrado, bloqueado, delegado o pendiente de rework. El contrato debe converger a
un unico grafo operativo:

`run_ref -> goal_ref -> external_goal_ref -> backend/process -> checkpoint/artifact -> receipt -> terminal/rework`.

Prioridad 2: receipts/write-set/materialized artifacts.

IDs:

- `BUG-ORQ-20260701-072`
- `BUG-ORQ-20260701-075`
- `BUG-ORQ-20260701-085`

Hipotesis:

Orquesta ya detecta muchos casos recuperables, pero aun falta que el runtime
fuerce o repare de forma automatica el receipt terminal cuando el filesystem
contiene artefactos utiles y el JSON durable sigue en checkpoint/invalid.

Prioridad 3: OPES/conector y gates ejecutables.

ID padre:

- `BUG-ORQ-20260701-058`

Pendientes indicados por OPES:

- T14 links/clicks.
- T15/T19 visual raster real y terna.
- T16 `html_ampliado`.
- T17 espanol/encoding.
- T18 proveedores.

Recomendacion:

Convertir esos puntos en required tests/gates de `completed_syllabus_package`,
no en prompts ni revisiones manuales.

## Preguntas para Claude

Actualizacion antes de parada 2026-07-03:

- `BUG-ORQ-20260703-145` queda cerrado en codigo: `AppSpecV0` conserva
  `technical`, Nueva App goal-first lo convierte en refs/criterios/manifest y
  `ObserveAppDirectorGoalV0` bloquea/rework si una entrega contradice lenguaje
  o framework declarado. Cobertura focal en `orquesta-factory`,
  `orquesta-app-director-service` y `cmd/orquesta-server`.

1. Revisar si el fallo Nueva App actual debe cerrarse como:
   - bug de prompt/spec por generar Python en vez de Go;
   - bug de validador por no cortar lenguaje/framework;
   - bug aceptable si el contrato tecnico era preferencia blanda;
   - o combinacion de prompt + validador + matriz de required tests.
2. Proponer un contrato minimo de `GoalWorkResultV0` terminal para Nueva App:
   artefactos, tests, lenguaje, estructura hexagonal, manifest y cierre.
3. Revisar si readiness debe bloquear por backend goal degradado cuando el
   servidor esta vivo pero solo se usara observacion/status. Para este smoke si
   debe bloquear porque va a lanzar goal real, pero el criterio puede necesitar
   matiz por modo.
4. Revisar si `/server/shutdown` puede publicar `shutdown_ready=true` antes de
   que el proceso HTTP contenedor haya terminado, o si el wrapper debe
   interpretar `ready` como "sin trabajo vivo" y esperar una salida posterior
   por protocolo separado.
5. Revisar si los contratos MCP que esta cerrando el remoto ya declaran toda la
   evidencia necesaria para que un operador no lea logs: `evidence_refs`,
   `checkpoint_refs`, `active_work_refs`, `recommended_action`, `issue_codes`.

## Comandos de verificacion recomendados

```bash
git diff --check
go test -count=1 ./cmd/orquesta-server
go test -count=1 ./modulos/orquesta-mcp
go test -count=1 ./modulos/orquesta-app-codex-stack
bash -n scripts/smoke_goal_first_app_server_real.sh
ORQUESTA_GOAL_FIRST_SMOKE_PREFLIGHT_ONLY=1 ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux ./scripts/smoke_goal_first_app_server_real.sh
```

Smoke real Nueva App, solo en entorno aislado y con cuota disponible:

```bash
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 \
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 \
ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux \
ORQUESTA_KEEP_SMOKE_DIR=1 \
./scripts/smoke_goal_first_app_server_real.sh
```

No ejecutar contra produccion ni con OPES productivo.
