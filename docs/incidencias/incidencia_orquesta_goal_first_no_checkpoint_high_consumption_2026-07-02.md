# Incidencia: Goal activo con alto consumo sin checkpoint

Fecha: 2026-07-02.

ID inventario: `BUG-ORQ-20260702-112`.

Relacionado con: `BUG-ORQ-20260701-079` y `BUG-ORQ-20260701-088`.

## Sintoma

Un goal Codex podia seguir `active` con consumo alto de tokens sin haber
materializado ningun fichero en el write-set, checkpoint, artefacto ni receipt
de dominio. La proyeccion publica podia clasificarlo como
`active_no_checkpoint_yet` informativo y recomendar esperar, aunque el patron
ya indicaba consumo sin entrega recuperable.

## Causa

La politica existente distinguia:

- goal activo con actividad reciente y sin checkpoint, como espera inicial;
- alto consumo con solo refs de checkpoint, como bloqueo.

Faltaba el tercer caso: alto consumo sin checkpoint ni artefactos.

## Cierre acotado

`autoprogramming/status` ahora emite
`goal_active_no_checkpoint_high_consumption` cuando se cumplen estas
condiciones:

- el backend Goal observado sigue activo;
- `tokens_used >= 100000`;
- no hay `artifact_refs`;
- no hay `domain_receipt_refs`.

El estado se publica como `blocked`, con accion `replan_narrow_context` y
evidencia `evidence-ref-autoprogramming-no-checkpoint-high-consumption`.

Este cierre no resuelve todavia todos los residuales de `BUG-079`: quedan la
politica de checkpoint temprano dentro del prompt/runtime, limites de salida de
herramientas y reconciliacion terminal tras shutdown forzado sin artefactos.

## Evidencia

```bash
go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPAutoprogrammingStatusExecutorV0(CheckpointOnlyHighConsumptionEsBloqueante|SinCheckpointHighConsumptionEsBloqueante|BackendActivoConSoloTokensHistoricosSigueBloqueado|BackendActivoConActividadRecienteNoMarcaStale)'
go test -count=1 ./modulos/orquesta-mcp
```
