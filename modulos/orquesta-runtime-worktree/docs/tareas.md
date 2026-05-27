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

## RTWT-005 - Reconciliacion T208

Objetivo: dejar claro que el guardian break-glass usa este modulo solo como
adaptador de verificacion externa, no como owner de promocion, servidor ni
Codex.

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-worktree`
- bateria T208 cruzada del paquete OrquestaV2.

Estado: documentado el 2026-05-27; no requiere codigo nuevo.

## RTWT-004 - Rail estricto de no borrado

Objetivo: convertir los borrados detectados por snapshot en issue bloqueante
`removed_path`, independiente de que el path pertenezca al write-set.

Write-set:

- `types_v0.go`
- `verify_v0.go`
- `worktree_v0_test.go`
- `docs/*.md`

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-worktree`

Bloqueos:

- No borrar automaticamente ni revertir archivos.
- No enviar rutas absolutas al nucleo.

Estado: cerrado localmente el 2026-05-24.

## RTWT-003 - Revision Git read-only para AppVCS

Objetivo: ampliar AppVCS con `review_repo` para inspeccionar una app/repo Git
temporal desde el conector externo.

Write-set:

- `app_vcs_types_v0.go`
- `app_vcs_validate_v0.go`
- `app_vcs_git_v0.go`
- `app_vcs_v0_test.go`
- `docs/*.md`

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-worktree`

Bloqueos:

- No hacer commit ni push.
- No devolver rutas absolutas ni convertir `branch_ref` en rama Git.

Estado: cerrada localmente el 2026-05-23.
