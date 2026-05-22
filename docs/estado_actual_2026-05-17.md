# Estado actual de Orquesta - 2026-05-17

Este documento es la foto vigente para orientar trabajo nuevo. Los documentos
anteriores siguen siendo contexto historico o tecnico, pero no todos describen
la frontera actual del proyecto.

## Estado vigente

Orquesta ya no debe leerse como una app cerrada de programacion. La direccion
vigente es convertirla en un nucleo reutilizable de orquestacion de agentes para
apps externas.

La regla central es: Orquesta aporta juicio mediante director y agentes; las
apps de dominio aportan datos, reglas, validadores, persistencia, UI/API y
ensamblado final.
OPES no es el producto base del nucleo: es una composicion consumidora. Cualquier
dominio externo, incluido OPES, entra por puertos, conectores y refs opacas, sin
compartir DB/filesystem interno ni mover reglas de producto al core.

Esto implica:

- el core gobierna runs, fases, tareas, artefactos, revision, evidencias,
  capacidad, handoff, shutdown logico y supervision;
- la seleccion de modelo, runtime, cuotas y proveedor pertenece a composiciones
  y adaptadores, no al core puro;
- REST, MCP, web y CLI son adaptadores sobre puertos, no el dominio;
- la superficie preferente para que una IA maneje Orquesta es MCP:
  `orquesta.domain_work.v0`, tools de director/supervisor y resources compactos;
  el bridge HTTP local existe como adaptador fino, y un servidor MCP/MCPO real
  productivo sigue siendo adaptador opt-in, no logica de core;
- programacion, OPES y otros dominios son consumidores/composiciones;
- los conectores traducen contratos de dominio a trabajo orquestable y devuelven
  artefactos al dominio propietario;
- los conectores externos son puertos/adaptadores de composicion; si una fuente
  real de dominio o cierre no existe, se marca como pendiente verificable;
- los perfiles de trabajo (`code_study`, `implementation`, `refactor`,
  `required_tests`, `documentation`, `review`, `domain_work`) son neutrales y se
  transportan como `WorkProfileV0`/`WorkflowTaskV0.work_profile_kind`;
- ninguna app externa debe copiar internals, compartir DB/filesystem interno ni
  decidir plan, runtime, modelo o paralelismo sin director de Orquesta.

## Nucleo neutral

El nucleo neutral vigente es el plano comun de orquestacion:

- contratos genericos de trabajo externo y artefactos;
- director, agentes, scheduler, fases, tareas y evidencias;
- seleccion de capacidad y contratos para que composiciones externas conecten
  runtime, proveedor, modelo y cuotas sin contaminar el core;
- control de progreso, cuota, bloqueos, loops, handoff y cierre;
- registro de entregas y estadisticas compactas.

La composicion Codex/programacion no define el nucleo. Es una app consumidora que
usa Orquesta para crear, modificar, auditar, migrar o desplegar software. Debe
entrar por contratos como `AppSpecV0` o trabajo de dominio equivalente.
El stack Codex puede ejecutar perfiles neutrales, pero no define su taxonomia.

La composicion OPES tampoco define el nucleo. OPES conserva oposiciones,
programas, temas, taxonomia, fuentes, reglas pedagogicas, validadores,
persistencia y ensamblado. Cuando necesita planificacion documental, pide trabajo
a Orquesta (`plan_temario`, `plan_tema`, `plan_documento`, etc.) y valida despues
los artefactos recibidos.
OPES aporta el dominio; el nucleo conserva el perfil neutral y la orquestacion.

## Evidencia y pruebas documentadas

La evidencia operativa mas reciente revisada es la prueba limpia OPES-Orquesta
`plan_tema` documentada el 2026-05-15:

- OPES envio un job externo neutral `plan_tema`.
- Orquesta arranco un agente Codex real (`gpt-5.5`, `xhigh`).
- Orquesta recupero/canonicalizo un `document_plan`.
- OPES recibio exactamente un artefacto `document_plan`.
- El job paso a `completed`.
- OPES creo 21 jobs derivados: 10 `draft_content_block`, 5
  `generate_visual_asset`, 1 `validate_topic`, 1 `review_legal`, 1
  `review_pedagogical`, 2 `review_quality` y 1 `assemble_topic`.

Pruebas documentadas como pasadas en ese corte:

- `go test -count=1 ./modulos/orquesta-domain-work`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCanonicalDomainWorkDeliveryPayloadBodyV0NormalizaPlanOPESReal|TestDefaultDomainWorkArtifactSubmissionBuilderV0CanonicalizaDocumentPlanAliasesOPES|TestDefaultDomainWorkArtifactSubmissionBuilderV0IdempotenciaEstablePorJobYArtefacto'`
- `go test -count=1 ./modulos/orquesta-opes-bridge`
- `go test -count=1 ./modulos/orquesta-runtime-codex -run 'TestCodexExecResolverV0PromptUsaControlFilesDelRuntime'`
- `go test -count=1 ./...`

Nota: esas pruebas son evidencia del corte documentado, no una reejecucion de
este cambio de documentacion. El arbol de trabajo actual contiene muchos cambios
concurrentes.

El corte del 2026-05-18 anade cobertura focal para `plan_temario` de OPES como
`document_plan`: mapper, policy editorial OPES, `xhigh`, drain fake por REST,
flujo local con Codex fake y envio por `DomainWork`. Ese mismo dia se cerro un
smoke real acotado de Operario por REST contra OPES temporal: OPES importo el
programa real, creo `job_ref=9784a562f074769a08043707fdd79eb2`, Orquesta lo
dreno por bridge residente, lanzo Codex real `gpt-5.5` con `xhigh`, envio
`document_plan` a OPES, OPES completo el job y creo 20 derivados pendientes.
No se ejecutaron los derivados en ese smoke. Despues de ese corte se anadio
automatizacion de pases por `ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE`: el bridge
residente consulta tipos en orden y solo drena la primera fase que OPES siga
mostrando como `pending`, usando el ledger para no relanzar jobs ya enviados.

## Estado real tras el primer corte del 2026-05-17

El primer corte ya no esta solo en documentos:

- `modulos/orquesta-director-operativo` contiene el contrato puro del Director
  Operativo V1. Modela `OperationalDirectorRequestV0`,
  `OperationalDirectorPlanV0`, `OperationalDirectorWaveWorkV0`, modos
  `programming` y `domain_work`, contexto insuficiente, presupuestos,
  write-set, tests requeridos y delegacion recursiva gobernada.
- `modulos/orquesta-director-operativo/README.md`,
  `modulos/orquesta-director-operativo/docs/contratos.md` y
  `modulos/orquesta-director-operativo/docs/pruebas.md` son la referencia local
  para ese contrato.
- `modulos/orquesta-orchestration-core/operational_director_materializer_v0.go`
  es el materializador actual: toma un plan `ready`, construye olas con
  `BuildOperationalDirectorWaveWorkV0`, selecciona items `launch_subagents`,
  guarda `WorkflowTaskV0` y emite comandos `CreateMicrotask` por el workflow.
- `modulos/orquesta-orchestration-core/operational_director_materializer_v0_test.go`
  demuestra dos guardas basicas: crea microtarea accionable para un plan listo y
  no lanza un plan bloqueado por contexto insuficiente.
- `WaitAgentRefs` existe en los loops progresivos y en
  `modulos/orquesta-app-codex-stack` para esperar agentes externos concretos.
  Ya puede elevarse a contrato de cohorte/ola mediante
  `WorkflowTaskWaitStateV0`, que guarda causa, refs de tasks, agentes objetivo y
  pendientes.
- P1 WaitAgentRefs queda cerrado para el stack Codex: el mismo scope limita
  pending, wait e ingesta de ACK/deliveries. `DrainRunRequestV0.WaitAgentRefs`
  llega al request de observaciones, el `DeliverySource` queda envuelto/filtrado
  y `DrainRunV0` vuelve a filtrar antes de aplicar observaciones. El source
  Codex acota descriptors antes de leer ACKs, y `domain_work` filtra submit y
  recovery antes de producir efectos. La evidencia focal incluye
  `TestRunProgressiveLoopV0PropagaWaitAgentRefsAObservationSource`,
  `TestCodexDeliveryObservationSourceV0WaitAgentRefs*`,
  `TestDrainRunV0WaitAgentRefsNoIngiereACKFueraDeScope` y
  `TestDomainWork*WaitAgentRefs`; el modo legacy con `WaitAgentRefs` vacio sigue
  observando el run completo.
- `docs/corte_director_funcionando_tarde_2026-05-17.md` deja el handoff de tarde:
  que ruta puede usarse hoy, que comandos ejecutar, que P0 queda cerrado y que
  queda pendiente despues del cierre de scope.
- `docs/corte_cierre_generico_director_operativo_2026-05-17.md` deja el handoff
  siguiente: P1 `WaitAgentRefs` queda cerrado y el foco pasa a cierre causal
  offline generico del Director Operativo. Lo que no tenga implementacion y test
  claro se trata como pendiente verificable.
- El cierre operativo generico ya tiene un tramo offline en codigo:
  `OperationalDirectorClosureV0` valida entrega registrada, review aceptada y
  evidencias de tests requeridos antes de cerrar task, abrir/registrar
  validacion final y cerrar run. Tambien valida por historial de eventos que la
  entrega, review request, resultado aceptado y accepted review pertenezcan a la
  misma cadena causal. `ContinueAppDirectorV0` lo invoca despues del loop cuando
  queda quiescent y la composicion inyecta `OperationalClosureSource`.
- `RequiredTestEvidenceV0` ya existe en `orquesta-orchestration-core` como
  comprobante durable minimo de tests requeridos. Guarda `run_ref`, `task_ref`,
  comando exacto, status `passed`/`failed` y refs causales de entrega, review
  request, review result y accepted review. `OperationalDirectorClosureV0`
  acepta solo evidencias `passed` que coinciden con cada
  `WorkflowTaskV0.RequiredTests` y con la cadena causal del cierre; un summary
  textual no cuenta como test ejecutado. La evidencia debe incluir artefactos en
  `evidence_refs`, su ref debe estar pedida por el cierre y el cierre exige
  `EventSink` para no mutar runs sin eventos persistidos.
- `orquesta-state-file` ya persiste `RequiredTestEvidenceV0`; el stack Codex lo
  usa como fuente de refs para cierre desde `RequiredTestEvidenceRefs` del
  request o, por compatibilidad, cuando el review result contiene esas
  evidencias, ignorando refs mixtas que no correspondan a evidencia de test.
  El guardado es idempotente solo si el payload coincide; la misma ref con
  payload distinto es conflicto. El `PlanState` ya consume esas evidencias en
  `run_required_tests`: con `passed` causal avanza a `replan_or_close`, con
  `failed` bloquea como `required-tests-failed`, y pasa las refs al cierre. Esto
  no equivale todavia a un runner shell: la ejecucion/generacion de esas
  evidencias sigue siendo un tramo pendiente de composicion opt-in.
- `orquesta-app-codex-stack` ya cablea `OperationalClosureSource` para el caso
  Director Operativo: lee `WorkflowTaskStore` y `LoadRunEventsV0`, respeta el
  scope `WaitAgentRefs`, exige cadena delivery/review requested/review result
  accepted/accepted review y prueba deliveries en orden estable. Si el cierre
  produce issues, `ContinueAppDirectorV0` los expone en
  `operational_closure_issues` en vez de tragarlos silenciosamente.
  scope resuelto (`WaitAgentRefs`) y solo actua sobre tasks con metadatos
  `operational_director`/contrato funcional, para no cerrar flujos legacy.
- `orquesta-app-codex-stack` incorpora desde el 2026-05-18 un supervisor Codex
  unitario: `SuperviseCodexV0` lanza en el primer tick y llama
  `ContinueV0(ctx, "sigue")` en ticks posteriores hasta `done`, `failed`,
  cancelacion/error o `max_ticks`. Este contrato no toca el core y debe leerse
  como ciclo de vida de agente: el adaptador real tiene que avanzar
  supervisor/drain/replan/outbox `LaunchRuntimeAgent`, registry, ACK y progreso
  ya existentes.
- Desde el corte del 2026-05-22 el conector Codex especializa el objetivo de
  `AgentStartTaskV0` segun `WorkflowTaskV0.WorkProfileKind`: `code_study`,
  `implementation`, `refactor` y `required_tests` mantienen `write_set`,
  `required_tests` y linaje, pero no reciben todos el mismo texto generico de
  "implementar".
- Desde el corte posterior del 2026-05-22, la entrada
  `POST /api/v0/autoprogramming/prepare-run` en `orquesta-app-codex-stack`
  persiste la run/tareas y tambien la deja como candidato de la cola global del
  stack Codex. El supervisor residente del servidor o
  `POST /api/v0/runs/supervise` pueden arrancarla sin `run_ref`; esto sigue
  siendo politica de composicion Codex, no contrato del nucleo ni de MCP/gateway.

Lo pendiente no debe confundirse con lo hecho:

- el `PlanState` ya cubre la salida positiva
  `review_deliveries -> run_required_tests/replan_or_close` cuando existe la
  cadena causal `DeliveryRegistered -> ReviewRequested ->
  ReviewResultRecorded(accepted) -> ReviewAccepted`, y `run_required_tests`
  consume `RequiredTestEvidenceV0` causal para avanzar o bloquear; el codigo
  local observa review negativa con `ReworkRequested` y
  `ReplanDecisionRecorded` en el `PlanState` con prueba focal, y reabre
  `wait_subagents` en una reentrada posterior cuando los followups causales ya
  estan materializados; el cierre
  operacional ya marca el state como `closed` o `blocked` con
  `closure_reason`; el runner/adaptador de tests por puerto existe y tiene
  replay focal sin reejecucion externa; siguen pendientes smoke Codex real con
  runner, replan automatico para blockers no cubiertos y replay/idempotencia
  completa;
- si una composicion real distinta del stack Codex todavia no inyecta una fuente
  `OperationalClosureSource`, el cierre queda pendiente de wiring real en esa
  composicion;
- `wait_subagents` ya tiene scope operativo, estado durable de espera, plan
  state inicial, reentrada por `operational_director_plan_ref`, avance a
  `review_deliveries` cuando el wait queda consumido y avance de review positiva
  a tests/replan por eventos causales del scope; tests requeridos durables ya
  avanzan o bloquean por evidencia causal; la observacion negativa de review y
  el cierre/bloqueo posterior del PlanState estan cubiertos offline; el cierre
  de ola multitarea ya progresa por reentradas sin bloquear por
  `run.open_tasks`; faltan smoke real Codex con runner, replan automatico de
  blockers y replay/idempotencia completa;
- la recursion Codex real sigue pendiente: ya existe contrato unitario para
  `launch -> sigue -> done`, pero falta el adaptador real que conecte `sigue` con
  el ciclo normal de agentes y el ciclo productivo de hijos de hijos con
  parent/child refs, presupuesto global, profundidad/fanout y review causal;
- OPES tiene evidencia de `plan_tema` y `plan_temario`; el bridge ya automatiza
  derivados por fases con `JOB_TYPE_SEQUENCE`, pero los derivados de review,
  validacion y ensamblado aun necesitan smoke real completo con guardas;
- no hay que borrar piezas antiguas sin revisar referencias. Si un documento o
  modulo queda historico, marcarlo como historico y enlazar al documento vigente.

## Documentos vigentes de referencia

- `principio_orquesta_piensa_director.md`: decision transversal vigente sobre
  reparto de juicio entre Orquesta y apps de dominio.
- `guia_nucleo_orquestacion_2026-05-17.md`: mapa operativo de piezas,
  invariantes y reglas para futuros agentes.
- `director_operativo_v1_2026-05-17.md`: contrato vigente del Director
  Operativo V1, corte razonable, espera por cohortes/oleadas y recursion
  gobernada.
- `corte_director_funcionando_tarde_2026-05-17.md`: handoff operativo para
  dejar Orquesta usable hoy sin confundir lo probado con la recursion completa.
- `corte_cierre_generico_director_operativo_2026-05-17.md`: handoff vigente del
  siguiente corte del Director: review por ola, tests durables, replan/close y
  plan state completo como ciclo causal offline generico.
- `matriz_pruebas_reales_y_smoke_2026-05-17.md`: matriz viva de smokes reales,
  guardas opt-in y pruebas focales.
- `corte_opes_como_consumidor_orquesta_2026-05-18.md`: corte vigente de OPES
  como consumidor generico de Orquesta, con `plan_temario -> document_plan`,
  policy editorial OPES y ruta REST/MCP generica aclarada.
- `runbooks/smoke_opes_plan_temario_operadores_2026-05-18.md`: runbook acotado
  para probar `plan_temario` de operadores contra OPES temporal.
- `resultado_prueba_opes_orquesta_plan_temario_operario_2026-05-18.md`:
  evidencia real acotada de OPES temporal, Orquesta REST bridge, Codex real
  `xhigh`, `document_plan` entregado y derivados creados.
- `corte_supervisor_codex_director_2026-05-18.md`: corte vigente de la pieza
  `launch -> sigue -> done` para sesiones Codex supervisadas por Orquesta.
- `arquitectura_plataforma_agentes_2026-05-13.md`: decision de evolucion hacia
  plataforma/nucleo de agentes con dominios consumidores.
- `resultado_prueba_opes_orquesta_plan_limpia_2026-05-15.md`: evidencia viva de
  integracion OPES-Orquesta para `plan_tema`.
- `informe_revision_hexagonal.md`: diagnostico historico util para detectar
  patrones de deuda, pero no es snapshot actual. El repo activo ya no conserva
  varias piezas antiguas que analiza.
- `db_hexagonal_refactor_plan.md`: plan historico util para entender la
  intencion de extraer policy de persistencia, no prueba de que exista hoy un
  `db/` activo en la raiz.

## Documentos historicos o stale

Leer con cuidado:

- `orquesta_v1_vision.md`, `orquesta_v1_roadmap.md` y `BIBLIA_APP_ORQUESTA.md`
  son utiles como historia y vision, pero cualquier lectura centrada en
  "app de programacion" queda subordinada al nucleo neutral reutilizable.
- `integracion_opes_orquesta_2026-05-13.md`,
  `estado_integracion_opes_orquesta_2026-05-13.md`,
  `corte_integracion_opes_orquesta_2026-05-13.md` y
  `resultado_prueba_opes_orquesta_plan_2026-05-14.md` quedan superados, para
  estado OPES, por la prueba limpia `plan_tema` del 2026-05-15 y el corte
  OPES-consumidor del 2026-05-18.
- `manual_programador.md` contiene deriva documentada: referencias tecnicas y
  URLs que no deben usarse como fuente canonica sin revision.
- `uso_actual_app_orquesta.md` conserva valor operativo, pero debe leerse junto
  con la politica servidor-primero y con este estado actual.
- Los recuentos de deuda en informes de arquitectura son utiles para priorizar,
  pero no deben tratarse como cifras actuales si no se recalculan.
- Cualquier documento que hable del antiguo `cmd/db/internal`, de
  `ensureLocalDB` o de rutas SQLite acopladas pertenece al periodo previo al
  saneamiento modular y debe verificarse contra el arbol actual antes de usarse.

## Riesgos abiertos

- La arquitectura hexagonal no esta cerrada: `cmd/` sigue concentrando
  composicion de producto y quedan riesgos de policy dentro de adaptadores de
  runtime/persistencia. No hay que reintroducir el antiguo `db/` por merge.
- El modo servidor-primero existe como direccion, pero aun convive con rutas
  locales/fallback que deben quedar como recuperacion explicita.
- OPES ya valida la composicion de planificacion, pero faltan controles finos
  para pruebas focales y arranque completo de temas derivados.
- Los smokes OPES deben seguir con guardas opt-in, instancia temporal y filtro
  por tipo de job o `job_ref`. El caso peligroso es drenar una cola OPES real
  sin `ORQUESTA_OPES_BRIDGE_JOB_TYPE` ni `ORQUESTA_OPES_BRIDGE_JOB_REF`.
- La espera por agentes ya puede derivar `WaitAgentRefs` desde cohortes u olas
  declaradas, persistirse como `WorkflowTaskWaitStateV0` y acotar la ingesta de
  ACK/deliveries en el stack Codex. Aun falta promover timeout/recovery/review
  posterior a ciclo durable completo. Esperar todos los procesos vivos del run
  vuelve a introducir cuelgues falsos.
- El cierre operativo desde `ContinueAppDirectorV0` depende de que el loop haya
  quedado quiescent y de una `OperationalClosureSource` real. Sin esa fuente, no
  hay que anunciar cierre productivo aunque exista el cerrador offline generico.
- La delegacion recursiva con Codex aun no esta cerrada end-to-end. No anunciar
  "director recursivo completo" hasta tener evidencia real con hijos, nietos,
  limites y review.
- El contrato i18n y su cargador activo tienen brechas historicas documentadas.
- Seguridad operativa, multi-tenant, TLS/mTLS, RBAC y auditoria fuerte siguen
  siendo frentes de cierre antes de considerar la plataforma lista para despliegue
  amplio.
- Hay cambios concurrentes en curso; cualquier nuevo corte debe tener write-set
  explicito y no pisar hotspots ajenos.

## Siguiente incremento

El siguiente incremento del Director Operativo sigue siendo cierre causal
offline generico. No debe presentarse como codigo hecho hasta que exista
implementacion y prueba integrada para cada frente:

- [x] Review por ola/cohorte: `review_deliveries` usa el scope activo y cadena
  causal de task/delivery/review aceptada. No avanza una ola por entregas de
  otra ola ni por cualquier ACK del run.
- [x] Tests durables por evidencia: en modo `programming`,
  `run_required_tests` consume `RequiredTestEvidenceV0` causal antes de cierre,
  avanza con evidencias `passed`, bloquea con `required-tests-failed` y no acepta
  refs de otra review. Pendiente separado: runner/adaptador por puerto que
  ejecute o valide y cree esas evidencias. En `domain_work`, usar artefactos y
  validadores de dominio sin inventar tests de programacion.
- [ ] Replan negativo: si falta entrega, review aceptada, evidencia de tests,
  outbox cero o cierre de tasks, emitir rework/replan causal con intento y refs;
  no cerrar por resumen ni por quietud aparente.
- [x] Plan state inicial: persistir estado vivo del plan con step activo
  `wait_subagents`, ola/cohorte activa, tasks, agentes, pendientes y `wait_ref`.
- [x] Plan state reentrada inicial: `ContinueAppDirectorV0` lee
  `operational_director_plan_ref` y recupera scope de wait sin mirar agentes
  vivos globales.
- [x] Plan state wait consumido: pasar de `wait_subagents` a
  `review_deliveries` cuando los agentes pendientes entregaron.
- [~] Plan state restante: review negativa observada, intentos de replan,
  reentrada tardia a followups causales ya materializados, razon de
  cierre/bloqueo y tests durables ya quedan persistidos. Falta materializar
  replan automatico de blockers y probar replay/idempotencia completa.
- [ ] Evento idempotente: todo avance de review, test, replan y cierre debe
  pasar por comando/evento idempotente con clave estable por refs causales; el
  replay no debe duplicar efectos.

La evidencia debe quedar reflejada en
`docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`. Si una composicion no tiene
`OperationalClosureSource` o validador real, documentarlo como pendiente
verificable para esa composicion.

## Siguiente ruta

1. Mantener la frontera conceptual: core neutral primero, composiciones despues.
2. Cerrar el siguiente tramo causal generico del Director Operativo:
   runner/adaptador real de tests, replan automatico para blockers negativos y
   replay/idempotencia usando los providers existentes, `DirectorTaskStore` y el
   cierre desde `ContinueAppDirectorV0` cuando el loop quede quiescent. Si falta
   fuente real `OperationalClosureSource` en una composicion, documentarlo como
   pendiente verificable.
3. Consolidar `orquesta-domain-work` y los contratos de artefactos como entrada
   comun para OPES, programacion y futuros dominios.
4. Seguir vaciando policy de `cmd` y adaptadores concretos hacia
   servicios/puertos de aplicacion.
5. Endurecer server-first: CLI/web/MCP como clientes finos del control plane.
6. Reforzar pruebas de frontera: contratos de dominio, no duplicacion de
   artefactos, idempotencia, canonicalizacion y ausencia de regresion
   arquitectonica.
7. Separar en documentacion lo vigente de lo historico antes de usar cualquier
   manual como guia operativa.
