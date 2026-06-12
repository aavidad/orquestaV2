# Contratos: orquesta-runtime-worktree

## WorktreeSnapshotV0

Snapshot compacto de ficheros regulares bajo un `project_work_dir` externo.

Invariantes:

- `project_work_dir` entra por configuracion de conector, nunca por el nucleo.
- Los paths del snapshot son relativos, normalizados con `/` y sin `..`.
- No se capturan symlinks ni rutas absolutas.
- Los prefijos de control a ignorar se inyectan por request.
- Por defecto, superar el presupuesto de lectura devuelve issue y no snapshot.
  Si el adaptador activa `allow_partial`, los ficheros omitidos por presupuesto
  quedan como `omitted_paths` relativos y `exclusion_receipts`, sin bloquear la
  captura ni filtrar rutas absolutas.

## VerifyWorktreeWriteSetV0

Compara un snapshot base con el estado actual y clasifica cambios contra un
write-set cerrado.

Invariantes:

- `write_set=["."]` permite una app nueva completa, pero los cambios siguen
  saliendo como paths relativos concretos.
- Un cambio fuera de write-set produce issue `outside_write_set`.
- Un borrado detectado produce issue `removed_path` aunque el path este dentro
  del write-set; el borrado es rail estricto y requiere decision humana/director
  antes de repetirse con autorizacion explicita futura.
- Un truncado fuerte, renombre/movimiento ambiguo o reemplazo con delta grande
  se clasifica como `truncated`, `renamed_or_moved` o
  `replaced_large_delta` desde el snapshot, sin depender de `ACK.files`.
- No conoce Codex, Claude, Gemini, DB, HOME, OAuth ni modelos.
- El adaptador que consuma el resultado decide si registra entrega, pide
  revision o corta agente.
- Si el snapshot actual es parcial por presupuesto, los paths omitidos no se
  clasifican como borrados; la incertidumbre queda como recibo/evidencia de
  snapshot parcial para que el Director decida follow-up.

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
- En autoprogramacion residente, la composicion puede pedir snapshot parcial
  para que un artefacto local grande no bloquee la preparacion de la worktree.
- La recuperacion tras reinicio debe reabrir o rematerializar el baseline por
  `baseline_ref` y las refs opacas guardadas; no debe reconstruir una rama desde
  nombres Git ni desde paths locales.

## AppVCS `review_repo`

Revision Git read-only para una app externa desde AppVCS.

Invariantes:

- reutiliza `GitAppVCSConnectorV0`, pero no ejecuta `add`, `commit` ni `push`;
- devuelve `commit_ref`, `commit_short_ref`, `changed_paths` relativos y status
  `clean` cuando el repo no tiene cambios;
- conserva `worktree_ref` y `branch_ref` como refs opacas;
- no devuelve `project_work_dir`, ramas Git concretas ni rutas absolutas.
