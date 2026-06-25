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

Reconciliacion backlog 2026-06-20:

- `task-ref-self-improvement-874937b97f16` no abre nueva implementacion para
  este frente: el intento padre previo quedo en checkpoint de apagado antes de
  tocar archivos o ejecutar pruebas;
- `scan-ref-backlog-c108d911342a` queda resuelto contra esta evidencia local,
  el runbook `docs/runbooks/opes_productor_causal_autonomo_2026-06-13.md` y las
  validaciones listadas arriba;
- si el planner vuelve a detectar el patron `productor causal OPES` sin
  regresion causal nueva, debe clasificarlo como `no-op documental` y no
  relanzar otro padre de codigo para SRV-TASK-015.
- el rework de revision
  `agent-ref-task-ref-review-rework-task-autoprogramming-874937b97f16-g01-162db2d326380eeab85c029bbfcfe285`
  solo confirma esta reconciliacion documental con contexto `ref_only` y prueba
  focal del servidor; no cambia el owner de codigo ni abre nueva implementacion.
- el rework de revision sobre rework
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-autoprogr-7b57471b0af67dc475be23b72a6c25c7`
  conserva esa entrega valida, resuelve el contexto `ref_only` por evidencia
  documental y mantiene SRV-TASK-015 cerrado salvo regresion causal nueva con
  refs concretas de job, receipt o artifact.
- el rework de revision materializado como
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-8f75b93913fef84c105ce29cf9734bca`
  corrige solo el rastro causal de la entrega rechazada: conserva el cierre
  documental, no relanza otro padre sobre la tarea original y no abre codigo
  nuevo sin regresion causal concreta.
- el agente externo de reemplazo
  `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-0b2-f385b5e4533d62b5c7d2a64b70950490`
  aplica el mismo contrato de correccion: sincroniza evidencia local, conserva
  la entrega valida anterior y limita el cierre al ACK causal y a la prueba
  focal obligatoria.
- la revalidacion OrquestaV2 del 2026-06-25
  `task-ref-self-improvement-30d589fce14c` mantiene la misma decision: el
  paquete trae contexto obligatorio `ref_only`, write-set cerrado a
  `modulos/orquesta-server`, `backlog_scan_epoch:backlog-scan-epoch-ba0be0570440`
  y `backlog_scan_ref:scan-ref-backlog-7c5facaa6c9e`; esos refs se resuelven
  contra esta evidencia local y no reabren codigo sin regresion causal nueva
  con refs concretas de job, receipt o artifact.
- el agente externo de correccion
  `agent-ref-task-ref-review-rework-task-autoprogramming-30d589fce14c-g01-8bb7a3491960a73cba493ed199be20e1`
  conserva esa decision: completa solo el rastro causal de la revision, resuelve
  `ref_only` por las docs locales de SRV-TASK-015 y cierra con la prueba focal
  obligatoria si termina verde.
- la correccion de entrega tras revision
  `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-dacb62e5ec1ae158a640baa674e72940`
  mantiene el mismo no-op documental: resuelve `ref_only` por evidencia local,
  conserva la entrega valida y no abre codigo salvo regresion causal nueva con
  refs concretas de job, receipt o artifact.
- la correccion externa de entrega tras revision
  `agent-ref-task-ref-review-rework-task-autoprogramming-abea33163b68-g01-04bebd80bd7c5afa1ff0cde8543ec8ec`
  conserva esa entrega valida: completa el rastro causal dentro del write-set
  del servidor, resuelve el contexto obligatorio `ref_only` mediante evidencia
  local y cierra solo con la prueba focal obligatoria verde.
- correccion de rastro 2026-06-25: las refs
  `agent-ref-task-ref-review-rework-task-autoprogramming-99f93b5dadeb-g01-46ac951607ee8f489f914ee25e92cb88`
  y
  `agent-ref-assessment-task-ref-review-rework-task-autoprogramming-99f93b5dadeb-g01-46a-de83dfc8cce6e142d5f9b4a9f2a47b24`
  pertenecen a la request de SRV-TASK-024, no a SRV-TASK-015; no se usan como
  evidencia nueva de este cierre documental.

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

## SRV-TASK-020: advance OPES no debe volver a initial_work cubierto

Estado: abierto 2026-06-20.

Origen:
`docs/incidencia_opes_servicios_multiples_advance_duplica_inicial_2026-06-20.md`.

Objetivo: el avance automático OPES debe reconocer artefactos aceptados por
`correlation_id`, payload `program_id`, `document_plan`, `content_block` y
`visual_asset`, aunque no estén materializados todavía como temas canónicos, y
no debe recrear `research_sources`, `split_syllabus_topic` ni
`draft_topic_outline` cuando esa fase ya está cubierta.

Criterio de cierre:

- smoke OPES con `document_plan`, 17 `content_block` y 17 `visual_asset`
  aceptados;
- llamada a `POST /api/programs/{id}/advance` crea solo fases siguientes reales
  o devuelve `wait/nothing_to_advance`;
- contador público `skipped_initial_work_already_covered`;
- cero jobs iniciales duplicados para temas ya aceptados.

## SRV-TASK-021: external-work convertible no puede cerrar run vacía

Estado: abierto 2026-06-20.

Origen:
`docs/incidencia_opes_servicios_multiples_tutor_empty_run_2026-06-20.md`.

Objetivo: cuando un `external_work` OPES convertible, como
`generate_tutor_assets`, falla en submit o supervisión, Orquesta no debe marcar
la run como `done` con cero tareas, cero agentes y sin artefacto. Debe quedar
`failed_empty_run`, `retry_pending` o rework causal relanzable.

Criterio de cierre:

- smoke real acotado de `generate_tutor_assets`;
- la supervisión nunca devuelve `done` con `tasks=0` para un trabajo
  convertible;
- `opes-drain-once` reconcilia `submit_failed + empty_run` sin crear
  `existing_change_conflict`;
- contador público `external_work_empty_run` y siguiente acción durable.

## SRV-TASK-022: supervise no debe emitir error terminal con agente vivo y ACK tardío

Estado: abierto 2026-06-20.

Origen:
`docs/incidencia_opes_servicios_multiples_supervision_ack_2026-06-20.md`.

Objetivo: `POST /api/v0/runs/supervise` y el bridge OPES deben distinguir
timeout de espera de fallo terminal. Si existe proceso vivo, ACK pendiente,
artefacto local recuperable o submit tardío probable, la respuesta publica debe
ser `wait/reconcile_pending` con siguiente accion durable, no
`runtime_error/blocked`. Ademas, si una recuperacion manual o residente registra
el artefacto antes de que llegue el ACK tardio, el ACK posterior debe
deduplicarse por `run_ref + job_ref + artifact_type + payload/evidence_ref` o
quedar marcado como replay reconciliado.

Criterio de cierre:

- smoke real OPES con un `generate_html_site` que tarde mas que la ventana de
  supervision inicial;
- la primera supervision devuelve estado no terminal y contador
  `external_wait_live_process`;
- el agente termina y el bridge registra un unico artefacto efectivo en OPES;
- si se simula una recuperacion previa con otra idempotency key, el ACK tardio
  no crea un segundo artefacto independiente;
- `director/stats` muestra proceso vivo, ACK pendiente y siguiente accion sin
  requerir inspeccion manual de `.orquesta-runtime`.

## SRV-TASK-023: trabajos OPES de audio requieren red controlada y stop causal

Estado: abierto 2026-06-20.

Origen:
`docs/incidencia_opes_servicios_multiples_supervision_ack_2026-06-20.md`.

Objetivo: los trabajos OPES `generate_audio_asset` que ejecutan `edge-tts` no
pueden lanzarse en una sandbox sin red y sin timeout por comando. Orquesta debe
clasificar la dependencia externa controlada, usar un perfil opt-in que permita
red solo para el runner autorizado o delegar a un runner de audio, y debe
bloquear de forma accionable si falta la herramienta canonica.

Alcance:

- detectar en el contrato del job que `edge-tts` necesita red externa
  controlada y no ejecutar el smoke dentro de `network: restricted`;
- imponer timeout duro por comando TTS y registrar `audio_tts_timeout` sin
  dejar el agente colgado;
- validar tool path efectivo antes de lanzar el agente:
  `scripts/opes_audio_app.py` canonico o wrapper compatible documentado;
- si solo existe `scripts/tcae_audio_app.py`, marcar compatibilidad explicita o
  crear rework causal de herramienta, no improvisar una ruta inexistente;
- prohibir fallback silencioso a `espeak-ng` u otro motor no canonico para
  audio publicable; esos resultados solo pueden ser diagnostico o insumo no
  publicable;
- `POST /api/v0/runs/control action=stop` debe materializar checkpoint en el
  workdir del agente y terminar comandos hijo bloqueados; status debe distinguir
  `stop_requested`, `stop_propagated` y `stop_confirmed`.

Criterio de cierre:

- smoke real OPES con `generate_audio_asset` y `edge-tts` genera un MP3 de
  prueba desde Orquesta sin colgarse;
- si se fuerza red restringida, Orquesta devuelve bloqueo accionable antes de
  ejecutar `edge-tts`;
- si se fuerza herramienta canonica ausente, crea rework causal y no lanza
  agente inutil;
- una run de audio bloqueada por TTS obedece `action=stop`, corta procesos hijo
  y no escribe artefactos nuevos despues de la parada;
- pruebas cubren clasificacion de dependencia externa, timeout de TTS, tool path
  ausente y stop propagado a hijos.

## SRV-TASK-024: run OPES aceptada no puede quedarse en resident_director_pending sin agente

Estado: abierto 2026-06-20.

Origen:
`docs/incidencia_opes_servicios_multiples_supervision_ack_2026-06-20.md`.

Objetivo: cuando `opes-drain-once` crea una run por `JOB_REF` exacto y devuelve
`status=submitted` con `supervision_status=resident_director_pending`, el
supervisor residente debe arrancar o reencontrar agente sin que el operador
tenga que llamar manualmente a `/api/v0/runs/supervise`. Si por capacidad,
lock, ledger o cola no puede hacerlo, debe publicar estado accionable y
siguiente accion durable.

Caso observado:

- job OPES: `b305970086cb77343865513f32df7a66`;
- tipo: `plan_temario`;
- run: `run-external-work-opes-b305970086cb77343865513f32df7a66-opes-job-b305970086cb77343865513f32df7a66`;
- `opes-drain-once` acepto la run en loopback con filtro por `JOB_REF`;
- tras mas de 30 segundos, el job OPES siguio `pending`, no aparecio proceso
  Codex nuevo y el servidor no expuso progreso residente visible.

Alcance:

- pulso residente debe barrer runs en `resident_director_pending` creadas por
  bridge OPES y convertirlas en dispatch real o en bloqueo causal;
- deduplicar por `run_ref + job_ref` y no crear runs repetidas;
- exponer contador publico `resident_pending_without_dispatch` y ultima razon;
- `opes-drain-once` debe poder devolver `supervision_status=started` si el
  supervisor residente ya despacha dentro de la ventana configurada;
- no depender de inspeccion manual de procesos para saber si la run avanza.

Criterio de cierre:

- smoke OPES acotado con `plan_temario` por `JOB_REF` exacto;
- tras `opes-drain-once`, sin llamada manual a `/runs/supervise`, aparece agente
  real o bloqueo causal publico antes de dos ticks residentes;
- el job OPES termina o queda con error accionable sin que el operador tenga que
  despertar el supervisor;
- replay del mismo `JOB_REF` no duplica run ni agente.

Reconciliacion documental 2026-06-25:

- `agent-ref-task-autoprogramming-99f93b5dadeb-g01` quedo bloqueado por
  shutdown antes de editar o probar; no cierra este owner.
- `agent-ref-task-ref-review-rework-task-autoprogramming-99f93b5dadeb-g01-46ac951607ee8f489f914ee25e92cb88`
  y
  `agent-ref-assessment-task-ref-review-rework-task-autoprogramming-99f93b5dadeb-g01-46a-de83dfc8cce6e142d5f9b4a9f2a47b24`
  solo aportaron rastro documental y no declararon la prueba obligatoria como
  pasada.
- `agent-ref-task-ref-review-rework-task-ref-review-rework-task-autoprogr-dc31be1188c5569b01263b5f388f0788`
  corrige la asociacion causal de esas refs a SRV-TASK-024. El estado tecnico
  sigue abierto: el cierre real requiere codigo que convierta
  `resident_director_pending` en dispatch o bloqueo causal publico y smoke OPES
  acotado; esta correccion no relanza padre ni amplia write-set.
- `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-autoprogr-dc3-0ff5f84c9556672a698237df3988111c`
  conserva esa correccion tras nueva evaluacion externa: contexto obligatorio
  `ref_only` resuelto por paquete y docs locales, sin relanzar padre ni cerrar
  SRV-TASK-024 por rastro documental.
- `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-autoprogr-dc3-36f24a2209e93ca38f4443ce22f527cf`
  reemplaza la evaluacion rechazada sin cambiar el owner tecnico: conserva la
  entrega documental valida, resuelve `ref_only` por el paquete de control y
  las docs locales, y mantiene SRV-TASK-024 abierto hasta que exista dispatch
  real o bloqueo causal publico con smoke OPES acotado.
- `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-autoprogr-dc3-a5a32e27fa8bf23a4b29ee07490e9a10`
  corrige la entrega rechazada dentro del mismo alcance documental: conserva lo
  valido, resuelve `ref_only` por paquete y fuentes locales, no relanza otro
  padre y mantiene SRV-TASK-024 abierto hasta evidencia tecnica causal.
- `agent-ref-task-ref-review-rework-task-ref-review-rework-task-autoprogr-72998ee36295059452b54a9eb4e875ee`
  completa el rework documental de burst 003: conserva la entrega valida ya
  acumulada, resuelve `ref_only` por paquete y docs locales, no relanza padre
  ni abre codigo dentro de este write-set, y mantiene SRV-TASK-024 abierto hasta
  dispatch real o bloqueo causal publico con smoke OPES acotado.
- `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-autoprogr-729-9fc3f13d44dbaff94cfe5adff747a3e9`
  valida la correccion anterior como agente externo gobernado por OrquestaV2:
  conserva la entrega documental util, resuelve `ref_only` por paquete de
  control y fuentes locales, no relanza padre ni abre implementacion, y mantiene
  SRV-TASK-024 abierto hasta dispatch real o bloqueo causal publico con smoke
  OPES acotado.
- `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-cfd9c13a92a5cd21fd6c622e6f405e46`
  corrige esta entrega rechazada sin ampliar alcance: conserva la evidencia
  documental util, resuelve `ref_only` por paquete de control y docs locales, no
  relanza otro padre ni toca codigo, y mantiene SRV-TASK-024 abierto hasta
  dispatch real o bloqueo causal publico con smoke OPES acotado.
- `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-cfd-c006963b24043636d557f12d4659b4a3`
  corrige la entrega externa asociada al mismo tramo de revision: conserva el
  rastro documental valido, resuelve `ref_only` por paquete de control y docs
  locales, no relanza padre ni abre codigo dentro del write-set documental, y
  mantiene SRV-TASK-024 abierto hasta dispatch real o bloqueo causal publico
  probado por smoke OPES acotado.
- `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-cfd-791ce4584f28597b364194874b2875a4`
  reemplaza el assessment rechazado sin ampliar alcance: conserva el rastro
  documental valido, resuelve `ref_only` por paquete de control y fuentes
  locales, no relanza padre ni abre codigo dentro del write-set documental, y
  mantiene SRV-TASK-024 abierto hasta dispatch real o bloqueo causal publico
  probado por smoke OPES acotado.
- `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-99773dd28e4df93406b2912127997d04`
  corrige la entrega rechazada de esta cadena sin ampliar alcance: conserva la
  evidencia documental util, resuelve `ref_only` por paquete de control,
  AGENTS raiz, README/AGENTS locales del servidor y docs locales, no relanza
  padre ni toca codigo, y mantiene SRV-TASK-024 abierto hasta dispatch real o
  bloqueo causal publico probado por smoke OPES acotado.
- `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-30ffd078ad6ee9febe4c50ac0bd29891`
  completa la correccion de entrega nueva sin ampliar alcance: conserva el
  rastro documental valido, resuelve `ref_only` por paquete de control, AGENTS
  raiz, README/AGENTS locales del servidor y docs locales, no relanza padre ni
  toca codigo, y mantiene SRV-TASK-024 abierto hasta dispatch real o bloqueo
  causal publico probado por smoke OPES acotado.
- `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-30f-c4501a4d6b170b706b55f9a3dd4a44a6`
  completa el assessment de esa correccion sin ampliar alcance: conserva el
  rastro documental valido, resuelve `ref_only` por paquete de control, AGENTS
  raiz, README/AGENTS locales del servidor y docs locales, no relanza padre ni
  toca codigo, y mantiene SRV-TASK-024 abierto hasta dispatch real o bloqueo
  causal publico probado por smoke OPES acotado.
- `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-30f-aad45cf56f890e04e96c6b86b7c9bc7b`
  corrige la entrega de replan por evaluacion sin ampliar alcance: conserva el
  rastro documental valido, resuelve `ref_only` por paquete de control, AGENTS
  raiz, README/AGENTS locales del servidor y docs locales, no relanza padre ni
  toca codigo, y mantiene SRV-TASK-024 abierto hasta dispatch real o bloqueo
  causal publico probado por smoke OPES acotado.
- `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-30f-6842075f2c746477bb06b52d2c34e0c5`
  completa la correccion de entrega tras revision sin ampliar alcance:
  conserva el rastro documental valido, resuelve `ref_only` por paquete de
  control, AGENTS raiz, README/AGENTS locales del servidor y docs locales, no
  relanza padre ni toca codigo, y mantiene SRV-TASK-024 abierto hasta dispatch
  real o bloqueo causal publico probado por smoke OPES acotado.
- `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-9ba51f68ffa908088c91ca219cb73b84`
  completa la correccion de entrega tras revision sin ampliar alcance: conserva
  el rastro documental valido, resuelve `ref_only` por paquete de control,
  AGENTS raiz, README/AGENTS locales del servidor y docs locales, no relanza
  padre ni toca codigo, y mantiene SRV-TASK-024 abierto hasta dispatch real o
  bloqueo causal publico probado por smoke OPES acotado.
- `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-5f1990c9bda61f11e4c4572d7d8ec5f9`
  completa esta correccion de entrega tras revision sin ampliar alcance:
  conserva el rastro documental valido, resuelve `ref_only` por paquete de
  control, AGENTS raiz, README/AGENTS locales del servidor y docs locales, no
  relanza padre ni toca codigo, y mantiene SRV-TASK-024 abierto hasta dispatch
  real o bloqueo causal publico probado por smoke OPES acotado.
- `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2f664d21881795ee2d8392327887480f`
  completa esta correccion de entrega tras revision sin ampliar alcance:
  conserva el rastro documental valido, resuelve `ref_only` por paquete de
  control, AGENTS raiz, README/AGENTS locales del servidor y docs locales, no
  relanza padre ni toca codigo, y mantiene SRV-TASK-024 abierto hasta dispatch
  real o bloqueo causal publico probado por smoke OPES acotado.
- `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2f6-4a88d678723f47ef86524e438127a1e1`
  completa esta evaluacion de correccion tras revision sin ampliar alcance:
  conserva el rastro documental valido, resuelve `ref_only` por paquete de
  control, AGENTS raiz, README/AGENTS locales del servidor y docs locales, no
  relanza padre ni toca codigo, y mantiene SRV-TASK-024 abierto hasta dispatch
  real o bloqueo causal publico probado por smoke OPES acotado.
- `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2f6-d07cfaec65f49155f8328d0512377fd7`
  corrige la entrega rechazada de esa evaluacion sin ampliar alcance: conserva
  el rastro documental valido, resuelve `ref_only` por paquete de control,
  AGENTS raiz, README/AGENTS locales del servidor y docs locales, no relanza
  padre ni toca codigo, y mantiene SRV-TASK-024 abierto hasta dispatch real o
  bloqueo causal publico probado por smoke OPES acotado.
- `agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2f6-20f9ab529a94f082f863a30e7a9275dd`
  corrige la evaluacion anterior sin ACK y no amplia alcance: conserva el
  rastro documental valido, resuelve `ref_only` por paquete de control, AGENTS
  raiz, README/AGENTS locales del servidor y docs locales, no relanza padre ni
  toca codigo, y mantiene SRV-TASK-024 abierto hasta dispatch real o bloqueo
  causal publico probado por smoke OPES acotado.
- `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-d20f765b96bef254281f4683e6ab3480`
  completa esta correccion de entrega tras revision sin ampliar alcance:
  conserva el rastro documental valido, resuelve `ref_only` por paquete de
  control, AGENTS raiz, README/AGENTS locales del servidor, foto vigente y docs
  locales, no relanza padre ni toca codigo, y mantiene SRV-TASK-024 abierto
  hasta dispatch real o bloqueo causal publico probado por smoke OPES acotado.
- `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-9e9d33b777c6565a38a9b5a6eb7529a2`
  completa esta correccion de entrega tras revision sin ampliar alcance:
  conserva el rastro documental valido, resuelve `ref_only` por paquete de
  control, AGENTS raiz, README/AGENTS locales del servidor, foto vigente y docs
  locales, no relanza padre ni toca codigo, y mantiene SRV-TASK-024 abierto
  hasta dispatch real o bloqueo causal publico probado por smoke OPES acotado.
- `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2b5521cd1c1dbd4767cebb3d9c1684d7`
  completa esta correccion de entrega tras revision sin ampliar alcance:
  conserva el rastro documental valido, resuelve `ref_only` por paquete de
  control, AGENTS raiz, README/AGENTS locales del servidor, foto vigente y docs
  locales, no relanza padre ni toca codigo, y mantiene SRV-TASK-024 abierto
  hasta dispatch real o bloqueo causal publico probado por smoke OPES acotado.
- `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-403a707ea7e18c0c29793d3df9266ede`
  completa esta correccion de entrega tras revision sin ampliar alcance:
  conserva el rastro documental valido, resuelve `ref_only` por paquete de
  control, AGENTS raiz, README/AGENTS locales del servidor, foto vigente y docs
  locales, no relanza padre ni toca codigo, y mantiene SRV-TASK-024 abierto
  hasta dispatch real o bloqueo causal publico probado por smoke OPES acotado.
- `agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-026aefcaa6977288909f6693516c13c1`
  completa esta correccion de entrega tras revision sin ampliar alcance:
  conserva el rastro documental valido, resuelve `ref_only` por paquete de
  control, AGENTS raiz, README/AGENTS locales del servidor, foto vigente y docs
  locales, no relanza padre ni toca codigo, y mantiene SRV-TASK-024 abierto
  hasta dispatch real o bloqueo causal publico probado por smoke OPES acotado.

## SRV-TASK-025: reconciliacion de ACK OPES no debe terminar en domain_work_submit_conflict sin publicar artefacto

Estado: abierto 2026-06-21.

Origen:
`docs/incidencia_opes_servicios_multiples_supervision_ack_2026-06-20.md`.

Objetivo: si una run OPES ya tiene ACK completado, test requerido pasado y
artefacto local valido, una llamada posterior a `POST /api/v0/runs/supervise`
debe reconciliar esa entrega con OPES o devolver una accion durable concreta.
No puede acabar en `runtime_error` con diagnostico `domain_work_submit_conflict`
mientras el job OPES sigue `pending` y sin artefactos.

Caso observado:

- job OPES: `b305970086cb77343865513f32df7a66`;
- tipo: `plan_temario`;
- artefacto local:
  `external/opes/plan_temario/b305970086cb77343865513f32df7a66/artifact_document_plan.json`;
- ACK retry materializado con prueba
  `opes-domain-test-plan_temario-b305970086cb77343865513f32df7a66` pasada;
- `runs/supervise` posterior encontro `open_tasks=0` y
  `requested_agents=2`, pero termino con
  `run_supervisor_execute_error` y diagnostico
  `domain_work_submit_conflict`;
- OPES siguio mostrando el job como `pending` y
  `/api/jobs/{id}/artifacts` siguio devolviendo `null`.

Alcance:

- deduplicar replays por `run_ref + job_ref + artifact_type +
  artifact_path/idempotency_key`;
- cuando haya ACK valido y artefacto local, intentar submit idempotente a OPES
  o registrar `reconcile_manual_required` con ruta y causa exacta;
- si existe conflicto, exponer que entidad conflictua y que accion debe tomar
  el director residente;
- no marcar la run como `failed` sin conservar estado de reconciliacion
  pendiente recuperable;
- añadir contador publico `domain_work_submit_conflict_recoverable`.

Criterio de cierre:

- smoke OPES con ACK tardio/retry y artefacto local ya escrito;
- una segunda supervision registra exactamente un artefacto en OPES o deja
  estado durable `reconcile_manual_required`;
- el job OPES deja de quedar `pending` silenciosamente;
- replay del mismo ACK no duplica artefactos ni lanza otro agente innecesario;
- el diagnostico incluye ruta del artefacto, job_ref, tipo de artefacto e
  idempotency efectiva.

## SRV-TASK-026: supervisor OPES no puede cerrar quiescent sin tareas y dejar job pending

Estado: abierto 2026-06-21.

Origen:
`docs/incidencia_opes_servicios_multiples_supervision_ack_2026-06-20.md`.

Objetivo: una run OPES creada por `opes-drain-once` no puede quedar como
aparentemente terminada con `status=done`, evidencias `quiescent`,
`projection-tasks-0` y `projection-open-tasks-0` si el job OPES asociado sigue
`pending` y sin artefacto. Ese estado es una falsa terminacion del supervisor y
debe convertirse en despacho real, reconciliacion de entrega local o bloqueo
causal publico.

Caso observado:

- job OPES: `b8702608bde2d7ebc6ca18f53d65495e`;
- tipo: `validate_topic`;
- run:
  `run-external-work-opes-b8702608bde2d7ebc6ca18f53d65495e-opes-job-b8702608bde2d7ebc6ca18f53d65495e`;
- `opes-drain-once` devolvio `submitted` y
  `supervision_status=resident_director_pending`;
- una llamada manual a `POST /api/v0/runs/supervise` devolvio `estado=ok`,
  `stop_reason=done`, `ticks=1`, `quiescent`,
  `projection-tasks-0`, `projection-open-tasks-0` y
  `projection-requested-agents-0`;
- despues de esa respuesta el job OPES seguia `pending`, `attempts=0`,
  `last_error=""` y no existia artefacto de validacion.

Segundo caso observado en el mismo flujo:

- job OPES: `28c7d0694eeb31f8fb65e5b54cefa9fa`;
- tipo: `generate_help_manual_assets`;
- run:
  `run-external-work-opes-28c7d0694eeb31f8fb65e5b54cefa9fa-opes-job-28c7d0694eeb31f8fb65e5b54cefa9fa`;
- `opes-drain-once` devolvio `submitted` y
  `supervision_status=resident_director_pending`;
- `POST /api/v0/runs/supervise` devolvio de nuevo `estado=ok`,
  `stop_reason=done`, `ticks=1`, `quiescent`,
  `projection-tasks-0`, `projection-open-tasks-0` y
  `projection-requested-agents-0`;
- el job OPES continuo `pending`, `attempts=0`, `last_error=""` y sin
  artefacto de manual.
- despues de que el operador completase el job manualmente por API con un
  `help_manual_package` validado, aparecio tarde un agente real de esa misma
  run, empezo a escribir ficheros auxiliares y pudo duplicar o sobrescribir la
  entrega; se solicito `runs/control action=stop` con idempotency key y el
  proceso termino tras `stop_requested` con `checkpoint_recorded=true`.

Alcance:

- antes de devolver `done/quiescent`, comprobar el estado del job OPES asociado;
- si el job externo sigue `pending`, no cerrar la run como terminada: crear
  tarea de agente, reencontrar artefacto local o publicar bloqueo causal;
- exponer diagnostico `quiescent_without_domain_progress` con `job_ref`,
  `work_kind`, `run_ref` y accion siguiente;
- proteger especialmente jobs de cierre (`validate_topic`, `assemble_topic`,
  `finalize_temario_package`) porque no pueden desaparecer de la cola por falta
  de tareas internas;
- si aparece un agente tardio despues de que el job de dominio ya este
  `completed`, debe entrar en modo reconciliacion/no-op y no escribir un segundo
  artefacto ni sobrescribir la entrega aceptada;
- cubrir el caso con prueba de bridge OPES donde `projection.tasks=0` pero el
  job externo sigue pendiente.

Criterio de cierre:

- smoke OPES con `validate_topic` por `JOB_REF` exacto;
- `opes-drain-once` seguido de supervision no devuelve `done/quiescent` si el
  job OPES continua `pending`;
- el resultado es agente lanzado, artefacto reconciliado o estado durable
  `quiescent_without_domain_progress`;
- replay del mismo job no duplica runs ni deja estados silenciosos;
- el operador no necesita inspeccionar manualmente procesos ni el API OPES para
  descubrir que el trabajo no avanzo.

## SRV-TASK-027: automejora idle no debe arrancar durante sesiones OPES sin opt-in

Estado: abierto 2026-06-22.

Origen:
`docs/incidencia_opes_autonomia_tractorista_ack_idle_stop_2026-06-22.md`.

Objetivo: el residente no debe lanzar automejoras de Orquesta por idle mientras
una sesion OPES acotada esta activa o acaba de cerrar, salvo opt-in explicito
del operador. Si el servidor esta usando `ORQUESTA_OPES_PROJECT_WORKDIR`,
`ORQUESTA_CODEX_PROJECT_WORKDIR` apunta a salidas OPES o hay runs OPES recientes,
la automejora idle debe quedar desactivada, en cola pausada o en estado
`idle_self_improvement_suppressed_by_domain_session`.

Caso observado:

- tras cerrar la run OPES de `operario-tractorista-grupo-5`, el servidor lanzo
  `request-ref-autoprogramming-backlog-scanner-7e8a6ae9`;
- la tarea tenia write-set en docs de Orquesta, no en el curso OPES;
- habia otro agente humano trabajando en el nucleo, por lo que el arranque podia
  pisar cambios ajenos.

Alcance:

- anadir guardia de dominio para idle self-improvement;
- hacer opt-in explicito con env/config canonica para permitir automejora en una
  sesion OPES;
- publicar en status/diagnostico por que se suprimio la automejora;
- no perder tareas de automejora: quedan aplazadas o visibles, no ejecutadas
  silenciosamente sobre otro write-set;
- cubrir que el cierre de una run OPES no rankea automaticamente automejora de
  Orquesta en el mismo servidor.

Criterio de cierre:

- con `ORQUESTA_OPES_PROJECT_WORKDIR` configurado y sin opt-in, una sesion OPES
  que queda idle no arranca agentes de automejora de Orquesta;
- con opt-in explicito, la automejora arranca y aparece claramente en stats;
- `director/stats` o `autoprogramming/status` muestra
  `idle_self_improvement_suppressed_by_domain_session` sin rutas locales;
- smoke OPES temporal demuestra cero procesos extra tras cierre de la run.

## SRV-TASK-028: stop de run debe confirmar parada real de procesos Codex

Estado: abierto 2026-06-22; avance local 2026-06-25 en `orquesta-server` y
`orquesta-app-codex-stack`.

Origen:
`docs/incidencia_opes_autonomia_tractorista_ack_idle_stop_2026-06-22.md`.

Objetivo: `POST /api/v0/runs/control action=stop forced=true` seguido de
`POST /api/v0/runs/supervise` no puede devolver `stop_reason=stopped` si quedan
procesos Codex reales vivos de esa run. Debe propagar parada al runtime,
confirmar `stop_confirmed` por `process_ref + session_ref` o devolver
`stop_pending` con siguiente accion durable.

Caso observado:

- run de automejora:
  `request-ref-autoprogramming-backlog-scanner-7e8a6ae9`;
- `runs/control` devolvio `stop_requested` y `checkpoint_recorded=true`;
- `runs/supervise` devolvio `stop_reason=stopped`;
- el proceso `codex --ask-for-approval never exec ... request-ref-autoprogramming-backlog-scanner-7e8a6ae9`
  siguio vivo hasta cortar la sesion del servidor.

Alcance:

- el supervisor debe consultar el registry de procesos antes de declarar
  terminalidad operativa;
- si la run esta `stop_requested`, escribir `orquesta_shutdown_request.json`,
  esperar `agent_shutdown_checkpoint_ack.json` cuando proceda y enviar señal al
  proceso controlado;
- confirmar que no quedan hijos del wrapper antes de reportar `stopped`;
- si el proceso no muere, emitir `stop_pending` o `stop_escalation_required`,
  no `stopped`;
- no matar por busqueda global de nombre: usar `process_ref + session_ref`.

Criterio de cierre:

- avance 2026-06-25: la envoltura residente de
  `/api/v0/server/shutdown` ya no acepta un `shutdown_ready=true` upstream si
  la respuesta conserva agentes en vuelo, checkpoints pendientes o runs pedidos
  sin confirmar; publica `shutdown_status=stop_pending` y mantiene congelado el
  supervisor hasta confirmacion real;
- avance 2026-06-25: `/api/v0/runs/supervise` ya proyecta
  `run_stop_requested`/`stop_requested` como `stop_pending`, no como `stopped`;
- avance 2026-06-25: la reconciliacion queued de run-control ya no trata
  `StoppedAgents` como confirmacion; exige `ConfirmedStoppedAgents` o
  terminalidad equivalente;
- avance 2026-06-25: antes de `CompleteRunControlV0(stopped)`, el stack
  consulta `ProcessRegistry + SnapshotV0` y bloquea el cierre si algun proceso
  registrado sigue `running`/`stopping` o no cuadra su identidad;
- cerrado local 2026-06-25: test con runtime fake que ignora la primera parada:
  `/runs/supervise` devuelve `stop_pending`, conserva `RunControl` en
  `stop_requested` y no confirma todos los agentes;
- cerrado local 2026-06-25: test con runtime fake que confirma parada:
  reconcile completa `RunControl` como `stopped`;
- cerrado local 2026-06-25: test con proceso registrado vivo impide
  terminalizar `RunControl` como `stopped`;
- smoke Codex real opt-in: una run parada no deja procesos `codex exec` vivos
  tras el cierre;
- status publico distingue `stop_requested`, `stop_propagated`,
  `stop_confirmed` y `stop_pending`.

## Rework documental SRV-TASK-024 2026-06-25

Origen:
`request-ref-autoprogramming-backlog-srv-task-024-90719eb4-reconcile-c83a27e7`,
agente
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-15e93ad10820843acec443a42cbac3c7`.

Resultado: entrega acotada a documentacion. El paquete trae contexto obligatorio
`ref_only`, write-set cerrado a docs del servidor/backlog y prueba focal
obligatoria. No se relanza otro padre ni se abre codigo porque no hay regresion
causal nueva con refs de job, receipt, artifact o smoke OPES acotado. La
evidencia local resuelve `required_ref_action=ack_evidence_required` mediante
ACK con `contexto_ref_only_resuelto`.

Criterio: SRV-TASK-024 sigue abierto hasta dispatch real o bloqueo causal
publico probado; esta correccion solo cierra su propia entrega tras
`go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.

## Rework documental SRV-TASK-024 2026-06-25 0f0bdd

Origen:
`request-ref-autoprogramming-backlog-srv-task-024-90719eb4-reconcile-c83a27e7`,
agente
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-0f0bdd6085651f057155f6a385ccea82`.

Resultado: correccion acotada a documentacion. El paquete trae contexto
obligatorio `ref_only`, `required_ref_action=ack_evidence_required`, write-set
cerrado a backlog y docs del servidor, y prueba focal obligatoria. No se
relanza otro padre ni se abre codigo porque no hay regresion causal nueva con
refs de job, receipt, artifact o smoke OPES acotado. La evidencia local
resuelve el contexto por paquete, AGENTS raiz, README/AGENTS locales, foto
vigente y docs locales.

Criterio: SRV-TASK-024 sigue abierto hasta dispatch real o bloqueo causal
publico probado; esta correccion solo cierra su propia entrega tras
`go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.

## Rework documental SRV-TASK-024 2026-06-25 d4aec

Origen:
`request-ref-autoprogramming-backlog-srv-task-024-90719eb4-reconcile-c83a27e7`,
agente
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-d4aec6d93452fd3b4f10a49c0bca437c`.

Resultado: correccion acotada a documentacion. El paquete trae contexto
obligatorio `ref_only`, `required_ref_action=ack_evidence_required`, write-set
cerrado a backlog y docs del servidor, y prueba focal obligatoria. No se
relanza otro padre ni se abre codigo porque no hay regresion causal nueva con
refs de job, receipt, artifact o smoke OPES acotado. La evidencia local
resuelve el contexto por paquete, AGENTS raiz, README/AGENTS locales, foto
vigente y docs locales.

Criterio: SRV-TASK-024 sigue abierto hasta dispatch real o bloqueo causal
publico probado; esta correccion solo cierra su propia entrega tras
`go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.

## Rework documental SRV-TASK-024 2026-06-25 403a707

Origen:
`request-ref-autoprogramming-backlog-srv-task-024-90719eb4-reconcile-c83a27e7`,
agente
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-403a707ea7e18c0c29793d3df9266ede`.

Resultado: correccion acotada a documentacion. El paquete trae contexto
obligatorio `ref_only`, `required_ref_action=ack_evidence_required`, write-set
cerrado a backlog y docs del servidor, y prueba focal obligatoria. No se
relanza otro padre ni se abre codigo porque no hay regresion causal nueva con
refs de job, receipt, artifact o smoke OPES acotado. La evidencia local
resuelve el contexto por paquete, AGENTS raiz, README/AGENTS locales, foto
vigente y docs locales.

Criterio: SRV-TASK-024 sigue abierto hasta dispatch real o bloqueo causal
publico probado; esta correccion solo cierra su propia entrega tras
`go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.
