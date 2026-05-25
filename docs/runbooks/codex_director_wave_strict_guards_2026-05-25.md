# Runbook: codex-director-wave con guardas estrictas

Fecha: 2026-05-25.

## Alcance

Este runbook cubre el lanzamiento de olas Codex gobernadas por el Director
Operativo desde `cmd/orquesta-server codex-director-wave`.

## Default seguro

En ejecucion real, la ola usa guardas estrictas por defecto. El comando no debe
rellenar implicitamente `write_set=.` ni `operator-validation-required`. Si
faltan `branch_ref`, `worktree_ref`, `write_set` o `required_tests`, el summary
devuelve issues y no materializa agentes.

`--dry-run` conserva compatibilidad para diagnostico local y puede rellenar
rails minimos si no se activa `--strict-director-guards`.

## Opt-in auditado

Un operador puede permitir `write_set=.` o tests placeholder solo con opt-in
explicito:

```bash
go run ./cmd/orquesta-server codex-director-wave \
  --allow-global-write-set \
  --allow-placeholder-tests \
  --guard-override-reason "operador autoriza auditoria completa" \
  --guard-override-evidence-ref "evidence-ref-guard-override-001" \
  --write-set "." \
  --required-tests "operator-validation-required" \
  --branch-ref "branch-ref-opaca" \
  --worktree-ref "worktree-ref-opaca" \
  --objective "objetivo acotado"
```

El prompt mantiene las guardas de no borrar sin revision, no salir del proyecto
y no ampliar alcance por texto libre. El shard asignado es ownership inicial; el
alcance total autorizado sigue siendo el write-set global.

## Validacion focal

```bash
go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-director-operativo ./modulos/orquesta-app-codex-stack
```
