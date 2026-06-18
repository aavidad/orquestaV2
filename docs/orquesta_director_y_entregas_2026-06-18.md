# Director, entregas y revisión — corte 2026-06-18

`doc_estado=vigente`. Foto operativa de una sesión de hardening centrada en que
**Orquesta no se cuelgue por nimiedades** y en cómo debe comportarse el Director
ante entregas imperfectas. Subordinado a `AGENTS.md`, `README.md` y
`docs/estado_actual_2026-05-17.md` (regla operativa sin filtros: no cortar trabajo
recuperable). Si algo aquí contradice esa foto, gana la foto.

## Principio rector (decisión del operador, 2026-06-18)

> **El disco es la fuente de verdad, no el ACK. Orquesta nunca se para si el
> agente ya entregó trabajo. El Director organiza y manda revisar; la revisión la
> hace otro agente por defecto, y solo se escala a humano cuando el Director no
> puede resolverlo.**

De aquí se derivan todos los cambios de este corte.

## Por qué: el problema real observado con Codex real

Probando olas de 3 agentes Codex **reales** (`scripts/smoke_codex_real_operational_wave.sh`)
se observó, de forma repetida y con tokens ilimitados:

- Los agentes **hacen el trabajo correcto** (los ficheros del write-set aparecen en
  disco, bien formados).
- Pero **NO escriben `agent_ack.json`** (el acuse). 0 de 3, una y otra vez.
- Endurecer el prompt a "ACK OBLIGATORIO, paso final ineludible" **no cambió nada**:
  el modelo da la tarea por terminada al producir el artefacto y cierra el turno
  antes del acuse. Es comportamiento estructural de agentes no deterministas, no un
  fallo de Orquesta.

Conclusión: **no se puede depender de que el agente escriba el ACK.** La causa raíz
de los "cuelgues por nimiedades" era que una entrega real sin acuse quedaba
huérfana y bloqueaba el run.

## Cambios implementados en este corte

Todos opt-in salvo donde se indique; por defecto no cambian el comportamiento
histórico. Suite completa (`go test ./...`) en verde.

### 1. Promoción de artefacto materializado sin ACK (núcleo)
`modulos/orquesta-runtime-codex-delivery/source_v0.go`
(`PromoteMaterializedArtifactWithoutAck`, opt-in en `ConfigV0`).

Si un agente no escribió ACK pero **materializó su write-set en disco**, Orquesta
**promueve la entrega** igualmente: deriva las refs del packet, pasa por la
verificación de worktree (evidencia/soft-gates del artefacto real) y adjunta
`gate-issue:artifact_without_ack_requires_review`. El trabajo no se pierde y el run
no se cuelga.

### 2. La revisión la hace un agente, no se acepta en silencio ni se bloquea hacia humano
`modulos/orquesta-autoprogramming/autoprogramming_review_gate_issue_*.go`.

`artifact_without_ack_requires_review` y `sensitive_detail_requires_review` son
**no advisory** (no se aceptan en silencio) pero **no bloquean cierre en duro hacia
humano**: derivan a `request_changes` → rework/review por **otro agente**. El
Director escala a humano (`ask_director`) solo si la resolución automática se agota.

### 3. Streaming por sub-ola (no esperar a la ola completa)
`modulos/orquesta-app-director-service/operational_director_wait_state_substream_v0.go`
(`StreamingSubwaveEnabled`, opt-in en `ContinueAppDirectorRequestV0`).

En cuanto un subconjunto de agentes entrega, su trabajo avanza a review sin esperar
al resto de la ola. Las tareas independientes fluyen por el pipeline
programar→revisar→documentar de forma incremental. Las dependencias se respetan
"gratis": el materializador de olas (`modulos/orquesta-director-operativo/wave_work_v0.go`)
hace orden topológico, así que dentro de una ola las tareas son independientes por
construcción; las dependientes van a olas posteriores. `WorkflowTaskV0.DependsOn`
ya bloquea el lanzamiento de tareas dependientes hasta que su prerequisito entrega
(verificado en `modulos/orquesta-orchestration-core/workflow_task_candidate_provider.go`).

### 4. Guard de límite de replan (anti bucle infinito) — pieza pura
`modulos/orquesta-core-replanner/replan_retry_limit_v0.go`
(`CapReplanActionByRetryLimitV0`, `DefaultReplanRetryLimitV0 = 3`).

Cuando una tarea ya se replanificó N veces, las acciones resolutivas
(retry/split/replace_agent/escalate_capacity) se convierten en `ask_director`
(escala a decisión/humano). **Pendiente de cablear** en el scheduler: hoy el
snapshot no expone el conteo de replans por tarea; cablearlo cruza el boundary de
tipos workflow↔replanner y debe hacerse con cuidado. Se dejó la pieza lista y
probada para integrar cuando una prueba real muestre el bucle.

### 5. Prompts: ACK obligatorio + caveman conviven
`modulos/orquesta-runtime-{codex,gemini,claude}/*_prompt_v0.go`.

El prompt pide el ACK como paso final obligatorio Y mantiene `caveman`/modo
compacto (ahorro de tokens, importante con cientos de agentes). Dado que el ACK no
se manda de forma fiable haga lo que haga el prompt, se prioriza el ahorro: la red
de seguridad real es la promoción del punto 1, no el prompt.

### 5b. Reasoning effort por defecto = high (fiabilidad de ACK)
`cmd/orquesta-server/codex_runtime_config_v0.go`, `codex_director_wave_config_v0.go`.

Evidencia: con `medium` los agentes Codex tendían a cerrar el turno sin escribir el ACK
(0 de 3 en varias olas); el primer agente Bolsa con `high` sí lo escribió. El default del
runtime Codex sube a `high`. Sigue siendo **configurable** por
`ORQUESTA_CODEX_REASONING_EFFORT` (override a `medium` si el coste de tokens aprieta, p.
ej. cientos de agentes). Importante: `high` **sube la tasa de ACK, no la garantiza al
100%** (1 caso observado); la recuperación de artefacto-sin-ACK sigue siendo la red de
seguridad. Es una cintura coste/fiabilidad, no una bala de plata.

### 6. Spec de app con write-set y dependencias por tarea (sin código por app)
`modulos/orquesta-autoprogramming/autoprogramming_task_group_v0.go`,
`autoprogramming_partition_policy_v0.go`.

`AutoprogrammingTaskGroupCandidateV0` admite ahora `write_set` y `depends_on` por
tarea. Si una tarea los declara, ganan sobre la inferencia automática por área. Esto
permite enviar por JSON a `/api/v0/autoprogramming/prepare-run` un spec de app con
dependencias finas (p. ej. hexagonal: dominio → puertos → casos de uso → adaptadores
→ tests) **sin escribir un harness Go por aplicación**. Aditivo y retrocompatible:
tareas sin declaración mantienen la inferencia por área. Un agente futuro que quiera
orquestar una app concreta construye el spec JSON, no código.

## Auditoría del Director (diagnóstico, no todo resuelto)

Auditoría crítica del cerebro de decisión (`modulos/orquesta-director-scheduler`,
`modulos/orquesta-core-replanner`). Hallazgos que un futuro agente debe conocer:

- **Sin límite de reintentos de replan** (riesgo de bucle retry/split infinito).
  Mitigación parcial: guard del punto 4, pendiente de cablear.
- **`schedulerProgressCandidateCanYieldToReplanV0`**
  (`scheduler_tick_progress_candidate_v0.go`) tiene una condición sospechosa
  (`started && !failed && stopped`) que parece contradictoria; revisar si la
  supervisión de progreso realmente cede a replan cuando debe.
- **Review como "sello de goma"**: el Director comprueba que existe un
  ReviewResult `accepted`, no evalúa calidad por sí mismo. La calidad depende del
  agente revisor / gate-issues.
- **Capacidad sin límite duro explícito**: el paralelismo lo marca el plan; no hay
  circuit breaker global. Con cientos de agentes, vigilar.
- **Estado en strings con separadores hardcodeados**
  (`#review_result:` etc. en `scheduler_tick_review_gate_candidate_v0.go`): frágil
  a cambios de formato.

Estos son riesgos abiertos, no regresiones nuevas. Priorizar con evidencia de
pruebas reales antes de refactors grandes.

## Cómo probar

- Offline determinista del streaming:
  `go test -count=1 -run 'TestStreamingSubwavePipeline' ./modulos/orquesta-app-director-service`
- Promoción artefacto sin ACK:
  `go test -count=1 -run 'PromueveArtefactoSinAck|SinPromocionDescarta|PromocionNoInventa' ./modulos/orquesta-runtime-codex-delivery`
- Gate de revisión por agente:
  `go test -count=1 -run 'ExigeRevision' ./modulos/orquesta-autoprogramming`
- Límite de replan:
  `go test -count=1 ./modulos/orquesta-core-replanner`
- Ola Codex real (opt-in, gasta cuota): `scripts/smoke_codex_real_operational_wave.sh`
  con `ORQUESTA_CODEX_REAL_OPERATIONAL_WAVE_CONFIRM=1`,
  `ORQUESTA_CODEX_REAL_OPERATIONAL_WAVE_EXECUTE_CODEX=1`,
  `ORQUESTA_CODEX_REAL_OPERATIONAL_WAVE_CODEX_EXECUTION_CONFIRMED=1`,
  `ORQUESTA_CODEX_HOME=$HOME`, `ORQUESTA_CODEX_CODE_HOME=$HOME/.codex`. El
  smoke pasa aunque los 3 agentes no escriban ACK: la promoción recupera el trabajo.

## BUG abierto: frontera de tareas no re-evaluada tras entrega (autoprogramming)

Descubierto en prueba real orquestando una app (Bolsa Diputación, 8 tareas hexagonales
con `depends_on`) por `/api/v0/autoprogramming/prepare-run` + `/api/v0/runs/supervise`
con Codex real (2026-06-18).

**Síntoma:** la tarea base (sin dependencias) se lanza, el agente Codex la programa y
entrega (ACK `completed`, queda en `run.DeliveredTasks`). Pero el director se queda
`quiescent` (`stop_reason=done`) **sin lanzar las tareas cuyas dependencias ya están
satisfechas**. Run parado en `agents_requested=1, tasks_open=7` indefinidamente, pese a
forzar supervise con más capacidad/bursts. Evidencia del drain:
`projection-tasks-8`, `projection-open-tasks-7`, `projection-requested-agents-1`,
`quiescent`.

**Causa raíz:** el flujo de autoprogramming/stack-Codex genera el trabajo por
"decision sources" (`modulos/orquesta-app-codex-stack/project_backlog_decision_source_v0.go`,
`dispatchers_v0.go`), NO por el `WorkflowTaskCandidateProviderV0`
(`modulos/orquesta-orchestration-core/workflow_task_candidate_provider.go`) — que es
justamente quien re-evalúa la frontera de tareas schedulables respetando `DependsOn`
(`workflowTaskSchedulableV0`). Ese provider **no está cableado en el stack Codex**
(cero referencias). Por eso la frontera no se recalcula tras una entrega: el scheduler
(`scheduler_tick_v0.go:69-73`) declara quiescent cuando `WorkCandidates` llega vacío,
y nadie regenera esos candidatos para las tareas recién desbloqueadas.

**Importante (instrucción del operador 2026-06-18):** los bugs que destapa una app de
prueba se arreglan en **Orquesta en general**, no parcheando para la app concreta.

**Diagnóstico afinado (2026-06-18, sesión "arregla todo"):**
- `ContinueAppDirectorV0` con run + `DirectorTaskStore` + dispatchers **SÍ relanza** la
  frontera tras una entrega, incluso con `WaitAgentRefs` acotado. Probado por
  `TestContinueAppDirectorV0RelanzaFronteraTrasEntregaV0` y
  `...AunConWaitScopeAcotadoV0` en `orquesta-app-director-service`
  (`frontier_redispatch_v0_test.go`). Es decir: **el director y el provider de frontera
  funcionan**. El bug NO está ahí.
- En el run real (autoprogramming + supervise), tras entregar la base **no se genera
  ningún outbox nuevo** para las dependientes (verificado en el ledger: solo 4 mensajes,
  los de la ola inicial). El director nunca emite `RequestCapacity`/`RequestAgent` para
  la frontera desbloqueada.
- Sospecha principal: el camino de `supervise` → coordinador → `drainRunAttemptControlV0`
  → `continueDrainRunControlAfterExternalV0` no está llegando a ejecutar el ciclo del
  director con el provider de frontera para este run, o lo ejecuta con un contexto que
  no re-pide (posible interacción con `AutoprogrammingDirectorDecisionSourceV0`, que solo
  abre review cuando TODAS entregaron y no emite lanzamientos; o con el managed loop /
  `ExternalWaiter` que acota el burst al scope ya consumido).
- **Descartado** (no era la causa): el wait-scope de
  `queuedOperationalDirectorWaitAgentRefsV0` excluye agentes no pedidos a propósito
  (invariante protegido por `TestCodexStackV0WaitRefsColaRepara...`); incluir ahí las
  schedulables rompe ese invariante y NO arregla el relanzamiento.

**Cerco del bug (2026-06-19):** `frontier_redispatch_v0_test.go` prueba que
`ContinueAppDirectorV0` relanza la frontera tras entrega en CUATRO configuraciones:
(1) directo, (2) con `WaitAgentRefs` acotado, (3) con managed loop + `ExternalWaiter`,
(4) con `DirectorDecisionSource` que no progresa. **Las cuatro pasan.** Por tanto el
director y el provider de frontera (`WorkflowTaskCandidateProviderV0`, cableado en
`provider_composition_v0.go` cuando hay `DirectorTaskStore`) están sanos. Los ports del
drain del stack preservan `DirectorTaskStore` y dispatchers
(`directorPortsWithClosureSourceV0`). El fallo, por descarte, está en el
`RunCoordinator`/cola del stack ANTES de llegar a `ContinueAppDirectorV0`, en
condiciones no reproducidas unitariamente. Pista de ejecución real: en la cola del run
Bolsa coexistían DOS runs (`bolsa-nucleo-001` y un `request-ref-autoprogramming-backlog-scanner-*`),
y el ledger solo tenía 4 mensajes (la ola inicial), sin outbox nuevo para las
dependientes — el coordinador nunca volvió a pedirlas.

**Pendiente:** depurar con trazas el `RunCoordinator`/drain en ejecución (no solo
lectura) para ver por qué este run no vuelve a `ContinueAppDirectorV0` con la frontera.
Reproducción end-to-end: spec `~/Trabajo/Bolsa_Diputacion_app/orquesta_spec_nucleo.json`
por `/api/v0/autoprogramming/prepare-run` + `/api/v0/runs/supervise` con Codex real.
NO arreglar tocando el wait-scope de `queuedOperationalDirectorWaitAgentRefsV0`: rompe el
invariante "wait refs no esperan agentes no materializados"
(`TestCodexStackV0WaitRefsColaRepara...`) y no resuelve el relanzamiento (ya probado).

## Hallazgos de la prueba real Bolsa (2026-06-18)

Lo que la prueba demostró que **SÍ funciona** (Codex real, app de verdad):
- Orquesta acepta un spec de app genérico por JSON (`prepare-run`) con `depends_on` y
  `write_set` por tarea, sin código a medida.
- El director **respeta las dependencias al lanzar**: con 8 tareas, lanzó solo la base
  (sin deps); las dependientes no se lanzaron antes de tiempo.
- El agente Codex **programó código de buena calidad**: dominio hexagonal de Bolsa
  (entidad Merit con máquina de estados Borrador→Presentado→Validado/Rechazado/Subsanación,
  `CanTransition`/`Transition` con validación y errores tipados, tabla de transiciones).
  Compila (`go build ./...` ok).
- Con `reasoning_effort=high`, el agente **escribió el ACK** (`completed`); con `medium`
  tendía a no escribirlo (la recuperación sin ACK sigue siendo la red de seguridad).

Bugs/carencias que la prueba **destapó** (arreglar en Orquesta, no en la app):
1. Frontera no re-evaluada tras entrega (ver sección BUG abajo) — el más grave y único
   bug de orquestación confirmado.

**Corrección (2026-06-19):** una sospecha inicial de "review sello de goma" (el agente
habría entregado sin tests) resultó **falsa** al revisar el ACK real: el agente base
SÍ ejecutó los `RequiredTests` (`go test ./...` y `go test ./internal/candidate/domain/...`)
y dejó `test_receipts` con `status=passed, exit_code=0`. El gate
(`autoprogramming_review_gate_v0.go: autoprogrammingReviewGateRequiredTestIssuesV0`)
exige que cada required test declarado esté ejecutado y pasado, y emite
`required_test_missing`/`required_test_failed` (bloqueantes) si no. Funciona. Lo que el
agente NO hizo fue escribir ficheros `*_test.go` propios del dominio — eso es calidad del
trabajo (mejorable por prompt/review de otro agente), no un fallo del gate de Orquesta.

## Frentes abiertos para el siguiente agente

- Cablear el guard de límite de replan en el scheduler (punto 4).
- Incidencia A2 / bucle del supervisor sobre runs ya completados
  (`docs/incidencia_opes_supervise_ack_reconciliation_a2_2026-06-13.md`): un
  `active_step` que no progresa sobre un run ya entregado se eleva a `tick_error`
  permanente. Pendiente; el arreglo correcto vive en la capa supervisor/composición,
  no en el contrato del plan-state.
- Revisar la condición de yield de progreso a replan (auditoría).
