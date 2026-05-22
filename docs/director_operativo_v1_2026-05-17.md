# Director Operativo V1 - 2026-05-17

Este documento orienta a futuros agentes sobre el siguiente cierre funcional del
director. No sustituye a `principio_orquesta_piensa_director.md`: lo aterriza en
un primer corte ejecutable y verificable.

## Problema observado

La experiencia real muestra una brecha entre el principio deseado y el
comportamiento disponible:

- Codex usado como coordinador asistido por una persona funciona mejor que el
  director interno actual cuando hay que mantener un plan vivo, esperar a otros
  agentes, revisar entregas parciales, replanificar y cerrar con pruebas.
- El director interno ya puede emitir decisiones y arrancar agentes, pero en
  pruebas reales se ha desviado hacia tareas historicas, ha dependido demasiado
  de validaciones de forma y ha necesitado supervision humana para entender el
  estado operativo completo.
- Si Orquesta quiere ser nucleo reutilizable, no basta con lanzar workers: debe
  conservar el juicio operativo dentro del director y dejar al humano como
  operador/revisor, no como coordinador manual de cada paso.

El objetivo no es "meter mas Codex" en el nucleo. El objetivo es capturar el
patron que hoy hace bien una coordinacion Codex asistida y convertirlo en un
loop interno neutral de Orquesta.

## Objetivo

Director Operativo V1 debe gobernar un run con estas capacidades minimas:

- mantener un plan vivo con tareas, dependencias, write-set, pruebas requeridas,
  bloqueos y siguiente accion;
- crear subagentes acotados por contrato, no por instruccion libre;
- permitir delegacion recursiva gobernada cuando el trabajo lo requiera: un
  agente puede pedir subagentes, pero el director conserva presupuesto,
  profundidad maxima, fanout, parent/child refs y review;
- esperar entregas, progreso, preguntas, checkpoints o timeouts sin inventar
  estado;
- esperar por cohorte u ola de trabajo, no por "todos los agentes vivos" del
  run;
- revisar entregas contra criterios, write-set, artefactos y pruebas;
- pedir rework o replan cuando una entrega no cierre el objetivo;
- cerrar fases y run solo cuando exista evidencia suficiente, incluida
  validacion final o pruebas requeridas segun el dominio;
- preguntar al usuario o a la app externa cuando falte contexto real.

El director dirige. No debe hacer todo el trabajo pesado. Puede permitir que un
agente coordine subagentes en un subarbol, pero ese subarbol sigue siendo parte
del plan vivo de Orquesta y consume presupuesto del director.

## Frontera arquitectonica

El comportamiento pertenece al plano de orquestacion:

- `orquesta-core-workflow`: contratos puros de plan, tareas, eventos,
  revision, replan y cierre, sin runtime concreto;
- `orquesta-orchestration-core`: loop de aplicacion por puertos para observar
  estado, seleccionar candidatos, arrancar/parar agentes y aplicar decisiones;
- adaptadores/composiciones: runtime real, modelo, cuota, proveedor, stores,
  Codex, OPES, web, CLI, MCP y smokes reales.

OPES, programacion u otra app externa aportan datos, reglas, validadores y refs
opacas. El director puede usar ese contexto por contrato, pero no debe inventar
taxonomia OPES, filesystem externo, DB ajena ni conocimiento de dominio que no
le hayan entregado.

## Corte de materializacion

El siguiente corte del Director Operativo no debe quedarse en DTOs bonitos. Debe
materializar el plan operativo neutral en el workflow real de Orquesta:

```text
plan operativo neutral
  -> workflow tasks versionadas
  -> candidatos accionables por capa de aplicacion
  -> comandos/eventos y outbox
  -> runtime/adaptador concreto fuera del nucleo
  -> entrega/review/replan/cierre
```

Reglas de materializacion:

- El plan operativo expresa intencion, dependencias, cohortes, refs opacas,
  write-set si aplica, presupuesto, profundidad y criterio de cierre.
- `orquesta-core-workflow` debe persistir el resultado como tasks/eventos puros:
  no prompt, no modelo, no Codex, no OPES interno y no ruta local.
- `orquesta-orchestration-core` lee tasks desbloqueadas y produce candidatos
  accionables: lanzar agente, detener agente, registrar progreso, registrar
  entrega, revisar, pedir rework, replanificar o cerrar.
- El outbox es la frontera causal hacia efectos externos. Nada arranca runtime
  real solo por existir en el plan; primero debe existir task/evento/candidato y
  despues comando en outbox con idempotencia y ACK.
- La composicion/adaptador decide runtime, proveedor, modelo, paths y stores
  concretos. Esa decision no vuelve al plan puro salvo como evidencia
  observada, refs y estado compacto.

## Estado tras los cortes del 2026-05-17

El corte actual ya existe, pero sigue siendo deliberadamente estrecho.

Implementado:

- `modulos/orquesta-director-operativo` define el contrato puro:
  `OperationalDirectorRequestV0`, `OperationalDirectorPlanV0`,
  `OperationalDirectorWaveWorkV0`, validaciones, presupuestos, contexto
  insuficiente, write-set/tests y delegacion recursiva gobernada.
- `BuildOperationalDirectorWaveWorkV0` proyecta el plan a olas topologicas y
  work items neutrales.
- `modulos/orquesta-orchestration-core/operational_director_materializer_v0.go`
  materializa planes `ready` mediante puertos: `RunStorePortV0`,
  `WorkflowTaskWriterPortV0` y `EventSinkPortV0`.
- El materializador actual solo selecciona items `launch_subagents`, crea
  `WorkflowTaskV0`, conserva acceptance criteria con `objetivo_actual`,
  write-set, tests requeridos, dependencias y function contract refs, y emite
  `CreateMicrotask` con idempotencia.
- `WorkflowTaskV0` conserva metadata neutral de linaje operativo:
  `parent_task_ref`, `cohort_ref`, `wave_ref`, `delegation_depth`,
  `max_child_agents` y `child_task_refs`.
- `WorkflowTaskWaitAgentRefsV0` deriva refs de agentes desde tasks abiertas del
  run filtrando por ola, cohorte o parent task.
- `app-director-service` acepta `wait_cohort_ref`, `wait_wave_ref` y
  `wait_parent_task_ref` en start/continue y los convierte en `WaitAgentRefs`
  antes de entrar al loop progresivo.
- `ContinueAppDirectorV0` acepta un `OperationalDirectorPlanV0` listo,
  materializa `launch_subagents` antes del loop, deriva la primera ola/cohorte
  materializada como espera acotada y reentra sin esperar agentes ajenos.
- `WorkflowTaskWaitStateV0` registra espera durable neutral con causa, refs de
  tasks, agentes objetivo y agentes pendientes. `orquesta-state-file` persiste
  ese estado en JSON file-based.
- El stack Codex cierra P1 WaitAgentRefs para ingesta: `DrainRunV0` transporta
  el scope a observaciones, filtra ACK/deliveries por `agent_ref` y conserva
  compatibilidad legacy cuando `WaitAgentRefs` esta vacio.
- `modulos/orquesta-orchestration-core/operational_director_materializer_v0_test.go`
  prueba que un plan listo genera microtarea y outbox pendiente, y que un plan
  bloqueado por contexto insuficiente no lanza agentes.
- `RequiredTestEvidenceV0` existe como contrato/store de evidencia durable para
  tests requeridos: ref de evidencia, run, task, comando exacto, status
  `passed`/`failed`, delivery, review request, review result, accepted review y
  refs adicionales de evidencia. El modelo exige `evidence_refs` y el store es
  inmutable por `(run_ref, evidence_ref)`: misma ref y payload igual es
  idempotente; misma ref y payload distinto es conflicto.
- `OperationalDirectorClosureV0` usa esas evidencias para bloquear el cierre de
  tareas con `WorkflowTaskV0.RequiredTests` si falta un resultado `passed`
  causal y solicitado por el request. Tambien valida que entrega, review
  request, review result y accepted review vengan del historial de eventos del
  mismo run, y exige `EventSink` para que el cierre no mute estado sin eventos.
- `orquesta-state-file` persiste `RequiredTestEvidenceV0`; el stack Codex puede
  construir un `OperationalDirectorClosureRequestV0` desde task store, eventos,
  scope `WaitAgentRefs` y evidencias de test ya guardadas para tasks del
  Director Operativo. La fuente del stack ignora evidencia mixta no-test dentro
  del review result y exige cadena completa de review requested/result/accepted.

Pendiente:

- materializar el avance posterior a `wait_subagents` como ciclo durable:
  `review_deliveries` por ola, ejecucion/generacion durable de
  `RequiredTestEvidenceV0` desde `run_required_tests`,
  `replan_or_close`/`close` y plan state completo;
- conectar review/rework/replan del Director Operativo con entregas reales del
  stack Codex y de `domain_work`;
- cerrar recursion Codex end-to-end: hijos y nietos con parent/child refs,
  presupuesto global, profundidad/fanout, ACK/artefactos y review antes de
  consumir decisiones;
- conectar el expander neutral `DomainDocumentPlanV0 -> DomainWorkJobRequestV0[]`
  con el ciclo real del director; ya existe un conector durable file-based de
  referencia para `DomainWorkJobCreatorPortV0`, pero el ciclo real del director
  aun no lo cablea como estado operativo;
- crear smoke real focal del ciclo completo sin tocar OPES productivo ni colas
  amplias.

## Ruta de cierre para la tarde del 2026-05-17

Alcance del corte: cerrar primero un ciclo durable offline antes de Codex real,
OPES real o recursion con nietos. La ruta no crea otro scheduler: reutiliza
workflow tasks, candidatos, outbox, waits por metadata y providers existentes.
El handoff especifico del siguiente tramo esta en
`corte_cierre_generico_director_operativo_2026-05-17.md`.

Estado del P0: cerrado para materializacion `launch_subagents`, wait acotado y
estado durable de espera. P1 WaitAgentRefs tambien queda cerrado para ingesta
Codex: pending, wait y ACK/deliveries usan el mismo scope. Queda P1 del ciclo
del Director para conectar review/rework/replan/cierre como flujo durable
completo. Ese P1 restante no debe describirse como integrado hasta que exista
codigo y prueba offline clara.

Tramo P0 ya disponible:

1. Mantener `OperationalDirectorPlanMaterializerV0` como entrada del plan
   operativo.
2. Materializar `launch_subagents` y conservar en `WorkflowTaskStore`
   `cohort_ref`, `wave_ref`, `parent_task_ref`, parent/child refs, profundidad,
   fanout y criterios.
3. Para `wait_subagents`, derivar `WaitAgentRefs` con
   `WorkflowTaskWaitAgentRefsV0`; no esperar todos los agentes vivos.
4. Usar `WorkflowTaskCandidateProviderV0` para pasar de tasks autorizadas a
   candidatos accionables.
5. Registrar `WorkflowTaskWaitStateV0` con causa visible y agentes pendientes.

Tramo P1 pendiente:

1. Usar providers existentes de delivery, review gate y rework/replan:
   `DeliveryCandidateProviderV0`, `ReviewGateCandidateProviderV0` y
   `ReviewReworkReplanCandidateProviderV0`.
2. Agrupar `review_deliveries` por ola/cohorte, no por cualquier entrega suelta
   del run.
3. Registrar `run_required_tests` como evidencia durable antes de permitir
   cierre en modo `programming`.
4. Persistir estado vivo del plan: el tramo inicial ya guarda step activo,
   ola/cohorte, tasks, agentes, pendientes y `wait_ref`; falta actualizar
   blockers, evidencias, intentos de replan y razon de cierre.
5. Cerrar solo con entrega, review aceptada, evidencias y tests requeridos si
   aplica.

Criterio de done para el corte offline:

- plan operativo listo produce tasks y outbox idempotente;
- waits se acotan por ola/cohorte/parent task;
- la espera queda registrada como `WorkflowTaskWaitStateV0`;
- una entrega invalida produce review/rework/replan causal;
- no hay cierre sin review aceptada, evidencias y pruebas requeridas;
- el run no parece colgado solo porque existan otros agentes vivos ajenos a la
  ola activa.

No empezar por OPES real ni recursion Codex real. Esos smokes vienen despues de
cerrar el ciclo durable offline.

## Corte de cierre generico causal

El siguiente corte es generico y offline. No prueba un runtime ni un dominio
concreto; prueba que el Director puede cerrar una ola por causalidad:

- entregas esperadas de la ola/cohorte activa;
- review aceptada o rework/replan causal;
- tests requeridos durables si el modo es `programming`;
- validadores/artefactos de dominio si el modo es `domain_work`;
- `CloseTask` o cierre de ola/run solo despues de evidencias completas;
- plan state reentrable y actualizado durante todo el ciclo.

`RequiredTestEvidenceV0` es el formato vigente de "test requerido evidenciado",
no una promesa textual del agente. Para que cierre una task de programacion con
tests declarados, debe existir una evidencia `passed` guardada por cada comando
de `WorkflowTaskV0.RequiredTests`, enlazada al mismo `run_ref`, `task_ref`,
`delivery_ref`, `review_request_id`, `review_result_ref` y
`accepted_review_ref` que el cierre. Si el store falta, la ref no existe, el
status es `failed` o la cadena causal no empata, el cierre se bloquea.

Lo pendiente es producir esas evidencias desde el paso operativo
`run_required_tests`: ejecutar o validar por puerto, guardar
`RequiredTestEvidenceV0`, registrar el fallo como causa de rework/replan y
mantener ese estado vivo en la reentrada del plan.

Hasta que ese tramo exista en codigo, debe figurar como pendiente verificable en
la matriz: no basta con que `WaitAgentRefs` ya filtre ACK/deliveries ni con que
un provider de review/replan pueda crear rework aislado.

## Espera por cohortes y oleadas

El director espera por la unidad de trabajo que el plan declaro, no por todos
los procesos vivos del run.

- Una cohorte es un conjunto de tasks hermanas con una barrera comun: por
  ejemplo "temas 1-5", "bloques de un tema", "reviews de calidad" o "validacion
  de una fase".
- Una oleada es una activacion concreta de una cohorte: tasks listas que se
  lanzan juntas bajo presupuesto, fanout y conflictos de write-set.
- El run puede tener agentes vivos de otras cohortes. Eso no bloquea avanzar una
  oleada ya completada si sus criterios locales estan aceptados.
- Si una oleada no puede avanzar, el estado de espera debe decir la causa:
  entregas pendientes, review pendiente, presupuesto agotado, timeout, bloqueo
  externo, falta de contexto o dependencia no cerrada.
- El avance entre oleadas requiere evidencia de la oleada anterior: artefactos,
  estado de review, pruebas requeridas y decision de cierre parcial.

Esta regla evita dos fallos: cerrar porque "alguien respondio" sin revisar la
cohorte, y parecer colgado porque existen agentes ajenos a la ola actual.

Implementacion parcial actual:

- `WaitAgentRefs` ya aparece en `modulos/orquesta-orchestration-core` y
  `modulos/orquesta-app-codex-stack` para limitar la espera a agentes concretos.
- `cohort_ref` y `wave_ref` ya existen en `WorkflowTaskV0`; cualquier extension
  debe leerlos desde workflow tasks o desde el plan materializado, no desde
  texto libre ni desde "agentes pendientes del run".
- `WorkflowTaskWaitAgentRefsV0` carga solo las tasks autorizadas por
  `run.Tasks`, descarta `run.ClosedTasks` y deduplica refs antes de esperar.
- `BuildWorkflowTaskWaitSnapshotV0` conserva tambien refs de tasks y agentes
  pendientes para explicar la espera.
- `WorkflowTaskWaitStateV0` guarda esa foto de espera con causa y puede
  persistirse via `orquesta-state-file`.

Nota de recuperacion: el evento `MicrotaskCreated` proyecta la ref de tarea en
el run, pero no sustituye al `WorkflowTaskStore` ni al estado de espera. Para
recuperar una espera por ola/cohorte tras reinicio hace falta restaurar o
rematerializar las `WorkflowTaskV0` completas, cargar el
`WorkflowTaskWaitStateV0` si existe y verificar que siguen autorizadas por
`run.Tasks`.
El servicio de aplicacion ya hace ese primer tramo en reentrada: si el step
`wait_subagents` trae `wait_ref` y hay `WaitStateStore`, usa el wait state
persistido en estado `waiting` para derivar agentes pendientes sin ampliar la
cohorte; si no existe, conserva el fallback compatible desde `PlanState`. Cuando
ese wait se consume y el plan avanza a `review_deliveries`, el wait state queda
marcado como `continued`.

## Delegacion recursiva gobernada

La recursividad es parte del plan vivo, no una libertad del agente hijo. Un
agente puede proponer subagentes cuando el objetivo sea demasiado grande, pero
el director valida antes de materializar esos hijos como workflow tasks.

Cada propuesta de hijo debe traer:

- `parent_task_ref` y, si aplica, `cohort_ref`/`wave_ref`;
- objetivo acotado y criterio de cierre revisable;
- refs de dominio opacas o write-set permitido;
- presupuesto estimado, timeout, profundidad y fanout solicitados;
- artefactos esperados y pruebas/validadores requeridos;
- razon de por que no basta con continuar en el agente actual.

El director puede aceptar, dividir, aplazar o rechazar la delegacion. Debe
rechazarla si supera profundidad/fanout, duplica trabajo existente, no declara
criterio de review, toca refs/write-set no permitidos o intenta mover decision
de producto/runtime/modelo al nucleo.

En el flujo offline `review/rework/replan` con `split_task`, el nucleo ya aplica
la primera guarda recursiva antes de aceptar followups: si una task trae
`parent_task_ref`, carga el parent desde `WorkflowTaskStore`, exige que ese
parent siga reflejado en `run.Tasks`, que `delegation_depth` sea `parent+1`, que
`wave_ref`/`cohort_ref` no cambien y que el fanout declarado en el parent mas
el propuesto no supere `max_child_agents`. Esto no ejecuta runtime ni decide
presupuesto de proveedor; esa frontera sigue en la composicion/adaptador.

Aplicacion por dominio:

- Programacion: los hijos operan sobre tareas de cambio con write-set,
  validacion de diff y pruebas requeridas. Un agente de arquitectura puede pedir
  hijos para implementacion, tests o review, pero el director conserva el plan,
  los locks de write-set y el cierre.
- OPES/dominio: los hijos operan sobre `domain_work` con refs opacas. Un agente
  de tema puede pedir hijos para bloques, visuales, revision pedagogica,
  revision legal o ensamblado parcial, pero OPES conserva taxonomia, fuentes,
  reglas, validadores y ensamblado final.

El resultado de cada hijo vuelve como entrega revisable del workflow. Ningun
subarbol se considera cerrado hasta que el director registre review y la tarea
padre incorpore esa evidencia.

## No hacer

- No lanzar agentes sin alcance, pruebas y criterio de cierre; el write-set puede
  ser amplio (`.`) cuando el trabajo requiera reparar cualquier parte del repo,
  y en ese caso sigue siendo trazabilidad, no una venda.
- No crear un bucle infinito de "observar -> pensar -> lanzar otro agente" sin
  presupuesto, timeout, limite de intentos y causa compacta.
- No aceptar planes bonitos que no esten anclados al objetivo actual.
- No inventar contexto OPES, datos de oposicion, fuentes, legislacion ni reglas
  pedagogicas. Si falta contexto, se pregunta o se bloquea.
- No meter Codex, OPES, web, MCP, DB concreta, HOME, OAuth, tokens, paths
  locales ni runtime real dentro de core/workflow/domain-work.
- No cerrar por ACK textual si faltan artefactos, diff, pruebas, review o
  validacion final requeridos.
- No usar subagentes como debate libre. Las consultas y segundas opiniones
  deben tener sujeto, evidencia, decision esperada y limite.
- No permitir spawn recursivo libre. Cada hijo necesita parent ref, objetivo,
  write-set o refs de dominio, limite de contexto, limite de presupuesto y
  criterio de review.
- No consumir decisiones de un agente director antes de registrar causalmente su
  ACK/artefacto. Primero evidencia, despues decisiones.

## Primer corte razonable

El primer corte debe ser pequeno y demostrar el ciclo operativo completo, aunque
use fakes/offline para la mayoria de piezas:

Estado del 2026-05-17: el contrato del Director Operativo, la proyeccion a olas,
el materializador inicial de `launch_subagents`, la derivacion de
`WaitAgentRefs` por metadata de tasks y el `TaskWriter` para
review/replan-split ya existen. El P0 de materializacion + espera durable ya
esta cerrado. El corte completo sigue pendiente hasta que review, replan y
cierre queden como estados durables reentrables.

1. Entrada compacta del run:
   objetivo actual, tipo de trabajo, dominio, restricciones, contexto opaco,
   write-set esperado si aplica y pruebas requeridas.
2. Plan vivo:
   lista versionada de pasos con estado `pending`, `running`, `blocked`,
   `changes_requested`, `accepted` o `closed`; dependencias y evidencia minima.
3. Scheduler de subagentes:
   solo lanza agentes para tareas desbloqueadas, con write-set y pruebas
   requeridas; respeta conflictos de read/write-set y presupuesto.
4. Delegacion recursiva gobernada:
   si el plan lo permite, un agente puede proponer subagentes para subtareas
   grandes. El director valida profundidad, fanout, presupuesto, parent/child
   refs y criterio de cierre antes de lanzarlos. El corte offline actual ya
   valida `split_task` recursivo contra la metadata persistida en
   `WorkflowTaskStore` antes de guardar nuevas microtareas.
5. Espera explicita:
   si no hay candidato accionable, el director registra por que espera:
   cohorte en progreso, agente en progreso, falta contexto, timeout pendiente,
   presupuesto agotado o bloqueo externo. El primer corte ya lo representa con
   `WorkflowTaskWaitStateV0`.
6. Review gate:
   cada entrega pasa por validacion de contrato, write-set, artefactos, pruebas
   declaradas y criterio de objetivo actual.
7. Replan controlado:
   si la entrega no sirve, se crea una tarea de rework o se modifica el plan con
   causa compacta; cada replan consume intento y deja evidencia.
8. Cierre:
   una fase/run solo se cierra cuando todas las tareas obligatorias estan
   aceptadas, las pruebas requeridas estan registradas y no quedan blockers
   abiertos.

Checklist de siguiente implementacion:

1. Mantener el materializador como fuente de metadata neutral: cada
   `WorkflowTaskV0` materializada debe conservar `wave_ref`, `cohort_ref`,
   parent/child refs, profundidad y fanout cuando existan.
2. Usar `WorkflowTaskWaitAgentRefsV0` para convertir ola/cohorte/parent task en
   `WaitAgentRefs` antes de esperar; no parsear `acceptance_criteria` para esto.
3. Mantener `WorkflowTaskWaitStateV0` actualizado cuando el wait continue,
   expire o se limpie; hoy la reentrada ya lee el estado `waiting`, el consumo
   lo marca `continued` y el agotamiento real de `MaxExternalWaits` del loop
   gestionado lo marca `expired` bloqueando el `PlanState`. Falta limpieza
   explicita.
4. Registrar entrega y review causal antes de permitir `replan_or_close`.
5. Si hay rework, crear task nueva enlazada a la entrega rechazada y consumir
   intento/presupuesto.
6. Persistir y leer estado vivo del plan; el tramo inicial existe, pero no basta
   con reconstruir un `OperationalDirectorPlanV0` inicial ni con mirar stats del
   run.
7. Usar el expander neutral de domain-work solo como materializador de jobs
   declarados por el plan; la ejecucion, DB, cola y conector pertenecen al
   adaptador de la app externa.
8. Para OPES, bloquear si faltan refs/contexto; no inventar taxonomia ni
   contenido.
9. Para Codex, no aceptar recursion si falta parent ref, child refs,
   profundidad/fanout, presupuesto y criterio de review.

Para programacion, el corte puede apoyarse en `runtime-worktree` y validacion de
diff por write-set. Para OPES, el corte debe tratar el trabajo como
`domain_work` con refs opacas y artefactos validados por OPES despues. Para
temarios completos, la forma objetivo es: director principal planifica temario,
agentes de tema trabajan en paralelo y, cuando el plan lo autoriza, cada agente
de tema delega subagentes para bloques, visuales, revisiones y ensamblado
parcial sin salir del presupuesto global.

## Criterios de exito

- Un operador puede pedir un trabajo y ver en stats/API el plan vivo, los
  agentes en curso, los bloqueos, el ultimo replan y la razon de cierre.
- El director no genera microtareas historicas si hay objetivo actual compacto.
- Ningun agente de implementacion puede entregar cambios fuera de un alcance
  declarado; si el write-set era estrecho y tuvo que ampliarlo, la entrega debe
  justificar esa ampliacion y pasar review.
- Una entrega invalida produce review/rework/replan trazable, no cierre falso.
- El run puede quedarse esperando sin parecer colgado y sin lanzar agentes
  duplicados.
- El cierre del run referencia pruebas/evidencias, no solo texto de resumen.
- Las capas neutrales siguen sin importar adaptadores de producto.
- OPES recibe artefactos por contrato y valida en su dominio; Orquesta no copia
  sus internals.

## Pruebas offline minimas

Estas pruebas deben ser deterministas y no consumir proveedor real:

- reducer/contratos: plan vivo, transiciones de tarea, cierre de fase/run y
  replay durable;
- scheduler: no lanza tareas bloqueadas, no duplica agentes y respeta conflicto
  de write-set;
- delegacion recursiva: no supera profundidad/fanout, conserva parent/child refs
  y no permite cierre sin review;
- espera por ola/cohorte: una ola completada puede avanzar aunque existan otros
  agentes vivos ajenos a esa ola; el drain acotado no ingiere ACK/delivery fuera
  de `WaitAgentRefs`;
- review/replan: entrega invalida genera `changes_requested` o rework con causa
  y no permite cierre;
- tests requeridos: una tarea de programacion no cierra sin evidencia durable de
  las pruebas declaradas;
- estado del plan: step inicial, ola/cohorte, tasks, agentes y `wait_ref` ya se
  persisten; blockers, replan attempts y razon de cierre deben sobrevivir a
  reentrada cuando se implemente el resto del ciclo;
- objetivo actual: decisiones sin anclaje al objetivo se rechazan;
- frontera: `TestNeutralOrchestrationPackagesDoNotImportProductAdapters` sigue
  verde;
- dominio externo: un fake no-OPES entra por refs opacas y no por DB/filesystem
  compartido.

Comandos focales del corte actual:

```sh
go test -count=1 ./modulos/orquesta-director-operativo ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack
go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack -run 'Test(OperationalDirectorPlanMaterializerV0|WorkflowTaskWaitAgentRefsV0|RunManagedProgressiveLoopV0|ProgressiveStatusAfterMaxBurstsV0|WorkflowTaskCandidateProviderV0|ReviewGateCandidateProviderV0|ReviewReworkReplanCandidateProviderV0|ExistingDirectorLoopRequestV0|AckAwareWaiterV0WaitAgentRefs|WaiterStartedAgentRefsForWaitV0|DrainRunV0WaitAgentRefsNoIngiereACKFueraDeScope|DrainRunHasPendingExternalAgentsV0FiltraCohorteObjetivo)'
go test -count=1 ./modulos/orquesta-app-director-service -run TestComposeStartAppDirectorProviderV0
```

## Pruebas reales y smokes

Las pruebas reales deben seguir la matriz vigente y ser opt-in:

- smoke Codex real de review/rework: Orquesta lanza director y subagentes,
  registra entrega, fuerza una entrega invalida, pide rework, acepta la
  correccion y cierra con evidencia;
- smoke de reinicio/continuidad: el plan vivo y la cola sobreviven a reinicio de
  daemon con estado temporal;
- smoke OPES `plan_tema`: OPES envia trabajo neutral, Orquesta entrega
  `document_plan`, OPES valida y crea derivados sin duplicar artefactos;
- smoke OPES temario completo futuro: director principal crea plan de temario,
  agentes por tema y subagentes por bloques/visuales/reviews con cohortes,
  parent/child refs y limites de profundidad/fanout;
- smoke shutdown cooperativo: agentes vivos reciben request de cierre,
  checkpoint y estado final trazable;
- smoke de stats: API/web muestran plan, progreso, procesos, uso si existe,
  bloqueos y cierre sin leer internals.

No ejecutar smokes reales contra instancias OPES productivas ni contra workdirs
sin aislamiento. Si falta script claro, se documenta como pendiente en la matriz
en vez de improvisar un comando.

## Guia para futuros agentes

- Si el cambio toca solo documentacion, enlaza este documento desde los indices
  y no edites Go.
- Si el cambio toca runtime, smokes, shutdown, OPES o pruebas reales, lee tambien
  `matriz_pruebas_reales_y_smoke_2026-05-17.md`.
- Si el cambio mueve frontera de nucleo, lee `estado_actual_2026-05-17.md`,
  `guia_nucleo_orquestacion_2026-05-17.md` y
  `principio_orquesta_piensa_director.md`.
- En arboles con trabajo concurrente, declara write-set antes de editar y no
  reviertas cambios ajenos.
