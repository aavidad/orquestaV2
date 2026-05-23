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

## RTWT-002 - Worktree aislada para autoprogramacion

Objetivo: preparar evidencia contractual de worktree aislada preservando
`worktree_ref` y `branch_ref` como refs opacas.

Write-set:

- `refs_v0.go`
- `isolation_v0.go`
- `worktree_v0_test.go`
- `docs/*.md`

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-worktree`
- `git diff --check -- modulos/orquesta-runtime-worktree`

Bloqueos:

- No ejecutar Git ni crear ramas.
- No devolver rutas absolutas al nucleo.
- No convertir `branch_ref` en nombre Git concreto.

Estado: cerrada localmente el 2026-05-23.
