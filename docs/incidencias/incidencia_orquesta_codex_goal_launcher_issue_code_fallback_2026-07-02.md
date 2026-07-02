# Incidencia: Codex Goal launcher perdia causa si faltaba IssueCode

Fecha: 2026-07-02

## Sintoma

Cuando el backend Codex Goal fallaba antes de devolver un `IssueCode`, el
adaptador neutral de `orquesta-runtime-codex-goal` invalidaba el lanzamiento u
observacion con `codex_goal_start_rejected` o
`codex_goal_observation_rejected`. Eso ocultaba causas operativas conocidas como
`ResetStdio`, sesion tmux salida o permisos de `bwrap`.

## Cierre aplicado

- `CodexGoalLauncherV0` conserva la prioridad del `IssueCode` explicito.
- Si el backend solo devuelve `error`, clasifica familias conocidas en codigos
  compactos: stdio wrapper, tmux exited, permisos/operation not permitted,
  auth/cuota, socket, standalone o comando ausente.
- `CodexGoalObserverV0` usa el mismo fallback.
- No se copia stderr completo, HOME, rutas ni transcript al receipt.

## Evidencia

```bash
go test -count=1 ./modulos/orquesta-runtime-codex-goal
```

Tests focales:

- `TestCodexGoalLauncherV0ClasificaErrorBackendSinIssueCodeV0`
- `TestCodexGoalObserverV0ClasificaErrorBackendSinIssueCodeV0`

## Estado

Cerrado en codigo local. No cambia `cmd/orquesta-server` ni stack.
