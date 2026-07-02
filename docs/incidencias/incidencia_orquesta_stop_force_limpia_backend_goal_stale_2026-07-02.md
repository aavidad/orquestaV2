# Incidencia: stop --force no limpiaba backend Goal con daemon muerto

Fecha: 2026-07-02.

ID inventario: `BUG-ORQ-20260702-111`.

Relacionado con: `BUG-ORQ-20260702-096`, `BUG-ORQ-20260702-101` y el
residual OPES de HTTP muerto tras `startup_ready`.

## Sintoma

En una oleada OPES, Orquesta llego a publicar `startup_ready`, pero despues el
HTTP dejo de responder. El statefile seguia con `status=running`,
`startup_ready=true` y un PID ya muerto. A la vez quedaban vivos recursos del
backend Codex Goal `app_server_tmux` configurado para esa instancia.

`status` ya reconciliaba el statefile como `stale`, pero `stop` fallaba antes
en `verifyLiveDaemonIdentityV0` con `daemon_identity_unavailable`. Por tanto
el operador no tenia una ruta CLI segura para pedir limpieza del backend Goal
propio cuando el daemon HTTP ya no existia.

## Causa

La ruta de parada exigia identidad HTTP viva antes de ejecutar cualquier
reconciliacion. Esa regla es correcta para evitar matar procesos ajenos, pero
dejaba sin salida el caso en el que el PID registrado no esta vivo y la sesion
tmux/socket pertenecen a la configuracion local de Orquesta.

## Cierre

`stopServerCommandV0` mantiene el comportamiento conservador por defecto:
si falta identidad HTTP y no hay `--force`, falla y no limpia nada.

Con `--force --reason ...`, si el PID del snapshot no esta vivo, la ruta:

- reconcilia el statefile mediante `MarkServerProcessStaleStateV0`;
- invoca la limpieza existente de `app_server_tmux` configurado;
- no envia senales a PID vivo ni reclama procesos ajenos;
- escribe una salida publica `stop forced stale`.

## Guardas de seguridad

El cierre conserva dos limites:

- si el endpoint HTTP sigue alcanzable pero la identidad no coincide
  (`daemon_identity_mismatch`), `stop --force` no limpia el backend configurado;
- si `/api/v0/server/shutdown` rechaza por `backend_still_running` o
  `active_goals_present`, `--force` no convierte ese rechazo en senal local.

`--force` queda reservado para snapshot stale confirmado o fallos de transporte
sin evidencia publica de trabajo vivo.

## Evidencia

Pruebas locales:

```bash
go test -count=1 ./cmd/orquesta-server -run 'TestStopServerCommandV0(ForceConDaemonMuertoLimpiaBackendGoalConfigurado|SinForceConDaemonMuertoNoLimpiaBackendGoal|ForceConDaemonIdentityMismatchNoLimpiaBackendGoal)|TestShutdownRequestErrorAllowsSignalV0SoloConTimeoutYEstadoDrenado|TestCleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0|TestStatusServerCommandV0ReconciliaStatefileConPIDMuerto|TestWaitForStateHealthyV0MarcaStaleSiServidorMuereTrasReadinessV0|TestCodexAppServerTmuxBackendV0ReadActiveShutdownWorkDetectaRestosPropiosV0'
go test -count=1 ./cmd/orquesta-server
```
