# Tareas

## SRV-001

Crear handler residente con `healthz`, `/api/v0/server/status`, alias legacy
`/api/status` y delegacion al handler de aplicacion.

## SRV-002

Crear statefile atomico para reconexion tras cortar la sesion de Codex.

## SRV-003

Crear bucle de supervision acotado y testeado.

## SRV-004

Crear comando fino que arranque `run`, `start`, `status` y `stop` sin meter
logica del nucleo en `cmd`.

## SRV-TASK-004: cablear estado durable

Estado: hecho local.

Objetivo: el servidor residente debe arrancar con conectores persistentes para
estado operativo, de forma que una ejecucion de autoprogramacion pueda
reanudarse tras corte o reinicio.

Write-set previsto:

- `cmd/orquesta-server/stack.go`
- modulo adaptador file-based bajo `modulos/`
- tests de arranque/recreacion de stores

Validacion:

- `TestFileStateStoreV0RecuperaStateV0` cubre recuperacion del statefile;
- `TestRuntimeV0RestauraEstadoDurableAlRecrearInstanciaV0` cubre recreacion de
  runtime desde estado durable y reinicio de campos volatiles;
- `TestRuntimeV0PersistStateFailureVisibleSinFiltrarDetallesV0` cubre estado
  degradado observable cuando falla la persistencia;
- `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`;
- ninguna dependencia de DB concreta queda en el servidor.

## SRV-TASK-005: conector OPES opt-in desde cmd

Objetivo: permitir que el servidor productivo inyecte `domain_work` hacia OPES
sin que el runtime residente conozca OPES.

Estado: hecho.

Validacion:

- `ORQUESTA_OPES_BASE_URL` activa el executor REST OPES;
- sin `ORQUESTA_OPES_BASE_URL`, el executor queda `nil`;
- `go test -count=1 ./cmd/orquesta-server`.

## SRV-TASK-006: automejora por capacidad libre

Estado: hecho.

Objetivo: el servidor residente no debe esperar a estar completamente idle para
seguir pensando trabajo de automejora. Si la cola visible esta por debajo del
objetivo configurado, hay capacidad libre y existe planner inyectado, pide nuevas
tareas sin duplicar las que ya estan en cola. Los skips no terminales del tick
cuentan como presion de cola, pero no bloquean por si solos la planificacion si
queda hueco bajo el objetivo.

Validacion:

- `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`;
- el planner recibe `known_request_refs`/`known_run_refs`;
- `cmd/orquesta-server` puede generar una tarea scanner para ampliar backlog
  cuando no quedan tareas nuevas concretas.

## SRV-TASK-007: supervisor residente reentrante

Estado: hecho local.

Origen: `task-ref-self-improvement-afbb86d34eb5`. Refs opacas preservadas:
`worktree-ref-orquesta-local-parallel-01`,
`branch-ref-orquesta-local-parallel-01`.

Objetivo: el supervisor residente debe observar el estado disponible, ejecutar
un pulso acotado y volver al bucle tras cada supervision o preparacion de
automejora. No debe quedar retenido esperando agentes largos ni mezclar trabajo
secundario con el trabajo principal.

Validacion:

- 2026-05-26, `go test -count=1 ./modulos/orquesta-server`.
- `TestRuntimeV0SupervisorAsyncCoalesceaUnTickPendienteV0` demuestra que el
  pulso residente no se solapa, coalescea un unico tick pendiente durante una
  supervision activa y libera el slot al terminar.
- `TestRuntimeV0SupervisorPreparaAutomejoraTrasIdleV0` demuestra que preparar
  automejora en segundo plano no bloquea el siguiente pulso del supervisor.
- `TestRuntimeV0SupervisorPreparaAutomejoraConCapacidadLibreV0` y
  `TestRuntimeV0SupervisorPreparaAutomejoraConCapacidadLibreAunqueHayaSkipsV0`
  cubren automejora por capacidad libre con cola visible y planner inyectado.
- `TestIdleSelfImprovementAuditPayloadRedactaMensajeDeBlockerV0` cubre que la
  auditoria de blockers conserva refs/evidencias compactas y redacta mensajes
  con rutas, HOME, tokens, prompts o transcripts.

## SRV-TASK-008: auditoria startup compacta

Estado: hecho local.

Objetivo: la auditoria JSONL del autodiagnostico de arranque no debe persistir
payloads crudos con paths de proyecto, runtime o state; debe conservar solo una
traza compacta para operador y Director.

Validacion:

- 2026-05-26, `go test -count=1 ./modulos/orquesta-server -run 'TestRuntimeV0PrepareStartup'`.
- `TestRuntimeV0PrepareStartupAuditaSummarySinPathsV0` demuestra que
  `startup_check_ready` usa `command_summary`/`result_summary`, no `command` ni
  `result` crudos, y no serializa paths sensibles.

## SRV-TASK-009: visibilidad de fallo de persistencia de estado

Estado: hecho local.

Origen: T95 `server-state-persist-failure-visibility`.

Objetivo: los fallos no fatales del `StateStorePortV0` posteriores al arranque
no deben quedar ocultos. El status residente debe exponer una proyeccion
compacta y redactada con contador, codigo publico, transicion afectada y ultimo
instante observado.

Validacion:

- `go test -count=1 ./modulos/orquesta-server`;
- `TestRuntimeV0PersistStateFailureVisibleSinFiltrarDetallesV0` cubre
  `state_persist_failed` en memoria/status sin rutas locales ni error crudo del
  store;
- `TestRuntimeV0PersistStateConfirmedRecuperaEstadoDegradadoV0` cubre que una
  persistencia posterior correcta marca `state_persist_status=ok` y conserva el
  contador historico de fallos.

## SRV-TASK-010: transportar progreso vivo T210 sin recalcular cierre

Estado: cerrado por reconciliacion documental T210.

Objetivo: el servidor residente y `cmd/orquesta-server` deben exponer la
proyeccion de stats generada por la capa de orquestacion sin degradarla a 0% ni
mezclar entrega con cierre.

Contrato:

- `TasksClosed` sigue saliendo de cierre/review aceptada.
- `percent_complete`, `progress_source` y senales de agente/proceso llegan ya
  saneadas desde `DirectorRunStatsV0`.
- El servidor no lee runtime, filesystem, HOME, proveedor, prompts ni logs para
  recomputar progreso.

Validacion:

- `go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-server ./modulos/orquesta-web ./cmd/orquesta-server`.

## SRV-TASK-011: reconciliacion T208 guardian

Estado: documentado 2026-05-27.

Objetivo: reflejar que el servidor residente ya trata el guardian break-glass
como opt-in estructurado, no como pendiente generico de backlog.

Validacion:

- `go test -count=1 ./modulos/orquesta-server`
- bateria T208 cruzada del paquete OrquestaV2.

Frontera:

- El servidor consume resultado estructurado y conserva retry seguro.
- `cmd/orquesta-guardian` conserva build/test/readiness/artefactos/repair.
- Codex, HOME, proveedor y paths locales no entran en `orquesta-server`.

## SRV-TASK-012: loop residente opt-in del Director

Estado: hecho local con adaptador real opt-in desde `cmd/orquesta-server`.

Objetivo: permitir que el proceso residente ejecute el Director autonomo sin
meter el nucleo, Codex, OPES, MCP ni proveedores dentro del servidor.

Implementado:

- `ResidentDirectorPortV0`, `ResidentDirectorCommandV0` y
  `ResidentDirectorResultV0`;
- `RuntimeDepsV0.ResidentDirector` y `ConfigV0.ResidentDirectorEnabled` como
  opt-in explicito;
- loop async con anti-solape, coalescing y recuperacion de panic;
- estado publico `resident_director_*`, contadores operacionales y actividad;
- self-watchdog reconoce ticks, progreso y errores del Director residente;
- `cmd/orquesta-server` lee
  `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED` y
  `ORQUESTA_SERVER_RESIDENT_DIRECTOR_MAX_ACTIONS`, publica ambos en
  `effective_config` e inyecta `ResidentDirectorPortV0` solo con opt-in;
- el adaptador real delega en el stack Codex y ejecuta
  `RunResidentDirectorBriefingLoopV0` sobre stores vivos;
- el adaptador respeta `MaxRunsPerTick`/`MaxExecutions`, no se queda en un unico
  run candidato por pulso.

Validacion:

- `go test -count=1 ./modulos/orquesta-server`
- `go test -count=1 ./cmd/orquesta-server`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'CodexStackResidentDirector'`

Pendiente separado:

- smoke largo real con servidor residente opt-in y cola amplia;
- consejo/votacion y resolucion de `SkillRefs` por rol pertenecen al Director,
  no a `orquesta-server`.

## SRV-TASK-013: OPES no puede arrancar sin Director residente

Estado: hecho local 2026-06-13.

Origen: incidencia OPES A2 Informatica con servidor `127.0.0.1:8792` arrancado
con `ORQUESTA_OPES_PROJECT_WORKDIR` y
`ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false`.

Objetivo: un servidor con contexto OPES no debe quedar en modo semiautomatico
sin Director residente. Si OPES esta activo, la autonomia efectiva debe activar
el Director residente por defecto; si alguien fuerza
`ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false`, el arranque debe fallar con
error accionable. El daemon arrancado por `start` debe recibir explicitamente la
configuracion efectiva de autonomia y el status publico no debe mostrar un `ok`
historico como si el loop siguiera activo.

Validacion:

- `TestServerConfigFromEnvV0ContextoOPESActivaDirectorResidenteV0`;
- `TestServerConfigFromEnvV0BloqueaOPESConDirectorResidenteApagadoV0`;
- `TestServerDaemonStartEnvironmentV0ProyectaDirectorResidenteEfectivo`;
- `TestResidentDirectorV0StatusPublicoNoReusaOkHistoricoSiEstaDesactivado`.

## SRV-TASK-014: app-change despierta supervision residente

Estado: hecho local 2026-06-13.

Objetivo: persistir una `app-change` aceptada debe despertar al supervisor y al
Director residente. La app-change aceptada no es materializacion: la
materializacion ocurre cuando el supervisor/director vuelve a drenar decisiones.
Por tanto el store durable debe emitir wakeup desde el composition root del
servidor, sin acoplar handlers HTTP con `/api/v0/runs/supervise`.

Validacion:

- `TestServerWakeupAppChangeStoreV0DisparaAlGuardarSolicitud`;
- `TestServerSupervisorWakeupDecoratorsV0DisparanSoloTrasMutacionesEjecutables`.

## SRV-TASK-015: productor causal OPES de rework y followups

Estado: hecho local 2026-06-13.

Objetivo: crear un adaptador OPES opt-in que convierta artefactos y receipts ya
aceptados o rechazados en nuevos `DomainWorkJobRequestV0` deduplicados. Debe
cubrir al menos `document_plan` aceptado, `director_review_matrix` con
`needs_rework`, `completed_syllabus_package` con `pendiente_continuar` y
`followup_refs`, y receipts rechazados. No debe vivir en el nucleo puro: debe
apoyarse en puertos `domain-work`, `orquesta-opes-bridge` y contratos OPES.

Implementado:

- `modulos/orquesta-opes-director` como adaptador causal OPES fuera del nucleo;
- el ledger de entregas conserva `correlation_id`, `summary`, payload,
  `payload_refs` y `external_refs` para poder reanudar desde artefactos
  duraderos;
- el Director residente ejecuta el productor causal OPES antes de su drenaje
  normal;
- el ledger de artefactos despierta supervisor y Director residente al guardar
  una entrega;
- `document_plan` aceptado expande trabajos derivados y pide rework si no cubre
  la secuencia completa OPES;
- paquetes finales con `pendiente_continuar`, `followup_refs`,
  `pending_followup_refs`, `rework_refs` o `missing_required_refs` crean la
  siguiente tarea causal;
- receipts rechazados crean una correccion de la misma fase cuando existe
  `source_work_kind`;
- idempotencia por `idempotency_key` y, si el backend expone job records,
  preconsulta para no relanzar el mismo trabajo.

Validacion:

- `TestProduceOPESCausalJobsV0ExpandeDocumentPlanYEsIdempotente`;
- `TestProduceOPESCausalJobsV0PaquetePendienteCreaFollowupSinCerrar`;
- `TestProduceOPESCausalJobsV0RejectedCreaCorreccionMismaFase`;
- `TestServerOPESCausalProducerV0CreaFollowupDesdeLedgerYNoDuplica`;
- `TestServerWakeupDomainWorkArtifactSubmissionLedgerV0DisparaAlRegistrar`;
- `go test -count=1 ./modulos/orquesta-opes-director`;
- `go test -count=1 ./cmd/orquesta-server -run 'OPESCausal|WakeupDomainWork|ResidentDirector'`.

## SRV-TASK-016: bridge OPES sin supervise manual por defecto

Estado: hecho local 2026-06-13.

Objetivo: el bridge OPES no debe depender de una llamada inmediata a
`/api/v0/runs/supervise` para avanzar. En modo autonomo, crear o reencontrar un
run debe bastar: el wakeup y el Director residente se encargan del drenaje. La
supervision directa queda solo como compatibilidad legacy con opt-in explicito.

Implementado:

- `ORQUESTA_OPES_BRIDGE_SUPERVISE_SUBMITTED=1` conserva el comportamiento
  anterior para smokes antiguos o diagnostico dirigido;
- sin esa variable, el bridge registra `supervision_status=resident_director_pending`
  y no llama a `/api/v0/runs/supervise`;
- los smokes legacy que simulaban progreso OPES desde `/runs/supervise` activan
  el opt-in de forma explicita.

Validacion:

- `TestRunOPESDrainOnceV0NoSupervisaPorDefectoTrasEnviarV0`;
- `go test -count=1 ./cmd/orquesta-server -run 'OPESDrain|ExternalBridgeInput|OPESBridge|OPESTemarioCycle'`.

## SRV-TASK-017: registro OPES por tema como trabajo causal

Estado: hecho local 2026-06-13.

Objetivo: evitar que los padres OPES queden parados porque el registro global de
temas está fuera de su write-set. Orquesta debe pedir una actualización causal
del registro para `course_id + topic_id` cuando reciba una entrega aceptada de
tema, sin reabrir el write-set del agente de contenido ni editar JSON bruto
desde el nucleo.

Implementado:

- nuevo artefacto `topic_registry_update` en `domain-work`;
- nuevo work kind OPES `update_topic_registry`;
- `update_topic_registry` entra en la secuencia completa OPES justo después de
  `plan_temario`;
- el productor causal OPES crea jobs `update_topic_registry` desde entregas
  aceptadas con `course_id` y `topic_id`;
- el propio artefacto `topic_registry_update` no vuelve a disparar otro job de
  registro;
- nuevo módulo `orquesta-opes-topic-registry` para construir y ejecutar
  `update`/`release` contra la herramienta oficial `registro_trabajo_temas.py`
  mediante runner inyectado;
- `cmd/orquesta-server` cablea el conector con opt-in por
  `ORQUESTA_OPES_TOPIC_REGISTRY_ENABLED` o
  `ORQUESTA_OPES_TOPIC_REGISTRY_TOOL_PATH`;
- si `ORQUESTA_OPES_PROJECT_WORKDIR` apunta a OPES y contiene la herramienta
  oficial, el servidor descubre el tool path sin otro env adicional;
- `effective_config` publica presencia redactada del tool path y no filtra la
  ruta local;
- idempotencia por fuente, artefacto, receipt y scope del tema.

Validación:

- `TestExpectedDomainWorkArtifactTypeForWorkKindV0`;
- `TestOPESFullTemarioJobTypeSequenceV0IncluyeCierreCompletoV0`;
- `TestProduceOPESCausalJobsV0CreaActualizacionRegistroPorTema`;
- `TestProduceOPESCausalJobsV0NoRepiteRegistroDesdeRegistro`;
- `TestTopicRegistryCLIArgsV0UpdateConEvidencias`;
- `TestApplyTopicRegistryUpdateV0EjecutaRunnerInyectado`;
- `TestTopicRegistryUpdateRequestFromDomainWorkJobV0`;
- `TestServerOPESTopicRegistryUpdaterV0AplicaDesdeDomainWork`;
- `TestOPESTopicRegistryEffectiveConfigRedactaToolPathV0`;
- `TestOPESTopicRegistryConfigDescubreToolDesdeOPESProjectWorkDirV0`;
- `go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-director`.

## SRV-TASK-018: reconciliacion OPES tras timeout y ACK incompleto

Estado: abierto 2026-06-20.

Origen:
`docs/incidencia_opes_servicios_multiples_supervision_ack_2026-06-20.md`.

Objetivo: el bridge OPES y el Director residente deben reconciliar trabajos
enviados aunque el comando de drenaje devuelva timeout o supervision no
disponible, y deben convertir ACKs/artifacts incompletos en normalizacion o
rework causal automatico sin intervencion manual.

Alcance:

- `opes-drain-once` no debe dejar un `retry_pending` indefinido cuando el run
  ya fue creado;
- los claims y ledgers deben detectar `submitted_after_timeout` y reconciliar
  el run existente antes de relanzar;
- el submit OPES debe validar o normalizar el contrato minimo de artefacto:
  `artifact_type`, `complete_job`, `idempotency_key` y payload trazable;
- si el contrato es recuperable pero no publicable, Orquesta debe crear el
  rework causal y despertarlo automaticamente;
- el status publico debe exponer contadores compactos de esos casos sin filtrar
  rutas locales ni errores crudos.

Criterio de cierre:

- smoke real OPES temporal con 17 `draft_content_block`;
- cero jobs atascados por `supervision_unavailable` sin siguiente accion
  durable;
- reworks causales para ACK incompleto se ejecutan sin llamada manual a
  `/api/v0/runs/supervise`;
- pruebas focales del bridge, servidor y productor causal OPES cubren timeout,
  reencontro de run, ACK incompleto y rework automatico.

## SRV-TASK-019: visuales OPES sin FK antes de tema canonico

Estado: abierto 2026-06-20.

Origen:
`docs/incidencia_opes_servicios_multiples_supervision_ack_2026-06-20.md`.

Objetivo: impedir que Orquesta/bridge marque como bloque visual final un
`generate_visual_asset` que solo tiene refs de programa (`program_topic:*`,
`placement_ref`) cuando OPES aun no tiene `canonical_topic_id` y `chapter_id`
materializables. El resultado debe conservarse como artefacto/insumo trazable o
reenviarse cuando existan refs vivas, sin rehacer el visual ni provocar
`FOREIGN KEY constraint failed`.

Alcance:

- antes de llamar a `POST /api/jobs/{id}/artifacts` para tipos que materializan
  bloques, comprobar si el job lleva `canonical_topic_id` y `chapter_id` vivos;
- si solo hay refs de programa, entregar como evidencia no materializable o
  crear una tarea causal de ensamblado/replay posterior;
- reconciliar artefactos ya escritos en `external/opes/generate_visual_asset`
  y jobs OPES `pending` sin relanzar agentes innecesariamente;
- exponer contador compacto `materialization_refs_missing` y siguiente accion
  durable en el status del bridge;
- testear que un visual de programa no produce HTTP 500/FK y no queda
  indefinidamente `pending` sin tarea causal.

Criterio de cierre:

- smoke OPES con curso basado en programa externo, sin `canonical_topics`
  iniciales;
- 17 visuales generados quedan en estado reconciliado: materializados si tienen
  tema/capitulo vivo, o registrados como insumo trazable para ensamblado;
- cero duplicados por reintento de visual ya entregado;
- ninguna entrega valida se descarta por fallo de FK recuperable.
