# AGENTS: orquesta-opes-director

Este modulo es adaptador de composicion OPES-Orquesta.

## Reglas

- No meter logica OPES en core, workflow, domain-work ni expander puro.
- Trabajar solo con puertos, refs compactas y contratos publicos.
- No leer DB, filesystem interno ni rutas locales de OPES.
- No elegir runtime, proveedor, modelo ni sesiones de agentes.
- No inferir por texto libre si falta una decision estructurada: crear trabajo
  de consolidacion/director con refs causales.
- Todo job creado debe tener `idempotency_key` estable por source job,
  artifact, receipt y followup/rework.
