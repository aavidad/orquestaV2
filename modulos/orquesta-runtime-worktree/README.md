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

La ruta vigente para agentes OrquestaV2 es el servidor residente y la cola
gobernada, no el wrapper local. `./arrancar_codex.sh` queda reservado a
compatibilidad historica o recuperacion manual con error publico cuando no haya
contrato versionado.

## Reconciliacion T208

En el guardian break-glass, este modulo solo verifica efectos sobre worktrees y
preserva refs opacas. No convierte `worktree_ref` ni `branch_ref` en rutas,
ramas Git o ubicaciones de control; la promocion/restart vive en servidor,
guardian y adaptadores opt-in.
