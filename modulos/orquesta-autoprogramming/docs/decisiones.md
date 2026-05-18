# Decisiones: orquesta-autoprogramming

## 2026-05-13: autoprogramacion sale del nucleo

Decision: `AutoprogrammingRequestV0`, agrupacion por area y review gate de
codigo viven en `orquesta-autoprogramming`.

Motivo: son reglas de producto/programacion, no capacidades esenciales del
nucleo de orquestacion. El core debe poder orquestar OPES u otros dominios sin
arrastrar contratos de programacion.

Consecuencia:

- MCP y delivery Codex dependen de este modulo.
- El core mantiene puertos genericos de observacion, progreso, review y
  scheduling.
- Cualquier nuevo dominio debe crear su propio modulo de contrato, no meter
  reglas en el core.
- La capacidad de juicio no vive aqui: Orquesta usa director/agentes para
  decidir plan, fases y estrategia. Este modulo solo aporta contratos de
  programacion para que el director tenga reglas verificables.

## 2026-05-13: sin adaptadores dentro del modulo

Decision: este modulo no lee disco ni ejecuta pruebas, aunque sus contratos
hablen de `write_set` y `required_tests`.

Motivo: la ejecucion real pertenece a conectores. Asi evitamos hardcodear DB,
runtime, VCS, proveedor, HOME o modelos.
