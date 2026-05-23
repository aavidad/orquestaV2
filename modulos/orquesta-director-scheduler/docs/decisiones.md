# Decisiones locales: orquesta-director-scheduler

```text
Fecha: 2026-05-23
Decision: Los solapes reparables con trabajo vivo se secuencian en el scheduler
antes de evaluar RequestAgent.
Motivo: la autoprogramacion no debe pedir decision manual en cada solape ni
competir por el mismo alcance mientras hay un agente arrancado sobre claims
compatibles. El scheduler ya recibe `work_claims` y lifecycle compacto; puede
distinguir ejecutar ahora, esperar, colar despues, revisar o estudiar sin leer
runtime ni adaptadores.
Alternativas:
  - Bloquear todo por gate: descartado porque frena candidates independientes.
  - Lanzar y confiar en el runtime: descartado porque compite por write-set vivo.
  - Preguntar siempre al director: descartado porque reintroduce operacion
    manual y no aporta contexto nuevo.
Impacto: `work_sequence_decisions` declara la accion por candidate. Un solape
con agente vivo queda en `wait_live_work` por defecto; una dependencia sobre
claim vivo queda en `queue_after_live_work`; candidates explicitos pueden crear
microtareas de revision o estudio acotado. Trabajo independiente conserva
`execute_now`.
Estado: aceptada
```

```text
Fecha: 2026-05-12
Decision: Un progress candidate ya parado puede ceder el tick a replan explicito.
Motivo: tras no_ack, interrupcion o capacity_limited, el tick debe conservar el
orden causal. Si falta `stopped_agents`, reemite `AssessAgentWork` para que el
workflow materialice la parada. Si `stopped_agents` ya refleja esa parada, el
progress candidate queda inservible como decision nueva y no debe impedir el
followup de replan que escala capacidad/modelo o pide agente desde candidates
explicitos.
Alternativas:
  - Mantener prioridad absoluta de progress: descartado porque un candidate
    obsoleto podia dejar el run quiescent y ocultar el replan causal.
  - Lanzar work directamente tras la parada: descartado porque saltaria
    `RecordReplanDecision` y los candidates de capacidad/agente.
Impacto: progress sigue antes que replan mientras falte assessment/stop; con
`stopped_agents` observado, el tick puede emitir
`RecordReplanDecision -> RequestCapacity` sin duplicar assessment ni inventar
refs, proveedor, modelo, HOME, OAuth, DB ni runtime.
Estado: aceptada
```

```text
Fecha: 2026-05-10
Decision: El scheduler puede ordenar `CreateMicrotask` como followup de `split_task` solo si llega dentro de `ReplanFollowupCandidates`.
Motivo: el scheduler debe conservar la prioridad replan > work y no debe inventar microtareas ni refs; aun asi necesita ordenar los comandos para que el workflow autorice tareas antes de planificarlas.
Alternativas: pedir siempre al director; generar work candidates directamente; aplicar microtareas fuera del workflow.
Impacto: para `review_rework`, el tick ordena RecordReplanDecision, OpenPhase y CreateMicrotask, deduplica por `snapshot.tasks` y deja la capacidad/agente posterior a WorkCandidates normales.
Estado: aceptada
```

```text
Fecha: 2026-05-10
Decision: Los gates de revision llegan como `SchedulableReviewGateCandidateV0` y
el scheduler solo los materializa en comandos durables del workflow.
Motivo: la evaluacion de calidad pertenece al caller o a un adaptador externo;
el scheduler solo debe ordenar `RequestReview`, `RecordReviewResult`,
`AcceptReview` o `RequestRework` segun refs ya observadas. Esto cierra el corte
`revision no aceptada -> RequestRework durable` sin relanzar agentes ni crear
replan automatico dentro del tick.
Alternativas:
  - Emitir `RequestRework` desde `RecordReviewResult`: descartado porque el
    workflow mantiene comandos atomicos y deduplicables.
  - Evaluar calidad dentro del scheduler: descartado porque meteria ACKs,
    tests, ficheros, runtime o proveedor en el nucleo.
  - Convertir rework directamente en work candidates: descartado porque no hay
    API implementada para microtareas de rework en este corte.
Impacto: prioridad `outbox > lease > phase_artifact > delivery > review_gate >
progress > replan > work`; en fase `revision`, un resultado `accepted` habilita
`AcceptReview`, y un resultado `changes_requested` o `rejected` habilita
`RequestRework`. El snapshot expone `reviews`, `review_results`,
`accepted_reviews` y `rework_requests` solo como refs compactas de dedupe.
Estado: aceptada
```

```text
Fecha: 2026-05-10
Decision: Un agente en vuelo no bloquea candidatos de trabajo nuevos.
Motivo: la orquestacion paralela y los cambios a mitad de ejecucion necesitan
que el scheduler pueda lanzar una microtarea nueva aunque otra entrega siga
pendiente. Bloquear todo por `started_agent` convertia el scheduler en una cola
secuencial y hacia inutil el cambio interactivo durante programacion.
Alternativas:
  - Esperar siempre a que termine todo agente activo: descartado porque rompe
    el objetivo multiagente.
  - Forzar el cambio desde el stack: descartado porque duplicaria politica
    fuera del scheduler.
Impacto: `agent_delivery_pending` solo corta si no hay candidates de trabajo
planificables; si hay candidates, se evaluan gates/capacidad/claims y se
emiten comandos seguros. Si no se emite nada nuevo y aun hay agentes en vuelo,
el tick sigue devolviendo espera externa.
Estado: aceptada
```

```text
Fecha: 2026-05-09
Decision: El filtro anti-detalles solo inspecciona campos operativos, no refs opacas.
Motivo: un slug legitimo como `db-admin` o `model-viewer` puede aparecer en
CandidateRef, ClaimRef o AgentRef sin implicar DB/modelo hardcodeado. Mezclar
refs opacas con summaries/evidence reales crea falsos positivos y bloquea apps
validas.
Alternativas:
  - Permitir cualquier texto: descartado porque dejaria pasar secretos, DSN,
    proveedor, HOME o runtime en campos operativos.
  - Prohibir tokens en todas las refs: descartado por falsos positivos en
    nombres de app.
Impacto: refs opacas siguen validandose por longitud/presencia; summaries,
roles y reasons siguen rechazando detalles operativos prohibidos. `evidence_refs`
se tratan como refs opacas y no se inspeccionan semanticamente.
Estado: aceptada
```

```text
Fecha: 2026-05-09
Decision: Los claims de una ola paralela viajan como `work_claims` compartidos del tick.
Motivo: duplicar todos los claims dentro de cada candidato hace crecer el payload
en O(n2), rompe el limite compacto y reintroduce contextos grandes. El scheduler
necesita ver la ola completa para detectar conflictos, pero no necesita repetirla
por candidato.
Alternativas:
  - Subir el limite de payload: descartado porque oculta el problema de base.
  - Evaluar cada candidato solo con su claim: descartado porque no detecta
    conflictos entre tareas listas de la misma ola.
Impacto: `DirectorSchedulerTickInputV0.WorkClaims` es la fuente compartida para
gates de concurrencia cuando existe; `candidate.Claims` queda como claim local y
compatibilidad. Se mantiene el limite compacto y no se transportan runtime, DB,
modelo, HOME ni credenciales.
Estado: aceptada
```

```text
Fecha: 2026-05-09
Decision: El payload compacto del tick permite una cohorte inicial de hasta 4
candidatos de trabajo.
Motivo: el arranque autonomo de una app puede necesitar director, web, API y
persistencia pensando en paralelo antes de esperar entregas. El limite anterior
de 8 KiB cortaba esa cohorte aunque seguia siendo contexto pequeno.
Alternativas: lanzar los agentes por tandas de 2; eliminar el limite; mover la
cohorte fuera del scheduler.
Impacto: `maxSchedulerTickPayloadBytesV0` queda en 16 KiB. El scheduler sigue
rechazando payloads grandes, proveedores/modelos/HOME/credenciales y no lee
runtime ni DB.
Estado: aceptada
```

```text
Fecha: 2026-05-09
Decision: Un agente arrancado sin entrega observada no es quiescent.
Motivo: Tras `AgentStarted`, un run puede no tener outbox pendiente y aun asi
seguir esperando ACK, delivery o artefacto de fase. Marcarlo como quiescent
oculta agentes vivos y favorece esperas externas manuales.
Alternativas:
  - Mantener quiescent y que el caller recuerde agentes vivos: descartado porque
    rompe el contrato del loop progresivo.
  - Esperar dentro del scheduler con sleeps: descartado porque el scheduler es
    puro y sin runtime.
Impacto: si `started_agents` activos supera entregas + artefactos observados, el
tick devuelve `waiting/agent_delivery_pending`. El supervisor lo clasifica como
`wait_external`.
Estado: aceptada
```

```text
Fecha: 2026-05-09
Decision: Los artefactos de fases no-programacion llegan como `SchedulablePhaseArtifactCandidateV0`.
Motivo: brainstorming, documentacion, revision o validacion producen evidencias de fase, no entregas de microtarea de programacion. El scheduler debe poder registrar esa memoria durable sin leer ACKs, rutas, logs ni proveedores.
Alternativas:
  - Reutilizar `SchedulableDeliveryCandidateV0`: descartado porque `RegisterDelivery` exige `programacion`, task y semantica de entrega.
  - Guardar solo el ACK en el runtime: descartado porque el run no podria deduplicar ni auditar el trabajo.
Impacto: prioridad vigente `outbox > lease > phase_artifact > delivery > review_gate > progress > replan > work`; dedupe por `phase_artifacts`; exige fase actual, agente existente y started_agent; no inventa refs ni toca DB/runtime/proveedor/HOME.
Estado: aceptada
```

```text
Fecha: 2026-05-09
Decision: Las entregas llegan al scheduler como `SchedulableDeliveryCandidateV0`.
Motivo: el nucleo necesita registrar `DeliveryRegistered` cuando un agente termina, pero no debe leer ACKs, ficheros, logs, transcripts ni detalles de proveedor.
Alternativas: leer el ACK desde scheduler; registrar entregas desde el conector saltando el workflow; esperar a una fase posterior manual.
Impacto: el caller convierte receipts externos en payload compacto; el scheduler valida snapshot minimo, deduplica por `deliveries` y solo emite `RegisterDelivery` si task/agente/started_agent existen y el agente no esta failed/stopped.
Estado: aceptada
```

```text
Fecha: 2026-05-07
Decision: Un replan ya registrado no implica que la solicitud de capacidad exista.
Motivo: El tick puede cortarse entre `RecordReplanDecision` y `RequestCapacity`; si el scheduler espera `CapacityDecided` sin `capacity_request` durable, crea un bloqueo silencioso.
Alternativas: Tratar cualquier `replan_ref` como espera; repetir todo el replan; delegar la recuperacion al caller.
Impacto: El scheduler reemite `RequestCapacity` si falta la solicitud durable y solo espera cuando `capacity_requests` o `planned_capacity_requests` ya contienen la ref.
Estado: aceptada
```

```text
Fecha: 2026-05-07
Decision: `blocking_quality_gate_refs` bloquea work si no existe `ReplanFollowupCandidates` valido para ese gate.
Motivo: Un gate de calidad bloqueante no debe dejar que el tick prepare documentacion o avance de fase como si el gate hubiese pasado.
Alternativas: Parsear `quality_gates` del workflow; generar replan dentro del scheduler; delegar todo al workflow.
Impacto: El caller aporta refs opacas del gate y, si quiere avanzar, tambien aporta followup de replan explicito cuyo `source_ref` coincide; el scheduler no lee DB/runtime/proveedor/HOME.
Estado: aceptada
```

```text
Fecha: 2026-05-06
Decision: La prioridad inicial del tick queda fija como outbox > lease > progress > replan > work.
Motivo: Outbox pendiente representa efectos externos no confirmados; lease y progreso son senales de recuperacion/stop; replan debe materializar decisiones antes de abrir trabajo nuevo.
Alternativas: Procesar todo en un tick; dar prioridad a work; mezclar replan con work por orden de entrada.
Impacto: El caller puede razonar el tick sin carreras: si un nivel prioritario existe, los niveles inferiores esperan. La prioridad vigente se amplio despues a outbox > lease > phase_artifact > delivery > review_gate > progress > replan > work.
Estado: aceptada
```

```text
Fecha: 2026-05-06
Decision: El scheduler solo deduplica por refs compactas del snapshot.
Motivo: `agent_assessments`, `director_questions`, `director_answered_questions` y `replan_refs` bastan para evitar repetir comandos sin transportar payloads completos.
Alternativas: Releer eventos, parsear proyecciones ricas o consultar persistence.
Impacto: El caller debe proyectar esas refs; el tick sigue puro, sin DB, runtime, provider, HOME ni OAuth.
Estado: aceptada
```

```text
Fecha: 2026-05-06
Decision: ReplanFollowupCandidates se delega a `BuildReplanFollowupsV0`.
Motivo: El scheduler no debe decidir replan ni construir agentes de reemplazo por su cuenta; el director ya tiene el contrato de materializacion con refs explicitas.
Alternativas: Duplicar la logica en scheduler; relanzar automaticamente agentes fallidos.
Impacto: Replan queda antes de work, no reutiliza agentes bloqueados y sigue respetando `CapacityDecided` antes de `RequestAgent`.
Estado: aceptada
```

```text
Fecha: 2026-05-06
Decision: ProgressSupervisionCandidates se delega a `BuildAgentProgressSupervisionV0`.
Motivo: El scheduler no evalua progreso ni lee runtime; solo transforma candidates ya observados por adaptadores externos.
Alternativas: Interpretar logs/progreso dentro del tick; consultar runtime directamente.
Impacto: loop_detected puede pedir stop y stalled puede preguntar al director sin mezclar runtime real ni contexto ampliado.
Estado: aceptada
```

```text
Fecha: 2026-05-12
Decision: El scheduler materializa `capacity_limited` como progreso prioritario,
sin interpretar runtime.
Motivo: la deteccion de capacidad externa limitada llega ya condensada por el
adaptador y el director. El scheduler debe aplicar esos comandos antes de abrir
trabajo nuevo, pero no debe leer stderr, proveedor, modelo, HOME ni rutas.
Alternativas:
  - Decidir el relevo dentro del scheduler: descartado porque requeriria
    politica de capacidad y conocimiento de runtime.
  - Tratarlo como stalled: descartado porque el proceso ya termino y no habra
    ACK tardio.
Impacto: `ProgressSupervisionCandidates` conserva prioridad sobre replan/work;
un candidate `capacity_limited` produce assessment y parada/relevo segun el
director, sin duplicar comandos si el snapshot ya contiene la decision.
Estado: aceptada
```

```text
Fecha: 2026-05-06
Decision: El scheduler v0 vive en `orquesta-director-scheduler`, no en `orquesta-core-workflow`.
Motivo: El workflow ya quedo cerrado como motor durable determinista; el scheduler compone politicas y candidates, por tanto pertenece a la capa director.
Alternativas: Meter scheduler en workflow; ampliar orquesta-director; crear orquesta-core-scheduler.
Impacto: Se mantiene un microproyecto con contexto propio y dependencias publicas hacia workflow, director y concurrency.
Estado: aceptada
```

```text
Fecha: 2026-05-06
Decision: Una lease ya incluida en `expired_lease_refs` deja el tick `quiescent`, aunque el candidate recomiende `stop_agent`.
Motivo: La expiracion ya fue observada y el scheduler no debe producir StopAgent o AskDirector tardios sin una observacion nueva del caller.
Alternativas: Emitir solo StopAgent si el agente no esta failed/stopped; repetir RegisterAgentLeaseExpired y delegar idempotencia al workflow.
Impacto: El caller debe aportar un nuevo candidate/observacion si necesita materializar una accion secundaria omitida en un tick anterior.
Estado: aceptada
```

```text
Fecha: 2026-05-06
Decision: Un lease candidate cuyo `agent_request_id` no aparece en `agents` ni `started_agents` devuelve `needs_director/candidate_missing`.
Motivo: El candidate puede ser correcto pero el snapshot compacto no trae la observacion minima que el workflow exige para aceptar la expiracion.
Alternativas: Rechazar con error local; construir el comando y dejar que workflow falle.
Impacto: El tick no procesa work candidates mientras ese lease candidate prioritario siga incompleto.
Estado: aceptada
```

```text
Fecha: 2026-05-06
Decision: El tick no inventa refs ni payloads operativos.
Motivo: Inventar capacity_ref, agent_ref, HOME, modelo o provider dentro del scheduler reintroduce los bucles de v1/v2.
Alternativas: Autogenerar refs; retry automatico; seleccionar modelo/HOME dentro del tick.
Impacto: La planificacion/capacity debe aportar candidates explicitos; si faltan, el tick devuelve `needs_director`.
Estado: aceptada
```
