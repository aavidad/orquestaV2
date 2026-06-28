# Smoke OPES plan_temario Operario

Objetivo: probar un job OPES `plan_temario` sin contaminar el nucleo de
Orquesta. OPES entra por bridge REST de dominio, Orquesta crea una run externa
generica, el supervisor empuja la run y el conector recibe un `document_plan`.
La superficie AI-first equivalente es `orquesta.domain_work.v0` en MCP; REST
OPES solo es el adaptador de composicion inyectado hoy.

Este runbook usa el programa real ya existente en OPES:

```text
/home/alberto/Trabajo/OPES/OPES/administracion-especial/Operario/Operario.txt
```

No crear encabezados de ejemplo si ese fichero esta disponible. OPES importa
ese programa y conserva la propiedad de sus datos; Orquesta solo consume el job
externo y lo resuelve mediante Director/supervisor, agentes y `DomainWork`.

## Quien orquesta

La ruta legacy vigente para este smoke no es un script paralelo ni stdin. Hasta
que exista external-work Goal-first, este camino debe tratarse como
compatibilidad `legacy_director_loop`:

```text
OPES plan_temario pending
  -> opes-drain-once / opes bridge loop
  -> /api/v0/external-work/run con director_execution_mode=legacy_director_loop
  -> cola de runs de Orquesta
  -> supervisor residente con allow_legacy_drain desde opt-in o POST /api/v0/runs/supervise con director_execution_mode=legacy_director_loop
  -> ContinueAppDirectorV0 / DrainRunV0
  -> outbox LaunchRuntimeAgent
  -> Codex xhigh
  -> DomainWork.submit_artifact(document_plan)
  -> OPES valida y crea derivados
```

Si `cmd/orquesta-server run` esta activo, el supervisor global del servidor ya
empuja la cola en cada tick. `POST /api/v0/runs/supervise` queda como empuje
manual/acotado para una run concreta, no como canal alternativo. En ambos
casos legacy hacen falta las dos llaves: opt-in de composicion
(`ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1` para esta ruta OPES) y
`director_execution_mode=legacy_director_loop`; el tick residente traduce ese
opt-in a `allow_legacy_drain`.

## Regla operativa

Para temarios se usa Codex con razonamiento `xhigh`.

El servidor actual aplica `ORQUESTA_CODEX_REASONING_EFFORT=xhigh` por defecto
si el entorno no lo declara. En smokes reales conviene declararlo igualmente
para que quede visible en la sesion:

```bash
export ORQUESTA_CODEX_MODEL=gpt-5.5
export ORQUESTA_CODEX_REASONING_EFFORT=xhigh
```

Nota 2026-06-11: la autovigilancia del servidor por CPU sostenida sin progreso
no cambia las guardas OPES. No autoriza drenados amplios, no toca OPES/TCAE
productivo y no sustituye `ORQUESTA_OPES_BRIDGE_JOB_TYPE`,
`ORQUESTA_OPES_BRIDGE_JOB_REF`, `ORQUESTA_OPES_BRIDGE_LIMIT` ni las
confirmaciones de instancia temporal.
Rework OrquestaV2 2026-06-11: la correccion
`agent-ref-task-ref-review-rework-task-autoprogramming-7c02f2568e45-g01-449fcc36b3c508d55429e771db52f5fb`
solo revalida pruebas de Orquesta y evidencia `ref_only`; no es permiso para
ejecutar `opes-drain-once` ni para ampliar scope de temario.

Guardas de destino y scope 2026-06-11:

- Loopback (`127.0.0.1`, `localhost`) solo demuestra destino local. No cuenta
  como OPES temporal confirmado ni autoriza efectos reales sin
  `ORQUESTA_OPES_BRIDGE_CONFIRM=1` y confirmacion de instancia temporal o
  productiva segun corresponda.
- Productivo requiere confirmacion explicita de operador y evidence ref
  compacta; no se usa para smokes de temario ni derivados.
- `ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE` ordena fases, pero no acota por si
  sola. En modo real debe combinarse con `ORQUESTA_OPES_BRIDGE_PROGRAM_ID`,
  `ORQUESTA_OPES_BRIDGE_TOPIC_ID`, `ORQUESTA_OPES_BRIDGE_CORRELATION_ID` o
  scope equivalente, y con `ORQUESTA_OPES_BRIDGE_LIMIT=1` salvo decision
  operatoria documentada.
- Si falta scope duro, ejecutar solo dry-run o parar con bloqueo verificable;
  no drenar jobs ajenos para "ver que pasa".

## Handoff Operario paralelo

Estado operativo que deben heredar futuros agentes antes de seguir el smoke de
derivados:

- Persistencia: para Operario se usa Postgres de forma obligatoria. SQLite no es
  ruta valida para nuevos smokes, colas de derivados ni ejecuciones paralelas;
  queda solo como compatibilidad historica.
- Salidas: las salidas, workdirs y evidencias del smoke deben quedar bajo un
  directorio `opes-salidas/` dedicado, por ejemplo
  `/tmp/opes-salidas/operario-20260518-<id>/`. No mezclar esas salidas con
  carpetas reales de temarios OPES.
- Limite del bridge: `ORQUESTA_OPES_BRIDGE_LIMIT` se interpreta como numero de
  envios nuevos a Orquesta por tick, no como tamano total de escaneo. En modo
  real sin `JOB_REF`, el bridge escanea una ventana mayor para poder saltar jobs
  ya enviados por ledger y aun asi producir hasta `LIMIT` envios nuevos.
- Concurrencia objetivo: para completar Operario, el objetivo operativo es 10
  temas en paralelo sobre OPES temporal/Postgres aislado. Subir la concurrencia
  solo despues de validar el plan y la secuencia con limites bajos.
- Calidad: mantener `ORQUESTA_CODEX_REASONING_EFFORT=xhigh` para
  `plan_temario`, `draft_content_block`, revisiones, validacion y ensamblado.
- Entregas materializables: no descartar una entrega solo porque el agente use
  nombres alternativos de `artifact_type` o campos del payload. Si la entrega es
  parseable, canonizable y materializable al contrato esperado, debe normalizarse
  y enviarse a OPES; solo se rechaza por contrato roto, evidencia insuficiente o
  calidad/materializacion imposible.

## Prueba local sin OPES real

```bash
go test -count=1 ./modulos/orquesta-opes-bridge \
  -run 'TestBuildExternalWorkRunRequestV0MapeaPlanTemario'

go test -count=1 ./cmd/orquesta-server \
  -run 'TestRunOPESDrainOnceV0PlanTemario|TestRunOPESDrainOnceV0SecuenciaPases|TestCodexRuntimeConfigV0|TestOPESBridgeLoopConfig|TestDomainWorkExecutorFromEnvV0'

go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run 'TestCodexStackV0OPESPlanTemario|TestAgentPacketV0PlanTemario'

go test -count=1 ./modulos/orquesta-opes-connector \
  -run 'TestRESTClientV0SubmitDomainWorkArtifactEnviaDocumentPlan|TestRESTClientV0ListExternalJobsFiltraRespuestaMixta'
```

Esta prueba valida:

- `plan_temario` se mapea a `document_plan`;
- el contrato inyecta la metodologia OPES: inventario, dependencias, maestros
  superiores, derivacion A1/A2 o A1 -> B/C1 -> C2/AP, asimilacion y criterios
  de calidad;
- el bridge filtra `job_type=plan_temario` y puede acotar por `job_ref`;
- `/api/v0/external-work/run` recibe el trabajo y declara
  `route_policy=legacy_director_loop`;
- `/api/v0/runs/supervise` empuja la run legacy o la sustituye el supervisor
  residente;
- el artefacto termina en `DomainWork.submit_artifact`;
- el paquete de agente y el wrapper Codex materializado usan `xhigh`.

## Autoprogramacion acotada

Cuando este smoke se use como tarea de autoprogramacion de Orquesta, el alcance
de edicion debe quedar limitado a los adaptadores OPES y sus runbooks/scripts:

- `scripts`
- `docs/runbooks`
- `modulos/orquesta-opes-bridge`
- `modulos/orquesta-opes-connector`

La rama y el worktree de esa tarea se tratan como refs opacas de Orquesta. Los
adaptadores OPES no deben interpretar ni reescribir esas refs: solo conservan
`job_ref`, `correlation_id`, `idempotency_key`, `run_ref`, `task_ref` y
`delivery_ref` cuando cruzan la frontera por contratos publicos.

Validacion focal para ese corte acotado:

```bash
go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector
```

## Preparar OPES Postgres con Operario

Usar Postgres tambien para pruebas acotadas. SQLite queda solo como historico o
compatibilidad, no como ruta nueva para Operario ni para Orquesta. Si hace falta
aislar la prueba, crear una base Postgres separada y documentar su DSN. No
apuntar a la cola productiva sin `job_ref` y confirmacion.

Importar el programa real en una base Postgres acotada:

```bash
cd /home/alberto/Trabajo/OPES/opes-uso
go run ./cmd/opes-cli \
  -persistence postgres \
  -dsn "postgres://opes_app_user:REPLACE_WITH_STRONG_PASSWORD@localhost:15432/opes_operario_orquesta_20260518?sslmode=disable" \
  -category "Operario" \
  -group "AP" \
  -area "administracion-especial" \
  -source "OPES/administracion-especial/Operario/Operario.txt" \
  -file /home/alberto/Trabajo/OPES/OPES/administracion-especial/Operario/Operario.txt
```

Resultado esperado de importacion:

- 10 epigrafes;
- temas 1 y 2 comunes AP;
- temas 3 a 10 especificos de planchado, lavado, limpieza, cocina, alimentos,
  manipulacion, conservacion/transporte y PRL.

Crear en OPES un job externo `plan_temario` para ese `program_id` usando la API,
MCP o la herramienta administrativa vigente de OPES. El payload debe incluir
como minimo:

```json
{
  "program_id": "<program_id>",
  "document_kind": "temario_oposicion",
  "language_code": "es",
  "level": "AP",
  "title": "Temario Operario AP",
  "official_source_id": "OPES/administracion-especial/Operario/Operario.txt",
  "expected_artifact_type": "document_plan",
  "reasoning": "xhigh"
}
```

## Prueba contra OPES acotado

Precondiciones:

- OPES expone un job pendiente `type=plan_temario`, `execution_mode=external`;
- `payload_json` del job es un string JSON, no un objeto embebido;
- Orquesta server esta arrancado con Codex real y `ORQUESTA_CODEX_REASONING_EFFORT=xhigh`;
- usar siempre scope de temario: `job_ref` exacto o `program_id`/`correlation_id`
  junto al tipo o secuencia, y limite bajo. Ese scope evita mezclar colas de
  otro temario; no descarta entregas por estilo, sinonimos o formato reparable.

Dry-run:

Sustituir `job-ref-plan-temario-operario-001` por el `job_ref` real del job de
smoke; si no se conoce, eliminar esa variable y conservar
`ORQUESTA_OPES_BRIDGE_JOB_TYPE=plan_temario`.

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_OPES_BRIDGE_DRY_RUN=1 \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=<program_id> \
ORQUESTA_OPES_BRIDGE_CORRELATION_ID=<correlation_id> \
ORQUESTA_OPES_BRIDGE_JOB_TYPE=plan_temario \
ORQUESTA_OPES_BRIDGE_JOB_REF=job-ref-plan-temario-operario-001 \
go run ./cmd/orquesta-server opes-drain-once
```

Crear run en Orquesta. Si el servidor residente esta activo, su supervisor
global empezara a empujar la run por la cola:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_BASE_URL=http://127.0.0.1:<puerto-orquesta> \
ORQUESTA_OPES_BRIDGE_CONFIRM=1 \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
ORQUESTA_OPES_BRIDGE_JOB_TYPE=plan_temario \
ORQUESTA_OPES_BRIDGE_JOB_REF=job-ref-plan-temario-operario-001 \
ORQUESTA_OPES_BRIDGE_WAIT_RESIDENT_SECONDS=15 \
ORQUESTA_OPES_BRIDGE_WAIT_RESIDENT_INTERVAL_MS=500 \
go run ./cmd/orquesta-server opes-drain-once
```

`ORQUESTA_OPES_BRIDGE_WAIT_RESIDENT_SECONDS` no llama a
`/api/v0/runs/supervise`: solo observa `/api/v0/director/stats` durante una
ventana corta para que `opes-drain-once` pueda devolver
`supervision_status=started` si el pulso residente ya despacho la run, o
`resident_director_pending` con `supervision_stop_reason=resident_dispatch_wait_timeout`
si no hay dispatch visible.

Empuje acotado opcional si se quiere acelerar una run concreta sin esperar al
siguiente tick del servidor:

```bash
curl -sS -X POST http://127.0.0.1:<puerto-orquesta>/api/v0/runs/supervise \
  -H 'Content-Type: application/json' \
  -d '{
    "director_execution_mode":"legacy_director_loop",
    "run_ref":"<run_ref>",
    "max_ticks":8,
    "max_bursts":16,
    "max_steps_per_burst":8,
    "max_dispatches_per_wait":8,
    "max_commands":20,
    "max_outbox_per_cycle":8,
    "max_external_waits":2,
    "continue_message":"sigue hasta entregar document_plan"
  }'
```

Resultado esperado:

- OPES recibe `POST /api/jobs/<job>/artifacts`;
- `artifact_type=document_plan`;
- `payload_json.schema_version=domain_document_plan.v0`;
- `payload_json.work_kind=plan_temario`;
- `sections` y `deliverables` no estan vacios;
- `quality_criteria` o `constraints` reflejan si el temario deriva de maestro
  A1/A2 o A1, o si procede `creacion_directa_nivel`;
- `complete_job=true`.

## Resultado obtenido 2026-05-18

Prueba real ejecutada sobre instancias temporales:

- OPES: `http://127.0.0.1:18084`
- Orquesta: `http://127.0.0.1:18787`
- raiz temporal: `/tmp/opes-orquesta-operario-e2e-FUZcGH`
- `program_id`: `1f897b2c119df92d371fea28043d7daf`
- `job_ref`: `9784a562f074769a08043707fdd79eb2`
- `run_ref`: `run-external-work-opes-9784a562f074769a08043707fdd79eb2-opes-job-9784a562f074769a08043707fdd79eb2`
- `artifact_id`: `576d0a93563c2c40e7a6dc41c74261e8`

El servidor Orquesta se arranco con bridge OPES residente, filtro
`JOB_TYPE=plan_temario`, `JOB_REF=9784a562f074769a08043707fdd79eb2`,
`LIMIT=1`, Codex `gpt-5.5` y razonamiento `xhigh`.

Resultado verificado:

- OPES importo `Operario.txt` con 10 epigrafes reales;
- el bridge residente envio un solo job a `/api/v0/external-work/run`;
- el supervisor residente lanzo un agente Codex real;
- el agente escribio `agent_ack.json` con `status=completed`;
- Orquesta envio `document_plan` a `POST /api/jobs/<job>/artifacts`;
- OPES marco el job como `completed`;
- OPES creo 20 derivados pendientes: 10 `draft_content_block`, 4
  `generate_visual_asset`, 2 `review_quality`, 1 `review_legal`, 1
  `review_pedagogical`, 1 `validate_topic` y 1 `assemble_topic`.

Evidencia detallada:

```text
docs/resultado_prueba_opes_orquesta_plan_temario_operario_2026-05-18.md
```

## Automatizar derivados por pases

Despues de que `plan_temario` entregue `document_plan`, OPES materializa los
jobs derivados. Ya no hace falta cambiar manualmente `JOB_TYPE` para cada fase:
el bridge residente acepta una secuencia segura de tipos.

Semantica:

- en cada tick, Orquesta consulta la secuencia en orden;
- drena solo el primer `job_type` que OPES siga mostrando como `pending`;
- si ese job ya estaba enviado segun el ledger, no lo reenvia y tampoco avanza
  a fases posteriores;
- cuando OPES deja de mostrar pendientes de una fase, el siguiente tick avanza
  a la siguiente;
- la cobertura offline verifica la secuencia completa
  `plan_temario -> update_topic_registry -> research_exam_precedents -> draft_content_block ->
  generate_visual_asset -> generate_question_bank -> review_legal -> review_pedagogical ->
  review_quality -> review_codex -> review_gemini -> review_claude ->
  review_pair_codex_gemini -> review_pair_codex_claude ->
  review_pair_gemini_claude -> review_director_consolidation ->
  validate_topic -> assemble_topic -> generate_audio_asset ->
  generate_tutor_assets -> generate_learning_games -> generate_html_site -> generate_help_manual_assets ->
  finalize_temario_package`, con ledger/idempotencia por fase,
  `assemble_topic` mapeado a `assembled_topic`, `generate_audio_asset` mapeado
  a `audio_asset`, `generate_question_bank` mapeado a `question_bank`,
  `generate_tutor_assets` mapeado a `tutor_bot_package`,
  `generate_learning_games` mapeado a `learning_games_package`, `generate_html_site`
  mapeado a `local_html_site`, `generate_help_manual_assets` mapeado a
  `help_manual_package` y `finalize_temario_package` mapeado a
  `completed_syllabus_package`;
- la secuencia canonica vive en `modulos/orquesta-opes-bridge` y el comando de
  ciclo en `cmd/orquesta-server`; el nucleo de orquestacion sigue sin conocer
  OPES.

Secuencia recomendada para temario Operario si se quiere fijar explicitamente:

```bash
export ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE=plan_temario,update_topic_registry,research_exam_precedents,draft_content_block,generate_visual_asset,generate_question_bank,review_legal,review_pedagogical,review_quality,review_codex,review_gemini,review_claude,review_pair_codex_gemini,review_pair_codex_claude,review_pair_gemini_claude,review_director_consolidation,validate_topic,assemble_topic,generate_audio_asset,generate_tutor_assets,generate_learning_games,generate_html_site,generate_help_manual_assets,finalize_temario_package
```

Arranque autonomo acotado hasta cierre:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_BASE_URL=http://127.0.0.1:<puerto-orquesta> \
ORQUESTA_OPES_BRIDGE_CONFIRM=1 \
ORQUESTA_OPES_BRIDGE_LIMIT=3 \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=<program_id> \
ORQUESTA_OPES_BRIDGE_CORRELATION_ID=<correlation_id> \
ORQUESTA_OPES_BRIDGE_INPUT_LEDGER_PATH=<ledger-temario>.json \
ORQUESTA_OPES_BRIDGE_MAX_TICKS=1000 \
ORQUESTA_OPES_BRIDGE_INTERVAL_SECONDS=5 \
go run ./cmd/orquesta-server opes-temario-cycle
```

Arranque paralelo objetivo para continuar Operario cuando OPES temporal este
aislado y la cuota Codex este confirmada:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_OPES_BRIDGE_ENABLED=1 \
ORQUESTA_OPES_BRIDGE_CONFIRM=1 \
ORQUESTA_OPES_BRIDGE_LIMIT=10 \
ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE=update_topic_registry,research_exam_precedents,draft_content_block,generate_visual_asset,generate_question_bank,review_legal,review_pedagogical,review_quality,review_codex,review_gemini,review_claude,review_pair_codex_gemini,review_pair_codex_claude,review_pair_gemini_claude,review_director_consolidation,validate_topic,assemble_topic,generate_audio_asset,generate_tutor_assets,generate_learning_games,generate_html_site,generate_help_manual_assets,finalize_temario_package \
ORQUESTA_OPES_BRIDGE_INITIAL_DELAY_SECONDS=5 \
ORQUESTA_OPES_BRIDGE_INTERVAL_SECONDS=60 \
ORQUESTA_SERVER_MAX_RUNS_PER_TICK=10 \
ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK=10 \
ORQUESTA_CODEX_MAX_BATCH_READY=10 \
ORQUESTA_CODEX_MAX_CONCURRENCY=10 \
ORQUESTA_CODEX_PROJECT_WORKDIR=/tmp/opes-salidas/operario-20260518/project \
ORQUESTA_CODEX_RUNTIME_WORKDIR=/tmp/opes-salidas/operario-20260518/runtime \
ORQUESTA_CODEX_MODEL=gpt-5.5 \
ORQUESTA_CODEX_REASONING_EFFORT=xhigh \
go run ./cmd/orquesta-server run
```

Para una prueba acotada, anadir `ORQUESTA_OPES_BRIDGE_MAX_TICKS=<n>`. No mezclar
`ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE` con `ORQUESTA_OPES_BRIDGE_JOB_TYPE` ni
con `ORQUESTA_OPES_BRIDGE_JOB_REF`: la secuencia automatiza fases completas por
tipo, no un job individual.

Limitacion operativa actual: OPES no ofrece filtro por `program_id` en
`GET /api/jobs`. Usar OPES temporal o una cola acotada para smokes; si hay jobs
fallidos historicos del mismo tipo, revisarlos desde OPES antes de dar por
cerrado el temario.

Pruebas offline focales antes de tocar OPES real:

```bash
go test -count=1 ./modulos/orquesta-opes-bridge
go test -count=1 ./cmd/orquesta-server -run 'TestOPESBridgeLoop|TestRunOPESDrainOnceV0'
bash -n scripts/smoke_opes_derivatives_rest.sh
bash -n scripts/smoke_opes_plan_temario_operadores.sh
ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER=1 \
  scripts/smoke_opes_derivatives_rest.sh
scripts/smoke_opes_consumer_isolated.sh
ORQUESTA_OPES_PLAN_TEMARIO_FAKE_SERVER=1 \
  scripts/smoke_opes_plan_temario_operadores.sh
ORQUESTA_OPES_PLAN_TEMARIO_FAKE_SERVER=1 \
ORQUESTA_OPES_PLAN_TEMARIO_SMOKE_MODE=drain-once \
ORQUESTA_OPES_PLAN_TEMARIO_EXECUTE=1 \
ORQUESTA_OPES_BRIDGE_WAIT_RESIDENT_SECONDS=1 \
  scripts/smoke_opes_plan_temario_operadores.sh
```

Estas pruebas no ejecutan Codex ni llaman a OPES real. El fake REST de
derivados permite una lectura seca de la fase pendiente y el wrapper
`smoke_opes_consumer_isolated.sh` recorre la secuencia completa hasta el ultimo
tipo configurado, hoy `generate_help_manual_assets`, contra un fake HTTP local,
supervisando cada `run_ref` sin Codex ni OPES real. El fake rechaza consultas
sin `job_type`, `status=pending`,
`execution_mode=external` y `limit` esperado.
El fake de `plan_temario` cubre tambien `drain-once`: simula OPES y Orquesta
locales, aisla el ledger en `SMOKE_OUT_DIR` y permite comprobar
`supervision_status=started` por observacion pasiva de `/api/v0/director/stats`
sin llamar a `/api/v0/runs/supervise`.
El smoke real de derivados sigue siendo opt-in, contra instancia temporal, y
debe comprobar que OPES recibe artefactos validos y deduplica reintentos; en
particular, `assemble_topic` debe entregar `artifact_type=assembled_topic` y
`generate_audio_asset` debe entregar `artifact_type=audio_asset`. Para el flujo
completo, `research_exam_precedents` entrega `exam_research_report`,
`generate_question_bank` entrega `question_bank`, `generate_tutor_assets`
entrega `tutor_bot_package`, `generate_html_site` entrega `local_html_site` y
`generate_help_manual_assets` entrega `help_manual_package`
operativo en local con logos USO y aspecto USO/TCAE promocion interna.

Wrapper operador para derivados:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
scripts/smoke_opes_derivatives_rest.sh
```

`scripts/smoke_opes_derivatives_real.sh` queda como wrapper compatible para los
operadores y la matriz: acepta la guarda historica
`ORQUESTA_OPES_DERIVATIVES_SMOKE_CONFIRM=1`, la traduce a
`ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM=1` y delega en
`scripts/smoke_opes_derivatives_rest.sh`. La ruta canonica es el script REST,
porque contiene tambien el modo fake aislado y el loop `run-until-finalize`.

Ese modo es `dry-run-once`: consulta OPES temporal y muestra la primera fase
pendiente de la secuencia sin crear runs en Orquesta. Si el OPES temporal no es
local, anadir `ORQUESTA_OPES_ALLOW_NONLOCAL_TEMPORAL=1` solo tras comprobar que
no es productivo.

Para crear runs de la primera fase pendiente:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_BASE_URL=http://127.0.0.1:<puerto-orquesta> \
ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_DERIVATIVES_EXECUTE=1 \
ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=drain-once \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=<program_id-temporal> \
ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_CONFIRMED=1 \
ORQUESTA_OPES_BRIDGE_SCOPE_PROBE_OUTPUT=<ruta-scope-probe-json-temporal> \
ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY=available \
ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_EVIDENCE_REFS=evidence-ref-tts-temporal-001 \
ORQUESTA_CODEX_GOAL_BACKEND=app_server_stdio \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
scripts/smoke_opes_derivatives_rest.sh
```

Antes de crear runs reales, usar el mismo entorno con
`ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=preflight-only`. Ese modo no consulta
OPES ni Orquesta; valida que no se ha confirmado productivo, que no hay
`ORQUESTA_OPES_BRIDGE_ALLOW_UNFILTERED`, que `LIMIT=1`, que el scope operativo
esta acotado y que la Orquesta temporal trabaja en modo goal-first. Si solo se
declara `ORQUESTA_OPES_BRIDGE_PROGRAM_ID`, anadir
`ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_CONFIRMED=1` solo despues de comprobar que
el OPES temporal filtra realmente por `program_id`; como alternativa usar
`ORQUESTA_OPES_BRIDGE_CORRELATION_ID`,
`ORQUESTA_OPES_BRIDGE_TOPIC_ID` o
`ORQUESTA_OPES_BRIDGE_DEDICATED_TEMPORAL_QUEUE=1`.
Si se usa `program_id` como unico scope, el preflight real exige ademas
`ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_EVIDENCE_REF` durable real o un JSON
`opes_derivatives_scope_probe.json` generado por `scope-probe`, con
`scope_probe_status=ok`, `job_type`, `seen > 0` y negative check de
`program_id`. Para operadores es preferible pasar ese JSON con
`ORQUESTA_OPES_BRIDGE_SCOPE_PROBE_OUTPUT=<ruta-scope-probe-json-temporal>`; no
inventar el ref de evidencia.
Si el target es `drain-once`, `run-until-finalize` o `run-until-final` y la
secuencia incluye `generate_audio_asset`, declarar
`ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY=available`; sin esa capability
el wrapper bloquea la prueba y `opes-drain-once` no postea el job de audio.
En ejecuciones reales temporales, `available` debe venir acompanado de
`ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_EVIDENCE_REFS` con refs compactas del
runner/capacidad TTS temporal.

Para dejar avanzar la secuencia completa hasta que OPES deje de exponer el
ultimo tipo configurado pendiente despues de supervisar su run, usar el modo
`run-until-finalize`. El alias historico `run-until-assemble` sigue aceptado
por compatibilidad.
Sigue siendo opt-in y temporal: crea runs fase a fase, supervisa cada `run_ref`
devuelto por Orquesta y repite la secuencia hasta observar que
`generate_html_site` y `generate_help_manual_assets` ya no quedan pendientes en
la secuencia vigente.

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_BASE_URL=http://127.0.0.1:<puerto-orquesta> \
ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_DERIVATIVES_EXECUTE=1 \
ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=run-until-finalize \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=<program_id-temporal> \
ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_CONFIRMED=1 \
ORQUESTA_OPES_BRIDGE_SCOPE_PROBE_OUTPUT=<ruta-scope-probe-json-temporal> \
ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY=available \
ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_EVIDENCE_REFS=evidence-ref-tts-temporal-001 \
ORQUESTA_CODEX_GOAL_BACKEND=app_server_stdio \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
ORQUESTA_OPES_BRIDGE_MAX_TICKS=20 \
scripts/smoke_opes_derivatives_rest.sh
```

Comando equivalente por compatibilidad historica:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_BASE_URL=http://127.0.0.1:<puerto-orquesta> \
ORQUESTA_OPES_DERIVATIVES_SMOKE_CONFIRM=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_DERIVATIVES_EXECUTE=1 \
ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=run-until-finalize \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=<program_id-temporal> \
ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_CONFIRMED=1 \
ORQUESTA_OPES_BRIDGE_SCOPE_PROBE_OUTPUT=<ruta-scope-probe-json-temporal> \
ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY=available \
ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_EVIDENCE_REFS=evidence-ref-tts-temporal-001 \
ORQUESTA_CODEX_GOAL_BACKEND=app_server_stdio \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
ORQUESTA_OPES_BRIDGE_MAX_TICKS=20 \
scripts/smoke_opes_derivatives_real.sh
```

El wrapper rechaza `ORQUESTA_OPES_BRIDGE_JOB_TYPE` y
`ORQUESTA_OPES_BRIDGE_JOB_REF` para derivados porque la ruta segura aqui es la
secuencia completa por fases. Cada ejecucion real debe revisar el JSON de salida
en `/tmp/opes-salidas/derivatives-rest-<smoke_id>/` antes de repetir o subir el
limite. El fake offline cubre tambien `run-until-finalize` y comprueba que
`research_exam_precedents`, `assemble_topic`, `generate_question_bank`,
`generate_audio_asset`, `generate_tutor_assets`, `generate_html_site` y
`generate_help_manual_assets` se
mapean a sus artefactos esperados sin meter OPES en el nucleo.

Estado de cierre OPES real: la composicion Orquesta ya declara tests de dominio
por job OPES, genera evidencia durable desde el ledger de `submit_artifact` y
la fuente de cierre del stack solo cierra tareas OPES con review aceptada,
`RequiredTestEvidenceV0` passed y receipt OPES aceptado. Desde el corte T18, el
ledger de entrega conserva tambien `domain_ref=opes`, `job_ref`,
`artifact_ref`, `artifact_type`, `complete_job` e idempotency key; el cierre no
acepta receipts que no empaten con el job externo, la delivery revisada y el
artefacto esperado (`assemble_topic -> assembled_topic`,
`generate_audio_asset -> audio_asset`, `generate_question_bank ->
question_bank`, `generate_tutor_assets -> tutor_bot_package`,
`generate_html_site -> local_html_site`,
`generate_help_manual_assets -> help_manual_package`). El conector REST trata como invalido
un receipt OPES con `job_id`, `artifact.type` o `job.status` incoherentes. El
smoke real completo de derivados sigue exigiendo OPES temporal vivo,
cuota/modelo confirmados y evidencia de que cada derivado fue aceptado por OPES
con refs causales suficientes; no se declara cerrado desde dry-run.

Nota goal-first 2026-06-27: si el submit a Orquesta devuelve
`route_policy=goal_first`, el ledger de input externo conserva `goal_ref`,
`external_goal_ref` y `next_actions`. En `already_submitted` con metadata
goal-first, el bridge no debe invocar `/api/v0/runs/supervise`; usa
`/api/v0/apps/director/goal/observe` cuando
`ORQUESTA_OPES_BRIDGE_SUPERVISE_SUBMITTED=1` o cuando se configure espera
residente. Si el ledger es antiguo y no trae esa metadata, una llamada acotada a
`/api/v0/runs/supervise` puede devolver `observe_goal`; en ese caso el bridge
persiste la metadata goal-first recuperada y cambia a observacion de Goal sin
drenar el loop legacy. Sin supervision forzada, el resultado queda como
`goal_first_observe_pending` porque Codex Goal mantiene su propio loop.

Bloqueo verificable T12 si no hay entorno temporal: ejecutar primero el smoke
fake aislado y despues repetir el comando `run-until-finalize` anterior cuando
existan OPES temporal, servidor Orquesta temporal y cuota/modelo confirmados.
La ausencia de `ORQUESTA_OPES_BASE_URL`, `ORQUESTA_BASE_URL`,
`ORQUESTA_OPES_TEMPORAL_CONFIRM=1` u `ORQUESTA_OPES_DERIVATIVES_EXECUTE=1`
debe tratarse como bloqueo operativo, no como fallo del conector.

Wrapper operador para repetir el plan exacto `plan_temario` de Operario sin
drenar colas amplias:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_OPES_PLAN_TEMARIO_SMOKE_CONFIRM=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_BRIDGE_JOB_REF=<job-ref-plan-temario-operario> \
scripts/smoke_opes_plan_temario_operadores.sh
```

Para crear la run, anadir `ORQUESTA_BASE_URL`, cambiar a
`ORQUESTA_OPES_PLAN_TEMARIO_SMOKE_MODE=drain-once` y declarar
`ORQUESTA_OPES_PLAN_TEMARIO_EXECUTE=1`. El wrapper fija
`JOB_TYPE=plan_temario`, exige `JOB_REF` exacto y rechaza secuencias.

## Criterio editorial para Operario

- temas comunes 1 y 2: derivar desde maestros superiores ya trabajados si
  existen (`constitucion_espanola`, `regimen_local_haciendas`,
  `administracion_publica_organizacion`, `empleo_publico`) y recortar a AP;
- temas especificos 3 a 10: marcar `creacion_directa_nivel` salvo que OPES
  aporte un maestro superior sectorial equivalente;
- no marcar `ready_profesional` sin revision especialista/editorial real;
- no borrar, mover ni regenerar carpetas de temarios existentes.
