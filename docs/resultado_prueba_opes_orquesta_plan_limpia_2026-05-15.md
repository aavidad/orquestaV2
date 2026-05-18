# Resultado prueba limpia OPES-Orquesta plan_tema 2026-05-15

## Alcance

Prueba real lanzada desde OPES hacia Orquesta con un job externo `plan_tema`.

Objetivo: validar que Orquesta puede recibir un trabajo documental neutral, arrancar un agente Codex real (`gpt-5.5`, `xhigh`), recuperar o entregar el artefacto `document_plan`, canonicalizarlo y devolverlo a OPES sin duplicar artefactos.

Run:

- Directorio: `/home/alberto/Trabajo/OPES/opes-uso/runs/orquesta-plan-clean-20260514T222606Z`
- OPES temporal: `127.0.0.1:19080`
- Orquesta temporal: `127.0.0.1:19081`
- Job OPES inicial: `5c8e19d826ee9fddaa840a92bd5ef465`

## Resultado

El agente real produjo un plan amplio y util, pero fallo el ACK visible/controlado:

- Creo `external/opes/plan_tema/5c8e19d826ee9fddaa840a92bd5ef465`.
- Termino con `ACK ... failed`.
- No escribio `agent_ack.json`.
- Gasto unos `59k` tokens, demasiado para una planificacion de este tamano.

El primer intento no se entrego a OPES porque el `document_plan` traia `work_kind` semanticos no ejecutables:

- `planificacion_documental`
- `redaccion_tema`
- `visual_asset_plan`
- `revision_pedagogica`
- `ensamblado_y_exportacion`

La solucion correcta no era tocar el JSON a mano, sino endurecer el contrato/canonicalizacion para que Orquesta convierta alias humanos a tipos ejecutables antes de entregar.

## Cambios Aplicados

- `orquesta-domain-work` normaliza alias de `work_kind` en `document_plan`.
- Se fuerza el `work_kind` raiz al tipo de job recibido cuando el agente usa una etiqueta semantica.
- Se convierten secciones a `draft_content_block`.
- Se convierten visuales a `generate_visual_asset`.
- Se convierten revisiones conocidas a `review_legal`, `review_pedagogical`, `review_quality`, `validate_topic` o `assemble_topic`.
- `orquesta-opes-bridge` inyecta `allowed_document_plan_work_kinds` en el contexto del agente.
- El prompt Codex prohibe imprimir diffs/artefactos largos y refuerza escribir `agent_ack.json` antes del ACK final.

## Validacion Real

Tras reiniciar Orquesta con el codigo corregido sobre el mismo run:

- OPES recibio exactamente `1` artefacto `document_plan`.
- El job `plan_tema` paso a `completed`.
- OPES creo `21` jobs derivados:
  - `10` `draft_content_block`
  - `5` `generate_visual_asset`
  - `1` `validate_topic`
  - `1` `review_legal`
  - `1` `review_pedagogical`
  - `2` `review_quality`
  - `1` `assemble_topic`

No hubo duplicado de `document_plan`.

Nota operativa: al arrancar el bridge sin `ORQUESTA_OPES_BRIDGE_JOB_TYPE=plan_tema`, tambien envio los primeros 10 jobs derivados a Orquesta. Se corto el servidor temporal antes de lanzar agentes derivados. Para pruebas focales de planificacion debe fijarse `ORQUESTA_OPES_BRIDGE_JOB_TYPE=plan_tema`.

## Tests

Pasados:

- `go test -count=1 ./modulos/orquesta-domain-work`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCanonicalDomainWorkDeliveryPayloadBodyV0NormalizaPlanOPESReal|TestDefaultDomainWorkArtifactSubmissionBuilderV0CanonicalizaDocumentPlanAliasesOPES|TestDefaultDomainWorkArtifactSubmissionBuilderV0IdempotenciaEstablePorJobYArtefacto'`
- `go test -count=1 ./modulos/orquesta-opes-bridge`
- `go test -count=1 ./modulos/orquesta-runtime-codex -run 'TestCodexExecResolverV0PromptUsaControlFilesDelRuntime'`
- `go test -count=1 ./...`

## Decision

Queda validado que Orquesta puede servir como nucleo neutral para trabajos documentales OPES:

- OPES decide el dominio y publica jobs.
- Orquesta arranca y gobierna agentes.
- Orquesta devuelve artefactos por contrato.
- OPES materializa trabajos derivados.

La siguiente mejora no es tocar el artefacto producido, sino controlar mejor el lanzamiento de jobs derivados en pruebas y reforzar OPES para que planifique el arranque completo de temas cuando aun no existen bloques.
