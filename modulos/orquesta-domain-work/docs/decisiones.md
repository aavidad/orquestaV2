# Decisiones: orquesta-domain-work

```text
Fecha: 2026-05-13
Decision: Crear `orquesta-domain-work` como contrato neutro para apps externas.
Motivo: Orquesta debe ser nucleo de orquestacion de agentes. Programacion, OPES
u otras apps deben usar la orquestacion por contratos sin integrar su nucleo ni
gestionar agentes.
Alternativas: mover codigo actual de programacion; crear conectores OPES
directos sin contrato comun; hacer que cada app gestione sus agentes.
Impacto: se anade un contrato puro de jobs y artefactos de dominio. No cambia
el flujo actual de programacion ni se introduce REST/MCP/DB.
Estado: aceptada_local
```

```text
Fecha: 2026-05-13
Decision: Preparar OPES en un modulo documental separado llamado
`orquesta-opes-connector`.
Motivo: OPES necesita un conector REST/MCP opt-in, pero el contrato generico no
debe absorber semantica editorial ni detalles de transporte.
Alternativas: meter OPES en `orquesta-domain-work`; crear cliente real ya;
dejar solo una nota global en `docs/`.
Impacto: se documenta la frontera del conector futuro sin tocar codigo
productivo ni OPES.
Estado: aceptada_local
```

```text
Fecha: 2026-05-13
Decision: El modulo transporta refs compactas, metadatos de correlacion y
campos de dominio normalizados, pero no JSON libre sin contrato.
Motivo: OPES necesita campos como `program_id`, `topic_id`, `level`,
`language_code`, `title` o `body` para crear jobs y artefactos. Pasarlos como
campos normalizados permite funcionar sin meter tipos OPES ni rutas internas en
el nucleo.
Alternativas: incluir `payload_json` arbitrario; pasar solo refs y obligar al
conector a leer internals; usar tipos propios de OPES.
Impacto: `input_fields` y `payload_fields` son datos de dominio declarados por
adaptadores superiores. Los conectores materializan JSON REST/MCP fuera de este
modulo y mantienen evidencia por refs.
Estado: aceptada_local
```
