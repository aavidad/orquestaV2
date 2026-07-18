# V15: análisis y contrato de presupuestos y efectos

Fecha de decisión: 2026-07-18.

Estado: **candidato implementado, cableado y ejercitado; acreditación
condicionada**. V15 solo cuenta cerrado cuando
`TestAcceptanceV15BudgetsEffectsReceipt` valida un receipt V3 `PASS` nuevo en
`product/evidence/v15_budgets_effects.json`. V16 no se abre antes de ese gate.

## Decisión

V15 amplía el núcleo existente; no crea otro motor. `internal/application`
sigue siendo el único escritor, `Goal` conserva el único lifecycle y se
reutilizan el mismo `StateRepository`, la misma base SQLite, el mismo outbox y
el mismo scheduler. Presupuestos, aprobaciones y efectos son hechos causales
del estado autoritativo, no stores, colas, daemons o servicios residentes.

Capacidades propiedad exclusiva de V15:

- `GOV-15`: gobernanza de efectos, permisos, presupuesto y riesgo;
- `STG-09`: presupuesto y riesgo dentro del plan ya existente, sin segundo
  planner ni otra unidad de trabajo;
- `ORC-08`: criticidad de seguridad separada del esfuerzo del modelo;
- `ORC-09`: presupuestos globales y por Goal;
- `ORC-10`: default Codex visible de 70 padres por ciclo y fanout directo 6;
- `ORC-11`: cuotas y fairness entre proyectos y Goals;
- `EVD-03`: receipt de efecto distinto del ACK/consumo de outbox;
- `EVD-14`: todo efecto externo requiere autoridad explícita.

Sus dependencias exactas son V06 `atomic_state_outbox`, V10
`identity_projects_rbac`, V12 `director_lease` y V14 `controls`, todas ya
acreditadas. Hasta validar el receipt V15, el progreso honesto permanece en
48/257 (18,68 %), 14/34 (41,18 %) y 14/14 receipts; tras validarlo pasa a
56/257 (21,79 %), 15/34 (44,12 %) y 15/15 receipts.

## Resultado del candidato

La implementación final conserva un único writer, scheduler, outbox y
`StateRepository`. No añadió `BudgetStore`, `EffectStore`, DB, daemon, loop,
lifecycle ni endpoint paralelo. La política de presupuesto confirmada queda
congelada con el Goal y sobrevive a una rotación posterior de configuración;
un registro parcial o corrupto falla cerrado, sin fallback silencioso.

SQLite y aplicación validan la misma cadena causal, incluidos pares exactos de
fuente/permiso, `policy_hash`, target, TTL, revisión de membership, fence,
tiempos, clase de efecto y settlement. Un launch exige orden durable `reserva
<= dispatching/preparación <= attempt <= receipt`. Claims, restart, backup y recovery no
pueden fabricar aprobación, reserva, receipt o uso. Acciones V14 sin gobernanza
se aparcan sin ejecutar nuevo efecto; un terminal local puede liquidarse sin
pedir aprobación retrospectiva.

## Huecos de partida cerrados

El límite anterior `runtime.codex.max_concurrent_executions=70` vivía en memoria
del adaptador y se comprueba después del claim. No es una reserva durable ni
prueba presupuesto global/Goal/proyecto. El claim SQLite es FIFO global y
puede producir hambre multiproyecto. `AgentObservation` no transporta uso de
tokens o coste. `launch_agent` y el stop físico cruzan una frontera externa sin
los cuatro hechos separados. Los tres campos `Effect*` de
`ActionConsumptionReceipt` mezclan el consumo del outbox con la evidencia del
efecto y solo cubren stop.

La cuota temporal podía provocar requeue en algunos errores de agente, pero no
existían reserva previa, estado causal, reconciliación, fairness ni restart. El
receipt V15 cierra esos huecos mediante los contratos de este documento.

## Modelo mínimo

### Recursos y envelopes

Existe un único vector entero y tipado:

```text
ResourceVector
  tokens
  money_micros + currency
  active_time_ns
  process_slots
  disk_bytes
```

No se usa coma flotante para dinero. Sumas, restas y productos rechazan
negativos y overflow. Una `BudgetEnvelope` finita gobierna los cinco recursos;
una `BudgetDemand` pertenece al WorkItem y viaja con su plan. Los scopes son:

- despliegue/global: exposición simultánea de todos los proyectos;
- proyecto: cuota simultánea y unidad superior de fairness;
- Goal: consumo acumulado más reservas pendientes del objetivo.

Los límites global/proyecto llegan como política tipada de composición. El
envelope del Goal se confirma con el Goal y queda durable. La demanda, la
criticidad y el esfuerzo son metadata inmutable del WorkItem dentro del plan
V05; no nace otro planner ni otra entidad de ejecución.

Una reserva enlaza action, intent, Goal, WorkItem, Execution, generaciones,
spec hash, vector, policy hash y fence. `ClaimNextAction` selecciona y reserva
en la misma transacción. Cien claims concurrentes no pueden superar ninguno de
los tres envelopes. Un claim expirado reutiliza o libera de forma fenced la
reserva exacta; jamás suma otra silenciosamente.

Al terminar una Execution, `BudgetSettlement` reconcilia uso y libera el slot
de proceso. Los recursos observados se cargan con su calidad. Si Codex o un
proveedor no informa tokens o dinero, esa dimensión queda `unknown` y se carga
el máximo reservado: nunca se inventa precisión ni se contabiliza cero. Tiempo
activo y bytes de artefacto pueden medirse por aplicación. Un overrun se
registra y bloquea nueva exposición; no reescribe historia ni fabrica
capacidad.

Capacidad o cuota temporal deja la misma action pendiente con `retry_at` y
causa tipada. No cambia a terminal el Goal, WorkItem, Execution o efecto, no
consume un intento de ejecución y conserva la idempotency key.

### Fairness y paralelismo visible

Entre launches elegibles, el único claim aplica round-robin durable
`project -> Goal`, con FIFO como desempate. Actualiza un ordinal de concesión
en la misma transacción; un proyecto saturado se salta sin bloquear otro.
`stop_agent`, observaciones y mailbox no esperan presupuesto ni fairness: son
seguridad, cierre y causalidad, no nuevo trabajo.

El ciclo residente conserva un único loop. El mismo valor canónico
`runtime.codex.max_concurrent_executions`, default 70, limita explícitamente
los launches padre de la composición Codex por ciclo y la capacidad real del
adaptador. Una nueva clave canónica fija fanout directo default 6. Ambos valores
aparecen en configuración efectiva. No habrá constantes ocultas del core ni un
tope global independiente; provider, runtime u OS pueden reducir capacidad y
deben devolver evidencia temporal visible.

### Riesgo y esfuerzo

`SecurityCriticality` y `ReasoningEffort` son ejes tipados independientes. La
criticidad se declara como metadata (`normal`, `sensitive`, `critical`) y no se
infiere de palabras, logs o nombre del proveedor. El esfuerzo no concede
permisos y una criticidad alta no obliga por sí sola a un modelo caro; routing
pertenece a V25/V27.

La política mínima es:

- launch `normal`: aprobación causal separada derivada de la confirmación o
  decisión del Director ya autorizada;
- launch `sensitive`: requiere `effects.approve` además del permiso que creó o
  dirigió el plan;
- launch `critical`: requiere `effects.approve` de un principal distinto del
  proponente y rol de autoridad del proyecto;
- stop cooperativo/forzado: intent y aprobación exactos; el stop forzado exige
  `effects.approve`, pero nunca se sustituye por `Shutdown` global;
- ampliar un envelope por encima del default exige `budgets.manage`.

Se añaden solo esos dos permisos. La aprobación queda ligada a revisión de
membership, policy revision, scope y expiración. Membresía revocada, aprobación
expirada, denegada o de otro proyecto/digest produce cero llamada al adaptador.

## Cadena de efectos

Todo efecto conserva hechos distintos:

```text
EffectIntent -> EffectApproval -> EffectAttempt -> EffectReceipt
       |              |                 |                |
   qué/scope       quién/policy      invocación       prueba externa
```

- El intent es inmutable y contiene proposer, proyecto, Goal, WorkItem,
  generaciones, spec hash, kind, target/scope digest, riesgo, presupuesto y
  política de idempotencia.
- La decisión de aprobación o denegación referencia el digest exacto, principal,
  permiso, revisión de membership, policy revision, expiración y momento.
- Cada llamada real tiene un attempt durable antes de invocar el adaptador,
  con claim/fence y la misma effect idempotency key en los reintentos.
- El receipt enlaza intent, aprobación, attempt, outcome, evidence ref, tiempo
  y uso observado. No contiene PID, argv, entorno, prompt, credencial o secreto.

El ACK de `Submit`, `Control`, claim o consumo de outbox nunca cuenta como
receipt externo. `ActionConsumptionReceipt` conserva solo la referencia al
receipt cuando existe; status y tiempo no se duplican allí.

V15 gobierna los dos efectos reales que ya existen: launch de agente y stop
selectivo. Se mantienen `AgentLauncher` y `AgentController`; no se añade un
mega-`EffectExecutor`. Fake y Codex acreditan los contratos tipados. Git, push,
deploy, publicación, email y Telegram incorporarán sus puertos concretos en
V16/V29 usando esta misma cadena y el mismo outbox.

Un crash después de persistir attempt pero antes de llamar deja un intento sin
receipt. Un crash después de aplicar el efecto pero antes de persistir receipt
reinvoca solo con la misma idempotency key; el adaptador reconcilia y devuelve
el receipt original. Puede haber varios attempts fenced, pero exactamente un
efecto y un receipt terminal. Un outcome desconocido nunca se reejecuta con
otra clave ni se declara éxito.

## Persistencia y autoridad

La migración V15 es `010_budgets_effects.sql`. Extiende las tablas actuales
con envelopes, reservas/settlements, ordinales de fairness e intents,
decisiones, attempts y receipts. Todo vive en la misma SQLite y en las mismas
transacciones que snapshot/eventos/outbox. Los registros son ledger causal, no
un `BudgetStore` o `EffectStore`.

Acciones históricas anteriores al schema V15 se distinguen explícitamente en
la migración; no se inventa una aprobación retroactiva. Triggers impiden crear
una action V15 launch/stop sin intent. Backup, restore y recovery validan refs,
digests, sumas, estado de reserva, approval vigente y unicidad del receipt.

`StateRepository` puede embeber una subinterfaz cohesiva de gobernanza para no
seguir creciendo como fichero monolítico, pero sigue siendo una sola
dependencia y una sola implementación activa. No se añaden lifecycle states,
DB, goroutines, loops, colas, command registry ni dependencias externas.

## Contrato de aceptación

`AC-V15-BUDGETS-EFFECTS` debe ejecutar, no solo buscar nombres:

1. reservas concurrentes contra global/proyecto/Goal, overflow y replay;
2. fairness de dos proyectos y dos Goals, sin starvation y con un contender
   saturado que no bloquea al otro;
3. cuota temporal, restart y recuperación sin fallo terminal ni intento nuevo;
4. uso exacto, parcial y desconocido con settlement conservador;
5. combinaciones riesgo/esfuerzo persistidas sin heurísticas;
6. negativos absent/denied/expired/revoked/wrong project/target/digest/fence;
7. intent, approval, attempt, receipt y ACK demostrablemente distintos;
8. crash antes/después de la llamada y retry con un solo receipt/efecto;
9. launch y stop fake, más launch/stop Codex real por composición;
10. backup/restore, tamper, concurrencia SQLite y `-race`;
11. defaults efectivos 70/6 y ausencia de cap provider-named en core;
12. ratchets V01-V14 y arquitectura de escritor/estado/outbox únicos.

Pruebas nominales obligatorias incluyen:

```text
TestBudgetContractUsesOneCanonicalEnvelopeAcrossLayers
TestConcurrentBudgetReservationsNeverExceedEnvelope
TestTemporaryQuotaParksActionWithoutTerminalFailure
TestHierarchicalFairnessBoundsProjectAndGoalStarvation
TestEffectRequiresExactLiveApprovalBeforeAdapterInvocation
TestEffectCrashAfterApplyBeforeReceiptReconcilesOnce
TestSQLiteBudgetsEffectsRestartRaceAndReplay
TestV15RecoveryRejectsBudgetEffectCausalTampering
TestV15BackupRestorePreservesBudgetsAndEffects
```

El candidato se sella como en V14: análisis -> rojo -> preparación de receipt ->
producto P -> sello S -> ejecución exacta en checkout `detached_clean` ->
evidencia E. `candidate_subjects` debe igualar el diff base..P; output y receipt
quedan fuera de ese sujeto.

## Diferido explícito

- workspace/Git/forge: V16;
- atestación, reviews y Consejo: V17-V19;
- bindings públicos HTTP/MCP/CLI y registro: V20;
- i18n total: V21;
- autoservicio Codex completo: V22;
- Wizard/web: V23-V24;
- proveedores, tools/skills/RAG/plugins: V25-V28;
- deploy y notificaciones: V29;
- OPES: V30;
- PostgreSQL/S3/multihost: V31;
- operación, apps externas y cutover: V32-V34.

La adopción masiva o adjudicación asistida de efectos V14 aparcados, así como
la remediación operativa de una aprobación revocada después de un attempt
ambiguo, quedan en V32. V15 conserva ambos estados visibles y seguros, con cero
reinvocación automática; no crea un motor legacy lateral.

V15 expone casos de uso de aplicación y puertos neutrales, no endpoints ad hoc.

## Presupuesto de simplicidad y write-set

Ratchets netos máximos antes del sello:

| Responsabilidad | Máximo |
|---|---:|
| governance/goal/application/identity/ports | 2.200 LOC |
| adapters/config/bootstrap | 2.400 LOC |
| migración SQL | 650 LOC |
| tests y aceptación | 6.500 LOC |
| producción total | 5.250 LOC |

Máximo dos paquetes productivos nuevos, fichero productivo 400 líneas y función
80 líneas. Superar un límite exige parar, justificar la garantía que lo obliga
y retirar complejidad equivalente; no se sube el ratchet para obtener verde.

Resultado corregido del candidato: core 2.186/2.200, adaptadores 2.389/2.400,
migración 650/650, tests 6.476/6.500 y producción 5.225/5.250. No se elevó
ningún ratchet. `BUG-REBUILD-20260718-229` quedó cerrado sin rebajar producción:
`Open` público y las pruebas de crash/publicación usan
`synchronous=FULL`; un seam privado de test usa `OFF` solo para semántica de
restart, y el gate `-race` exige `-timeout=120s` más deadline total de 150 s.
`BUG-REBUILD-20260718-230` elimina el hot-loop/starvation de stop pendiente con
backoff durable exponencial y acotado, conservando intent, idempotency key,
fence, intentos y estado de control. El receipt nuevo sigue siendo requisito
separado: una prueba local verde no acredita por sí sola el corte.

Write-set V15:

```text
product/roadmap.json y tests de roadmap
acceptance/v15_budgets_effects_test.go
acceptance/fixtures/v15_budgets_effects.json
product/evidence/v15_budgets_effects.{json,output.txt}
product/traceability/** solo enlaces de lecciones V15 verificadas
internal/governance/**
internal/goal/** solo metadata de plan/snapshot
internal/identity/** solo dos permisos/política
internal/application/** solo gobernanza y extensión del escritor existente
internal/ports/agent*.go solo budget/usage/receipt neutral
internal/adapters/state/sqlite/** + migrations/010_budgets_effects.sql
internal/adapters/agent/{fake,codex}/** solo contratos V15
config/registry.json y generados
internal/bootstrap/** solo composición/scheduler/E2E
docs/reconstruccion/** de análisis, estado, ruta y handoff
```

Lecciones históricas multi-capacidad conservan `closure_evidence=not_verified`
y solo añaden los IDs V15 verificados. El bug único
`BUG-ORQ-20260706-BUDGET-CONTRACT-DESALINEADO` queda ligado a `ORC-09`, al test
nominal canónico, fixture y receipt V15; esa cobertura del rebuild no se
interpreta como verificación retroactiva del cierre legacy.
