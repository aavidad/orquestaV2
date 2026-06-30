# Tareas: orquesta-domain-work

## Hecho v0

- Crear mini-proyecto `orquesta-domain-work`.
- Definir `DomainWorkJobRequestV0`.
- Definir `DomainWorkArtifactSubmissionV0`.
- Definir `DomainWorkFieldV0` para entradas y payloads normalizados.
- Definir puertos `DomainWorkJobCreatorPortV0` y
  `DomainWorkArtifactSubmitterPortV0`.
- Definir `DomainDocumentPlanV0` con aliases `PlanTemaV0` y `PlanTemarioV0`
  para planes documentales producidos por director.
- Validar refs compactas e idempotencia minima.
- Validar planes documentales con secciones, visuales, revisiones y
  entregables.
- Definir frontera neutral de capacidades externas para jobs de dominio:
  `audio_asset` exige `speech_synthesis`, la composicion declara disponibilidad
  por puerto y el contrato bloquea con razon operativa si falta red, tool path,
  cuota, heartbeat/progreso o timeout de proveedor.
- Documentar frontera con OPES/programacion/otras apps.
- Crear primer conector REST OPES de jobs/artefactos en
  `orquesta-opes-connector`.

## Siguientes Microtareas

- Adaptar progresivamente `app-change` o el modulo de programacion para emitir
  `DomainWorkJobRequestV0` cuando reciba `external_work`.
- Cablear `orquesta-opes-connector` desde stack/MCP/web por configuracion
  explicita.
- Anadir MCP OPES si aporta ventaja frente a REST.
- Promover contratos si otro dominio los consume.

## Corte OPES

- Crear `modulos/orquesta-opes-connector` como miniproyecto con contrato y
  cliente REST opt-in.
- Mantener OPES fuera de `orquesta-domain-work`: el contrato generico sigue sin
  semantica editorial ni dependencias REST/MCP concretas.
