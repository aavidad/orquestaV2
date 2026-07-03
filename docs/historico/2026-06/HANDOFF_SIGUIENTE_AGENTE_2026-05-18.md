# Handoff para siguiente agente - Orquesta / OPES Operario AP

Fecha de corte: 2026-05-18, sesion cerrada por el usuario antes de terminar.

Este fichero es el punto de entrada para continuar. Leerlo antes de tocar
codigo, ledgers, runtime o salidas OPES.

## Mandato del usuario

- La calidad no debe bajar ni un apice.
- Orquesta es generico y hexagonal; OPES es solo consumidor/conector.
- No borrar ni mover nada de OPES sin revisar. Hay temas ya generados.
- No usar SQLite para nuevas ejecuciones. Postgres es obligatorio.
- Las salidas nuevas van en `opes-salidas/`, no en `opes-uso/`.
- Para OPES/temarios se usa Codex `xhigh`.
- No ser fragil con nombres: aceptar aliases de `artifact_type`, campos y refs si
  el contenido es materializable. No tirar un trabajo entero por un nombre.
- El objetivo final es que el Director de Orquesta arranque agentes Codex,
  los mantenga vivos con bucles/supervision y paralelice temas/capitulos.
- Para OPES, la regla editorial sigue siendo: si existe tema maestro A1/A2/A1,
  derivar hacia abajo; si no existe, crear AP directamente con trazabilidad
  `creacion_directa_nivel`. No marcar `ready_profesional` sin revision real.

## Rutas vivas

Repo Orquesta:

```bash
/home/alberto/Trabajo/orquesta
```

Repo OPES:

```bash
/home/alberto/Trabajo/OPES/opes-uso
```

Run root Operario AP:

```bash
/home/alberto/Trabajo/OPES/opes-salidas/codex_directo/operario/AP/orquesta_full_2026-05-18
```

ORQ_ROOT:

```bash
/home/alberto/Trabajo/OPES/opes-salidas/codex_directo/operario/AP/orquesta_full_2026-05-18/orquesta
```

Programa OPES importado:

```text
program_id = 8d8ccd3720e1aa0e420923fc71818fa6
fuente = /home/alberto/Trabajo/OPES/OPES/administracion-especial/Operario/Operario.txt
```

## Servicios y procesos al corte

OPES API:

```text
http://127.0.0.1:18086
OPES_PERSISTENCE_PROVIDER=postgres
DSN=postgres://uso:uso@localhost:15433/opes_operario_orquesta_20260518?sslmode=disable
OPES_DOCUMENT_ROOT=/home/alberto/Trabajo/OPES/opes-salidas/codex_directo/operario/AP/orquesta_full_2026-05-18/api_exports
```

Postgres:

```text
container = opes-operario-postgres-20260518
port = localhost:15433
volume = opes_operario_pg_20260518
```

Orquesta server:

```text
addr = http://127.0.0.1:18789
pid observado = 3055016
bin = $ORQ_ROOT/bin/orquesta-server
state = $ORQ_ROOT/state
project = $ORQ_ROOT/project
runtime = $ORQ_ROOT/runtime
modelo configurado = gpt-5.5
reasoning real en launches = xhigh
bridge OPES habilitado
```

Al corte no habia procesos Codex hijos activos para esta corrida; solo el server
Orquesta seguia vivo imprimiendo `opes_bridge_sequence_loop`.

Comprobar estado:

```bash
ps -eo pid,ppid,etime,stat,cmd | rg 'orquesta_full_2026-05-18|orquesta-server|codex --ask-for-approval never exec' | rg -v 'rg '
curl -fsS http://127.0.0.1:18086/api/health | jq .
curl -fsS http://127.0.0.1:18789/api/health | jq .
```

## Estado OPES al corte

Jobs externos `draft_content_block`:

```text
Tema 1  bbafac3aebf370e7d691d728e3652be3  pending
Tema 2  bd6a64c4c492cfe5a06abbbd65acdca8  pending
Tema 3  5460ae29c4ede762e8462650cf9941c1  pending
Tema 4  085008c256ff76e746b4e9ed1f13c78e  pending
Tema 5  f60fb6567f75ab60a85740b42279d020  pending
Tema 6  9a22253c9e092950f558137d5409b808  pending
Tema 7  3980c4070cd4f7e4342a8a504a8aa838  pending
Tema 8  9ae546da7ab27a718e7b48043a011ca9  completed
Tema 9  a84cc9d1841a7e50d769316f1cb24786  completed
Tema 10 21c04dfcbd06f3b39feed3a64e3a8677  pending
Manual d67f69946187f3694fa9a3c334626105 cancelled
```

Consultar:

```bash
curl -fsS 'http://127.0.0.1:18086/api/jobs?execution_mode=external&job_type=draft_content_block&limit=20' \
  | jq -r '(["order","job_id","status","last_error"] | @tsv), (.[] | [(.external_refs.official_order // ""), .id, .status, (.last_error // "")] | @tsv)'
```

## Estado Orquesta al corte

Hay 10 artifact files locales:

```bash
$ORQ_ROOT/project/external/opes/draft_content_block/*
```

Conteo observado:

```text
artifacts_local = 10
agent_ack.json = 7
domain-work-artifact-ledger records = 2
```

Ledger de entrada OPES -> Orquesta:

```bash
$ORQ_ROOT/state/external-bridge-input-ledger.json
```

Contiene los 10 jobs como `submitted`. No significa que OPES haya recibido el
artifact final; solo significa que esos jobs ya fueron convertidos en runs de
Orquesta.

Ledger de salida artifact -> OPES:

```bash
$ORQ_ROOT/state/domain-work-artifact-ledger.json
```

Solo tiene 2 registros, los Temas 8 y 9:

```text
9ae546da7ab27a718e7b48043a011ca9 -> receipt a3e80f770f6f2e5c322a9be1444eee06
a84cc9d1841a7e50d769316f1cb24786 -> receipt 8ef4f067a04294a952e09532d1f11393
```

No borrar ninguno de esos ledgers.

## Sintoma principal

Orquesta genero los 10 contenidos locales, pero OPES solo recibio 2.

El bridge OPES imprime que los 8 pendientes estan `already_submitted` porque el
input ledger ya los envio a Orquesta. Eso es correcto y no debe "arreglarse"
borrando el ledger.

El bloqueo esta en el paso de submit idempotente de artifact local a OPES.

Prueba observada:

```bash
curl -sS -i 'http://127.0.0.1:18789/api/v0/runs/supervise' \
  -X POST \
  -H 'Content-Type: application/json' \
  -d '{"run_ref":"run-external-work-opes-21c04dfcbd06f3b39feed3a64e3a8677-opes-job-21c04dfcbd06f3b39feed3a64e3a8677","max_ticks":8,"max_bursts":16,"max_steps_per_burst":8,"max_dispatches_per_wait":8,"max_commands":20,"max_outbox_per_cycle":8,"max_external_waits":2,"allow_repeated_runs":true}'
```

Respuesta:

```json
{"estado":"error","run_ref":"run-external-work-opes-21c04dfcbd06f3b39feed3a64e3a8677-opes-job-21c04dfcbd06f3b39feed3a64e3a8677","last":{},"errores_publicos":[{"code":"run_supervisor_http_error","field":"executor","message":"run_supervisor_error"}]}
```

## Causa probable del 500 / no submit

El endpoint OPES `POST /api/jobs/{id}/artifacts` para `content_block` espera:

```json
{
  "artifact_type": "content_block",
  "payload_json": {
    "topic_id": "string",
    "chapter_id": "string",
    "block_type": "string",
    "type": "string",
    "title": "string",
    "markdown": "string",
    "body": "string",
    "language_code": "string",
    "source_refs": ["string"],
    "citations": [
      {"source_ref": "string", "locator": "string", "excerpt": "string", "note": "string"}
    ]
  },
  "complete_job": true
}
```

Los artifact files locales usan en varias citas objetos con `ref` en vez de
`source_ref`, por ejemplo Tema 10:

```json
{"ref":"BOE-A-1995-24292","title":"Ley 31/1995...","url":"...","locator":"..."}
```

OPES puede rechazar esas citas porque `source_ref` queda vacio. La normalizacion
de aliases ya compacta `source_refs`, pero hay que revisar/parchear tambien
`citations`: `ref` -> `source_ref` y, si procede, conservar `title/url/usage`
en `note` o `excerpt` sin romper el contrato.

Ficheros relevantes:

```text
modulos/orquesta-app-codex-stack/domain_work_delivery_builder_v0.go
modulos/orquesta-app-codex-stack/domain_work_delivery_aliases_v0.go
modulos/orquesta-app-codex-stack/domain_work_delivery_builder_v0_test.go
modulos/orquesta-app-codex-stack/domain_work_delivery_bridge_v0.go
modulos/orquesta-opes-connector/payload_v0.go
/home/alberto/Trabajo/OPES/opes-uso/internal/application/job/register_job_artifact.go
```

Tambien conviene mejorar temporalmente el error publico de
`run_supervisor_mcp_executor_v0.go` o loguear la causa real, porque ahora
convierte todo en `run_supervisor_error` y oculta el fallo de OPES/builder.

## Cambios ya hechos en esta sesion

Orquesta:

- Builder de entregas tolera nombres flexibles:
  - unwrap de `payload_json` aunque el envelope tenga otro `artifact_type`.
  - tipo esperado inferido desde `work_kind`.
  - aliases OPES para campos como `tema_id`, `id_capitulo`, `titulo`,
    `contenido`, `idioma`, `fuentes`.
  - `topic_summary` conserva markdown canonico.
- Bridge OPES: `ORQUESTA_OPES_BRIDGE_LIMIT` cuenta envios nuevos, no las
  primeras N filas escaneadas.
- Docs actualizadas sobre no descartar artifacts por nombres y no usar SQLite.

OPES:

- Postgres idempotency index corregido para jobs largos:
  `ux_generation_jobs_idempotency` usa `md5(payload_json)` en vez de indexar el
  JSON completo. Esto evita el error de row-size de btree.

Tests ejecutados y pasados durante la sesion:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestDefaultDomainWorkArtifactSubmissionBuilderV0(AceptaEnvelopeConNombreLibreOPES|NormalizaSourceRefsRicosContentBlockOPES|EntregaVisualAssetOPES)'
go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector
go test -count=1 ./cmd/orquesta-server -run 'TestRunOPESDrainOnceV0(EscaneaMasQueLimitYEnviaSiguientesNoEnviados|SecuenciaPasesNoAvanzaSiPrimerTipoYaEnviado|LedgerEvitaReenviarJobPendiente)'
OPES_POSTGRES_TEST_DSN='postgres://uso:uso@localhost:15433/opes_operario_orquesta_20260518?sslmode=disable' go test -count=1 ./internal/adapters/persistence/postgres -run 'TestPostgresJobRepositorySaveIdempotentMatrix'
```

## Lo que NO hay que hacer

- No borrar `$ORQ_ROOT/state/external-bridge-input-ledger.json`.
- No borrar `$ORQ_ROOT/state/domain-work-artifact-ledger.json`.
- No borrar `$ORQ_ROOT/project` ni `$ORQ_ROOT/runtime`.
- No regenerar los 10 temas como primera opcion: ya existen artifact files.
- No cambiar a SQLite.
- No meter logica OPES dentro del nucleo generico. La normalizacion pertenece al
  borde/adapter de domain work o conector.
- No marcar `ready_profesional` ni decir que el temario esta terminado: solo hay
  2 jobs completados en OPES y falta revision editorial.

## Siguiente paso recomendado

1. Parar o reiniciar Orquesta solo si hace falta, conservando el mismo ORQ_ROOT.
2. Escribir test que reproduzca artifact OPES con `citations` usando `ref` en
   vez de `source_ref`.
3. Parchear normalizacion de citas en el builder/conector para que OPES reciba
   `source_ref`.
4. Mejorar visibilidad del error en `runs/supervise` o logs.
5. Recompilar `$ORQ_ROOT/bin/orquesta-server`.
6. Reiniciar server con el mismo state/runtime/project.
7. Forzar supervise o subir prioridad de runs pendientes sin borrar ledgers.
8. Verificar que OPES pasa de 2 completados a 10 completados.
9. Exportar Markdown/HTML/validation/source-verification/blocks al run root.
10. Ejecutar revision de calidad OPES antes de declarar nada como final.

Comandos utiles para despues del parche:

```bash
RUN=/home/alberto/Trabajo/OPES/opes-salidas/codex_directo/operario/AP/orquesta_full_2026-05-18/orquesta
curl -fsS 'http://127.0.0.1:18086/api/jobs?execution_mode=external&job_type=draft_content_block&limit=20' | jq 'group_by(.status) | map({status:.[0].status,count:length})'
jq '.records|length' "$RUN/state/domain-work-artifact-ledger.json"
find "$RUN/project/external/opes/draft_content_block" -maxdepth 1 -type f | wc -l
find "$RUN/runtime" -path '*agent_ack.json' | wc -l
```

## Export cuando esten completados

Usar `PROGRAM_ID=8d8ccd3720e1aa0e420923fc71818fa6`.

Endpoints OPES utiles:

```text
GET  /api/programs/{program_id}/topic-links
GET  /api/topics/{topic_id}/markdown
POST /api/topics/{topic_id}/exports
GET  /api/topics/{topic_id}/validation
GET  /api/topics/{topic_id}/source-verification
GET  /api/topics/{topic_id}/blocks
GET  /api/jobs/{job_id}/artifacts
```

No usar `scripts/export_temario_ordenado.py` sin revisar: tiene logica historica
de Psicologia y espera 90 temas.

## Nota pendiente sobre ComfyUI/local

El usuario comento que en remoto no hay ComfyUI y que se podria crear un sistema
de tareas para generar assets localmente. No se implemento por el cierre de
sesion. Idea segura para futuro: un spool JSONL versionado, por ejemplo:

```text
external/local_tasks/comfyui_requests.jsonl
external/local_tasks/comfyui_results.jsonl
```

El agente remoto escribiria solicitudes con `task_id`, `prompt`,
`negative_prompt`, `size`, `output_contract` y `target_path`; la maquina local
las procesaria y devolveria resultados. Mantener esto como conector externo,
no dentro del nucleo de Orquesta.

## Estado de git

El worktree esta muy sucio por trabajo previo y de esta sesion. No revertir nada
sin revisar. Hay muchos ficheros modificados/no trackeados que no son todos de
esta intervencion. Antes de limpiar, hacer `git status --short` y revisar por
modulo.

