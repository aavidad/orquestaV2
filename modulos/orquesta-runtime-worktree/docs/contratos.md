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
