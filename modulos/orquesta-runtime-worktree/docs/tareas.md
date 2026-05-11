# Tareas: orquesta-runtime-worktree

## RTWT-001 - Snapshot y diff por write-set

Objetivo: capturar un snapshot base del proyecto y verificar cambios reales
contra write-set cerrado.

Write-set:

- `types_v0.go`
- `paths_v0.go`
- `snapshot_v0.go`
- `verify_v0.go`
- `memory_store_v0.go`
- `worktree_v0_test.go`
- `docs/*.md`

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-worktree`
- `git diff --check -- modulos/orquesta-runtime-worktree`

Bloqueos:

- No depender de Git.
- No filtrar rutas absolutas fuera del conector.
- No acoplar a Codex ni a ningun proveedor concreto.
