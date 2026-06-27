# Corte: OPES como consumidor de Orquesta - 2026-05-18

Este corte fija la frontera vigente para leer OPES dentro de Orquesta.
Documenta que OPES es una app de dominio consumidora, conectada por adaptadores,
y que el nucleo de Orquesta sigue siendo generico. El mismo corte deja cobertura
focal de `plan_temario -> document_plan` y, desde el smoke del 2026-05-18,
evidencia real acotada contra OPES temporal por REST.

## Decision

Orquesta no es el producto OPES ni una extension interna de su base de datos.
Orquesta es un nucleo hexagonal de orquestacion de agentes para apps externas.
OPES consume ese nucleo por contratos de trabajo externo y por conectores
opt-in, igual que podria hacerlo otra app de dominio.

La regla de reparto queda asi:

- Orquesta aporta director, agentes, plan operativo, oleadas, tareas,
  evidencias, revision, rework, cierre, supervision y trazabilidad causal.
- OPES aporta oposiciones, programas, temas, taxonomia, fuentes, reglas
  pedagogicas, validadores, persistencia, UI/API, ensamblado y publicacion.
- El bridge/conector traduce entre OPES y Orquesta sin compartir internals.

## Nucleo hexagonal generico

El nucleo generico debe poder orquestar trabajo de cualquier dominio. Por eso
sus contratos hablan de runs, tasks, refs, artefactos, candidatos, evidencias,
reviews y cierres; no de tablas OPES, rutas locales OPES, plantillas OPES ni
workers OPES.

La direccion correcta es:

```text
app externa
  -> conector de dominio
  -> external-work/domain-work
  -> director/agentes de Orquesta
  -> artefactos y evidencias causales
  -> conector de dominio
  -> validacion/ensamblado en la app propietaria
```

En ese flujo, las refs que cruzan la frontera son opacas. Orquesta puede
conservar `job_ref`, `domain_ref`, `request_ref`, `artifact_ref` o `evidence_ref`,
pero no debe deducir reglas internas desde la DB o filesystem de OPES.

## Superficie MCP para IA

La intencion de uso por IA debe pasar por MCP. El contrato vigente es el tool
generico `orquesta.domain_work.v0`, con acciones `create_job` y
`submit_artifact`, y los tools de director/supervisor ya publicados por
`modulos/orquesta-mcp`. El bridge HTTP `/api/v0/domain-work` existe como
adaptador fino del mismo executor.

OPES no necesita definir un MCP propio dentro del core. La composicion actual
usa REST OPES como adaptador de dominio y lo inyecta en el executor generico de
MCP. Si mas adelante se usa MCPO o un servidor MCP real, debe envolverse como
transporte/adaptador opt-in de `orquesta-mcp`, no como nueva logica OPES dentro
del nucleo.

## OPES sobre external-work y domain-work

OPES entra hoy por adaptadores de dominio:

- `modulos/orquesta-opes-bridge` toma jobs externos pendientes de OPES y los
  traduce a `StartExternalWorkRunRequestV0` para `/api/v0/external-work/run`.
- `modulos/orquesta-opes-connector` representa el cliente publico OPES.
- `modulos/orquesta-domain-work` aporta contratos neutrales de trabajo,
  payloads, artefactos y records filtrables.
- `modulos/orquesta-document-plan-expander` puede expandir un
  `DomainDocumentPlanV0` a `DomainWorkJobRequestV0[]` sin conocer OPES.

El caso `plan_tema -> document_plan -> derivados` debe leerse como evidencia de
OPES consumiendo Orquesta, no como senal de que OPES define el nucleo. OPES pide
trabajo documental con contexto y reglas; Orquesta planifica y coordina; OPES
valida y ensambla los resultados segun sus reglas.

Para derivados OPES, la integracion real debe mantenerse focal:

- instancia temporal o entorno explicitamente opt-in;
- guarda de confirmacion para producir efectos;
- scope por tipo de job o por `job_ref` exacto solo para no tocar jobs ajenos o
  productivos;
- no drenar colas amplias por defecto;
- no tocar OPES productivo sin una ruta de prueba acotada.
- hasta nueva orden, OPES no descarta textos o artefactos por no cumplir el uso
  previsto. Si una entrega falla validacion editorial, formato, alcance o
  objetivo, se revisa para ver si puede aprovecharse total o parcialmente como
  otro artefacto, borrador, insumo documental, evidencia, nota de revision o
  nueva tarea derivada. Solo se corta sin reaprovechar por seguridad,
  causalidad, refs imposibles, datos sensibles o efectos externos no
  autorizados.

## Frontera DB y conector

La DB de OPES pertenece a OPES. Orquesta no debe abrir conexiones a esa DB desde
core, workflow, domain-work, director, expander ni runtime. Si se necesita
leer o escribir estado OPES, debe hacerse por API publica o por un conector de
composicion explicitamente opt-in que preserve la propiedad del dominio.

`orquesta-domain-work-sql` existe como adaptador SQL de referencia para
`domain_work`, pero no es persistencia global de Orquesta ni un puente implicito
a OPES. Un conector SQL productivo debe vivir fuera del nucleo o en una
composicion concreta y declarar driver, DSN, schema, migraciones, secretos,
permisos, pruebas y guardas. El core solo debe ver puertos y refs opacas.

La salida hacia OPES tambien cruza por conector: Orquesta entrega artefactos,
reviews y evidencias; OPES decide persistencia final, ensamblado editorial y
aceptacion deterministica.

## Que no debe meterse en core

No debe entrar en core, workflow, director puro, domain-work ni expander:

- tipos, tablas, migraciones, queries o repositorios OPES;
- rutas locales, filesystem interno, HOME, tokens, OAuth o secretos OPES;
- reglas pedagogicas especificas como codigo obligatorio del nucleo;
- plantillas editoriales, categorias, programas u oposiciones como dependencias
  internas;
- runtime Codex, modelo, proveedor, sesiones o cuotas;
- REST/MCP/web/CLI concretos de OPES;
- workers OPES que planifiquen con juicio propio sin pasar por director;
- lectura directa de colas amplias o productivas sin guarda opt-in.

Si una decision requiere producto, datos reales, proveedor, modelo, cuota,
runtime, permisos o DB, debe quedarse en adaptador/composicion y documentarse
como frontera, no ascender al core.

## Relacion con supervisor generico

El supervisor generico pertenece al plano de orquestacion: observa progreso,
ticks, bloqueos, entregas, reviews, waits, rework y cierre. No es un supervisor
OPES. OPES puede beneficiarse de esa supervision cuando sus jobs se traducen a
trabajo externo o domain-work, pero la politica de ciclo de vida sigue siendo
de Orquesta y se expresa con contratos neutrales.

Para OPES, el supervisor debe operar sobre refs de Orquesta:

- run/task/job/artifact refs opacas;
- scope por job type/job ref cuando se drenan trabajos de dominio;
- waits acotados por cohorte, ola o agentes objetivo si el Director Operativo
  materializa subtareas;
- evidencias causales de entrega, review y tests/validaciones cuando existan.

Lo especifico de OPES queda fuera: validadores editoriales, persistencia,
ensamblado y publicacion final. El supervisor puede detectar que falta
evidencia o que un artefacto fue rechazado; no debe convertirse en lector de la
DB OPES ni en motor de reglas pedagogicas internas.

## Estado y pendientes

Estado documentado:

- OPES es consumidor/conector, no producto base del nucleo.
- El camino `external-work/domain-work` es la frontera correcta para jobs OPES.
- El bridge OPES debe seguir siendo opt-in, filtrable y sin acceso a internals.
- `orquesta.domain_work.v0` es la superficie MCP generica para una IA; REST OPES
  es el adaptador de dominio inyectado hoy.
- La supervision generica puede aplicarse a runs OPES solo mediante refs y
  eventos neutrales de Orquesta.

Pendiente verificable:

- cablear los derivados OPES al ciclo real del Director/conectores sin mover
  reglas OPES al core;
- cerrar smokes focales de derivados con guardas opt-in y filtro por job type;
- documentar cualquier fuente real de cierre o evidencia OPES como conector de
  composicion, no como dependencia del nucleo.

Cerrado en smoke real acotado:

- `plan_temario` de Operario por REST sobre OPES temporal;
- Codex real `gpt-5.5` con `xhigh`;
- `document_plan` registrado en OPES;
- job OPES completado;
- derivados OPES creados como backlog pendiente.

## Validacion plan_temario Operario

Corte anadido el 2026-05-18:

- `plan_temario` de Operario queda cubierto por mapper OPES: se transforma en
  `external-work` con `artifact_type=document_plan`, `context_budget_profile=large`,
  `expected_schema=domain_document_plan.v0` y write-set
  `external/opes/plan_temario/<job_ref>`.
- El bridge OPES inyecta como contexto de dominio la politica editorial vigente:
  inventario completo, agrupacion, mapa de dependencias, temas maestros,
  derivacion por nivel, revision, HTML publicable, metodo de asimilacion y
  requisitos de calidad. La regla de niveles queda explicita: si existe maestro
  A1/A2 o A1 equivalente, se planifica primero ese maestro y despues se derivan
  B/C1/C2/AP por resumen, reduccion editorial y adaptacion; si no existe, el
  plan debe marcar `creacion_directa_nivel`.
- El drain REST fake de `cmd/orquesta-server` prueba `GET /api/jobs` con
  `job_type=plan_temario`, `job_ref` opcional y envio a
  `/api/v0/external-work/run`.
- El stack prueba el flujo local completo: run externa, supervisor publico
  `POST /api/v0/runs/supervise`, agente Codex fake, entrega `document_plan` y
  `submit_artifact` al puerto `DomainWork`.
- Para temarios, la composicion Codex debe usar `xhigh`. El servidor deja
  `ORQUESTA_CODEX_REASONING_EFFORT` con valor por defecto `xhigh` y permite
  override explicito. El core sigue sin conocer modelos, proveedores ni OPES.
- `DrainRunV0` vuelve a comprobar artefactos `DomainWork` pendientes despues de
  `ContinueAppDirectorV0`, evitando que el supervisor pare en `quiescent` justo
  despues de registrar una entrega pero antes de enviarla al conector.
- El conector REST OPES prueba `submit_artifact` de `document_plan` contra
  `POST /api/jobs/<job_ref>/artifacts`.
- El servidor acepta `ORQUESTA_OPES_BASE_URL` y `OPES_BASE_URL` para cablear el
  executor generico `orquesta.domain_work.v0`; el loop residente exige
  `ORQUESTA_OPES_BRIDGE_CONFIRM=1` y filtro por `JOB_TYPE`, `JOB_REF` o
  `JOB_TYPE_SEQUENCE`.
- Para automatizar derivados por pases, el bridge residente acepta
  `ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE`. En cada tick consulta los tipos en
  orden y drena solo el primer tipo con jobs `pending`; si el ledger indica
  `already_submitted`, bloquea igualmente las fases posteriores hasta que OPES
  deje de exponer pendientes de ese tipo. Esto automatiza el flujo
  `research_exam_precedents -> draft_content_block -> generate_visual_asset ->
  generate_question_bank -> review_* -> validate_topic -> assemble_topic ->
  generate_audio_asset -> generate_tutor_assets -> generate_learning_games ->
  generate_html_site` sin
  meter OPES en el nucleo. Desde el 2026-06-02, el flujo local completo de
  temario queda fijado en `docs/opes_flujo_temario_operativo_2026-06-02.md`:
  investigacion externa, tests, audios por tema/apartado, tutor/bots e HTML
  local USO/TCAE son derivados obligatorios antes de produccion.
- Corte 2026-06-27: cuando `/api/v0/external-work/run` devuelve
  `route_policy=goal_first`, el bridge OPES conserva en su ledger `goal_ref`,
  `external_goal_ref`, `director_execution_mode` y `next_actions`. En ticks
  posteriores reconstruye esa metadata para `already_submitted` y, si el
  operador fuerza supervision o espera residente, llama a
  `/api/v0/apps/director/goal/observe` en vez de `/api/v0/runs/supervise`.
  Los runs legacy sin metadata goal-first mantienen el comportamiento anterior.
- Corte 2026-06-27 noche: el fallback legacy de `external-work/run` requiere
  opt-in doble: composicion con `ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1`
  y payload con `director_execution_mode=legacy_director_loop`. El bridge OPES
  normal no debe mandar esa marca; debe esperar `route_policy=goal_first` cuando
  el backend Goal este configurado o recibir error operativo si falta.

Smoke real acotado del 2026-05-18:

- `program_id`: `1f897b2c119df92d371fea28043d7daf`;
- `job_ref`: `9784a562f074769a08043707fdd79eb2`;
- `run_ref`: `run-external-work-opes-9784a562f074769a08043707fdd79eb2-opes-job-9784a562f074769a08043707fdd79eb2`;
- `artifact_id`: `576d0a93563c2c40e7a6dc41c74261e8`;
- OPES completo el job `plan_temario` y creo 20 derivados pendientes:
  10 `draft_content_block`, 4 `generate_visual_asset`, 2 `review_quality`,
  1 `review_legal`, 1 `review_pedagogical`, 1 `validate_topic` y
  1 `assemble_topic`.

Nota historica: este smoke del 2026-05-18 no cubrio audio, tests, tutor/bots ni
HTML local. La secuencia vigente posterior anade investigacion de examenes,
`generate_question_bank -> question_bank`, `generate_audio_asset ->
audio_asset`, `generate_tutor_assets -> tutor_bot_package` y
`generate_html_site -> local_html_site`; su smoke real completo debe ejecutarse
contra OPES temporal igual que el resto de derivados.

Handoff vigente de ese tramo:
`docs/runbooks/handoff_opes_derivados_reales_hasta_local_html_site_2026-06-07.md`.

Documento de evidencia:

```text
docs/resultado_prueba_opes_orquesta_plan_temario_operario_2026-05-18.md
```

Comandos focales:

```bash
go test -count=1 ./modulos/orquesta-opes-bridge -run 'TestBuildExternalWorkRunRequestV0MapeaPlanTemario'
go test -count=1 ./cmd/orquesta-server -run 'TestRunOPESDrainOnceV0PlanTemario|TestCodexRuntimeConfigV0|TestOPESBridgeLoopConfig|TestDomainWorkExecutorFromEnvV0'
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackV0OPESPlanTemario|TestAgentPacketV0PlanTemario'
go test -count=1 ./modulos/orquesta-opes-connector -run 'TestRESTClientV0.*DocumentPlan|TestRESTClientV0ListExternalJobsFiltraRespuestaMixta'
```

## Archivos cambiados

- `docs/corte_opes_como_consumidor_orquesta_2026-05-18.md`
- `docs/runbooks/smoke_opes_plan_temario_operadores_2026-05-18.md`
- `modulos/orquesta-opes-bridge/document_plan_contract_v0.go`
- `modulos/orquesta-opes-bridge/mapper_v0_test.go`
- `modulos/orquesta-app-codex-stack/opes_document_plan_flow_v0_test.go`
- `modulos/orquesta-opes-connector/rest_client_v0_test.go`
- `modulos/orquesta-opes-connector/job_query_v0.go`
- `cmd/orquesta-server/opes_bridge*.go`
- `cmd/orquesta-server/stack.go`
