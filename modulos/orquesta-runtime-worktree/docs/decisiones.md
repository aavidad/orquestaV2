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

```text
Fecha: 2026-05-23
Decision: La worktree aislada de autoprogramacion se valida por contrato neutral
y preserva `branch_ref` como ref opaca.
Motivo: el runtime puede necesitar una worktree aislada antes de lanzar agentes,
pero este conector no debe decidir Git, nombres de rama, proveedor, HOME ni
merge.
Impacto: `PrepareIsolatedWorktreeV0` exige `isolated=true`, refs opacas y
snapshot relativo; no devuelve `project_work_dir` ni convierte `branch_ref` en
ruta o nombre Git.
Estado: aceptada.
```
