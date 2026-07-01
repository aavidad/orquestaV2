# Incidencia: goal-first no debe propagar base64 ni salidas gigantes en resultados

Fecha: 2026-07-02

## Sintoma

En ejecuciones goal-first OPES se observo que salidas multimodales, por ejemplo
`data:image/png;base64,...`, podian entrar en el historial operativo y degradar
coste, observabilidad y cierre. Aunque el bug de origen esta en la proyeccion de
herramientas visuales al contexto, el resultado terminal tambien debia proteger
la persistencia y los receipts.

## Riesgo

Un `ORQUESTA_GOAL_RESULT_V0` o `orquesta_goal_result_v0.json` con resumen,
artefactos o evidencias gigantes puede contaminar:

- `GoalWorkResultV0`.
- `AppGoalStateStore`.
- `autoprogramming/status`.
- decisiones posteriores de replan, shutdown y QA.

## Cierre aplicado

- `cmd/orquesta-server` sanea `summary`, `artifact_refs`,
  `domain_receipt_refs`, `evidence_refs` y evidencias de required tests del
  resultado goal-first.
- Si detecta `data:*;base64,` o un valor mayor del limite operativo, sustituye
  el contenido por una ref hash compacta con bytes.
- Anade evidencia
  `evidence-ref-codex-app-server-goal-result-output-sanitized`.
- La extraccion del marcador final recorta el texto desde
  `ORQUESTA_GOAL_RESULT_V0` para no arrastrar prefijos enormes del transcript.

## Evidencia

Tests:

```bash
go test -count=1 ./cmd/orquesta-server -run 'TestServerCodexAppServerGoalBackendV0(SanitizaResultadoDurableConBase64|ObservaResultadoDurableSinMarcador|ObservaResultadoMarcado)|TestCodexAppServer(FinalMarkerTextV0RecortaPrefijoGrande|GoalResultMarkerV0AceptaFallback)'
go test -count=1 ./cmd/orquesta-server
```

## Residual

Queda fuera de este cierre la politica completa de herramienta visual: la salida
de `view_image` debe registrarse como referencia, hash, dimensiones y resumen,
no como payload textual base64 dentro del historial que consume Codex.
