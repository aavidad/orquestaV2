# Decisiones: orquesta-app-codex-stack

```text
Fecha: 2026-05-15
Decision: El paquete Codex distingue director global de worker de area.
Motivo: la prueba real de "director total" demostro que el stack podia lanzar
varios Codex en paralelo, pero seguia encerrando al director en el mismo patron
que un worker: write-set de arquitectura/plan y prompt que favorecia terminar
tras escribir decisiones. Eso producia documentos parciales y bloqueaba los
manuales exigidos por `documentar_app`.
Impacto: `directorTaskV0` da al area `director` un write-set global con
manuales, decisiones, pruebas y pendientes cuando el request_kind lo requiere;
el objetivo aclara que el director no debe limitarse a planificar si se piden
artefactos reales. Las areas `web`, `api`, `i18n`, `calidad` y persistencia
siguen con write-set cerrado de un solo documento.
Estado: aceptada.
```

```text
Fecha: 2026-05-22
Decision: La app Codex corrige entregas incompletas por review/rework antes de
fallar el smoke final.
Motivo: en prueba real el director y el agente de programacion ya funcionaban,
pero el agente omitio `web/` aunque habia generado una app Go util. Eso no debe
invalidar todo el trabajo: el director debe conservar lo valido y lanzar una
correccion acotada con el faltante.
Impacto: el smoke multiagente detecta destinos reales del `write_set` ausentes,
abre `revision`, deja que el review gate pida cambios y espera el agente de
rework. `ReviewReworkReplanSourceV0` describe los faltantes en el summary del
follow-up y `programmingObjectiveV0` los pasa al agente como contexto de
correccion, sin meter Codex ni paths locales en el nucleo. Si el agente cumple
la superficie web como paquete Go embebido `internal/webadmin`, no se exige
ademas una carpeta literal `web/`.
Estado: aceptada.
```

```text
Fecha: 2026-05-22
Decision: El stack debe preferir correccion incremental sobre rechazo completo
cuando un agente entrega una variante razonable.
Motivo: los agentes no son deterministas; en ejecuciones reales pueden usar
alias, rutas hijas, globs o entregar solo la parte corregida de un write-set
amplio. Si Orquesta exige la forma exacta en cada rail, convierte validadores en
NLU pobre y pierde trabajo valido.
Impacto: la composicion Codex normaliza decisiones del director cuando hay
causalidad suficiente, evita ampliar write-sets ya al limite, permite ACKs
parciales dentro de alcance y deja la comprobacion de completitud a
director/review/rework/tests. Seguridad, refs, archivos de control y datos
sensibles siguen siendo cortes duros.
Estado: aceptada.
```

```text
Fecha: 2026-05-22
Decision: Los lanzamientos Codex reales de Orquesta no bajan de high por
configuracion accidental.
Motivo: con cuota suficiente, `medium` en trabajos de programacion/revision del
director deja demasiado trabajo a medias. El objetivo operativo es dar manga
ancha y ajustar hacia abajo solo con evidencia, no al reves.
Impacto: la composicion del servidor, `codex-launch-wave`,
`codex-launch-director-wave` y los smokes reales elevan `low`/`medium` a
`high`; `xhigh` sigue permitido. El core no conoce modelos, proveedor ni cuotas.
Estado: aceptada.
```

```text
Fecha: 2026-05-22
Decision: La fuente compuesta normaliza errores menores de refs causales del
director antes de validar, sin relajar el contrato material.
Motivo: en prueba real el director emitio `publish_function_contract.decision_ref`
apuntando a su propia decision de publicacion, aunque el lote contenia un
`accept_decision` previo unico y valido. Bloquear por esa errata impide avanzar
por un rail demasiado estrecho.
Impacto: si `publish_function_contract.decision_ref` no referencia una decision
aceptada previa y hay un `accept_decision` previo claro, el stack lo corrige a
ese ref aceptado antes de la validacion causal. No se inventan contratos,
microtareas, write-set ni tests.
Estado: aceptada.
```

```text
Fecha: 2026-05-22
Decision: En `crear_app_completa` normal el director no debe sustituir una
tarea de programacion por una microtarea solo documental si ya hay objetivo
funcional suficiente.
Motivo: en smoke real el director recibio contexto pobre y aplico `CONSULTA AL
DIRECTOR` como rail de seguridad, creando solo documentacion. Eso bloqueaba el
run antes de lanzar el worker de programacion aunque la solicitud real pedia una
app Go pequena verificable.
Impacto: el contrato del director conserva `CONSULTA AL DIRECTOR` para falta
real de informacion, pero en `crear_app_completa` normal pide fijar supuestos
menores y crear una `create_microtask` de programacion real con `go.mod`,
entrypoint, paquete interno, README, tests y `required_tests=["go test ./..."]`
cuando la solicitud apunta a Go. No se toca core ni se introduce conocimiento
OPES.
Estado: aceptada.
```

```text
Fecha: 2026-05-22
Decision: El stack normaliza `open_phase` si el agente usa la fase destino como
`decision.phase_id`.
Motivo: en smoke real el director produjo una microtarea de programacion valida,
pero bloqueo el run porque `open_phase(planificacion_microtareas)` venia con
`decision.phase_id=planificacion_microtareas` en vez de la fase actual previa.
Era un fallo menor de formato, no un plan invalido.
Impacto: `normalizeCompositeDirectorDecisionBatchV0` infiere la fase actual por
secuencia y corrige solo ese caso (`phase_id` vacio o igual a la fase destino).
Los refs causales, contratos, `required_tests` y write-set siguen validados.
Estado: aceptada.
```

```text
Fecha: 2026-05-18
Decision: El nombre que devuelva un agente para el artefacto no es vinculante
si el job externo ya declara el contrato esperado.
Motivo: en trabajos OPES/Orquesta el director o un subagente puede cambiar
`artifact_type` o nombres de campos (`titulo`, `contenido`, `tema_id`) aunque
el contenido sea correcto. Rechazar todo el trabajo por nomenclatura recrea
fallos falsos y desperdicia ejecuciones xhigh.
Impacto: el builder de entregas desenvuelve `payload_json` aunque el
`artifact_type` del sobre no coincida, usa el tipo esperado por `work_kind` y
normaliza aliases frecuentes de campos antes de cruzar el puerto DomainWork.
La aceptacion sigue dependiendo de que el payload sea materializable para el
job: por ejemplo, `content_block` necesita tema/capitulo reales y cuerpo
usable. El core sigue neutral; esta tolerancia vive en el borde de entrega.
Estado: aceptada.
```

```text
Fecha: 2026-05-15
Decision: El stack trata `AgentLost` como cierre terminal recuperable para
drenaje, estadisticas y replanificacion.
Motivo: un agente real puede quedar arrancado en el workflow pero perder su
proceso/sesion tras reinicio o fallo del conector. Si el stack espera
indefinidamente, vuelve el bucle de v1/v2; si confirma parada sin evidencia,
falsea el estado. La salida correcta es registrar perdida y pedir replan con
fuente durable `agent_lost`.
Impacto: `drainRunHasPendingExternalAgentsV0` considera `lost_agents` como
terminal, `AssessmentReplanSource` puede crear reemplazo cuando hay stop
solicitado y perdida, `ProgressReportHandled` no deja reportes vivos para ese
agente, y las estadisticas del director muestran estado `lost` con
`needs_attention=true`.
Estado: aceptada.
```

```text
Fecha: 2026-05-14
Decision: `plan_tema`, `plan_temario` y `plan_documento` se entregan como
`document_plan` validado por contrato de dominio.
Motivo: para que OPES cree temarios solo, Orquesta debe poder pensar y devolver
un plan ejecutable antes de redactar. Si el stack lo aceptase como
`work_delivery` generico, volveriamos al fallo de v1/v2: trabajos que parecen
completos pero no se pueden validar ni encadenar.
Impacto: el builder default mapea los work kinds de planificacion a
`document_plan`, exige `DomainDocumentPlanV0`, proyecta `content_type=application/json`
y rechaza planes sin secciones o entregables. El core sigue sin conocer OPES ni
reglas editoriales.
Estado: aceptada.
```

```text
Fecha: 2026-05-13
Decision: OPES consulta progreso por `external_job_ref`; Orquesta no obliga a
OPES a interpretar la run completa.
Motivo: OPES es el director de dominio editorial y solo necesita saber como va
su job externo. Exponerle solo `run_ref` lo obligaria a conocer detalles de
tareas, agentes y deliveries internas de Orquesta.
Impacto: el stack inyecta `CodexStackExternalJobStatsSourceV0` en
`orquesta.director.stats.v0`. La fuente resuelve `job_ref -> run_ref/task_ref`
desde `AppChangeStore`, deriva `agent_ref`, consulta deliveries por
`ReceiptStore` y proyecta `status` compacto. El `RunFileStoreV0` conserva
`external_work` persistido para que esa resolucion sobreviva a reinicios.
Estado: aceptada.
```

```text
Fecha: 2026-05-14
Decision: El runner residente de puentes externos es generico; OPES es solo un adaptador opt-in.
Motivo: Orquesta debe servir a OPES, programacion, documentacion, auditorias u
otras apps futuras sin absorber su dominio. Convertir el server en un bridge de
OPES repetiria el acoplamiento de v1/v2 y dificultaria reutilizar el nucleo.
Impacto: `cmd/orquesta-server` arranca un `external_bridge_loop` neutral y OPES
solo aporta configuracion y una funcion `opes-drain-once`. El core no conoce
temarios, visuales, categorias ni politica editorial. Cualquier app externa
debe entrar por contratos/puertos equivalentes, con idempotencia y supervisores
acotados.
Estado: aceptada.
```

```text
Fecha: 2026-05-14
Decision: Los puentes externos residentes tienen ledger de entrada generico.
Motivo: un job externo puede seguir como `pending` mientras Orquesta ya lo ha
aceptado. Sin memoria local, el runner residente reenviaria el mismo trabajo
en cada tick y volveriamos al patron de bucles de v1/v2.
Impacto: `external_bridge_input_ledger` guarda `external_system`, `job_ref`,
`run_ref`, `change_ref` y estado. OPES lo usa para marcar `already_submitted`,
pero el ledger no contiene reglas de OPES y puede ser reutilizado por cualquier
adaptador futuro. El submit sigue siendo idempotente; el ledger reduce ruido y
token/coste sin tocar el core.
Estado: aceptada.
```

```text
Fecha: 2026-05-14
Decision: Orquesta es nucleo agnostico; Codex es solo un conector de agente.
Motivo: OPES confirmo que el mismo nucleo debe servir para documentar temarios,
programar apps, auditar seguridad, investigar, refactorizar o cualquier otro
trabajo externo. Si el core se disena como "orquestador de programacion" o
"orquestador de documentacion", volveremos al error de v1/v2: mezclar dominio,
proveedor, runtime y politica de producto en los mismos flujos.
Impacto: `orquesta-app-codex-stack` solo coordina runs, tareas, fases,
capacidad, agentes, observabilidad y entregas por contratos. Codex, Claude,
Gemini, agentes locales, REST/MCP, OPES o una web son adaptadores/puertos. El
contrato neutral para dominio externo es `ApplyExternalDomainWorkV0`; OPES
decide temarios, bloques, visuales y calidad pedagogica. Orquesta no sabe de
psicologia, oposiciones, Go, SQLite ni PDF salvo por campos de contrato.
Estado: aceptada.
```

```text
Fecha: 2026-05-14
Decision: La recuperacion de ACK fallido se hace por contrato neutral de
DomainWork, no por reglas OPES ni por parches de proveedor.
Motivo: agentes Codex reales generaron artefactos validos dentro del write-set
pero fallaron al escribir `agent_ack.json` en ruta de control fuera del proyecto.
El problema raiz no era OPES ni el contenido: era una entrega externa valida
sin confirmacion formal por una restriccion de runtime/sandbox.
Impacto: el stack recupera solo cuando falta ACK, existe artefacto no vacio bajo
write-set, la tarea tiene `ApplyExternalDomainWorkV0`, el builder produce una
submission valida, el ledger no la tiene registrada y hay evidencia explicita
del fallo de ACK. Si el agente ya esta parado o terminal, el tick global puede
hacer `direct submit` antes de que el filtro de control ignore la run. Los
agentes de assessment quedan excluidos. El core sigue sin conocer OPES.
Estado: aceptada.
```

```text
Fecha: 2026-05-13
Decision: El ledger de artefactos de dominio externo tiene conector de fichero
opt-in en servidor.
Motivo: el ledger en memoria evita duplicados dentro del proceso, pero tras un
reinicio Orquesta debe conservar la deduplicacion local además de la
idempotencia del dominio propietario.
Impacto: `FileDomainWorkArtifactSubmissionLedgerV0` implementa el puerto de
ledger y persiste por escritura atomica JSON. `cmd/orquesta-server` lo cablea en
`ORQUESTA_DOMAIN_DELIVERY_LEDGER_PATH` o en `StateDir` por defecto. El core no
conoce filesystem, OPES, DB ni runtime.
Estado: aceptada.
```

```text
Fecha: 2026-05-13
Decision: Las entregas de trabajos de dominio externo se devuelven desde el
stack mediante un bridge hexagonal de artefactos, no desde el core.
Motivo: OPES y cualquier otra app de dominio deben seguir siendo independientes
de Orquesta. El core solo sabe registrar entregas de agentes; la composition es
la que conoce el conector `orquesta-domain-work`, el job externo y el contrato
REST/MCP del dominio propietario. Meter el envio de artefactos en core mezclaria
workflow con integracion de dominio.
Impacto: `DrainRunV0` intenta reenviar artefactos externos ya registrados y
tambien los recien observados. El builder exige `external_work.job_ref` explicito
y convierte el ACK validado en `DomainWorkArtifactSubmissionV0`; el ledger evita
doble envio dentro del proceso y la `idempotency_key` determinista permite que
el dominio deduplique reintentos. El contexto del agente se materializa desde
`external_work.input_fields`, incluidos campos `value_json`, para que OPES pueda
pasar paquetes editoriales suficientes sin exponer internals ni compartir DB.
Estado: aceptada.
```

```text
Fecha: 2026-05-13
Decision: El builder de artefactos no copia rutas de `ack.files` a
`payload_refs`.
Motivo: `ack.files` contiene rutas relativas del worktree del agente, como
`external/opes/draft_content_block`. `domain-work` exige refs compactas sin `/`
y OPES no debe recibir rutas internas de Orquesta como contrato de dominio.
Impacto: el contenido del fichero sigue entrando en `payload_fields.body`, y la
trazabilidad queda en `evidence_refs`/`external_refs` con refs opacas.
Estado: aceptada.
```

```text
Fecha: 2026-05-12
Decision: El smoke real de app completa corta por causa cuando el proyecto
compila pero falta un ACK de programacion.
Motivo: un smoke real lanzo la cohorte inicial, consumio decisiones del
director, programo varias microtareas en paralelo y produjo una app Go que
compilaba manualmente, pero un agente HTTP/i18n no escribio ACK antes del
deadline global. Esperar al timeout ocultaba el dato causal: el write-set ya
existia y el contrato roto era la ausencia de ACK.
Impacto: el helper de drenaje de app completa usa presupuestos acotados por
pasada, considera progreso por secuencia/proyecciones/descriptores/ACKs y
detecta `project_compiles_but_ack_missing` cuando el write-set del descriptor
esta materializado, `go test ./...` pasa y la entrega no esta registrable por
falta de ACK valido. El smoke tambien exige una ola de programacion con varios
`AgentStarted` antes del primer `DeliveryRegistered`, para probar
orquestacion paralela real. No se introduce logica de dominio, DB, proveedor,
modelo ni paths productivos.
Estado: aceptada.
```

```text
Fecha: 2026-05-13
Decision: El stack cablea shutdown de servidor mediante un caso de uso externo
hexagonal.
Motivo: el cierre controlado no debe estar repartido entre CLI, gateway y
supervisor. El stack ya conoce los puertos productivos de cola, control,
supervisor y stats, por lo que es el lugar correcto para componerlos sin que el
gateway conozca internos.
Impacto: `server_shutdown_v0.go` adapta `StackV0` a
`orquesta-server-shutdown` con wrappers pequenos. La ruta
`/api/v0/server/shutdown` queda disponible para CLI, web y MCP; el proceso
servidor solo se senaliza despues de readiness positivo.
Estado: aceptada.
```

```text
Fecha: 2026-05-13
Decision: El preparador de checkpoint del stack usa protocolo cooperativo por
ficheros de control, no stdin interactivo inventado.
Motivo: Codex real en modo `exec` no garantiza un canal vivo para ordenar
checkpoint. Sin embargo, si el prompt inicial obliga a comprobar una request de
shutdown, el runtime dir si es una frontera verificable y durable.
Alternativas:
  - Registrar checkpoint solo si no hay agentes en vuelo: insuficiente para
    trabajos largos.
  - Cortar procesos y asumir continuidad: descartado porque no deja ACK.
  - Meter rutas/proveedor en core: descartado por romper hexagonal.
Impacto: cuando `forced=false`, el stack lista agentes en vuelo, escribe
`orquesta_shutdown_request.json` en el runtime de cada agente y solo llama a
`RecordRunCheckpointV0` cuando todos devuelven
`agent_shutdown_checkpoint_ack.json` valido. Si falta algo, devuelve
`pending_agent_refs` y mantiene `waiting_checkpoint`.
Estado: aceptada.
```


```text
Fecha: 2026-05-12
Decision: El stack valida las referencias causales del lote del director antes
de consumirlo.
Motivo: en un smoke real el director escribio `request_vote.vote_request_id` con
una ref y `accept_decision.vote_ref` con otra. El core hizo bien al considerar
pendiente la aceptacion, pero el stack necesitaba un diagnostico temprano y
legible para no perder tiempo esperando una cadena imposible.
Impacto: `compositeDirectorDecisionSourceV0` comprueba que `accept_decision`
referencie un `request_vote` previo, que `publish_function_contract` referencie
un `accept_decision` previo y que las microtareas referencien contratos ya
publicados en el lote o en el run. No corrige refs inventadas ni acopla modelo,
proveedor, DB o runtime; rechaza el batch para que el director regenere.
Estado: aceptada.
```

```text
Fecha: 2026-05-12
Decision: El smoke real de review/rework solo inyecta incidencias cuando la
cohorte de programacion esta estable.
Motivo: en una ejecucion real el test modifico `go.mod` tras la primera entrega
de programacion mientras otro agente seguia activo. El verificador de diff hizo
lo correcto: rechazo el ACK posterior porque el worktree contenia un cambio
fuera del write-set de ese agente. El problema no era el verificador, sino que
la prueba contaminaba trabajo concurrente.
Alternativas:
  - Relajar `CodexReceiptWorktreeVerifierV0`: descartado porque aceptaria
    interferencias reales entre agentes.
  - Danar solo ficheros no Go: descartado porque no resuelve la carrera de
    write-set.
  - Esperar ausencia de agentes pendientes y ACKs completados sin registrar:
    aceptado.
Impacto: el helper de smoke espera una entrega de programacion estable antes de
forzar `changes_requested`; asi la revision prueba rework real sin invalidar
entregas independientes.
Estado: aceptada.
```

```text
Fecha: 2026-05-12
Decision: El smoke real de review/rework comprueba reparacion aceptada, no solo
arranque de agente.
Motivo: un rework real puede escribir ACK sin corregir la evidencia que origino
`changes_requested`. Si la prueba solo mira que existe agente/ACK, da falso
verde y el director creeria que Orquesta sabe reparar cuando solo sabe
reintentar.
Alternativas:
  - Validar solo prefijo interno de agente de rework: descartado porque acopla
    el test a naming del adapter.
  - Esperar cierre completo de la app: descartado para este smoke, porque
    mezcla calidad del rework con exito de tareas independientes.
  - Derivar el rework por task/ACK original y pasar review gate real: aceptado.
Impacto: `TestNuevaAppWebCodexStackRealReviewReworkOptInV0` toma la primera
entrega de programacion registrada, fuerza una incidencia de tamano, valida
`RequestRework`/`retry_task`, localiza el rework por contrato de task y exige
que su entrega sea aceptada por el review gate con evidencia real.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: El stack exige un puerto explicito de evidencia para el review gate.
Motivo: si el review gate no lee ficheros reales, una entrega con ACK correcto
puede aceptar codigo inexistente o demasiado grande. Si lo lee por default
oculto, repetimos acoplamiento a filesystem. La composition debe recibir el
adaptador como puerto.
Impacto: `ConfigV0.ReviewGate.FileEvidence` es requerido; los tests y smokes
inyectan `CodexReviewGateProjectFileEvidenceV0`; el core sigue sin conocer
filesystem, Codex, proveedor, modelo ni DB.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: `DrainRunV0` no espera indefinidamente un agente externo cuyo proceso
ya paro sin ACK.
Motivo: en una prueba real Codex devolvio error de cuota antes de escribir
`agent_ack.json`. Si el stack solo espera al ACK, una app pequena puede quedarse
minutos sin producir nada y repetir el problema historico de loops de bugfix.
Impacto: el stack consume `AgentProgressReportV0 stopped`, deja que el director
genere assessment y parada logica, ejecuta el stopper por puerto y registra
confirmacion. El runtime concreto sigue fuera del core y la decision queda
visible para web/MCP/estadisticas.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: El smoke real multiagente estricto debe drenar hasta entregar o cerrar
todas las tareas emitidas por el director.
Motivo: una prueba que solo valida bootstrap puede ocultar que Orquesta lanza la
primera ola pero no la frontera dependiente. Para medir autonomia real hay que
seguir hasta que cada task del plan este en `delivered_tasks` o `closed_tasks`.
Impacto: el helper opt-in de smoke real reentra por `DrainRunV0`, aplica ACKs
tardios y falla si quedan tareas vivas sin entrega/cierre. Sigue siendo opt-in
por coste de cuota.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: El conector Codex recibe `reasoning_effort` explicito desde la
configuracion del stack.
Motivo: los smokes reales heredaban `model_reasoning_effort` del entorno del
operador. Una prueba pequena podia ejecutarse como `xhigh` y parecer bloqueada
aunque Orquesta hubiera lanzado bien al agente.
Impacto: el wrapper llama `codex exec -c model_reasoning_effort=...`. Los
smokes opt-in usan `ORQUESTA_CODEX_REASONING_EFFORT` con fallback `medium`.
Produccion puede seguir elevando capacidad por politica de conectores.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: El adaptador de ficheros del director normaliza campos redundantes de
decisiones externas antes de validar contra el nucleo.
Motivo: agentes reales omitieron `schema_version` por decision o mezclaron
`decision.phase_id` con la fase fuente. El nucleo debe seguir siendo estricto,
pero el borde puede convertir una salida compacta y no ambigua al DTO canonico.
Impacto: se completan schemas faltantes y `decision.phase_id` se deriva del
payload ejecutable en comandos con phase_id. `open_phase` conserva la semantica
source -> target. El resto de validaciones siguen rechazando contratos ambiguos.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: El stack completa `depends_on` hacia bootstrap Go cuando el plan lo
hace deducible sin ambiguedad.
Motivo: un director real creo bootstrap con `go.mod` y microtareas posteriores
sin dependencia explicita. Rechazarlo protege, pero bloquea por un dato que
Orquesta puede inferir de forma determinista.
Impacto: antes de validar, las microtareas Go que necesitan modulo y no cubren
`go.mod` dependen del primer bootstrap con `go.mod`. Si no hay bootstrap, la
politica sigue rechazando el plan.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: El supervisor global ejecuta pasadas acotadas, no un daemon interno.
Motivo: un daemon con sleeps o bucles abiertos volveria a consumir cuota y
tiempo sin control cuando una app no produce progreso. La unidad segura es una
pasada con `MaxTicks`, `MaxExecutions` y parada por ausencia de ejecucion.
Impacto: `orquesta-run-supervisor` queda como modulo puro; el stack expone
`RunGlobalSupervisorV0`. Quien quiera continuidad debe invocar nuevas pasadas
desde un operador externo, MCP futuro o servicio opt-in, siempre con
presupuesto.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: Una misma pasada supervisada no repite una run ya ejecutada salvo
permiso explicito.
Motivo: si la run de mayor prioridad queda en cola, un supervisor ingenuo la
ejecutaria una y otra vez y bloquearia otras apps. Mutar prioridad desde el
supervisor seria mezclar politica de cola con ejecucion.
Impacto: el supervisor pasa `ExcludeRunRefs` al coordinador tras cada ejecucion.
La exclusion es temporal, visible como skip `run_excluded` y no modifica la
cola persistente.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: El tick global multiapp es API interna de composition, no herramienta
MCP publica en esta fase.
Motivo: MCP ya puede cambiar prioridad y controlar runs. Exponer el tick ahora
duplicaria superficie antes de estabilizar el contrato y haria que el transporte
conociera detalles operativos del scheduler global.
Impacto: se crea `orquesta-run-coordinator` puro y el stack lo compone con
`RunGlobalTickV0`. REST/MCP/web seguiran pidiendo control, stats y prioridad por
contratos finos; cuando haya consumidor externo real se abrira una herramienta
opt-in para disparar ticks.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: Al arrancar un director de app, el stack encola automaticamente la
run en la cola global.
Motivo: si la cola solo se alimenta con `set_priority` manual, Orquesta no puede
autogobernar varias apps a la vez. El alta en cola pertenece a composition
porque conecta el resultado publico del arranque con el puerto global de
prioridad.
Impacto: `QueuedArrancarDirectorExecutorV0` envuelve el executor MCP existente,
usa `RunQueuePriorityWriterPortV0` inyectado y registra `queue_ref`,
`priority_score`, `app_ref` y `updated_at`. No toca DB ni scheduler interno.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: El control de una app completa vive fuera del core workflow y entra
por puertos `RunControl`.
Motivo: un humano, el director o MCP deben poder pausar, reanudar, pedir stop
o cancelar una app sin conocer procesos, DB ni estado interno del scheduler.
Impacto: `orquesta-run-control` define el contrato puro; `orquesta-run-memory`
es solo conector de memoria; `orquestacionnucleoapp` consulta el reader antes
de planificar y antes de despachar. Si no hay estado de control para una run,
el contrato tipado `RunControlStateNotFoundErrorV0` se interpreta como
`running` por defecto; otros errores siguen siendo fatales.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: La prioridad multiapp se gestiona como cola global independiente.
Motivo: Orquesta debe poder crear multiples apps simultaneas y decidir cual
recibe trabajo segun `priority_score`, aging y estado de ejecucion, sin meter
la politica global dentro del scheduler interno de cada run.
Impacto: `orquesta-run-queue` rankea candidatos por puerto; MCP expone
`orquesta.run_queue.priority.v0` con `rank` y `set_priority`; el stack lo
cablea por REST/MCP usando solo conectores inyectados.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: Las estadisticas de agentes separan progreso de uso de recursos.
Motivo: el director, la web y el humano necesitan ver modelo, capacidad, cuota
y tokens por agente, pero mezclar esos datos con `AgentProgressReportV0`
romperia el contrato de progreso compacto y podria filtrar detalles operativos.
Impacto: el nucleo expone `AgentUsageStatsProviderPortV0` y
`DirectorAgentStatsV0.Usage` y `UsageSummary`. MCP/web pueden pedir
`include_agent_usage=true`. El stack Codex publica modelo/capacidad desde
configuracion, acepta un `CodexStackAgentUsageMetricsProviderPortV0` inyectado
y marca cuota como `not_configured` cuando no hay conector real. No se exponen
HOME, OAuth, tokens de credencial ni rutas locales.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: La multitarea de apps necesita cola global por run, no cambios en el
scheduler interno de cada run.
Motivo: el scheduler actual ordena trabajo dentro de un `run_ref`; si se usa
para decidir entre apps distintas se mezclan responsabilidades y se repite el
error de v1/v2 de acoplar control global a detalles internos.
Impacto: se crea un miniproyecto separado `orquesta-run-queue` con ranking por
`priority_score`, aging/fairness y filtro de runs no ejecutables. El score solo
decide que run recibe el siguiente ciclo; dentro del run sigue mandando el
director/scheduler existente.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: Parar una app completa requiere control de run con checkpoint, no
matar procesos directamente.
Motivo: detener agentes sin checkpoint puede perder diffs, entregas parciales,
leases y decisiones del director. Ademas `StoppedAgents` significa solicitud,
no confirmacion.
Impacto: `PauseRun`, `ResumeRun`, `StopRun` y `CancelRun` quedan como contrato
pendiente de core/app service. La politica segura sera bloquear nuevo
scheduling/outbox, pedir checkpoint al director y agentes activos, confirmar
estado durable y solo despues emitir stops. Forced stop sera modo explicito y
auditado.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: La parada de runtime por supervision exige identidad de proceso y los
agentes de direccion protegidos no son parables automaticamente.
Motivo: el smoke real mostro que el director puede tardar mas que otros agentes
y que un falso positivo de progreso no debe matar el proceso que gobierna el
run. Ademas, parar solo por `process_ref` es insuficiente cuando existen varias
sesiones/procesos registrados.
Impacto: `ProcessAgentStopperV0` valida `process_ref`, `session_ref` y
`launch_ref` cuando el runtime expone snapshot; el progreso de un director en
`brainstorming_arquitectura` se degrada a `ask_director`, con `CanStop=false`
en stats, en vez de emitir `StopRuntimeAgent`. Esta proteccion aplica a la ref
explicita del director inicial; los directores especializados como
`director_web`, `director_api` o `director_persistencia` son agentes de area y
deben seguir siendo parables cuando existe proceso registrado.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: Un ACK valido registrado limpia el proceso externo como operacion de
runtime, sin marcar el agente como parado en el workflow.
Motivo: en smoke real varios Codex escribieron `agent_ack.json`, Orquesta
registro entrega y siguieron vivos durante minutos. Eso consume cuota y puede
confundirse con trabajo pendiente.
Impacto: el stack envuelve el `EventSink` con cleanup terminal para eventos
`DeliveryRegistered` y `PhaseArtifactRegistered`. La limpieza usa el stopper
seguro inyectado y no emite `AgentStopConfirmed`, no toca `StoppedAgents` y no
cambia la semantica de entrega.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: El siguiente cierre de madurez debe introducir presupuestos por
agente/tarea/ACK, no solo timeout global del run.
Motivo: la app Go/API/web parcial compilo y tenia calidad suficiente, pero el
tercer agente de HTTP/web/i18n no alcanzo ACK antes del limite global. El
problema no es la arquitectura de app generada sino la falta de contrato
temporal por microtarea.
Impacto: queda pendiente separar microtareas grandes automaticamente y exponer
en stats/MCP edad de agente, ultima actividad, tiempo sin ACK y accion tomada
por presupuesto. No se debe resolver aumentando sin limite el timeout global.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: El stack rechaza planes Go iniciales que no cubren `go.mod`,
`cmd/server` y `go test ./...` antes de lanzar la programacion.
Motivo: el smoke real multiagente `multiagent3` demostro que aceptar solo ACKs
y archivos generados no basta. Los agentes reales trabajaron bien dentro de
write-set, pero el plan no les dio permiso para crear modulo ni entrypoint, por
lo que la app no podia ser autonoma.
Alternativas: parchear la app generada; ampliar write-set desde el runtime;
relajar el smoke final; esperar al timeout para descubrirlo tarde.
Impacto: `compositeDirectorDecisionSourceV0` valida el lote de decisiones del
director. Para planes Go iniciales exige al menos una microtarea con `go.mod`,
una con `cmd/server` o `cmd/**`, y `required_tests` con `go test ./...`. Los
cambios ya respondidos por el director no se tratan como bootstrap inicial.
Estado: aceptada.
```

```text
Fecha: 2026-05-23
Decision: La guarda de bootstrap Go no se aplica a microtareas procedentes de
`app-change` o `external-work`.
Motivo: la autoprogramacion de Orquesta entra por `/external-work/run` como
cambio incremental sobre modulos existentes. Exigir `go.mod`, `cmd/server` y
`go test ./...` a cada microtarea de cambio confundia un test focal de modulo
con una app Go nueva completa y bloqueaba el director antes de lanzar agentes.
Impacto: `compositeDirectorDecisionSourceV0` sigue validando refs causales y
planes Go iniciales; para contratos `ApplyAppChangeV0` y
`ApplyExternalDomainWorkV0` conserva las pruebas declaradas del cambio y deja
que el review/rework corrija problemas reales.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: Una smoke multiagente verde no basta para aceptar una app Go completa.
Motivo: la prueba real multiagente lanzo director y agentes reales en paralelo,
registro ACKs y artefactos, pero la revision manual encontro una app Go no
autonoma: faltaba go.mod, no habia entrypoint bajo cmd/ y varios paquetes usaban
imports relativos. Eso es trabajo real, pero no producto terminado.
Impacto: las microtareas pueden transportar `required_tests`; el prompt de
programacion exige modulo Go autonomo, entrypoint cmd/server o equivalente,
imports de modulo y ausencia de imports relativos. La smoke real de app stack
ahora falla si el proyecto Go generado no tiene go.mod, entrypoint bajo cmd/,
imports limpios y `go test ./...` verde.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: La revision se conecta en `StartAppDirectorPortsV0`, no en web ni MCP.
Motivo: web/API/MCP solo expresan intenciones y consultan estado. La decision
de aceptar o pedir cambios debe salir del scheduler/director con los mismos
puertos que el resto del workflow.
Impacto: `buildDirectorPortsV0` inyecta `ReviewGateSource`; cuando el run esta
en `revision`, `ReviewGateCandidateProviderV0` crea candidatos de review sin
intervencion de la sesion principal.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: El stack Codex inyecta tambien `ProcessRegistry` y `ProgressSource`
en el tool de stats del director.
Motivo: si web/MCP o el director piden estadisticas, necesitan saber si cada
agente tiene control de parada y si hay senales de progreso, no solo contadores
del run.
Impacto: `/director-stats` puede devolver `source_status=loaded`,
`no_signal_agent_refs` y `can_stop` usando puertos hexagonales; el core sigue
sin conocer Codex, filesystem, DB, proveedor ni modelo.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: El smoke real de cambio a mitad valida Orquesta, no coordinacion manual.
Motivo: la prueba valida el producto solo si la orden entra por la frontera de
app (`/nueva-app` y `/app-change`) y Orquesta lanza director/agentes por sus
propios puertos. Intervenir desde Codex humano ocultaria fallos de scheduler,
idempotencia o drenaje.
Impacto: `TestNuevaAppWebCodexStackRealCambioMitadOptInV0` crea una app real,
espera decisiones iniciales del director, inyecta un cambio por HTTP y exige un
agente adicional con write-set `docs/change-request-midrun.md`.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: El prompt del director aplica minimos por `request_kind` y politica
de recorte por `execution_mode`.
Motivo: una peticion de documentacion no debe forzar codigo, y una app completa
no puede cerrar solo con docs salvo debug. El criterio debe estar en el encargo
del agente, no improvisado por el operador.
Impacto: `directorTaskV0` extrae los tokens neutros del summary y anade
instrucciones/done criteria para app completa, documentacion, analisis,
seguridad, deploy y debug, manteniendo write-set pequeno.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-10
Decision: El drenaje con agentes pendientes solo corta temprano si se anaden
agentes nuevos.
Motivo: una evaluacion interna de supervision puede incrementar la secuencia
del run sin cerrar ni abrir trabajo. Cortar `DrainRunV0` en ese punto impedia
registrar ACKs tardios ya disponibles y reproducia esperas falsas.
Impacto: cuando hay agentes externos pendientes, `DrainRunV0` sigue esperando y
observando si el director solo hizo progreso interno; si el director lanza una
nueva microtarea, retorna para dejar ese agente correr en paralelo.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: El contrato textual del director exige refs superiores unicas por
comando.
Motivo: un payload puede apuntar a una decision aceptada previa, pero la
decision ejecutable de nivel superior debe tener `decision_ref` y `command_ref`
propios. Reutilizar refs provoca colision de idempotencia y mezcla comandos.
Impacto: el prompt distingue `decision_ref`/`command_ref` del comando actual de
los refs de negocio que viajan dentro del payload.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Crear un modulo exterior separado para composition real con Codex.
Motivo: app-gateway productivo debe seguir siendo una raiz neutra de handlers,
y el core no debe importar runtime Codex ni conectores reales.
Impacto: el stack opt-in puede importar runtime Codex y stores externos sin
contaminar el nucleo ni la ruta productiva.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: El stack compone la fuente de decisiones de fichero con una fuente
derivada de `AppChangeRecordV0`.
Motivo: los cambios escritos desde web/MCP/API deben entrar por el mismo puerto
del director que las decisiones generadas por agentes, sin leer DB ni runtime
desde el transporte.
Impacto: `BuildStackV0` exige `AppChangeStore`; `RequestAppChange` persiste la
solicitud y solo usa el notifier para levantar la pregunta al director. El
drenaje del run puede crear microtareas de cambio sin intervencion manual.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: La solicitud de cambio se registra primero como `AskDirector`.
Motivo: el transporte web/REST/MCP no debe decidir tareas ni agentes. El usuario
expresa intencion; el director decide si pregunta, replanifica, sube capacidad o
lanza microtareas.
Impacto: `orquesta-app-codex-stack` usa `AppChangeRequestV0` y puertos del
workflow para reflejar el cambio como pregunta durable del director. La ruta
queda preparada para que el siguiente cierre consuma esa pregunta y emita
decisiones ejecutables.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: El contrato del director debe fijar IDs encadenados y vocabulario
seguro para campos que cruzan el core.
Motivo: el core no puede inferir que `vote-ref-x` significa
`vote-request-x`; aceptar decision exige el mismo ref emitido por
`request_vote`. Ademas, terminos sensibles literales en summaries provocan
rechazo antes de programacion.
Impacto: el prompt obliga a que `accept_decision.vote_ref` sea igual al
`request_vote.vote_request_id`, que contratos y microtareas referencien refs
previos exactos, y que seguridad se exprese como `datos sensibles` en campos
de decision. No se anade traduccion tolerante.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Las `evidence_refs` del director son identificadores compactos, no
rutas ni texto humano.
Motivo: en smoke real el director escribio `director_decisions.json` con DTO
correcto, pero uso refs con slash, espacios y etiquetas de documento. El puerto
las rechazo con razon: esas refs cruzan core y deben ser trazabilidad opaca.
Impacto: el contrato textual ahora prohibe espacios, slash, rutas y etiquetas
humanas en `evidence_refs`; solo acepta letras, numeros, punto, guion, guion
bajo o dos puntos. No se anade conversor ad hoc ni se relaja el core.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Validar la ruta real multiagente desde `/nueva-app`, no desde
coordinacion manual externa.
Motivo: la app debe demostrar que Orquesta crea la cohorte y gobierna los
agentes; si Codex humano coordina uno a uno, la prueba no valida el producto.
Impacto: `TestNuevaAppWebCodexStackRealMultiagentOptInV0` arranca 4 Codex
reales en paralelo por batch, espera ACKs validos y drena los artefactos de
fase. La evidencia real paso en 148.207s.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: No interpretar una firma de runtime sin cambios como bucle real.
Motivo: un agente Codex puede estar pensando decenas de segundos sin escribir
ACK ni logs. Pararlo por ausencia temporal de cambios reproduce el fallo de
v1/v2: gastar cuota arreglando falsos positivos del supervisor.
Impacto: el adaptador Codex conserva conteo de no-progreso para reportar
`stalled`, pero no genera `loop_detected` sin una senal explicita de accion
repetida. Los umbrales del stack real son conservadores.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Dividir los smokes del stack en ficheros pequenos por responsabilidad.
Motivo: aunque sean tests, un fichero grande degrada revision, depuracion y
contexto de agentes; repetir esa pauta fue uno de los problemas historicos.
Impacto: el flujo queda separado de config, fixtures, runtime fake,
peticiones y assertions. La repeticion real posterior paso en 82.09s.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: El primer smoke real del stack exterior usa un solo director desde
`/nueva-app`.
Motivo: valida la frontera completa web -> REST interno -> MCP -> director ->
runtime real con coste y tiempo acotados antes de escalar a cohorte completa.
Impacto: el test real exige ACK, documentos del write-set y
`PhaseArtifactRegistered`; la prueba multiagente real queda como siguiente
escalon, no como prerequisito del primer cierre.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: `ack_not_ready` es espera valida en el smoke, no fallo definitivo.
Motivo: el POST de `/nueva-app` puede devolver con el proceso externo vivo
mientras el ACK aun no existe; cortar en ese punto mataria agentes que siguen
trabajando correctamente.
Impacto: el test espera hasta timeout acotado y solo falla si el ACK queda
invalido, si no hay progreso antes del limite o si falta el artefacto durable.
Estado: aceptada.
```

```text
Fecha: 2026-05-12
Decision: Los helpers de drenaje real usan avance de `LastSequence` como
progreso aunque `tasks=[]` sea temporal.
Motivo: en ejecucion con director real, Orquesta puede aplicar varias decisiones
de control antes de materializar microtareas. Tratar `tasks=[]` como ausencia
de progreso reabre el problema historico de falsos bloqueos y timeouts
parcheados: el run avanzo, pero el helper miro solo una proyeccion incompleta.
Impacto: los smokes reales deben distinguir entre ausencia de tareas y ausencia
de progreso. El criterio de progreso incluye `LastSequence`, proyecciones del
run, ACKs, descriptors y procesos observables, siempre con presupuesto acotado.
No se mete runtime, proveedor, filesystem ni DB en core; el criterio vive en
helpers/adaptadores de stack.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: No definir defaults de DB, proveedor ni modelo.
Motivo: un default oculto convierte una prueba real en comportamiento
productivo implicito y repite acoplamientos historicos.
Impacto: la composition falla si el operador no inyecta puertos/configuracion
explicitos. La documentacion y el script solo validan presencia de config.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Reutilizar `StartAppDirectorPortsV0` como frontera de arranque.
Motivo: web, REST y MCP ya convergen en el servicio de director; crear un camino
paralelo para Codex duplicaria semantica y tests.
Impacto: el stack prepara puertos reales para el servicio existente y mantiene
transportes finos.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Implementar composition Go local, pero solo como stack exterior
opt-in.
Motivo: necesitamos probar web/API/MCP con agentes reales inyectables sin
convertir el app-gateway productivo ni el core en adaptadores de proveedor.
Impacto: `BuildStackV0` compone handlers, stores, runtime, dispatcher batch,
waiter de ACK y capacidad por configuracion. La ruta solo existe si el operador
inyecta todos los puertos.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: El smoke real debe ser opt-in y delegar en comandos explicitos.
Motivo: Codex real consume cuota, credenciales y procesos locales; no puede
ejecutarse por defecto en tests ni arranques ordinarios.
Impacto: `arrancar_codex.sh` exige opt-in y configuracion completa antes de
delegar en el comando indicado por el operador.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Las refs publicas del stack exterior son neutrales.
Motivo: el core debe poder validar que ningun evento, outbox o paquete de
agente filtra proveedor, modelo, HOME, credenciales, runtime ni nombre del
conector real.
Impacto: el paquete puede usar internamente el conector real, pero evidence
refs, target_module, profile_ref, connector_ref y summaries que cruzan core
usan nombres opacos tipo app-stack.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: La prueba web/API debe lanzar ejecuciones independientes con el mismo
nombre visible de app.
Motivo: reutilizar solo el slug de la app en task/agent refs ocultaba una
colision de outbox entre solicitudes distintas.
Impacto: el intake queda corregido para generar refs por spec; el stack prueba
que API y web arrancan cohortes separadas aunque el usuario repita nombre.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: El director principal debe emitir decisiones ejecutables en
`director_decisions.json` cuando el objetivo lo requiera.
Motivo: documentar arquitectura no basta para autonomia; Orquesta ya tenia el
puerto de decision y el task store, pero el agente real no recibia run_id,
brainstorm_ref ni secuencia minima de fases.
Impacto: el contrato del director incluye run_id, brainstorm_ref, schema
`director_agent_decisions_file.v0` y cadena minima hasta `programacion`. Las
areas especializadas documentan, pero no emiten decisiones de control.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Los agentes con rol `implementacion` se clasifican como area
`programacion`, no como `director`.
Motivo: el fallback a `director` mezclaba prompts, target_module y ficheros de
control del director con trabajo de programacion.
Impacto: el stack separa director y programacion sin tocar el core; el
programming agent recibe su task concreta y no contrato de decisiones.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: El contrato textual del director nombra los campos ejecutables
obligatorios y prohibe aliases genericos.
Motivo: En prueba real el director genero un JSON razonable pero no aplicable
porque uso campos como action, decision_id, payload y refs. El contrato debe
guiar al agente hacia el DTO exacto que consume el puerto de decisiones.
Impacto: el prompt del director enumera phase_id de nivel superior,
decision_ref, command_type, command_ref, summary, evidence_refs y payloads
tipados por comando, incluidos los campos internos obligatorios de open_phase,
request_vote, accept_decision, publish_function_contract y create_microtask.task.
El stack mantiene el parseo estricto y no acepta conversiones ad hoc.
Estado: aceptada.
```

```text
Fecha: 2026-05-12
Decision: `create_microtask` separa fase de decision y fase objetivo.
Motivo: en smoke real el director escribio microtareas validas en contenido,
pero puso `decision.phase_id=programacion`. El core solo debe crear microtareas
iniciales antes de abrir programacion desde `planificacion_microtareas`;
`programacion` es la fase objetivo de ejecucion dentro de
`create_microtask.task.phase_id`.
Impacto: el prompt del director ahora exige
`decision.phase_id=planificacion_microtareas` y
`create_microtask.task.phase_id=programacion` para la primera entrega. El DTO
sigue permitiendo microtareas en `programacion` para cambios en caliente; el
lector de fichero normaliza el error inequivoco de directores externos antes de
validar.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: `DrainRunV0` continua el director completo y no solo registra ACKs.
Motivo: en ejecucion real `director_decisions.json` aparece despues del primer
arranque. Registrar ACKs sin reentrar al servicio del director deja la fase en
documentacion y no lanza agentes de programacion.
Impacto: el stack usa `ContinueAppDirectorV0` para drenar runs existentes,
consume decisiones tardias y arranca microtareas. Mantiene la espera externa en
el borde del stack para no meter temporizadores en core.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: El endpoint inicial web/API/MCP arranca el run pero no consume
decisiones del director ni espera la siguiente ola de programacion.
Motivo: en prueba real el POST llego a lanzar programacion, pero quedo
bloqueado esperando ACKs de la nueva ola y devolvio 502 aunque Orquesta estaba
trabajando. La entrada de usuario debe ser corta; la continuidad pertenece al
drain/worker.
Impacto: `BuildStackV0` compone el handler inicial con puertos start-only
sin `DirectorDecisionSource` ni `ExternalWaiter`; `StackV0` conserva los
puertos completos para `DrainRunV0`. Si `DrainRunV0` lanza nuevos agentes
externos, devuelve control y deja el siguiente avance a otro drain.
Estado: aceptada.
```

```text
Fecha: 2026-05-13
Decision: El stack cablea `external_work_run` separado de `arrancar_director`.
Motivo: el smoke OPES no debe levantar un director LLM inicial para un job
externo ya descrito; hacerlo deja procesos sobrantes y consume cuota sin aportar
decision.
Impacto: `/api/v0/external-work/run` usa stores, app-change, event-sink y
run-queue inyectados. El supervisor global procesa despues la microtarea por la
cola normal.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: El stack marca `SendDirectorQuestion` como entregado mediante un
dispatcher fino.
Motivo: las preguntas no bloqueantes ya quedan persistidas en el run; dejarlas
como outbox sin dispatcher paraba el loop en `wait_unhandled_outbox` e impedia
recoger ACKs tardios.
Impacto: el dispatcher no decide ni responde la pregunta; solo ACKea la
entrega para que web/MCP/estadisticas lean la pregunta durable sin bloquear el
flujo de agentes.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: La ola de programacion se recoge en una reentrada posterior, no en
el mismo drain que la lanza.
Motivo: si `DrainRunV0` lanza agentes externos y ademas espera sus ACKs en la
misma llamada, vuelve el bloqueo largo que hizo fallar el POST inicial. Cada
reentrada debe consumir lo ya producido y, si crea nueva ola externa, devolver
control.
Impacto: `DrainRunV0` usa un control interno `StopAfterAttempt` cuando hay
agentes pendientes. El siguiente drain recoge ACKs y registra entregas.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Las refs de entrega del paquete de agente son unicas por agente.
Motivo: varios agentes de programacion compartian refs derivadas solo del area,
por ejemplo `ack-ref-app-stack-programacion`. Eso podia colapsar ACKs y
mailboxes entre microtareas.
Impacto: `DeliveryRefs` y refs de bundle usan sufijo de agente/tarea mas area.
El test `TestAgentPacketV0UsaDeliveryRefsUnicasPorAgente` fija el contrato.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Un write-set puede apuntar a fichero o directorio.
Motivo: la prueba real `real-11` fallo porque el smoke intento leer
`internal/agenda/domain` como fichero. El contrato de trabajo por modulo puede
ser directorio si el agente crea varios ficheros pequenos dentro.
Impacto: el verificador acepta directorios con contenido y sigue rechazando
write-sets vacios o ausentes.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: La guarda automatizada inicial de tamano se aplica a ficheros Go
generados con limite de 300 lineas.
Motivo: el problema historico eran ficheros de codigo enormes dificiles de
depurar. La primera guarda debe proteger codigo ejecutable sin bloquear specs
largas que requieran politica propia.
Impacto: el smoke real falla si cualquier `.go` generado supera 300 lineas.
`web/agenda/openapi.yaml` de 310 lineas queda como observacion y futura regla
si se decide limitar tambien especificaciones.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: APP-CODEX-STACK-011 cierra estadisticas y replanificacion antes de
llamar maduro al nucleo.
Motivo: Orquesta ya puede crear una app pequena con agentes reales, pero el
director aun necesita datos estructurados de progreso, tests, duracion,
write-set y no-progreso para tomar decisiones como las de supervision humana.
Impacto: la siguiente tarea no cambia el modelo de orquestacion; expone datos
por puertos hexagonales para MCP/web/director y permite rechazar o replanificar
entregas invalidas.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Cambiar una app existente usa `AppChangeRequestV0`, separado de
`AppSpecRequestV0`.
Motivo: crear una app y modificar una app viva tienen contexto, write-set,
criterios de aceptacion y riesgos distintos. Reutilizar el contrato de nueva app
empujaria a los transportes a inferir estado desde DB, filesystem o UI.
Impacto: web, REST y MCP solo validan y entregan la solicitud; el director
reconstruye contexto por puertos, pregunta si falta informacion, y replanifica
con microtareas ligadas a `change_ref`. El modulo mantiene i18n en bordes
visibles y no hardcodea DB, proveedor, modelo ni paths.
Estado: aceptada.
```

```text
Fecha: 2026-05-14
Decision: El stack drena ACKs segun microtarea durable, no segun nombre de
fase.
Motivo: documentacion, integracion y revision pueden contener trabajo de tarea
real, mientras que el brainstorming del director produce artefactos de fase.
La frontera correcta es `task_id` presente en `run.Tasks`; no una constante
`programacion` ni el mero hecho de traer `task_id` en el packet.
Impacto: `DrainRunV0` registra `DeliveryRegistered` cuando el ACK corresponde
a una microtarea durable del run y `PhaseArtifactRegistered` cuando corresponde
a trabajo de fase del director. Asi el stack conserva cleanup por ACK,
causalidad de sidecars y compatibilidad con fases no-programacion.
Estado: aceptada.
```

```text
Fecha: 2026-05-12
Decision: El stack no deja pendiente un agente real parado por capacidad
externa limitada.
Motivo: en una prueba real un agente inicial genero documentacion parcial, pero
no escribio ACK porque el runtime externo respondio que la capacidad seleccionada
no estaba disponible. La app debe distinguir esa condicion de basura, bucle o
cuota agotada y debe cerrar el agente para que el run pueda replanificar.
Impacto: `DrainRunV0` consume el reporte `capacity_limited`, registra
assessment y `stop_agent`, y evita que `drainRunHasPendingExternalAgentsV0`
mantenga una espera indefinida. La seleccion de modelo/proveedor queda fuera
del stack, detras del conector de capacidad.
Estado: aceptada.
```

```text
Fecha: 2026-05-12
Decision: El drain respeta orden causal ACK -> sidecars -> decisiones.
Motivo: en smoke real los agentes iniciales trabajaron en paralelo y el
director produjo `director_decisions.json` antes de que su ACK quedase
registrado como artefacto de fase. Consumir primero las decisiones abria
`programacion`; despues el ACK de `brainstorming_arquitectura` fallaba por
fase actual incorrecta.
Impacto: el stack no consume decisiones de un agente hasta que el conector
Codex confirma que el ACK productor ya esta proyectado. La prueba
`TestDrainRunV0RegistraACKDirectorAntesDeConsumirDecisionFile` fija que
`PhaseArtifactRegistered` del director ocurre antes del primer `PhaseOpened`
derivado de sus decisiones. No se anaden sleeps, excepciones de fase ni logica
de proveedor/modelo.
Estado: aceptada.
```

```text
Fecha: 2026-05-13
Decision: Los trabajos externos documentales son unidades de trabajo
adaptativas, no microtareas minimas obligatorias.
Motivo: OPES puede pedir bloque, subcapitulo o capitulo si aporta paquete de
dominio suficiente. Forzar microtareas por parrafo romperia continuidad y
repetiria el error de v1/v2 de arreglar sintomas sin entender el dominio.
Impacto: `programmingObjectiveV0` presenta `ApplyExternalDomainWorkV0` como
unidad de trabajo externa. Si el contexto requerido llega truncado, el packet
anade una prueba/criterio de cierre que obliga a justificar materializacion
externa o bloquear con consulta al director.
Estado: aceptada.
```

```text
Fecha: 2026-05-13
Decision: El contexto externo documental usa ventanas elasticas acotadas.
Motivo: OPES necesita mandar paquetes editoriales suficientes para temas largos,
pero Orquesta no puede volver a prompts gigantes. Un limite fijo de 1800 bytes
por campo era demasiado agresivo para `draft_content_block` y trabajos de
expansion/revision; quitar el limite recrearia problemas de v1/v2.
Impacto: el stack usa perfiles `compact`, `standard` y `large`. Los trabajos
longform (`draft_content_block`, `expand_topic_from_summary`, revisiones,
validacion y ensamblado) reciben por defecto hasta 12 KB por campo y 72 KB
totales. Si se agota la ventana, el packet queda marcado como truncado y el
ACK no puede cerrar sin justificar materializacion externa o bloquear.
Estado: aceptada.
```

```text
Fecha: 2026-05-13
Decision: `generate_visual_asset` se entrega como `visual_asset`.
Motivo: OPES tratara esquemas, vinetas, flujogramas, mapas conceptuales e
infografias como artefactos de dominio, no como bloques de texto ni ficheros
locales. El fallback `content_block` solo seria temporal y perderia semantica.
Impacto: el builder default de entregas externas mapea `generate_visual_asset`
a `visual_asset`, conserva campos editoriales de OPES y lee el fichero del ACK
como `body`. Si `format=svg`, declara `content_type=image/svg+xml`. El core no
conoce visuales ni OPES; esto vive en el bridge de dominio externo.
Estado: aceptada.
```

```text
Fecha: 2026-05-15
Decision: Las entregas `content_block` normalizan `source_refs` ricos antes de
cruzar el puerto de dominio externo.
Motivo: en prueba real OPES -> Orquesta -> Codex, un agente `gpt-5.5 xhigh`
genero un bloque valido, pero devolvio `source_refs` como objetos con metadatos
bibliograficos. OPES espera `source_refs` como lista compacta de strings y
rechazo la entrega, aunque el contenido y el ACK fueran correctos.
Impacto: el builder del bridge DomainWork convierte
`source_refs:[{source_ref:...}]` en `source_refs:["..."]` y conserva el original
en `source_ref_details`. El core sigue neutral; la adaptacion vive en el borde
de entrega externa y sirve para cualquier agente que devuelva fuentes ricas.
Estado: aceptada.
```

```text
Fecha: 2026-05-21
Decision: El director puede avanzar con ACKs parciales y recoger ACKs tardios
de la ola anterior.
Motivo: los agentes reales no terminan todos a la vez. Esperar al ultimo deja
al director parado, pero rechazar el ACK de un agente legitimo porque el director
ya abrio otra fase corta trabajo valido.
Impacto: el core proyecta la fase de arranque por agente y `DrainRunV0` puede
consumir decisiones mientras quedan agentes vivos. Si un ACK tardio declara la
fase donde ese agente fue arrancado, se registra; si declara otra fase, se
rechaza. Los defaults de espera del stack suben a 120 ciclos cortos para dar
margen de minutos sin convertir cada ciclo en una espera opaca.
Estado: aceptada.
```
