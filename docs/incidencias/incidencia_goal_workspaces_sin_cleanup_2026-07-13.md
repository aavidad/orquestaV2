# Incidencia: goal workspaces sin cleanup gobernado — 2026-07-13

## Estado observado

En el runner Docker local aislado se observaron:

- 117 worktrees registrados por Git;
- 115 directorios físicos y 115 manifests activos;
- 106 estados de goal: 31 complete, 41 blocked, 26 invalid y 8 running;
- 19 receipts de integración y 19 manifests de archive;
- un registro que `git worktree prune --dry-run` considera prunable.

Durante carga concurrente, `prepare-run` devolvió
`worktree_isolation_invalid / git.worktree_add / worktree_filesystem_error`.
El status agregado llegó a 504 mientras coexistían decenas de procesos de tests.

## Causa

`PrepareGoalWorkspaceV0` crea un `git worktree`, pero el puerto no posee una
operación de cleanup. La integración promueve el commit y
`ArchiveStagingWorktreeV0` solo escribe un manifest JSON. El propio plan anuncia
`archive_staging_without_delete`: archived no significa retirado.

Además, el lock actual es por goal. Dos goals distintos pueden ejecutar
`git worktree add/remove` concurrentemente contra el mismo registro
`.git/worktrees`. El error público conserva la operación y el exit code, pero no
una causa redacted suficientemente útil.

## Contrato del arreglo

1. Serializar `add`, `remove` y `prune` mediante un lock común por repositorio.
2. Persistir un receipt de cleanup por fases: `planned -> removing -> removed`.
3. Autorizar cleanup solo cuando existan estado terminal durable, closure/receipt
   válido, integración acreditada o workspace limpio, y backend concreto
   observable como inactivo.
4. Un goal blocked/invalid/dirty nunca usa `--force` sin preservar antes un
   bundle/patch con receipt.
5. Ejecutar `git worktree remove`, verificar ruta y registro, y solo después
   sustituir manifest/índice por tombstones.
6. Reconciliar de forma acotada y reejecutable cualquier caída entre fases.
7. Los huérfanos sin manifest solo se auditan; no se eliminan automáticamente.

## Evidencia exigida

- altas paralelas de goals distintos;
- rechazo de cleanup activo, no terminal, sin receipt o dirty;
- cleanup real de goal integrado y replay idempotente;
- crash antes/después de cada fase;
- carrera prepare-vs-cleanup;
- smoke que devuelve el número de worktrees al baseline;
- verificación de confinamiento al workspace root.

Rework activo: `request-ref-runner-goal-workspace-cleanup-20260713-013`, goal
`goal-ref-task-autoprogramming-36f69dcb5cc9-g01`.
