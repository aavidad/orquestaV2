# Contratos: orquesta-document-plan-expander

## `ExpandDomainDocumentPlanV0`

Entrada: `DomainDocumentPlanExpansionRequestV0`.

Campos:

- `plan`: `DomainDocumentPlanV0` producido por Orquesta;
- `correlation_id`: correlacion opcional para todos los jobs derivados;
- `requested_by`: actor opcional, por defecto `orquesta-document-plan-expander`;
- `interface_refs`, `input_fields`, `external_refs` y `evidence_refs`: contexto
  compacto que se copia a cada job derivado.

Salida: `DomainDocumentPlanExpansionResultV0`.

- `jobs`: requests `DomainWorkJobRequestV0` normalizados y validados;
- `issues`: errores de validacion del plan o de cualquier job derivado.

Regla de error: si hay issues, el resultado no devuelve jobs parciales.

## Jobs derivados

- Cada seccion produce un job con el `work_kind` de la seccion, normalmente
  `draft_content_block`, y `expected_artifact_type=content_block`.
- Cada visual produce un job con el `work_kind` del visual, normalmente
  `generate_visual_asset`, y `expected_artifact_type=visual_asset`.
- Cada review produce un job con su `work_kind`; `review_*` y `validate_topic`
  esperan `block_revision`.
- `assemble_topic` espera `assembled_topic`.
- Si el plan declara `generate_audio_asset` o alias normalizado, espera
  `audio_asset` y solo transporta refs opacas del tema ensamblado o paquete
  final. El motor de sintesis de audio pertenece al adaptador de la app externa,
  no al expander.

El expander solo materializa lo declarado por el plan. No crea taxonomia de
ninguna app externa ni decide si faltan secciones, revisiones o entregables mas
alla de la validacion de `orquesta-domain-work`.

## `CreateDomainDocumentPlanDerivedJobsV0`

Caso de uso hexagonal para crear jobs derivados mediante puerto inyectado.

Entrada:

- `context.Context`;
- `DomainDocumentPlanExpansionRequestV0`;
- `DomainDocumentPlanDerivedJobsCreationPortsV0` con
  `DomainWorkJobCreatorPortV0`.

Flujo:

1. Normaliza y valida el plan con `ExpandDomainDocumentPlanV0`.
2. Si hay `issues`, devuelve resultado `invalid` sin llamar al puerto.
3. Si el puerto falta, devuelve `domain_document_plan_job_creator_required`.
4. Si la expansion es valida, invoca `CreateDomainWorkJobV0` una vez por cada
   `DomainWorkJobRequestV0`.
5. `accepted` con `job_ref` no vacio se trata como exito.
6. `invalid`, status desconocido o `accepted` sin `job_ref` se tratan como
   resultado `invalid` y detienen la secuencia.
7. Un error Go del puerto se devuelve como error operativo, conservando el
   progreso ya reflejado en el resultado.

Reglas de frontera:

- el caso de uso no debe importar DB, cola, REST, apps concretas, Codex, runtime,
  filesystem productivo ni proveedor/modelo;
- la implementacion concreta de `DomainWorkJobCreatorPortV0` vive fuera de este
  paquete y decide si crea jobs en DB, cola, app propietaria u otro mecanismo;
- ejemplos externos de implementacion del puerto son
  `orquesta-domain-work-memory` y `orquesta-domain-work-file`; este paquete no
  importa esos adaptadores;
- el caso de uso no reinterpreta el plan ni inventa taxonomia: crea exactamente
  los jobs derivados que devuelve el expander;
- si falla la expansion, no hay efectos externos; si falla la creacion de un
  job tras haber creado otros, el reintento debe usar las mismas
  `IdempotencyKey`.
