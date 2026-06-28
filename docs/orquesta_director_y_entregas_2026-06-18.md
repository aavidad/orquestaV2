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

### 5b. Reasoning effort por defecto = medium (Goal-first y coste)
`cmd/orquesta-server/codex_runtime_config_v0.go`, `codex_director_wave_config_v0.go`.

Evidencia anterior: con `medium` algunos agentes Codex cerraban el turno sin
escribir ACK; por eso se subio temporalmente a `high`. Corte posterior: con
Goal-first, result files durables, receipts de dominio y promocion controlada de
artefacto-sin-ACK, el ACK ya no debe forzar el coste base de todas las olas. El
default vuelve a `medium`; `ORQUESTA_CODEX_REASONING_EFFORT` y
`ORQUESTA_CODEX_WAVE_REASONING_EFFORT` siguen permitiendo subir a `high` o
`xhigh` cuando haya riesgo tecnico justificado u orden explicita. La red de
seguridad real sigue siendo el cierre por evidencias, no el prompt ni el effort.

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

## P1 cerrado real: frontera de tareas autoprogramming

Descubierto en prueba real orquestando una app (Bolsa Diputación, 8 tareas hexagonales
con `depends_on`) por `/api/v0/autoprogramming/prepare-run` + `/api/v0/runs/supervise`
con Codex real (2026-06-18). El síntoma era que la tarea base se entregaba, pero las
dependientes no se relanzaban y el run quedaba `quiescent` con tareas abiertas.

**Corrección 2026-06-19:** el fallo principal no era `WaitAgentRefs` ni el
`WorkflowTaskCandidateProviderV0`; era la traducción de dependencias declaradas en el
spec de autoprogramming. Specs reales como Bolsa declaran `depends_on` usando refs fuente
(`bolsa-base`), mientras el scheduler satisface dependencias por `WorkflowTaskV0.TaskID`
materializado (`task-autoprogramming-...`). Ahora `orquesta-autoprogramming` remapea
esas refs fuente al `TaskID` generado, conserva refs opacas desconocidas y evita
autodependencias tras agrupar tareas.

El stack Codex también devuelve `WaitAgentRefs` como frontera viva del run: al preparar
un run nuevo incluye solo tareas sin dependencias; al repetir `prepare-run` sobre un run
existente omite tareas ya entregadas/cerradas e incluye las dependientes ya desbloqueadas
por `DeliveredTasks`/`ClosedTasks`. Esto mantiene el invariante cerrado: `WaitAgentRefs`
acota espera e ingesta; no se rellena con agentes futuros no materializados.

**Evidencia offline/focal:**
- `TestBuildAutoprogrammingProgrammableWorkV0HonraWriteSetYDependsOnPorTareaV0`.
- `TestCodexStackAutoprogrammingRunGlobalTickV0RelanzaFronteraDependienteTrasACKV0`.
- `TestCodexStackAutoprogrammingPrepareRunAPIV0TickGlobalTrasACKLanzaFronteraDependienteV0`.
- `TestAutoprogrammingResidentModeV0RelanzaFronteraDependienteTrasACKV0`.
- Suite afectada: `go test -count=1 ./modulos/orquesta-autoprogramming
  ./modulos/orquesta-app-codex-stack ./modulos/orquesta-app-director-service
  ./modulos/orquesta-orchestration-core`.

**Evidencia real 2026-06-19:**
- `bolsa-nucleo-001` sobre
  `~/Trabajo/Bolsa_Diputacion_app/orquesta_spec_nucleo.json` cerró las 8 tareas con
  Codex real: `status=cerrada`, `phase=cierre`, 8/8 tareas entregadas/cerradas,
  8 reviews aceptadas, 1 validación y 1 closure.
- `bolsa-tests-001` cerró 4 tareas adicionales de tests por supervisor residente, sin
  `runs/supervise` manual: `g01+g03` iniciales, `g02` tras ACK de `g01`, `g04` tras ACK
  de `g02+g03`, 4 reviews aceptadas, validación final y closure.
- Verificación externa de la app: `go test -count=1 ./...` en
  `~/Trabajo/Bolsa_Diputacion_app` devuelve paquetes `ok` reales para dominio, auth,
  i18n, repositorio, casos de uso y handler HTTP.

Si falla un caso futuro, diagnosticar runtime/proveedor/coordinador con evidencia nueva,
sin reabrir `queuedOperationalDirectorWaitAgentRefsV0` salvo regresión demostrada.

Riesgo lateral no cerrado en este P1: dependencias generadas por `live_works` usan refs
opacas de trabajos externos y necesitan su propio caso de promoción/replay; no afecta al
spec Bolsa actual.

## Hallazgos de la prueba real Bolsa (2026-06-18)

Lo que las pruebas demostraron que **SÍ funciona** (Codex real, app de verdad):
- Orquesta acepta un spec de app genérico por JSON (`prepare-run`) con `depends_on` y
  `write_set` por tarea, sin código a medida.
- El director **respeta las dependencias al lanzar**: con 8 tareas, lanzó solo la base
  (sin deps); las dependientes no se lanzaron antes de tiempo.
- Tras corregir P1, el supervisor residente reevalúa la frontera tras cada entrega,
  arranca solo dependientes desbloqueadas y cierra el run sin supervisión manual.
- El agente Codex **programó código de buena calidad**: dominio hexagonal de Bolsa
  (entidad Merit con máquina de estados Borrador→Presentado→Validado/Rechazado/Subsanación,
  `CanTransition`/`Transition` con validación y errores tipados, tabla de transiciones).
  Compila (`go build ./...` ok).
- Con `reasoning_effort=high`, el agente **escribió el ACK** (`completed`); con `medium`
  tendía a no escribirlo (la recuperación sin ACK sigue siendo la red de seguridad).
- Un run posterior orquestado por Orquesta añadió tests reales para dominio, auth/i18n,
  repositorio, usecase y HTTP handler; `go test -count=1 ./...` ya no es un pase vacío.

Bugs/carencias que la prueba **destapó** (arreglar en Orquesta, no en la app):
1. Frontera no re-evaluada tras entrega (ver sección BUG abajo) — el más grave y único
   bug de orquestación confirmado.

**Corrección (2026-06-19):** una sospecha inicial de "review sello de goma" (el agente
habría entregado sin tests) resultó **parcial** al revisar el ACK real: el agente base
SÍ ejecutó los `RequiredTests` declarados y dejó `test_receipts` con `status=passed,
exit_code=0`, pero el `go test` inicial no ejecutó pruebas porque aún no había ficheros
`*_test.go`. El gate
(`autoprogramming_review_gate_v0.go: autoprogrammingReviewGateRequiredTestIssuesV0`)
exige que cada required test declarado esté ejecutado y pasado, y emite
`required_test_missing`/`required_test_failed` (bloqueantes) si no.

Mitigación genérica añadida: `LocalCommandExecutorV0` marca `failed` cuando el comando es
`go test` y toda la salida son paquetes sin tests (`[no test files]`/`[no tests to run]`),
de modo que un required test vacío ya no puede cerrar una app Go. La carencia de
acceptance criteria semánticos queda como mejora de calidad separada.

## Frentes abiertos para el siguiente agente

- Cablear el guard de límite de replan en el scheduler (punto 4).
- Incidencia A2 / bucle del supervisor sobre runs ya completados
  (`docs/incidencia_opes_supervise_ack_reconciliation_a2_2026-06-13.md`): un
  `active_step` que no progresa sobre un run ya entregado se eleva a `tick_error`
  permanente. Pendiente; el arreglo correcto vive en la capa supervisor/composición,
  no en el contrato del plan-state.
- Revisar la condición de yield de progreso a replan (auditoría).
- Convertir la evidencia real Bolsa (`bolsa-nucleo-001` + `bolsa-tests-001`) en
  script/runbook opt-in repetible, sin tocar Bolsa productiva ni depender de rutas
  locales salvo por variables de entorno.
