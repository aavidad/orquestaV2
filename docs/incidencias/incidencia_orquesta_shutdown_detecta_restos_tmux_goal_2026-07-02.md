# Incidencia: shutdown no debe ignorar restos tmux/app-server propios

Fecha: 2026-07-02

## Sintoma

En varias ejecuciones goal-first se observo que `server/shutdown forced=true`
podia declarar `shutdown_ready` aunque siguieran vivos restos del backend Codex
Goal: sesion `tmux`, socket y proceso `codex app-server`. El `AppGoalStateStore`
no siempre conservaba un goal `running`, por lo que la lectura de trabajo activo
se quedaba corta.

## Cierre aplicado

- El backend `app_server_tmux` implementa `ReadActiveShutdownWorkV0`.
- El detector solo bloquea por identidad propia: owner marker valido, socket
  propio o sesion tmux configurada/recuperable.
- Si encuentra restos, publica `goal_backend/backend_still_running` con
  evidencias de marker, socket, sesion y PID de pane vivo cuando exista.
- La composicion `orquesta-app-codex-stack` consulta lectores activos del
  backend Goal ademas del `AppGoalStateStore`.
- Los puertos Goal del servidor conservan esa lectura activa al envolver
  launcher/observer.

## Evidencia

```bash
go test -count=1 ./cmd/orquesta-server -run 'TestCodexAppServerTmuxBackendV0ReadActiveShutdownWork|TestServerSupervisorWithCodexGoalBackendV0ExponePuertosSoloConBackendV0'
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestStackShutdownActiveWorkReaderV0(BloqueaRestosBackendAunqueStateStoreNoRunning|BloqueaBackendRunningSinTimeoutLocal)'
go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack
```

## Residual

Procesos `codex app-server` nacidos fuera de la identidad tmux/owner marker de
Orquesta siguen perteneciendo al inventario operativo general. Este cierre no
mata procesos ajenos ni asume ownership por nombre de proceso.
