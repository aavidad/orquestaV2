# Handoff Codex F3 remoto - 2026-07-10

Alcance ejecutado: reparación offline de TAREA-F3 por la excepción documentada
para Codex directo. No se lanzaron goals, no se observaron ni mutaron secretos,
no se ejecutaron drain/deploy/restart, no se señalaron procesos vivos y no se
tocó `uso-app` ni otra aplicación.

## Resultado

- El drain inventaría `/proc` y tmux con PID, PPID, PGID, sesión de proceso,
  starttime, cmdline redactado, exe, cwd, pane/sesión tmux y markers de
  socket/FIFO/owner. La clasificación es `drainable`, `protected` o
  `ambiguous`; la protección se propaga causalmente a todos los descendientes
  de `uso-app`.
- El backup precede cualquier posible señal y usa por defecto
  `/srv/orquesta-self/backups/drain-<timestamp>`. Copia checkpoints,
  `orquesta_goal_result`, `agent_ack`, logs y estado sin borrar el origen.
- El shutdown HTTP genera JSON parseable con autoridad del Director,
  idempotencia y limpieza de backends. TERM, espera, tmux exacto y KILL final
  revalidan identidad; ninguna mutación usa nombre o patrón.
- El receipt v1 conserva before/after, identidades, clasificación, acciones
  estructuradas, backup, evidencia de protección de `uso-app` y estado
  `clean|residual|refused`.
- Deploy, nightly y smoke compuesto consumen el perfil aislado común. El nuevo
  runner Go divide paquetes en lotes con timeout y receipt, sin invocar una
  tanda monolítica.

## Evidencia offline de esta sesión

- `bash -n` de todos los scripts F3 y consumidores: verde.
- `scripts/test_orquesta_server_drain.sh`: verde con `/proc`/tmux/curl/kill
  sintéticos; cubre dry-run, árbol protegido, ambigüedad, backup previo, orden,
  payload JSON, SIGKILL gobernado, clean y residual.
- `scripts/test_orquesta_test_batches.sh`: verde; cubre partición 2/2/1,
  timeout por lote, receipt verde y conservación de evidencia tras fallo.
- `scripts/test_orquesta_server_deploy.sh`: verde tras limpiar del entorno las
  URLs/ref externas para mantener el test estrictamente offline.
- Guards Go focales de contratos de scripts: verdes con cache de módulos local.

Bloqueos honestos del sandbox:

- La ruta pedida `/srv/orquesta-self/runtime/test-cache` es read-only en esta
  sesión. Las ejecuciones se aislaron bajo la raíz permitida
  `/srv/orquesta-self/runtime/tmp-deploy/test-cache`; el operador debe repetir
  en la ruta canónica.
- `scripts/test_orquesta_smoke_nightly.sh` alcanza el caso de notificación
  local, pero el sandbox rechaza crear el socket de loopback con `EPERM`. No se
  declara verde ese test aquí ni se debilitó su cobertura.
- Por orden del operador no se ejecutaron los dos pases amplios reales ni el
  receipt de drain real. Son el paso de revisión/operación descrito en
  `orquesta_server_drain_f3_2026-07-10.md`.

Revisión adversarial: el camino activo se rehúsa completo ante cualquier
identidad ambigua; cambiar PID manteniendo el nombre no supera el CAS; un hijo
sin nombre `uso-app` sigue protegido por PPID; una sesión tmux con pane o
created distinto no puede recibir `kill-session`; un fake que ignora señales
termina `residual`, nunca `clean`; el KILL solo aparece detrás del helper final
revalidado.

estado_final: ready_for_operator_f3
