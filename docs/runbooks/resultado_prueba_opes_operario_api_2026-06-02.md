# Resultado prueba OPES Operario por API 2026-06-02

## Alcance

Prueba externa por API, sin acceso a DB ni ficheros internos de OPES. El
objetivo era comprobar si Orquesta y OPES podian aceptar el `document_plan`
real del temario Operario AP y preparar los derivados del flujo completo.

## Entrada usada

- Artefacto Orquesta real:
  `/home/alberto/Trabajo/orquesta/external/opes/plan_temario/115a8e68b27a03ad59c6caf61356fab6`
- Job real previo en OPES activo `18086`:
  `115a8e68b27a03ad59c6caf61356fab6`
- Programa Operario AP:
  `6e6cc9dd7275f6c32d4f9d044a12d98d`

## Diagnostico

Se confirmaron dos problemas distintos:

- Orquesta generaba un `document_plan` util para Operario AP, con 10 temas y
  flujo completo: investigacion de examenes, redaccion, visuales, banco de
  tests, revisiones, validacion, ensamblado, audios, tutor/bots y HTML local.
- Actualizacion posterior 2026-06-02: el flujo vigente anade tambien
  `generate_help_manual_assets -> help_manual_package` para manuales graficos
  de ayuda USO derivados del HTML local. Esta prueba se ejecuto antes de
  incorporar ese paso.
- OPES activo en `18086` estaba ejecutando un binario anterior y rechazaba ese
  plan con HTTP 400 `invalid job artifact` porque su contrato no aceptaba todos
  los `work_kind` y `artifact_type` del flujo completo.
- Orquesta ocultaba ese HTTP 400 como `domain_work_port_no_disponible`, lo que
  hacia mirar al puerto/adaptador en vez del contrato OPES.

No fue un fallo de contenido del plan ni un caso para filtrar agentes. Fue una
brecha de contrato entre el plan que Orquesta ya podia producir y lo que OPES
aceptaba.

## Cambios aplicados

En OPES `opes-uso`:

- Se anadieron tipos documentales para:
  `research_exam_precedents`, `generate_question_bank`,
  `generate_html_site`, `generate_tutor_assets`.
- `review_textual` queda aceptado tambien como trabajo documental.
- `document_plan` crea derivados con artefactos esperados:
  `exam_research_report`, `question_bank`, `local_html_site`,
  `tutor_bot_package`, `assembled_topic`.
- `submit_job_artifact` acepta esos artefactos para su job correcto y los
  rechaza si llegan al job equivocado.

En Orquesta:

- El conector OPES conserva el status HTTP en errores no retryables:
  `opes_http_status_<status>`.
- En `submit_artifact`, un HTTP 4xx de OPES se devuelve como
  `DomainWorkArtifactReceiptV0` invalido con issue
  `opes_http_status_<status>` y `field=artifact_submitter`.
- MCP propaga ese receipt invalido como error de dominio, no como
  `domain_work_port_no_disponible`.

## Evidencia API

Se levanto una instancia OPES temporal en `127.0.0.1:18120` con SQLite temporal
solo para contrato. No se uso DB productiva ni acceso interno.

Secuencia API:

- `POST /api/jobs` con `job_type=plan_temario`: aceptado.
- `POST /api/jobs/{id}/artifacts` con el `payload_json` del artefacto real:
  HTTP 201.
- Job de plan completado.
- Derivados creados: 28.

Distribucion de derivados creados:

```text
draft_content_block: 10
generate_visual_asset: 8
research_exam_precedents: 1
generate_question_bank: 1
review_legal: 1
review_pedagogical: 1
review_quality: 1
validate_topic: 1
assemble_topic: 1
generate_audio_asset: 1
generate_tutor_assets: 1
generate_html_site: 1
```

## Pruebas

Orquesta:

```sh
go test -count=1 ./...
git diff --check
```

OPES:

```sh
go test -count=1 ./...
git diff --check
```

Ambas baterias pasaron el 2026-06-02.

## Estado operativo

- Orquesta temporal de prueba se apago por API y despues se interrumpio la
  sesion porque el proceso seguia escuchando.
- OPES temporal `18120` se apago.
- OPES activo `18086` se reinicio con el codigo parcheado el 2026-06-02,
  conservando la instancia SQLite temporal
  `/tmp/opes-operario-api-u1UPV2/opes_operario_api.db`.
- El job real `115a8e68b27a03ad59c6caf61356fab6` recibio el `document_plan`
  por API con HTTP 201, artefacto `c5069948ddb02a262cb5bfaa99cea72e`, estado
  `completed` y 28 derivados pendientes visibles por `GET /api/jobs`.
- El artefacto existente es un plan de temario, no los manuales finales. Los
  manuales de Operario aun no estan generados: el siguiente paso real es drenar
  esos 28 derivados hasta `generate_html_site`.
