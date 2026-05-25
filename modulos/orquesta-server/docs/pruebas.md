# Pruebas

- `go test -count=1 ./modulos/orquesta-server`
- `go test -count=1 ./cmd/orquesta-server`
- `cmd/orquesta-server` prueba que `ORQUESTA_OPES_BASE_URL` activa un executor
  `domain_work` OPES opt-in y que sin esa variable queda apagado.
- `cmd/orquesta-server` prueba que `ORQUESTA_DOMAIN_WORK_FILE_ENABLED=1`
  activa un creator durable file-based para `create_job`, sin habilitar
  `submit_artifact` ni `DomainDelivery`.
- `cmd/orquesta-server` prueba que `ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL` activa
  un backend HTTP neutral para `create_job`, `submit_artifact` y
  `DomainDelivery`.
- `cmd/orquesta-server` prueba que OPES, backend file y backend HTTP neutral de
  `domain_work` no pueden activarse a la vez.
- `cmd/orquesta-server` prueba que los umbrales productivos por defecto para
  agentes Codex no vuelven a valores agresivos de debug, y que siguen siendo
  sobreescribibles por entorno.
- `cmd/orquesta-server` prueba que el supervisor residente expone
  `ORQUESTA_SERVER_SUPERVISOR_MAX_TICKS` y
  `ORQUESTA_SERVER_ALLOW_REPEATED_RUNS`.
- `modulos/orquesta-server` prueba que un panic del supervisor residente queda
  registrado como error observable y que el siguiente tick recupera metricas de
  ejecucion, skips y cola sin bloquear el loop.
- `cmd/orquesta-server` prueba que el planner de automejora salta tareas ya
  visibles en cola y anade una tarea scanner cuando hay capacidad libre para
  descubrir nuevos huecos.
- `cmd/orquesta-server` prueba que `provider_auth_blocked` se detecta desde
  runs activos con agente perdido, pregunta al director y assessment critico,
  y que la automejora residente no prepara trabajo nuevo mientras ese bloqueo
  este vigente.
- `modulos/orquesta-server` prueba que `NormalizeConfigV0` conserva
  `SupervisorCommand.MaxTicks` positivo y usa default solo para valores vacios o
  invalidos.
- `modulos/orquesta-server` prueba que la automejora se prepara tambien por
  capacidad libre con cola visible, y que no se prepara si la cola ya alcanzo el
  objetivo configurado.
- `modulos/orquesta-server` prueba que un bloqueo posterior del supervisor no
  pisa la razon de una automejora aceptada y la conserva como intento pendiente.
- `modulos/orquesta-server` prueba que `StartupCheckPortV0` publica
  `startup_ready` con mensaje/evidencias y bloquea el arranque cuando la
  composicion no esta lista.
- `modulos/orquesta-server` prueba que `/healthz` es liveness y
  `/api/v0/server/readiness` es readiness, responde 503 si startup no esta
  listo y no filtra paths ni runtime dirs.
- `cmd/orquesta-server` prueba que el daemon espera readiness y no acepta
  `/healthz` como senal suficiente.
- Prueba manual recomendada:
  - `go run ./cmd/orquesta-server run`
  - `curl http://127.0.0.1:8787/healthz`
  - `curl http://127.0.0.1:8787/api/v0/server/readiness`
  - `curl http://127.0.0.1:8787/api/status`
