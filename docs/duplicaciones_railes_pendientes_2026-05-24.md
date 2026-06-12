# Duplicaciones de rails pendientes 2026-05-24

Objetivo: evitar que reglas equivalentes queden copiadas en muchos ficheros y
se ajusten de forma divergente.

Indice vivo: `docs/rails/registro_railes_2026-05-24.md`.

## Cerrado en este corte

- `modulos/orquesta-rails/text_policy_v0.go`: politica comun para texto
  sensible.
- `modulos/orquesta-core-workflow`: los validadores de capacidad, agentes,
  entregas, review, rework, cierre, eventos, outbox, workflow tasks y AppSpec
  usan el helper comun.
- `modulos/orquesta-director-agent`: el contrato JSON del agente director usa
  el helper comun para el mismo rail de texto.

## Candidatos detectados

- `modulos/orquesta-context/context_bundle_validation_v0.go`: lista propia de
  terminos prohibidos. Revisar si debe pasar a helper local permisivo por campo.
- `modulos/orquesta-director-agent/director_decision_helpers_v0.go`: lista
  propia del contrato JSON del agente director. Ya esta concentrada en un
  fichero, pero conviene alinearla con la politica permisiva del core.
- `modulos/orquesta-persistence/outbox_ledger_helpers_v0.go` y
  `modulos/orquesta-persistence/persistence_repository_helpers_v0.go`: listas
  defensivas de persistencia. Revisar si son frontera real o falsos positivos
  historicos.
- `modulos/orquesta-director-scheduler/scheduler_tick_validation_v0.go` y
  `modulos/orquesta-director/outbox_dispatch_cycle_validation_v0.go`: listas de
  seguridad de ciclo director/scheduler. Validar con matriz externa antes de
  endurecer o relajar.
- `modulos/orquesta-core-replanner/replan_proposal_validation_v0.go`: lista
  propia de replan. Revisar alias reparables y refs opacas.
- `modulos/orquesta-app-change-director-source/readiness_v0.go`: lista de
  readiness de autoplanning. Comprobar si corta producto/adaptador por palabras
  normales.
- `modulos/orquesta-director-operativo/validation_v0.go`: lista compacta de
  sensibles. Mantenerla solo si corta valores reales, no vocabulario normal.
- `modulos/orquesta-runtime-required-test/local_command_executor_v0.go`: rail
  de ejecucion local. Es frontera de efecto externo; no relajar sin test
  especifico.
- `modulos/orquesta-runtime-required-test/local_command_executor_v0.go` y
  `cmd/orquesta-server/required_test_runner_stack_v0.go`: ademas del rail de
  ejecucion, falta propietario de redaccion/retencion para stdout/stderr de
  tests requeridos. No debe duplicar auditoria JSONL ni tail de logs Codex.
- `modulos/orquesta-observability/orquesta_event_types_v0.go`: marcadores de
  privacidad/observabilidad. Puede seguir separado si no bloquea workflow.
- `modulos/orquesta-app-codex-stack/spec_external_context_v0.go`: lista local
  de nombres sensibles al construir refs de contexto externo. Revisar si debe
  usar helper comun o mantenerse como normalizador de refs de packet.
- `modulos/orquesta-app-codex-stack/review_rework_replan_evidence_v0.go`:
  lista local de tokens ambientales y fragmentos sensibles para evidencias de
  rework/replan. Clasificar si protege evidencia causal real o duplica el rail
  comun.
- `modulos/orquesta-operator-mcp/operator_ref_validation_v0.go` y
  `modulos/orquesta-operator-mcp/operator_validation_v0.go`: wrappers locales
  sobre `orquesta-rails` para refs/preguntas de operador. Mantener como frontera
  de operador si la matriz demuestra que no corta refs opacas validas.
- `modulos/orquesta-core-leases/lease_policy_v0.go`: rail de leases/heartbeat.
  Revisar como frontera de ciclo de vida, no como rail de detalle generico.
- `cmd/orquesta-server` auditoria JSONL y `modulos/orquesta-observability`:
  ambos documentan privacidad/proyecciones de auditoria. Antes de anadir visor,
  retencion o captura opt-in de payload completo, fijar propietario unico de
  redaccion/retencion para no duplicar rails de datos sensibles.
- `modulos/orquesta-domain-work-sql` y futuros bundles SQL reales: pueden
  duplicar reglas sobre DSN, driver, placeholders y unique violation. El paquete
  neutral debe conservar contract tests y el bundle opt-in debe poseer driver,
  schema, DSN y sanitizado de secretos.
- Scripts de smoke reales: las guardas de confirmacion estan repartidas por
  script y matriz. Conviene un catalogo comun para distinguir offline/fake,
  real con proveedor, OPES temporal, DB temporal y efectos externos.
- `cmd/orquesta-server/detail_rails_env_v0.go` y
  `modulos/orquesta-rails/detail_field_policy_v0.go`: el servidor deja
  `ORQUESTA_DETAIL_PROHIBITED_RAILS=off` por defecto para priorizar ejecucion
  real y documentar falsos positivos como deuda. El propietario del default/scope
  debe quedar separado del propietario de la politica comun antes de reactivar
  restricciones por campo.
- Transporte MCP real: `cmd/orquesta-server/mcp_real_transport_v0.go` y
  `mcp_real_smoke_v0.go` poseen el transporte JSON-RPC HTTP de composicion;
  `orquesta-mcp` conserva el registro/tooling puro. Si se expone en modo
  residente, evitar duplicar validacion/auditoria/payloads con los bridges HTTP.
- Compactacion de arranque: `cmd/orquesta-server/startup_check_compaction.go`
  interpreta `codex_last_message.txt` para inferir ACK completado, mientras
  `orquesta-runtime-codex` ya posee contrato/validador de `agent_ack.json`.
  Debe quedar una unica fuente estructurada para ACK terminal.
- Planner de backlog: `cmd/orquesta-server/idle_self_improvement_backlog_planner_v0.go`
  tiene otra lectura de `agent_ack.json` que solo mira `status=completed`. Debe
  compartir fuente estructurada/correlada con runtime Codex y no cerrar backlog
  por ACK ambiguo.
- Roadmap MCP: `modulos/orquesta-mcp/project_roadmap_descriptors_v0.go` y
  `shared_contracts_descriptors_v0.go` duplican estado de roadmap/backlog que ya
  vive en docs vigentes. Si siguen estaticos, necesitan freshness y refs de
  fuente para no divergir del backlog Txx.
- Calidad `domain_work`: `modulos/orquesta-app-codex-stack` concentra reglas de
  placeholders/extension para paquetes de tema. El propietario debe ser una
  politica de dominio/adaptador por puerto, no una lista local del stack.
- Guardas de ola Codex: `cmd/orquesta-server/codex_director_wave_command_v0.go`
  tiene prompts y defaults no estrictos para write-set/tests, mientras el
  contrato de autoprogramacion espera write-set cerrado y ACK con pruebas
  requeridas. El propietario de la excepcion debe ser el director, no texto libre
  del agente.
- Startup cleanup: `cmd/orquesta-server/config.go` fija `forced_stop` por
  defecto y `startup_check_compaction.go` compacta cola/control. Ese rail debe
  coordinarse con reconciliacion de agentes vivos y ACK estructurado para no
  duplicar politicas de terminalidad.
- ACK terminal Codex: `modulos/orquesta-runtime-codex/codex_ack_validation_v0.go`
  normaliza campos desde la spec antes de validar, mientras startup, planner y
  observation source necesitan una decision estricta sobre si un ACK trae refs
  propias o solo defaults de compatibilidad.
- Parser de backlog: `cmd/orquesta-server/idle_self_improvement_backlog_planner_v0.go`
  decide estado cerrado y required tests desde Markdown con heuristicas locales.
  Debe compartir contrato de estado/verificacion con el backlog vivo para no
  perder scripts, `git diff --check` o pruebas documentales.
- T43 cerrado para scanners: las requests de scanner ya llevan epoch documental,
  hash/linea por doc y reserva opaca de write-set; el ACK con foto obsoleta no
  cierra silenciosamente y el planner expone colision compacta para rebase/merge.
  Sigue pendiente el parser general de pruebas/estado documental de T37.
- Observaciones Codex: `modulos/orquesta-runtime-codex/codex_delivery_observation_v0.go`
  conserva una lista local de terminos sensibles/operativos solo usada como
  detector de rail pendiente. Si debe proteger produccion, debe usar politica
  comun por campo; si no, debe quedar como deuda historica eliminable.
- `modulos/orquesta-app-codex-stack/director_decision_contract_v0.go`: el
  prompt de `director_decisions.json` ya no conserva lista textual de terminos
  prohibidos desde el cierre T62 del 2026-05-25. La frontera queda alineada con
  `orquesta-rails`: refs opacas y vocabulario operativo pasan; valores reales,
  rutas privadas, secretos efectivos, prompts/transcripts crudos y payloads
  completos siguen bloqueados.
- `modulos/orquesta-runtime-codex-delivery` y
  `modulos/orquesta-director-agent-file-source`: el sidecar
  `director_decisions.json` se resuelve desde `agent_ack.json`, pero la
  correlacion, hash, consumo idempotente y conflicto durable no tienen un unico
  propietario visible.
- `modulos/orquesta-cli/command_flags_v0.go`: la entrada JSON comun de CLI lee
  `stdin`/`--input` sin limite compartido ni clasificacion de origen. No debe
  duplicar rails de HTTP ni salida publica; necesita propietario de entrada
  local antes de construir requests.
- `modulos/orquesta-runtime-codex`, `modulos/orquesta-runtime-codex-delivery`,
  `modulos/orquesta-app-codex-stack` y el planner del servidor leen ficheros de
  control Codex con politicas distintas. Correlacion, write-set y sidecar causal
  ya tienen tareas vecinas; falta owner comun de tamano/redaccion para
  `agent_ack.json`, `director_decisions.json` y checkpoints.
- Detalle runtime de `/ops`: `modulos/orquesta-app-codex-stack` vuelve a leer
  control files/logs de agentes y `modulos/orquesta-web` los renderiza en
  bloques de detalle. Aunque T143 gobierna validadores/fuentes de observacion y
  T211 frescura del dashboard, falta owner unico de presupuesto/redaccion para
  contenido operacional visible. Backlog asociado: `T227
  ops-runtime-detail-redaction-and-budget`. Cerrado localmente el 2026-05-27:
  `/ops` consume envelopes redactados por fichero, refs compactas y tail
  acotado; no renderiza prompts, packets, ACKs ni logs crudos.
- Reloj y generacion de refs: cerrado 2026-05-26 para T161 con owner comun en
  `modulos/orquesta-runtime/clock_ref_policy_v0.go`. `cmd/orquesta-server`,
  `modulos/orquesta-server`,
  `modulos/orquesta-runtime`, `modulos/orquesta-runtime-codex-delivery`,
  `modulos/orquesta-app-codex-stack`, `modulos/orquesta-web` y
  `modulos/orquesta-cli` conservan usos de reloj para evidencia o duraciones,
  pero las fronteras focales detectadas ya usan reloj/generador compartido,
  colision observable y fallback degradado sin `UnixNano` silencioso.
- Identidad publica de mutaciones: web, CLI, MCP, gateways y comandos del
  servidor repiten normalizacion de `request_id`, `correlation_id`,
  `idempotency_key` y `X-Correlation-ID`. Falta una politica compartida que
  distinga lecturas de mutaciones reintentables y no duplique reglas por
  cliente. Backlog asociado: `T236 public-mutation-identity-contract`.
- Promocion/restauracion del guardian: T214 cubre copia atomica y manifiesto,
  T220 retencion/replay, T221 concurrencia de promocion y T228 ownership de
  address. Cierre local 2026-05-27 para T237: el lease durable bajo `state_dir`
  serializa `check-promote`, `restore-last-good`, `shutdown-server` y el
  servidor trata busy/lost como bloqueo retryable sin promocion exitosa ni
  reparacion Codex automatica.
- Politicas de autoprogramacion: cierre local 2026-05-27 para T238. Review gate
  queda dividido entre resultado/acciones, issue codes y stems canonicos; el
  particionado queda dividido entre plan base, steps/bloqueos por trabajo vivo y
  matching de paths/aliases. No quedan ficheros Go productivos del modulo por
  encima de 300 lineas en ese alcance.
- Shutdown checkpoint Codex: runtime Codex define el schema de request/ACK,
  `orquesta-app-codex-stack` genera `checkpoint_ref` por run/agente y
  `cmd/orquesta-server` expone stop/shutdown. Falta un owner visible para la
  identidad causal por intento de shutdown y para rechazar ACKs stale entre
  mutaciones distintas del mismo agente. Backlog asociado: `T239
  codex-shutdown-checkpoint-attempt-correlation`.
- Recuperacion por auth de proveedor en automejora idle: `cmd/orquesta-server`
  detecta `provider_auth_blocked`, `modulos/orquesta-server` proyecta estado
  publico/auditoria y `orquesta-app-codex-stack` conoce proveedor/runtime. Falta
  owner comun para accion de operador, retry idempotente y no duplicacion de
  scanners cuando las credenciales se recuperan. Backlog asociado: `T241
  idle-self-improvement-provider-auth-recovery-contract`.
- Replan por quality gate en `orquesta-orchestration-core`: el provider mezcla
  seleccion de evidencias, validacion causal, decision de replan y candidatos en
  un fichero productivo por encima del limite operativo. Backlog asociado:
  `T253 orchestration-core-quality-gate-replan-provider-split`. Cierre
  2026-05-27: T253 separa proyeccion causal, followups y politica local del
  provider; rework sincroniza backlog y rail observado con ese cierre, sin
  relanzar agente padre. No abrir duplicados salvo regresion nueva.
- Numeracion humana de backlog: las tareas `T248`, `T249`, `T250` y `T251`
  cubren identidad de escaneos, dedupe y canonicalizacion, pero la reserva del
  numero visible `Txx` queda como rail separado. El backlog ya conserva varios
  `## T250` distintos; el owner debe reservar `task_id_ref` causal antes de
  escribir, bloquear colisiones y preservar historico por linea/hash sin
  renumerar. Backlog asociado: `T252
  backlog-task-number-reservation-policy`. Cierre 2026-05-27: el servidor
  residente emite `task_id_ref`, reserva opaca y rango `Txx` por request/epoch,
  y publica `backlog_task_number_collision` con instancias por linea/hash sin
  renumerar historico.
- Rework T249 2026-05-27: la correccion
  `agent-ref-task-ref-review-rework-task-autoprogramming-1c574ac7c424-g01-5cba0f0171e79bd80bbbcf4c49f5bc8a`
  no abre owner nuevo ni duplica backlog. T249 sigue como contrato de identidad
  de entradas de scanner; T250/T251/T252 conservan sus fronteras de dedupe,
  preflight y reserva de numero humano.
- Revalidacion scanner 2026-05-27: el paquete
  `agent-ref-assessment-task-autoprogramming-8f23d8e2a22e-g01-556dde497c2e0ff3e929391ffeb28c93`
  no abre duplicacion nueva. T44, T249, T250, T251, T252, T254, T255, T256 y
  T257 siguen siendo owners visibles; el contexto `ref_only` se resuelve por
  lectura local/evidencia ACK y las refs de worktree/branch se conservan opacas.
- Revalidacion scanner retry 5465c1 burst 002 2026-05-27: el paquete
  `agent-ref-assessment-task-autoprogramming-87d5c800f528-g01-4fe334168e77c5b5a5a29a19c299e7e8`
  no abre duplicacion nueva. El patron ya esta cubierto por owners visibles:
  T44 para contexto obligatorio `ref_only`, T249/T250/T251 para identidad,
  dedupe, preflight y alcance de pruebas de scanners, T252/T254 para ids
  humanos `Txx`, y T253/T255/T256/T257/T258 para splits recientes. Las refs
  `worktree_ref` y `branch_ref` se conservan opacas.
- Reconciliacion T198 2026-05-27: el paquete
  `agent-ref-task-autoprogramming-974732911968-g01` no abre duplicacion nueva.
  T198 queda como owner visible de `descriptor_source` para resources MCP, con
  `orquesta-mcp` como owner del envelope/registro y los modulos
  `orquesta-operator-mcp`, `orquesta-observability`, `orquesta-governance` y
  `orquesta-core` como fuentes canonicas documentadas. El pendiente residual se
  trata como stale reconciliado, no como nueva tarea.
- Reconciliacion adicional T198 2026-05-27: el paquete
  `agent-ref-task-autoprogramming-c3678e9bc306-g01` confirma que el patron
  vuelve por backlog stale, no por brecha nueva. Mantiene T198 como owner,
  conserva refs de worktree/branch opacas, resuelve `ref_only` por ACK y no
  duplica owners ni codigo.
- Revalidacion retry 69e1ba 2026-05-27: el paquete
  `agent-ref-assessment-task-autoprogramming-8f23d8e2a22e-g01-697704c71decfb03744436e559856b7f`
  no abre duplicacion nueva. T44, T249, T250, T251, T252, T253, T254, T255,
  T256, T257 y T258 siguen como owners visibles; el contexto `ref_only` se
  resuelve por lectura local/evidencia ACK y las refs de worktree/branch se
  conservan opacas.
- Revalidacion retry a3abed 2026-05-27: el paquete
  `agent-ref-task-autoprogramming-30b4d5ecfe9b-g01` no abre duplicacion nueva.
  T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y T258 siguen como
  owners visibles; el contexto `ref_only` se resuelve por lectura
  local/evidencia ACK y las refs de worktree/branch se conservan opacas.
- Revalidacion retry 69e1ba burst 002 2026-05-27: el paquete
  `agent-ref-assessment-task-autoprogramming-8f23d8e2a22e-g01-89e006236c585aa247a81664e7b037e1`
  no abre duplicacion nueva. T44, T249, T250, T251, T252, T253, T254, T255,
  T256, T257 y T258 siguen como owners visibles; el contexto `ref_only` se
  resuelve por lectura local/evidencia ACK y las refs de worktree/branch se
  conservan opacas.
- Revalidacion retry 5465c1 burst 002 2026-05-27: el paquete
  `agent-ref-assessment-task-autoprogramming-87d5c800f528-g01-219c350ad3df01efd266321f376e4f7b`
  no abre duplicacion nueva. T44, T249, T250, T251, T252, T253, T254, T255,
  T256, T257 y T258 siguen como owners visibles; el contexto `ref_only` se
  resuelve por lectura local/evidencia ACK y las refs de worktree/branch se
  conservan opacas.
- Revalidacion burst 002 1db27 2026-05-27: el paquete
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-1db27b63b7ddb14755938b05f1eff32f`
  no abre duplicacion nueva. El patron ya esta cubierto por T44 para contexto
  obligatorio `ref_only`, T249/T250/T251 para identidad, dedupe/preflight y
  alcance de pruebas de scanners, T252/T254 para ids humanos, reserva,
  colisiones, aliases y read model, y T253/T255/T256/T257/T258 para splits
  concretos recientes. Las refs `worktree_ref` y `branch_ref` se conservan
  opacas.
- Revalidacion burst 002 c10c 2026-05-27: el paquete
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-c10cb19de09867b08fe7a7c947dc4467`
  no abre duplicacion nueva. T44, T249, T250, T251, T252, T253, T254, T255,
  T256, T257 y T258 siguen como owners visibles; el contexto `ref_only` se
  resuelve por lectura local/evidencia ACK y las refs de worktree/branch se
  conservan opacas.
- Revalidacion burst 002 562c49 2026-05-27: el paquete
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-562c496eddcc7c55bd3b622c5e825749`
  no abre duplicacion nueva. T44, T249, T250, T251, T252, T253, T254, T255,
  T256, T257 y T258 siguen como owners visibles; el contexto `ref_only` se
  resuelve por lectura local/evidencia ACK y las refs de worktree/branch se
  conservan opacas.
- Revalidacion burst 002 f8e40 2026-05-27: el paquete
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-f8e40a7bedded7d2878462d370a72a30`
  no abre duplicacion nueva. T44, T249, T250, T251, T252, T253, T254, T255,
  T256, T257 y T258 siguen como owners visibles; el contexto `ref_only` se
  resuelve por lectura local/evidencia ACK y las refs de worktree/branch se
  conservan opacas.
- Revalidacion burst 002 9fa6 2026-05-27: el paquete
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-9fa6b01dc86ff93d9228ca93a474de5c`
  no abre duplicacion nueva. El patron ya esta cubierto por T44 para contexto
  obligatorio `ref_only`, T249/T250/T251 para identidad, dedupe/preflight y
  alcance de pruebas de scanners, T252/T254 para ids humanos, reserva,
  colisiones, aliases y read model, y T253/T255/T256/T257/T258 para splits
  concretos recientes. Las refs `worktree_ref` y `branch_ref` se conservan
  opacas.
- Revalidacion retry 598e4b burst 002 2026-05-27: el paquete
  `agent-ref-task-autoprogramming-65f3f5860bf2-g01` no abre duplicacion nueva.
  Reobserva backlog degradado antes de programar codigo, pero el patron ya esta
  cubierto por T44 para contexto obligatorio `ref_only`, T249/T250/T251 para
  identidad, dedupe/preflight y alcance de pruebas de scanners, T252/T254 para
  ids humanos, reserva, colisiones, aliases y read model, y
  T253/T255/T256/T257/T258 para splits concretos recientes. Las refs
  `worktree_ref` y `branch_ref` se conservan opacas.
- Revalidacion retry 6ed415 burst 002 2026-05-27: el paquete
  `agent-ref-task-autoprogramming-6050b74fc6c4-g01` no abre duplicacion nueva.
  Reobserva contexto obligatorio `ref_only`, write-set documental cerrado,
  prueba global obligatoria y backlog degradado antes de programar codigo, pero
  el patron ya esta cubierto por T44 para contexto obligatorio `ref_only`,
  T249/T250/T251 para identidad, dedupe/preflight y alcance de pruebas de
  scanners, T252/T254 para ids humanos, reserva, colisiones, aliases y read
  model, y T253/T255/T256/T257/T258 para splits concretos recientes. Las refs
  `worktree_ref` y `branch_ref` se conservan opacas.
- Revalidacion retry 4da183 burst 002 2026-05-27: el paquete
  `agent-ref-task-autoprogramming-3533ffab219a-g01` no abre duplicacion nueva.
  Reobserva contexto obligatorio `ref_only`, write-set documental cerrado,
  prueba global obligatoria y backlog degradado antes de programar codigo, pero
  el patron ya esta cubierto por T44 para contexto obligatorio `ref_only`,
  T249/T250/T251 para identidad, dedupe/preflight y alcance de pruebas de
  scanners, T252/T254 para ids humanos, reserva, colisiones, aliases y read
  model, y T253/T255/T256/T257/T258 para splits concretos recientes. Las refs
  `worktree_ref` y `branch_ref` se conservan opacas.

## Cerrado 2026-05-26: T125 work-profiles-empty-module-quarantine

- `modulos/orquesta-work-profiles` queda clasificado como placeholder historico
  no ejecutable con `README.md` y `AGENTS.md` locales.
- La fuente vigente de perfiles sigue siendo `WorkProfileV0` y
  `WorkflowTaskV0.work_profile_kind` en `modulos/orquesta-core-workflow`; la
  resolucion operativa sigue en `modulos/orquesta-orchestration-core`.
- El indice federado del planner no acepta directorios de modulo como fuentes:
  solo documentos Markdown bajo `docs/` o `*/docs/`. La regresion queda cubierta
  por `TestIdleSelfImprovementFederatedBacklogV0IgnoraDirectoriosSinContratoEfectivoV0`.

## Cerrado 2026-05-26: T126 module-local-codex-launcher-compatibility-contract

- Los wrappers `modulos/*/arrancar_codex.sh` quedan inventariados por categoria:
  compatibilidad historica con error publico, prompt/contexto local,
  recuperacion manual opt-in y smoke opt-in.
- Los wrappers que llamaban a `../_comun/arrancar_codex_modulo.sh` ya no
  dependen de un helper inexistente ni lanzan una via paralela de runtime Codex.
- Las READMEs que recomendaban el wrapper local como arranque normal apuntan al
  servidor residente/cola OrquestaV2 como ruta vigente con write-set, ACK,
  checkpoint y shutdown gobernados.
- Revalidado 2026-05-27 por
  `agent-ref-task-autoprogramming-089a03f67381-g01`: 36 wrappers clasificados,
  helper comun ausente no recreado, sin referencias versionadas al helper roto y
  pruebas obligatorias de shell/CLI/servidor en verde.

## Cerrado 2026-05-26: T142 cli-json-input-bounds-and-source-policy

- `modulos/orquesta-cli/command_flags_v0.go` concentra el owner local de
  entrada JSON por CLI: limite por comando, lectura acotada, error publico
  `input_too_large` y sin volcar cuerpos recibidos.
- La fuente queda clasificada como `stdin`, `file_explicit` o `inline` futuro;
  `arg` se conserva solo para comandos sin payload JSON.
- La entrada por fichero explicito bloquea rutas absolutas, salida del
  directorio actual, `.orquesta-runtime`, prompts, transcripts, logs y ficheros
  de control Codex antes de abrir el fichero. T103/T138/T139 siguen siendo
  owners de respuesta HTTP, salida publica y stdout.

## Priorizacion scanner 2026-05-24

Backlog asociado: `T15 detail-rails-reactivation`.

Prioridad alta:

- `modulos/orquesta-context/context_bundle_validation_v0.go`, porque puede
  bloquear formas reparables antes de que el director normalice contexto.
- `modulos/orquesta-director-agent/director_decision_helpers_v0.go`, porque
  participa en decisiones y reviews donde ya se observaron falsos positivos por
  vocabulario operativo.
- `cmd/orquesta-server` y scripts de rails, porque desde 2026-06-02 deben
  mantener `ORQUESTA_RAILS_MODE=offline` y
  `ORQUESTA_DETAIL_PROHIBITED_RAILS=off` por defecto, con matriz y docs
  sincronizados antes de cualquier reactivacion opt-in.

Prioridad media:

- Persistencia, scheduler, director y replanner: conservar listas defensivas si
  son frontera real, pero exigir caso reproducible, propietario unico y matriz
  externa antes de relajar o endurecer.
- `modulos/orquesta-director-operativo/validation_v0.go`: mantener corte de
  secretos efectivos, no de refs opacas o vocabulario normal del plan.
- `modulos/orquesta-app-codex-stack/spec_external_context_v0.go` y
  `review_rework_replan_evidence_v0.go`: prioridad media-alta, porque afectan
  contexto y replan de autoprogramacion; deben aceptar refs/politicas opacas y
  cortar solo datos sensibles efectivos o evidencia ambiental insuficiente.
- Auditoria del servidor y observability: prioridad media-alta antes de abrir
  payload HTTP completo, porque ahi un rail duplicado o demasiado laxo puede
  persistir material sensible aunque el flujo operativo sea local.
- `modulos/orquesta-operator-mcp`: prioridad media. El rail protege preguntas y
  refs de operador, pero no debe bloquear ausencia de conector ni consejo
  publico no sensible.
- Bundles SQL reales y catalogo de smokes: prioridad media. Son fronteras de
  efecto externo; deben tener guardas propias, pero no bloquear vocabulario
  operativo normal como `db`, `sql`, `driver`, `dsn`, `provider` o `codex`.
- Transporte MCP residente opt-in: prioridad media. El smoke temporal esta
  separado, pero cualquier exposicion permanente de `/mcp` debe tener un unico
  propietario de transporte, auditoria compacta y sin duplicar logica de
  `orquesta-mcp`.

Fuera de relajacion general:

- `modulos/orquesta-runtime-required-test/local_command_executor_v0.go`: es
  frontera de efecto externo. Cualquier cambio debe probar permisos, working dir,
  comandos permitidos y ausencia de secretos en salida.
- `modulos/orquesta-core-leases/lease_policy_v0.go`: es frontera de ciclo de
  vida de agentes; no relajar tiempos, acciones o refs sin matriz de heartbeat,
  lease expirado y shutdown.
- `modulos/orquesta-observability/orquesta_event_types_v0.go`: puede seguir
  separado si solo marca privacidad/observabilidad y no bloquea comandos
  operativos validos.

## Priorizacion scanner 2026-05-24 cuarta pasada

Backlog asociado: `T22 detail-rails-doc-state-sync` y
`T23 mcp-run-transport-opt-in`.

Prioridad alta:

- Sincronizar `docs/rails/registro_railes_2026-05-24.md` con el default real de
  `cmd/orquesta-server`: `ORQUESTA_RAILS_MODE=offline` y
  `ORQUESTA_DETAIL_PROHIBITED_RAILS=off`. Cualquier vuelta a bloqueo no se
  reactiva por env: debe ser tarea futura, con owner y prueba viva.
- Sustituir la matriz rapida antigua de bloqueo por una comprobacion de build y
  smoke API que confirme que los rails quedan offline aunque el entorno intente
  activar `enforced`, scopes o `ORQUESTA_DETAIL_PROHIBITED_RAILS=on`.

Prioridad media:

- Decidir si el transporte MCP real vive solo como comando temporal
  `mcp-real-smoke` o si se expone en `run` con env opt-in. En ambos casos, el
  transporte debe quedar en `cmd/orquesta-server`; `orquesta-mcp` no asume red,
  stores, runtime ni proveedor.

## Priorizacion scanner 2026-05-24 quinta pasada

Backlog asociado: `T24 startup-structured-ack-compaction`,
`T25 mcp-roadmap-backlog-state-sync` y `T26 domain-work-quality-policy-port`.

Prioridad alta:

- Sustituir inferencia textual de ACK en compactacion de arranque por lectura
  estructurada y correlada de `agent_ack.json`. Es frontera de recuperacion y
  purga, por lo que un falso positivo puede archivar runtime o sacar candidatos
  vivos de cola.
- Sincronizar roadmap MCP con backlog/foto vigente antes de que el residente o
  una IA externa lo use como fuente de planificacion.

Prioridad media:

- Extraer la calidad de `domain_work` a politica por puerto/adaptador con matriz
  externa. Mantener el gate actual como proteccion hasta que exista reemplazo
  probado; no relajar strings de placeholders sin evidencia.

## Priorizacion scanner 2026-05-24 sexta pasada

Backlog asociado: `T27 docs-source-of-truth-state-sync`,
`T28 restart-live-agent-reconciliation` y
`T29 provider-usage-quota-accounting`.

Prioridad alta:

- Sincronizar `AGENTS.md`, estado vigente, guia, matriz y backlog cuando una
  prueba real pasa de pendiente a cerrada. Esos documentos gobiernan agentes
  externos y no deben duplicar estados opuestos para `CODEX-WAVE-REAL`,
  `CODEX-RECURSION-REAL` u OPES temporal pendiente.
- Separar la recuperacion de arranque en dos propietarios: ACK estructurado para
  compactacion (`T24`) y reconciliacion/resume de agentes vivos (`T28`). No
  volver a usar `codex_last_message.txt` ni summaries como fuente suficiente de
  cierre.

Prioridad media:

- Unificar la exposicion de uso/coste de proveedor: stats fake/offline,
  fuente Codex, MCP, web y servidor deben compartir puerto o proyeccion comun,
  con redaccion por campo y sin mezclar proveedor real con auditoria o ACKs.
- Cierre parcial 2026-05-25: el query vivo minimo de estado operativo queda
  centralizado en `OperationalStatusQueryV0` y `/api/v0/operational-status/query`;
  no sustituye la agregacion historica ni el recibo detallado de proveedor.
- Mantener `USAGE-METRICS-REAL` dentro del catalogo de smokes reales con guarda
  explicita hasta que exista conector productivo de cuota/tokens probado.

## Priorizacion scanner 2026-05-24 septima pasada

Backlog asociado: `T30 runtime-shutdown-checkpoint-port`,
`T31 run-queue-reservation-lease` y
`T32 resident-supervisor-event-driven-wakeup`.

Prioridad alta:

- Unificar el propietario del checkpoint de shutdown: `orquesta-server-shutdown`
  debe expresar el contrato neutral y `orquesta-runtime-codex` solo el adaptador
  por ficheros/prompt. Evitar que cada runtime copie su propio schema, rail de
  detalle y politica de deadline.
- Anadir propietario unico para reserva/lease de cola antes de ampliar
  supervisores residentes. Hoy `orquesta-run-queue` rankea sin reservar y los
  stores file/memory solo mutan prioridad/estado; la proteccion contra doble
  launch no debe quedar duplicada entre supervisor, servidor y adaptadores.
- Consolidar wakeups event-driven del residente: el tick del servidor, la cola,
  la auditoria JSONL y la politica documental no deben divergir sobre que evento
  autoriza un nuevo supervisor tick o automejora idle.

Prioridad media:

- Las nuevas payloads de checkpoint, lease y wakeup deben reutilizar politica de
  texto sensible por campo: refs opacas y vocabulario operativo pasan; HOME,
  rutas reales, PID/host como identidad, prompts, transcripts, tokens y payloads
  HTTP completos no se persisten como evidencia operativa.
- Si el tick periodico queda como fallback, documentar su relacion con wakeups,
  cooldowns y leases para que no vuelva a ser fuente paralela de decisiones.

## Priorizacion scanner 2026-05-24 octava pasada

Backlog asociado: `T33 autoprogramming-backlog-ack-correlation`,
`T34 director-wave-strict-guards-default` y
`T35 startup-cleanup-safe-mode`.

Prioridad alta:

- Unificar las lecturas de `agent_ack.json`: observation source, startup
  compaction y backlog planner deben usar ACK estructurado/correlado o marcar
  ambiguedad. No debe haber un cierre por `codex_last_message.txt` y otro por
  `status=completed` sin schema.
- Cambiar el default de startup cleanup a modo conservador antes de ampliar
  reinicio/resume con agentes vivos; `forced_stop` debe ser accion explicita.

Prioridad media:

- Endurecer defaults de `codex-director-wave` para ejecucion real: write-set y
  tests concretos obligatorios, con `write_set=.` o tests placeholder solo como
  opt-in auditado.
- Revisar prompts de ola/subola para que "ownership inicial" se aplique al
  shard dentro del alcance autorizado, no como permiso general para editar fuera
  del write-set.

## Priorizacion scanner 2026-05-24 novena pasada

Backlog asociado: `T36 codex-ack-strict-terminal-validation`,
`T37 autoprogramming-backlog-parser-fidelity` y
`T38 codex-delivery-observation-rail-owner`.

Prioridad alta:

- Separar validacion estricta de ACK terminal de compatibilidad legacy con
  defaults. T33/T24 no quedan realmente seguros si el validador comun sigue
  aceptando ACK minimo para cerrar observaciones nuevas.
- Corregir el parser de backlog antes de confiar en tareas Txx con scripts,
  `git diff --check` o verificaciones documentales. El required_tests del
  paquete debe reflejar la seccion, no solo el test base ni comandos Go.

Prioridad media:

- Clasificar el rail local de observacion Codex como shard de T15: o se elimina
  como detector historico, o pasa a politica comun por campo con matriz externa.
- Mantener tolerancia a refs opacas y vocabulario operativo en ACK/observacion;
  cortar solo secretos efectivos, rutas privadas, prompts/transcripts y payloads
  de proveedor.

## Priorizacion scanner 2026-05-24 decima pasada

Backlog asociado: `T39 app-vcs-write-set-evidence-guards`,
`T40 autoprogramming-promotion-real-e2e-and-doc-state` y
`T41 server-audit-payload-redaction-contract`.

Prioridad alta:

- Unificar propietario de efectos VCS. `orquesta.app_vcs.v0`,
  `GitAppVCSConnectorV0`, `GitStagingPromotionConnectorV0` y la promocion de
  automejora no deben tener guardas divergentes sobre write-set, commit paths,
  push, redaccion de errores Git y evidencia causal.
- Convertir la promocion de automejora en estado verificable de composicion:
  T13 no debe seguir como pendiente generico si el codigo ya tiene piezas
  parciales, pero tampoco debe declararse cerrado sin e2e temporal y recibo
  durable de promocion/archivo.
- Definir contrato de payload auditado por evento para el servidor antes de
  abrir visores o payloads completos. La auditoria no debe depender de que cada
  caller recuerde no meter prompts, transcripts, rutas privadas, remotos Git o
  cuerpos de dominio en `map[string]interface{}`.

Prioridad media:

- Mantener `review_repo` como accion read-only separada de commit/push; sus
  resultados pueden compartir redaccion, pero no las mismas autorizaciones de
  efecto externo.
- Si AppVCS y promocion comparten helper de Git, debe seguir en
  `orquesta-runtime-worktree` como adaptador opt-in; `orquesta-mcp`,
  `orquesta-autoprogramming` y el nucleo conservan solo DTOs, puertos y refs
  opacas.
- La observabilidad debe consumir proyecciones compactas de auditoria, no
  payloads crudos ni listas locales de terminos sensibles.

## Priorizacion scanner 2026-05-24 undecima pasada

Backlog asociado: `T42 codex-ack-strict-write-set-terminal-proof`,
`T43 backlog-scanner-doc-merge-lease` y `T44 required-context-ref-only-guard`.

Prioridad alta:

- Separar rail blando historico de modo terminal estricto. `file_outside_write_set`
  puede seguir como advisory para entregas legacy, pero agentes gobernados por
  OrquestaV2 no deben cerrar `completed` si el ACK declara archivos fuera de
  alcance, globs, directorios o control files.
- Anadir lease/epoch de merge para scanners de backlog antes de lanzar mas
  bursts sobre los tres documentos vivos. Sin esa reserva, dos agentes pueden
  anadir Txx solapados y el planner puede deduplicar por ACK sin revisar foto
  documental.
- Hacer verificable el contexto requerido. Una entrada obligatoria `ref_only`
  debe estar justificada por diseno o por evidencia de lectura/consulta; no debe
  depender solo de que el agente recuerde leer docs manualmente.

Prioridad media:

- Mantener `codex_ack_validation` como propietario de validacion terminal de
  ACK, pero delegar snapshot/write-set real en `orquesta-runtime-worktree` por
  puerto para no meter filesystem en core.
- El merge de backlog debe crear followups publicos y compactos; no debe
  intentar resolver conflictos borrando secciones historicas ni reordenando
  docs grandes desde el scanner.
- El guard de contexto debe vivir en `orquesta-context`/materializer y solo
  proyectarse al runtime/ACK como refs/evidencia; no meter rutas locales ni
  contenido largo en paquetes durables.

## Priorizacion scanner 2026-05-24 duodecima pasada

Backlog asociado: `T45 autoprogramming-go-file-line-budget-baseline`.

Estado 2026-05-24: cerrado localmente. Queda como rail unificado:
`file_too_large` es advisory legacy; `go_file_line_budget_strict_blocking` es
bloqueo terminal estricto con conteo real de snapshot/worktree/proyecto y
baseline de no crecimiento.
Actualizacion 2026-05-25: el owner de activacion queda explicito en
`StrictGoLineBudget`. `LineCountSource` es evidencia, no switch de modo; el
stack Codex estricto cruza esa evidencia con snapshot/worktree para cubrir
ficheros omitidos del ACK.
Rework 2026-05-25: T45 queda cerrado documentalmente como rail unificado; las
particiones futuras se abren solo ante violaciones concretas del baseline.

Prioridad historica:

- Unificar propietario del rail de tamano de fichero Go. El prompt Codex, el
  review gate de autoprogramacion, el snapshot/worktree y el cierre terminal no
  deben divergir entre "limite obligatorio" y "advisory que acepta completed".
- Medir lineas desde snapshot/worktree real cuando exista. `ACK.files` puede
  listar archivos tocados, pero no debe ser fuente suficiente para el conteo ni
  para probar que un fichero grande no crecio.

Prioridad media:

- Crear baseline de deuda historica por fichero y regla de no crecimiento antes
  de partir modulos grandes. El scanner no debe abrir refactors masivos sin
  write-set propio, pero si debe registrar followups concretos por fichero o
  modulo cuando una entrega toca una pieza ya sobre 300 lineas.
- Mantener compatibilidad legacy de `file_too_large` como advisory solo donde
  no gobierne OrquestaV2 estricto; no mezclar ese modo con ACK terminal nuevo.

## Priorizacion scanner 2026-05-24 decimotercera pasada

Backlog asociado: `T46 agent-packet-write-set-precedence`,
`T47 external-agent-required-test-receipts` y
`T48 destructive-worktree-change-proof`.

Prioridad alta:

- Unificar propietario de precedencia del paquete: `spec_task_v0.go`,
  `codex_prompt_v0.go`, policies del packet y validacion terminal no deben
  emitir instrucciones incompatibles sobre ampliar write-set.
- Estado 2026-05-24: T46 queda cerrado para contradiccion packet/prompt. La
  precedencia vive en objetivo generado, prompt Codex y validacion de
  `AgentStartPacketV0`; lo pendiente de esta lista pasa a T47/T48.
- Revalidacion 2026-05-25: el backlog ejecutable deja T46 como cerrado y no
  debe volver a listar la contradiccion `programmingObjectiveV0`/prompt como
  hueco nuevo salvo regresion demostrada.
- Rework de revision 2026-05-25: el retry estricto de T46 solo actualiza la
  evidencia documental dentro del write-set, resuelve `required ref_only` con
  lectura local/ACK y conserva T47/T48 como propietarios de los restos.
- Separar prueba declarada de prueba ejecutada. `ACK.tests`, runner de tests
  requeridos, receipts de runtime y cierre operativo deben compartir contrato
  de evidencia para modo estricto.
- Usar snapshot/worktree como fuente para efectos destructivos. Borrados,
  truncados fuertes, renombres y reemplazos masivos no deben depender de lo que
  el agente liste en `ACK.files`.

Prioridad media:

- Mantener modo legacy/advisory para agentes antiguos, pero nombrarlo
  explicitamente y evitar que el planner de automejora lo use para cerrar
  paquetes OrquestaV2.
- No convertir cada prompt en un rail local nuevo: la politica debe vivir en
  runtime/packet/ACK y exponerse como refs compactas a stack, servidor y review
  gate.
- Las evidencias de tests y worktree deben estar redactadas por campo; stdout,
  stderr, prompts, transcripts, HOME, tokens, remotos Git y rutas privadas no
  son payload durable de cierre.

## Priorizacion scanner 2026-05-24 decimocuarta pasada

Backlog asociado: `T49 codex-runtime-security-profile-strict-mode`,
`T50 runtime-control-files-worktree-exclusion` y
`T51 capacity-reasoning-default-policy`.

Prioridad alta:

- Unificar propietario de perfil de seguridad Codex. `cmd/orquesta-server`,
  `orquesta-app-codex-stack` y `orquesta-runtime-codex` no deben divergir entre
  "rechazar sandbox invalido" y "normalizarlo silenciosamente". Las politicas de
  approval interactivo deben quedar separadas para operador vivo y no para
  autoprogramacion residente. Cubierto 2026-05-24: el servidor/stack conservan
  sandbox invalido para que `orquesta-runtime-codex` emita error publico,
  `approval_policy` interactivo exige opt-in explicito de operador vivo y el
  perfil/prompt declara la ubicacion del runtime de control. La revalidacion
  estricta exige resolver contexto `ref_only` requerido por evidencia explicita
  en ACK.
- Revalidacion 2026-05-24 r2: paquete estricto retry confirma evidencia
  explicita de contexto `ref_only` en ACK y no cambia el propietario del perfil.
- Excluir control files del flujo de producto. `.orquesta-runtime`,
  `.orquesta-codex-runtime`, `.orquesta-local-runtime-*`, prompts, packets,
  ACKs, logs y checkpoints no deben aparecer en snapshots, AppVCS, staging
  promotion, contexto de producto ni `ACK.files`, incluso cuando el write-set
  sea raiz. Cubierto 2026-05-25 r3 en `orquesta-runtime-worktree`,
  `orquesta-runtime-codex` y `orquesta-app-codex-stack`: exclusiones por
  defecto, issue `control_path`, filtrado de AppVCS/staging promotion, rechazo
  de control files en ACK terminal y consumo de prefijos comunes en baseline,
  verificacion y review/rework.
- Fijar politica unica de capacidad/razonamiento. El servidor, el stack Codex,
  los packets y los smokes no deben mezclar default `xhigh`, elevacion a `high`
  y regla documental de `medium` sin una matriz por work profile, dominio y
  riesgo. Cubierto 2026-05-25: la matriz compacta vive en
  `orquesta-autoprogramming`, el servidor conserva `medium` sin elevarlo, OPES
  documental sigue opt-in `xhigh`, y packets/stats exponen refs de politica y
  evidencia.

Prioridad media:

- Si el runtime de control queda fuera del proyecto, debe ser excepcion de
  control declarada por ref y no segundo workdir de producto. Si queda dentro,
  debe estar ignorado por snapshot/promocion y protegido contra lectura de otros
  agentes.
- Los cambios de capacidad deben preservar OPES/temarios reales como casos
  opt-in de alta capacidad, pero los scanners y tareas documentales de fondo no
  deben heredar ese coste por defecto.
- Las decisiones de sandbox, approval, control dir y razonamiento deben
  registrarse como evidencia compacta para auditoria sin filtrar HOME, rutas
  privadas, prompts, transcripts ni proveedor concreto al nucleo.

## Priorizacion scanner 2026-05-24 decimoquinta pasada

Backlog asociado: `T52 app-director-service-file-split`,
`T53 orchestration-core-file-split` y
`T54 codex-stack-server-file-split`.

Prioridad alta:

- `modulos/orquesta-app-director-service` ya separa el codigo productivo de
  continue, wait-state, review/replan y cierre causal en ficheros menores. La
  deuda restante de ese modulo queda como baseline de tests historicos que no
  debe crecer sin shard propio.
- `modulos/orquesta-orchestration-core` ya partio el materializador,
  plan-state, cierre y runner de tests por responsabilidad neutral el
  2026-05-25. Mantener esos shards como owner local antes de anadir mas ciclo
  operativo; la deuda residual de otros ficheros grandes queda para shards
  explicitos.

Prioridad media:

- Separar `modulos/orquesta-app-codex-stack` y `cmd/orquesta-server` por flujo
  de composicion: drain, supervisor, closure source, comandos Codex, backlog
  planner, config y smokes reales. No mover adaptadores al nucleo.
- Los smokes reales largos pueden seguir como tests opt-in, pero deben
  dividirse por escenario cuando se toquen y declarar baseline si exceden 300
  lineas por deuda historica.
- La particion no debe relajar T49-T51: seguridad Codex, control files y
  capacidad/razonamiento siguen como rails separados con propietario propio.

Avance 2026-05-25: T54 separo `cmd/orquesta-server/codex_wave_command_v0.go`,
`cmd/orquesta-server/codex_director_wave_command_v0.go` y
`modulos/orquesta-app-codex-stack/drain_v0.go` en ficheros productivos menores
por responsabilidad. Los tests/smokes historicos largos quedan como baseline y
no se consideran permiso para ampliar controladores productivos.

## Priorizacion scanner 2026-05-24 decimosexta pasada

Backlog asociado: `T55 server-control-plane-exposure-guards`,
`T56 codex-code-home-credential-projection` y
`T57 outbox-dispatch-durable-ack-recovery`.

Prioridad alta:

- Unificar propietario de exposicion del control plane. `orquesta-server`,
  HTTP gateway, MCP real, AppVCS, run control y shutdown no deben tener guardas
  divergentes sobre loopback, bind remoto, autenticacion, TLS/mTLS, operador y
  auditoria. El default debe seguir local; cualquier exposicion remota necesita
  opt-in y evidencia.
- Separar proyeccion de credenciales Codex de contexto/producto. Cerrado el
  2026-05-25 para `codex-wave`/`codex-director-wave`: la copia de `auth.json`,
  config, skills, plugins, rules y memories pasa por politica de categorias,
  receipt compacto y modo estricto recuperable. T152 conserva el shard mecanico
  de presupuesto, symlinks y modos.
- Cerrar propietario de outbox dispatch durable. La cola de runs (`T31`) no
  sustituye al claim/lease/ACK por mensaje de outbox; ambos rails deben estar
  coordinados pero con contratos y pruebas propias.

Prioridad media:

- Health/status pueden conservar lectura limitada sin auth fuerte si el bind es
  loopback; endpoints mutables requieren principal/permiso. La auditoria debe
  guardar decision y ref de permiso, no cabeceras ni tokens.
- Cierre 2026-05-25: el propietario comun queda en `orquesta-server`; bind
  remoto requiere opt-in y token de composicion, las mutaciones HTTP/MCP quedan
  guardadas por loopback o token, y auditoria redacta valores de query, tokens,
  cabeceras completas y payloads.
- La politica de credenciales permite vocabulario operativo normal (`auth`,
  `token policy`, `skills`, `plugins`, `HOME ref`) cuando es ref opaca, pero
  bloquea valores reales, rutas privadas y material completo en la evidencia de
  proyeccion.
- Rework de revision 2026-05-25: backlog, rail observado y esta matriz quedan
  sincronizados como cierre T56; no se reabre el shard mecanico T152.
- La recuperacion de outbox debe convivir con batches parciales: success cierra
  solo el item ACK correlacionado, failed queda publico y missing no permite
  declarar cierre de plan/run.

## Priorizacion scanner 2026-05-24 decimoseptima pasada

Backlog asociado: `T58 codex-wave-log-tail-redaction-access`,
`T59 codex-wave-runtime-purge-proof` y
`T60 codex-wave-stop-registry-pid-proof`.

Prioridad alta:

- Separar diagnostico de evidencia terminal. `codex-wave-tail` puede ayudar a un
  operador local, pero no debe duplicar T24/T33 ni volver a usar
  `codex_last_message.txt`, stdout o stderr como fuente de cierre. Si imprime
  algo, debe ser summary redactado por defecto y fragmento crudo solo con opt-in
  local y limite fuerte.
  Cierre T58 2026-05-25: tail queda en JSON diagnostico, con razon requerida,
  scope runtime/agente, limites antes de lectura y redaccion `orquesta-rails`.
  Rework de revision 2026-05-25: backlog, rail observado y esta matriz quedan
  sincronizados como cierre T58.
- Unificar el propietario de operaciones destructivas sobre runtime Codex. T59
  queda implementado desde 2026-05-25 para `cmd/orquesta-server`: confirmacion,
  raiz permitida, manifest, bloqueo por agentes vivos/checkpoints/outbox y
  evidencia compacta propia.
  Rework de revision 2026-05-25: el retry estricto no abre scope nuevo,
  conserva la evidencia T59 y resuelve `required ref_only` por lectura
  local/ACK, dejando T60 como owner separado de stop/status.
- Unificar confianza de control de procesos. `codex-wave-stop`, status y tail no
  deben confiar en un registro JSON arbitrario bajo `runtime-dir`; deben usar el
  mismo contrato de descriptor/store que el registry de agentes y run-control.
  Cierre T60 2026-05-25: launch escribe proof por agente, status/tail/stop
  comparten runtime permitido, stop bloquea registry suelto como
  `blocked_registry_untrusted`, usa shutdown cooperativo por defecto y solo
  senala con confirmacion, razon y `--force`.
  Rework de revision 2026-05-25: backlog, rail observado y esta matriz quedan
  sincronizados como cierre T60; el retry estricto no abre scope nuevo y resuelve
  `required ref_only` por lectura local/ACK.

Prioridad media:

- La redaccion de logs debe reutilizar politica comun de rails donde sea
  aplicable, pero conservar owner de runtime Codex para los casos de prompt,
  transcript, last-message y logs operacionales.
- `--purge-runtime` debe quedar fuera de flujos automaticos de automejora hasta
  que exista proof de no borrar trabajo vivo. La limpieza de archivos de control
  no sustituye promocion, archivo ni cierre causal.
- El stop forzado debe coordinarse con T30: si el runtime soporta checkpoint, el
  flujo normal es cooperativo; la senal directa requiere decision/opt-in y causa
  publica.

## Priorizacion scanner 2026-05-24 decimoctava pasada

Backlog asociado: `T61 required-test-output-redaction-retention`,
`T62 director-decisions-prompt-rail-sync` y
`T63 director-decisions-sidecar-receipt-correlation`.

Prioridad alta:

- Separar evidencia causal de tests requeridos de logs crudos. El cierre del
  Director puede necesitar refs durables, pero no necesita persistir stdout o
  stderr sin redaccion para demostrar que un comando paso o fallo.
- `T62 director-decisions-prompt-rail-sync` queda cerrado el 2026-05-25 para el
  prompt y validacion focal: se retiro la lista textual vieja y se sustituyo por
  politica positiva de refs opacas. Si reaparecen falsos positivos, deben entrar
  como regresion con evidencia nueva.
- Rework T62 2026-05-25: la documentacion ya no atribuye el cierre a smokes de
  servidor de T65; el alcance queda en prompt, validacion focal y matriz de
  rails del write-set.
- `T63 director-decisions-sidecar-receipt-correlation` queda cerrado el
  2026-05-25 para el tramo Codex/file-source/app-stack: `director_decisions.json`
  tiene receipt propio con hash, ACK productor, correlacion y consumo
  idempotente en el store de receipts. Si reaparece ambiguedad, debe entrar como
  regresion con sidecar mutado/stale o store no durable concreto.

Prioridad media:

- La redaccion de test output puede reutilizar politica comun de rails, pero el
  owner del artefacto y retencion pertenece a `orquesta-runtime-required-test` y
  composicion de servidor, no a auditoria HTTP ni a tail Codex.
- Cierre 2026-05-25: `orquesta-runtime-required-test` ya es owner del artefacto
  `required-test-output-v0/*.log`; reutiliza `orquesta-rails`, escribe salida
  redactada con limite/retencion y el servidor expone
  `ORQUESTA_REQUIRED_TEST_OUTPUT_MAX_ARTIFACTS`. T41/T58/T97 no deben duplicar
  este owner; solo pueden consumir refs/summary redactados.
- Las reglas de prompt y validacion de decisiones deben permitir refs opacas de
  runtime/proveedor/modelo cuando son contexto arquitectonico, y cortar solo
  valores reales, rutas privadas, secretos, prompt/transcript crudos o
  causalidad rota.
- La correlacion del sidecar debe conservar compatibilidad con flujos legacy
  mientras el modo estricto OrquestaV2 exige receipt completo para cerrar como
  `completed`.

## Priorizacion scanner 2026-05-24 decimonovena pasada

Backlog asociado: `T64 director-cycle-source-of-truth-sync`,
`T65 director-cycle-resident-restart-smoke` y
`T66 neutral-process-stop-e2e`.

Estado 2026-05-25: `T64` queda cerrado como sincronizacion documental de fuente
de verdad. La espina `DirectorCycleStepV0 -> director-runner ->
director-scheduler -> core-workflow -> director-cycle-outbox` ya aparece en la
foto vigente, guia, README raiz, matriz y docs locales de
`orquesta-orchestration-core`. `T65` queda cerrado localmente con smoke focal de
composicion residente/restart en `cmd/orquesta-server`, usando outbox durable,
dispatch fake y ACK correlado. `T66` queda cerrado localmente con
`TestNeutralProcessStopE2EV0`: proceso temporal real neutral, stop causal por
`DirectorCycleStepV0`/scheduler/outbox, ACK de parada e idempotencia de replay
tras reinicio.

Prioridad alta:

- Sincronizar la fuente de verdad del ciclo Director V2. `T27` cubre autoridad
  documental general, pero la espina `director-cycle` necesita entrada propia
  para no duplicar pendientes historicos de review/tests/cierre ni ocultar los
  nuevos modulos en contextos de agente.
- Cubrir la composicion residente de ciclo + outbox durable. `T57` gobierna
  claim/lease/ACK por mensaje de outbox; `T65` demuestra que el servidor usa esa
  frontera desde `DirectorCycleStepV0` y no solo desde tests locales.
- Mantener `DIR-P006` como rail neutral de runtime cerrado. `T60` protege stop
  de ola Codex por registry/PID; `T66` prueba proceso temporal real sin Codex,
  con stop causal y ACK/evidencia por refs opacas.

Prioridad media:

- Evitar que cada modulo local mantenga su propio estado de "pendiente" sin
  clasificarlo por codigo offline, composicion residente, smoke real o proveedor
  real. Los AGENTS locales deben apuntar a la foto vigente o declarar frontera
  local clara.
- No duplicar rails de outbox: la cola de runs, el ledger de outbox, el dispatch
  batch y el ciclo del Director comparten refs, pero cada uno conserva owner y
  prueba distinta.
- La parada de proceso neutral debe reutilizar politica de runtime y rails de
  redaccion existentes; no debe heredar comandos, env, HOME, pid crudo o rutas
  locales como evidencia durable de cierre.

## Priorizacion scanner 2026-05-24 vigesima pasada

Backlog asociado: `T67 director-supervisor-burst-source-of-truth-sync`,
`T68 director-supervised-burst-resident-budget-smoke` y
`T69 nested-supervisor-stop-reason-contract`.

Prioridad alta:

- Completar la fuente de verdad del ciclo Director V2 con la capa de supervision
  de pasos. `T64` cubre cycle/scheduler/runner/tick-input; `T67` debe evitar que
  `director-supervisor` y `director-supervised-burst` queden como reglas locales
  invisibles y se dupliquen en `orquesta-orchestration-core` o servidor.
- Probar presupuestos anidados en composicion residente. `T65` cubre outbox
  durable/restart; `T68` debe demostrar que `MaxTicks`, `MaxBursts` y
  `MaxStepsPerBurst` producen acciones finales visibles sin cerrar por error ni
  relanzar la misma run.
- Razones de parada: `T69` quedo cerrado el 2026-05-25 con
  `orquesta_supervisor_stop_reason_projection.v0`. `no_execution`,
  `max_ticks`, `wait_outbox`, `wait_external` y `stop_max_steps` se proyectan a
  categorias publicas compartidas para cola, stats, auditoria y planificador de
  automejora.

Prioridad media:

- Mantener propietarios separados: `run-supervisor` gobierna runs/ticks,
  `director-supervisor` decide repetir/parar pasos, `director-supervised-burst`
  ejecuta la rafaga y outbox/dispatch conserva ACKs. No crear un rail generico
  que mezcle todo el control loop.
- La proyeccion publica debe usar refs y contadores compactos; no duplicar el
  payload completo de command/result que T41 debe redactar por evento.
- Los docs locales de los tres modulos deben apuntar a la foto vigente sin
  convertir pruebas unitarias en smoke residente cerrado.

## Priorizacion scanner 2026-05-24 vigesimoprimera pasada

Backlog asociado: `T70 core-workflow-docs-rail-policy-sync`,
`T71 domain-work-adapters-file-budget-split` y
`T72 domain-work-artifact-contract-map-owner`.

Prioridad alta:

- Unificar el owner del mapa `work_kind -> expected_artifact_type`. El expander,
  stack Codex, bridge OPES y tests de servidor no deben tener tablas paralelas
  para `draft_content_block`, `generate_visual_asset`, revisiones,
  `validate_topic`, `expand_topic_from_summary` y `assemble_topic`.

Cierre T72 2026-05-25: `orquesta-domain-work` publica
`ExpectedDomainWorkArtifactTypeForWorkKindV0` como owner neutral. Expander,
stack Codex, bridge OPES y prueba focal de servidor consumen el mismo mapa; el
fallback para work kinds desconocidos sigue siendo `work_delivery`.

Cierre T70 2026-05-25: los docs locales de `orquesta-core-workflow` quedan
sincronizados con la politica comun de `orquesta-rails`. Ya no tratan
`runtime`, `provider`, `DB`, `SQL`, `HOME`, `modelo` o `Codex` como
prohibiciones genericas; distinguen refs opacas validas de valores sensibles
efectivos y conservan el bloqueo de secretos, rutas privadas,
prompts/transcripts crudos y payloads masivos.

Cierre T71 2026-05-26: `orquesta-domain-work-sql`, `orquesta-domain-work-file`,
`orquesta-domain-work-memory` y `contracttest` quedaron partidos por
responsabilidad local; no quedan ficheros Go >300 lineas en el alcance T71.

Prioridad media:

- Mantener no crecimiento del alcance T71 antes de anadir mas dialectos,
  snapshots o derivados. Contract tests compartidos deben seguir siendo la
  barrera comun de memory/file/sql.
- El mapa de artefactos puede vivir en un helper/contrato neutral, pero no debe
  arrastrar OPES, Codex, HTTP, runtime, DB ni reglas editoriales al nucleo.
- La correccion documental de rails no debe convertirse en relajacion nueva:
  valores sensibles efectivos, rutas privadas, prompts/transcripts crudos y
  payloads masivos siguen bloqueados por campo.

## Priorizacion scanner 2026-05-24 vigesimosegunda pasada

Backlog asociado: `T73 factory-connector-capability-policy-port`,
`T74 deployment-plan-composition-wiring` y
`T75 i18n-docs-active-composition-owner`.

Prioridad alta:

- El rail de nombres de integracion en `orquesta-factory` debe dejar de ser una
  lista local de substrings. La frontera correcta es capacidad de dominio frente
  a proveedor/backend/credencial impuesto. Esta politica no debe duplicar
  `orquesta-rails` ni los validadores de web/MCP.
- Cierre T73 2026-05-25: `orquesta-factory` valida integraciones con una
  politica de capacidad/proveedor y permite vocabulario operativo opaco o
  reparable (`db`, `runtime`, `queue`, `cola`, `deploy`, `database`) sin
  reimplementar listas en web/MCP. Siguen bloqueados proveedor/backend impuesto,
  SDK cloud, credenciales y DSN. La revalidacion del mismo dia cubre proveedor
  concreto combinado con capacidad (`postgres database`) sin bloquear
  capacidades genericas (`database audit`, `runtime metrics`, `cola de tareas`).
- `DeploymentPlan v0` debe tener un owner de composicion visible. Factory,
  planner y MCP ya lo nombran, y `orquesta-deploy` lo implementa como dry-run,
  y desde el 2026-05-25 T74 enlaza el owner por puerto dry-run, AppSpec,
  microtarea de deploy y refs MCP/backlog.

Prioridad media:

- `orquesta-i18n-docs` deja de ser isla en el corte T75 del 2026-05-25: se
  consume como owner activo para bundles, loader y docs generadas. Mantener
  otro owner paralelo en web/factory/MCP reabriria la brecha documentada de
  i18n.
- Deploy e i18n no deben arrastrar proveedores, filesystem productivo,
  toolchains, tiendas, cloud, DB ni runtime al nucleo. La primera integracion
  debe ser dry-run/contrato con refs y evidencia compacta.
- Si un modulo ya tiene contrato local y descriptors MCP, el backlog debe exigir
  sincronizacion de estado antes de lanzar otra implementacion paralela.

## Priorizacion scanner 2026-05-24 vigesimotercera pasada

Backlog asociado: `T76 appspec-entrypoints-director-v2-routing`,
`T77 app-change-director-source-readiness-rail-policy` y
`T78 director-decisions-existing-planstate-merge`.

Prioridad alta:

- Evitar que dos entrypoints de `AppSpecV0` proyecten estados incompatibles:
  `app-runner`/`AppPlan` puede seguir como preview o compatibilidad, pero la
  ruta operativa con juicio debe pasar por Director V2 o declarar bloqueo
  verificable.
  Estado 2026-05-25, refrescado 2026-05-26: cubierto para T76 con
  `route_policy` publico en MCP, tambien en descriptor compacto, y bloqueo
  `director_v2_required` en `orquesta-app-runner`.
- Sacar `orquesta-app-change-director-source` del patron de listas locales de
  terminos prohibidos. Su readiness de autoplanning toca trabajos reales de
  app/domain_work y no debe reabrir falsos positivos por vocabulario operativo.
  Estado 2026-05-25: cerrado para T77; readiness delega en `orquesta-rails` por
  campo y conserva vocabulario operativo opaco. Revision 2026-05-26: el retry
  confirma el contrato y queda bloqueado solo por compilacion de
  `orquesta-observability`/`orquesta-web` fuera del write-set al probar
  `orquesta-app-codex-stack`; la reejecucion posterior del comando requerido
  paso completa y T77 queda cerrado sin cambios de codigo adicionales.
  Validacion OrquestaV2 2026-05-26: contexto `ref_only` resuelto por evidencia
  explicita de ACK y comando requerido verde dentro del write-set T77.
- El merge de `director_decisions` tardias en plan state existente quedo cerrado
  el 2026-05-25 para el camino offline de `app-director-service`: tasks nuevas
  marcadas reabren `wait_subagents` con agent refs acotados e idempotencia.
  T62/T63 protegen prompt/receipt; T78 protege estado vivo y wait scope.
  Revalidacion OrquestaV2 2026-05-26: comando requerido T78 verde y contexto
  `ref_only` resuelto por lectura local/evidencia en ACK, sin cambios de
  alcance ni rails adicionales.

Prioridad media:

- La solucion de `AppSpecV0` no debe borrar `app-runner` ni `app-planner` sin
  revisar consumidores MCP/web/CLI; debe marcar freshness, compatibilidad y
  criterios de uso.
- La sanitizacion de app-change debe preservar el motivo del dominio en issues
  estructurados en vez de reemplazar palabras hasta hacer invisible la causa.
  Estado 2026-05-25: cerrado para T77; los criterios se compactan sin
  reescribir terminos de dominio, y el detalle sensible efectivo queda como
  bloqueo de autoplanning.
- El merge de plan state debe conservar compatibilidad con decisiones legacy que
  no traen metadata operativa: esas decisiones no cierran el plan ni crean wait
  ambiguo.

## Priorizacion scanner 2026-05-24 vigesimocuarta pasada

Backlog asociado: `T79 autoprogramming-backlog-doc-sharding`,
`T80 domain-work-http-egress-policy` y
`T81 observability-global-workspace-timeline`.

Prioridad alta:

- Separar el rail de concurrencia documental en dos capas: T43 reserva/merge de
  scanners; T79 estructura el backlog en indice/shards para que el hotspot deje
  de crecer en tres ficheros monoliticos. No borrar historico ni romper el
  parser residente durante la migracion.
  Estado 2026-05-25: T79 ya tiene contrato inicial residente. El backlog
  historico declara indice vivo, el planner lee indice/shards con fallback al
  documento unico y transporta fichero/linea/hash por shard en las leases. La
  particion fisica a nuevos shards debe entrar como tarea posterior con alcance
  documental propio.
  Revalidacion OrquestaV2 2026-05-25: paquete estricto con contexto `ref_only`
  requerido resuelto por lectura local y recibo ACK; no se amplia write-set ni
  se trunca historico.
- Dar propietario al egress de `DomainWork` HTTP neutral. OPES tiene su propio
  conector y guardas; el HTTP neutral debe tener allowlist/modo smoke y auditoria
  compacta sin duplicar politica OPES ni meter red en `orquesta-domain-work`.
  Estado 2026-05-26: T80 cerrado con validacion transversal limpia. El cliente
  HTTP neutral aplica politica
  `smoke_local`/`allowlist`, rechaza credenciales, query y fragment en base URL,
  bloquea metadata/redes internas no allowlisted, conserva paths relativos sin
  query/host override y el servidor exige modo de egress explicito para
  `ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL`. La revalidacion ejecutada pasa con
  `go test -count=1 ./...`.
- Cerrar la lectura global de workspace como contrato de observability, no como
  mezcla nueva de auditoria, stats Git locales, transcript y comandos shell.
  Estado 2026-05-25: T81 ya tiene contrato inicial
  `WorkspaceTimelineQueryV0`, resource MCP y endpoint server-first compartido;
  quedan adaptadores reales de fuentes historicas donde hoy se declara
  `not_available`.

Prioridad media:

- La particion del backlog debe interoperar con `ContextMaterializedBundleV0` y
  T44: si un shard requerido llega como `ref_only`, el ACK debe demostrar
  lectura o pedir al director, no cerrar por heading conocido.
- La politica de egress HTTP debe aceptar smokes temporales locales de forma
  explicita; bloquear por defecto redes internas/productivas no debe romper el
  caso `EXT-NO-OPES` cuando declara su app temporal.
- La timeline global debe reutilizar T29 para coste/cuota y T41 para redaccion
  de payloads. No crear otro rail de privacidad ni otro contador de uso con
  semantica distinta.

## Priorizacion scanner 2026-05-24 vigesimoquinta pasada

Backlog asociado: `T82 governance-catalog-public-route-shape-sync`,
`T83 operational-status-public-query-source` y
`T84 function-contract-readonly-public-index`.

Prioridad alta:

- Unificar los shapes publicos read-only antes de seguir ampliando CLI/web/MCP.
  `GovernanceCatalog` ya tiene handler y cliente, pero usan envelopes distintos;
  aceptar ambos sin owner produciria otra compatibilidad permanente.
  Estado 2026-05-25: T82 cerrado con shape unico para request, response y error
  publico. Gateway y servidor exponen la ruta read-only; sin provider hay error
  publico recuperable, no fallback documental ni catalogo inventado.
- Publicar el query minimo de `OperationalStatus` desde source residente antes
  de la timeline global T81. Los clientes ya anuncian el endpoint, asi que el
  fallo debe ser `proyeccion_no_disponible` verificable o diagnostico real, no
  transporte inexistente ambiguo.
- Separacion FunctionContract read-only cerrada el 2026-05-25: la consulta viva
  sale del indice causal de eventos/stores; registrar contratos sigue bloqueado
  hasta que exista flujo durable de workflow/outbox.

Prioridad media:

- Los tres endpoints deben pasar por el mismo propietario de gateway y auditoria
  compacta. No crear handlers locales que devuelvan bodies privados, docs
  historicos completos o errores con shape diferente por cliente.
- `OperationalStatus` debe convivir con T81: respuesta de estado actual y
  timeline historica son contratos relacionados, no el mismo rail.
- FunctionContract no se reconstruye desde Markdown ni DB v1 como canon. Si solo
  existe `contract_ref`, se devuelve `evidencia_insuficiente` y queda para que
  el director materialice o repare.

## Priorizacion scanner 2026-05-24 vigesimosexta pasada

Backlog asociado: `T85 server-status-config-snapshot-redaction`,
`T86 bootstrap-appspec-legacy-route-quarantine` y
`T87 server-liveness-readiness-contract`.

Prioridad alta:

- Separar propietario de estado publico y configuracion efectiva del servidor.
  `/api/v0/server/status` no debe ser a la vez health, diagnostico, config local
  y fuga de paths; T83 cubre diagnostico operativo, T85 cubre snapshot de config
  redactado.
- Poner en cuarentena la ruta CLI legacy de bootstrap AppSpec antes de cerrar
  T76. Cerrado 2026-05-26: `app spec bootstrap` devuelve bloqueo publico
  `contrato_no_configurado` con `route_policy`, no llama la ruta legacy
  `/api/v0/director/bootstrap/appspec` y remite a `/api/v0/apps/director` /
  `orquesta.apps.arrancar_director.v0`.
- Corregir el uso de `/healthz` como readiness en smokes y daemon. Liveness del
  proceso no prueba que startup cleanup, reconciliacion o supervisor residente
  esten listos para lanzar efectos externos. Cerrado 2026-05-25 para daemon y
  smokes con Codex/OPES/domain_work/Director/automejora: usan
  `/api/v0/server/readiness`; `/healthz` queda como liveness. Verificado como
  cierre offline en T87 con `go test -count=1 ./modulos/orquesta-server
  ./modulos/orquesta-cli ./cmd/orquesta-server` y `bash -n scripts/*.sh`.

Prioridad media:

- Los endpoints de estado/readiness deben compartir redaccion y errores
  publicos con gateway/auditoria. No crear otro DTO que vuelva a exponer HOME,
  runtime dirs, comandos o provider/model como valores crudos.
- La migracion del bootstrap CLI debe preservar una salida recuperable para
  operadores: ruta vigente sugerida, error estable o compatibilidad real con
  tests. No redirigir silenciosamente a un flujo que no arranca Director cuando
  el usuario pidio juicio operativo.
- Los scripts de smoke pueden seguir usando `/healthz` solo para esperar socket
  vivo en casos sin trabajo externo. Cuando hay Codex, OPES, domain_work o
  automejora, deben comprobar readiness o documentar el bloqueo.

## Priorizacion scanner 2026-05-24 vigesimoseptima pasada

Backlog asociado: `T88 federated-module-backlog-index` y
`T89 director-operativo-local-doc-state-sync`.

Prioridad alta:

- Evitar otra fuente de verdad paralela para backlog. Los `docs/tareas.md`
  locales deben alimentar un indice federado con owner, estado, linea/hash y
  alias Txx, no otra lectura ad hoc del planner ni otro recurso MCP estatico.
- Sincronizar primero las fuentes que cada agente lee por obligacion. Si el
  README local del Director Operativo dice que waits/review/recursion siguen
  pendientes, un scanner puede reabrir trabajo ya cerrado aunque AGENTS y matriz
  digan lo contrario.

Prioridad media:

- El indice federado debe clasificar docs historicos, locales vigentes y tareas
  promocionadas. No programar automaticamente entradas `APG-*`, `SSH-*`,
  `RTDELIVERY-*` o `DEP-*` sin owner, alcance, tests y freshness.
- La sincronizacion documental del Director Operativo debe reutilizar T27 y la
  matriz de smokes; no crear otra lista manual de excepciones para
  `CODEX-WAVE-REAL`, `CODEX-RECURSION-REAL` u OPES temporal pendiente.

Revalidacion 2026-05-25: T88 deja un indice federado inicial en el backlog
global y el planner residente lo consume con owner, estado, alias local, hash y
lease/epoch. Las fuentes locales no vigentes no se programan; las vigentes sin
tests/owner pasan a revision documental. El cierre verde queda pendiente porque
la prueba obligatoria falla por `modulos/orquesta-observability`
(`WorkspaceTimelineV0` indefinido), fuera del write-set de T88.

Resolucion T89 2026-05-26: README/docs locales del Director Operativo quedan
sincronizados con la matriz: no reabren waits, cierre offline, ola Codex amplia
ni recursion Codex real; el pendiente real sigue siendo OPES temporal de
derivados/cierre. El modulo incorpora check documental focal.

## Priorizacion scanner 2026-05-24 vigesimoctava pasada

Backlog asociado: `T90 residual-go-file-budget-splits` y
`T91 smoke-script-ops-library`.

Prioridad alta:

- Convertir el rail de tamano de ficheros Go en shards ejecutables para el
  residuo fuera de T52-T54/T71. Si no hay baseline por fichero y propietario por
  modulo, el limite de 300 lineas queda duplicado entre prompt, review gate y
  criterio humano sin accion verificable.
- Unificar helpers de smokes antes de ampliar scripts reales. Confirmaciones,
  readiness, cleanup y redaccion no deben vivir como copias divergentes en cada
  smoke largo.
  Resolucion inicial 2026-05-26: `scripts/lib/smoke_common.sh` concentra helpers
  compartidos y los smokes OPES largos lo consumen como primer shard verificable.

Prioridad media:

- Los splits residuales deben ser mecanicos o con pruebas focales equivalentes:
  no mezclar reduccion de tamano con cambios de contratos, eventos, refs,
  payloads o orden causal.
- La libreria de smokes debe coordinarse con T21 y T87: catalogo de riesgo y
  readiness operativa son rails de autorizacion; los helpers solo evitan
  duplicacion y errores de forma.
- Los helpers de shell no sustituyen evidencia durable. Salidas de smokes,
  stdout/stderr, payloads HTTP y logs de proveedor siguen sujetos a redaccion,
  retencion corta y refs compactas.

## Priorizacion scanner 2026-05-24 vigesimonovena pasada

Backlog asociado: `T92 capacity-decision-policy-port` y
`T93 codex-prompt-toolbelt-source-sync`.

Prioridad alta:

- Dar propietario unico a la decision de capacidad ejecutable. Hoy el core
  emite `RequestCapacityDecision`, `orquesta-capacity` tiene DTOs/fixtures/tests
  de politica y el stack despacha con `CapacityDecisionExecutorV0` estatico. Sin
  puerto comun, T51 puede fijar defaults de razonamiento pero la seleccion real
  de tier/reasoning/pool/model/quota queda duplicada entre config de servidor,
  stack y docs locales.
  Resolucion 2026-05-26: T92 anadio `CapacityDecisionPolicyPortV0`, adapter
  stack -> `orquesta-capacity` y refs opacas centralizadas en config/env del
  servidor; el fallback estatico queda solo como legacy auditable.
- Sincronizar prompts/toolbelt con los registries reales. `codex_prompt_hints`
  no debe tener una lista manual distinta de HTTP/MCP, runbooks, paquetes de
  agente y `orquesta-mcp`; si una capability como `directed_query` existe solo
  como tool opt-in con puerto ausente, el agente debe ver esa frontera publica,
  no una instruccion contradictoria.

Prioridad media:

- La politica de capacidad debe conservar `orquesta-capacity` como contrato de
  composicion: no importar proveedor, HOME, OAuth, tokens ni cuota real en
  core/runtime/domain-work, y no convertir `medium/high/xhigh` en strings de
  prompt sin evidencia.
- La fuente de toolbelt debe reutilizar las rutas de `orquesta-app-gateway`,
  los descriptors de `orquesta-mcp` y los errores publicos de operador. No
  crear otra tabla paralela dentro de `cmd/orquesta-server` salvo como
  descriptor generado/probado.
- Ambos frentes son sincronizacion de ownership, no relajacion de rails:
  secretos efectivos, payloads crudos, prompts/transcripts y conectores reales
  no inyectados siguen bloqueados o devuelven error publico verificable.

Cierre T93 2026-05-26: los prompts Codex ya usan una fuente compacta
verificable en `cmd/orquesta-server/codex_toolbelt_source_v0.go`: HTTP sale de
constantes publicas de `orquesta-mcp` usadas por gateway y MCP se valida contra
`MCPTransportToolsV0`. La fuente incluye `orquesta.operator.directed_query.v0`,
expone `orquesta.operator.operations.v0` como resource de subtools y distingue
transporte HTTP vivo, tool registrado y puerto opt-in no configurado sin
prometer operador humano, runtime, OPES, DB ni red real no inyectada.
Verificacion OrquestaV2 2026-05-26: agente externo valido contexto `ref_only`
por evidencia explicita y reejecuto la matriz requerida de T93.

## Priorizacion scanner 2026-05-24 trigesima pasada

Backlog asociado: `T94 server-audit-write-failure-visibility`.

Estado 2026-05-26: cerrado local para visibilidad base del fallo de escritura
de auditoria del servidor residente. `modulos/orquesta-server` es el propietario
unico de la proyeccion `audit_*`; HTTP, supervisor, startup y `cmd` consumen el
estado comun en vez de inventar contadores por ruta.

Prioridad alta:

- Separar redaccion/schema de auditoria (T41) de visibilidad de fallo de
  escritura. Un sink JSONL roto no debe desaparecer como best-effort silencioso
  cuando el mismo servidor usa esa auditoria como evidencia operativa de
  automejora, supervisor, startup y control plane. Cerrado con
  `audit_write_failed` en estado/status/readiness.
- Unificar propietario del estado de auditoria degradada en
  `modulos/orquesta-server`: HTTP, supervisor loop, startup checks y
  composicion de `cmd/orquesta-server` no deben inventar cada uno su contador,
  error publico o politica de bloqueo. Cerrado para la proyeccion base `audit_*`.

Prioridad media:

- La proyeccion de fallo debe ser compacta y no recursiva: contador, evento,
  codigo publico y timestamp; sin rutas locales, HOME, permisos exactos,
  payloads HTTP, prompts, transcripts ni tokens. Cerrado localmente con tests de
  sink roto y state store roto.
- Coordinar con T19/T41 antes de exponer visor o retencion: un fallo de
  escritura no autoriza fallback a stdout/stderr crudo ni a otra persistencia
  paralela sin redaccion por campo.

## Priorizacion scanner 2026-05-24 trigesimoprimera pasada

Backlog asociado: `T95 server-state-persist-failure-visibility`.

Prioridad alta:

- Separar fallo de persistencia de estado de fallo de auditoria. T94 cubre sink
  JSONL; T95 debe cubrir `StateStore` y estado durable del servidor, porque CLI,
  restart, startup checks y status pueden leer una foto distinta a la memoria
  viva.
- Unificar propietario en `modulos/orquesta-server`: `persistStateV0`, startup
  checks, supervisor loop, idle autoprogramming, shutdown y `cmd/orquesta-server`
  no deben inventar contadores o codigos publicos distintos para
  `state_persist_failed`.

Prioridad media:

- La proyeccion degradada debe ser compacta y no recursiva: contador,
  transicion afectada, codigo publico y timestamp; sin rutas locales, HOME,
  permisos exactos, payloads HTTP, prompts, transcripts ni tokens.
- Coordinar con T87 y T94: liveness puede seguir vivo, readiness puede degradar,
  y auditoria puede registrar el fallo solo si su sink esta sano. Ninguna de
  esas capas sustituye la evidencia de estado durable confirmado.

## Priorizacion scanner 2026-05-24 trigesimosegunda pasada

Backlog asociado: `T96 server-daemon-stop-process-identity`,
`T97 server-daemon-log-redaction-retention` y
`T98 external-bridge-input-ledger-claim-recovery`.

Prioridad alta:

- Unificar el rail de parada del daemon residente. `orquesta-server stop`,
  shutdown HTTP, state file y registry de procesos no deben tener criterios
  distintos para identidad de proceso, forced/cooperativo y estado stale. T60
  cubre olas Codex; T96 cubre el proceso servidor.
  Cierre T96 2026-05-26: el daemon residente usa `process_ref` y
  `daemon_epoch_ref` opacos en state/status, `stop` verifica correspondencia
  HTTP-state antes de senalar y `forced` queda como opt-in con `--reason`.
- Convertir el ledger de entrada de bridges externos en claim/recovery antes de
  crear runs. Cierre 2026-05-26: `cmd/orquesta-server` ya registra claim
  `claimed` con refs de correlacion/idempotencia antes del POST, bloquea drains
  concurrentes, rechaza sobrescritura de `submitted` con otro `run_ref` y usa
  `external_bridge_recovery_required` cuando hay que reconciliar tras fallo de
  finalizacion. T31 sigue protegiendo la cola interna; T98 protege el job
  externo de entrada.

Prioridad media:

- Separar logs operacionales del daemon de auditoria JSONL, tail Codex y
  required-test output. Si se exponen, deben ser resumenes redactados con
  retencion/rotacion y no una nueva fuente terminal de cierre.
- Las tres tareas deben compartir politica de redaccion por campo: refs opacas,
  codigos publicos y contadores pasan; HOME, rutas privadas, PIDs como verdad
  unica, argv completos, prompts, transcripts, tokens, payloads HTTP completos
  y respuestas OPES crudas no se persisten como evidencia de cierre.

Revalidacion 2026-05-26: T97 queda cerrada para el servidor residente. El
daemon tiene owner/politica propia, rotacion y retencion local; status y
diagnostico solo publican resumen/politica redactada; stdout/stderr crudos no
se capturan por defecto y el opt-in local queda separado de auditoria, tail
Codex y required-test output.

## Priorizacion scanner 2026-05-24 trigesimotercera pasada

Backlog asociado: `T99 server-http-resource-guardrails`,
`T100 startup-revision-archive-redaction-retention` y
`T101 domain-work-artifact-submission-ledger-recovery`.

Prioridad alta:

- T99 queda cerrado en el primer alcance residente: servidor, MCP HTTP,
  app-gateway y web comparten politica de recursos por perfil para timeouts,
  decode JSON, tamano de body, trailing tokens y auditoria compacta. T102 queda
  como owner de handlers publicos/legacy fuera de este write-set.
- Separar archivo de revision de startup de purga/reconciliacion. T35 decide
  si compactar y T59 protege purga de runtime; T100 debe gobernar permisos,
  redaccion, retencion y exposicion publica de los artefactos generados.
- El ledger de salida `domain_work` ya tiene claim/recovery offline para
  `submit_artifact`: claim antes del efecto, bloqueo de claims no terminales y
  conflicto terminal compacto. T98 protege entrada de jobs; T101 protege el
  efecto externo de devolver artefactos.

Prioridad media:

- La auditoria de HTTP ya registra metadatos compactos del request residente
  para T99; T41/T94 siguen gobernando detalle prohibido y otros artefactos de
  diagnostico.
- El archivo de revision puede conservar material crudo solo como diagnostico
  local opt-in con limite; no debe entrar en contexto, AppVCS, promocion,
  timeline ni recursos MCP.
- Los ledgers de entrada/salida deben compartir conceptos de idempotency,
  correlation, receipt y conflicto, pero sin fusionar OPES con el contrato
  neutral `domain_work`.

Revalidacion 2026-05-26: T100 queda cerrado para el archivo de revision de
startup. `cmd/orquesta-server` genera `revision_ref` opaco, snapshots redactados
por campo, manifest con retencion/limite/permisos y archivo de runtime solo como
metadata redactada; `modulos/orquesta-server` proyecta `startup_revision` en
readiness/status sin rutas locales.

Revalidacion 2026-05-26: T101 queda cerrado para claim/recovery offline de
salida `domain_work`. El stack registra `claimed/submitting` antes del efecto
externo, bloquea claims no terminales con recovery compacto y conserva
accepted/rejected terminales sin sobrescritura conflictiva.

## Priorizacion scanner 2026-05-24 trigesimocuarta pasada

Backlog asociado: `T102 legacy-http-json-boundary-policy`,
`T103 outbound-http-response-limit-redaction` y
`T104 file-store-durable-write-policy`.

Prioridad alta:

- Unificar la frontera JSON de handlers HTTP antes de ampliar endpoints publicos
  o transporte MCP residente. T99 cubre recursos del servidor; T102 debe evitar
  que factory/governance/MCP/web/app-gateway conserven decoders, limites y
  shapes de error incompatibles.
- T103 queda cubierto el 2026-05-26 para conectores OPES/domain-work, CLI, web,
  app-gateway por cliente web in-process y comandos de servidor: limite de
  respuesta, `Content-Type` JSON o legacy `text/plain` con JSON parseable,
  trailing data y errores publicos redactados. T80 sigue decidiendo adonde puede salir
  `domain_work-http`.
- Revalidacion T103 2026-05-26: la bateria focal del alcance sigue verde y no se
  abre duplicado nuevo frente a T138.
- Revalidacion OrquestaV2 2026-05-26:
  `agent-ref-task-autoprogramming-49b26a6e419d-g01` mantiene T103 cerrado como
  canon y refuerza la prueba de `text/plain` legacy con JSON parseable en el
  write-set asignado.
- Normalizar escritura durable file-based antes de depender de restart,
  reconciliacion, ledgers y estado residente como prueba terminal. T95 detecta
  fallo de estado, pero T104 debe cerrar permisos, fsync, temp/lock y recovery
  de snapshot.

Revalidacion 2026-05-26: T104 queda cerrado local para stores y ledgers del
write-set. `orquesta-run-file`, `orquesta-domain-work-file`,
`orquesta-runtime-codex-delivery`, `orquesta-server`,
`orquesta-app-codex-stack` y `cmd/orquesta-server` usan temp unico o append
durable segun contrato, permisos `0600`, `fsync`/close, `rename` y sync de
directorio donde aplica. Los control files Codex de prompt/wrapper/ACK no se
fusionan aqui; siguen como T159.

Revalidacion OrquestaV2 2026-05-27: T104 queda cerrado con bateria obligatoria
completa del alcance:
`go test -count=1 ./modulos/orquesta-run-file ./modulos/orquesta-domain-work-file ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-server ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
No se abre duplicado nuevo para control files Codex; T159 conserva ese owner.

Prioridad media:

- Los helpers HTTP deben preservar compatibilidad legacy de forma explicita y
  con tests; no endurecer campos desconocidos si rompe clientes finos vigentes
  sin plan de migracion.
- La redaccion de respuestas debe compartir politica con T41/T94/T97, pero no
  convertir cuerpos externos en auditoria paralela ni logs operacionales.
- La politica durable puede reutilizar helpers existentes de `orquesta-run-file`
  y `orquesta-domain-work-file`, pero debe respetar owners concretos de outbox,
  cola, ledgers OPES/domain_work y runtime Codex.

Revalidacion 2026-05-26: T102 queda cerrado para los handlers del write-set.
`orquesta-factory-http`, governance, MCP, web y `/mcp` del servidor tienen
limites JSON, trailing-token check, politica de `Content-Type`, modo de campos
desconocidos documentado por perfil y errores publicos compactos con
correlacion cuando existe. `orquesta-app-gateway` y `orquesta-http-gateway`
siguen como compositores de handlers inyectados, sin parseo JSON propio.

## Priorizacion scanner 2026-05-24 trigesimoquinta pasada

Backlog asociado: `T105 process-runtime-launch-env-io-receipt`,
`T106 codex-wave-public-summary-redaction` y
`T107 real-smoke-go-diagnostic-redaction`.

Prioridad alta:

- Separar launch neutral de stop neutral. T66 gobierna parada causal; T105 debe
  gobernar recibo de resolucion de comando/env/working dir e IO para que el
  runtime no dependa de valores reales invisibles ni de stdout/stderr crudo.
  Actualizacion 2026-05-26: T105 queda cubierto por
  `ProcessRuntimeLaunchReceiptV0` y refs compactas en registry/stats; no reabrir
  stop neutral, logs daemon ni salida publica de `codex-wave`.
- Redactar la salida publica de `codex-wave` antes de usarla como evidencia o
  diagnostico compartible. Tail/purge/stop tienen owners propios, pero launch y
  status aun pueden exponer rutas y ficheros de control completos.
  Actualizacion 2026-05-26: T106 queda cubierto en
  `cmd/orquesta-server`; stdout de launch/status/stop/director-wave usa
  summaries publicos con refs opacas y la registry local queda como artefacto
  operativo no publico. T58 conserva el owner de fragmentos de log opt-in.
- Cubrir smokes Go opt-in, no solo scripts shell. Los fallos de pruebas reales
  deben ser utiles para operador sin imprimir prompts, transcripts, bodies,
  stdout/stderr completos ni rutas privadas.
  Actualizacion 2026-05-26: T107 queda cubierto localmente para los helpers de
  smokes Go reales Codex y el harness Go de procesos: salida redactada/acotada
  por defecto y crudo solo como artefacto local opt-in, no como evidencia
  terminal.

Prioridad media:

- Los recibos de launch, summaries de ola y diagnosticos de smokes pueden
  compartir helper de redaccion por campo, pero cada owner conserva semantica:
  runtime neutral, comando Codex local y pruebas reales.
- El modo crudo puede existir solo como diagnostico local opt-in con limite,
  retencion y causa publica; no debe alimentar ACK, cierre, auditoria JSONL,
  timeline, AppVCS ni contexto de agentes.
- No convertir rutas reales de runtime, HOME, CODE_HOME, PID, prompt path o
  stdout/stderr path en refs de producto. Si hace falta correlacion, usar refs
  opacas y hash/receipt.

## Priorizacion scanner 2026-05-24 trigesimosexta pasada

Backlog asociado: `T108 decision-council-operational-rounds` y
`T109 agent-lease-progress-policy-bridge`.

Prioridad alta:

- Dar owner operativo a las rondas de consejo. Cerrado offline focal el
  2026-05-26: `orquesta-decision-council` expone rondas operativas y
  `orquesta-orchestration-core` las materializa como `WorkflowTaskV0` con
  gate/wait scope por cohorte/ola, deps causales y aceptacion final por
  `AcceptDecision` tras `DecisionCouncilVoteV0` aceptado. No reabrir salvo
  regresion o smoke con proveedor real explicitamente pedido.
- Unificar leases y progreso. `orquesta-core-leases`, heartbeat de
  `orquesta-runtime`, observacion Codex y scheduler de lease actions no deben
  mantener umbrales paralelos para stopped/stalled/loop/retry/stop/replan.
  Actualizacion 2026-05-26: T109 queda cubierto por
  `AgentProgressLeaseBridgeV0` y cableado Codex opt-in con budget de progreso;
  la misma observacion compacta alimenta supervision y lease assessment, y el
  scheduler consume `LeaseActionCandidates` sin payloads sensibles.

Prioridad media:

- El consejo de decision debe seguir puro: familias, capacidad y evidencias son
  refs opacas; proveedor/modelo/cuota/runtime pertenecen a capacidad y
  composicion.
- El puente leases/progreso debe recibir reloj y snapshots por adaptador, no
  desde core. Sus resultados son refs, reason codes y commands idempotentes,
  no PID, paths, stdout/stderr ni payloads de runtime.
- Ambas tareas deben coordinarse con T31/T57/T69/T105: leases de cola, outbox,
  razones de parada y recibos de launch son rails vecinos, no sustitutos.

## Priorizacion scanner 2026-05-24 trigesimoseptima pasada

Backlog asociado: `T110 director-scheduler-cycle-rail-policy-sync`,
`T111 local-agents-required-doc-refs-sync` y
`T112 run-queue-workset-concurrency-bridge`.

Estado T110 2026-05-26: cerrado localmente. El scheduler y el ciclo de
dispatch siguen con owners separados, pero ambos delegan la semantica de detalle
sensible en `orquesta-rails`; no bloquean vocabulario operativo opaco como
`runtime`, `provider`, `model`, `db`, `sql`, `filesystem` o `docker`.

Prioridad alta:

- El scheduler y el ciclo de dispatch del Director estan en la ruta viva de
  ejecucion; sus listas locales de terminos no deben contradecir la politica
  comun de `orquesta-rails` ni convertir vocabulario operativo en bloqueo
  generico antes de que el director normalice.
- Las instrucciones locales obligatorias deben apuntar a documentos existentes.
  Una ruta historica ausente en `AGENTS.md` no puede seguir siendo prerequisito
  implicito para agentes externos gobernados por OrquestaV2.
- La cola global necesita un puente de concurrencia por work-set antes de
  ampliar automejora residente en paralelo. T31 evita doble claim de la misma
  run; T112 debe evitar launch concurrente de runs distintas que pisan el mismo
  alcance.

Prioridad media:

- No fusionar todos los rails en un helper global opaco: scheduler, dispatch,
  contexto requerido y run queue conservan owners distintos, pero deben
  compartir semantica de refs opacas, secretos efectivos y errores publicos.
- La reparacion de docs locales debe preferir sustituir refs por la foto vigente
  o marcar historico; no restaurar carpetas stale solo para satisfacer rutas.
- El puente de work-set debe usar claims declarados y refs compactas; snapshot,
  Git real y verificacion destructiva siguen en runtime/worktree y ACK
  terminal, no en `orquesta-run-queue`.

## Priorizacion scanner 2026-05-24 trigesimoctava pasada

Backlog asociado: `T113 module-historical-doc-ref-sync` y
`T114 run-queue-fairness-group-policy`. `T115 web-app-intake-session-contract`
queda cerrado el 2026-05-26 con contrato web de sesion y handoff compacto.

Prioridad alta:

- Separar contexto historico de contexto requerido vivo. T111 ya sincronizo
  instrucciones obligatorias en `AGENTS.md`; T113 debe limpiar docs locales que
  aun apuntan a `docs/reinicio_orquesta_v2` para que el indice federado no
  convierta DBV1/control-plane stale en tareas ejecutables.
- Dar semantica a `fairness_group_ref` antes de ampliar colas paralelas de
  automejora. T31 reserva, T112 evita solapes de write-set y T114 evita hambre
  o monopolio entre grupos con politica secundaria por prioridad, reason codes
  publicos y reloj inyectado.

Prioridad media:

- `WEB-013` queda cubierto por `WebNuevaAppIntakeSessionV0`: la sesion conserva
  AppSpec parcial, preguntas i18n, refs opacas y handoff al Director V2; la web
  no decide arquitectura, runtime, proveedor ni DB.
- T113 no debe borrar documentos historicos ni recrear carpetas ausentes; debe
  marcar fuente, freshness y sustituto vigente.
- Estado T113 2026-05-26: cerrado para el write-set asignado. Las refs locales
  a reinicio v2 quedan historicas/stale con sustitutos vigentes, y
  `modulos/*/docs/tareas.md` ya no usa el inventario DB v1 como prerequisito
  ejecutable.
- Fairness de cola no puede saltarse guardas de smokes reales, OPES temporal,
  proveedor, DB o efectos externos. Solo ordena candidatos ya ejecutables y
  confirmados por sus rails propios.

## Priorizacion scanner 2026-05-24 trigesimonovena pasada

Backlog asociado: `T116 module-task-doc-integrity-linter`,
`T117 app-codex-review-gate-policy-owner` y
`T118 legacy-generated-doc-artifact-contract-sync`.

Prioridad alta:

- Blindar la integridad de docs locales antes de ampliar el indice federado:
  IDs duplicados como `APP-CODEX-STACK-012` y estados stale en pruebas CLI no
  pueden decidir terminalidad de tareas ni alimentar automejora residente.
- Dar owner unico al review gate de entregas Codex. La regla de tamano, tests,
  write-set y progreso real debe vivir como politica verificable, no repartida
  entre prompt, ACK, snapshot, review gate y narrativa local.
- Separar planes documentales historicos/debug de contratos vivos de Orquesta.
  Los artefactos esperados de apps generadas no deben convertirse en docs raiz
  obligatorios del nucleo por estar citados en `docs/plan_microtareas.md`.

Prioridad media:

- T116 debe coordinarse con T43/T79/T88: primero evitar colisiones y estados
  ambiguos, despues partir backlog o federar tareas por modulo.
- T117 debe coordinarse con T42/T45/T61: ACK estricto, presupuesto de ficheros y
  redaccion de tests son rails vecinos, no sustitutos de la politica de review.
  Resolucion 2026-05-26: el owner ejecutable queda en
  `ReviewGateDeliveryPolicyPortV0` dentro de `orquesta-app-codex-stack`; el
  recibo de snapshot/write-set queda en `orquesta-runtime-worktree`; prompt y
  ACK solo transportan instrucciones/evidencia, no deciden cierre.
  Revalidacion 2026-05-26: T117 concentra el owner ejecutable en
  `codexStackReviewGatePolicyV0`; el ACK no decide presupuesto, el snapshot no
  decide severidad y el review gate aplica la politica con aliases/globs seguros
  y perfiles documentales/no-Go sin forzar reglas Go. La evidencia focal queda
  en el test de los cuatro modulos del write-set T117 y en los tests locales de
  `review_gate_policy_v0`.
  Revalidacion OrquestaV2 retry 2026-05-26: no se duplica rail nuevo; T117 queda
  cerrado por owner ejecutable en stack, recibo worktree y prueba focal del
  write-set completo.
- T118 queda cerrado como sincronizacion documental: planes historico/debug,
  artefactos de app generada como plantillas/refs del proyecto objetivo, sin
  borrar docs antiguos ni recrear carpetas stale para satisfacer pruebas
  documentales heredadas.

## Priorizacion scanner 2026-05-24 cuadragesima pasada

Backlog asociado: `T119 server-first-usage-doc-route-sync`,
`T120 legacy-sqlite-forensic-doc-quarantine` y
`T121 manual-agent-ops-compatibility-contract`.

Prioridad alta:

- Sincronizar manuales de uso antes de que el residente o un agente externo los
  trate como API viva. Las rutas legacy sin `/api/v0`, OpenClaw y AP-077 no
  deben competir con README/AGENTS/foto vigente ni con clientes finos actuales.
- Cuarentenar snapshots SQLite/DBV1 y rutas absolutas historicas en docs locales.
  La evidencia forense puede conservarse, pero no debe convertirse en
  prerequisito de ejecucion ni en persistencia global de Orquesta.
- Acotar wrappers manuales como recuperacion asistida. `scripts/inicio_agente.sh`
  y Terminator no deben ser fuente paralela de runtime, sesiones, worktrees ni
  cierre frente al daemon residente.

Resolucion T121 2026-05-26:

- Los wrappers manuales quedan cerrados como recuperacion asistida subordinada a
  readiness `/api/v0/server/readiness` y CLI/API publica.
- `cargar_agentes.sh` y `terminator_agentes.sh` conservan dry-run por defecto;
  no siembran flotas legacy sin confirmacion.
- `agente_console.sh` bloquea runtime directo y conserva solo la consola de
  operador.

Prioridad media:

- T119 se coordina con T27/T82/T83/T87: documentacion, rutas publicas, estado
  operacional y readiness son fuentes relacionadas, no listas manuales
  independientes.
- T120 se coordina con T88/T116 y `orquesta-domain-work-sql`: indice federado y
  docs locales deben distinguir forense/historico/effective sin promover DBV1 ni
  un driver SQL real al nucleo.
- T121 se coordina con T28/T30/T66/T96: recuperacion de agentes vivos,
  checkpoint, stop neutral y daemon identity gobiernan el ciclo; wrappers
  manuales solo pueden llamar a esas superficies o declarar bloqueo.

Resolucion T119 2026-05-26:

- `docs/uso_actual_app_orquesta.md` queda como manual server-first de la
  composicion actual y lista solo rutas publicas versionadas como superficie
  viva para clientes nuevos.
- `README.md` y `docs/README.md` enlazan ese manual y declaran historicos los
  manuales V1 que anuncien `./orquesta serve`, rutas `/api/*` sin version,
  OpenClaw, AP-077 como requisito operativo o DBV1.
- La ruta sin version `/api/status` queda documentada como alias legacy de
  compatibilidad; el uso nuevo debe citar `/api/v0/server/status` y
  `/api/v0/server/readiness`.
- Estado sincronizado en backlog: T119 queda cerrado; las secciones V1
  preservadas en `docs/uso_actual_app_orquesta.md` son cuarentena historica, no
  fuente para clientes web/CLI nuevos ni para pruebas de cierre.

Cierre T120 2026-05-26: snapshots SQLite/DBV1 quedan como evidencia forense en
cuarentena con refs relativas opacas, no como prerequisito vivo ni persistencia
global. T88/T116 deben conservar la separacion `forense`/`historico`/
`quarantine`/`effective` y requerir decision explicita del director antes de
programar lecturas DBV1 o comandos `sqlite3`.

## Priorizacion scanner 2026-05-24 cuadragesima primera pasada

Backlog asociado: `T122 canonical-doc-precedence-self-sync`,
`T123 factory-backlog-preview-director-handoff` y
`T124 legacy-external-orchestrator-doc-quarantine`.

Prioridad alta:

- Arreglar precedencia documental desde los propios documentos historicos. Un
  indice o agente que lea primero `BIBLIA_APP_ORQUESTA.md` no debe ver una
  autoridad canonica superior a `AGENTS.md`, `README.md` y la foto vigente.
- Separar preview determinista de backlog de plan operativo. `orquesta-factory`,
  web, MCP e intake no deben duplicar el juicio del Director ni convertir
  `BacklogInicialPropuestoV0` en cola ejecutable sin handoff causal.
- Cuarentenar docs de OpenClaw, `tmux` y MCP real antes de que el indice
  federado o el residente los use para abrir rutas, transportes o procesos no
  cableados.

Prioridad media:

- T122 coordina con T27/T119/T120/T116: el objetivo no es borrar historia, sino
  dar freshness, estado y sustituto vigente a cada fuente.
- T123 coordina con T76/T115/T118: AppSpec, intake web y artefactos de app
  generada son vecinos, pero solo el Director produce plan operativo ejecutable.
- T124 coordina con T16/T23/T30/T66/T121: MCP real, shutdown, stop neutral y
  wrappers manuales siguen como adaptadores opt-in o recuperacion, no nucleo.

Resolucion T123 2026-05-26: T123 queda cerrado para el alcance factory/HTTP/web/
MCP/intake. El backlog inicial queda marcado como `preview_no_ejecutable`, lleva
freshness y handoff causal a `orquesta.apps.arrancar_director.v0`, y las
superficies publicas lo muestran como preview compacta/insumo. Ninguna de esas
capas materializa cola, plan-state, wait, review, tests ni cierre.

Resolucion T124 2026-05-26: T124 queda cerrado como cuarentena documental. Los
docs de OpenClaw, `tmux`, Terminator, MCP real y transportes externos conservan
historia pero no alimentan planificacion automatica; el indice federado los
declara `quarantine`, y cualquier activacion requiere tarea explicita de
composicion externa/legacy con decision y tests propios.

Resolucion T122 2026-05-26: `BIBLIA_APP_ORQUESTA.md` se autoidentifica como
historico/stale con sustitutos vigentes; `00_INDICE.md` y `docs/README.md`
separan fuentes vigentes, historicas, forenses y plantillas. La historia se
conserva, pero no puede gobernar planes actuales sin enlace a la foto vigente.
Refuerzo retry 2026-05-26: los textos internos V1 que se declaraban
"fuentes de verdad" o "doctrina" quedan nombrados como historicos para que
T88/T116 prioricen marcadores de stale y sustituto vivo frente al cuerpo antiguo.

## Priorizacion scanner 2026-05-24 cuadragesima segunda pasada

Backlog asociado: `T125 work-profiles-empty-module-quarantine`,
`T126 module-local-codex-launcher-compatibility-contract` y
`T127 ignored-local-artifact-exclusion-policy`.

Prioridad alta:

- Cerrar la fuente de verdad de perfiles de trabajo. `orquesta-work-profiles`
  existe como path vacio, pero el contrato vivo esta en `orquesta-core-workflow`
  y la resolucion en `orquesta-orchestration-core`; el indice federado no debe
  convertir un placeholder en owner nuevo.
- Gobernar los wrappers `arrancar_codex.sh` por modulo. T121 cubre scripts
  manuales raiz, pero estos wrappers locales pueden estar rotos por helper
  comun ausente y competir con daemon/cola si docs locales los anuncian como
  camino vigente.
- Pasar de `.gitignore` a politica ejecutable para artefactos locales. T127
  queda cerrado el 2026-05-26 con exclusion por categoria/reason code en
  snapshots, contexto, AppVCS, promocion y ACK terminal; `.gitignore` queda como
  defensa de operador, no como contrato unico.

Prioridad media:

- T125 debe coordinarse con T51/T117: perfiles, razonamiento y review gate son
  contratos vecinos, pero no se corrigen creando un modulo vacio como owner.
- T126 debe coordinarse con T121/T30/T66/T96: wrapper manual, checkpoint,
  parada neutral e identidad de daemon conservan owners separados; el wrapper
  solo puede llamar a esas superficies o quedar historico.
- T127 queda coordinado con T43/T50/T56/T59: merge documental, control files,
  proyeccion de credenciales y purga de runtime no deben duplicar listas
  parciales. La salida publica es categoria/contador/ref, no path local ni
  contenido crudo.
- Cierre T127 2026-05-26: la lista de ignores queda subordinada a la politica
  ejecutable y a recibos por categoria. `.gitignore` se mantiene como defensa
  local sincronizada, no como criterio unico para snapshots, contexto, AppVCS,
  promocion ni ACK terminal.
- Refuerzo puntual T127: `.orquesta-logs`, `orquesta.env`, `orquesta.db*` y
  `.orquesta-inbox.md` se tratan como artefactos locales no exportables.

## Priorizacion scanner 2026-05-24 cuadragesima tercera pasada

Backlog asociado: `T128 agent-progress-supervisor-rail-policy-sync`,
`T129 server-daemon-start-env-policy` y
`T130 architecture-guard-test-rail-policy-owner`.

Prioridad alta:

- Unificar el rail del supervisor de progreso antes de que oculte evidencia
  operacional valida. La decision del Director necesita refs opacas completas,
  no una lista local que borre palabras como `provider`, `model` o `runtime`.
- Revision 2026-05-26: T128 sustituye la lista local del supervisor por el
  helper comun por campo de `orquesta-rails` y alinea runtime/leases para
  permitir vocabulario operativo opaco, conservando cortes de secreto, HOME real
  y contenido crudo.
- Acotar el entorno efectivo de `start` del daemon. Heredar el proceso padre sin
  recibo de categorias/redaccion mezcla conveniencia operativa con superficie
  sensible de HOME, tokens, proveedor y diagnostico.
  Cerrado 2026-05-26: `start` usa allowlist y recibo `effective_config` por
  categorias/conteos sin valores crudos; Codex, bootstrap, logs y runtime quedan
  como owners separados.
- Separar tests de arquitectura de railes por vocabulario. Los imports/efectos
  prohibidos deben seguir fallando, pero refs y comentarios validos no deben
  depender de substrings globales copiados por modulo.

Prioridad media:

- T128 debe coordinarse con T109/T110: leases, progreso, scheduler y outbox
  pueden compartir helper de politica, pero no deben perder su semantica causal.
- T129 debe coordinarse con T56/T85/T97/T105: proyeccion Codex, bootstrap,
  logs y entorno de runtime siguen owners distintos; el recibo debe unirlos sin
  centralizar secretos.
- T130 debe coordinarse con T70/T77/T116: docs, app-change y freshness
  documental necesitan neutralidad, no listas textuales nuevas por cada test.

## Cerrado 2026-05-26: T130 architecture-guard-test-rail-policy-owner

- `modulos/orquesta-rails` es el owner comun para helpers de tests de
  arquitectura que distinguen imports concretos de literales sensibles.
- Los modulos del alcance conservan listas locales solo para imports, prefijos o
  fragmentos de paquetes reales; ya no escanean vocabulario global como `runtime`,
  `http`, `mcp`, `sql` o `provider` sobre todo el fuente.
- La politica compartida permite refs opacas y nombres de politica, y bloquea
  solo valores sensibles efectivos o adaptadores concretos.

## Priorizacion scanner 2026-05-24 cuadragesima cuarta pasada

Backlog asociado: `T131 documentation-local-path-redaction-linter`,
`T132 observability-privacy-taxonomy-rail-sync` y
`T133 module-boundary-local-agent-doc-coverage`.

Prioridad alta:

- Bloquear la propagacion de rutas locales como evidencia publica. La auditoria
  runtime puede documentar variables y refs opacas, pero contexto, snapshots,
  ACKs y backlog no deben convertir HOME, state dirs ni ficheros de control en
  artefactos de producto.
- Unificar la taxonomia de privacidad de observability con `orquesta-rails` y
  auditoria. Observability puede conservar contrato propio de evento, pero no
  otra lista de substrings que contradiga la politica comun o la redaccion por
  campo.
- Cubrir modulos sensibles sin guia local. `orquesta-rails`,
  `orquesta-domain-work-http` y `orquesta-factory-http` deben tener owner
  documental o exclusion explicita antes de que el indice federado los trate
  como modulos planificables normales.

Prioridad media:

- T131 debe coordinarse con T43/T50/T85/T127: merge, control files, status
  publico y artefactos ignorados comparten frontera, pero el foco aqui es
  documentacion/contexto/ACK.
Cierre T131 2026-05-26: la frontera queda cubierta en contexto materializado,
ACK Codex y linter documental de autoprogramacion. La evidencia publica queda
redactada por documento/linea y conserva variables/refs opacas como sustituto.
- T132 debe coordinarse con T41/T81/T85: audit payloads, timeline y status
  publico consumen privacidad/redaccion, pero no deben poseer tres politicas
  incompatibles.
Cierre T132 2026-05-26: la taxonomia visible de privacidad/redaccion queda en
`orquesta-rails`; observability, auditoria/status, MCP y web consumen esa
politica mediante `privacy.redaction_level` y refs opacas permitidas, sin listas
paralelas por substring para `*_policy_ref` o `*_redaction_ref`.
- T133 queda cerrado para su alcance el 2026-05-26: refs documentales locales,
  linter de integridad y modulos placeholder siguen como vecinos, pero
  `orquesta-rails`, `orquesta-domain-work-http`, `orquesta-factory-http`,
  `orquesta-context` y `cmd/orquesta-server` ya tienen guia local o test
  documental que falla con `module_boundary_local_agent_doc_missing` antes de
  preparar una automejora sobre fronteras sensibles sin cobertura.
  Revalidacion OrquestaV2 retry 2026-05-26: el cierre se conserva sin crear otro
  rail local; el owner verificable sigue siendo el linter documental residente
  y las guias locales del alcance.

## Priorizacion scanner 2026-05-24 cuadragesima quinta pasada

Backlog asociado: `T134 server-status-legacy-alias-sunset`,
`T135 idle-self-improvement-planner-fallback-safety` y
`T136 resident-drain-external-wait-budget-policy`.

Prioridad alta:

- Aplicacion T134 2026-05-26 revalidada en retry OrquestaV2:
  `/api/v0/server/status` queda canonico en comando y clientes finos;
  `/api/status` queda solo como alias legacy con owner, headers de
  deprecacion/canonical y mismo DTO redactado que la ruta versionada. El CLI,
  toolbelt y docs no deben normalizar una ruta sin version como contrato
  vigente. La prueba
  `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-cli ./modulos/orquesta-web ./cmd/orquesta-server`
  pasa completa.
  Retry 2026-05-27 fija owner y sunset como headers publicos del alias:
  `X-Orquesta-Status-Owner=orquesta-server` y
  `X-Orquesta-Status-Sunset-Policy=no_new_use`.
- Hacer seguro el fallback del planner de automejora. Un backlog ilegible,
  vacio o ambiguo debe producir bloqueo/revision o scanner documental acotado,
  no una tarea generica de codigo con write-set historico.
- Gobernar `MaxExternalWaits` por modo. El residente puede mantener smokes
  rapidos, pero Codex real, OPES temporal y operador necesitan presupuesto
  observable sin confundir espera externa viva con idle.

Estado T135 2026-05-26: cerrado para el fallback degradado. El planner no hereda
la request base generica ante documento ilegible; en `capacity_free` con backlog
ya conocido no prepara scanner nuevo, y el residente no convierte errores del
puerto planner en automejora de codigo.

Prioridad media:

- T134 coordina con T119/T85/T87: manuales, estado publico y readiness siguen
  siendo owners separados; aqui se decide el alias runtime.
- T135 coordina con T17/T33/T37/T43/T79: reservas, ACK terminal, artefactos,
  merge documental y plan asociado son vecinos, pero el foco es impedir que el
  scanner degradado amplie scope.
- T136 coordina con T32/T68/T69: wakeups, timeouts y taxonomia de parada no se
  reabren; esta tarea fija presupuesto de espera externa y status efectivo del
  residente.

Estado 2026-05-26: T136 queda cerrado para servidor residente y stack Codex. La
politica efectiva mantiene `MaxExternalWaits=1` en modo residente, publica ese
valor en configuracion efectiva, bloquea overrides incompatibles con error
publico recuperable y bloquea
automejora idle/capacity cuando el supervisor observa `wait_external` o
`candidate_pending` con refs vivas.

## Priorizacion scanner 2026-05-24 cuadragesima sexta pasada

Backlog asociado: `T137 public-http-request-body-bounds`,
`T138 outbound-http-response-bounds-redaction` y
`T139 command-output-public-shape-contract`.

Prioridad alta:

- Unificar lectura de cuerpos HTTP publicos. Los handlers MCP/web/gateway no
  deben decidir por separado limites, `Content-Type`, trailing data, campos
  desconocidos ni error publico.
- Limitar respuestas salientes antes de stdout/stderr/auditoria. Un peer
  defectuoso o mal configurado no debe poder convertir HTML, transcript, ruta
  local, token o payload de dominio en error publico.
- Definir shape publico de comandos antes de que agentes o toolbelts consuman
  `status`, `run-status`, `opes-drain-once` o `codex-wave-*` como fuente de
  verdad.

Prioridad media:

- T137 coordina con T55/T82/T134: exposicion de control plane, shapes publicos
  y alias legacy son vecinos, pero aqui el owner es parseo de entrada.
- Cierre 2026-05-26: T137 queda cerrado para MCP/HTTP publico, web JSON/form,
  gateway/app-gateway y `/mcp` JSON-RPC real. La evidencia focal cubre perfiles
  control, autoprogramacion, domain_work y MCP JSON-RPC sin mezclar limites.
- Retry 2026-05-26: la composicion `cmd/orquesta-server` deja de duplicar la
  lectura JSON entre `/mcp` y workspace timeline; ambos pasan por helper local
  perfilado, conservando limite compatible JSON-RPC y limite control-plane para
  timeline.
- T138 coordina con T41/T85/T131: auditoria, status y rutas locales comparten
  redaccion, pero aqui el owner es respuesta HTTP recibida por clientes.
  Estado 2026-05-26: T138 cerrado para su write-set; queda como politica comun
  reutilizable, no como duplicado abierto de cada cliente HTTP.
- T139 coordina con T58/T61/T85/T138: logs, tests, status y respuestas HTTP
  alimentan comandos, pero el cierre terminal sigue en ACK/eventos/evidencias
  estructuradas, no en stdout.

Estado 2026-05-26: T139 queda cerrado para el write-set de servidor/stack. Los
comandos cubiertos publican schema/freshness/redaction/canonical source; salidas
diagnosticas locales quedan acotadas y los smokes OPES conservan shape operativo
sin exponer URLs/base URLs crudas. No reabre T58/T85/T131/T138.

## Priorizacion scanner 2026-05-24 cuadragesima septima pasada

Backlog asociado: `T140 backlog-overlap-canonicalization` y
`T141 public-boundary-no-panic-contract`.

Prioridad alta:

- Consolidar solapes ya escritos en el backlog antes de lanzar mas agentes
  sobre HTTP request/response bounds. `T102`/`T137` y `T103`/`T138` deben tener
  canon/alias y merge de criterios; no borrar historia ni declarar cerrado por
  una sola entrada.
- Implementacion 2026-05-26: T102/T103 quedan canonicas; T137/T138 declaran
  `Fusionada_con` hacia su canon. El planner fusiona criterios/tests/write-set,
  marca la duplicada como `duplicate_backlog_task` y bloquea la canonica si el
  alias ya esta visible en cola/runtime. Cierre completo pendiente hasta que la
  bateria obligatoria deje de fallar en el smoke OPES fake `run-until-assemble`.
- Rechazar `panic` no-test en fronteras publicas o neutral-core. Un smoke opt-in
  o validador de leases debe devolver error publico/issue, no tumbar daemon o
  proceso de comando.

Prioridad media:

- T140 coordina con T43/T79/T116/T135: merge lease, sharding, linter y fallback
  seguro siguen como owners vecinos; aqui el foco es deduplicar backlog ya
  existente.
- T141 coordina con T85/T97/T107/T139: status, logs, diagnostico y stdout
  siguen owners separados; aqui el rail es crash/no-panic y reason code
  recuperable.
- Cierre 2026-05-26: T141 queda cerrado para los dos `panic(` no-test del
  write-set detectados por scanner; futuros solapes con T85/T97/T107/T139 deben
  tratar salida/diagnostico como vecinos, no reabrir este rail salvo regresion
  de `panic` ejecutable.

## Priorizacion scanner 2026-05-24 cuadragesima octava pasada

Backlog asociado: `T142 cli-json-input-bounds-and-source-policy` y
`T143 codex-control-file-size-and-redaction-policy`.

Prioridad alta:

- Acotar entrada JSON de CLI antes de usarla como toolbelt de agentes. Los
  limites de HTTP no protegen `stdin` ni `--input`, y un fichero local sensible
  no debe entrar como payload de control plane por accidente.
- Unificar lectura de ficheros de control Codex. Un ACK, sidecar o checkpoint
  enorme debe producir issue compacto, no consumo de memoria, path local ni
  cuerpo crudo en diagnostico.

Prioridad media:

- T142 coordina con T103/T138/T139: respuesta HTTP, respuesta saliente y stdout
  siguen como owners vecinos; aqui el foco es input local de CLI.
- T143 coordina con T33/T36/T42/T63/T104: correlacion, terminalidad estricta,
  sidecar causal y escritura durable siguen separados; aqui el foco es lectura
  acotada y redaccion de control files.
- Estado local 2026-05-26: el owner comun de lectura acotada queda en
  `orquesta-runtime-codex` y se reutiliza desde ACK, observation source,
  checkpoint, sidecar de decisiones y startup strict con reason publico
  `control_file_too_large`; logs/progreso mantienen su tail acotado separado.

## Priorizacion scanner 2026-05-24 cuadragesima novena pasada

Backlog asociado: `T144 domain-work-delivery-artifact-intake-policy`,
`T145 operator-mcp-client-deadline-budget` y
`T146 factory-http-json-boundary-coverage`.

Prioridad alta:

- Unificar intake de artefactos `domain_work`: el stack no debe leer ficheros de
  producto completos y convertirlos en payloads de dominio sin limite, tipo,
  redaccion y reason code. T26/T72/T101/T143 no cubren por si solas el momento
  de lectura del fichero entregado.
- Dar deadline al cliente MCP de operador. El transporte real puede seguir
  opt-in, pero un `CallToolV0` sin contexto acotado es otra politica de espera
  paralela al Director y al supervisor residente.
- Meter `orquesta-factory-http` en el rail comun de JSON publico. Si queda
  fuera de T137/T133, una entrada de AppSpec puede conservar `io.ReadAll` sin
  bound y sin guia local mientras otros handlers ya migran a helper comun.

Prioridad media:

- T144 coordina con T80/T101: egress HTTP y ledger de submit siguen siendo
  owners vecinos; aqui el owner es el payload leido desde `ACK.files` antes de
  `submit_artifact`.
- Estado 2026-05-26: T144 queda cerrado en `orquesta-app-codex-stack` con owner
  de intake por artefacto, lectura limitada, errores publicos estables,
  distincion texto/JSON/binario y bloqueo de campos sensibles antes de
  `PayloadFields`. T80/T101 conservan egress HTTP y ledger; no se reabren por
  este cierre.
- T145 coordina con T23/T55/T85/T136: MCP residente, control plane, status y
  espera externa publican estado; el cliente de operador debe aportar timeout
  y cancelacion sin filtrar transporte.
- Estado local 2026-05-26: T145 queda cubierto en
  `orquesta-operator-mcp-client`; cada llamada usa contexto padre opcional,
  timeout de composicion y codigos publicos `operator_mcp_timeout` /
  `operator_mcp_cancelled`, sin reabrir transporte MCP residente ni waits.
- T146 coordina con T123/T133/T137: preview de factory, docs locales y helper
  JSON publico deben converger sin convertir factory en planner operativo.
- Estado local 2026-05-26: T146 queda cerrado en
  `orquesta-factory-http`. El handler de `POST /api/v0/apps/spec` usa decode
  acotado por `MaxBytesReader`, politica de `Content-Type`, rechazo de trailing
  data/campos desconocidos y guia local en `README.md`; gateways siguen como
  compositores sin parseo JSON propio.

## Priorizacion scanner 2026-05-24 quincuagesima pasada

Backlog asociado: `T147 codex-progress-failure-log-tail-bounds` y
`T148 file-ledger-snapshot-read-bounds`.

Prioridad alta:

- Hacer que los logs internos de progreso Codex usen la misma disciplina de
  tail acotado que las superficies publicas. Un log grande no debe cargarse
  completo para decidir `capacity_warning`, `interrupted` o `no_ack`, ni debe
  convertirse en evidencia terminal.
- Separar lectura acotada de snapshots/ledgers de la politica de escritura
  durable. T104 gobierna temp/fsync/permisos; T148 debe impedir que un ledger
  enorme o corrupto se lea entero o se trate como vacio y genere efectos
  duplicados.

Prioridad media:

- T147 coordina con T58/T106/T128/T143: tail publico, summaries, railes de
  progreso y control files siguen siendo owners separados; aqui el foco es
  failure context interno.
- Estado local 2026-05-26: T147 queda cerrado para failure context interno y
  lectores vecinos de progreso/recovery ops Codex. La lectura usa tail acotado
  antes de clasificar y conserva stdout/stderr/last-message fuera de evidencia
  terminal; T58/T106/T128/T143 siguen como owners separados.
- T148 coordina con T98/T101/T104: claim/recovery de bridge, submit ledger y
  escritura durable no se fusionan; aqui el foco es presupuesto de lectura,
  `max_records`, corrupcion recuperable y error publico redactado.
- Estado local 2026-05-26: T148 queda cerrado para el write-set asignado.
  Bridge externo, ledger `domain_work`, `domain-work-file`, `run-file`,
  `state-file/outbox` y stores Codex delivery leen snapshots con limite de
  bytes, `max_records` y errores compactos; no asumen estado vacio ante
  sobrelimite, corrupcion o schema invalido. T98/T101/T104 no se reabren.

## Priorizacion scanner 2026-05-24 quincuagesima primera pasada

Backlog asociado: `T149 worktree-snapshot-read-budget`,
`T150 project-tree-scan-budget-for-review-recovery` y
`T151 codex-wave-operator-file-input-bounds`.

Prioridad alta:

- Dar propietario al presupuesto de snapshot de worktree antes de usarlo como
  prueba terminal de write-set, line budget o efectos destructivos. T42/T45/T48
  necesitan snapshot fiable; T149 debe impedir que la evidencia cargue ficheros
  completos o incluya artefactos locales/control files.
- Unificar los scans de arbol usados por review/rework y recovery. Hoy
  `orquesta-runtime-codex-delivery` y `orquesta-app-codex-stack` tienen helpers
  parecidos con ignore lists propias; T150 debe evitar que globs amplios
  bloqueen o lean medio proyecto como heuristica de existencia.
  Revalidado el 2026-05-26: esos helpers delegan en
  `ProjectTreeScanHasFileV0`/`ProjectTreeScanFilesV0`, con presupuesto,
  `context`, ignore prefixes comunes y reason codes publicos.
- Acotar ficheros de operador usados como prompt/objetivo/contexto de olas
  Codex. T151 separa esos inputs locales de T56 (`CODEX_HOME`) y de T143
  (control files), con limite y redaccion antes de construir paquetes.
  Cerrado localmente el 2026-05-26: `cmd/orquesta-server` concentra la politica
  de lectura de `--prompt-file`, `--objective-file` y `--domain-context-file`
  en un helper unico, con limite por fichero, validacion UTF-8/texto, bloqueo de
  control files/logs/symlinks/no regulares y summaries `operator_inputs` con
  categoria/hash/ref compactos.

Prioridad media:

- Estado local 2026-05-26: T149 queda cerrado para snapshot de worktree
  acotado. El presupuesto vive en `orquesta-runtime-worktree`, el stack Codex y
  `cmd/orquesta-server` solo inyectan defaults/configuracion de composicion; los
  issues de presupuesto se publican como rails compactos.
- T149 coordina con T50/T127/T143: control files, artefactos ignorados y ACKs
  siguen como owners separados; snapshot solo debe emitir hashes, tamanos,
  paths relativos seguros y issues publicos.
- T150 coordina con T144/T149: no decide payload final de artefactos ni snapshot
  completo; solo existencia/recovery por globs/carpetas con presupuesto.
- T151 coordina con T106/T147: salida publica y logs de progreso no deben
  depender de prompt/contexto crudo; los summaries deben exponer categorias y
  refs compactas.

## Regla

Si una lista bloquea flujo operativo, debe tener:

- un unico fichero propietario por modulo;
- helper comun por modulo, no copias por comando;
- matriz externa de falsos positivos antes de integrarse en Orquesta completa;
- tolerancia por defecto a refs opacas, alias razonables y vocabulario de
  runtime/adaptadores.

## Priorizacion scanner 2026-05-24 quincuagesima segunda pasada

Backlog asociado: `T152 codex-code-home-copy-bounds-symlink-policy`,
`T153 workflow-command-event-payload-budget` y
`T154 effect-port-deadline-context-policy`.

Prioridad alta:

- Separar T56 de su shard mecanico: cerrado 2026-05-26 para
  `codex-wave`/`codex-director-wave`. La politica de categorias de
  `CODEX_HOME` ahora se combina con max de bytes/ficheros, tratamiento seguro de
  symlinks/modos y receipt compacto en la composicion Codex, sin moverlo al
  nucleo.
- Alinear comandos/eventos con el outbox: cerrado 2026-05-26 para
  `T153 workflow-command-event-payload-budget` en core/state-file. El workflow
  usa presupuesto comun alineado con outbox, overrides tipados de microtareas y
  validacion antes de decode/compactacion/persistencia.
- T154 queda cerrado en el write-set asignado el 2026-05-26: HTTP domain_work,
  OPES REST, runtime process launch/stop y bridge OPES residente tienen deadline
  por efecto y errores publicos compactos. MCP operador ya estaba cubierto por
  T145; shutdown avanzado y espera externa siguen como owners vecinos.
- Reintento `agent-ref-task-autoprogramming-c22c7438ddc1-g01`: el cierre queda
  reflejado tambien en el estado vivo del backlog para que el planner no reprograme
  T154 por una cabecera stale.

Prioridad media:

- La copia de `CODEX_HOME` debe coordinarse con T50/T127 para que credenciales y
  control files no entren en snapshot, AppVCS, promocion ni ACK.files.
- El presupuesto de payload del workflow conserva compatibilidad con eventos
  compactos historicos y mantiene cuerpos grandes por refs de artefacto; no
  mover cuerpos HTTP, prompts ni transcripts a eventos por comodidad.
- La politica de deadline no sustituye T136: una espera externa puede ser
  valida si tiene presupuesto y wait scope; un conector colgado debe producir
  timeout publico y no consumo silencioso de ticks.

## Priorizacion scanner 2026-05-24 quincuagesima tercera pasada

Backlog asociado: `T155 idle-backlog-runtime-ack-scan-budget` y
`T156 app-vcs-git-output-and-path-budget`.

Prioridad alta:

- Acotar el scan de ACKs antes de ampliar automejora idle. T33 decide
  correlacion de ACK y T143 tamano/redaccion de control files, pero T155 debe
  impedir que el planner camine todo `.orquesta-runtime` y confunda presupuesto
  agotado con backlog vacio o completado.
- Acotar salida Git antes de promocion o AppVCS productivo. T39 exige
  write-set/evidencia y T139 shape publico, pero T156 debe poner limite de
  stdout/stderr y numero de rutas en `git status` para no procesar repos enormes
  ni devolver errores sensibles.

Prioridad media:

- T155 debe coordinarse con T79 y T135: sharding/fallback no sustituyen el scan
  acotado de ACKs completados; si el indice falta o el scan se degrada, la
  salida debe ser bloqueo/scanner documental visible.
- Estado 2026-05-26: T155 cerrado para el planner idle del servidor. El scan de
  ACKs queda acotado a runs backlog y profundidad `run/agent/agent_ack.json`,
  con presupuesto de directorios, entradas, agentes, bytes y duracion; agotado o
  ambiguo conserva scanner documental visible.
- Revalidacion OrquestaV2 2026-05-26: el rework
  `agent-ref-task-ref-review-rework-task-autoprogramming-a8f8edf38716-g01-4dce1eb3cf44`
  conserva T155 cerrado, no abre rails nuevos y resuelve `required ref_only`
  mediante lectura local/evidencia explicita en ACK.
- T156 debe coordinarse con T105 y T154: deadline e IO neutral son vecinos, pero
  los comandos Git de AppVCS necesitan codigos propios de truncation, timeout y
  exceso de paths.

Estado 2026-05-26: T156 queda cerrado para AppVCS y promocion de staging con
captura stdout/stderr presupuestada, `max_changed_paths`, errores publicos
`git_output_too_large`, `git_status_too_many_paths` y `git_command_timeout`, y
evidencia Git compacta sin salida cruda ni datos sensibles.
Revalidacion OrquestaV2 2026-05-26: el rework
`agent-ref-task-ref-review-rework-task-autoprogramming-a61a87a140a8-g01-14fd8c3348e6`
mantiene T156 cerrado, no abre rails nuevos y resuelve `required ref_only`
mediante lectura local/evidencia explicita en ACK.

## Priorizacion scanner 2026-05-24 quincuagesima cuarta pasada

Backlog asociado: `T157 codex-control-file-root-and-symlink-policy` y
`T158 opes-bridge-destination-and-summary-policy`.

Prioridad alta:

- Unificar raiz/tipo de ficheros de control Codex antes de hacer mas estricto el
  ACK terminal. T143 limita bytes y redaccion, pero T157 debe impedir que un
  descriptor corrupto, symlink o fichero no regular entre como ACK, sidecar,
  checkpoint o tail de progreso.
- Separar OPES del HTTP neutral en politica de destino. T80 cubre
  `domain_work` HTTP generico; T158 debe gobernar `ORQUESTA_OPES_BASE_URL`,
  confirmacion temporal/productiva y summary publico del bridge OPES.

Prioridad media:

- T157 debe coordinarse con T50, T63 y T147: control files excluidos del
  producto, sidecar correlado y tails de progreso siguen owners separados; aqui
  el foco es path/raiz/symlink antes de leer.
- Estado 2026-05-26: T157 queda cerrado para el write-set asignado. El helper
  comun de `orquesta-runtime-codex` valida nombre base, raiz no trivial, `Lstat`,
  symlink/hardlink/no regular y cambio entre `Lstat`/`open` antes de leer ACK,
  checkpoint o sidecar; delivery/progreso y `codex-wave-tail` rechazan tails
  symlink/hardlink/no regulares antes de ingerir diagnostico.
- Revalidacion OrquestaV2 2026-05-26: el rework
  `agent-ref-task-ref-review-rework-task-autoprogramming-0149f7cf3c20-g01-935bd05e71ff`
  mantiene T157 cerrado, no abre rails nuevos y resuelve `required ref_only`
  mediante lectura local/evidencia explicita en ACK.
- T158 debe coordinarse con T12, T21, T98, T103 y T139: smoke OPES, guardas
  reales, claim de entrada, respuesta HTTP y salida de comandos siguen como
  owners vecinos; aqui el foco es destino OPES y redaccion de summary.

Estado 2026-05-26: T158 queda cerrado para el bridge OPES. La configuracion
`ORQUESTA_OPES_BASE_URL`/`OPES_BASE_URL` y `ORQUESTA_BASE_URL` pasa por politica
de destino con esquema permitido, sin credenciales/query, confirmacion
loopback/temporal/productiva y evidence ref compacta para productivo; el summary
publico de `opes-drain-once` expone refs opacas, categorias, filtro aplicado,
estado y contadores sin URL completa ni payload crudo. Revalidacion OrquestaV2
`agent-ref-task-autoprogramming-453f91b133d4-g01`: no abre owner vecino y
resuelve `required ref_only` mediante lectura local/evidencia explicita en ACK.
Retry OrquestaV2 2026-05-26
`agent-ref-task-autoprogramming-985c0ad5e009-g01`: revalida T158 sin duplicar
owner vecino y conserva `required ref_only` resuelto por evidencia explicita en
ACK.
Retry OrquestaV2 2026-05-26
`agent-ref-task-autoprogramming-bf0d81417acc-g01`: revalida T158 sin duplicar
owner vecino ni cambiar codigo; conserva `required ref_only` resuelto por
lectura local y evidencia explicita en ACK.
Retry OrquestaV2 2026-05-26
`agent-ref-task-autoprogramming-761a129dc1b4-g01`: revalida T158 sin duplicar
owner vecino ni relanzar smoke real; conserva `required ref_only` resuelto por
lectura local/evidencia explicita en ACK.

## Priorizacion scanner 2026-05-24 quincuagesima quinta pasada

Backlog asociado: `T159 codex-control-file-durable-write-policy` y
`T160 director-decisions-batch-budget-and-source-limit`.

Prioridad alta:

- Unificar escritura durable de ficheros de control Codex. La lectura ya tiene
  tareas de tamano/raiz/symlink, pero packet, prompt, wrappers, registry y
  requests de checkpoint se escriben desde owners distintos con `os.WriteFile`
  sobre ruta final. La politica debe ser atomica, con permisos cerrados,
  receipt compacto y sin path local ni contenido crudo.
  Estado 2026-05-26: cerrado local para runtime Codex y comandos de ola con
  `WriteCodexControlFileBytesV0`; si reaparece debe tratarse como regresion de
  owner concreto o frontera no migrada.
- Acotar fan-in de `director_decisions.json`. Un limite de bytes por fichero no
  basta si varias fuentes/descriptors aportan muchas decisiones o microtareas
  antes de que `MaxCommands`, outbox o waits entren en juego.

Prioridad media:

- La escritura de control files debe coordinarse con T30/T50/T63/T104/T143/T157:
  checkpoint, exclusion del producto, sidecar causal, escritura durable
  generica, lectura acotada y raiz/symlink son rails vecinos, no sustitutos.
- El presupuesto de decisiones debe coordinarse con T31/T57/T63/T68/T153:
  leases, outbox durable, sidecar causal, presupuestos anidados y payload de
  workflow conservan owners propios. Aqui el foco es rechazar batches excesivos
  antes de materializar efectos.
- Ambos frentes deben conservar tolerancia a refs opacas y vocabulario operativo
  normal; los rechazos son por frontera de IO, tamano, cardinalidad,
  correlacion o seguridad, no por substrings del contenido.

Cierre parcial 2026-05-26:

- `T160` queda cubierto para `director-agent-file-source`, `director-agent-workflow`
  y `orquesta-app-codex-stack`: presupuesto compartido, reason publico
  `director_decisions_batch_too_large` y contadores compactos. `T159` sigue como
  owner separado de escritura durable de control files.

## Priorizacion scanner 2026-05-24 quincuagesima sexta pasada

Backlog asociado: `T161 clock-and-ref-generation-policy` y
`T162 public-client-mutation-idempotency-policy`.

Prioridad alta:

- Cerrado T161 2026-05-26: reloj y generador de refs quedan unificados para el
  alcance focal antes de ampliar olas concurrentes,
  reinicio/resume o automejora idle. Los timestamps pueden seguir como evidencia
  temporal, pero no deben ser identidad causal unica ni producir refs
  colisionables bajo rafagas del mismo segundo.
- Alinear identidad/idempotencia de mutaciones publicas antes de exponer mas
  control plane por web/CLI/MCP. Un retry de prepare-run, run-control,
  queue-priority, AppVCS o shutdown debe poder reconciliarse por key estable y
  correlation, no crear efectos duplicados.

Prioridad media:

- El clock/ref generator debe coordinarse con leases, outbox, sidecars y
  progreso, pero no sustituye sus receipts ni eventos causales. Cada frontera
  conserva owner y prueba propia.
- La politica de idempotencia publica debe reutilizar limites JSON, deadlines y
  redaccion existentes; no debe meter tokens, URLs con credenciales, HOME,
  rutas privadas ni payloads completos en headers, auditoria o errores.

Cierre 2026-05-26:

- `T162` queda cerrado para el stack publico acotado: helpers MCP canonicos de
  identidad de mutacion, headers `X-Correlation-ID`/`Idempotency-Key`,
  derivacion estable desde `request_id`, clasificacion HTTP de rutas mutables y
  consumo desde web, CLI, MCP, app-gateway, http-gateway y shutdown CLI del
  servidor.
- Evidencia:
  `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./modulos/orquesta-mcp ./cmd/orquesta-server ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway`.

## Priorizacion scanner 2026-05-24 quincuagesima septima pasada

Backlog asociado: `T163 state-file-event-log-index-compaction-budget`,
`T164 workflow-task-store-parent-index-budget` y
`T165 server-config-global-env-defaults-policy`.

Prioridad alta:

- Separar crecimiento del log de eventos de presupuesto de payload. T153 evita
  payloads grandes y T148 limita lecturas de snapshots, pero T163 debe gobernar
  indice, paginacion, compaction y coste de append/replay para runs residentes
  largas.
- Indexar parentesco de `WorkflowTaskV0` antes de cerrar recursion Codex real.
  T160 limita batches que crean tasks; T164 debe impedir que el cierre por
  arbol dependa de escanear todas las tasks del run o de interpretar un indice
  faltante como ausencia de hijos.
- Dejar de mutar entorno global en lectura de config. T129 decide que entorno
  se proyecta al daemon; T165 decide como se calculan defaults sin contaminar
  tests, smokes ni procesos hijos.

Prioridad media:

- T163 debe coordinarse con escritura durable y snapshots sin duplicar T104 ni
  T148: aqui el owner es log de eventos por run, no todos los ledgers.
  Cierre 2026-05-26: `orquesta-state-file` agrega indice durable por run,
  registros de evento, budgets y lectura paginada; T104/T148/T153 conservan sus
  fronteras vecinas.
- T164 debe conservar metadata completa del `WorkflowTaskStore`; no reconstruir
  parent/child, ola o cohorte desde eventos compactos si el indice puede
  recuperarse o rematerializarse con presupuesto.
  Cierre 2026-05-26: `orquesta-state-file` conserva esa metadata en un indice
  durable por run, lee hijos por refs indexadas y rematerializa el indice solo
  con presupuesto. Si el indice falta/corrupto y el rebuild excede limites,
  devuelve `workflow_task_parent_index_budget_exhausted` para bloquear
  recursion/cierre en vez de asumir que no hay hijos.
- T165 cerrado 2026-05-26: el default operativo queda separado de la
  autorizacion explicita. Los defaults de rails y cleanup se leen como config
  efectiva y se proyectan al daemon como `defaulted`; el input del operador
  queda marcado como `explicit` y los valores derivados como `derived`.

## Priorizacion scanner 2026-05-24 quincuagesima octava pasada

Backlog asociado: `T166 resident-runtime-async-shutdown-quiescence`,
`T167 external-bridge-resident-loop-lifecycle-state`,
`T168 state-file-run-event-load-validation` y
`T169 mcp-real-transport-registration-collision-guard`.

Prioridad alta:

- Cerrar la quiescencia async del runtime residente antes de ampliar
  automejora idle o bridges residentes. T30/T154 cubren checkpoints/deadlines,
  pero T166 debe demostrar que no se publica `stopped` mientras siguen vivos
  ticks, preparaciones idle o escrituras de estado/auditoria.
- Hacer observable el loop de bridge externo. T98/T158 gobiernan claim/destino
  OPES, pero T167 debe evitar que un bridge residente quede como goroutine
  paralela solo visible por stderr.
- Validar run/eventos al cargar desde `orquesta-state-file`. T163 gobierna
  crecimiento/indice del log, pero T168 debe impedir que un snapshot corrupto
  con schema/ref correctos alimente replay, cierre o supervision.

Prioridad media:

- Las colisiones de registro MCP deben coordinarse con T93: toolbelt/prompts no
  pueden estar frescos si el transporte real permite sobrescribir un tool o
  resource sin error. T169 es mas estrecha que la sincronizacion global del
  catalogo.
- Estado 2026-05-26: T169 cerrado para el transporte MCP real. El registro
  rechaza resources duplicados por `name`/`uri`, tools duplicados por `name`,
  no deja mutacion parcial en el item rechazado y expone solo errores compactos
  `mcp_duplicate_resource`/`mcp_duplicate_tool`; T93 sigue como sincronizacion
  global de catalogo/toolbelt si aparece evidencia propia.
- T166 y T167 deben compartir taxonomia de lifecycle (`running`, `stopping`,
  `stopped`, `timeout`, `degraded`) con T87/T134, pero no deben meter OPES,
  Codex ni proveedor en `orquesta-server`.
- Estado 2026-05-26: T166 cerrado para el runtime residente. El owner de
  quiescencia async queda en `modulos/orquesta-server`: `Serve`, supervisor
  loop, ticks y preparaciones idle se registran y se esperan con deadline de
  composicion antes de publicar `stopped`; `stop_timeout` conserva contador
  compacto.
- Estado 2026-05-26: T167 cerrado para el loop OPES/bridge externo residente.
  La composicion publica lifecycle compacto en status/readiness, audita errores
  de tick con codigos redactados y coordina shutdown con join; OPES sigue fuera
  de `orquesta-server` salvo el wiring opt-in de `cmd/orquesta-server`.
- T168 debe conservar compatibilidad de snapshots validos y no convertirse en
  migrador silencioso. Reparacion o migracion de estado debe entrar por puerto
  opt-in con evidencia durable.
- Estado 2026-05-26: T168 cerrado para carga state-file. `LoadRunV0` valida la
  proyeccion de run con `ValidateOrchestrationRunV0`; `LoadRunEventsV0` valida
  eventos, refs y duplicados antes de entregar historial. Los documentos
  corruptos quedan bloqueados con error publico compacto y no se reparan ni se
  migran en silencio.

## Priorizacion scanner 2026-05-24 quincuagesima novena pasada

Backlog asociado: `T170 outbound-http-redirect-policy`,
`T171 public-query-form-parameter-bounds-redaction` y
`T172 http-response-encode-write-error-visibility`.

Prioridad alta:

- Definir politica de redirects antes de ampliar conectores externos o bridges
  residentes. T80/T103/T158 validan destino y egress iniciales, pero T170 debe
  demostrar que cada salto 3xx conserva la misma frontera y que no se reenvian
  credenciales/correlation refs a otro origen por defecto.
- Separar URL/form de body JSON. T137 gobierna cuerpos JSON, pero T171 debe
  evitar que `raw_query`, filtros libres o formularios publicos entren completos
  en auditoria, status o comandos por rutas no cubiertas.
- Hacer visible el fallo de entrega HTTP. T94/T139 gobiernan auditoria/output
  publico, pero T172 debe impedir que una operacion interna exitosa quede
  registrada como entregada cuando la serializacion o escritura al cliente fallo.

Prioridad media:

- T170 debe reutilizar allowlists y reason codes de destino; no debe meter OPES,
  Codex, proveedor, tokens ni URLs productivas en el nucleo.
- T171 debe normalizar alias razonables en adaptadores cuando sea seguro, pero
  mantener cortes fuertes por datos sensibles, refs imposibles, causalidad rota
  o efectos externos no autorizados.
- Cierre T171 2026-05-26: la web concentra limites de query/form publicos antes
  de `URL.Query`/`ParseForm`; la auditoria HTTP residente guarda solo claves
  acotadas/redactadas y reason codes, nunca `raw_query` completo.
- Revalidacion OrquestaV2 2026-05-26
  `agent-ref-task-ref-review-rework-task-autoprogramming-03f6b0927ba2-g01-24a1e410cd44`:
  no reabre T171 ni owners vecinos; el contexto `ref_only` requerido queda
  resuelto por lectura local de paquete/docs y evidencia explicita en ACK.
- T172 debe registrar errores compactos despues de headers emitidos sin intentar
  reescribir status tarde; la evidencia debe servir para smokes y shutdown
  quiescent sin almacenar payloads internos.

Estado 2026-05-26: T172 queda cerrado para el residente HTTP del write-set. El
helper local serializa antes de headers cuando puede, emite
`response_encode_failed` estable si falla la serializacion, y la auditoria HTTP
registra `response_write_failed` con etapa compacta cuando la escritura al
cliente falla antes o despues del header observable.

Estado 2026-05-26: T170 queda cerrado para el write-set asignado. Los clientes
salientes de `orquesta-domain-work-http`, `orquesta-opes-connector`,
`orquesta-web`, `orquesta-mcp` y los comandos/bridge residentes del servidor
aplican politica de redirect por defecto: cada salto revalida esquema, host,
puerto, ausencia de credenciales/fragmento y frontera de path cuando el
conector la declara. Los saltos a otro origen se bloquean con reason code
publico compacto y no reenvian cabeceras de correlacion, idempotencia o
autorizacion fuera del origen inicial.
Revalidacion OrquestaV2 2026-05-26:
`agent-ref-task-ref-review-rework-task-autoprogramming-74734b6a029e-g01-9160302b840b`
mantiene T170 cerrado, no relanza otro agente padre sobre la tarea original y
resuelve `required ref_only` por evidencia explicita en ACK.

## Priorizacion scanner 2026-05-24 sexagesima pasada

Backlog asociado: `T173 browser-origin-csrf-intent-guard` y
`T174 mcp-jsonrpc-protocol-strictness`.

Prioridad alta:

- Separar origen/intencion de auth remota. T55 decide bind, principal y permiso,
  pero las mutaciones web/gateway necesitan owner propio para `Origin`,
  `Referer`, CSRF/intent token y decision compacta antes de exponer mas rutas a
  navegador local o bind opt-in.
  Cierre 2026-05-26: owner en `orquesta-http-gateway` +
  `orquesta-app-gateway`, con headers compactos de decision y bloqueo
  same-origin/token para browser/form.
- Fijar el contrato JSON-RPC de `/mcp` antes de usar el transporte residente
  como superficie publica estable. T23 y T169 no bastan si el handler acepta
  formas ambiguas de `jsonrpc`, `id`, batch, notification o params.
  Cierre 2026-05-26: `/mcp` valida JSON-RPC 2.0 estricto, `id` seguro,
  presupuesto de params, `Accept`/`Content-Type`, batch rechazado y
  notification acotada a `notifications/initialized`; `resources/read` y
  `tools/call` validan params estrictos con errores compactos.

Prioridad media:

- T173 debe coordinarse con T102/T137/T162: body/form/query e idempotencia no
  sustituyen la prueba de intencion del navegador. GET/status siguen read-only.
- T174 debe coordinarse con T102/T137/T172: el parser JSON-RPC puede compartir
  helpers de limite y respuesta, pero los codigos y semantica MCP deben tener
  matriz propia.
- Ambos rails deben registrar solo reason codes, refs y contadores. No guardar
  cookies, URLs completas, query cruda, argumentos MCP, resource payloads,
  prompts, transcripts, HOME, rutas privadas ni tokens.

## Priorizacion scanner 2026-05-24 sexagesima primera pasada

Backlog asociado: `T175 mcp-tool-resource-output-budget`,
`T176 control-plane-http-security-cache-headers` y
`T177 rest-client-base-url-endpoint-policy`.

Prioridad alta:

- T175 queda cerrado 2026-05-26 para salida MCP: cada resource/tool declara
  `output_budget` comun y el transporte real bloquea payloads sobre presupuesto
  con error compacto antes de envolverlos como texto JSON-RPC.
- T176 queda cerrado 2026-05-26 para headers/cache del control plane:
  `orquesta-http-gateway` aplica perfiles HTML/JSON/MCP con anti-frame,
  `nosniff`, `Referrer-Policy`, cache explicita `no-store` y reason code
  publico, sin copiar datos privados de request.
- T177 queda cerrado 2026-05-26 para clientes REST de web, CLI, MCP y
  composicion: base URL parseada, base path preservado, endpoints relativos
  controlados y rechazo de destinos ambiguos antes de egress, redirects o
  lectura de respuesta.

Prioridad media:

- T175 debe reutilizar catalogo/freshness de recursos MCP cuando exista, sin
  duplicar descriptores estaticos de T25/T93.
- T176 no sustituye T99/T137/T171/T172: los limites de body, query/form y
  visibilidad de fallo de escritura siguen con sus owners.
- T177 distingue transporte interno in-process de egress real. El alias
  `http://orquesta.internal` queda vivo solo como destino interno probado por
  `orquesta-app-gateway`, no como URL externa.

## Priorizacion scanner 2026-05-24 sexagesima segunda pasada

Backlog asociado: `T178 outbound-http-client-transport-proxy-policy` y
`T179 inprocess-http-transport-budget-parity`.

Prioridad alta:

- Fijar politica de transporte/proxy para clientes HTTP salientes antes de
  exponer mas control plane, OPES temporal o conectores de dominio. T177 evita
  URLs ambiguas y T170 evita redirects peligrosos, pero sin T178 el proxy del
  entorno y el transporte por defecto siguen siendo una decision implicita por
  proceso.
- Acotar transportes in-process del gateway y smokes MCP. Hoy sirven para tests
  y rutas internas sin abrir sockets, pero si acumulan respuestas sin limite o
  no publican cancelacion/timeout pueden ocultar fallos que el servidor HTTP real
  si deberia observar.

Prioridad media:

- T178 debe coordinarse con T80/T103/T154/T170/T177: no sustituye egress,
  respuesta, deadline, redirects ni join de URL; gobierna solo transporte,
  proxy, TLS/keepalive y perfil de cliente.
- Estado 2026-05-26: T178 queda cerrado localmente para clientes HTTP salientes
  del write-set. Los perfiles `domain_egress`, `opes_temporal`,
  `loopback_control_plane` e `internal_inprocess` declaran proxy deny,
  dial/TLS/headers, pool y keepalive cuando hay red real; no habilitan proxy/TLS
  custom ni cambian los owners vecinos de egress, redirects, URL final,
  respuesta o idempotencia.
- T179 debe coordinarse con T102/T137/T141/T172/T176: el in-process debe
  compartir limites y errores publicos sin duplicar cada helper HTTP ni guardar
  payloads, prompts, transcripts, HOME, tokens o rutas privadas.
- Estado 2026-05-26: T179 queda cerrado localmente para el gateway, helpers web
  de test y smoke MCP in-process del servidor mediante transporte comun
  `inprocesshttp`, con limite de respuesta, cancelacion/timeout observable,
  recover de panic y errores publicos compactos.

## Priorizacion scanner 2026-05-24 sexagesima tercera pasada

Backlog asociado: `T180 command-stdio-write-error-visibility`.

Estado 2026-05-26: cerrado local para visibilidad base en comandos stdio. El
contrato compacto vive en `orquesta-observability`; `cmd/orquesta-server` y
`orquesta-cli` comprueban escritura de salida publica y devuelven fallo
observable con reason code redactado si stdout se rompe.

Prioridad media-alta:

- Hacer visible el fallo de escritura de stdout/stderr en comandos locales antes
  de que el residente o un operador consuma una salida parcial como exito. T139
  gobierna el shape publico y T172 gobierna HTTP; T180 debe cubrir stdio sin
  crear otro formato de cierre terminal.

Prioridad media:

- Coordinar comandos de servidor y CLI con observabilidad: un fallo de stdout
  posterior al efecto debe quedar como reason code compacto, no como reejecucion
  del efecto ni como dump de payloads, rutas privadas, HOME, prompts,
  transcripts, tokens o salidas crudas.

## Priorizacion scanner 2026-05-24 sexagesima cuarta pasada

Backlog asociado:
`T181 domain-document-plan-ref-uniqueness-and-diagnostics` cerrado 2026-05-26,
`T182 mcp-tool-execution-budget-and-cancellation` y
`T183 web-html-render-error-contract`.

Prioridad alta:

- T181 ya fija unicidad/diagnostico del plan documental neutral: rechaza refs
  duplicados por tipo, diagnostica arrays raw invalidos por campo y bloquea
  identidades derivadas duplicadas antes de expansion.
- Resolver T182 antes de tratar MCP real como transporte residente de
  autoprogramacion: T174/T175 cubren request y salida, pero sin presupuesto de
  ejecucion un tool lento puede agotar el worker sin reason code estable.

Prioridad media-alta:

- T183 cerrado 2026-05-27 junto a fronteras web T172/T176: template roto y
  writer tardio quedan como reason codes compactos (`web_html_render_failed`,
  `web_response_write_failed`) y observabilidad no guarda query, formularios,
  payloads, HOME, rutas privadas, prompts, transcripts, cookies ni tokens.

Duplicaciones a evitar:

- T181 no sustituye validadores editoriales OPES ni semantica de `DomainWork`;
  solo fija unicidad/diagnostico del plan documental neutral y la idempotencia
  previa a expansion.
- T182 no reabre deadlines internos de puertos (T154), parsing MCP (T174),
  output budget (T175) ni paridad in-process (T179); gobierna solo presupuesto
  de ejecucion del handler MCP real. Cerrado 2026-05-26 con `execution_budget`
  por perfil en el registro MCP, timeout/cancel publico en transporte real y
  observacion compacta sin argumentos ni payloads.
- T183 no duplica T172/T176/T173/T75: usa sus helpers cuando existan, pero el
  owner cerrado es el contrato de render HTML, fallback localizado y contador
  compacto de render/write.

## Priorizacion scanner 2026-05-24 sexagesima quinta pasada

Backlog asociado:
`T184 deterministic-ref-hash-collision-proof` y
`T185 smoke-script-temp-root-deletion-guard`.

Prioridad alta:

- Resolver T185 antes de ampliar smokes opt-in o ejecuciones reales con raiz
  inyectada por entorno. T59/T91/T121/T126/T127 acotan shutdown y semantica del
  smoke, pero no prueban que un cleanup recursivo solo borre directorios
  generados y marcados por el propio script. Cerrado el 2026-05-26:
  `scripts/lib/smoke_common.sh` centraliza la guarda y los smokes/runners ya
  limpian raices temporales a traves del helper.
- Resolver T184 antes de usar refs derivadas como contrato de recuperacion o
  cierre causal. T161/T162/T169/T181 cubren generacion, unicidad de tasks y
  refs documentales, pero no separan digest visual/advisory de identidad causal
  ni fuerzan prueba de colision. Cerrado 2026-05-27: los owners asignados usan
  builders canonicos con longitud de campos y `sha256`; el retry de
  autoprogramacion deja de unir `request_ref` y timestamp con separador textual
  y el core suma prueba focal de namespace/payload no ambiguo.
  Rework de revision 2026-05-27:
  `agent-ref-task-ref-review-rework-task-autoprogramming-298dd18fed1e-g01-c005a1d6953bff0a3fc8364f7d4ac61b`
  conserva el cierre sin relanzar otro agente padre; el contexto
  `required ref_only` queda resuelto por lectura local del paquete y evidencia
  explicita en ACK.

Prioridad media:

- T184 debe empezar por inventario de usos de FNV32, SHA1 truncado y
  `strings.Join` en derivaciones deterministas. Si una huella solo es
  diagnostica, basta documentar esa frontera; si decide idempotencia o
  causalidad, debe migrar a builder canonico y conflicto reparable. La
  revalidacion 2026-05-27 conserva `strings.Join` solo en texto diagnosticado,
  comandos normalizados o listas ordenadas que no son identidad causal directa.
- T185 queda centralizado en `scripts/lib`: los scripts mantienen su contrato
  funcional y el borrado de raiz queda detras de un unico helper probado.
  Rework de revision 2026-05-26:
  `agent-ref-task-ref-review-rework-task-autoprogramming-66353e1f46d9-g01-33cae0148aa014c3f796cbabf72f5114`
  conserva ese cierre sin relanzar otro agente padre; el contexto
  `required ref_only` queda resuelto por lectura local del paquete y evidencia
  explicita en ACK.

Duplicaciones a evitar:

- T184 no reemplaza el generador de refs ni la persistencia de
  `WorkflowTaskStore`; solo cierra la ambiguedad/colision de huellas
  deterministas usadas para refs o firmas.
- T185 no reabre la politica de instancia temporal ni el shutdown de smokes;
  cubre exclusivamente limpieza local y rutas prohibidas. La implementacion
  publica solo refs compactas de raiz temporal y reason codes, sin guardar HOME,
  tokens, prompts, payloads ni rutas privadas en evidencia durable.

## Priorizacion scanner 2026-05-24 sexagesima sexta pasada

Backlog asociado:
`T186 domain-work-job-ref-fingerprint-collision-proof`,
`T187 http-audit-client-identity-redaction-policy` y
`T188 http-method-allow-options-contract`.

Prioridad alta:

- Resolver T186 antes de usar `domain_work` file/SQL como cola durable
  productiva de varios conectores. T181 protege refs del plan documental y T184
  hashes de otros paquetes, pero la identidad de job vive en los adaptadores de
  `domain_work` y debe ser equivalente entre memory, file y SQL.
- Cierre T186 2026-05-26: `BuildDomainWorkJobIdentityV0` concentra
  fingerprint/base de `job_ref` con `sha256`; memory/file/SQL preservan replay
  legacy por request guardado y reparan colisiones visibles con sufijo
  explicito. Evidencia focal:
  `go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-domain-work-memory ./modulos/orquesta-domain-work-file ./modulos/orquesta-domain-work-sql`.
- Resolver T187 antes de exponer visor o export de auditoria. T171 evita query
  cruda y T132 clasifica privacidad, pero `remote_addr` y headers de proxy son
  identidad operativa y no deben quedar como payload estable sin perfil.

Prioridad media:

- T188 queda cerrado desde el 2026-05-26: metodo, `Allow` y `OPTIONS` tienen
  contrato comun por perfil en MCP HTTP, web, governance, factory y transporte
  MCP real. Esto no sustituye CSRF ni headers de seguridad.
- Revalidacion T188 2026-05-27: bateria focal de MCP, web, governance, factory
  HTTP y `cmd/orquesta-server` pasada sin ampliar alcance.
- T186 debe coordinarse con T20 y T101: cambiar refs de job sin compatibilidad
  rompe replay, ledgers de submit y smokes OPES/no-OPES.
- T187 debe coordinarse con T55: audit puede clasificar origen, pero la decision
  de bind/autorizacion sigue en control plane, no en cabeceras declaradas por el
  cliente.
- Estado 2026-05-26: T187 queda cerrado para `orquesta-server`; auditoria HTTP
  publica `client_identity` con categoria, redaccion `category_only`, scope de
  bind/autorizacion y solo nombres de headers de forwarding presentes bajo
  politica `ignored_untrusted`, sin valores ni IP:puerto durables.
- Revalidacion 2026-05-27: mantener T187 como frontera de redaccion de
  auditoria; no reabrir autenticacion, geolocalizacion, tracking ni confianza
  en headers declarados por el cliente.

Duplicaciones a evitar:

- T186 no reabre todos los hashes deterministas del repo; cubre solo identidad
  de jobs `domain_work` y sus stores memory/file/SQL.
- T187 no crea autenticacion, geolocalizacion ni tracking de clientes; solo
  redaccion/clasificacion de identidad en auditoria HTTP.
- T188 no decide origen/intencion ni CORS amplio; su cierre cubre solo contrato
  de metodos, `Allow`, `OPTIONS` y error publico por perfil.

## Priorizacion scanner 2026-05-24 sexagesima septima pasada

Backlog asociado:
`T189 server-resident-signal-shutdown-policy`,
`T190 http-gateway-route-manifest-collision-guard` y
`T191 process-runtime-stop-signal-escalation-policy`.

Prioridad alta:

- T189 queda resuelto el 2026-05-26 para el servidor residente: politica por
  plataforma, primera senal cooperativa, segunda senal `signal_escalated`,
  timeout observable y status/auditoria compacta. T96 sigue siendo identidad de
  proceso y T166 quiescencia interna.
- Rework de revision 2026-05-26:
  `agent-ref-task-ref-review-rework-task-autoprogramming-13b40a6e5fc4-g01-a9146f390bba643a490794bd55481129`
  mantiene T189 cerrado y corrige solo la clasificacion del backlog, sin
  relanzar agente padre ni reabrir T30/T96/T166.
- T190 queda cubierto 2026-05-26 antes de sumar mas endpoints bajo
  `/api/v0/apps/` o overlays de gateway: hay manifiesto canonico, validator de
  colisiones/shadows y tests de dispatch AppVCS. T169 protege el registro MCP y
  T188 metodo/OPTIONS; esta entrada solo debe reabrirse por regresion de
  precedencia HTTP.

Prioridad media-alta:

- T191 queda cerrado el 2026-05-26: el E2E neutral de stop ya tiene politica de
  senal/escalado real para procesos que no cooperan, con `stopping`,
  `stop_grace_deadline`, `stop_reason_code` y sin filtrar argv/env/stdout/stderr.
- Rework de revision 2026-05-26: se conserva T191 cerrado y solo se sincroniza
  la clasificacion documental; no abre rail adicional ni modifica launch env/io.

Duplicaciones a evitar:

- T189 no reabre checkpoint de shutdown ni stop de agentes; gobierna senales del
  servidor residente foreground/daemon.
- T190 no sustituye catalogos publicos, docs de rutas ni CSRF; solo verifica
  manifest, precedencia y colisiones de gateway HTTP y queda cerrado salvo
  regresion.
- T191 no cambia launch env/io ni stop de olas Codex; el cierre 2026-05-26
  solo define deadline y escalation del conector neutral de procesos.

## Priorizacion scanner 2026-05-24 sexagesima octava pasada

Backlog asociado:
`T192 server-status-operational-message-projection`,
`T193 factory-appspec-time-source-contract` y
`T194 governance-catalog-output-budget-freshness`.

Prioridad alta:

- Resolver T192 antes de exponer o exportar status/auditoria de automejora idle:
  T41 puede aportar redaccion general, pero falta owner para mensajes
  operativos en `StateV0`.
  Cierre 2026-05-26: T192 queda cubierto por `*_operational_message` en
  `StateV0`/status publico y por resumen/redaccion de payloads operativos en
  auditoria; los campos legacy se conservan como compatibilidad.
  Rework 2026-05-26: entrega revalidada sin relanzar otro padre; el contexto
  `required ref_only` queda resuelto por lectura local y evidencia explicita en
  ACK.
- T193 queda cerrado el 2026-05-26 para el corte factory/AppSpec: el caso de uso
  ya no tiene `time.Now()` oculto, exige reloj inyectado y rechaza `now` cero
  con issue estable. Las ampliaciones de preview/backlog deben seguir usando esa
  frontera.
  Rework 2026-05-26: entrega revalidada sin relanzar otro padre; el contexto
  `required ref_only` queda resuelto por lectura local y evidencia explicita en
  ACK.

Prioridad media:

- T194 queda cerrado el 2026-05-26 para la proyeccion publica de gobernanza:
  `output_budget` bounded, version/freshness/source refs y resumen acotado de
  estados no publicos quedan en `GovernanceCatalogPublicQuery v0`.

Duplicaciones a evitar:

- T192 no sustituye T41, T85 ni T95: cubre proyeccion de mensajes operativos en
  status, no payload general de auditoria, configuracion ni persistencia.
- T193 no sustituye T161: la politica global de reloj/ref sigue siendo comun,
  pero factory/AppSpec necesita un contrato focal.
- T194 no sustituye T82 ni T175: forma HTTP y presupuesto MCP consumen la
  proyeccion de gobernanza, no la definen por si solos.

## Priorizacion scanner 2026-05-24 sexagesima novena pasada

Backlog asociado:
`T195 mcp-tool-input-schema-descriptor-sync` y
`T196 cli-command-catalog-help-dispatch-sync`.

Prioridad alta:

- T195 queda cerrado en codigo el 2026-05-26: `/mcp tools/list` deriva
  `inputSchema` desde DTOs registrados por tool, las capabilities de operador
  declaran shape/refs requeridas/errores publicos y el fallback marca
  `mcp_transport_schema_stale` de forma verificable. T174/T175/T182 siguen
  gobernando protocolo, salida y deadline.
- Resolver T196 antes de ampliar CLI como superficie publica de operador. El
  dispatch, los flagsets y la ayuda localizada deben tener una fuente comun
  para no guiar a operadores o agentes hacia comandos incompletos.

Prioridad media:

- T195 debe coordinarse con T93: el toolbelt de prompts puede consumir el
  descriptor canonico, pero no debe ser la fuente de verdad de schemas.
- T196 debe coordinarse con T75 y T119: i18n y docs server-first consumen el
  catalogo CLI, pero no sustituyen pruebas de dispatch/flags.

Duplicaciones a evitar:

- T195 no reabre parsing JSON-RPC ni presupuesto de salida; solo verifica
  fidelidad de descriptors de tools frente a implementacion. Resources quedan
  separados en T198.
- T196 no cambia semantica de comandos ni clientes REST; solo gobierna catalogo,
  ayuda, flags y ruta de dispatch sin volcar argumentos sensibles.

Estado 2026-05-26: T196 queda cerrado localmente para `orquesta-cli`.
`cliCommandCatalogV0` concentra rutas, flags visibles ES/EN, cliente destino,
perfil de efecto, estado y handler; `cliHelpTextForArgsV0` y
`dispatchOrquestaCLIV0` consumen el catalogo, con prueba de paridad y error de
comando desconocido redactado.

## Priorizacion scanner 2026-05-24 septuagesima pasada

Backlog asociado:
`T197 cli-rest-response-bounds-redaction-parity`,
`T198 mcp-resource-descriptor-source-sync` y
`T199 mcp-public-error-code-catalog`.

Prioridad alta:

- Resolver T197 antes de ampliar clientes CLI read-only o mutantes. T138 cubre
  respuestas HTTP salientes en otros adaptadores, pero la CLI sigue teniendo
  readers propios que acaban en envelopes publicos.
- Resolver T198 antes de usar resources MCP como fuente de planificacion de
  agentes externos. T195 cubre tools; operational-status, shared contracts,
  roadmap/governance y operator capabilities necesitan freshness y owner
  propios.
- T199 queda cerrado localmente el 2026-05-27 con catalogo publico inicial en
  `orquesta-i18n-docs` y paridad focal en MCP, operador, web, CLI y servidor.
  Nuevos handlers MCP/HTTP deben consumir ese catalogo antes de sumar codigos.
  Si reciben un error no catalogado, deben devolver codigo generico publico y
  no propagar detalle de `err.Error()` aunque este redactado.

Prioridad media:

- T197 debe coordinar con T139 para que stdout/JSON de CLI siga siendo shape
  publico, no body HTTP crudo ni diagnostico local.
- T198 debe coordinar con T175 para que un resource stale, no configurado o
  demasiado grande degrade con reason code compacto.
- T199 debe coordinar con T75/i18n: un codigo publico puede tener fallback
  temporal, pero la clave debe ser estable y testeable.

Estado 2026-05-27: T198 queda cerrado focalmente para el transporte MCP real.
Cada `MCPTransportResourceEnvelopeV0` declara `descriptor_source` y
`resources/list` lo expone con owner, fuente canonica, freshness, DTO/validador,
fuente de errores publicos y verificacion, saneado contra HOME, rutas locales,
tokens, provider, prompts, transcripts, DB y payloads crudos. La cobertura
comprueba paridad contra owners reales para operational-status, timeline,
FunctionContract y operator capabilities.
Revalidacion burst 002 2026-05-27:
`agent-ref-task-autoprogramming-85571f97bc5e-g01` conserva este cierre como
focal; el contexto `ref_only` queda resuelto por lectura local/evidencia ACK y
no reabre T195, T197 ni T199.
Revalidacion retry `agent-ref-task-autoprogramming-44e164597ee4-g01`:
conserva el cierre con lectura local del contexto `ref_only` y prueba
obligatoria focal, sin ampliar owners ni write-set.

Estado 2026-05-27: T197 queda cerrado para clientes REST de `orquesta-cli`.
`transport_rest_response_v0.go` concentra limite por comando, validacion de
`Content-Type`, rechazo de trailing JSON y detalle no-2xx redactado para
FunctionContract, OperationalStatus, GovernanceCatalog, ServerStatus,
bootstrap AppSpec, autoprogramacion/run control/queue y solicitudes de nueva
app.

Duplicaciones a evitar:

- T197 no sustituye entrada CLI (T142), catalogo/help (T196), idempotencia
  publica (T162) ni construccion de URL (T177).
- T198 no sustituye query operativo, governance ni FunctionContract; solo
  verifica descriptors de resource contra esos owners.
- T199 no redefine body parsing, headers, protocolo JSON-RPC ni schemas; solo
  gobierna catalogo y equivalencia de error publico.

## Priorizacion scanner 2026-05-24 septuagesima primera pasada

Backlog asociado:
`T200 cmd-server-management-rest-client-policy`.

Prioridad media-alta:

- Resolver T200 antes de ampliar comandos de gestion en `cmd/orquesta-server` o
  reutilizarlos en automatizacion residente. El binario servidor no debe quedar
  como segunda CLI con politicas propias de lectura HTTP, errores y redaccion.
- Alinear `status`, `run-status` y `stop` con los helpers publicos ya previstos
  para CLI/web/MCP cuando proceda, conservando que son comandos locales de
  operador y no surface remota nueva.

Duplicaciones a evitar:

- T200 no sustituye T197: `modulos/orquesta-cli` conserva sus clientes REST,
  catalogo y envelopes propios.
- T200 no sustituye T139 ni T199: consume shape de salida y catalogo de errores
  cuando existan, pero su owner es la lectura HTTP y redaccion de clientes del
  binario servidor.
- T200 no reabre T177/T162: URL base e idempotencia publica siguen con sus
  owners; aqui solo se exige que comandos de gestion no lean ni impriman bodies
  crudos sin presupuesto.

## Priorizacion scanner 2026-05-24 septuagesima segunda pasada

Backlog asociado:
`T201 outbound-connector-retry-backoff-rate-policy`,
`T202 outbound-domain-correlation-idempotency-headers` y
`T203 opes-bridge-pagination-window-policy`.

Prioridad media-alta:

- Resolver T201 antes de ampliar loops residentes de OPES o conectores HTTP de
  dominio. Los timeouts y ledgers actuales evitan algunos duplicados, pero no
  gobiernan retry/backoff/rate limit cuando una app externa responde 429/503,
  queda lenta o devuelve errores temporales.
  Aplicacion focal 2026-05-27: T201 ya gobierna retry/backoff opt-in en
  `domain_work-http`, OPES REST y `runExternalBridgeLoopV0`, con `Retry-After`
  acotado, jitter/sleep inyectables, presupuesto de intentos y bloqueo de
  mutaciones no idempotentes. Revalidacion assessment OrquestaV2 2026-05-27:
  la bateria obligatoria de T201 pasa completa en el write-set declarado.
- Resolver T203 antes de ejecutar secuencias OPES largas contra instancia
  temporal con muchos derivados. Un limite bajo es necesario para seguridad,
  pero sin cursor/ventana estable puede ocultar jobs posteriores detras de refs
  ya enviadas o no convertibles.

Prioridad media:

- Resolver T202 junto a T201: si un retry cruza proxy, balanceador o app externa,
  la correlacion/idempotencia debe estar en headers compactos ademas de en el
  DTO JSON para que la otra frontera no tenga que parsear payloads.

Duplicaciones a evitar:

- T201 no sustituye T154 ni T103: deadline y limite/redaccion de respuesta son
  owners separados; aqui solo se decide si/cuando reintentar y como publicar
  backoff/rate.
- T202 no sustituye T162: idempotencia de clientes publicos del control plane
  sigue aparte; esta tarea gobierna cabeceras salientes de conectores de
  dominio.
- T203 no sustituye T98/T101: claim de entrada y ledger de submit siguen
  necesarios; paginacion/ventana decide que trabajos puede ver el bridge antes
  de reclamar o enviar.

## Priorizacion scanner 2026-05-27 primera pasada

Backlog asociado:
`T211 ops-dashboard-live-progress-cache-policy`.

Prioridad media:

- Resolver T211 antes de usar `/ops` como senal unica de salud de la
  autoprogramacion residente. T210 gobierna el calculo de progreso en stats, y
  T209 gobierna el reporte de uso/quota Codex; `/ops` necesita su propio owner
  para frescura, cache y agregacion visual de esas senales.
  Aplicacion focal 2026-05-27: T211 queda cerrada para el panel web con
  proyeccion cliente `progress_source`/`freshness`/`stats_fetch_status`/reason
  code; completadas cacheadas no contaminan agregados vivos y uso sin reporte se
  muestra como `unknown`/`unavailable` con causa publica.
- La cache cliente no debe duplicar reglas de cierre: `tasks_closed` sigue
  perteneciendo al workflow/review, `tasks_delivered` al progreso vivo y
  `agent/process` a telemetria parcial. El dashboard solo proyecta esos estados
  y debe publicar reason code si la fuente fresca falla.

Duplicaciones a evitar:

- T211 no sustituye T210: no recalcula progreso base, solo impide que la
  agregacion/cache de `/ops` lo degrade a 0% o lo mezcle con snapshots
  completados.
- T211 no sustituye T209: usage/quota sigue viniendo de puerto/reporte
  redactado; `/ops` solo muestra `unknown`/`unavailable` con causa cuando no
  hay dato fresco.
- T211 no sustituye T103/T138/T197/T199: limites de respuesta, redaccion y
  catalogo de errores publicos siguen con sus owners; esta tarea gobierna
  frescura/cache y fallback visual del dashboard.

## Priorizacion scanner 2026-05-27 segunda pasada

Backlog asociado:
`T212 guardian-public-result-and-repair-packet-redaction`,
`T213 guardian-command-output-budget-and-env-isolation` y
`T214 guardian-binary-promotion-artifact-policy`.

Prioridad media-alta:

- T212 queda cubierto el 2026-05-27 para salida y repair packet redactados:
  `cmd/orquesta-guardian` separa manifest local de payload automatizable con
  refs opacas, freshness, reason codes, contadores y comandos como refs.
- T213 queda cubierto el 2026-05-27 para output/env del guardian:
  build/test/healthcheck/repair capturan tail redactado con presupuesto y
  reason code publico, y ejecutan con entorno minimo allowlistado mas variables
  explicitas de composicion.
- T214 queda cubierto localmente el 2026-05-27 para artefactos binarios del
  guardian: copia streaming con limite, SHA-256, guardas `Lstat`/open/post-copy,
  bloqueo de symlinks/hardlinks inseguros y raiz declarada opt-in, tmp no
  colisionable, fsync, rename atomico y manifest de recuperacion de
  `candidate/current/last_good`.

Duplicaciones a evitar:

- T212 no sustituye T208: el break-glass puede seguir pendiente como ciclo
  operativo aunque salida/repair packet queden redactados.
- T213 no reabre seguridad Codex general: solo fija presupuesto de output y
  entorno minimo para comandos lanzados por el guardian.
- T214 no reabre AppVCS ni refs deterministas generales: solo gobierna copia,
  hashes, atomicidad y estados recuperables de binarios del guardian.

## Priorizacion scanner 2026-05-27 tercera pasada

Backlog asociado:
`T215 guardian-candidate-readiness-gate`,
`T216 promotion-guardian-result-contract` y
`T217 guardian-shutdown-http-client-policy`.

Prioridad media-alta:

- Resolver T215 antes de usar el guardian como barrera de promocion residente.
  `/healthz` solo demuestra proceso HTTP vivo; la promocion necesita readiness
  operativa de `/api/v0/server/readiness` contra estado/runtime temporal antes
  de tocar `current_bin`.
  Estado 2026-05-27: cerrado para el guardian. El healthcheck del candidato
  exige liveness y readiness `ready=true`; un candidato con HTTP vivo pero sin
  readiness publica bloquea con `candidate_readiness_not_ready` y conserva
  estado/runtime temporal, dry-run de bridges y automejora idle desactivada.
- T216 queda cerrado el 2026-05-27: el servidor ya no cierra promocion por solo
  exit code; consume `orquesta_guardian_result.v0`, conserva evidence refs y
  distingue fallo de candidato, parser, timeout y exit no cero.
- Resolver T217 antes de depender de `shutdown-server` para restart de guardian.
  T189 gobierna senales del servidor y T200 comandos locales del binario, pero
  el guardian conserva otro cliente HTTP que debe tener limite, content-type,
  trailing-data check e idempotencia propia.

Duplicaciones a evitar:

- T215 no sustituye T176 ni T188: headers/metodos siguen en el servidor; aqui
  solo se exige que el guardian consulte readiness suficiente antes de promocion.
- T216 no sustituye T212: el repair packet puede quedar redactado, pero la
  promocion residente necesita validar resultado estructurado y no stdout libre.
- T217 no sustituye T200 ni T189: el cliente shutdown del guardian es frontera
  break-glass; no redefine comandos `status/stop` ni politica general de
  senales del servidor.

## Priorizacion scanner 2026-05-27 cuarta pasada

Backlog asociado:
`T218 guardian-repair-agent-launch-contract`,
`T219 guardian-path-root-and-control-surface-policy` y
`T220 guardian-manifest-retention-and-replay-policy`.

Prioridad media-alta:

- T218 queda cerrado el 2026-05-27 para el contrato de lanzamiento
  `--repair-codex`: packet versionado, write-set cerrado, ACK terminal, tests
  requeridos, presupuesto de un agente, break-glass unmanaged auditado y sandbox
  amplio solo con opt-in/evidence ref. La preferencia por cola normal cuando
  Orquesta residente siga sana queda como regla operativa del runbook.
- T219 queda cerrado localmente el 2026-05-27 para guardian/server: resultados,
  auditoria y packets de reparacion publican `path_policy` con refs hash y
  clasificaciones; las rutas criticas se bloquean si salen de raices declaradas
  o cruzan symlinks. T212 redacta salida y T214 gobierna copia de binarios; T219
  ya no debe reabrirse salvo regresion demostrada.
- Resolver T220 antes de ejecutar muchos intentos de guardian en modo residente.
  Sin manifest idempotente, escritura durable y retencion por categoria, logs,
  repair packets y runtimes temporales pueden pisarse, crecer sin limite o
  desaparecer antes de servir como evidencia causal.
  Estado 2026-05-27: cerrado localmente para el estado durable propio del
  guardian: manifests/repair packets/prompts/logs redactados usan
  tmp+fsync+rename, los intentos quedan indexados por ref, la salida publica
  expone `retention_status` y la limpieza por categoria conserva evidencias
  actuales y `last_good`.

Duplicaciones a evitar:

- T218 no sustituye T47/T49/T63/T143/T157: consume ACK, seguridad Codex,
  sidecar y control-file policy existentes; solo gobierna el lanzamiento
  break-glass del reparador.
- T219 no sustituye T50/T127/T131: exclusiones de control files, artefactos
  locales ignorados y redaccion documental siguen con sus owners; aqui se fija
  la superficie de rutas del guardian.
- T220 no sustituye T97/T100/T104/T159 ni T214: logs del daemon, archivos de
  revision, escritura durable general, control files y binarios siguen aparte;
  aqui se gobiernan manifests, repair packets, prompts, logs y runtimes
  temporales propios del guardian.

## Priorizacion scanner 2026-05-27 quinta pasada

Backlog asociado:
`T221 guardian-promotion-lease-and-concurrency-policy`,
`T222 guardian-command-effect-profile-policy` y
`T223 promotion-guardian-causal-receipt`.

Prioridad media-alta:

- Resolver T221 antes de permitir guardian residente o break-glass concurrente.
  T214 hace atomica la copia de binarios y T220 hace durable el manifest, pero
  ninguno decide quien posee el intento activo cuando dos procesos trabajan
  sobre el mismo `current_bin`, `last_good_bin` o `state_dir`.
  Cierre local 2026-05-27: `cmd/orquesta-guardian` serializa
  `check-promote`, `restore-last-good` y `shutdown-server` con lease durable por
  intento/promocion, bloquea `guardian_promotion_lease_busy` y revalida el lease
  antes de tocar binario vivo o senalizar.
- Resolver T222 antes de ampliar comandos configurables del guardian. T213
  limita salida y entorno, pero el contrato publico todavia no clasifica si un
  comando de build/test/repair puede tocar red, Git remoto, OPES, proveedor real
  o artefactos fuera del candidato.
  Cierre local 2026-05-27: `cmd/orquesta-guardian` clasifica comandos con
  `guardian-command-effect-profile-policy-v0`, permite por defecto solo build
  canonico y `go test`, bloquea efectos externos sin evidence ref y publica
  reason codes/refs sin shell completo.
- Resolver T223 antes de usar la salida del guardian como evidencia de
  promocion final. T216 valida el schema del resultado, pero el servidor necesita
  receipt causal propio que enlace ese resultado con staging, refs opacas y hash
  del candidato.
  Estado 2026-05-27: cerrado en `cmd/orquesta-server` con
  `promotion_guardian_receipt.v0`; el receipt enlaza resultado del guardian,
  staging aplicado, refs opacas, manifest/evidence refs y hash/tamano compacto
  del candidato si existe. `--promote=false` queda como
  `candidate_verified` bloqueante, no como promocion activa.

Duplicaciones a evitar:

- T221 no sustituye T31/T57: colas y outbox siguen siendo owners de claims
  generales; aqui solo se gobierna exclusion mutua del guardian sobre artefactos
  y estado local.
- T222 no sustituye T139/T180/T212/T213: esos rails gobiernan forma publica,
  redaccion, output y entorno; aqui se decide si el comando esta autorizado y
  que perfil de efecto declara.
- T223 no sustituye T13/T40/T156/T216: staging Git, promocion general,
  output Git y contrato del resultado del guardian siguen separados; aqui se
  fija el receipt causal servidor-guardian-promocion.

## Priorizacion scanner 2026-05-27 sexta pasada

Backlog asociado:
`T224 guardian-candidate-process-tree-lifecycle`,
`T225 guardian-forced-shutdown-escalation-contract` y
`T226 guardian-repair-attempt-budget-and-idempotency`.

Prioridad media-alta:

- T224 quedo cerrado el 2026-05-27 para el guardian: el healthcheck publica
  politica de proceso y receipt de stop compacto; Unix confirma parada de grupo
  de proceso con deadline/escalado antes de promocionar, Windows declara alcance
  de proceso padre y un stop ambiguo bloquea la promocion. T215 sigue validando
  readiness HTTP.
- T225 quedo cerrado el 2026-05-27: `shutdown-server` ya no escala
  `force_after_timeout` por defecto; el break-glass exige opt-in explicito y
  evidence ref, y publica estados de escalado sin PID/host/URLs/cuerpo HTTP.
  T217 sigue acotando el cliente HTTP, no la decision de escalado.
- T226 quedo cerrado el 2026-05-27 para presupuesto e idempotencia de repair:
  cada intento tiene `repair_attempt_ref`, `failure_packet_hash`, registro
  durable por scope de promocion/intento/run y bloqueo publico
  `guardian_repair_attempt_duplicate` o
  `guardian_repair_attempt_budget_exhausted`. T218 fija el contrato del agente
  reparador y T222 autoriza comandos; no se reabren aqui.

Duplicaciones a evitar:

- T224 no sustituye T191 ni T215: senales generales del runtime y readiness del
  servidor siguen con sus owners; aqui se gobierna solo el proceso candidato del
  guardian.
- T225 no sustituye T189/T199/T217: politica de senales, shutdown del servidor
  y cliente HTTP siguen separados; aqui se fija la decision de escalado forzado
  del guardian.
- T226 no sustituye T47/T218/T220/T221/T222: receipts de tests, launch del
  reparador, retencion, leases y comandos autorizados siguen aparte; aqui se
  limita el numero y reentrada de intentos de repair.

## Priorizacion scanner 2026-05-27 octava pasada

Backlog asociado:
`T228 guardian-candidate-address-ownership-policy` cerrado el 2026-05-27,
`T229 promotion-guardian-runner-env-isolation` y
`T230 guardian-skip-health-breakglass-policy`.

Prioridad media-alta:

- T228 queda cerrado para ownership de direccion del candidato: addr automatico
  por `127.0.0.1:0` con lectura de statefile del candidato, addr declarado solo
  loopback con puerto concreto y bloqueo `candidate_addr_unowned` cuando no se
  puede probar propiedad.
- Retry OrquestaV2 2026-05-27
  `agent-ref-task-autoprogramming-db88e93cffaf-g01`: cierre revalidado con la
  bateria focal requerida y contexto `ref_only` resuelto por lectura local mas
  evidencia explicita en ACK.
- T229 queda cerrado localmente el 2026-05-27 para el runner servidor ->
  guardian: entorno `minimal_allowlist`, vars `ORQUESTA_GUARDIAN_*` explicitas,
  cache local derivada del state dir y bloqueo de HOME/Codex/proxies/Git/
  proveedor/secretos salvo allowlist declarada con evidence refs.
- Retry OrquestaV2 2026-05-27
  `agent-ref-task-autoprogramming-22cfff60721c-g01`: cierre revalidado con la
  bateria focal requerida y contexto `ref_only` resuelto por lectura local mas
  evidencia explicita en ACK.
- Rework de revision 2026-05-27
  `agent-ref-task-ref-review-rework-task-autoprogramming-22cfff60721c-g01-938062230e08e3a836b342951d5c588d`:
  el runner servidor -> guardian descarta `ORQUESTA_GUARDIAN_*` heredadas y
  deduplica el entorno para que las variables explicitas del request sean la
  unica fuente de configuracion del guardian.
- Retry de rework OrquestaV2 2026-05-27
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-autoprogr-3b1cd0fc9711a3bf097b5db163987e69`:
  conserva el cierre de T229, revalida la bateria focal requerida y resuelve
  contexto `ref_only` mediante lectura local mas evidencia explicita en ACK.
- Retry final de rework OrquestaV2 2026-05-27
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-602dc9edc1ac71b6268bcede5729e588`:
  conserva el cierre de T229, revalida la bateria focal requerida y resuelve
  contexto `ref_only` mediante lectura local mas evidencia explicita en ACK.
- T230 queda aplicado localmente el 2026-05-27: `--skip-health` con
  `promote=true` bloquea por defecto con `guardian_healthcheck_required`; el
  uso autorizado requiere evidence ref break-glass y publica
  `candidate_promoted_breakglass` con causa compacta, no una readiness viva
  equivalente.

Duplicaciones a evitar:

- T228 no sustituye T87/T215/T224: liveness/readiness y ciclo de vida del
  proceso siguen con sus owners; aqui solo se gobierna ownership de direccion y
  anti-hijack.
- T229 no sustituye T213/T216/T222: output/env de comandos internos, parseo del
  resultado y autorizacion de comandos siguen separados; aqui se aisla el
  runner que arranca el guardian.
- T230 no sustituye T215/T223/T225: readiness, receipt causal y escalado de
  shutdown siguen aparte; aqui se gobierna la excepcion de saltar healthcheck.

## Priorizacion scanner 2026-05-27 novena pasada

Backlog asociado:
`T231 guardian-config-env-strictness` y
`T232 guardian-cli-extra-args-strictness`.

Prioridad media:

- T231 queda resuelto el 2026-05-27 para uso residente frecuente del guardian:
  bool, duracion, entero, budget y allowlist invalidos bloquean con reason code
  publico, el resultado publica `config_effective` compacto y el servidor
  distingue config invalida de fallo de candidato. T213 aisla entorno y salida,
  T216 parsea resultado y T225 decide escalado; siguen siendo owners separados.
- T232 queda resuelto el 2026-05-27 para `orquesta-guardian`: la misma
  superficie de configuracion ya no acepta typos posicionales y bloquea antes
  de ejecutar efectos si quedan argumentos sobrantes tras `flag.Parse`.

Duplicaciones a evitar:

- T231 no sustituye T85/T213/T216/T217/T225/T228: config canonica del servidor,
  entorno/output, resultado del guardian, cliente shutdown, escalado y runner
  servidor -> guardian siguen con owners propios; aqui solo se gobierna parseo
  estricto de env/flags del guardian.
- T232 no sustituye T196/T200/T222: catalogo CLI, cliente REST de gestion y
  autorizacion de comandos siguen separados; aqui se rechazan argumentos
  sobrantes en `orquesta-guardian`.

## Priorizacion scanner 2026-05-27 decima pasada

Backlog asociado:
`T233 guardian-local-diagnostic-ref-stability`,
`T234 guardian-command-ref-template-profile` y
`T235 guardian-repair-packet-read-budget-and-type`.

Prioridad media:

- T233 queda cerrado localmente el 2026-05-27: las refs publicas del guardian
  para manifest, repair packet, repair launch, outputs y diagnosticos locales
  usan ref causal, tipo de diagnostico y hash permitido para outputs, sin
  depender de paths absolutos locales. T219 clasifica rutas y T220 conserva
  manifests; este cierre solo gobierna identidad publica portable.
  Revalidacion OrquestaV2
  `agent-ref-task-autoprogramming-a35df67e3b46-g01`: el contexto obligatorio
  `ref_only` se resuelve por lectura local/evidencia ACK y no reabre T233 ni
  convierte `worktree_ref`/`branch_ref` en rutas o nombres Git.
- Resolver T234 junto a T222: la autorizacion de comandos necesita perfiles,
  pero la identidad publica tambien debe dejar de hashear shell expandido con
  rutas locales o comandos configurables.
  Cierre local 2026-05-27: `cmd/orquesta-guardian` deriva `command_ref` desde
  `phase`, `command_profile`, `template_ref` y `attempt_ref`; templates
  canonicas ignoran `state_dir`/`candidate_bin`, y comandos no canonicos exponen
  `command_template_changed` con evidence ref compacta.
- T235 queda cerrado localmente el 2026-05-27: el repair packet del guardian se
  valida con lectura `lstat/open/read` acotada, schema, redaction level,
  hash/attempt ref, presupuesto y fase causal antes de construir prompt o
  comando Codex; si falla bloquea con `guardian_repair_packet_invalid`.

Duplicaciones a evitar:

- T233 no sustituye T184/T219/T220/T223: hash global, rutas, retencion y receipt
  causal siguen separados; aqui se fijan refs publicas de diagnosticos locales.
- T234 no sustituye T180/T216/T222/T223: salida, schema, autorizacion y receipt
  siguen con sus owners; aqui se gobierna `command_ref` publico.
- T235 no sustituye T50/T143/T157/T218/T220: control files, tamano/redaccion,
  hardening de lectura, launch del reparador y retencion siguen aparte; aqui se
  valida el packet que el guardian convierte en contexto reparador.

## Priorizacion scanner 2026-05-27 undecima pasada

Backlog asociado: `T236 public-mutation-identity-contract`.

Prioridad media-alta:

- Resolver T236 antes de ampliar acciones mutables de `/ops`, MCP real o CLI.
  Hoy MCP tiene helper propio de identidad publica, web/CLI generan ids por
  adaptador y el servidor/auditoria extraen headers por rutas concretas. Esa
  duplicacion puede hacer que una misma mutacion sea idempotente en una
  superficie y no en otra.
- Mantener lecturas read-only separadas de mutaciones aunque usen POST por
  compatibilidad. La politica compartida debe exigir idempotency solo cuando el
  contrato declara efecto externo o cambio durable.
- Actualizacion 2026-05-27: T236 queda cubierto localmente por
  `modulos/orquesta-server/publicidentity`, consumido desde MCP, web, CLI,
  servidor y `cmd/orquesta-server` sin mover seguridad de transporte, auditoria
  completa ni limites de entrada fuera de sus owners.

Duplicaciones a evitar:

- T236 no sustituye T55/T99/T102: seguridad de transporte, recursos HTTP y
  boundary legacy siguen con sus owners. Aqui solo se gobierna identidad publica
  de request/correlation/idempotency.
- T236 no sustituye T94/T138/T139: auditoria y salida publica siguen separados;
  solo deben consumir la identidad efectiva ya normalizada.
- T236 no sustituye T142: limites de entrada CLI y clasificacion de origen
  siguen en el owner local de CLI; la identidad compartida no autoriza leer
  ficheros ni payloads mayores.

## Priorizacion scanner 2026-05-27 decimotercera pasada

Backlog asociado: `T239 codex-shutdown-checkpoint-attempt-correlation`
(cerrado localmente 2026-05-27).

Prioridad media:

- Resolver T239 antes de usar shutdown cooperativo Codex como evidencia fuerte
  en reinicios frecuentes. El ACK de checkpoint ya valida run/agente/ref, pero
  la ref deterministica por run/agente no distingue dos mutaciones de shutdown
  con razon, evidencia o escalado distintos.
- Mantener idempotencia por intento: reintentar la misma mutacion debe aceptar
  el mismo ACK; una nueva mutacion debe requerir ACK nuevo o mismatch publico.

Duplicaciones a evitar:

- T239 no sustituye T24/T143/T157: ACK terminal y lectura/redaccion de control
  files siguen con sus owners. Aqui solo se gobierna identidad causal del
  intento de shutdown checkpoint.
- T239 no sustituye T199/T225: shutdown general y escalado forzado siguen
  separados; aqui se fija cuando un ACK cooperativo pertenece al intento vivo.
- T239 no sustituye T223: receipt causal de promocion guardian sigue aparte; la
  tarea nueva aplica al protocolo Codex de checkpoint durante shutdown.

Cierre local 2026-05-27: runtime Codex valida `shutdown_attempt_ref` en request
y ACK de checkpoint; `orquesta-app-codex-stack` deriva ese ref desde la mutacion
de shutdown y mantiene `waiting_checkpoint` ante
`shutdown_checkpoint_attempt_mismatch`.
Rework 2026-05-27: T239 queda conservado como cierre local; el contexto
obligatorio `ref_only` se resuelve por lectura local del paquete y evidencia en
ACK, sin ampliar owner ni relanzar agente padre.

## Priorizacion scanner 2026-05-27 decimoquinta pasada

Backlog asociado: `T240 backlog-scan-doc-path-budget-policy`.

Prioridad media-alta:

- Resolver T240 antes de ampliar el scanner a mas shards o permitir que paquetes
  externos reinyecten `backlog_scan_doc`. La lectura local para hash debe quedar
  tan acotada como el write-set: catalogo canonico, ruta relativa dentro del
  proyecto, sin control files/symlinks y con presupuesto explicito.
- Mantener la semantica de merge lease de T43: si la foto documental no coincide
  o trae docs no canonicos, el resultado correcto es bloqueo/rebase o
  `CONSULTA AL DIRECTOR`, no lectura local fuera de catalogo ni `missing`
  silencioso.

Duplicaciones a evitar:

- T240 no sustituye T37: el parser de backlog sigue siendo owner de extraer
  secciones, tests y estado desde Markdown; aqui solo se gobierna path/budget de
  docs usados para epoch.
- T240 no sustituye T44: contexto `required ref_only` sigue validandose en
  materializer/runtime/ACK; aqui solo se evita que el merge lease lea fuentes no
  canonicas.
- T240 no sustituye T50/T142: control files y entrada CLI mantienen sus owners;
  esta tarea consume esas politicas para bloquear rutas antes de hashear docs.

Cierre local 2026-05-27: T240 queda resuelto en `cmd/orquesta-server` con
catalogo canonico de documentos de scanner, lectura acotada por presupuesto,
rechazo de rutas no canonicas/control files/symlinks/no regulares y bloqueo de
`backlog_scan_doc` fuera del catalogo actual. No reabre T37/T43/T44/T50/T142 ni
los owners de identidad de scanner.

## Priorizacion scanner 2026-05-27 decimocuarta pasada

Backlog asociado: `T241 idle-self-improvement-provider-auth-recovery-contract`.

Prioridad media-alta:

- Resolver T241 antes de tratar `provider_auth_blocked` como error generico de
  automejora. El bloqueo nace en composicion Codex/proveedor, pero el residente
  lo convierte en decision de cola; si no hay contrato de recuperacion, puede
  preparar scanners repetidos o quedarse bloqueado sin accion verificable.
- Mantener la accion de operador fuera del nucleo. El servidor puede exponer
  reason codes y evidence refs compactas, pero renovacion de credenciales,
  comprobacion de proveedor y confirmacion de cuota pertenecen a adaptadores de
  composicion.

Duplicaciones a evitar:

- T241 no sustituye T29/T56: contabilidad de proveedor y proyeccion de
  credenciales siguen con sus owners. Aqui solo se gobierna el ciclo de bloqueo
  y recuperacion de automejora idle.
- T241 no sustituye T94/T138/T139/T187: auditoria, salida publica y redaccion
  siguen separados; deben consumir un reason code ya normalizado.
- T241 no sustituye T236: identidad/idempotencia de mutaciones publicas sigue
  aparte; T241 solo debe usar esa identidad cuando exponga una accion de
  operador o reintento.

Estado 2026-05-27: T241 cerrado en el write-set del servidor residente. La
recuperacion de auth de proveedor ya tiene contrato publico compacto y la
accion real queda en composicion/run-control, sin mover credenciales, proveedor
ni cuotas al nucleo.
Rework 2026-05-27: el backlog vivo queda sincronizado con este cierre; no se
abre owner nuevo para `provider_auth_blocked`, solo se conserva la frontera entre
diagnostico publico, accion de operador y reintento idempotente.

## Priorizacion scanner 2026-05-27 decimocuarta pasada

Backlog asociado: `T242 server-supervisor-loop-file-split-before-growth` y
`T243 mcp-public-tool-file-split-before-growth`.

Prioridad media:

- Resolver T242 antes de anadir mas comportamiento residente al supervisor del
  servidor. El fichero concentra tick, wakeup, leases, observabilidad y
  adaptacion de puertos, lo que aumenta el riesgo de duplicar politica de
  shutdown, proveedor o automejora.
- Resolver T243 antes de ampliar tools MCP con identidad publica, budgets,
  repair handoff o nuevos descriptors. Los tools deben seguir siendo
  adaptadores finos y no mezclar transporte, validacion, ejecucion y salida en
  un solo owner grande.

Duplicaciones a evitar:

- T242 no sustituye T32, T109, T136, T166 ni T210: wakeup, leases, waits,
  shutdown async y telemetria viva siguen con sus owners; aqui solo se parte el
  loop residente sin mover composicion al nucleo.
- T243 no sustituye T195, T198, T199 ni T236: schemas/descriptors, recursos,
  errores publicos e identidad global siguen separados; aqui solo se divide la
  implementacion local de tools MCP publicos.
- Ambas tareas dependen de la regla T45: no legitiman aumentar ficheros grandes
  ni cerrar entregas con crecimiento sobre baseline sin followup causal.

Estado 2026-05-27: T243 queda cerrado localmente en `orquesta-mcp`. Los tools
`orquesta.director.human_work.review_plan.v0` y
`orquesta.autoprogramming.self_improvement.propose.v0` conservan metadata,
nombres publicos y transporte, pero reparten input flexible, advice de operador,
proyeccion de errores/resultados y builders en ficheros productivos menores de
300 lineas. Evidencia T243: `go test -count=1 ./modulos/orquesta-mcp`.
T242 queda cerrado localmente en `orquesta-server`: el loop
residente queda separado de scheduling, decision, freshness, prepare, request y
refs de automejora, manteniendo ficheros productivos del owner bajo 300 lineas.
Revalidacion OrquestaV2 2026-05-27: los tests locales del supervisor se separan
en shards de tick, idle async, capacidad, blockers y helpers para que la
cobertura no vuelva a concentrar politica residente en un solo fichero.
Assessment OrquestaV2 2026-05-27: `task-autoprogramming-594d493666c4-g01`
repite el paquete de T242 y queda como revalidacion documental, no como nuevo
frente de implementacion.

## Priorizacion scanner 2026-05-27 decimosexta pasada

Backlog asociado: `T244 codex-stack-residual-product-file-split`.

Estado 2026-05-27: cerrado localmente por
`task-autoprogramming-bbfaee8d0f40-g01`; conservar solo como rail de regresion
si un cambio posterior vuelve a concentrar bridge, specs, contexto externo,
replan o ejecutor MCP/supervisor por encima del presupuesto.
Rework de revision 2026-05-27:
`task-ref-review-rework-task-autoprogramming-bbfaee8d0f40-g01-d8a1fe905f79cb6b486cecd73b06d1df`
mantiene T244 como cerrado, resuelve el contexto obligatorio `ref_only` con
evidencia ACK y no sustituye los owners vecinos listados abajo.

Prioridad media:

- Resolver regresiones de T244 antes de ampliar `orquesta-app-codex-stack` con
  mas puentes de autoprogramacion, specs, contexto externo, replan por
  assessment o ejecutores MCP/supervisor. T54 dejo cerrado el corte
  `codex-wave`/`director-wave` y `drain`; el residuo productivo que quedaba
  sobre 300 lineas se cerro localmente el 2026-05-27.

Duplicaciones a evitar:

- T244 no sustituye T54: no reabre el split ya cerrado de comandos Codex ni
  drain; cubre el residuo productivo no tocado por ese corte.
- T244 no sustituye T238: las politicas neutrales de autoprogramacion siguen en
  `modulos/orquesta-autoprogramming`; el stack Codex solo adapta esas politicas
  a runtime/composicion.
- T244 no sustituye T33, T36, T42, T47, T50, T56, T117, T143 ni T241: ACK
  terminal, pruebas requeridas, write-set, control files, credenciales,
  review gate, redaccion y provider auth mantienen owners propios.
- T244 depende de T45/T90 como baseline de no crecimiento: una entrega no debe
  cerrar si aumenta ficheros grandes sin split o followup causal.

## Priorizacion scanner 2026-05-27 decimosexta pasada

Backlog asociado: `T248 backlog-scan-section-id-and-sequence-policy`.

Estado 2026-05-27: cerrado para el servidor residente. El planner publica
`backlog_scan_ref` en contexto/criterios de ACK para cada merge lease de
scanner y reutiliza `scan_entry_ref`/digest/linea/hash de T249 para diferenciar
bloques `Escaneo backlog` duplicados o fuera de orden, sin renumerar historico.

Prioridad media:

- Resolver T248 antes de aumentar la concurrencia de scanners que escriben en
  el backlog vivo. Las etiquetas ordinales humanas ya se repiten y pueden
  quedar fuera de orden; la evidencia de merge/rebase necesita un `scan_ref`
  estable por bloque para no depender del titulo visible.
- Mantener historico intacto: no renumerar bloques antiguos ni usar el
  `scan_ref` como sustituto de `Txx`. Debe servir para trazabilidad de escaneo,
  no para cambiar el contrato ejecutable de tareas.

Duplicaciones a evitar:

- T248 no sustituye T37: el parser de Markdown sigue siendo owner de extraer
  tareas, alcance, criterios y tests.
- T248 no sustituye T43 ni T240: lease, epoch, hash y lectura segura de docs
  siguen separados; esta tarea solo da identidad estable a la seccion de
  escaneo que produjo o reviso esos datos.
- T248 no sustituye T236: identidad publica de mutaciones sigue siendo global;
  aqui se consume esa identidad para evidencia documental del scanner.
- Rework 2026-05-27: la correccion
  `agent-ref-task-ref-review-rework-task-autoprogramming-ce0100f5ae08-g01-a522429e98460b11fc001393c9f07b9e`
  no duplica rail ni owner nuevo. Conserva T248 cerrado como identidad/secuencia
  de escaneo, con `ref_only` resuelto por lectura local/evidencia ACK y prueba
  requerida del servidor residente.

## Priorizacion scanner 2026-05-27 decimosexta pasada

Backlog asociado: `T247 app-codex-autoprogramming-bridge-file-split`.

Prioridad media:

- Resolver T247 antes de anadir mas logica de automejora residente al stack
  Codex. El bridge actual concentra run, tasks, wait refs, plan-state, replay y
  continue request; partirlo evita duplicar politicas que ya tienen owners en
  `orquesta-autoprogramming`, `app-director-service` y el loop residente del
  servidor.
- Mantener el stack como composicion: la division no debe mover runtime Codex,
  state-file, proveedor ni rutas de control al core puro.

Estado 2026-05-27: T247 queda cerrado localmente. La composicion mantiene los
owners separados en `autoprogramming_bridge_request_v0.go`,
`autoprogramming_bridge_run_v0.go`, `autoprogramming_bridge_store_v0.go` y
`autoprogramming_bridge_continue_v0.go`; los tests del bridge se shardearon
para evitar un fichero local monolitico.

Rework 2026-05-27: la revision/rework de T247 queda acotada a evidencia
documental y ACK. No debe crear otro owner paralelo para request/run/store/continue
ni relanzar la tarea padre original; una reapertura futura necesita una
regresion concreta del bridge o una duplicacion nueva no cubierta por estos
owners.

Duplicaciones a evitar:

- T247 no sustituye T238: las politicas puras de programacion/particionado viven
  en `modulos/orquesta-autoprogramming`; el bridge solo adapta esas decisiones a
  run/tasks/continue del stack Codex.
- T247 no sustituye T241 ni T242: recuperacion por proveedor y supervisor
  residente siguen en `modulos/orquesta-server`/`cmd/orquesta-server`; el bridge
  no debe decidir auth, idle windows ni wakeups.
- T247 no sustituye T43/T240: epoch documental, merge lease y lectura acotada de
  docs del scanner siguen en el servidor. El bridge solo debe conservar refs ya
  validadas y no reparsear backlog.

## Priorizacion scanner 2026-05-27 decimosexta pasada

Backlog asociado: `T249 backlog-scan-entry-identity-and-order-policy`.

Prioridad media:

- Resolver T249 antes de usar titulos humanos de `Escaneo backlog` como
  evidencia suficiente para cierre, rebase o deduplicacion. Ya existen titulos
  repetidos/fuera de orden, asi que el contrato debe depender de refs compactas,
  hash de bloque y tipo de seccion, no solo de texto visible.
- Preservar el historico documental: si una entrada antigua queda duplicada, el
  sistema debe bloquear con reason code publico o pedir rebase, no renumerar ni
  borrar evidencia.

Duplicaciones a evitar:

- T249 no sustituye T37: el parser de backlog sigue siendo owner de extraer
  tareas, estado, criterios y tests desde Markdown.
- T249 no sustituye T43 ni T240: merge lease y path/budget de documentos siguen
  separados; T249 solo aporta identidad y orden de secciones de scanner.
- T249 no sustituye T116/T122/T140: integridad documental, precedencia canonica
  y solapes de backlog siguen como owners vecinos; aqui solo se corrige la
  correlacion de entradas de evidencia de scanner.
- Cierre T249 2026-05-27: `cmd/orquesta-server` mantiene el parser ejecutable
  limitado a `## Txx` y anade identidad/digest/issues para
  `## Escaneo backlog ...` en el merge lease. Los reason codes publicos son
  `backlog_scan_entry_duplicate` y `backlog_scan_entry_order_ambiguous`; el
  scanner debe pedir rebase o `CONSULTA AL DIRECTOR`, no renumerar historico.
  Revalidado con
  `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.

## Priorizacion scanner 2026-05-27 decimoctava pasada

Backlog asociado: `T251 backlog-scanner-required-test-scope-policy`.

Prioridad media:

- Resolver T251 antes de aumentar tandas documentales de assessment/backlog
  scanner. Un paquete con write-set solo documental no debe heredar siempre
  `go test -count=1 ./...` como prueba obligatoria si la seccion Txx o el
  director no lo pidieron; esa decision mezcla validacion transversal,
  verificacion documental y fiabilidad de codigo no tocado.
- Mantener compatibilidad estricta de ACK: si una prueba obligatoria viene en
  el paquete, el agente debe ejecutarla y declararla solo si pasa. La correccion
  futura vive en el planner/materializador de paquetes, no en relajar ACKs de
  agentes externos.

Duplicaciones a evitar:

- T251 no sustituye T37: el parser sigue extrayendo tests desde Markdown.
- T251 no sustituye T44 ni T47: contexto `ref_only` y receipts de pruebas
  requeridas siguen con sus owners; aqui solo se decide que comandos entran al
  paquete de scanner.
- T251 no sustituye `git diff --check` ni `go test -count=1 ./...` como
  verificacion transversal manual cuando un cambio realmente toca codigo o
  frontera de nucleo.

Cierre T251 2026-05-27: `cmd/orquesta-server` acota `required_tests` para
scanners documentales; el default global `go test -count=1 ./...` no se hereda
cuando el write-set solo cubre backlog/rail errors/duplicaciones, pero las
pruebas declaradas por una seccion Txx se conservan. La decision queda marcada
en `ContextRefs` mediante `required_test_origin:*` y
`required_test_scope_policy:*`. Revalidado con
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.
Rework de revision 2026-05-27: se confirma que T251 no reabre T37/T44/T47 ni
la verificacion transversal manual; backlog, rail error y esta matriz quedan
alineados, con contexto `ref_only` resuelto por lectura local/evidencia ACK.

## Priorizacion scanner 2026-05-27 decimoseptima pasada

Backlog asociado: `T250 federated-backlog-epoch-scope-policy`.

Prioridad media:

- Resolver T250 antes de aumentar fuentes federadas del backlog. El indice ya
  mezcla fuentes ejecutables y cuarentenadas; si todas entran en el epoch del
  scanner, un cambio historico puede bloquear trabajo vivo sin aportar tarea
  ejecutable.
- Mantener la cuarentena T124 como evidencia, no como dependencia activa del
  scanner. Las fuentes legacy solo deben volver al epoch cuando una tarea
  explicita de rescate, composicion externa o `related_txx` lo pida.

Duplicaciones a evitar:

- T250 no sustituye T240: path/budget de lectura sigue separado; aqui solo se
  decide que fuentes participan en el epoch activo.
- T250 no sustituye T88/T116/T124: indice federado, precedencia documental y
  cuarentena legacy siguen con sus owners; la nueva tarea consume esos estados.
- T250 no sustituye T248/T249: identidad/orden de secciones de scanner sigue
  aparte; aqui se acota freshness/scope de fuentes federadas.

Resolucion 2026-05-27: el scanner calcula `BacklogScanDocs`, epoch y reservas
solo con fuentes federadas ejecutables (`vigente`, `promocionada`, `promoted`,
`active`). Las fuentes historicas/quarantine quedan como contexto compacto
`federated_backlog_source_not_executable` y no fuerzan rebase de backlog vivo.

Revalidacion retry 2026-05-27 burst 002: el paquete
`agent-ref-assessment-task-autoprogramming-88a96698afb2-g01-b0935e0be19681ea514f9a2acacaff39`
no abre duplicacion nueva. T250 queda como owner cerrado de freshness/scope del
epoch federado; T44 mantiene contexto `ref_only`, T249 identidad de entradas y
T251 alcance de pruebas de scanners.

## Priorizacion scanner 2026-05-27 decimoctava pasada

Backlog asociado: `T251 backlog-scanner-canonical-preflight`.

Prioridad media-alta:

- Resolver T251 antes de lanzar mas bursts de scanners sobre los tres documentos
  vivos. T140 ya implemento canon/alias para solapes escritos; el scanner debe
  consumir esa informacion antes de proponer un nuevo Txx, o volvera a crear
  tareas equivalentes como T250 sobre una frontera ya cerrada parcialmente.
- El resultado correcto ante solape con tarea cerrada es evidencia de cobertura
  o regresion concreta con causa nueva. Ante solape con pendiente canonica, debe
  crear alias/fusion aditiva o bloquear con reason publico y rebase, sin editar
  historico destructivamente.

Duplicaciones a evitar:

- T251 no sustituye T140: T140 gobierna la canonicalizacion de tareas ya
  escritas; T251 es el preflight que evita escribir otro duplicado antes de
  programarlo.
- T251 no sustituye T248/T249: identidad y orden de secciones de scanner siguen
  con su owner; el preflight usa esa identidad como evidencia, pero no cambia el
  formato de seccion por si solo.
- T251 no sustituye T240: lectura path/budget de documentos sigue separada y
  debe ejecutarse antes de confiar en hashes o lineas de backlog.

Cierre T251 2026-05-27: el planner aplica preflight canonico sobre secciones de
backlog, bloquea solapes con canonicas cerradas mediante
`backlog_scanner_canonical_preflight_required` y entrega a los scanners un
indice compacto con refs, estado, linea y fingerprint. Rework de revision: no
se relanza agente padre; backlog, rail error y esta matriz quedan alineados, con
contexto `ref_only` resuelto por lectura local/evidencia ACK.

## Priorizacion scanner 2026-05-27 decimonovena pasada

Backlog asociado: `T252 backlog-duplicate-task-id-read-model`.

Prioridad alta:

- Resolver T252 antes de cerrar, encolar o deduplicar tareas por numero humano
  `Txx` cuando el backlog ya contiene duplicados. T250/T251 previenen nuevos
  duplicados, pero no bastan para leer historico ambiguo ya escrito.
- El lector debe exponer instancias separadas por path, linea/hash, titulo y
  evidence refs. Si el caller pide `T250` sin instancia canonica, debe bloquear
  con `backlog_duplicate_task_id_ambiguous` o exigir alias/cobertura de director,
  nunca elegir por orden de fichero.

Estado 2026-05-27: T252 cerrado para planner residente. Las secciones duplicadas
exponen `task_instance_ref` y las peticiones por Txx ambiguo bloquean con
`backlog_duplicate_task_id_ambiguous`; aliases/cobertura siguen siendo
reparacion aditiva.
Rework 2026-05-27: entrega valida conservada; el cierre se apoya en lectura
local, evidencia ACK para contexto `ref_only` y test focal requerido.

Duplicaciones a evitar:

- T252 no sustituye T37: el parser Markdown sigue siendo owner de extraer
  estado, criterios, tests y bloques; T252 anade identidad de instancia cuando
  el id humano colisiona.
- T252 no sustituye T250: la reserva/dedupe de ids futuros sigue en T250; T252
  solo trata lectura y cierre de duplicados ya existentes.
- T252 no sustituye T251: el preflight de scanner evita escribir otro Txx
  equivalente; T252 protege APIs, cola y cierre que consumen backlog historico.

## Priorizacion scanner 2026-05-27 decimoseptima pasada

Backlog asociado: `T252 runtime-codex-delivery-progress-source-file-split`.

Estado 2026-05-27: cerrado para el split local del adaptador Codex delivery.
Los tres ficheros productivos objetivo quedaron por debajo de 300 lineas y el
nucleo sigue recibiendo solo observaciones, refs opacas y puertos.

Prioridad cerrada:

- T252 ya separa observacion de progreso, estado durable y fuente de delivery
  en helpers locales del adaptador; `progress_state_v0.go`,
  `progress_source_v0.go` y `source_v0.go` quedan bajo 300 lineas productivas.
- Rework 2026-05-27: el mismo rail de tamano queda aplicado a los tests locales
  grandes del adaptador mediante shards de progress, delivery, review gate,
  worktree y fixtures, todos por debajo de 300 lineas.
- Mantener el split dentro del adaptador Codex delivery. El nucleo debe seguir
  viendo solo observaciones, refs opacas y puertos; filesystem, prompts, HOME,
  proveedor y rutas locales no deben migrar a core ni workflow.

Duplicaciones a evitar:

- T252 no sustituye T36 ni T143: ACK terminal estricto y lectura/redaccion de
  control files ya tienen owners cerrados; esta tarea solo divide codigo grande
  que consume esas politicas.
- T252 no sustituye T38: el rail de observacion Codex ya quedo clasificado; el
  split no debe reabrir listas locales de terminos ni relajar tolerancia a refs
  opacas.
- T252 no sustituye T247: el bridge de autoprogramacion del stack Codex sigue
  en composicion; runtime-codex-delivery solo observa progreso/entrega y no
  decide plan-state, waits ni cierre de cola.

## Priorizacion scanner 2026-05-27 decimoseptima pasada

Backlog asociado: `T250 backlog-task-id-allocation-lease`.

Estado 2026-05-27: cerrado para el servidor residente. El merge lease anade
`task_id_ref`, `reservation-ref-backlog-task-id-*` y rango `Txx` al paquete de
scanner; el lector de ids distingue `## Txx` de `## Escaneo backlog ...` y
publica `backlog_task_number_collision` sin renumerar historico.
Rework 2026-05-27: la revision
`agent-ref-task-ref-review-rework-task-autoprogramming-9ad7b5061f27-g01-d2d05e084cc48aa9566f43b01d7d8f5e`
solo revalida el cierre T252, conserva la entrega valida y resuelve
`ref_only` mediante evidencia ACK sin abrir otro owner ni tocar historico.

Prioridad media:

- Resolver T250 antes de aumentar la concurrencia de scanners que crean tareas
  `Txx` en el backlog vivo. La identidad de seccion de T248/T249 ayuda a
  trazar el escaneo, pero el numero ejecutable de la tarea tambien necesita
  reserva causal para no depender del ultimo encabezado visto o de una
  insercion manual concurrente.
- Preservar historico: huecos o saltos ya persistidos se conservan como
  evidencia; la mejora debe bloquear nuevas colisiones o huecos no reservados,
  no renumerar ni reescribir tareas existentes.

Duplicaciones a evitar:

- T250 no sustituye T37: el parser sigue extrayendo estado, criterios y tests
  desde Markdown.
- T250 no sustituye T43 ni T240: merge lease documental y path/budget de docs
  siguen separados; T250 consume esa epoch para reservar ids ejecutables.
- T250 no sustituye T248/T249: `scan_ref`/`scan_entry_ref` identifican evidencia
  de escaneo; `task_id_ref` o reserva equivalente identifica tareas `Txx`.
- T250 no sustituye T116/T122/T140: precedencia, solapes e integridad de backlog
  siguen con owners vecinos; aqui solo se corrige asignacion y colision de ids
  ejecutables durante inserciones concurrentes.

## Priorizacion scanner 2026-05-27 decimoseptima pasada

Backlog asociado: `T250 backlog-task-overlap-canonical-merge-policy`.

Prioridad media-alta:

- Resolver T250 antes de permitir que scanners concurrentes lancen tareas Txx
  nuevas sin revisar solape conceptual. La evidencia actual muestra T248 y T249
  sobre la misma frontera practica de identidad/orden de entradas de scanner,
  con una nota de fusion manual pero sin contrato ejecutable que evite dos
  trabajos paralelos sobre el mismo owner.
- La salida correcta debe ser aditiva: canonical task ref, aliases o reason code
  publico de rebase/decision. No debe renumerar ni borrar historico del backlog.

Duplicaciones a evitar:

- T250 no sustituye T37: el parser sigue extrayendo tareas, estado, criterios y
  tests desde Markdown.
- T250 no sustituye T43/T240: merge lease y lectura segura de docs siguen
  gobernando epoch, hashes y path/budget.
- T250 no sustituye T248/T249: esas tareas gobiernan identidad de entradas de
  scanner; T250 gobierna solape entre tareas ejecutables antes de lanzarlas.
- T250 no sustituye T140: T140 cubre solapes historicos generales; T250 acota
  la regla a tareas nuevas producidas por scanners concurrentes y a su
  canonicalizacion previa al lanzamiento.

Resolucion 2026-05-27: cerrado en el planner residente. Las tareas pendientes
con misma frontera ejecutable reciben una firma `backlog-task-overlap-*`; el
planner elige canonica por doc/linea/ref, conserva aliases mediante
`covered_by`, fusiona criterios/tests/write-set sin borrar historico y emite
`backlog_task_overlap_canonicalization_required` para rebase/decision cuando una
alias ya esta viva. Evidencia focal:
`go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.

Rework 2026-05-27: la correccion de revision no abre duplicacion nueva. T250
permanece como owner cerrado de canonicalizacion de solapes; esta pasada solo
sincroniza la evidencia documental con el ACK estricto y deja T44 como guarda de
contexto `ref_only`. No se relanza agente padre ni se anaden tareas Txx nuevas.

## Priorizacion scanner 2026-05-27 decimoseptima pasada

Backlog asociado: `T250 backlog-proposal-deduplication-fingerprint`.

Prioridad media:

- Resolver T250 antes de aumentar la paralelizacion de scanners que escriben en
  el backlog vivo. T248/T249 gobiernan identidad de bloques de escaneo, pero no
  impiden que dos propuestas Txx distintas cubran la misma frontera ejecutable
  con titulos o granularidad parecida.
- Usar una huella estructurada de propuesta permite bloquear duplicados antes de
  encolarlos y conservar evidencia de la propuesta descartada para rebase o
  revision humana.

Duplicaciones a evitar:

- T250 no sustituye T37: el parser de Markdown sigue siendo owner de extraer
  tareas, estado, alcance, criterios y tests.
- T250 no sustituye T43/T240: lease, epoch documental y lectura segura de docs
  siguen separados; T250 consume esos datos para comparar propuestas.
- T250 no sustituye T248/T249: identidad y orden de secciones de escaneo siguen
  aparte; aqui se deduplican propuestas ejecutables `Txx`.
- T250 debe respetar T45/T90: una tarea puede quedar como subshard solo si
  declara frontera reducida, criterio no cubierto y motivo causal; si no, debe
  bloquearse como solape.

Resolucion 2026-05-27: cerrado en el planner residente. La huella
`backlog_proposal_fingerprint` se anade a los context refs de cada propuesta
ejecutable; el dedupe compara objetivo, scope y tokens normalizados, registra
`duplicate_backlog_proposal_fingerprint` y deja evidence refs compactas para
rebase/revision humana. Si una de las propuestas solapadas ya esta viva, ambas
refs quedan bloqueadas y el planner devuelve trabajo conocido en cola en vez de
relanzar otro agente padre.

## Priorizacion scanner 2026-05-27 decimonovena pasada

Backlog asociado: `T254 backlog-task-id-collision-alias-index`.

Estado 2026-05-27: cerrado en el planner residente de automejora. Las secciones
homonimas `## Txx` conservan el numero humano como alias visible, pero cada
request publica `task_instance_ref`/`backlog_task_entry_ref`, alias `Txx#NN`,
reason `backlog_duplicate_task_id_ambiguous` y colision
`backlog_task_number_collision` con `instance_refs`.

Prioridad alta:

- Resolver T254 antes de usar un id humano `Txx` como unica clave de cierre,
  encolado, roadmap o ACK. La evidencia actual ya no es solo riesgo futuro:
  varias secciones `## T250` existen en el backlog vivo y representan fronteras
  distintas, por lo que un cierre por numero puede afectar a la tarea equivocada.
- Mantener reparacion aditiva. No renumerar ni borrar historico; crear refs de
  entrada estables, aliases y reason codes publicos para que planner, MCP,
  stats y auditoria puedan distinguir secciones homonimas.

Duplicaciones a evitar:

- T254 no sustituye T37: el parser de Markdown sigue siendo owner de extraer
  estado, criterios y tests; T252 anade identidad resoluble ante colision.
- T254 no sustituye T43/T240: merge lease y lectura segura de docs siguen
  separados; T252 consume fichero/linea/hash como evidencia de entrada.
- T254 no sustituye T248/T249: scan refs e identidad de bloques de evidencia
  siguen aparte; aqui se resuelve la identidad ejecutable de tareas `Txx`.
- T254 no sustituye T250/T251: dedupe/preflight evita nuevos duplicados o
  solapes; T252 repara de forma aditiva colisiones ya persistidas.

## Priorizacion scanner 2026-05-27 vigesima pasada

Backlog asociado: T44, T249, T250, T251 y T252.

Prioridad alta:

- No abrir otro Txx para el mismo patron mientras los owners anteriores sigan
  pendientes y visibles. El scanner debe registrar evidencia de lectura local,
  resolver `required ref_only` en ACK y cerrar la pasada documental sin generar
  una tarea solapada.
- Ejecutar primero T251/T252 antes de aumentar concurrencia de scanners: una
  nueva seccion sin reserva causal o sin preflight canonico agrava la colision
  de ids humanos que ya se esta intentando reparar.

Duplicaciones a evitar:

- No duplicar T44: la guarda de contexto `required ref_only` ya existe; esta
  pasada solo aporta evidencia de que se uso lectura local y nota de ACK.
- No duplicar T249: identidad y orden de entradas de escaneo siguen en ese
  owner.
- No duplicar T250/T251: reserva/dedupe/preflight de propuestas nuevas quedan
  ahi.
- No duplicar T252: read model, alias y reparacion aditiva de ids humanos
  duplicados quedan ahi; no renumerar ni borrar historico.

## Priorizacion scanner 2026-05-27 vigesima primera pasada

Backlog asociado: T44, T249, T250, T251, T252 y T254.

Prioridad alta:

- Tratar las pasadas nuevas que solo reobservan `required_ref_action`,
  write-set documental cerrado y colisiones `Txx` ya registradas como no-op con
  evidencia, no como generadoras de mas backlog. El ACK debe resolver el
  contexto `ref_only` con lectura local o consulta al director y conservar refs
  opacas de worktree/branch sin convertirlas en rutas.
- Ejecutar primero los owners pendientes de dedupe/preflight/read model antes de
  lanzar mas scanners paralelos que escriban en los mismos shards.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` se resuelve en ACK/evidencia.
- No duplicar T249: identidad de bloques de escaneo sigue ahi.
- No duplicar T250/T251: dedupe, preflight y alcance de pruebas siguen ahi.
- No duplicar T252/T254: ids humanos duplicados y aliases aditivos siguen ahi.

## Priorizacion scanner 2026-05-27 vigesimoprimera pasada

Backlog asociado: `T255 opes-bridge-document-plan-contract-file-split`.

Prioridad media:

Estado 2026-05-27: cerrado para el split local del bridge OPES. La frontera
queda en `orquesta-opes-bridge`: contrato documental, politica editorial,
politica/plantilla HTML y metodologia/calidad OPES se separan sin mover reglas a
core, workflow, director ni `orquesta-domain-work`.

- Mantener T255 como cierre aplicado antes de anadir mas politica editorial,
  HTML o tipos documentales OPES al bridge. La evidencia original mostro
  `document_plan_contract_v0.go` por encima de 300 lineas y mezclando varias
  responsabilidades de dominio; el cierre local ya separa esos owners.
- Mantener la division dentro del adaptador OPES. La mejora debe reducir
  tamano/ownership local y conservar los campos publicos existentes; no debe
  mover reglas editoriales a `orquesta-domain-work`, core, workflow ni director.

Duplicaciones a evitar:

- T255 no sustituye T204: presupuesto/ventana de `topic_blocks` sigue en su
  owner.
- T255 no sustituye T205: schema y normalizacion de payload por `job_type`
  siguen separados.
- T255 no sustituye T206: readback y receipt causal OPES siguen separados.
- T255 no sustituye T244 ni T253: esos splits pertenecen al stack Codex y al
  provider de replan por quality gate, no al bridge OPES.

## Priorizacion scanner 2026-05-27 vigesima segunda pasada

Backlog asociado: `T256 server-shutdown-usecase-file-split` como owner
canonico; la variante `server-shutdown-usecase-file-split-before-growth` queda
fusionable y no debe programarse por separado.

Prioridad media:

- Resolver T256 antes de anadir nuevos modos de apagado, reason codes,
  reintentos o integracion de checkpoint al caso de uso residente. La evidencia
  actual muestra `shutdown_v0.go` con 319 lineas y varias responsabilidades
  mezcladas en una frontera que debe seguir siendo hexagonal.
- Mantener el split dentro de `orquesta-server-shutdown`: el modulo coordina
  por puertos `RunControl`, `RunQueue`, `RunSupervisor` y checkpoints, pero no
  debe matar procesos ni importar runtime concreto, web, MCP, DB o `cmd`.

Estado 2026-05-27: T256 queda resuelto dentro de `orquesta-server-shutdown`.
`shutdown_v0.go` conserva la fachada publica y el detalle local queda separado
en autorizacion, control, targets/checkpoint y supervision; no se abre la
variante `server-shutdown-usecase-file-split-before-growth` como tarea
independiente.
Rework 2026-05-27: la entrega
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-846263ad27220f7ca7d97aeec4d5e7d1`
mantiene esa fusion, conserva T256 como owner canonico y cierra el contexto
`ref_only` con evidencia ACK, sin tocar owners vecinos.
Rework adicional 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-40ea96487c3783404f7d6a48b0cb8411`
confirma la misma fusion y evita relanzar otra tarea para el alias
`server-shutdown-usecase-file-split-before-growth`.
Rework final 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-247e6d495674dee5a1b996468956a6be`
mantiene T256 como owner canonico, conserva la absorcion del alias
`server-shutdown-usecase-file-split-before-growth` y deja el contexto
`ref_only` cerrado por evidencia ACK sin crear owner nuevo.
Rework correctivo 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-ced6b146bcf5b10e5253ef166816de66`
revalida el cierre sin relanzar agente padre, mantiene la variante
`server-shutdown-usecase-file-split-before-growth` absorbida por T256 y conserva
el contexto `ref_only` resuelto en ACK.
Rework de reemplazo 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-75ba20e2214f2eaaea16f312c68208f0`
mantiene la fusion con T256 canonico, evita crear owner nuevo para el alias
`server-shutdown-usecase-file-split-before-growth` y deja el contexto
`ref_only` resuelto por evidencia ACK.
Rework de evaluacion 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-f878e21cfa5e67e7481d20a5f9bd24c5`
conserva la misma fusion, no crea owner nuevo para el alias
`server-shutdown-usecase-file-split-before-growth` y resuelve el contexto
`ref_only` por lectura local y evidencia ACK.
Rework de evaluacion 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-3d04cc0bcd1f59b871869ec824d2c996`
conserva la fusion con T256 canonico, no crea owner nuevo para el alias
`server-shutdown-usecase-file-split-before-growth` y resuelve el contexto
`ref_only` por lectura local y evidencia ACK.
Rework de evaluacion 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-46d0d95f62343cc1e51b9ca398f83f84`
conserva la fusion con T256 canonico, no crea owner nuevo para el alias
`server-shutdown-usecase-file-split-before-growth` y resuelve el contexto
`ref_only` por lectura local y evidencia ACK.
Rework de reemplazo 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-b6686ba67b0d901948d8ce067d8771ec`
conserva la fusion con T256 canonico, no crea owner nuevo para el alias
`server-shutdown-usecase-file-split-before-growth` y resuelve el contexto
`ref_only` por lectura local y evidencia ACK.
Rework de evaluacion 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-bff7aa189e7e24662071cf55dffcb95f`
conserva la fusion con T256 canonico, no crea owner nuevo para el alias
`server-shutdown-usecase-file-split-before-growth` y resuelve el contexto
`ref_only` por lectura local y evidencia ACK.
Rework de reemplazo 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-55de335961e2cd74deaeda6d56c611bb`
conserva la fusion con T256 canonico, no crea owner nuevo para el alias
`server-shutdown-usecase-file-split-before-growth` y resuelve el contexto
`ref_only` por lectura local y evidencia ACK.
Rework de evaluacion 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-c853a308c21a89806299b2db6b75edfe`
conserva la fusion con T256 canonico, no crea owner nuevo para el alias
`server-shutdown-usecase-file-split-before-growth` y resuelve el contexto
`ref_only` por lectura local y evidencia ACK. La revalidacion no puede cerrar
la prueba obligatoria porque el build de `cmd/orquesta-server` falla en
`modulos/orquesta-app-director-service`, fuera del write-set.
Rework de reemplazo 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-3cf9f50f2a1ee0b94df79560b6eeca53`
conserva la fusion con T256 canonico, no crea owner nuevo para el alias
`server-shutdown-usecase-file-split-before-growth` y resuelve el contexto
`ref_only` por lectura local y evidencia ACK.
Rework de evaluacion 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-a0fa2702f7f819236261be3abc537d86`
conserva la fusion con T256 canonico, no crea owner nuevo para el alias
`server-shutdown-usecase-file-split-before-growth` y resuelve el contexto
`ref_only` por lectura local, evidencia ACK y prueba obligatoria pasada.
Rework de reemplazo 2026-05-27:
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-autoprogr-2e9-81707decc46db22fc599c9343d59d3cb`
mantiene la fusion con T256 canonico, no crea owner nuevo para el alias
`server-shutdown-usecase-file-split-before-growth` y resuelve el contexto
`ref_only` por lectura local, evidencia ACK y prueba obligatoria pasada.
Revalidacion OrquestaV2 2026-06-11:
`agent-ref-assessment-task-autoprogramming-fe9cf6f32e89-g01-d18f387e937eb820ba9d7cf0b788b9a3`
mantiene T256 como owner canonico, conserva la absorcion del alias
`server-shutdown-usecase-file-split-before-growth`, separa la validacion de
dependencias de la fachada `ShutdownServerV0` y resuelve el contexto
obligatorio `ref_only` mediante lectura local del paquete y fuentes vigentes.
Revalidacion OrquestaV2 2026-06-11 retry b1c7:
`agent-ref-task-autoprogramming-f02e03774215-g01` conserva la fusion con T256
canonico, no abre owner nuevo para el alias `before-growth` y anade cobertura
del parser para estados fechados (`Estado 2026-..:`/`Cierre local 2026-..:`),
evitando que el scanner trate notas cerradas como secciones pendientes.

Duplicaciones a evitar:

- T256 no sustituye T30: el puerto neutral de checkpoint de shutdown sigue en
  su owner.
- T256 no sustituye T226 ni T239: intentos/idempotencia y correlacion de ACKs
  stale siguen como rails de Codex/guardian/stack.
- T256 no sustituye T52/T54/T90/T242: esos splits pertenecen a
  app-director-service, stack Codex, residuo general y supervisor residente.
- T256 no sustituye T213/T227: redaccion/budget de salida publica y detalle
  runtime siguen separados; aqui solo se reduce tamano y ownership local del
  caso de uso de apagado.

## Priorizacion scanner 2026-05-27 vigesimosegunda pasada

Backlog asociado: sin Txx nuevo; owners existentes T44, T249, T250, T251,
T252, T254, T255, T227 y T241.

Prioridad alta:

- Evitar que scanners documentales equivalentes vuelvan a crear tareas cuando
  solo reobservan `required_ref_action=ack_evidence_required`, pruebas globales
  para write-set documental, colisiones `Txx` historicas, control files/ACK o
  `provider_auth_blocked` ya cubiertos por owners pendientes.
- Cerrar estas pasadas por evidencia local y ACK explicito de contexto
  `ref_only`, conservando `worktree_ref` y `branch_ref` como refs opacas.

Duplicaciones a evitar:

- No duplicar T44 para contexto `ref_only`.
- No duplicar T249/T250/T251/T252/T254 para identidad, dedupe, preflight,
  reserva y read model de backlog.
- No duplicar T143/T155/T157/T227 para ACK/control files/detalle runtime.
- No duplicar T241 para recuperacion por `provider_auth_blocked`.
- No duplicar T244/T253/T255 para splits por presupuesto de ficheros.

## Priorizacion scanner 2026-05-27 vigesima segunda pasada

Backlog asociado: T44, T250, T251, T252, T254 y T255.

Prioridad alta:

- No abrir otra tarea de scanner para el mismo paquete documental mientras T250,
  T251, T252 y T254 sigan pendientes: la colision de ids humanos y el preflight
  canonico ya tienen owner.
- No abrir otra tarea de split OPES por el hueco ya cerrado en T255: el alcance,
  criterios y test focal del bridge quedan documentados y verificados.

Duplicaciones a evitar:

- No duplicar T44: el contexto `ref_only` se resuelve en ACK/evidencia, no con
  otro owner.
- No duplicar T250/T251/T252/T254: dedupe, scope de pruebas, read model,
  alias y reserva de ids `Txx` siguen en esos owners.
- No duplicar T255: la division de politica editorial/HTML/contrato OPES
  pertenece al bridge OPES y no debe mezclarse con T204, T205, T206, T244 ni
  T253.

## Priorizacion scanner 2026-05-27 vigesima segunda pasada

Backlog asociado: T44, T249, T250, T251, T252, T254 y T255.

Prioridad alta:

- Cerrar las pasadas documentales repetidas como evidencia de cobertura cuando
  el scanner solo reobserva contexto `ref_only`, write-set de shards y huecos
  ya visibles. No abrir un Txx nuevo hasta que T249/T250/T251/T252/T254
  resuelvan identidad, dedupe, preflight y colisiones de ids humanos.
- Mantener T255 como owner separado del split OPES ya detectado. Las pasadas
  posteriores no deben duplicarlo salvo que aporten una frontera distinta con
  evidencia propia.

Duplicaciones a evitar:

- No duplicar T44: el contexto `ref_only` se resuelve en ACK con lectura local o
  consulta al director.
- No duplicar T249/T250/T251: identidad de entradas, dedupe de propuestas,
  preflight canonico y alcance de tests siguen en esos owners.
- No duplicar T252/T254: la reparacion de ids `Txx` ambiguos debe ser aditiva,
  con aliases/read model, sin renumerar historico.
- No duplicar T255: el split de `orquesta-opes-bridge` es un owner local de
  tamano/ownership, no una razon para abrir otro scanner programable.

## Priorizacion scanner 2026-05-27 vigesima segunda pasada

Backlog asociado: T44, T249, T250, T251, T252, T254 y T255.

Prioridad alta:

- Tratar las nuevas pasadas con el mismo paquete `ref_only`, write-set
  documental cerrado y owners ya visibles como no-op con evidencia. No abrir un
  nuevo Txx solo para confirmar T44/T249/T250/T251/T252/T254/T255.
- Mantener `worktree_ref` y `branch_ref` como refs opacas en la entrega; no
  convertirlas en rutas ni nombres Git.
- Resolver el contexto obligatorio por lectura local o consulta al director, y
  reflejarlo en ACK con `contexto_ref_only_resuelto`.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` se resuelve en ACK/evidencia.
- No duplicar T249: identidad y orden de bloques de escaneo siguen ahi.
- No duplicar T250/T251: reserva, dedupe, preflight y alcance de pruebas siguen
  ahi.
- No duplicar T252/T254: ids humanos duplicados, alias/read model y reparacion
  aditiva siguen ahi.
- No duplicar T255: la division del contrato documental OPES ya tiene owner
  local en `orquesta-opes-bridge`.

## Priorizacion scanner 2026-05-27 vigesimosegunda pasada

Backlog asociado: `T256 server-shutdown-usecase-file-split`.

## Priorizacion scanner 2026-05-27 vigesima tercera pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255 y T256.

Prioridad alta:

- Tratar las nuevas pasadas con el mismo paquete `ref_only`, write-set
  documental cerrado y owners ya visibles como no-op con evidencia. No abrir un
  nuevo Txx solo para confirmar T44/T249/T250/T251/T252/T254/T255/T256.
- Mantener `worktree_ref` y `branch_ref` como refs opacas en la entrega; no
  convertirlas en rutas ni nombres Git.
- Resolver el contexto obligatorio por lectura local o consulta al director y
  reflejarlo en ACK con `contexto_ref_only_resuelto`.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` se resuelve en ACK/evidencia.
- No duplicar T249: identidad y orden de bloques de escaneo siguen ahi.
- No duplicar T250/T251: reserva, dedupe, preflight y alcance de pruebas siguen
  ahi.
- No duplicar T252/T254: ids humanos duplicados, alias/read model y reparacion
  aditiva siguen ahi.
- No duplicar T255/T256: los splits locales de OPES bridge y shutdown ya tienen
  owners propios; no crear otro scanner programable para esas mismas fronteras.

Prioridad media:

- No relanzar T256 como tarea nueva: el cierre local ya deja `shutdown_v0.go`
  como fachada menor de 300 lineas y reparte detalle en owners locales de
  autorizacion, control, checkpoint, targets, supervision y resumen.
- La revision/rework final del 2026-05-27 solo sincroniza evidencia documental
  y ACK; nuevos cambios de shutdown deben usar T256 canonico o un owner vecino
  real, no el alias `before-growth`.
- La correccion
  `agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-ced6b146bcf5b10e5253ef166816de66`
  mantiene esa regla: evidencia documental y ACK, sin nuevo owner programable.
- La correccion
  `agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-75ba20e2214f2eaaea16f312c68208f0`
  revalida la misma absorcion de `before-growth`: no hay nueva tarea
  programable ni cambio de contrato de shutdown.
- La correccion
  `agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-f878e21cfa5e67e7481d20a5f9bd24c5`
  mantiene la absorcion de `before-growth`, sin nuevo owner programable ni
  cambio de contrato de shutdown.
- La correccion
  `agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-3d04cc0bcd1f59b871869ec824d2c996`
  mantiene la misma absorcion de `before-growth`, sin nuevo owner programable
  ni cambio de contrato de shutdown.
- La correccion
  `agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-46d0d95f62343cc1e51b9ca398f83f84`
  mantiene la misma absorcion de `before-growth`, sin nuevo owner programable
  ni cambio de contrato de shutdown.
- La correccion
  `agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-33ca4ea060412afa902e83e3765c1d23`
  mantiene la misma absorcion de `before-growth`, sin nuevo owner programable
  ni cambio de contrato de shutdown.
- La correccion
  `agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-b6686ba67b0d901948d8ce067d8771ec`
  mantiene la misma absorcion de `before-growth`, sin nuevo owner programable
  ni cambio de contrato de shutdown.
- La correccion
  `agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-bff7aa189e7e24662071cf55dffcb95f`
  mantiene la misma absorcion de `before-growth`, sin nuevo owner programable
  ni cambio de contrato de shutdown.
- La correccion
  `agent-ref-assessment-task-ref-review-rework-task-autoprogramming-68fe872ff80c-g01-864-c853a308c21a89806299b2db6b75edfe`
  mantiene la misma absorcion de `before-growth`, sin nuevo owner programable
  ni cambio de contrato de shutdown; el fallo de build observado pertenece a
  `modulos/orquesta-app-director-service`, fuera del write-set cerrado.
- Mantener nuevos cambios de apagado dentro de `orquesta-server-shutdown` y
  `cmd/orquesta-server` solo para wiring/pruebas de composicion. El modulo
  sigue siendo hexagonal: no mata procesos, no conoce runtime real, no importa
  Codex/MCP/web/DB/filesystem y no expone HOME, tokens ni rutas.

Duplicaciones a evitar:

- T256 no sustituye T30: checkpoint cooperativo neutral y ACK durable siguen en
  su owner.
- T256 no sustituye T225: escalado forzado del guardian sigue cerrado en su
  owner.
- T256 no sustituye T239: correlacion de intento de checkpoint Codex sigue
  separada.
- T256 no sustituye T199 ni T217: catalogo publico de errores y politica de
  senales/shutdown del servidor siguen como rails vecinos.

## Priorizacion scanner 2026-05-27 vigesima segunda pasada

Backlog asociado: T44, T249, T250, T251, T252, T254 y T255.

Prioridad alta:

- Tratar nuevas pasadas equivalentes del scanner documental como evidencia de
  cobertura, no como generadoras de mas `Txx`, cuando el paquete solo aporta
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog y huecos ya visibles en owners pendientes.
- Resolver primero T250/T251/T252/T254 antes de aumentar concurrencia de
  scanners que escriben ids humanos o fusionan propuestas; resolver T255 antes
  de crecer el contrato documental OPES.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` se resuelve por lectura local/evidencia
  en ACK.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos duplicados y aliases aditivos siguen ahi.
- No duplicar T255: el split OPES pertenece al bridge OPES y no reabre reglas
  de nucleo ni owners OPES previos.

## Priorizacion scanner 2026-05-27 vigesima segunda pasada

Backlog asociado: T44, T249, T250, T251, T252, T254 y T255.

Prioridad alta:

- Tratar nuevas pasadas documentales equivalentes como evidencia de cobertura,
  no como backlog programable nuevo, mientras los owners anteriores sigan
  pendientes y visibles en los shards. El ACK debe declarar la resolucion del
  contexto `ref_only` y las pruebas obligatorias ejecutadas.
- No aumentar colisiones `Txx` ni duplicar splits por presupuesto de fichero:
  T253/T255 ya tienen owners locales y T250/T251/T252/T254 gobiernan identidad,
  preflight, read model y aliases.

Duplicaciones a evitar:

- No duplicar T44: esta pasada resuelve el contexto por lectura local/evidencia,
  sin crear otra guarda.
- No duplicar T249/T250/T251: identidad de escaneo, dedupe, reserva y preflight
  siguen en esos owners.
- No duplicar T252/T254: ids humanos duplicados y reparacion aditiva siguen ahi.
- No duplicar T253/T255: los splits de ficheros grandes ya tienen owner local.

## Priorizacion scanner 2026-05-27 vigesima cuarta pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256 y T257.

Prioridad alta:

- Ejecutar T257 antes de anadir mas reglas al indice federado de backlog: la
  separacion debe dejar una frontera clara entre catalogo canonico, parsing de
  indice, parsing de entradas locales, cuarentena documental y refs de scanner.
- Mantener T249/T250/T251/T252/T254 como owners de identidad, dedupe, preflight,
  alcance de pruebas e ids humanos; T257 no debe absorber esas politicas.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` se resuelve por lectura local/evidencia
  en ACK.
- No duplicar T249/T250/T251/T252/T254: el split federado debe conservar la
  semantica existente y delegar esos rails a sus owners.
- No duplicar T255/T256: los splits OPES bridge y server-shutdown siguen en sus
  modulos; T257 solo cubre `cmd/orquesta-server` para backlog federado.

Cierre 2026-05-27: T257 queda cerrado en `cmd/orquesta-server` dividiendo
carga federada, parser de indice, parser local y clasificacion de estado; los
owners T249/T250/T251/T252/T254 siguen gobernando identidad, dedupe, preflight,
alcance de pruebas e ids humanos.

Revalidacion OrquestaV2 2026-05-27: el paquete
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-ec8e0c187f3a9ac12eb4a2a1b1343950`
confirma que T257 sigue cerrado; resolver `ref_only` por lectura local y ACK,
ejecutar la prueba focal del servidor y no duplicar owners ni politicas de
T249/T250/T251/T252/T254/T255/T256/T258.

Revalidacion OrquestaV2 burst 002 2026-05-27: el paquete
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-a2fd4b485cb6c9eb3e118699b8c263e8`
solo reobserva T257; mantener cierre existente, resolver `ref_only` con lectura
local/evidencia en ACK y no duplicar owners de identidad, dedupe, preflight,
ids humanos ni splits recientes.

Revalidacion OrquestaV2 burst 003 2026-05-27: el paquete
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-6e9bb2925e8315f5a8e8a4458c5dafec`
solo reobserva T257; no abrir owner nuevo ni duplicar T249/T250/T251/T252/T254,
T255/T256/T258. El ACK debe declarar contexto `ref_only` resuelto y bloqueo de
la prueba obligatoria por compilacion fuera del write-set si persiste.

Revalidacion OrquestaV2 burst 002 adicional 2026-05-27: el paquete
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-d42ec6b2c4feefb41c1a8bd1217aba3a`
solo reobserva T257; mantener el cierre existente, resolver `ref_only` por
lectura local/evidencia en ACK y no duplicar owners de identidad, dedupe,
preflight, ids humanos ni splits recientes. La prueba obligatoria queda
bloqueada por compilacion de `modulos/orquesta-app-director-service` fuera del
write-set.

Revalidacion OrquestaV2 burst 002 adicional 6dc6 2026-05-27: el paquete
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-6dc6bdca36a88bfa03d0be4e0ef98dbd`
confirma que el paquete solo reobserva T257 ya cerrado; mantener split vigente,
resolver `ref_only` por lectura local/evidencia ACK y no duplicar owners de
identidad, dedupe, preflight, ids humanos ni splits recientes.

Revalidacion OrquestaV2 burst 002 adicional 3519 2026-05-27: el paquete
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-3519b447ee0e247812f0a2de19e107ff`
solo reobserva T257; conservar el cierre existente, resolver `ref_only` por
lectura local/evidencia ACK y mantener `worktree_ref`/`branch_ref` como refs
opacas sin abrir owner nuevo.

Revalidacion OrquestaV2 burst 003 adicional 688572 2026-05-27: el paquete
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-688572c0228e7735f6c5a3d0b137d96f`
solo reobserva T257; mantener el cierre existente de carga federada, parser de
indice, parser local y clasificacion de estado, resolver `ref_only` por lectura
local/evidencia en ACK, ejecutar la prueba focal del servidor y no duplicar
T44/T249/T250/T251/T252/T254/T255/T256/T258.

Revalidacion OrquestaV2 burst 002 adicional b744 2026-05-27: el paquete
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-b744e45b4bc69e90339caae12c59ddac`
solo reobserva T257; mantener el cierre existente, resolver `ref_only` por
lectura local/evidencia en ACK, conservar `worktree_ref`/`branch_ref` como refs
opacas y no duplicar T44/T249/T250/T251/T252/T254/T255/T256/T258.

Revalidacion OrquestaV2 burst 003 adicional e3e841 2026-05-27: el paquete
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-e3e8416382f03b021e17fad4f48672c6`
solo reobserva T257; mantener el cierre existente de carga federada, parser de
indice, parser local, clasificacion de estado y refs de contexto, resolver
`ref_only` por lectura local/evidencia en ACK, conservar `worktree_ref` y
`branch_ref` como refs opacas y no duplicar T44/T249/T250/T251/T252/T254/T255/T256/T258.

Revalidacion OrquestaV2 burst 002 adicional 6ad2 2026-05-27: el paquete
`agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-6ad2ce2a420fdd4406624de54709433a`
solo reobserva T257; mantener el cierre existente de carga federada, parser de
indice, parser local, clasificacion de estado y refs de contexto, resolver
`ref_only` por lectura local/evidencia ACK, conservar `worktree_ref` y
`branch_ref` como refs opacas y no duplicar T44/T249/T250/T251/T252/T254/T255/T256/T258.

## Priorizacion scanner 2026-05-27 vigesima segunda pasada

Backlog asociado: T44, T249, T250, T251, T252, T254 y T255.

Prioridad alta:

- Cerrar las nuevas pasadas del scanner que solo reobservan `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set documental cerrado y
  owners pendientes visibles como no-op con evidencia. No generar otro Txx
  mientras no aparezca un hueco causal distinto.
- Mantener la prueba global `go test -count=1 ./...` como validacion requerida
  del paquete, pero no usarla para abrir una politica nueva si T251 ya cubre el
  alcance de pruebas de scanners documentales.

Duplicaciones a evitar:

- No duplicar T44: el contexto requerido por ref se resuelve en ACK con lectura
  local/evidencia explicita.
- No duplicar T249/T250/T251: identidad de escaneo, dedupe, preflight y alcance
  de pruebas siguen en esos owners.
- No duplicar T252/T254: ids humanos duplicados, read model y aliases aditivos
  siguen en esos owners.
- No duplicar T255: el split OPES bridge ya esta abierto; esta pasada no anade
  otro owner para el mismo fichero ni cambia la frontera OPES como consumidor.

## Priorizacion scanner 2026-05-27 vigesima tercera pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255 y T256.

Prioridad alta:

- Tratar la nueva pasada del scanner documental
  `agent-ref-assessment-task-autoprogramming-721523601cc1-g01-520f979fdde9866f084ddfa1d5cc7425`
  como no-op cubierto: solo reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a los tres
  shards y owners pendientes ya visibles.
- Resolver el contexto requerido por lectura local/evidencia en ACK, conservar
  `worktree_ref` y `branch_ref` como refs opacas y no crear otro `Txx` mientras
  no aparezca un hueco causal distinto.

Duplicaciones a evitar:

- No duplicar T44: la guarda `ref_only` se resuelve en ACK/evidencia.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen ahi.
- No duplicar T252/T254: ids humanos duplicados, read model y aliases aditivos
  siguen en esos owners.
- No duplicar T255/T256: los splits concretos de OPES bridge y shutdown ya
  tienen owners locales; este scanner no anade frontera nueva.

## Priorizacion scanner 2026-05-27 vigesima tercera pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255 y T256.

Prioridad alta:

- Cerrar como no-op cubierto las pasadas documentales que solo reobservan
  `ref_only`, `required_ref_action=ack_evidence_required`, write-set cerrado a
  shards de backlog y owners pendientes ya visibles. El ACK debe resolver el
  contexto por lectura local/evidencia y no convertir refs opacas de worktree o
  branch en rutas ni nombres Git.
- No abrir mas splits por presupuesto de fichero desde este scanner mientras
  T253, T255 y T256 sigan pendientes y cubran los ficheros concretos detectados.
  Cualquier nuevo fichero debe llegar con medicion, owner local y ausencia de
  solape contra esos Txx.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad de escaneo, dedupe, preflight y alcance
  de pruebas de scanners siguen en esos owners.
- No duplicar T252/T254: lectura ambigua de ids humanos y aliases aditivos
  siguen en esos owners.
- No duplicar T255/T256: OPES bridge y server-shutdown ya tienen owners locales
  para split por responsabilidad; esta pasada solo conserva evidencia de
  cobertura.

## Scanner 2026-05-27: conector de proceso runtime por dividir

Backlog asociado: `T256 runtime-process-connector-file-split`.

Evidencia:

- `modulos/orquesta-runtime/process_runtime_connector_v0.go` tiene 313 lineas y
  concentra launch, adopcion, wait interno y stop de proceso.
- `modulos/orquesta-runtime/process_runtime_connector_types_v0.go` tiene 318
  lineas y concentra DTOs, errores, validacion de request/snapshot, guardas de
  env, shell y rutas.

Estado 2026-05-27: T256 queda resuelto dentro de `orquesta-runtime`.
`ProcessRuntimeConnectorV0` conserva contrato publico y refs opacas, mientras
launch validation, env policy, value guards, adopcion, estado interno y watchers
quedan en owners locales bajo 300 lineas. El cierre no reabre
`NEUTRAL-PROCESS-STOP-E2E`, T65 ni T66.

Rework de revision 2026-05-27: el paquete
`agent-ref-task-ref-review-rework-task-autoprogramming-d1516ddebab4-g01-736b5902b9f28a1666a612eccb83a9c2`
mantiene T256 como owner canonico, no crea duplicacion nueva y cierra el
contexto `ref_only` mediante lectura local/evidencia ACK.

Reglas:

- Resolver T256 antes de anadir mas politica de proceso local, escalado de kill,
  receipts o redaccion de snapshots al runtime neutral.
- Mantener la separacion conceptual: `orquesta-runtime` ejecuta ordenes y
  devuelve evidencia; no decide plan, proveedor, Codex, OPES, DB, HOME ni UI.
- T256 no sustituye `NEUTRAL-PROCESS-STOP-E2E`: el smoke/proceso temporal ya
  valida comportamiento vivo y debe seguir pasando.
- T256 no sustituye T65/T66: composicion residente, restart y smoke neutral de
  proceso real siguen en sus owners.

## Priorizacion scanner 2026-05-27 vigesima cuarta pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255 y T256.

Prioridad alta:

- Cerrar como no-op cubierto las pasadas documentales que solo reobservan
  `ref_only`, `required_ref_action=ack_evidence_required`, write-set cerrado a
  shards de backlog, prueba global obligatoria y owners pendientes visibles.
  El ACK debe resolver el contexto por lectura local/evidencia y conservar
  refs de worktree o branch como opacas.
- No abrir mas Txx desde este scanner mientras no aparezca una frontera causal
  nueva con owner local, evidencia concreta y no solapada contra T44/T249/T250/
  T251/T252/T254/T255/T256.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad de escaneo, dedupe, preflight y alcance
  de pruebas de scanners siguen en esos owners.
- No duplicar T252/T254: lectura ambigua de ids humanos y aliases aditivos
  siguen en esos owners.
- No duplicar T255/T256: OPES bridge, server-shutdown y runtime-process ya
  tienen owners locales para splits por responsabilidad.

## Scanner 2026-05-27: suite historica app-director-service por dividir

Backlog asociado: `T258 app-director-service-test-suite-file-split`.

Evidencia:

- T52 cerro el split de ficheros productivos de
  `modulos/orquesta-app-director-service`, pero dejo baseline historico de
  tests grandes con regla de no crecimiento.
- `operational_director_v0_test.go`, `operational_closure_v0_test.go`,
  `operational_director_full_statefile_replay_v0_test.go`,
  `director_decision_source_v0_test.go` y tests vecinos siguen por encima del
  limite operativo de 300 lineas.

Reglas:

- Resolver T258 antes de anadir mas escenarios de Director Operativo, cierre
  causal, replay statefile, domain work o replan negativo a esos ficheros
  historicos.
- No duplicar T52: T258 solo reparte tests y helpers de test, no reabre codigo
  productivo ni contratos publicos.
- No duplicar T53/T253: materializador/cierre de orchestration-core y replan
  por quality gate tienen owners propios.

## Priorizacion scanner 2026-05-27 vigesima quinta pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-350b93e475a8-g01-a9fca45e81d848b2c2a00b8349e4817b`
  como no-op cubierto: solo reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Mantener el ACK como evidencia de resolucion de contexto por lectura local;
  no convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.
- No abrir mas `Txx` desde retries equivalentes mientras no aparezca una
  frontera causal nueva con medicion, owner local y ausencia de solape contra
  T44/T249/T250/T251/T252/T254/T255/T256/T257/T258.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad de escaneo, dedupe, preflight y alcance
  de pruebas de scanners siguen en esos owners.
- No duplicar T252/T254: lectura ambigua de ids humanos, reserva, colisiones y
  aliases aditivos siguen ahi.
- No duplicar T255/T256/T257/T258: los splits concretos de OPES bridge,
  server-shutdown, backlog federado y suite historica app-director-service ya
  tienen owners locales.

## Cierre T258 2026-05-27

Backlog asociado: `T258 app-director-service-test-suite-file-split`.

Resultado:

- La suite historica de `modulos/orquesta-app-director-service` queda dividida
  por escenarios de Director Operativo, cierre causal, replay statefile, tests
  requeridos, domain work y replan negativo.
- No abrir otro owner para este mismo split salvo regresion medida: los casos
  publicos siguen trazables por nombre de test y la verificacion focal es
  `go test -count=1 ./modulos/orquesta-app-director-service`.
- Los refs `worktree_ref` y `branch_ref` de autoprogramacion se conservan como
  refs opacas; no son rutas ni nombres Git.
- Rework de revision 2026-05-27: T258 queda confirmado como cierre del rail de
  tamano para la suite local; no se relanza agente padre ni se duplica T52/T53.

## Priorizacion scanner 2026-05-27 retry d64a6a tercera pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-757fa45b2845-g01-9a0012e071185631d9f7652eb10ef3bb`
  como no-op cubierto: solo reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No abrir mas `Txx` desde scanners equivalentes mientras no aparezca una
  frontera causal nueva con medicion, owner local y ausencia de solape contra
  T44/T249/T250/T251/T252/T254/T255/T256/T257/T258.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad de escaneo, dedupe, preflight y alcance
  de pruebas de scanners siguen en esos owners.
- No duplicar T252/T254: lectura ambigua de ids humanos, reserva, colisiones y
  aliases aditivos siguen ahi.
- No duplicar T255/T256/T257/T258: los splits concretos de OPES bridge,
  server-shutdown, backlog federado y suite historica app-director-service ya
  tienen owners locales.

## Priorizacion scanner 2026-05-27 vigesima septima pasada

Backlog asociado: T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y
T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-721523601cc1-g01-6b02071be67727512a14341842bf4332`
  como no-op cubierto: repite `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local/evidencia en ACK, preservar
  `worktree_ref` y `branch_ref` como refs opacas y no abrir otro `Txx` salvo
  frontera causal nueva con medicion, owner local y ausencia de solape.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos, colisiones, aliases y read model siguen
  en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya
  tienen owner local y este scanner no aporta otro fichero ni frontera.

## Priorizacion scanner 2026-05-27 retry 7a5673

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-3ddbc1385ac2-g01-00ef1096db1ef4935750f733b3fe3ccf`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya
  tienen owner local y test asociado.

## Rework T257 2026-05-27 reemplazo 5a8806

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el rework documental
  `agent-ref-assessment-task-ref-review-rework-task-autoprogramming-c4dae25a0488-g01-385-5a8806ef75f49aae3734e7a512136f48`
  como correccion acotada de T257: el paquete solo exige conservar la entrega
  valida, resolver `ref_only` por lectura local/evidencia ACK y ejecutar la
  prueba focal del servidor.
- Mantener T257 como owner canonico de `federated-backlog-source-file-split`;
  no relanzar agente padre ni abrir otro `Txx` mientras no aparezca frontera
  causal nueva con fichero, medicion y ausencia de solape.
- Registrar en ACK `contexto_ref_only_resuelto` y no convertir refs de runtime
  en rutas de producto ni nombres Git.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto en ACK/evidencia.
- No duplicar T249/T250/T251: identidad, dedupe/preflight y alcance de pruebas
  siguen en esos owners.
- No duplicar T252/T254: ids humanos, aliases, colisiones y read model siguen
  en esos owners.
- No duplicar T255/T256/T258: los otros splits recientes ya tienen owners
  locales; este rework solo revalida T257.

## Priorizacion scanner 2026-05-27 retry 5465c1

Backlog asociado: T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y
T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-87d5c800f528-g01-d77d5415386c2758420eb50d9de1c982`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local/evidencia en ACK, preservar
  `worktree_ref` y `branch_ref` como refs opacas y no abrir otro `Txx` salvo
  frontera causal nueva con medicion, owner local y ausencia de solape.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos, reserva, colisiones, aliases y read model
  siguen ahi.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya
  tienen owner local y este scanner no aporta otro fichero ni frontera.

## Priorizacion scanner 2026-05-27 retry d64a6a

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-757fa45b2845-g01-5ab69cd762c0b3eea5d2721713dd49d0`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad de escaneo, dedupe, preflight y alcance
  de pruebas de scanners siguen en esos owners.
- No duplicar T252/T254: lectura ambigua de ids humanos, reserva, colisiones y
  aliases aditivos siguen ahi.
- No duplicar T255/T256/T257/T258: los splits concretos de OPES bridge,
  server-shutdown, backlog federado y suite historica app-director-service ya
  tienen owners locales.

## Priorizacion scanner 2026-05-27 vigesima octava pasada

Backlog asociado: T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y
T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-350b93e475a8-g01-9be0de2c97caf766173a451288993ca0`
  como no-op cubierto: solo reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya
  tienen owner local y test asociado.

## Priorizacion scanner 2026-05-27 vigesima septima pasada retry 69e1ba

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-8f23d8e2a22e-g01-957608057af975d94befe452b621687e`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- Conservar `worktree_ref` y `branch_ref` como refs opacas; no convertirlas en
  rutas ni nombres Git.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  de ACK en esta pasada.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos, reserva, colisiones, aliases y read model
  siguen ahi.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-06-11 T256 burst 002

Backlog asociado: `T256 server-shutdown-usecase-file-split`.

Prioridad alta:

- Cerrar el paquete `agent-ref-task-autoprogramming-7a851a71f125-g01` como
  revalidacion del owner canonico T256, no como nuevo owner para el alias
  `server-shutdown-usecase-file-split-before-growth`.
- Cerrar tambien el retry f39e
  `agent-ref-task-autoprogramming-510467d5758e-g01` como revalidacion cubierta
  del mismo owner canonico, sin convertir `worktree_ref` ni `branch_ref` en
  rutas o nombres Git.
- Mantener el split dentro de `orquesta-server-shutdown` y el wiring opcional
  en `cmd/orquesta-server`; el modulo neutral sigue coordinando por puertos y
  no mata procesos ni importa runtime concreto, Codex, MCP, web, DB,
  filesystem ni `cmd`.
- Conservar `worktree_ref` y `branch_ref` como refs opacas y resolver el
  contexto obligatorio `ref_only` mediante lectura local y evidencia en ACK.

Duplicaciones a evitar:

- No duplicar T256: el alias `before-growth` queda absorbido por el owner
  canonico y solo se refuerza con una prueba de presupuesto de ficheros Go.
- No duplicar T30/T225/T239/T199/T217: checkpoint cooperativo, escalado
  guardian, correlacion de ACK, catalogo de errores y politica de senales siguen
  en sus owners vecinos.
- No duplicar T52/T54/T90/T242: splits de app-director-service, stack Codex,
  residuo general y supervisor residente no pertenecen a este cierre local.

## Priorizacion T257 2026-05-27 burst 003 7caabb

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el paquete
  `agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-7caabb5af4687f69a91e450ea47d88db`
  como no-op cubierto: reobserva T257 ya cerrado, contexto obligatorio
  `ref_only`, `required_ref_action=ack_evidence_required`, write-set cerrado y
  prueba focal del servidor.
- Mantener el cierre existente de T257: carga federada, parser de indice,
  parser local, clasificacion de estado y refs de contexto quedan separados en
  `cmd/orquesta-server`.
- Resolver el contexto requerido por lectura local/evidencia ACK y conservar
  `worktree_ref`/`branch_ref` como refs opacas.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo valida resolucion operacional de contexto
  por ACK.
- No duplicar T249/T250/T251: identidad, dedupe/preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos, reservas, colisiones, aliases y read
  model siguen ahi.
- No duplicar T255/T256/T258: OPES bridge, server-shutdown y suite historica
  app-director-service conservan sus splits propios.

## Priorizacion T257 2026-05-27 burst 002 ebeb

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar la revalidacion
  `agent-ref-assessment-task-autoprogramming-c4dae25a0488-g01-ebebfa72236082cbc973b4c48b012e70`
  como no-op cubierto: reobserva T257, contexto `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado y prueba focal
  del servidor.
- Mantener el cierre existente de T257: carga federada, parser de indice,
  parser local y clasificacion de estado quedan separados en
  `cmd/orquesta-server`; `modulos/orquesta-server` no asume politicas de
  producto ni persistencia concreta.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK.
- No duplicar T249/T250/T251: identidad, dedupe/preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos, reservas, colisiones, aliases y read
  model siguen ahi.
- No duplicar T255/T256/T258: los splits OPES bridge, server-shutdown y suite
  historica app-director-service ya tienen owner local.

## Priorizacion scanner 2026-05-27 rework T254 09624d3f

Backlog asociado: `T254 backlog-task-id-collision-alias-index`.

Estado 2026-05-27: rework cerrado como correccion de entrega, no como owner
nuevo. El paquete
`agent-ref-task-ref-review-rework-task-autoprogramming-6ed65028eb0b-g01-0154a0e80595eed31f9c3059a95a2f93`
exige resolver contexto `ref_only` por evidencia ACK y conservar la entrega ya
valida de T254: aliases `Txx#NN`, refs de instancia/entrada y reason
`backlog_duplicate_task_id_ambiguous` para colisiones de numero humano.

Prioridad alta:

- No relanzar agente padre sobre la tarea original ni abrir otro Txx para el
  mismo patron de colision/alias de backlog.
- Mantener reparacion aditiva: no renumerar, borrar ni mover historico; usar
  `task_instance_ref`, `backlog_task_entry_ref`, alias canonico o decision del
  director cuando un consumidor llegue solo con `Txx`.
- Dejar el contexto requerido resuelto por lectura local en ACK con nota
  `contexto_ref_only_resuelto`.

Duplicaciones a evitar:

- No duplicar T44: la guarda `required ref_only` se satisface en ACK.
- No duplicar T249/T250/T251: identidad de escaneo, reservas y preflight siguen
  en sus owners.
- No duplicar T252/T254: read model, aliases y colisiones de ids humanos siguen
  cerrados en el planner residente.

## Priorizacion scanner 2026-05-27 retry 9607b2

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-task-autoprogramming-1db274df2c82-g01` como no-op cubierto:
  reobserva `ref_only`, `required_ref_action=ack_evidence_required`,
  write-set cerrado a shards de backlog, prueba global obligatoria y owners
  pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry db2473 burst 002

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-task-autoprogramming-ba3b7ec7b59c-g01` como no-op cubierto:
  reobserva `ref_only`, `required_ref_action=ack_evidence_required`, write-set
  cerrado a shards de backlog, prueba global obligatoria y owners pendientes
  visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry ca7a01 burst 002

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental `agent-ref-task-autoprogramming-b06fa6486ff1-g01`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y backlog degradado ya cubierto por owners
  pendientes visibles antes de programar codigo.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git:
  `worktree-ref-orquesta-server-idle-self-improvement` y
  `branch-ref-orquesta-server-idle-self-improvement` son refs opacas.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe/preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, reserva, aliases, colisiones y read
  model siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry 4da183 burst 003

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-3533ffab219a-g01-3b2c0d9af98bdc2180840a7c40a31ccc`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry debe09

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-task-autoprogramming-62119e82f3e7-g01` como no-op cubierto:
  reobserva `ref_only`, `required_ref_action=ack_evidence_required`, write-set
  cerrado a shards de backlog, prueba global obligatoria y owners pendientes
  visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry c848ae

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-task-autoprogramming-d5b73c0f6568-g01` como no-op cubierto:
  reobserva `ref_only`, `required_ref_action=ack_evidence_required`, write-set
  cerrado a shards de backlog, prueba global obligatoria y backlog degradado ya
  cubierto por owners pendientes.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry 33cf37 burst 002

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-task-autoprogramming-29c1c237ce8c-g01` como no-op cubierto:
  reobserva `ref_only`, `required_ref_action=ack_evidence_required`,
  write-set cerrado a shards de backlog, prueba global obligatoria y owners
  pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry 6ed415 burst 002

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-6050b74fc6c4-g01-b49cac799ca28e09e2cf85d868d4d056`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe/preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry 3d4312 burst 002

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-task-autoprogramming-f924bc43a081-g01` como no-op cubierto:
  reobserva `ref_only`, `required_ref_action=ack_evidence_required`, write-set
  cerrado a shards de backlog, prueba global obligatoria y owners pendientes
  visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry 6ed415 burst 002

Backlog asociado: T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y
T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-task-autoprogramming-6050b74fc6c4-g01` como no-op cubierto:
  reobserva `ref_only`, `required_ref_action=ack_evidence_required`, write-set
  cerrado a shards de backlog, prueba global obligatoria, backlog degradado y
  owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry d3160 burst 002

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-task-autoprogramming-205bd061b6ad-g01` como no-op cubierto:
  reobserva `ref_only`, `required_ref_action=ack_evidence_required`,
  write-set cerrado a shards de backlog, prueba global obligatoria y owners
  pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.
- No abrir otro `Txx` desde scanners equivalentes mientras no aparezca una
  frontera causal nueva con medicion, owner local y ausencia de solape contra
  T44/T249/T250/T251/T252/T254/T255/T256/T257/T258.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Revalidacion scanner retry ed6089 burst 002 2026-05-27

Backlog asociado: T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y
T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-task-autoprogramming-72215ef37936-g01` como no-op cubierto:
  reobserva `ref_only`, `required_ref_action=ack_evidence_required`, write-set
  cerrado a shards de backlog, prueba global obligatoria y owners pendientes
  visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, reserva, colisiones, aliases y read
  model siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry bb17

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-task-autoprogramming-6ae909f7547f-g01` como no-op cubierto:
  reobserva `ref_only`, `required_ref_action=ack_evidence_required`, write-set
  cerrado a shards de backlog, prueba global obligatoria y owners pendientes
  visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry 4e3533

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-task-autoprogramming-8e83bd89baf6-g01` como no-op cubierto:
  reobserva `ref_only`, `required_ref_action=ack_evidence_required`, write-set
  cerrado a shards de backlog, prueba global obligatoria y owners pendientes
  visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- Preservar `worktree_ref` y `branch_ref` como refs opacas; no convertirlas en
  rutas ni nombres Git.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, reserva, colisiones, aliases y read
  model siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya
  tienen owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry 15eeecb9 burst 002

Backlog asociado: T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y
T258.

Prioridad alta:

- Cerrar el retry documental `agent-ref-task-autoprogramming-7b90ca195eaf-g01`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- Revisar backlog/rail errors/duplicaciones antes de programar codigo: el
  patron degradado ya tiene owners, asi que este scanner no debe abrir Txx ni
  ampliar alcance.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Revalidacion T250 2026-05-27 acff973f

Backlog asociado: `T250 federated-backlog-epoch-scope-policy`.

Prioridad cerrada:

- Cerrar la evaluacion
  `agent-ref-assessment-task-autoprogramming-88a96698afb2-g01-6dab0da79e1908567a8bf8e7f3250f9b`
  como revalidacion de T250: no abre un segundo owner y no convierte fuentes
  `historico`, `stale` o `quarantine` en dependencias activas del epoch.
- Mantener la regla operativa: solo fuentes federadas ejecutables (`vigente`,
  `promocionada`, `promoted`, `active`) participan en `BacklogScanDocs`, epoch
  y reservas; las demas viajan como contexto compacto
  `federated_backlog_source_not_executable`.
- Resolver `ref_only` en ACK con lectura local/evidencia explicita y conservar
  `worktree_ref`/`branch_ref` como refs opacas.

Duplicaciones a evitar:

- No duplicar T43/T240: lease, epoch documental base y lectura acotada siguen
  separados.
- No duplicar T88/T116/T124: indice federado, precedencia documental y
  cuarentena legacy siguen con sus owners.
- No duplicar T248/T249/T251: identidad de entradas, dedupe/preflight y alcance
  de pruebas de scanner siguen separados.

## Priorizacion scanner 2026-05-27 retry 35a079

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental `agent-ref-task-autoprogramming-49af6e594adb-g01`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry a3abed burst 002

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental `agent-ref-task-autoprogramming-30b4d5ecfe9b-g01`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry d1d80

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-task-autoprogramming-798614d0f80e-g01` como no-op cubierto:
  reobserva `ref_only`, `required_ref_action=ack_evidence_required`, write-set
  cerrado a shards de backlog, prueba global obligatoria y owners pendientes
  visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 burst 003 f7d3

Backlog asociado: T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y
T258.

Prioridad alta:

- Cerrar el paquete
  `agent-ref-assessment-task-autoprogramming-fba55e3296a0-g01-f7d3f88efa3648cbceb5112f1cf4a040`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a los tres
  shards, prueba global obligatoria y owners pendientes ya visibles.
- Resolver el contexto requerido por lectura local/evidencia en ACK, conservar
  `worktree_ref` y `branch_ref` como refs opacas y no crear otro `Txx` mientras
  no aparezca un hueco causal distinto.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad de escaneo, dedupe/preflight y alcance
  de pruebas de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos duplicados, reserva, aliases y read model
  siguen ahi.
- No duplicar T253/T255/T256/T257/T258: los splits recientes ya tienen owners
  concretos; este scanner solo conserva evidencia de cobertura.

## Priorizacion scanner 2026-05-27 scanner 15eeecb9 burst 002 8a1ece

Backlog asociado: T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y
T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-fba55e3296a0-g01-8a1ecec98ecf3323c5488e327056f703`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete, AGENTS/README y
  fuentes vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe/preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, reserva, colisiones, aliases y read
  model siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 burst 002 fba55

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-task-autoprogramming-fba55e3296a0-g01` como no-op cubierto:
  reobserva `ref_only`, `required_ref_action=ack_evidence_required`, write-set
  cerrado a shards de backlog, prueba global obligatoria y owners pendientes
  visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 scanner 15eeecb9 burst 002 f13a

Backlog asociado: T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y
T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-fba55e3296a0-g01-f13a1353960542f949b1c478324a27ec`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete, AGENTS/README y
  fuentes vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe/preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, reserva, colisiones, aliases y read
  model siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 burst 002 da14

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-da14e4762930936ab46118bb39f7aa9a`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 burst 002 17978

Backlog asociado: T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y
T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-17978e79d655060ec0da8977cea2a0ee`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- Conservar `worktree_ref` y `branch_ref` como refs opacas; no convertirlas en
  rutas ni nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 burst 002 2048

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-2048ee92b24ec50625c2242b8eb85463`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 burst 002 52b63

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-52b63d817ae78008276d7d3e084d76df`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 burst 002 585742

Backlog asociado: T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y
T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-585742898dfa4be257b5a2282769c9c2`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, reserva, colisiones, aliases y read
  model siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry 69e1ba burst 002 c86d18

Backlog asociado: T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y
T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-8f23d8e2a22e-g01-c86d18cc769b57d176d2617c4d0990a6`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry 7a567340

Backlog asociado: T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y
T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-3ddbc1385ac2-g01-0d1f79ec31384d00b4994f75b7e11068`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry 69e1ba burst 002 406b68

Backlog asociado: T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y
T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-8f23d8e2a22e-g01-406b68b132e5ad3f6495920aad6064eb`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry d64a6a burst 002 644f

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-757fa45b2845-g01-644fcb17f3eb0e2fc3b46dc77aa883b8`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe/preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry 42fbc4 burst 002 abd199

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-721523601cc1-g01-abd199fbb4db7baa1c08602cd2d3dade`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 burst 002 988548

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-988548c1efac02c34d8ab6cf81a53d19`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 burst 002 70a7

Backlog asociado: T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y
T258.

Prioridad alta:

- Cerrar el paquete
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-70a7d8db3b4ebd33b19ae663b484aef6`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry d0cf1e burst 002 0d330c

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-350b93e475a8-g01-0d330c4e1579d03076ce1e175634e42b`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe/preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 burst 002 1db27

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-1db27b63b7ddb14755938b05f1eff32f`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 burst 002 e7040

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el burst documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-e7040a9c028b6c9195871408101972c0`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 burst 002 25ebab

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-25ebab4465052369f6e1f66d949cd514`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 burst 002 31b587

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el burst documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-31b58785a2bc25303ea28ac1a457f2d3`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 burst 002 c0f755

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-c0f7550c04cbfe4cddf673cf37c73725`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 burst 002 5b79

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el burst documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-5b79f9accf52b539a3be229cf4e83bdb`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry 7a5673 burst 002 7aa0

Backlog asociado: T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y
T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-3ddbc1385ac2-g01-7aa072fa3bae532da402f4b748f1904d`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry 5465c1 burst 002 ef66

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-87d5c800f528-g01-ef66a8153742b6161a7f9836a4c2779e`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 retry 42fbc4 burst 002

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-721523601cc1-g01-2e79a5694817170a5384a408d1acb03b`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 vigesima septima pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-e078c7be6425160a0ccb6bb8f8b78d11`
  como no-op cubierto: solo reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No abrir mas `Txx` desde scanners equivalentes mientras no aparezca una
  frontera causal nueva con medicion, owner local y ausencia de solape contra
  T44/T249/T250/T251/T252/T254/T255/T256/T257/T258.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad de escaneo, dedupe, preflight y alcance
  de pruebas de scanners siguen en esos owners.
- No duplicar T252/T254: lectura ambigua de ids humanos, reserva, colisiones y
  aliases aditivos siguen ahi.
- No duplicar T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 vigesima septima pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-4b676d31f48327f48f51c6ce1cc855b4`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- Mantener `worktree_ref` y `branch_ref` como refs opacas; no convertirlas en
  rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya
  tienen owner local y test asociado.

## Priorizacion scanner 2026-05-27 vigesima septima pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-ae7b1168a2882f5580acd71e20ec5732`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo aporta evidencia operacional de resolucion
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, reserva, colisiones, aliases y read
  model siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya
  tienen owner local y test asociado.

## Priorizacion scanner 2026-05-27 vigesima septima pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-72e651512c5a9856094dd77d5b947474`
  como no-op cubierto: solo reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- Preservar `worktree_ref` y `branch_ref` como refs opacas; no convertirlas en
  rutas ni nombres Git.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad de escaneo, dedupe, preflight y alcance
  de pruebas de scanners siguen en esos owners.
- No duplicar T252/T254: lectura ambigua de ids humanos, reserva, colisiones y
  aliases aditivos siguen ahi.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya
  tienen owner local y test asociado.

## Priorizacion scanner 2026-05-27 vigesima septima pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-1957b876997d6c6e6b87bbb19f48735f`
  como no-op cubierto: solo reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo aporta evidencia operacional de contexto
  resuelto, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 vigesima septima pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-41c250bd381a729b6b7a942fa03db578`
  como no-op cubierto: solo reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 vigesima sexta pasada

Backlog asociado: T44, T249, T250, T251, T252, T253, T254, T255, T256, T257 y
T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-30cb4ab771802c1a99b591d81d13d88d`
  como no-op cubierto: solo reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Mantener el ACK como evidencia de resolucion de contexto por lectura local;
  no convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.
- No abrir mas `Txx` desde scanners equivalentes mientras no aparezca una
  frontera causal nueva con medicion, owner local y ausencia de solape contra
  T44/T249/T250/T251/T252/T253/T254/T255/T256/T257/T258.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad de escaneo, dedupe, preflight y alcance
  de pruebas de scanners siguen en esos owners.
- No duplicar T252/T254: lectura ambigua de ids humanos, reserva, colisiones y
  aliases aditivos siguen ahi.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya
  tienen owners locales y pruebas asociadas.

## Priorizacion scanner 2026-05-27 vigesima septima pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-55fbe00917266eb9dce3cb7ef06512d2`
  como no-op cubierto: solo reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Mantener el ACK como evidencia de resolucion de contexto por lectura local y
  fuentes vigentes; no convertir `worktree_ref` ni `branch_ref` en rutas o
  nombres Git.
- No abrir otro `Txx` desde scanners equivalentes mientras no aparezca una
  frontera causal nueva con medicion, owner local y ausencia de solape contra
  T44/T249/T250/T251/T252/T254/T255/T256/T257/T258.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad de escaneo, dedupe, preflight y alcance
  de pruebas de scanners siguen en esos owners.
- No duplicar T252/T254: lectura ambigua de ids humanos, reserva, colisiones y
  aliases aditivos siguen ahi.
- No duplicar T255/T256/T257/T258: los splits concretos de OPES bridge,
  server-shutdown, backlog federado y suite historica app-director-service ya
  tienen owners locales.

## Priorizacion scanner 2026-05-27 vigesima septima pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-0baf5a292701de903673675f4731b8c4`
  como no-op cubierto: solo reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Mantener el ACK como evidencia de resolucion de contexto por lectura local;
  no convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.
- No abrir mas `Txx` desde scanners equivalentes mientras no aparezca una
  frontera causal nueva con medicion, owner local y ausencia de solape contra
  T44/T249/T250/T251/T252/T254/T255/T256/T257/T258.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad de escaneo, dedupe, preflight y alcance
  de pruebas de scanners siguen en esos owners.
- No duplicar T252/T254: lectura ambigua de ids humanos, reserva, colisiones y
  aliases aditivos siguen ahi.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya
  tienen owner local y este scanner no aporta otro fichero ni frontera.

## Priorizacion scanner 2026-05-27 vigesima septima pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-4f550c019e0d-g01-ed50a0e5d063ce9367242318b76013ca`
  como no-op cubierto: solo reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Mantener el ACK como evidencia de resolucion de contexto por lectura local;
  no convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.
- No abrir mas `Txx` desde scanners equivalentes mientras no aparezca una
  frontera causal nueva con medicion, owner local y ausencia de solape contra
  T44/T249/T250/T251/T252/T254/T255/T256/T257/T258.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad de escaneo, dedupe, preflight y alcance
  de pruebas de scanners siguen en esos owners.
- No duplicar T252/T254: lectura ambigua de ids humanos, reserva, colisiones y
  aliases aditivos siguen ahi.
- No duplicar T255/T256/T257/T258: los splits concretos de OPES bridge,
  server-shutdown, backlog federado y suite historica app-director-service ya
  tienen owners locales.

## Priorizacion scanner 2026-05-27 retry 42fbc4

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-721523601cc1-g01-a88632e247fc4b53576ed868990269ec`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad de escaneo, dedupe, preflight y alcance
  de pruebas de scanners siguen en esos owners.
- No duplicar T252/T254: lectura ambigua de ids humanos, reserva, colisiones y
  aliases aditivos siguen ahi.
- No duplicar T255/T256/T257/T258: los splits concretos de OPES bridge,
  server-shutdown/runtime-process, backlog federado y suite historica
  app-director-service ya tienen owners locales.

## Priorizacion scanner 2026-05-27 vigesima sexta pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-87d5c800f528-g01-7099771d60152a40426c551b9fed8fdb`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local/evidencia en ACK, preservar
  `worktree_ref` y `branch_ref` como refs opacas y no abrir otro `Txx` salvo
  frontera causal nueva con medicion, owner local y ausencia de solape.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos, reserva, colisiones y aliases aditivos
  siguen ahi.
- No duplicar T253/T255/T256/T257/T258: los splits concretos ya tienen owner
  local y este scanner no aporta otro fichero ni frontera.

## Priorizacion scanner 2026-05-27 vigesima sexta pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-3ddbc1385ac2-g01-a39762df8a44697f801b68e1a8195f6b`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.

## Priorizacion scanner 2026-05-27 vigesima sexta pasada

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el scanner documental
  `agent-ref-assessment-task-autoprogramming-757fa45b2845-g01-ec8943db18ad09ba2b17a1035b324e9b`
  como no-op cubierto: solo reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Mantener el ACK como evidencia de resolucion de contexto por lectura local;
  no convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.
- No abrir mas `Txx` desde scanners equivalentes mientras no aparezca una
  frontera causal nueva con medicion, owner local y ausencia de solape contra
  T44/T249/T250/T251/T252/T254/T255/T256/T257/T258.

Duplicaciones a evitar:

- No duplicar T44: contexto `ref_only` queda resuelto por lectura local y nota
  `contexto_ref_only_resuelto` en ACK.
- No duplicar T249/T250/T251: identidad de escaneo, dedupe, preflight y alcance
  de pruebas de scanners siguen en esos owners.
- No duplicar T252/T254: lectura ambigua de ids humanos, reserva, colisiones y
  aliases aditivos siguen ahi.
- No duplicar T255/T256/T257/T258: los splits concretos de OPES bridge,
  server-shutdown, backlog federado y suite historica app-director-service ya
  tienen owners locales.
## Priorizacion scanner 2026-05-27 retry d0cf1e

Backlog asociado: T44, T249, T250, T251, T252, T254, T255, T256, T257 y T258.

Prioridad alta:

- Cerrar el retry documental
  `agent-ref-assessment-task-autoprogramming-350b93e475a8-g01-7b6283b42d81ab988ec874761f9ce9e0`
  como no-op cubierto: reobserva `ref_only`,
  `required_ref_action=ack_evidence_required`, write-set cerrado a shards de
  backlog, prueba global obligatoria y owners pendientes visibles.
- Resolver el contexto requerido por lectura local del paquete y fuentes
  vigentes, dejando nota `contexto_ref_only_resuelto` en ACK.
- No convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git.

Duplicaciones a evitar:

- No duplicar T44: esta pasada solo prueba resolucion operacional de contexto
  por ACK, no cambia la guarda general.
- No duplicar T249/T250/T251: identidad, dedupe, preflight y alcance de pruebas
  de scanners siguen en esos owners.
- No duplicar T252/T254: ids humanos `Txx`, aliases, colisiones y read model
  siguen en esos owners.
- No duplicar T253/T255/T256/T257/T258: los splits concretos recientes ya tienen
  owner local y test asociado.
