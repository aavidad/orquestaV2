# Backlog operativo OPES/Orquesta - 2026-06-28

Este shard consolida incidencias OPES/Orquesta observadas en los ficheros
`TAREA_OPES_*` sin convertir esos ficheros en cola directa. Los `TAREA_OPES_*`
siguen siendo entradas de diagnostico e historia; este documento es la lista
canonica compacta para programar reparaciones en Orquesta.

Reglas de uso:

- No tocar OPES productivo para validar estas entradas.
- Validar con instancia temporal, fakes o smokes opt-in acotados.
- Conservar trabajo recuperable: ACKs, borradores, artefactos parciales y notas
  de revision no se descartan por formato reparable.
- Si una entrada afecta a OPES, Orquesta debe dirigir el trabajo; Codex directo
  solo puede observar, auditar o reparar un tapon local documentado.

## ORQ-OPES-001 estado_operativo_global_accionable

Estado: casi cerrado para la superficie local MCP/web; pendiente contrato
global exhaustivo por cada run visible y smoke temporal real.

Problema: las vistas de estado mezclan procesos vivos, runs stale, colas y
contenedores goal-first sin una decision unica accionable. El operador necesita
ver si debe observar goal, esperar agente, reparar state, revisar entrega o
liberar bloqueo.

Alcance inicial:

- `autoprogramming/status`;
- `/ops`;
- stats MCP relacionadas con runs goal-first y legacy.

Criterio de cierre: para cada run visible debe existir una accion segura
explicita o una razon compacta de no accion. Los runs goal-first sin
`GoalWorkStateV0` pero con `GoalWorkRunMarkerV0` deben mostrarse como state
faltante reparable, no como supervision legacy.

Avance 2026-06-28: `autoprogramming/status` lee `GoalWorkRunMarkerV0` por
puerto opt-in, publica `autoprogramming_goal_first_state_missing`, cuenta el
run como bloqueado reparable, emite accion `goal_first_state_missing` con
`repair_goal_state_before_legacy_supervision` y suprime acciones legacy para
esa run. `ops_snapshot.decision` publica `repair_goal_state` en vez de derivar
`wait_deliveries` desde stats legacy residuales. Evidencia:
`TestMCPAutoprogrammingStatusExecutorV0GoalFirstMarkerSinStateNoSupervisaLegacy`.

Avance 2026-06-28 adicional: `director.stats` tambien lee el marker goal-first
por puerto opt-in cuando falta `GoalWorkStateV0`, proyecta
`stats.status=goal_first_state_missing`, `closure.blocked_by=
goal_first_state_missing` y `ops_snapshot.decision.action=repair_goal_state`.
`/queue/global-status` normaliza esa situacion como accion
`repair_goal_state`, y `/ops` evita los botones de supervision legacy para esa
run; muestra reparar `GoalWorkStateV0` antes de observar o supervisar.
Evidencia:
`TestMCPDirectorStatsToolExecutorV0GoalFirstMarkerSinStatePublicaRepairGoalState`,
`TestMCPQueueGlobalStatusHTTPHandlerV0GoalFirstStateMissingRecomiendaRepararState`
y `TestOpsDashboardWebEndpointV0GoalFirstUsaObserveGoalEnAvance`.
Sigue vivo el cierre de contrato unico exhaustivo "accion segura o razon de no
accion" para todos los tipos de run visible.

## ORQ-OPES-002 reconciliacion_ack_artefactos_cierre_cola

Estado: parcial; cierre goal-first DomainWork ya no acepta receipts
incompletos y la secuencia fake/offline de derivados OPES ya cierra por ledger.

Problema: en trabajos OPES hay ACKs y entregas parciales utiles, pero el cierre
de cola no siempre reconcilia artefactos canonicos, pendientes reales,
derivados, visuales, tests, audio, HTML, RAG y QA.

Alcance inicial:

- reconciliacion de ACK/deliveries;
- cierre de external/domain work;
- criterios OPES de derivados y consolidacion canonica.

Criterio de cierre: un smoke OPES temporal puede demostrar que los artefactos
entregados se inventarian, se asignan a estado canonico o rework y la cola no
queda en falso `ready`, `running_stale` o `complete` sin evidencias.

Avance 2026-06-28: el cierre goal-first por DomainWork exige que el receipt
aceptado del ledger cubra contratos requeridos y que el record declare
`complete_job=true`. Un receipt aceptado con `complete_job=false` bloquea con
`domain_work_receipt_artifact_incomplete` y rework, evitando cierre falso de
artefactos parciales. Evidencia:
`TestCodexStackV0ExternalWorkGoalFirstNoCierraReceiptDomainWorkIncompleteV0`.
La recuperacion DomainWork sin `agent_ack.json` deja de depender de frases en
`last_message`/`stderr`: ahora requiere contrato causal
`ApplyExternalDomainWorkV0`, external work y artefacto materializado dentro del
write-set/project workdir. Evidencia:
`TestRecoverDomainWorkAckV0RecuperaArtefactoValidoSinSenalDeLog` y
`TestCodexStackV0OPESExternalWorkRecuperaEntregaSinACKConArtefactoValido`.
El 2026-06-28 tambien queda cubierta la cadena fake/offline de derivados OPES
posterior a `plan_temario`: `update_topic_registry` ->
`finalize_temario_package`, 23 work kinds en total, usando el bridge OPES como
consumidor, sin OPES productivo. Cada run `goal_first` sube artefacto por
`DomainWork`, registra receipt aceptado en ledger y cierra por
`evidence-ref-goal-domain-receipt-ledger-accepted`, sin receipts inventados.
Evidencia:
`TestCodexStackV0ExternalWorkGoalFirstCierraSecuenciaOPESDerivadosConReceiptsLedgerV0`.
Sigue vivo el smoke OPES temporal real de derivados/cierre, inventario de
artefactos canonicos y cola final sin pendientes reales.

Avance 2026-06-28 adicional: `external_work.input_fields` goal-first conserva
refs durables `input_field_payload`/`app_change_payload:<run>:<change>:...`
resolubles desde `AppChangeStore` por `RunRef`/`ChangeRef`, sin exponer rutas
locales absolutas al Goal. Evidencia:
`TestBuildExternalWorkGoalWorkSpecV0InlineaInputFieldsOperativosSeguros` y
`TestCodexStackV0ExternalWorkRunGoalFirstConservaInputFieldsOPESV0`.

## ORQ-OPES-003 external_work_no_agent_no_delivery

Estado: parcial; legacy 1+6 y guard de cierre goal-first cubiertos, pendiente
smoke OPES temporal real.

Problema: algunas tareas de external work aparecen preparadas o listas pero no
materializan agente real ni entrega observable. El sistema debe distinguir
pendiente por capacidad, bloqueo de proveedor/runtime, contrato no
materializable o bug de dispatch.

Alcance inicial:

- `orquesta.external_work.run.v0`;
- bridge/drain de external work;
- materializacion de subagentes y entrega durable.

Criterio de cierre: si el contrato requiere agente, existe ACK, entrega,
bloqueo operativo explicito o rework causal. No basta con una tabla escrita por
un padre cuando el contrato exige subroles reales.

Avance 2026-06-28: la ruta goal-first de external-work OPES que declara
`opes.padre-tema-6-subroles.v1` ya no cierra con un unico receipt aceptado,
aunque `complete_job=true`. El validador exige seis evidencias compactas de
subrol `domain-work-opes-subrole-*` en el resultado o en el receipt aceptado, o
bloquea con `domain_work_opes_subroles_evidence_missing`. Evidencia:
`TestCodexStackV0ExternalWorkGoalFirstNoCierraOPESSubrolesSinSeisEvidenciasV0`
y `TestCodexStackV0ExternalWorkGoalFirstCierraOPESSubrolesConSeisEvidenciasV0`.
Sigue pendiente prueba real temporal con agentes/subagentes y cola OPES acotada.

## ORQ-OPES-004 capability_externa_tts_edge_host_runner

Estado: parcial; contrato neutral y bridge OPES cubiertos, pendiente smoke OPES
temporal real de derivados/cierre.

Problema: la generacion de audio/TTS para OPES necesita frontera de capacidad
externa clara. Orquesta no debe asumir runner local, modelo, credenciales ni
host de edge; debe declararlo por puerto/adaptador opt-in.

Alcance inicial:

- jobs OPES de audio;
- capability discovery;
- runners externos para TTS.

Criterio de cierre: una composicion temporal puede declarar capacidad TTS,
fallar con razon operativa si no existe, y registrar receipts/evidencias cuando
produce audio sin acoplar el nucleo a proveedor concreto.

Avance 2026-06-28:

- `audio_asset` deriva requisito neutral `speech_synthesis`;
- `DomainWorkExternalCapabilitySourcePortV0` permite que la composicion declare
  capacidades disponibles o no disponibles;
- `EvaluateDomainWorkExternalCapabilitiesV0` bloquea con
  `domain_work_external_capability_missing` y `operational_reason` cuando falta
  TTS;
- `opes-drain-once` evalua la capacidad declarada por env antes de dry-run,
  claim o submit; `generate_audio_asset` sin `speech_synthesis` queda en
  `external_capability_missing` sin postear a Orquesta;
- el resumen publico de `opes-drain-once` expone `operational_reason` tanto en
  `results[]` como en `errors[]`, para que el operador vea
  `external_capability_missing:speech_synthesis` sin revisar logs;
- el wrapper de derivados reales bloquea `drain-once`/finalizacion con audio si
  no se declara `ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY=available`;
- el smoke fake de derivados declara `speech_synthesis` y sigue cubriendo la
  cadena hasta `generate_audio_asset`/finalizacion. Evidencia:
  `TestRunOPESDrainOnceV0AudioSinSpeechSynthesisNoPosteaOrquestaV0`,
  `TestRunOPESDrainOnceV0AudioConSpeechSynthesisPosteaOrquestaV0`,
  `TestSmokeOPESDerivativesRESTWrapperPreflightRealBloqueaSinSpeechSynthesisV0`
  y
  `TestSmokeOPESDerivativesRESTWrapperFakeServerRunUntilFinalizeV0`;
- sigue pendiente smoke OPES temporal real que use esa capacidad y registre
  receipt/evidencia de audio real o fake controlado.

## ORQ-OPES-005 robustez_api_supervision_stream_fd_timeout

Estado: parcial; respuesta finita de supervise cubierta para
`autoprogramming/supervise` y `/runs/supervise`.

Problema: `/autoprogramming/supervise` y rutas de supervision pueden despachar
trabajo pero dejar al cliente HTTP esperando indefinidamente. La API debe
separar despacho, observacion y streaming, cerrar respuestas con timeout
controlado y exponer la accion siguiente.

Alcance inicial:

- endpoints de supervision;
- streaming/flush HTTP;
- timeouts y cancelacion cooperativa;
- diagnostico de fd/proceso vivo.

Criterio de cierre: un test o smoke local reproduce despacho con cliente HTTP y
verifica respuesta finita, cuerpo operativo en castellano y continuidad del
trabajo por observacion posterior.

Avance 2026-06-28: `/api/v0/runs/supervise` tiene smoke con cliente HTTP real
contra `httptest.Server` en MCP y servidor ensamblado: si el executor queda
vivo, responde `202 accepted_background` con `operation_ref`, diagnostico y JSON
decodificable sin colgar. Evidencia:
`TestMCPRunSupervisorHTTPHandlerV0ClienteRealRecibeCuerpoSinColgar` y
`TestServerRunSuperviseHTTPClienteRealRecibeCuerpoSinColgarV0`.
`/api/v0/autoprogramming/supervise` ya tenia cobertura equivalente en servidor
con `TestServerAutoprogrammingSuperviseHTTPClienteRealRecibeCuerpoSinColgarV0`.
Sigue vivo el cierre de streaming/flush amplio y observacion posterior sobre
colas OPES reales temporales.
