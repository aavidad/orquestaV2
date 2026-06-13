# orquesta-opes-director

Adaptador OPES para producir trabajos causales desde artefactos de dominio ya
recibidos por Orquesta.

Responsabilidad:

- observar registros de entrega `DomainWork` de la composicion;
- expandir `document_plan` aceptados mediante `orquesta-document-plan-expander`;
- convertir `followup_refs`, `rework_refs`, faltas obligatorias y rechazos en
  nuevos `DomainWorkJobRequestV0`;
- evitar cierres falsos de paquetes OPES con `pendiente_continuar`;
- mantener idempotencia estable sin tocar core ni internals de OPES.

Fuera de alcance:

- ejecutar agentes;
- leer colas OPES amplias;
- publicar en produccion;
- decidir por heuristicas de texto libre.
