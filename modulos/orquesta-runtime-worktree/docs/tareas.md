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

## RTWT-006 - Snapshot OPES grande con error accionable

Estado: abierto 2026-06-22.

Origen:
`docs/incidencia_opes_autonomia_tractorista_ack_idle_stop_2026-06-22.md`.

Objetivo: cuando `prepare-run` no pueda capturar snapshot de un workdir grande,
como la raiz de OPES, el error publico debe dar una causa accionable sin filtrar
rutas privadas. El operador no debe tener que adivinar si el problema era un
directorio enorme, symlink, fichero no legible, presupuesto excedido o falta de
scope.

Caso observado:

- `project_workdir=/home/alberto/Trabajo/OPES`;
- `prepare-run` devolvio
  `worktree_isolation_invalid / worktree_snapshot_unreadable`;
- el campo publico fue solo `field=file`;
- la prueba pudo continuar al acotar manualmente `project_workdir` al padre del
  curso, pero esa decision no fue sugerida por Orquesta.

Alcance:

- conservar refs opacas hacia el nucleo, pero anadir evidencia redactada en el
  borde externo: tipo de fallo, prefijo relativo redactado o categoria
  `budget_exceeded`, `permission_denied`, `symlink_unreadable`,
  `ignored_prefix_missing`;
- soportar politica de include/scope por write-set para snapshots de
  autoprogramacion cuando sea seguro;
- permitir perfil OPES que ignore caches, backups, runtime, codex homes,
  paquetes generados y salidas historicas fuera del write-set;
- documentar recomendacion publica: usar workdir acotado o configurar ignore
  prefixes.

Criterio de cierre:

- test con arbol artificial que contiene ruta no legible: el error no filtra la
  ruta absoluta y si expone categoria accionable;
- test con arbol grande y write-set acotado: snapshot limitado al scope no falla
  por directorios ajenos;
- smoke OPES opt-in o fixture equivalente demuestra que la raiz OPES no rompe
  `prepare-run`, o que el error propone scope/ignore concreto.

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
