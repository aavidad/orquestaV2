# V14: análisis y contrato de controles

Fecha de decisión: 2026-07-16.

Estado: arquitectura aceptada; implementación todavía no iniciada. La
contrarrevisión independiente de dominio, persistencia/concurrencia y
alcance/trazabilidad terminó en `ACCEPT`; queda autorizado abrir el contrato
rojo `AC-V14-CONTROLS`.

## Resultado del estudio

V14 no corrige una regresión de V13. Añade una capacidad que todavía no existe:
control selectivo, durable y causal sobre trabajo vivo. La base reutilizable ya
está en V05, V06, V12 y V13: `Goal`, DAG, CAS, outbox, leases, fencing, Director,
mailbox y una única implementación de `StateRepository`.

El hueco no se resolverá copiando `run-control`, creando un daemon o añadiendo
otra máquina de estados. Los controles serán mutaciones del mismo `Goal`,
escritas por `internal/application`, persistidas junto al snapshot, eventos y
outbox en la misma transacción.

Capacidades propiedad exclusiva de V14:

- `GOV-07`: intervención humana gobernada;
- `STG-15`: rework/replan causal;
- `ORC-03`: split/replan adaptativo;
- `ORC-16`: pausa, resume y stop selectivo/forzado por identidad exacta.

V14 no acreditará capacidades posteriores por groundwork incidental.

## Vocabulario sin solapes

| Operación | Semántica canónica |
|---|---|
| `pause` | Gate reversible de despacho. Impide nuevos `launch_agent` del Goal o WorkItem objetivo. No detiene ni congela procesos vivos; `observe_agent` y mailbox continúan. |
| `resume` | Retira el gate. Hace reclamable la misma acción pendiente; no crea otra Execution ni duplica outbox. |
| `stop` | Efecto sobre una `ExecutionRef` exacta. Puede ser `cooperative` o `forced` solo si el adaptador lo anuncia. No equivale a cancelación del objetivo. |
| `retry` | Tras stop confirmado, reemplaza el intento interrumpido con una Execution nueva. La Execution detenida permanece terminal e inmutable. |
| `replan` | Decisión del Director bajo lease/fence que divide trabajo pendiente o sustituye causalmente trabajo interrumpido/fallido por uno o más sucesores y aumenta `PlanGeneration`. No reescribe historia. |
| `cancel` | Intención irreversible sobre Goal o WorkItem. Retira trabajo no lanzado, solicita stop de cada Execution viva exacta y solo termina al confirmar todos los targets. |

La idempotencia del retry de **efectos externos** pertenece a V15. V14 sí debe
cerrar el retry de **ejecución** tras stop confirmado, porque sin él `stop` dejaría
un WorkItem sin salida gobernada. Ambos contratos quedan separados.

## Autoridad y modelo mínimo

### Goal y WorkItem

- `Goal.state` conserva lifecycle: `pending`, `running`, `succeeded`, `failed` y
  el nuevo terminal `canceled`.
- Pausa no será `GoalStatePaused`. Será modo de control dentro del agregado;
  así un Goal pausado sigue aceptando observaciones y entregas de ejecuciones
  que ya estaban vivas.
- La solicitud de cancelación también vive dentro del agregado y bloquea nuevos
  launches mientras se resuelven sus targets.
- WorkItem añade `interrupted`, `canceled` y `superseded`:
  - `interrupted` es no terminal, conserva una causa durable
    (`execution_stopped` o `execution_failed`) y exige `retry` cuando proceda,
    `replan` o `cancel`;
  - `canceled` es terminal por decisión del operador;
  - `superseded` es terminal local, pero su resultado lógico depende de los
    sucesores causales del replan.
- Ciclos repetidos de pause/resume conservan una secuencia causal monotónica en
  el snapshot. Nunca se infiere la revisión contando solo el valor booleano
  final.
- Parent, dependency y `HandoffRequired` mantienen sus significados V05/V13.
  No se reutilizan para representar rework o supersesión.

### Execution y control

- Execution añade dos terminales: `canceled` para un intento retirado antes de
  cualquier lanzamiento externo y `stopped` para un proceso cuya parada exacta
  fue confirmada. Ninguno se reabre nunca.
- `stop requested` y `stop confirmed` son hechos distintos. La petición vive en
  el control/outbox; solo el receipt exacto permite escribir `stopped`.
- Un control durable contiene como mínimo:
  - principal y proyecto autenticados;
  - Goal, AppSpec generation/hash y PlanGeneration exactos;
  - WorkItem/ref/revisión cuando aplique;
  - Execution/ref/intento e identidad provider/model/agent/external cuando
    aplique;
  - tipo, modo, motivo, `request_ref`, fingerprint y tiempos;
  - estado y receipt de confirmación sin PID, argv, entorno, prompt o secreto.
- Replay idéntico devuelve el mismo control actual sin IDs, reloj, revisión,
  outbox ni efecto nuevos. Reutilizar `request_ref` con otra semántica produce
  conflicto.

No habrá `RunControl`, `runtime_orders`, DB, scheduler, store o lifecycle
paralelos. Puede existir un ledger transaccional de requests/receipts dentro del
mismo adaptador SQLite porque es evidencia e idempotencia, no autoridad de
cierre.

## Matriz de targets

| Operación | Goal | WorkItem | Execution |
|---|---:|---:|---:|
| pause/resume | sí | sí | no: una ejecución viva sigue observándose |
| cancel | sí | sí | no: cancelar Execution sin su WorkItem sería ambiguo |
| stop | no | no | sí, exacta |
| retry | no | sí, ligado a su última Execution stopped | sí como fence causal |
| replan | sí | WorkItem fuente exacto | Execution fuente cuando exista |

Las combinaciones no declaradas fallan antes de cualquier persistencia o efecto.
`PermissionGoalsDirect` se reutiliza: ya permite owner/admin/operator y deniega
contributor/reviewer/viewer. Pausa, cancelación y stop no requieren poseer el
lease del Director; replan sí.

## Transiciones

### Pause y resume

1. Application valida acceso, fingerprint, proyecto y fences.
2. Goal cambia su modo de control y revisión.
3. SQLite persiste snapshot, evento, receipt de control y auditoría en una
   transacción.
4. `ClaimNextAction` deja de reclamar `launch_agent` dentro del scope pausado,
   pero puede reclamar `observe_agent`, `deliver_mailbox` y stops pendientes.
5. Resume no recrea acciones: vuelve reclamable la ya persistida.

Una carrera pause/claim queda resuelta por Goal CAS y fencing. Si launch ya fue
aceptado, cuenta como in-flight y se observa; pausa no se convierte
silenciosamente en stop.

La frontera exacta de despacho es `RecordLaunchPrepared`:

- antes de prepararlo, una pausa que gana el CAS invalida el claim y el launch
  no puede llamar al adaptador;
- desde que el prepared queda durable, `dispatching` ya cuenta como in-flight:
  la llamada externa, su receipt de aceptación/rechazo y la observación deben
  terminar aunque la pausa llegue después;
- el gate efectivo es `goal.paused || work_item.paused`; reanudar uno de los
  dos scopes nunca levanta la pausa que siga activa en el otro.

### Stop y retry

1. Application persiste control + `stop_agent` en el outbox existente.
2. El scheduler existente reclama esa acción; no nace otro loop.
3. `AgentController` recibe identidad completa y modo soportado.
4. Solo un receipt que confirme el target exacto permite marcar Execution
   `stopped` y WorkItem `interrupted`.
5. Retry crea una ExecutionRef e idempotency key nuevas, incrementa intento y
   conserva `replaces_execution_ref`; el WorkItem vuelve a ejecución activa.

Si el proceso ya terminó, el adaptador devuelve estado exacto. `already
completed` no se falsifica como `stopped`: se consume la acción stop y se deja
o restaura exactamente una acción `observe_agent`, por clave única, hasta que
el receipt terminal quede persistido. Si ya estaba persistido, no nace otra.
Una ejecución terminal no recibe launch, stop destructivo ni transición de
estado posterior.

Retry solo es válido si la última Execution exacta está `stopped`, el Goal no es
terminal, el WorkItem sigue `interrupted`, no quedaron mensajes de destinatario
retirados por ese stop y `AttemptNo < MaxExecutionAttempts`. Superar el límite
devuelve conflicto sin efectos y exige replan o cancel; jamás se incrementa el
límite implícitamente ni se reutiliza una ref/clave anterior.

Existe una sola lease viva por WorkItem para `launch_agent`, `observe_agent` o
`stop_agent`. Un stop pendiente no roba una lease anterior. Un launch prepared
debe registrar primero su aceptación/rechazo exacto; al liberar o expirar de
forma fenced esa lease, stop tiene prioridad sobre nuevas observaciones y no
puede sufrir starvation. Stop pendiente bloquea nuevos claims de observe, pero
no invalida uno ya reclamado; completion y stop compiten mediante CAS.

### Cancel

- Pending/queued se neutraliza localmente: Execution termina `canceled` y su
  launch queda consumido con receipt sin invocar el adaptador.
- Si cancel gana antes de `RecordLaunchPrepared`, invalida el claim y el launch
  no ocurre. Si llega después del prepared, no retira ni invalida el claim de
  launch: persiste además la intención/acción stop. `RecordLaunchAccepted`
  conserva la identidad externa exacta aunque haya cambiado la revisión de
  control del Goal y, acto seguido, stop obtiene prioridad. Así no puede quedar
  un proceso lanzado sin identidad durable. Un rechazo de launch se persiste
  igualmente y resuelve el target sin inventar un stop.
- Running se conserva como in-flight y recibe stop exacto.
- Un Goal no publica `canceled` mientras quede un target vivo o pendiente de
  confirmación.
- Goal cancel y completion compiten en un único CAS: si completion cerró
  primero, cancel devuelve conflicto; si cancel ganó, el resultado final del
  Goal será `canceled` aunque alguna Execution termine entretanto. Toda esa
  evidencia terminal se conserva.
- Cancelar un Goal bloquea todo launch nuevo. Cancelar solo un WorkItem preserva
  los independientes, deriva sus dependientes como `skipped` con razón estable
  `dependency_canceled` y, cuando el grafo se resuelve, el Goal termina
  `failed`, no `canceled`.
- WorkItem cancel y su completion compiten también mediante CAS. Si la
  completion quedó durable primero, cancel devuelve conflicto. Si cancel ganó,
  el WorkItem termina `canceled` al resolverse su Execution aunque esta complete
  durante la carrera; el receipt terminal exacto de la Execution se conserva
  como evidencia, pero no transforma el WorkItem en succeeded.
- Stop requested, por sí solo, no bloquea una entrega tardía V13: delivery,
  consume y ACK exactos pueden completar hasta que stop quede confirmado.
  Cancel, como intención irreversible, sí bloquea nuevos delivery claims desde
  que gana su CAS. La transición de la Execution a `stopped`/`canceled` retira
  en la misma transacción todos sus inbox aún no resueltos. No hay readdress,
  ACK sintético, `ChildHandoffResolution` fabricado ni pérdida de receipts
  previos. Haber retirado al menos un mensaje queda como evidencia durable e
  impide retry de esa Execution: el operador debe replanificar o cancelar,
  porque V14 no mueve envelopes a un intento nuevo.

### Replan

`ProposeDirectorPlan` seguirá siendo la única entrada del Director. Se ampliará,
no se sustituirá:

- source Goal revision, PlanGeneration y WorkItem revision deben coincidir;
- las causas V14 son tipadas y durables: `split_pending`,
  `execution_stopped`, `execution_failed` o una `assessment_ref` ya persistida
  por un puerto autorizado. V14 no genera reviews ni assessments; V18 podrá
  aportar esa señal sin cambiar el contrato de replan;
- `split_pending` exige WorkItem pendiente y sin launch prepared;
  `execution_stopped` exige WorkItem `interrupted` y Execution exacta stopped;
  `execution_failed` exige WorkItem `interrupted` y el fallo/receipt causal de
  su Execution exacta. Cuando se agotan los intentos de ejecución, esa Execution
  queda terminal `failed`, pero application interrumpe el WorkItem y mantiene
  el Goal abierto en la misma transacción: no llama a `FailWorkItem` ni cierra
  el Goal antes de la decisión. Una assessment futura podrá pedir rework de un
  resultado sin que V14 conozca su proveedor;
- el plan append-only añade 1..N sucesores y registra una única relación
  canónica `rework_of`, dirigida de cada sucesor a su source. `superseded_by` es
  una proyección derivada, no una segunda relación persistida;
- el source pasa a `superseded`; artefactos, eventos, intentos y mailbox
  históricos permanecen;
- la resolución lógica es recursiva: todos los sucesores requeridos con éxito
  lógico hacen exitoso al source; cualquier sucesor `failed`, `canceled` o
  `skipped` con cualquier razón terminal lo hace fallido; un sucesor activo o
  `interrupted` lo deja sin resolver. Un source superseded anidado aplica la
  misma regla y los ciclos se rechazan;
- al agotar intentos, la misma transacción que deja el source `interrupted`
  conserva pendientes sus dependientes; no ejecuta la propagación V05 de
  `dependency_failed`. Tras el replan, esas dependencias evalúan el resultado
  lógico recursivo del source superseded y se vuelven ready solo si todos sus
  sucesores requeridos triunfan;
- si ya existe un descendiente terminal `skipped` por ese source, V14 no lo
  reabre: rechaza el replan completo. Tampoco admite un corte que introduzca
  ciclo o conflicto de write-set;
- lease/fence expirado, otra generación o principal revocado dejan cero efectos.

Al aplicar `split_pending`, la Execution queued preexistente pasa a `canceled` y
su `launch_agent` queda consumido con receipt en la misma transacción que marca
el source `superseded` y añade los sucesores. No puede sobrevivir una acción
reclamable del source ni producirse un launch parcial.

Un source que participe en una arista V13 `HandoffRequired`, como padre o como
hijo, no puede replanificarse en V14: se rechaza toda la transacción. No se
heredan, readdressan ni resuelven envelopes. Una versión futura podrá introducir
un protocolo causal específico; V14 preserva el contrato exacto ya sellado.

### Matriz de cierre

| Situación al quedar quiescent | Resultado |
|---|---|
| Goal cancel ganó CAS y todos sus targets están terminales | Goal `canceled`, incluso si algún target acabó con éxito después de la solicitud |
| Solo un WorkItem fue cancelado | ese WorkItem `canceled`, dependientes `skipped/dependency_canceled`, independientes conservados y Goal `failed` al resolverse el grafo |
| Source superseded con todos los sucesores lógicamente exitosos | source lógicamente exitoso; puede satisfacer dependencias |
| Un sucesor, incluso anidado, falla, cancela o queda `skipped` por cualquier razón | source lógicamente fallido; el Goal no puede cerrar con éxito |
| Source superseded con sucesor activo o interrupted | sin resolución; Goal no cierra |
| WorkItem `interrupted`, incluido retry agotado | sin resolución; Goal no cierra hasta replan o cancel |

Ningún estado local `superseded` equivale por sí solo a éxito y ningún
`interrupted` se cuenta como terminal de cierre.

Un WorkItem cancelado que participe en `HandoffRequired` es un bloqueo causal
terminal para el cierre exitoso, no un ACK. Permite que el Goal se estabilice en
`failed` sin fabricar `ChildHandoffResolution`, sin mover su mailbox y sin quedar
esperando para siempre una resolución imposible.

Cancelar un Goal activo habilita además el amendment causal V04: el Goal
`canceled` es fuente terminal válida de un AppSpec/Goal sucesor. Eso no mueve
evidencia ni sustituye el replan interno de WorkItems.

## Puerto neutral y adaptadores

Se añadirá una sola frontera externa consumida por application:

```text
AgentController
  ControlCapabilities(ctx)
  Stop(ctx, AgentStopRequest) -> AgentStopReceipt
```

El contrato transporta refs/generaciones/hash/intento e identidad externa
exactos, modo e idempotency key. Capabilities declaran cooperative y forced por
separado. `unsupported` nunca se convierte en `stopped` ni llama a `Shutdown`.

Fake implementa ambos modos y la suite contractual. Codex implementará stop
selectivo por ejecución y grupo de procesos, sin usar shutdown global. Su
descriptor privado debe poder demostrar ownership e identidad tras restart y
rechazar PID/PGID reutilizado; la evidencia pública permanece opaca. Si el
adaptador no puede confirmar ausencia real, devuelve pending/error y Goal no
cierra.

El descriptor privado durable enlaza como mínimo ExecutionRef, owner del
runtime, PID, PGID y marca de nacimiento del proceso. Vive en el mismo SQLite
transaccional como estado opaco del adaptador; no crea otra autoridad. En
bootstrap, una instancia nueva adopta solo descriptores cuya identidad completa
sigue coincidiendo. El E2E de crash mata el servidor sin invocar `Shutdown`,
deja vivos los hijos A/C/D y demuestra que el nuevo adaptador adopta y controla
B sin tocarlos. El cierre limpio sí conserva su contrato distinto: `Shutdown`
detiene todos los procesos propios aún vivos. Nunca se simula un crash mediante
ese shutdown cooperativo.

No se añade configuración V14 salvo necesidad demostrada por un adaptador. En
particular, forced stop no necesita un default nuevo. Si un futuro conector
requiere gracia cooperativa configurable, la clave deberá nacer primero en
`config/registry.json`.

## Persistencia y concurrencia

Migración SQLite siguiente: `009_controls.sql`.

Debe cubrir:

- modo/secuencia de control en snapshot de Goal/WorkItem;
- estados nuevos y relación causal de replan;
- ledger compacto de requests/receipts;
- `stop_agent` en el mismo outbox y action-consumption receipts;
- claim gating de launch pausado/cancelado;
- exclusión por WorkItem: una acción stop no puede robar el fence a un
  launch/observe que aún tiene lease vivo;
- prioridad durable de stop tras completar un launch prepared; stop pendiente
  impide nuevos observe claims pero respeta el observe ya leased;
- descriptor opaco de adopción Codex y verificación anti-reuso de PID/PGID;
- retiro exacto de mailbox al terminal controlado y marca que prohíbe retry si
  había inbox sin resolver;
- replay, backup, restore y validación de cada frontera;
- ningún índice, trigger o adapter decide lifecycle por su cuenta.

Carreras obligatorias: pause/claim, cancel/launch, stop/completion,
stop/retry, cancel/replan y dos controles con la misma revisión. En cada una un
solo CAS gana y el perdedor no crea efecto huérfano.

### Evolución explícita del cierre V06

V14 sustituye de forma deliberada una sola expectativa de V06: al agotar todos
los intentos/provider replacements, `internal/application/closure_test.go` ya
no debe exigir Goal `failed` inmediato. La última Execution conserva `failed`,
el número máximo de intentos, sus receipts, refs, fences e idempotencia V06; el
WorkItem queda `interrupted/execution_failed` y el Goal permanece abierto para
replan o cancel. La prueba V06 se actualizará en el mismo candidato V14 y sus
demás invariantes seguirán verdes. Esto es evolución versionada del contrato,
no reapertura de una Execution ni regresión encubierta.

## Contrato rojo obligatorio

El `AC-V14-CONTROLS` actual es insuficiente y se ampliará antes de producción.
Fixture mínimo:

1. Cuatro WorkItems disjuntos A/B/C/D, cada uno con Execution de proceso real;
   stop de la Execution B preserva procesos, estado y progreso de A/C/D antes y
   después de un crash/restart sin `Shutdown`.
2. Pause positivo de Goal y de WorkItem, y resume positivo de cada scope. Con A
   in-flight y B queued, A se observa terminal, B no lanza hasta retirar **ambos**
   gates efectivos y entonces lanza una vez. Las carreras se prueban antes y
   después de `RecordLaunchPrepared`.
3. Crash antes del stop físico y crash después del stop físico pero antes del
   receipt; ambos replays convergen sin repetir efecto.
4. Cooperative/forced según capabilities y negativo unsupported.
5. Cancel positivo de Goal y de WorkItem. Cancel pre-prepared deja la Execution
   queued en `canceled`; cancel post-prepared persiste identidad y stop; el
   control permanece pendiente hasta todos los receipts exactos. La carrera
   completion/cancel de WorkItem se prueba en ambos órdenes y un child
   `HandoffRequired` cancelado cierra el Goal como failed sin ACK ni resolución
   sintéticos.
6. Retry de WorkItem/Execution entra en el contrato canónico: crea otra
   Execution; la stopped sigue terminal. Negativos por Goal terminal, mailbox
   retirado y límite de intentos.
7. Replan 1→N bajo lease/fence cubre split pending, stopped y failed, conserva
   historia/generación y resuelve replans anidados según la matriz. Un caso con
   un único WorkItem agota intentos, queda `interrupted` sin cerrar el Goal y se
   replantea 1→N; no depende de que exista otro trabajo activo. Otro caso
   demuestra que su dependiente permanece pending y solo se vuelve ready tras
   éxito lógico de los sucesores; un descendiente ya skipped rechaza todo el
   replan sin efectos parciales.
8. Restart SQLite en cada frontera y terminales nunca reejecutados.
9. Negativos de RBAC, proyecto, Goal, revisiones, generaciones, Execution,
   intento, spec hash, principal revocado, todos los pares operación/target no
   permitidos y replan sobre cualquier extremo de `HandoffRequired`. Cada
   negativo acredita cero snapshot/evento/outbox/efecto parcial.
10. Ratchets explícitos V02, V05, V06, V07, V08, V09, V10, V12 y V13, más la
    suite completa previa. Son pruebas preventivas; no implican un fallo ya
    detectado en esas versiones. El ratchet V06 usa la evolución de cierre
    declarada arriba y conserva intentos/receipts/idempotencia.
11. Suite neutral del puerto y proceso real Codex selectivo, incluido árbol de
    procesos y ausencia de interferencia.

Mutaciones que deben romper el gate: sustituir stop por shutdown global, quitar
el filtro de pause, aceptar `unsupported` como stopped, cerrar antes del receipt,
reusar una Execution terminal o omitir cualquier fence exacto.

## Diferido explícito

- aprobación/cuotas/fairness y retry de efectos: V15;
- workspace/Git: V16;
- atestador independiente: V17;
- review/refinery/Consejo: V18–V19;
- bindings públicos y registro único HTTP/MCP/CLI: V20;
- traducción completa de superficies: V21;
- autoservicio Codex: V22;
- UI: V24;
- paridad Claude/Gemini/Ollama/Hermes y backends compartidos específicos: V25;
- mensajes genéricos/sesiones/handoff de provider: V27.

V14 expone casos de uso application reales; no añade tools MCP ad hoc antes del
registro único V20. Esto acredita el fragmento `GOV-07` de la lección histórica
“tool registrada sin puerto real”; `UI-02` y el cierre global esperan al registro
de V20.

## Presupuesto de simplicidad

- cero stores, schedulers, loops, daemons o DB adicionales;
- una interfaz externa nueva (`AgentController`);
- una migración;
- una entrada application genérica de control y la ampliación del Director;
- objetivo máximo: 2.500 LOC de producción no generada y 4.000 LOC de tests,
  fixtures y migración;
- fichero de producción preferente menor de 400 líneas y función menor de 80;
- superar el presupuesto exige parar, justificar la responsabilidad nueva y
  retirar complejidad equivalente antes de continuar.

## Lecciones históricas convertidas en invariantes

V14 acreditará únicamente el fragmento de sus capacidades en
`BUG-ORQ-20260705-197`, `BUG-ORQ-20260709-198`,
`BUG-ORQ-20260710-208`, `BUG-ORQ-20260710-208C`,
`BUG-ORQ-20260710-208S`, `BUG-ORQ-20260711-208Z`,
`BUG-ORQ-20260711-239`, `BUG-ORQ-20260711-240`,
`BUG-ORQ-20260711-243`, `BUG-ORQ-20260711-260`,
`BUG-ORQ-20260711-261`, `BUG-ORQ-20260711-267` y
`BUG-ORQ-20260711-270`. V14 conserva como ratchet el ya cerrado
`BUG-REBUILD-20260714-036`: disponer de primitivas no demuestra comportamiento
adaptativo ejecutado, pero V14 no reabre ni vuelve a cerrar ese bug.

No se copiarán soluciones legacy ni se marcará cerrado un bug multi-owner. El
test V14 se enlaza como evidencia parcial; `closure_evidence=not_verified` se
mantiene hasta que todas sus capacidades propietarias posteriores estén
acreditadas. Solo los bugs cuya totalidad pertenezca a V14 podrán cerrarse en
este vertical.

## Write-set previsto

```text
product/roadmap.json
product/evidence/v14_controls.{json,output.txt}
product/traceability/rebuild_bugs.jsonl
acceptance/v14_controls_test.go
acceptance/fixtures/v14_controls.json
internal/goal/** acotado a control, estados, snapshot y replan
internal/application/** acotado a control, processing, Director y StateRepository
internal/ports/agent.go
internal/adapters/state/sqlite/** + migrations/009_controls.sql
internal/adapters/agent/fake/**
internal/adapters/agent/codex/**
internal/bootstrap/** solo wiring/E2E
docs/reconstruccion/** de estado, mapa y handoff
```

Orden: contrato rojo → dominio/application → SQLite/fake → Codex/composición →
contrarrevisión → sellado P/C/E → receipt reproducible. No se abre V15 hasta
cerrar V14.
