# Guia del nucleo de orquestacion - 2026-05-17

Este documento deja la ruta para futuros agentes. Su objetivo es que cualquier
cambio nuevo encaje con el nucleo reutilizable y no vuelva a convertir Orquesta
en una app cerrada de programacion, OPES o Codex.
OPES es un consumidor por conectores, no el producto base del nucleo generico.

## Principio

Orquesta piensa y coordina. Las apps externas aportan dominio.

- Orquesta: director, fases, tareas, capacidad, agentes, evidencias, review,
  rework, cierre, shutdown logico, normalizacion de intencion razonable y
  estadisticas compactas.
- App externa: datos, reglas, permisos, validadores, persistencia, UI/API,
  ensamblado y publicacion final.
- Adaptador: traduce entre contrato externo y trabajo orquestable; nunca copia
  internals ni DB de la app externa.
- Conector externo: implementa puertos en composicion con refs opacas. No
  convierte reglas de producto, DSN, filesystem, OAuth, proveedor ni runtime en
  dependencia del nucleo.

## Mapa de piezas

| Pieza | Responsabilidad | Estado frente al nucleo |
| --- | --- | --- |
| `modulos/orquesta-core-workflow` | Maquina pura: comandos, eventos, reducer, outbox, replay. | Alineada. No debe importar runtime, DB, Codex, OPES, web ni MCP. |
| `modulos/orquesta-orchestration-core` | Loop de aplicacion por puertos: candidatos, stores, perfiles de trabajo, outbox, launch/stop/delivery/review. | Alineada como capa de aplicacion. Puede importar `orquesta-runtime` neutral, no adaptadores concretos. |
| `modulos/orquesta-director-operativo` | Contrato puro de Director Operativo V1: plan vivo, subagentes, espera, review, replan y delegacion recursiva gobernada. | Alineada como DTO/validacion previa a integracion. No ejecuta runtime ni duplica scheduler/replanner. |
| `modulos/orquesta-domain-work` | Contratos genericos de jobs, records filtrables y artefactos externos. | Alineada. Debe seguir sin OPES, programacion, REST, MCP, runtime ni modelo. |
| `modulos/orquesta-document-plan-expander` | Expande `DomainDocumentPlanV0` a `DomainWorkJobRequestV0[]` por puerto. | Alineada. No ejecuta jobs ni importa adaptadores. |
| `modulos/orquesta-domain-work-memory` | Conector volatil de referencia para crear jobs y leer records `Request+Job` filtrados. | Adaptador neutral sin IO; util para tests offline. |
| `modulos/orquesta-domain-work-file` | Conector durable file-based para crear jobs y leer records `Request+Job` filtrados. | Adaptador filesystem. No es nucleo puro ni bundle DB. |
| `modulos/orquesta-domain-work-sql` | Adaptador SQL driver-neutral para crear jobs y leer records `Request+Job` filtrados. | Adaptador DB externo. Recibe `*sql.DB`; no registra drivers ni entra al nucleo. |
| `modulos/orquesta-app-change` | Cambio de app y proyeccion de `external_work` hacia contratos. | Alineada si mantiene refs opacas y no conoce runtime/proveedor. |
| `modulos/orquesta-app-change-director-source` | Fuente de decisiones/tareas para cambios externos ya aceptados. | Alineada con cautela: contiene taxonomia documental concentrada; no debe crecer como OPES interno. |
| `modulos/orquesta-app-runner` | Flujo historico app-plan para preparar/ejecutar planes de app. | Separado del Director Operativo: no participa en waits por cohorte/ola/parent ni en `OperationalDirectorPlanStateV0`. |
| `modulos/orquesta-external-work-run` | Crea run operativo para trabajo externo ya especificado. | Alineada. No decide contenido ni arranca director inicial. |
| `modulos/orquesta-app-director-service` | Servicio de entrada/continuacion del director de app; materializa plan, espera por scope y puede cerrar despues de loop quiescent. | Capa de aplicacion. Depende de puertos inyectados; `OperationalClosureSource` real pertenece a composicion/adaptador. |
| `modulos/orquesta-opes-connector` | Cliente publico OPES. | Adaptador de dominio, permitido fuera del nucleo. |
| `modulos/orquesta-opes-bridge` | Traduce cola OPES a `external-work/run`. | Adaptador de dominio. Debe seguir siendo opt-in y filtrable por tipo. |
| `modulos/orquesta-runtime` | Contratos neutrales de runtime/proceso/launch/stop/snapshot. | Alineada como puerto/adaptador neutral. |
| `modulos/orquesta-runtime-codex*` | Protocolo concreto Codex: prompt, ACK, progress, files. | Adaptador concreto. Nunca debe entrar al core. |
| `modulos/orquesta-app-codex-stack` | Composicion real actual con Codex, web/API/MCP y domain-work delivery. | Producto exterior, no nucleo. Es el camino real vigente, pero no debe definir los contratos puros. |
| `modulos/orquesta-state-file` y `modulos/orquesta-run-file` | Persistencia JSON file-based actual. | Adaptadores concretos. Utiles hoy, sustituibles por bundle futuro. |
| `cmd/orquesta-server` | Composition root productivo del servidor. | Sigue siendo el punto mas acoplado: Codex/file-store/OPES se cablean aqui. |

## Flujo vigente de trabajo externo

```text
app externa
  -> contrato DomainWork/AppChange external_work
  -> /api/v0/external-work/run o bridge de dominio
  -> run + programacion + microtarea externa
  -> stack/runtime configurado
  -> artifact validado
  -> conector devuelve artefacto a la app propietaria
```

Para OPES, el bridge y el conector son solo una implementacion de ese flujo. Si
otra app entra manana, debe reutilizar el contrato y aportar su propio adaptador,
no meter reglas de esa app en `orquesta-domain-work`.
La misma regla aplica al cierre: una app o composicion externa debe aportar por
puerto la fuente que traduce entregas, reviews y evidencias reales a refs
causales del nucleo; Orquesta no lee internals del producto para deducirlas.

Los perfiles `code_study`, `implementation`, `refactor`, `required_tests`,
`documentation`, `review` y `domain_work` viven como contrato neutral
`WorkProfileV0`/`WorkflowTaskV0.work_profile_kind`. El scheduler resuelve rol y
capacidad por `WorkflowTaskProfileResolverPortV0`; Codex, OPES o cualquier otro
conector solo mapean ese perfil a su ejecucion concreta.

## Corte Director Operativo

El corte vigente del Director Operativo esta en
`director_operativo_v1_2026-05-17.md`. Para futuros agentes, la idea clave es
que el plan neutral no es la ejecucion: debe materializarse como tasks puras del
workflow, despues como candidatos accionables en la capa de aplicacion y solo
entonces como comandos/eventos de outbox hacia adaptadores concretos.
El handoff practico para dejarlo funcionando esta tarde esta en
`corte_director_funcionando_tarde_2026-05-17.md`.
El handoff vigente para el siguiente corte esta en
`corte_cierre_generico_director_operativo_2026-05-17.md`: no reabre P1
`WaitAgentRefs`; convierte la salida de la espera en cierre causal offline
generico con review por ola, tests durables, replan/close y plan state completo.

El director espera por cohortes u oleadas declaradas en el plan, no por todos
los agentes vivos del run. La delegacion recursiva es valida para programacion y
dominio/OPES solo si queda gobernada por parent/child refs, presupuesto,
profundidad/fanout, refs opacas o write-set, artefactos esperados y review de
Orquesta.

`orquesta-app-runner` no es una entrada paralela al `PlanState` del Director:
pertenece al flujo app-plan historico. Sus unidades, progreso y refs no deben
mezclarse con `wait_cohort_ref`, `wait_wave_ref`, `wait_parent_task_ref` ni con
metadata de `WorkflowTaskV0` usada para waits acotados. Una migracion futura
debe delegar en `app-director-service` o anadir metadata compatible en
`WorkflowTaskStore`/`OperationalDirectorPlanStateV0`; no debe adaptar refs de
app-plan como si fueran cohortes, olas o parent tasks del Director.

Estado real del primer corte:

- Contrato puro implementado en `modulos/orquesta-director-operativo`.
- Proyeccion a olas implementada por `BuildOperationalDirectorWaveWorkV0`.
- Materializador inicial en
  `modulos/orquesta-orchestration-core/operational_director_materializer_v0.go`.
- El materializador actual solo convierte items `launch_subagents` listos en
  `WorkflowTaskV0` y comandos `CreateMicrotask`; no debe saltarse workflow ni
  outbox.
- `WaitAgentRefs` ya permite esperar agentes concretos en
  `orquesta-orchestration-core` y `orquesta-app-codex-stack`.
- `app-director-service` ya puede recibir `wait_cohort_ref`, `wait_wave_ref` o
  `wait_parent_task_ref` en start/continue y derivar `WaitAgentRefs` desde
  `WorkflowTaskStore`.
- `ContinueAppDirectorV0` ya puede recibir un `OperationalDirectorPlanV0` listo,
  materializar `launch_subagents`, derivar la primera ola/cohorte materializada
  como wait acotado y reentrar al loop progresivo.
- `StartAppDirectorV0` ya puede hacer ese primer tramo desde un plan operativo
  inicial cuando la request trae contratos funcionales explicitos: abre run,
  publica voto/decision/fase/contratos por causa, materializa
  `launch_subagents`, registra wait/plan state por puertos y entra al loop con
  scope de ola/cohorte. Sin esos contratos, el plan directo se rechaza antes de
  persistir.
- `WorkflowTaskWaitStateV0` deja la espera registrada con causa, refs de tasks,
  agentes objetivo y agentes pendientes; `orquesta-state-file` la persiste.
- `app-director-service` ya puede recuperar ese wait state persistido en
  reentrada, validarlo contra `PlanState` y usar sus agentes pendientes sin
  ampliar la espera por ola/cohorte. Al consumir el wait hacia review lo marca
  `continued` de forma idempotente. El stack Codex y el servidor cablean el store
  por puerto.
- El stack Codex ya cierra P1 WaitAgentRefs: el scope no vacio limita pending,
  wait e ingesta de ACK/deliveries; `WaitAgentRefs` vacio conserva el drenaje
  legacy del run completo.
- La composicion de `app-director-service` ya inyecta `DirectorTaskStore` en el
  provider de review/replan, por lo que un rework `split_task` puede persistir
  nuevas `WorkflowTaskV0`.
- Ese camino de `split_task` ya valida el primer corte de recursion gobernada:
  si un followup declara `parent_task_ref`, el nucleo lee el parent desde
  `WorkflowTaskStore`, exige parent reflejado, `delegation_depth=parent+1`,
  `wave_ref`/`cohort_ref` consistentes y fanout dentro de `max_child_agents`
  antes de guardar microtareas nuevas.
- `OperationalDirectorClosureV0` existe en `orquesta-orchestration-core` como
  cerrador generico offline: exige entrega registrada, review aceptada y
  evidencia durable de tests requeridos, valida la cadena causal por eventos y
  emite comandos causales para cerrar task, validacion final y run.
- La evidencia de tests requeridos vive como `RequiredTestEvidenceV0`: ref
  durable por run/task/comando, status `passed` o `failed`, y refs de delivery,
  review request, review result y accepted review. El cierre solo acepta
  evidencias `passed` que empatan con `WorkflowTaskV0.RequiredTests` y con la
  cadena causal de la entrega revisada. La evidencia debe tener artefactos en
  `evidence_refs`, ser solicitada por el cierre y no puede sobrescribirse con
  otro payload bajo la misma ref.
- `ContinueAppDirectorV0` intenta ese cierre solo despues de que el loop quede
  quiescent y solo si la composicion inyecta `OperationalClosureSource`.
  `orquesta-app-codex-stack` ya aporta una fuente real para tasks del Director
  Operativo, derivada de task store + eventos; otras composiciones deben
  inyectar su fuente por puerto.
- `OperationalDirectorPlanStateV0` ya existe con store en memoria, store
  file-based y escritura inicial desde `ContinueAppDirectorV0` al tramo
  `launch_subagents -> wait_subagents`; `ContinueAppDirectorV0` tambien puede
  leerlo para reentrada inicial por `operational_director_plan_ref` y avanzar a
  `review_deliveries` cuando el wait queda consumido.
- La salida de esa espera ya tiene ciclo offline/fake-runtime probado:
  `review_deliveries` por ola, `run_required_tests` con
  `RequiredTestEvidenceV0`, runner por puerto, review negativa observada,
  replan causal de tests/cierre, `replan_or_close`, `close`, blockers durables y
  replay/idempotencia del ciclo probado. Esto no convierte en cerrados los
  smokes reales de ola/cohorte Codex amplia, recursion Codex real ni OPES
  temporal real de derivados/cierre.

Checklist antes de extenderlo:

1. Si el plan esta `needs_context`, no materialices agentes.
2. Si un item no trae write-set o refs opacas suficientes, bloquea o pide
   contexto.
3. Si produces una task, conserva objetivo actual, tests requeridos, write-set,
   function contract refs y dependencias.
4. Si produces un efecto externo, debe pasar por comando/evento/outbox
   idempotente.
5. Si esperas, espera `WaitAgentRefs` o cohorte/ola concreta; no bloquees por
   todos los agentes vivos.
6. Si aceptas recursion, registra parent/child refs y aplica limites de
   profundidad, fanout, presupuesto y review.

## Invariantes

- Core puro no conoce Codex, OPES, DB concreta, web, MCP, HTTP, CLI, HOME,
  OAuth, tokens, paths locales ni proveedor/modelo.
- `orquesta-domain-work` solo define DTOs, validacion y puertos genericos,
  incluido el source filtrado de records `Request+Job`.
- El director de Orquesta decide plan/fases/agentes/modelo/runtime mediante el
  plano de orquestacion. La app externa no esconde planificacion inteligente en
  workers propios.
- Los contratos no deben depender de que un agente use el string perfecto. Los
  adaptadores pueden normalizar alias y formas equivalentes; el director debe
  poder reparar nombres cercanos. La validacion deterministica corta solo cuando
  no puede preservar seguridad, causalidad, refs opacas o permisos.
- Los adaptadores pueden ser concretos, pero deben vivir fuera del nucleo y ser
  opt-in si tocan red, procesos, proveedor o datos reales.
- La persistencia real actual es file-based JSON, no una verdad arquitectonica.
  Cambiarla requiere bundle/puertos y pruebas de conformidad.
- Codex es el runtime productivo actual, no el unico runtime permitido por el
  diseno.

## Conectores DB genericos

`orquesta-domain-work` define el contrato puro: DTOs, validacion y puertos. No
debe conocer `database/sql`, drivers, DSN, migraciones ni schema operativo.

`orquesta-domain-work-sql` es un adaptador externo de referencia sobre
`database/sql`: recibe `*sql.DB`, tabla, estilo de placeholders y un clasificador
opcional de unique violation. Sirve para futuros bundles DB, pero no abre
conexiones, no registra drivers, no crea schema y no comparte DB interna con apps
externas.

Un conector DB productivo debe vivir en composicion o en un modulo exterior:
elige driver, DSN, pooling, secretos, migraciones, schema, politica de backend y
smoke real opt-in. No se importa desde core, director, domain-work ni expander.

## Que funciona hoy

- La matriz documenta la suite y los cortes focales que han pasado en sus
  respectivos cierres. Para un cambio transversal nuevo hay que reejecutar
  `go test -count=1 ./...`; este documento no es evidencia de una reejecucion
  actual.
- `orquesta-core-workflow` y `orquesta-domain-work` se mantienen puros por
  imports.
- `orquesta-orchestration-core` coordina por puertos y no importa Codex/OPES.
- `WorkflowTaskV0` ya conserva metadata neutral de linaje operativo:
  `parent_task_ref`, `cohort_ref`, `wave_ref`, `delegation_depth`,
  `max_child_agents` y `child_task_refs`.
- El contrato de director externo `create_microtask` y su puente a workflow ya
  preservan ese mismo linaje al crear `WorkflowTaskV0`; esto solo cierra la
  trazabilidad durable, no el smoke real de recursion con agentes vivos.
- El paquete neutral `AgentStartTaskV0` tambien puede transportar ese linaje, y
  el stack Codex lo copia desde `WorkflowTaskStore` al resolver tareas de
  programacion.
- `orquesta-orchestration-core` puede derivar `WaitAgentRefs` desde
  `WorkflowTaskV0` persistidas con `WorkflowTaskWaitAgentRefsV0`, filtrando por
  ola, cohorte o parent task.
- `orquesta-app-director-service` conecta esa derivacion al loop progresivo:
  `wait_cohort_ref`, `wait_wave_ref` y `wait_parent_task_ref` se resuelven a
  refs de agente antes de llamar al core.
- `ContinueAppDirectorV0` conecta el primer tramo del Director Operativo:
  plan listo -> materializador -> workflow tasks -> wait acotado -> loop
  progresivo.
- `StartAppDirectorV0` conecta el mismo tramo para planes iniciales listos solo
  si la composicion aporta contratos funcionales explicitos y stores requeridos;
  no sustituye al director vivo ni relaja causalidad.
- `WorkflowTaskWaitStateV0` registra la causa del wait y
  `orquesta-state-file` lo guarda en JSON atomico.
- `DrainRunV0` transporta `WaitAgentRefs` a
  `AgentDeliveryObservationRequestV0`, envuelve/filtra `DeliverySource` y filtra
  otra vez antes de aplicar ACK/deliveries. La prueba focal
  `TestDrainRunV0WaitAgentRefsNoIngiereACKFueraDeScope` fija que un ACK fuera
  del scope no queda registrado.
- `app-director-service` pasa `DirectorTaskStore` al provider de review/replan,
  asi que el camino de `split_task` puede guardar microtareas de rework.
- `review/rework/replan split_task` usa esa misma metadata para rechazar
  followups recursivos con parent/depth/fanout/ola/cohorte imposibles antes de
  materializarlos; ademas ya no queda fijado a `programacion`: acepta la fase
  actual neutral del run y, desde `revision`, solo fases de trabajo soportadas
  como `programacion`, `documentacion` o `integracion`. Sigue sin meter Codex,
  OPES, runtime ni presupuesto proveedor.
- `OperationalDirectorClosureV0` puede cerrar causalmente un run con review
  aceptada y tests requeridos evidenciados. `ContinueAppDirectorV0` lo conecta
  al final del loop quiescent cuando recibe un `OperationalClosureSource`; el
  stack Codex ya cablea ese source para tasks con metadatos del Director
  Operativo.
- `RequiredTestEvidenceV0`, `RequiredTestEvidenceReaderPortV0`,
  `RequiredTestEvidenceWriterPortV0` y `RequiredTestEvidenceStorePortV0` ya
  existen en `orquesta-orchestration-core`; `orquesta-state-file` los persiste.
  El runner por puerto y el executor local opt-in ya generan evidencia cuando la
  composicion los inyecta; el nucleo sigue sin conocer shell/runtime concreto.
- OPES tiene integracion real documentada para `plan_tema` y `plan_temario`
  acotados a plan, smokes REST de `domain_work`/visual y automatizacion offline
  de derivados hasta `assemble_topic`; falta smoke real completo de derivados y
  cierre OPES temporal.
- El stack Codex entrega artefactos de domain-work y recupera algunos casos de
  ACK ausente con artefacto valido.
- `EXT-NO-OPES` cubre una app externa temporal HTTP/file con submitter real
  opt-in, `codex-fake`, review, tests requeridos durables y cierre operativo por
  refs opacas.

## Huecos abiertos

- `cmd/orquesta-server` todavia compone el producto sobre Codex y file stores.
  No hay modo productivo runtime-neutral completo.
- OPES sigue siendo la app externa real mas ejercitada, pero no es la unica ruta
  no-core validada: `EXT-NO-OPES` cubre una app HTTP/file temporal con runtime
  fake y cierre por refs opacas. Falta una app externa no-OPES productiva o con
  proveedor Codex real.
- El bridge generico existe solo como base parcial; la CLI visible sigue siendo
  `opes-drain-once`.
- El Director Operativo ya tiene contrato, materializador parcial, espera
  durable, plan state reentrable, avance wait->review, review positiva/negativa,
  runner de tests por puerto, tests durables, replan causal, `replan_or_close`,
  cierre/bloqueo y replay focal para el ciclo probado. Fuentes
  `OperationalClosureSource` de otras composiciones siguen siendo trabajo de
  wiring por puerto.
- `WaitAgentRefs` cubre refs de agente y ya puede derivarse desde
  `cohort_ref`/`wave_ref` de tasks en el servicio del director. Tambien acota
  ingesta de ACK/deliveries en el stack Codex. Hay harness fake y test opt-in
  real para ola/cohorte amplia (`CODEX-WAVE-REAL`); falta ejecutarlo con Codex
  real y dejar evidencia operativa.
- La recursion Codex sigue pendiente como prueba real completa; el linaje
  neutral ya llega a `WorkflowTaskV0` y al paquete de arranque del agente, pero
  no se debe asumir que el stack productivo ya gobierna hijos de hijos.
- `modulos/orquesta-document-plan-expander` materializa de forma neutral
  `DomainDocumentPlanV0 -> DomainWorkJobRequestV0[]`, fuera de adaptadores
  concretos. Sirve a cualquier dominio que entregue un plan documental validado;
  la creacion real de jobs, DB, colas y persistencia queda en conectores.
- `modulos/orquesta-domain-work-memory` implementa en memoria
  `DomainWorkJobRecordStorePortV0` para probar esos flujos sin producto externo.
  Es referencia volatil de idempotencia, lectura filtrada y frontera.
- `modulos/orquesta-domain-work-file` implementa el mismo store con snapshot JSON
  atomico, replay tras reinicio y lectura filtrada de records. Es referencia
  durable file-based para jobs de dominio, sin OPES, runtime, red ni DB.
- `modulos/orquesta-domain-work-sql` implementa el store sobre `database/sql` con
  `*sql.DB` inyectado. Es adaptador externo para futuros bundles DB; soporta
  placeholders `question`/`dollar` y clasificador opcional de unique violation,
  pero no abre DSN, no registra drivers y no se consume desde el nucleo.
- `modulos/orquesta-domain-work/contracttest` fija la conformidad comun de esos
  stores; cualquier futuro conector DB/cola/REST debe ejecutar esa suite antes de
  entrar en composicion.
- `cmd/orquesta-server` puede inyectar ese backend file de forma opt-in para
  `/api/v0/domain-work create_job`; no habilita entrega de artefactos ni lo
  importa desde el nucleo.
- Hay adaptador SQL externo para `domain_work`, pero no hay bundle productivo DB
  alternativo a JSON file stores ni wiring de servidor para seleccionarlo.
- Falta cerrar una matriz real completa: derivados OPES de revision/ensamblado,
  reinicio con agentes reales vivos, shutdown real y metricas de uso por
  proveedor.

## Recuperacion de microtareas

`OrchestrationRunV0.Tasks` y el evento `MicrotaskCreated` conservan refs
compactas. La descripcion completa de cada `WorkflowTaskV0` vive en
`WorkflowTaskStore`: write-set, criteria, metadata de ola/cohorte, parent/child
refs y limites de delegacion. La espera viva se guarda como
`WorkflowTaskWaitStateV0`. Por tanto, un replay solo de eventos no basta para
reconstruir waits por cohorte/ola ni recursion gobernada. Cualquier bundle
productivo debe persistir run, tasks y wait state, o rematerializar las tasks
desde una fuente idempotente antes de reanudar el director.
La reentrada actual prefiere `WorkflowTaskWaitStateV0` persistido cuando hay
`wait_ref` y estado `waiting`, y el consumo normal del wait ya persiste
`continued`. Cuando el loop gestionado agota `MaxExternalWaits` con senial real
(`attempts=max+1` y `external_waits=max`, todos continuados), el servicio marca
el wait como `expired` y bloquea el `PlanState` en `external-wait-exhausted`
para no reabrir la misma espera. Cuando el cierre operativo exitoso deja un wait
antiguo aun `waiting` referenciado por el plan, el servicio lo marca `cleared`
por puerto y no toca waits ya `continued` o `expired`.

## Cierre de ingesta por scope

P1 WaitAgentRefs queda cerrado para el stack Codex. Para una espera acotada por
cohorte/ola, el mismo scope gobierna pending, wait e ingesta de observaciones:

- `DrainRunRequestV0.WaitAgentRefs` llega a
  `AgentDeliveryObservationRequestV0`.
- `DrainRunV0` envuelve el `DeliverySource` con
  `drainDeliverySourceForWaitAgentRefsV0` cuando el scope no esta vacio.
- `drainAvailableObservationsV0`, `applyDrainObservationsV0` y el
  `DeliveryCandidateProviderV0` vuelven a filtrar por `agent_ref` antes de
  registrar ACK/deliveries.
- `CodexDeliveryObservationSourceV0` acota la consulta de descriptors y omite
  agentes fuera de scope antes de leer/verificar ACKs.
- Las recuperaciones y submits de artefactos de `domain_work` reciben el mismo
  `WaitAgentRefs` por el request de drain y no producen efectos fuera de scope.
- El modo legacy se conserva solo cuando `WaitAgentRefs` esta vacio: ahi el
  drain puede ingerir observaciones de todo el run.
- La evidencia minima cubre core, source Codex, drain y `domain_work`:
  `TestDeliveryCandidateProviderV0PropagaYFiltraWaitAgentRefs`,
  `TestRunProgressiveLoopV0PropagaWaitAgentRefsAObservationSource`,
  `TestCodexDeliveryObservationSourceV0WaitAgentRefs*`,
  `TestDrainRunV0WaitAgentRefsNoIngiereACKFueraDeScope` y
  `TestDomainWork*WaitAgentRefs`.

Esto cierra el scope de ingesta, no los smokes reales amplios. El ciclo
funcional de ola esta probado offline/fake-runtime con review, tests durables,
`replan_or_close`, `close` y plan state; queda pendiente reproducirlo con Codex
real de ola/cohorte amplia, recursion real y OPES temporal real de
derivados/cierre. El handoff de ese tramo es
`corte_cierre_generico_director_operativo_2026-05-17.md`.

## Como trabajar sin romper el nucleo

1. Clasifica la pieza: core, aplicacion, contrato, adaptador, composicion o UI.
2. Lee el `AGENTS.md` local antes de editar.
3. Si necesitas importar algo concreto, verifica que la capa lo permite.
4. Anade o actualiza pruebas de frontera cuando cruces capas.
5. Documenta el cambio en `docs/decisiones.md` o `docs/pruebas.md` del modulo
   si afecta contrato o evidencia.
6. Para cambios transversales, actualiza este documento o
   `docs/estado_actual_2026-05-17.md`.
7. Ejecuta `git diff --check` y `go test -count=1 ./...`.

## Prueba de frontera

La prueba raiz `TestNeutralOrchestrationPackagesDoNotImportProductAdapters`
bloquea imports concretos en paquetes neutrales:

- core;
- core workflow;
- domain-work;
- orchestration-core;
- director;
- director-operativo;
- external-work-run;
- document-plan-expander;
- domain-work-memory;
- app-change;
- app-change-director-source.

`orquesta-domain-work-file` y `orquesta-domain-work-sql` no entran en esa lista
porque son adaptadores externos, pero quedan prohibidos como imports desde
paquetes neutrales. La prueba raiz tambien bloquea `database/sql` y drivers
conocidos (`pgx`, `lib/pq`, MySQL y SQLite) en esas capas.

Desde el corte del 2026-05-23,
`TestNeutralOrchestrationPackagesDoNotDependOnFactoryOrHTTP` bloquea tambien
dependencia transitiva de los paquetes neutrales ya limpios hacia
`orquesta/modulos/orquesta-factory` y `net/http`. El hueco conocido queda fuera
de esa prueba para no crear rojo: `orquesta-app-director-intake`,
`orquesta-app-director-service`, `orquesta-app-planner` y `orquesta-app-runner`
siguen importando `orquesta-factory`, que a su vez importa `net/http`. Cuando
ese acoplamiento se retire o se mueva a un adaptador/composicion, esos paquetes
deben anadirse a la prueba transitiva.

Si falla, no la relajes para hacer pasar el cambio. Primero decide si la pieza
esta en la capa correcta. Si hace falta un adaptador, crea o usa un modulo
exterior.
