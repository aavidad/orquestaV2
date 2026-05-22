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

La ruta valida no es un script paralelo ni stdin:

```text
OPES plan_temario pending
  -> opes-drain-once / opes bridge loop
  -> /api/v0/external-work/run
  -> cola de runs de Orquesta
  -> supervisor residente o POST /api/v0/runs/supervise
  -> ContinueAppDirectorV0 / DrainRunV0
  -> outbox LaunchRuntimeAgent
  -> Codex xhigh
  -> DomainWork.submit_artifact(document_plan)
  -> OPES valida y crea derivados
```

Si `cmd/orquesta-server run` esta activo, el supervisor global del servidor ya
empuja la cola en cada tick. `POST /api/v0/runs/supervise` queda como empuje
manual/acotado para una run concreta, no como canal alternativo.

## Regla operativa

Para temarios se usa Codex con razonamiento `xhigh`.

El servidor actual aplica `ORQUESTA_CODEX_REASONING_EFFORT=xhigh` por defecto
si el entorno no lo declara. En smokes reales conviene declararlo igualmente
para que quede visible en la sesion:

```bash
export ORQUESTA_CODEX_MODEL=gpt-5.5
export ORQUESTA_CODEX_REASONING_EFFORT=xhigh
```

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
- `/api/v0/external-work/run` recibe el trabajo;
- `/api/v0/runs/supervise` empuja la run;
- el artefacto termina en `DomainWork.submit_artifact`;
- el paquete de agente y el wrapper Codex materializado usan `xhigh`.

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
- usar siempre filtro por tipo o por `job_ref` exacto, y limite bajo.

Dry-run:

Sustituir `job-ref-plan-temario-operario-001` por el `job_ref` real del job de
smoke; si no se conoce, eliminar esa variable y conservar
`ORQUESTA_OPES_BRIDGE_JOB_TYPE=plan_temario`.

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_OPES_BRIDGE_DRY_RUN=1 \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
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
go run ./cmd/orquesta-server opes-drain-once
```

Empuje acotado opcional si se quiere acelerar una run concreta sin esperar al
siguiente tick del servidor:

```bash
curl -sS -X POST http://127.0.0.1:<puerto-orquesta>/api/v0/runs/supervise \
  -H 'Content-Type: application/json' \
  -d '{
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
  `draft_content_block -> generate_visual_asset -> review_legal ->
  review_pedagogical -> review_quality -> validate_topic -> assemble_topic`,
  con ledger/idempotencia por fase y `assemble_topic` mapeado a
  `assembled_topic`;
- la secuencia vive en `cmd/orquesta-server`; el nucleo de orquestacion sigue
  sin conocer OPES.

Secuencia recomendada para temario Operario:

```bash
export ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE=draft_content_block,generate_visual_asset,review_legal,review_pedagogical,review_quality,validate_topic,assemble_topic
```

Arranque residente conservador:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_OPES_BRIDGE_ENABLED=1 \
ORQUESTA_OPES_BRIDGE_CONFIRM=1 \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE=draft_content_block,generate_visual_asset,review_legal,review_pedagogical,review_quality,validate_topic,assemble_topic \
ORQUESTA_OPES_BRIDGE_INITIAL_DELAY_SECONDS=5 \
ORQUESTA_OPES_BRIDGE_INTERVAL_SECONDS=60 \
ORQUESTA_SERVER_MAX_RUNS_PER_TICK=1 \
ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK=1 \
ORQUESTA_CODEX_MAX_BATCH_READY=1 \
ORQUESTA_CODEX_MAX_CONCURRENCY=1 \
ORQUESTA_CODEX_MODEL=gpt-5.5 \
ORQUESTA_CODEX_REASONING_EFFORT=xhigh \
go run ./cmd/orquesta-server run
```

Arranque paralelo objetivo para continuar Operario cuando OPES temporal este
aislado y la cuota Codex este confirmada:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_OPES_BRIDGE_ENABLED=1 \
ORQUESTA_OPES_BRIDGE_CONFIRM=1 \
ORQUESTA_OPES_BRIDGE_LIMIT=10 \
ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE=draft_content_block,generate_visual_asset,review_legal,review_pedagogical,review_quality,validate_topic,assemble_topic \
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
```

Estas pruebas no ejecutan Codex ni llaman a OPES real. El smoke real de
derivados sigue siendo opt-in, contra instancia temporal, y debe comprobar que
OPES recibe artefactos validos y deduplica reintentos; en particular,
`assemble_topic` debe entregar `artifact_type=assembled_topic`.

## Criterio editorial para Operario

- temas comunes 1 y 2: derivar desde maestros superiores ya trabajados si
  existen (`constitucion_espanola`, `regimen_local_haciendas`,
  `administracion_publica_organizacion`, `empleo_publico`) y recortar a AP;
- temas especificos 3 a 10: marcar `creacion_directa_nivel` salvo que OPES
  aporte un maestro superior sectorial equivalente;
- no marcar `ready_profesional` sin revision especialista/editorial real;
- no borrar, mover ni regenerar carpetas de temarios existentes.
