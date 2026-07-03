# Contexto para agentes: Orquesta

Lee este archivo antes de trabajar en el repo. Luego lee solo los documentos y
`AGENTS.md` locales necesarios para tu modulo.

## Foto vigente

La direccion actual no es "app de programacion con Codex". Orquesta es un
nucleo reutilizable de orquestacion de agentes para apps externas. Programacion
con Codex y OPES son composiciones consumidoras, no el nucleo.
El nucleo generico tampoco es producto OPES: OPES entra solo por conectores de
dominio sobre puertos/refs opacas, igual que cualquier otra app externa.

Regla principal operativa: cuando el trabajo implique OPES, temarios, tests,
audios, visuales, paquetes, produccion, coordinacion de agentes o subagentes,
el director debe usar Orquesta como superficie de direccion por defecto. Codex
directo no sustituye a Orquesta como flujo productivo principal; puede observar,
integrar, auditar, reparar un tapon acotado o preparar scripts auxiliares, pero
debe volver a lanzar/coordinar el trabajo por Orquesta. Si Orquesta no puede
ejecutar por trust, runtime, puerto, proveedor o bloqueo externo, se informa al
operador con el comando/accion necesaria y se documenta la excepcion; no se
trabaja media hora alrededor del bloqueo ni se convierte la excepcion en nuevo
flujo normal. Gemini y Claude se usan como revisores compactos cuando aporten
valor, no como lectores de contexto bruto completo salvo necesidad justificada.

Tras los cortes del 2026-05-17, el estado real es:

- `modulos/orquesta-director-operativo` existe como contrato puro del Director
  Operativo V1: plan, olas, guardas, contexto insuficiente y delegacion
  recursiva gobernada. No ejecuta runtime.
- `modulos/orquesta-orchestration-core/operational_director_materializer_v0.go`
  materializa solo items `launch_subagents` listos como `WorkflowTaskV0` y
  comandos `CreateMicrotask` con outbox causal. Aun no materializa todo el ciclo
  de espera/review/replan/cierre.
- `WaitAgentRefs` ya existe en los loops progresivos y en el stack Codex para
  esperar agentes concretos. `app-director-service` puede derivarlos desde
  `wait_cohort_ref`, `wait_wave_ref` o `wait_parent_task_ref` usando las
  `WorkflowTaskV0` persistidas. `WorkflowTaskWaitStateV0` registra esa espera
  con causa, tasks, agentes objetivo y pendientes; `orquesta-state-file` lo
  persiste. La regla nueva es usar refs de cohorte/ola cuando esten disponibles,
  no esperar todos los agentes vivos del run.
- `ContinueAppDirectorV0` ya puede recibir un `OperationalDirectorPlanV0` listo,
  materializar `launch_subagents` antes del loop, derivar la ola y reentrar al
  loop con scope acotado.
- El cierre operativo generico ya empieza desde `ContinueAppDirectorV0` tras un
  loop quiescent: si la composicion inyecta `OperationalClosureSource`,
  `app-director-service` construye un `OperationalDirectorClosureRequestV0` y
  `OperationalDirectorClosureV0` cierra task, validacion final y run con review
  aceptada y evidencia durable de tests requeridos. El stack Codex ya inyecta
  una fuente real acotada a microtareas materializadas por el Director
  Operativo; deriva la peticion desde `WorkflowTaskStore` + eventos persistidos
  y no cierra tareas legacy de app-change. Si otra composicion no tiene fuente
  real, ese wiring sigue pendiente y debe documentarse como tal.
- P1 WaitAgentRefs queda cerrado para el stack Codex: `WaitAgentRefs` no vacio
  acota pending, wait e ingesta de ACK/deliveries. `DrainRunRequestV0`
  transporta el scope a `AgentDeliveryObservationRequestV0`, las fuentes de
  observacion quedan filtradas y `DrainRunV0` ignora ACK/delivery de agentes
  fuera del scope. `CodexDeliveryObservationSourceV0` no lee ACKs fuera de scope
  y `domain_work` no hace submit/recovery fuera del scope. `WaitAgentRefs` vacio
  conserva compatibilidad legacy de run completo.
- `app-director-service` ya inyecta `DirectorTaskStore` en review/replan para
  que un `split_task` pueda persistir nuevas `WorkflowTaskV0`. Ese camino ya
  tiene cobertura offline/focal cuando los followups causales existen en
  `WorkflowTaskStore` y estan reflejados en `run.Tasks`; no equivale a
  recursion Codex real completa ni a cierre productivo del arbol.
- El siguiente tramo no es mas scope de espera ni source basico del stack: la
  salida positiva de `review_deliveries` por ola ya avanza el `PlanState` por
  cadena causal de eventos, y `run_required_tests` ya consume
  `RequiredTestEvidenceV0` durable para pasar a `replan_or_close` o bloquear por
  `required-tests-failed`. El runner por puerto, el ejecutor local opt-in, la
  rama negativa review/replan, el replan por tests fallidos, `replan_or_close`,
  `close`, replay/idempotencia del ciclo probado y estado vivo posterior ya
  tienen evidencia offline/fake-runtime. `CODEX-REQTEST-REAL-E2E` cierra un caso
  Codex real acotado con un agente, `EXT-NO-OPES` cierra una app externa
  temporal con `codex-fake`, `CODEX-WAVE-REAL` cierra ola/cohorte Codex amplia,
  `CODEX-RECURSION-REAL` cierra recursion Codex real y OPES derivados/cierre
  quedo cerrado funcionalmente por goal-first temporal en
  `docs/runbooks/resultado_smoke_opes_derivados_goal_first_real_2026-06-28.md`;
  quedan residuales de calidad editorial, coste y automatizacion larga.
- El nuevo handoff de cierre es
  `docs/corte_cierre_generico_director_operativo_2026-05-17.md`: P1
  `WaitAgentRefs` no se reabre salvo regresion; el foco es cierre causal
  offline generico del Director Operativo. Si no hay codigo/prueba integrada,
  documentalo como pendiente verificable, no como hecho.
- OPES es consumidor, no producto base del nucleo. Ya paso el primer corte
  `plan_tema -> document_plan -> derivados` y el
  expander neutral `DomainDocumentPlanV0 -> DomainWorkJobRequestV0[]` ya existe.
  El conector REST OPES opt-in ya existe para `GET /api/jobs`,
  `GET /api/topics/{id}/blocks`, `POST /api/jobs` y
  `POST /api/jobs/{id}/artifacts`; `plan_temario` local queda cubierto como
  `document_plan` con politica editorial OPES y `xhigh`. La superficie para IA
  es `orquesta.domain_work.v0` en `orquesta-mcp`; REST OPES es solo el
  adaptador inyectado hoy, y MCPO/servidor MCP real debe quedar como transporte
  opt-in. El smoke real acotado de `plan_temario` contra OPES temporal ya cerro
  el plan y la creacion de derivados pendientes; el smoke goal-first temporal
  del 2026-06-28 cerro derivados/cierre hasta `completed_syllabus_package`, sin
  tocar OPES productivo ni drenar colas amplias. El flujo vigente de temario
  completo esta
  en `docs/opes_flujo_temario_operativo_2026-06-02.md`: investigacion web de
  examenes relacionados, redaccion, infografias, banco de tests, revisiones,
  ensamblado, audios por tema/apartado, tutor/bots y HTML local USO/TCAE antes
  de produccion.
- La recursion Codex real ya tiene evidencia opt-in: arbol 1 -> 2 -> 4 con
  parent/child refs, presupuesto global, profundidad/fanout, waits acotados,
  entregas vivas, review causal y cierre del arbol por
  `CODEX-RECURSION-REAL`. `CODEX-WAVE-REAL` cubre tambien ola/cohorte amplia con
  proveedor real. Esto no convierte a Codex en nucleo ni cierra nuevos blockers
  futuros; esos casos deben documentarse con evidencia propia.
- La metadata completa de `WorkflowTaskV0` vive en `WorkflowTaskStore`. Un
  replay solo desde eventos compactos reconstruye refs de tareas, no
  `wave_ref`, `cohort_ref`, parent/child refs, criterios completos ni el estado
  vivo de espera; cualquier recuperacion debe restaurar o rematerializar ese
  store y el `WorkflowTaskWaitStateV0`.
- La espina neutral `DirectorCycleStepV0 -> director-runner ->
  director-scheduler -> core-workflow -> director-cycle-outbox` es la fuente de
  verdad para un tick acotado del Director V2. No sustituye al flujo historico
  `app-director-service`/loop progresivo: lo complementa como ciclo puro de
  scheduler/workflow/outbox, sin daemon, runtime real ni composicion residente.
  Lo pendiente en esa linea debe clasificarse como codigo offline, composicion
  residente, smoke real, proveedor real u OPES temporal.
- Corte 2026-06-25: cuando una composicion tenga Codex Goal o un runtime con
  `goal` persistente, el goal actua como Director operativo interno del trabajo.
  Orquesta debe adelgazar su loop: compila `GoalWorkSpecV0` con objetivo,
  reglas, contexto, write-set, tests y artefactos, lanza/observa por adaptador
  opt-in y valida cierre por evidencias. El loop historico
  `app-director-service`/`OperationalDirectorPlanStateV0` queda como
  compatibilidad para rutas no migradas y smokes existentes; no se borra sin
  evidencia equivalente. Corte 2026-06-27: en `StartAppDirectorV0` y
  `orquesta.apps.arrancar_director.v0`, modo vacio significa `goal_first`; si
  falta backend Goal se devuelve `goal_backend_unavailable` y no se cae al loop
  historico. El loop antiguo exige `director_execution_mode=legacy_director_loop`
  explicito. Sunset 2026-07-03: `legacy_director_loop` queda solo para
  mantenimiento correctivo, sin features nuevas, y es candidato a retirada
  cuando goal-first cubra Claude y Gemini (ver T18).

Orden de autoridad documental:

1. `AGENTS.md` y `docs/estado_actual_2026-05-17.md` fijan la foto vigente y la
   frontera conceptual.
2. Los cortes, la matriz y el mapa de generaciones vigentes
   (`docs/guia_nucleo_orquestacion_2026-05-17.md`,
   `docs/corte_cierre_generico_director_operativo_2026-05-17.md`,
   `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md` y
   `docs/mapa_generaciones_director_2026-07-03.md`) fijan estado operativo,
   generaciones de Director, evidencias y smokes.
3. `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md` es backlog
   ejecutable. No debe relanzar un frente que la foto vigente y la matriz ya
   declaran cerrado salvo regresion demostrada.
4. `AGENTS.md` y docs locales de modulo gobiernan solo su alcance y quedan
   subordinados a las fuentes anteriores si estan stale.
5. Documentos historicos o stale sirven como contexto; antes de usarlos como
   evidencia o plan, enlazalos a una fuente vigente.

Documentos de entrada obligatorios para cambios transversales:

- `docs/estado_actual_2026-05-17.md`
- `docs/guia_nucleo_orquestacion_2026-05-17.md`
- `docs/principio_orquesta_piensa_director.md`
- `docs/director_operativo_v1_2026-05-17.md` si cambias Director Operativo,
  plan vivo, workflow tasks, candidatos/outbox, cohortes/oleadas o delegacion
  recursiva.
- `docs/corte_director_funcionando_tarde_2026-05-17.md` si el objetivo es dejar
  Orquesta operativa hoy con el Director, waits, review/rework o recursion.
- `docs/corte_cierre_generico_director_operativo_2026-05-17.md` si cambias
  cierre causal, review por ola, tests requeridos durables, replan/close o
  estado vivo del plan.
- `docs/opes_flujo_temario_operativo_2026-06-02.md`,
  `docs/corte_opes_como_consumidor_orquesta_2026-05-18.md` y
  `docs/runbooks/smoke_opes_plan_temario_operadores_2026-05-18.md` si tocas
  OPES, `orquesta-opes-*`, `DomainWork` aplicado a OPES, bridge/drain OPES o
  reglas editoriales de temarios.
- `/home/alberto/Trabajo/OPES/AGENTS.md` si tocas reglas editoriales,
  operativas o de produccion de temarios OPES. Ese canon se aplica como dominio
  OPES/composicion y no como regla del nucleo generico.
- `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/GUIA_AGENTES_CREACION_TESTS_TEMARIOS_TCAE_OPES_2026-06-02.md`
  si tocas bancos de preguntas, tests, `generate_question_bank` o importacion
  local de tests OPES/TCAE.
- `/home/alberto/Trabajo/OPES/skills/opes-supuestos-practicos/SKILL.md` y
  `skills/orquesta-supuestos-practicos-opes/SKILL.md` si tocas supuestos
  prácticos OPES, casos clínicos, casos operativos o supuestos tipo test.
- `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md` si cambias smokes,
  runtime, OPES, shutdown o pruebas reales.

## Herramientas persistentes de agentes

La instalacion/bootstrap de cualquier entorno Orquesta para agentes debe dejar
activas y documentadas las herramientas auxiliares canonicas. No son preferencias
de una sesion: forman parte del contrato operativo del agente y deben revisarse
cuando cambien versiones, runtimes o skills.

- `codebase-memory-mcp`: herramienta opt-in, no obligatoria. Usala solo cuando
  aporte valor claro para relaciones de simbolos, callers/callees, hotspots o
  arquitectura; para strings exactos, Markdown, configs, incidencias y lectura
  local acotada usa `rg`/`sed` primero. No la uses por defecto en subagentes:
  cuando delegues, indica expresamente `no usar codebase-memory-mcp` salvo que
  un subagente tenga asignada una consulta de grafo concreta. No indexes el repo
  ni arranques varias instancias en paralelo sin orden explicita del operador;
  antes/despues de usarla comprueba que no quedan `codebase-memory-mcp` vivos
  consumiendo CPU. Si quedan, documentalo como incidencia operativa y cierralos
  cooperativamente si no estan atendiendo una consulta activa.
- Ahorro de contexto: lanzar agentes y subagentes con comunicacion compacta tipo
  `caveman` si la skill existe; si no existe, pedir salida compacta equivalente:
  hecho, tests, riesgos/bloqueos y siguiente accion. No cargar contexto bruto
  completo si una busqueda, indice, RAG o resumen acotado basta.
- Registro de herramientas: el bootstrap debe ejecutar
  `scripts/bootstrap_agent_tooling.sh` o una preparacion equivalente para
  instalar/actualizar MCPs/skills, escribir la norma en `~/.codex/AGENTS.md`,
  desactivar UI/puertos externos por defecto y dejar evidencia de version.
- OPES/temarios: no uses `codebase-memory-mcp` como indice principal de contenido
  pedagogico. Para temarios usa el RAG canonico OPES (`rag/corpus/chunks.jsonl`,
  `rag/corpus/summary.json`, `rag/manifest.json`) o un conector documental con
  refs compactas. `codebase-memory-mcp` sirve para codigo OPES/Orquesta,
  scripts, validadores y relaciones, no para leer todos los temas.
- Higiene de disco: todo agente que trabaje en remoto o en sesiones largas debe
  presupuestar espacio antes de lanzar smokes, builds, indexadores o subagentes,
  y debe cerrar el frente dejando limpios temporales y caches que haya creado.
  Usa rutas aisladas y declaradas (`TMPDIR`, `GOTMPDIR`, `GOCACHE`,
  `GOMODCACHE`, `GOPATH`, `ORQUESTA_FLAKY_HARNESS_CACHE_ROOT` y runtimes bajo
  `/srv/orquesta-self/runtime` o `.orquesta-runtime`) para que la limpieza sea
  verificable. No dejes copias obsoletas, bundles, `go-build`, `go-cache`,
  workdirs `audit-*-next`, smokes temporales ni logs enormes sin retencion
  explicita. Antes de borrar, comprueba procesos vivos, sesiones tmux, PIDs,
  worktrees Git, markers de owner, `agent_ack`, `director_decisions`, `outbox`,
  `plan_state`, manifests de artefactos y evidencias citadas por incidencias.
  Si hay duda, no borres: resume la evidencia, marca la ruta como pendiente de
  retencion y pide/crea una tarea de limpieza gobernada. No limpies rutas de
  produccion ni OPES productivo desde una sesion de programacion.

## Capas

- Core puro: `modulos/orquesta-core-workflow`.
- Loop de aplicacion del nucleo: `modulos/orquesta-orchestration-core`.
- Trabajo goal-first neutral: `modulos/orquesta-goal`.
- Adaptador Codex Goal opt-in: `modulos/orquesta-runtime-codex-goal`.
- Plan operativo del director: `modulos/orquesta-director-operativo`.
- Trabajo externo neutral: `modulos/orquesta-domain-work`,
  `modulos/orquesta-app-change`, `modulos/orquesta-external-work-run`.
- Deploy declarativo/dry-run: `modulos/orquesta-deploy`.
- Adaptadores de dominio: `modulos/orquesta-opes-*`.
- Runtime neutral: `modulos/orquesta-runtime`.
- Adaptador Codex: `modulos/orquesta-runtime-codex*`.
- Composicion real actual: `modulos/orquesta-app-codex-stack` y
  `cmd/orquesta-server`.
- Persistencia concreta actual: `modulos/orquesta-state-file` y
  `modulos/orquesta-run-file`.
- Adaptador SQL externo de referencia: `modulos/orquesta-domain-work-sql`, solo
  para `domain_work` y sin bundle productivo, driver, DSN, migraciones ni wiring
  de servidor.

## Reglas

- No reintroduzcas `cmd/db/internal`, `ensureLocalDB` ni control-plane legacy.
- No metas Codex, OPES, web, MCP, DB concreta, HOME, OAuth, tokens, paths
  locales ni runtime real dentro de core/workflow/domain-work.
- No trates `orquesta-domain-work-sql` como persistencia global del sistema: una
  DB real debe entrar por composicion opt-in con driver, schema y pruebas propias.
- Toda app externa debe entrar por contratos y refs opacas. No compartir DB ni
  filesystem interno con Orquesta.
- Toda app completa generada por Orquesta debe nacer hexagonal pura:
  dominio/aplicacion, puertos y adaptadores separados; HTTP/UI/DB/runtime/
  proveedores solo como adaptadores; composicion y wiring solo en bootstrap/cmd.
  No mezclar handlers, persistencia, configuracion ni reglas de dominio.
- El dominio externo aporta datos, reglas, validadores y ensamblado. Orquesta
  aporta juicio mediante director/agentes.
- No conviertas validadores deterministicos en NLU pobre por strings exactos. Si
  un agente devuelve una intencion razonable con alias, nombre cercano o forma
  equivalente (`web_application` por `web_app`, globs por alcance, rutas hijas
  por carpeta), normaliza en el adaptador o deja que el director repare la
  forma. Corta fuerte solo por seguridad, causalidad, refs imposibles, datos
  sensibles o efectos externos no autorizados.
- Regla de fuego vigente hasta nueva orden: no anadas filtros, rails,
  cinturones ni clasificadores por palabras para cortar trabajo de agentes. No
  pares, rechaces, marques
  `capacity_limited`, `garbage`, `failed` ni descartes una entrega por un string
  suelto, alias, nombre cercano, formato recuperable, contexto omitido por
  presupuesto o heuristica de logs. Las formas recuperables se normalizan en el
  adaptador o las corrige el Director con rework/replan; el trabajo se conserva.
  En OPES y otros dominios, si un texto o artefacto no cumple el uso previsto,
  revisalo para aprovecharlo total o parcialmente como otro artefacto, borrador,
  insumo documental, evidencia, nota de revision o tarea derivada antes de
  descartarlo.
  Para temarios y cursos, no rehagas material existente por defecto. Primero
  inventaria temas, comunes, tests, audios, visuales, HTML, tutor/RAG y paquetes
  ya creados; revisalos contra el canon oficial del nuevo curso; copia o enlaza
  lo valido al nuevo trabajo; marca como borrador/revision lo recuperable; y
  rehace solo lo que este obsoleto, sea incorrecto, no cubra el programa o tenga
  un fallo estructural real. Esta regla aplica a todos los Orquesta/agentes que
  preparen temarios, no solo al curso en curso.
  Los unicos cortes fuertes aceptables son seguridad, causalidad, refs
  imposibles, datos sensibles o efectos externos no autorizados.
  Los rails de detalle operativo/local (`detalle_prohibido`, nombres de
  ficheros de control, runtime, provider, modelo, HOME escrito como referencia
  o diagnostico sin valor sensible) son advisory en produccion: se registran
  como evidencia o nota de revision, pero no bloquean una entrega ni paran un
  agente. Si vuelven a bloquear trabajo valido, hay que quitarlos del camino de
  ejecucion, no endurecerlos.
  El rail no decide: el Director o el agente que orquesta el trabajo decide si
  una regla blanda se ignora, se elimina, se conserva como diagnostico o se
  transforma en tarea de mejora. El codigo no debe convertir una heuristica
  blanda en veto automatico.
- Si una decision requiere producto, runtime, modelo, cuota o proveedor,
  documenta la frontera y dejala en adaptador/composicion.
- Configuracion canonica: Orquesta y cualquier app generada o modificada por
  sus agentes deben concentrar variables globales, nombres de entorno, defaults,
  limites y metadata editable en una superficie unica por composicion/app. No
  dupliques variables con nombres distintos repartidas por el codigo; los
  adaptadores deben consumir constantes/registro canonico y documentar ahi los
  cambios que requieran reinicio.
- Pendiente prioritario 2026-06-08: ver T259 en
  `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`. Orquesta debe
  autoverificarse y parar cooperativamente cuando consuma CPU de forma sostenida
  sin causa operativa ni progreso observable. Esto pertenece a servidor/
  composicion por puertos de telemetria y shutdown, no al core puro, y no debe
  convertirse en rail de contenido para agentes. Al cerrar sesiones manuales,
  comprueba que no quedan `orquesta-server run` locales de prueba activos salvo
  instruccion expresa del usuario.
- No borres documentos o codigo antiguo sin revisar. Si algo es historico,
  marcalo como historico y apunta al documento vigente.
- No borres archivos, docs, tests ni piezas "legacy" solo porque parezcan
  obsoletas: revisa primero referencias, historial y uso actual; si no puedes
  cerrarlo con evidencia, deja nota de pendiente.
- Mantén los archivos manejables: responsabilidad clara por fichero, sin
  controladores enormes ni mezclar wiring, dominio, validacion y UI en la misma
  pieza. Si una implementacion crece, separa helpers/puertos/adaptadores/tests
  siguiendo el patron local antes de seguir añadiendo codigo.
- Inventario de bugs: cualquier bug, falso verde, cuelgue, regresion,
  incidencia de agente, desviacion de contrato o fallo operativo observado debe
  registrarse en `docs/inventario_bugs_orquesta_2026-06-30.md` o en una
  incidencia enlazada desde ese inventario. No lo trates como caso aislado hasta
  clasificar area, evidencia, estado e hipotesis arquitectonica; si se arregla
  con codigo, conserva la fila y anade commit/prueba de cierre.
- Delegacion operativa: si un agente necesita ayuda y el runtime/composicion lo
  permite, debe activar subagentes para paralelizar analisis, implementacion,
  pruebas o revision. Cada Codex padre puede usar hasta 6 subagentes. No hay
  limite global artificial de Codex padres ni de agentes vivos salvo limite duro
  del proveedor/runtime/OS o instruccion explicita del operador; si existe un
  limite duro, documentalo como frontera externa temporal, no como politica del
  nucleo. La composicion Codex debe arrancar con 70 padres/ejecuciones por tick
  como default operativo amplio, no con 10 como cuello de botella silencioso.
  Conserva refs/parentesco, write-set, presupuesto y evidencia en el ACK. Para
  evitar tormentas de CPU, los subagentes no deben usar `codebase-memory-mcp` ni
  indexadores salvo autorizacion explicita y acotada a una sola consulta/frente.
- Cuando una tarea ya trae `child_task_refs`, esos hijos no son una sugerencia
  de ayuda: son contrato causal del plan. El padre no debe cerrar como completo
  sin ACK, entrega, bloqueo o rework documentado por cada hijo. En OPES, si el
  contrato declara `subroles_required=6` o `opes.padre-tema-6-subroles.v1`,
  Orquesta debe materializar seis subagentes/subroles reales o dejar bloqueo
  operativo explícito; no basta con una tabla de roles escrita por el padre.
- Economia de tokens: al lanzar agentes o subagentes, pide comunicacion compacta
  y tecnicas de ahorro como `caveman` si estan disponibles. Para exploracion y
  pruebas usa razonamiento `medium` por defecto; no uses `xhigh` salvo orden
  explicita o riesgo tecnico justificado y documentado. Acota contexto con `rg`
  y lecturas parciales, resume documentos largos en vez de pegarlos completos y
  pide finales breves con solo cambios, pruebas y bloqueos.

## Checklist operativo para futuros agentes

1. Declara write-set antes de editar y mantenlo estrecho.
2. Lee `AGENTS.md` local del modulo y los documentos vigentes listados arriba.
3. Si tocas Director Operativo, comprueba la cadena:
   `orquesta-director-operativo` -> `BuildOperationalDirectorWaveWorkV0` ->
   `OperationalDirectorPlanMaterializerV0` -> `WorkflowTaskStore` ->
   `WorkflowTaskCandidateProviderV0` -> wait refs de `app-director-service` ->
   providers de review/replan -> outbox.
4. Si tocas waits, usa `WaitAgentRefs`/cohorte/ola; no conviertas la espera en
   "todos los agentes del run". Conserva la regla cerrada: `WaitAgentRefs` no
   vacio limita pending, wait e ingesta de ACK/deliveries; `WaitAgentRefs`
   vacio conserva compatibilidad legacy.
5. Si tocas OPES, exige instancia temporal y guarda de confirmacion. Usa
   `job_type`/`job_ref` solo como scope opt-in para no tocar jobs ajenos o OPES
   productivo; no lo conviertas en filtro de agentes, de capacidad ni de
   entregas.
6. Si tocas recursion Codex, conserva parent/child refs, limites de
   profundidad/fanout, presupuesto y review causal.
7. El tramo `OperationalDirectorPlanMaterializerV0 -> WorkflowTaskStore ->
   WorkflowTaskWaitAgentRefsV0 -> WorkflowTaskWaitStateV0 ->
   ContinueAppDirectorV0 -> DrainRunV0 con ingesta acotada por WaitAgentRefs`
   esta verificado offline. El cierre desde `ContinueAppDirectorV0` solo es real
   cuando el loop queda quiescent y hay `OperationalClosureSource` inyectado con
   refs/evidencias causales. `run_required_tests` ya valida
   `RequiredTestEvidenceV0` causal desde el state y entrega esas refs al cierre.
   Si hay `RequiredTestRunner` inyectado, genera evidencia por puerto sin que el
   nucleo conozca shell/runtime concreto. El siguiente foco esta en
   `docs/corte_cierre_generico_director_operativo_2026-05-17.md`: OPES temporal
   real de derivados/cierre, sin reabrir `WaitAgentRefs`, `CODEX-WAVE-REAL` ni
   `CODEX-RECURSION-REAL` salvo regresion demostrada.

## Verificacion minima

Para cambios transversales:

```bash
git diff --check
go test -count=1 ./...
```

Para cambios de frontera del nucleo, la prueba raiz
`TestNeutralOrchestrationPackagesDoNotImportProductAdapters` debe seguir verde.
