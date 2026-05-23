# Control de versiones por app - 2026-05-23

## Alcance

El control de versiones de apps entra como puerto-conector exterior. La
superficie publica usa refs opacas (`app_ref`, `repo_ref`, `worktree_ref`,
`branch_ref`) y no expone rutas locales, HOME, proveedor, modelo ni runtime.

## Inventario reutilizado

Se busco primero el inventario pedido:

- `archive/legacy-root-before-purge-2026-05-11`;
- `archive/branches/2026-05-11/orq-orquesta-codex2`.

Esas rutas no existen en el workdir actual. Como compatibilidad historica se
reutilizo el patron de las piezas archivadas `gitoperaciones` y
`gitestadisticasapp`: inspeccion con `git status --porcelain`, HEAD por
`rev-parse`, rutas relativas de estado, commit local por `git add -A` +
`git commit`, push sin prompt interactivo y salida publica compacta.

## Contrato

Entrada MCP/REST:

```text
POST /api/v0/apps/vcs
tool: orquesta.app_vcs.v0
action: prepare_repo | commit | push
```

Campos principales:

- `app_ref`, `repo_ref`: refs opacas obligatorias;
- `worktree_ref`, `branch_ref`: refs opacas opcionales preservadas;
- `commit_message`: obligatorio para `commit`;
- `allow_push`: obligatorio para `push` y opcional para intentar push despues
  de `commit`.

El stack Codex inyecta el conector Git contra el repo configurado por la
composicion. MCP y gateway solo delegan.

## Semantica

- `prepare_repo`: valida repo local y devuelve `commit_ref` de HEAD, rutas
  relativas cambiadas y evidencia compacta.
- `commit`: registra un commit local si hay cambios. Devuelve `commit_ref`
  publico y deja el repo limpio.
- `commit` con `allow_push=true` o `push`: intenta push remoto solo por opt-in.
  Si el remoto no esta disponible, el commit local queda registrado y la salida
  marca `status=pending_push`, `push_pending=true` y `retryable=true`.

## Verificacion focal

```bash
go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack -run 'TestGitAppVCS|TestMCPAppVCS|TestAppVCSAPI|TestCodexStackAppVCS'
```
