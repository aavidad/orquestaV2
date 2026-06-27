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
- `modulos/orquesta-server` prueba que `/api/v0/server/shutdown` congela el
  supervisor residente mientras no haya confirmacion completa y normaliza a
  `stop_pending` si la respuesta upstream conserva agentes vivos, checkpoints
  pendientes o runs pedidos sin todos los stops confirmados.
- `modulos/orquesta-server` prueba que el Director residente opt-in ejecuta
  ticks asincronos por puerto, no arranca sin `ResidentDirectorEnabled`,
  despierta por wakeup no bloqueante aunque el ticker este lejos, coalescea un
  tick pendiente, persiste OK/error, recupera panic como error durable, aparece
  en status/operational status y cuenta como causa operativa para el
  self-watchdog.
- `cmd/orquesta-server` prueba que guardar outbox pendiente de ciclo del
  Director despierta tambien al Director residente opt-in con causa
  `director_cycle_outbox_saved`, ademas del supervisor legacy.
- `modulos/orquesta-server` prueba que `GET /api/v0/server/status` no reutiliza
  un OK historico si el Director residente esta desactivado y hay
  `waiting_outbox`: publica `resident_director_disabled` con contador
  `waiting_outbox`.
- `modulos/orquesta-server` prueba que `/api/v0/operational-status/query`
  degrada el estado y crea blocker publico cuando el residente esta desactivado
  con outbox pendiente.
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
- `modulos/orquesta-server` prueba que la automejora goal-first genera
  `GoalWorkSpecV0` cuando hay launcher inyectado, y que con el flag activo pero
  sin launcher publica `goal_launcher_unavailable` sin preparar trabajo legacy.
  El reason publicado y el mensaje operativo estructurado conservan `goal_ref`
  para observar/cerrar el goal sin depender de semantica legacy de `run_ref`.
- `modulos/orquesta-server` prueba que, si hay `goal_refs` pendientes, el tick
  goal-first observa el goal por puerto antes de planificar otra tanda, y que
  `complete` queda como `goal_complete_pending_closure_validation` sin cierre
  automatico.
- `modulos/orquesta-server` prueba que la automejora goal-first persiste
  `GoalWorkSpecV0`, `GoalLaunchReceiptV0`, `GoalWorkResultV0` y
  `GoalClosureValidationV0`: un `complete` sin spec sigue pendiente y un
  `complete` con evidencia requerida por el spec pasa a
  `goal_closure_accepted`.
- `modulos/orquesta-server` prueba que `GET /api/v0/server/status` proyecta
  automejora goal-first como resumen publico compacto con spec/receipt/result/
  closure y sin filtrar objetivo, paths, comandos ni payloads completos.
- `modulos/orquesta-server` prueba que `/api/v0/operational-status/query`
  proyecta automejora goal-first como diagnostico compacto: conserva contadores
  residentes existentes, anade contadores `goal_*`, refs opacas, salud,
  actividad, bloqueo semantico para `invalid`/`blocked` y no filtra objetivo,
  paths, summaries completos ni payloads completos.
- `modulos/orquesta-server` prueba que `/api/v0/server/readiness` expone campos
  informativos de goal-first sin cambiar `ready` por un goal en curso y sin
  filtrar objetivo ni paths.
- `modulos/orquesta-server` prueba que `goal_launcher_unavailable` sin
  `goal_ref` ni spec/receipt/result/closure no se publica como goal activo en
  status ni readiness.
- `modulos/orquesta-server` prueba que `/api/v0/operational-status/query`
  marca `goal_launcher_unavailable` como salud/bloqueo de capacidad degradada,
  sin publicar refs ni contadores de goal activo.
- `cmd/orquesta-server` prueba que el wrapper de composicion solo expone
  `IdleSelfImprovementGoalLauncherPortV0`/`ObserverPortV0` cuando hay backend
  goal real configurado, y que los backends app-server mapean
  `thread/start`, `thread/goal/set`, `turn/start`, `thread/goal/get` y
  `thread/read` sin usar `codex exec`; al observar un goal terminal extrae
  `ORQUESTA_GOAL_RESULT_V0` o `orquesta_goal_result_v0.json` para
  artefactos/evidencias de cierre.
- `cmd/orquesta-server` prueba que el preflight app-server usa
  `thread/loaded/list`, conserva puertos goal-first degradados si el socket o
  standalone de Codex faltan, y publica issue codes compactos sin stderr crudo.
- `modulos/orquesta-server` prueba que los contadores historicos
  `supervisor_error_ticks` y `resident_director_error_ticks` siguen visibles
  como telemetria, pero no degradan estado ni crean blockers si el supervisor y
  el Director residente ya publican estado actual recuperado; un error vigente
  sigue degradando aunque el contador historico este a cero.
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
- `modulos/orquesta-server` prueba que el guard remoto permite sin token la
  excepcion exacta de lectura `/api/v0/apps/intake/guided-turn`, y conserva el
  prefijo dinamico `/api/v0/apps/{app_ref}/changes` como mutacion protegida.
- `modulos/orquesta-server` prueba que los fallos no fatales de persistencia de
  estado quedan visibles como `state_persist_failed`, sin filtrar paths del
  store, y que una escritura posterior confirmada devuelve la proyeccion a `ok`.
- `modulos/orquesta-server` prueba que un `response_write_failed` queda visible
  con contador y ultima fecha, y que una respuesta posterior correcta limpia el
  blocker vigente sin borrar el contador historico.
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
- El agente de reemplazo
  `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-0b2-f385b5e4533d62b5c7d2a64b70950490`
  usa el mismo criterio estricto: en su ACK solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 durante esta ejecucion.
- La revalidacion OrquestaV2
  `agent-ref-assessment-task-autoprogramming-30d589fce14c-g01-3dff4105035295423d6bf932af420185`
  conserva SRV-TASK-015 como no-op documental salvo regresion causal nueva y
  solo declara pasada la prueba obligatoria si el comando exacto
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` termina
  con exit code 0 en esta ejecucion.
- El agente externo
  `agent-ref-task-ref-review-rework-task-autoprogramming-30d589fce14c-g01-8bb7a3491960a73cba493ed199be20e1`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion.
- La correccion
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-dacb62e5ec1ae158a640baa674e72940`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion.
- La correccion externa
  `agent-ref-task-ref-review-rework-task-autoprogramming-abea33163b68-g01-04bebd80bd7c5afa1ff0cde8543ec8ec`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion.
- La correccion externa
  `agent-ref-task-ref-review-rework-task-autoprogramming-99f93b5dadeb-g01-46ac951607ee8f489f914ee25e92cb88`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion.
- El assessment externo
  `agent-ref-assessment-task-ref-review-rework-task-autoprogramming-99f93b5dadeb-g01-46a-de83dfc8cce6e142d5f9b4a9f2a47b24`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion.
- La correccion de rastro SRV-TASK-024
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-autoprogr-dc31be1188c5569b01263b5f388f0788`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion. Las ejecuciones
  previas de la misma request que fallaron por entorno no se reclasifican como
  pasadas.
- El assessment externo de la correccion SRV-TASK-024
  `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-autoprogr-dc3-0ff5f84c9556672a698237df3988111c`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion y mantiene
  SRV-TASK-024 abierto salvo codigo y smoke causal.
- El assessment externo de reemplazo
  `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-autoprogr-dc3-36f24a2209e93ca38f4443ce22f527cf`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- La correccion de entrega
  `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-autoprogr-dc3-a5a32e27fa8bf23a4b29ee07490e9a10`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- La correccion de entrega tras revision
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-autoprogr-72998ee36295059452b54a9eb4e875ee`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- El assessment externo de esa correccion
  `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-autoprogr-729-9fc3f13d44dbaff94cfe5adff747a3e9`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- La correccion externa de entrega tras revision
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-cfd9c13a92a5cd21fd6c622e6f405e46`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- El assessment externo de la correccion posterior
  `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-cfd-c006963b24043636d557f12d4659b4a3`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- El assessment externo de reemplazo
  `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-cfd-791ce4584f28597b364194874b2875a4`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- La correccion externa de entrega tras revision
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-99773dd28e4df93406b2912127997d04`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- La correccion externa de entrega tras revision
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-30ffd078ad6ee9febe4c50ac0bd29891`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- El assessment externo de esa correccion
  `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-30f-c4501a4d6b170b706b55f9a3dd4a44a6`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- La correccion externa de entrega tras revision
  `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-30f-aad45cf56f890e04e96c6b86b7c9bc7b`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- El assessment externo de esta correccion
  `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-30f-6842075f2c746477bb06b52d2c34e0c5`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- La correccion externa de entrega tras revision
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-9ba51f68ffa908088c91ca219cb73b84`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- La correccion externa de entrega tras revision
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-5f1990c9bda61f11e4c4572d7d8ec5f9`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- La correccion externa de entrega tras revision
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2f664d21881795ee2d8392327887480f`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- El assessment externo de la correccion tras revision
  `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2f6-4a88d678723f47ef86524e438127a1e1`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- La correccion externa de esa evaluacion
  `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2f6-d07cfaec65f49155f8328d0512377fd7`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- El assessment externo de reemplazo
  `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2f6-20f9ab529a94f082f863a30e7a9275dd`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- La correccion externa de entrega tras revision
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-d20f765b96bef254281f4683e6ab3480`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- La correccion externa de entrega tras revision
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-9e9d33b777c6565a38a9b5a6eb7529a2`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- La correccion externa de entrega tras revision
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2b5521cd1c1dbd4767cebb3d9c1684d7`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- La correccion externa de entrega tras revision
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-15e93ad10820843acec443a42cbac3c7`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- La correccion externa de entrega tras revision
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-0f0bdd6085651f057155f6a385ccea82`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- La correccion externa de entrega tras revision
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-d4aec6d93452fd3b4f10a49c0bca437c`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- La correccion externa de entrega tras revision
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-403a707ea7e18c0c29793d3df9266ede`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
- La correccion externa de entrega tras revision
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-026aefcaa6977288909f6693516c13c1`
  aplica el mismo criterio estricto: solo declara pasada
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server` si ese
  comando exacto termina con exit code 0 en esta ejecucion; no hereda pruebas
  previas ni cierra SRV-TASK-024 sin codigo y smoke causal.
