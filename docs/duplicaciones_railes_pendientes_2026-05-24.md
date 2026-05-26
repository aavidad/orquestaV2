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
- Reloj y generacion de refs: `cmd/orquesta-server`, `modulos/orquesta-server`,
  `modulos/orquesta-runtime`, `modulos/orquesta-runtime-codex-delivery`,
  `modulos/orquesta-app-codex-stack`, `modulos/orquesta-web` y
  `modulos/orquesta-cli` usan `time.Now`, `UnixNano`, refs con timestamp o
  fallbacks locales. Debe haber owner comun de clock/ref generator por
  composicion para evitar colisiones y replay no determinista.
- Identidad publica de mutaciones: web, CLI, MCP, gateways y comandos del
  servidor repiten normalizacion de `request_id`, `correlation_id`,
  `idempotency_key` y `X-Correlation-ID`. Falta una politica compartida que
  distinga lecturas de mutaciones reintentables y no duplique reglas por
  cliente.

## Priorizacion scanner 2026-05-24

Backlog asociado: `T15 detail-rails-reactivation`.

Prioridad alta:

- `modulos/orquesta-context/context_bundle_validation_v0.go`, porque puede
  bloquear formas reparables antes de que el director normalice contexto.
- `modulos/orquesta-director-agent/director_decision_helpers_v0.go`, porque
  participa en decisiones y reviews donde ya se observaron falsos positivos por
  vocabulario operativo.
- `cmd/orquesta-server` y scripts de rails, porque hoy fijan
  `ORQUESTA_DETAIL_PROHIBITED_RAILS=on` por defecto con scope acotado y deben
  mantener sincronizados registro vivo, matriz y docs antes de ampliar ese
  scope o volver a cambiar el default.

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
  `cmd/orquesta-server`: `ORQUESTA_DETAIL_PROHIBITED_RAILS=on` con scope
  acotado. El registro actual no pertenece a este write-set, por lo que queda
  como tarea separada y no como cambio silencioso.
- Asegurar que `./scripts/test_rails_fast.sh` siga siendo la matriz rapida que
  prueba default, override `off`, scopes y falsos positivos de vocabulario
  operativo antes de compilar servidor completo.

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

## Priorizacion scanner 2026-05-24 trigesima pasada

Backlog asociado: `T94 server-audit-write-failure-visibility`.

Prioridad alta:

- Separar redaccion/schema de auditoria (T41) de visibilidad de fallo de
  escritura. Un sink JSONL roto no debe desaparecer como best-effort silencioso
  cuando el mismo servidor usa esa auditoria como evidencia operativa de
  automejora, supervisor, startup y control plane.
- Unificar propietario del estado de auditoria degradada en
  `modulos/orquesta-server`: HTTP, supervisor loop, startup checks y
  composicion de `cmd/orquesta-server` no deben inventar cada uno su contador,
  error publico o politica de bloqueo.

Prioridad media:

- La proyeccion de fallo debe ser compacta y no recursiva: contador, evento,
  codigo publico y timestamp; sin rutas locales, HOME, permisos exactos,
  payloads HTTP, prompts, transcripts ni tokens.
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
- Convertir el ledger de entrada de bridges externos en claim/recovery antes de
  crear runs. T31 protege la cola interna, pero no impide que dos drains OPES o
  un fallo post-submit generen duplicacion por job externo.

Prioridad media:

- Separar logs operacionales del daemon de auditoria JSONL, tail Codex y
  required-test output. Si se exponen, deben ser resumenes redactados con
  retencion/rotacion y no una nueva fuente terminal de cierre.
- Las tres tareas deben compartir politica de redaccion por campo: refs opacas,
  codigos publicos y contadores pasan; HOME, rutas privadas, PIDs como verdad
  unica, argv completos, prompts, transcripts, tokens, payloads HTTP completos
  y respuestas OPES crudas no se persisten como evidencia de cierre.

## Priorizacion scanner 2026-05-24 trigesimotercera pasada

Backlog asociado: `T99 server-http-resource-guardrails`,
`T100 startup-revision-archive-redaction-retention` y
`T101 domain-work-artifact-submission-ledger-recovery`.

Prioridad alta:

- Unificar limites de transporte HTTP antes de ampliar control plane residente:
  servidor, MCP HTTP, app-gateway y web no deben copiar cada uno su decode JSON,
  tamano de body, politica de campos desconocidos ni timeouts.
- Separar archivo de revision de startup de purga/reconciliacion. T35 decide
  si compactar y T59 protege purga de runtime; T100 debe gobernar permisos,
  redaccion, retencion y exposicion publica de los artefactos generados.
- Convertir el ledger de salida `domain_work` en claim/recovery antes de
  ejecutar mas derivados OPES reales. T98 protege entrada de jobs; T101 protege
  el efecto externo de devolver artefactos.

Prioridad media:

- La auditoria de HTTP debe registrar solo metadatos compactos del limite
  aplicado, coordinada con T41/T94; no guardar bodies ni usar logs daemon como
  fallback.
- El archivo de revision puede conservar material crudo solo como diagnostico
  local opt-in con limite; no debe entrar en contexto, AppVCS, promocion,
  timeline ni recursos MCP.
- Los ledgers de entrada/salida deben compartir conceptos de idempotency,
  correlation, receipt y conflicto, pero sin fusionar OPES con el contrato
  neutral `domain_work`.

## Priorizacion scanner 2026-05-24 trigesimocuarta pasada

Backlog asociado: `T102 legacy-http-json-boundary-policy`,
`T103 outbound-http-response-limit-redaction` y
`T104 file-store-durable-write-policy`.

Prioridad alta:

- Unificar la frontera JSON de handlers HTTP antes de ampliar endpoints publicos
  o transporte MCP residente. T99 cubre recursos del servidor; T102 debe evitar
  que factory/governance/MCP/web/app-gateway conserven decoders, limites y
  shapes de error incompatibles.
- Fijar limites de respuesta saliente antes de usar conectores HTTP como fuente
  durable de diagnostico. T80 decide adonde puede salir `domain_work-http`;
  T103 decide cuanto y como se lee lo que vuelve.
- Normalizar escritura durable file-based antes de depender de restart,
  reconciliacion, ledgers y estado residente como prueba terminal. T95 detecta
  fallo de estado, pero T104 debe cerrar permisos, fsync, temp/lock y recovery
  de snapshot.

Prioridad media:

- Los helpers HTTP deben preservar compatibilidad legacy de forma explicita y
  con tests; no endurecer campos desconocidos si rompe clientes finos vigentes
  sin plan de migracion.
- La redaccion de respuestas debe compartir politica con T41/T94/T97, pero no
  convertir cuerpos externos en auditoria paralela ni logs operacionales.
- La politica durable puede reutilizar helpers existentes de `orquesta-run-file`
  y `orquesta-domain-work-file`, pero debe respetar owners concretos de outbox,
  cola, ledgers OPES/domain_work y runtime Codex.

## Priorizacion scanner 2026-05-24 trigesimoquinta pasada

Backlog asociado: `T105 process-runtime-launch-env-io-receipt`,
`T106 codex-wave-public-summary-redaction` y
`T107 real-smoke-go-diagnostic-redaction`.

Prioridad alta:

- Separar launch neutral de stop neutral. T66 gobierna parada causal; T105 debe
  gobernar recibo de resolucion de comando/env/working dir e IO para que el
  runtime no dependa de valores reales invisibles ni de stdout/stderr crudo.
- Redactar la salida publica de `codex-wave` antes de usarla como evidencia o
  diagnostico compartible. Tail/purge/stop tienen owners propios, pero launch y
  status aun pueden exponer rutas y ficheros de control completos.
- Cubrir smokes Go opt-in, no solo scripts shell. Los fallos de pruebas reales
  deben ser utiles para operador sin imprimir prompts, transcripts, bodies,
  stdout/stderr completos ni rutas privadas.

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

- Dar owner operativo a las rondas de consejo. `orquesta-decision-council`
  planifica propuesta/critica/voto y `orquesta-director-candidates` sabe
  convertir asignaciones en candidatos, pero el ciclo vivo del Director no debe
  recrear deliberacion como texto libre ni saltarse quorum/evidencia.
- Unificar leases y progreso. `orquesta-core-leases`, heartbeat de
  `orquesta-runtime`, observacion Codex y scheduler de lease actions no deben
  mantener umbrales paralelos para stopped/stalled/loop/retry/stop/replan.

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

Backlog asociado: `T113 module-historical-doc-ref-sync`,
`T114 run-queue-fairness-group-policy` y
`T115 web-app-intake-session-contract`.

Prioridad alta:

- Separar contexto historico de contexto requerido vivo. T111 cubre
  instrucciones obligatorias en `AGENTS.md`; T113 debe limpiar docs locales que
  aun apuntan a `docs/reinicio_orquesta_v2` para que el indice federado no
  convierta DBV1/control-plane stale en tareas ejecutables.
- Dar semantica a `fairness_group_ref` antes de ampliar colas paralelas de
  automejora. T31 reserva, T112 evita solapes de write-set y T114 debe evitar
  hambre o monopolio entre grupos.

Prioridad media:

- Completar `WEB-013` como contrato de intake web solo despues de mantener T76
  como ruta publica del Director V2. La web conserva AppSpec/refs opacas y no
  decide arquitectura, runtime, proveedor ni DB.
- T113 no debe borrar documentos historicos ni recrear carpetas ausentes; debe
  marcar fuente, freshness y sustituto vigente.
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
- T118 debe coordinarse con T27/T111/T113: marcar historico o plantilla, sin
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
- Pasar de `.gitignore` a politica ejecutable para artefactos locales. T50
  cubre control files, pero snapshots, contexto, AppVCS, promocion y ACK
  terminal necesitan excluir tambien logs/diagnostico locales ignorados sin
  depender de Git ni de un walk ad hoc.

Prioridad media:

- T125 debe coordinarse con T51/T117: perfiles, razonamiento y review gate son
  contratos vecinos, pero no se corrigen creando un modulo vacio como owner.
- T126 debe coordinarse con T121/T30/T66/T96: wrapper manual, checkpoint,
  parada neutral e identidad de daemon conservan owners separados; el wrapper
  solo puede llamar a esas superficies o quedar historico.
- T127 debe coordinarse con T43/T50/T56/T59: merge documental, control files,
  proyeccion de credenciales y purga de runtime no deben duplicar listas
  parciales. La salida publica es categoria/contador/ref, no path local ni
  contenido crudo.

## Priorizacion scanner 2026-05-24 cuadragesima tercera pasada

Backlog asociado: `T128 agent-progress-supervisor-rail-policy-sync`,
`T129 server-daemon-start-env-policy` y
`T130 architecture-guard-test-rail-policy-owner`.

Prioridad alta:

- Unificar el rail del supervisor de progreso antes de que oculte evidencia
  operacional valida. La decision del Director necesita refs opacas completas,
  no una lista local que borre palabras como `provider`, `model` o `runtime`.
- Acotar el entorno efectivo de `start` del daemon. Heredar el proceso padre sin
  recibo de categorias/redaccion mezcla conveniencia operativa con superficie
  sensible de HOME, tokens, proveedor y diagnostico.
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
- T132 debe coordinarse con T41/T81/T85: audit payloads, timeline y status
  publico consumen privacidad/redaccion, pero no deben poseer tres politicas
  incompatibles.
- T133 debe coordinarse con T111/T116/T125: refs documentales locales, linter
  de integridad y modulos placeholder son vecinos; aqui el foco es cobertura de
  frontera por modulo sensible.

## Priorizacion scanner 2026-05-24 cuadragesima quinta pasada

Backlog asociado: `T134 server-status-legacy-alias-sunset`,
`T135 idle-self-improvement-planner-fallback-safety` y
`T136 resident-drain-external-wait-budget-policy`.

Prioridad alta:

- Canonizar `/api/v0/server/status` y tratar `/api/status` como alias legacy
  con owner. El CLI, toolbelt y docs no deben normalizar una ruta sin version
  como si fuera contrato vigente.
- Hacer seguro el fallback del planner de automejora. Un backlog ilegible,
  vacio o ambiguo debe producir bloqueo/revision o scanner documental acotado,
  no una tarea generica de codigo con write-set historico.
- Gobernar `MaxExternalWaits` por modo. El residente puede mantener smokes
  rapidos, pero Codex real, OPES temporal y operador necesitan presupuesto
  observable sin confundir espera externa viva con idle.

Prioridad media:

- T134 coordina con T119/T85/T87: manuales, estado publico y readiness siguen
  siendo owners separados; aqui se decide el alias runtime.
- T135 coordina con T17/T33/T37/T43/T79: reservas, ACK terminal, artefactos,
  merge documental y plan asociado son vecinos, pero el foco es impedir que el
  scanner degradado amplie scope.
- T136 coordina con T32/T68/T69: wakeups, timeouts y taxonomia de parada no se
  reabren; esta tarea fija presupuesto de espera externa y status efectivo del
  residente.

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
- T138 coordina con T41/T85/T131: auditoria, status y rutas locales comparten
  redaccion, pero aqui el owner es respuesta HTTP recibida por clientes.
- T139 coordina con T58/T61/T85/T138: logs, tests, status y respuestas HTTP
  alimentan comandos, pero el cierre terminal sigue en ACK/eventos/evidencias
  estructuradas, no en stdout.

## Priorizacion scanner 2026-05-24 cuadragesima septima pasada

Backlog asociado: `T140 backlog-overlap-canonicalization` y
`T141 public-boundary-no-panic-contract`.

Prioridad alta:

- Consolidar solapes ya escritos en el backlog antes de lanzar mas agentes
  sobre HTTP request/response bounds. `T102`/`T137` y `T103`/`T138` deben tener
  canon/alias y merge de criterios; no borrar historia ni declarar cerrado por
  una sola entrada.
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
- T145 coordina con T23/T55/T85/T136: MCP residente, control plane, status y
  espera externa publican estado; el cliente de operador debe aportar timeout
  y cancelacion sin filtrar transporte.
- T146 coordina con T123/T133/T137: preview de factory, docs locales y helper
  JSON publico deben converger sin convertir factory en planner operativo.

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
- T148 coordina con T98/T101/T104: claim/recovery de bridge, submit ledger y
  escritura durable no se fusionan; aqui el foco es presupuesto de lectura,
  `max_records`, corrupcion recuperable y error publico redactado.

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
- Acotar ficheros de operador usados como prompt/objetivo/contexto de olas
  Codex. T151 separa esos inputs locales de T56 (`CODEX_HOME`) y de T143
  (control files), con limite y redaccion antes de construir paquetes.

Prioridad media:

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

- Separar T56 de su shard mecanico: la politica de que categorias de
  `CODEX_HOME` se proyectan no basta si la copia sigue sin max de bytes/ficheros
  ni tratamiento seguro de symlinks/modos. El propietario de copia debe vivir en
  la composicion Codex y no en el nucleo.
- Alinear comandos/eventos con el outbox: si outbox tiene presupuesto de
  payload pero comandos/eventos no, un payload grande puede entrar por workflow
  y persistencia antes de las guardas de lectura/escritura durable.
- Hacer visible el deadline de efectos externos. Un timeout de HTTP, MCP,
  runtime o shutdown debe tener error publico y correlation/ref; no debe quedar
  como espera indefinida ni como fallo opaco del cliente inyectado.

Prioridad media:

- La copia de `CODEX_HOME` debe coordinarse con T50/T127 para que credenciales y
  control files no entren en snapshot, AppVCS, promocion ni ACK.files.
- El presupuesto de payload del workflow debe conservar compatibilidad con
  eventos historicos y usar refs de artefacto para contenido grande; no mover
  cuerpos HTTP, prompts ni transcripts a eventos por comodidad.
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
- T156 debe coordinarse con T105 y T154: deadline e IO neutral son vecinos, pero
  los comandos Git de AppVCS necesitan codigos propios de truncation, timeout y
  exceso de paths.

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
- T158 debe coordinarse con T12, T21, T98, T103 y T139: smoke OPES, guardas
  reales, claim de entrada, respuesta HTTP y salida de comandos siguen como
  owners vecinos; aqui el foco es destino OPES y redaccion de summary.

## Priorizacion scanner 2026-05-24 quincuagesima quinta pasada

Backlog asociado: `T159 codex-control-file-durable-write-policy` y
`T160 director-decisions-batch-budget-and-source-limit`.

Prioridad alta:

- Unificar escritura durable de ficheros de control Codex. La lectura ya tiene
  tareas de tamano/raiz/symlink, pero packet, prompt, wrappers, registry y
  requests de checkpoint se escriben desde owners distintos con `os.WriteFile`
  sobre ruta final. La politica debe ser atomica, con permisos cerrados,
  receipt compacto y sin path local ni contenido crudo.
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

## Priorizacion scanner 2026-05-24 quincuagesima sexta pasada

Backlog asociado: `T161 clock-and-ref-generation-policy` y
`T162 public-client-mutation-idempotency-policy`.

Prioridad alta:

- Unificar reloj y generador de refs antes de ampliar olas concurrentes,
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
- T164 debe conservar metadata completa del `WorkflowTaskStore`; no reconstruir
  parent/child, ola o cohorte desde eventos compactos si el indice puede
  recuperarse o rematerializarse con presupuesto.
- T165 debe distinguir default operativo de autorizacion explicita. Un default
  de rails o cleanup puede ser valido para servidor residente, pero debe quedar
  visible como `defaulted`, no como env original del operador.

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
- T166 y T167 deben compartir taxonomia de lifecycle (`running`, `stopping`,
  `stopped`, `timeout`, `degraded`) con T87/T134, pero no deben meter OPES,
  Codex ni proveedor en `orquesta-server`.
- T168 debe conservar compatibilidad de snapshots validos y no convertirse en
  migrador silencioso. Reparacion o migracion de estado debe entrar por puerto
  opt-in con evidencia durable.

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
- T172 debe registrar errores compactos despues de headers emitidos sin intentar
  reescribir status tarde; la evidencia debe servir para smokes y shutdown
  quiescent sin almacenar payloads internos.

## Priorizacion scanner 2026-05-24 sexagesima pasada

Backlog asociado: `T173 browser-origin-csrf-intent-guard` y
`T174 mcp-jsonrpc-protocol-strictness`.

Prioridad alta:

- Separar origen/intencion de auth remota. T55 decide bind, principal y permiso,
  pero las mutaciones web/gateway necesitan owner propio para `Origin`,
  `Referer`, CSRF/intent token y decision compacta antes de exponer mas rutas a
  navegador local o bind opt-in.
- Fijar el contrato JSON-RPC de `/mcp` antes de usar el transporte residente
  como superficie publica estable. T23 y T169 no bastan si el handler acepta
  formas ambiguas de `jsonrpc`, `id`, batch, notification o params.

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

- Limitar salida MCP antes de tratar `/mcp` como transporte residente estable.
  T174 endurece request/protocolo, pero sin T175 un resource/tool puede devolver
  payload grande o diagnostico no redactado como texto JSON-RPC.
- Fijar headers/cache del control plane antes de ampliar uso web/browser. T173
  decide origen/intencion, pero T176 debe evitar framing, sniffing, referrers y
  cache implicita en rutas con estado operativo.
- Unificar base URL + endpoint en clientes REST antes de sumar nuevos gateways.
  T80/T170/T103 operan despues del destino construido; T177 debe impedir que el
  destino final nazca ambiguo.

Prioridad media:

- T175 debe reutilizar catalogo/freshness de recursos MCP cuando exista, sin
  duplicar descriptores estaticos de T25/T93.
- T176 debe coordinarse con T99/T137/T171/T172: no sustituye limites de body,
  query/form ni visibilidad de fallo de escritura.
- T177 debe distinguir transporte interno in-process de egress real. El alias
  `http://orquesta.internal` puede seguir vivo solo como destino interno
  probado, no como URL externa.

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
- T179 debe coordinarse con T102/T137/T141/T172/T176: el in-process debe
  compartir limites y errores publicos sin duplicar cada helper HTTP ni guardar
  payloads, prompts, transcripts, HOME, tokens o rutas privadas.

## Priorizacion scanner 2026-05-24 sexagesima tercera pasada

Backlog asociado: `T180 command-stdio-write-error-visibility`.

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
`T181 domain-document-plan-ref-uniqueness-and-diagnostics`,
`T182 mcp-tool-execution-budget-and-cancellation` y
`T183 web-html-render-error-contract`.

Prioridad alta:

- Resolver T181 antes de ampliar OPES temporal o nuevos conectores documentales:
  si el plan neutral permite refs duplicados o pierde diagnostico de arrays raw,
  el adaptador recibe jobs ambiguos y la reparacion queda demasiado tarde.
- Resolver T182 antes de tratar MCP real como transporte residente de
  autoprogramacion: T174/T175 cubren request y salida, pero sin presupuesto de
  ejecucion un tool lento puede agotar el worker sin reason code estable.

Prioridad media-alta:

- T183 debe entrar junto a T172/T176 cuando se endurezcan fronteras web. Un
  template roto o writer tardio no debe quedar como exito visual parcial ni como
  auditoria con datos privados.

Duplicaciones a evitar:

- T181 no sustituye validadores editoriales OPES ni semantica de `DomainWork`;
  solo fija unicidad/diagnostico del plan documental neutral y la idempotencia
  previa a expansion.
- T182 no reabre deadlines internos de puertos (T154), parsing MCP (T174),
  output budget (T175) ni paridad in-process (T179); gobierna solo presupuesto
  de ejecucion del handler MCP real.
- T183 no duplica T172/T176/T173/T75: usa sus helpers cuando existan, pero el
  owner nuevo es el contrato de render HTML y fallback localizado.

## Priorizacion scanner 2026-05-24 sexagesima quinta pasada

Backlog asociado:
`T184 deterministic-ref-hash-collision-proof` y
`T185 smoke-script-temp-root-deletion-guard`.

Prioridad alta:

- Resolver T185 antes de ampliar smokes opt-in o ejecuciones reales con raiz
  inyectada por entorno. T59/T91/T121/T126/T127 acotan shutdown y semantica del
  smoke, pero no prueban que un cleanup recursivo solo borre directorios
  generados y marcados por el propio script.
- Resolver T184 antes de usar refs derivadas como contrato de recuperacion o
  cierre causal. T161/T162/T169/T181 cubren generacion, unicidad de tasks y
  refs documentales, pero no separan digest visual/advisory de identidad causal
  ni fuerzan prueba de colision.

Prioridad media:

- T184 debe empezar por inventario de usos de FNV32, SHA1 truncado y
  `strings.Join` en derivaciones deterministas. Si una huella solo es
  diagnostica, basta documentar esa frontera; si decide idempotencia o
  causalidad, debe migrar a builder canonico y conflicto reparable.
- T185 debe centralizarse en `scripts/lib` para no duplicar guardas en cada
  smoke. Los scripts pueden mantener su contrato funcional, pero el borrado de
  raiz debe quedar detras de un unico helper probado.

Duplicaciones a evitar:

- T184 no reemplaza el generador de refs ni la persistencia de
  `WorkflowTaskStore`; solo cierra la ambiguedad/colision de huellas
  deterministas usadas para refs o firmas.
- T185 no reabre la politica de instancia temporal ni el shutdown de smokes;
  cubre exclusivamente limpieza local y rutas prohibidas, sin guardar HOME,
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
- Resolver T187 antes de exponer visor o export de auditoria. T171 evita query
  cruda y T132 clasifica privacidad, pero `remote_addr` y headers de proxy son
  identidad operativa y no deben quedar como payload estable sin perfil.

Prioridad media:

- Resolver T188 junto a T173/T176 cuando se endurezcan rutas web o MCP. Metodo,
  `Allow` y `OPTIONS` no sustituyen CSRF ni headers de seguridad, pero evitan
  que cada handler tenga una semantica distinta para clientes, probes y
  navegadores.
- T186 debe coordinarse con T20 y T101: cambiar refs de job sin compatibilidad
  rompe replay, ledgers de submit y smokes OPES/no-OPES.
- T187 debe coordinarse con T55: audit puede clasificar origen, pero la decision
  de bind/autorizacion sigue en control plane, no en cabeceras declaradas por el
  cliente.

Duplicaciones a evitar:

- T186 no reabre todos los hashes deterministas del repo; cubre solo identidad
  de jobs `domain_work` y sus stores memory/file/SQL.
- T187 no crea autenticacion, geolocalizacion ni tracking de clientes; solo
  redaccion/clasificacion de identidad en auditoria HTTP.
- T188 no decide origen/intencion ni CORS amplio; solo contrato de metodos,
  `Allow`, `OPTIONS` y error publico por perfil.

## Priorizacion scanner 2026-05-24 sexagesima septima pasada

Backlog asociado:
`T189 server-resident-signal-shutdown-policy`,
`T190 http-gateway-route-manifest-collision-guard` y
`T191 process-runtime-stop-signal-escalation-policy`.

Prioridad alta:

- Resolver T189 antes de tratar el servidor residente como daemon de sistema:
  T96 valida identidad de proceso y T166 espera quiescencia interna, pero sin
  politica de SIGTERM/segunda senal/timeout el operador no sabe si el cierre fue
  cooperativo, escalado o parcial.
- Resolver T190 antes de sumar mas endpoints bajo `/api/v0/apps/` o overlays de
  gateway. T169 protege el registro MCP y T188 metodo/OPTIONS; el riesgo nuevo
  es dispatch a handler equivocado por prefijo o colision no declarada.

Prioridad media-alta:

- Resolver T191 junto a T66: el E2E neutral de stop necesita una politica de
  senal/escalado real para procesos que no cooperan, con estado observable y sin
  filtrar argv/env/stdout/stderr.

Duplicaciones a evitar:

- T189 no reabre checkpoint de shutdown ni stop de agentes; gobierna senales del
  servidor residente foreground/daemon.
- T190 no sustituye catalogos publicos, docs de rutas ni CSRF; solo verifica
  manifest, precedencia y colisiones de gateway HTTP.
- T191 no cambia launch env/io ni stop de olas Codex; solo define deadline y
  escalation del conector neutral de procesos.

## Priorizacion scanner 2026-05-24 sexagesima octava pasada

Backlog asociado:
`T192 server-status-operational-message-projection`,
`T193 factory-appspec-time-source-contract` y
`T194 governance-catalog-output-budget-freshness`.

Prioridad alta:

- Resolver T192 antes de exponer o exportar status/auditoria de automejora idle:
  T41 puede aportar redaccion general, pero falta owner para mensajes
  operativos en `StateV0`.
- Resolver T193 antes de ampliar previews/backlog generados desde AppSpec:
  mantener `time.Now()` oculto en el caso de uso dificulta replay y pruebas
  deterministas.

Prioridad media:

- Resolver T194 antes de estabilizar el catalogo publico de gobernanza para
  CLI/MCP/gateway. Si los catalogos siguen chicos puede esperar, pero el limite
  debe estar antes de abrirlo como API durable.

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

- Resolver T195 antes de tratar `/mcp` como transporte estable para agentes
  externos. T174/T175/T182 cubren protocolo, salida y deadline, pero no evitan
  que `InputSchema`/`Output` queden stale frente a DTOs y validadores reales.
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
- Resolver T199 antes de sumar mas handlers MCP/HTTP. Sin catalogo de codigos,
  `metodo_no_permitido`, `not_configured`, `unavailable` y errores de schema
  pueden divergir entre HTTP, JSON-RPC, CLI/web y operador.

Prioridad media:

- T197 debe coordinar con T139 para que stdout/JSON de CLI siga siendo shape
  publico, no body HTTP crudo ni diagnostico local.
- T198 debe coordinar con T175 para que un resource stale, no configurado o
  demasiado grande degrade con reason code compacto.
- T199 debe coordinar con T75/i18n: un codigo publico puede tener fallback
  temporal, pero la clave debe ser estable y testeable.

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
