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

Estado: parcial, con marcador goal-first cubierto en MCP y `/ops` el
2026-06-28.

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
Sigue vivo el cierre global de `/ops` y de reconciliacion completa de todos los
casos visibles.

## ORQ-OPES-002 reconciliacion_ack_artefactos_cierre_cola

Estado: vivo.

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

## ORQ-OPES-003 external_work_no_agent_no_delivery

Estado: vivo.

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

## ORQ-OPES-004 capability_externa_tts_edge_host_runner

Estado: vivo.

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
