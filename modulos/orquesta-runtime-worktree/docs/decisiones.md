# Decisiones: orquesta-runtime-worktree

```text
Fecha: 2026-05-10
Decision: Verificar efectos reales fuera del nucleo.
Motivo: el ACK de un agente declara artifacts, pero para apps grandes tambien
hay que comprobar que el worktree no recibio cambios fuera del write-set. Hacerlo
en el nucleo filtraria filesystem y rutas operacionales.
Alternativas:
  - Confiar solo en ACK: descartado porque un agente puede omitir cambios.
  - Meter diff de Git en core: descartado por acoplamiento a herramienta y VCS.
Impacto: se crea un conector de snapshot/diff por filesystem inyectable desde
adaptadores de runtime.
Estado: aceptada.
```
