# Autoprogramacion Orquesta: cola pendiente 2026-05-23

Este documento alimenta la cola de autoprogramacion del servidor Orquesta.
Cada agente debe leer `AGENTS.md`, el `AGENTS.md` local de su modulo y este
documento antes de editar.

## Indice vivo de backlog

Este fichero sigue siendo la fuente historica y el indice vivo durante la
migracion a shards. No se borra ni se trunca historico. El planner residente
lee este indice y despues parsea los shards Markdown declarados aqui con el
mismo formato `## Txx`, conservando fichero, linea y hash por seccion.

- Backlog historico y shard canonico actual: `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`.
- Shard rail errors observados: `docs/rail_errors_observados_2026-05-23.md`.
- Shard matriz de duplicaciones/rails pendientes: `docs/duplicaciones_railes_pendientes_2026-05-24.md`.
- Mapa operativo no programable para ola complementaria 10x6:
  `docs/autoprogramacion_ola_complementaria_10x6_2026-05-25.md`.

## Indice federado de backlog local

El planner residente puede leer estas fuentes locales solo como backlog
vigente/promocionado. Cada entrada conserva owner, estado, alias local y hash en
la request generada; si la entrada local ya esta cerrada o cubierta por un Txx,
queda como evidencia y no se programa como tarea nueva. Fuentes historicas o
stale deben permanecer fuera de este indice o declararse con estado historico.

- source_path: `modulos/orquesta-autoprogramming/docs/tareas.md`; source_kind:
  module_tasks; owner: `modulos/orquesta-autoprogramming`; estado: vigente;
  aliases: APG-*; tests: `go test -count=1 ./modulos/orquesta-autoprogramming`.
- source_path: `modulos/orquesta-server/docs/tareas.md`; source_kind:
  module_tasks; owner: `modulos/orquesta-server`; estado: vigente; aliases:
  SRV-TASK-*; tests: `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.
- source_path: `modulos/orquesta-mcp/docs/tareas.md`; source_kind:
  module_tasks; owner: `modulos/orquesta-mcp`; estado: vigente; aliases:
  MCP-*; tests: `go test -count=1 ./modulos/orquesta-mcp`.

## Reglas globales

- Orquesta nucleo no importa CLI, web, MCP, Codex, OPES, DB concreta, HOME,
  tokens, OAuth ni paths locales.
- CLI, web, API, MCP, Codex y OPES son adaptadores/composiciones.
- Mantener hexagonal puro: contratos, puertos y refs opacas en nucleo.
- No borrar codigo, docs ni tests sin revisar referencias y dejar evidencia.
- Cambios pequenos, con test focal antes de ampliar.
- Si un agente devuelve nombres o refs cercanos pero no exactos, preferir
  normalizar/corregir en director/adaptador antes que tirar todo el trabajo.
- Dar manga ancha a trabajos largos: no cortar por timeouts estrechos; detectar
  bucles por falta real de progreso.
- Si se abre un rail para desbloquear ejecucion real, documentarlo aqui como
  apertura intencional y pendiente de revision futura. Por defecto se abre para
  que Orquesta funcione; despues se estrecha con evidencia, no por suposicion.
- Las aperturas de rail son deuda viva de automejora: el director puede
  convertirlas en tareas de fondo cuando detecte poca carga, ejecutarlas con
  agentes y cerrarlas solo con pruebas reales.
- Si una ejecucion real queda parada por un rail de forma, vocabulario,
  normalizacion o validacion demasiado estrecha, la politica vigente es abrir
  el rail para que Orquesta funcione, registrar la apertura aqui y dejar su
  cierre fino como automejora futura. No tirar trabajos completos por fallos
  reparables de interpretacion.
- Mensaje para automejora futura: la prioridad actual es que Orquesta complete
  trabajos reales aunque haya que abrir validaciones de forma. Cualquier rail
  abierto debe quedar documentado como deuda, con caso reproducible y test
  externo/focal. Cuando el sistema este estable o con poca carga, el director
  puede crear tareas de automejora para cerrar esas aperturas una a una, siempre
  verificando que no rompe ejecuciones reales. No reintroducir cortes estrictos
  por suposicion.
- Los cortes fuertes se reservan para seguridad real demostrada, causalidad
  rota, refs imposibles, datos sensibles efectivos o efectos externos no
  autorizados; no para listas de palabras o equivalencias que el director pueda
  normalizar o reparar.
- Los errores de rail se registran como casos acumulables en
  `docs/rail_errors_observados_2026-05-23.md`. Antes de recompilar servidor o
  probar Orquesta completa, ejecutar la matriz rapida `./scripts/test_rails_fast.sh`
  y anadir ahi cada nuevo caso observado para pasar todos de golpe.
- Las regresiones de autoprogramacion/cierre se prueban con
  `./scripts/test_autoprogramming_fast.sh`: ejecuta en paralelo los focos de
  cierre, plan-state, prepare-run, cola y web/MCP. La bateria combinada
  `./scripts/test_orquesta_fast_parallel.sh` ejecuta en paralelo rails,
  autoprogramacion y frontera hexagonal.
- Mejora futura de seguridad: antes de enviar contexto a agentes premium o
  remotos, una IA local o sanitizador local inyectado por puerto debe poder
  revisar y limpiar datos sensibles, claves, tokens, secretos, rutas privadas y
  material no publicable. El nucleo solo debe conocer refs opacas y politica
  neutral; proveedor/modelo local y transporte quedan en adaptador/composicion.
- Las instrucciones largas se compactan antes de `WorkflowTaskV0`; el detalle
  amplio debe viajar por `context_refs`/docs/artefactos, no como payload durable
  gigante. Si el core rechaza una forma reparable, el issue debe conservar el
  subcampo causal para que el director cree followup o normalizacion.
- Comunicacion compacta: usar `$caveman full` si esta disponible, o equivalente.
- ACK valido solo con archivos reales tocados y pruebas ejecutadas.

## Tareas futuras tras estabilizar la automejora

- Extrapolar el patron de autoprogramacion estable de Orquesta a las apps
  creadas por Orquesta: modo opt-in de mejora continua por app, con backlog
  propio, revision de si cada mejora merece la pena, pruebas causales y cierre
  verificable. No activar como producto hasta que Orquesta se automejore sin
  bloqueos operativos.
- Revisar legado v0/v1/v2/v3 cuando la automejora sea estable: inventariar
  tareas pendientes y codigo ya programado, rescatar solo lo compatible y util,
  y adaptarlo a la arquitectura actual sin romper hexagonal, refs opacas,
  i18n, conectores ni reglas vigentes de esta version.

## Aperturas de rail pendientes de revision futura

### R01 concurrency-gate-detalle-prohibido

Fecha: 2026-05-23.

Motivo: `RecordConcurrencyGate` estaba reutilizando la lista estricta de
reviews y bloqueaba metadata operativa razonable (`codex`, `runtime`, `git`,
`provider`, `model`, `adapter`, `filesystem`). Ese bloqueo impedia que el
director arrancara agentes en autoprogramacion residente.

Apertura aplicada: el gate permite refs opacas de adaptador/ejecucion/pruebas.
El 2026-05-23 se retiro tambien el filtro global por palabras en
`RecordConcurrencyGate` y su evento `ConcurrencyGateRecorded`, porque bloqueaba
falsos positivos como `secrets_policy` antes de que el director pudiera lanzar
agentes y reparar.
Tambien se excluyen `concurrency_gates` y `command_effects` del filtro global
por palabras en `ValidateOrchestrationRunV0`: son proyecciones/efectos
estructurados con refs opacas y hashes, y seguian bloqueando refs legitimas como
`request-ref-app-completion-loop`.

Pendiente de revision futura: cuando haya mas ejecuciones reales, estudiar si
conviene reintroducir una guarda de datos sensibles por campo o por clasificador
semantico, no por lista global de palabras. La revision debe conservar
tolerancia a refs opacas y cortar solo seguridad real, causalidad, refs
imposibles, datos sensibles efectivos o efectos externos no autorizados. Esta
revision debe entrar como tarea de automejora de baja prioridad y puede
ejecutarse en momentos de poca carga del servidor residente.

### R02 workflow-task-detalle-prohibido

Fecha: 2026-05-23.

Motivo: `WorkflowTaskV0` bloqueaba instrucciones y refs operativas por palabras
como `runtime`, `provider`, `db`, `sql`, `oauth`, `docker`, `tmux` o `token`
aunque fueran contexto opaco o presupuesto, impidiendo crear tareas reales para
autoprogramacion.

Apertura aplicada: `WorkflowTaskV0` permite refs/texto opacos de adaptador,
ejecucion o pruebas y mantiene corte fuerte para secretos o credenciales
(`secret`, `password`, `credential`, `api_key`, tokens concretos y variantes
equivalentes). La validacion estructural de refs, phase, write-set, criterios,
linaje, payload compacto y causalidad se conserva.

Pendiente de revision futura: estudiar validacion por campo y clasificacion de
refs sensibles en vez de bloqueo por palabras globales. Debe entrar como tarea
de automejora de baja prioridad cuando haya ejecuciones reales suficientes.

### R03 request-capacity-agent-detalle-prohibido

Fecha: 2026-05-23.

Motivo: `RequestCapacity` cortaba la cola de automejora aunque el backlog ya
hubiera encolado varias tareas listas. La causa eran falsos positivos por
palabras operativas (`runtime`, `provider`, `model`, `codex`, `$HOME`, OAuth)
en refs opacas, nombres de modulos o resumenes compactos. Ese rail impedia que
el director lanzara agentes para tareas reales y repetia el patron de bloquear
trabajo recuperable antes de que el director pudiera normalizarlo.

Apertura aplicada: `RequestCapacity`, `RequestAgent`, sus outbox y el contrato
del `director-agent` permiten textos operativos opacos. En el core, los rails
de palabras sensibles quedan centralizados en
`modulos/orquesta-rails/text_policy_v0.go` y se conservan
solo patrones con pinta de valor efectivo como `client_secret=`, `api_key=`,
`authorization: Bearer` o material tipo clave privada. La causalidad, fases,
refs requeridas, tamanos maximos, idempotencia y validacion de write-set no se
abren.

Pendiente de revision futura: sustituir listas de palabras por sanitizador local
o clasificador por campo inyectado por puerto. La regla futura debe conservar
tolerancia a alias y refs opacas, y solo cortar seguridad real, causalidad,
refs imposibles, datos sensibles efectivos o efectos externos no autorizados.

### R04 delivery-review-close-detalle-prohibido

Fecha: 2026-05-24.

Motivo: tras desbloquear `RequestCapacity`, Orquesta podia lanzar agentes pero
volvia a cortarse al ingerir entregas, pedir review, registrar resultados,
aceptar review o cerrar tareas/runs. La causa era la misma lista de palabras
copiada en varios validadores (`runtime`, `provider`, `codex`, `git`, `db`,
`sql`, etc.), lo que hacia que una entrega real aceptable quedara bloqueada por
vocabulario operativo.

Apertura aplicada: los validadores del ciclo normal del workflow usan un helper
comun y la politica `modulos/orquesta-rails/text_policy_v0.go`. Se mantienen
tests que demuestran que las palabras operativas pasan y que patrones sensibles
con valor siguen cortando.

Pendiente de revision futura: revisar modulos fuera de
`orquesta-core-workflow` que aun tienen listas propias de terminos prohibidos y
decidir si deben depender de un helper local equivalente, un puerto de
sanitizado o reglas especificas por frontera.

Seguimiento: `docs/duplicaciones_railes_pendientes_2026-05-24.md`.

## Cierres aplicados 2026-05-23

- Autoprogramacion residente no se da por terminada tras una pasada global con
  ejecucion real; exige una pasada posterior sin ejecuciones para cerrar la
  llamada.
- `PrepareAutoprogrammingRunV0` marca las `WorkflowTaskV0` con
  `operational_director.task_source:autoprogramming` como `context_ref` opaca.
- `orquesta-app-director-service` acepta ese marker en `context_refs` y puede
  sembrar `OperationalDirectorPlanState` desde tareas persistidas sin meter
  Codex ni producto en el nucleo.
- El bridge de autoprogramacion devuelve `operational_director_plan_ref` en el
  `Continue` cuando hay store/writer inyectados; asi el ciclo puede reentrar por
  wait, review, tests requeridos, replan/cierre. El campo es opcional en
  MCP/web para conservar compatibilidad con composiciones legacy sin plan-store.
- La cola no marca una run autoprogramming entregada como terminal si quedan
  tareas abiertas pendientes de revision/cierre formal.
- La decision source de composicion abre `revision` cuando todas las entregas
  autoprogramming estan listas y no hay agentes externos pendientes.
- El clasificador de cierre operativo del stack reconoce el marker de
  autoprogramacion en `ContextRefs`, no solo en criterios o contratos. Se evita
  un rail de forma donde el plan-state si aceptaba la senal pero el cierre la
  ignoraba.
- Prueba nueva de ciclo completo: `prepare_run` MCP con plan-state, supervisor,
  Codex fake, entrega real en proyecto temporal, review aceptada, `go test ./...`
  real via `RequiredTestRunner`, evidencia durable y cierre de run/plan-state.
- Verificado con:
  `./scripts/test_autoprogramming_fast.sh`,
  `./scripts/test_orquesta_fast_parallel.sh`,
  `go test -count=1 ./modulos/orquesta-app-codex-stack`,
  `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-web`,
  `go test -count=1 ./modulos/orquesta-app-director-service` y focales
  `TestAutoprogrammingDirectorDecisionSourceV0`,
  `TestCodexStackAutoprogrammingPrepareRunAPIV0CierraConPlanStateYTestsRealesV0`,
  `TestOperationalClosureTaskClassifierV0AceptaMarkerEnContextRefsV0`,
  `TestStackDrainQueueStatus...`,
  `TestPrepareAutoprogrammingRunV0Persiste...`,
  `TestAutoprogrammingResidentModeV0`.
- Smoke temporal verificado:
  `ORQUESTA_AUTOPROGRAMMING_SUPERVISED_SMOKE_CONFIRM=1 SMOKE_ID=post-planstate-20260523 ORQUESTA_KEEP_SMOKE_DIR=0 ./scripts/smoke_autoprogramming_supervised.sh`.
  Resultado: servidor temporal, Codex fake, un agente arrancado por supervisor
  residente, replay idempotente OK, `external-work/run` OK,
  `codex_real_executed=false`, `opes_touched=false`.
- El residente recoge automejoras ya auto-preparadas desde la cola global
  cuando no recibe `run_ref`: `self_improvement` con `auto_prepare_run=true`
  crea una run de baja prioridad y el supervisor residente la arranca como
  backlog normal. Prueba:
  `TestAutoprogrammingResidentModeV0TomaAutomejoraAutoPreparadaCuandoEstaParadoV0`.
- Un drain acotado por `WaitAgentRefs` ya no declara `quiescent` al llegar la
  ultima entrega del scope: reentra al Director para permitir review, replan o
  cierre. Prueba:
  `TestDrainRunV0ConACKCompletoReentraDirectorConWaitAgentRefsV0`.
- El supervisor del stack no marca como `done` una run activa con tareas abiertas
  aunque el loop quede `quiescent`; conserva el ciclo vivo para que el residente
  siga revisando/cerrando. Prueba:
  `TestCodexSupervisorSnapshotFromDrainV0NoCierraAutomejoraActivaConEntregaAbiertaV0`.
- Revision `request-ref-automejora-web-cli-004`: web y CLI quedan cerrados como
  clientes finos de autoprogramacion para preparar runs, listar cola/runs, ver
  detalle, supervisar y controlar. La cobertura operativa se mantiene por API
  publica (`prepare-run`, `status`, `runs/queue/priority`, `director/stats`,
  `autoprogramming/supervise` y `runs/control`), sin acceso a stores, runtime,
  Git/worktrees, HOME, proveedor ni paths locales desde web/CLI. Las refs
  `worktree-ref-orquesta-automejora-web-cli-004` y
  `trabajo-plataforma-agentes` se preservan solo como refs opacas.
- El materializador del Director Operativo no deja colgada una run si el servidor
  cambia el contrato de una microtarea ya persistida: detecta la colision contra
  `WorkflowTaskStore`, conserva la tarea antigua intacta y crea una ref versionada
  `...-contract-<hash>` para que el nuevo ciclo pueda seguir. La reparacion de
  autoprogramacion del stack Codex tampoco intenta mutar in-place tareas legacy
  inmutables; si encuentra `microtarea existente con contrato distinto`, registra
  la compatibilidad como no bloqueante y permite que el supervisor continue. Esta
  regla se aplica a futuro: los cambios de servidor o nuevas tareas no deben
  dejar trabajos anteriores colgados por contratos antiguos reparables.
- Revalidacion `task-autoprogramming-7447aef8a77d-g01`: la entrega web/CLI se
  corrige como contrato real ya implementado en los adaptadores. La evidencia de
  cierre queda acotada a `modulos/orquesta-web`, `modulos/orquesta-cli` y
  `cmd/orquesta-server`, con prueba obligatoria
  `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./cmd/orquesta-server`.
  No se declara promocion, staging ni cleanup de worktree desde web/CLI.
- El corte `request-ref-automejora-mcp-operador-004` reviso y cerro la frontera
  acotada MCP/HTTP de operador humano/automejora: `status`, `supervise`,
  `self_improvement`, `human_work.review_plan` y `operator.operations` quedan
  como adaptadores finos sobre puertos/refs opacas, con consejo de operador
  normalizado como observacion no bloqueante y conector MCP externo opt-in.
  Validacion focal:
  `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-operator-mcp-client`.
  Frontera pendiente: servidor MCP real, red productiva y operadores concretos
  siguen fuera del nucleo y deben entrar por composicion opt-in.
- Rework `task-ref-self-improvement-18eca7b1eae7`: se confirma la frontera del
  conector MCP/HTTP sin ampliar write-set. `status` y `supervise` no leen stores
  ni runtime; `self_improvement` conserva evidencia y prepara run solo con
  opt-in; `human_work.review_plan` conserva plan si falta operador; el cliente
  MCP externo preserva errores publicos y reduce fallos opacos a
  `operator_mcp_port_error`. Refs opacas preservadas:
  `worktree-ref-orquesta-automejora-mcp-operador-004` y
  `trabajo-plataforma-agentes`.
- Revalidacion `task-autoprogramming-18eca7b1eae7-g01`: contrato MCP/HTTP de
  operador humano/automejora revisado contra docs locales y bateria focal
  obligatoria. No se abre deuda nueva; servidor MCP real, red productiva y
  operadores concretos siguen como composicion opt-in fuera del nucleo.
- Rework focal `agent-ref-task-autoprogramming-18eca7b1eae7-g01-562352fb086c`:
  se anade cobertura para el patron revisado. `operator_advice` queda probado
  como observacion no bloqueante en HTTP `status` y `supervise`, incluido error
  por puerto ausente, y en `self_improvement` como contexto/regla compacta de
  automejora. Validacion focal:
  `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-operator-mcp-client`.
- Revalidacion `agent-ref-task-autoprogramming-18eca7b1eae7-g01-dd735cd48a5e`:
  se revisa de nuevo la frontera MCP/HTTP de operador humano/automejora sin
  cambios de codigo ni apertura nueva de rail. `status`, `supervise`,
  `self_improvement`, `human_work.review_plan`, `operator.operations` y
  `orquesta-operator-mcp-client` conservan puertos inyectados, refs opacas,
  errores publicos y frontera opt-in para transporte real/operadores concretos.
  Validacion focal:
  `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-operator-mcp-client`.
- Revalidacion `agent-ref-task-autoprogramming-18eca7b1eae7-g01-a84f653220c2`:
  se cierra el rework de la frontera MCP/HTTP de operador humano/automejora como
  contrato implementado dentro del write-set. La revision confirma `status` y
  `supervise` por puertos inyectados, `operator_advice` no bloqueante,
  `self_improvement` de baja prioridad con `auto_prepare_run` opt-in,
  `human_work.review_plan` con consulta humana opt-in, `operator.operations`
  sobre refs opacas y cliente MCP externo sin acoplar `orquesta-mcp`. No se
  abre deuda nueva: servidor MCP real, red productiva y operadores concretos
  siguen como composicion opt-in fuera del nucleo. Refs opacas preservadas:
  `task-ref-self-improvement-18eca7b1eae7`,
  `worktree-ref-orquesta-automejora-mcp-operador-004` y
  `trabajo-plataforma-agentes`. Validacion focal:
  `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-operator-mcp-client`.
- Revalidacion `agent-ref-task-autoprogramming-18eca7b1eae7-g01-ca965c319ed5`:
  se repite la revision tras entrega no aceptada sin ampliar write-set. El
  contrato MCP/HTTP de operador humano/automejora queda documentado como
  frontera ya implementada: `status`, `supervise`, `self_improvement`,
  `human_work.review_plan`, `operator.operations` y
  `orquesta-operator-mcp-client` siguen por puertos inyectados, refs opacas,
  consejo de operador no bloqueante y errores publicos. No se abre servidor MCP
  real, red productiva, runtime concreto ni operador especifico desde estos
  modulos. Refs opacas preservadas: `task-ref-self-improvement-18eca7b1eae7`,
  `worktree-ref-orquesta-automejora-mcp-operador-004` y
  `trabajo-plataforma-agentes`. Validacion focal:
  `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-operator-mcp-client`.
- Rework `task-ref-self-improvement-1089cc70db2b`: el servidor residente puede
  proponer automejora idle por puerto de composicion. Por defecto espera 60
  segundos con ticks `no_execution`; `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0`
  desactiva el rail. La propuesta entra por `self_improvement` con
  `auto_prepare_run=true` y no bloquea ticks posteriores del supervisor.
  Pruebas:
  `TestRuntimeV0SupervisorPreparaAutomejoraTrasIdleV0` y
  `TestServerConfigFromEnvV0ConfiguraAutomejoraIdleV0`.
- Rework `task-ref-self-improvement-1089cc70db2b`: una run de
  autoprogramacion con agente externo pendiente no se relanza en ticks globales
  posteriores. Si quedan tareas autoprogramming abiertas y agentes externos
  pendientes, el stack mantiene `wait_external` hasta ACK, delivery, review o
  cierre causal. Prueba:
  `TestCodexStackAutoprogrammingPrepareRunAPIV0NoRelanzaAgentePendienteEnTicksGlobalesV0`.
- Revalidacion `agent-ref-task-autoprogramming-1089cc70db2b-g01-5eb20ecf12e9`:
  se repite la entrega tras revision no aceptada sin ampliar write-set. Queda
  confirmado el contrato: automejora idle configurable a 60 segundos por
  defecto, desactivable con `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0`,
  entrada por `self_improvement` con `auto_prepare_run=true`, preparacion en
  segundo plano y guarda contra relanzar runs de autoprogramacion con agentes
  externos pendientes. Las aperturas de staging, promocion y limpieza siguen
  como rail futuro por puertos/adaptadores opt-in. Refs opacas preservadas:
  `task-ref-self-improvement-1089cc70db2b`,
  `worktree-ref-orquesta-automejora-residente-004` y
  `trabajo-plataforma-agentes`.
- Revalidacion `agent-ref-task-autoprogramming-1089cc70db2b-g01-cf0b5d995b28`:
  se revisa de nuevo la entrega residente sin ampliar write-set. El contrato
  queda cerrado como servidor configurable: idle por defecto a 60 segundos,
  desactivacion por `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0`,
  propuesta por `self_improvement` con `auto_prepare_run=true`, preparacion en
  segundo plano y espera `wait_external` si una run de autoprogramacion conserva
  agente externo pendiente. Staging, promocion y limpieza siguen documentados
  como rail futuro opt-in, fuera del nucleo. Refs opacas preservadas:
  `task-ref-self-improvement-1089cc70db2b`,
  `worktree-ref-orquesta-automejora-residente-004` y
  `trabajo-plataforma-agentes`. Evidencia:
  `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-app-codex-stack`.
- Cierre `backlog-autoprogramacion-residente`: el servidor ya no prepara solo
  una automejora generica cuando queda idle. `orquesta-server` pide por puerto
  un plan de backlog y la composicion `cmd/orquesta-server` lee este documento
  para convertir secciones `Txx` pendientes en varias requests concretas, con
  write-set, tests, criterios y refs de contexto por seccion. El nucleo del
  servidor no conoce Git, Codex, OPES, web ni contenido de producto; si no hay
  planner configurado conserva el fallback legacy. La tanda se limita por
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_MAX_REQUESTS` para no crear mas
  agentes padre que tareas detectadas. Evidencia focal:
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.
- Apertura operativa relacionada: la deteccion idle no bloquea por una lista
  `Ranked` informativa si no hay ejecuciones ni skips reales. Bloquean
  ejecuciones y skips efectivos; una proyeccion residual no debe impedir que
  el director revise backlog y cree automejora. Evidencia:
  `go test -count=1 ./modulos/orquesta-server`.
- Correccion de adaptador: las rutas del backlog no se pasan como
  `context_refs` con `/`; el planner usa refs opacas compactas como
  `backlog-doc-autoprogramacion-2026-05-23` y deja la ruta real en docs/evidencia
  de composicion. Esto evita rechazos `workflow_task.context_refs` sin relajar
  el contrato puro.
- Correccion de backlog Markdown: el planner normaliza items de `Alcance`
  escritos como codigo inline antes de poblar `write_set`, de modo que
  el valor se entrega como ruta limpia y no conserva backticks. Tambien separa
  varios spans inline dentro de un mismo bullet de Markdown y acepta el formato
  compacto `Alcance: ...`. No se relaja el validador de ACK; la correccion
  queda en el adaptador que lee Markdown.
  Evidencia focal:
  `go test -count=1 ./cmd/orquesta-server`.
- Correccion de automejora residente bloqueada: si una automejora aceptada ya
  dejo `run_ref` y candidato ejecutable visible, los ticks idle posteriores no
  sustituyen la evidencia por `attempt_blocked`; conservan la razon `prepared`
  con `pending=accepted_attempt`. Si el `prepare-run` devuelve una run que solo
  aparece en cola con estado no ejecutable, la composicion la rechaza como
  `idle_self_improvement_queue_candidate_not_executable` para permitir retry y
  no dejar al residente bloqueado por una aceptacion falsa. Evidencia focal:
  `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.

## Pendientes de rail futuro

- El rail completo de staging/promocion de automejora residente sigue pendiente:
  worktree temporal, merge/promocion, limpieza y control de solapes de write-set
  deben entrar por puertos/adaptadores opt-in. El servidor solo cablea refs
  opacas y configuracion; no convierte `worktree_ref` ni `branch_ref` en rutas
  ni nombres Git.

## Ciclo de automejora residente

Regla vigente:

```text
cola normal sin trabajo ejecutable o con huecos detectados
  -> revisar backlog/automejora
  -> preparar run aislada de baja prioridad
  -> ejecutar agentes y subagentes por Orquesta
  -> revisar entrega y pruebas requeridas
  -> si esta cerrada y no pisa trabajos vivos, promocionar por conector VCS
  -> limpiar/archivar staging temporal
  -> volver a revisar
```

La cola operativa actual es `RunQueue` global. Las automejoras entran por
`orquesta.autoprogramming.self_improvement.propose.v0` con
`auto_prepare_run=true`, que las convierte en `prepare_run` y las encola con
prioridad baja. El residente debe tratar esa cola como trabajo autonomo normal
cuando esta parado.

Frontera pendiente de codigo: el staging temporal completo debe implementarse
fuera del nucleo, en puertos/adaptadores. El contrato puro puede vivir en
`orquesta-autoprogramming`; Git/worktree/promocion/cleanup pertenecen a
`orquesta-runtime-worktree` y la composicion `orquesta-app-codex-stack`; el
servidor solo cablea rutas/env opt-in. El core no conoce rutas temporales,
shell, Git ni proveedor. La promocion solo puede ocurrir con evidencia durable
de tests y sin tareas vivas solapadas en write-set; si hay solape, se deja la
promocion como tarea posterior en cola, no se pisa el trabajo a medias.

## T01 server-autonomy

Objetivo: reforzar el servidor residente para autoprogramacion desatendida.

Estado: completada por Orquesta en
`request-ref-autoprogramming-backlog-t01-server-autonomy-aa79787f` y rework
posterior `request-ref-automejora-residente-attempt-blocked-20260524`.
Evidencia focal: `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-server`

Criterios:

- El loop de supervisor debe seguir funcionando aunque un tick falle; registrar
  error durable pero permitir ticks posteriores.
- Exponer/registrar suficientes numeros para saber cola, ejecuciones, skips,
  ultimo error y ultimo tick sin depender del operador.
- No meter conocimiento de Codex ni de tareas de producto en `orquesta-server`.
- Tests: `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.

## T02 director-self-repair

Objetivo: mejorar la capacidad del director/stack para corregir desviaciones
pequenas de agentes sin descartar trabajos completos.

Estado: completada por Orquesta en
`request-ref-autoprogramming-backlog-t02-director-self-repair-bce34e34`.
Evidencia focal:
`go test -count=1 ./modulos/orquesta-app-codex-stack -run 'Review|Rework|Repair|Autoprogramming|Drain'`.

Alcance:

- `modulos/orquesta-app-codex-stack`

Criterios:

- Normalizar pequenos desajustes de entregas: aliases de rutas, refs derivables,
  nombres equivalentes y tests declarados con diferencias inocuas.
- Si falta algo real, pedir rework acotado; no rehacer todo salvo que sea mas
  barato y quede justificado.
- Limitar reintentos por task para evitar bucles, pero con margen suficiente.
- Tests: `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'Review|Rework|Repair|Autoprogramming|Drain'`.

## T03 replay-state

Objetivo: cerrar huecos de replay/idempotencia de metadata viva del Director.

Estado: completada por Orquesta en
`request-ref-autoprogramming-backlog-t03-replay-state-5785a896`.
Evidencia focal:
`go test -count=1 ./modulos/orquesta-state-file ./modulos/orquesta-orchestration-core`.

Alcance:

- `modulos/orquesta-state-file`
- `modulos/orquesta-orchestration-core`

Criterios:

- Restaurar o rematerializar de forma verificable `WorkflowTaskStore`,
  `WorkflowTaskWaitStateV0` y `OperationalDirectorPlanStateV0`.
- No reconstruir desde eventos compactos si faltan metadatos que solo viven en
  stores; documentar y probar la frontera.
- Tests: `go test -count=1 ./modulos/orquesta-state-file ./modulos/orquesta-orchestration-core`.

## T04 api-mcp-autoprogramming

Objetivo: completar la superficie API/MCP de gestion de autoprogramacion.

Estado: completada por Orquesta para la frontera MCP/API acotada en
`request-ref-autoprogramming-backlog-t04-api-mcp-autoprogramming-c843b9a4`.
La ampliacion gateway/producto queda fuera de esta tarea y debe entrar con ref
nueva si se detecta hueco concreto.

Alcance:

- `modulos/orquesta-mcp`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-http-gateway`

Criterios:

- Endpoints/herramientas finas para estado de cola/run, supervision puntual y
  diagnostico de autoprogramacion.
- No ejecutar logica de negocio en HTTP/MCP; solo adaptar a puertos/casos de uso.
- i18n/errores publicos coherentes con los modulos existentes.
- Tests: `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway`.

Estado MCP del corte `request-ref-automejora-mcp-operador-004`: completado para
`modulos/orquesta-mcp`, `modulos/orquesta-operator-mcp` y
`modulos/orquesta-operator-mcp-client` con la prueba focal
`go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-operator-mcp-client`.
La parte de gateway amplio queda como trabajo separado porque esta fuera de este
write-set.

## T05 web-autoprogramming

Objetivo: mejorar la web como cliente fino de autoprogramacion.

Alcance:

- `modulos/orquesta-web`

Criterios:

- Vista/modelos para cola, runs, agentes, errores publicos y progreso.
- La web no accede a stores ni a runtime; consume endpoints/cliente inyectado.
- No meter textos de instrucciones visibles en la UI; i18n ES/EN si hay textos.
- Tests: `go test -count=1 ./modulos/orquesta-web`.

Estado: completada para cliente/proyeccion fina. `modulos/orquesta-web`
transporta `prepare-run` y `status`, proyecta cola, runs, agentes, progreso,
diagnosticos y errores publicos, y conserva la frontera de adaptador sin leer
stores ni runtime. Pendiente futuro separado: montar una vista HTML completa
encima de esos view-models si producto la requiere.

Revalidacion `task-autoprogramming-7447aef8a77d-g01`: `/run-queue`,
`/director-stats`, `/run-control`, `prepare-run` y `status` quedan cubiertos
como superficie web/API de operador. La web no interpreta refs como paths ni
ramas Git.

## T06 cli-thin-client

Objetivo: completar CLI como cliente fino del servidor, sin entrar en nucleo.

Alcance:

- `cmd/orquesta-cli`
- `modulos/orquesta-cli`

Criterios:

- Comandos para ver estado de servidor, cola y runs de autoprogramacion usando
  HTTP/API cuando proceda.
- Sin acceso directo a stores internos del nucleo ni imports de producto en core.
- Help ES/EN y rechazo de argumentos sobrantes.
- Tests: `go test -count=1 ./cmd/orquesta-cli ./modulos/orquesta-cli`.

Estado: completada para autoprogramacion supervisada. `modulos/orquesta-cli`
expone comandos para estado de servidor, prepare-run, status, cola, detalle de
run, supervision acotada y control seguro por HTTP/API. No queda fallback local
ni conversion de `worktree_ref`/`branch_ref` a ruta o rama Git.

Revalidacion `task-autoprogramming-7447aef8a77d-g01`: la CLI conserva el flujo
`servidor estado -> cola listar -> preparar -> estado ver -> supervisar -> run
ver -> run controlar` como cliente fino sobre rutas publicas. La prueba
obligatoria compartida con web y servidor queda documentada arriba.

Revalidacion final `agent-ref-task-autoprogramming-7447aef8a77d-g01-95364243f6e0`:
web y CLI quedan revisados como adaptadores de API para listado de cola/runs,
detalle, prepare-run, status, supervise y control seguro. Evidencia ejecutada el
2026-05-23: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./cmd/orquesta-server`.
No se abre deuda nueva ni se promocionan refs opacas a paths, ramas Git,
proveedor, HOME o stores locales.

Revalidacion de rework `agent-ref-task-autoprogramming-7447aef8a77d-g01-6ea621489958`:
se revisa de nuevo la entrega web/CLI sin ampliar write-set. Los contratos de
listado de cola/runs, detalle, prepare-run, status, supervise y control seguro
siguen implementados como clientes finos de API; no hay acceso directo a stores,
runtime, Git/worktrees, HOME, proveedor ni paths locales. Evidencia ejecutada el
2026-05-23: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./cmd/orquesta-server`.

Revalidacion de rework `agent-ref-task-autoprogramming-7447aef8a77d-g01-35bb4432e77b`:
se cierra la repeticion de revision para web/CLI como contrato ya implementado.
La superficie operativa sigue siendo `prepare-run`, `status`, cola, detalle,
supervise y control seguro por API publica; `worktree-ref-orquesta-automejora-web-cli-004`
y `trabajo-plataforma-agentes` se conservan solo como refs opacas. Evidencia
ejecutada el 2026-05-23:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./cmd/orquesta-server`.

Revalidacion de rework `agent-ref-task-autoprogramming-7447aef8a77d-g01-ff7908424b11`:
se revisa de nuevo la entrega tras revision no aceptada y se mantiene el cierre
como contrato web/CLI ya implementado. Web y CLI cubren listado de cola/runs,
detalle, prepare-run, status, supervise y control seguro por APIs publicas del
servidor; no leen stores, runtime, Git/worktrees, HOME, proveedor ni paths
locales. `worktree-ref-orquesta-automejora-web-cli-004` y
`trabajo-plataforma-agentes` siguen siendo refs opacas. Evidencia ejecutada el
2026-05-23:
`go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./cmd/orquesta-server`.

## T07 opes-consumer-smoke

Objetivo: cerrar el smoke OPES aislado pendiente como consumidor, no como nucleo.

Estado: completada por Orquesta para script/runbook aislado en
`request-ref-autoprogramming-backlog-t07-opes-consumer-smoke-0922de29`.
La ejecucion real opt-in completa contra instancia temporal queda como tarea
nueva `T12`, para no reutilizar una run ya materializada.

Alcance:

- `scripts`
- `docs/runbooks`
- `modulos/orquesta-opes-bridge`
- `modulos/orquesta-opes-connector`

Criterios:

- Smoke temporal aislado para derivados/cierre OPES sin tocar OPES productivo.
- Guardas de confirmacion y filtro por tipo de job.
- Documentar evidencia y limites; no drenar colas amplias.
- Tests: `go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector`.

## T08 runtime-neutral-e2e

Objetivo: probar runtime externo no-Codex a nivel contrato neutral.

Estado: completada localmente el 2026-05-24. Evidencia focal:
`RUNTIME-020`, `TestRuntimeNeutralE2EV0LaunchProgressStopSinProviderConcreto` y
`go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-runtime-worktree`.
No cierra proveedor, CLI real, cuota ni transporte productivo; eso debe entrar
con tarea nueva si se decide.

Alcance:

- `modulos/orquesta-runtime`
- `modulos/orquesta-runtime-worktree`

Criterios:

- E2E fake/contractual para launch/progress/stop sin provider concreto.
- El nucleo no debe saber de Codex ni CLI real; usar refs opacas y puertos.
- Tests: `go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-runtime-worktree`.

## T09 local-sensitive-data-sanitizer

Objetivo: anadir una frontera de saneamiento local antes de mandar contexto a
agentes premium/remotos.

Estado: completada localmente el 2026-05-24 para contrato, puerto e inyeccion
opt-in en stack Codex. Evidencia focal:
`docs/runbooks/local_sensitive_data_sanitizer.md`,
`APP-CODEX-STACK-025` y
`go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-runtime ./modulos/orquesta-app-codex-stack`.
No convierte el sanitizador en proveedor/modelo ni lo activa si la composicion
no lo inyecta.

Alcance:

- `modulos/orquesta-context`
- `modulos/orquesta-runtime`
- `modulos/orquesta-app-codex-stack`
- `docs/runbooks`

Criterios:

- Definir puerto neutral para sanitizar contexto saliente sin meter IA,
  proveedor, modelo ni transporte en el nucleo.
- Implementar adaptador opt-in para IA local o sanitizador local que detecte y
  sustituya claves, tokens, secretos, rutas privadas y material no publicable
  por refs opacas.
- Conservar evidencia durable de saneamiento sin persistir el dato sensible.
- Si el sanitizador duda, pedir revision humana/director o enviar contexto
  minimo por refs, no bloquear toda la ejecucion salvo riesgo efectivo.
- Tests: `go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-runtime ./modulos/orquesta-app-codex-stack`.

## T10 flaky-tests-observability

Objetivo: tratar fallos intermitentes como prioridad de fiabilidad, no como
ruido.

Estado: completada localmente el 2026-05-24 para el caso observado T10.
Evidencia focal:
`docs/runbooks/flaky_tests_observability_2026-05-24.md`,
`TestFlakyHarnessV0RepiteCasoDirectorRecursiveFakeRuntimeV0` y
`go test -count=1 ./cmd/orquesta-server`. Nuevos flakes deben registrarse como
casos nuevos, no reabrir este cierre.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-runtime`
- `modulos/orquesta-app-codex-stack`
- `docs/runbooks`

Criterios:

- Registrar tests intermitentes con fecha, comando, fallo exacto y reintento
  posterior.
- Aislar fuentes de no determinismo: orden de mapas, tiempos, concurrencia,
  ficheros temporales, salida stdout/stderr y estado compartido.
- Anadir harness de repeticion acotado para tests criticos antes de marcarlos
  como fiables.
- Caso observado 2026-05-23: `go test -count=1 ./...` fallo una vez en
  `cmd/orquesta-server` con
  `TestCodexLaunchDirectorWaveCommandV0RecursiveFakeRuntimeEjecutableConLinaje`
  por stdout sin prompt esperado; el test focal y `go test -count=1
  ./cmd/orquesta-server` pasaron al reintentar. Debe investigarse como flake.
- Tests: `go test -count=1 ./cmd/orquesta-server` y repeticion focal del caso
  observado.

## T11 rails-blandos-y-falsos-positivos

Objetivo: mantener los rails dudosos como material de automejora, no como cortes
duros de produccion.

Alcance:

- `modulos/orquesta-autoprogramming`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-runtime-codex-delivery`
- `modulos/orquesta-app-codex-stack`

Criterios:

- `file_too_large`, `file_outside_write_set` y
  `write_set_target_missing:*` no deben relanzar agentes ni tirar entregas:
  quedan como evidencia/advisory para revision posterior.
- Los detectores por palabras genericas (`token`, `secret`, `prompt`,
  `provider`, `home`, etc.) no deben cortar ACKs ni observaciones; se conservan
  como rails pendientes para probar fuera de Orquesta.
- Antes de volver a endurecer un rail hay que probarlo fuera del servidor con
  matrices amplias de falsos positivos/falsos negativos y documentar evidencia.
- Regla de concurrencia de programacion: un agente padre por tarea. Los
  subagentes pueden ayudar al padre, hasta 6, pero el rework no crea otro padre
  sobre la misma tarea; debe crear una tarea nueva de correccion/revision si hace
  falta.
- Tests externos ya introducidos:
  `TestAutoprogrammingReviewGateExternalMatrixV0`,
  `TestCodexAckPendingRailExternalMatrixV0`,
  `TestReviewReworkReplanSourceV0NoCreaSegundoPadreParaMismaTarea`.

## T12 opes-consumer-smoke-real-opt-in

Objetivo: ejecutar y documentar el smoke OPES completo contra instancia temporal
aislada, sin tocar OPES productivo.

Alcance:

- `scripts`
- `docs/runbooks`
- `modulos/orquesta-opes-bridge`
- `modulos/orquesta-opes-connector`

Criterios:

- Usar solo instancia temporal/confirmacion explicita y filtro por tipo de job.
- Cubrir derivados/cierre hasta `assemble_topic` cuando el conector temporal lo
  permita; si falta entorno, dejar bloqueo verificable con comando exacto.
- No drenar colas amplias ni tocar OPES productivo.
- Tests: `go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector`.

## Escaneo backlog 2026-05-24

Evidencia revisada: `AGENTS.md`, `README.md`,
`docs/estado_actual_2026-05-17.md`,
`docs/guia_nucleo_orquestacion_2026-05-17.md`,
`docs/principio_orquesta_piensa_director.md`,
`docs/corte_cierre_generico_director_operativo_2026-05-17.md`,
`docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`,
`docs/corte_opes_como_consumidor_orquesta_2026-05-18.md`,
este backlog y los indices de rails. No se programa codigo desde este scanner:
se anaden tareas ejecutables para ciclos posteriores y se evita duplicar T08-T12.

Huecos concretos nuevos o reencuadrados:

- El rail de staging/promocion de automejora residente esta descrito como
  pendiente, pero no tenia seccion Txx ejecutable propia.
- El smoke no-OPES temporal ya cierra el ciclo con validador fake; falta
  convertir la politica productiva de tests de dominio no-OPES en puerto/adaptador
  verificable.
- `ORQUESTA_DETAIL_PROHIBITED_RAILS=off` quedo como apertura temporal por
  defecto; falta plan de reactivacion por campo con matrices externas antes de
  volver a endurecer.
- El conector MCP/HTTP de operador esta cerrado como frontera fina, pero el
  servidor/transporte MCP real productivo sigue opt-in y sin smoke de composicion.

## T13 autoprogramming-staging-promotion

Objetivo: implementar el rail completo de staging/promocion de automejora
residente fuera del nucleo, con refs opacas y control de solapes.

Estado: completada 2026-05-24. T40 anade el cierre e2e acotado con repo Git
temporal y replay idempotente; push remoto/productivo sigue fuera de T13.

Alcance:

- `modulos/orquesta-autoprogramming`
- `modulos/orquesta-runtime-worktree`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/runbooks`

Criterios:

- Definir puerto neutral para staging/promocion/cleanup sin meter Git, rutas,
  HOME, shell ni proveedor en core.
- Conservar `worktree_ref` y `branch_ref` como refs opacas; la composicion opt-in
  resuelve rutas/nombres reales y guarda evidencia durable.
- Promocionar solo tras cierre causal, tests requeridos pasados y ausencia de
  tareas vivas solapadas por write-set; si hay solape, crear followup o dejar
  promocion pendiente.
- Cleanup/archivo debe ser idempotente y no borrar trabajo vivo ni artefactos de
  evidencia.
- Tests: `go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

## T14 domain-work-required-tests-productivos

Objetivo: convertir la validacion fake de tests de dominio no-OPES en politica
productiva por puerto para apps externas genericas.

Estado: completada localmente el 2026-05-24 para el planner residente de
automejora. Evidencia focal:
`go test -count=1 ./cmd/orquesta-server -run TestIdleSelfImprovementBacklogPlannerV0`.
Evidencia requerida:
`go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.
Runbook:
`docs/runbooks/autoprogramacion_backlog_state_sync_2026-05-24.md`.

Alcance:

- `modulos/orquesta-domain-work`
- `modulos/orquesta-app-change`
- `modulos/orquesta-app-director-service`
- `modulos/orquesta-app-codex-stack`
- `docs/runbooks`

Criterios:

- `DomainWorkRequiredTestPolicyPortV0` debe resolver criterios, evidencias y
  validadores de dominio sin conocer OPES, DB interna, filesystem del producto ni
  tests de programacion inventados.
- La politica debe aceptar refs opacas y artefactos causales; rechazar URL/DB/ruta
  interna como evidencia suficiente de cierre.
- El cierre de `domain_work` debe distinguir falta de evidencia, rechazo de
  dominio y latencia de ingesta, con reentrada idempotente.
- Reusar el smoke no-OPES temporal como evidencia de ruta viva, pero separar el
  validador productivo del fake de pruebas.
- Tests: `go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-app-change ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack`.

## T15 detail-rails-reactivation

Objetivo: reemplazar la apertura temporal de `detalle_prohibido` por una politica
reactivable por campo y por frontera, con matriz de falsos positivos antes de
volver a endurecer.

Estado: parcialmente cerrado para reactivacion acotada del servidor. El codigo
vigente de `cmd/orquesta-server` ya fija por defecto
`ORQUESTA_DETAIL_PROHIBITED_RAILS=on` con scope limitado a
`core_workflow.*`, `context_bundle_request.*`,
`context_materialization.content`, `context_materialization.ref` y
`director_agent_decision.*`; queda pendiente sincronizar docs de rails y ampliar
matriz antes de declarar cerrado el frente completo.

Alcance:

- `modulos/orquesta-rails`
- `modulos/orquesta-core-workflow`
- `modulos/orquesta-context`
- `modulos/orquesta-director-agent`
- `cmd/orquesta-server`
- `scripts`

Criterios:

- Mantener por defecto tolerancia a refs opacas, alias razonables y vocabulario
  operativo (`runtime`, `provider`, `model`, `codex`, `git`, `db`, `sql`,
  `prompt policy`, `transcript policy`).
- Cortar solo valores sensibles efectivos, material crudo o efectos externos no
  autorizados, con decision por campo y evidencia reproducible.
- Mantener la reactivacion acotada de `ORQUESTA_DETAIL_PROHIBITED_RAILS=on`
  solo en fronteras con matriz externa suficiente; cualquier nuevo scope debe
  demostrar que no bloquea autoprogramacion, review, cierre ni ACKs validos.
- Registrar cada lista local restante en un propietario por modulo o sustituirla
  por helper/puerto de sanitizado.
- Tests: `go test -count=1 ./modulos/orquesta-rails ./modulos/orquesta-core-workflow ./modulos/orquesta-context ./modulos/orquesta-director-agent ./cmd/orquesta-server`.

## T16 mcp-server-real-opt-in

Objetivo: cerrar un smoke opt-in del transporte MCP real para herramientas de
operador/automejora, sin mover logica de negocio a MCP.

Estado: completada localmente el 2026-05-24 para transporte JSON-RPC HTTP
opt-in en `cmd/orquesta-server` y smoke temporal. Evidencia local:
`docs/runbooks/smoke_mcp_real_transport_opt_in_2026-05-24.md`,
`go run ./cmd/orquesta-server mcp-real-smoke` con
`ORQUESTA_MCP_REAL_SMOKE_CONFIRM=1`, y
`go test -count=1 ./cmd/orquesta-server -run TestMCPRealTransport`.
Pendiente separado: activar el transporte en `run` residente por env opt-in si
producto lo necesita.

Alcance:

- `modulos/orquesta-mcp`
- `modulos/orquesta-operator-mcp`
- `modulos/orquesta-operator-mcp-client`
- `cmd/orquesta-server`
- `docs/runbooks`

Criterios:

- Mantener `status`, `supervise`, `self_improvement`,
  `human_work.review_plan`, `operator.operations` y `domain_work` como
  adaptadores finos sobre puertos inyectados.
- El transporte MCP real debe ser opt-in, con errores publicos, i18n donde haya
  texto visible y sin leer stores, runtime, Git/worktrees, HOME, proveedor ni
  paths locales.
- Si falta operador real, conservar plan y registrar consulta/observacion
  publica; no bloquear automejora por ausencia de conector.
- Smoke temporal debe probar al menos una llamada de lectura, una propuesta de
  automejora no destructiva y un error de operador normalizado.
- Tests: `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-operator-mcp-client ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 segunda pasada

Evidencia revisada: backlog vigente, `cmd/orquesta-server` planner de
automejora idle, docs locales de `orquesta-runtime`, `orquesta-runtime-worktree`,
`orquesta-app-codex-stack`, `orquesta-mcp`, `orquesta-operator-mcp` y matriz de
smokes. No se programa codigo desde este scanner.

Cierres re-clasificados para evitar duplicacion de cola:

- `T08 runtime-neutral-e2e` ya tiene `RUNTIME-020` y test contractual no-Codex.
- `T09 local-sensitive-data-sanitizer` ya tiene puerto, adaptador opt-in,
  runbook y tests.
- `T10 flaky-tests-observability` ya tiene runbook, harness acotado y causa del
  flake observado.

Huecos concretos nuevos o reencuadrados:

- El planner de backlog salta ACKs completados en `.orquesta-runtime` y refs ya
  visibles en cola, pero el documento sigue siendo la fuente humana. Si una
  seccion completada queda sin `Estado: completada`, puede volver a generar
  trabajo cuando el ACK no este disponible. Hace falta sincronizacion durable
  entre ACKs/evidencia/modulo-docs y este backlog.
- El smoke OPES de derivados/cierre no debe cerrarse solo por wrapper o dry-run.
  Falta una fuente/validador causal OPES que traduzca aceptacion de artefactos,
  deduplicacion y `assemble_topic` a refs opacas consumibles por cierre
  operativo.

## T17 autoprogramming-backlog-state-sync

Objetivo: sincronizar estado de backlog de autoprogramacion con evidencia
durable para no relanzar secciones ya completadas ni depender solo de ACKs
locales presentes en `.orquesta-runtime`.

Estado: completada localmente el 2026-05-24 para sincronizacion de estado de
backlog. Evidencia focal: `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.
Contrato cerrado en
`docs/runbooks/autoprogramacion_backlog_state_sync_2026-05-24.md`.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/runbooks`

Criterios:

- El planner debe poder detectar secciones cerradas por ACK durable, docs locales
  de modulo o evidencia declarada y reflejar/consumir esa clausura sin convertir
  refs opacas en rutas, ramas Git ni nombres de proveedor.
- Si una tarea ya esta visible en cola, no debe crear fallback generico ni
  scanner duplicado; debe dejar evidencia compacta `backlog_tareas_ya_visibles_en_cola`.
- Una seccion sin estado explicito pero con evidencia ambigua debe producir
  candidato de revision documental, no ejecucion amplia de codigo.
- No leer Git como fuente de verdad ni tocar worktrees desde `orquesta-server`.
- Tests: `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.

## T18 opes-operational-closure-source

Objetivo: cerrar OPES temporal real de derivados/cierre con fuente causal de
aceptacion por refs opacas, sin convertir OPES en producto base del nucleo.

Estado: completada 2026-05-24.

Alcance:

- `modulos/orquesta-opes-bridge`
- `modulos/orquesta-opes-connector`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-app-director-service`
- `cmd/orquesta-server`
- `docs/runbooks`

Criterios:

- La composicion OPES debe aportar fuente/validador de cierre por puerto: job
  externo, artefacto aceptado, receipt, deduplicacion, validacion de dominio y
  `assemble_topic -> assembled_topic` via refs opacas.
- El nucleo no debe leer DB/filesystem OPES ni deducir aceptacion por URL,
  ruta local, nombre de conector o summary textual.
- El smoke real debe exigir OPES temporal, confirmacion explicita, filtro por
  tipo/job y limites bajos; no drenar cola OPES productiva.
- Si una fase derivada queda pendiente o rechazada, debe bloquear/replanificar
  causalmente el task afectado, no cerrar el plan completo por haber entregado
  un artefacto formal.
- Tests: `go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector ./modulos/orquesta-app-codex-stack ./modulos/orquesta-app-director-service ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 tercera pasada

Evidencia revisada: backlog vigente,
`docs/auditoria_runtime_orquesta_2026-05-24.md`,
`docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`,
`docs/guia_nucleo_orquestacion_2026-05-17.md`,
`docs/runbooks/resultado_smoke_opes_consumer_t12_2026-05-24.md`,
`docs/rails/registro_railes_2026-05-24.md`, indice de duplicaciones de rails,
scripts de smoke y docs locales de `orquesta-observability`,
`orquesta-domain-work-sql`, `orquesta-server`, `orquesta-web` y
`cmd/orquesta-server`. No se programa codigo desde este scanner.

Huecos concretos nuevos o reencuadrados:

- La auditoria JSONL del servidor ya existe y contiene diagnostics utiles, pero
  no hay visor filtrable, politica de rotacion/retencion ni captura opt-in
  saneada de payloads HTTP completos.
- `orquesta-domain-work-sql` es adaptador driver-neutral con contract tests,
  pero la matriz conserva pendiente un bundle real opt-in por dialecto con DB
  temporal, schema/migraciones, DSN externo y teardown.
- Los smokes reales quedan repartidos entre scripts con guardas explicitas y
  filas documentales; `USAGE-METRICS-REAL` declara que su script no tiene guarda
  propia de confirmacion aunque puede lanzar Codex. Hace falta una politica
  comun para marcar y validar smokes reales antes de meterlos en ejecucion
  automatica.

## T19 server-audit-ops-surface

Objetivo: convertir la auditoria JSONL del servidor residente en superficie
operativa filtrable y retenible, sin crear persistencia paralela ni exponer
payloads sensibles por defecto.

Estado: cerrado en 2026-05-24 para lectura terminal estricta en observacion,
startup compaction y planner de automejora; la lectura compatible queda solo
para diagnostico/legacy y reparaciones explicitas del adaptador.

Alcance:

- `modulos/orquesta-server`
- `cmd/orquesta-server`
- `modulos/orquesta-web`
- `docs/runbooks`

Criterios:

- Exponer lectura filtrable por fecha, event, run_ref, outcome, status,
  target_port y message_type usando puertos de lectura/adaptadores, no acceso
  directo a internals desde web/API.
- Mantener metadata HTTP por defecto; captura de payload completo solo opt-in,
  local, con saneamiento previo y evidencia de que no persiste secretos,
  transcripts crudos ni cuerpos grandes.
- Anadir rotacion/retencion configurable con defaults conservadores y prueba de
  que el servidor sigue arrancando si el fichero de auditoria falta, rota o esta
  vacio.
- La web debe ser cliente fino con i18n si aparece texto visible; no interpreta
  refs como rutas ni lee stores internos.
- Tests: `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server ./modulos/orquesta-web`.

## T20 domain-work-sql-real-dialect-bundle

Objetivo: cerrar un bundle opt-in de dialecto SQL real para `domain_work`, sin
convertir SQL en persistencia global de Orquesta ni compartir DB con apps
externas.

Estado: completado.

Alcance:

- `modulos/orquesta-domain-work-sql`
- `modulos/orquesta-domain-work`
- `cmd/orquesta-server`
- `docs/runbooks`

Criterios:

- Probar al menos un dialecto real temporal con driver, schema/migraciones,
  DSN inyectado por entorno de smoke, `IsUniqueViolation` propio y teardown
  limpio.
- Ejecutar la contract suite comun de `DomainWorkJobRecordStorePortV0` contra
  el dialecto real: idempotencia, conflictos, unicidad, filtros AND, limite,
  replay tras reinicio y carrera concurrente replay/conflict.
- Mantener el wiring productivo apagado por defecto; cualquier seleccion de SQL
  debe ser opt-in de composicion y no entrar al nucleo, director ni
  `orquesta-domain-work`.
- No registrar drivers ni abrir DSN desde el paquete neutral; secretos y DSN
  reales no se escriben en eventos, ACKs, audit logs ni docs de resultado.
- Tests: `go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-domain-work-sql ./cmd/orquesta-server`.

## T21 real-smoke-guards-and-catalog

Objetivo: normalizar guardas, catalogo y auditoria de smokes reales para que el
residente no ejecute por accidente pruebas con Codex, OPES, red, DB temporal o
efectos externos.

Estado: cerrado 2026-05-24 para la contradiccion de paquete/prompt; T47 y T48
siguen como frentes separados.

Alcance:

- `scripts`
- `cmd/orquesta-server`
- `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`
- `docs/runbooks`

Criterios:

- Cada smoke real o invasivo debe declarar confirm env explicito, recursos
  temporales requeridos, limites bajos, salida de evidencia y criterio de no
  tocar OPES productivo ni colas amplias.
- Smokes solo offline/fake deben quedar diferenciados de smokes reales para que
  la automejora residente no los trate igual.
- `USAGE-METRICS-REAL` debe recibir guarda de confirmacion o quedar excluido de
  ejecucion automatica con razon durable.
- El catalogo debe permitir al planner seleccionar solo pruebas razonables para
  el workdir actual y dejar bloqueos verificables cuando falten env, cuota,
  proveedor, DB temporal o instancia OPES temporal.
- Tests: `bash -n scripts/*.sh` y `go test -count=1 ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 cuarta pasada

Evidencia revisada: backlog vigente, paquete de agente de scanner,
`cmd/orquesta-server/detail_rails_env_v0.go`,
`cmd/orquesta-server/mcp_real_smoke_v0.go`,
`cmd/orquesta-server/mcp_real_transport_v0_test.go`,
`scripts/test_rails_fast.sh`, runbook MCP real opt-in, documentos de estado
vigente y docs de rails dentro del write-set permitido. No se programa codigo
desde este scanner.

Huecos concretos nuevos o reencuadrados:

- `T16` ya tiene transporte y smoke local opt-in, pero el backlog seguia como
  pendiente; queda re-clasificado para evitar relanzar trabajo cerrado.
- La reactivacion de `detalle_prohibido` ya no es solo propuesta: el servidor
  arranca con `ORQUESTA_DETAIL_PROHIBITED_RAILS=on` por defecto y scope acotado.
  `docs/rails/registro_railes_2026-05-24.md` queda fuera de este write-set y
  aun dice que el servidor usa `off`, por lo que hace falta sincronizacion
  documental y matriz de alcance antes de abrir mas scopes.
- El smoke MCP real existe como comando temporal `mcp-real-smoke`; falta decidir
  si el servidor residente debe exponer `/mcp` en modo `run` con opt-in, guardas
  y auditoria compacta, o si el comando temporal es la frontera suficiente.

## T22 detail-rails-doc-state-sync

Objetivo: sincronizar estado documental, matriz rapida y backlog tras la
reactivacion acotada de `detalle_prohibido` en el servidor.

Estado: completado.

Evidencia focal 2026-05-24: `ContextMaterializedBundleV0` clasifica cada
entrada `required` que queda como `ref_only` con `ref_only_reason` y
`required_ref_action`; `AgentStartPacketV0` propaga la policy
`required_ref_only_context_guard`; el prompt Codex declara la accion esperada
para `required ref_only`; y la validacion de ACK completed rechaza contexto
`required ref_only` no diseñado si falta nota `contexto_ref_only_resuelto`.

Alcance:

- `docs/rails`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`
- `cmd/orquesta-server`
- `modulos/orquesta-rails`
- `scripts`

Criterios:

- Actualizar el registro vivo de rails para reflejar que el default del servidor
  es `on` con scope acotado, no apertura global `off`.
- Mantener tolerancia a vocabulario operativo opaco fuera de los scopes
  reactivados; cualquier scope nuevo requiere caso externo de falsos positivos.
- La matriz rapida debe cubrir env explicito `off`, default `on` acotado, scope
  por frontera/campo y casos reales de autoprogramacion con `runtime`,
  `provider`, `model`, `prompt policy` y `transcript policy`.
- No usar la reactivacion de detalle como permiso para persistir payloads crudos
  en auditoria, ACKs, eventos ni docs de resultado.
- Tests: `./scripts/test_rails_fast.sh` y `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-rails`.

## T23 mcp-run-transport-opt-in

Objetivo: decidir e implementar, si procede, exposicion MCP real en el servidor
residente como transporte opt-in, separada del smoke temporal ya cerrado.

Estado: cerrado localmente el 2026-05-24.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-mcp`
- `modulos/orquesta-operator-mcp`
- `modulos/orquesta-operator-mcp-client`
- `docs/runbooks`

Criterios:

- Si se expone `/mcp` en modo `run`, debe requerir env opt-in explicito,
  loopback o bind configurado, errores publicos, payload compacto y auditoria
  sin secretos ni transcripts.
- Mantener `orquesta-mcp` como registro/contrato fino; el transporte real vive
  en composicion y no lee stores, runtime, Git/worktrees, HOME, OPES, Codex,
  proveedor ni paths locales por si mismo.
- Si falta operador real, conservar `operator_mcp_port_unavailable` como error
  publico no bloqueante.
- Si se decide no activar `/mcp` residente, documentar la frontera como comando
  temporal suficiente y marcar el backlog cerrado con evidencia.
- Tests: `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-operator-mcp-client ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 quinta pasada

Evidencia revisada: paquete del scanner de automejora,
`cmd/orquesta-server/startup_check*.go`,
`cmd/orquesta-server/idle_self_improvement_backlog_planner_v0.go`,
`modulos/orquesta-runtime-codex/docs/contratos.md`,
`modulos/orquesta-mcp/project_roadmap_descriptors_v0.go`,
`modulos/orquesta-mcp/shared_contracts_descriptors_v0.go`,
`modulos/orquesta-app-codex-stack/domain_work_delivery_quality_v0.go`,
`modulos/orquesta-domain-work`, este backlog, rail errors y duplicaciones de
rails dentro del write-set permitido. No se programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos o reencuadrados:

- La purga/compactacion de arranque detecta ACKs completados en runtime por
  texto de `codex_last_message.txt` con `ack` y `completed`. El contrato Codex
  ya exige `agent_ack.json` estructurado; usar texto de salida como fuente de
  cierre puede producir falsos positivos o saltarse correlacion.
- `orquesta.project.roadmap.v0` y `orquesta.shared_contracts.v0` conservan
  descriptores estaticos con estados `pendiente_*` de una foto anterior. Como
  MCP es superficie preferente para IA, esta duplicacion puede reabrir trabajo
  cerrado o esconder backlog vigente.
- La calidad de `topic_expansion_package` vive en el stack Codex con reglas de
  strings (`TODO`, `placeholder`, `pendiente_revision`) y recuento de palabras.
  Es una politica de dominio/adaptador, no una regla del stack; necesita matriz
  por campo antes de endurecer o relajar.

## T24 startup-structured-ack-compaction

Objetivo: reemplazar la deteccion textual de ACK completado en la compactacion
de arranque por lectura estructurada y correlada de `agent_ack.json`/receipt.

Estado: completada 2026-05-24 para paquetes OrquestaV2 estrictos.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-app-codex-stack`
- `docs/runbooks`

Criterios:

- `startupRunHasCompletedAckV0` o su reemplazo debe leer ACK estructurado,
  validar `schema_version`, `status=completed`, `run_ref`/`agent_ref` cuando
  existan y correlacion con el descriptor o run observado.
- `codex_last_message.txt` puede quedar como diagnostico, no como fuente
  suficiente para purgar cola, archivar runtime ni declarar cierre.
- La purga debe distinguir ACK ausente, ACK corrupto, ACK de otro agente y ACK
  valido sin convertir rutas de runtime en refs de producto.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack`.

## T25 mcp-roadmap-backlog-state-sync

Objetivo: sincronizar los recursos MCP de roadmap/contratos compartidos con la
foto vigente y el backlog vivo para que una IA no consuma estado historico como
pendiente actual.

Estado: completada 2026-05-24.

Alcance:

- `modulos/orquesta-mcp`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/estado_actual_2026-05-17.md`
- `docs/runbooks`

Criterios:

- Revisar `orquesta.project.roadmap.v0` y `orquesta.shared_contracts.v0` contra
  estado actual, backlog Txx y docs locales; marcar historico lo que ya no sea
  fuente vigente o enlazar a la fuente viva.
- Evitar duplicar estados `pendiente_*` hardcodeados si el backlog ya posee una
  tarea ejecutable; el recurso MCP debe exponer refs canonicas y freshness.
- Si una entrada sigue pendiente, debe tener owner, alcance, prueba focal y
  relacion clara con una seccion Txx o doc local.
- Tests: `go test -count=1 ./modulos/orquesta-mcp`.

Evidencia de cierre:

- `orquesta.project.roadmap.v0` y `orquesta.contracts.shared.v0` publican
  `freshness` con refs a foto vigente, backlog T25 y verificacion focal.
- Los `progress_key` y estados ya no duplican `pendiente_*` de la foto antigua;
  lo historico queda como compatibilidad y lo abierto enlaza backlog/doc local.
- OPES temporal de derivados/cierre queda visible como frente abierto separado
  con owner, guardas y runbook; no reabre Codex wave/recursion ni WaitAgentRefs.
- Cierre verificado en `modulos/orquesta-mcp` con proyeccion estatica:
  `resource_freshness_v0.go`, `project_roadmap_*_v0.go`,
  `shared_contracts_*_v0.go` y docs locales MCP. La validacion focal de cierre
  es `go test -count=1 ./modulos/orquesta-mcp`.

## T26 domain-work-quality-policy-port

Objetivo: mover las reglas de calidad de artefactos `domain_work` a una politica
por puerto/adaptador con matriz externa, sin bloquear contenido valido por
strings genericas.

Estado: cerrado 2026-05-24 para bloqueo estricto offline por snapshot/worktree
en runtime-worktree, review gate de autoprogramacion y stack Codex.

Alcance:

- `modulos/orquesta-domain-work`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-opes-bridge`
- `docs/runbooks`

Criterios:

- Separar reglas genericas de artefacto causal de reglas de dominio como
  placeholders, extension minima, paquetes de tema o criterios editoriales.
- Mantener tolerancia a alias y vocabulario normal; cortar solo placeholders
  efectivos, artefactos incompletos demostrables o validaciones de dominio
  fallidas.
- La politica debe devolver issues estructurados con campo causal para review,
  rework o replan, no solo `domain_work_artifact_quality_gate_failed`.
- Tests: `go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-app-codex-stack ./modulos/orquesta-opes-bridge`.

## Escaneo backlog 2026-05-24 sexta pasada

Evidencia revisada: `AGENTS.md`, `README.md`,
`docs/estado_actual_2026-05-17.md`,
`docs/guia_nucleo_orquestacion_2026-05-17.md`,
`docs/principio_orquesta_piensa_director.md`,
`docs/corte_cierre_generico_director_operativo_2026-05-17.md`,
`docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`, matriz vigente de
autoprogramacion, `cmd/orquesta-server/startup_check*.go`,
`modulos/orquesta-runtime/docs/contratos.md`,
`modulos/orquesta-runtime-codex/docs/contratos.md`, rail errors y duplicaciones
de rails dentro del write-set permitido. No se programa codigo desde este
scanner. `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos o reencuadrados:

- Hay deriva entre fuentes de verdad que leen agentes: `AGENTS.md` aun presenta
  como pendiente la recursion Codex productiva completa, mientras
  `docs/estado_actual_2026-05-17.md`, la guia y la matriz ya declaran
  `CODEX-WAVE-REAL` y `CODEX-RECURSION-REAL` cerrados. Esa divergencia puede
  relanzar smokes reales caros o priorizar mal el unico frente abierto real:
  OPES temporal de derivados/cierre.
- `DAEMON-RESTART` cubre cola file-based sin agentes reales y el arranque tiene
  reconciliacion/compactacion local, pero la matriz todavia dice que no valida
  rehidratacion de procesos Codex vivos. El contrato runtime tambien deja
  `resume` fuera de `launch_mode`. Falta un caso de reinicio con agentes vivos
  que no cierre ni archive por heuristicas locales.
- `USAGE-METRICS-OFFLINE` cubre telemetry fake y `USAGE-METRICS-REAL` observa
  stats durante una run, pero la matriz mantiene pendiente el conector productivo
  de cuota/tokens por proveedor. Sin puerto explicito, el coste real puede quedar
  mezclado con stats, auditoria o detalles de proveedor.

## T27 docs-source-of-truth-state-sync

Objetivo: sincronizar las fuentes canonicas que consumen agentes y operadores
para que el estado vigente no contradiga el backlog ni relance trabajo real ya
cerrado.

Estado: completada el 2026-05-24.

Alcance:

- `AGENTS.md`
- `README.md`
- `docs/estado_actual_2026-05-17.md`
- `docs/guia_nucleo_orquestacion_2026-05-17.md`
- `docs/corte_cierre_generico_director_operativo_2026-05-17.md`
- `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`

Criterios:

- Definir un orden explicito de autoridad documental para foto vigente, backlog
  ejecutable, matriz de smokes y docs historicos.
- Actualizar las referencias a `CODEX-WAVE-REAL` y `CODEX-RECURSION-REAL` para
  que no aparezcan a la vez como cerradas y pendientes en documentos de entrada
  obligatoria.
- Mantener OPES temporal real de derivados/cierre como frente abierto separado,
  sin reabrir `WaitAgentRefs`, ola/cohorte Codex ni recursion Codex salvo
  regresion demostrada.
- Marcar documentos historicos con enlace a fuente vigente antes de usarlos como
  evidencia de planificacion.
- Tests: `git diff --check` y prueba documental focal que busque contradicciones
  conocidas de `CODEX-WAVE-REAL`, `CODEX-RECURSION-REAL` y OPES pendiente.

Evidencia de cierre:

- `AGENTS.md`, `README.md`, la foto vigente, la guia, el corte de cierre y la
  matriz declaran un orden de autoridad documental.
- `CODEX-WAVE-REAL` y `CODEX-RECURSION-REAL` quedan cerrados por evidencia
  opt-in de la matriz; no son backlog abierto salvo regresion demostrada.
- OPES temporal real de derivados/cierre queda como frente abierto separado.
- Los documentos historicos quedan subordinados a fuentes vigentes antes de
  usarse como evidencia de planificacion.

## T28 restart-live-agent-reconciliation

Objetivo: cerrar recuperacion de servidor tras reinicio con agentes reales vivos
o parados, sin depender de heuristicas de texto ni esperar todos los agentes del
run.

Estado: cubierto 2026-05-24 para el perfil Codex estricto en servidor/stack:
el sandbox invalido ya no se normaliza antes de validar, `approval_policy`
interactivo queda bloqueado salvo opt-in explicito de operador vivo y el
perfil/prompt declaran la ubicacion del `runtime_work_dir` como control interno
o writable root externo.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-runtime`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-agent-process-registry`
- `modulos/orquesta-run-control`
- `modulos/orquesta-run-queue`
- `modulos/orquesta-state-file`
- `docs/runbooks`

Criterios:

- Definir contrato de resume/reconciliacion para agentes existentes: descriptor,
  `run_ref`, `agent_ref`, wait scope, `agent_ack.json`, checkpoint y estado del
  proceso.
- Tras reinicio, reconstruir cola/control/plan state desde stores y runtime
  descriptors sin usar `codex_last_message.txt` como cierre suficiente.
- Si un agente sigue vivo, conservar espera acotada por `WaitAgentRefs`; si esta
  parado sin ACK, aplicar politica explicita de lost/stopped/retry sin cerrar por
  summary textual.
- El smoke real debe ser opt-in, con agente Codex temporal, shutdown controlado
  del servidor, reinicio y entrega/review/cierre o bloqueo causal verificable.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./modulos/orquesta-state-file`.

## T29 provider-usage-quota-accounting

Objetivo: convertir uso real de proveedor y cuota en puerto/adaptador explicito,
observable y redactado, sin meter proveedor/modelo/coste en el nucleo.

Estado: cubierto 2026-05-24 r2 para `orquesta-runtime-worktree`,
`orquesta-runtime-codex`, `orquesta-app-codex-stack` y `cmd/orquesta-server`.

Alcance:

- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-runtime`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-mcp`
- `modulos/orquesta-web`
- `cmd/orquesta-server`
- `docs/runbooks`

Criterios:

- Separar stats operativas (`progress`, `process_refs`, `agents_observed`) de
  uso/coste de proveedor por puerto opt-in.
- No persistir prompt, transcript, tokens crudos, cuenta real, HOME ni payloads
  de proveedor en eventos, auditoria, ACKs o docs de resultado.
- MCP/web/API deben exponer uso solo con `include_agent_usage` o equivalente y
  errores publicos/i18n; si no hay puerto productivo, devolver estado no
  disponible sin bloquear el director.
- El smoke real debe tener confirmacion explicita y limites bajos; la bateria
  rapida debe poder validar telemetry fake sin proveedor.
- Tests: `go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp ./modulos/orquesta-web ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 septima pasada

Evidencia revisada: paquete del scanner de automejora, `AGENTS.md`, `README.md`,
este backlog, rail errors, duplicaciones de rails,
`docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`,
`docs/runbooks/apagado_controlado_servidor_y_agentes.md`,
`docs/politica_supervision_residente_event_driven.md`,
`modulos/orquesta-server-shutdown`, `modulos/orquesta-runtime-codex`,
`modulos/orquesta-runtime`, `modulos/orquesta-run-queue`,
`modulos/orquesta-run-supervisor`, `modulos/orquesta-server` y
`cmd/orquesta-server`. No se programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos o reencuadrados:

- El apagado cooperativo ya tiene contrato de servidor y caso Codex real, pero
  el protocolo de checkpoint vivo sigue materializado como ficheros/prompt
  Codex. La propia matriz declara pendientes otros runtimes/proveedores.
- La cola global ordena candidatos y permite prioridad, pero no posee reserva o
  lease de ejecucion. Si dos supervisores residentes o ticks solapados leen el
  mismo candidato ejecutable, la prevencion de doble launch queda repartida en
  capas superiores en vez de ser contrato de cola.
- La politica documental pide supervisor residente event-driven, pero el loop
  actual del servidor sigue gobernado por `time.NewTicker` con auditoria de tick
  y checks idle. Falta un puerto de wakeup/debounce para reaccionar a eventos
  reales sin reinyectar trabajo por pulsos ciegos.

## T30 runtime-shutdown-checkpoint-port

Objetivo: elevar el checkpoint cooperativo de apagado a contrato neutral de
runtime/servidor, con Codex como adaptador existente y otros runtimes como
implementaciones futuras opt-in.

Estado: cerrada 2026-05-25 para lectura terminal estricta sin fallback legacy
en observacion, startup y planner de automejora.

Alcance:

- `modulos/orquesta-runtime`
- `modulos/orquesta-server-shutdown`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/runbooks`

Criterios:

- Definir puerto/DTO neutral para preparar request de shutdown, observar ACK de
  checkpoint, distinguir `checkpoint_ready`, `checkpoint_pending`,
  `checkpoint_not_supported` y deadline expirado.
- Codex debe implementar ese puerto usando los ficheros de control actuales sin
  convertir `orquesta_shutdown_request.json` ni
  `agent_shutdown_checkpoint_ack.json` en artefactos de producto.
- `forced=false` no puede cerrar como listo si un runtime declara soporte de
  checkpoint y falta ACK correlado; si el runtime no soporta checkpoint, debe
  quedar razon publica y politica explicita de stop/deadline.
- Resultados y auditoria deben transportar refs opacas, estado y evidencia
  compacta; no persistir HOME, rutas reales, prompts, transcripts, tokens ni
  payloads de proveedor.
- Tests: `go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-server-shutdown ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

## T31 run-queue-reservation-lease

Objetivo: anadir reserva/lease de ejecucion a la cola global para que ranking y
prioridad no basten como coordinacion cuando hay supervisores, ticks o procesos
residentes concurrentes.

Estado: cerrado 2026-05-25.

Alcance:

- `modulos/orquesta-run-queue`
- `modulos/orquesta-run-memory`
- `modulos/orquesta-run-file`
- `modulos/orquesta-run-supervisor`
- `modulos/orquesta-server`
- `cmd/orquesta-server`

Criterios:

- Definir puerto neutral de claim/reserva con `run_ref`, `queue_ref`,
  `lease_ref`, owner opaco, TTL, idempotency key, correlation id y evidencia.
- `RunSupervisorV0` debe reservar antes de lanzar una run; un conflicto de
  reserva produce skip causal y no segundo launch.
- Stores memory/file deben implementar claim atomico, release/renewal y
  expiracion recuperable tras reinicio sin usar PID, host, ruta local ni Git
  como verdad del nucleo.
- Cambios de estado terminal (`closed`, `delivered`, `stopped`, `canceled`)
  deben limpiar o invalidar reservas vivas de forma idempotente.
- La automejora idle no debe preparar fallback ni scanner si ya existe una run
  reservada o visible para el mismo frente.
- Tests: `go test -count=1 ./modulos/orquesta-run-queue ./modulos/orquesta-run-memory ./modulos/orquesta-run-file ./modulos/orquesta-run-supervisor ./modulos/orquesta-server ./cmd/orquesta-server`.

## T32 resident-supervisor-event-driven-wakeup

Objetivo: consolidar el supervisor residente hacia wakeups por eventos reales,
manteniendo el tick como fallback de salud y no como fuente primaria de
reinstruccion.

Estado: completada por Orquesta en
`request-ref-autoprogramming-backlog-t47-external-agent-required-test-receipts-3ea943a0`.

Alcance:

- `modulos/orquesta-server`
- `modulos/orquesta-run-supervisor`
- `modulos/orquesta-run-queue`
- `cmd/orquesta-server`
- `docs/runbooks`

Criterios:

- Definir puerto/proyeccion de wakeup para eventos como candidato de cola
  creado, prioridad cambiada, delivery registrada, checkpoint de shutdown,
  cierre de run o capacidad liberada.
- Debounce/coalescing por `run_ref`, `queue_ref` y causa para evitar ticks
  solapados, reinyeccion de guidance y automejora duplicada.
- El tick periodico queda como health check y watchdog con razon durable, no
  como sustituto de eventos observables.
- Auditoria y status deben explicar si el supervisor actuo por evento, tick,
  cooldown, lease/reserva, cola vacia o skip causal.
- Payloads de wakeup deben ser compactos y saneados: refs opacas, causa y
  contadores; sin prompts, transcripts, rutas locales, HOME, tokens ni payloads
  HTTP completos.
- Tests: `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-run-supervisor ./modulos/orquesta-run-queue ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 octava pasada

Evidencia revisada: paquete del scanner de automejora, `AGENTS.md`, `README.md`,
este backlog, rail errors, duplicaciones de rails,
`cmd/orquesta-server/idle_self_improvement_backlog_planner_v0.go`,
`cmd/orquesta-server/startup_check_compaction.go`,
`cmd/orquesta-server/config.go`, `cmd/orquesta-server/codex_director_wave_command_v0.go`,
`modulos/orquesta-runtime-codex/codex_ack_validation_v0.go` y docs locales de
`orquesta-run-control`. No se programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos o reencuadrados:

- El planner de backlog ya lee `agent_ack.json` para no relanzar requests
  completadas, pero el cierre documental actual acepta cualquier JSON bajo
  `.orquesta-runtime` con `status=completed`. No valida `schema_version`,
  `request_id`, `correlation_id`, `ack_ref`, `task_ref`, pruebas obligatorias ni
  que el ACK pertenezca a la request normalizada. Esto puede saltar una seccion
  pendiente por un ACK incompleto, antiguo o de otro agente.
- El comando de ola Codex del Director tiene modo estricto, pero por defecto
  queda desactivado. En modo no estricto rellena `write_set=.` y
  `operator-validation-required`, y el prompt permite tocar ficheros fuera del
  write-set asignado si el agente lo justifica. Para ejecuciones reales de
  autoprogramacion, ese rail debe ser opt-in inverso: estricto por defecto o
  rechazo temprano cuando falten write-set/tests concretos.
- El arranque del servidor fija `ORQUESTA_STARTUP_CLEANUP_MODE=forced_stop` si
  no hay valor explicito. Hasta cerrar reconciliacion de agentes vivos, ese
  default puede convertir una reanudacion ambigua en parada logica y compactar
  cola/control antes de probar ACK estructurado, checkpoint o descriptor de
  runtime.

## T33 autoprogramming-backlog-ack-correlation

Objetivo: usar ACK estructurado y correlado como fuente de verdad para decidir
que una request de backlog/automejora ya esta completada.

Estado: cerrado 2026-05-24.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-app-codex-stack`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`

Criterios:

- El planner no debe marcar completada una request solo por
  `status=completed`; debe validar schema, request/correlation/ack/task refs,
  target module, pruebas obligatorias y ausencia de issues de ACK.
- Un ACK incompleto, corrupto, de otra request o sin pruebas requeridas debe
  dejar la seccion visible y producir evidencia compacta de skip/retry, no
  cerrar ni ocultar el trabajo.
- Reutilizar el validador de `orquesta-runtime-codex` o un adaptador equivalente
  por puerto; no duplicar una tercera heuristica textual distinta de T24.
- Si no hay `agent_packet.json` o spec suficiente para correlacion fuerte, el
  planner debe tratar el ACK como ambiguo y preferir tarea de revision
  documental acotada antes que asumir completado.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack`.

Cierre aplicado 2026-05-24:

- `orquesta-runtime-codex` expone validacion estricta de ACK completado:
  no hidrata refs ausentes ni normaliza correlacion para cierre terminal.
- El planner de backlog de `cmd/orquesta-server` solo salta una seccion cuando
  `agent_ack.json` tiene `agent_packet.json` vecino, correlaciona con
  request/agent runtime, valida schema, refs, target module, task ref, tests
  requeridos y queda sin issues del validador Codex.
- ACK ausente de packet, minimo, corrupto o ambiguo conserva la seccion visible
  y anade evidencia compacta `evidence-ref-autoprogramming-backlog-ack-ambiguous`.

## T34 director-wave-strict-guards-default

Objetivo: hacer que las olas Codex del Director no arranquen con write-set
global, pruebas placeholder o permiso implicito de ampliar alcance salvo opt-in
explicito y auditable.

Estado: completado.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-director-operativo`
- `modulos/orquesta-app-codex-stack`
- `docs/runbooks`

Criterios:

- `codex-director-wave` debe exigir write-set y tests concretos para ejecucion
  real, o activar modo estricto por defecto cuando no sea dry-run.
- Si un operador quiere `write_set=.` o tests placeholder, debe declararlo como
  opt-in con razon, evidence refs y guardas de no borrar/no salir del proyecto.
- Los prompts de agente y subagente deben distinguir ownership de shard frente
  a alcance autorizado total: el shard puede ser flexible dentro del write-set
  global, pero no fuera de el sin decision del director.
- La delegacion recursiva debe conservar parent/child refs, presupuesto y ACK
  compacto sin permitir que un hijo amplie write-set por texto libre.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-director-operativo ./modulos/orquesta-app-codex-stack`.

## T35 startup-cleanup-safe-mode

Objetivo: cambiar el arranque del servidor para que la purga/compactacion de
runs transitorias sea segura por defecto y no haga `forced_stop` sin evidencia
de reconciliacion.

Estado: cerrado 2026-05-25.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-run-control`
- `modulos/orquesta-runtime`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-state-file`
- `docs/runbooks`

Criterios:

- Si `ORQUESTA_STARTUP_CLEANUP_MODE` no esta definido, el servidor debe preferir
  bloqueo/diagnostico o reconciliacion conservadora antes que marcar runs como
  `stopped`.
- `forced_stop` debe quedar como opt-in explicito con evidencia y razon publica,
  nunca como default silencioso mientras existan agentes vivos o ACK ambiguos.
- La compactacion debe consultar ACK estructurado, run control, descriptors de
  runtime y wait scope antes de purgar cola/control; no usar summaries ni
  ausencia transitoria de pending como cierre.
- El resultado de startup debe exponer contadores de kept/blocked/compacted y
  una causa por run para que el operador pueda reintentar o pedir shutdown
  cooperativo.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-run-control ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-state-file`.

## Escaneo backlog 2026-05-24 novena pasada

Evidencia revisada: paquete del scanner de automejora, `AGENTS.md`, `README.md`,
este backlog, rail errors, duplicaciones de rails,
`cmd/orquesta-server/idle_self_improvement_backlog_planner_v0.go`,
`cmd/orquesta-server/idle_self_improvement_backlog_planner_v0_test.go`,
`cmd/orquesta-server/startup_check_compaction.go`,
`modulos/orquesta-runtime-codex/codex_ack_validation_v0.go`,
`modulos/orquesta-runtime-codex/codex_ack_v0.go`,
`modulos/orquesta-runtime-codex/codex_delivery_observation_v0.go` y sus tests.
No se programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos o reencuadrados:

- El validador Codex de ACK completa `request_id`, `correlation_id`, `ack_ref`,
  `target_module` y `task_ref` desde la spec antes de validar. Eso conserva
  compatibilidad para ACKs legacy, pero no sirve como fuente estricta para cierre
  terminal, compactacion de arranque o planner de backlog: un
  `{"schema_version":"codex_agent_ack.v0","status":"completed"}` puede producir
  observacion correlada si la spec trae defaults.
- El parser del backlog solo propaga required tests que empiezan por `go test`
  y marca completada una seccion por textos amplios como `revalidacion final`.
  Las secciones que piden `git diff --check`, `bash -n scripts/*.sh`,
  `./scripts/test_rails_fast.sh` o una prueba documental focal pueden salir con
  un contrato de tests incompleto o quedar saltadas por narrativa no canonica.
- `codex_delivery_observation_v0.go` conserva un rail local/test-only para
  detectar vocabulario como `codex`, `runtime`, `provider`, `db` o `git` en
  observaciones. No bloquea produccion hoy, pero duplica la politica de
  `orquesta-rails` y deja ambiguo si la observacion debe sanearse por campo o
  solo registrarse como candidato pendiente.

## T36 codex-ack-strict-terminal-validation

Objetivo: separar lectura tolerante de ACKs legacy de validacion estricta de ACK
terminal para observacion, cierre, startup y planner de automejora.

Estado: cubierto 2026-05-26 r5.

Alcance:

- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`

Criterios:

- Definir modo estricto que rechace ACK `completed` si el JSON no trae
  explicitamente schema, request/correlation/ack/task refs, target module,
  status, files/tests requeridos y ausencia de issues.
- Conservar la normalizacion con defaults solo para lectura diagnostica o
  compatibilidad legacy, nunca para declarar delivery terminal, purga de startup
  o backlog completado.
- `ReadCodexDeliveryObservationFileV0`, startup compaction y planner de backlog
  deben elegir explicitamente modo estricto o diagnostico, con tests que prueben
  que el ACK minimo `{"schema_version":"codex_agent_ack.v0","status":"completed"}`
  no cierra trabajo nuevo.
- La evidencia de tests fallidos en objetos o strings sigue bloqueando
  `completed`; no basta con que el comando requerido aparezca como texto.
- Tests: `go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

Evidencia 2026-05-24:

- `ReadCodexDeliveryObservationFileV0` usa validacion estricta y rechaza el ACK
  minimo hidratable.
- Startup compaction ya no purga una cola `ready` por texto terminal en
  `codex_last_message.txt`; exige `agent_packet.json` vecino y
  `agent_ack.json` completado, correlado y con files/tests explicitos.
- `completedBacklogRequestRefsV0` conserva la validacion estricta con
  `agent_packet.json` vecino antes de marcar requests como completados.

Evidencia 2026-05-25:

- `ReadCodexDeliveryObservationFileV0` ya no degrada paquetes con policy
  terminal estricta a lectura legacy cuando faltan `test_receipts`; la
  compatibilidad legacy queda limitada a paquetes sin policy estricta.

Evidencia 2026-05-26:

- Revalidado en codigo que `ReadCodexDeliveryObservationFileV0` usa solo
  `ValidateStrictCompletedCodexAgentAckBytesForSpecV0` cuando el packet trae
  `write_set_closed`/policy estricta. La lectura tolerante queda como camino
  legacy diagnostico para packets sin policy estricta y no declara entrega
  terminal OrquestaV2.
- Rework `agent-ref-task-ref-review-rework-task-autoprogramming-e44e32f96f05-g01-a6215dfd6ac4`:
  entrega acotada sin relanzar el padre original; se conserva el cierre de T36,
  se reejecuta la prueba obligatoria focal y el contexto `required ref_only`
  queda resuelto por `ack_evidence_required` en el ACK tras lectura local del
  packet y docs locales.

## T37 autoprogramming-backlog-parser-fidelity

Objetivo: hacer que el planner de backlog preserve estado y verificacion
declarados en Markdown sin heuristicas narrativas ni perdida de comandos no-Go.

Estado: completada 2026-05-24. El planner preserva los comandos declarados en
`Tests:` en orden, incluidos `git diff --check`, scripts y comandos no-Go
ejecutables; las verificaciones documentales quedan como criterios/context refs
de verificacion manual sin reemplazarse por el test base; y las secciones sin
`Estado:` canonico no se ocultan por narrativa ambigua como revalidaciones.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`

Criterios:

- Una seccion Txx solo queda cerrada por `Estado:` canonico o evidencia
  estructurada/correlada; frases como `revalidacion final` no deben ocultarla por
  si solas.
- El parser debe transportar comandos de `Tests:` completos y ordenados,
  incluidos `git diff --check`, scripts locales y pruebas documentales
  declaradas, no solo comandos que empiezan por `go test`.
- Si una seccion declara una prueba no ejecutable automaticamente, el planner
  debe marcarla como verificacion/manual-blocker publica en vez de eliminarla o
  sustituirla por el test base.
- Los tests deben cubrir secciones con varios comandos unidos por `y`, scripts,
  `git diff --check`, prueba documental focal y estado ambiguo.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.

## T38 codex-delivery-observation-rail-owner

Objetivo: decidir propietario y uso real del rail de observaciones Codex para no
duplicar listas de detalle prohibido ni bloquear vocabulario operativo opaco.

Estado: cerrado el 2026-05-25 para codigo productivo de los cuatro ficheros
objetivo; la deuda residual de line-count del modulo queda fuera de este shard
si pertenece a otros ficheros historicos o tests.

Alcance:

- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-rails`
- `modulos/orquesta-app-codex-stack`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Clasificar `codexDeliveryObservationUnsafeForCoreV0`: eliminarlo si solo es
  detector historico de test, o moverlo a una politica comun por campo si debe
  proteger produccion.
- Las observaciones deben permitir refs opacas y vocabulario operativo
  (`codex`, `runtime`, `provider`, `git`, `db`, `sql`) cuando no contienen valor
  sensible efectivo.
- Secretos, HOME/rutas privadas, prompts/transcripts y payloads de proveedor
  siguen bloqueados o redactados por helper comun, con matriz externa.
- La matriz de rails debe cubrir ACK, delivery observation y pending rail notes
  sin listas locales divergentes en runtime Codex.
- Tests: `go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-rails ./modulos/orquesta-app-codex-stack`.

## Escaneo backlog 2026-05-24 decima pasada

Evidencia revisada: paquete del scanner de automejora, `AGENTS.md`, `README.md`,
este backlog, rail errors, duplicaciones de rails,
`modulos/orquesta-mcp/app_vcs_tool_v0.go`,
`modulos/orquesta-app-codex-stack/app_vcs_executor_v0.go`,
`modulos/orquesta-runtime-worktree/app_vcs_git_v0.go`,
`modulos/orquesta-runtime-worktree/staging_promotion_v0.go`,
`modulos/orquesta-app-codex-stack/autoprogramming_staging_promotion_v0.go`,
`cmd/orquesta-server/autoprogramming_promotion_v0.go`,
`modulos/orquesta-server/audit_v0.go` y
`modulos/orquesta-server/supervisor_loop_v0.go`. No se programa codigo desde
este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos o reencuadrados:

- `orquesta.app_vcs.v0` ya expone `commit`/`push` por MCP/HTTP fino y el stack
  lo cablea al `ProjectWorkDir` de la composicion. El conector Git usa
  `git add -A` si `commit_paths` viene vacio, y el contrato MCP no transporta
  `write_set`, run causal, cierre/review aceptada ni evidencia de tests. Es una
  superficie de efecto externo distinta del rail de promocion T13 y necesita
  guardas propias antes de quedar disponible para operadores o agentes.
- La promocion de automejora ya tiene piezas de codigo opt-in
  (`AutoprogrammingStagingPromotion*`, `GitStagingPromotionConnectorV0` y wiring
  de servidor), pero el backlog T13 sigue describiendo el rail completo como
  pendiente. Falta cerrar la foto documental y un e2e acotado con repo temporal
  que demuestre cierre causal -> promocion -> archivo -> terminalidad de cola
  con recibo durable, sin depender solo de fake port.
- La auditoria JSONL del servidor acepta `payload any` y varios eventos del
  supervisor guardan requests/results completos de automejora. T19 cubre la
  superficie de auditoria general, pero falta contrato por evento para redaccion
  de payloads internos antes de ampliar visores, payload HTTP completo o
  automejora residente con mas conectores.

## T39 app-vcs-write-set-evidence-guards

Objetivo: cerrar las guardas de efecto externo de `orquesta.app_vcs.v0` para que
commit/push no puedan capturar cambios fuera de alcance ni publicarse sin
evidencia causal.

Estado: cerrado 2026-05-25 para el corte acotado de comandos y drain Codex.

Alcance:

- `modulos/orquesta-mcp`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-runtime-worktree`
- `cmd/orquesta-server`
- `docs/runbooks`

Criterios:

- `commit` debe exigir `commit_paths` o `write_set` explicito; `git add -A`
  solo puede quedar como opt-in auditado con razon y scope acotado.
- `commit`/`push` deben transportar refs causales suficientes: request/run,
  review aceptada o decision de operador, pruebas requeridas y evidencia de
  ausencia de solape cuando aplique.
- `push` requiere opt-in de composicion y de request; no basta `allow_push=true`
  si el servidor no declara guardas de remoto, rama y ventana operativa.
- Los errores de Git deben redactar rutas privadas, remotos con credenciales y
  salida sensible antes de volver por MCP/HTTP, audit logs o ACKs.
- Tests: `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-worktree ./cmd/orquesta-server`.

## T40 autoprogramming-promotion-real-e2e-and-doc-state

Objetivo: cerrar el estado real del rail T13 con evidencia end-to-end acotada y
documentacion sincronizada, sin convertir refs de worktree/branch en rutas Git
del nucleo.

Estado: completada 2026-05-24.

Evidencia:

- `TestCodexStackAutoprogrammingPromotionV0E2ERepoTemporalReplayV0` crea un repo
  Git temporal, siembra una run de autoprogramacion cerrada con review aceptada
  y `RequiredTestEvidenceV0` passed, ejecuta la promocion opt-in del stack sobre
  `GitStagingPromotionConnectorV0`, verifica commit solo de `feature.md`,
  archivo idempotente y replay sin nuevo commit ni nuevo manifest.
- `TestCodexStackAutoprogrammingPromotionV0NoCierraColaConEfectoIncompletoV0`
  cubre que `pending_push` o archivo bloqueado no devuelven
  `queue_status=closed`, de modo que el retry queda vivo con evidencias
  compactas.
- Validacion requerida:
  `go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

Pendiente fuera de este cierre: push remoto/productivo con guardas de remoto,
rama y ventana operativa; queda delegado a T39 y a un opt-in posterior.

Alcance:

- `modulos/orquesta-autoprogramming`
- `modulos/orquesta-runtime-worktree`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`
- `docs/runbooks`

Criterios:

- Probar con repo temporal que una run de automejora cerrada causalmente,
  con review aceptada y `RequiredTestEvidenceV0` passed, dispara promocion
  opt-in, crea commit solo dentro del write-set y genera archivo idempotente.
- La cola/run no debe quedar como cerrada promocionada si promocion, push o
  archivo queda `pending`, `blocked` o `pending_push`; debe exponer retry
  compacto y evidence refs.
- La prueba debe cubrir replay: repetir el tick no duplica commits, archivos ni
  efectos de cola.
- Actualizar T13, matriz y runbook para distinguir piezas cerradas, e2e cerrado
  y pendientes reales de push remoto/productivo.
- Tests: `go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

## T41 server-audit-payload-redaction-contract

Objetivo: definir un contrato de payload auditado por evento para que la
auditoria JSONL del servidor no persista material crudo de automejora,
supervision, HTTP, proveedor o dominio.

Estado: cerrado 2026-05-25 tras rework de revision; queda pendiente solo crear
followups de particion concretos cuando el rail detecte crecimiento real.

Alcance:

- `modulos/orquesta-server`
- `modulos/orquesta-observability`
- `cmd/orquesta-server`
- `docs/auditoria_runtime_orquesta_2026-05-24.md`
- `docs/runbooks`

Criterios:

- `auditEventV0` debe recibir payloads ya normalizados o pasar por sanitizador
  por campo antes de escribir JSONL; `map[string]interface{}` libre no debe ser
  contrato estable.
- Los eventos internos de automejora/supervisor deben guardar refs, contadores,
  estados y errores publicos, no prompts, transcripts, payload HTTP completo,
  rutas privadas, tokens, remotos Git con credenciales ni cuerpos de dominio.
- La captura de payload HTTP completo sigue apagada por defecto; cualquier opt-in
  debe tener retencion corta, redaccion probada y marca publica de riesgo.
- La observabilidad debe consumir un esquema compacto o proyeccion, no inferir
  privacidad desde cuerpos crudos de auditoria.
- Tests: `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-observability ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 undecima pasada

Evidencia revisada: paquete del scanner de automejora, `AGENTS.md`,
`README.md`, docs vigentes de estado/nucleo/cierre/smokes, este backlog, rail
errors, duplicaciones de rails,
`modulos/orquesta-runtime-codex/codex_ack_validation_v0.go`,
`modulos/orquesta-runtime-codex/codex_ack_path_policy_v0.go`,
`cmd/orquesta-server/idle_self_improvement_backlog_planner_v0.go`,
`modulos/orquesta-context/context_materialization_v0.go` y
`modulos/orquesta-orchestration-core/runtime_context_materializer.go`. No se
programa codigo desde este scanner. `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
solo como refs opacas.

Huecos concretos nuevos o reencuadrados:

- La validacion de `agent_ack.json` sigue teniendo modo compatible que hidrata
  identidad desde la spec y acepta `files` fuera del write-set como rail blando.
  Ese comportamiento sirve para entregas legacy, pero no basta para agentes
  externos gobernados con rail estricto `write_set_closed`: un ACK terminal no
  debe cerrar si declara archivos de control, globs, rutas no tocadas o cambios
  fuera de alcance sin `CONSULTA AL DIRECTOR`.
- El planner de backlog lanza scanners con el mismo write-set documental y lee
  ACKs completados desde `.orquesta-runtime`, pero no existe contrato explicito
  de reserva/merge para evitar que dos scanners de burst anadan Txx duplicados o
  trabajen sobre una foto documental ya cambiada.
- El paquete real de esta pasada trae una entrada de contexto requerida como
  `doc_ref` en modo `ref_only` y `total_bytes=0`. El contrato general permite
  `ref_only`, pero para tareas de scanner que dependen de documentos concretos
  falta una guarda que fuerce materializacion, evidencia de lectura o bloqueo
  `CONSULTA AL DIRECTOR` antes de cerrar como completado.

## T42 codex-ack-strict-write-set-terminal-proof

Objetivo: separar compatibilidad legacy de modo terminal estricto para ACKs de
agentes gobernados por OrquestaV2, de forma que `completed` solo cierre cuando
la evidencia de archivos/pruebas respeta el paquete y el write-set.

Estado: cerrado 2026-05-25.

Alcance:

- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-runtime-worktree`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`

Criterios:

- Introducir modo estricto activado por policy/packet para agentes externos
  gobernados: `files` debe listar archivos reales de producto, concretos, sin
  globs/directorios, sin archivos de control y dentro del write-set salvo
  `status=failed` con `CONSULTA AL DIRECTOR`.
- Conservar modo legacy/advisory para casos historicos donde `file_outside_write_set`
  no debe tirar entregas, pero no usarlo para cerrar colas de OrquestaV2.
- Si hay snapshot de worktree, validar que `files` corresponde a cambios reales
  tocados y que no existen cambios fuera de alcance; si no hay snapshot,
  bloquear cierre estricto con error publico recuperable.
- `tests` solo cuenta como pasado si coincide exactamente con required tests y
  no hay evidencia textual/estructurada de fallo; cuando exista
  `RequiredTestEvidenceV0`, preferir esa evidencia durable.
- Tests: `go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

Evidencia 2026-05-24:

- `orquesta-runtime-codex` separa ACK legacy de ACK terminal estricto por
  policy de packet (`write_set_closed`, `ack_terminal_strict` u
  `orquestav2_strict`): el modo estricto exige refs explicitas, `files`
  concretos dentro del write-set, sin rutas de control, y `tests` exactamente
  iguales a los requeridos.
- `orquesta-runtime-worktree` permite contrastar `ACK.files` contra snapshot:
  cada archivo declarado debe aparecer como cambio real y cada cambio del
  snapshot debe estar declarado; borrados y cambios fuera del write-set siguen
  bloqueando.
- La lectura legacy/advisory conserva compatibilidad para ACKs historicos sin
  policy estricta, pero la observacion terminal de paquetes OrquestaV2 usa el
  modo estricto.

Revalidacion 2026-05-25:

- `ReadCodexDeliveryObservationFileV0` no cae a modo legacy cuando el packet
  exige ACK terminal estricto; un ACK OrquestaV2 sin `test_receipts` queda
  bloqueado por `missing_required_test_receipt`, mientras los packets sin policy
  estricta conservan compatibilidad diagnostica.
- El cierre documental queda alineado con T47: strings en `ACK.tests` no bastan
  para cerrar entregas gobernadas si falta recibo estructurado o evidencia
  durable equivalente en el adaptador.

## T43 backlog-scanner-doc-merge-lease

Objetivo: evitar que scanners de automejora concurrentes con el mismo write-set
documental dupliquen Txx, oculten pendientes o escriban sobre una foto obsoleta
del backlog.

Estado: completado.

Evidencia focal 2026-05-24: el scanner de backlog transporta
`backlog_scan_epoch`, hashes/lineas de documentos leidos y refs de reserva de
write-set documental desde `cmd/orquesta-server` hacia
`orquesta-autoprogramming`; los ACK completados con foto documental obsoleta
quedan como `backlog_docs_changed_after_plan` y se exponen en el resultado del
planner para rebase/merge dirigido. La deduplicacion incorpora firma documental
de seccion y colisiones compactas de scanner/heading.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-server`
- `modulos/orquesta-autoprogramming`
- `modulos/orquesta-run-queue`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Cada request de scanner debe transportar `backlog_scan_epoch`, hash/linea de
  los documentos leidos y refs de reserva de write-set documental antes de
  lanzar agente.
- Si el backlog o rail docs cambian entre plan y ACK, el cierre debe quedar en
  estado rebase/merge pendiente, no `completed` silencioso.
- La deduplicacion debe comparar heading Txx, objetivo, alcance y evidencia, no
  solo status de ACK o request_ref.
- El supervisor debe exponer colisiones compactas para que el director cree
  followup de merge o descarte duplicado sin borrar contenido.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-autoprogramming ./modulos/orquesta-run-queue`.

## T44 required-context-ref-only-guard

Objetivo: impedir que una tarea se cierre como completada cuando su contexto
obligatorio llega solo como `ref_only` sin evidencia de materializacion, lectura
manual o consulta al director.

Estado: completado.

Evidencia focal 2026-05-25: el contrato vigente ya evita cerrar como
`completed` una tarea con contexto obligatorio en `ref_only` sin resolucion
explicita. `ContextMaterializedBundleV0` conserva `ref_only_reason` y
`required_ref_action`; `AgentStartPacketV0` publica
`required_ref_only_context_guard`; el stack Codex anade el test requerido
`validar contexto required ref_only mediante lectura local, consulta al director
o evidencia explicita`; el prompt declara la accion esperada por entrada; y la
validacion de ACK completed rechaza refs obligatorias no materializadas si falta
`contexto_ref_only_resuelto`. Revalidado con
`go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-orchestration-core ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

Revalidacion de rework 2026-05-25: el paquete de correccion trae contexto
obligatorio `ref_only` con accion `ack_evidence_required`. La resolucion queda
acotada a evidencia explicita en ACK tras leer `agent_packet.json`, AGENTS/README
locales y verificar las guardas existentes por `rg`, sin relanzar otro agente
padre ni ampliar write-set.

Alcance:

- `modulos/orquesta-context`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-runtime`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`

Criterios:

- `ContextMaterializedBundleV0` debe distinguir refs `required` que pueden ser
  `ref_only` por diseno de refs que debian materializarse y quedaron sin
  contenido.
- El prompt/packet debe declarar accion esperada para cada `required ref_only`:
  leer documento local, pedir `CONSULTA AL DIRECTOR`, o continuar solo si el
  ACK incluye evidencia `contexto_truncado_resuelto`/equivalente.
- Para scanners de backlog, los documentos de entrada obligatorios deben quedar
  materializados o tener evidencia explicita de lectura local antes de aceptar
  `completed`.
- No meter paths locales, HOME ni contenido largo en core; el reader/adaptador
  de composicion resuelve refs y conserva el nucleo con refs opacas.
- Tests: `go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-orchestration-core ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 duodecima pasada

Evidencia revisada: paquete del scanner de automejora, `AGENTS.md`,
`README.md`, docs vigentes de estado/nucleo/cierre/smokes, este backlog, rail
errors, duplicaciones de rails, `modulos/orquesta-autoprogramming`,
`cmd/orquesta-server/idle_self_improvement_backlog_planner_v0.go` y conteo
local de ficheros Go con `rg --files -g '*.go' cmd modulos | xargs wc -l`.
No se programa codigo desde este scanner. `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
solo como refs opacas.

Hueco concreto nuevo:

- El prompt de agentes exige mantener cada fichero Go por debajo de 300 lineas,
  pero `EvaluateAutoprogrammingReviewGateV0` trata `file_too_large` como rail
  advisory y la matriz externa comprueba que una entrega de 301..520 lineas
  sigue aceptada con followup. En el repo actual ya hay una base historica
  amplia por encima de 300 lineas, por lo que endurecer sin baseline bloquearia
  trabajo real; pero aceptar crecimiento nuevo en ficheros ya grandes convierte
  una regla operativa en consejo no verificable.

## T45 autoprogramming-go-file-line-budget-baseline

Objetivo: convertir el limite de 300 lineas por fichero Go en un contrato
verificable de autoprogramacion, sin bloquear de golpe la deuda historica ya
existente.

Estado: cerrado 2026-05-25.

Alcance:

- `modulos/orquesta-autoprogramming`
- `modulos/orquesta-runtime-worktree`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Crear una linea base de ficheros Go historicos por encima de 300 lineas y
  tratarla como deuda viva, no como permiso para seguir creciendo.
- En modo OrquestaV2 estricto, una entrega nueva no debe cerrar `completed` si
  crea un fichero Go sobre 300 lineas o aumenta un fichero ya sobre el baseline
  sin followup de particion aceptado por el director.
- La medicion debe salir del snapshot/worktree real cuando exista; el ACK no
  debe poder inventar `LineCount` para saltarse el rail.
- `file_too_large` puede seguir como advisory en modo legacy, pero el modo
  terminal estricto debe distinguir `legacy_advisory` de `strict_blocking`.
- El backlog debe poder generar followups de particion por fichero sin borrar
  historial ni exigir refactors masivos en el mismo agente.
- Tests: `go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

Cierre 2026-05-24:

- `orquesta-runtime-worktree` guarda `line_count` de ficheros Go en snapshots y
  `VerifyWorktreeWriteSetV0` puede activar `StrictGoLineBudget` para bloquear
  ficheros Go nuevos sobre 300 lineas o crecimiento sobre baseline historico,
  salvo followup de particion aceptado.
- `orquesta-autoprogramming` mantiene `file_too_large` como advisory legacy y
  anade bloqueo estricto `go_file_line_budget_strict_blocking` cuando la
  evidencia trae conteo real desde `snapshot`, `worktree` o `project_file`.
- `orquesta-app-codex-stack` envuelve el puerto de evidencia del review gate en
  modo estricto y puede enriquecer lineas base desde `WorktreeSnapshotStore`.
  `cmd/orquesta-server` lo expone por
  `ORQUESTA_REVIEW_GATE_STRICT_GO_LINE_BUDGET=1`.

Ajuste 2026-05-25:

- El modo legacy ya no se endurece solo porque el adaptador aporte
  `line_count_source` real; conserva `file_too_large` como advisory.
- El stack Codex en modo estricto revalida el snapshot/worktree y bloquea
  crecimiento o ficheros Go nuevos sobre 300 lineas aunque el ACK omita el
  fichero tocado.
- La correccion de revision no relanza la tarea original: conserva la entrega
  valida y deja el cierre documental alineado con el rail implementado.

## Escaneo backlog 2026-05-24 decimotercera pasada

Evidencia revisada: paquete del scanner de automejora, `AGENTS.md`,
`README.md`, este backlog, rail errors, duplicaciones de rails,
`modulos/orquesta-app-codex-stack/spec_task_v0.go`,
`modulos/orquesta-runtime-codex/codex_prompt_v0.go`,
`modulos/orquesta-runtime-codex/codex_ack_validation_v0.go`,
`modulos/orquesta-runtime-worktree/verify_v0.go` y
`modulos/orquesta-runtime-worktree/snapshot_v0.go`. No se programa codigo desde
este scanner. `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
solo como refs opacas.

Huecos concretos revisados:

- La contradiccion historica entre `write_set_closed`, prompt Codex y
  `programmingObjectiveV0` queda cerrada por T46: el objetivo ya exige alcance
  cerrado/`CONSULTA AL DIRECTOR`, el prompt declara precedencia del packet y
  `BuildAgentStartPacketV0` invalida frases que permitan ampliar write-set por
  simple justificacion en ACK.
- El ACK de Codex valida que `tests` contenga los comandos requeridos y detecta
  frases de fallo, pero no hay recibo estructurado de ejecucion para agentes
  externos estrictos. En tareas gobernadas por OrquestaV2, declarar el comando
  en `ACK.tests` no debe ser la unica prueba de que el test paso.
- `VerifyWorktreeWriteSetV0` detecta paths eliminados y cambios fuera de
  alcance, pero no existe politica terminal unificada para operaciones
  destructivas dentro del write-set: truncados fuertes, reemplazos masivos o
  renombres pueden quedar como cambio permitido aunque el paquete prohiba
  borrar, mover fuera o truncar archivos existentes sin decision del director.

## T46 agent-packet-write-set-precedence

Objetivo: eliminar contradicciones entre politicas `write_set_closed`, prompt
Codex y objetivo de tarea para que el alcance estricto tenga una unica
precedencia verificable.

Estado: cerrado 2026-05-25 para la contradiccion packet/prompt. T47 conserva el
recibo estructurado de tests y T48 conserva la prueba de efectos destructivos
como frentes separados.

Alcance:

- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-runtime`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Si `AgentStartPacketV0.policies` contiene `write_set_closed`, ninguna linea
  generada en `objective`, prompt o criterios debe autorizar tocar archivos
  fuera del write-set con una simple nota de ACK.
- La ampliacion de alcance solo puede viajar como decision explicita del
  director o policy opt-in distinta, con refs causales y estado `failed`/
  `CONSULTA AL DIRECTOR` si falta permiso.
- Las pruebas deben cubrir paquetes de autoprogramacion, rework y director wave
  para demostrar que la instruccion final no contiene reglas incompatibles.
- El modo legacy que permita alcance primario amplio debe quedar nombrado como
  compatibilidad y no mezclarse con paquetes OrquestaV2 estrictos.
- Tests: `go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime`.

Evidencia 2026-05-24:

- `programmingObjectiveV0` ya emite alcance cerrado y `CONSULTA AL DIRECTOR`
  ante falta de alcance, sin permiso textual de ampliar write-set por ACK.
- `BuildCodexAgentPromptV0` declara precedencia de `write_set_closed` sobre
  objetivo, criterios, hints y documentos locales; el modo sin policy queda
  nombrado como compatibilidad legacy.
- `BuildAgentStartPacketV0` invalida paquetes estrictos que conserven frases de
  ampliacion por simple justificacion en ACK.

Revalidacion 2026-05-25:

- `go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime`.
- Rework de revision `agent-ref-task-ref-review-rework-task-autoprogramming-e8eba749c154-g01-c80cde88cf66`:
  se conserva T46 cerrado, no se relanza un padre sobre la tarea original y el
  contexto obligatorio `ref_only` queda resuelto por lectura local de docs
  vigentes y recibo explicito en el ACK.

## T47 external-agent-required-test-receipts

Objetivo: hacer verificable la ejecucion de tests requeridos por agentes
externos estrictos, sin depender solo de strings en `ACK.tests`.

Estado: cerrado 2026-05-25 para el tramo ACK `test_receipts` ->
`RequiredTestEvidenceV0` -> cierre de la app base en stack Codex. Queda fuera de
este cierre cualquier smoke OPES temporal de derivados/cierre.

Alcance:

- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-runtime-required-test`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-orchestration-core`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`

Criterios:

- En modo OrquestaV2 estricto, `ACK.tests` debe coincidir con los comandos
  requeridos, pero el cierre terminal necesita ademas `RequiredTestEvidenceV0`
  causal o un recibo estructurado de ejecucion del runtime/adaptador.
- El recibo debe transportar comando exacto, status, exit code, refs de
  evidencia, hora/orden compactos y redaccion de stdout/stderr; no debe guardar
  prompts, transcripts, HOME, tokens ni rutas privadas.
- Si no existe runner ni recibo verificable, el ACK puede ser diagnostico, pero
  no debe cerrar `completed`; debe quedar bloqueo recuperable para reejecutar
  tests por puerto o pedir `CONSULTA AL DIRECTOR`.
- La compatibilidad legacy puede seguir aceptando strings para observacion, pero
  el planner, startup compaction y cierre estricto deben preferir evidencia
  durable.
- Tests: `go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-required-test ./modulos/orquesta-app-codex-stack ./modulos/orquesta-orchestration-core ./cmd/orquesta-server`.

Evidencia 2026-05-24:

- `codex_agent_ack.v0` admite `test_receipts` estructurados con
  `schema_version=codex_required_test_receipt.v0`, comando exacto, `status`,
  `exit_code`, refs compactas de evidencia, `occurred_at`, `sequence` y
  `output_redacted`.
- La validacion terminal estricta exige un recibo `passed` por cada test
  requerido, rechaza recibos ausentes, comando no coincidente, `exit_code`
  distinto de cero, evidencia ausente o stdout/stderr crudo, y conserva el modo
  legacy solo como observacion compatible.
- `ReadCodexDeliveryObservationFileV0`, startup compaction y el planner de
  backlog usan esa validacion estricta para no cerrar paquetes OrquestaV2 solo
  porque `ACK.tests` contenga strings; las refs de recibo viajan como
  `evidence_refs` compactas de la entrega.
- El prompt de agente externo ya pide `test_receipts` cuando hay tests
  obligatorios, sin guardar prompts, transcripts, HOME, tokens ni salidas crudas.

Evidencia 2026-05-25:

- `codexAckRequiredTestRunnerV0` ya no convierte `ACK.tests` legacy en
  `RequiredTestEvidenceV0`: solo materializa evidencias desde
  `test_receipts` estructurados y, si faltan, delega al runner inyectado o deja
  el bloqueo recuperable.
- `codexStackOperationalClosureSourceV0` ya no reconcilia ACK legacy sin
  `test_receipts` como evidencia de tests; el cierre estricto requiere recibos
  con schema, comando exacto, `status=passed`, `exit_code=0`, refs de evidencia,
  hora, secuencia y `output_redacted=true`.
- Cobertura focal anadida: ACK legacy sin recibos no materializa evidencia
  durable ni cierra el Director Operativo; ACK archivados o por descriptor
  siguen cerrando solo cuando traen `test_receipts` verificables.
- Revalidacion acotada del cierre de app base:
  `go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-app-director-service ./modulos/orquesta-orchestration-core`.
- Rework de revision `agent-ref-task-ref-review-rework-task-autoprogramming-1b2080dfe689-g01-e461228762ca`:
  se conserva el cierre de T47 sin relanzar un padre sobre la tarea original y
  el contexto obligatorio `ref_only` queda resuelto por lectura local de
  `agent_packet.json`, docs vigentes y recibo explicito en el ACK.

## T48 destructive-worktree-change-proof

Objetivo: bloquear cambios destructivos no autorizados en entregas gobernadas,
incluidos borrados, truncados fuertes, reemplazos masivos y renombres ambiguos,
con evidencia de snapshot/worktree y decision del director cuando proceda.

Estado: cerrado 2026-05-24 para bloqueo estricto offline por snapshot/worktree
en runtime-worktree, review gate de autoprogramacion y stack Codex.

Alcance:

- `modulos/orquesta-runtime-worktree`
- `modulos/orquesta-autoprogramming`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`

Criterios:

- El snapshot debe clasificar `removed`, `truncated`, `renamed_or_moved`,
  `replaced_large_delta` y `outside_write_set` como efectos distintos, con refs
  compactas y sin exponer rutas privadas fuera del proyecto.
- En modo estricto, un borrado, truncado fuerte o rename no autorizado bloquea
  `completed` aunque el path este dentro del write-set; solo pasa con decision
  explicita del director, criterio de cierre y test focal.
- Los cambios dentro de write-set siguen permitidos para ediciones normales; el
  rail no debe bloquear refactors pequenos ni generacion de archivos nuevos
  autorizados.
- El review gate de autoprogramacion y el validador de ACK deben consumir la
  misma evidencia de worktree, no inferir destruccion desde `ACK.files`.
- Tests: `go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

Evidencia 2026-05-24:

- `VerifyWorktreeWriteSetV0` clasifica `removed`, `truncated`,
  `renamed_or_moved`, `replaced_large_delta` y `outside_write_set` desde
  snapshots con paths relativos.
- `orquesta-autoprogramming` trata esos issues destructivos como bloqueo de
  cierre, no como rail blando legacy.
- `orquesta-app-codex-stack` consume el baseline de worktree inyectado en el
  review gate y bloquea un truncado fuerte aunque el path este dentro del
  write-set.
- El stack Codex cablea el mismo store de snapshots para capturar baseline al
  resolver launch specs, verificar el diff real antes de aceptar ACK como
  delivery y alimentar el review gate; el servidor residente inyecta ese store
  sin convertir `worktree_ref` ni `branch_ref` en rutas Git.

## Escaneo backlog 2026-05-24 decimocuarta pasada

Evidencia revisada: paquete del scanner de automejora, `AGENTS.md`,
`README.md`, docs vigentes de estado/nucleo/cierre/smokes, este backlog, rail
errors, duplicaciones de rails, `cmd/orquesta-server/config.go`,
`cmd/orquesta-server/stack.go`, `cmd/orquesta-server/codex_wave_command_v0.go`,
`cmd/orquesta-server/codex_director_worktree_v0.go`,
`modulos/orquesta-runtime-codex/codex_profile_v0.go`,
`modulos/orquesta-runtime-codex/codex_wrapper_v0.go`,
`modulos/orquesta-runtime-codex/docs/decisiones.md`,
`modulos/orquesta-runtime-worktree/snapshot_v0.go`,
`modulos/orquesta-runtime-worktree/staging_promotion_v0.go` y
`modulos/orquesta-app-codex-stack/spec_resolver_v0.go`. No se programa codigo
desde este scanner. `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
solo como refs opacas.

Huecos concretos nuevos:

- El perfil Codex rechaza `sandbox` distinto de `workspace-write`, pero
  `cmd/orquesta-server` y `orquesta-app-codex-stack` normalizan cualquier valor
  invalido a `workspace-write` antes de validar. Tambien permiten
  `DirectorApprovalPolicy=on-request` en pruebas/configuracion. Para
  autoprogramacion residente y agentes OrquestaV2, esa correccion silenciosa
  oculta misconfiguracion de seguridad y puede dejar un perfil interactivo en
  una ejecucion desatendida.
- El runtime de control puede estar dentro de `.orquesta-runtime` del proyecto
  o fuera del proyecto con `--add-dir`. La documentacion historica y el prompt
  no tienen una unica regla: unos cortes buscaban runtime externo para aislar
  agentes, mientras los paquetes estrictos ordenan no salir del workdir salvo
  control files indicados. Falta contrato verificable para distinguir control
  dir autorizado de workdir de producto.
- El servidor y varias rutas Codex siguen elevando razonamiento/capacidad a
  `high`/`xhigh` por defecto, mientras las reglas vigentes piden `medium` para
  exploracion y pruebas salvo orden explicita o riesgo justificado. Esto puede
  convertir automejora idle y scanners documentales en consumo caro por defecto.

## T49 codex-runtime-security-profile-strict-mode

Objetivo: hacer que el perfil de runtime Codex usado por autoprogramacion
estricta falle o bloquee de forma publica ante sandbox, approval o director
policy incompatibles, sin correcciones silenciosas.

Estado: cerrado 2026-05-24.

Alcance:

- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-runtime-codex-delivery`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Si el operador configura un sandbox no permitido, el sistema debe devolver
  error publico recuperable o decision requerida; no normalizarlo a
  `workspace-write` sin evidencia.
- En autoprogramacion residente/OrquestaV2 estricta, `approval_policy` debe ser
  no interactiva por contrato o quedar como opt-in explicito con operador vivo;
  no usar `on-request` en ejecucion desatendida.
- El perfil debe declarar si `runtime_work_dir` vive dentro del proyecto o como
  writable root externo de control. Si esta fuera, el packet/prompt debe traer
  excepcion de control concreta y prohibir ediciones de producto alli.
- `orquesta-runtime` conserva politica neutral por refs opacas; Codex traduce
  esa politica en flags concretos solo en el adaptador.
- Tests: `go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

Evidencia focal 2026-05-24:

- `modulos/orquesta-runtime-codex` valida sandbox permitido, approval
  interactivo con opt-in y hint de control dir en prompt.
- `modulos/orquesta-app-codex-stack` conserva el sandbox configurado para que
  el validador produzca error publico en vez de corregirlo en silencio.
- `cmd/orquesta-server` conserva sandbox invalido para validacion publica y
  exige `ORQUESTA_CODEX_ALLOW_INTERACTIVE_APPROVAL=1` para approval interactivo.
- Revalidacion requerida: contexto `ref_only` se resuelve con evidencia explicita
  en ACK y la bateria focal es
  `go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
- Revalidacion 2026-05-24 r2: paquete OrquestaV2 estricto con contexto
  `ref_only` requerido resuelto por evidencia explicita en ACK y bateria focal
  re-ejecutada sin reabrir el alcance T49.

## T50 runtime-control-files-worktree-exclusion

Objetivo: asegurar que ficheros de control del runtime no entren como artefactos
de producto, contexto de agente, snapshot de cambios ni promocion Git, incluso
cuando el write-set sea `.`.

Estado: cerrado 2026-05-25 para la proyeccion por politica en `codex-wave` y
`codex-director-wave`; queda separado T152 para limites mecanicos de copia,
symlinks y presupuesto.

Alcance:

- `modulos/orquesta-runtime-worktree`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `.orquesta-runtime`, `.orquesta-codex-runtime`,
  `.orquesta-local-runtime-*`, prompts, logs, packets, ACKs, checkpoints y
  decision files deben excluirse por defecto de snapshots, verification,
  staging promotion y AppVCS de producto.
- El rail debe aplicar aunque `write_set` sea `.`; alcance raiz no autoriza
  commitear ni promocionar control files.
- Si un agente necesita leer su propio `agent_packet.json`, esa lectura queda
  como control autorizado, no como contexto de producto ni evidence ref
  exportable.
- Las rutas de control detectadas en `git status`, snapshot o `ACK.files` deben
  producir issue estructurado y followup si hace falta, no borrado automatico.
- Tests: `go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

Implementacion 2026-05-24 r2:

- `orquesta-runtime-worktree` aplica exclusiones de control por defecto:
  `.orquesta-runtime`, `.orquesta-codex-runtime`, `.orquesta-control`,
  packets, prompts, ACKs, checkpoints, decision files y logs Codex no entran en
  snapshots ni verificacion, aunque `write_set=["."]`.
- `VerifyWorktreeWriteSetV0` rechaza `ACK.files` que declaren control files con
  issue `control_path`.
- Staging promotion y AppVCS filtran rutas de control detectadas por
  `git status`, las dejan como issue estructurado y solo promocionan/commitean
  paths de producto.
- `orquesta-runtime-codex` mantiene control files como artefactos prohibidos en
  ACK terminal, incluido el caso de write-set raiz.

Revalidacion 2026-05-25 r3:

- La politica comun tambien cubre directorios locales de control con sufijo
  temporal como `.orquesta-local-runtime-*` y `.orquesta-runtime-*`.
- `orquesta-app-codex-stack` consume la lista comun de prefijos de
  `orquesta-runtime-worktree` para baseline, verificacion y review/rework.

## T51 capacity-reasoning-default-policy

Objetivo: sincronizar politica de capacidad/razonamiento con la economia de
tokens vigente para que `xhigh` sea opt-in o justificado, no default silencioso
de automejora documental.

Estado: cerrada 2026-05-25 para el servidor residente Codex/file-based.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-autoprogramming`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Definir matriz por `work_profile_kind`, dominio y riesgo: scanners,
  documentacion y pruebas usan `medium` salvo override; OPES/temarios reales o
  decisiones de arquitectura amplia pueden pedir `high`/`xhigh` con evidencia.
- `cmd/orquesta-server` no debe elevar `ORQUESTA_CODEX_REASONING_EFFORT=medium`
  a `high` ni default `xhigh` para automejora idle sin decision documentada.
- Los prompts/packets deben conservar la razon de capacidad como ref/evidencia
  compacta, no como dato de proveedor en core.
- Stats/uso por proveedor deben reflejar la politica elegida para poder auditar
  gasto de automejora de fondo.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex ./modulos/orquesta-autoprogramming`.

Implementacion 2026-05-25:

- `orquesta-autoprogramming` define una politica compacta por perfil/dominio/
  riesgo: fondo, documentacion, scanners y tests quedan en `medium`; OPES o
  planes documentales reales conservan `xhigh`; riesgo/arquitectura amplia usa
  `high` con evidencia.
- `cmd/orquesta-server` deja de elevar `ORQUESTA_CODEX_REASONING_EFFORT=medium`
  a `high` y deja de usar `xhigh` como default de runtime/stack/wave. Los
  overrides explicitos `low`, `medium`, `high` y `xhigh` se conservan.
- Los packets Codex incluyen `capacity_policy_ref:*` y
  `capacity_policy_evidence_ref:*`; stats/uso heredan esa evidencia para auditar
  el gasto sin meter proveedor/modelo en el nucleo.
- Revalidacion requerida: contexto `ref_only` resuelto por evidencia explicita
  en ACK y bateria focal declarada en esta tarea.

## Escaneo backlog 2026-05-24 decimoquinta pasada

Evidencia revisada: paquete del scanner de automejora, `AGENTS.md`,
`README.md`, docs vigentes de estado/nucleo/principio, este backlog, rail
errors, duplicaciones de rails, busquedas `rg` sobre ACK, control files,
shutdown, tests requeridos, capacidad/razonamiento y delegacion, y medicion
`find modulos cmd -name '*.go' -type f ... wc -l` acotada a ficheros Go por
encima de 300 lineas. No se programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- El rail de 300 lineas ya esta declarado para agentes OrquestaV2, pero el repo
  contiene deuda historica grande. Sin shards concretos, T45 puede quedarse en
  baseline general y futuras entregas tocar ficheros enormes sin propietario de
  particion.
- `modulos/orquesta-app-director-service` concentra ciclo operativo, cierre y
  pruebas en ficheros muy grandes (`operational_director_v0.go`,
  `operational_closure_v0.go`, `continue_v0.go` y tests asociados). Es la
  frontera mas sensible para review/replan/cierre y conviene partirla antes de
  seguir anadiendo comportamiento.
- `modulos/orquesta-orchestration-core` mantiene materializador, cierre,
  plan-state y runner de tests en piezas por encima del rail. Como es nucleo de
  aplicacion neutral, la particion debe preservar puertos y no introducir Codex,
  OPES, web, MCP ni persistencia concreta.
- `modulos/orquesta-app-codex-stack` y `cmd/orquesta-server` acumulan ficheros
  largos de composicion, smokes y comandos. Son adaptadores reales, pero un
  cambio pequeno de autoprogramacion puede acabar mezclando runtime, review,
  VCS, control files y servidor si no se separan responsabilidades.

## T52 app-director-service-file-split

Objetivo: partir la deuda de ficheros grandes del ciclo operativo en
`orquesta-app-director-service` sin cambiar comportamiento, para que nuevas
tareas de cierre/replan no sigan creciendo sobre controladores enormes.

Estado: cerrado 2026-05-25.

Alcance:

- `modulos/orquesta-app-director-service`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Separar `operational_director_v0.go`, `continue_v0.go` y
  `operational_closure_v0.go` por responsabilidad local: bootstrap/continue,
  wait-state, review/replan, cierre causal y helpers de validacion.
- Mantener APIs publicas, DTOs, puertos e invariantes existentes; no meter
  Codex, OPES, HTTP, MCP, filesystem ni runtime concreto en el servicio.
- No hacer refactor funcional amplio: cambios mecanicos con cobertura existente
  y, si aparece riesgo, test focal por helper extraido.
- Registrar baseline de ficheros que sigan por encima de 300 lineas como deuda
  historica, con regla de no crecimiento si no se pueden partir en una sola
  tarea.
- Tests: `go test -count=1 ./modulos/orquesta-app-director-service ./modulos/orquesta-orchestration-core ./modulos/orquesta-state-file`.

Cierre 2026-05-25:

- `operational_director_v0.go`, `continue_v0.go` y
  `operational_closure_v0.go` quedan partidos por responsabilidad local en
  ficheros de bootstrap/continue, wait-state, review/replan, cierre causal,
  required tests y helpers de validacion. No cambia la API publica del paquete
  ni introduce adaptadores concretos.
- No queda ningun fichero Go productivo de
  `modulos/orquesta-app-director-service` por encima de 300 lineas.
- Baseline historico aun por partir, con regla de no crecimiento:
  `operational_director_v0_test.go` 5249,
  `operational_closure_v0_test.go` 2577,
  `operational_director_full_statefile_replay_v0_test.go` 1191,
  `director_decision_source_v0_test.go` 889,
  `operational_closure_prerequisite_retry_v0_test.go` 481,
  `operational_director_required_tests_statefile_v0_test.go` 480,
  `operational_closure_replan_blockers_v0_test.go` 396,
  `provider_composition_v0_test.go` 368 y
  `operational_closure_domain_work_v0_test.go` 305.

## T53 orchestration-core-file-split

Objetivo: partir ficheros grandes del nucleo de aplicacion de orquestacion sin
modificar contratos, para que materializacion, plan-state, cierre y tests
requeridos tengan propietarios pequenos.

Estado: cerrado 2026-05-25.

Alcance:

- `modulos/orquesta-orchestration-core`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Dividir `operational_director_materializer_v0.go`,
  `operational_director_closure_v0.go`,
  `operational_director_plan_state_v0.go` y `required_test_runner_v0.go` por
  responsabilidad, manteniendo paquetes y nombres publicos compatibles.
- Preservar frontera neutral: solo puertos, workflow, tareas, evidencias y refs
  opacas; nada de Codex, OPES, web, MCP, DB concreta, HOME ni paths locales.
- Mantener idempotencia, outbox causal, `WaitAgentRefs`, cierre causal y
  `RequiredTestEvidenceV0` con pruebas existentes verdes.
- Si un fichero no puede bajar de 300 lineas sin cambio funcional, dejar
  baseline explicita y followup mas acotado en docs locales.
- Tests: `go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service`.

Evidencia 2026-05-25:

- `operational_director_materializer_v0.go`,
  `operational_director_closure_v0.go`,
  `operational_director_plan_state_v0.go` y `required_test_runner_v0.go`
  quedaron divididos por responsabilidad neutral y por debajo de 300 lineas.
- Los shards nuevos separan refs/conflictos, request/validacion, task/command,
  causalidad de cierre, comandos de cierre, tests requeridos, store de
  `PlanState` y validacion de `PlanState` sin introducir adaptadores concretos.

## T54 codex-stack-server-file-split

Objetivo: reducir ficheros grandes de composicion Codex/servidor y separar
smokes/comandos por responsabilidad, sin cambiar contratos de runtime,
autoprogramacion ni API.

Estado: completado el 2026-05-25 para diagnostico local `codex-wave-tail`.

Alcance:

- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-runtime-codex-delivery`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Separar comandos y smokes largos de `cmd/orquesta-server` por dominio
  (`codex-wave`, `director-wave`, backlog planner, config/tests) sin mover
  logica al nucleo.
- En `orquesta-app-codex-stack`, partir piezas de drain, supervisor,
  operational closure, composite decision source y smokes reales en helpers por
  flujo; conservar refs opacas, wait scope, ACK estructurado y tests requeridos.
- No relajar rails pendientes T49-T51: seguridad Codex, control files y
  capacidad/razonamiento deben conservar su propietario mientras se separan
  ficheros.
- Los tests/smokes opt-in reales pueden quedar en ficheros separados aunque
  sigan largos temporalmente; documentar baseline y evitar crecimiento.
- Tests: `go test -count=1 ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery`.

Evidencia 2026-05-25: se separaron `codex-wave` y `director-wave` en
`cmd/orquesta-server` por config, launch, prompts/helpers y copia de CODEX_HOME;
`drain_v0.go` en `orquesta-app-codex-stack` quedo partido en normalizacion,
observaciones y scope de espera. Los ficheros productivos tocados quedan por
debajo de 300 lineas. Los smokes/tests historicos largos quedan como baseline
documentada y no deben crecer sin shard propio.

## Escaneo backlog 2026-05-24 decimosexta pasada

Evidencia revisada: paquete del scanner de automejora, `AGENTS.md`,
`README.md`, docs vigentes de estado/nucleo/principio/cierre/smokes, este
backlog, rail errors, duplicaciones de rails,
`modulos/orquesta-server/config_v0.go`,
`modulos/orquesta-server/runtime_v0.go`, `modulos/orquesta-server/handler_v0.go`,
`cmd/orquesta-server/config.go`, `cmd/orquesta-server/mcp_real_transport_v0.go`,
`cmd/orquesta-server/codex_wave_command_v0.go`,
`cmd/orquesta-server/codex_director_wave_command_v0.go`,
`modulos/orquesta-outbox-dispatch`, `modulos/orquesta-orchestration-core` y
`modulos/orquesta-persistence` en la superficie de outbox. No se programa codigo
desde este scanner. `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
solo como refs opacas.

Huecos concretos nuevos:

- El servidor escucha por defecto en loopback, pero `ORQUESTA_SERVER_ADDR`
  permite configurar otro bind sin un rail unificado de exposicion remota,
  autenticacion/autorizacion, TLS/mTLS u opt-in operativo. Los endpoints de
  control incluyen estado, shutdown, cola/runs, autoprogramacion, AppVCS opt-in
  y futuros transportes MCP; no deben quedar expuestos por accidente al cambiar
  el bind.
- Las olas Codex copian `auth.json`, `config.toml`, skills, plugins, rules y
  memories desde `CODEX_HOME`/`ORQUESTA_CODEX_WAVE_SOURCE_CODEX_HOME` a homes de
  agentes. Eso desbloquea runtime real, pero falta una politica de proyeccion de
  credenciales/config por agente: allowlist, redaccion, evidencia compacta,
  exclusion de snapshots/promocion y separacion entre dato secreto real y refs
  opacas de capacidad.
- `orquesta-outbox-dispatch` ya tiene contratos puros de claim/lease/batch/ACK y
  `orquesta-orchestration-core` los usa en tests de lote, pero falta cerrar la
  recuperacion durable de outbox en composicion residente: claims vivos,
  ACK parcial, restart, reintento sin duplicar efectos externos y visibilidad
  publica de items `pending`/`ack_failed`.

## T55 server-control-plane-exposure-guards

Objetivo: impedir que el control plane HTTP/MCP del servidor quede expuesto fuera
de loopback sin confirmacion, autenticacion/autorizacion y auditoria compacta.

Estado: cerrado el 2026-05-25 para guarda general HTTP residente.

Alcance:

- `modulos/orquesta-server`
- `modulos/orquesta-http-gateway`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-mcp`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Mantener loopback como default seguro; cualquier bind no-loopback o transporte
  residente sensible requiere opt-in explicito, razon publica y guardas de
  operador.
- Las rutas mutables (`shutdown`, `runs/control`, cola, `prepare-run`,
  `self_improvement`, AppVCS y MCP real si se activa) deben pasar por un puerto
  de autorizacion o token/mTLS opt-in de composicion; health/status pueden quedar
  en modo lectura limitada.
- CLI/web/MCP deben transportar credencial o principal por adaptador, sin meter
  auth, TLS, HOME, tokens ni proveedor en el nucleo.
- Auditoria debe registrar principal/ref de permiso, bind y decision sin guardar
  tokens, cabeceras completas, payloads HTTP crudos ni rutas privadas.
- Tests: `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-mcp ./cmd/orquesta-server`.

Cierre 2026-05-25:

- `orquesta-server` mantiene `127.0.0.1:8787` como default y rechaza bind no
  loopback salvo opt-in `ORQUESTA_SERVER_REMOTE_CONTROL_PLANE_CONFIRM=1` con
  `ORQUESTA_SERVER_CONTROL_TOKEN`.
- El residente aplica una guarda comun sobre mutaciones del control plane:
  rutas `POST` bajo `/api/v0/`, `/mcp` real y formularios web mutables quedan
  autorizadas por loopback local o por token opt-in de composicion.
- La auditoria registra decision, principal/ref de permiso, bind, metodo y path;
  no guarda tokens, cabeceras completas, payloads ni valores de query.
- Tests ejecutados: `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-mcp ./cmd/orquesta-server`.

## T56 codex-code-home-credential-projection

Objetivo: convertir la copia de `CODEX_HOME` hacia agentes Codex en una politica
explicita de proyeccion de credenciales/config, con minimo privilegio y exclusion
de artefactos de producto.

Estado: cerrado 2026-05-25 para `codex-wave` y `codex-director-wave`. El
shard mecanico de presupuesto, symlinks y limites de copia queda separado en
T152.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-runtime-worktree`
- `modulos/orquesta-app-codex-stack`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Declarar por puerto/politica que ficheros de `CODEX_HOME` pueden proyectarse a
  un agente (`auth.json`, config, skills, plugins, rules, memories) y cuales
  requieren opt-in separado.
- La evidencia durable debe decir categorias copiadas y refs opacas, no valores
  secretos, HOME real, rutas privadas, tokens, prompts, transcripts ni contenido
  completo de memorias/plugins.
- Las credenciales proyectadas no deben entrar en snapshot de producto,
  `ACK.files`, contexto de agentes, AppVCS, staging promotion ni audit payloads.
- En modo estricto, un agente no debe compartir o modificar credenciales de otro
  agente; si falta auth/config, bloquear con error publico recuperable en vez de
  copiar todo el home por defecto.
- Evidencia 2026-05-25: `codexWaveCredentialProjectionPolicyV0` declara
  categorias permitidas (`auth`, `config`, `skills`, `plugins`, `rules`,
  `instructions`, `runtime_config`) y `memories` solo por opt-in. Cada agente
  aislado registra receipt compacto `orquesta_codex_code_home_projection.v0`
  con categorias y refs opacas, sin valores ni ruta fuente. El modo estricto
  exige home aislado y bloquea `auth`/`config` faltantes con error recuperable.
- Rework de revision 2026-05-25: se corrige la entrega documental para no dejar
  T56 como pendiente despues de la evidencia ya registrada. No se relanza otro
  agente padre sobre la tarea original y el contexto obligatorio `ref_only`
  queda resuelto por lectura local y recibo explicito en ACK.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack`.

## T57 outbox-dispatch-durable-ack-recovery

Objetivo: cerrar la recuperacion durable de dispatch de outbox con claim/lease,
ACK correlacionado y reintentos idempotentes en el servidor residente.

Estado: completado el 2026-05-25 para `cmd/orquesta-server`.

Alcance:

- `modulos/orquesta-outbox-dispatch`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-persistence`
- `modulos/orquesta-app-director-service`
- `modulos/orquesta-server`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`

Criterios:

- Persistir y rehidratar claims/leases de outbox por `message_id`, `run_id`,
  `target_port`, `claim_ref`, `lease_ref` y ACK correlacionado; no depender de
  memoria de proceso ni de que el tick actual vea todos los efectos.
- Tras reinicio, un item con ACK success no se reejecuta; ACK failed queda
  visible como fallo publico; ACK ausente o claim expirado permite retry
  idempotente sin duplicar efectos externos.
- Los lotes parciales deben cerrar solo items ACK success y conservar los demas
  `pending`/`ack_failed` con evidencia compacta.
- El loop del Director no debe cerrar plan/run si queda outbox pendiente no
  reconciliada; debe bloquear/reintentar con causa durable y contadores.
- Tests: `go test -count=1 ./modulos/orquesta-outbox-dispatch ./modulos/orquesta-orchestration-core ./modulos/orquesta-persistence ./modulos/orquesta-app-director-service ./modulos/orquesta-server ./cmd/orquesta-server`.

Evidencia 2026-05-25:

- `orquesta-persistence.NewFileOutboxLedgerV0` queda cableado en
  `cmd/orquesta-server` como ledger durable del servidor residente. Persiste
  mensajes, claims con `claim_ref`/`lease_ref` y ACK terminales; al reabrir,
  claims sin ACK quedan recuperables para retry idempotente y ACK `success` no
  vuelve a entrar en pendientes.
- `RunOutboxDispatchBatchAckClosureV0` usa el puerto opcional
  `OutboxDispatchAckObservationPortV0` para persistir ACK `failed` sin cerrar
  por arrastre otros items del lote; los ACK success siguen retirando solo su
  `message_id` correlacionado.
- El ledger file-based expone snapshots compactos de ACK para auditoria publica
  (`dispatched`/`failed`) sin rutas, HOME, proveedor, DB ni transcripts.
- Rework de revision 2026-05-25: se conserva la entrega valida de T57, se
  confirma el contexto obligatorio `ref_only` por evidencia explicita en ACK y
  no se relanza otro agente padre sobre la tarea original.

## Escaneo backlog 2026-05-24 decimoseptima pasada

Evidencia revisada: paquete del scanner de automejora, `AGENTS.md`,
`README.md`, docs vigentes de estado/nucleo/principio/smokes, este backlog,
rail errors, duplicaciones de rails,
`cmd/orquesta-server/codex_wave_control_v0.go`,
`cmd/orquesta-server/codex_wave_command_v0.go`,
`cmd/orquesta-server/codex_director_wave_command_v0.go`,
`modulos/orquesta-runtime-codex/docs/contratos.md`,
`modulos/orquesta-runtime-codex/docs/pruebas.md` y busquedas `rg` sobre
`codex-wave-tail`, `purge-runtime`, `RemoveAll`, `stdout`, `stderr`,
`last-message` y `Stop`. No se programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- `codex-wave-tail` lee y muestra lineas crudas de `codex_stdout.log`,
  `codex_stderr.log` o `codex_last_message.txt` desde el runtime de la ola. Es
  util para diagnostico local, pero no tiene politica comun de redaccion,
  permiso, limite de tamano ni marca de riesgo; puede exponer prompts,
  transcripts, HOME, rutas privadas, tokens o diffs completos fuera de los
  recibos compactos que ya gobiernan ACK/delivery.
- `codex-launch-wave` y `codex-launch-director-wave` aceptan
  `--purge-runtime`/`ORQUESTA_CODEX_WAVE_PURGE_RUNTIME` y llaman `RemoveAll`
  sobre el runtime resuelto. Hay guardas basicas por `wave_ref`, pero falta un
  contrato opt-in de borrado seguro: raiz permitida, manifest previo, dry-run,
  bloqueo si hay agentes vivos/checkpoints/outbox pendiente y evidencia
  compacta de lo purgado.
- `codex-wave-stop` carga el registro desde un `runtime-dir` dado y senala los
  PIDs listados. Sin prueba de propiedad del registro/descriptores ni enlace
  con registry de procesos de Orquesta, un runtime mal seleccionado o corrupto
  puede convertir una herramienta de control de ola en parada de proceso fuera
  de scope.

## T58 codex-wave-log-tail-redaction-access

Objetivo: convertir `codex-wave-tail` en una superficie de diagnostico acotada,
redactada y autorizada, sin usar logs crudos como evidencia terminal.

Estado: completada.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-runtime-codex-delivery`
- `modulos/orquesta-rails`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `codex-wave-tail` debe aplicar redaccion por campo antes de imprimir
  stdout/stderr/last-message: sin HOME real, tokens, prompts completos,
  transcripts, rutas privadas, credenciales, payloads HTTP crudos ni diffs
  largos.
- La salida debe tener limites de lineas/bytes, modo `summary` por defecto y
  opt-in explicito para mostrar fragmentos crudos en diagnostico local.
- El acceso debe enlazar `wave_ref`, `agent_ref`, runtime permitido y razon de
  diagnostico; si el control plane expone esta funcion, debe heredar T55.
- `codex_last_message.txt` sigue siendo diagnostico, no fuente de cierre,
  compactacion, delivery ni ACK completado; T24/T33 conservan la fuente
  estructurada.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-rails`.

Evidencia 2026-05-25: `codex-wave-tail` emite JSON de diagnostico con
`summary` por defecto, exige `--reason`/`ORQUESTA_CODEX_WAVE_TAIL_REASON`,
limita lineas y bytes antes de leer, valida que el log pertenezca al runtime de
la ola/agente y solo muestra fragmentos con `--mode fragment`. Los fragmentos
pasan por `orquesta-rails` para redactar valores sensibles, HOME/rutas privadas,
prompts, transcripts, payloads y credenciales antes de imprimir. La salida no
se usa como ACK, delivery, cierre ni compactacion terminal.

Rework de revision 2026-05-25: se conserva la entrega util y se corrige el
estado documental de T58 para no dejar el backlog en `pendiente` cuando la
evidencia focal ya esta integrada.

## T59 codex-wave-runtime-purge-proof

Objetivo: endurecer la purga de runtime de olas Codex para que ningun borrado de
directorio ocurra sin opt-in, scope verificable, bloqueo de agentes vivos y
evidencia compacta.

Estado: completado el 2026-05-25 para `cmd/orquesta-server`.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-runtime-worktree`
- `modulos/orquesta-server-shutdown`
- `modulos/orquesta-agent-process-registry`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `--purge-runtime` requiere confirmacion explicita, `wave_ref`, runtime bajo una
  raiz permitida de Orquesta y manifest previo de archivos/directorios a purgar.
- La purga debe bloquear si hay agentes vivos, checkpoints pendientes,
  `agent_ack.json` no reconciliado, `director_decisions.json` no consumido,
  outbox pendiente o plan state abierto para esa ola.
- Modo dry-run/report debe mostrar contadores y refs compactas, no rutas
  privadas completas ni contenido de logs/prompts.
- La implementacion no debe borrar fuera del runtime de la ola aunque
  `runtime-dir` sea absoluto o contenga `codex-waves` por casualidad.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-server-shutdown ./modulos/orquesta-agent-process-registry`.

Evidencia 2026-05-25: `--purge-runtime` exige `--confirm-purge-runtime=<wave_ref>`,
solo acepta runtime bajo raiz permitida `.orquesta-runtime/codex-waves` del
proyecto o `ORQUESTA_CODEX_RUNTIME_WORKDIR/codex-waves`, crea reporte compacto
con contadores/refs, soporta `--purge-runtime-report` sin borrar y bloquea por
agentes vivos, checkpoints pendientes, ACKs no reconciliados,
`director_decisions.json`, outbox o plan state detectados en el runtime.
Rework de revision 2026-05-25: la correccion
`agent-ref-task-ref-review-rework-task-autoprogramming-27e211875a6e-g01-c0a72b474b18`
conserva la entrega valida, no relanza otro agente padre sobre T59 y resuelve
el contexto obligatorio `ref_only` con lectura local del paquete y evidencia
explicita en ACK.

## T60 codex-wave-stop-registry-pid-proof

Objetivo: hacer que `codex-wave-stop` solo pueda parar procesos que Orquesta
lanzó y sigue reconociendo como agentes de la ola solicitada.

Estado: cerrado en codigo el 2026-05-25.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-runtime`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-agent-process-registry`
- `modulos/orquesta-run-control`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Antes de senalar un PID, validar descriptor firmado o store de proceso:
  `run_ref`, `wave_ref`, `agent_ref`, command ref, runtime dir permitido,
  start time/owner opaco y estado vivo.
- Un registro JSON suelto bajo un `runtime-dir` arbitrario no basta para parar
  procesos; debe quedar como `blocked_registry_untrusted` con error publico.
- `stop` debe respetar shutdown cooperativo cuando el runtime soporta
  checkpoint; `forced` requiere politica explicita y evidencia de decision.
- Status/tail/stop deben compartir la misma resolucion segura de runtime y
  registry para no tener tres rails distintos de confianza en ficheros.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-agent-process-registry ./modulos/orquesta-run-control`.

Evidencia 2026-05-25: `codex-launch-wave` escribe un descriptor
`codex_wave_process_proof_v0.json` por agente con refs opacas de proceso,
launch, session, command, owner/start y digest. `codex-wave-status`,
`codex-wave-tail` y `codex-wave-stop` cargan el registry por la misma resolucion
segura de runtime permitido; un registry suelto o fuera de raiz permitida queda
bloqueado como `blocked_registry_untrusted`. `codex-wave-stop` valida el proof
antes de senalar, exige `--confirm-stop=<wave_ref>` y `--reason`, escribe
shutdown cooperativo por defecto y solo senala el PID con `--force` explicito.
Rework de revision 2026-05-25: la correccion estricta conserva la entrega T60,
no relanza otro agente padre sobre la tarea original y resuelve el contexto
obligatorio `required ref_only` mediante lectura local del paquete mas evidencia
explicita en ACK.

## Escaneo backlog 2026-05-24 decimoctava pasada

Evidencia revisada: paquete del scanner de automejora, `AGENTS.md`,
`README.md`, docs vigentes de estado/nucleo/principio/smokes, este backlog,
rail errors, duplicaciones de rails,
`cmd/orquesta-server/required_test_runner_stack_v0.go`,
`modulos/orquesta-runtime-required-test/local_command_executor_v0.go`,
`modulos/orquesta-runtime-codex-delivery/director_decision_file_descriptors_v0.go`,
`modulos/orquesta-director-agent-file-source/source_v0.go`,
`modulos/orquesta-app-codex-stack/director_decision_contract_v0.go` y
`docs/corte_plan_state_director_decisions_2026-05-21.md`. No se programa codigo
desde este scanner. `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
solo como refs opacas.

Huecos concretos nuevos:

- `LocalCommandExecutorV0` captura stdout/stderr de tests requeridos y escribe
  el artefacto completo con `output.String()` bajo `required-test-output`. Hay
  limite de bytes y env aislado, pero no hay redaccion por campo, politica de
  retencion ni separacion entre evidencia causal compacta y log diagnostico
  local. Un test real puede imprimir rutas privadas, payloads HTTP, variables
  de entorno, tokens simulados o datos de dominio.
- El prompt de `director_decisions.json` en
  `directorDecisionInstructionsV0` conservaba una lista textual de "terminos
  prohibidos" (`provider`, `model`, `db`, `sql`, `runtime`, `git`, `home`,
  etc.). T62 lo cierra el 2026-05-25 sustituyendola por politica positiva de
  refs opacas y datos no publicables.
- La fuente de `director_decisions.json` deriva el path desde el `AckPath` del
  recibo y filtra por run/reflejo de ACK, pero el sidecar no tiene recibo propio
  con hash, correlacion, producer ACK, estado consumido ni idempotencia visible.
  Si aparece un fichero stale o ambiguo junto a un ACK valido, la frontera entre
  diagnostico de runtime y decision ejecutable queda repartida entre delivery,
  file-source y app-codex-stack.

## T61 required-test-output-redaction-retention

Objetivo: separar evidencia causal de tests requeridos de logs diagnosticos
crudos, con redaccion y retencion explicitas para stdout/stderr.

Estado: cerrado 2026-05-25 para el tramo local `LocalCommandExecutorV0`:
stdout/stderr se persisten como artefacto diagnostico redactado, acotado y con
retencion configurable. `RequiredTestEvidenceV0` conserva refs causales y
comando exacto; el cierre no depende de log crudo. Queda fuera de este cierre
definir una politica productiva de datos de dominio no publicables mas alla de
los patrones comunes de rails.

Alcance:

- `modulos/orquesta-runtime-required-test`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-state-file`
- `modulos/orquesta-app-director-service`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `RequiredTestEvidenceV0` debe seguir guardando comando exacto, status y refs
  causales, pero los logs completos deben ser artefactos diagnosticos con
  redaccion por campo antes de persistir.
- La redaccion debe cortar HOME real, rutas privadas, tokens, secretos,
  cabeceras Authorization, payloads HTTP crudos, prompts, transcripts y datos de
  dominio no publicables; vocabulario operativo opaco debe pasar.
- El output debe tener limite de bytes, retencion configurable y summary estable
  que permita diagnostico sin depender del log crudo para cerrar el plan.
- Si la redaccion detecta material sensible no redactable, el test puede quedar
  `failed` o `blocked` con causa publica, pero no debe persistir el material.
- Tests: `go test -count=1 ./modulos/orquesta-runtime-required-test ./modulos/orquesta-orchestration-core ./modulos/orquesta-state-file ./modulos/orquesta-app-director-service ./cmd/orquesta-server`.

Evidencia 2026-05-25:

- `LocalCommandExecutorV0` aplica `orquesta-rails` antes de escribir
  `required-test-output-v0/*.log`, marca `output_redacted=true`, conserva
  `redaction_applied`, limite efectivo de bytes y `retention_max_artifacts`.
- La salida no UTF-8 o material privado no normalizable bloquea la evidencia del
  comando como `failed` y escribe solo una causa publica compacta, sin bytes
  crudos.
- El servidor expone `ORQUESTA_REQUIRED_TEST_OUTPUT_MAX_ARTIFACTS` para acotar
  retencion de artefactos diagnosticos del runner opt-in.
- Cobertura focal: redaccion de token/Bearer/ruta HOME, bloqueo de salida no
  redactable, retencion de artefactos y wiring de configuracion del servidor.

## T62 director-decisions-prompt-rail-sync

Objetivo: alinear las instrucciones de `director_decisions.json` con la politica
actual de rails blandos: refs opacas y vocabulario operativo pasan; secretos
efectivos y payloads crudos se bloquean.

Estado: cerrado 2026-05-25 para el prompt de `director_decisions.json`,
validacion focal y sincronizacion documental del backlog. Si reaparecen falsos
positivos de vocabulario operativo, deben entrar como regresion con evidencia
nueva.

Alcance:

- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-director-agent`
- `modulos/orquesta-director-agent-file-source`
- `modulos/orquesta-rails`
- `modulos/orquesta-runtime-codex-delivery`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Retirar del prompt listas de palabras prohibidas por vocabulario normal
  (`provider`, `model`, `db`, `sql`, `runtime`, `adapter`, `git`, `home`) y
  sustituirlas por regla positiva: usar refs opacas, no valores reales ni rutas
  privadas.
- Validadores de decision y tests deben compartir owner de politica sensible o
  declarar frontera local con matriz externa; no duplicar listas divergentes
  entre prompt, JSON contract y rail comun.
- Mantener cortes fuertes para secretos efectivos, credenciales, prompts o
  transcripts crudos, refs imposibles y causalidad rota.
- Incluir caso realista donde una decision mencione `runtime` o `provider` como
  ref opaca sin ser rechazada ni inducir al director a perder contexto.
- Tests: `go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-director-agent ./modulos/orquesta-director-agent-file-source ./modulos/orquesta-rails ./modulos/orquesta-runtime-codex-delivery`.

Evidencia 2026-05-25:

- `directorDecisionInstructionsV0` deja de enumerar palabras prohibidas y usa
  regla positiva: refs opacas para runtime/provider/modelo/DB/git/filesystem y
  bloqueo de valores reales, rutas privadas, credenciales, secretos,
  prompts/transcripts crudos y payloads completos.
- Los validadores siguen delegando detalle sensible en `orquesta-rails` por
  `ValuesContainOperationalSensitiveDetailForFieldV0`; la matriz permite
  vocabulario operativo opaco y corta valores sensibles efectivos.
- Casos focales anadidos/reforzados: el prompt no contiene lista negativa vieja
  y una decision con `runtime`/`provider` como refs opacas valida correctamente.
- Rework de revision `agent-ref-task-ref-review-rework-task-autoprogramming-c5b294454f0a-g01-299e80bd0cfe`:
  se conserva el cierre de T62 sin relanzar otro agente padre, se corrige la
  evidencia documental para no mezclarla con smokes de servidor de T65 y el
  contexto obligatorio `ref_only` queda resuelto por lectura local del paquete,
  AGENTS/README y recibo explicito en ACK.

## T63 director-decisions-sidecar-receipt-correlation

Objetivo: hacer que `director_decisions.json` tenga recibo estructurado propio y
correlado antes de convertirse en decisiones ejecutables.

Estado: cerrado 2026-05-25.

Alcance:

- `modulos/orquesta-runtime-codex-delivery`
- `modulos/orquesta-director-agent-file-source`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-app-director-service`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Cada `director_decisions.json` consumible debe quedar enlazado a
  `run_ref`, `agent_ref`, `ack_ref`, `correlation_id`, hash del fichero y ref de
  recibo; el mero path junto a `agent_ack.json` no basta.
- El source debe ignorar o bloquear sidecars stale, no regulares, con hash
  cambiado, sin ACK productor reflejado, fuera de scope `WaitAgentRefs` o ya
  consumidos bajo otro receipt.
- La aplicacion debe registrar consumo idempotente: reaplicar el mismo receipt
  no duplica decisiones, tasks, outbox ni `PlanState`; un payload distinto bajo
  la misma ref es conflicto publico.
- `director_decisions.json` sigue siendo sidecar de composicion Codex; el core
  puro no conoce nombres de archivo, paths ni runtime.
- Tests: `go test -count=1 ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-director-agent-file-source ./modulos/orquesta-app-codex-stack ./modulos/orquesta-app-director-service ./cmd/orquesta-server`.

Cierre aplicado 2026-05-25:

- `CodexReceiptDirectorDecisionFileDescriptorProviderV0` calcula un receipt
  propio para `director_decisions.json` con `ack_ref` productor, `run_id`,
  `agent_ref`, `correlation_id`, `sha256`, tamano y `receipt_ref` estable; solo
  expone el sidecar tras ACK reflejado.
- `DirectorAgentDecisionFileSourceV0` valida hash/tamano antes de decodificar y
  marca el receipt como `consumed` por puerto inyectado; el stack Codex inyecta
  el `ReceiptStore` como recorder cuando el store lo soporta.
- Los stores de receipts en memoria y file-based persisten el receipt del
  sidecar; un sidecar ya consumido se omite idempotentemente y un payload
  cambiado bajo el mismo receipt bloquea con conflicto publico compacto.
- Cobertura focal: sidecar correlado, consumo idempotente, hash cambiante y
  lectura del source con receipt. La frontera sigue en adaptador Codex; el core
  puro no conoce `director_decisions.json`.

## Escaneo backlog 2026-05-24 decimonovena pasada

Evidencia revisada: paquete del scanner de automejora, `AGENTS.md`,
`README.md`, docs vigentes de estado/nucleo/principio/smokes, este backlog,
rail errors, duplicaciones de rails,
`modulos/orquesta-orchestration-core/AGENTS.md`,
`modulos/orquesta-director-cycle/README.md`,
`modulos/orquesta-director-scheduler/README.md`,
`modulos/orquesta-director-runner/README.md`,
`modulos/orquesta-director-tick-input/README.md`,
`modulos/orquesta-director/docs/pruebas.md`,
`cmd/orquesta-server/stack.go` y busquedas `rg` sobre `DirectorCycle`,
`DirectorScheduler`, `OutboxLedger`, `ProcessRuntime` y `StopRuntimeAgent`. No
se programa codigo desde este scanner. `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
solo como refs opacas.

Huecos concretos nuevos:

- La foto raiz y la matriz de smokes no nombran aun la nueva espina
  `orquesta-director-cycle`/`scheduler`/`runner`/`tick-input`/`cycle-outbox`,
  mientras `orquesta-orchestration-core` ya la importa y su `AGENTS.md` local
  conserva una lista vieja de pendientes de review/tests/rework/cierre. Esa
  divergencia puede hacer que un scanner reabra trabajo ya cerrado offline o
  ignore los nuevos contratos al preparar contexto.
- Hay pruebas locales de scheduler, runner, outbox y state-file, pero falta un
  caso de composicion residente que demuestre `DirectorCycleStepV0` con
  `FileOutboxLedgerV0`, dispatch/ACK de outbox, reinicio y reentrada sin
  duplicar comandos ni cerrar si queda outbox pendiente.
- `modulos/orquesta-director/docs/pruebas.md` conserva `DIR-P006` como parada
  de proceso real pendiente. `ProcessRuntimeConnectorV0` y el wiring de servidor
  existen, pero falta un smoke neutral que recorra Director/scheduler/outbox
  hasta `StopRuntimeAgent` contra proceso temporal real, con ACK/evidencia e
  idempotencia. `T60` cubre el stop de ola Codex por registry/PID; no cierra este
  caso neutral.

## T64 director-cycle-source-of-truth-sync

Objetivo: sincronizar los documentos de autoridad con la espina actual de ciclo
del Director V2 para que agentes y scanners no mezclen pendientes historicos con
contratos vivos.

Estado: completada 2026-05-25.

Alcance:

- `AGENTS.md`
- `README.md`
- `docs/estado_actual_2026-05-17.md`
- `docs/guia_nucleo_orquestacion_2026-05-17.md`
- `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`
- `modulos/orquesta-orchestration-core/AGENTS.md`
- `modulos/orquesta-orchestration-core/README.md`
- `modulos/orquesta-director-cycle`
- `modulos/orquesta-director-scheduler`
- `modulos/orquesta-director-runner`
- `modulos/orquesta-director-tick-input`
- `modulos/orquesta-director-cycle-outbox`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- La foto vigente debe nombrar la espina `DirectorCycleStepV0 -> runner ->
  scheduler -> workflow -> cycle-outbox` y distinguirla del flujo historico
  `app-director-service`/loop progresivo.
- `orquesta-orchestration-core/AGENTS.md` no debe seguir declarando
  review/tests/rework/cierre como pendientes genericos si ya tienen cobertura
  offline; debe listar solo pendientes reales verificables.
- La matriz debe tener caso offline de fuente de verdad para scheduler/cycle y
  remitir a pruebas locales existentes sin declarar smokes reales como cerrados.
- Cualquier documento que diga "pendiente" para Director debe indicar si falta
  codigo, composicion residente, smoke real, proveedor real u OPES temporal.
- Tests: `go test -count=1 ./modulos/orquesta-director-cycle ./modulos/orquesta-director-scheduler ./modulos/orquesta-director-runner ./modulos/orquesta-director-tick-input ./modulos/orquesta-orchestration-core`.

Evidencia 2026-05-25: `AGENTS.md`, `README.md`,
`docs/estado_actual_2026-05-17.md`,
`docs/guia_nucleo_orquestacion_2026-05-17.md`,
`docs/matriz_pruebas_reales_y_smoke_2026-05-17.md` y los docs locales de
`orquesta-orchestration-core` nombran la espina
`DirectorCycleStepV0 -> director-runner -> director-scheduler ->
core-workflow -> director-cycle-outbox`, la separan de
`app-director-service`/loop progresivo y clasifican pendientes por codigo
offline, composicion residente/restart, smoke real neutral, proveedor real u
OPES temporal. La matriz anade `DIRECTOR-CYCLE-SOURCE-OFFLINE` sin declarar
cerrados smokes reales. Refs opacas conservadas:
`worktree-ref-orquesta-server-idle-self-improvement` y
`branch-ref-orquesta-server-idle-self-improvement`.

## T65 director-cycle-resident-restart-smoke

Objetivo: demostrar en composicion de servidor que el ciclo del Director usa
outbox durable, dispatch/ACK y reentrada tras reinicio sin duplicar efectos ni
cerrar con outbox pendiente.

Estado: completada 2026-05-25.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-app-director-service`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-state-file`
- `modulos/orquesta-outbox-dispatch`
- `modulos/orquesta-director-cycle`
- `modulos/orquesta-director-cycle-outbox`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Un servidor temporal debe crear o continuar una run que pase por
  `DirectorCycleStepV0`, registre outbox en `FileOutboxLedgerV0` y despache un
  mensaje por puerto fake/temporal con ACK correlacionado.
- Tras reiniciar la composicion con el mismo state dir, el ledger debe reabrir
  pendientes/claims sin ACK, no reejecutar ACK success y conservar ACK failed
  como causa publica.
- El loop debe bloquear `close`/`completed` si queda outbox pendiente no
  reconciliada, exponiendo contadores y refs compactas.
- El smoke debe distinguir compatibilidad legacy de run completo frente a ciclo
  Director V2; no basta una prueba local con `InMemoryOutboxLedgerV0`.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack ./modulos/orquesta-state-file ./modulos/orquesta-outbox-dispatch`.

Evidencia 2026-05-25: `cmd/orquesta-server` incorpora el smoke focal
`TestDirectorCycleResidentRestartSmokeV0`. El caso crea composicion temporal con
`ORQUESTA_SERVER_STATE_DIR`, ejecuta `DirectorCycleStepV0`, guarda outbox en
`FileOutboxLedgerV0`, reconstruye el stack sobre el mismo estado, verifica que la
reentrada queda en `waiting` por outbox pendiente, despacha por puerto fake con
ACK correlacionado y valida que un ACK fallido se proyecta como causa publica
compacta. El tercer ciclo no reutiliza la ref del outbox ya ACKeado, por lo que
no duplica el efecto anterior. Esto cierra la composicion residente/restart
local de T65; no cierra el smoke neutral de proceso real de T66.

## T66 neutral-process-stop-e2e

Objetivo: cerrar el pendiente neutral `DIR-P006` con un proceso temporal real
controlado por puertos de runtime, sin Codex ni proveedor externo.

Estado: completada 2026-05-25.

Alcance:

- `modulos/orquesta-runtime`
- `modulos/orquesta-director`
- `modulos/orquesta-director-scheduler`
- `modulos/orquesta-director-cycle`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-state-file`
- `cmd/orquesta-server`
- `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Lanzar un proceso temporal explicito con `ProcessRuntimeConnectorV0`, sin
  shell heredado, HOME real, entorno completo ni comando de proveedor.
- El Director/scheduler debe producir `StopRuntimeAgent` por causa durable
  (lease, progreso detenido o orden controlada), enviarlo por outbox y registrar
  ACK/evidencia de parada por `run_ref`, `agent_ref` y `process_ref`.
- Reintentar el mismo stop tras reinicio debe ser idempotente; un `process_ref`
  ajeno, stale o de otro run debe bloquear con error publico.
- El snapshot/ACK no debe persistir command path, env completo, HOME, tokens ni
  payloads crudos; solo refs compactas y estado de parada.
- El caso debe quedar separado de `T30` shutdown checkpoint y `T60` stop de ola
  Codex por registry/PID.
- Tests: `go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-director ./modulos/orquesta-director-scheduler ./modulos/orquesta-director-cycle ./modulos/orquesta-orchestration-core ./cmd/orquesta-server`.

Evidencia 2026-05-25: `cmd/orquesta-server` incorpora
`TestNeutralProcessStopE2EV0`. El caso lanza un proceso temporal explicito con
`ProcessRuntimeConnectorV0` usando el binario de test, sin shell, HOME ni env
heredado; siembra una run neutral con agente/proceso registrado por refs opacas;
`DirectorCycleStepV0` consume una supervision de progreso `loop_detected` y el
scheduler/workflow producen `StopRuntimeAgent` en outbox durable. El dispatcher
usa `AgentStopperExecutorV0` + `ProcessAgentStopperV0` contra el runtime real,
registra `AgentStopConfirmed`, ACKea el outbox y, tras reconstruir el stack con
el mismo state dir, el replay queda `no_pending` sin reejecutar el stop. La
prueba valida que el estado persistido no incluya command path, env de test,
`working_dir`, HOME, tokens ni secretos. Este cierre es neutral y no toca Codex,
OPES, T30 shutdown checkpoint ni T60 stop de ola Codex.

## Escaneo backlog 2026-05-24 vigesima pasada

Evidencia revisada: paquete del scanner de automejora, `AGENTS.md`,
`README.md`, docs vigentes de estado/nucleo/principio, este backlog, rail
errors, duplicaciones de rails, `modulos/orquesta-orchestration-core/AGENTS.md`,
`modulos/orquesta-director-supervisor/README.md`,
`modulos/orquesta-director-supervised-burst/README.md`,
`modulos/orquesta-run-supervisor/README.md`,
`modulos/orquesta-orchestration-core/types.go`,
`modulos/orquesta-orchestration-core/progressive_loop.go`,
`modulos/orquesta-director-supervisor/supervisor_types_v0.go`,
`modulos/orquesta-director-supervised-burst/burst_types_v0.go`,
`modulos/orquesta-run-supervisor/supervisor_v0.go`,
`modulos/orquesta-server/supervisor_loop_v0.go`,
`cmd/orquesta-server/config.go` y busquedas `rg` sobre
`director-supervised-burst`, `director-supervisor`, `run-supervisor`,
`MaxSteps`, `MaxTicks` y `DirectorSupervisorAction`. No se programa codigo
desde este scanner. `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
solo como refs opacas.

Huecos concretos nuevos:

- `T64` sincroniza la espina `director-cycle`/`scheduler`/`runner`/`tick-input`,
  pero la foto raiz y la matriz siguen sin nombrar dos piezas que
  `orquesta-orchestration-core` ya importa como parte del nucleo vivo:
  `orquesta-director-supervisor` y `orquesta-director-supervised-burst`. Sin
  esa fuente de verdad, un agente puede leer solo el ciclo/runner y duplicar la
  politica de repetir/parar pasos.
- El servidor configura presupuestos anidados (`MaxTicks`, `MaxRunsPerTick`,
  `MaxExecutions`, `MaxBursts`, `MaxStepsPerBurst`, `MaxCommands`), pero falta
  un smoke de composicion que pruebe que la rafaga supervisada corta por
  `wait_outbox`, `wait_external`, `stop_max_steps` o `stop_error` sin relanzar
  la misma run ni ocultar outbox pendiente como exito.
- `run-supervisor`, `director-supervisor` y `director-supervised-burst` ya
  tienen razones de parada locales, pero no hay contrato publico unico para
  propagar esas razones a auditoria, stats, cola e idle autoprogramming. `T31`
  cubre reserva/lease de cola y `T65` outbox residente; no cubren la taxonomia
  de parada del supervisor anidado.

## T67 director-supervisor-burst-source-of-truth-sync

Objetivo: sincronizar la fuente de verdad del ciclo Director V2 para incluir
`orquesta-director-supervisor` y `orquesta-director-supervised-burst` como
piezas vivas del nucleo de aplicacion, sin duplicar `T64`.

Estado: cerrado con implementacion focal el 2026-05-25.

Alcance:

- `AGENTS.md`
- `README.md`
- `docs/estado_actual_2026-05-17.md`
- `docs/guia_nucleo_orquestacion_2026-05-17.md`
- `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`
- `modulos/orquesta-orchestration-core/AGENTS.md`
- `modulos/orquesta-orchestration-core/README.md`
- `modulos/orquesta-director-supervisor`
- `modulos/orquesta-director-supervised-burst`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- La foto vigente debe nombrar la cadena
  `DirectorCycleStepV0 -> director-supervisor -> director-supervised-burst ->
  progressive loop` y distinguirla de `run-supervisor` global.
- `modulos/orquesta-orchestration-core/AGENTS.md` debe dejar de declarar
  review/tests/rework/cierre como pendientes genericos y explicar que el
  supervisor decide repeticion/parada, no runtime ni dispatch.
- La matriz debe remitir a pruebas locales de `director-supervisor` y
  `director-supervised-burst` sin declarar smoke residente como cerrado.
- Mantener `T64` como sincronizacion de ciclo/scheduler/runner; esta tarea solo
  agrega la capa de supervision de pasos.
- Tests: `go test -count=1 ./modulos/orquesta-director-supervisor ./modulos/orquesta-director-supervised-burst ./modulos/orquesta-orchestration-core`.

## T68 director-supervised-burst-resident-budget-smoke

Objetivo: demostrar en composicion de servidor que los presupuestos anidados de
supervisor global y rafaga del Director cortan de forma observable e idempotente.

Estado: completada.

Alcance:

- `modulos/orquesta-server`
- `cmd/orquesta-server`
- `modulos/orquesta-run-supervisor`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-director-supervisor`
- `modulos/orquesta-director-supervised-burst`
- `modulos/orquesta-outbox-dispatch`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Un servidor temporal debe ejecutar una run fake/temporal con
  `ORQUESTA_SERVER_SUPERVISOR_MAX_TICKS`,
  `ORQUESTA_SERVER_DRAIN_MAX_BURSTS` y `ORQUESTA_SERVER_DRAIN_MAX_STEPS`
  acotados, exponiendo `final_action`, `reason_code`, pasos ejecutados y outbox
  pendiente sin cerrar como `completed`.
- `stop_max_steps` y `wait_outbox` deben quedar diferenciados en auditoria,
  estado y respuesta de supervision; ninguno debe activar idle autoprogramming
  como si no hubiera trabajo.
- Repetir el tick con el mismo state dir no debe duplicar comandos, ACKs ni
  arranques de agente; si el run queda excluido por la pasada actual, debe
  explicarse con ref compacta.
- El smoke debe conservar la frontera: `run-supervisor` gobierna runs/ticks,
  `director-supervisor` gobierna pasos, y dispatch/ACK sigue en outbox.
- Tests: `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server ./modulos/orquesta-run-supervisor ./modulos/orquesta-orchestration-core ./modulos/orquesta-director-supervisor ./modulos/orquesta-director-supervised-burst ./modulos/orquesta-outbox-dispatch`.

## T69 nested-supervisor-stop-reason-contract

Objetivo: unificar la taxonomia publica de parada entre `run-supervisor`,
`director-supervisor` y `director-supervised-burst` para que auditoria, stats e
idle autoprogramming no interpreten razones locales de forma contradictoria.

Estado: cerrado 2026-05-25.

Alcance:

- `modulos/orquesta-run-supervisor`
- `modulos/orquesta-director-supervisor`
- `modulos/orquesta-director-supervised-burst`
- `modulos/orquesta-server`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Definir proyeccion publica comun para razones como `max_ticks`,
  `max_executions`, `no_execution`, `wait_outbox`, `wait_external`,
  `needs_director`, `blocked`, `stop_max_steps` y `stop_error`.
- La decision de idle autoprogramming debe distinguir capacidad libre real de
  parada por presupuesto, outbox pendiente, espera externa o error recuperable.
- Auditoria y stats no deben guardar comandos completos ni resultados enormes;
  deben publicar contadores, refs compactas y `reason_code` estable.
- Las pruebas deben cubrir combinaciones anidadas: run supervisor sin ejecucion,
  rafaga con outbox pendiente, rafaga cortada por max steps y paso con error.
- Tests: `go test -count=1 ./modulos/orquesta-run-supervisor ./modulos/orquesta-director-supervisor ./modulos/orquesta-director-supervised-burst ./modulos/orquesta-server ./cmd/orquesta-server`.

Cierre 2026-05-25:

- `modulos/orquesta-run-supervisor/stopreason` define
  `orquesta_supervisor_stop_reason_projection.v0` como proyeccion publica comun.
- `run-supervisor`, `director-supervisor` y `director-supervised-burst` publican
  `stop_projection` con categoria, reason estable, refs compactas y contadores.
- `orquesta-server` persiste stats con `last_supervisor_stop_public` y
  `last_supervisor_stop_category`, audita solo summaries compactos del tick y
  usa la proyeccion idle para automejora.
- Evidencia: `go test -count=1 ./modulos/orquesta-run-supervisor ./modulos/orquesta-director-supervisor ./modulos/orquesta-director-supervised-burst ./modulos/orquesta-server ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 vigesimoprimera pasada

Evidencia revisada: paquete del scanner de automejora, `AGENTS.md`,
`README.md`, docs vigentes de estado/nucleo/principio, este backlog, rail
errors, duplicaciones de rails, `modulos/orquesta-rails/text_policy_v0.go`,
`modulos/orquesta-core-workflow/sensitive_detail_rails_v0.go`, busquedas `rg`
en `modulos/orquesta-core-workflow/docs`, pruebas de capacidad/agente del core,
line-counts de `orquesta-domain-work-*` y `orquesta-document-plan-expander`,
AGENTS de esos modulos, `modulos/orquesta-document-plan-expander/expander_v0.go`
y busquedas `rg` sobre `ExpectedArtifactType`, `expected_artifact_type`,
`draft_content_block`, `generate_visual_asset`, `review_*`,
`validate_topic` y `assemble_topic`. No se programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- La politica viva permite vocabulario operativo opaco y corta solo valores
  sensibles efectivos, pero docs locales de `orquesta-core-workflow` siguen
  diciendo que comandos/eventos rechazan `runtime`, `provider`, `DB`, `HOME` o
  `modelo` como palabras genericas. Eso puede inducir a futuros agentes a
  reintroducir rails estrictos ya cerrados.
- El rail de tamano de fichero Go ya existe como pendiente general, pero quedan
  focos fuera de `app-director-service`/`orchestration-core`/stack:
  `orquesta-domain-work-sql/store_v0.go` supera 500 lineas,
  `orquesta-document-plan-expander/expander_v0.go` ronda 480 y los
  `job_creator_v0.go` de file/memory mezclan validacion, mutacion, filtros,
  snapshot/clone o idempotencia.
- El mapa `work_kind -> expected_artifact_type` esta duplicado en el expander
  documental, stack Codex, bridge OPES y tests de servidor. Si se anade un
  work kind o alias, una ruta puede pedir `assembled_topic` y otra entregar
  `work_delivery` sin que exista propietario unico neutral.

## T70 core-workflow-docs-rail-policy-sync

Objetivo: sincronizar los docs locales de `orquesta-core-workflow` con la
politica viva de rails permisivos por campo, sin cambiar la frontera de core.

Estado: cerrado el 2026-05-25 para factory/web/MCP: la validacion de
integraciones pasa por una politica de capacidad/proveedor, deja de cortar
vocabulario operativo por substrings y conserva errores publicos recuperables.

Alcance:

- `modulos/orquesta-core-workflow/docs`
- `modulos/orquesta-core-workflow`
- `modulos/orquesta-rails`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Los contratos/pruebas/decisiones locales no deben decir que `runtime`,
  `provider`, `DB`, `SQL`, `HOME`, `modelo` o `Codex` son prohibidos por si
  mismos; deben distinguir refs opacas validas de valores sensibles efectivos.
- La documentacion debe citar la politica comun de `orquesta-rails` y conservar
  el corte fuerte para `api_key=`, `client_secret=`, `authorization: Bearer`,
  rutas privadas, prompts/transcripts crudos y payloads masivos.
- Las pruebas existentes que permiten detalles operativos opacos deben quedar
  como evidencia, no como excepcion contradictoria.
- No relajar validadores ni tocar adaptadores concretos desde este cambio
  documental salvo que una prueba demuestre divergencia real.
- Tests: `go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-rails`.

Evidencia 2026-05-25: los docs locales de `orquesta-core-workflow` citan la
politica comun de `orquesta-rails` y sustituyen prohibiciones genericas por el
corte real: refs opacas y vocabulario operativo pasan; `api_key=`,
`client_secret=`, `authorization: Bearer`, rutas privadas,
prompts/transcripts crudos y payloads masivos siguen bloqueados. Se actualizan
contratos, pruebas, decisiones y tarea local `NCW-081` sin tocar validadores ni
adaptadores concretos.

## T71 domain-work-adapters-file-budget-split

Objetivo: partir y baselinar ficheros grandes de adaptadores `domain_work` y
expander documental sin cambiar contratos neutrales ni meter DB/producto en el
nucleo.

Estado: completada en corte focal 2026-05-25.

Evidencia adicional 2026-05-26: `store_v0.go`, `job_creator_v0.go`, contract
tests compartidos y tests locales de memory/file/sql quedaron partidos por
responsabilidad y por debajo de 300 lineas por fichero, conservando contrato
neutral, fingerprints, idempotencia y filtros.

Alcance:

- `modulos/orquesta-domain-work`
- `modulos/orquesta-domain-work-memory`
- `modulos/orquesta-domain-work-file`
- `modulos/orquesta-domain-work-sql`
- `modulos/orquesta-document-plan-expander`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Separar responsabilidades en ficheros menores: validacion/normalizacion,
  idempotencia, mutacion, filtros/listado, snapshot/clone y dialecto SQL.
- Mantener `orquesta-domain-work` puro: sin filesystem, SQL, HTTP, OPES, Codex,
  runtime, drivers ni wiring de servidor.
- `orquesta-domain-work-sql` sigue driver-neutral: no registra driver, no abre
  DSN, no crea schema productivo y no se convierte en persistencia global.
- Contract tests compartidos de `DomainWorkJobRecordStorePortV0` deben seguir
  ejecutandose en memory/file/sql; el split no debe cambiar refs, fingerprints,
  idempotencia ni filtros.
- Si un fichero historico no puede bajar de 300 lineas en una pasada, dejar
  baseline explicito de no crecimiento y followup por responsabilidad.
- Tests: `go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-domain-work-memory ./modulos/orquesta-domain-work-file ./modulos/orquesta-domain-work-sql ./modulos/orquesta-document-plan-expander`.

## T72 domain-work-artifact-contract-map-owner

Objetivo: unificar el propietario del mapa neutral `work_kind ->
expected_artifact_type` para derivados documentales y entregas `domain_work`.

Estado: hecho.

Alcance:

- `modulos/orquesta-domain-work`
- `modulos/orquesta-document-plan-expander`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-opes-bridge`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Elegir un owner neutral para `draft_content_block -> content_block`,
  `generate_visual_asset -> visual_asset`, `review_*`/`validate_topic ->
  block_revision`, `expand_topic_from_summary -> topic_expansion_package` y
  `assemble_topic -> assembled_topic`.
- El owner no debe importar OPES, Codex, HTTP, MCP, runtime, DB ni servidor; los
  adaptadores consumen el mapa o una proyeccion por puerto.
- Unknown work kinds deben conservar fallback explicito (`work_delivery`) o
  issue publico, de forma identica en expander, delivery builder y bridge OPES.
- Las matrices de tests deben probar que el mismo work kind produce el mismo
  artefacto esperado en expander, stack Codex, bridge OPES y drain de servidor.
- No mover reglas editoriales OPES ni validadores de calidad de dominio al
  nucleo; solo el contrato de correspondencia artefacto/trabajo.
- Tests: `go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-document-plan-expander ./modulos/orquesta-app-codex-stack ./modulos/orquesta-opes-bridge ./cmd/orquesta-server`.

Cierre 2026-05-25: el owner neutral del mapa queda en `orquesta-domain-work`
mediante `ExpectedDomainWorkArtifactTypeForWorkKindV0`. El expander
documental, el builder de entregas `domain_work`, el bridge OPES y la prueba
focal de drain del servidor consumen ese contrato. Los desconocidos conservan
fallback explicito `work_delivery`.

## Escaneo backlog 2026-05-24 vigesimosegunda pasada

Evidencia revisada: paquete del scanner de automejora, `AGENTS.md`,
`README.md`, docs vigentes de estado/nucleo/principio/matriz, este backlog,
rail errors, duplicaciones de rails, busquedas `rg` sobre `orquesta-deploy`,
`DeploymentPlan`, `orquesta-i18n-docs`, consumidores Go de ambos modulos,
`validateConnectorNamesV0`, `request_kind=deploy`, `i18n_l10n`, textos visibles
web/factory y descriptors MCP. No se programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- `modulos/orquesta-factory/appspec_request_v0.go` conserva un rail de
  conectores por substrings (`postgres`, `sqlite`, `runtime`, `filesystem`,
  `llm`, `queue`, `deploy`, `db`). Protege contra proveedores concretos en el
  contrato base, pero puede rechazar capacidades legitimas y duplica la
  politica viva de rails por campo.
- `DeploymentPlan v0` aparece en factory, MCP y `orquesta-deploy`, pero
  `orquesta-deploy` no esta en el mapa raiz de capas ni lo consume ninguna
  composicion Go fuera del propio modulo. Las tareas de deploy generadas pueden
  quedar como docs/write-set (`docs/deploy.md`, `deploy/`) sin ejecutar el
  contrato dry-run ya existente por puerto.
- `orquesta-i18n-docs` tiene contrato/builder/validator para bundles, loader y
  docs generadas, pero no hay consumidores Go fuera del propio modulo. Web y
  factory mantienen mapas i18n locales y la foto vigente todavia declara brechas
  historicas de i18n; falta decidir si el modulo es owner activo o pieza local
  historica.

## T73 factory-connector-capability-policy-port

Objetivo: reemplazar el rail de nombres de integracion de `orquesta-factory`
por una politica semantica de capacidad/proveedor que conserve la frontera del
contrato sin falsos positivos por vocabulario operativo.

Estado: cerrado 2026-05-25.

Alcance:

- `modulos/orquesta-factory`
- `modulos/orquesta-mcp`
- `modulos/orquesta-web`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `validateConnectorNamesV0` no rechaza por substrings sueltos como
  `db`, `runtime`, `queue` o `deploy` cuando la integracion describe una
  capacidad de dominio valida o una ref opaca reparable.
- La politica distingue proveedor concreto no autorizado (`postgres`
  como seleccion de backend, SDK cloud, credencial, DSN) de una capacidad
  legitima (`cola de tareas`, `runtime metrics`, `database audit`) que el
  director/adaptador puede normalizar.
- El resultado emite errores publicos recuperables con campo y causa, sin
  listas locales divergentes de palabras prohibidas en web/MCP.
- Web/MCP preservan i18n de errores y no reinterpretan la decision con
  otra lista de terminos.
- Tests: `go test -count=1 ./modulos/orquesta-factory ./modulos/orquesta-mcp ./modulos/orquesta-web`.

Revalidacion 2026-05-25: la politica tambien corta nombres que imponen
proveedor junto a backend/capacidad (`postgres database`, `cloud sdk`) y mantiene
web/MCP como proyecciones de errores publicos sin listas locales de substrings.

## T74 deployment-plan-composition-wiring

Objetivo: conectar `DeploymentPlan v0` como contrato ejecutable de
composicion, no solo como modulo dry-run aislado ni como descriptor MCP.

Estado: cerrado el 2026-05-25 para wiring de `DeploymentPlan v0` como dry-run
de composicion. `orquesta-deploy` queda declarado como owner adaptador/contrato,
`orquesta-app-planner` puede invocarlo desde `AppSpecV0` por puerto,
`orquesta-factory` enlaza contrato y test dry-run en la microtarea de deploy, y
MCP publica roadmap/contrato con refs T74 y verificacion conjunta.

Alcance:

- `AGENTS.md`
- `README.md`
- `docs/guia_nucleo_orquestacion_2026-05-17.md`
- `modulos/orquesta-deploy`
- `modulos/orquesta-factory`
- `modulos/orquesta-app-planner`
- `modulos/orquesta-mcp`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- La foto raiz debe ubicar `orquesta-deploy` como adaptador/contrato de
  despliegue, separado de core, runtime y factory.
- Cuando `AppSpecV0` o una microtarea declarada pidan deploy, la composicion
  debe poder invocar `orquesta-deploy` por puerto dry-run y devolver refs de
  plan/evidencia, sin ejecutar Docker/Kubernetes/cloud ni tocar secretos.
- `orquesta-factory` no debe generar solo `docs/deploy.md`/`deploy/` si ya hay
  un contrato `DeploymentPlan v0` aplicable; debe enlazarlo como dependencia o
  dejar bloqueo verificable.
- MCP/roadmap deben leer el estado real del contrato, no duplicar owners o
  estados `pendiente_*` hardcodeados.
- Tests: `go test -count=1 ./modulos/orquesta-deploy ./modulos/orquesta-factory ./modulos/orquesta-app-planner ./modulos/orquesta-mcp`.

Evidencia 2026-05-25:

- `DeploymentPlanDryRunPortV0`, `DefaultDeploymentPlanDryRunPortV0` y
  `BuildDeploymentPlanForCompositionDryRunV0` conectan el contrato con refs de
  plan/evidencia sin efectos externos.
- `PrepareDeploymentPlanDryRunFromAppSpecV0` en `orquesta-app-planner` consume
  `AppSpecV0` validado y conserva deploy como adaptador opt-in.
- Las tareas de deploy de factory incluyen `contracts/deployment_plan_v0.json`
  y `tests/deployment_plan_dry_run_test.go`.

## T75 i18n-docs-active-composition-owner

Objetivo: decidir y cablear el owner activo de i18n/documentacion generada para
que `orquesta-i18n-docs` no quede como isla local mientras web/factory mantienen
mapas paralelos.

Estado: completada 2026-05-25 para owner activo de i18n/documentacion generada:
`orquesta-i18n-docs` publica `ActiveI18nDocsCompositionOwnerV0` y
`AppI18nDocsPlanV0`; factory, web y MCP consumen esa proyeccion sin crear otro
owner paralelo de bundles, loader, fallback locale, required keys ni docs
generadas.

Alcance:

- `modulos/orquesta-i18n-docs`
- `modulos/orquesta-factory`
- `modulos/orquesta-web`
- `modulos/orquesta-mcp`
- `docs/estado_actual_2026-05-17.md`
- `docs/guia_nucleo_orquestacion_2026-05-17.md`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Declarar si `orquesta-i18n-docs` es el owner vigente de bundles, loader shape
  y docs generadas, o marcarlo como historico con enlace a la fuente activa.
- Si es owner activo, factory/web/MCP deben consumir contrato o proyeccion por
  puerto en vez de mantener mapas incompatibles de claves, required keys,
  default locale y docs iniciales.
- Todo texto visible generado para apps o docs debe tener locale declarado y
  clave estable; texto hardcodeado solo queda permitido como seed que entra en
  catalogos versionados.
- La validacion debe cubrir paridad de locales/catalogos, `required_keys`,
  `required_doc_types`, `fallback_locale` y hash de loader, no solo schema
  JSON local.
- Tests: `go test -count=1 ./modulos/orquesta-i18n-docs ./modulos/orquesta-factory ./modulos/orquesta-web ./modulos/orquesta-mcp`.

Cierre 2026-05-25:

- `orquesta-i18n-docs` es owner vigente de `AppI18nDocsPlanV0`, bundles,
  loader shape, fallback locale, `required_keys`, `required_doc_types` y hash
  de loader para documentacion generada.
- Factory consume el owner con `BuildI18nDocsPlanFromAppSpecV0` y
  `I18nDocsActiveOwnerForAppSpecV0`.
- Web consume la proyeccion `NuevaAppI18nActiveOwnerProjectionV0`; su catalogo
  local sigue siendo superficie de adaptador, pero queda validado contra owner,
  fallback locale y loader hash de `orquesta-i18n-docs`.
- MCP publica `GenerarI18nDocsIniciales v0` en `orquesta.contracts.shared.v0`
  con owner `orquesta-i18n-docs`, guardas de fallback/loader hash y backlog ref
  T75.
- MCP publica `GenerarI18nDocsIniciales v0` desde la proyeccion activa, sin
  duplicar owner ni declarar otro contrato de i18n.

## Escaneo backlog 2026-05-24 vigesimotercera pasada

Evidencia revisada: paquete del scanner de automejora, `AGENTS.md`,
`README.md`, docs vigentes de estado/nucleo/principio/matriz, este backlog,
rail errors, duplicaciones de rails, `modulos/orquesta-app-runner`,
`modulos/orquesta-app-planner`, `modulos/orquesta-app-director-intake`,
`modulos/orquesta-app-director-service`,
`modulos/orquesta-director-agent-workflow`,
`modulos/orquesta-app-change-director-source`, herramientas MCP
`orquesta.apps.preparar_orquestacion.v0`,
`orquesta.apps.arrancar_director.v0` y
`orquesta.director.human_work.review_plan.v0`, documento
`docs/corte_plan_state_director_decisions_2026-05-21.md`, busquedas `rg`
sobre `AppSpecV0`, `app-runner`, `director_decisions`, `plan_ref`,
`forbiddenAutoPlanFragmentsV0` y `OperationalDirectorPlanStateV0`. No se
programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- La superficie MCP conserva dos entradas cercanas para apps:
  `orquesta.apps.preparar_orquestacion.v0` delega en `orquesta-app-runner` y
  devuelve un `AppPlan` determinista, mientras
  `orquesta.apps.arrancar_director.v0` delega en `app-director-service` y abre
  el flujo con director. La foto vigente dice que Orquesta piensa mediante
  director; sin routing/freshness explicito, una IA puede elegir el camino
  historico de plan fijo como si fuera el camino operativo principal.
- `orquesta-app-change-director-source/readiness_v0.go` mantiene una lista
  local de fragments prohibidos (`runtime`, `provider`, `model`, `db`, `sql`,
  `codex`, etc.) y `external_work_v0.go` reescribe vocabulario operativo en
  criterios. Esto evita algunos falsos positivos, pero tambien puede ocultar
  semantica de dominio o bloquear refs opacas fuera del scope ya cubierto por
  `T15`.
- El corte de `director_decisions` ya crea un `OperationalDirectorPlanStateV0`
  inicial desde tareas marcadas, pero el propio documento de corte deja
  pendiente fusionar nuevas tasks operativas en un plan state existente cuando
  una decision posterior abre otra ola. El codigo de ensure actual retorna si el
  state existe, por lo que ese caso necesita prueba/contrato propio antes de
  declarar cerrado el flujo de decisiones tardias.

## T76 appspec-entrypoints-director-v2-routing

Objetivo: unificar la ruta publica de `AppSpecV0` para que el camino operativo
por defecto arranque Director V2 y el flujo `app-runner`/`app-planner` quede
marcado como compatibilidad, preview o adaptador historico con frontera clara.

Estado: cerrado 2026-05-25; refrescado 2026-05-26. La ruta publica preferente
para `AppSpecV0` es `orquesta.apps.arrancar_director.v0`. Los entrypoints
`orquesta.apps.preparar_orquestacion.v0` y
`orquesta.apps.ejecutar_orquestacion.v0` quedan como preview/compatibilidad,
publican `route_policy` en resultado y descriptor, y `ejecutar_orquestacion`
expone `require_director_v2` para bloquear con `director_v2_required` cuando el
caller necesita Director V2, plan-state, waits, review/tests/cierre o recursion.
Evidencia: `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-runner ./modulos/orquesta-app-planner ./modulos/orquesta-app-director-intake ./modulos/orquesta-app-director-service`.

Alcance:

- `modulos/orquesta-mcp`
- `modulos/orquesta-app-runner`
- `modulos/orquesta-app-planner`
- `modulos/orquesta-app-director-intake`
- `modulos/orquesta-app-director-service`
- `docs/estado_actual_2026-05-17.md`
- `docs/guia_nucleo_orquestacion_2026-05-17.md`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `orquesta.apps.arrancar_director.v0` debe quedar documentada como entrada
  operativa preferente para apps nuevas cuando se necesita juicio del Director.
- `orquesta.apps.preparar_orquestacion.v0` debe declarar si es preview,
  compatibilidad legacy o preparacion sin ejecucion; no debe aparentar cierre
  operativo, plan-state ni runtime si solo devuelve `AppPlan`.
- Si `app-runner` sigue siendo ejecutable, debe enlazar con
  `OperationalDirectorPlanStateV0` o bloquear con razon publica cuando el
  objetivo requiera Director V2, waits por ola/cohorte, review/tests/cierre o
  recursion.
- Web/MCP/CLI no deben duplicar routing ni exponer `CandidateProvider` como
  contrato publico; solo refs compactas, estado y errores publicos.
- Tests: `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-runner ./modulos/orquesta-app-planner ./modulos/orquesta-app-director-intake ./modulos/orquesta-app-director-service`.

Cierre 2026-05-25:

- `orquesta-app-runner` publica `route_policy` de preview/compatibilidad y
  conserva `orquesta.apps.arrancar_director.v0` como entrypoint preferente.
- `RunPreparedAppOrchestrationV0` acepta `require_director_v2` y bloquea antes
  de efectos externos con `director_v2_required` cuando el caller declara que
  necesita plan-state, waits por ola/cohorte, review/tests/cierre o recursion.
- MCP propaga `route_policy` en `preparar_orquestacion`, `ejecutar_orquestacion`
  y `arrancar_director`, lo declara en descriptores compactos y no expone
  `CandidateProvider` como contrato publico.
- Validacion focal: `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-runner ./modulos/orquesta-app-planner ./modulos/orquesta-app-director-intake ./modulos/orquesta-app-director-service`.

## T77 app-change-director-source-readiness-rail-policy

Objetivo: sustituir la lista local de readiness/autoplan de
`orquesta-app-change-director-source` por una politica por campo alineada con
`orquesta-rails`, sin perder la guarda contra secretos efectivos ni efectos
externos no autorizados.

Estado 2026-05-25: cerrado para T77. La readiness de
`orquesta-app-change-director-source` ya no mantiene lista local de fragments
prohibidos y delega en `orquesta-rails` por campo.

Revision 2026-05-26: el retry de Orquesta V2 valida que el contrato vigente es
permitir vocabulario operativo opaco y refs de dominio, bloquear solo detalle
sensible/raw efectivo, y mantener el contexto `ref_only` como evidencia de ACK
cuando el paquete no materializa documento local.

Alcance:

- `modulos/orquesta-app-change-director-source`
- `modulos/orquesta-app-change`
- `modulos/orquesta-rails`
- `modulos/orquesta-app-codex-stack`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `forbiddenAutoPlanFragmentsV0` no debe bloquear por vocabulario operativo
  opaco (`runtime`, `provider`, `model`, `db`, `sql`, `codex`, `docker`,
  `home`) cuando aparece en refs, criterios o nombres de modulo reparables.
- La sanitizacion de criterios no debe reemplazar terminos de dominio de forma
  que oculte la causa real al director; debe emitir issues/refs estructurados o
  redaccion por campo cuando haya valores sensibles efectivos.
- Mantener bloqueo fuerte para `api_key=`, `client_secret=`,
  `authorization: Bearer`, passwords, DSN con credenciales, prompts/transcripts
  crudos, rutas privadas y efectos externos no autorizados.
- Cubrir casos `external_work` documentales, visuales y cambios de app con
  vocabulario operativo normal, sin introducir OPES/Codex en el modulo.
- Tests: `go test -count=1 ./modulos/orquesta-app-change-director-source ./modulos/orquesta-app-change ./modulos/orquesta-rails ./modulos/orquesta-app-codex-stack`.

Evidencia 2026-05-25: `orquesta-app-change-director-source` elimino la lista
local de fragments prohibidos para readiness y delega en la politica por campo
de `orquesta-rails`. El autoplanning permite vocabulario operativo opaco en
criterios, refs y nombres reparables, conserva terminos de dominio sin
reescritura y bloquea detalle sensible efectivo o raw (`api_key=`,
`client_secret=`, `authorization: Bearer`, DSN con credenciales, rutas privadas,
`prompt=` y `transcript=`). La cobertura focal incluye cambios de app,
`external_work` documental y visual.

Evidencia 2026-05-26: el comando requerido de retry queda bloqueado por
compilacion de `orquesta-observability` fuera del write-set cerrado. Primer
intento observado: `WorkspaceTimelineV0`/`WorkspaceTimelineSourceStatusV0` no
definidos al construir la dependencia transitiva de `orquesta-app-codex-stack`.
Reintento posterior: `cloneWorkspaceTimelineProjectionV0` aparece redeclarado
entre `workspace_timeline_filter_v0.go` y `workspace_timeline_clone_v0.go`.
Retry 2026-05-26 en esta tarea: hubo un fallo externo transitorio al compilar
`orquesta-web` contra el DTO vigente de
`orquesta-observability.WorkspaceTimelineItemV0`, fuera del write-set T77, pero
la reejecucion posterior del comando requerido paso completa. Los paquetes T77
directos `orquesta-app-change-director-source`, `orquesta-app-change` y
`orquesta-rails` compilaron en ambos intentos, y T77 queda cerrado sin reabrir
la politica de rails.
Validacion OrquestaV2 2026-05-26: el paquete `ref_only` se resolvio por
evidencia explicita en ACK y el comando requerido paso completo dentro del
write-set T77.

## T78 director-decisions-existing-planstate-merge

Objetivo: fusionar decisiones tardias del director en un
`OperationalDirectorPlanStateV0` existente cuando `director_decisions` abre una
nueva ola, sin recrear el state ni esperar agentes fuera de scope.

Estado: cerrado el 2026-05-26 por revalidacion OrquestaV2. El comando
obligatorio paso completo dentro del write-set T78 y el contexto `ref_only`
requerido quedo resuelto por lectura local del paquete y evidencia explicita en
ACK.

Alcance:

- `modulos/orquesta-director-agent-workflow`
- `modulos/orquesta-app-director-service`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-state-file`
- `modulos/orquesta-app-codex-stack`
- `docs/corte_plan_state_director_decisions_2026-05-21.md`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Si existe un plan state abierto y aparece una `WorkflowTaskV0` nueva marcada
  con `operational_director.task_source: director_decision`, el servicio debe
  anadir un paso/ola causal o reabrir `wait_subagents` solo para esa task y sus
  `agent_refs`.
- La fusion debe preservar `plan_ref`, `task_ref`, `wave_ref`/`cohort_ref` o
  `parent_task_ref`, `required_tests`, write-set, criterios, eventos causales y
  wait scope; no debe reconstruir desde stats ni desde agentes vivos globales.
- Reaplicar la misma decision o reentrar tras reinicio no debe duplicar steps,
  wait refs, tasks ni comandos de outbox.
- Si la decision no trae metadata suficiente para merge seguro, bloquear con
  issue publico y followup reparable; no cerrar el plan como `completed`.
- Tests: `go test -count=1 ./modulos/orquesta-director-agent-workflow ./modulos/orquesta-app-director-service ./modulos/orquesta-orchestration-core ./modulos/orquesta-state-file ./modulos/orquesta-app-codex-stack`.

## Escaneo backlog 2026-05-24 vigesimocuarta pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`, `README.md`,
`docs/estado_actual_2026-05-17.md`,
`docs/guia_nucleo_orquestacion_2026-05-17.md`,
`docs/principio_orquesta_piensa_director.md`, este backlog, rail errors,
duplicaciones de rails, `docs/op_096_control_total_estado_proyecto_y_estadisticas.md`,
`cmd/orquesta-server/idle_self_improvement_backlog_planner_v0.go`,
`cmd/orquesta-server/domain_work_stack_v0.go`,
`modulos/orquesta-domain-work-http/client_v0.go`,
`modulos/orquesta-domain-work/README.md` y
`modulos/orquesta-observability/README.md`. No se programa codigo desde este
scanner. `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- El backlog de automejora y los documentos de rails ya son monolitos grandes:
  `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md` supera miles de
  lineas, mientras el planner residente lee un unico documento hardcodeado. T43
  cubre lease/merge de scanners concurrentes, pero no una estructura de shards
  que reduzca colisiones y conserve parser, hashes y lineas de evidencia.
- El adaptador `orquesta-domain-work-http` queda opt-in por entorno y valida
  esquema/base/path, pero no declara una politica de egress por host/scope,
  allowlist, modo smoke temporal ni redaccion de destino. Es distinto de OPES:
  el conector neutral podria apuntar por error a endpoints no temporales o redes
  internas si una composicion lo configura mal.
- `OP 096` documenta que el control total por agente/proyecto ya existe y que
  el hueco vivo esta en agregacion global de workspace, timeline temporal y
  coste. `orquesta-observability` tiene eventos/diagnostico compactos y el
  servidor tiene auditoria/estado, pero falta contrato unico para responder por
  agente, proyecto y workspace sin reconstruccion ad hoc por shell, transcript
  crudo o estadisticas locales divergentes.

## T79 autoprogramming-backlog-doc-sharding

Objetivo: partir el backlog documental de automejora/rails en indice y shards
append-only sin perder compatibilidad del planner residente ni evidencia de
linea/hash para scanners concurrentes.

Estado: completada para contrato inicial de indice/shards residente
2026-05-25. Migrar fisicamente tareas a shards nuevos queda como trabajo
posterior con write-set propio; este cierre no borra ni trunca historico.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-autoprogramming`
- `modulos/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Indice vivo definido arriba: backlog historico, rail errors y duplicaciones
  quedan declarados como shards append-only actuales.
- El planner lee el indice si existe y mantiene compatibilidad con el documento
  historico si no existe indice. Cada seccion `Txx` conserva `section.Ref`,
  `backlog_doc:<fichero>`, linea, `BacklogScanDocumentV0.path/start_line/sha256`
  y hash de shard para lease/ACK.
- T43 puede comparar hashes por shard mediante `backlog_scan_doc:<path>:line:<n>:sha256:<hash>`
  y pedir rebase al director cuando el ACK traiga foto documental obsoleta.
- El parser sigue aceptando el formato actual durante migracion y queda cubierto
  por prueba focal con shard externo al documento historico.
- Tests pasados 2026-05-25:
  `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-autoprogramming ./modulos/orquesta-server`.
- Revalidacion OrquestaV2 2026-05-25: el paquete estricto con contexto
  `ref_only` requerido queda resuelto por lectura local y recibo explicito en
  ACK; no se borra, mueve ni trunca ningun documento durante este cierre.

## T80 domain-work-http-egress-policy

Objetivo: cerrar una politica de egress opt-in para el adaptador HTTP neutral
de `DomainWork`, separada de OPES y sin meter HTTP/red en el contrato puro.

Estado: cerrado 2026-05-26. `orquesta-domain-work-http` clasifica destino,
rechaza credenciales, exige politica explicita desde
`ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL` en composicion, permite `smoke_local` solo
para loopback temporal y `allowlist` por host/puerto, y conserva paths relativos
sin query ni host override. La revalidacion transversal completa pasa con
`go test -count=1 ./...`.

Alcance:

- `modulos/orquesta-domain-work-http`
- `modulos/orquesta-domain-work`
- `modulos/orquesta-mcp`
- `modulos/orquesta-app-gateway`
- `cmd/orquesta-server`
- `docs/runbooks`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL` pasa por politica explicita de
  destino: esquema permitido, host/puerto allowlist o modo smoke temporal,
  rechazo de credenciales en URL y errores publicos compactos.
- La politica permite smokes locales temporales de forma declarada, pero
  bloquear destinos ambiguos como metadata services, redes internas no
  allowlisted o endpoints productivos sin confirmacion de composicion.
- Las rutas `create_job` y `submit_artifact` siguen siendo paths
  relativos controlados; no deben transportar query sensible ni sobreescribir
  host.
- Auditoria/MCP/web solo exponen refs, host clasificado y estado; no persisten
  URL completa con secretos, headers, payloads de dominio crudos ni cuerpos de
  respuesta.
- Tests 2026-05-26: `go test -count=1 ./modulos/orquesta-domain-work-http`
  pasa y la revalidacion transversal requerida queda cubierta por
  `go test -count=1 ./...`.

## T81 observability-global-workspace-timeline

Objetivo: definir y cablear una timeline global de workspace para control total
server-first, reutilizando observability/auditoria/stats por puertos y sin
reconstruccion ad hoc desde shell o transcript crudo.

Estado: completado 2026-05-25.

Alcance:

- `modulos/orquesta-observability`
- `modulos/orquesta-server`
- `modulos/orquesta-mcp`
- `modulos/orquesta-web`
- `cmd/orquesta-server`
- `docs/op_096_control_total_estado_proyecto_y_estadisticas.md`
- `docs/auditoria_runtime_orquesta_2026-05-24.md`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Un contrato de consulta debe cubrir `agent_ref`, `project_ref`, `task_ref`,
  ventana temporal y scope `workspace`, con paginacion y sources declaradas.
- API/MCP/web deben consumir el mismo puerto de lectura; no deben recomponer
  estado leyendo shell, git local o ficheros internos de runtime directamente.
- La timeline combina eventos, auditoria, run queue, runtime/progress, Git stats
  y uso/coste cuando exista puerto T29; si una fuente falta, devuelve
  `not_available` y evidencia de fuente ausente, no datos inventados.
- La clasificacion de transcript debe ser compacta y redactada; prompts,
  transcripts crudos, HOME, rutas privadas, tokens y payloads HTTP quedan fuera
  de la respuesta por defecto.
- Tests: `go test -count=1 ./modulos/orquesta-observability ./modulos/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-web ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 vigesimoquinta pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`, `README.md`,
docs vigentes de estado/nucleo/principio, este backlog, rail errors,
duplicaciones de rails, `modulos/orquesta-governance`,
`modulos/orquesta-cli/governance_catalog_client_v0.go`,
`modulos/orquesta-cli/operational_status_client_v0.go`,
`modulos/orquesta-cli/function_contract_client_v0.go`,
`modulos/orquesta-observability/operational_status_*`,
`modulos/orquesta-mcp/operational_status_resource_v0.go`,
`modulos/orquesta-mcp/shared_contracts_descriptors_v0.go`,
`modulos/orquesta-core/docs/contratos.md`,
`modulos/orquesta-core-workflow/function_contract_publish_*.go` y rutas
registradas en `modulos/orquesta-http-gateway/gateway_v0.go`. No se programa
codigo desde este scanner. `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
solo como refs opacas.

Huecos concretos nuevos:

- `orquesta-governance` ya incluye handler HTTP para
  `/api/v0/governance/catalog/query`, pero la CLI envia un body plano
  `{module, role, phase, tags}` y espera `GovernanceCatalogQueryResultV0`
  directo con errores `errores/codigo`, mientras el handler espera
  `{filters:{...}}`, responde envelope `result` y errores `errors/code`.
  Ademas el gateway/servidor no cablean esa ruta.
- `OperationalStatusQueryV0` esta promovido en observability, CLI, web y MCP
  como contrato read-only con endpoint recomendado
  `/api/v0/operational-status/query`, pero hoy solo hay adapter en memoria y
  resource descriptivo MCP; no hay handler publico ni source residente que
  derive `DiagnosticoCompactoV0` desde stats/run queue/runtime con redaccion.
- La CLI consume rutas candidatas de solo lectura
  `/api/v0/core/function-contracts/list` y
  `/api/v0/core/function-contracts/view`, mientras `core` conserva la consulta
  como candidata documental y `core-workflow` solo publica refs compactas de
  contratos en eventos/runs. Falta indice read-only real por puerto y wiring de
  gateway antes de que CLI/MCP/web traten FunctionContract como consulta viva.

## T82 governance-catalog-public-route-shape-sync

Objetivo: sincronizar shape, errores y wiring publico de la consulta read-only
de `GovernanceCatalog v0` sin activar reglas historicas ni meter gobernanza en
el core.

Estado: completado 2026-05-25.

Alcance:

- `modulos/orquesta-governance`
- `modulos/orquesta-cli`
- `modulos/orquesta-mcp`
- `modulos/orquesta-http-gateway`
- `modulos/orquesta-app-gateway`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Elegir un unico contrato HTTP para request, response y errores publicos de
  `/api/v0/governance/catalog/query`; CLI, handler, MCP docs y tests deben usar
  el mismo shape.
- La salida expone solo `effective` y contadores de `proposed`/`quarantine`;
  ninguna ruta activa historicos, lee DB v1 productiva, muta reglas ni concede
  permisos.
- El servidor/gateway cablea la ruta solo si existe provider inyectado; si falta
  provider devuelve error publico recuperable, no fallback a docs locales ni
  datos inventados.
- La correlacion/request id viajan sin filtrar tokens, rutas HOME, rowids DB v1,
  prompts, transcripts ni catalogos completos de cuarentena.
- Tests: `go test -count=1 ./modulos/orquesta-governance ./modulos/orquesta-cli ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.

Evidencia 2026-05-25:

- Shape publico unificado: request `{request_id, correlation_id, filters}`,
  response `{request_id, correlation_id, effective, counters}` y errores
  `{request_id, correlation_id, errors:[{code, field}]}`.
- `orquesta-cli`, `orquesta-governance`, descriptor MCP compartido, gateway
  HTTP, app-gateway y `cmd/orquesta-server` usan el mismo contrato; no se
  exponen `current_block`, `inactive_blocks`, DB v1, prompts, transcripts ni
  cuarentena completa.
- Si falta provider, la ruta responde
  `governance_catalog_source_unavailable` como error publico recuperable, sin
  fallback a docs locales ni datos inventados.

## T83 operational-status-public-query-source

Objetivo: publicar `/api/v0/operational-status/query` sobre un source residente
compacto y redactado, reutilizando `OperationalStatusQueryV0` sin construir la
timeline global completa de T81.

Estado: completada el 2026-05-25.

Cierre 2026-05-25: `/api/v0/operational-status/query` queda publicado por
source residente del servidor y handler comun inyectable en gateway/app gateway.
La proyeccion deriva `DiagnosticoCompactoV0` desde estado vivo del residente,
supervisor y cola compacta disponible; no expone paths/runtime dirs y declara
fuentes no agregadas como `not_available` mediante warnings. CLI, web, MCP y
REST consumen el mismo contrato. T81 sigue reservado para timeline global
historica.

Alcance:

- `modulos/orquesta-observability`
- `modulos/orquesta-server`
- `modulos/orquesta-cli`
- `modulos/orquesta-web`
- `modulos/orquesta-mcp`
- `modulos/orquesta-http-gateway`
- `modulos/orquesta-app-gateway`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Definir un puerto de lectura para `OperationalStatusQueryV0` que el servidor
  pueda poblar desde stats, cola, runs, runtime/progress y auditoria ya
  disponibles; fuentes ausentes producen `not_available`/warning, no datos
  inventados.
- Gateway/API, CLI, web y MCP consumen el mismo contrato; no duplican
  diagnosticos locales ni leen stores, shell, Git, runtime dirs o transcripts.
- La respuesta cumple `ValidateDiagnosticoCompactoV0`, conserva paginacion/
  limites compactos y redaccion por campo; vocabulario operativo opaco como
  `runtime`, `provider` o `model` no bloquea, valores sensibles efectivos si.
- T81 queda para workspace timeline y agregacion historica; esta tarea cierra el
  query minimo vivo que los clientes ya anuncian.
- Tests: `go test -count=1 ./modulos/orquesta-observability ./modulos/orquesta-server ./modulos/orquesta-cli ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.

## T84 function-contract-readonly-public-index

Objetivo: convertir las operaciones candidatas `listar/ver FunctionContractV0`
en una consulta read-only real por puerto y gateway, sin promover registro
mutante ni reconstruir contratos desde texto libre.

Estado: cerrado 2026-05-26.

Alcance:

- `modulos/orquesta-core`
- `modulos/orquesta-core-workflow`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-state-file`
- `modulos/orquesta-cli`
- `modulos/orquesta-mcp`
- `modulos/orquesta-http-gateway`
- `modulos/orquesta-app-gateway`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Definir store/index read-only de `FunctionContractV0` o resumen derivado de
  eventos `FunctionContractPublished` con refs, version y estado; si solo hay
  ref compacta y falta payload contractual, devolver bloqueo verificable.
- Publicar `/api/v0/core/function-contracts/list` y
  `/api/v0/core/function-contracts/view` con filtros acotados, cursor opaco y
  errores publicos; no crear, registrar, archivar ni reemplazar contratos desde
  estas rutas.
- CLI/MCP/web deben tratar `registrar` como bloqueado hasta que exista flujo
  durable por OrchestrationRun/CommandHandler/Outbox y decision del director.
- La consulta no lee docs historicos ni DB v1 como fuente canonica; puede citar
  refs documentales como evidencia, pero el estado vivo sale de stores/eventos
  inyectados.
- Tests: `go test -count=1 ./modulos/orquesta-core ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core ./modulos/orquesta-state-file ./modulos/orquesta-cli ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.

Cierre 2026-05-25: `orquesta-state-file` implementa un indice read-only desde
eventos `FunctionContractPublished`; lista refs causales con estado
`evidencia_insuficiente` cuando solo existe payload compacto y bloquea `view`
con error publico verificable. Las rutas
`/api/v0/core/function-contracts/list` y
`/api/v0/core/function-contracts/view` quedan publicadas por gateway/HTTP y el
servidor residente las monta desde el store causal inyectado. `registrar`
sigue bloqueado en CLI/MCP como operacion no promovida.

## Escaneo backlog 2026-05-24 vigesimosexta pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`, `README.md`,
docs vigentes de estado/nucleo/principio/matriz, este backlog, rail errors,
duplicaciones de rails, `modulos/orquesta-server/handler_v0.go`,
`modulos/orquesta-server/state_v0.go`,
`modulos/orquesta-server/status_tracker_v0.go`,
`modulos/orquesta-cli/server_status_client_v0.go`,
`modulos/orquesta-cli/bootstrap_appspec_client_v0.go`,
`modulos/orquesta-cli/docs/contratos.md`,
`modulos/orquesta-cli/docs/tareas.md`,
`modulos/orquesta-http-gateway/gateway_v0.go`, scripts de smoke que esperan
`/healthz` y busquedas `rg` sobre `server/status`, `bootstrap/appspec`,
`healthz`, `startup_ready` y rutas `/api/v0/apps/spec`. No se programa codigo
desde este scanner. `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
solo como refs opacas.

Huecos concretos nuevos:

- `/api/v0/server/status` existe y la CLI lo consume como fuente publica, pero
  `StateV0` expone `project_work_dir` y `runtime_work_dir` absolutos y no
  transporta un snapshot explicito/redactado de los valores efectivos que la
  propia CLI documenta como necesario para auditar automejora residente. Falta
  separar estado publico compacto, config efectiva por campos permitidos y refs
  opacas para no convertir el endpoint en fuga de paths locales.
- La CLI conserva `BootstrapAppSpecCliEndpointV0` en
  `/api/v0/director/bootstrap/appspec`, mientras web/gateway usan
  `/api/v0/apps/spec` para preview de `AppSpec` y `/api/v0/apps/director` para
  arranque con Director. No hay ruta gateway ni composition root visible para
  ese endpoint legacy, y T76 no debe cerrarse sin decidir si se elimina,
  redirige, bloquea o migra como compatibilidad explicita.
- Los smokes y daemon usan `/healthz` como "servidor listo", pero el handler lo
  responde solo con `{"status":"ok"}` y la readiness real vive en
  `startup_ready/startup_status` dentro de `/api/v0/server/status`. Falta un
  contrato unico de liveness/readiness para que scripts y supervisor residente
  no arranquen trabajos antes de terminar startup check, cleanup o
  reconciliacion.

## T85 server-status-config-snapshot-redaction

Objetivo: separar el estado publico del servidor y el snapshot efectivo de
configuracion residente, con redaccion por campo y sin exponer paths locales ni
internals como contrato normal de CLI/web/MCP.

Estado: cerrado 2026-05-26. Contrato implementado y documentado:
`/healthz` queda como liveness, `/api/v0/server/readiness` como readiness
operativa con `startup_ready`, `startup_status`, mensaje publico y evidence
refs. Daemon y smokes que preparan runs, drenan OPES, lanzan Codex, ejecutan
Director/domain_work o automejora esperan readiness; los checks de `/healthz`
restantes son liveness de socket o apps externas temporales.

Alcance:

- `modulos/orquesta-server`
- `modulos/orquesta-cli`
- `modulos/orquesta-web`
- `modulos/orquesta-mcp`
- `modulos/orquesta-http-gateway`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Definir que campos de `/api/v0/server/status` son estado publico, cuales son
  config efectiva visible y cuales deben viajar solo como refs opacas o
  `redacted`.
- `project_work_dir`, `runtime_work_dir`, state dir, HOME, comandos, paths de
  runtime, proveedor/modelo real y tokens no deben salir como valores crudos en
  la respuesta publica por defecto.
- La CLI puede mostrar `configuracion_residente_no_visible` cuando el servidor
  no publique un campo, pero no reconstruye valores leyendo entorno local,
  statefile, runtime dirs, Git ni docs historicos.
- Si hace falta modo operador-local con mas detalle, debe ser opt-in de
  composicion, con redaccion, test de ausencia de secretos y auditoria compacta.
- Tests: `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-cli ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./cmd/orquesta-server`.

## T86 bootstrap-appspec-legacy-route-quarantine

Objetivo: cerrar la ruta legacy CLI
`/api/v0/director/bootstrap/appspec` como compatibilidad explicita o migrarla a
los endpoints vigentes de `AppSpec`/Director, sin dejar un cliente que apunta a
una ruta no cableada.

Estado: completada 2026-05-26.

Alcance:

- `modulos/orquesta-cli`
- `modulos/orquesta-director`
- `modulos/orquesta-factory`
- `modulos/orquesta-web`
- `modulos/orquesta-mcp`
- `modulos/orquesta-http-gateway`
- `modulos/orquesta-app-gateway`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Inventariar consumidores de `BootstrapProyectoDesdeAppSpec` y decidir si el
  contrato sigue vivo, queda en cuarentena legacy o se reemplaza por
  `/api/v0/apps/spec` y `/api/v0/apps/director`.
- Si sigue vivo, gateway/servidor deben cablearlo con shape, errores publicos y
  tests verticales; si no, la CLI debe devolver bloqueo publico estable y docs
  deben apuntar al flujo vigente.
- No reintroducir `app-runner`/plan determinista como camino operativo de juicio
  cuando el objetivo requiere Director V2, waits, review, tests requeridos o
  cierre causal.
- El cambio debe coordinarse con T76 para no duplicar entrypoints ni dejar dos
  rutas publicas con semantica incompatible para la misma solicitud.
- Tests: `go test -count=1 ./modulos/orquesta-cli ./modulos/orquesta-director ./modulos/orquesta-factory ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.

Cierre 2026-05-26: inventario acotado confirma que
`BootstrapProyectoDesdeAppSpec` sigue vivo como contrato puro local en
`orquesta-director`, web y MCP, pero no como ruta HTTP publica del servidor. La
CLI deja `app spec bootstrap` en cuarentena legacy: devuelve
`contrato_no_configurado` con campo `route_policy`, no llama
`/api/v0/director/bootstrap/appspec`, no requiere `server_url` para informar el
bloqueo y rechaza credenciales si se configuran. El flujo vigente de AppSpec con
juicio operativo queda en `/api/v0/apps/director` /
`orquesta.apps.arrancar_director.v0`, coordinado con T76 sin reintroducir
`app-runner` como camino operativo.

## T87 server-liveness-readiness-contract

Objetivo: distinguir liveness (`healthz`) de readiness/startup operativo antes
de que smokes, daemon o supervisor residente lancen trabajo real.

Estado: completada 2026-05-26.

Evidencia de cierre 2026-05-26:

- `/healthz` queda limitado a liveness y `/api/v0/server/readiness` gobierna
  readiness operativa del servidor Orquesta.
- `cmd/orquesta-server` espera readiness y no acepta `/healthz` como listo.
- Los smokes de Orquesta que necesitan servidor operativo esperan readiness;
  `smoke_orquesta_server_restart_state.sh` se corrige para no arrancar el caso
  de persistencia de cola con solo liveness.
- La respuesta de readiness redacta mensajes de startup que puedan contener
  paths, runtime dirs, prompts, transcripts o payloads.

Alcance:

- `modulos/orquesta-server`
- `modulos/orquesta-cli`
- `cmd/orquesta-server`
- `scripts`
- `docs/runbooks`
- `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `/healthz` queda documentado como liveness del proceso, no como autorizacion
  para preparar runs, drenar OPES, lanzar Codex ni ejecutar automejora.
- Publicar o reutilizar una lectura de readiness que incluya `startup_ready`,
  `startup_status`, cleanup/reconciliacion y bloqueos recuperables; scripts de
  smoke deben esperar esa senal cuando el caso necesita servidor operativo.
- Si readiness no esta disponible en una composicion legacy, el script debe
  bloquear con error publico verificable o degradar a liveness solo para casos
  que no lanzan trabajo externo.
- La respuesta no debe exponer paths, runtime dirs, prompts, transcripts ni
  payloads de startup; solo estado compacto y evidence refs.
- Tests de cierre: `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-cli ./cmd/orquesta-server` y `bash -n scripts/*.sh`.

## Escaneo backlog 2026-05-24 vigesimoseptima pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`, `README.md`,
docs vigentes de estado/nucleo/principio, este backlog, rail errors,
duplicaciones de rails,
`cmd/orquesta-server/idle_self_improvement_backlog_planner_v0.go`,
`modulos/orquesta-autoprogramming/docs/tareas.md`,
`modulos/orquesta-runtime-codex-delivery/docs/tareas.md`,
`modulos/orquesta-server-shutdown/docs/tareas.md`,
`modulos/orquesta-director-operativo/README.md`,
`modulos/orquesta-director-operativo/docs/tareas.md` y busquedas `rg` sobre
`CODEX-WAVE-REAL`, `CODEX-RECURSION-REAL`, backlog local y `ref_only`.
No se programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- El planner residente de automejora lee un unico backlog hardcodeado
  (`docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`) y los scanners
  escriben solo tres documentos globales. Mientras tanto, varios modulos
  conservan backlog local en `docs/tareas.md` o README (`APG-*`,
  `RTDELIVERY-*`, `SSH-*`, `DEP-*`, etc.). Sin indice federado, esas entradas
  pueden quedar invisibles para el residente o duplicarse como Txx sin alias,
  owner, linea/hash ni freshness.
- Algunas fuentes locales que los agentes leen por regla obligatoria siguen
  contradiciendo la foto vigente del Director Operativo. En particular,
  `modulos/orquesta-director-operativo/README.md` declara pendientes
  `wait`, review/rework/replan/cierre durable y recursion Codex real, mientras
  los documentos vigentes ya cierran `WaitAgentRefs`, ciclo offline y
  `CODEX-WAVE-REAL`/`CODEX-RECURSION-REAL`; el frente real abierto sigue siendo
  OPES temporal de derivados/cierre y nuevos blockers demostrados.

## T88 federated-module-backlog-index

Objetivo: hacer visible el backlog local de modulos para automejora residente
sin convertir documentos historicos en tareas ejecutables ambiguas ni duplicar
Txx ya existentes.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-autoprogramming`
- `modulos/orquesta-mcp`
- `modulos/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Definir un indice federado de backlog que registre `source_path`,
  `source_line`, hash, tipo de fuente, owner, estado, alias local y Txx
  relacionado para entradas de `modulos/*/docs/tareas.md`, runbooks y docs
  globales vigentes.
- El planner residente debe seguir leyendo el backlog global, pero puede
  considerar entradas locales solo cuando esten marcadas como vigentes o
  promocionadas a Txx; docs historicos o stale producen aviso/freshness, no
  trabajo automatico.
- Si una entrada local ya esta cubierta por Txx, registrar alias y evidencia en
  vez de crear otra tarea. Si falta owner, alcance o tests, crear consulta o
  tarea de clasificacion antes de programar codigo.
- La lectura de shards/backlog local debe interoperar con T43/T44/T79:
  reserva/epoch, evidencia de lectura de contexto requerido y soporte de
  indice/shards sin pisar escrituras concurrentes.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-autoprogramming ./modulos/orquesta-mcp ./modulos/orquesta-server`.

Evidencia 2026-05-25: `cmd/orquesta-server` define un indice federado
de fuentes locales (`source_path`, `source_kind`, owner, estado, alias, tests),
lee solo fuentes `vigente`/`promocionada`, genera secciones locales con
`backlog_doc`, alias, owner, hash y lease/epoch por documento, y convierte
fuentes sin tests/owner en revision documental antes de programar codigo. El
scanner tambien incluye fuentes federadas en su lease. La prueba obligatoria
queda bloqueada antes de ejecutar estos paquetes porque
`modulos/orquesta-observability` no compila en el worktree actual
(`WorkspaceTimelineV0` indefinido), fuera del alcance cerrado de T88.

## T89 director-operativo-local-doc-state-sync

Objetivo: sincronizar los documentos locales y obligatorios del Director
Operativo con la foto vigente, sin reabrir WaitAgentRefs ni smokes Codex ya
cerrados.

Estado: hecho.

Alcance:

- `modulos/orquesta-director-operativo/README.md`
- `modulos/orquesta-director-operativo/docs/contratos.md`
- `modulos/orquesta-director-operativo/docs/tareas.md`
- `modulos/orquesta-director-operativo/docs/pruebas.md`
- `docs/director_operativo_v1_2026-05-17.md`
- `docs/corte_operational_director_plan_state_v0_2026-05-17.md`
- `docs/corte_cierre_generico_director_operativo_2026-05-17.md`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Sustituir afirmaciones stale que declaran pendientes `wait` por cohorte/ola,
  review/rework/replan/cierre durable basico, ola Codex real amplia o recursion
  Codex real por el estado vigente: cerrados donde hay evidencia, pendientes
  solo OPES temporal real de derivados/cierre y blockers nuevos demostrables.
- Mantener clara la frontera: `orquesta-director-operativo` sigue siendo
  contrato puro; materializacion, waits, plan-state, cierre y smokes reales
  viven en composiciones/puertos.
- Anadir una prueba documental o check focal que busque contradicciones
  conocidas (`CODEX-WAVE-REAL`, `CODEX-RECURSION-REAL`, OPES derivados/cierre)
  en fuentes obligatorias y docs locales del modulo.
- No borrar historia: marcar contenido historico o stale con enlace a fuente
  vigente, sin eliminar docs antiguos ni reordenar cortes.
- Tests: `go test -count=1 ./modulos/orquesta-director-operativo` y check
  documental focal de contradicciones conocidas.

Evidencia 2026-05-26: los docs locales del Director Operativo declaran el
modulo como contrato puro, enlazan el mapa
`OperationalDirectorPlanV0 -> OperationalDirectorWaveWorkV0 -> WorkflowTaskV0`
y sustituyen contradicciones conocidas: `WaitAgentRefs`/wait por ola-cohorte,
ciclo offline de review/tests/cierre, `CODEX-WAVE-REAL` y
`CODEX-RECURSION-REAL` quedan cerrados salvo regresion demostrada. El pendiente
real vigente queda acotado a OPES temporal real de derivados/cierre. Se anadio
`TestDirectorOperativoLocalDocsAlineadosConFotoVigenteV0` como check focal.

## Escaneo backlog 2026-05-24 vigesimoctava pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`, `README.md`,
docs vigentes de estado/nucleo/principio/matriz, este backlog, rail errors,
duplicaciones de rails, `rg` sobre `TODO|pendiente|falta` en `modulos`, `cmd`,
`docs` y `scripts`, line-count de ficheros Go bajo `modulos`/`cmd`,
line-count de `scripts/*.sh`, docs locales de `orquesta-run-control`,
`orquesta-app-gateway`, `orquesta-domain-work`,
`orquesta-agent-process-registry` y busquedas sobre `run-control`,
`agent_ack`, `healthz`, smokes y line-budget. No se programa codigo desde este
scanner. `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- T45 fija el baseline general de ficheros Go y T52-T54/T71 cubren los focos
  mayores ya identificados, pero el escaneo de lineas deja un segundo grupo de
  ficheros grandes fuera de esos shards: `orquesta-runtime-codex-delivery`,
  `orquesta-runtime-required-test`, `orquesta-run-coordinator`,
  `orquesta-external-work-run`, `orquesta-app-gateway`, `orquesta-director`,
  `orquesta-core-workflow` y algunos tests de `cmd/orquesta-server`. Sin shard
  residual, el rail de 300 lineas queda como advisory generico y no como trabajo
  ejecutable por modulo.
- Los smokes largos concentran bootstrap de servidor temporal, confirmaciones,
  readiness, cleanup, parseo de stats y validaciones Python/shell en ficheros de
  400-800 lineas (`smoke_external_domain_non_opes_real.sh`,
  `smoke_external_domain_fake_real.sh`, `smoke_autoprogramming_supervised.sh`,
  `smoke_codex_required_test_runner_state_file.sh`,
  `smoke_opes_derivatives_rest.sh`). T21/T87 gobiernan guardas/readiness, pero
  falta un owner de libreria comun para no duplicar rails de confirmacion,
  redaccion y espera operativa en cada script.

## T90 residual-go-file-budget-splits

Objetivo: convertir el segundo grupo de ficheros Go grandes en shards de
refactor acotados con baseline de no crecimiento, sin mezclarlo con los frentes
ya cubiertos por T52-T54/T71.

Estado: pendiente.

Alcance:

- `modulos/orquesta-runtime-codex-delivery`
- `modulos/orquesta-runtime-required-test`
- `modulos/orquesta-run-coordinator`
- `modulos/orquesta-external-work-run`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-director`
- `modulos/orquesta-core-workflow`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Medir lineas desde el arbol real y declarar baseline por fichero antes de
  editar; no usar `ACK.files` como unica prueba de tamano.
- Partir por responsabilidad local: source/validacion/review gate en runtime
  Codex delivery, executor/redaccion en required-test, coordinator flow, tests
  gateway y harnesses de director/core-workflow.
- No cambiar contratos publicos, refs, eventos, JSON tags, errores ni orden
  causal; el primer corte debe ser mecanico o con pruebas focales equivalentes.
- Si un fichero historico no baja de 300 lineas en una pasada, dejar baseline de
  no crecimiento y followup por responsabilidad concreta.
- Mantener T52-T54/T71 como shards prioritarios; esta tarea cubre el residuo que
  no pertenece a esos owners.
- Tests: `go test -count=1 ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-runtime-required-test ./modulos/orquesta-run-coordinator ./modulos/orquesta-external-work-run ./modulos/orquesta-app-gateway ./modulos/orquesta-director ./modulos/orquesta-core-workflow ./cmd/orquesta-server`.

## T91 smoke-script-ops-library

Objetivo: extraer helpers comunes para smokes reales/temporales largos sin
relajar guardas de confirmacion, readiness ni redaccion.

Estado: pendiente.

Alcance:

- `scripts`
- `scripts/lib`
- `docs/runbooks`
- `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Extraer funciones reutilizables para prerequisitos, confirmaciones opt-in,
  espera de liveness/readiness, arranque/parada de servidor temporal, cleanup,
  parseo JSON compacto y redaccion de salidas.
- Conservar por script la politica de riesgo: Codex real, OPES temporal, DB/red
  o efectos externos siguen requiriendo confirmacion explicita y limites bajos.
- No convertir `/healthz` en readiness operativa; coordinar con T87 para que los
  helpers distingan socket vivo de servidor listo para lanzar efectos externos.
- Los helpers no deben imprimir HOME, rutas privadas, prompts, transcripts,
  tokens, payloads HTTP completos ni salidas crudas extensas por defecto.
- El split no debe cambiar el comportamiento observable de los smokes cerrados;
  cualquier cambio funcional debe ir en tarea separada con caso focal.
- Tests: `bash -n scripts/*.sh scripts/lib/*.sh` y pruebas focales de los
  smokes afectados con confirmaciones fake/temporales cuando existan.

## Escaneo backlog 2026-05-24 vigesimonovena pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`, `README.md`,
docs vigentes de estado/nucleo/principio/matriz, este backlog, rail errors,
duplicaciones de rails, `rg` sobre capacidad, toolbelt, prompt hints,
`directed_query`, `CapacityDecisionExecutorV0`, `RequestCapacityDecision`,
docs locales de `orquesta-capacity`, `orquesta-runtime`,
`orquesta-orchestration-core`, `orquesta-app-codex-stack`,
`orquesta-mcp`, `orquesta-operator-mcp` y `cmd/orquesta-server`. No se
programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- `RequestCapacityDecision` ya sale por outbox y el stack lo despacha con
  `CapacityDecisionExecutorV0`, pero ese executor decide con configuracion
  estatica/default (`Tier`, `ReasoningEffort`) y no consume como puerto la
  politica/fixtures de `orquesta-capacity`. T51 cubre defaults de razonamiento
  y T29 cubre uso/cuota, pero falta el owner de la decision de capacidad como
  politica ejecutable por composicion. Ademas, `orquesta-capacity/docs/pruebas.md`
  mantiene casos iniciales `Comando: pendiente` que ya conviven con DTOs,
  fixtures y tests Go; ese estado documental puede reabrir trabajo cerrado o
  ocultar la falta real de wiring.
- Las instrucciones de toolbelt para agentes estan duplicadas entre
  `cmd/orquesta-server/codex_prompt_hints_v0.go`, runbooks/paquetes y registros
  MCP. El hint MCP enumera `orquesta.operator.operations.v0` pero no
  `orquesta.operator.directed_query.v0`, mientras otra linea recomienda usar
  `directed_query` si falta puente humano. Como el tool existe bajo
  `orquesta-operator-mcp`, la deuda no es crear otro contrato, sino fijar una
  fuente de verdad/test para que prompts, paquetes y tool registry no diverjan.

## T92 capacity-decision-policy-port

Objetivo: conectar la decision de capacidad del outbox con una politica
ejecutable por puerto, reutilizando `orquesta-capacity` como contrato/politica
de composicion sin meter proveedor, HOME, cuota real ni modelo concreto en el
nucleo.

Estado: pendiente.

Alcance:

- `modulos/orquesta-capacity`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-runtime`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `CapacityDecisionExecutorV0` debe delegar en un puerto de politica/capacidad
  cuando este inyectado; el fallback estatico solo queda como modo legacy o
  fake documentado y auditable.
- La politica debe aceptar `CapacityDecisionRequestV0` del core y producir
  `RegisterCapacityDecision` con tier, reasoning, pool/model/quota refs opacas,
  motivos y evidencias compactas; no decide desde strings de prompt ni desde
  proveedor hardcodeado.
- Integrar el contrato de `orquesta-capacity` sin que `orquesta-core-workflow`,
  `orquesta-runtime` ni `orquesta-domain-work` importen adaptadores, HOME,
  OAuth, tokens, cuota real ni runtime concreto.
- Sincronizar `modulos/orquesta-capacity/docs/pruebas.md`: marcar historicos
  los casos CAP-CT ya cubiertos por DTO/fixtures/tests Go, y dejar como
  pendiente solo el wiring por puerto, quota source real o benchmarks reales.
- Coordinar con T51: scanners/automejora documental conservan `medium` salvo
  override con evidencia; OPES/temarios reales y arquitectura amplia pueden
  pedir `high`/`xhigh` con razon auditable.
- Tests: `go test -count=1 ./modulos/orquesta-capacity ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime ./cmd/orquesta-server`.

## T93 codex-prompt-toolbelt-source-sync

Objetivo: tener una fuente verificable para toolbelt/prompts de agentes Codex,
derivada de rutas y tools registrados, para no entregar instrucciones
contradictorias o stale a agentes externos.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-mcp`
- `modulos/orquesta-operator-mcp`
- `modulos/orquesta-app-gateway`
- `docs/runbooks`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Las listas `Toolbelt HTTP` y `Toolbelt MCP` usadas en prompts/packets deben
  salir de un descriptor comun o de un test que compare contra rutas/tools
  registrados; no mantener listas libres divergentes.
- Si un prompt recomienda `orquesta.operator.directed_query.v0`, ese tool debe
  aparecer como capability disponible o como subtool de
  `orquesta.operator.operations.v0` con error publico cuando falta puerto.
- El descriptor debe distinguir tool disponible, conector opt-in no configurado
  y transporte real no arrancado; no prometer red, operador humano, Codex,
  OPES ni DB si la composicion no los inyecto.
- Los hints no deben incluir rutas locales, HOME, tokens, prompts completos,
  transcripts ni payloads de control. Solo nombres publicos, metodos, paths,
  refs de contrato y errores recuperables.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-app-gateway`.

## Escaneo backlog 2026-05-24 trigesima pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`, `README.md`,
docs vigentes de estado/nucleo/principio, este backlog, rail errors,
duplicaciones de rails, `rg` sobre auditoria/readiness/toolbelt/capacidad y
lectura focal de `modulos/orquesta-server/audit_v0.go` y
`audit_http_v0.go`. No se programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Hueco concreto nuevo:

- `RuntimeV0.auditEventV0` descarta el error de `AppendAuditEventV0`. Si el
  sink JSONL queda sin permisos, sin espacio, corrupto o bloqueado por IO, el
  servidor puede seguir preparando automejora, supervisor ticks, startup checks
  o mutaciones HTTP sin evidencia durable ni diagnostico publico del fallo de
  auditoria. T19 cubre superficie/retencion y T41 cubre redaccion de payloads,
  pero falta un rail propio para visibilidad de fallo de escritura sin crear
  recursion de auditoria ni filtrar rutas locales.

## T94 server-audit-write-failure-visibility

Objetivo: hacer visible y verificable el fallo de escritura de auditoria del
servidor residente sin bloquear liveness ni persistir detalles sensibles.

Estado: pendiente.

Alcance:

- `modulos/orquesta-server`
- `cmd/orquesta-server`
- `docs/auditoria_runtime_orquesta_2026-05-24.md`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `auditEventV0` no debe ignorar silenciosamente errores del sink; debe exponer
  un estado compacto por puerto/proyeccion del servidor con contador, ultimo
  codigo publico, evento afectado y timestamp redactado.
- Evitar recursion: un fallo de auditoria no se intenta auditar escribiendo en
  el mismo sink fallido. El recibo de fallo vive en estado/proyeccion compacta
  o sink alternativo inyectado por composicion.
- Definir severidad por evento: liveness puede seguir respondiendo, pero
  mutaciones de automejora/supervisor/control deben advertir o bloquear segun
  politica inyectada; no cerrar trabajos como completados si la auditoria
  obligatoria estaba fallando y era criterio de cierre.
- No devolver rutas locales, HOME, permisos exactos, payloads HTTP, prompts,
  transcripts ni tokens en errores publicos; solo codigos como
  `audit_write_failed`.
- Coordinar con T19/T41: esta tarea cubre visibilidad de fallo de escritura,
  no schema de payload ni visor de auditoria.
- Tests: `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 trigesimoprimera pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`, `README.md`,
docs vigentes de estado/nucleo/principio, este backlog, rail errors,
duplicaciones de rails, `rg` sobre persistencia de estado/auditoria HTTP y
lectura focal de `modulos/orquesta-server/runtime_v0.go`,
`startup_check_v0.go`, `supervisor_loop_v0.go` y `audit_http_v0.go`. No se
programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Hueco concreto nuevo:

- `RuntimeV0.persistStateV0` descarta errores de `saveStateV0` en checks de
  arranque, ticks de supervisor, automejora idle, shutdown y marca de error.
  `RunV0` ya devuelve error si falla al guardar `serving`, pero las
  transiciones posteriores pueden quedar solo en memoria. Si el state file
  queda sin permisos, corrupto, sin espacio o bloqueado por IO, el servidor
  puede seguir operando y exponer `/api/status` vivo mientras un reinicio,
  CLI o lectura de store ve estado obsoleto, sin codigo publico de
  `state_persist_failed`. T94 cubre fallo de auditoria; este es el rail
  separado de persistencia de estado operativo.

## T95 server-state-persist-failure-visibility

Objetivo: hacer visible y verificable el fallo de persistencia del estado del
servidor residente sin depender de auditoria JSONL ni filtrar detalles locales.

Estado: completada local el 2026-05-26.

Cierre local 2026-05-26: `orquesta-server` registra los fallos no fatales de
`StateStorePortV0` en una proyeccion compacta `state_persist_*` del `StateV0`.
`persistStateV0` conserva compatibilidad y delega en una ruta con transicion
causal; si el store falla, el estado vivo queda `degraded` con
`state_persist_failed`, contador, timestamp y transicion sin exponer paths ni el
error crudo. Una escritura posterior confirmada marca `state_persist_status=ok`
sin borrar el contador historico. El source residente de operational-status
degrada salud, anade blocker redactado y expone contador compacto.

Evidencia local: `go test -count=1 ./modulos/orquesta-server`.

Alcance:

- `modulos/orquesta-server`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `persistStateV0` no debe ignorar silenciosamente errores del `StateStore`; debe
  actualizar una proyeccion compacta en memoria con contador, ultimo codigo
  publico, transicion afectada y timestamp redactado.
- Separar severidad por transicion: guardar `serving` puede seguir siendo fatal
  para arranque; supervisor ticks, automejora idle y shutdown deben quedar como
  estado degradado visible y no como exito durable implicito.
- `/api/status`, CLI/status y pruebas de startup deben distinguir estado vivo en
  memoria de estado durable confirmado. No cerrar ni preparar automejora como
  evidencia durable si la persistencia requerida estaba fallando.
- No intentar reparar el fallo escribiendo en el mismo store fallido ni depender
  de auditoria JSONL para saber que el store de estado fallo; coordinar con T94
  solo para observabilidad secundaria.
- No devolver rutas locales, HOME, permisos exactos, payloads HTTP, prompts,
  transcripts ni tokens en errores publicos; usar codigos como
  `state_persist_failed`.
- Tests: `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 trigesimosegunda pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`, `README.md`,
docs vigentes de estado/nucleo/principio, este backlog, rail errors,
duplicaciones de rails, `rg` sobre daemon, stop, PID, logs, bridge input ledger
y lectura focal de `cmd/orquesta-server/daemon.go`, `commands.go`,
`shutdown_client.go`, `opes_bridge.go`, `opes_bridge_submit.go`,
`external_bridge_input_ledger.go` y `modulos/orquesta-server/runtime_v0.go`.
No se programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- `orquesta-server stop` lee PID/addr desde el state file, pide shutdown con
  `forced=true` y despues senala el PID del snapshot. T30 cubre checkpoint
  neutral, T35 cubre cleanup de arranque y T60 cubre stop de olas Codex por PID,
  pero falta el rail del daemon residente: validar identidad de proceso/epoch y
  reconciliar shutdown cooperativo antes de senalar un PID que puede ser stale o
  reutilizado.
- `orquesta-server start` redirige stdout/stderr del daemon a `stdout.log` y
  `stderr.log` bajo `StateDir`, sin propietario de redaccion, rotacion,
  retencion ni proyeccion publica. T41 cubre auditoria JSONL, T58 logs Codex y
  T61 salida de tests requeridos; falta el owner de logs operacionales del
  daemon para no convertirlos en otra persistencia cruda paralela.
- El ledger de entrada del bridge externo se consulta antes de crear la run y
  se escribe despues de recibir `run_ref`. Si el submit funciona pero el ledger
  falla, o si dos drains compiten por el mismo job externo, puede quedar una run
  creada sin claim durable o una entrada sobrescrita. T31 cubre reserva de cola
  interna; falta claim/recovery por job externo antes del efecto de crear run.

## T96 server-daemon-stop-process-identity

Objetivo: hacer que `orquesta-server stop` valide identidad de proceso y modo de
shutdown antes de senalar un PID desde estado durable potencialmente obsoleto.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-server`
- `modulos/orquesta-server-shutdown`
- `modulos/orquesta-agent-process-registry`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- El state del daemon debe llevar descriptor/epoch verificable: pid, addr,
  started_at, executable/process_ref o evidencia equivalente, sin exponer rutas
  privadas como verdad publica.
- `stop` debe comprobar que el servidor HTTP vivo corresponde al snapshot antes
  de senalar PID; si hay mismatch, bloquear con error publico recuperable en vez
  de matar un proceso potencialmente ajeno.
- `forced=true` no debe ser default silencioso del CLI cuando el runtime puede
  hacer shutdown cooperativo; forced requiere opt-in, causa publica y evidencia
  de que no quedan checkpoints/agentes vivos o de que el operador lo pidio.
- La salida publica no debe filtrar HOME, rutas locales, argv completos,
  prompts, transcripts, tokens ni detalles del sistema operativo.
- Coordinar con T30/T35/T60: esta tarea cubre el daemon residente, no el
  checkpoint neutral, cleanup de arranque ni stop de ola Codex.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-server-shutdown ./modulos/orquesta-agent-process-registry`.

## T97 server-daemon-log-redaction-retention

Objetivo: definir propietario, redaccion y retencion de `stdout.log`/`stderr.log`
del daemon residente para que no dupliquen auditoria, tail Codex ni evidencias
de tests requeridos.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-server`
- `modulos/orquesta-observability`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Los logs del daemon deben tener politica explicita de tamano, rotacion,
  retencion y acceso local; no pueden crecer sin limite ni usarse como evidencia
  terminal de ACK, delivery, cierre o auditoria.
- Por defecto, exponer solo resumen compacto/redactado en status/diagnostico;
  fragmento crudo queda como opt-in local con limite fuerte y causa publica.
- Redactar o bloquear HOME, rutas privadas, env, comandos completos, remotos
  Git, payloads HTTP, prompts, transcripts, tokens y salida cruda de proveedor.
- Coordinar con T41/T58/T61/T85: auditoria JSONL, logs Codex, required-test
  output y status/config snapshot conservan owners separados.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-observability`.

## T98 external-bridge-input-ledger-claim-recovery

Objetivo: convertir el ledger de entrada de bridges externos en claim/recovery
causal por job externo antes de crear runs, empezando por OPES sin meter OPES en
el nucleo.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-opes-bridge`
- `modulos/orquesta-opes-connector`
- `modulos/orquesta-run-queue`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Antes de llamar a `/api/v0/external-work/run`, el bridge debe adquirir claim
  durable por `external_system` + `external_job_ref` + idempotency/correlation;
  otro drain concurrente debe ver `claimed/submitted` y no crear segunda run.
- Si el submit devuelve `run_ref` pero falla la escritura del ledger, recovery
  debe reconciliar por idempotency_key/refs de run antes de reintentar; no basta
  con registrar `submitted_ledger_error` en summary.
- El ledger no debe sobrescribir una entrada `submitted` con otro `run_ref`
  salvo decision de recuperacion explicita y auditada.
- Errores publicos deben usar codigos compactos (`external_bridge_claim_failed`,
  `external_bridge_recovery_required`) sin URL sensible, payload OPES completo,
  rutas locales, HOME, tokens ni respuestas crudas.
- OPES sigue siendo adaptador opt-in; el contrato general debe servir a otros
  bridges externos por refs opacas.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector ./modulos/orquesta-run-queue`.

## Escaneo backlog 2026-05-24 trigesimotercera pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`, `README.md`,
docs vigentes de estado/nucleo/principio/matriz, este backlog, rail errors,
duplicaciones de rails, `rg` sobre HTTP decode/timeouts, archivos de revision
de startup, ledgers `domain_work`, y lectura focal de
`modulos/orquesta-server/runtime_v0.go`,
`modulos/orquesta-mcp/autoprogramming_prepare_run_http_v0.go`,
`modulos/orquesta-mcp/domain_work_http_v0.go`,
`modulos/orquesta-mcp/run_control_http_v0.go`,
`cmd/orquesta-server/startup_check_compaction.go`,
`cmd/orquesta-server/startup_check_compaction_runtime.go`,
`modulos/orquesta-app-codex-stack/domain_work_delivery_bridge_v0.go` y
`modulos/orquesta-app-codex-stack/domain_work_delivery_file_ledger_v0.go`.
No se programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- El servidor residente crea `http.Server` solo con `Handler`, sin timeouts
  explicitos de lectura/escritura/idle. Varias rutas HTTP/MCP mutables
  (`prepare-run`, `domain_work`, `run_control` y equivalentes) decodifican
  `r.Body` directamente sin helper comun de limite de tamano, trailing-token
  check ni politica uniforme de campos desconocidos. El transporte MCP real ya
  usa un limite local con `io.LimitReader`, pero esa regla no gobierna el resto
  de la superficie HTTP.
- La compactacion de startup escribe snapshots `before`, `removed`, `manifest`
  y mueve runtime dirs a una revision con directorios/ficheros de modo amplio,
  ademas de proyectar la ruta de revision en el mensaje de readiness. T35 cubre
  si se debe compactar; T59 cubre purga de runtime Codex; falta owner de
  redaccion, permisos, retencion y proyeccion publica de esos artefactos de
  revision.
- La entrega de artefactos `domain_work` consulta el ledger por
  `idempotency_key`, ejecuta `submit_artifact` y solo despues registra accepted
  o rejected. Si el submit externo funciona pero la escritura del ledger falla,
  o dos drains compiten por la misma entrega, puede duplicarse el artefacto o
  sobrescribirse una decision de idempotencia. T98 cubre el ledger de entrada
  antes de crear runs; falta el ledger de salida de artefactos.

## T99 server-http-resource-guardrails

Objetivo: unificar limites de recursos HTTP del servidor residente y de los
handlers MCP/HTTP publicos sin meter producto ni transporte concreto en el
nucleo.

Estado: pendiente.

Alcance:

- `modulos/orquesta-server`
- `modulos/orquesta-mcp`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-web`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `RuntimeV0.RunV0` debe configurar timeouts de servidor adecuados para
  loopback/residente: `ReadHeaderTimeout`, lectura de body, escritura e idle,
  con defaults seguros y override por composicion si procede.
- Las rutas mutables deben decodificar JSON con helper comun: limite de body por
  perfil, error publico `request_body_too_large`, rechazo de trailing tokens y
  politica documentada para campos desconocidos.
- Los limites deben distinguir control plane, MCP JSON-RPC, domain_work y
  autoprogramacion; no usar un numero magico local por handler salvo que quede
  conectado a politica comun.
- La auditoria debe guardar solo tamano/codigo/endpoint/correlacion, no body
  crudo, prompts, transcripts, tokens, HOME, rutas privadas ni payload externo
  completo.
- Coordinar con T55/T80/T87: esta tarea cubre recursos de transporte HTTP local,
  no autenticacion remota, politica de egress ni readiness semantica.
- Tests: `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-web ./cmd/orquesta-server`.

## T100 startup-revision-archive-redaction-retention

Objetivo: hacer seguros y verificables los artefactos de revision generados por
compactacion de startup sin convertirlos en otra persistencia cruda paralela.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-server`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-run-control`
- `modulos/orquesta-state-file`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Los snapshots de revision (`before`, `removed`, manifest y runtime archivado)
  deben tener permisos restrictivos, retencion explicita, tamano maximo y
  redaccion por campo antes de conservarlos.
- La proyeccion publica de readiness/status debe devolver un `revision_ref`
  opaco y contadores, no rutas locales ni nombres de directorio como evidencia
  operativa.
- Archivar runtime dirs no debe copiar prompts, transcripts, stdout/stderr,
  payloads de proveedor ni control files completos a una zona que luego entre en
  contexto, auditoria, AppVCS o promocion.
- Si la revision necesita material crudo para diagnostico local, debe requerir
  opt-in, limite fuerte y causa publica; no debe ser default de startup.
- Coordinar con T35/T50/T59/T97: esta tarea no decide si compactar ni como
  parar agentes; solo gobierna redaccion, permisos, retencion y exposicion del
  archivo de revision.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-run-control ./modulos/orquesta-state-file`.

## T101 domain-work-artifact-submission-ledger-recovery

Objetivo: convertir el ledger de salida de artefactos `domain_work` en una
frontera de claim/recovery causal para `submit_artifact`.

Estado: pendiente.

Alcance:

- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-domain-work`
- `modulos/orquesta-opes-bridge`
- `modulos/orquesta-opes-connector`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Antes de invocar `submit_artifact`, el stack debe registrar claim durable por
  `idempotency_key` + `run_ref` + `task_ref` + `delivery_ref`; otro drain debe
  ver `claimed/submitting/submitted` y no repetir el efecto externo.
- Si `submit_artifact` devuelve receipt pero falla el ledger, recovery debe
  reconciliar por idempotency/correlation/receipt antes de reintentar; no basta
  con perder el registro y volver a enviar.
- El ledger no debe sobrescribir `accepted` con otro receipt ni `rejected` con
  otro payload salvo decision causal explicita; debe detectar conflicto publico.
- Required tests de dominio deben consumir solo records causales y estables:
  accepted/rejected con refs de entrega, receipt/evidence/issue y timestamp
  redactado, no URL, DB, ruta local ni nombre de conector como prueba.
- Errores publicos compactos: `domain_work_submit_claim_failed`,
  `domain_work_submit_recovery_required` y `domain_work_submit_conflict`, sin
  payload OPES completo, respuestas crudas, HOME, rutas locales, tokens ni
  prompts/transcripts.
- Coordinar con T14/T72/T98: esta tarea cubre idempotencia y recovery de salida,
  no mapa de tipos de artefacto ni claim de jobs externos de entrada.
- Tests: `go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-domain-work ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 trigesimocuarta pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`, `README.md`,
docs vigentes de estado/nucleo/principio/matriz, este backlog, rail errors,
duplicaciones de rails, `rg` sobre decode HTTP, `ReadAll`, clientes HTTP y
escrituras file-based, y lectura focal de
`modulos/orquesta-factory-http/appspec_http_v0.go`,
`modulos/orquesta-governance/governance_catalog_public_query_v0.go`,
`modulos/orquesta-mcp/autoprogramming_prepare_run_http_v0.go`,
`modulos/orquesta-mcp/domain_work_http_v0.go`,
`modulos/orquesta-opes-connector/http_v0.go`,
`modulos/orquesta-domain-work-http/client_v0.go`,
`modulos/orquesta-cli/governance_catalog_client_v0.go`,
`modulos/orquesta-cli/function_contract_client_v0.go`,
`modulos/orquesta-run-file/atomic_json_v0.go`,
`modulos/orquesta-server/file_state_store_v0.go`,
`modulos/orquesta-domain-work-file/snapshot_v0.go`,
`modulos/orquesta-runtime-codex-delivery/file_descriptor_store_v0.go` y
`modulos/orquesta-app-codex-stack/domain_work_delivery_file_ledger_v0.go`.
No se programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- T99 cubre recursos HTTP del servidor residente, pero quedan handlers publicos
  y legacy con reglas propias: `orquesta-factory-http` lee todo `r.Body`,
  governance aplica `DisallowUnknownFields` y trailing-token check local, y MCP
  replica decoders por ruta sin limite comun. Si el planner usa estas fronteras
  como equivalentes, puede haber limites, shape de error y politica de campos
  desconocidos distintos segun adaptador.
- Los clientes HTTP salientes decodifican o leen respuestas con reglas
  divergentes: OPES y `domain-work-http` decodifican `response.Body` sin limite
  de respuesta; CLI/governance y function-contract usan `io.ReadAll`; comandos
  de servidor leen cuerpos completos para status/diagnostico. T80 decide host y
  egress de `domain_work`; falta el contrato de respuesta: tamano, status,
  content-type, trailing tokens y redaccion de errores.
- Los stores file-based no comparten contrato durable: `orquesta-run-file` y
  `orquesta-domain-work-file` hacen temp unico, sync y sync de directorio;
  `orquesta-server` y `orquesta-runtime-codex-delivery` usan `path+".tmp"` sin
  fsync/dir sync; el ledger de artefactos crea directorio `0755` y no sincroniza
  antes/despues de `rename`. T95/T98/T101 cubren errores visibles y ledgers
  concretos, pero no el owner comun de escritura durable, permisos y concurrencia
  entre procesos.

## T102 legacy-http-json-boundary-policy

Objetivo: unificar la frontera JSON de handlers HTTP publicos y legacy que
quedan fuera del primer scope de T99, sin meter HTTP en el nucleo puro.

Estado: pendiente.

Alcance:

- `modulos/orquesta-factory-http`
- `modulos/orquesta-governance`
- `modulos/orquesta-mcp`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-http-gateway`
- `modulos/orquesta-web`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Todos los handlers publicos deben usar una politica comun o documentada por
  perfil para limite de body, trailing-token check, `Content-Type`, campos
  desconocidos y error publico.
- `orquesta-factory-http`, governance y MCP no deben conservar reglas
  incompatibles si exponen endpoints equivalentes del control plane.
- La respuesta de error debe preservar correlacion y devolver codigos compactos,
  no body crudo, prompts, transcripts, rutas locales, HOME, tokens ni detalles
  internos de adaptador.
- Si una ruta necesita aceptar campos extra por compatibilidad, debe declararlo
  como modo legacy con test y no como omision accidental del helper comun.
- Coordinar con T99: T99 cubre timeouts/recursos del servidor residente; esta
  tarea cubre adaptadores HTTP publicos que quedaron con decoders locales.
- Tests: `go test -count=1 ./modulos/orquesta-factory-http ./modulos/orquesta-governance ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway ./modulos/orquesta-web ./cmd/orquesta-server`.

## T103 outbound-http-response-limit-redaction

Objetivo: fijar una politica comun para respuestas HTTP salientes consumidas por
conectores, CLI, web y comandos de servidor.

Estado: pendiente.

Alcance:

- `modulos/orquesta-opes-connector`
- `modulos/orquesta-domain-work-http`
- `modulos/orquesta-cli`
- `modulos/orquesta-web`
- `modulos/orquesta-app-gateway`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Los clientes deben aplicar limite de respuesta por perfil antes de `Decode` o
  `ReadAll`, con error publico `response_body_too_large` o equivalente local.
- El manejo de status no-2xx debe descartar o leer con limite y redaccion; nunca
  exponer cuerpos crudos, URL con credenciales, HOME, tokens, prompts,
  transcripts, payload OPES completo ni respuestas de proveedor.
- Las respuestas JSON deben comprobar content-type cuando proceda, trailing
  tokens y shape esperado sin convertir errores de decode en diagnostico crudo.
- OPES y `domain_work` conservan conectores separados: T80 decide egress/host
  del HTTP neutral; esta tarea gobierna la frontera de respuesta para todos los
  clientes salientes.
- Tests: `go test -count=1 ./modulos/orquesta-opes-connector ./modulos/orquesta-domain-work-http ./modulos/orquesta-cli ./modulos/orquesta-web ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.

## T104 file-store-durable-write-policy

Objetivo: unificar la politica de escritura durable para stores file-based de
estado, runs, descriptors, ledgers y snapshots.

Estado: pendiente.

Alcance:

- `modulos/orquesta-run-file`
- `modulos/orquesta-domain-work-file`
- `modulos/orquesta-runtime-codex-delivery`
- `modulos/orquesta-server`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Cada writer durable debe declarar temp unico o lock/claim, permisos de archivo
  y directorio, `fsync`/close, `rename`, sync de directorio y limpieza de temp
  interrumpido, o justificar por que no lo necesita.
- Stores compartidos por servidor, CLI, runtime o drains concurrentes no deben
  depender solo de mutex en memoria ni de `path+".tmp"` si puede haber mas de un
  proceso.
- Los snapshots no deben quedar con permisos amplios ni copiar control files,
  prompts, transcripts, stdout/stderr, HOME, rutas privadas, tokens o payloads
  externos completos como evidencia durable.
- Los errores publicos deben ser compactos (`write_failed`, `sync_failed`,
  `lock_conflict`, `snapshot_corrupt`) y coordinarse con T95, T57, T98 y T101
  sin sustituir sus contratos de estado, outbox o ledgers concretos.
- Tests: `go test -count=1 ./modulos/orquesta-run-file ./modulos/orquesta-domain-work-file ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-server ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 trigesimoquinta pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`, `README.md`,
docs vigentes de estado/nucleo/principio/matriz, este backlog, rail errors,
duplicaciones de rails, `rg` sobre `ProcessRuntime`, `ExternalAgentLaunchSpec`,
`codex-wave`, stdout/stderr, prompts, summaries publicos y tests/smokes Go, y
lectura focal de `modulos/orquesta-runtime/process_runtime_connector_v0.go`,
`modulos/orquesta-runtime/process_runtime_connector_types_v0.go`,
`modulos/orquesta-runtime/external_agent_connector_v0.go`,
`cmd/orquesta-server/codex_wave_command_v0.go`,
`cmd/orquesta-server/codex_wave_control_v0.go`,
`cmd/orquesta-server/codex_director_wave_command_v0_test.go` y smokes Go de
`modulos/orquesta-app-codex-stack`. No se programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- `ProcessRuntimeConnectorV0` ya valida shell, env y rutas, pero el paso final
  de launch convierte refs opacas de `ExternalAgentCommandV0` en
  `CommandPath`, `Args`, `Env` y `WorkingDir` reales sin recibo redacted que
  demuestre que la resolucion respeta el perfil. Ademas descarta stdout/stderr
  con `io.Discard`; T66 cubre stop neutral, pero no el owner de launch/env/IO.
- `codex-wave` devuelve summaries con rutas reales de runtime, registry,
  prompts, ACK, last message, stdout/stderr, HOME/CODE_HOME de agente y PID.
  T58-T60 cubren tail/purge/stop, pero falta separar diagnostico local crudo de
  salida publica compacta del comando de launch/status.
- Los smokes Go y tests opt-in reales siguen imprimiendo cuerpos HTTP,
  stdout/stderr o prompts completos en `t.Fatalf` cuando fallan. T61 cubre
  artefactos de tests requeridos y T91 cubre scripts shell, pero no hay rail
  comun para diagnosticos de pruebas Go con proveedor/OPES/control plane real.

## T105 process-runtime-launch-env-io-receipt

Objetivo: cerrar la frontera de launch del runtime neutral para que la
resolucion de comando/env/working dir desde refs opacas tenga recibo compacto,
politica de IO y evidencia verificable.

Estado: pendiente.

Alcance:

- `modulos/orquesta-runtime`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-agent-process-registry`
- `modulos/orquesta-agent-process-registry-memory`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- El resolver de `ExternalAgentProcessCommandV0` debe emitir un recibo de
  resolucion con `command_ref`, `executable_ref`, `arg_refs`, `env_refs`,
  `working_dir_ref`, hash/policy y causa publica; no persistir command path,
  args, env ni working dir reales como evidencia durable.
- `ProcessRuntimeConnectorV0` debe aplicar politica explicita de env: allowlist
  por nombre, valores redactados o refs, PATH controlado y rechazo de HOME,
  tokens, secretos, remotos Git, prompt/transcript y rutas privadas fuera del
  perfil permitido.
- La decision de IO debe ser visible: stdout/stderr descartados con recibo
  `io_discarded`, capturados con limite/redaccion o enviados a sink local
  opt-in. No depender de logs crudos para readiness, ACK o cierre.
- Registry y stats deben conservar solo refs compactas de launch/session/process
  y recibos de politica, coordinados con T66/T96/T97 sin duplicar stop ni logs
  daemon.
- Tests: `go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-orchestration-core ./modulos/orquesta-agent-process-registry ./modulos/orquesta-agent-process-registry-memory ./cmd/orquesta-server`.

## T106 codex-wave-public-summary-redaction

Objetivo: separar la salida publica de `codex-wave`/`codex-director-wave` de los
artefactos locales crudos de runtime, sin romper diagnostico opt-in de operador.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-runtime-worktree`
- `modulos/orquesta-agent-process-registry`
- `docs/runbooks`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Los summaries por stdout deben exponer refs opacas, contadores, estados y
  codigos publicos; rutas absolutas de runtime, prompt, ACK, stdout/stderr,
  HOME/CODE_HOME, registry file y PID crudo quedan fuera salvo modo local
  diagnostico opt-in.
- `codex-wave-status` y `codex-wave-tail` deben compartir la misma politica de
  redaccion: summary por defecto, fragmento crudo limitado solo con flag/env
  explicito y causa publica.
- La registry de ola debe seguir disponible para operaciones locales, pero la
  confianza de stop/purge se gobierna por T59/T60; esta tarea no relaja
  operaciones destructivas ni copia credenciales.
- Los tests deben probar que una salida publica no contiene rutas privadas,
  HOME, prompts, transcripts, stdout/stderr crudo, tokens ni payload de
  proveedor, y que el modo diagnostico queda claramente opt-in.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-agent-process-registry`.

## T107 real-smoke-go-diagnostic-redaction

Objetivo: unificar redaccion y limites de diagnosticos en tests/smokes Go
opt-in que pueden tocar Codex real, OPES temporal, control plane o payloads de
dominio.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-runtime-codex-delivery`
- `modulos/orquesta-opes-bridge`
- `modulos/orquesta-opes-connector`
- `modulos/orquesta-server`
- `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`
- `docs/runbooks`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Smokes Go opt-in y harnesses que ejecutan procesos reales deben usar helper
  comun para diagnostico de fallo: limite de bytes, redaccion por campo y
  summary con refs/codigos, no `t.Fatalf` con prompt, stdout/stderr, body HTTP
  o payload completo.
- Los cuerpos HTTP, respuestas OPES, prompts, transcripts, completions,
  stdout/stderr de proveedor, rutas privadas, HOME, env, tokens, remotos Git y
  control files no deben aparecer completos en logs de test por defecto.
- Cuando un diagnostico crudo sea necesario para operador local, debe requerir
  opt-in explicito, directorio temporal aislado, limite fuerte y nota publica de
  retencion; no cuenta como evidencia terminal de cierre.
- Coordinar con T21/T58/T61/T91/T97/T103: esta tarea cubre Go tests/smokes y su
  salida de fallo, no scripts shell, logs daemon, required-test output ni
  respuesta HTTP saliente productiva.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector ./modulos/orquesta-server`.

## Escaneo backlog 2026-05-24 trigesimosexta pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`, `README.md`,
docs vigentes de estado/nucleo/principio/matriz, este backlog, rail errors,
duplicaciones de rails, `modulos/orquesta-decision-council`,
`modulos/orquesta-director-candidates`,
`modulos/orquesta-core-leases`, `modulos/orquesta-runtime`,
`modulos/orquesta-runtime-codex-delivery`,
`modulos/orquesta-director-scheduler`,
`modulos/orquesta-director-tick-input`,
`modulos/orquesta-director` y busquedas `rg` sobre `DecisionCouncil`,
`BuildSchedulableWorkCandidatesFromCouncilPlanV0`, `AgentLease`,
`AgentProgressHeartbeat` y `lease_action_candidates`. No se programa codigo
desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- `orquesta-decision-council` y `orquesta-director-candidates` ya construyen
  planes de propuesta/critica/voto y candidatos schedulables, pero no hay
  composicion que materialice rondas vivas del consejo dentro del ciclo del
  Director. Si una tarea de arquitectura compleja necesita deliberacion, hoy el
  planner puede saltar directo a un agente normal o duplicar la logica como
  prompt, sin gates por ronda, wait scope ni aceptacion durable por votos.
- `orquesta-core-leases` define politica pura de leases/timeouts, mientras
  `orquesta-runtime` y `orquesta-runtime-codex-delivery` tienen otra politica
  de heartbeat/progreso y el scheduler consume `lease_action_candidates`
  separados. Falta un bridge de composicion que convierta progreso estancado,
  loop o proceso parado en `AgentTimeoutAssessmentV0`/`AgentLeaseExpired`
  causal, sin duplicar umbrales ni usar relojes ocultos como verdad de core.

## T108 decision-council-operational-rounds

Objetivo: cablear el consejo de decision multiagente como rondas operativas del
Director, con gates y waits causales, sin convertirlo en prompt libre ni en
runtime concreto.

Estado: pendiente.

Alcance:

- `modulos/orquesta-decision-council`
- `modulos/orquesta-director-candidates`
- `modulos/orquesta-director-scheduler`
- `modulos/orquesta-director-tick-input`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-app-director-service`
- `modulos/orquesta-app-codex-stack`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Una decision que pida consejo debe producir plan `DecisionCouncilPlanV0` y
  materializar rondas de propuesta, critica y voto como `WorkflowTaskV0`/
  candidatos con refs opacas, write-set, evidencias y capacity policy.
- Cada ronda debe tener gate/wait scope propio: no avanzar a critica sin
  propuestas entregadas, no votar sin criticas esperadas y no aceptar decision
  sin quorum/evidencia segun `DecisionCouncilVoteV0`.
- La aceptacion durable final debe pasar por comandos/eventos del workflow
  (`AcceptDecision` o equivalente vivo), no por summary textual del director ni
  por presencia de entregas sueltas.
- Reentrada/replay no duplica asignaciones, gates, waits, outbox ni votos; una
  ronda incompleta queda como bloqueo causal reparable.
- El modulo puro no decide proveedor, modelo, HOME, cuota, runtime ni familias
  reales; esas refs se resuelven en composicion/capacity.
- Tests: `go test -count=1 ./modulos/orquesta-decision-council ./modulos/orquesta-director-candidates ./modulos/orquesta-director-scheduler ./modulos/orquesta-director-tick-input ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack`.

## T109 agent-lease-progress-policy-bridge

Objetivo: unir la politica pura de leases/timeouts con la observacion de
progreso de runtime para que stop/retry/replan no dependan de umbrales
duplicados ni de timers ocultos.

Estado: pendiente.

Alcance:

- `modulos/orquesta-core-leases`
- `modulos/orquesta-runtime`
- `modulos/orquesta-runtime-codex-delivery`
- `modulos/orquesta-director`
- `modulos/orquesta-director-scheduler`
- `modulos/orquesta-director-tick-input`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Definir un puente por puerto que tome `AgentProgressReportV0`/heartbeat de
  runtime y una `AgentLeasePolicyV0`, y emita `AgentTimeoutAssessmentV0` o
  candidato `AgentLeaseExpired` con `observed_at` inyectado por adaptador.
- Unificar umbrales de stalled/loop/stopped con leases de launch, heartbeat y
  response timeout; no mantener una politica en runtime y otra en
  `orquesta-core-leases` con significados divergentes.
- El scheduler debe consumir esas candidates igual que las lease actions
  actuales: registrar expiracion, parar agente, preguntar al director, reintentar
  o replanificar segun accion recomendada y estado vivo del run.
- La decision debe transportar solo refs compactas, reason code y evidencia;
  no PID, HOME, rutas, stdout/stderr, prompts, transcripts, proveedor/modelo ni
  payloads de runtime.
- Replay/idempotencia: la misma observacion no duplica `AgentLeaseExpired`,
  `StopAgent`, `AskDirector` ni replan, y una observacion posterior con
  progreso nuevo limpia o neutraliza la expiracion segun politica explicita.
- Tests: `go test -count=1 ./modulos/orquesta-core-leases ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-director ./modulos/orquesta-director-scheduler ./modulos/orquesta-director-tick-input ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 trigesimoseptima pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`,
`README.md`, docs vigentes de estado/nucleo/principio/matriz, este backlog,
rail errors, duplicaciones de rails, `modulos/orquesta-director-scheduler`,
`modulos/orquesta-director`, `modulos/orquesta-run-queue`,
`modulos/orquesta-run-supervisor`, `modulos/orquesta-run-memory`,
`modulos/orquesta-run-file`, `modulos/orquesta-core-concurrency` y busquedas
`rg` sobre `forbiddenSchedulerFragmentsV0`,
`forbiddenOutboxDispatchCycleTermsV0`, `reinicio_orquesta_v2`,
`EvaluateParallelGroupsV0`, `WorksetClaim`, `RunSchedulingCandidateV0` y
`RankRunCandidatesV0`. No se programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- `modulos/orquesta-director-scheduler/scheduler_tick_validation_v0.go` aun
  corta summaries/roles/lease candidates por una lista local que incluye
  `runtime`, `provider`, `model`, `db`, `sql`, `home`, `filesystem` y
  `docker`. `modulos/orquesta-director/outbox_dispatch_cycle_validation_v0.go`
  conserva otra lista local para error codes de dispatch. Ambas fronteras viven
  en el ciclo operativo y pueden reabrir falsos positivos ya cerrados por
  `orquesta-rails` si una entrega menciona vocabulario operativo opaco.
- Varios `AGENTS.md` locales obligan a clasificar bugs con
  `../../docs/reinicio_orquesta_v2/protocolo_anti_bucles.md`, pero el arbol
  `docs/reinicio_orquesta_v2` no existe en la foto actual. Un agente que siga
  literalmente esa instruccion queda sin contexto requerido o inventa un
  protocolo historico.
- `orquesta-core-concurrency` ya evalua claims/read-write sets y el scheduler
  usa gates dentro de un tick, mientras `orquesta-run-queue` declara que no
  cubre bloqueos, leases ni despacho. Falta un puente global que evite arrancar
  runs concurrentes con write-sets solapados antes de que entren al scheduler
  interno de cada run.

## T110 director-scheduler-cycle-rail-policy-sync

Objetivo: alinear los rails de texto del scheduler y del ciclo de dispatch del
Director con la politica viva de `orquesta-rails`, sin bloquear vocabulario
operativo opaco ni relajar secretos efectivos.

Estado: pendiente.

Alcance:

- `modulos/orquesta-director-scheduler`
- `modulos/orquesta-director`
- `modulos/orquesta-rails`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-app-codex-stack`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `DirectorSchedulerTickInputV0` debe permitir summaries y refs opacas con
  `runtime`, `provider`, `model`, `db`, `sql`, `filesystem`, `docker` o
  equivalentes cuando son contexto operativo normal.
- El ciclo de dispatch debe normalizar error codes publicos sin listas locales
  que conviertan `provider_timeout`, `db_adapter_unavailable` o
  `runtime_backpressure` en fallo generico si no hay secreto efectivo.
- Los cortes fuertes se mantienen por campo para valores reales de secretos,
  HOME/rutas privadas, prompts/transcripts crudos, payloads masivos y
  causalidad rota.
- Las pruebas deben cubrir scheduler tick con vocabulario operativo, rechazo de
  secretos efectivos y dispatch error-code redacted sin duplicar la politica
  comun en cada comando.
- Tests: `go test -count=1 ./modulos/orquesta-director-scheduler ./modulos/orquesta-director ./modulos/orquesta-rails ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-codex-stack`.

## T111 local-agents-required-doc-refs-sync

Objetivo: reparar o clasificar las referencias obligatorias a docs historicos
inexistentes en `AGENTS.md` locales para que los agentes no cierren con contexto
inventado ni se bloqueen por una ruta ausente.

Estado: pendiente.

Alcance:

- `modulos/orquesta-core-workflow/AGENTS.md`
- `modulos/orquesta-core-concurrency/AGENTS.md`
- `modulos/orquesta-core-leases/AGENTS.md`
- `modulos/orquesta-core-replanner/AGENTS.md`
- `modulos/orquesta-director-scheduler/AGENTS.md`
- `docs/estado_actual_2026-05-17.md`
- `docs/guia_nucleo_orquestacion_2026-05-17.md`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Cada referencia a `docs/reinicio_orquesta_v2/*` debe resolverse a un archivo
  existente, marcarse como historica con sustituto vigente o retirarse de las
  lecturas obligatorias locales.
- Si el protocolo anti-bucles sigue siendo necesario, debe vivir como doc
  vigente con owner, fecha, relacion con la foto actual y pruebas/criterios que
  no contradigan OrquestaV2.
- El materializer/packet de contexto debe poder distinguir doc requerido
  inexistente de `ref_only` permitido por diseno, coordinado con `T44`.
- No restaurar carpetas historicas completas ni docs stale solo para satisfacer
  rutas antiguas; corregir la fuente de autoridad o dejar bloqueo verificable.
- Tests: `go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-core-workflow ./modulos/orquesta-core-concurrency ./modulos/orquesta-core-leases ./modulos/orquesta-core-replanner ./modulos/orquesta-director-scheduler`.

## T112 run-queue-workset-concurrency-bridge

Objetivo: conectar la cola global de runs con los claims de
`orquesta-core-concurrency` para no arrancar runs concurrentes con write-sets
solapados antes de que el scheduler interno pueda gatearlos.

Estado: pendiente.

Alcance:

- `modulos/orquesta-run-queue`
- `modulos/orquesta-run-supervisor`
- `modulos/orquesta-run-memory`
- `modulos/orquesta-run-file`
- `modulos/orquesta-core-concurrency`
- `modulos/orquesta-director-scheduler`
- `modulos/orquesta-server`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `RunSchedulingCandidateV0` o una proyeccion asociada debe transportar claims
  compactos de read/write-set, fairness group y evidencia suficiente para
  evaluar solapes entre runs sin leer filesystem ni Git real.
- El supervisor residente debe filtrar o serializar candidatos solapados con
  runs vivos/reservados, usando `EvaluateConcurrencyGateV0` o contrato
  equivalente, y publicar reason code estable (`run_workset_conflict`,
  `run_workset_serialized`, `run_workset_claim_missing`).
- `T31` sigue gobernando reserva/lease de cola; esta tarea gobierna conflicto
  logico de work-set entre runs. `T43` sigue gobernando merge documental de
  scanners; esta tarea no resuelve conflictos de Markdown.
- Reentrada/replay no debe relanzar dos runs conflictivas si una quedo
  `wait_external`, `running`, `stop_requested` o con outbox pendiente.
- Las pruebas deben cubrir candidatos con write-set disjunto, solape exacto,
  carpeta padre/hijo, `write_set=.` opt-in y ausencia de claim como bloqueo
  recuperable.
- Tests: `go test -count=1 ./modulos/orquesta-run-queue ./modulos/orquesta-run-supervisor ./modulos/orquesta-run-memory ./modulos/orquesta-run-file ./modulos/orquesta-core-concurrency ./modulos/orquesta-director-scheduler ./modulos/orquesta-server ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 trigesimoctava pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`,
`README.md`, docs vigentes de estado/nucleo/principio/matriz, este backlog,
rail errors, duplicaciones de rails, `rg` sobre `reinicio_orquesta_v2`,
`RunSchedulingCandidateV0`, `fairness_group_ref`, `RankRunCandidatesV0`,
`WEB-013`, `AppPlanResolvers`, `LaunchRuntimeAgentRequestV0` y estados locales
de `modulos/*/docs/tareas.md`. No se programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- `T111` cubre `AGENTS.md` locales con referencias obligatorias a
  `docs/reinicio_orquesta_v2`, pero el mismo arbol inexistente sigue apareciendo
  en docs locales de `orquesta-core`, `orquesta-core-workflow`,
  `orquesta-capacity`, `orquesta-governance` y `orquesta-observability`. Esas
  fuentes alimentan contexto y backlog federado; si no se clasifican, pueden
  reabrir DBV1/control-plane historico o pedir snapshots inexistentes.
- `RunSchedulingCandidateV0` ya tiene `fairness_group_ref` y ranking con aging,
  pero la politica v0 declara que el aging solo desempata dentro de la misma
  prioridad. Despues de T31/T112, falta owner para fairness por grupo/app:
  automejora, smokes reales y trabajo humano no deben monopolizar ni quedar
  hambrientos por prioridad estatica.
- `modulos/orquesta-web/docs/tareas.md` conserva `WEB-013` como pendiente:
  sustituir el formulario largo de nueva app por una sesion conversacional de
  intake. T76 cubre routing AppSpec hacia Director V2, pero no el contrato web
  de sesion parcial, preguntas, i18n y puente fino al intake/director.

## T113 module-historical-doc-ref-sync

Objetivo: clasificar y reparar referencias historicas a
`docs/reinicio_orquesta_v2` en docs locales de modulos, sin restaurar carpetas
stale ni reintroducir DBV1/control-plane legacy.

Estado: pendiente.

Alcance:

- `modulos/orquesta-core`
- `modulos/orquesta-core-workflow`
- `modulos/orquesta-capacity`
- `modulos/orquesta-governance`
- `modulos/orquesta-observability`
- `modulos/orquesta-context`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Cada referencia local a `docs/reinicio_orquesta_v2/*` debe quedar marcada como
  historica con sustituto vigente, eliminada de requisitos vivos o convertida en
  bloqueo verificable con owner.
- `modulos/*/docs/tareas.md` no debe usar `DBV1-000` ni snapshots inexistentes
  como prerequisito ejecutable para automejora residente.
- El indice federado de T88 debe poder registrar estas entradas como historicas
  o stale con `source_path`, hash/freshness y Txx relacionado, sin programarlas
  como codigo.
- No borrar docs antiguos ni crear de nuevo `docs/reinicio_orquesta_v2`; la
  reparacion corrige la fuente que gobierna agentes actuales.
- Tests: `go test -count=1 ./modulos/orquesta-context ./cmd/orquesta-server`.

## T114 run-queue-fairness-group-policy

Objetivo: dar semantica ejecutable a `fairness_group_ref` en la cola global para
evitar hambre o monopolio entre apps, automejora, smokes reales y trabajo humano
sin romper prioridad explicita ni leases de T31.

Estado: pendiente.

Alcance:

- `modulos/orquesta-run-queue`
- `modulos/orquesta-run-supervisor`
- `modulos/orquesta-run-memory`
- `modulos/orquesta-run-file`
- `modulos/orquesta-server`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Definir politica de fairness por grupo/app con reloj inyectado, ventanas,
  limites y reason codes publicos (`fairness_group_paused`,
  `fairness_group_boosted`, `fairness_group_missing`).
- La prioridad manual sigue siendo primera senal; fairness no debe elevar smokes
  reales ni trabajos con efectos externos por encima de guardas opt-in o
  confirmaciones obligatorias.
- Coordinar con T31 y T112: primero claim/lease de candidato, despues evaluar
  conflicto de write-set y fairness; replay no duplica reservas ni cambia orden
  por reloj global oculto.
- Los candidatos sin `fairness_group_ref` deben recibir grupo derivado estable o
  bloqueo recuperable, no caer en ordenacion silenciosa que favorezca siempre al
  mismo productor.
- Tests: `go test -count=1 ./modulos/orquesta-run-queue ./modulos/orquesta-run-supervisor ./modulos/orquesta-run-memory ./modulos/orquesta-run-file ./modulos/orquesta-server ./cmd/orquesta-server`.

## T115 web-app-intake-session-contract

Objetivo: completar el contrato web de sesion conversacional para nueva app como
cliente fino de intake/director, sin convertir la web en planificador ni duplicar
`AppSpecV0`.

Estado: pendiente.

Alcance:

- `modulos/orquesta-web`
- `modulos/orquesta-app-director-intake`
- `modulos/orquesta-mcp`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-http-gateway`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Modelar `WebNuevaAppIntakeSessionV0` o contrato equivalente para estado
  parcial, pregunta pendiente, respuesta capturada, AppSpec parcial y
  `requiere_datos`, con textos i18n.
- La web solo adapta a puertos/API de intake/director; no decide arquitectura,
  agentes, modelo, proveedor, runtime, DB, deploy ni filesystem.
- Cuando el puerto de Director V2 este configurado, la sesion debe entregar refs
  opacas y contexto compacto; si falta, conservar fallback documentado a
  `SolicitarNuevaApp` sin aparentar cierre operativo.
- Coordinar con T76: routing publico AppSpec queda en Director V2; esta tarea
  solo cubre UX/contrato web de intake y tests de adaptador fino.
- Tests: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-app-director-intake ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 trigesimonovena pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`, `README.md`,
docs vigentes de estado/nucleo/principio/matriz, este backlog, rail errors,
duplicaciones de rails, `rg` sobre docs locales, IDs de `modulos/*/docs/tareas.md`,
`APP-CODEX-STACK-012`, `CLI-P001`, `CLI-P007`, `docs/plan_microtareas.md` y
referencias a manuales generados. No se programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- `modulos/orquesta-app-codex-stack/docs/tareas.md` contiene dos secciones con
  el mismo ID `APP-CODEX-STACK-012`, una para review gate y otra para cambios
  sobre app existente. Un indice federado o planner local puede colapsarlas,
  tomar la evidencia de una por la otra o reabrir/cerrar trabajo equivocado.
- `modulos/orquesta-cli/docs/pruebas.md` conserva `CLI-P001`, `CLI-P007` y
  `CLI-P008` como pendientes aunque el modulo ya tiene tests de ayuda sin red,
  server status sin fallback local e inventario V1 en cuarentena. La cola global
  declara T06 completada; falta sincronizacion verificable de docs locales para
  no relanzar trabajo cerrado ni ocultar huecos reales.
- `docs/plan_microtareas.md` y decisiones de `orquesta-app-director-intake`
  siguen nombrando artefactos generados (`docs/manual_desarrollador.md`,
  `docs/manual_sistemas_deploy.md`, `docs/pruebas_documentales.md`,
  `docs/pendientes.md`) que no existen en este repo vigente. Si esos documentos
  se leen como requisitos vivos de Orquesta, un agente puede fallar por contexto
  inexistente o crear docs historicos fuera de la arquitectura actual.

## T116 module-task-doc-integrity-linter

Objetivo: anadir una comprobacion automatica de integridad documental para
backlogs locales de modulos, centrada en IDs duplicados, estados stale y pruebas
locales desalineadas con la evidencia real.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-context`
- `modulos/orquesta-app-codex-stack/docs/tareas.md`
- `modulos/orquesta-cli/docs/pruebas.md`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Detectar IDs duplicados en `modulos/*/docs/tareas.md`, con caso inicial
  `APP-CODEX-STACK-012`, y reportarlos como bloqueo documental, no como cierre
  silencioso ni merge automatico.
- Detectar casos `Ultima ejecucion: pendiente` en docs locales cuando existe
  evidencia de tests Go vigentes o cuando el backlog global ya declara la tarea
  completada; pedir sincronizacion documental antes de relanzar trabajo.
- El planner residente y el futuro indice federado deben distinguir
  `pendiente real`, `historico`, `stale` y `completado con evidencia`, sin
  deducir terminalidad solo por heading o texto narrativo.
- No borrar ni renumerar tareas locales sin revisar referencias; si hay colision
  de ID, crear decision de renombrado o alias historico con refs fuente.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-context ./modulos/orquesta-cli ./modulos/orquesta-app-codex-stack`.

## T117 app-codex-review-gate-policy-owner

Objetivo: separar y hacer ejecutable la politica productiva del review gate de
autoprogramacion para entregas Codex, sin mezclarla con estadisticas, prompts ni
heuristicas de docs locales.

Estado: pendiente.

Alcance:

- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-runtime-worktree`
- `modulos/orquesta-app-director-service`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- La aceptacion/rechazo de entregas debe salir de una politica por puerto con
  issues estructurados: tests requeridos, write-set incompleto, fichero Go por
  encima del presupuesto, falta de progreso real, artefacto ausente o evidencia
  fuera de scope.
- La regla de 300 lineas debe tener un unico owner de politica y recibo de
  evidencia; no debe quedar duplicada entre prompt, ACK validator, worktree
  snapshot y review gate.
- El review gate debe tolerar alias/rutas hijas/globs seguros y pedir rework
  acotado cuando la intencion sea reparable; cortar fuerte solo por seguridad,
  causalidad, efectos externos, secreto efectivo o write-set imposible.
- La politica debe soportar perfiles no-Go o documentales sin aplicar
  automaticamente `go test` ni limites pensados para codigo Go.
- Tests: `go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-director-service`.

## T118 legacy-generated-doc-artifact-contract-sync

Objetivo: clasificar los planes documentales historicos de app generada y
sincronizarlos con la foto vigente para que no gobiernen agentes actuales como
requisitos vivos del repo Orquesta.

Estado: pendiente.

Alcance:

- `docs/plan_microtareas.md`
- `docs/arquitectura.md`
- `modulos/orquesta-app-director-intake/docs/decisiones.md`
- `docs/plantillas_documentacion`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Marcar `docs/plan_microtareas.md` como historico/debug o actualizarlo para
  que no exija artefactos inexistentes en el repo vigente.
- Si los nombres `manual_desarrollador`, `manual_sistemas_deploy`,
  `pruebas_documentales` y `pendientes` pertenecen a apps generadas, deben vivir
  como contrato/plantilla de proyecto generado, no como requisito de Orquesta.
- Los adaptadores de intake/director deben devolver refs de artefacto esperadas
  por tipo de app sin confundirlas con docs raiz del nucleo.
- No crear archivos historicos vacios solo para satisfacer `test -s`; si falta
  producto o contexto, el director debe pedir revision o marcar el plan stale.
- Tests: `go test -count=1 ./modulos/orquesta-app-director-intake ./modulos/orquesta-context ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 cuadragesima pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`, `README.md`,
docs vigentes de estado/nucleo/principio/matriz, este backlog, rail errors,
duplicaciones de rails, `docs/uso_actual_app_orquesta.md`,
`docs/operacion_agentes_manuales.md`, `modulos/orquesta-governance`,
`modulos/orquesta-core`, `modulos/orquesta-capacity`, `modulos/orquesta-web` y
`rg` sobre APIs legacy, snapshots SQLite forenses y wrappers manuales. No se
programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- `docs/uso_actual_app_orquesta.md` sigue describiendo una operativa V1 con
  `./orquesta serve`, rutas `/api/*` sin version, OpenClaw y politica AP-077.
  La foto vigente usa servidor/composicion actual y rutas `/api/v0/*`; si una
  IA usa ese manual como fuente viva, puede preparar pruebas o clientes contra
  endpoints legacy y reabrir control-plane antiguo.
- Docs locales de governance, core y capacity conservan snapshots forenses de
  SQLite/DBV1 como fuente documental, incluso con rutas absolutas historicas y
  comandos `sqlite3` de lectura. Aunque estan pensados como cuarentena, no hay
  linter que impida que el indice federado o un agente los trate como requisito
  vivo de persistencia.
- `docs/operacion_agentes_manuales.md` y el manual de uso mantienen wrappers
  `scripts/inicio_agente.sh`/Terminator como compatibilidad, pero no existe una
  prueba/catalogo que demuestre que esos scripts siguen subordinados al daemon,
  no mutan estado fuera del control plane y no se ofrecen como via principal a
  agentes gobernados por OrquestaV2.

## T119 server-first-usage-doc-route-sync

Objetivo: sincronizar la documentacion de uso operativo para que no anuncie
rutas, comandos ni control-plane V1 como camino vigente del servidor actual.

Estado: pendiente.

Alcance:

- `docs/uso_actual_app_orquesta.md`
- `docs/README.md`
- `README.md`
- `modulos/orquesta-cli`
- `modulos/orquesta-web`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Clasificar `docs/uso_actual_app_orquesta.md` como historico o actualizarlo
  con rutas versionadas `/api/v0/*`, comandos reales del servidor y frontera de
  composicion actual.
- No anunciar OpenClaw, API legacy sin version, `./orquesta serve` ni AP-077
  como requisito vigente salvo que queden marcados como compatibilidad historica
  con sustituto actual.
- CLI y web deben enlazar a clientes finos vigentes; si una ruta solo existe en
  docs antiguas, el planner debe verla como stale y no generar tareas de codigo
  contra ella.
- Mantener la regla server-first sin reintroducir DB local, control-plane V1,
  `cmd/db/internal` ni `ensureLocalDB`.
- Tests: `go test -count=1 ./modulos/orquesta-cli ./modulos/orquesta-web ./cmd/orquesta-server`.

## T120 legacy-sqlite-forensic-doc-quarantine

Objetivo: poner bajo cuarentena verificable las referencias documentales a
snapshots SQLite/DBV1 para que no gobiernen agentes ni tareas actuales.

Estado: pendiente.

Alcance:

- `modulos/orquesta-governance`
- `modulos/orquesta-core`
- `modulos/orquesta-capacity`
- `modulos/orquesta-context`
- `docs/estado_actual_2026-05-17.md`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Las refs a `backups/legacy-sqlite-20260422/orquesta.db`, DBV1 y comandos
  `sqlite3` deben quedar marcadas como forenses/historicas, con estado
  `quarantine` o equivalente, no como prerequisito vivo de modulo.
- Ningun doc local vigente debe contener rutas absolutas de desarrollador como
  fuente de verdad; usar refs relativas opacas o notas historicas sin convertir
  el snapshot en dependencia de ejecucion.
- El indice federado de T88/T116 debe poder distinguir `forense`, `historico`,
  `quarantine` y `effective`, y no debe programar lecturas DBV1 sin decision
  explicita del director.
- Mantener `orquesta-domain-work-sql` como adaptador externo de referencia, no
  como persistencia global ni sucesor implicito de DBV1.
- Tests: `go test -count=1 ./modulos/orquesta-governance ./modulos/orquesta-core ./modulos/orquesta-capacity ./modulos/orquesta-context`.

## T121 manual-agent-ops-compatibility-contract

Objetivo: cerrar el contrato de compatibilidad de wrappers manuales de agentes
para que no compitan con el daemon residente ni con el runtime gobernado.

Estado: pendiente.

Alcance:

- `docs/operacion_agentes_manuales.md`
- `docs/uso_actual_app_orquesta.md`
- `scripts/inicio_agente.sh`
- `scripts/cargar_agentes.sh`
- `scripts/terminator_agentes.sh`
- `scripts/agente_console.sh`
- `modulos/orquesta-cli`
- `modulos/orquesta-server`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Los wrappers manuales deben quedar catalogados como recuperacion/operacion
  asistida, nunca como camino principal para agentes gobernados por OrquestaV2.
- Cualquier script manual que inicie, pause, continue o cierre agentes debe
  pasar por CLI/API del servidor o declarar bloqueo verificable; no debe mutar
  DB, stores, worktrees ni runtime por fuera del control plane.
- La documentacion debe explicar como se correlacionan `external_session_id`,
  run/task/agent refs, worktree opaca y cierre de sesion sin exponer rutas
  privadas ni depender de Terminator/tmux como contrato de nucleo.
- Las pruebas deben cubrir sintaxis de scripts, modo sin servidor, error publico
  recuperable y ausencia de arranque automatico de flotas legacy por seed.
- Tests: `bash -n scripts/inicio_agente.sh scripts/cargar_agentes.sh scripts/terminator_agentes.sh scripts/agente_console.sh` y `go test -count=1 ./modulos/orquesta-cli ./modulos/orquesta-server ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 cuadragesima primera pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`, `README.md`,
docs vigentes de estado/nucleo/principio/matriz, este backlog, rail errors,
duplicaciones de rails, `docs/00_INDICE.md`, `docs/README.md`,
`docs/BIBLIA_APP_ORQUESTA.md`, `docs/op_088_orquesta_servidor_mcp.md`,
`docs/runtime_worker_contract.md`, `modulos/orquesta-factory`,
`modulos/orquesta-factory-http`, `modulos/orquesta-web` y busquedas `rg`
sobre `BIBLIA_APP_ORQUESTA`, `OpenClaw`, `tmux`, `BacklogInicialPropuestoV0`
y `/api/v0/apps/spec`. No se programa codigo desde este scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- `docs/BIBLIA_APP_ORQUESTA.md` aun se declara a si misma como doctrina
  canonica y `docs/00_INDICE.md` la coloca como `M00`, mientras
  `docs/estado_actual_2026-05-17.md` ya la clasifica como historica/stale y
  `docs/README.md` mantiene el titulo "Documentacion de Orquesta v1". Sin una
  marca interna de obsolescencia y una regla de precedencia verificable, un
  agente puede tomar OpenClaw, SQLite, rutas `/api/*` o control-plane V1 como
  camino vigente.
- `modulos/orquesta-factory` devuelve `BacklogInicialPropuestoV0` como
  `backlog` desde `/api/v0/apps/spec`; la web lo renderiza como preview, pero el
  contrato HTTP no transporta estado `preview/no_ejecutable`, freshness ni
  handoff al Director V2. Un planner externo puede tratar esas microtareas
  deterministas como backlog operativo y saltarse juicio del director.
- Docs historicos de orquestadores/transporte (`BIBLIA_APP_ORQUESTA.md`,
  `op_088_orquesta_servidor_mcp.md`, `runtime_worker_contract.md` y OPs
  cercanas) siguen nombrando OpenClaw, `tmux`, `/api/mcp` y servidor MCP como
  superficies canonicas. La foto vigente conserva MCP/transporte real como
  composicion opt-in; falta una cuarentena documental que impida que esas piezas
  se conviertan en requisitos vivos o rutas publicas inventadas.

## T122 canonical-doc-precedence-self-sync

Objetivo: hacer que los documentos historicos que antes se declaraban
canonicos se autoidentifiquen como historicos/stale y apunten a la foto vigente
sin depender de que el lector ya haya abierto `estado_actual`.

Estado: pendiente.

Alcance:

- `docs/BIBLIA_APP_ORQUESTA.md`
- `docs/00_INDICE.md`
- `docs/README.md`
- `docs/estado_actual_2026-05-17.md`
- `docs/guia_nucleo_orquestacion_2026-05-17.md`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `BIBLIA_APP_ORQUESTA.md` debe dejar de decir que manda sobre el proyecto
  vigente; debe marcarse como historico/vision y enlazar a `AGENTS.md`,
  `README.md`, `estado_actual`, `guia_nucleo` y backlog como fuentes vivas.
- `docs/00_INDICE.md` y `docs/README.md` deben distinguir documentos
  `vigentes`, `historicos`, `forenses` y `plantillas`, sin promover Orquesta V1
  ni OpenClaw como producto base del nucleo neutral.
- El indice/linter de T116/T88 debe poder detectar un doc que se declara
  canonico pero esta clasificado como stale por la foto vigente.
- No borrar documentos antiguos; conservarlos con fecha, estado y sustituto
  vigente.
- Tests: `go test -count=1 ./modulos/orquesta-context ./cmd/orquesta-server`.

## T123 factory-backlog-preview-director-handoff

Objetivo: separar el backlog determinista inicial de `orquesta-factory` de un
plan operativo ejecutable, dejando claro que es preview/insumo y que el cierre
real de apps nuevas pasa por Director V2 cuando se necesita juicio.

Estado: pendiente.

Alcance:

- `modulos/orquesta-factory`
- `modulos/orquesta-factory-http`
- `modulos/orquesta-web`
- `modulos/orquesta-app-director-intake`
- `modulos/orquesta-mcp`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `BacklogInicialPropuestoV0` debe transportar estado/fuente suficiente para
  distinguir `preview`, `needs_director` y `operational_plan_ref`; no debe
  aparentar plan-state, wait, review, tests ni cierre.
- `/api/v0/apps/spec`, web y MCP deben exponer ese backlog como vista previa o
  insumo de intake, no como cola ejecutable; si se arranca trabajo real, debe
  enlazar con Director V2/T76.
- Los artefactos sugeridos de app generada (`docs/manual_*`, `i18n/`,
  `adapters/`) deben quedar como paths del proyecto generado, no requisitos del
  repo Orquesta.
- Mantener i18n y cliente fino: la web no decide arquitectura, runtime,
  proveedor, DB, deploy ni agentes.
- Tests: `go test -count=1 ./modulos/orquesta-factory ./modulos/orquesta-factory-http ./modulos/orquesta-web ./modulos/orquesta-app-director-intake ./modulos/orquesta-mcp ./cmd/orquesta-server`.

## T124 legacy-external-orchestrator-doc-quarantine

Objetivo: cuarentenar la documentacion historica que presenta OpenClaw, `tmux`
o MCP real como superficies canonicas vivas, para que no gobierne tareas ni
conectores actuales sin decision explicita de composicion.

Estado: pendiente.

Alcance:

- `docs/BIBLIA_APP_ORQUESTA.md`
- `docs/op_088_orquesta_servidor_mcp.md`
- `docs/runtime_worker_contract.md`
- `docs/op_087_autogestion_supervisada_agentes.md`
- `docs/benchmark_orquestadores_externos_2026-04-12.md`
- `modulos/orquesta-mcp`
- `modulos/orquesta-runtime`
- `modulos/orquesta-runtime-codex`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Docs de OpenClaw, MCP real, `tmux` y transportes externos deben indicar si son
  inspiracion historica, compatibilidad de rescate o composicion opt-in; no
  deben declarar endpoints, procesos ni gestores externos como requisito vivo
  del nucleo.
- La superficie MCP vigente debe quedar enlazada a `orquesta-mcp` y al wiring
  real opt-in de `cmd/orquesta-server`, sin inventar `/api/mcp` o rutas legacy
  si no estan cableadas.
- `tmux`/Terminator no deben aparecer como runtime canonico del nucleo; si se
  mantienen, deben ser adaptador o recuperacion con refs opacas y tests propios.
- El indice federado debe poder excluir estos docs de planificacion automatica
  salvo que una tarea declare explicitamente composicion externa/legacy.
- Tests: `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 cuadragesima segunda pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`, `README.md`,
docs vigentes de estado/nucleo/principio/matriz, este backlog, rail errors,
duplicaciones de rails, `modulos/orquesta-work-profiles`,
`modulos/*/arrancar_codex.sh`, `.gitignore`, raiz del proyecto y busquedas `rg`
sobre `work_profile`, `arrancar_codex`, `_comun`, `.ssl-key.log`,
`.orquesta-server` y `.orquesta-smoke-work`. No se programa codigo desde este
scanner.
`worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan solo
como refs opacas.

Huecos concretos nuevos:

- `modulos/orquesta-work-profiles` existe como directorio vacio salvo `docs/`,
  no aparece en las fuentes vigentes ni en el backlog, y
  `orquesta-core-workflow/docs/decisiones.md` ya descarta crear ese modulo como
  owner de perfiles. Un indice federado podria tratarlo como modulo pendiente o
  asignar alli cambios que deben vivir en el contrato neutral actual.
- Hay decenas de `modulos/*/arrancar_codex.sh`; muchos delegan en
  `../_comun/arrancar_codex_modulo.sh`, pero no existe `modulos/_comun` en la
  foto revisada. Varias READMEs aun anuncian esos wrappers como arranque manual.
  Sin catalogo de compatibilidad, agentes OrquestaV2 pueden recibir una via rota
  o paralela al daemon residente.
- `.gitignore` excluye artefactos locales sensibles o de control como
  `.ssl-key.log`, `.orquesta-runtime/`, `.orquesta-server` y
  `.orquesta-smoke-work`, y la raiz contiene algunos de ellos. T50 cubre control
  files, pero falta una politica verificable para que snapshots, contexto,
  AppVCS, promocion y ACK terminal no dependan solo de `.gitignore` ni incluyan
  artefactos locales ignorados como producto.

## T125 work-profiles-empty-module-quarantine

Objetivo: clasificar `modulos/orquesta-work-profiles` para que no compita con
`WorkProfileV0` en `orquesta-core-workflow` ni genere trabajo falso desde un
directorio vacio.

Estado: pendiente.

Alcance:

- `modulos/orquesta-work-profiles`
- `modulos/orquesta-core-workflow`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-app-planner`
- `modulos/orquesta-context`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Declarar si `modulos/orquesta-work-profiles` queda historico, placeholder
  prohibido para planificacion o futuro modulo con owner explicito.
- El indice federado y el planner de automejora no deben abrir tareas por un
  directorio vacio sin README/AGENTS/contrato efectivo.
- `WorkProfileV0`, `WorkflowTaskV0.work_profile_kind` y el resolver de
  perfiles siguen gobernados por `orquesta-core-workflow` y
  `orquesta-orchestration-core` hasta decision nueva.
- No mover ni borrar el directorio sin revisar referencias; si se mantiene,
  debe tener estado/freshness y sustituto vigente.
- Tests: `go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-planner ./modulos/orquesta-context ./cmd/orquesta-server`.

## T126 module-local-codex-launcher-compatibility-contract

Objetivo: inventariar y gobernar los wrappers `modulos/*/arrancar_codex.sh`
para que no sean una via rota o paralela al runtime gobernado.

Estado: pendiente.

Alcance:

- `modulos/*/arrancar_codex.sh`
- `modulos/*/README.md`
- `modulos/*/AGENTS.md`
- `scripts`
- `modulos/orquesta-cli`
- `modulos/orquesta-server`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Detectar wrappers que referencian `../_comun/arrancar_codex_modulo.sh` cuando
  el helper comun no existe o no esta versionado como contrato.
- Clasificar cada wrapper como recuperacion manual, compatibilidad historica,
  smoke opt-in o ruta vigente por CLI/API; no dejarlo como arranque canonico de
  agentes OrquestaV2.
- Las READMEs locales no deben recomendar wrappers rotos ni saltarse daemon,
  cola, write-set, refs de runtime, ACK, checkpoint o shutdown.
- Si se conserva un helper comun, debe tener prueba `bash -n`, modo sin
  servidor con error publico y no mutar stores/worktrees fuera del control
  plane.
- Tests: `bash -n $(find modulos -name arrancar_codex.sh | sort)` y `go test -count=1 ./modulos/orquesta-cli ./modulos/orquesta-server ./cmd/orquesta-server`.

## T127 ignored-local-artifact-exclusion-policy

Objetivo: convertir la lista de artefactos locales ignorados en una politica
ejecutable para snapshots, contexto, AppVCS, promocion y ACK terminal.

Estado: pendiente.

Alcance:

- `.gitignore`
- `modulos/orquesta-runtime-worktree`
- `modulos/orquesta-context`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-runtime-codex`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Snapshots, contexto, AppVCS, promocion y validacion terminal deben excluir
  artefactos locales ignorados de control/diagnostico, incluso si el write-set
  es `.`.
- La politica no debe depender solo de Git ni de `.gitignore`: debe producir
  recibo compacto de categorias excluidas y reason codes publicos.
- `.orquesta-runtime`, `.orquesta-server`, `.orquesta-smoke-work`, logs locales
  sensibles y runtime dirs no pueden entrar en `ACK.files`, contexto, commits ni
  evidencias de cierre como producto.
- Si un artifact local ignorado es necesario para diagnostico, debe quedar como
  ref de control opt-in, redactada y fuera del artefacto de producto.
- Tests: `go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-context ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-codex ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 cuadragesima tercera pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md` y documentos vigentes de
  direccion, nucleo y pruebas reales.
- Backlog, railes observados y duplicaciones vigentes hasta `T127`.
- `modulos/orquesta-director/agent_progress_supervisor_helpers_v0.go` y
  `modulos/orquesta-director/agent_progress_supervisor_v0_test.go`.
- `cmd/orquesta-server/daemon.go` y `cmd/orquesta-server/codex_env_v0.go`.
- `modulos/orquesta-run-control/architecture_v0_test.go` y busqueda de guards
  `forbidden*` en tests de arquitectura.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
  como refs opacas, no como rutas ni nombres Git.

Huecos detectados:

- El supervisor de progreso del Director filtra evidencias por fragmentos
  locales amplios (`provider`, `model`, `db`, `sql`, `home`, `runtime`) antes
  de exponer refs al Director. Eso puede borrar evidencia operacional opaca y
  duplica railes de redaccion sin owner comun.
- El arranque daemon hereda `os.Environ()` casi completo al proceso residente.
  Falta contrato de proyeccion/redaccion de entorno efectivo para start,
  diagnostico, estado y logs.
- Varias pruebas de arquitectura mezclan guards de imports/effects con listas
  de terminos por substring (`runtime`, `http`, `mcp`, `sql`). Sin politica
  comun pueden bloquear refs/comentarios validos o crear railes divergentes.

## T128 agent-progress-supervisor-rail-policy-sync

Objetivo: alinear la sanitizacion de evidencias del supervisor de progreso de
agentes con una politica comun de railes por campo, tolerante a refs opacas.

Estado: pendiente.

Alcance:

- `modulos/orquesta-director`
- `modulos/orquesta-runtime`
- `modulos/orquesta-core-leases`
- `modulos/orquesta-rails`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-app-codex-stack`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Sustituir el filtro local de fragmentos genericos en refs/evidencias o
  delegarlo en helper comun por campo y severidad.
- Permitir vocabulario operacional en refs opacas (`runtime`, `provider`,
  `model`, `db`, `sql`, `home`) cuando no expone valor local, secreto ni
  contenido crudo.
- Bloquear patrones efectivos de secreto o dato sensible: `api_key=`,
  `client_secret=`, `Authorization`, prompts/transcripts crudos, rutas HOME
  reales y payloads masivos.
- Conservar decisiones de progreso: detenido, bucle, sin ACK, sin artefacto y
  sin avance deben seguir detectandose con evidencia durable.
- Incluir matriz de falsos positivos y coordinar con T109/T110 para no crear
  otra lista paralela de railes.
- Tests: `go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-core-leases ./modulos/orquesta-rails ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-codex-stack`.

## T129 server-daemon-start-env-policy

Objetivo: hacer explicita, acotada y redactada la proyeccion de entorno usada
al arrancar el daemon residente.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-server`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-observability`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `start` no debe transferir silenciosamente todo `os.Environ()` como contrato
  implicito; debe existir perfil/allowlist o snapshot de categorias efectivas.
- Estado, diagnostico, auditoria y logs solo exponen categorias, refs o valores
  redactados; nunca HOME real, tokens, prompts, transcripts, remotes privados
  ni detalles crudos de proveedor.
- Variables requeridas ausentes deben producir issue publico accionable sin
  volcar entorno completo.
- La politica debe coordinar T56/T85/T97/T105: proyeccion Codex, bootstrap,
  stdout/stderr y entorno de runtime siguen owners separados.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./modulos/orquesta-observability`.

## T130 architecture-guard-test-rail-policy-owner

Objetivo: separar los guards de arquitectura reales de las listas locales de
terminos prohibidos para evitar railes por substring divergentes.

Estado: pendiente.

Alcance:

- `modulos/orquesta-run-control`
- `modulos/orquesta-run-queue`
- `modulos/orquesta-run-memory`
- `modulos/orquesta-director-candidates`
- `modulos/orquesta-app-director-intake`
- `modulos/orquesta-app-runner`
- `modulos/orquesta-director-agent-workflow`
- `modulos/orquesta-app-director-service`
- `modulos/orquesta-rails`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Los guards de imports/paquetes/efectos concretos siguen siendo estrictos para
  modulos neutrales.
- Las comprobaciones por substring deben quedar acotadas por campo, helper
  comun o matriz de falsos positivos, no como vocabulario global prohibido.
- Deben permitirse comentarios, refs opacas y nombres de politica como
  `runtime_ref`, `sql_policy_ref`, `http_transport_ref` o `mcp_surface_ref`
  cuando no importan ni ejecutan adaptadores concretos.
- Deben seguir fallando imports o efectos reales no permitidos en modulos
  neutrales, incluso si el texto no contiene una palabra prohibida.
- Coordinar con T70/T77/T110/T116 para no duplicar el rail de docs, app-change,
  scheduler/outbox ni freshness documental.
- Tests: `go test -count=1 ./modulos/orquesta-run-control ./modulos/orquesta-run-queue ./modulos/orquesta-run-memory ./modulos/orquesta-director-candidates ./modulos/orquesta-app-director-intake ./modulos/orquesta-app-runner ./modulos/orquesta-director-agent-workflow ./modulos/orquesta-app-director-service ./modulos/orquesta-rails`.

## Escaneo backlog 2026-05-24 cuadragesima cuarta pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, docs vigentes de estado,
  nucleo, principio, matriz y corte de autoprogramacion residente.
- Backlog, railes observados y duplicaciones vigentes hasta `T130`.
- `docs/auditoria_runtime_orquesta_2026-05-24.md`.
- `modulos/orquesta-observability/orquesta_event_types_v0.go`,
  `modulos/orquesta-observability/orquesta_event_payload_validation_v0.go` y
  `modulos/orquesta-observability/operational_status_types_v0.go`.
- Busquedas de cobertura local `AGENTS.md`/`README.md` en `modulos`, con foco
  en `orquesta-rails`, `orquesta-domain-work-http` y `orquesta-factory-http`.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
  como refs opacas, no como rutas ni nombres Git.

Huecos detectados:

- La documentacion de auditoria runtime incluye una ruta absoluta local de la
  composicion actual como ejemplo operativo. Aunque sea util para operador
  local, el backlog, el contexto materializado y los ACKs no tienen una prueba
  que impida propagar HOME, state dirs o rutas de control como artefacto de
  producto.
- `orquesta-observability` mantiene taxonomia propia de privacidad, claves
  prohibidas y fragmentos sensibles, separada de `orquesta-rails` y de la
  auditoria del servidor. Puede ser frontera legitima, pero necesita owner para
  no convertirse en otro rail por substring que bloquee refs opacas o deje pasar
  payloads crudos por otro canal.
- Algunos modulos de frontera nueva o sensible no tienen `AGENTS.md` local o
  README de modulo: `orquesta-rails` carece de ambos, y
  `orquesta-domain-work-http`/`orquesta-factory-http` carecen de `AGENTS.md`.
  Un agente puede operar solo con reglas raiz y omitir restricciones locales de
  HTTP/rails/factory, o el indice federado puede tratarlos como modulos sin
  owner documental.

## T131 documentation-local-path-redaction-linter

Objetivo: evitar que docs, contexto materializado, backlog y ACKs propaguen
rutas locales absolutas, state dirs, HOME o ficheros de control como evidencia
de producto.

Estado: pendiente.

Alcance:

- `docs`
- `modulos/orquesta-context`
- `modulos/orquesta-runtime-worktree`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Detectar en docs y contexto rutas absolutas locales, HOME, state dirs,
  `.orquesta-runtime`, `.orquesta-server`, prompts, transcripts y logs locales
  cuando se presenten como evidencia publica o artefacto de producto.
- Permitir ejemplos con variables o refs opacas, como
  `${ORQUESTA_SERVER_STATE_DIR}` o `state-dir-ref-*`, sin exponer valores
  concretos.
- `ACK.files`, snapshots, contexto, promocion y backlog no deben incluir
  ficheros de control ni rutas locales ignoradas como producto.
- Si una ruta local es necesaria para diagnostico operador, debe quedar en doc
  local marcada como ejemplo no exportable y con sustituto opaco para agentes.
- Tests: `go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-runtime-worktree ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

## T132 observability-privacy-taxonomy-rail-sync

Objetivo: alinear la taxonomia de privacidad de `orquesta-observability` con la
politica viva de railes y auditoria, sin duplicar listas de substrings ni
relajar secretos efectivos.

Estado: pendiente.

Alcance:

- `modulos/orquesta-observability`
- `modulos/orquesta-rails`
- `modulos/orquesta-server`
- `cmd/orquesta-server`
- `modulos/orquesta-mcp`
- `modulos/orquesta-web`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Mantener estrictas las reglas de payload compacto, profundidad, tamano y
  claves realmente sensibles.
- Distinguir `*_policy_ref`, `*_redaction_ref` o refs opacas validas de valores
  crudos como credenciales, prompts, transcripts, HOME, DSN o payloads masivos.
- Definir owner unico entre observability, auditoria JSONL y `orquesta-rails`
  para privacidad/redaccion visible; evitar tres listas divergentes con nombres
  parecidos.
- API/MCP/web deben consumir proyecciones con `redaction_level` verificable y
  no inferir privacidad desde texto libre.
- Tests: `go test -count=1 ./modulos/orquesta-observability ./modulos/orquesta-rails ./modulos/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-web ./cmd/orquesta-server`.

## T133 module-boundary-local-agent-doc-coverage

Objetivo: asegurar que modulos de frontera sensible tienen `AGENTS.md` local o
README suficiente para agentes OrquestaV2, sin crear boilerplate stale.

Estado: pendiente.

Alcance:

- `modulos/orquesta-rails`
- `modulos/orquesta-domain-work-http`
- `modulos/orquesta-factory-http`
- `modulos/orquesta-context`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Cada modulo con railes, HTTP o frontera factory debe declarar responsabilidad,
  capa, imports prohibidos, pruebas focales y docs vigentes antes de recibir
  tareas automaticas.
- Si un modulo no necesita `AGENTS.md`, el indice federado debe tener una razon
  explicita y una fuente sustituta; ausencia silenciosa no cuenta como contrato.
- Las guias locales no deben contradecir la foto vigente: core neutral,
  adaptadores opt-in, sin HOME/tokens/proveedor/DB concreta dentro del nucleo.
- Anadir linter o test documental que liste modulos sensibles sin guia local y
  falle con error publico antes de preparar una run automatica.
- Tests: `go test -count=1 ./modulos/orquesta-rails ./modulos/orquesta-domain-work-http ./modulos/orquesta-factory-http ./modulos/orquesta-context ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 cuadragesima quinta pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md` y documentos vigentes de estado,
  nucleo, principio, cierre y matriz.
- Backlog, railes observados y duplicaciones vigentes hasta `T133`.
- `cmd/orquesta-server/commands.go`,
  `modulos/orquesta-server/handler_v0.go`,
  `cmd/orquesta-server/config.go`,
  `modulos/orquesta-server/supervisor_loop_v0.go`,
  `cmd/orquesta-server/idle_self_improvement_backlog_planner_v0.go` y
  busqueda `rg` sobre `MaxExternalWaits`, `/api/status`,
  `/api/v0/server/status` y fallback de automejora.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
  como refs opacas, no como rutas ni nombres Git.

Huecos detectados:

- `orquesta-server status` consulta `/api/status` y el handler sigue aceptando
  esa ruta legacy junto a `/api/v0/server/status`. T119/T85/T87 cubren docs,
  estado publico y readiness, pero falta un owner para deprecacion o
  compatibilidad de alias legacy dentro del servidor/comando real.
- Si el planner de backlog no puede leer el documento, o no detecta pendientes
  claros, devuelve una request fallback basada en la configuracion generica de
  automejora. Hoy esa base conserva write-set de `orquesta-server` y puede
  convertir un fallo de scanner documental en una tarea de codigo demasiado
  amplia.
- El supervisor residente fija `MaxExternalWaits=1` y recorta cualquier valor
  mayor, mientras el director directo usa 120 y otros flujos reales calculan
  presupuestos mas amplios. Ese rail estrecho puede confundir espera externa
  viva con idle/capacidad libre antes de que ACK/delivery lleguen.

## T134 server-status-legacy-alias-sunset

Objetivo: hacer explicita la compatibilidad o retirada de `/api/status` para que
la ruta vigente `/api/v0/server/status` sea la unica fuente publica normal de
estado del servidor.

Estado: pendiente.

Alcance:

- `modulos/orquesta-server`
- `cmd/orquesta-server`
- `modulos/orquesta-cli`
- `modulos/orquesta-web`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `orquesta-server status` y cualquier cliente fino deben usar
  `/api/v0/server/status` como ruta primaria; `/api/status` solo queda como
  alias legacy auditado o se bloquea con error publico de migracion.
- Si se conserva el alias, debe documentar freshness, version, headers o warning
  suficiente para que docs/toolbelt/planner no lo promuevan como ruta vigente.
- La compatibilidad no debe exponer mas campos que la ruta versionada ni
  saltarse la redaccion de T85 o la readiness de T87.
- Tests de handler/comando deben cubrir ruta versionada, alias legacy y ausencia
  de referencias nuevas a `/api/status` en prompts/toolbelt/docs vigentes.
- Coordinar con T119, T85 y T87: esta tarea gobierna el alias runtime, no la
  sincronizacion completa de manuales ni el snapshot de configuracion.
- Tests: `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-cli ./modulos/orquesta-web ./cmd/orquesta-server`.

## T135 idle-self-improvement-planner-fallback-safety

Objetivo: impedir que un fallo o vacio del planner de backlog se convierta en
una automejora generica de codigo con write-set historico de servidor.

Estado: pendiente.

Alcance:

- `modulos/orquesta-server`
- `modulos/orquesta-autoprogramming`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Si el backlog no se puede leer, parsear o correlacionar, el residente debe
  dejar bloqueo/revision documental publica o request scanner acotada a docs,
  no reutilizar automaticamente el write-set generico de `orquesta-server`.
- El fallback de codigo solo puede activarse por decision/opt-in explicito con
  write-set, criterios, tests y causa; nunca por error silencioso del scanner.
- `planner_empty`, `backlog_sin_tareas_pendientes_detectadas` y ACKs ambiguos
  deben quedar visibles en status/auditoria sin marcar automejora como preparada
  ni ocultar huecos pendientes.
- La capacidad libre (`capacity_free`) no debe crear trabajo nuevo si el planner
  esta degradado y ya hay tareas conocidas, reservadas o ambiguas.
- Tests: `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-autoprogramming ./cmd/orquesta-server`.

## T136 resident-drain-external-wait-budget-policy

Objetivo: definir una politica observable para `MaxExternalWaits` del supervisor
residente, separando smokes rapidos, automejora con agentes vivos y flujos reales
largos.

Estado: pendiente.

Alcance:

- `modulos/orquesta-server`
- `modulos/orquesta-run-supervisor`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- El valor efectivo de `MaxExternalWaits` debe salir de politica por modo
  (`fake/offline`, automejora residente, Codex real, OPES temporal, operador)
  y quedar visible en auditoria/status; no recortar env mayor que 1 sin causa.
- El residente no debe disparar idle automejora ni declarar capacidad libre si
  una run sigue en `wait_external` con `WaitAgentRefs`/lease/ACK pendiente y
  presupuesto de espera razonable.
- Un override demasiado alto o incompatible debe bloquear con error publico
  recuperable, no abrir esperas infinitas ni consumir ticks sin progreso.
- Mantener tests rapidos con presupuesto bajo, pero separar ese modo de
  ejecuciones reales o temporales que necesitan varios polls de ACK/delivery.
- Coordinar con T32, T68 y T69: esta tarea gobierna presupuesto de espera
  externa; wakeups event-driven y taxonomia de parada siguen siendo owners
  separados.
- Tests: `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-run-supervisor ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 cuadragesima sexta pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md` y documentos vigentes de estado,
  nucleo y principio.
- Backlog, railes observados y duplicaciones vigentes hasta `T136`.
- `modulos/orquesta-mcp/*_http_v0.go`,
  `modulos/orquesta-web/*endpoint*_v0.go`,
  `modulos/orquesta-http-gateway/gateway_v0.go`,
  `modulos/orquesta-app-gateway/handler_v0.go`,
  `cmd/orquesta-server/commands.go`,
  `cmd/orquesta-server/run_status_command_v0.go`,
  `cmd/orquesta-server/opes_bridge.go` y
  `cmd/orquesta-server/mcp_real_transport_v0.go`.
- Busquedas `rg` sobre `json.NewDecoder(r.Body)`, `io.ReadAll`,
  `json.NewEncoder(stdout)`, `stdout.Write(body)`, `MaxBytesReader` y
  `LimitReader`.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
  como refs opacas, no como rutas ni nombres Git.

Huecos detectados:

- El transporte MCP real ya limita cuerpos JSON-RPC a 1 MiB con `LimitReader`,
  pero muchas rutas HTTP publicas MCP/web decodifican `r.Body` directamente y
  no comparten limite, `Content-Type`, trailing data ni politica de campos
  desconocidos.
- Varios clientes/comandos leen respuestas completas sin limite antes de
  devolverlas o incluirlas en error publico (`run-status`, `status` y clientes
  web/API). Un servidor o conector defectuoso podria devolver payload grande,
  transcript, rutas locales o cuerpo de dominio y hacerlo visible en stdout,
  error o auditoria.
- Las salidas de comandos de composicion mezclan diagnostico local y contrato
  publico: `status` escribe el body crudo o el state fallback, `run-status`
  escribe el body completo, `opes-drain-once` emite URLs base y los comandos
  Codex/MCP emiten summaries propios. Falta un owner de shape/redaccion para
  decidir que puede consumir un agente, una auditoria o un operador.

## T137 public-http-request-body-bounds

Objetivo: unificar limites y decodificacion estricta para cuerpos de entrada en
HTTP publico/MCP/web sin duplicar reglas por handler.

Estado: pendiente.

Alcance:

- `modulos/orquesta-mcp`
- `modulos/orquesta-web`
- `modulos/orquesta-http-gateway`
- `modulos/orquesta-app-gateway`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Crear helper/puerto local para leer JSON HTTP con limite configurable,
  `Content-Type` admitido, trailing data bloqueado y decision explicita sobre
  campos desconocidos por endpoint.
- Aplicar la politica a rutas mutables de autoprogramacion, run control, cola,
  shutdown, domain_work, app_change y formularios web que aceptan JSON.
- Mantener el limite de `/mcp` como caso compatible o moverlo al mismo owner,
  sin rebajar protecciones del transporte JSON-RPC real.
- Los errores deben ser publicos, compactos y localizables; no deben devolver el
  body crudo ni campos sensibles.
- Tests: `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-web ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.

## T138 outbound-http-response-bounds-redaction

Objetivo: limitar y redactar respuestas HTTP leidas por clientes/comandos antes
de escribirlas en stdout, errores publicos o auditoria.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-web`
- `modulos/orquesta-mcp`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-domain-work-http`
- `modulos/orquesta-opes-connector`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Sustituir lecturas sin limite de respuestas por helper con limite, descarte
  seguro del resto del body y error publico estable.
- En errores HTTP, devolver codigo, ref/correlation y resumen redactado; no
  propagar cuerpos completos, HTML, transcripts, rutas locales, tokens ni
  payloads de dominio.
- `run-status`, `status`, clientes web, clientes MCP y conectores HTTP deben
  compartir politica o declarar frontera local equivalente con pruebas.
- El happy path puede seguir devolviendo JSON completo cuando el endpoint es
  publico y el body queda dentro del limite; el exceso debe bloquear o truncar
  con marcador verificable.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-domain-work-http ./modulos/orquesta-opes-connector`.

## T139 command-output-public-shape-contract

Objetivo: separar salida publica de comandos, diagnostico local y evidencia
durable para que agentes y operadores no consuman payloads crudos por stdout.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-server`
- `modulos/orquesta-observability`
- `modulos/orquesta-rails`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-app-codex-stack`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Definir DTO publico por comando (`status`, `run-status`, `opes-drain-once`,
  `mcp-real-smoke`, `codex-wave-*`) con campos permitidos, version, freshness y
  `redaction_level`.
- URLs, PID, dirs runtime, state dirs, base URLs de dominio, errores Git/Codex,
  stdout/stderr y bodies HTTP deben salir como refs/categorias o quedar
  marcados como diagnostico local opt-in.
- La salida publica no debe ser fuente terminal de cierre; ACK estructurado,
  eventos, evidencias de tests y stores causales siguen siendo canon.
- Coordinar con T58, T85, T131 y T138: tail/logs, status config, rutas locales y
  respuesta HTTP son vecinos, pero esta tarea gobierna el shape de comandos.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-observability ./modulos/orquesta-rails ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack`.

## Escaneo backlog 2026-05-24 cuadragesima septima pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, documentos vigentes de estado,
  nucleo, principio y matriz.
- Backlog, railes observados y duplicaciones vigentes hasta `T139`.
- Busquedas `rg` sobre `T[0-9]+`, `panic(`, `os.Exit`, `log.Fatal`,
  `json.NewDecoder`, `io.ReadAll`, `agent_ack`, `write_set` y control files.
- Lectura focal de `cmd/orquesta-server/mcp_real_smoke_v0.go`,
  `modulos/orquesta-core-leases/lease_evaluator_validation_v0.go` y
  `modulos/orquesta-runtime-codex/codex_ack_validation_v0.go`.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
  como refs opacas, no como rutas ni nombres Git.

Huecos detectados:

- El backlog ya contiene tareas con solape material: `T102` y `T137` gobiernan
  lectura/validacion de cuerpos HTTP publicos; `T103` y `T138` gobiernan
  respuestas HTTP salientes con limite/redaccion. T43 evita colisiones futuras
  de scanners y T79 parte el backlog, pero falta una politica para consolidar
  duplicados ya escritos sin borrar historia ni lanzar dos agentes sobre el
  mismo owner.
- Quedan `panic(` en codigo no-test: `mustMarshalMCPRealSmokeV0` en el comando
  temporal MCP real y `mustParseAgentLeaseInstantV0` en leases. Aunque hoy se
  usan con invariantes previas, las fronteras publicas/control-plane no tienen
  contrato comun de "no panic": un fallo de marshal, parse o invariante rota
  deberia volver como error publico recuperable o issue durable, no tumbar el
  proceso residente.

## T140 backlog-overlap-canonicalization

Objetivo: consolidar tareas Txx ya duplicadas o solapadas en el backlog de
automejora sin borrar historia y sin perder criterios de cierre.

Estado: pendiente.

Alcance:

- `modulos/orquesta-autoprogramming`
- `modulos/orquesta-server`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Detectar solapes por objetivo, alcance, tests y evidencia, no solo por numero
  Txx ni heading textual.
- Resolver pares existentes como `T102`/`T137` y `T103`/`T138` con una seccion
  canonica y aliases `reemplazada_por`/`fusionada_con`; no borrar las secciones
  historicas sin revision.
- El planner residente debe elegir la tarea canonica y bloquear o marcar como
  `duplicate_backlog_task` cualquier candidata no canonica equivalente.
- El merge debe conservar tests, riesgos y criterios no redundantes de ambas
  entradas y exponer cambios como refs/lineas compactas para review del
  director.
- Coordinar con T43, T79, T116 y T135: esta tarea corrige duplicados ya
  presentes; no sustituye leases de merge, sharding, linter de docs ni fallback
  seguro del planner.
- Tests: `go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-server ./cmd/orquesta-server`.

## T141 public-boundary-no-panic-contract

Objetivo: impedir que codigo no-test de control plane, smokes opt-in o nucleo
neutral use `panic` para errores que pueden llegar desde configuracion, IO,
JSON, reloj o invariantes rotas.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-core-leases`
- `modulos/orquesta-server`
- `modulos/orquesta-observability`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Sustituir `panic` no-test en rutas ejecutables por errores publicos
  recuperables, issues durables o validaciones previas con test que demuestre
  que la rama no alcanza input externo.
- Helpers `must*` solo pueden quedar en tests o en inicializacion cerrada con
  prueba de invariante; en comandos y validadores deben devolver error.
- El servidor residente y los smokes opt-in deben reportar `internal_invariant`
  o codigo equivalente con correlation/ref, sin volcar stack, prompt,
  transcript, body HTTP, rutas locales ni tokens.
- La observabilidad debe poder contar panics recuperados como fallo operacional
  si se instala `recover` en frontera HTTP/comando, pero no usar `recover` como
  sustituto de validar entradas.
- Coordinar con T85, T97, T107 y T139: esta tarea gobierna crash/no-panic; las
  otras gobiernan status, logs, diagnosticos de tests y shape publico.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-core-leases ./modulos/orquesta-server ./modulos/orquesta-observability`.

## Escaneo backlog 2026-05-24 cuadragesima octava pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, documentos vigentes de estado,
  nucleo, principio y matriz.
- Backlog, railes observados y duplicaciones vigentes hasta `T141`.
- Busquedas `rg` sobre `io.ReadAll`, `os.ReadFile`, `json.NewDecoder`,
  `agent_ack.json`, `director_decisions.json`, `orquesta_shutdown_request`,
  `agent_shutdown_checkpoint_ack.json` y `codex_last_message`.
- Lectura focal de `modulos/orquesta-cli/command_flags_v0.go`,
  `modulos/orquesta-runtime-codex/codex_ack_validation_v0.go`,
  `modulos/orquesta-runtime-codex/codex_ack_v0.go`,
  `modulos/orquesta-runtime-codex/codex_shutdown_checkpoint_v0.go` y
  `cmd/orquesta-server/idle_self_improvement_backlog_planner_v0.go`.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
  como refs opacas, no como rutas ni nombres Git.

Huecos detectados:

- La CLI comun (`readCLIInputBytesV0`) lee `stdin` con `io.ReadAll` y
  `--input` con `os.ReadFile` sin limite comun ni clasificacion de origen. En
  uso por agentes/toolbelts, un input grande o una ruta local sensible puede
  convertirse en payload de control plane antes de que apliquen rails de HTTP o
  respuesta.
- Los ficheros de control Codex se validan por contenido, pero varias lecturas
  siguen siendo completas: `agent_ack.json`, `agent_shutdown_checkpoint_ack.json`
  y lecturas compatibles usadas por planner/observacion. T33/T36 gobiernan
  correlacion y terminalidad; falta el rail de tamano/redaccion para que un
  ACK/sidecar enorme no bloquee memoria ni termine en error publico crudo.

## T142 cli-json-input-bounds-and-source-policy

Objetivo: acotar y clasificar la entrada JSON de CLI antes de enviarla a APIs
publicas o convertirla en contrato operativo.

Estado: pendiente.

Alcance:

- `modulos/orquesta-cli`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `readCLIInputBytesV0` debe leer `stdin` y `--input` con limite configurable
  por comando, error publico `input_too_large` o equivalente local y sin volcar
  el contenido recibido.
- La fuente de entrada debe quedar clasificada como `stdin`, `file_explicit` o
  `inline` si aparece en futuro; paths absolutos, HOME, `.orquesta-runtime`,
  prompts, transcripts, logs y ficheros de control deben bloquearse o requerir
  opt-in local con mensaje no exportable.
- Mantener compatibilidad para operador local que pasa `--input -`, pero los
  agentes/toolbelts deben preferir refs o JSON pequeno y no cargar ficheros
  arbitrarios del workspace.
- Coordinar con T103, T138 y T139: esas tareas gobiernan respuestas y salida
  publica; esta gobierna la entrada local antes de construir requests.
- Tests: `go test -count=1 ./modulos/orquesta-cli ./cmd/orquesta-server`.

## T143 codex-control-file-size-and-redaction-policy

Objetivo: unificar limite, lectura y error publico para ficheros de control
Codex antes de validar ACK, decisiones o checkpoint.

Estado: pendiente.

Alcance:

- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-runtime-codex-delivery`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Crear helper de lectura acotada para `agent_ack.json`,
  `director_decisions.json`, `agent_shutdown_checkpoint_ack.json` y lecturas
  compatibles del planner/observation source; sobrelimite devuelve issue
  compacto (`control_file_too_large` o equivalente), no cuerpo ni path local.
- Reutilizar ese helper desde validadores y fuentes de observacion sin meter
  filesystem, Codex ni paths locales en el nucleo neutral.
- Mantener el tail acotado de logs/progreso como frontera separada; no mezclar
  logs crudos con ACK/decisiones/checkpoint ni usarlos como cierre terminal.
- La politica debe coordinar con T33, T36, T42 y T63: correlacion, write-set
  estricto y sidecar causal siguen siendo owners vecinos, pero aqui el foco es
  tamano, lectura y redaccion de ficheros de control.
- Tests: `go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 cuadragesima novena pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, documentos vigentes de estado,
  nucleo y principio.
- Backlog, railes observados y duplicaciones vigentes hasta `T143`.
- Busquedas `rg` sobre `io.ReadAll`, `os.ReadFile`, `CallToolV0`,
  `context.Background`, `DomainWork`, `payload`, `artifact`, `factory-http` y
  fronteras HTTP/MCP.
- Lectura focal de `modulos/orquesta-app-codex-stack/domain_work_delivery_builder_v0.go`,
  `modulos/orquesta-domain-work/validation_v0.go`,
  `modulos/orquesta-operator-mcp-client/operator_mcp_client_v0.go` y
  `modulos/orquesta-factory-http/appspec_http_v0.go`.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
  como refs opacas, no como rutas ni nombres Git.

Huecos detectados:

- La construccion de entregas `domain_work` lee el primer fichero declarado en
  `ACK.files` con `os.ReadFile` y convierte su contenido en payload de
  artefacto. La ruta queda acotada al `ProjectWorkDir`, pero falta limite por
  tipo de artefacto, redaccion por campo y reason code si el fichero es enorme,
  binario o contiene material local que no debe enviarse a la app de dominio.
- `orquesta-operator-mcp-client` llama `CallToolV0` con
  `context.Background()`. Si el conector MCP externo queda colgado, el
  director/operador no recibe timeout publico ni presupuesto observable; T23
  gobierna transporte MCP residente y T136 wait externo, pero no el deadline
  del cliente de operador.
- `orquesta-factory-http` expone `POST /api/v0/apps/spec` con `io.ReadAll` sin
  limite comun ni politica de `Content-Type`/trailing data. T137 cubre
  MCP/web/gateway y T133 documenta falta de guia local en `factory-http`, pero
  el handler concreto queda fuera del alcance de pruebas declarado.

## T144 domain-work-delivery-artifact-intake-policy

Objetivo: acotar la lectura y normalizacion de ficheros de entrega antes de
convertirlos en artefactos `domain_work` o payloads hacia apps externas.

Estado: pendiente.

Alcance:

- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-domain-work`
- `modulos/orquesta-runtime-codex-delivery`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Leer ficheros declarados por ACK con limite por artefacto y error publico
  estable (`domain_work_artifact_too_large`, `domain_work_artifact_unreadable`
  o equivalente), sin volcar path local ni cuerpo crudo.
- Distinguir markdown/texto, JSON estructurado y binario/adjunto; si el
  artefacto requiere binario, debe viajar por ref/attachment validado, no como
  `ValueJSON` o string gigante.
- Aplicar redaccion por campo antes de construir `PayloadFields`: bloquear
  HOME/rutas privadas, prompts/transcripts, tokens, payloads HTTP crudos y
  datos de proveedor; permitir refs opacas y vocabulario operativo.
- Coordinar con T26, T72, T80, T101 y T143: calidad, mapa de artefactos,
  egress HTTP, ledger y control files son vecinos; esta tarea gobierna intake
  del fichero de producto entregado.
- Tests: `go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-domain-work ./modulos/orquesta-runtime-codex-delivery`.

## T145 operator-mcp-client-deadline-budget

Objetivo: dar deadline, cancelacion y error publico al cliente MCP de operador
sin meter transporte real ni operador concreto en el nucleo.

Estado: pendiente.

Alcance:

- `modulos/orquesta-operator-mcp-client`
- `modulos/orquesta-operator-mcp`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `OperatorMCPClientConnectorV0` no debe usar `context.Background()` sin limite;
  el caller debe poder inyectar contexto/deadline o timeout de composicion.
- Timeout/cancelacion del conector debe mapearse a error publico estable
  (`operator_mcp_timeout` o equivalente) y conservar correlation/connector refs
  opacas sin exponer URL, token, prompt, transcript ni detalle del transporte.
- La ausencia de conector sigue siendo `operator_mcp_connector_unavailable`;
  timeout de operador real no debe confundirse con consejo negativo ni con fallo
  de validacion del plan.
- Coordinar con T23, T55, T85 y T136: transporte MCP residente, exposicion de
  control plane, status y presupuesto de espera externa son vecinos; esta tarea
  gobierna el deadline del cliente.
- Tests: `go test -count=1 ./modulos/orquesta-operator-mcp-client ./modulos/orquesta-operator-mcp ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

## T146 factory-http-json-boundary-coverage

Objetivo: incorporar `orquesta-factory-http` a la politica comun de fronteras
JSON publicas y a la cobertura documental local antes de usarlo como entrada de
apps generadas o preview de backlog.

Estado: pendiente.

Alcance:

- `modulos/orquesta-factory-http`
- `modulos/orquesta-factory`
- `modulos/orquesta-http-gateway`
- `modulos/orquesta-app-gateway`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `POST /api/v0/apps/spec` debe leer JSON con limite, content-type esperado,
  rechazo de trailing data/campos desconocidos cuando aplique y error publico
  compacto; no `io.ReadAll` sin bound.
- `factory-http` debe tener guia local o sustituto documental claro antes de
  que agentes automaticos lo traten como modulo sensible modificable.
- La respuesta sigue siendo preview/contrato de factory; no crea plan operativo
  ni cola ejecutable sin handoff causal al Director.
- Coordinar con T123, T133 y T137: preview de backlog, guias locales y helper
  JSON publico son vecinos; esta tarea cubre el modulo omitido en alcance.
- Tests: `go test -count=1 ./modulos/orquesta-factory-http ./modulos/orquesta-factory ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway`.

## Verificacion final esperada

- `git diff --check`
- `go test -count=1 ./...`
- Prueba HTTP real con servidor residente: preparar run, supervisar, revisar
  stats y shutdown limpio sin 500.

## Escaneo backlog 2026-05-24 quincuagesima pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, documentos vigentes de estado,
  nucleo, principio, Director Operativo y cierre generico.
- Backlog, railes observados y duplicaciones vigentes hasta `T146`.
- Busquedas `rg` sobre `os.ReadFile`, `io.ReadAll`, `json.NewDecoder`,
  `context.Background`, ledgers file-based, logs Codex y smokes.
- Lectura focal de
  `modulos/orquesta-runtime-codex-delivery/progress_failure_context_v0.go`,
  `modulos/orquesta-runtime-codex-delivery/progress_action_signature_v0.go`,
  `cmd/orquesta-server/external_bridge_input_ledger.go`,
  `modulos/orquesta-app-codex-stack/domain_work_delivery_file_ledger_v0.go`
  y `modulos/orquesta-domain-work-file/snapshot_v0.go`.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
  como refs opacas, no como rutas ni nombres Git.

Huecos detectados:

- `codexProgressReadFailureLogV0` lee stdout/stderr/last-message completos con
  `os.ReadFile` y recorta despues a 64 KiB. La firma de accion ya usa tail
  acotado, pero la clasificacion de fallo puede cargar un log grande antes de
  detectar capacidad/interrupcion/no-ACK. T58 cubre `codex-wave-tail` publico y
  T143 cubre control files; falta el rail de lectura tail para logs internos de
  progreso/fallo.
- Varios ledgers/snapshots JSON file-based leen todo el fichero antes de validar
  schema o numero de records: el ledger de entrada de bridge externo y el ledger
  de artefactos `domain_work` son ejemplos directos. T98/T101 cubren
  idempotencia/recovery y T104 cubre escritura durable; falta una politica de
  lectura con limite, corrupcion recuperable, max records y error publico
  redactado.

## T147 codex-progress-failure-log-tail-bounds

Objetivo: leer logs Codex de progreso/fallo con tail acotado antes de
clasificar no-ACK, interrupcion o capacidad, sin convertir logs crudos en
evidencia terminal.

Estado: pendiente.

Alcance:

- `modulos/orquesta-runtime-codex-delivery`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `codexProgressReadFailureLogV0` y lectores equivalentes deben usar helper de
  tail acotado antes de asignar memoria al contenido completo del log.
- La clasificacion debe conservar codigos compactos (`capacity_warning`,
  `interrupted`, `no_ack`) y refs de evidencia, pero no exponer stdout/stderr,
  `codex_last_message`, prompts, transcripts, HOME, rutas privadas, tokens ni
  salida cruda de proveedor.
- Coordinar con T58, T106, T128 y T143: tail publico, summaries de wave,
  politica de progreso y control files son vecinos; esta tarea gobierna logs
  internos usados para decidir failure context.
- Tests: `go test -count=1 ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-runtime-codex ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

## T148 file-ledger-snapshot-read-bounds

Objetivo: fijar limites de lectura, records y error publico para ledgers y
snapshots JSON file-based antes de usarlos como evidencia de estado vivo.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-domain-work-file`
- `modulos/orquesta-run-file`
- `modulos/orquesta-state-file`
- `modulos/orquesta-runtime-codex-delivery`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Lectores de snapshots/ledgers deben declarar limite de bytes por tipo,
  `max_records` razonable y error publico compacto (`snapshot_too_large`,
  `ledger_too_large`, `snapshot_corrupt` o equivalente), sin path local ni
  cuerpo crudo.
- El ledger de entrada de bridge externo y el ledger de entrega `domain_work`
  deben rechazar snapshots enormes/corruptos de forma recuperable, sin crear
  runs duplicadas ni reenviar artefactos por asumir ledger vacio.
- La lectura acotada no sustituye T104: permisos, temp, fsync y lock siguen
  siendo politica de escritura durable. T98/T101 siguen siendo owners de claim,
  idempotencia y recovery causal.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack ./modulos/orquesta-domain-work-file ./modulos/orquesta-run-file ./modulos/orquesta-state-file ./modulos/orquesta-runtime-codex-delivery`.

## Escaneo backlog 2026-05-24 quincuagesima primera pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, documentos vigentes de estado,
  nucleo y principio.
- Backlog, railes observados y duplicaciones vigentes hasta `T148`.
- Busquedas `rg` sobre `os.ReadFile`, `io.ReadAll`, `WalkDir`,
  `http.Client`, `context.Background`, `ProcessRuntime`, prompts de ola y
  snapshots de worktree.
- Lectura focal de
  `modulos/orquesta-runtime-worktree/snapshot_v0.go`,
  `modulos/orquesta-runtime-codex-delivery/source_review_gate_project_files_v0.go`,
  `modulos/orquesta-app-codex-stack/review_rework_replan_targets_v0.go`,
  `modulos/orquesta-app-codex-stack/domain_work_delivery_recovery_files_v0.go`,
  `cmd/orquesta-server/codex_director_wave_command_v0.go` y
  `cmd/orquesta-server/codex_wave_command_v0.go`.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
  como refs opacas, no como rutas ni nombres Git.

Huecos detectados:

- `CaptureWorktreeSnapshotV0` recorre el worktree y `hashWorktreeFileV0` lee
  cada fichero completo con `os.ReadFile`. T42/T45/T48 usan el snapshot como
  evidencia para write-set, line budget y efectos destructivos, pero falta
  presupuesto propio de snapshot: max files, max bytes por fichero, max bytes
  total, streaming hash y reason codes redactados.
- Los gates de review/rework y recuperacion de artefactos usan `filepath.WalkDir`
  sobre rutas/globs de proyecto para probar que existe algun fichero. La logica
  esta copiada entre `orquesta-runtime-codex-delivery` y
  `orquesta-app-codex-stack`, sin max de entradas visitadas, presupuesto de
  tiempo, contador de skips ni helper comun de ignore prefixes.
- `codex-director-wave` y `codex-wave` leen `--objective-file`,
  `--prompt-file` y `domain_context_files` con `os.ReadFile` completo. Esos
  ficheros vienen de operador/composicion local, no de contrato de producto, y
  todavia no tienen limite, politica de origen, UTF-8/texto, redaccion ni error
  publico. T56 cubre proyeccion de `CODEX_HOME`; este hueco gobierna inputs de
  prompt/contexto de comando.

## T149 worktree-snapshot-read-budget

Objetivo: convertir el snapshot de worktree en evidencia acotada y streaming,
sin leer ficheros completos ni recorrer arboles sin presupuesto.

Estado: pendiente.

Alcance:

- `modulos/orquesta-runtime-worktree`
- `modulos/orquesta-runtime-codex-delivery`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `CaptureWorktreeSnapshotV0` debe declarar y aplicar `max_files`,
  `max_file_bytes` y `max_total_bytes` con defaults de composicion; el hash debe
  calcularse por streaming, no con `os.ReadFile` completo.
- Los issues deben ser publicos y compactos (`worktree_snapshot_file_too_large`,
  `worktree_snapshot_too_many_files`, `worktree_snapshot_unreadable` o
  equivalente), sin path absoluto, HOME, contenido de fichero ni datos de
  proveedor.
- Reutilizar o coordinar ignore prefixes con T50/T127 para excluir
  `.orquesta-runtime`, `.orquesta-codex-runtime`, artefactos locales, logs,
  prompts, transcripts, caches y control files antes de contar como producto.
- T42/T45/T48 deben consumir este snapshot acotado como evidencia; si no hay
  snapshot valido, el cierre estricto debe bloquear o pedir followup, no confiar
  solo en `ACK.files`.
- Tests: `go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

## T150 project-tree-scan-budget-for-review-recovery

Objetivo: unificar los escaneos de arbol usados por review/rework y recuperacion
de artefactos para que globs y carpetas no recorran proyectos completos sin
limite ni reglas divergentes.

Estado: pendiente.

Alcance:

- `modulos/orquesta-runtime-codex-delivery`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-runtime-worktree`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `codexReviewGateProjectGlobHasFileV0`,
  `codexReviewGateDirHasFileV0`, `reviewReworkProjectGlobHasFileV0`,
  `reviewReworkDirHasFileV0` y `domainWorkRecoveryFilesUnderDirV0` deben
  delegar en un helper comun o politica equivalente con `context`, max entradas,
  max profundidad opcional, ignore prefixes y reason codes.
- Un glob amplio (`**`, `.` o carpeta raiz) debe producir resultado
  determinista: encontrado acotado, `scan_budget_exhausted` o issue publico; no
  debe bloquear el supervisor ni esconder worktree/control files.
- La politica debe permitir alias y rutas hijas razonables, pero cortar rutas
  absolutas, HOME, traversal, `.orquesta-runtime`, `.orquesta-codex-runtime`,
  prompts, transcripts, logs y binarios enormes como evidencia de producto.
- Coordinar con T144 y T149: intake de artefactos y snapshot completo son
  vecinos; esta tarea solo gobierna escaneos heuristicos de existencia/recovery.
- Tests: `go test -count=1 ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./modulos/orquesta-runtime-worktree`.

## T151 codex-wave-operator-file-input-bounds

Objetivo: acotar los ficheros locales que alimentan prompts, objetivos y
contexto de olas Codex antes de construir paquetes para agentes.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-runtime-worktree`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `codexWavePromptTextV0`, `codexDirectorObjectiveTextV0` y
  `codexDirectorDomainContextBlocksFromFilesV0` deben leer con limite,
  validar texto/UTF-8 razonable y devolver errores publicos
  (`prompt_file_too_large`, `objective_file_invalid`,
  `domain_context_file_unreadable` o equivalente), sin cuerpo crudo ni path
  local completo.
- La politica de origen debe distinguir fichero local de operador, ref opaca y
  fichero de control. `.orquesta-runtime`, `.orquesta-codex-runtime`, logs,
  ACKs, checkpoints, prompts/transcripts previos, HOME y rutas privadas no deben
  entrar como contexto salvo opt-in local auditado y redactado.
- Los summaries de `codex-wave`/`codex-director-wave` deben registrar categorias
  y hashes/refs compactas de esos inputs, no contenido completo ni rutas
  privadas.
- Coordinar con T56, T106, T143 y T147: proyeccion de `CODEX_HOME`, salida
  publica, control files y logs de progreso son vecinos; esta tarea gobierna
  inputs de prompt/contexto antes del launch.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree`.

## Escaneo backlog 2026-05-24 quincuagesima segunda pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, documentos vigentes de estado,
  nucleo, principio y matriz.
- Backlog, railes observados y duplicaciones vigentes hasta `T151`.
- Busquedas `rg` sobre `os.ReadFile`, `filepath.WalkDir`, `json.RawMessage`,
  `context.Background`, `http.Client`, `CODEX_HOME`, comandos/eventos/outbox y
  puertos de efecto externo.
- Lectura focal de `cmd/orquesta-server/codex_wave_command_v0.go`,
  `modulos/orquesta-core-workflow/commands_v0.go`,
  `modulos/orquesta-core-workflow/events_validation_v0.go`,
  `modulos/orquesta-core-workflow/outbox_validation_v0.go`,
  `modulos/orquesta-state-file/event_sink_v0.go`,
  `modulos/orquesta-domain-work-http/client_v0.go`,
  `modulos/orquesta-opes-connector/http_v0.go`,
  `modulos/orquesta-operator-mcp-client/operator_mcp_client_v0.go` y rutas de
  launch/drain/shutdown del servidor.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
  como refs opacas, no como rutas ni nombres Git.

Huecos detectados:

- La copia de `CODEX_HOME` para olas Codex esta allowlisteada por nombres, pero
  `codexWaveCopyFileIfExistsV0` usa `os.Stat`/`os.ReadFile` completos y
  `codexWaveCopyDirIfExistsV0` recorre `skills`, `plugins`, `rules` y
  `memories` sin presupuesto de ficheros/bytes ni recibo de symlinks, modos o
  categorias omitidas. T56 gobierna que categorias pueden proyectarse; falta la
  seguridad mecanica de la copia.
- `outbox` ya tiene limite de payload, pero comandos y eventos del workflow no
  comparten un presupuesto equivalente: `newOrchestrationCommandV0` serializa
  payloads sin limite y `validateEventPayloadV0` hace `json.Unmarshal` completo
  antes de validar. `orquesta-state-file` compacta y persiste esos payloads como
  eventos, por lo que un payload grande o crudo puede llegar antes de los rails
  de persistencia.
- Hay puertos de efecto externo que aceptan contexto nil o se invocan desde
  `context.Background()` y dependen de timeouts locales o del cliente inyectado:
  domain-work HTTP, OPES REST, MCP operador, launch de ola Codex, bridge externo
  y shutdown. T145 cubre el cliente MCP operador y T136 la espera externa del
  residente; falta una politica comun para que cada efecto tenga deadline,
  cancelacion y error publico.

## T152 codex-code-home-copy-bounds-symlink-policy

Objetivo: hacer que la proyeccion de `CODEX_HOME` hacia agentes Codex tenga
presupuesto de copia, politica de symlinks/modos y recibo compacto, como shard
operativo de T56.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-runtime-worktree`
- `modulos/orquesta-app-codex-stack`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- La copia de `auth.json`, `config.toml`, `skills`, `plugins`, `rules` y
  `memories` debe aplicar `max_files`, `max_file_bytes`, `max_total_bytes` y
  categorias permitidas por politica de composicion.
- Usar `Lstat`/resolucion segura: symlinks, hardlinks raros, dispositivos,
  sockets y entradas no regulares se rechazan o se omiten con reason code
  publico; `os.Stat` no debe seguir symlinks a material fuera de la fuente
  permitida.
- No conservar modos ejecutables ni permisos amplios salvo scripts allowlist
  explicitados; defaults seguros `0600` para ficheros y `0700` para dirs.
- El recibo durable debe listar categorias copiadas, contadores, bytes y
  omissions; no rutas absolutas, HOME, tokens, contenido de memorias/plugins ni
  prompts/transcripts.
- Coordinar con T50/T56/T106/T127: control files, credenciales, summary publico
  y artefactos ignorados siguen como owners vecinos.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack`.

## T153 workflow-command-event-payload-budget

Objetivo: unificar presupuesto y redaccion basica para payloads de comandos y
eventos del workflow antes de persistirlos o proyectarlos.

Estado: pendiente.

Alcance:

- `modulos/orquesta-core-workflow`
- `modulos/orquesta-state-file`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-director`
- `modulos/orquesta-app-director-service`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Definir limite comun o por tipo para `OrchestrationCommandV0.Payload` y
  `OrchestrationEventV0.Payload`, equivalente al rail ya existente de outbox,
  con error publico `payload_too_large` o mapeo estable a `payload_invalido`.
- Validar tamano antes de `json.Unmarshal` completo y antes de compactar/persistir
  en `orquesta-state-file`.
- Payloads grandes deben viajar por refs de artefacto/evidencia y no por JSON
  crudo dentro de comandos/eventos; prompts, transcripts, cuerpos HTTP, HOME,
  rutas privadas, tokens y datos de proveedor quedan fuera de eventos durables.
- Mantener compatibilidad de eventos existentes mediante limites razonables y
  pruebas de replay; no cambiar JSON tags, refs, idempotency keys ni orden
  causal.
- Coordinar con T41/T104/T148: auditoria, escritura durable y lectura de
  ledgers/snapshots son vecinos; esta tarea gobierna el contrato de payload del
  workflow.
- Tests: `go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-state-file ./modulos/orquesta-orchestration-core ./modulos/orquesta-director ./modulos/orquesta-app-director-service`.

## T154 effect-port-deadline-context-policy

Objetivo: hacer que puertos y adaptadores con efectos externos tengan deadline
observable y cancelacion propagada, sin depender de `context.Background()` como
contrato de ejecucion.

Estado: pendiente.

Alcance:

- `modulos/orquesta-runtime`
- `modulos/orquesta-domain-work-http`
- `modulos/orquesta-opes-connector`
- `modulos/orquesta-operator-mcp-client`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Launch/stop runtime, HTTP domain_work, OPES REST, MCP operador, bridge externo
  y shutdown deben recibir `context.Context` con deadline o politica de timeout
  de composicion; nil context solo queda como compatibilidad legacy documentada.
- Timeouts deben mapear a errores publicos estables (`effect_timeout`,
  `operator_mcp_timeout`, `domain_work_http_timeout`, etc.) con correlation/ref
  opaca y sin URL completa, HOME, tokens, prompt, transcript ni payload crudo.
- Si un cliente inyectado no tiene timeout propio, el caller debe envolver la
  llamada con deadline por perfil de trabajo; no usar `context.Background()` en
  rutas residentes con efecto externo.
- La politica debe distinguir espera externa legitima de bloqueo por conector
  colgado, y alimentar status/auditoria con contadores compactos.
- Coordinar con T23/T80/T136/T145: transporte MCP, egress HTTP, presupuesto de
  wait externo y cliente operador son vecinos; esta tarea gobierna la
  propagacion general de deadline en efectos.
- Tests: `go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-domain-work-http ./modulos/orquesta-opes-connector ./modulos/orquesta-operator-mcp-client ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 quincuagesima tercera pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, documentos vigentes de estado,
  nucleo y principio.
- Backlog, railes observados y duplicaciones vigentes hasta `T154`.
- Busquedas `rg` sobre autoprogramacion idle, scanner de backlog, ACKs,
  `WalkDir`, `os.ReadFile`, `CombinedOutput`, AppVCS, promotion y comandos Git.
- Lectura focal de `cmd/orquesta-server/idle_self_improvement_backlog_planner_v0.go`,
  `modulos/orquesta-runtime-worktree/app_vcs_git_v0.go`,
  `modulos/orquesta-runtime-worktree/staging_promotion_v0.go`,
  `modulos/orquesta-runtime-codex-delivery/source_review_gate_project_files_v0.go`
  y docs de tareas vecinas T33/T39/T79/T135/T139/T143/T150/T154.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
  como refs opacas, no como rutas ni nombres Git.

Huecos detectados:

- `completedBacklogRequestRefsV0` recorre toda `.orquesta-runtime` buscando
  `agent_ack.json` para saber que requests de backlog ya cerraron. En un
  workspace con olas Codex previas, homes copiados, plugins y children, ese
  `WalkDir` puede atravesar material ajeno al planner. T33 gobierna correlacion
  del ACK, T79 sharding, T135 fallback y T143 tamano/redaccion de control
  files; falta el presupuesto mecanico de scan, profundidad y degradacion
  observable del planner.
- `GitAppVCSConnectorV0` y la promocion de staging usan `git status
  --porcelain --untracked-files=all`, `commit`, `push` y `rev-parse` mediante
  `CombinedOutput()`. T39 cubre write-set/evidencia causal y T139 la salida
  publica de comandos, pero falta un limite de salida Git/numero de paths antes
  de parsear, auditar o devolver errores.

## T155 idle-backlog-runtime-ack-scan-budget

Objetivo: acotar el scan de ACKs de automejora idle para que el planner no
recorra todo el runtime local ni confunda falta de presupuesto con backlog
cerrado o vacio.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `completedBacklogRequestRefsV0` no debe hacer `WalkDir` ilimitado sobre toda
  `.orquesta-runtime`; debe usar indice/refs esperadas o limites de
  `max_dirs`, `max_files`, profundidad y tiempo, omitiendo homes de agentes,
  plugins, waves historicas, prompts/transcripts y artefactos grandes.
- Cada ACK leido para deduplicar backlog debe tener limite de bytes, schema,
  `request_id`/`ack_ref` correlacionable y status terminal; si falta correlacion
  se reporta como ambiguo, no como completed.
- Si el scan agota presupuesto o encuentra material corrupto, el planner debe
  exponer `backlog_ack_scan_budget_exhausted`/`backlog_ack_scan_ambiguous` en
  status/auditoria y conservar un scanner documental acotado; no relanzar codigo
  generico ni ocultar huecos pendientes.
- Coordinar con T33/T79/T135/T143: esta tarea no redefine correlacion de ACK,
  sharding del backlog, fallback del planner ni politica general de control
  files; gobierna el scan runtime usado para deduplicar trabajo completado.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`.

## T156 app-vcs-git-output-and-path-budget

Objetivo: limitar salida y cardinalidad de rutas en comandos Git de AppVCS y
promocion de staging antes de usarlos como evidencia, errores publicos o lista
de cambios.

Estado: pendiente.

Alcance:

- `modulos/orquesta-runtime-worktree`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-mcp`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Sustituir `CombinedOutput()` por captura con `max_output_bytes` para stdout y
  stderr de Git; errores por exceso deben devolver codigo publico estable
  (`git_output_too_large`, `git_status_too_many_paths`) sin salida cruda.
- `git status --porcelain --untracked-files=all` debe aplicar `max_changed_paths`
  y reason code cuando el repo excede presupuesto; commit/promotion no deben
  pasar de ese estado a `git add -A` ni a push sin `write_set`/`commit_paths`
  explicitos y decision auditable.
- La evidencia de error Git debe redactar remotos, URLs con credenciales, rutas
  locales, HOME, tokens, ramas privadas y fragmentos de diff; conservar solo
  accion, contadores, timeout/truncation y refs opacas.
- Reutilizar la politica de deadline de T154 o el timeout AppVCS existente, pero
  reportar timeout como `git_command_timeout` con salida truncada/redactada.
- Coordinar con T39/T105/T139/T154: write-set/push causal, launch IO neutral,
  shape publico de comandos y deadline general siguen como owners vecinos; esta
  tarea gobierna output/path budget especifico de Git AppVCS.
- Tests: `go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 quincuagesima cuarta pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, documentos vigentes de estado,
  nucleo y principio.
- Backlog, railes observados y duplicaciones vigentes hasta `T156`.
- Busquedas `rg` sobre `os.ReadFile`, `os.Stat`, `io.ReadAll`,
  `director_decisions.json`, `agent_ack.json`, checkpoint, OPES bridge,
  base URLs, summaries publicos y conectores HTTP.
- Lectura focal de `modulos/orquesta-runtime-codex/codex_ack_v0.go`,
  `modulos/orquesta-runtime-codex/codex_shutdown_checkpoint_v0.go`,
  `modulos/orquesta-runtime-codex-delivery/director_decision_file_descriptors_v0.go`,
  `modulos/orquesta-runtime-codex-delivery/progress_action_signature_v0.go`,
  `cmd/orquesta-server/opes_bridge_config.go`,
  `cmd/orquesta-server/opes_bridge.go`,
  `cmd/orquesta-server/opes_bridge_submit.go` y
  `modulos/orquesta-opes-connector/http_v0.go`.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
  como refs opacas, no como rutas ni nombres Git.

Huecos detectados:

- T143 cubre tamano/redaccion de ficheros de control Codex, pero no fija una
  politica comun de raiz autorizada, symlinks y tipo de fichero. `agent_ack`,
  checkpoint, `director_decisions` y tails de progreso se leen desde paths de
  descriptor con `os.ReadFile`, `os.Stat` u `os.Open`; si el descriptor queda
  corrupto o apunta a symlink/entrada no regular, la proteccion depende de
  helpers locales distintos.
- El bridge OPES ya exige confirmacion y filtro por job antes de crear runs,
  pero `ORQUESTA_OPES_BASE_URL`/`OPES_BASE_URL` y `ORQUESTA_BASE_URL` se aceptan
  como strings de composicion y se vuelcan en el summary publico. T80 gobierna
  egress HTTP neutral de `domain_work` y T139 salida de comandos; falta la
  politica OPES especifica de destino temporal/productivo, URL sin credenciales
  y summary redactado.

## T157 codex-control-file-root-and-symlink-policy

Objetivo: asegurar que ACKs, sidecars, checkpoints y tails de progreso Codex se
leen solo desde raices de control autorizadas, sin seguir symlinks ni aceptar
entradas no regulares como evidencia operativa.

Estado: pendiente.

Alcance:

- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-runtime-codex-delivery`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Centralizar una politica de lectura para `agent_ack.json`,
  `director_decisions.json`, `agent_shutdown_checkpoint_ack.json`, checkpoint
  request y logs/tails de progreso que valide raiz autorizada, nombre base,
  agente/run esperados y descriptor correlacionado antes de abrir el fichero.
- Usar `Lstat`/apertura segura: symlinks, hardlinks raros, dispositivos,
  sockets, dirs y ficheros no regulares se rechazan con reason code publico
  (`control_file_not_regular`, `control_file_outside_root` o equivalente), sin
  path local ni contenido.
- Coordinar con T143: T143 mantiene limite de bytes/redaccion de contenido; esta
  tarea gobierna frontera de path, raiz y tipo de fichero antes de leer.
- Coordinar con T50/T63/T147: control files fuera del producto, recibo del
  sidecar y tails de progreso siguen como owners vecinos; no usar logs como
  cierre terminal.
- Tests: `go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

## T158 opes-bridge-destination-and-summary-policy

Objetivo: cerrar la politica OPES especifica de destino HTTP y summary publico
del bridge, separada del adaptador HTTP neutral y sin tocar OPES productivo por
defecto.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-opes-connector`
- `modulos/orquesta-opes-bridge`
- `docs/runbooks`
- `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `ORQUESTA_OPES_BASE_URL`/`OPES_BASE_URL` y `ORQUESTA_BASE_URL` deben pasar por
  politica de destino: esquema permitido, credenciales en URL rechazadas,
  loopback/temporal permitido con confirmacion, productivo solo con opt-in
  explicito y evidence ref compacta.
- El summary de `opes-drain-once` debe exponer refs/categorias de destino,
  contadores, filtro aplicado y estado; no URL completa, query sensible, host
  privado, payload OPES, tokens ni cuerpos HTTP.
- El cliente OPES debe conservar paths controlados y errores publicos compactos;
  status no-2xx y errores de decode no deben filtrar respuesta cruda ni destino
  completo.
- Coordinar con T12/T21/T80/T98/T103/T139: smoke OPES temporal, catalogo de
  smokes, HTTP neutral, ledger de entrada, respuestas salientes y shape publico
  siguen como owners vecinos; esta tarea cubre destino y summary del bridge OPES.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-opes-connector ./modulos/orquesta-opes-bridge`.

## Escaneo backlog 2026-05-24 quincuagesima quinta pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, documentos vigentes de estado,
  nucleo, principio y matriz de smokes.
- Backlog, railes observados y duplicaciones vigentes hasta `T158`.
- Busquedas `rg` sobre `os.WriteFile`, `os.MkdirAll`,
  `director_decisions.json`, `agent_packet.json`, `agent_ack.json`,
  checkpoint y limites de batches de decisiones.
- Lectura focal de `modulos/orquesta-runtime-codex/codex_resolver_v0.go`,
  `modulos/orquesta-runtime-codex/codex_shutdown_checkpoint_v0.go`,
  `cmd/orquesta-server/codex_wave_command_v0.go`,
  `modulos/orquesta-director-agent-file-source/source_v0.go`,
  `modulos/orquesta-director-agent-file-source/file_reader_v0.go`,
  `modulos/orquesta-runtime-codex-delivery/director_decision_file_descriptors_v0.go`,
  `modulos/orquesta-app-codex-stack/composite_decision_source_v0.go` y
  `modulos/orquesta-app-codex-stack/director_decision_contract_v0.go`.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
  como refs opacas, no como rutas ni nombres Git.

Huecos detectados:

- T143/T157 cubren lectura de ficheros de control Codex, pero la escritura de
  `agent_packet.json`, `agent_prompt.txt`, wrappers, registry de ola y
  `orquesta_shutdown_request.json` sigue repartida entre runtime Codex, comando
  de ola y stack con `os.WriteFile` directo. Sin politica comun de escritura
  atomica, permisos, `fsync`, rechazo de symlinks y receipt compacto, una
  escritura parcial o una ruta corrupta puede dejar al agente con packet
  incompleto o evidencia terminal ambigua.
- `orquesta-director-agent-file-source` limita bytes por fichero de
  `director_decisions.json`, pero no hay presupuesto visible para numero de
  descriptores, numero total de decisiones agregadas desde varias fuentes ni
  numero de `create_microtask` aceptadas desde sidecars. T63 gobierna correlacion
  del sidecar y T153 gobierna payload de comandos/eventos, pero falta el rail de
  fan-in antes de que una decision batch amplia produzca outbox, waits o tareas
  fuera del presupuesto operativo.

## T159 codex-control-file-durable-write-policy

Objetivo: unificar la escritura durable de ficheros de control Codex para que
packet, prompt, wrappers, registry y requests de checkpoint no dependan de
`os.WriteFile` directo ni de rutas finales sin receipt.

Estado: pendiente.

Alcance:

- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-runtime-codex-delivery`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Centralizar helper/puerto de escritura de control files con raiz autorizada,
  nombre permitido, `Lstat` previo, rechazo de symlink/dir/no regular,
  temp-file en la misma raiz, `rename`, `fsync` cuando aplique y permisos
  cerrados (`0600` o wrapper ejecutable justificado).
- Cada escritura debe producir receipt compacto con `control_file_ref`, tipo,
  bytes, hash, modo, run/agent/checkpoint esperado y estado; no path local,
  prompt, transcript, token, HOME ni contenido crudo.
- La politica debe distinguir crear, reemplazar idempotente y conflicto de
  payload; no sobrescribir packet/prompt/request divergente sin reason code
  publico y correlacion.
- Coordinar con T30/T50/T63/T104/T143/T157: checkpoint neutral, exclusion de
  control files, sidecar causal, escritura durable generica, limites de lectura
  y raiz/symlink siguen como owners vecinos; esta tarea gobierna escritura de
  control files Codex.
- Tests: `go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

## T160 director-decisions-batch-budget-and-source-limit

Objetivo: acotar el fan-in de `director_decisions.json` y fuentes de decisiones
del Director para que varios sidecars o descriptors no generen microtareas,
outbox ni waits por encima del presupuesto operativo.

Estado: pendiente.

Alcance:

- `modulos/orquesta-director-agent-file-source`
- `modulos/orquesta-director-agent-workflow`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-orchestration-core`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Definir presupuesto por request para descriptors leidos, decisions totales,
  `create_microtask`, tasks nuevas, outbox esperado y bytes acumulados; al
  exceder, devolver reason code publico (`director_decisions_batch_too_large` o
  equivalente) sin consumir parcialmente un batch ambiguo.
- `compositeDirectorDecisionSourceV0` debe agregar fuentes con limite y reporte
  causal; una fuente invalida o excesiva no debe ocultar decisiones validas ya
  reflejadas ni relanzar tareas historicas sin revision.
- Los limites deben respetar presupuesto de fanout/profundidad ya existente para
  recursion, `MaxCommands` del ciclo y `WorkflowTaskStore`; no crear otro rail
  textual por prompt ni depender del agente para auto-limitarse.
- La observabilidad/status debe emitir contadores compactos de descriptors,
  decisions aceptadas, decisions omitidas y causa; sin rutas, bodies JSON,
  prompts, transcripts ni contenido de tareas completas.
- Coordinar con T31/T57/T63/T68/T153: leases de cola, outbox durable, sidecar
  causal, presupuestos anidados y payload de workflow siguen como owners
  vecinos; esta tarea gobierna fan-in de decisiones antes de materializar
  efectos.
- Tests: `go test -count=1 ./modulos/orquesta-director-agent-file-source ./modulos/orquesta-director-agent-workflow ./modulos/orquesta-app-codex-stack ./modulos/orquesta-orchestration-core`.

## Escaneo backlog 2026-05-24 quincuagesima sexta pasada

Evidencia revisada: paquete del scanner, `AGENTS.md`, `README.md`,
`docs/estado_actual_2026-05-17.md`,
`docs/guia_nucleo_orquestacion_2026-05-17.md`, este backlog, rail errors,
duplicaciones de rails y busquedas acotadas con `rg` sobre `time.Now`,
`UnixNano`, `RFC3339`, `RequestID`, `CorrelationID`, `IdempotencyKey` y
`X-Correlation-ID` en `cmd` y `modulos`. No se programa codigo desde este
scanner. `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
`branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan como
refs opacas, no como rutas ni nombres Git.

Huecos detectados:

- La generacion de timestamps y refs temporales esta repartida: `codex-wave`
  genera `wave_ref` con resolucion de segundo, el stack y runtime rellenan
  `OccurredAt`/`RequestedAt` con `time.Now`, el planner usa refs derivadas por
  hash local y web/CLI caen a IDs basados en `UnixNano` si falta entropia. T31,
  T33, T57 y T63 cubren leases/correlacion en fronteras concretas, pero no hay
  owner comun para reloj, generador de refs y colisiones bajo concurrencia.
- Los clientes publicos web/CLI/MCP y comandos de servidor propagan
  `request_id`, `correlation_id` e `idempotency_key` con reglas distintas:
  algunas mutaciones aceptan idempotency opcional, otras reutilizan
  `request_id` como correlacion y otras generan IDs locales. T102/T137 cubren
  JSON publico y T139 shape de salida, pero falta una politica comun para
  mutaciones reintentables de control plane.

## T161 clock-and-ref-generation-policy

Objetivo: unificar reloj y generacion de refs publicas/operativas para evitar
colisiones, replay no determinista y timestamps usados como identidad causal.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-server`
- `modulos/orquesta-runtime`
- `modulos/orquesta-runtime-codex-delivery`
- `modulos/orquesta-app-codex-stack`
- `modulos/orquesta-web`
- `modulos/orquesta-cli`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Definir puerto/helper de composicion para `ClockV0` y `RefGeneratorV0` con
  reloj inyectable en tests, monotonicidad donde aplique y salida UTC
  normalizada; el core puro solo recibe valores ya resueltos.
- No usar timestamps de segundo como refs de ola/run/request si puede haber
  concurrencia; combinar prefijo opaco, scope, contador/entropia y deteccion de
  colision o exigir ref explicita del director.
- Las marcas temporales (`occurred_at`, `started_at`, `updated_at`,
  `observed_at`) deben ser evidencia, no identidad terminal ni criterio unico
  de orden causal cuando existan eventos/outbox/receipts.
- Los fallbacks por falta de entropia deben producir issue publico o modo
  degradado observable, no IDs silenciosos basados solo en `UnixNano`.
- Coordinar con T31/T33/T57/T63/T92/T128: leases, ACK correlado, outbox durable,
  sidecar causal, capacidad y progreso conservan owners propios; esta tarea
  gobierna reloj/ref generation compartido.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack ./modulos/orquesta-web ./modulos/orquesta-cli`.

## T162 public-client-mutation-idempotency-policy

Objetivo: alinear `request_id`, `correlation_id` e `idempotency_key` en clientes
publicos y mutaciones del control plane para que retries, timeouts y errores
parciales no dupliquen efectos ni pierdan trazabilidad.

Estado: pendiente.

Alcance:

- `modulos/orquesta-web`
- `modulos/orquesta-cli`
- `modulos/orquesta-mcp`
- `cmd/orquesta-server`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-http-gateway`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Clasificar endpoints/tools/comandos mutables frente a lecturas; mutaciones
  como prepare-run, run-control, queue-priority, app-vcs, shutdown y bootstrap
  deben exigir o derivar `idempotency_key` estable por intento, con error
  publico si no puede preservarse.
- `correlation_id` debe ser trazabilidad transversal y no sustituir
  `request_id` ni `idempotency_key`; si se deriva por compatibilidad, debe
  quedar indicado en la proyeccion publica.
- Web, CLI, MCP y gateways deben compartir normalizacion, validacion, header
  `X-Correlation-ID` y redaccion de errores, sin copiar reglas por cliente.
- En retries tras timeout, respuesta 5xx o cierre de conexion, el cliente debe
  poder reenviar la misma idempotency key y obtener receipt/estado observable en
  vez de crear una mutacion nueva silenciosa.
- No filtrar tokens, URLs con credenciales, HOME, rutas privadas ni payloads
  completos en headers, errores, auditoria o ACKs; solo refs opacas, codigos y
  contadores.
- Coordinar con T31/T39/T47/T55/T102/T137/T139/T154: leases, AppVCS, receipts
  de test, control plane, JSON publico, shape de salida y deadlines siguen como
  owners vecinos; esta tarea gobierna identidad/idempotencia de entrada publica.
- Tests: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./modulos/orquesta-mcp ./cmd/orquesta-server ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway`.

## Escaneo backlog 2026-05-24 quincuagesima septima pasada

Evidencia revisada: paquete del scanner, `AGENTS.md`, `README.md`,
`docs/estado_actual_2026-05-17.md`,
`docs/guia_nucleo_orquestacion_2026-05-17.md`,
`docs/principio_orquesta_piensa_director.md`,
`docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`, este backlog, rail
errors, duplicaciones de rails y busquedas acotadas con `rg` sobre
`AppendRunEventsV0`, `LoadRunEventsV0`, `LoadWorkflowTasksByParentV0`,
`os.ReadDir`, `os.Setenv`, `ORQUESTA_STARTUP_CLEANUP_MODE` y rails de entorno.
No se programa codigo desde este scanner. `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
como refs opacas, no como rutas ni nombres Git.

Huecos detectados:

- `orquesta-state-file` guarda todos los eventos de un run en un unico JSON por
  run: `AppendRunEventsV0` lee el documento completo, deduplica por
  `event_id` recorriendo todo el slice y reescribe el snapshot completo. T148
  cubre limites de lectura de ledgers/snapshots y T153 cubre payloads de
  comandos/eventos, pero falta owner para crecimiento de log de eventos,
  indice por `event_id`, paginacion/compaction y degradacion observable durante
  runs residentes largas.
- `LoadWorkflowTasksByParentV0` hace `os.ReadDir` del directorio de tasks del
  run y abre cada JSON para filtrar por `parent_task_ref`. La recursion real,
  cierre por arbol y waits por parent dependen de ese camino; T160 limita fan-in
  de decisiones antes de crear tasks, pero falta presupuesto de lectura/listado
  e indice de parent/child en el store durable.
- `serverConfigFromEnvV0` y `ensureServerDetailRailsDefaultV0` fijan defaults
  con `os.Setenv` sobre el entorno global del proceso. T129 gobierna la
  proyeccion de entorno al daemon, pero no el rail de configuracion que muta
  estado global y puede afectar tests, comandos hijos, smokes o lecturas
  posteriores de config como si fueran input explicito del operador.

## T163 state-file-event-log-index-compaction-budget

Objetivo: acotar y hacer indexable el log de eventos file-based por run para
que append, replay y deduplicacion no dependan de leer y reescribir un JSON
creciente completo en cada ciclo residente.

Estado: pendiente.

Alcance:

- `modulos/orquesta-state-file`
- `modulos/orquesta-core-workflow`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-app-director-service`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `AppendRunEventsV0` debe aplicar presupuesto por append y por run
  (`max_events`, `max_event_bytes`, `max_snapshot_bytes` o equivalente) antes
  de cargar o persistir, con error publico compacto si excede.
- La deduplicacion por `event_id` debe usar indice durable o estructura
  auxiliar verificable, no recorrido O(n) ilimitado del documento completo en
  cada append.
- `LoadRunEventsV0` debe admitir lectura paginada o por ventana causal para
  cierres/replay; si una composicion necesita el historial completo, debe
  declarar presupuesto y degradar con reason code observable, no asumir log
  vacio ni completed.
- Compaction/snapshot debe preservar orden causal, idempotencia por payload
  canonico y compatibilidad con documentos existentes; un evento conflictivo no
  puede sobrescribir el anterior.
- Coordinar con T104/T148/T153/T161: escritura durable, lectura acotada,
  payload de comandos/eventos y clock/ref generation son owners vecinos; esta
  tarea gobierna crecimiento, indice y replay del log de eventos.
- Tests: `go test -count=1 ./modulos/orquesta-state-file ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service`.

## T164 workflow-task-store-parent-index-budget

Objetivo: evitar que la recuperacion de hijos por `parent_task_ref` escanee
todas las tasks del run sin presupuesto ni indice durable.

Estado: pendiente.

Alcance:

- `modulos/orquesta-state-file`
- `modulos/orquesta-core-workflow`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-app-director-service`
- `modulos/orquesta-app-codex-stack`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `LoadWorkflowTasksByParentV0` no debe depender de `os.ReadDir` ilimitado y
  lectura completa de cada task del run; debe usar indice parent/child durable,
  presupuesto de entradas/bytes o consulta acotada por refs conocidas.
- El indice debe conservar `run_ref`, `task_ref`, `parent_task_ref`,
  `wave_ref`, `cohort_ref`, `delegation_depth` y estado suficiente para waits
  por parent sin reconstruir metadata desde eventos compactos.
- Si el indice falta o esta corrupto, recovery debe rematerializarlo con
  presupuesto y reason code (`workflow_task_parent_index_rebuild_required`,
  `workflow_task_parent_index_budget_exhausted`), no cerrar ni reabrir arboles
  silenciosamente.
- Recursion y cierre de arbol deben bloquear si hay hijos no indexados,
  outbox pendiente o fanout fuera de presupuesto; no convertir una ausencia de
  indice en "sin hijos".
- Coordinar con T28/T31/T57/T78/T160: reconciliacion de agentes vivos, leases
  de cola, outbox durable, merge de plan state y fan-in de decisiones conservan
  owners propios; esta tarea cubre lectura/indexacion durable de tasks por
  parentesco.
- Tests: `go test -count=1 ./modulos/orquesta-state-file ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack`.

## T165 server-config-global-env-defaults-policy

Objetivo: separar defaults de configuracion del servidor de mutaciones al
entorno global del proceso para que smokes, tests y comandos hijos distingan
input explicito del operador frente a defaults de composicion.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-server`
- `modulos/orquesta-rails`
- `modulos/orquesta-app-codex-stack`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `serverConfigFromEnvV0`, `setDefaultStartupCleanupModeV0` y
  `ensureServerDetailRailsDefaultV0` no deben usar `os.Setenv` para fijar
  defaults de configuracion normal; deben devolver config efectiva, receipt o
  env proyectado solo en el punto que arranca un proceso hijo.
- La proyeccion a daemon debe marcar cada variable como `explicit`,
  `defaulted` o `derived`, con summary publico por categorias y sin valores
  sensibles, HOME, tokens, paths privados ni prompt/transcript.
- Tests deben poder ejecutar varias lecturas de config en el mismo proceso sin
  que una lectura previa cambie el resultado posterior salvo que el test lo
  pida explicitamente.
- Defaults de rails de detalle y cleanup de startup deben coordinarse con modo
  estricto/productivo y emitir issue publico si un default abre un rail que el
  operador esperaba cerrado.
- Coordinar con T35/T49/T55/T85/T129/T154: cleanup seguro, perfil Codex,
  exposicion de control plane, status redactado, env del daemon y deadlines son
  vecinos; esta tarea gobierna mutacion global de entorno en config.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-rails ./modulos/orquesta-app-codex-stack`.

## Escaneo backlog 2026-05-24 quincuagesima octava pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, documentos vigentes de estado,
  nucleo, principio y matriz de smokes.
- Backlog, railes observados y duplicaciones vigentes hasta `T165`.
- Busquedas `rg` sobre goroutines no-test, `server.Shutdown`,
  `context.Background`, stores file-based, validadores de carga,
  `ValidateOrchestrationRunV0`, registro MCP real y duplicados de tools.
- Lectura focal de `modulos/orquesta-server/runtime_v0.go`,
  `modulos/orquesta-server/supervisor_loop_v0.go`,
  `cmd/orquesta-server/commands.go`,
  `cmd/orquesta-server/external_bridge_loop.go`,
  `modulos/orquesta-state-file/run_store_v0.go`,
  `modulos/orquesta-state-file/event_sink_v0.go`,
  `cmd/orquesta-server/mcp_real_transport_v0.go` y
  `modulos/orquesta-mcp/mcp_transport_registry_v0.go`.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
  como refs opacas, no como rutas ni nombres Git.

Huecos detectados:

- `RuntimeV0.RunV0` lanza `server.Serve`, el loop de supervisor y preparaciones
  idle en goroutines. En shutdown usa `server.Shutdown(context.Background())`,
  marca estado `stopped` con `context.Background()` y no espera a ticks o
  preparaciones async vivas. Una salida residente puede quedar marcada como
  parada mientras aun hay auditoria, state writes, `prepare-run` o bridge
  externo en curso.
- `cmd/orquesta-server run` arranca `runOPESBridgeLoopV0` como goroutine
  paralela al runtime; sus ticks escriben solo JSON a stderr. La observabilidad
  residente no tiene owner comun para estado del bridge, ultimo tick, ultimo
  error, cancelacion, draining o join durante apagado.
- `StoreV0.LoadRunV0` valida schema/ref del documento de run, pero no aplica
  `ValidateOrchestrationRunV0` sobre la proyeccion cargada. `LoadRunEventsV0`
  valida schema/ref del documento de eventos, pero no valida cada
  `OrchestrationEventV0` cargado. Si un snapshot queda corrupto pero con
  schema/ref correctos, el cierre/replay puede operar sobre estado invalido.
- El transporte MCP real registra tools/resources en mapas por nombre/URI y
  sobrescribe duplicados silenciosamente. El contrato puro
  `RegisterMCPTransportV0` delega en el port, pero el port real no devuelve
  `duplicate_tool` o `duplicate_resource`, asi que `/mcp` puede ocultar un tool
  o resource si dos descriptores colisionan.

## T166 resident-runtime-async-shutdown-quiescence

Objetivo: asegurar que el servidor residente apaga con quiescencia verificable
de goroutines internas antes de publicar estado `stopped` o cerrar el proceso.

Estado: pendiente.

Alcance:

- `modulos/orquesta-server`
- `cmd/orquesta-server`
- `modulos/orquesta-observability`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `RuntimeV0` debe registrar y esperar, con deadline de composicion, goroutines
  internas de `Serve`, supervisor tick async e idle self-improvement antes de
  marcar `stopped`; si vence el deadline, publicar reason code durable.
- `server.Shutdown` no debe usar `context.Background()` sin deadline; debe
  recibir contexto derivado de politica de apagado y reportar error si no cierra.
- Las preparaciones idle y ticks en curso deben ver cancelacion, dejar receipt
  compacto y no seguir escribiendo estado/auditoria despues de `stopped`.
- El estado publico debe distinguir `stopping`, `stopped`, `stop_timeout` y
  `async_work_draining`, sin exponer rutas, HOME, payloads, prompts ni logs.
- Coordinar con T30/T32/T87/T94/T95/T136/T154: checkpoints, wakeups,
  readiness, audit/state failure, waits externos y deadlines siguen con owners
  propios; esta tarea gobierna quiescencia async del runtime residente.
- Tests: `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server ./modulos/orquesta-observability`.

## T167 external-bridge-resident-loop-lifecycle-state

Objetivo: hacer observable y gobernable el ciclo residente de bridges externos
sin depender solo de stderr ni de goroutines fire-and-forget.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-server`
- `modulos/orquesta-opes-bridge`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- El loop OPES/bridge externo debe registrar estado compacto por tick:
  `last_tick_ref`, `last_success_at`, `last_error_code`, filtros aplicados,
  contadores y si esta `running`, `idle`, `error`, `stopping` o `stopped`.
- `cmd/orquesta-server run` no debe lanzar el bridge sin join/cancelacion
  coordinada con el runtime; el apagado debe esperar o reportar timeout del
  bridge sin cerrar como exito silencioso.
- Los errores del tick deben ir a status/auditoria redactada ademas de stderr;
  no incluir URL completa, payload OPES, respuestas crudas, HOME, tokens ni
  rutas locales.
- La readiness debe poder distinguir servidor listo con bridge deshabilitado,
  bridge esperando initial delay, bridge degradado y bridge bloqueado por
  confirmacion/filtro.
- Coordinar con T21/T87/T98/T158/T166: catalogo de smokes, readiness, ledger de
  entrada, destino OPES y shutdown async son vecinos; esta tarea gobierna el
  estado vivo del loop de bridges residentes.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-opes-bridge`.

## T168 state-file-run-event-load-validation

Objetivo: validar proyecciones cargadas desde `orquesta-state-file` antes de
usarlas como verdad durable en replay, cierre o supervision.

Estado: pendiente.

Alcance:

- `modulos/orquesta-state-file`
- `modulos/orquesta-core-workflow`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-app-director-service`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `LoadRunV0` debe aplicar `ValidateOrchestrationRunV0` o equivalente estable
  al `OrchestrationRunV0` cargado, no solo comprobar schema/ref externa.
- `LoadRunEventsV0` debe validar cada `OrchestrationEventV0` cargado y detectar
  eventos duplicados, schema/event type invalido o refs inconsistentes antes de
  entregar historial a cierres/replay.
- Un documento invalido debe producir error publico compacto y bloquear cierre
  o replan automatico; no debe interpretarse como run vacia, sin eventos o
  completada.
- La validacion debe conservar compatibilidad con snapshots existentes validos,
  y si hay migracion/reparacion, debe quedar por puerto opt-in con evidence ref.
- Coordinar con T3/T27/T104/T148/T153/T163: replay state, source of truth,
  escritura durable, lectura acotada, payload budget y compaction son vecinos;
  esta tarea gobierna validacion semantica al cargar run/eventos.
- Tests: `go test -count=1 ./modulos/orquesta-state-file ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service`.

## T169 mcp-real-transport-registration-collision-guard

Objetivo: evitar que el transporte MCP real oculte resources/tools por
sobrescritura silenciosa al registrar nombres o URIs duplicadas.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-mcp`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `RegisterResourceV0` y `RegisterToolV0` del transporte real deben rechazar
  colisiones de `name` y `uri` con error publico `mcp_duplicate_resource` o
  `mcp_duplicate_tool`, no sobrescribir mapas.
- `RegisterMCPTransportV0` debe propagar esos errores y dejar evidencia de que
  el catalogo no quedo parcialmente publicado si hay colision.
- Tests deben cubrir duplicado por nombre, duplicado por URI, orden estable de
  listado y ausencia de mutacion parcial tras error.
- El error publico no debe incluir payload de tool, argumentos, URLs sensibles,
  HOME, rutas privadas ni detalles de handler; solo name/uri refs compactas.
- Coordinar con T23/T82/T93/T137/T146: transporte MCP, catalogo publico,
  toolbelt, limites JSON y cobertura factory HTTP son vecinos; esta tarea
  gobierna colisiones de registro del transporte MCP real.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-mcp`.

## Escaneo backlog 2026-05-24 quincuagesima novena pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, documentos vigentes de estado,
  nucleo, principio y matriz de smokes.
- Backlog, railes observados y duplicaciones vigentes hasta `T169`.
- Busquedas `rg` sobre `CheckRedirect`, `http.Client`, `URL.Query`,
  `ParseForm`, `RawQuery`, `json.NewEncoder`, `ResponseWriter` y clientes
  HTTP en `cmd`, `modulos`, `scripts` y `docs`.
- Lectura focal de `modulos/orquesta-server/audit_http_v0.go`,
  `modulos/orquesta-domain-work-http/client_v0.go`,
  `modulos/orquesta-opes-connector/http_v0.go`,
  `modulos/orquesta-web/*endpoint*_v0.go`,
  `modulos/orquesta-mcp/*_http_v0.go` y piezas HTTP de
  `cmd/orquesta-server`.
- Se conservaron `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
  y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` como refs
  opacas; no se trataron como rutas ni nombres Git.

Huecos nuevos:

- Los clientes HTTP salientes revisados declaran timeout, pero no una politica
  comun de redirects. T80/T103/T138/T154/T158 gobiernan egress, respuestas,
  deadlines y destino, pero no la cadena efectiva tras 3xx.
- `auditHTTPHandlerV0` persiste `raw_query` y varios endpoints leen
  query/form params sin limite o redaccion comun. T137 gobierna cuerpos JSON,
  no parametros de URL/form.
- Muchos handlers escriben `_ = json.NewEncoder(w).Encode(...)` o
  `w.Write(...)` sin proyectar fallo de serializacion/escritura a status,
  auditoria o observabilidad compacta. T94/T139 son vecinos, pero no son owner
  de la frontera HTTP response-write.

## T170 outbound-http-redirect-policy

Objetivo: declarar y aplicar una politica comun para redirects en clientes HTTP
salientes de composiciones, sin ampliar destino efectivo ni reenviar credenciales
por defecto.

Estado: pendiente.

Alcance:

- `modulos/orquesta-domain-work-http`
- `modulos/orquesta-opes-connector`
- `modulos/orquesta-web`
- `modulos/orquesta-mcp`
- `modulos/orquesta-app-gateway`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Todo cliente HTTP saliente con efecto, lectura de dominio o superficie publica
  debe declarar `redirect_policy`: `deny`, `same_origin`, `allowlisted` o
  `legacy_default` solo para compatibilidad documentada.
- Cada redirect debe revalidar esquema, host, puerto y metodo efectivo contra la
  misma politica de destino que el request original; no basta validar solo la URL
  inicial.
- Headers sensibles, autorizacion, idempotency keys y correlation refs no deben
  reenviarse a otro origen salvo permiso explicito de la composicion y evidencia
  auditada redactada.
- Los errores publicos deben usar reason codes compactos como
  `http_redirect_blocked` o `redirect_target_not_allowed`; no incluir URL
  completa, query, tokens, payloads, HOME ni rutas privadas.
- OPES/domain-work/MCP/web deben tener tests de redirect intra-origen permitido,
  cross-origin bloqueado, esquema no permitido, bucle/limite de saltos y
  redaccion de error.
- Coordinar con T80/T103/T138/T154/T158: esta tarea no sustituye validacion de
  destino, deadline ni response limit; gobierna la cadena 3xx posterior.
- Tests: `go test -count=1 ./modulos/orquesta-domain-work-http ./modulos/orquesta-opes-connector ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.

## T171 public-query-form-parameter-bounds-redaction

Objetivo: aplicar limites y redaccion consistentes a parametros publicos de
query/form antes de auditoria, status, validacion de dominio o construccion de
comandos.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-server`
- `modulos/orquesta-web`
- `modulos/orquesta-mcp`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-http-gateway`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- El audit/status publico no debe persistir `raw_query` completo; debe guardar
  conteos, claves permitidas, longitudes y reason codes redactados.
- `ParseForm`, `FormValue`, `URL.Query().Get` y lecturas equivalentes deben
  pasar por helpers que limiten tamano total, numero de parametros, longitud por
  valor y repeticion de claves.
- Los parametros con rutas, URLs, tokens, prompts, filtros de dominio o payloads
  libres deben clasificarse antes de log/auditoria y truncarse con marca
  `redacted`, `omitted` o `too_large`.
- Las validaciones deben aceptar alias razonables cuando el adaptador pueda
  normalizar sin riesgo, pero cortar por refs imposibles, datos sensibles,
  efectos externos no autorizados o causalidad rota.
- Tests deben cubrir query larga, clave repetida, form body grande, parametros
  sensibles, path con hijo permitido y error publico sin raw query.
- Coordinar con T85/T131/T137/T139/T153: paths, errores publicos, body JSON,
  salida publica y payload budget son vecinos; esta tarea gobierna URL/form.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway`.

## T172 http-response-encode-write-error-visibility

Objetivo: hacer visibles y testeables los fallos al serializar o escribir
respuestas HTTP publicas, sin filtrar payloads ni confundirlos con exito de la
operacion interna.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-server`
- `modulos/orquesta-web`
- `modulos/orquesta-mcp`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-observability`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Los handlers que hoy ignoran `json.NewEncoder(w).Encode(...)` o `w.Write(...)`
  deben usar helper comun o patron local que capture error de serializacion y
  fallo de escritura cuando sea observable.
- Si el header/status ya fue enviado, el sistema debe registrar un evento o
  metrica compacta `response_write_failed` en vez de intentar cambiar el status
  HTTP tarde.
- Si la serializacion falla antes de enviar headers, debe emitirse error publico
  estable sin payload interno, structs completos, prompts, HOME, tokens ni rutas
  privadas.
- La auditoria debe distinguir exito de dominio con fallo de entrega HTTP frente
  a fallo de dominio, para no cerrar smokes o supervisiones como entregados si
  el cliente no recibio respuesta.
- Tests deben cubrir encoder que falla, writer que falla antes de header, writer
  que falla despues de header y ausencia de payload sensible en observabilidad.
- Coordinar con T94/T139/T153/T166: auditoria, output publico, payload budget y
  shutdown async son vecinos; esta tarea gobierna response-write HTTP.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-observability`.

## Escaneo backlog 2026-05-24 sexagesima pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, documentos vigentes de estado,
  nucleo, principio y matriz de smokes.
- Backlog, railes observados y duplicaciones vigentes hasta `T172`.
- Busquedas `rg` sobre `Origin`, `Access-Control`, `CSRF`, `ServeMux`,
  `HandleFunc`, `r.Method`, `jsonrpc`, `RawMessage`, `Content-Type`,
  `CheckRedirect`, `json.NewEncoder` y endpoints HTTP/MCP en `cmd` y
  `modulos`.
- Lectura focal de `cmd/orquesta-server/mcp_real_transport_v0.go`,
  `modulos/orquesta-mcp/mcp_transport_registry_v0.go`,
  `modulos/orquesta-web/*endpoint*_v0.go`,
  `modulos/orquesta-app-gateway/*route*_v0.go` y handlers MCP HTTP.
- Se conservaron `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
  y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` como refs
  opacas; no se trataron como rutas ni nombres Git.

Huecos nuevos:

- Las rutas web/API mutables aceptan POST desde navegador o cliente local, pero
  no hay owner visible para una politica de `Origin`/CSRF/intent token. T55
  gobierna exposicion remota/auth del control plane y T102/T137 gobiernan JSON
  y form/query; falta separar la defensa contra POST de navegador hacia
  loopback o hacia un bind opt-in.
- El transporte MCP real implementa un subconjunto JSON-RPC sobre `/mcp`, pero
  no valida de forma estricta `jsonrpc=2.0`, trailing tokens, tipo/tamano de
  `id`, batch/notification ni presupuesto de `params` por metodo. T23/T169
  gobiernan opt-in y colisiones de registro; falta contrato de protocolo para
  que errores y compatibilidad no diverjan de los handlers HTTP/MCP puros.

## T173 browser-origin-csrf-intent-guard

Objetivo: definir una guarda comun de origen/intencion para mutaciones publicas
del control plane cuando se acceden desde navegador, web local o gateway.

Estado: pendiente.

Alcance:

- `modulos/orquesta-web`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-mcp`
- `modulos/orquesta-http-gateway`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Las rutas mutables (`prepare-run`, `self-improvement`, `runs/control`,
  `queue/priority`, `app_vcs`, `shutdown`, app-change y equivalentes web)
  deben declarar si aceptan navegador, cliente local no-browser o MCP.
- Para navegador/form, exigir `Origin`/`Referer` same-origin o token de
  intencion/CSRF emitido por la composicion; para clientes CLI/MCP, exigir
  header/intent ref equivalente cuando haya bind no-loopback o auth opt-in.
- GET/health/status siguen read-only y no deben mutar estado; cualquier ruta
  que use query/form para mutacion debe fallar con error publico estable.
- Auditoria/status deben registrar decision compacta (`origin_allowed`,
  `csrf_required`, `intent_missing`, `origin_blocked`) sin guardar URLs
  completas, cookies, tokens, query cruda, HOME, rutas privadas ni payloads.
- Coordinar con T55/T102/T137/T162: exposicion remota, JSON/form, parametros e
  idempotencia siguen con owners propios; esta tarea gobierna origen/intencion
  de mutaciones publicas.
- Tests: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-app-gateway ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./cmd/orquesta-server`.

## T174 mcp-jsonrpc-protocol-strictness

Objetivo: fijar el contrato JSON-RPC del transporte MCP real para que `/mcp`
sea opt-in y predecible sin aceptar formas ambiguas ni payloads sin presupuesto.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-mcp`
- `modulos/orquesta-app-gateway`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `serveJSONRPCV0` debe rechazar requests sin `jsonrpc:"2.0"`, con trailing
  tokens, `method` vacio, `id` de tipo no permitido o `params` que excedan el
  presupuesto del metodo.
- Declarar comportamiento para notifications y batches: soportarlos con
  semantica probada o rechazarlos con errores JSON-RPC compactos; no depender
  del error de decode accidental.
- Los metodos `resources/read` y `tools/call` deben validar params con shape
  estricto y error `mcp_invalid_params`, sin eco de argumentos, resource payload,
  rutas, URLs, prompts, transcripts ni tokens.
- `Content-Type`/`Accept` deben tener politica compatible con MCP real y los
  errores deben conservar `id` solo si es valido y seguro.
- Coordinar con T23/T102/T137/T169/T172: transporte opt-in, frontera JSON,
  parametros, colisiones de registro y escritura de respuesta son vecinos; esta
  tarea gobierna el protocolo JSON-RPC.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway`.

## Escaneo backlog 2026-05-24 sexagesima primera pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, `docs/README.md`, documentos
  vigentes de estado, nucleo, principio, cierre generico y matriz de smokes.
- Backlog, railes observados y duplicaciones vigentes hasta `T174`.
- Busquedas `rg` sobre headers HTTP, `Content-Security-Policy`,
  `Cache-Control`, `X-Frame`, `resources/read`, `tools/call`, base URLs,
  `joinEndpoint`, `url.Parse`, `strings.TrimRight`, handlers MCP/HTTP y clientes
  REST en `cmd` y `modulos`.
- Lectura focal de `cmd/orquesta-server/mcp_real_transport_v0.go`,
  `modulos/orquesta-mcp/mcp_transport_registry_v0.go`,
  `modulos/orquesta-web/*html*_v0.go`, endpoints web/MCP y clientes REST de
  web/CLI/MCP.
- Se conservaron `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
  y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` como refs
  opacas; no se trataron como rutas ni nombres Git.

Huecos nuevos:

- El transporte MCP real limita el request JSON-RPC, pero `resources/read` y
  `tools/call` convierten payloads completos a `text` y los devuelven sin
  presupuesto comun de salida, freshness, redaccion ni degradacion por recurso o
  herramienta. T174 cubre el protocolo de entrada; T25 cubre freshness de
  roadmap; falta owner de salida MCP.
- Las superficies HTML/JSON del control plane fijan `Content-Type`, `Allow` y a
  veces correlacion, pero no hay politica comun para headers de seguridad y
  cache (`CSP`/`frame-ancestors`, `X-Content-Type-Options`, `Referrer-Policy`,
  `Cache-Control`). T173 cubre origen/CSRF; falta owner de headers/cache.
- Varios clientes REST construyen endpoints con concatenacion `BaseURL +
  endpoint` o `strings.TrimRight/TrimLeft`; CLI normaliza parte de la URL, pero
  web/MCP/clientes internos no comparten politica para userinfo, query,
  fragment, base path, endpoint absoluto o path confuso. T80/T170 cubren egress
  y redirects; falta el rail de construccion de URL antes de enviar.

## T175 mcp-tool-resource-output-budget

Objetivo: limitar, redactar y clasificar la salida de `resources/read` y
`tools/call` del transporte MCP real antes de envolverla como contenido textual.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-mcp`
- `modulos/orquesta-observability`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Cada resource/tool debe declarar presupuesto de bytes, modo de salida
  (`full`, `summary`, `blocked`) y freshness esperada; el transporte debe cortar
  o resumir antes de serializar la respuesta JSON-RPC.
- El error por exceso debe ser compacto (`mcp_output_too_large`,
  `mcp_resource_payload_blocked` o equivalente), con `id` seguro y sin eco del
  payload, argumentos, rutas, URLs, prompts, transcripts, HOME ni tokens.
- Recursos de roadmap, contratos, operational-status y herramientas de control
  deben compartir reason codes y counters; no cada handler con su propio limite
  invisible.
- El modo opt-in de diagnostico crudo, si existe, debe exigir flag de
  composicion, scope acotado y redaccion previa; no debe ser default del
  transporte residente.
- Coordinar con T23/T25/T93/T137/T139/T174: transporte opt-in, freshness de
  roadmap, toolbelt, limites JSON, salida publica y protocolo JSON-RPC son
  vecinos; esta tarea gobierna payload de respuesta MCP.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-observability`.

## T176 control-plane-http-security-cache-headers

Objetivo: definir headers de seguridad y cache para respuestas web/API/MCP del
control plane sin mezclarlo con autenticacion ni con la politica CSRF.

Estado: pendiente.

Alcance:

- `modulos/orquesta-web`
- `modulos/orquesta-mcp`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-http-gateway`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Las respuestas HTML deben declarar `Content-Security-Policy` o politica
  equivalente compatible con los estilos actuales, `frame-ancestors`/anti-frame,
  `Referrer-Policy`, `X-Content-Type-Options` y `Cache-Control` apropiado para
  control plane.
- JSON/API/MCP deben usar `nosniff` y cache policy explicita; status/read-only
  puede permitir cache corta solo si freshness queda declarada y no contiene
  datos operativos sensibles.
- La politica debe evitar copiar cookies, tokens, URLs completas, query cruda,
  payloads, prompts, transcripts, HOME o rutas privadas a headers, errores o
  auditoria.
- Si una ruta necesita headers distintos por compatibilidad, debe declararlo con
  test y reason code; no dejar defaults de `net/http` como decision implicita.
- Coordinar con T55/T99/T137/T171/T172/T173: exposicion remota, recursos HTTP,
  JSON, query/form, fallos de escritura y CSRF siguen owners separados.
- Tests: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway ./cmd/orquesta-server`.

## T177 rest-client-base-url-endpoint-policy

Objetivo: unificar normalizacion de base URL y union de endpoints en clientes
REST de web, CLI, MCP y composicion para evitar destinos ambiguos antes de
aplicar egress, redirects o lectura de respuesta.

Estado: pendiente.

Alcance:

- `modulos/orquesta-web`
- `modulos/orquesta-cli`
- `modulos/orquesta-mcp`
- `modulos/orquesta-app-gateway`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Los clientes deben parsear base URL con helper compartido o politica
  equivalente: esquema permitido, host requerido, rechazo de userinfo, query y
  fragment, y manejo explicito de base path.
- Los endpoints deben ser paths relativos controlados; rechazar endpoint con
  esquema/host, `..`, path vacio ambiguo o query no declarada.
- El resultado de join debe conservar correlacion y endpoint publico, pero no
  imprimir URL completa con credenciales, HOME, rutas privadas, query cruda ni
  tokens en errores, auditoria o ACKs.
- `orquesta-app-gateway` puede mantener `http://orquesta.internal` para
  transporte in-process, pero debe quedar marcado como destino interno
  no-egress y cubierto por tests de no red real.
- Coordinar con T80/T103/T170/T171/T162: egress, respuesta HTTP, redirects,
  query/form e idempotencia son vecinos; esta tarea gobierna URL final antes de
  enviar.
- Tests: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-cli ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 sexagesima segunda pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, `docs/README.md`, documentos
  vigentes de estado, nucleo, principio y matriz de smokes.
- Backlog, railes observados y duplicaciones vigentes hasta `T177`.
- Busquedas `rg` sobre `http.Client`, `DefaultTransport`, `ProxyFromEnvironment`,
  `CheckRedirect`, `context.Background`, `r.Context`, `InProcessTransport`,
  `httptest.NewRecorder`, handlers MCP/HTTP y clientes REST.
- Lectura focal de `modulos/orquesta-app-gateway/inprocess_transport_v0.go`,
  `modulos/orquesta-app-gateway/clients_v0.go`,
  `cmd/orquesta-server/mcp_real_smoke_v0.go`, clientes REST web/CLI/MCP y
  conectores HTTP OPES/domain-work.
- Se conservaron `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
  y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` como refs
  opacas; no se trataron como rutas ni nombres Git.

Huecos nuevos:

- Muchos clientes salientes solo declaran `http.Client{Timeout: ...}`. Con el
  transporte por defecto de Go, proxy, TLS, keepalive y redirects quedan como
  decisiones implicitas del proceso o del entorno. T170 gobierna redirects y
  T177 gobierna la URL final, pero falta owner para politica de transporte y
  proxy por perfil.
- El transporte in-process del gateway y el roundtripper del smoke MCP usan
  recorders en memoria y llaman al handler directamente. Es correcto como
  adaptador local, pero no hay contrato comun para limite de respuesta,
  cancelacion/timeout observable, panic recovery o paridad de errores frente a
  HTTP real. T137/T172 cubren rutas publicas, pero falta el rail del transporte
  in-process.

## T178 outbound-http-client-transport-proxy-policy

Objetivo: declarar una politica comun de transporte HTTP saliente para clientes
de control plane, dominio y composicion antes de aplicar egress, redirect,
respuesta o idempotencia.

Estado: pendiente.

Alcance:

- `modulos/orquesta-domain-work-http`
- `modulos/orquesta-opes-connector`
- `modulos/orquesta-web`
- `modulos/orquesta-cli`
- `modulos/orquesta-mcp`
- `modulos/orquesta-app-gateway`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Cada cliente debe declarar perfil de transporte: `internal_inprocess`,
  `loopback_control_plane`, `domain_egress`, `opes_temporal`,
  `operator_mcp` o `legacy_default` justificado.
- El perfil debe fijar proxy policy (`deny`, `env_allowlisted`, `explicit`),
  timeouts de dial/TLS/headers cuando haya red real, limites de conexiones y
  keepalive; los perfiles internos/loopback no deben heredar proxy del entorno
  por accidente.
- Si un proxy o TLS custom es opt-in, debe validarse y auditarse por categoria
  compacta, sin URL completa, userinfo, certificados crudos, HOME, rutas
  privadas, tokens ni payloads.
- El uso de `http.DefaultTransport` o transporte nil en mutaciones de control
  plane debe quedar como modo legacy con test o sustituirse por factory local
  compartida; no duplicar una politica por cliente.
- Coordinar con T80/T103/T154/T170/T177: destino, respuesta, deadline,
  redirects y URL final conservan owners propios; esta tarea gobierna transporte
  y proxy antes de enviar.
- Tests: `go test -count=1 ./modulos/orquesta-domain-work-http ./modulos/orquesta-opes-connector ./modulos/orquesta-web ./modulos/orquesta-cli ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.

## T179 inprocess-http-transport-budget-parity

Objetivo: hacer que transportes HTTP in-process usados por gateway/smokes tengan
presupuestos, cancelacion y errores publicos equivalentes a la frontera HTTP
real, sin abrir red ni meter HTTP en el nucleo.

Estado: pendiente.

Alcance:

- `modulos/orquesta-app-gateway`
- `modulos/orquesta-web`
- `modulos/orquesta-mcp`
- `cmd/orquesta-server`
- `modulos/orquesta-observability`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `InProcessTransportV0` y roundtrippers equivalentes deben tener limite de
  respuesta, reason code por exceso y tests que demuestren que un handler no
  puede llenar memoria con un body sin presupuesto.
- El transporte debe preservar `request.Context()` y publicar `inprocess_timeout`
  o `inprocess_cancelled` si vence deadline/cancelacion; si un handler ignora el
  contexto, el caller debe recibir error observable en vez de bloqueo indefinido.
- Panics o fallos de escritura del handler deben mapearse a error publico
  compacto en frontera de transporte/smoke, sin stack, payload, prompts,
  transcripts, HOME, rutas privadas ni tokens.
- La paridad debe cubrir metodo, headers relevantes, `Content-Type`,
  correlacion, status y body para que tests in-process no oculten fallos que
  aparecerian en HTTP real.
- Coordinar con T102/T137/T141/T172/T177/T178: JSON publico, body bounds,
  no-panic, write errors, URL interna y transporte/proxy son vecinos; esta tarea
  gobierna el adaptador in-process.
- Tests: `go test -count=1 ./modulos/orquesta-app-gateway ./modulos/orquesta-web ./modulos/orquesta-mcp ./cmd/orquesta-server ./modulos/orquesta-observability`.

## Escaneo backlog 2026-05-24 sexagesima tercera pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, `docs/README.md`, documentos
  vigentes de estado, nucleo, principio y matriz de smokes.
- Backlog, railes observados y duplicaciones vigentes hasta `T179`.
- Busquedas `rg` sobre `json.NewEncoder(stdout).Encode`, `fmt.Fprintf`,
  `stdout`, `stderr`, handlers HTTP, `command-output-public-shape`, `T139` y
  `T172`.
- Lectura focal de comandos en `cmd/orquesta-server`, `modulos/orquesta-cli` y
  las tareas vecinas de salida publica, respuesta HTTP y observabilidad.
- Se conservaron `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
  y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` como refs
  opacas; no se trataron como rutas ni nombres Git.

Hueco nuevo:

- T139 gobierna el shape publico de comandos y T172 gobierna errores de escritura
  HTTP, pero los comandos locales siguen ignorando fallos al escribir stdout o
  stderr (`_ = json.NewEncoder(stdout).Encode(...)`, `_, _ = fmt.Fprintf(...)`).
  Un pipe roto, stdout cerrado o writer de test fallando puede dejar una
  operacion interna correcta con salida no entregada y exit code ambiguo.

## T180 command-stdio-write-error-visibility

Objetivo: hacer visibles los fallos de escritura en stdout/stderr de comandos
locales sin convertir stdout en evidencia terminal ni filtrar diagnostico crudo.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-cli`
- `modulos/orquesta-observability`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Los comandos que emiten JSON o texto publico (`status`, `opes-drain-once`,
  `mcp-real-smoke`, `codex-wave-*`, `run-status` y runners CLI equivalentes)
  deben comprobar errores de escritura y devolver exit code o error publico
  estable (`command_output_write_failed`, `command_error_write_failed` o
  equivalente).
- Si stdout falla despues de ejecutar un efecto, registrar decision compacta en
  auditoria/status cuando exista puerto disponible; no reintentar el efecto ni
  declarar cierre por la salida textual.
- Stderr no debe ser unica evidencia del fallo: si stderr tambien falla,
  conservar solo reason code y exit code, sin bucles de escritura ni panics.
- Mantener T139 como owner del shape publico y T172 como owner de HTTP; esta
  tarea gobierna solo stdio de comandos locales. No escribir rutas privadas,
  HOME, prompts, transcripts, stdout/stderr crudos, tokens ni payloads de
  proveedor como diagnostico durable.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-cli ./modulos/orquesta-observability`.

## Escaneo backlog 2026-05-24 sexagesima cuarta pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, `docs/README.md`, documentos
  vigentes de estado, nucleo, principio, Director Operativo, cierre generico y
  matriz de smokes.
- Backlog, railes observados y duplicaciones vigentes hasta `T180`.
- Busquedas `rg` sobre `document_plan`, `DomainDocumentPlan`, `jobRef`,
  `IdempotencyKey`, `template.Execute`, `Execute(w`, `Invoke`, `Resource` y
  handlers MCP reales.
- Lectura focal de `modulos/orquesta-domain-work`,
  `modulos/orquesta-document-plan-expander`, `modulos/orquesta-opes-bridge`,
  `modulos/orquesta-web` y `cmd/orquesta-server/mcp_real_transport_v0.go`.
- Se conservaron `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
  y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` como refs
  opacas; no se trataron como rutas ni nombres Git.

Huecos nuevos:

- El plan documental neutral valida presencia y forma compacta, pero no hay rail
  visible para duplicados de `section_ref`, `visual_ref`, `review_ref` o
  `deliverable_ref`. El expander construye `RequestID`/`IdempotencyKey` desde
  esos refs; refs duplicados pueden producir jobs duplicados antes de que un
  adaptador de dominio tenga una causa publica clara.
- Las piezas raw de `DomainDocumentPlanV0` colapsan arrays malformados a `nil`
  durante canonicalizacion. El resultado posterior puede verse como campo
  ausente o plan invalido generico, no como diagnostico de forma reparable.
- T174 gobierna protocolo MCP y T175 salida, pero el transporte real invoca
  tools/resources sin un presupuesto de ejecucion propio por metodo ni reason
  code estable para deadline/cancelacion del handler.
- T172 cubre fallos de escritura HTTP en general, pero los renderers HTML
  concretos ignoran errores de `template.Execute`. Falta distinguir fallo de
  render, fallo tardio de socket y fallback localizado sin filtrar payloads.

## T181 domain-document-plan-ref-uniqueness-and-diagnostics

Objetivo: cerrar la frontera de diagnostico del plan documental neutral antes
de expandir jobs, sin meter OPES, Codex, HTTP ni runtime en el nucleo de
domain-work.

Estado: pendiente.

Alcance:

- `modulos/orquesta-domain-work`
- `modulos/orquesta-document-plan-expander`
- `modulos/orquesta-opes-bridge`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `DomainDocumentPlanV0` debe rechazar refs duplicados por tipo
  (`section_ref`, `visual_ref`, `review_ref`, `deliverable_ref`) con issue
  publico estable y campo causal antes de crear trabajos derivados.
- La canonicalizacion de arrays raw malformados debe conservar diagnostico
  especifico por campo; no debe convertir `sections`, `visuals`, `review_steps`
  o `deliverables` invalidos en ausencia silenciosa.
- El expander no debe emitir dos `DomainWorkJobRequestV0` con el mismo
  `RequestID` o `IdempotencyKey`; un duplicado debe ser causa terminal
  reparable del plan, no decision implicita del adaptador OPES.
- La reparacion por alias sigue permitida: si un ref falta y se deriva de titulo
  u objetivo, el ref derivado debe ser unico, determinista y trazable.
- No guardar markdown completo, payloads, prompts, transcripts, HOME, rutas
  privadas ni tokens en issues, auditoria o ACKs.
- Tests: `go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-document-plan-expander ./modulos/orquesta-opes-bridge`.

## T182 mcp-tool-execution-budget-and-cancellation

Objetivo: dar presupuesto temporal observable a cada tool/resource MCP real,
separado del limite de parsing JSON-RPC y del limite de salida.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-mcp`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-observability`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- El registro o transporte MCP debe asignar presupuesto de ejecucion por perfil
  de metodo antes de invocar handlers, derivando `context.Context` con deadline
  y causa publica.
- Cancelacion del request debe mapearse a `mcp_tool_cancelled` o equivalente;
  deadline agotado debe mapearse a `mcp_tool_timeout`, sin serializar args,
  payloads, resource contents, prompts, transcripts, HOME, rutas privadas ni
  tokens.
- Recursos read-only, mutaciones de control plane y llamadas largas de
  autoprogramacion deben tener perfiles distintos; trabajo largo debe devolver
  ref/accepted o encolar, no bloquear indefinidamente `/mcp`.
- Observabilidad debe registrar solo metodo, perfil, reason code, bucket de
  duracion y correlacion compacta.
- Coordinar con T154/T174/T175/T179: esta tarea gobierna tiempo de ejecucion,
  no shape de request, tamano de salida, transporte in-process ni deadlines de
  puertos internos.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-observability`.

## T183 web-html-render-error-contract

Objetivo: hacer explicitos los errores de render HTML y escritura tardia en
vistas web sin duplicar efectos ni convertir detalles privados en diagnostico.

Estado: pendiente.

Avance parcial 2026-05-26: `modulos/orquesta-web` cubre el slice local de
renderers HTML con helper comun que bufferiza templates antes de escribir,
fallback publico `web_html_render_failed` y reason compacto
`web_response_write_failed` para fallo observable de escritura. T183 sigue
pendiente para auditoria/contadores en `orquesta-app-gateway`,
`orquesta-observability` y `cmd/orquesta-server`.

Alcance:

- `modulos/orquesta-web`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-observability`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Renderers HTML deben comprobar errores de `template.Execute` y escritura,
  devolviendo reason code publico (`web_html_render_failed`,
  `web_response_write_failed` o equivalente) cuando el fallo sea observable.
- Un fallo de template/render debe distinguirse de error de dominio, error de
  cliente y fallo tardio de socket. Ningun caso debe reejecutar efectos externos
  ni marcar cierre operativo por texto parcialmente escrito.
- Fallback/error page debe preservar locale cuando sea posible; si el fallback
  tambien falla, observabilidad registra solo contador/reason compacto.
- No guardar formularios, query cruda, payloads, prompts, transcripts, HOME,
  rutas privadas, cookies ni tokens en errores publicos o auditoria.
- Coordinar con T172/T176/T173/T75: T183 gobierna render HTML; no sustituye
  headers/cache, origen/intencion, i18n ni escritura HTTP generica.
- Tests: `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-app-gateway ./modulos/orquesta-observability ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 sexagesima quinta pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, `docs/README.md` y los
  documentos vigentes del nucleo.
- Backlog, railes observados y duplicaciones vigentes hasta T183.
- Busquedas focales con `rg` sobre `hash/fnv`, `sha1.Sum`, `strings.Join`,
  refs deterministas, cleanup de smokes y variables `ORQUESTA_SMOKE_ROOT`.
- Lectura focal de derivacion de refs en app-change, MCP, orchestration-core,
  runtime Codex delivery, app Codex stack y scripts de smoke.
- Se conservaron `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
  y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` como refs
  opacas, sin convertirlas en rutas ni nombres Git.

Huecos nuevos:

- Varias refs o firmas deterministas se derivan con FNV32, SHA1 truncado o
  concatenacion con separador textual. Algunas son advisory, pero otras quedan
  cerca de causalidad, idempotencia, requests o recuperacion. Falta owner que
  separe huella visual de identidad causal y obligue canonicalizacion no
  ambigua.
- Varios smokes aceptan raices temporales por entorno y ejecutan `rm -rf` sobre
  la raiz al limpiar. Falta rail comun que pruebe que la raiz fue creada por el
  smoke o esta bajo prefijo temporal permitido antes de borrar recursivamente.

## T184 deterministic-ref-hash-collision-proof

Objetivo: endurecer refs y firmas deterministas para que identidad causal,
idempotencia y recuperacion no dependan de hashes cortos, delimitadores
ambiguos ni colisiones silenciosas.

Estado: pendiente.

Alcance:

- `modulos/orquesta-app-change-director-source`
- `modulos/orquesta-mcp`
- `modulos/orquesta-orchestration-core`
- `modulos/orquesta-runtime-codex-delivery`
- `modulos/orquesta-app-codex-stack`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Toda ref usada como identidad causal, idempotencia, recuperacion o evidencia
  durable debe derivarse desde una codificacion canonica no ambigua
  (JSON canonico, campos con longitud o builder tipado equivalente) y digest
  con margen suficiente. FNV32 o digest truncado quedan permitidos solo para
  display/advisory si no deciden causalidad.
- Si una ref determinista ya existe con payload/fuente distinta, el sistema debe
  publicar conflicto reparable con reason code estable; no debe aliasar ni
  sobrescribir de forma silenciosa.
- Las firmas de acciones progress/rework deben distinguir campos vacios,
  separadores embebidos y orden cuando el contrato lo exija; si el orden es
  irrelevante, debe normalizarse de forma explicita antes de hashear.
- Observabilidad y errores deben registrar solo modulo, tipo de ref, reason
  code y digest compacto; no guardar payloads, prompts, transcripts, HOME,
  rutas privadas, tokens ni entradas completas usadas para la huella.
- Coordinar con T161/T162/T169/T181: esta tarea no sustituye reloj/ref generator,
  unicidad de tasks ni validacion documental; gobierna derivacion y colision de
  huellas/ref deterministas.
- Tests: `go test -count=1 ./modulos/orquesta-app-change-director-source ./modulos/orquesta-mcp ./modulos/orquesta-orchestration-core ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack`.

## T185 smoke-script-temp-root-deletion-guard

Objetivo: hacer segura la limpieza recursiva de directorios temporales de
smokes y runners, especialmente cuando la raiz entra por variable de entorno.

Estado: pendiente.

Alcance:

- `scripts`
- `scripts/lib`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Todo `rm -rf` de raiz temporal controlada por `ORQUESTA_SMOKE_ROOT`,
  `ORQUESTA_PARALLEL_TEST_TMP` o variable equivalente debe pasar por helper
  comun que valide prefijo permitido, marcador/manifest de creacion y que la
  ruta no sea proyecto, HOME, `/`, `.orquesta-runtime` ni ruta vacia.
- Si la raiz fue inyectada por entorno y no tiene marcador valido, el smoke debe
  fallar cerrado o conservar la raiz con reason code publico; no debe intentar
  reparar borrando un padre.
- `ORQUESTA_KEEP_SMOKE_DIR` y equivalentes deben evitar borrado y publicar la
  ruta solo como ref/diagnostico compacto cuando aplique, sin HOME real ni rutas
  privadas en ACKs o logs de largo plazo.
- Los smokes que creen subdirectorios deben borrar solo hijos registrados por
  manifest. La limpieza parcial debe quedar observable sin ocultar el resultado
  principal del smoke.
- Coordinar con T59/T91/T121/T126/T127: esta tarea gobierna seguridad de
  limpieza local, no shutdown, aislamiento semantico del smoke ni el contrato
  funcional de cada prueba real.
- Tests: `bash -n scripts/*.sh scripts/lib/*.sh` y prueba focal del helper de
  cleanup con raiz valida, raiz sin marcador y raiz prohibida.

## Escaneo backlog 2026-05-24 sexagesima sexta pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, documentos vigentes de estado,
  nucleo, principio, cierre generico y matriz de smokes.
- Backlog, railes observados y duplicaciones vigentes hasta `T185`.
- Busquedas focales con `rg` sobre `fnv`, `hash`, `remote_addr`,
  `RawQuery`, `StatusMethodNotAllowed`, `Allow`, `r.Method`,
  `json.NewDecoder`, `context.Background` y fronteras HTTP/bridge.
- Lectura focal de `modulos/orquesta-domain-work-memory`,
  `modulos/orquesta-domain-work-file`, `modulos/orquesta-domain-work-sql`,
  `modulos/orquesta-server/audit_http_v0.go`, `modulos/orquesta-mcp`,
  `modulos/orquesta-web` y `cmd/orquesta-server/mcp_real_transport_v0.go`.
- Se preservaron `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
  y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` como refs
  opacas; no se trataron como rutas ni nombres Git.

Huecos nuevos:

- `domain_work` memory/file/sql repiten huella FNV64 base36 para derivar refs
  de job desde fingerprints de request. T184 detecta hashes cortos en otros
  paquetes, pero su alcance no incluye estos adaptadores de dominio, que son
  frontera de idempotencia/replay.
- La auditoria HTTP del servidor persiste `remote_addr` completo junto a metodo,
  path y query. T171 cubre query/form y T132 privacidad general, pero falta
  owner concreto de identidad de cliente/proxy para no grabar IP:puerto o
  headers de forwarding como evidencia durable sin politica.
- Muchos handlers publicos retornan 405 por metodo, pero el contrato de
  `Allow`, `OPTIONS` y shape de error no esta centralizado. Algunas pruebas lo
  esperan en rutas concretas y otras rutas HTML/JSON/MCP responden con patrones
  locales.

## T186 domain-work-job-ref-fingerprint-collision-proof

Objetivo: hacer que las refs deterministas de jobs `domain_work` sean
canonicas, resistentes a colision y equivalentes entre memory, file y SQL sin
convertir SQL en persistencia global.

Estado: pendiente.

Alcance:

- `modulos/orquesta-domain-work`
- `modulos/orquesta-domain-work-memory`
- `modulos/orquesta-domain-work-file`
- `modulos/orquesta-domain-work-sql`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Extraer o documentar un builder canonico de fingerprint/job ref compartido
  para memory, file y SQL; FNV64/base36 queda permitido solo como compatibilidad
  legacy si no decide identidad nueva.
- Si dos requests distintas producen la misma ref determinista, el store debe
  detectar conflicto reparable y no sobrescribir ni aliasar silenciosamente.
- La canonicalizacion debe distinguir campos vacios, arrays `nil` vs vacios
  cuando el contrato lo requiera, orden significativo y version de schema.
- La migracion debe preservar replay de jobs existentes o declarar modo legacy
  con test de compatibilidad; no cambiar refs durables sin plan de lectura
  antigua.
- Coordinar con T20/T72/T101/T181/T184: esta tarea gobierna identidad de jobs
  `domain_work`, no dialectos SQL reales, mapa de artefactos, ledger de submit,
  refs documentales ni todos los hashes del repo.
- Tests: `go test -count=1 ./modulos/orquesta-domain-work ./modulos/orquesta-domain-work-memory ./modulos/orquesta-domain-work-file ./modulos/orquesta-domain-work-sql`.

## T187 http-audit-client-identity-redaction-policy

Objetivo: definir que identidad de cliente puede persistir la auditoria HTTP
del servidor y como se redacta cuando hay loopback, proxy o headers de
forwarding.

Estado: pendiente.

Alcance:

- `modulos/orquesta-server`
- `modulos/orquesta-observability`
- `cmd/orquesta-server`
- `docs/auditoria_runtime_orquesta_2026-05-24.md`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `auditHTTPHandlerV0` no debe guardar `RemoteAddr` crudo como payload durable
  por defecto; debe guardar categoria (`loopback`, `private`, `external`,
  `unknown`) y, si hace falta, hash/redaccion con sal y retencion local.
- Si se aceptan `X-Forwarded-*` o headers equivalentes, deben estar habilitados
  por perfil/proxy confiable; por defecto se ignoran o se resumen sin confiar en
  ellos como identidad real.
- Status, auditoria y errores publicos deben conservar correlacion por ref, no
  IP completa, puerto efimero, host privado, cabeceras crudas, cookies ni
  tokens.
- La politica debe coordinarse con bind/loopback y control plane: una ruta
  local no debe volverse "remota segura" por un header enviado por el cliente.
- Coordinar con T55/T132/T171/T176: esta tarea gobierna identidad de cliente en
  auditoria, no bind/autorizacion, taxonomia general, query/form ni headers de
  cache/seguridad.
- Tests: `go test -count=1 ./modulos/orquesta-server ./modulos/orquesta-observability ./cmd/orquesta-server`.

## T188 http-method-allow-options-contract

Objetivo: unificar el contrato de metodo HTTP para rutas publicas HTML, JSON y
MCP real: 405 coherente, `Allow` correcto y comportamiento `OPTIONS` explicito.

Estado: pendiente.

Alcance:

- `modulos/orquesta-mcp`
- `modulos/orquesta-web`
- `modulos/orquesta-governance`
- `modulos/orquesta-factory-http`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Todo handler publico que rechace metodo debe setear `Allow` con los metodos
  permitidos y devolver shape de error compatible con su perfil HTML/JSON/MCP.
- `OPTIONS` debe estar definido por perfil: responder metadata sin ejecutar
  efectos o devolver 405/204 documentado; no debe invocar tools, crear runs ni
  drenar bridges.
- Las pruebas deben cubrir al menos MCP/autoprogramacion, web HTML, governance,
  factory HTTP y transporte MCP real con metodo incorrecto y `OPTIONS`.
- La respuesta de metodo no debe incluir body crudo, query/form, payloads,
  prompts, transcripts, HOME, rutas privadas, cookies ni tokens.
- Coordinar con T102/T137/T173/T174/T176: esta tarea gobierna metodo/Allow/
  OPTIONS; no sustituye parsing JSON, CSRF/origen, JSON-RPC estricto ni headers
  de cache/seguridad.
- Tests: `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-web ./modulos/orquesta-governance ./modulos/orquesta-factory-http ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 sexagesima septima pasada

Evidencia revisada: paquete OrquestaV2 del scanner, `AGENTS.md`, `README.md`,
docs vigentes de estado/nucleo/principio/matriz, backlog actual hasta `T188`,
rail errors, duplicaciones de rails y busquedas focales sobre `signal`,
`ServeMux`, rutas HTTP, `ProcessRuntimeConnectorV0`, `os.Interrupt`, `Kill`,
`Allow`, `X-Correlation-ID`, `panic`, `http.Client` y handlers publicos. No se
programa codigo desde este scanner. `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement`
y `branch_ref=branch-ref-orquesta-server-idle-self-improvement` se conservan
solo como refs opacas.

Huecos nuevos:

- `cmd/orquesta-server run` solo escucha `os.Interrupt`, mientras `stop` tambien
  envia `os.Interrupt` al proceso daemon. T96 cubre identidad de proceso y T166
  quiescencia async, pero falta contrato de senales del servidor residente:
  SIGTERM de service managers, segunda senal, gracia/escalado y observabilidad.
- El gateway HTTP registra rutas exactas y prefijos (`/api/v0/apps/`) en
  `ServeMux`, y AppVCS se superpone desde un mux externo bajo `/api/v0/apps/vcs`.
  T169 protege colisiones MCP y T188 metodo/OPTIONS, pero falta manifiesto que
  pruebe shadowing/colision de rutas HTTP antes de publicar nuevas superficies.
- `ProcessRuntimeConnectorV0` envia `os.Interrupt` y solo llama `Kill` si
  `Signal` falla. Si el proceso ignora la senal, no hay deadline de gracia,
  escalation, reason code ni snapshot `stopping`/`kill_timeout`; T66 prueba stop
  neutral, pero no la politica de senal y escalado del conector.

## T189 server-resident-signal-shutdown-policy

Objetivo: definir y probar la politica de senales del servidor residente para
arranque foreground/daemon, shutdown cooperativo y escalado controlado.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-server`
- `modulos/orquesta-server-shutdown`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `run` debe declarar que senales atiende en cada plataforma: al menos
  interrupcion local y terminacion de servicio cuando el SO la soporte, sin
  depender de una unica senal interactiva.
- La primera senal debe iniciar shutdown cooperativo con deadline; una segunda
  senal o timeout debe producir escalado documentado con reason code compacto.
- `stop` y el daemon deben compartir el mismo vocabulario de causa
  (`signal_interrupt`, `signal_terminate`, `signal_escalated`, etc.) sin
  imprimir PID crudo como verdad, rutas locales, HOME, env, logs, prompts ni
  tokens.
- Status/auditoria deben distinguir `stopping_by_signal`, `stop_timeout` y
  `stopped` para que smokes y CLI no interpreten salida parcial como cierre
  limpio.
- Coordinar con T30/T96/T166: esta tarea gobierna senales del servidor
  residente; no reabre checkpoint de agentes, identidad de PID ni quiescencia de
  goroutines internas.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-server-shutdown`.

## T190 http-gateway-route-manifest-collision-guard

Objetivo: hacer explicito y testeable el manifiesto de rutas HTTP publicas para
evitar shadowing, colisiones exactas o prefijos ambiguos al crecer el gateway.

Estado: pendiente.

Alcance:

- `modulos/orquesta-http-gateway`
- `modulos/orquesta-app-gateway`
- `modulos/orquesta-mcp`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Declarar un inventario canonico de rutas exactas y prefijos (`/api/v0/apps/`,
  `/api/v0/apps/director`, `/api/v0/apps/vcs`, `/mcp`, web HTML y control
  plane) con owner, metodo esperado y perfil de seguridad.
- El builder del mux o un test de arquitectura debe detectar rutas exactas
  duplicadas, prefijos que capturan endpoints no previstos y overlays externos
  que cambian precedencia sin declararlo.
- AppVCS y futuras rutas bajo prefijos existentes deben tener test de dispatch
  que demuestre que llegan al handler esperado y no al catch-all de app-change.
- La evidencia publica debe contener refs de ruta compactas y owner, no URL con
  query, headers, cookies, payloads, HOME, rutas privadas, prompts ni tokens.
- Coordinar con T82/T119/T169/T173/T188: esta tarea gobierna registro y
  precedencia de rutas HTTP; no sustituye catalogo de funciones, docs de uso,
  colisiones MCP, origen/CSRF ni metodo/Allow.
- Tests: `go test -count=1 ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-mcp ./cmd/orquesta-server`.

## T191 process-runtime-stop-signal-escalation-policy

Objetivo: definir parada cooperativa y escalado del runtime de procesos cuando
un agente/proceso local no atiende la senal inicial.

Estado: pendiente.

Alcance:

- `modulos/orquesta-runtime`
- `modulos/orquesta-runtime-required-test`
- `modulos/orquesta-orchestration-core`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `ProcessRuntimeConnectorV0` debe exponer estado `stopping` o equivalente,
  deadline de gracia y causa de escalado antes de matar un proceso.
- Si la senal cooperativa se acepta pero el proceso no termina, el conector debe
  escalar por timeout y registrar snapshot/ACK compacto; no basta con hacer
  `Kill` solo cuando `Signal` devuelve error.
- La politica debe diferenciar senal no soportada, proceso ya detenido, timeout
  de gracia y kill fallido con codigos publicos estables.
- Tests deben cubrir proceso que sale con interrupcion, proceso que ignora la
  senal y proceso ya detenido, sin persistir command path, argv completo, env,
  HOME, stdout/stderr crudos, prompts ni tokens.
- Coordinar con T30/T60/T66/T105/T166: esta tarea gobierna escalado del conector
  de procesos; no reabre checkpoint Codex, stop de olas, E2E neutral, launch
  env/io ni shutdown async del servidor.
- Tests: `go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-runtime-required-test ./modulos/orquesta-orchestration-core ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 sexagesima octava pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, `docs/README.md` y documentos
  vigentes de nucleo/director/cierre.
- Backlog, railes observados y duplicaciones vigentes hasta `T191`.
- Busquedas focales con `rg` sobre estado publico del servidor, eventos de
  auditoria, reloj de AppSpec y catalogo de gobernanza.
- Lectura focal de `modulos/orquesta-server/status_tracker_v0.go`,
  `modulos/orquesta-server/supervisor_loop_v0.go`,
  `modulos/orquesta-factory/appspec_usecase_v0.go`,
  `modulos/orquesta-factory-http/appspec_http_v0.go`,
  `modulos/orquesta-governance/governance_catalog_v0.go` y
  `modulos/orquesta-governance/governance_catalog_public_query_v0.go`.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` preservadas
  como refs opacas.

Huecos nuevos:

- El estado publico del servidor puede propagar texto libre de automejora idle,
  supervisor y arranque en campos de status. `T41` cubre auditoria sensible y
  `T85` cubre configuracion, pero falta un propietario para la proyeccion
  estructurada de mensajes operativos.
- `SolicitarNuevaAppV0` recibe reloj por puerto, pero si llega `now` cero usa
  `time.Now().UTC()` dentro del caso de uso. `T161` cubre reloj/ref global, no
  el contrato especifico de AppSpec/factory.
- La consulta publica del catalogo de gobernanza expone entradas efectivas sin
  presupuesto de salida ni metadatos de frescura/source refs. `T82` cubre forma
  HTTP y `T175` MCP, pero no esta proyeccion de gobernanza.

## T192 server-status-operational-message-projection

Objetivo: introducir una proyeccion estable y acotada para mensajes operativos
del servidor antes de persistirlos o exponerlos en `StateV0`, status o
auditoria.

Estado: parcial local 2026-05-26 para `StateV0` del runtime residente:
`StatusTrackerV0` ya proyecta mensajes de startup, supervisor, automejora idle
y errores recientes con helper canonico, truncado y redaccion de paths/secrets.
Sigue pendiente extender la misma politica a auditoria general y a otros
adaptadores que formen mensajes publicos fuera del tracker.

Alcance:

- `modulos/orquesta-server`
- `cmd/orquesta-server`
- `modulos/orquesta-observability`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `IdleSelfImprovementReason`, errores de supervisor, mensajes de arranque y
  causas equivalentes deben transportar codigo de razon, refs opacas, contadores
  y texto truncado/redactado, no strings crudos de adaptadores, agentes o
  herramientas.
- Los campos publicos de status usan una estructura o helper canonico con limite
  de bytes, redaccion de paths locales/secrets y refs opacas separadas.
- Los eventos de auditoria conservan causalidad sin volcar payloads completos de
  planes, requests o resultados cuando solo se necesita resumen.
- La compatibilidad JSON se conserva o se documenta una migracion versionada.
- Tests inyectan mensajes largos, paths locales, tokens aparentes y evidence
  refs, y verifican salida acotada.
- Coordinar con T41/T85/T95: esta tarea cubre proyeccion de mensajes
  operativos en status; no sustituye payload general de auditoria, configuracion
  ni fallos de persistencia.
- Tests: `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server ./modulos/orquesta-observability`.

## T193 factory-appspec-time-source-contract

Objetivo: cerrar el contrato de reloj de AppSpec/factory para que la fecha de
recepcion no dependa de un fallback silencioso a `time.Now()` dentro del caso de
uso.

Estado: pendiente.

Alcance:

- `modulos/orquesta-factory`
- `modulos/orquesta-factory-http`
- `modulos/orquesta-web`
- `modulos/orquesta-mcp`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `SolicitarNuevaAppV0` no llama a `time.Now()` de forma implicita o lo hace
  solo mediante una politica inyectada/versionada.
- Los adaptadores reales inyectan reloj UTC y los tests pueden fijarlo.
- Si `now` cero sigue siendo aceptado, queda registrado como issue de contrato
  con razon estable y sin afectar refs causales.
- Pruebas de determinismo ejecutan dos solicitudes equivalentes con reloj fijo y
  comparan recibos/campos temporales.
- Coordinar con `T161` para no crear dos helpers de reloj/ref. Esta tarea es el
  corte focal de factory/AppSpec y no mezcla validacion editorial ni i18n.
- Tests: `go test -count=1 ./modulos/orquesta-factory ./modulos/orquesta-factory-http ./modulos/orquesta-web ./modulos/orquesta-mcp`.

## T194 governance-catalog-output-budget-freshness

Objetivo: definir presupuesto de salida y frescura para la proyeccion publica
del catalogo de gobernanza.

Estado: pendiente.

Alcance:

- `modulos/orquesta-governance`
- `modulos/orquesta-cli`
- `modulos/orquesta-mcp`
- `modulos/orquesta-app-gateway`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- La query acepta limites explicitos con defaults seguros y maximos de
  entradas/bytes.
- La respuesta incluye version/freshness/source refs o una razon estable cuando
  no existan.
- Entradas inactivas, propuestas y cuarentena se resumen por conteo/refs, no por
  payload completo en modo publico por defecto.
- Las pruebas cubren catalogos grandes, filtros vacios, limite excedido y
  preservacion de refs opacas.
- Coordinar con T82/T175: forma HTTP y presupuesto MCP consumen esta proyeccion
  bounded; no la definen por si solos.
- Tests: `go test -count=1 ./modulos/orquesta-governance ./modulos/orquesta-cli ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway`.

## Escaneo backlog 2026-05-24 sexagesima novena pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, documentos vigentes de estado,
  nucleo y principio.
- Backlog, railes observados y duplicaciones vigentes hasta `T194`.
- Busquedas focales con `rg` sobre `InputSchema`, `OutputShape`, `Shape`,
  comandos CLI, help, flagsets, `operator.*` y registro MCP.
- Lectura focal de `modulos/orquesta-mcp/*tool*_v0.go`,
  `modulos/orquesta-mcp/mcp_transport_registry_v0.go`,
  `modulos/orquesta-operator-mcp/operator_capabilities_v0.go`,
  `modulos/orquesta-cli/command_runner_v0.go` y
  `modulos/orquesta-cli/command_flags_v0.go`.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` preservadas
  como refs opacas.

Huecos nuevos:

- Los tools MCP publican `InputSchema`/`Output` como strings compactos
  escritos a mano. El transporte real los reexpone como schema de tool y los
  recursos de operador publican shapes paralelos. Si un DTO o validador cambia,
  un agente externo puede recibir una forma stale y producir requests invalidas
  aunque el handler real acepte otra estructura.
- La CLI mantiene tres fuentes manuales para comandos: el texto de ayuda
  ES/EN, el dispatch por `hasCLIPathV0` y los flagsets por comando. Al sumar
  rutas como gobernanza, contratos, autoprogramacion o run control, una entrada
  puede existir en dispatch pero faltar en help/docs, o anunciar flags que no
  corresponden al handler.

## T195 mcp-tool-input-schema-descriptor-sync

Objetivo: hacer verificable la relacion entre descriptors MCP, DTOs de entrada,
validadores y transporte real para que agentes externos no consuman schemas
stale.

Estado: pendiente.

Alcance:

- `modulos/orquesta-mcp`
- `modulos/orquesta-operator-mcp`
- `modulos/orquesta-app-gateway`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Cada tool registrado en `orquesta-mcp` debe tener descriptor derivado de una
  fuente canonica o una prueba que compare `InputSchema`/`Output` contra DTO,
  validador y handler real.
- Los subtools de `orquesta.operator.operations.v0` deben declarar shapes
  alineados con `OperatorStatusQueryV0`, `OperatorSupervisedBurstRequestV0`,
  `OperatorPendingOutboxQueryV0` y `OperatorDirectedQueryV0`, incluyendo puerto
  ausente como error publico y no como tool inexistente.
- El transporte MCP real no debe prometer campos, defaults, batch, payloads ni
  tools que la composicion no registro; debe distinguir `unavailable`,
  `not_configured` y `schema_stale` con reason codes compactos.
- Los descriptors no deben incluir HOME, rutas locales, tokens, prompts,
  transcripts, payloads de dominio ni ejemplos crudos; solo nombres publicos,
  refs opacas, campos admitidos y errores publicos.
- Coordinar con T23/T25/T93/T174/T175/T182: esta tarea gobierna fidelidad de
  descriptor de tool; no sustituye transporte residente, roadmap, toolbelt de
  prompts, protocolo JSON-RPC, presupuesto de salida ni deadline de ejecucion.
- Tests: `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-app-gateway ./cmd/orquesta-server`.

## T196 cli-command-catalog-help-dispatch-sync

Objetivo: unificar catalogo de comandos CLI, ayuda localizada y dispatch real
para que la CLI no anuncie rutas obsoletas ni oculte comandos soportados.

Estado: pendiente.

Alcance:

- `modulos/orquesta-cli`
- `cmd/orquesta-server`
- `docs/README.md`
- `README.md`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Declarar un catalogo canonico de comandos CLI con path, flags, cliente
  destino, perfil de efecto, help ES/EN y estado `active|legacy|hidden`.
- `cliHelpTextForArgsV0`, `dispatchOrquestaCLIV0` y los `FlagSet` deben salir
  del catalogo o tener test de paridad que falle cuando una ruta aparece solo
  en una fuente.
- La ayuda debe localizarse por catalogo, no por duplicar bloques largos; si un
  comando no esta traducido, debe caer a clave publica estable y no a texto
  incompleto.
- La salida de error por comando desconocido debe incluir codigo, path
  normalizado y sugerencias acotadas, sin volcar argumentos completos que puedan
  traer URLs con credenciales, rutas privadas, HOME, tokens o payloads inline.
- Coordinar con T75/T119/T139/T142/T180: esta tarea gobierna catalogo/help/
  dispatch; no sustituye owner i18n global, docs de uso server-first, shape de
  salida, limites de entrada ni visibilidad de errores de stdio.
- Tests: `go test -count=1 ./modulos/orquesta-cli ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 septuagesima pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, `docs/README.md` y documentos
  vigentes de estado, nucleo, principio, cierre, OPES y matriz de smokes.
- Backlog, railes observados y duplicaciones vigentes hasta `T196`.
- Busquedas focales con `rg` sobre `io.ReadAll`, `json.NewDecoder`,
  `InputSchema`, `OutputShape`, `errores_publicos`, `metodo_no_permitido`,
  `not_configured`, `unavailable`, resources MCP y clientes CLI.
- Lectura focal de `modulos/orquesta-cli/function_contract_client_v0.go`,
  `modulos/orquesta-cli/operational_status_client_v0.go`,
  `modulos/orquesta-mcp/operational_status_resource_v0.go`,
  `modulos/orquesta-operator-mcp/operator_capabilities_v0.go` y rutas HTTP
  MCP con errores publicos locales.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` preservadas
  como refs opacas.

Huecos nuevos:

- `T138` gobierna respuestas HTTP salientes para comandos, web, MCP,
  domain-work y OPES, pero el alcance no incluye los clientes REST de
  `orquesta-cli`. La CLI de `FunctionContract`, `OperationalStatus`,
  gobernanza y otros clientes lee bodies con `io.ReadAll` o decoders directos;
  `T142` solo cubre entrada CLI y `T196` catalogo/help/dispatch. Falta paridad
  explicita de limite/redaccion para respuestas que acaban en envelopes CLI.
- `T195` cubre descriptors de tools MCP, pero los resources MCP publican
  `InputShape`, `PublicErrors`, refs canonicas y capacidades con strings
  estaticos. Resources como operational-status, shared/core contracts,
  governance/roadmap y operator capabilities pueden quedar stale aunque el tool
  schema este verificado.
- Los errores publicos MCP/HTTP estan repartidos en helpers locales con strings
  como `metodo_no_permitido`, `request_body_invalido`,
  `*_no_configurado`, `*_error` y wrappers distintos en `orquesta-operator-mcp`.
  Falta un catalogo verificable de error codes publicos, i18n y retryability
  para no prometer shapes distintos entre HTTP, JSON-RPC, CLI y resources.

## T197 cli-rest-response-bounds-redaction-parity

Objetivo: aplicar a los clientes REST de `orquesta-cli` la misma politica de
limite, decode y redaccion de respuestas que el resto del control plane.

Estado: pendiente.

Alcance:

- `modulos/orquesta-cli`
- `modulos/orquesta-web`
- `modulos/orquesta-mcp`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Los clientes CLI que leen respuestas HTTP deben usar helper con limite por
  comando, `Content-Type` esperado cuando aplique, trailing-data check para JSON
  y error publico estable (`cli_response_too_large`,
  `cli_response_invalid_json` o equivalente).
- Aplicar la politica a function contracts, operational status, governance,
  server status, bootstrap AppSpec, autoprogramacion, run control/queue y
  solicitudes de nueva app, o dejar excepcion documentada con prueba focal.
- Status no-2xx debe devolver codigo, status, retryable y resumen redactado; no
  body completo, HTML, URLs con credenciales, paths locales, HOME, tokens,
  payloads de dominio, prompts ni transcripts.
- La salida OK puede transportar el DTO completo solo si esta dentro del limite
  publico del comando; si excede, debe fallar o degradar con marcador
  verificable, no truncar JSON silenciosamente.
- Coordinar con T138, T139, T142, T162 y T177: esta tarea extiende la politica
  de respuesta a CLI; no redefine salida publica, entrada local, idempotencia ni
  construccion de URL.
- Tests: `go test -count=1 ./modulos/orquesta-cli ./modulos/orquesta-web ./modulos/orquesta-mcp ./cmd/orquesta-server`.

## T198 mcp-resource-descriptor-source-sync

Objetivo: verificar que los resources MCP publicados reflejan DTOs,
validadores, public errors y fuentes vigentes, no strings estaticos stale.

Estado: pendiente.

Alcance:

- `modulos/orquesta-mcp`
- `modulos/orquesta-operator-mcp`
- `modulos/orquesta-observability`
- `modulos/orquesta-governance`
- `modulos/orquesta-core`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Cada resource MCP registrado debe declarar fuente canonica, version/freshness,
  DTO o validador asociado, presupuesto de salida y estado `active|stale|legacy`
  verificable por test.
- `PublicErrors`, `AllowedConsumers`, `AllowedScopes`, `AllowedSections`,
  `InputShape`, `OutputShape` y capabilities deben derivarse de constantes del
  owner o tener test de paridad contra el owner real.
- Si falta puerto, provider, store o fuente canonica, el resource debe publicar
  `not_configured`, `unavailable` o `descriptor_stale` sin inventar schema ni
  leer docs historicos como verdad viva.
- Los descriptors no deben exponer HOME, rutas locales, tokens, payloads de
  dominio, prompts, transcripts, DB, provider real ni ejemplos crudos; solo refs
  opacas, codigos publicos, source refs y limites.
- Coordinar con T25, T83, T84, T175, T194 y T195: esta tarea gobierna
  descriptors de resources; no sustituye roadmap freshness, query operativo,
  indice FunctionContract, presupuesto de salida, governance ni tool schemas.
- Tests: `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-observability ./modulos/orquesta-governance ./modulos/orquesta-core ./cmd/orquesta-server`.

## T199 mcp-public-error-code-catalog

Objetivo: centralizar y verificar los codigos de error publicos MCP/HTTP para
que tools, resources, transporte JSON-RPC, web/CLI y operador no diverjan.

Estado: pendiente.

Alcance:

- `modulos/orquesta-mcp`
- `modulos/orquesta-operator-mcp`
- `modulos/orquesta-web`
- `modulos/orquesta-cli`
- `modulos/orquesta-i18n-docs`
- `cmd/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Declarar catalogo de error codes publicos por tool/resource/HTTP adapter con
  i18n key, HTTP/JSON-RPC mapping, retryable flag y severidad compacta.
- Strings locales como `metodo_no_permitido`, `request_body_invalido`,
  `*_no_configurado`, `*_no_disponible` y `*_error` deben salir del catalogo o
  fallar en prueba de paridad.
- `err.Error()` solo puede propagarse si el error ya es publico y allowlisted;
  errores desconocidos deben mapearse a codigo generico con campo/ref opaco y
  sin payload crudo.
- El mismo fallo de puerto ausente, metodo no permitido, body invalido,
  timeout, schema stale o output demasiado grande debe producir codigo estable
  entre HTTP, JSON-RPC, CLI/web y resource cuando cruzan la misma frontera.
- Coordinar con T137, T139, T174, T176, T195 y T196: esta tarea gobierna
  catalogo de codigos; no redefine parseo JSON, salida de comandos, protocolo
  JSON-RPC, headers de seguridad, schemas de tool ni catalogo CLI.
- Tests: `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-web ./modulos/orquesta-cli ./modulos/orquesta-i18n-docs ./cmd/orquesta-server`.

## Escaneo backlog 2026-05-24 septuagesima primera pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, `docs/README.md` y documentos
  vigentes de estado, nucleo, principio, cierre, OPES y matriz de smokes.
- Backlog, railes observados y duplicaciones vigentes hasta `T199`.
- Busquedas focales con `rg` sobre `io.ReadAll`, `json.NewDecoder`,
  `http.Client`, `Content-Type`, `X-Correlation-ID`, comandos de
  `cmd/orquesta-server` y clientes REST CLI/web/MCP.
- Lectura focal de `cmd/orquesta-server/commands.go`,
  `cmd/orquesta-server/run_status_command_v0.go`,
  `cmd/orquesta-server/shutdown_client.go` y
  `modulos/orquesta-cli/transport_rest_v0.go`.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` preservadas
  como refs opacas.

Hueco nuevo:

- `T197` cubre clientes REST de `orquesta-cli`, y `T139` gobierna la forma de
  salida de comandos, pero el binario `cmd/orquesta-server` mantiene clientes
  HTTP propios para `status`, `run-status` y `stop`. `getStatusBodyV0` y
  `postRunStatusBodyV0` leen bodies con `io.ReadAll`; `run-status` incluye el
  body no-2xx completo en el error; `requestServerShutdownV0` decodifica sin
  limite ni trailing-data check. Es una frontera de operador local y control
  plane distinta de `modulos/orquesta-cli`, por lo que necesita helper compartido
  o paridad probada antes de ampliar comandos residentes.

## T200 cmd-server-management-rest-client-policy

Objetivo: aplicar limites, redaccion y shape publico a los clientes HTTP de
gestion embebidos en `cmd/orquesta-server`, sin confundirlos con los clientes de
`modulos/orquesta-cli`.

Estado: completada 2026-05-26 para clientes HTTP de gestion del binario servidor.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-server`
- `modulos/orquesta-observability`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `status`, `run-status`, `stop` y futuros comandos de gestion del binario
  servidor deben leer respuestas HTTP con limite por comando, `Content-Type`
  esperado cuando aplique, trailing-data check para JSON y reason codes
  compactos.
- Los errores no-2xx deben devolver codigo, status, retryability y resumen
  redactado; no body completo, HTML, URLs con credenciales, rutas privadas,
  HOME, tokens, prompts, transcripts ni payloads de dominio.
- `getStatusBodyV0`, `postRunStatusBodyV0` y `requestServerShutdownV0` deben
  compartir helper o prueba de paridad con la politica CLI/web cuando proceda,
  pero conservar la semantica local del binario servidor.
- La salida OK puede imprimir DTO completo solo si entra en el presupuesto
  publico del comando; si excede, debe fallar o degradar con marcador
  verificable, no truncar JSON silenciosamente.
- Coordinar con T139, T162, T177, T197 y T199: esta tarea gobierna clientes de
  gestion del binario servidor; no redefine salida publica general,
  idempotencia de mutaciones, construccion de URL, clientes `orquesta-cli` ni
  catalogo global de errores MCP/HTTP.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server ./modulos/orquesta-observability`.

Evidencia 2026-05-26:

- `cmd/orquesta-server/command_http_response_v0.go` centraliza limite de body,
  rechazo de no-2xx sin propagar body crudo, `Content-Type` JSON y validacion
  de JSON sin trailing-data para `status`, `run-status` y `stop`.
- `cmd/orquesta-server/shutdown_client.go` reutiliza el helper comun antes de
  decodificar la respuesta de shutdown.
- Cobertura focal:
  `cmd/orquesta-server/command_http_response_v0_test.go`,
  `cmd/orquesta-server/shutdown_client_v0_test.go` y
  `cmd/orquesta-server/run_status_command_v0_test.go`.

## Escaneo backlog 2026-05-24 septuagesima segunda pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, `docs/README.md`, foto vigente
  de estado, guia del nucleo, principio director y matriz de smokes.
- Backlog, railes observados y duplicaciones vigentes hasta `T200`.
- Busquedas focales con `rg` sobre `http.Client`, `CheckRedirect`,
  `Retry-After`, `X-Correlation-ID`, `Idempotency-Key`, `ListExternalJobsV0`,
  loops residentes, conectores OPES y `domain_work-http`.
- Lectura focal de `cmd/orquesta-server/external_bridge_loop.go`,
  `cmd/orquesta-server/opes_bridge_loop.go`,
  `cmd/orquesta-server/opes_bridge.go`,
  `modulos/orquesta-domain-work-http/client_v0.go`,
  `modulos/orquesta-opes-connector/http_v0.go` y
  `modulos/orquesta-opes-connector/job_query_v0.go`.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` preservadas
  como refs opacas.

Huecos nuevos:

- Los conectores HTTP y el loop OPES tienen timeout, ledger e idempotencia, pero
  no una politica comun de retry/backoff/rate. `runExternalBridgeLoopV0`
  reintenta cada intervalo fijo tras error; `domain_work-http` y OPES devuelven
  errores compactos sin `Retry-After`, jitter, presupuesto de reintentos ni
  circuit breaker. Esto puede martillear una app temporal, consumir ticks y
  generar ruido de auditoria aunque T80/T103/T154/T158 ya cubran destino,
  respuesta, deadline y resumen.
- Las requests salientes de dominio transportan `correlation_id` e
  `idempotency_key` dentro del JSON, pero `domain_work-http` solo fija
  `Content-Type` y OPES fija `Accept`/`Content-Type`. No hay politica de
  cabeceras `X-Correlation-ID`, `Idempotency-Key` o `Orquesta-Request-Ref` para
  que proxies, apps externas o ledgers HTTP correlacionen retries sin parsear el
  cuerpo.
- `ListExternalJobsV0` consume una sola respuesta de OPES con `limit`; el bridge
  escanea hasta `limit*10` con maximo 100 y filtra localmente. Si OPES ignora
  limit, no ofrece orden estable o los primeros N jobs ya estan enviados en el
  ledger, el residente puede quedarse revisando la misma ventana y no avanzar a
  jobs pendientes posteriores.

## T201 outbound-connector-retry-backoff-rate-policy

Objetivo: definir politica comun de retry, backoff, jitter y rate budget para
conectores HTTP salientes y loops residentes de dominio sin meter red ni
proveedor en el nucleo.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-domain-work-http`
- `modulos/orquesta-opes-connector`
- `modulos/orquesta-opes-bridge`
- `modulos/orquesta-server`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `runExternalBridgeLoopV0`, OPES drain, `domain_work-http` y conectores de
  dominio deben compartir razon publica para retry/backoff/rate limited, con
  presupuesto por destino/ref y jitter opcional inyectado por composicion.
- Respetar `Retry-After` o equivalente solo si pasa politica de destino y
  deadline; no dormir indefinidamente ni consumir ticks sin publicar estado
  `rate_limited`, `retry_scheduled`, `retry_budget_exhausted` o similar.
- No reintentar automaticamente mutaciones no idempotentes si falta
  `idempotency_key`, ledger o recibo causal; en ese caso bloquear con codigo
  publico y permitir decision del director.
- Los errores deben estar redactados: sin URLs con credenciales, HOME, tokens,
  prompts, transcripts, bodies HTTP, payload OPES completo ni datos de proveedor.
- Coordinar con T80, T98, T101, T103, T154, T158, T162 y T167: esta tarea no
  redefine destino HTTP, ledgers, limites de respuesta, deadline, idempotencia
  publica ni estado de ciclo del bridge; solo gobierna retry/backoff/rate.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-domain-work-http ./modulos/orquesta-opes-connector ./modulos/orquesta-opes-bridge ./modulos/orquesta-server`.

## T202 outbound-domain-correlation-idempotency-headers

Objetivo: propagar correlacion e idempotencia en cabeceras HTTP de conectores de
dominio, manteniendo el cuerpo JSON como contrato canonico y refs opacas.

Estado: pendiente.

Alcance:

- `modulos/orquesta-domain-work-http`
- `modulos/orquesta-opes-connector`
- `cmd/orquesta-server`
- `modulos/orquesta-app-codex-stack`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Para mutaciones salientes, fijar cabeceras compactas como
  `X-Correlation-ID`, `Idempotency-Key` y/o `Orquesta-Request-Ref` desde campos
  ya validados del contrato; no generar IDs nuevos si el contrato trae refs
  causales.
- Las cabeceras nunca deben transportar body, prompt, transcript, rutas locales,
  HOME, tokens, DSN, URL con credenciales ni payload de dominio; si un valor no
  es ref compacta, bloquear o redirigirlo a evidencia por campo.
- Los conectores deben demostrar que retries y ledgers externos pueden
  correlacionar una request sin parsear JSON, pero el contrato canonico sigue
  siendo el DTO de dominio validado por Orquesta.
- OPES conserva sus rutas/adaptador propios; `domain_work-http` conserva el
  conector neutral. La politica comun debe vivir en helper/adaptador, no en
  `orquesta-domain-work` puro.
- Coordinar con T80, T101, T158, T162, T177 y T201: esta tarea no decide host,
  ledger de artefactos, idempotencia publica de control plane, construccion de
  URL ni backoff; solo cabeceras de correlacion/idempotencia salientes.
- Tests: `go test -count=1 ./modulos/orquesta-domain-work-http ./modulos/orquesta-opes-connector ./cmd/orquesta-server ./modulos/orquesta-app-codex-stack`.

## T203 opes-bridge-pagination-window-policy

Objetivo: evitar starvation y perdida silenciosa de jobs OPES cuando el bridge
residente lista una sola ventana de jobs pendientes.

Estado: pendiente.

Alcance:

- `cmd/orquesta-server`
- `modulos/orquesta-opes-connector`
- `modulos/orquesta-opes-bridge`
- `docs/runbooks`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `ListExternalJobsV0` y `opes-drain-once` deben declarar contrato de orden,
  cursor/page token o ventana estable; si OPES no lo ofrece, el bridge debe
  publicar `pagination_not_supported` o `window_exhausted` con contadores
  compactos.
- Si una ventana contiene solo jobs ya enviados, no convertibles o con error de
  contexto/ledger, el loop debe poder avanzar a la siguiente ventana o bloquear
  con razon verificable; no repetir indefinidamente los mismos refs.
- La secuencia por tipos no debe tratar `Seen > 0` como progreso suficiente si
  `submitted=0` y todos los resultados son `already_submitted`/`skipped` por
  ventana agotada; debe distinguir progreso real, ventana agotada y pausa
  intencional por fase.
- Mantener guardas OPES: instancia temporal o filtro seguro, limite bajo,
  ledger idempotente, sin drenar colas amplias ni tocar OPES productivo.
- Coordinar con T12, T21, T98, T101, T158 y T201: esta tarea gobierna
  paginacion/ventanas del bridge OPES; no redefine smokes reales, claim de
  entrada, ledger de submit, destino OPES ni backoff.
- Tests: `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-opes-connector ./modulos/orquesta-opes-bridge`.

## Escaneo backlog 2026-05-24 septuagesima tercera pasada

Evidencia revisada:

- `agent_packet.json`, `AGENTS.md`, `README.md`, `docs/README.md`, foto vigente
  de estado, guia del nucleo, principio director, corte OPES y matriz de
  smokes.
- Backlog, railes observados y duplicaciones vigentes hasta `T203`.
- Lectura focal de `cmd/orquesta-server/opes_bridge.go`,
  `cmd/orquesta-server/opes_bridge_submit.go`,
  `modulos/orquesta-opes-connector/http_v0.go`,
  `modulos/orquesta-opes-connector/job_query_v0.go`,
  `modulos/orquesta-opes-connector/topic_blocks_v0.go`,
  `modulos/orquesta-opes-connector/rest_client_v0.go` y
  `modulos/orquesta-opes-bridge/mapper_v0.go`.
- Busquedas focales con `rg` sobre `topic_blocks`, `PayloadJSON`,
  `InputFields`, `GET /api/jobs/{id}/artifacts`, `ListTopicBlocksV0`,
  conectores OPES y docs de smoke.
- `worktree_ref=worktree-ref-orquesta-server-idle-self-improvement` y
  `branch_ref=branch-ref-orquesta-server-idle-self-improvement` preservadas
  como refs opacas.

Huecos nuevos:

- `summarize_topic` hidrata todos los bloques con `GET /api/topics/{id}/blocks`
  y `appendTopicBlocksFieldV0` inyecta markdown, citas y sources completos en
  `DomainWorkField.ValueJSON`. No hay cursor, limite de bloques, presupuesto de
  bytes por markdown/citations ni degradacion a refs cuando el contexto excede
  presupuesto.
- `payload_json` OPES se transforma casi entero en `DomainWorkFieldV0` y solo
  `summarize_topic` tiene comprobacion especifica de `topic_id`. Falta contrato
  por `job_type` para campos requeridos, aliases reparables, campos desconocidos
  y errores publicos causales antes de crear runs.
- La documentacion OPES menciona `GET /api/jobs/{id}/artifacts`, pero el
  conector actual solo crea jobs, lista jobs/bloques y envia artefactos. Para
  cierre/recovery OPES temporal hace falta readback opt-in de recibos/artefactos
  aceptados, sin usar DB OPES ni summary textual como evidencia.

## T204 opes-topic-block-context-budget-window

Objetivo: acotar y paginar el contexto de bloques OPES usado por
`summarize_topic` antes de meterlo en `DomainWork`.

Estado: pendiente.

Alcance:

- `modulos/orquesta-opes-connector`
- `modulos/orquesta-opes-bridge`
- `cmd/orquesta-server`
- `docs/runbooks`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- `ListTopicBlocksV0` debe declarar orden, cursor/page token o limite estable;
  si OPES no lo ofrece, el bridge debe bloquear con razon publica como
  `topic_blocks_window_unbounded` antes de inyectar contexto enorme.
- `appendTopicBlocksFieldV0` o su reemplazo debe aplicar presupuesto por numero
  de bloques, markdown, citas, source refs y total JSON; si excede, degradar a
  `input_refs`/evidence refs compactas o pedir replan, no truncar silencioso.
- Mantener orden reproducible de capitulo/bloque y contadores
  `context_blocks_seen`, `context_blocks_included` y
  `context_blocks_truncated` en resumen/auditoria, sin payload OPES completo.
- No persistir rutas internas OPES, DB, HOME, tokens, prompts, transcripts,
  markdown masivo ni citas crudas fuera de la frontera opt-in redactada.
- Coordinar con T14, T18, T80, T103, T144, T150, T153, T158 y T203: esta tarea
  gobierna presupuesto/ventana de bloques de contexto OPES; no redefine destino
  HTTP, respuesta saliente, artifact intake, snapshot general ni paginacion de
  jobs.
- Tests: `go test -count=1 ./modulos/orquesta-opes-connector ./modulos/orquesta-opes-bridge ./cmd/orquesta-server`.

## T205 opes-job-payload-schema-by-kind

Objetivo: validar y normalizar `payload_json` OPES por tipo de job antes de
crear runs Orquesta.

Estado: pendiente.

Alcance:

- `modulos/orquesta-opes-bridge`
- `modulos/orquesta-opes-connector`
- `modulos/orquesta-domain-work`
- `cmd/orquesta-server`
- `docs/runbooks`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Definir schema/adaptador por `job_type` para `plan_temario`,
  `draft_content_block`, `generate_visual_asset`, `review_*`,
  `validate_topic`, `assemble_topic` y `summarize_topic`, con campos
  requeridos y aliases reparables.
- Rechazar con error publico causal (`opes_payload_schema_invalid`,
  `opes_payload_required_field_missing`, etc.) payload JSON roto, campos clave
  imposibles o valores que no puedan convertirse en refs/campos seguros.
- No pasar claves arbitrarias de OPES a `DomainWorkField.Name` sin clasificar;
  campos desconocidos solo pueden viajar como extras compactos allowlisted o
  como refs opacas, no como `ValueJSON` masivo.
- Mantener la semantica OPES en el adaptador; `orquesta-domain-work` sigue como
  contrato neutral con validacion estructural, no como catalogo de jobs OPES.
- Coordinar con T14, T18, T26, T72, T80, T144, T153 y T204: esta tarea gobierna
  schema de payload de entrada OPES; no redefine calidad editorial, artifact map
  ni presupuesto de bloques hidratados.
- Tests: `go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector ./modulos/orquesta-domain-work ./cmd/orquesta-server`.

## T206 opes-artifact-readback-acceptance-receipt

Objetivo: anadir readback opt-in de artefactos/receipts OPES para cierre y
recovery causal, sin leer internals de OPES.

Estado: pendiente.

Alcance:

- `modulos/orquesta-opes-connector`
- `modulos/orquesta-opes-bridge`
- `modulos/orquesta-app-codex-stack`
- `cmd/orquesta-server`
- `docs/runbooks`
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
- `docs/rail_errors_observados_2026-05-23.md`
- `docs/duplicaciones_railes_pendientes_2026-05-24.md`

Criterios:

- Implementar puerto/adaptador opt-in para consultar receipts/artefactos por
  `job_ref`, `artifact_ref`, `idempotency_key` o correlation refs mediante API
  publica OPES como `GET /api/jobs/{id}/artifacts`.
- El recovery de `submit_artifact` debe poder distinguir `accepted`,
  `rejected`, `pending`, `duplicate` y `not_found` con refs causales; no cerrar
  por URL, ruta, nombre de conector, summary textual ni estado OPES ambiguo.
- Si el readback falta o OPES no expone endpoint estable, bloquear con razon
  verificable (`opes_artifact_readback_unavailable`) en vez de reintentar
  envios o cerrar el plan.
- No exponer payload completo de artefacto, HTML, markdown, rutas OPES, DB,
  HOME, tokens, prompts, transcripts ni respuestas HTTP crudas en auditoria,
  ACK, status o docs de resultado.
- Coordinar con T18, T21, T98, T101, T103, T158, T201 y T202: esta tarea no
  sustituye ledger de claim/submit ni smoke real OPES completo; aporta readback
  causal para recovery/cierre.
- Tests: `go test -count=1 ./modulos/orquesta-opes-connector ./modulos/orquesta-opes-bridge ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

## T207 rotacion-sesiones-handoff-experimental

Objetivo: mejora futura experimental para estudiar si conviene rotar sesiones
largas de director/agentes mediante handoff durable y relanzamiento con reglas
frescas, sin cortar trabajos ni perder contexto util.

Estado: pendiente futura.

Alcance:

- `modulos/orquesta-director-runner`
- `modulos/orquesta-director-agent-workflow`
- `modulos/orquesta-runtime`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-orchestration-core`
- `cmd/orquesta-server`
- `docs/runbooks`

Dependencias:

- t01
- t02
- t04
- t11
- t17
- t33

Entrada:

- estado vivo del run, task, wave/cohort refs, agentes pendientes y evidencias
  causales persistidas.
- reglas vigentes del proyecto, write-set, pruebas requeridas y decisiones ya
  tomadas por el director/agente saliente.
- presupuesto observado de tiempo, tokens/contexto aproximado, numero de
  acciones, compactaciones y senales de deriva o bloqueo.

Salida:

- handoff estructurado versionado con objetivo original, avance real, archivos
  tocados, pendientes, riesgos, pruebas obligatorias y siguiente accion.
- nueva sesion de director/agente arrancada con reglas frescas y contexto
  acotado desde el handoff, manteniendo refs causales y sin duplicar trabajo.
- metrica comparativa que permita decidir si la rotacion mejora calidad,
  continuidad y coste frente a dejar sesiones largas.

Criterios:

- No implementar como rail estricto inicial ni cortar procesos vivos sin handoff
  aceptado; debe entrar primero como experimento opt-in y de baja prioridad.
- Definir `session_epoch` o equivalente fuera del nucleo puro si depende de
  proveedor/runtime; el nucleo solo debe ver refs opacas, estados y eventos.
- El cierre de una sesion saliente debe ser ordenado: ACK/handoff durable,
  pruebas/evidencias si existen, estado de pendientes y razon de rotacion.
- Si el handoff no contiene datos suficientes, el director debe pedir completar
  handoff o continuar la sesion actual; no relanzar a ciegas.
- Coordinar con automejora, recursion Codex y waits por refs: la rotacion no
  puede convertir la espera en "todos los agentes vivos" ni perder parent/child
  refs.
- Tests: `go test -count=1 ./modulos/orquesta-director-runner ./modulos/orquesta-director-agent-workflow ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-orchestration-core ./cmd/orquesta-server`.

## T208 autoprogramming-guardian-breakglass

Objetivo: implementar un guardian externo minimo para que la autoprogramacion no
dependa de una Orquesta ya rota cuando una version candidata no compila, no
arranca o deja de responder.

Estado: base implementada 2026-05-24; pendiente integrar en el ciclo residente
de promocion/restart.

Alcance:

- `cmd/orquesta-guardian`
- `cmd/orquesta-server`
- `modulos/orquesta-autoprogramming`
- `modulos/orquesta-runtime-worktree`
- `modulos/orquesta-runtime-codex`
- `modulos/orquesta-server`
- `docs/runbooks`

Dependencias:

- t13-autoprogramming-staging-promotion

Entrada:

- binario `last_good`, binario candidato, worktree candidato, commit/ref
  candidato, logs de build/test/arranque y estado de servidor.
- comando de build, pruebas requeridas, healthcheck HTTP/local y ruta de estado
  temporal del candidato.
- diff o refs de cambios de la automejora que han dejado la version candidata
  en fallo.

Salida:

- resultado de promocion o rollback con evidencia durable: `candidate_failed`,
  `last_good_restored`, `repair_agent_started`, `candidate_promoted`.
- tarea de reparacion encolada con logs, diff, tests fallidos, write-set y
  criterio de cierre verificable.
- agente reparador externo opt-in lanzado con permisos amplios y contexto
  acotado solo cuando Orquesta no puede repararse desde dentro.

Criterios:

- No reemplazar el binario vivo si el candidato no compila, no pasa tests
  requeridos o no responde a healthcheck en estado temporal.
- Mantener siempre `last_good` y manifest de promocion atomico; si falla el
  candidato, restaurar/seguir con `last_good` sin necesitar que el director vivo
  funcione.
- El guardian externo no decide producto ni reordena backlog: solo observa
  salud, conserva evidencias, restaura ultimo bueno y lanza una reparacion
  acotada cuando el bootstrap esta roto.
- El agente reparador externo debe cerrarse al terminar, entregar ACK/handoff y
  no pisar agentes vivos ni borrar worktrees/evidencias.
- Si Orquesta esta sana, la reparacion debe entrar por la cola normal; el
  guardian solo es break-glass para build/startup/healthcheck rotos.
- Base cubierta por `cmd/orquesta-guardian`: build candidato, tests requeridos,
  healthcheck temporal, `last_good`, manifest, restore, repair packet,
  `repair-command` opt-in, `--repair-codex` con `codex-launch-wave --agents 1`
  y `shutdown-server` cooperativo con escalado forzado tras timeout o `--now`.
- Pendiente: sustituir el instalador manual del residente por este guardian en
  el ciclo de automejora.
- Tests: `go test -count=1 ./cmd/orquesta-guardian ./cmd/orquesta-server ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-worktree ./modulos/orquesta-runtime-codex ./modulos/orquesta-server`.
