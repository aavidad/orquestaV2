# Incidencia: goal-first activo con solo checkpoint y consumo alto

Fecha: 2026-07-02.

ID inventario: `BUG-ORQ-20260702-099`.

Relacionado con: `BUG-ORQ-20260701-073`, `BUG-ORQ-20260701-079` y
`BUG-ORQ-20260701-088`.

## Sintoma

En varias olas OPES goal-first el backend Codex seguia `active`, consumia mas
de 100k tokens y solo habia materializado `checkpoint_started.txt` o refs de
checkpoint. Sin una causa publica especifica, `autoprogramming/status` podia
quedar entre esperar, bloquear o replanificar sin diferenciar progreso real de
consumo sin entrega recuperable.

## Causa

La capa MCP ya distinguia backend activo, timeout activo y checkpoint
materializado, pero no tenia una politica operacional para el caso combinado:

```text
goal active + tokens altos + artifact_refs solo checkpoint + sin domain_receipt_refs
```

## Cierre acotado

`autoprogramming/status` emite ahora `checkpoint_only_high_consumption` con
severidad `blocked`, accion recomendada `replan_narrow_context`, tokens,
checkpoint refs y evidencia
`evidence-ref-autoprogramming-checkpoint-only-high-consumption`.

No se anaden rails editoriales ni se cierra el backend automaticamente. El
residual de `BUG-088` sigue abierto para stop seguro/reconciliacion terminal
del backend y politica con ventana temporal mas fina.

## Evidencia

Pruebas locales:

```bash
go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPAutoprogrammingStatusExecutorV0(CheckpointOnlyHighConsumptionEsBloqueante|GoalBloqueadoConBackendActivoPublicaSnapshotAccionable|GoalActiveTimeoutConBackendActivoEsReplanAccionable|BackendActivoConSoloTokensHistoricosSigueBloqueado)'
go test -count=1 ./modulos/orquesta-mcp
```
