# Incidencia OPES: external-work goal-first falla por tmux session exited

Fecha: 2026-06-29

## Contexto

Desde `/home/alberto/Trabajo/OPES` se intentó lanzar la revisión 100% de tres
temarios existentes mediante Orquesta, usando la ruta vigente:

- `POST /api/v0/external-work/dry-run`
- `POST /api/v0/external-work/run`

Servidor activo:

- Base URL: `http://127.0.0.1:19041`
- Readiness: `GET /api/v0/server/readiness` devuelve `ready=true`.
- Estado: `startup_ready`, mensaje `director: orquesta preparada; autodiagnostico sin runs transitorios ni cola sucia`.

Requests OPES usados:

- `opes-salidas/coordinacion_temarios/revision_100_psicologo_asg_operario_2026-06-29/requests/asg_cierre_tecnico_external_work.json`
- `opes-salidas/coordinacion_temarios/revision_100_psicologo_asg_operario_2026-06-29/requests/operario_cierre_tecnico_external_work.json`
- `opes-salidas/coordinacion_temarios/revision_100_psicologo_asg_operario_2026-06-29/requests/psicologo_audio_qa_external_work.json`

## Resultado observado

Los tres dry-runs compilan correctamente:

- `estado=ok`
- `route_policy=goal_first`
- `director_execution_mode=goal_first`
- `goal_ref` generado
- `write_set` correcto y separado
- `required_tests` expuestos

Al lanzar por `POST /api/v0/external-work/run`, los tres devuelven HTTP 400 con:

```json
{
  "estado": "error",
  "route_policy": "goal_first",
  "director_execution_mode": "goal_first",
  "next_actions": [
    "configure_codex_goal_backend",
    "do_not_fallback_to_legacy_director_loop",
    "observe_goal"
  ],
  "errores_publicos": [
    {
      "code": "external_work_goal_launch_failed",
      "field": "goal_launcher",
      "message": "codex_app_server_tmux_session_exited"
    }
  ]
}
```

Después, `POST /api/v0/autoprogramming/status` devuelve cola vacía o no visible:

```json
{
  "estado": "ok",
  "diagnostics": [
    {
      "code": "queue_empty_or_not_visible",
      "scope": "queue",
      "message": "cola sin candidatos visibles; no declarar supervision de cola como accion segura"
    }
  ]
}
```

## Impacto

Orquesta informa readiness `ready=true`, pero no puede materializar trabajos
`external-work/run` goal-first porque el backend Codex goal por `app_server_tmux`
termina antes de entregar. Desde OPES esto bloquea el uso normal de Orquesta
para cerrar ASG, Operario y Psicólogo.

No parece un error del payload OPES: los dry-runs son válidos.

## Esperado

Una de estas salidas:

1. Si el backend goal no está operativo, readiness/status deben avisar que
   Orquesta no está lista para trabajos `external-work/run` goal-first.
2. Si `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux` está configurado, Orquesta
   debe levantar o validar la sesión antes de aceptar readiness operativa para
   trabajo externo real.
3. Si el proceso tmux sale, el error debe publicar causa accionable con ruta de
   log o diagnóstico del arranque de `codex app-server`.

## No hacer

No reactivar fallback legacy para estos trabajos OPES. La ruta vigente debe
seguir siendo goal-first.

## Reintento 2026-06-29 16:41

Desde `/home/alberto/Trabajo/OPES` se reintentó `external-work/run` para:

- `opes-salidas/coordinacion_temarios/revision_100_psicologo_asg_operario_2026-06-29/requests/operario_cierre_tecnico_external_work.json`

El `dry-run` seguía compilando correctamente con `route_policy=goal_first`, pero
la ejecución real volvió a fallar con:

```json
{
  "code": "external_work_goal_launch_failed",
  "field": "goal_launcher",
  "message": "codex_app_server_tmux_session_exited"
}
```

Impacto específico OPES: impide que Orquesta cierre por sí sola el informe de
Operario aunque el paquete local tenga validaciones técnicas en verde.

## Revisión 2026-06-29

Estado: corregido a nivel de código y pruebas focales en la rama local
`trabajo/plataforma-agentes`; pendiente de reintento real con los payloads OPES
originales contra un Orquesta reconstruido con estos cambios.

Evidencia local:

- `cmd/orquesta-server/codex_goal_app_server_tmux_v0.go` ya lanza
  `codex app-server --listen unix://<socket>` en tmux, usa `CODEX_HOME`
  aislado y detecta una sesión tmux muerta como
  `codex_app_server_tmux_session_exited`.
- `modulos/orquesta-app-codex-stack/external_work_goal_first_executor_v0.go`
  ya publica `external_work_goal_launch_failed` sin fallback legacy y persiste
  un `GoalWorkStateV0` reparable cuando el lanzamiento falla.
- Pruebas focales ejecutadas:
  - `go test -count=1 ./cmd/orquesta-server -run 'Test(ServerCodexGoalBackendFromEnvV0TmuxPreflightOKV0|CodexAppServerTmuxBackendV0DetectaSesionMuertaSinEsperarSocketTimeoutV0|CodexAppServerTmuxSocketPathV0SeMantieneCortoV0|CodexAppServerTmuxStartupTimeoutV0DaMargenAlPrimerArranqueV0)'`
  - `go test -count=1 ./modulos/orquesta-goal ./modulos/orquesta-app-codex-stack -run 'Test(StartGoalWorkV0|CodexStackV0ExternalWorkRunGoalFirstLaunchFailedPublicaReasonCode|ExternalWorkRunGoalFirst)'`

Fixes aplicados localmente:

- `cf3d0d40 Harden tmux goal app-server socket handling` endurece permisos del
  directorio/socket, valida longitud de socket Unix, repara socket existente
  antes de reusar sesión y añade shutdown de la sesión propia.
- `d345ee87 Persist partial goal launch state` persiste estado parcial si el
  launcher devuelve `GoalRef`/`ExternalGoalRef` antes de fallar.
- `c3ac8154 Require terminal goal before durable result closure` impide cerrar
  desde un resultado durable si el goal remoto sigue activo o pertenece a otro
  `external_goal_ref`.

Validación manual del CLI:

- `codex-cli 0.142.3` acepta `codex app-server --listen unix://PATH`.
- Con `CODEX_HOME` aislado, el app-server crea socket Unix `srw-------` y queda
  vivo hasta ser detenido por `timeout`; no sale inmediatamente por argumentos
  inválidos.

Pendiente para cerrar la incidencia al 100%:

1. Incorporar en la rama de trabajo local los endurecimientos de
   `refs/remotes/self/orquesta-remote-goal-100` si aún no están mergeados.
   Estado 2026-06-29: hecho en `trabajo/plataforma-agentes` con
   `cf3d0d40`, `d345ee87` y `c3ac8154`.
2. Reconstruir/reiniciar el Orquesta aislado que atiende `/api/v0/external-work/run`.
3. Repetir al menos el payload
   `operario_cierre_tecnico_external_work.json` y comprobar que deja de devolver
   `codex_app_server_tmux_session_exited`.
4. Si vuelve a fallar, conservar el log tmux/app-server como evidencia y
   devolver un código accionable distinto de un readiness genérico.

## Reintento 2026-06-29 post-fix

Estado: incidencia de lanzamiento cerrada para el caso `operario`; quedan
evidencias aisladas y un hallazgo residual de shutdown.

Entorno usado:

- Repo Orquesta: `trabajo/plataforma-agentes`.
- Orquesta aislado: `127.0.0.1:19041`.
- Raíz de evidencia:
  `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/revision_100_psicologo_asg_operario_2026-06-29/orquesta_goal_retest_20260629`.
- Backend: `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`.
- Legacy loop desactivado:
  `ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=0` y
  `ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=0`.
- Bridge/producción OPES desactivados:
  `ORQUESTA_OPES_BRIDGE_ENABLED=false`, `ORQUESTA_OPES_BASE_URL=` y
  `OPES_BASE_URL=`.

Secuencia:

1. Primer reintento con runtime largo:
   - `dry-run`: HTTP 200.
   - `run`: HTTP 400.
   - Nuevo diagnóstico accionable:
     `codex_app_server_tmux_socket_path_too_long`.
   - Causa: `RuntimeWorkDir` profundo entraba completo en el socket Unix.
2. Fix aplicado:
   - `cmd/orquesta-server/codex_goal_app_server_tmux_v0.go` mantiene el socket en
     `RuntimeWorkDir/goal-srv` si cabe; si no, usa fallback corto privado y
     determinista en `/tmp/oq-gsrv-<uid>-<hash>/s.sock`.
   - `ensureCodexAppServerTmuxRuntimeDirV0` rechaza directorios symlink y fuerza
     permisos `0700`; socket/marker quedan privados.
   - Tests focales añadidos para fallback corto, determinismo/aislamiento y
     rechazo de symlink.
3. Segundo reintento:
   - `dry-run`: HTTP 200.
   - `run`: HTTP 504 `external_work_run_timeout`.
   - Causa: `/api/v0/external-work/run` tenía ventana HTTP interna de 2s y
     cancelaba el lanzamiento antes de persistir el estado goal.
4. Fix aplicado:
   - `modulos/orquesta-mcp/external_work_run_http_v0.go` sube el default de
     respuesta acotada a 30s.
   - `modulos/orquesta-app-gateway/handler_v0.go` conecta `config.Timeout` a
     `ExternalWorkRun`, conservando tests de timeout explícito.
5. Reintento final:
   - `operario_http30_dry_run.json`: HTTP 200, `route_policy=goal_first`.
   - `operario_http30_run.json`: HTTP 200, `estado=ok`,
     `director_execution_mode=goal_first`, `next_actions=[observe_active_goals]`.
   - `external_goal_ref=019f14b3-1fa2-7b10-9d25-d061e269be33`.
   - `operario_http30_observe_goal.json`: HTTP 200, `goal_status=running`,
     resumen `codex_app_server_goal_status_active`.
   - Audit:
     `state_http30/audit/audit_http30_19041.jsonl` registra
     `evidence-ref-codex-app-server-thread-started`,
     `evidence-ref-codex-app-server-goal-set`,
     `evidence-ref-codex-app-server-turn-started` y
     `evidence-ref-codex-app-server-goal-observed`.

Resultado:

- Ya no se reproduce `codex_app_server_tmux_session_exited`.
- Ya no se reproduce `codex_app_server_tmux_socket_path_too_long` con runtime
  largo.
- Ya no se reproduce `external_work_run_timeout` en el lanzamiento normal del
  payload OPES probado.
- El goal queda persistido en
  `state_http30/orchestration-state/app_director_goal_states/`.

Hallazgo residual:

- `POST /api/v0/server/shutdown` devolvió `shutdown_ready=true` en una prueba sin
  agentes, pero el proceso no salió por sí mismo; en otra prueba con goal activo
  la llamada HTTP agotó el timeout del cliente. Para no dejar trabajo OPES
  aislado ejecutándose, se pararon manualmente el servidor de prueba, la sesión
  tmux propia y el app-server Codex. Esto debe tratarse como mejora separada de
  shutdown operativo, no como regresión del launcher goal-first.
