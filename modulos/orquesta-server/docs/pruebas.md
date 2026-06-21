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
- `modulos/orquesta-server` prueba que el supervisor residente ejecuta ticks
  asincronos sin solaparse, coalescea un unico tick pendiente si otro pulso llega
  durante la supervision activa y puede preparar automejora en segundo plano sin
  bloquear nuevos pulsos.
- `modulos/orquesta-server` prueba que el Director residente opt-in ejecuta
  ticks asincronos por puerto, no arranca sin `ResidentDirectorEnabled`,
  despierta por wakeup no bloqueante aunque el ticker este lejos, coalescea un
  tick pendiente, persiste OK/error, recupera panic como error durable, aparece
  en status/operational status y cuenta como causa operativa para el
  self-watchdog.
- `cmd/orquesta-server` prueba que `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED`
  y `ORQUESTA_SERVER_RESIDENT_DIRECTOR_MAX_ACTIONS` se publican en
  `effective_config`, y que el adaptador del Director residente solo se inyecta
  con opt-in y stack disponible.
- `cmd/orquesta-server` prueba que `ORQUESTA_SERVER_AUTONOMY_ENABLED=true`
  activa el Director residente como default autonomo y que
  `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false` lo desactiva si se declara
  explicitamente.
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
  capacidad libre con cola visible, incluso con skips no terminales si queda
  hueco bajo el objetivo, y que no se prepara si la cola ya alcanzo el objetivo
  configurado.
- `modulos/orquesta-server` prueba que refs de request retryables descuentan
  presion de cola igual que refs de run y llegan al planner con evidencia
  compacta del guardian/promocion.
- `modulos/orquesta-server` prueba que un bloqueo posterior del supervisor no
  pisa la razon de una automejora aceptada y la conserva como intento pendiente.
- `modulos/orquesta-server` prueba que la auditoria de blockers de automejora
  redacta mensajes con rutas, HOME, tokens, prompts o transcripts sin perder
  refs opacas ni evidencias compactas.
- `modulos/orquesta-server` prueba que `StartupCheckPortV0` publica
  `startup_ready` con mensaje/evidencias y bloquea el arranque cuando la
  composicion no esta lista.
- `modulos/orquesta-server` prueba que la auditoria JSONL de `startup_check_*`
  usa summaries compactos y no persiste paths de proyecto, runtime o state.
- `modulos/orquesta-server` prueba que `/healthz` es liveness y
  `/api/v0/server/readiness` es readiness, responde 503 si startup no esta
  listo y no filtra paths ni runtime dirs.
- `modulos/orquesta-server` prueba que los fallos no fatales de persistencia de
  estado quedan visibles como `state_persist_failed`, sin filtrar paths del
  store, y que una escritura posterior confirmada devuelve la proyeccion a `ok`.
- `cmd/orquesta-server` prueba que el daemon espera readiness y no acepta
  `/healthz` como senal suficiente.
- T210 requiere que el transporte de stats no degrade progreso vivo:
  `go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-server ./modulos/orquesta-web ./cmd/orquesta-server`.
- T208 reconciliado requiere la bateria cruzada de guardian, servidor,
  autoprogramacion, worktree y Codex sin promover smokes reales.
- Prueba manual recomendada:
  - `go run ./cmd/orquesta-server run`
  - `curl http://127.0.0.1:8787/healthz`
  - `curl http://127.0.0.1:8787/api/v0/server/readiness`
  - `curl http://127.0.0.1:8787/api/v0/server/status`
  - `curl -i http://127.0.0.1:8787/api/status` solo para validar headers del
    alias legacy.
- Reconciliacion SRV-TASK-015 2026-06-20 requiere conservar verde:
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.
  Esta prueba valida que la sincronizacion documental del backlog no rompe el
  servidor residente ni su composition root; las pruebas historicas focales del
  productor causal OPES siguen citadas en `docs/tareas.md` y en el runbook.
  El rework
  `agent-ref-task-ref-review-rework-task-autoprogramming-874937b97f16-g01-162db2d326380eeab85c029bbfcfe285`
  usa ese mismo comando como prueba obligatoria y no declara pasadas las
  pruebas historicas si no se reejecutan en esta tarea.
- El rework de revision sobre rework
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-autoprogr-7b57471b0af67dc475be23b72a6c25c7`
  debe registrar como pasada solo
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si el
  comando obligatorio termina con exit code 0 en esta ejecucion.
- La correccion de revision
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-8f75b93913fef84c105ce29cf9734bca`
  usa el mismo criterio: solo puede declarar pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion.
