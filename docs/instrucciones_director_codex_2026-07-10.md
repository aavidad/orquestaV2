# Instrucciones del director para Codex/Orquesta - 2026-07-10: corte F3 + F5

Autor: Claude (director/revisor). Contexto: decision documentada en
`docs/runbooks/revision_codex_orquesta_remoto_2026-07-10.md` (Codex revisor,
Orquesta ejecuta, g01/g01-rework-1 aceptados como avance parcial, autonomia
NO cerrada) y analisis estructural en
`docs/analisis_fallos_estructurales_orquesta_2026-07-10.md` (fallos F1-F6,
verificado contra codigo en `43aea232b`).

Este documento define SOLO el siguiente corte: TAREA-F3 y despues TAREA-F5.
No se abren F1/F2/F4/F6 ni tareas amplias de autonomia hasta cerrar estas
dos. Regla de la revision: no lanzar mas goals amplios; un goal acotado cada
vez, sin batch paralelo (el binario remoto vivo `9541e2f0...` NO lleva la
secuenciacion de write-sets solapados ni el fix del 504).

Reglas vigentes: un commit por tarea, `git diff --check`, bitacora al
cerrar, no tocar `uso-app` ni OPES productivo bajo ningun concepto.

Reparto de roles para este corte:

- Orquesta (goal-first) programa el tooling y sus tests (trabajo de codigo).
- Codex revisa como revisor externo; no programa a mano salvo bloqueo.
- La EJECUCION real del drain y del deploy en el servidor es accion de
  operador/Codex con los scripts resultantes, no un goal.

---

## TAREA-F3: drain gobernado por identidad runtime + harness aislado

Ataca F3 de `docs/analisis_fallos_estructurales_orquesta_2026-07-10.md` y el
subfallo `BUG-ORQ-20260710-208E`. Es el corte que rompe el ciclo bloqueado
actual: sin drain fiable no hay deploy, y sin deploy los fixes de control no
llegan al binario vivo.

Restriccion de diseno NO negociable: el drain NO puede depender de
`POST /api/v0/runs/control`, porque `BUG-ORQ-20260710-208C` demuestra que no
propaga parada fiable al backend. Debe operar por identidad runtime real:
procesos, sesiones tmux `orquesta-goal-*`, sockets/FIFOs, owner markers,
pidfiles. Para el propio servidor puede y debe usar el contrato HTTP de
shutdown vigente (`idempotency_key` + `requested_by=orquesta-director`,
`cleanup_goal_backends=true`) y caer a proceso solo para residuos.

Cambios requeridos:

1. `scripts/orquesta_server_drain.sh`, con el patron de seguridad de
   `scripts/orquesta_runtime_retention.sh` (dry-run por defecto, borrado o
   parada real solo con `--drain --confirm-drain orquesta-server-drain`):
   a. Fase inventario (siempre, tambien en dry-run): enumerar procesos
      `orquesta-server*`, `codex app-server`, sesiones tmux
      `orquesta-goal-*`, hijos `go test`/build lanzados por goals, sockets y
      FIFOs `stdin.pipe`, y clasificarlos `drainable` o `protected` por
      identidad (cmdline, cwd, owner marker), nunca por nombre generico.
      Cualquier proceso que no case identidad Orquesta explicita queda
      `protected`; `uso-app` y sus hijos son `protected` incondicionalmente
      y el script debe tener test que lo demuestre.
   b. Fase backup (antes de parar nada): copiar artefactos vivos
      (checkpoints, `orquesta_goal_result*`, `agent_ack`, logs de goal) a
      `/srv/orquesta-self/backups/drain-<timestamp>/`, siguiendo la regla S2
      de la sesion 2026-07-10: parar no es limpiar; no se borra nada en este
      script.
   c. Fase drain: shutdown HTTP con contrato al servidor propio; despues
      SIGTERM cooperativo a procesos `drainable` restantes por PID exacto;
      despues `tmux kill-session` de sesiones `orquesta-goal-*`; kill -9
      solo como ultima fase y dejando constancia. Espera observable de
      salida real (pane_pid/pgid), como ya hace el adaptador
      `app_server_tmux`.
   d. Recibo durable JSON: inventario antes/despues, accion por proceso,
      backup path, `drain_status=clean|residual|refused`, y `residual`
      nunca silencioso.
2. Perfil de harness aislado reutilizable `scripts/lib/isolated_test_env.sh`:
   exporta TMPDIR, GOCACHE, GOMODCACHE (si aplica), CODEX_HOME y rango de
   puertos bajo un directorio privado del corte; lo consumen la verificacion
   amplia, `scripts/orquesta_server_deploy.sh` (verificacion post-deploy) y
   `scripts/orquesta_smoke_nightly.sh`. Hoy solo
   `smoke_self_programming_composite_goal_first.sh` aisla GOCACHE por su
   cuenta; ese caso debe migrar al perfil comun o quedar justificado.
3. La verificacion amplia bajo el perfil corre por lotes de paquetes con
   timeout por lote (no un unico `go test ./...` monolitico), para separar
   fallo real de agotamiento y no tumbar la sesion que la lanza.
4. Guards: `bash -n` de los scripts nuevos; test tipo
   `scripts/test_orquesta_server_drain.sh` con procesos falsos que pruebe
   dry-run, clasificacion protected (incluido un fake `uso-app`), backup
   previo, orden SIGTERM->kill y recibo; registrar el script en la tabla
   `scriptContractGuardV0` de
   `cmd/orquesta-server/smoke_goal_first_script_guard_v0_test.go` (shutdown
   con contrato, sin arranque de servidor gestionado fuera de ctl/deploy).

Guia para el goal (evitar reabrir F4 mientras se cierra F3): un solo goal,
write-set declarado `["docs/runbooks", "scripts"]` con el scope directorio
`docs/runbooks` PRIMERO, para que el resultado durable
`orquesta_goal_result*.json` cuelgue de `docs/runbooks/docs/` y no de
`scripts/docs/` (el generador usa el primer scope directorio; los artefactos
bajo `scripts/docs/` son justamente el patron S13 que estamos conteniendo).

Criterio de cierre TAREA-F3:

- Ejecucion real en el servidor: recibo de drain con inventario
  antes/despues, `uso-app` intacto (evidencia en el recibo), y cero procesos
  Orquesta drainables vivos al terminar.
- Verificacion amplia remota verde DOS veces seguidas bajo el perfil
  aislado, sin `claude_goal_result_invalid`/`gemini_goal_result_invalid` ni
  umbrales temporales excedidos (si reaparecen con harness limpio, dejan de
  ser ruido F3 y pasan a contrato F6: registrarlo, no taparlo).
- Runbook corto en `docs/runbooks/` con comando unico, orden drain->deploy y
  rollback.

## TAREA-F5: deploy del binario nuevo + guard de identidad al arrancar

Ataca F5 del analisis estructural y `BUG-ORQ-20260710-208A`/S1/S3/S14. Se
ejecuta SOLO despues del drain de TAREA-F3.

Cambios requeridos:

1. Ejecutar (operador/Codex, no goal) el deploy atomico existente
   `scripts/orquesta_server_deploy.sh` a `e271d1196` o superior, con
   `ORQUESTA_DEPLOY_REF=trabajo/plataforma-agentes`, verificando
   `runtime_identity.binary_sha256` == sha del build en `/api/status` y
   recibo durable. El binario objetivo lleva los patches focales de 208
   (secuenciacion write-sets, 504, workdir, rework residente) que hoy NO
   estan en el proceso vivo.
2. Guard de identidad en `scripts/orquesta_server_ctl.sh` (trabajo de codigo,
   puede ser goal acotado): antes de arrancar, validar que
   `ORQUESTA_CTL_WORKDIR` existe, es worktree git valido
   (`git -C rev-parse --git-dir`) y no esta marcado retirado; abortar con
   reason code `ctl_workdir_invalid` si falla. Revisar el default actual,
   que apunta a `/srv/orquesta-self/worktrees/pilot-remoto-1` (worktree
   documentado como stale): o se actualiza al worktree canonico o se exige
   env explicita sin default.
3. Guard de identidad en el servidor (`cmd/orquesta-server`): al arrancar,
   validar workdir/worktree y, si la validacion falla, publicar modo
   `degraded_identity` en `/api/status` y rechazar
   `POST /api/v0/autoprogramming/prepare-run` amplio con causa tipada, en
   vez de aceptar goals contra un arbol muerto (asi el caso S1 no se repite
   en silencio).
4. Registrar el remote GitHub canonico (o remote explicito adicional al
   bundle) en el receipt de deploy, para que S3 (origin=bundle temporal) sea
   detectable.

Criterio de cierre TAREA-F5:

- `/api/status` remoto publica `binary_sha256` del build nuevo y
  `startup_ready` con supervisor sin error en N muestras.
- Repro negativa: arranque por ctl contra un workdir inexistente/retirado
  aborta con `ctl_workdir_invalid`; arranque del server contra worktree
  invalido publica `degraded_identity` y rechaza prepare-run amplio, con
  test focal.
- Recibo durable del deploy con ref, sha, remote y resultado.

## Despues de este corte (no abrir ahora)

Con drain + binario nuevo verificado por API, el orden del analisis
estructural continua: repro/API de los patches 208 (write-sets solapados,
504, forced-stop), luego F1+F2 (reconciliador causal unico + actuador de
proceso para `runs/control`), luego F4 (auditoria de los 61 artefactos de
ejecucion versionados; los 4 nuevos del diff remoto de g01/g01-rework-1
NO se commitean como fuente, segun la revision), luego F6 (normalizador
neutral compartido de goal result). Los 57 `stale_running` y los 6 `invalid`
de `operator_active` se reconcilian tras el deploy, cuando el binario nuevo
pueda distinguir estado stale de proceso vivo; no se limpian a mano antes.
