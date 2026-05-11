# orquesta-runtime-worktree

Conector externo para capturar snapshots compactos de un arbol de trabajo y
verificar que un agente solo cambio paths permitidos por su write-set.

Este modulo puede leer filesystem real porque vive fuera del nucleo. El nucleo
solo debe recibir el resultado neutral por puertos de adaptadores superiores.

Validacion local:

```bash
go test -count=1 ./modulos/orquesta-runtime-worktree
git diff --check -- modulos/orquesta-runtime-worktree
```

## Arranque

```bash
./arrancar_codex.sh "microtarea concreta"
```
