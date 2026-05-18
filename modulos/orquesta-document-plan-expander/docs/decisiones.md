# Decisiones: orquesta-document-plan-expander

```text
Fecha: 2026-05-17
Decision: Crear un expander neutral de `DomainDocumentPlanV0` a
`DomainWorkJobRequestV0[]`.
Motivo: Orquesta necesita una pieza reutilizable para materializar jobs
derivados de un plan documental sin depender de una app externa concreta ni del
stack Codex.
Alternativas: dejar la expansion en un adaptador concreto; meterla en
`orquesta-domain-work`; hacer que el stack Codex cree jobs; usar strings ad hoc
desde docs.
Impacto: el paquete solo importa `orquesta-domain-work`, valida el plan y cada
job derivado, conserva refs/idempotencia y no ejecuta efectos externos.
Estado: aceptada.
```

```text
Fecha: 2026-05-17
Decision: Implementar un caso de uso hexagonal para crear
jobs derivados mediante `DomainWorkJobCreatorPortV0`.
Motivo: el expander neutral ya existe, pero la creacion real de jobs debe pasar
por un puerto de `orquesta-domain-work` para servir a cualquier dominio sin
meter DB, cola, REST, Codex ni runtime en el paquete neutral.
Alternativas: hacer que el expander llame directamente a una cola; dejar la
creacion en un adaptador concreto; mover el caso de uso a
`orquesta-domain-work`; crear jobs desde el stack Codex.
Impacto: el paquete conserva la expansion pura y anade una aplicacion
hexagonal: expande, valida y llama al puerto. La implementacion concreta del
puerto vive en un adaptador/composicion exterior y aporta persistencia,
reintentos e idempotencia operativa.
Estado: aceptada.
```
