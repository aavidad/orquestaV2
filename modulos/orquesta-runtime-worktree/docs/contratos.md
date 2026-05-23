# Contratos: orquesta-runtime-worktree

## WorktreeSnapshotV0

Snapshot compacto de ficheros regulares bajo un `project_work_dir` externo.

Invariantes:

- `project_work_dir` entra por configuracion de conector, nunca por el nucleo.
- Los paths del snapshot son relativos, normalizados con `/` y sin `..`.
- No se capturan symlinks ni rutas absolutas.
- Los prefijos de control a ignorar se inyectan por request.

## VerifyWorktreeWriteSetV0

Compara un snapshot base con el estado actual y clasifica cambios contra un
write-set cerrado.

Invariantes:

- `write_set=["."]` permite una app nueva completa, pero los cambios siguen
  saliendo como paths relativos concretos.
- Un cambio fuera de write-set produce issue `outside_write_set`.
- No conoce Codex, Claude, Gemini, DB, HOME, OAuth ni modelos.
- El adaptador que consuma el resultado decide si registra entrega, pide
  revision o corta agente.

## PrepareIsolatedWorktreeV0

Prepara evidencia neutral de una worktree aislada para autoprogramacion.

Invariantes:

- `project_work_dir` es entrada del conector filesystem y no se devuelve en el
  resultado publico.
- `project_ref`, `worktree_ref`, `branch_ref`, `isolation_ref` y `baseline_ref`
  son refs opacas; `branch_ref` no puede ser ruta ni URL.
- `isolated=true` es obligatorio; si falta, el conector rechaza la preparacion.
- El resultado conserva `worktree_ref` y `branch_ref` sin convertirlos en nombres
  Git, rutas, proveedor ni HOME.
- La evidencia publica contiene snapshot con paths relativos e ignora prefijos
  de control inyectados.
- La recuperacion tras reinicio debe reabrir o rematerializar el baseline por
  `baseline_ref` y las refs opacas guardadas; no debe reconstruir una rama desde
  nombres Git ni desde paths locales.
